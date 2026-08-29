// Package fake provides an in-memory llmprovider.Provider for tests and for
// the walking-skeleton demo (no real LLM calls).
package fake

import "resumefetcher/internal/domain"

type Provider struct {
	FiltersToReturn domain.Filters
	// ScoreFunc computes a Candidate's Match Score. Defaults to 0 for every
	// Candidate when nil.
	ScoreFunc func(domain.Candidate) float64
	// RankOverride, when set, replaces the default ScoreFunc-driven Rank
	// entirely — for tests exercising malformed/incomplete LLM output.
	RankOverride func(jd string, candidates []domain.Candidate) ([]domain.Match, error)
}

func (p *Provider) ExtractFilters(_ string) (domain.Filters, error) {
	return p.FiltersToReturn, nil
}

func (p *Provider) Rank(jd string, candidates []domain.Candidate) ([]domain.Match, error) {
	if p.RankOverride != nil {
		return p.RankOverride(jd, candidates)
	}
	matches := make([]domain.Match, len(candidates))
	for i, c := range candidates {
		var score float64
		if p.ScoreFunc != nil {
			score = p.ScoreFunc(c)
		}
		matches[i] = domain.Match{CandidateID: c.ID, Score: score}
	}
	return matches, nil
}
