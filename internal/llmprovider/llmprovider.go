// Package llmprovider defines the LLMProvider seam: turning a JD into search
// Filters, and ranking Candidates against a JD into a Match Score.
package llmprovider

import "resumefetcher/internal/domain"

type Provider interface {
	ExtractFilters(jd string) (domain.Filters, error)
	Rank(jd string, candidates []domain.Candidate) ([]domain.Match, error)
}
