// Package cli wires the pipeline seams into runnable commands. The fetch
// command currently wires fake PlatformClients and a fake LLMProvider
// (ADR-0001-0003 govern what the real implementations will look like, added
// in later tickets); nothing here does real network or browser work yet.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"resumefetcher/internal/domain"
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *jdPath == "" {
		return fmt.Errorf("--jd is required")
	}

	jdBytes, err := os.ReadFile(*jdPath)
	if err != nil {
		return fmt.Errorf("read JD: %w", err)
	}

	orch := &pipeline.Orchestrator{
		Platforms: []pipeline.PlatformEntry{
			{Source: domain.SourceNaukri, Client: fakeplatform.New(domain.SourceNaukri, sampleNaukriProfiles())},
			{Source: domain.SourceLinkedIn, Client: fakeplatform.New(domain.SourceLinkedIn, sampleLinkedInProfiles())},
		},
		LLM:      &fakellm.Provider{ScoreFunc: sampleScore},
		Reviewer: reviewer.NewStdin(in, out),
		Config:   pipeline.DefaultConfig(),
	}

	result, err := orch.Run(string(jdBytes))
	if err != nil {
		return fmt.Errorf("run pipeline: %w", err)
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
