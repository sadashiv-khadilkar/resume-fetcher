package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"resumefetcher/internal/cli"
	"resumefetcher/internal/output"
)

// writeFixtureRun creates a fixture per-run output folder shaped like one
// output.WriteRun (+ output.WriteRunMeta) would produce, without running
// any real Fetch Run - candidates.json with candidateCount entries, plus
// run.json when meta is non-nil.
func writeFixtureRun(t *testing.T, baseDir, folder string, candidateCount int, meta *output.RunMeta) string {
	t.Helper()
	runDir := filepath.Join(baseDir, folder)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture run dir: %v", err)
	}

	candidates := make([]map[string]string, candidateCount)
	for i := range candidates {
		candidates[i] = map[string]string{"id": folder + "-cand-" + string(rune('a'+i))}
	}
	data, err := json.Marshal(map[string]any{"candidates": candidates})
	if err != nil {
		t.Fatalf("marshal fixture candidates.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "candidates.json"), data, 0o644); err != nil {
		t.Fatalf("write fixture candidates.json: %v", err)
	}

	if meta != nil {
		if err := output.WriteRunMeta(runDir, *meta); err != nil {
			t.Fatalf("write fixture run.json: %v", err)
		}
	}
	return runDir
}

func TestRunRuns_ListsPastRunsMostRecentFirst(t *testing.T) {
	baseDir := t.TempDir()

	older := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)

	writeFixtureRun(t, baseDir, "20260101-090000-aaa", 2, &output.RunMeta{
		JD:        "# Backend Engineer\n\nGo, gRPC, distributed systems.",
		StartedAt: older,
	})
	writeFixtureRun(t, baseDir, "20260315-143000-bbb", 5, &output.RunMeta{
		JD:        "# Data Scientist\n\nPython, ML.",
		StartedAt: newer,
	})

	var out bytes.Buffer
	if err := cli.RunRuns([]string{"--out", baseDir}, &out); err != nil {
		t.Fatalf("RunRuns() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 3 { // header + 2 runs
		t.Fatalf("got %d lines, want 3 (header + 2 runs):\n%s", len(lines), out.String())
	}

	if !strings.Contains(lines[1], "Data Scientist") {
		t.Errorf("expected the most recent run (Data Scientist) listed first, got:\n%s", out.String())
	}
	if !strings.Contains(lines[1], "5") {
		t.Errorf("expected the Data Scientist run's candidate count (5) in its line, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "Backend Engineer") {
		t.Errorf("expected the older run (Backend Engineer) listed second, got:\n%s", out.String())
	}
	if !strings.Contains(lines[2], "2") {
		t.Errorf("expected the Backend Engineer run's candidate count (2) in its line, got %q", lines[2])
	}
}

func TestRunRuns_FallsBackToFolderTimestampWhenRunJSONMissing(t *testing.T) {
	baseDir := t.TempDir()

	// A run folder from before ticket 09 added run.json: only
	// candidates.json exists, matching exactly what ticket 01's
	// output.WriteRun alone produces.
	writeFixtureRun(t, baseDir, "20250601-120000-ccc", 3, nil)

	var out bytes.Buffer
	if err := cli.RunRuns([]string{"--out", baseDir}, &out); err != nil {
		t.Fatalf("RunRuns() error = %v", err)
	}

	if !strings.Contains(out.String(), "2025-06-01") {
		t.Errorf("expected the folder-name timestamp to be used as a fallback date, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "3") {
		t.Errorf("expected the candidate count (3) to still be read from candidates.json, got:\n%s", out.String())
	}
}

func TestRunRuns_NoRunsFound(t *testing.T) {
	var out bytes.Buffer
	if err := cli.RunRuns([]string{"--out", filepath.Join(t.TempDir(), "does-not-exist")}, &out); err != nil {
		t.Fatalf("RunRuns() error = %v", err)
	}
	if !strings.Contains(out.String(), "No Fetch Runs found") {
		t.Errorf("expected a no-runs-found message, got %q", out.String())
	}
}

func TestRunRuns_IgnoresNonRunFolders(t *testing.T) {
	baseDir := t.TempDir()
	// A directory with no candidates.json (e.g. a stray folder) must not
	// be mistaken for a run.
	if err := os.MkdirAll(filepath.Join(baseDir, "not-a-run"), 0o755); err != nil {
		t.Fatalf("mkdir stray dir: %v", err)
	}
	writeFixtureRun(t, baseDir, "20260101-090000-aaa", 1, &output.RunMeta{JD: "# JD", StartedAt: time.Now()})

	var out bytes.Buffer
	if err := cli.RunRuns([]string{"--out", baseDir}, &out); err != nil {
		t.Fatalf("RunRuns() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 { // header + 1 real run
		t.Fatalf("got %d lines, want 2 (header + 1 run), stray folder should be ignored:\n%s", len(lines), out.String())
	}
}
