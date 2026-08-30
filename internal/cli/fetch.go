// Package cli wires the pipeline seams into runnable commands. The fetch
// command currently wires fake PlatformClients and a fake LLMProvider
// (ADR-0001-0003 govern what the real implementations will look like, added
// in later tickets); nothing here does real network or browser work yet.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/llmprovider"
	claudellm "resumefetcher/internal/llmprovider/claude"
	fakellm "resumefetcher/internal/llmprovider/fake"
	"resumefetcher/internal/output"
	"resumefetcher/internal/pipeline"
	fakeplatform "resumefetcher/internal/platform/fake"
	"resumefetcher/internal/reviewer"
)

func RunFetch(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(out)
	jdPath := fs.String("jd", "", "path to the JD Markdown file")
	outDir := fs.String("out", "output", "base directory for Fetch Run output")
	sourceFlag := fs.String("source", "both", "which Source(s) to search: naukri, linkedin, or both")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *jdPath == "" {
		return fmt.Errorf("--jd is required")
	}

	sources, err := parseSources(*sourceFlag)
	if err != nil {
		return err
	}

	jdBytes, err := os.ReadFile(*jdPath)
	if err != nil {
		return fmt.Errorf("read JD: %w", err)
	}

	llm, err := buildLLMProvider()
	if err != nil {
		return err
	}

	orch := &pipeline.Orchestrator{
		Platforms: selectPlatforms(sources),
		LLM:       llm,
		Reviewer:  reviewer.NewStdin(in, out),
		Config:    pipeline.DefaultConfig(),
	}

	result, err := orch.Run(string(jdBytes))
	if err != nil {
		return writePartialOnRankFailure(out, *outDir, err)
	}

	runDir, err := output.WriteRun(*outDir, result)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	printSummary(out, result.Candidates, runDir)
	return nil
}

func printSummary(out io.Writer, candidates []domain.Candidate, runDir string) {
	fmt.Fprintf(out, "\nFetch Run complete: %s\n", runDir)
	fmt.Fprintf(out, "%-24s %-8s %s\n", "NAME", "SCORE", "SOURCE")
	for _, c := range candidates {
		fmt.Fprintf(out, "%-24s %-8.1f %s\n", c.Profiles[0].Name, c.MatchScore, sourcesOf(c))
	}
}

func sourcesOf(c domain.Candidate) string {
	seen := map[domain.Source]bool{}
	s := ""
	for _, p := range c.Profiles {
		if seen[p.Source] {
			continue
		}
		seen[p.Source] = true
		if s != "" {
			s += "+"
		}
		s += string(p.Source)
	}
	return s
}

// parseSources maps the --source flag value to the Sources a Fetch Run
// should search, case-insensitively.
func parseSources(value string) ([]domain.Source, error) {
	switch strings.ToLower(value) {
	case "both":
		return []domain.Source{domain.SourceNaukri, domain.SourceLinkedIn}, nil
	case "naukri":
		return []domain.Source{domain.SourceNaukri}, nil
	case "linkedin":
		return []domain.Source{domain.SourceLinkedIn}, nil
	default:
		return nil, fmt.Errorf("invalid --source %q: must be naukri, linkedin, or both", value)
	}
}

// buildLLMProvider selects the LLMProvider via the LLM_PROVIDER env var
// (case-insensitive; defaults to the fake so existing runs/tests stay
// network-free unless the operator opts in). LLM_PROVIDER=claude reads
// ANTHROPIC_API_KEY (required) and ANTHROPIC_MODEL (optional override) from
// the environment via internal/llmprovider/claude.
func buildLLMProvider() (llmprovider.Provider, error) {
	value := os.Getenv("LLM_PROVIDER")
	switch strings.ToLower(value) {
	case "", "fake":
		return &fakellm.Provider{ScoreFunc: sampleScore}, nil
	case "claude":
		return claudellm.New(os.Getenv("ANTHROPIC_MODEL")), nil
	default:
		return nil, fmt.Errorf("invalid LLM_PROVIDER %q: must be fake or claude", value)
	}
}

// writePartialOnRankFailure persists the raw (unranked) candidate pool when
// err is a *pipeline.RankFailedError, so a Rank failure after retries are
// exhausted (ticket 02) doesn't lose already-fetched data. It always returns
// a non-nil error derived from err.
func writePartialOnRankFailure(out io.Writer, outDir string, err error) error {
	var rankErr *pipeline.RankFailedError
	if !errors.As(err, &rankErr) {
		return fmt.Errorf("run pipeline: %w", err)
	}

	runDir, writeErr := output.WriteRun(outDir, rankErr.RawResult)
	if writeErr != nil {
		return fmt.Errorf("run pipeline: %w; also failed to persist raw candidate pool: %v", err, writeErr)
	}
	fmt.Fprintf(out, "\nRanking failed after retries: %v\nRaw (unranked) candidate pool saved to %s\n", rankErr.Err, runDir)
	return fmt.Errorf("run pipeline: %w", err)
}

// selectPlatforms builds the PlatformEntry values for the requested Sources,
// always in Naukri-then-LinkedIn order regardless of flag order.
func selectPlatforms(sources []domain.Source) []pipeline.PlatformEntry {
	all := []pipeline.PlatformEntry{
		{Source: domain.SourceNaukri, Client: fakeplatform.New(domain.SourceNaukri, sampleNaukriProfiles())},
		{Source: domain.SourceLinkedIn, Client: fakeplatform.New(domain.SourceLinkedIn, sampleLinkedInProfiles())},
	}
	want := make(map[domain.Source]bool, len(sources))
	for _, s := range sources {
		want[s] = true
	}
	entries := make([]pipeline.PlatformEntry, 0, len(all))
	for _, e := range all {
		if want[e.Source] {
			entries = append(entries, e)
		}
	}
	return entries
}

// sampleNaukriProfiles and sampleLinkedInProfiles stand in for real search
// results until tickets 04 and 07 wire in real PlatformClients.
func sampleNaukriProfiles() []domain.Profile {
	return []domain.Profile{
		{ExternalID: "naukri-1", Source: domain.SourceNaukri, Name: "Asha Rao", Email: "asha.rao@example.com", Company: "Acme Corp", Title: "Backend Engineer", Skills: []string{"Go", "Postgres"}, Experience: 5, Location: "Bangalore"},
		{ExternalID: "naukri-2", Source: domain.SourceNaukri, Name: "Ravi Kumar", Email: "ravi.kumar@example.com", Company: "Globex Inc", Title: "SRE", Skills: []string{"Kubernetes", "Go"}, Experience: 7, Location: "Pune"},
	}
}

func sampleLinkedInProfiles() []domain.Profile {
	return []domain.Profile{
		{ExternalID: "linkedin-1", Source: domain.SourceLinkedIn, Name: "Asha Rao", Email: "asha.rao@example.com", Company: "Acme Corp", Title: "Backend Engineer", Skills: []string{"Go", "gRPC"}, Experience: 5, Location: "Bangalore"},
		{ExternalID: "linkedin-2", Source: domain.SourceLinkedIn, Name: "Meera Iyer", Company: "Initech", Title: "Platform Engineer", Skills: []string{"Go", "AWS"}, Experience: 4, Location: "Hyderabad"},
	}
}

// sampleScore is a placeholder ranking heuristic (more Skills = higher
// score), replaced by the real Claude-backed ranking in ticket 02.
func sampleScore(c domain.Candidate) float64 {
	return float64(len(c.Profiles[0].Skills)) * 10
}
