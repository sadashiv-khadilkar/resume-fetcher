package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"resumefetcher/internal/domain"
	claudellm "resumefetcher/internal/llmprovider/claude"
	fakellm "resumefetcher/internal/llmprovider/fake"
	"resumefetcher/internal/pipeline"
	"resumefetcher/internal/platform/linkedinclient"
	"resumefetcher/internal/platform/naukriclient"
)

func TestParseSources(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []domain.Source
		wantErr bool
	}{
		{name: "both", value: "both", want: []domain.Source{domain.SourceNaukri, domain.SourceLinkedIn}},
		{name: "naukri", value: "naukri", want: []domain.Source{domain.SourceNaukri}},
		{name: "linkedin", value: "linkedin", want: []domain.Source{domain.SourceLinkedIn}},
		{name: "case-insensitive", value: "NauKri", want: []domain.Source{domain.SourceNaukri}},
		{name: "invalid", value: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSources(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for an invalid --source value")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSources() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseSources() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseSources()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestSelectPlatforms checks the real-client wiring decision itself
// (naukriclient for Naukri per ticket 04, linkedinclient for LinkedIn per
// ticket 07) without exercising either client's browser automation - real
// PlatformClient construction does no network/browser work, only Search and
// Login do.
func TestSelectPlatforms(t *testing.T) {
	var out bytes.Buffer
	entries := selectPlatforms([]domain.Source{domain.SourceNaukri, domain.SourceLinkedIn}, &out)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Source != domain.SourceNaukri {
		t.Errorf("entries[0].Source = %v, want %v", entries[0].Source, domain.SourceNaukri)
	}
	if _, ok := entries[0].Client.(*naukriclient.Client); !ok {
		t.Errorf("entries[0].Client is %T, want *naukriclient.Client", entries[0].Client)
	}

	if entries[1].Source != domain.SourceLinkedIn {
		t.Errorf("entries[1].Source = %v, want %v", entries[1].Source, domain.SourceLinkedIn)
	}
	if _, ok := entries[1].Client.(*linkedinclient.Client); !ok {
		t.Errorf("entries[1].Client is %T, want *linkedinclient.Client", entries[1].Client)
	}
}

func TestBuildLLMProvider(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		unset      bool
		wantClaude bool
		wantErr    bool
	}{
		{name: "unset defaults to fake", unset: true},
		{name: "fake", envValue: "fake"},
		{name: "claude", envValue: "claude", wantClaude: true},
		{name: "case-insensitive", envValue: "CLAUDE", wantClaude: true},
		{name: "invalid value", envValue: "bogus", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				t.Setenv("LLM_PROVIDER", "")
				os.Unsetenv("LLM_PROVIDER")
			} else {
				t.Setenv("LLM_PROVIDER", tt.envValue)
			}

			got, err := buildLLMProvider()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for an invalid LLM_PROVIDER value")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildLLMProvider() error = %v", err)
			}

			_, isClaude := got.(*claudellm.Provider)
			_, isFake := got.(*fakellm.Provider)
			switch {
			case tt.wantClaude && !isClaude:
				t.Errorf("got %T, want *claude.Provider", got)
			case !tt.wantClaude && !isFake:
				t.Errorf("got %T, want *fake.Provider", got)
			}
		})
	}
}

func TestWritePartialOnRankFailure(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	rankErr := &pipeline.RankFailedError{
		Err: os.ErrDeadlineExceeded,
		RawResult: &pipeline.RunResult{
			Candidates: []domain.Candidate{
				{ID: "c1", Profiles: []domain.Profile{{Name: "Asha Rao", Source: domain.SourceNaukri}}},
			},
			Resumes: map[string]domain.ResumeFile{},
		},
	}

	err := writePartialOnRankFailure(&out, outDir, "# Backend Engineer JD", time.Now(), rankErr)
	if err == nil {
		t.Fatal("expected writePartialOnRankFailure to still return an error")
	}

	entries, readErr := os.ReadDir(outDir)
	if readErr != nil {
		t.Fatalf("read outDir: %v", readErr)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries in outDir, want 1 run folder", len(entries))
	}

	raw, readErr := os.ReadFile(filepath.Join(outDir, entries[0].Name(), "candidates.json"))
	if readErr != nil {
		t.Fatalf("candidates.json not found: %v", readErr)
	}
	var doc struct {
		Candidates []struct {
			ID string `json:"id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("candidates.json invalid: %v", err)
	}
	if len(doc.Candidates) != 1 || doc.Candidates[0].ID != "c1" {
		t.Errorf("got candidates %+v, want the raw unranked pool preserved", doc.Candidates)
	}

	if out.Len() == 0 {
		t.Error("expected a message about the ranking failure to be printed")
	}
}

func TestWritePartialOnRankFailure_NonRankError(t *testing.T) {
	var out bytes.Buffer
	err := writePartialOnRankFailure(&out, t.TempDir(), "# JD", time.Now(), os.ErrPermission)
	if err == nil {
		t.Fatal("expected an error to be returned")
	}
	if out.Len() != 0 {
		t.Errorf("did not expect any output for a non-rank error, got %q", out.String())
	}
}
