// Package linkedinclient implements platform.Client for LinkedIn Recruiter,
// driving a real browser via go-rod + go-rod/stealth (ADR-0002). Ticket 06
// implements Login and session persistence; ticket 07 implements Search;
// ticket 08 implements DownloadResume.
//
// The browser interface below is the seam that keeps the session
// reuse-vs-manual-login decision (shared by Login, Search and DownloadResume)
// unit-testable: the real automation (rodBrowser, in rod.go) is verified
// manually against the operator's real LinkedIn Recruiter account instead,
// per .scratch/resume-fetcher/spec.md's Testing Decisions.
package linkedinclient

import (
	"errors"
	"fmt"
	"io"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/session"
)

// browser is the go-rod-backed automation Login needs.
type browser interface {
	// ValidateSession reports whether state still represents a logged-in
	// LinkedIn Recruiter session.
	ValidateSession(state platform.SessionState) (bool, error)
	// ManualLogin opens a visible browser at LinkedIn's login page and waits
	// for the operator to complete login by hand (including any
	// 2FA/CAPTCHA), then returns the resulting session. It never submits
	// credentials itself (ADR-0003).
	ManualLogin() (platform.SessionState, error)
	// Search drives a native LinkedIn Recruiter search with state's session
	// applied, pausing the visible browser for the operator to resolve any
	// CAPTCHA/2FA/rate-limit prompt hit along the way, and returns the raw
	// result Profiles.
	Search(state platform.SessionState, filters domain.Filters) ([]domain.Profile, error)
	// DownloadResume drives a Recruiter profile-PDF export for profileID with
	// state's session applied, pausing the visible browser for the operator
	// to resolve any CAPTCHA/2FA/rate-limit prompt hit along the way, and
	// returns the exported file (or an unavailable ResumeFile if this
	// candidate's settings block export).
	DownloadResume(state platform.SessionState, profileID string) (domain.ResumeFile, error)
}

// Client is a real platform.Client for LinkedIn Recruiter.
type Client struct {
	store   *session.Store
	browser browser
	out     io.Writer
}

var _ platform.Client = (*Client)(nil)

// New builds a Client whose sessions persist via store; progress and
// reuse/expiry messages are written to out.
func New(store *session.Store, out io.Writer) *Client {
	return &Client{store: store, browser: newRodBrowser(out), out: out}
}

// Login reuses a still-valid persisted LinkedIn Recruiter session. If none
// exists, or the persisted one no longer validates, it reports that clearly
// and falls back to a fresh manual login, persisting the result for next
// time.
func (c *Client) Login() (platform.SessionState, error) {
	return c.ensureSession()
}

// ensureSession is the reuse-vs-manual-login decision Login relies on, and
// that Search/DownloadResume (tickets 07/08) will share exactly as their
// naukriclient counterparts do.
func (c *Client) ensureSession() (platform.SessionState, error) {
	state, err := c.store.Load(domain.SourceLinkedIn)
	switch {
	case err == nil:
		valid, verr := c.browser.ValidateSession(state)
		if verr != nil {
			return platform.SessionState{}, fmt.Errorf("validate linkedin session: %w", verr)
		}
		if valid {
			fmt.Fprintln(c.out, "LinkedIn session is still valid; reusing it.")
			return state, nil
		}
		fmt.Fprintln(c.out, "LinkedIn session has expired or is no longer valid; starting a fresh login.")
	case errors.Is(err, session.ErrNotFound):
		fmt.Fprintln(c.out, "No LinkedIn session found; starting a fresh login.")
	default:
		return platform.SessionState{}, fmt.Errorf("load linkedin session: %w", err)
	}

	fresh, err := c.browser.ManualLogin()
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("linkedin manual login: %w", err)
	}
	if err := c.store.Save(domain.SourceLinkedIn, fresh); err != nil {
		return platform.SessionState{}, fmt.Errorf("persist linkedin session: %w", err)
	}
	return fresh, nil
}

// Search ensures a valid LinkedIn Recruiter session - reusing a persisted
// one, or falling back to a fresh manual login exactly as Login does, so a
// bare `fetch` works without a separate `login linkedin` step first - then
// drives a real Recruiter search for filters.
func (c *Client) Search(filters domain.Filters) ([]domain.Profile, error) {
	state, err := c.ensureSession()
	if err != nil {
		return nil, fmt.Errorf("linkedin search: %w", err)
	}
	profiles, err := c.browser.Search(state, filters)
	if err != nil {
		return nil, fmt.Errorf("linkedin search: %w", err)
	}
	return profiles, nil
}

// DownloadResume ensures a valid LinkedIn Recruiter session exactly as Login
// and Search do - so a bare `fetch` works without a separate `login
// linkedin` step first - then drives a real Recruiter profile-PDF export for
// profileID. The pipeline (ticket 01) only calls this for its top-N ranked
// LinkedIn candidates, matching ticket 05's Naukri approach.
func (c *Client) DownloadResume(profileID string) (domain.ResumeFile, error) {
	state, err := c.ensureSession()
	if err != nil {
		return domain.ResumeFile{}, fmt.Errorf("linkedin download resume: %w", err)
	}
	rf, err := c.browser.DownloadResume(state, profileID)
	if err != nil {
		return domain.ResumeFile{}, fmt.Errorf("linkedin download resume: %w", err)
	}
	return rf, nil
}
