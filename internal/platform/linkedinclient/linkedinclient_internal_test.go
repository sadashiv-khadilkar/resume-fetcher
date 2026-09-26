package linkedinclient

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/session"
)

// fakeBrowser stands in for the real go-rod automation so Login's
// reuse/manual-login decision can be tested without a real browser.
type fakeBrowser struct {
	validateResult bool
	validateErr    error
	manualState    platform.SessionState
	manualErr      error
	manualCalled   bool

	searchProfiles []domain.Profile
	searchErr      error
	searchCalled   bool
	searchState    platform.SessionState
	searchFilters  domain.Filters

	downloadResult    domain.ResumeFile
	downloadErr       error
	downloadCalled    bool
	downloadState     platform.SessionState
	downloadProfileID string
}

func (f *fakeBrowser) ValidateSession(platform.SessionState) (bool, error) {
	return f.validateResult, f.validateErr
}

func (f *fakeBrowser) ManualLogin() (platform.SessionState, error) {
	f.manualCalled = true
	return f.manualState, f.manualErr
}

func (f *fakeBrowser) Search(state platform.SessionState, filters domain.Filters) ([]domain.Profile, error) {
	f.searchCalled = true
	f.searchState = state
	f.searchFilters = filters
	return f.searchProfiles, f.searchErr
}

func (f *fakeBrowser) DownloadResume(state platform.SessionState, profileID string) (domain.ResumeFile, error) {
	f.downloadCalled = true
	f.downloadState = state
	f.downloadProfileID = profileID
	return f.downloadResult, f.downloadErr
}

func newTestClient(t *testing.T, b browser) (*Client, *session.Store, *bytes.Buffer) {
	t.Helper()
	store := session.New(filepath.Join(t.TempDir(), "sessions"))
	var out bytes.Buffer
	return &Client{store: store, browser: b, out: &out}, store, &out
}

