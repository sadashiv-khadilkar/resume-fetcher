package linkedinclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"

	"resumefetcher/internal/platform"
)

// recruiterURL is LinkedIn Recruiter's own entry point. Navigating here
// unauthenticated redirects to LinkedIn's login/authwall flow; navigating
// here with a valid session lands (and stays) on recruiterHost. Using the
// same URL for both ManualLogin's starting point and ValidateSession's check
// means "did we end up on Recruiter" is the one signal both need - unverified
// against the real DOM in this sandbox (no browser/network access here),
// same caveat as naukriclient/rod.go.
const recruiterURL = "https://recruiter.linkedin.com/"

// recruiterHost is recruiterURL's host. Reaching it (rather than being
// redirected to LinkedIn's general login/authwall pages under a different
// host or path) is how both ManualLogin and ValidateSession tell a session
// is authenticated for Recruiter specifically.
const recruiterHost = "recruiter.linkedin.com"

// manualLoginTimeout bounds how long ManualLogin waits for the operator to
// finish logging in (including any 2FA/CAPTCHA) by hand.
const manualLoginTimeout = 5 * time.Minute

// pollInterval is how often ManualLogin checks whether login has completed.
const pollInterval = 2 * time.Second

// maxConsecutivePollErrors bounds how many consecutive failed page.Info()
// polls ManualLogin tolerates before concluding the browser was closed or
// crashed, rather than silently polling for the full manualLoginTimeout.
const maxConsecutivePollErrors = 3

// validateSessionTimeout bounds the entire ValidateSession browser
// interaction, so a stalled network or a Recruiter page that never settles
// fails clearly instead of hanging `login linkedin` indefinitely.
const validateSessionTimeout = 30 * time.Second

// rodBrowser is the real, go-rod + go-rod/stealth-backed browser
// implementation. It is not unit tested (see linkedinclient.go's package
// doc) - verify it manually against the operator's real LinkedIn Recruiter
// account.
type rodBrowser struct {
	out io.Writer
}

func newRodBrowser(out io.Writer) *rodBrowser {
	return &rodBrowser{out: out}
}

// launchBrowser launches a local browser and connects rod to it, returning a
// cleanup func that closes the connection and the launched process. Both
// ManualLogin and ValidateSession share this rather than each duplicating
// the launcher/connect/cleanup sequence.
func launchBrowser(headless bool) (*rod.Browser, func(), error) {
	l := launcher.New().Headless(headless)
	controlURL, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("launch browser: %w", err)
	}

	br := rod.New().ControlURL(controlURL)
	if err := br.Connect(); err != nil {
		l.Cleanup()
		return nil, nil, fmt.Errorf("connect to browser: %w", err)
	}

	cleanup := func() {
		br.Close()
		l.Cleanup()
	}
	return br, cleanup, nil
}

// ManualLogin opens a visible, non-headless browser at recruiterURL and
// waits for the operator to complete login by hand; it never fills in or
// submits credentials itself (ADR-0003).
func (b *rodBrowser) ManualLogin() (platform.SessionState, error) {
	br, cleanup, err := launchBrowser(false)
	if err != nil {
		return platform.SessionState{}, err
	}
	defer cleanup()

	page, err := stealth.Page(br)
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("open page: %w", err)
	}
	if err := page.Navigate(recruiterURL); err != nil {
		return platform.SessionState{}, fmt.Errorf("navigate to LinkedIn Recruiter: %w", err)
	}

	fmt.Fprintln(b.out, "Waiting for you to log in to LinkedIn Recruiter in the opened browser window (including any 2FA/CAPTCHA)...")
	if err := waitForManualLogin(page); err != nil {
		return platform.SessionState{}, err
	}

	cookies, err := br.GetCookies()
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("capture session cookies: %w", err)
	}
	data, err := json.Marshal(cookies)
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("encode session: %w", err)
	}
	return platform.SessionState{Data: data}, nil
}

// waitForManualLogin polls page's URL until it reaches recruiterHost,
// treating that as a signal the operator finished logging in. It fails
// fast, rather than polling for the full manualLoginTimeout, once
// maxConsecutivePollErrors consecutive polls can't even read the page's
// URL - the operator having closed or crashed the browser window.
func waitForManualLogin(page *rod.Page) error {
	deadline := time.Now().Add(manualLoginTimeout)
	consecutiveErrors := 0
	for {
		info, err := page.Info()
		switch {
		case err == nil:
			consecutiveErrors = 0
			if isRecruiterURL(info.URL) {
				return nil
			}
		default:
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutivePollErrors {
				return fmt.Errorf("lost the browser window while waiting for manual LinkedIn login (is it still open?): %w", err)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for manual LinkedIn login", manualLoginTimeout)
		}
		time.Sleep(pollInterval)
	}
}

// ValidateSession loads state's cookies into a fresh headless browser and
// checks whether recruiterURL is reachable without being redirected away
// from recruiterHost.
func (b *rodBrowser) ValidateSession(state platform.SessionState) (bool, error) {
	if len(state.Data) == 0 {
		return false, nil
	}
	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(state.Data, &cookies); err != nil {
		return false, fmt.Errorf("decode session: %w", err)
	}

	br, cleanup, err := launchBrowser(true)
	if err != nil {
		return false, err
	}
	defer cleanup()
	br = br.Timeout(validateSessionTimeout)

	if err := br.SetCookies(proto.CookiesToParams(cookies)); err != nil {
		return false, fmt.Errorf("apply session cookies: %w", err)
	}

	page, err := br.Page(proto.TargetCreateTarget{URL: recruiterURL})
	if err != nil {
		return false, fmt.Errorf("open LinkedIn Recruiter page: %w", err)
	}
	defer page.Close()
	if err := page.WaitStable(pollInterval); err != nil {
		return false, fmt.Errorf("wait for LinkedIn Recruiter page: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return false, fmt.Errorf("read page info: %w", err)
	}
	return isRecruiterURL(info.URL), nil
}

// isRecruiterURL reports whether rawURL is on recruiterHost, by host rather
// than a substring match that could misfire on a query string or path
// segment.
func isRecruiterURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false // can't confirm we've arrived, so don't claim success
	}
	return u.Host == recruiterHost
}
