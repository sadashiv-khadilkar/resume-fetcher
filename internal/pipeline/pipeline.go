// Package pipeline is the orchestrator: it wires the PlatformClient and
// LLMProvider seams together into a Fetch Run, per CONTEXT.md and
// docs/adr/0001-0003.
package pipeline

import (
	"fmt"
	"sort"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/llmprovider"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/reviewer"
)

// Config bounds a Fetch Run per .scratch/resume-fetcher/spec.md's scale
// decisions. Zero is not a safe default for any of these fields (it means
// "unlimited" for the two pool caps but "download nothing" for
// ResumeDownloadTopN) — always start from DefaultConfig rather than a bare
// Config{}.
type Config struct {
	RawPoolCapPerPlatform int
	MaxCandidates         int
	ResumeDownloadTopN    int
}

func DefaultConfig() Config {
	return Config{RawPoolCapPerPlatform: 50, MaxCandidates: 50, ResumeDownloadTopN: 10}
}

// PlatformEntry pairs a Source with its Client. A slice (not a map) keeps
// platform iteration order deterministic.
type PlatformEntry struct {
	Source domain.Source
	Client platform.Client
}

type Orchestrator struct {
	Platforms []PlatformEntry
	LLM       llmprovider.Provider
	Reviewer  reviewer.Reviewer
	Config    Config
}

// RunResult is one Fetch Run's output: the final ranked Candidates, and any
// Resume bytes downloaded for the top-ranked ones (keyed by Candidate.ID).
type RunResult struct {
	Candidates []domain.Candidate
	Resumes    map[string]domain.ResumeFile
}

func (o *Orchestrator) Run(jd string) (*RunResult, error) {
	filters, err := o.LLM.ExtractFilters(jd)
	if err != nil {
		return nil, fmt.Errorf("extract filters: %w", err)
	}

	filters, err = o.Reviewer.Review(filters)
	if err != nil {
		return nil, fmt.Errorf("review filters: %w", err)
	}

	var allProfiles []domain.Profile
	for _, entry := range o.Platforms {
		results, err := entry.Client.Search(filters)
		if err != nil {
			return nil, fmt.Errorf("search %s: %w", entry.Source, err)
		}
		if cap := o.Config.RawPoolCapPerPlatform; cap > 0 && len(results) > cap {
			results = results[:cap]
		}
		allProfiles = append(allProfiles, results...)
	}

	candidates := MergeProfiles(allProfiles)

	matches, err := o.LLM.Rank(jd, candidates)
	if err != nil {
		return nil, fmt.Errorf("rank: %w", err)
	}
	scoreByID := make(map[string]float64, len(matches))
	for _, m := range matches {
		scoreByID[m.CandidateID] = m.Score
	}
	for i := range candidates {
		score, ok := scoreByID[candidates[i].ID]
		if !ok {
			return nil, fmt.Errorf("rank: no Match Score returned for candidate %s", candidates[i].ID)
		}
		candidates[i].MatchScore = score
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].MatchScore > candidates[j].MatchScore
	})

	if max := o.Config.MaxCandidates; max > 0 && len(candidates) > max {
		candidates = candidates[:max]
	}

	clientBySource := make(map[domain.Source]platform.Client, len(o.Platforms))
	for _, entry := range o.Platforms {
		clientBySource[entry.Source] = entry.Client
	}

	resumes := make(map[string]domain.ResumeFile)
	topN := o.Config.ResumeDownloadTopN
	if topN > len(candidates) {
		topN = len(candidates)
	}
	for i := 0; i < topN; i++ {
		c := &candidates[i]
		primary := c.Profiles[0]
		client, ok := clientBySource[primary.Source]
		if !ok {
			continue
		}
		rf, err := client.DownloadResume(primary.ExternalID)
		if err != nil {
			return nil, fmt.Errorf("download resume for %s: %w", c.ID, err)
		}
		if rf.Available {
			resumes[c.ID] = rf
		} else {
			c.ResumeNote = "resume not available"
		}
	}

	return &RunResult{Candidates: candidates, Resumes: resumes}, nil
}