func TestLogin_ReusesValidPersistedSession(t *testing.T) {
	fb := &fakeBrowser{validateResult: true}
	client, store, out := newTestClient(t, fb)
	want := platform.SessionState{Data: []byte("existing-cookies")}
	if err := store.Save(domain.SourceLinkedIn, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := client.Login()
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if string(got.Data) != string(want.Data) {
		t.Errorf("Login() = %q, want the reused persisted session %q", got.Data, want.Data)
	}
	if fb.manualCalled {
		t.Error("Login() called ManualLogin despite a valid persisted session")
	}
	if !strings.Contains(out.String(), "reusing") {
		t.Errorf("expected a message about reusing the session, got %q", out.String())
	}
}

func TestLogin_NoPersistedSession_FallsBackToManualLogin(t *testing.T) {
	fb := &fakeBrowser{manualState: platform.SessionState{Data: []byte("fresh-cookies")}}
	client, store, out := newTestClient(t, fb)

	got, err := client.Login()
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !fb.manualCalled {
		t.Error("expected Login() to fall back to ManualLogin when no session is persisted")
	}
	if string(got.Data) != "fresh-cookies" {
		t.Errorf("Login() = %q, want the freshly captured session", got.Data)
	}

	persisted, err := store.Load(domain.SourceLinkedIn)
	if err != nil {
		t.Fatalf("expected the fresh session to be persisted, Load() error = %v", err)
	}
	if string(persisted.Data) != "fresh-cookies" {
		t.Errorf("persisted session = %q, want %q", persisted.Data, "fresh-cookies")
	}
	if !strings.Contains(out.String(), "No LinkedIn session found") {
		t.Errorf("expected a message about no session being found, got %q", out.String())
	}
}

func TestLogin_ExpiredPersistedSession_FallsBackToManualLoginAndOverwrites(t *testing.T) {
	fb := &fakeBrowser{
		validateResult: false,
		manualState:    platform.SessionState{Data: []byte("fresh-cookies")},
	}
	client, store, out := newTestClient(t, fb)
	if err := store.Save(domain.SourceLinkedIn, platform.SessionState{Data: []byte("stale-cookies")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := client.Login()
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !fb.manualCalled {
		t.Error("expected Login() to fall back to ManualLogin when the persisted session is invalid")
	}
	if string(got.Data) != "fresh-cookies" {
		t.Errorf("Login() = %q, want the freshly captured session", got.Data)
	}

	persisted, err := store.Load(domain.SourceLinkedIn)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(persisted.Data) != "fresh-cookies" {
		t.Errorf("persisted session = %q, want the stale one overwritten with %q", persisted.Data, "fresh-cookies")
	}
	if !strings.Contains(out.String(), "expired") {
		t.Errorf("expected a message about the session having expired, got %q", out.String())
	}
}

func TestLogin_ValidateSessionError_IsSurfacedWithoutFallingBack(t *testing.T) {
	fb := &fakeBrowser{validateErr: errors.New("boom")}
	client, store, _ := newTestClient(t, fb)
	if err := store.Save(domain.SourceLinkedIn, platform.SessionState{Data: []byte("cookies")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := client.Login()
	if err == nil {
		t.Fatal("expected Login() to surface the ValidateSession error")
	}
	if fb.manualCalled {
		t.Error("Login() should not fall back to ManualLogin when validation itself errored")
	}
}

func TestLogin_ManualLoginError_IsSurfacedAndNotPersisted(t *testing.T) {
	fb := &fakeBrowser{manualErr: errors.New("operator closed the browser")}
	client, store, _ := newTestClient(t, fb)

	_, err := client.Login()
	if err == nil {
		t.Fatal("expected Login() to surface the ManualLogin error")
	}

	if _, loadErr := store.Load(domain.SourceLinkedIn); !errors.Is(loadErr, session.ErrNotFound) {
		t.Errorf("expected no session to be persisted after a failed manual login, Load() error = %v", loadErr)
	}
}

func TestSearch_ReusesValidSessionAndDelegatesToBrowser(t *testing.T) {
	want := []domain.Profile{{ExternalID: "l1", Name: "Asha Rao"}}
	fb := &fakeBrowser{validateResult: true, searchProfiles: want}
	client, store, _ := newTestClient(t, fb)
	existing := platform.SessionState{Data: []byte("existing-cookies")}
	if err := store.Save(domain.SourceLinkedIn, existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	filters := domain.Filters{Skills: []string{"Go"}}
	got, err := client.Search(filters)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].ExternalID != "l1" {
		t.Errorf("Search() = %+v, want %+v", got, want)
	}
	if fb.manualCalled {
		t.Error("Search() called ManualLogin despite a valid persisted session")
	}
	if !fb.searchCalled {
		t.Fatal("expected Search() to delegate to browser.Search")
	}
	if string(fb.searchState.Data) != "existing-cookies" {
		t.Errorf("browser.Search got session %q, want the reused persisted session", fb.searchState.Data)
	}
	if len(fb.searchFilters.Skills) != 1 || fb.searchFilters.Skills[0] != "Go" {
		t.Errorf("browser.Search got filters %+v, want %+v", fb.searchFilters, filters)
	}
}

func TestSearch_NoPersistedSession_FallsBackToManualLoginThenSearches(t *testing.T) {
	fb := &fakeBrowser{
		manualState:    platform.SessionState{Data: []byte("fresh-cookies")},
		searchProfiles: []domain.Profile{{ExternalID: "l2"}},
	}
	client, store, _ := newTestClient(t, fb)

	got, err := client.Search(domain.Filters{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !fb.manualCalled {
		t.Error("expected Search() to fall back to ManualLogin when no session is persisted")
	}
	if string(fb.searchState.Data) != "fresh-cookies" {
		t.Errorf("browser.Search got session %q, want the freshly captured session", fb.searchState.Data)
	}
	if len(got) != 1 || got[0].ExternalID != "l2" {
		t.Errorf("Search() = %+v, want the browser's results", got)
	}
	if _, err := store.Load(domain.SourceLinkedIn); err != nil {
		t.Errorf("expected the fresh session to be persisted, Load() error = %v", err)
	}
}

func TestSearch_PropagatesBrowserError(t *testing.T) {
	fb := &fakeBrowser{validateResult: true, searchErr: errors.New("recruiter boom")}
	client, store, _ := newTestClient(t, fb)
	if err := store.Save(domain.SourceLinkedIn, platform.SessionState{Data: []byte("cookies")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := client.Search(domain.Filters{}); err == nil {
		t.Fatal("expected Search() to surface the browser's error")
	}
}

func TestSearch_ManualLoginFailure_IsSurfacedWithoutSearching(t *testing.T) {
	fb := &fakeBrowser{manualErr: errors.New("operator closed the browser")}
	client, _, _ := newTestClient(t, fb)

	if _, err := client.Search(domain.Filters{}); err == nil {
		t.Fatal("expected Search() to surface the ManualLogin error")
	}
	if fb.searchCalled {
		t.Error("Search() should not call browser.Search when ensuring the session failed")
	}
}

func TestDownloadResume_ReusesValidSessionAndDelegatesToBrowser(t *testing.T) {
	want := domain.ResumeFile{Available: true, Data: []byte("pdf-bytes"), Ext: "pdf"}
	fb := &fakeBrowser{validateResult: true, downloadResult: want}
	client, store, _ := newTestClient(t, fb)
	existing := platform.SessionState{Data: []byte("existing-cookies")}
	if err := store.Save(domain.SourceLinkedIn, existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := client.DownloadResume("profile-1")
	if err != nil {
		t.Fatalf("DownloadResume() error = %v", err)
	}
	if !got.Available || string(got.Data) != "pdf-bytes" || got.Ext != "pdf" {
		t.Errorf("DownloadResume() = %+v, want %+v", got, want)
	}
	if fb.manualCalled {
		t.Error("DownloadResume() called ManualLogin despite a valid persisted session")
	}
	if !fb.downloadCalled {
		t.Fatal("expected DownloadResume() to delegate to browser.DownloadResume")
	}
	if string(fb.downloadState.Data) != "existing-cookies" {
		t.Errorf("browser.DownloadResume got session %q, want the reused persisted session", fb.downloadState.Data)
	}
	if fb.downloadProfileID != "profile-1" {
		t.Errorf("browser.DownloadResume got profileID %q, want %q", fb.downloadProfileID, "profile-1")
	}
}

func TestDownloadResume_NoPersistedSession_FallsBackToManualLoginThenDownloads(t *testing.T) {
	fb := &fakeBrowser{
		manualState:    platform.SessionState{Data: []byte("fresh-cookies")},
		downloadResult: domain.ResumeFile{Available: true, Data: []byte("bytes")},
	}
	client, store, _ := newTestClient(t, fb)

	got, err := client.DownloadResume("profile-2")
	if err != nil {
		t.Fatalf("DownloadResume() error = %v", err)
	}
	if !fb.manualCalled {
		t.Error("expected DownloadResume() to fall back to ManualLogin when no session is persisted")
	}
	if string(fb.downloadState.Data) != "fresh-cookies" {
		t.Errorf("browser.DownloadResume got session %q, want the freshly captured session", fb.downloadState.Data)
	}
	if !got.Available {
		t.Errorf("DownloadResume() = %+v, want the browser's result", got)
	}
	if _, err := store.Load(domain.SourceLinkedIn); err != nil {
		t.Errorf("expected the fresh session to be persisted, Load() error = %v", err)
	}
}

func TestDownloadResume_PropagatesBrowserError(t *testing.T) {
	fb := &fakeBrowser{validateResult: true, downloadErr: errors.New("recruiter boom")}
	client, store, _ := newTestClient(t, fb)
	if err := store.Save(domain.SourceLinkedIn, platform.SessionState{Data: []byte("cookies")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := client.DownloadResume("profile-3"); err == nil {
		t.Fatal("expected DownloadResume() to surface the browser's error")
	}
}

func TestDownloadResume_ManualLoginFailure_IsSurfacedWithoutDownloading(t *testing.T) {
	fb := &fakeBrowser{manualErr: errors.New("operator closed the browser")}
	client, _, _ := newTestClient(t, fb)

	if _, err := client.DownloadResume("profile-4"); err == nil {
		t.Fatal("expected DownloadResume() to surface the ManualLogin error")
	}
	if fb.downloadCalled {
		t.Error("DownloadResume() should not call browser.DownloadResume when ensuring the session failed")
	}
}

func TestDownloadResume_UnavailableResume_IsNotAnError(t *testing.T) {
	fb := &fakeBrowser{validateResult: true, downloadResult: domain.ResumeFile{Available: false}}
	client, store, _ := newTestClient(t, fb)
	if err := store.Save(domain.SourceLinkedIn, platform.SessionState{Data: []byte("cookies")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := client.DownloadResume("profile-5")
	if err != nil {
		t.Fatalf("DownloadResume() error = %v, want no error for an unavailable resume", err)
	}
	if got.Available {
		t.Errorf("DownloadResume() = %+v, want Available = false", got)
	}
}
