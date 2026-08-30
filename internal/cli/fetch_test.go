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

	// --source linkedin: LinkedIn still uses a fake PlatformClient (ticket
	// 07 not yet implemented). Naukri now uses the real naukriclient
	// (ticket 04), which needs a real browser/account and is verified
	// manually instead - see TestSelectPlatforms and TestParseSources in
	// fetch_internal_test.go for its wiring/parsing coverage.
	err := cli.RunFetch([]string{"--jd", jdPath, "--out", outDir, "--source", "linkedin"}, in, &out)
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

// TestRunFetch_SourceFlag only exercises --source values resolving to
// LinkedIn, which still uses a fake PlatformClient. naukri/both now route
// through the real naukriclient (ticket 04), which needs a real
// browser/account and is verified manually instead - --source parsing
// itself (including the naukri and both cases) is covered by
// TestParseSources in fetch_internal_test.go, and the real-vs-fake wiring
// decision by TestSelectPlatforms there.
func TestRunFetch_SourceFlag(t *testing.T) {
	jdPath := filepath.Join(t.TempDir(), "jd.md")
	if err := os.WriteFile(jdPath, []byte("# Backend Engineer\n\nGo, distributed systems."), 0o644); err != nil {
		t.Fatalf("write JD fixture: %v", err)
	}

	tests := []struct {
		name        string
		source      string
		wantSources []string // sources expected across the run's candidates
	}{
		{name: "linkedin only", source: "linkedin", wantSources: []string{"linkedin"}},
		{name: "case-insensitive", source: "LinkedIn", wantSources: []string{"linkedin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			var out bytes.Buffer
			in := strings.NewReader("\n")

			err := cli.RunFetch([]string{"--jd", jdPath, "--out", outDir, "--source", tt.source}, in, &out)
			if err != nil {
				t.Fatalf("RunFetch() error = %v", err)
			}

			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("read outDir: %v", err)
			}
			runDir := filepath.Join(outDir, entries[0].Name())
			raw, err := os.ReadFile(filepath.Join(runDir, "candidates.json"))
			if err != nil {
				t.Fatalf("candidates.json not found: %v", err)
			}
			var doc struct {
				Candidates []struct {
					Profiles []struct {
						Source string `json:"source"`
					} `json:"profiles"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("candidates.json invalid: %v", err)
			}
			if len(doc.Candidates) == 0 {
				t.Fatal("expected at least one candidate")
			}

			seen := map[string]bool{}
			for _, c := range doc.Candidates {
				for _, p := range c.Profiles {
					seen[p.Source] = true
				}
			}
			for _, want := range tt.wantSources {
				if !seen[want] {
					t.Errorf("expected a candidate from source %q, got sources %v", want, seen)
				}
			}
			if len(seen) != len(tt.wantSources) {
				t.Errorf("got sources %v, want exactly %v", seen, tt.wantSources)
			}
		})
	}
}

func TestRunFetch_RejectsInvalidSource(t *testing.T) {
	jdPath := filepath.Join(t.TempDir(), "jd.md")
	if err := os.WriteFile(jdPath, []byte("# Backend Engineer\n\nGo."), 0o644); err != nil {
		t.Fatalf("write JD fixture: %v", err)
	}

	var out bytes.Buffer
	err := cli.RunFetch([]string{"--jd", jdPath, "--out", t.TempDir(), "--source", "bogus"}, strings.NewReader("\n"), &out)
	if err == nil {
		t.Fatal("expected an error for an invalid --source value")
	}
}
