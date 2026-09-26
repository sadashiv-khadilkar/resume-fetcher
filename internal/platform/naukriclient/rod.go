package naukriclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
)

// loginURL is where a visible browser is pointed for manual Naukri login.
const loginURL = "https://login.naukri.com/nLogin/Login.php"

// loginHost is loginURL's host: comparing against it (rather than a bare
// substring match on the full URL) avoids misreading an authenticated
// page's URL as still-on-the-login-page just because it happens to contain
// "login" somewhere in a path segment or query string.
const loginHost = "login.naukri.com"

// dashboardURL is an authenticated Resdex page. Reaching it (rather than
// being bounced back to loginHost) is how ValidateSession tells a persisted
// session is still live.
const dashboardURL = "https://resdex.naukri.com/"

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
// interaction, so a stalled network or a dashboard page that never settles
// fails clearly instead of hanging login naukri indefinitely.
const validateSessionTimeout = 30 * time.Second

// resdexSearchURL is Resdex's native search UI. Search navigates here with
// the operator's session cookies applied and drives the on-page filter
// form. Like the rest of this file (see the package doc in naukriclient.go),
// this URL and the selector constants below are unverified against a live
// Resdex account in this sandbox (no browser/network access here) - confirm
// and adjust them against the real DOM before relying on this ticket's
// automation in production, per this ticket's own manual-verification
// checklist item.
const resdexSearchURL = "https://resdex.naukri.com/search"

// Resdex search-form and result-card selectors. Unverified against the real
// DOM - see resdexSearchURL's doc comment.
const (
	skillsInputSelector   = "#keywordSkillsFreeText"
	minExpInputSelector   = "#expMin"
	maxExpInputSelector   = "#expMax"
	locationInputSelector = "#location"
	noticePeriodSelector  = "#noticePeriod"
	searchButtonSelector  = "#searchButton"

	resultCardSelector           = ".resultCard"
	candidateNameSelector        = ".name"
	candidateEmailSelector       = ".email"
	candidatePhoneSelector       = ".phone"
	candidateCompanySelector     = ".company"
	candidateTitleSelector       = ".title"
	candidateSkillsSelector      = ".skills"
	candidateExperienceSelector  = ".experience"
	candidateEducationSelector   = ".education"
	candidateLocationSelector    = ".location"
	candidateProfileLinkSelector = "a.profile-link"

	downloadResumeButtonSelector = ".downloadResumeButton"
)

// downloadButtonProbeTimeout bounds how long DownloadResume waits to see
// whether a Resdex profile page shows a download-resume control at all,
// distinct from resumeDownloadTimeout's much longer bound on the actual
// download (including any challenge resolution) once a click is underway. A
// missing control isn't an error - it just means no resume is available for
// this candidate.
const downloadButtonProbeTimeout = 5 * time.Second

// resumeDownloadTimeout bounds DownloadResume's entire browser interaction,
// mirroring searchTimeout's role for Search: it has to accommodate a full
// challengeTimeout wait on top of ordinary navigation and the download
// itself.
const resumeDownloadTimeout = challengeTimeout + 2*time.Minute

// maxSearchResults defensively bounds how many result cards Search scrapes
// per call, matching pipeline.DefaultConfig's RawPoolCapPerPlatform so a
// single search never walks an unbounded DOM even before the pipeline
// applies its own cap on the returned slice.
const maxSearchResults = 50

// challengeTimeout bounds how long Search waits for the operator to resolve
// a CAPTCHA/2FA/rate-limit prompt encountered mid-search, mirroring
// manualLoginTimeout's role during ManualLogin.
const challengeTimeout = 5 * time.Minute

// searchTimeout bounds Search's entire browser interaction (unlike
// ValidateSession's short validateSessionTimeout, it has to accommodate a
// full challengeTimeout wait on top of ordinary navigation/fill/scrape
// steps), so a stalled page with no CAPTCHA/2FA/rate-limit marker present
// still fails clearly instead of hanging `fetch` indefinitely.
const searchTimeout = challengeTimeout + 2*time.Minute

