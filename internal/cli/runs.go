package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"resumefetcher/internal/output"
)

// runFolderTimestampLayout matches the prefix output.WriteRun gives each run
// folder (time.Now().Format("20060102-150405") + "-" + a random suffix from
// os.MkdirTemp), used as a fallback date/time source for run folders written
// before run.json existed.
const runFolderTimestampLayout = "20060102-150405"

// jdSummaryMaxRunes bounds how much of the JD is shown per run in the
// listing, so one long JD doesn't blow out the table.
const jdSummaryMaxRunes = 60

// runInfo is what `runs` shows for one Fetch Run, derived entirely from
// files already on disk in that run's output folder (candidates.json and,
// when present, run.json) - no separate run-tracking store.
type runInfo struct {
	Folder         string
	When           time.Time
	CandidateCount int
	JDSummary      string
}

// RunRuns implements `runs`: it lists past Fetch Run output folders under
// --out (most recent first), showing each run's JD, date/time, and
// Candidate count, reading only from the existing per-run output folder
// structure established in ticket 01.
func RunRuns(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("runs", flag.ContinueOnError)
	fs.SetOutput(out)
	outDir := fs.String("out", "output", "base directory for Fetch Run output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	runs, err := listRuns(*outDir)
	if err != nil {
		return err
	}

	if len(runs) == 0 {
		fmt.Fprintf(out, "No Fetch Runs found under %q.\n", *outDir)
		return nil
	}

	sort.SliceStable(runs, func(i, j int) bool { return runs[i].When.After(runs[j].When) })

	fmt.Fprintf(out, "%-20s %-24s %-10s %s\n", "DATE/TIME", "RUN FOLDER", "CANDIDATES", "JD")
	for _, r := range runs {
		fmt.Fprintf(out, "%-20s %-24s %-10d %s\n", r.When.Format("2006-01-02 15:04:05"), r.Folder, r.CandidateCount, r.JDSummary)
	}
	return nil
}

// listRuns scans baseDir for per-run output folders (one per past Fetch
// Run, per ticket 01) and derives a runInfo for each valid one. A missing
// baseDir is treated as "no runs yet" rather than an error.
func listRuns(baseDir string) ([]runInfo, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %q: %w", baseDir, err)
	}

	var runs []runInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, ok := loadRunInfo(filepath.Join(baseDir, e.Name()), e.Name())
		if ok {
			runs = append(runs, info)
		}
	}
	return runs, nil
}

// loadRunInfo reads one run folder's candidates.json (required - it's how
// we know dir is actually a run folder and not something else) and its
// run.json (optional, only present for runs written since ticket 09 added
// output.WriteRunMeta), returning ok=false for dirs that aren't run folders.
func loadRunInfo(dir, folderName string) (runInfo, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, "candidates.json"))
	if err != nil {
		return runInfo{}, false
	}
	var doc struct {
		Candidates []json.RawMessage `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return runInfo{}, false
	}

	info := runInfo{
		Folder:         folderName,
		CandidateCount: len(doc.Candidates),
		JDSummary:      "(unknown - run predates run.json)",
		When:           fallbackRunTime(dir, folderName),
	}

	if metaRaw, err := os.ReadFile(filepath.Join(dir, output.RunMetaFileName)); err == nil {
		var meta output.RunMeta
		if err := json.Unmarshal(metaRaw, &meta); err == nil {
			info.When = meta.StartedAt
			info.JDSummary = summarizeJD(meta.JD)
		}
	}

	return info, true
}

// fallbackRunTime derives a run's date/time from its folder name's
// timestamp prefix (the format output.WriteRun's os.MkdirTemp pattern
// produces), falling back to the folder's mtime if that prefix can't be
// parsed. Only used for run folders written before run.json existed.
func fallbackRunTime(dir, folderName string) time.Time {
	if len(folderName) >= len(runFolderTimestampLayout) {
		if t, err := time.ParseInLocation(runFolderTimestampLayout, folderName[:len(runFolderTimestampLayout)], time.Local); err == nil {
			return t
		}
	}
	if fi, err := os.Stat(dir); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}

// summarizeJD reduces a JD's full Markdown text down to its first
// non-blank line (with any leading Markdown heading markers stripped),
// truncated so the `runs` listing stays one line per run.
func summarizeJD(jd string) string {
	for _, line := range strings.Split(jd, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
		if line != "" {
			return truncateRunes(line, jdSummaryMaxRunes)
		}
	}
	return "(empty JD)"
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
