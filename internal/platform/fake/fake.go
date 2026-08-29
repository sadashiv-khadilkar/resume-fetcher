// Package fake provides an in-memory platform.Client for tests and for the
// walking-skeleton demo (no network or browser calls).
package fake

import (
	"fmt"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
)

type Client struct {
	Source   domain.Source
	Profiles []domain.Profile
	Resumes  map[string]domain.ResumeFile // keyed by Profile.ExternalID
}

func New(source domain.Source, profiles []domain.Profile) *Client {
	return &Client{Source: source, Profiles: profiles, Resumes: map[string]domain.ResumeFile{}}
}

func (c *Client) Login() (platform.SessionState, error) {
	return platform.SessionState{Data: []byte(fmt.Sprintf("fake-session-%s", c.Source))}, nil
}

func (c *Client) Search(_ domain.Filters) ([]domain.Profile, error) {
	return c.Profiles, nil
}

func (c *Client) DownloadResume(profileID string) (domain.ResumeFile, error) {
	if rf, ok := c.Resumes[profileID]; ok {
		return rf, nil
	}
	return domain.ResumeFile{Available: false}, nil
}