// challengeMarkers are page-text substrings (case-insensitive) treated as
// evidence Naukri has interrupted the search with a CAPTCHA, 2FA, or
// rate-limit prompt requiring operator resolution. Best-effort and
// unverified against real Naukri copy - confirm/expand against the real
// site before relying on this in production.
var challengeMarkers = []string{
	"captcha",
	"verify you are human",
	"unusual traffic",
	"too many requests",
	"enter otp",
	"one time password",
}

// experienceYearsPattern extracts the leading integer from an experience
// string like "5 years" scraped off a result card.
var experienceYearsPattern = regexp.MustCompile(`\d+`)

// rodBrowser is the real, go-rod + go-rod/stealth-backed browser
// implementation. It is not unit tested (see naukriclient.go's package
// doc) - verify it manually against the operator's real Naukri account.
type rodBrowser struct {
	out io.Writer
}

func newRodBrowser(out io.Writer) *rodBrowser {
	return &rodBrowser{out: out}
}

// launchBrowser launches a local browser and connects rod to it, returning
// a cleanup func that closes the connection and the launched process. Both
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

// ManualLogin opens a visible, non-headless browser at loginURL and waits
// for the operator to complete login by hand; it never fills in or submits
// credentials itself (ADR-0003).
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
	if err := page.Navigate(loginURL); err != nil {
		return platform.SessionState{}, fmt.Errorf("navigate to login page: %w", err)
	}

	fmt.Fprintln(b.out, "Waiting for you to log in to Naukri in the opened browser window (including any 2FA/CAPTCHA)...")
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

// waitForManualLogin polls page's URL until it moves away from loginHost,
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
			if !isLoginURL(info.URL) {
				return nil
			}
		default:
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutivePollErrors {
				return fmt.Errorf("lost the browser window while waiting for manual Naukri login (is it still open?): %w", err)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for manual Naukri login", manualLoginTimeout)
		}
		time.Sleep(pollInterval)
	}
}

// ValidateSession loads state's cookies into a fresh headless browser and
// checks whether an authenticated Resdex page is reachable without being
// redirected back to loginHost.
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

	page, err := br.Page(proto.TargetCreateTarget{URL: dashboardURL})
	if err != nil {
		return false, fmt.Errorf("open dashboard page: %w", err)
	}
	defer page.Close()
	if err := page.WaitStable(pollInterval); err != nil {
		return false, fmt.Errorf("wait for dashboard page: %w", err)
	}

	info, err := page.Info()
	if err != nil {
		return false, fmt.Errorf("read page info: %w", err)
	}
	return !isLoginURL(info.URL), nil
}

// isLoginURL reports whether rawURL is (still) Naukri's login page, by host
// rather than a substring match that could misfire on a query string or
// path segment that happens to contain "login".
func isLoginURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true // can't confirm we've moved on, so don't claim success
	}
	return u.Host == loginHost
}

// Search loads state's cookies into a visible browser (visible, rather than
// ValidateSession's headless check, since the operator may need to resolve
// a CAPTCHA/2FA/rate-limit prompt mid-search), drives resdexSearchURL's
// native filter form for filters, and scrapes the result cards into
// Profiles.
func (b *rodBrowser) Search(state platform.SessionState, filters domain.Filters) ([]domain.Profile, error) {
	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(state.Data, &cookies); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}

	br, cleanup, err := launchBrowser(false)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := br.SetCookies(proto.CookiesToParams(cookies)); err != nil {
		return nil, fmt.Errorf("apply session cookies: %w", err)
	}
	br = br.Timeout(searchTimeout)

	page, err := stealth.Page(br)
	if err != nil {
		return nil, fmt.Errorf("open page: %w", err)
	}

	if err := page.Navigate(resdexSearchURL); err != nil {
		return nil, fmt.Errorf("navigate to Resdex search: %w", err)
	}
	if err := page.WaitStable(pollInterval); err != nil {
		return nil, fmt.Errorf("wait for Resdex search page: %w", err)
	}
	if err := requireLoggedIn(page); err != nil {
		return nil, err
	}
	if err := waitForChallengeResolution(b.out, page); err != nil {
		return nil, err
	}

	if err := fillSearchForm(page, filters); err != nil {
		return nil, fmt.Errorf("fill Resdex search filters: %w", err)
	}
	if err := clickSearch(page); err != nil {
		return nil, fmt.Errorf("run Resdex search: %w", err)
	}
	if err := waitForChallengeResolution(b.out, page); err != nil {
		return nil, err
	}

	profiles, err := scrapeResults(page)
	if err != nil {
		return nil, fmt.Errorf("scrape Resdex results: %w", err)
	}
	if len(profiles) == 0 {
		fmt.Fprintln(b.out, "Naukri Resdex search returned no result cards; if candidates were expected, the result-card selector may need updating against the real DOM.")
	}
	return profiles, nil
}

