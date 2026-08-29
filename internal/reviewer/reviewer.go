// Package reviewer defines the operator confirm/edit step for extracted
// search Filters, before any platform search runs.
package reviewer

import "resumefetcher/internal/domain"

type Reviewer interface {
	Review(filters domain.Filters) (domain.Filters, error)
}

// AutoAccept accepts extracted Filters unchanged, for tests and any
// non-interactive use.
type AutoAccept struct{}

func (AutoAccept) Review(f domain.Filters) (domain.Filters, error) {
	return f, nil
}
