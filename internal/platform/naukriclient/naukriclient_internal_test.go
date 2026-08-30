package naukriclient

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
	if err := store.Save(domain.SourceNaukri, want); err != nil {
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

	persisted, err := store.Load(domain.SourceNaukri)
	if err != nil {
		t.Fatalf("expected the fresh session to be persisted, Load() error = %v", err)
	}
	if string(persisted.Data) != "fresh-cookies" {
		t.Errorf("persisted session = %q, want %q", persisted.Data, "fresh-cookies")
	}
	if !strings.Contains(out.String(), "No Naukri session found") {
		t.Errorf("expected a message about no session being found, got %q", out.String())
	}
}

func TestLogin_ExpiredPersistedSession_FallsBackToManualLoginAndOverwrites(t *testing.T) {
	fb := &fakeBrowser{
		validateResult: false,
		manualState:    platform.SessionState{Data: []byte("fresh-cookies")},
	}
	client, store, out := newTestClient(t, fb)
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("stale-cookies")}); err != nil {
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

	persisted, err := store.Load(domain.SourceNaukri)
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
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("cookies")}); err != nil {
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

	if _, loadErr := store.Load(domain.SourceNaukri); !errors.Is(loadErr, session.ErrNotFound) {
		t.Errorf("expected no session to be persisted after a failed manual login, Load() error = %v", loadErr)
	}
}

func TestSearch_ReusesValidSessionAndDelegatesToBrowser(t *testing.T) {
	want := []domain.Profile{{ExternalID: "n1", Name: "Asha Rao"}}
	fb := &fakeBrowser{validateResult: true, searchProfiles: want}
	client, store, _ := newTestClient(t, fb)
	existing := platform.SessionState{Data: []byte("existing-cookies")}
	if err := store.Save(domain.SourceNaukri, existing); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	filters := domain.Filters{Skills: []string{"Go"}}
	got, err := client.Search(filters)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].ExternalID != "n1" {
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
		searchProfiles: []domain.Profile{{ExternalID: "n2"}},
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
	if len(got) != 1 || got[0].ExternalID != "n2" {
		t.Errorf("Search() = %+v, want the browser's results", got)
	}
	if _, err := store.Load(domain.SourceNaukri); err != nil {
		t.Errorf("expected the fresh session to be persisted, Load() error = %v", err)
	}
}

func TestSearch_PropagatesBrowserError(t *testing.T) {
	fb := &fakeBrowser{validateResult: true, searchErr: errors.New("resdex boom")}
	client, store, _ := newTestClient(t, fb)
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("cookies")}); err != nil {
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

func TestDownloadResume_NotYetImplemented(t *testing.T) {
	client, _, _ := newTestClient(t, &fakeBrowser{})

	if _, err := client.DownloadResume("some-id"); err == nil {
		t.Error("expected DownloadResume to return an error before ticket 05")
	}
}