// DownloadResume loads state's cookies into a visible browser (visible, for
// the same CAPTCHA/2FA/rate-limit resolution reason as Search), navigates to
// profileID - the profile URL captured by Search's scrape (see
// profileFromCard) - and clicks Resdex's native resume-download control. If
// the profile page shows no such control (no resume on file, or the
// operator's Resdex credits are exhausted), it returns an unavailable
// ResumeFile rather than an error, per this ticket's checklist item about
// telling a missing file apart from a bug.
func (b *rodBrowser) DownloadResume(state platform.SessionState, profileID string) (domain.ResumeFile, error) {
	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(state.Data, &cookies); err != nil {
		return domain.ResumeFile{}, fmt.Errorf("decode session: %w", err)
	}

	br, cleanup, err := launchBrowser(false)
	if err != nil {
		return domain.ResumeFile{}, err
	}
	defer cleanup()

	if err := br.SetCookies(proto.CookiesToParams(cookies)); err != nil {
		return domain.ResumeFile{}, fmt.Errorf("apply session cookies: %w", err)
	}
	br = br.Timeout(resumeDownloadTimeout)

	page, err := stealth.Page(br)
	if err != nil {
		return domain.ResumeFile{}, fmt.Errorf("open page: %w", err)
	}

	if err := page.Navigate(profileID); err != nil {
		return domain.ResumeFile{}, fmt.Errorf("navigate to Resdex profile: %w", err)
	}
	if err := page.WaitStable(pollInterval); err != nil {
		return domain.ResumeFile{}, fmt.Errorf("wait for Resdex profile page: %w", err)
	}
	if err := requireLoggedIn(page); err != nil {
		return domain.ResumeFile{}, err
	}
	if err := waitForChallengeResolution(b.out, page); err != nil {
		return domain.ResumeFile{}, err
	}

	button, err := page.Timeout(downloadButtonProbeTimeout).Element(downloadResumeButtonSelector)
	if err != nil {
		fmt.Fprintln(b.out, "No resume-download control found on the Resdex profile page; treating this candidate's resume as unavailable.")
		return domain.ResumeFile{}, nil
	}

	dir, err := os.MkdirTemp("", "resumefetcher-naukri-resume-*")
	if err != nil {
		return domain.ResumeFile{}, fmt.Errorf("create download dir: %w", err)
	}
	defer os.RemoveAll(dir)

	wait := br.WaitDownload(dir)
	if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return domain.ResumeFile{}, fmt.Errorf("click resume-download control: %w", err)
	}
	if err := waitForChallengeResolution(b.out, page); err != nil {
		return domain.ResumeFile{}, err
	}

	downloaded := make(chan *proto.PageDownloadWillBegin, 1)
	go func() { downloaded <- wait() }()

	var info *proto.PageDownloadWillBegin
	select {
	case info = <-downloaded:
	case <-time.After(resumeDownloadTimeout):
		return domain.ResumeFile{}, fmt.Errorf("timed out after %s waiting for the Naukri resume download to complete", resumeDownloadTimeout)
	}
	if info == nil {
		return domain.ResumeFile{}, errors.New("naukri resume download did not start")
	}

	data, err := os.ReadFile(filepath.Join(dir, info.GUID))
	if err != nil {
		return domain.ResumeFile{}, fmt.Errorf("read downloaded resume: %w", err)
	}

	return domain.ResumeFile{
		Available: true,
		Data:      data,
		Ext:       strings.TrimPrefix(filepath.Ext(info.SuggestedFilename), "."),
	}, nil
}

