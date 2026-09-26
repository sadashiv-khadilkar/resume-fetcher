// Package linkedinclient implements platform.Client for LinkedIn Recruiter,
// driving a real browser via go-rod + go-rod/stealth (ADR-0002). Ticket 06
// implements Login and session persistence; Search and DownloadResume are
// implemented by tickets 07/08.
//
// The browser interface below is the seam that keeps the session
// reuse-vs-manual-login decision unit-testable: the real automation
// (rodBrowser, in rod.go) is verified manually against the operator's real
// LinkedIn Recruiter account instead, per .scratch/resume-fetcher/spec.md's
// Testing Decisions.
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

// Search is not implemented until ticket 07.
func (c *Client) Search(domain.Filters) ([]domain.Profile, error) {
	return nil, errors.New("linkedin search is not implemented yet (ticket 07)")
}

// DownloadResume is not implemented until ticket 08.
func (c *Client) DownloadResume(string) (domain.ResumeFile, error) {
	return domain.ResumeFile{}, errors.New("linkedin resume download is not implemented yet (ticket 08)")
}
