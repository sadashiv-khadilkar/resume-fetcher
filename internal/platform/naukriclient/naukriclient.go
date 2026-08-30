// Package naukriclient implements platform.Client for Naukri Resdex,
// driving a real browser via go-rod + go-rod/stealth (ADR-0002). Ticket 03
// implements Login and session persistence only; Search (ticket 04) and
// DownloadResume (ticket 05) land in later tickets and are stubbed out here.
//
// The browser interface below is the seam that keeps Login's
// reuse-vs-manual-login decision unit-testable: the real automation
// (rodBrowser, in rod.go) is verified manually against the operator's real
// Naukri account instead, per .scratch/resume-fetcher/spec.md's Testing
// Decisions.
package naukriclient

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
	// Naukri session.
	ValidateSession(state platform.SessionState) (bool, error)
	// ManualLogin opens a visible browser at Naukri's login page and waits
	// for the operator to complete login by hand (including any
	// 2FA/CAPTCHA), then returns the resulting session. It never submits
	// credentials itself (ADR-0003).
	ManualLogin() (platform.SessionState, error)
}

// Client is a real platform.Client for Naukri Resdex.
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

// Login reuses a still-valid persisted Naukri session. If none exists, or
// the persisted one no longer validates, it reports that clearly and falls
// back to a fresh manual login, persisting the result for next time.
func (c *Client) Login() (platform.SessionState, error) {
	state, err := c.store.Load(domain.SourceNaukri)
	switch {
	case err == nil:
		valid, verr := c.browser.ValidateSession(state)
		if verr != nil {
			return platform.SessionState{}, fmt.Errorf("validate naukri session: %w", verr)
		}
		if valid {
			fmt.Fprintln(c.out, "Naukri session is still valid; reusing it.")
			return state, nil
		}
		fmt.Fprintln(c.out, "Naukri session has expired or is no longer valid; starting a fresh login.")
	case errors.Is(err, session.ErrNotFound):
		fmt.Fprintln(c.out, "No Naukri session found; starting a fresh login.")
	default:
		return platform.SessionState{}, fmt.Errorf("load naukri session: %w", err)
	}

	fresh, err := c.browser.ManualLogin()
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("naukri manual login: %w", err)
	}
	if err := c.store.Save(domain.SourceNaukri, fresh); err != nil {
		return platform.SessionState{}, fmt.Errorf("persist naukri session: %w", err)
	}
	return fresh, nil
}

// Search is not implemented until ticket 04.
func (c *Client) Search(_ domain.Filters) ([]domain.Profile, error) {
	return nil, errors.New("naukriclient: Search not implemented until ticket 04")
}

// DownloadResume is not implemented until ticket 05.
func (c *Client) DownloadResume(_ string) (domain.ResumeFile, error) {
	return domain.ResumeFile{}, errors.New("naukriclient: DownloadResume not implemented until ticket 05")
}
