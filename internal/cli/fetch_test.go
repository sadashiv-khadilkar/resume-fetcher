package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resumefetcher/internal/cli"
)

func TestRunFetch_ProducesARunFolderAndSummary(t *testing.T) {
	jdPath := filepath.Join(t.TempDir(), "jd.md")
	if err := os.WriteFile(jdPath, []byte("# Backend Engineer\n\nGo, distributed systems."), 0o644); err != nil {
		t.Fatalf("write JD fixture: %v", err)
	}
	outDir := t.TempDir()

	var out bytes.Buffer
	in := strings.NewReader("\n") // accept extracted filters as-is

	err := cli.RunFetch([]string{"--jd", jdPath, "--out", outDir}, in, &out)
	if err != nil {
		t.Fatalf("RunFetch() error = %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries in outDir, want 1 run folder", len(entries))
	}

	runDir := filepath.Join(outDir, entries[0].Name())
	raw, err := os.ReadFile(filepath.Join(runDir, "candidates.json"))
	if err != nil {
		t.Fatalf("candidates.json not found: %v", err)
	}
	var doc struct {
		Candidates []struct {
			ID string `json:"id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("candidates.json invalid: %v", err)
	}
	if len(doc.Candidates) == 0 {
		t.Error("expected at least one candidate from the fake platform data")
	}

	if !strings.Contains(out.String(), "Fetch Run complete") {
		t.Errorf("expected a summary to be printed, got:\n%s", out.String())
	}
}

func TestRunFetch_RequiresJDFlag(t *testing.T) {
	var out bytes.Buffer
	err := cli.RunFetch([]string{"--out", t.TempDir()}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("expected an error when --jd is missing")
	}
}
