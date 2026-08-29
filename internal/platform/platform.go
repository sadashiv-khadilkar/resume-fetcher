// Package platform defines the PlatformClient seam: everything the pipeline
// needs from a source site (Naukri, LinkedIn), with all browser-automation
// detail hidden behind it. See ADR-0001 and ADR-0002 for why that automation
// exists and how it's built; nothing above this interface needs to know.
package platform

import "resumefetcher/internal/domain"

// SessionState is an opaque, platform-specific persisted login session.
type SessionState struct {
	Data []byte
}

// Client is the PlatformClient seam.
type Client interface {
	// Login captures a session via a visible, manual login (ADR-0003).
	Login() (SessionState, error)
	// Search runs filters as a native search and returns the raw results.
	Search(filters domain.Filters) ([]domain.Profile, error)
	// DownloadResume attempts to obtain a Resume for one profile.
	DownloadResume(profileID string) (domain.ResumeFile, error)
}
