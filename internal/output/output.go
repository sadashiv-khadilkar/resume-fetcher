// Package output writes a Fetch Run's Candidates and Resumes to disk: one
// folder per run, a candidates.json, and a resumes/ subfolder.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/pipeline"
)

type candidatesDocument struct {
	Candidates []domain.Candidate `json:"candidates"`
}

// RunMetaFileName is the file WriteRunMeta writes into a run's output
// folder, and the file ticket 09's `runs` command reads back.
const RunMetaFileName = "run.json"

// RunMeta captures the per-run metadata that isn't implied by
// candidates.json but that `runs` (ticket 09) needs to list past Fetch
// Runs: the JD used and when the run started. It's written into the same
// per-run folder WriteRun already creates, so `runs` still reads only from
// the existing per-run output folder structure - no separate run-tracking
// store is introduced.
type RunMeta struct {
	JD        string    `json:"jd"`
	StartedAt time.Time `json:"started_at"`
}

// WriteRunMeta writes meta into runDir (a folder previously returned by
// WriteRun) as run.json.
func WriteRunMeta(runDir string, meta RunMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, RunMetaFileName), data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", RunMetaFileName, err)
	}
	return nil
}

// WriteRun writes result under a new timestamped folder inside baseDir and
// returns that folder's path. The folder name is made unique via
// os.MkdirTemp, since two runs completing within the same wall-clock second
// would otherwise collide and silently overwrite each other.
func WriteRun(baseDir string, result *pipeline.RunResult) (string, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("create base dir: %w", err)
	}
	runDir, err := os.MkdirTemp(baseDir, time.Now().Format("20060102-150405")+"-*")
	if err != nil {
		return "", fmt.Errorf("create run dir: %w", err)
	}
	if err := os.Mkdir(filepath.Join(runDir, "resumes"), 0o755); err != nil {
		return "", fmt.Errorf("create resumes dir: %w", err)
	}

	for i := range result.Candidates {
		c := &result.Candidates[i]
		rf, ok := result.Resumes[c.ID]
		if !ok {
			continue
		}
		ext := rf.Ext
		if ext == "" {
			ext = "bin"
		}
		relPath := filepath.Join("resumes", c.ID+"."+ext)
		if err := os.WriteFile(filepath.Join(runDir, relPath), rf.Data, 0o644); err != nil {
			return "", fmt.Errorf("write resume for %s: %w", c.ID, err)
		}
		c.ResumePath = relPath
	}

	payload := candidatesDocument{Candidates: result.Candidates}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal candidates: %w", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "candidates.json"), data, 0o644); err != nil {
		return "", fmt.Errorf("write candidates.json: %w", err)
	}

	return runDir, nil
}
