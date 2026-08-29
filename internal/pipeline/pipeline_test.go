package pipeline_test

import (
	"testing"

	"resumefetcher/internal/domain"
	fakellm "resumefetcher/internal/llmprovider/fake"
	"resumefetcher/internal/pipeline"
	fakeplatform "resumefetcher/internal/platform/fake"
	"resumefetcher/internal/reviewer"
)

func TestOrchestrator_Run_RanksAndCapsCandidates(t *testing.T) {
	naukri := fakeplatform.New(domain.SourceNaukri, []domain.Profile{
		{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Low Match", Email: "low@example.com"},
		{ExternalID: "n2", Source: domain.SourceNaukri, Name: "High Match", Email: "high@example.com"},
	})
	linkedin := fakeplatform.New(domain.SourceLinkedIn, []domain.Profile{
		{ExternalID: "l1", Source: domain.SourceLinkedIn, Name: "Mid Match", Email: "mid@example.com"},
	})

	llm := &fakellm.Provider{
		ScoreFunc: func(c domain.Candidate) float64 {
			switch c.Profiles[0].Name {
			case "High Match":
				return 90
			case "Mid Match":
				return 50
			default:
				return 10
			}
		},
	}

	orch := &pipeline.Orchestrator{
		Platforms: []pipeline.PlatformEntry{
			{Source: domain.SourceNaukri, Client: naukri},
			{Source: domain.SourceLinkedIn, Client: linkedin},
		},
		LLM:      llm,
		Reviewer: reviewer.AutoAccept{},
		Config:   pipeline.Config{RawPoolCapPerPlatform: 50, MaxCandidates: 2, ResumeDownloadTopN: 1},
	}

	result, err := orch.Run("some JD text")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.Candidates) != 2 {
		t.Fatalf("got %d candidates, want 2 (MaxCandidates cap)", len(result.Candidates))
	}
	if result.Candidates[0].Profiles[0].Name != "High Match" {
		t.Errorf("Candidates[0] = %q, want %q (highest Match Score first)", result.Candidates[0].Profiles[0].Name, "High Match")
	}
	if result.Candidates[1].Profiles[0].Name != "Mid Match" {
		t.Errorf("Candidates[1] = %q, want %q", result.Candidates[1].Profiles[0].Name, "Mid Match")
	}
}

func TestOrchestrator_Run_DownloadsResumesForTopNOnly(t *testing.T) {
	naukri := fakeplatform.New(domain.SourceNaukri, []domain.Profile{
		{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Top", Email: "top@example.com"},
		{ExternalID: "n2", Source: domain.SourceNaukri, Name: "Bottom", Email: "bottom@example.com"},
	})
	naukri.Resumes["n1"] = domain.ResumeFile{Available: true, Data: []byte("resume-bytes"), Ext: "pdf"}
	naukri.Resumes["n2"] = domain.ResumeFile{Available: true, Data: []byte("should-not-download"), Ext: "pdf"}

	llm := &fakellm.Provider{
		ScoreFunc: func(c domain.Candidate) float64 {
			if c.Profiles[0].Name == "Top" {
				return 100
			}
			return 1
		},
	}

	orch := &pipeline.Orchestrator{
		Platforms: []pipeline.PlatformEntry{{Source: domain.SourceNaukri, Client: naukri}},
		LLM:       llm,
		Reviewer:  reviewer.AutoAccept{},
		Config:    pipeline.Config{RawPoolCapPerPlatform: 50, MaxCandidates: 50, ResumeDownloadTopN: 1},
	}

	result, err := orch.Run("some JD text")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	topID := result.Candidates[0].ID
	bottomID := result.Candidates[1].ID

	if _, ok := result.Resumes[topID]; !ok {
		t.Errorf("expected a downloaded resume for the top-ranked candidate")
	}
	if _, ok := result.Resumes[bottomID]; ok {
		t.Errorf("did not expect a downloaded resume for a candidate outside ResumeDownloadTopN")
	}
}

func TestOrchestrator_Run_ErrorsWhenRankOmitsACandidate(t *testing.T) {
	naukri := fakeplatform.New(domain.SourceNaukri, []domain.Profile{
		{ExternalID: "n1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "asha@example.com"},
	})

	// A buggy/hallucinating LLMProvider that returns no Match at all for the
	// one Candidate in the pool.
	llm := &fakellm.Provider{ScoreFunc: nil}
	llm.RankOverride = func(_ string, _ []domain.Candidate) ([]domain.Match, error) {
		return nil, nil
	}

	orch := &pipeline.Orchestrator{
		Platforms: []pipeline.PlatformEntry{{Source: domain.SourceNaukri, Client: naukri}},
		LLM:       llm,
		Reviewer:  reviewer.AutoAccept{},
		Config:    pipeline.DefaultConfig(),
	}

	if _, err := orch.Run("some JD text"); err == nil {
		t.Fatal("expected an error when Rank() omits a candidate, got nil")
	}
}

func TestOrchestrator_Run_UsesReviewedFilters(t *testing.T) {
	naukri := fakeplatform.New(domain.SourceNaukri, nil)
	llm := &fakellm.Provider{FiltersToReturn: domain.Filters{Location: "Pune"}}

	var gotFilters domain.Filters
	spyReviewer := reviewerFunc(func(f domain.Filters) (domain.Filters, error) {
		gotFilters = f
		f.Location = "Bangalore"
		return f, nil
	})

	captured := &capturingClient{Client: naukri}

	orch := &pipeline.Orchestrator{
		Platforms: []pipeline.PlatformEntry{{Source: domain.SourceNaukri, Client: captured}},
		LLM:       llm,
		Reviewer:  spyReviewer,
		Config:    pipeline.Config{RawPoolCapPerPlatform: 50, MaxCandidates: 50, ResumeDownloadTopN: 0},
	}

	if _, err := orch.Run("JD"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if gotFilters.Location != "Pune" {
		t.Errorf("reviewer received Location = %q, want %q (the extracted filters)", gotFilters.Location, "Pune")
	}
	if captured.searchedWith.Location != "Bangalore" {
		t.Errorf("Search called with Location = %q, want %q (the reviewed filters)", captured.searchedWith.Location, "Bangalore")
	}
}

type reviewerFunc func(domain.Filters) (domain.Filters, error)

func (f reviewerFunc) Review(filters domain.Filters) (domain.Filters, error) { return f(filters) }

type capturingClient struct {
	*fakeplatform.Client
	searchedWith domain.Filters
}

func (c *capturingClient) Search(filters domain.Filters) ([]domain.Profile, error) {
	c.searchedWith = filters
	return c.Client.Search(filters)
}
