package output_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/output"
	"resumefetcher/internal/pipeline"
)

func TestWriteRun_BackToBackRunsDoNotCollide(t *testing.T) {
	baseDir := t.TempDir()
	result := &pipeline.RunResult{
		Candidates: []domain.Candidate{{ID: "cand-1", Profiles: []domain.Profile{{Name: "Someone"}}}},
	}

	firstDir, err := output.WriteRun(baseDir, result)
	if err != nil {
		t.Fatalf("first WriteRun() error = %v", err)
	}
	secondDir, err := output.WriteRun(baseDir, result)
	if err != nil {
		t.Fatalf("second WriteRun() error = %v", err)
	}

	if firstDir == secondDir {
		t.Fatalf("two back-to-back runs produced the same run dir %q, want distinct dirs", firstDir)
	}
	for _, dir := range []string{firstDir, secondDir} {
		if _, err := os.Stat(filepath.Join(dir, "candidates.json")); err != nil {
			t.Errorf("candidates.json missing for run dir %q: %v", dir, err)
		}
	}
}

func TestWriteRun_WritesCandidatesJSONAndResumeFiles(t *testing.T) {
	baseDir := t.TempDir()

	result := &pipeline.RunResult{
		Candidates: []domain.Candidate{
			{
				ID:         "cand-with-resume",
				Profiles:   []domain.Profile{{Name: "Has Resume", Source: domain.SourceNaukri}},
				MatchScore: 90,
			},
			{
				ID:         "cand-no-resume",
				Profiles:   []domain.Profile{{Name: "No Resume", Source: domain.SourceLinkedIn}},
				MatchScore: 40,
				ResumeNote: "resume not available",
			},
		},
		Resumes: map[string]domain.ResumeFile{
			"cand-with-resume": {Available: true, Data: []byte("%PDF-fake-bytes"), Ext: "pdf"},
		},
	}

	runDir, err := output.WriteRun(baseDir, result)
	if err != nil {
		t.Fatalf("WriteRun() error = %v", err)
	}

	if _, err := os.Stat(runDir); err != nil {
		t.Fatalf("run dir %q was not created: %v", runDir, err)
	}

	jsonPath := filepath.Join(runDir, "candidates.json")
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("candidates.json not written: %v", err)
	}

	var doc struct {
		Candidates []domain.Candidate `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("candidates.json is not valid JSON: %v", err)
	}
	if len(doc.Candidates) != 2 {
		t.Fatalf("wrote %d candidates, want 2", len(doc.Candidates))
	}

	var withResume, withoutResume *domain.Candidate
	for i := range doc.Candidates {
		switch doc.Candidates[i].ID {
		case "cand-with-resume":
			withResume = &doc.Candidates[i]
		case "cand-no-resume":
			withoutResume = &doc.Candidates[i]
		}
	}
	if withResume == nil || withResume.ResumePath == "" {
		t.Fatalf("expected cand-with-resume to have a non-empty ResumePath, got %+v", withResume)
	}

	resumeOnDisk, err := os.ReadFile(filepath.Join(runDir, withResume.ResumePath))
	if err != nil {
		t.Fatalf("resume file not found at %q: %v", withResume.ResumePath, err)
	}
	if string(resumeOnDisk) != "%PDF-fake-bytes" {
		t.Errorf("resume file contents = %q, want %q", resumeOnDisk, "%PDF-fake-bytes")
	}

	if withoutResume == nil || withoutResume.ResumePath != "" {
		t.Errorf("expected cand-no-resume to have no ResumePath, got %+v", withoutResume)
	}
	if withoutResume.ResumeNote != "resume not available" {
		t.Errorf("ResumeNote = %q, want %q", withoutResume.ResumeNote, "resume not available")
	}
}
