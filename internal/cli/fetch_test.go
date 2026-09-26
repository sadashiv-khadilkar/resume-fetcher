package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resumefetcher/internal/cli"
)

// RunFetch's full end-to-end behavior (real search + merge/dedup + ranking +
// output) is no longer exercisable at this CLI layer without a real
// browser/account: ticket 04 wired Naukri to the real naukriclient, and
// ticket 07 wires LinkedIn to the real linkedinclient, so every --source
// value now reaches real PlatformClient automation. That behavior is
// verified manually against real accounts instead (see this repo's
// .scratch/resume-fetcher/issues/04, 05, 06, 07 tickets), while the
// orchestration logic itself (merge/dedup, ranking, capping, resume-download
// gating) stays covered by internal/pipeline's tests against fake
// PlatformClients, output writing by internal/output's tests, and this
// package's own flag-parsing/wiring decisions by TestParseSources and
// TestSelectPlatforms in fetch_internal_test.go. What remains testable here
// without touching real automation is argument validation that fails before
// any PlatformClient is ever used.

func TestRunFetch_RequiresJDFlag(t *testing.T) {
	var out bytes.Buffer
	err := cli.RunFetch([]string{"--out", t.TempDir()}, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("expected an error when --jd is missing")
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
