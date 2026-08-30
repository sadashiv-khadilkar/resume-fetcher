package claude_test

import (
	"os"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/llmprovider/claude"
)

// These are smoke tests against the real Claude API (ticket 02): they skip
// without a live ANTHROPIC_API_KEY rather than fail, since this repo's bulk
// LLMProvider-consuming logic is exercised via the fake in
// internal/llmprovider/fake instead (see .scratch/resume-fetcher/spec.md's
// Testing Decisions).
func skipWithoutAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping live Claude smoke test")
	}
}

func TestProvider_ExtractFilters_Smoke(t *testing.T) {
	skipWithoutAPIKey(t)

	p := claude.New("")
	jd := "# Backend Engineer\n\nWe need a Go engineer with 4-8 years of experience, based in Bangalore, skilled in Go and Postgres."

	filters, err := p.ExtractFilters(jd)
	if err != nil {
		t.Fatalf("ExtractFilters() error = %v", err)
	}
	if len(filters.Skills) == 0 {
		t.Error("expected at least one extracted skill")
	}
	if filters.Location == "" {
		t.Error("expected a non-empty extracted location")
	}
}

func TestProvider_Rank_Smoke(t *testing.T) {
	skipWithoutAPIKey(t)

	p := claude.New("")
	jd := "# Backend Engineer\n\nWe need a Go engineer with strong distributed-systems experience."

	candidates := []domain.Candidate{
		{
			ID: "c1",
			Profiles: []domain.Profile{
				{Name: "Asha Rao", Title: "Backend Engineer", Skills: []string{"Go", "Postgres", "Kubernetes"}, Experience: 6},
			},
		},
		{
			ID: "c2",
			Profiles: []domain.Profile{
				{Name: "Sam Lee", Title: "Frontend Engineer", Skills: []string{"React", "CSS"}, Experience: 6},
			},
		},
	}

	matches, err := p.Rank(jd, candidates)
	if err != nil {
		t.Fatalf("Rank() error = %v", err)
	}
	if len(matches) != len(candidates) {
		t.Fatalf("got %d matches, want %d (one per candidate)", len(matches), len(candidates))
	}

	scoreByID := make(map[string]float64, len(matches))
	for _, m := range matches {
		scoreByID[m.CandidateID] = m.Score
	}
	if scoreByID["c1"] <= scoreByID["c2"] {
		t.Errorf("expected the Go backend candidate (c1, score %v) to outscore the frontend candidate (c2, score %v)", scoreByID["c1"], scoreByID["c2"])
	}
}
