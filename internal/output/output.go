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