// requireLoggedIn reports a clear "session invalid" error if page has been
// redirected to Naukri's login page, rather than letting the search proceed
// into fillSearchForm and fail on a confusing "element not found" once the
// applied session cookies turn out to be expired or single-use. This check
// runs in Search's own browser instance, separately from
// ensureSession/ValidateSession's earlier check in a different browser.
func requireLoggedIn(page *rod.Page) error {
	info, err := page.Info()
	if err != nil {
		return fmt.Errorf("read Resdex search page info: %w", err)
	}
	if isLoginURL(info.URL) {
		return errors.New("naukri session is invalid (redirected to login); run `login naukri` again")
	}
	return nil
}

// fillSearchForm enters filters into resdexSearchURL's native filter
// inputs, leaving a field untouched when filters has nothing to put there.
func fillSearchForm(page *rod.Page, filters domain.Filters) error {
	if len(filters.Skills) > 0 {
		if err := setInput(page, skillsInputSelector, strings.Join(filters.Skills, ", ")); err != nil {
			return err
		}
	}
	if filters.MinExperience > 0 {
		if err := setInput(page, minExpInputSelector, strconv.Itoa(filters.MinExperience)); err != nil {
			return err
		}
	}
	if filters.MaxExperience > 0 {
		if err := setInput(page, maxExpInputSelector, strconv.Itoa(filters.MaxExperience)); err != nil {
			return err
		}
	}
	if filters.Location != "" {
		if err := setInput(page, locationInputSelector, filters.Location); err != nil {
			return err
		}
	}
	if filters.NoticePeriod != "" {
		if err := setInput(page, noticePeriodSelector, filters.NoticePeriod); err != nil {
			return err
		}
	}
	return nil
}

func setInput(page *rod.Page, selector, value string) error {
	el, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("find %s: %w", selector, err)
	}
	if err := el.Input(value); err != nil {
		return fmt.Errorf("enter value into %s: %w", selector, err)
	}
	return nil
}

func clickSearch(page *rod.Page) error {
	el, err := page.Element(searchButtonSelector)
	if err != nil {
		return fmt.Errorf("find search button: %w", err)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click search button: %w", err)
	}
	return page.WaitStable(pollInterval)
}

// scrapeResults reads up to maxSearchResults result cards off page into
// Profiles.
func scrapeResults(page *rod.Page) ([]domain.Profile, error) {
	cards, err := page.Elements(resultCardSelector)
	if err != nil {
		return nil, fmt.Errorf("find result cards: %w", err)
	}
	if len(cards) > maxSearchResults {
		cards = cards[:maxSearchResults]
	}

	profiles := make([]domain.Profile, 0, len(cards))
	for i, card := range cards {
		profiles = append(profiles, profileFromCard(card, i))
	}
	return profiles, nil
}

// profileFromCard reads one result card's fields. Optional fields (email,
// phone, education, ...) that a given card doesn't show are simply left
// empty rather than failing the whole scrape.
//
// index disambiguates ExternalID when candidateProfileLinkSelector doesn't
// match: pipeline.MergeProfiles/candidateID key Candidates on Source plus
// ExternalID, so two profiles that both fell back to an empty ExternalID
// would otherwise collide onto the same Candidate and silently overwrite
// each other's downloaded resume.
func profileFromCard(card *rod.Element, index int) domain.Profile {
	profileURL := attrOf(card, candidateProfileLinkSelector, "href")
	externalID := profileURL
	if externalID == "" {
		externalID = fmt.Sprintf("unmatched-%d", index)
	}
	return domain.Profile{
		ExternalID: externalID,
		Source:     domain.SourceNaukri,
		Name:       textOf(card, candidateNameSelector),
		Email:      textOf(card, candidateEmailSelector),
		Phone:      textOf(card, candidatePhoneSelector),
		Company:    textOf(card, candidateCompanySelector),
		Title:      textOf(card, candidateTitleSelector),
		Skills:     splitList(textOf(card, candidateSkillsSelector)),
		Experience: parseExperienceYears(textOf(card, candidateExperienceSelector)),
		Location:   textOf(card, candidateLocationSelector),
		Education:  textOf(card, candidateEducationSelector),
		URL:        profileURL,
	}
}

// textOf returns the trimmed text of selector under el, or "" if el has no
// such child.
func textOf(el *rod.Element, selector string) string {
	child, err := el.Element(selector)
	if err != nil {
		return ""
	}
	text, err := child.Text()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// attrOf returns selector's attr attribute under el, or "" if el has no
// such child or the attribute is unset.
func attrOf(el *rod.Element, selector, attr string) string {
	child, err := el.Element(selector)
	if err != nil {
		return ""
	}
	val, err := child.Attribute(attr)
	if err != nil || val == nil {
		return ""
	}
	return *val
}

// splitList splits a comma-separated list scraped off a result card,
// dropping empty entries.
func splitList(text string) []string {
	if text == "" {
		return nil
	}
	var items []string
	for _, item := range strings.Split(text, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// parseExperienceYears extracts the leading integer from an experience
// string like "5 years", or 0 if none is found.
func parseExperienceYears(text string) int {
	match := experienceYearsPattern.FindString(text)
	if match == "" {
		return 0
	}
	years, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return years
}

// waitForChallengeResolution pauses the visible browser and waits for the
// operator to resolve a CAPTCHA/2FA/rate-limit prompt if page currently
// shows one, per this ticket's resilience checklist item; it returns
// immediately if no such prompt is present.
func waitForChallengeResolution(out io.Writer, page *rod.Page) error {
	present, err := pageShowsChallenge(page)
	if err != nil {
		return fmt.Errorf("check for a CAPTCHA/2FA/rate-limit prompt: %w", err)
	}
	if !present {
		return nil
	}

	fmt.Fprintln(out, "Naukri is showing a CAPTCHA/2FA/rate-limit prompt; resolve it in the opened browser window to continue...")
	deadline := time.Now().Add(challengeTimeout)
	for {
		time.Sleep(pollInterval)
		present, err := pageShowsChallenge(page)
		if err != nil {
			return fmt.Errorf("check for a CAPTCHA/2FA/rate-limit prompt: %w", err)
		}
		if !present {
			fmt.Fprintln(out, "Challenge resolved; continuing the search.")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for the operator to resolve a Naukri CAPTCHA/2FA/rate-limit prompt", challengeTimeout)
		}
	}
}

// pageShowsChallenge reports whether page looks like a CAPTCHA/2FA/
// rate-limit prompt rather than a normal search page: a challengeMarkers
// substring in the page's HTML AND no result cards present. Requiring the
// absence of result cards keeps a real candidate's own skills or bio text
// (which can legitimately contain a marker word, e.g. "OTP verification"
// experience) from being misread as a challenge once results have loaded.
func pageShowsChallenge(page *rod.Page) (bool, error) {
	html, err := page.HTML()
	if err != nil {
		return false, fmt.Errorf("read page HTML: %w", err)
	}
	lower := strings.ToLower(html)
	matched := false
	for _, marker := range challengeMarkers {
		if strings.Contains(lower, marker) {
			matched = true
			break
		}
	}
	if !matched {
		return false, nil
	}

	cards, err := page.Elements(resultCardSelector)
	if err != nil {
		return false, fmt.Errorf("check for result cards: %w", err)
	}
	return len(cards) == 0, nil
}
