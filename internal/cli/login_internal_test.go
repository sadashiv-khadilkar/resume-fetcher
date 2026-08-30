package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/platform/naukriclient"
)

func TestBuildLoginClient(t *testing.T) {
	var out bytes.Buffer

	client, source, err := buildLoginClient("naukri", &out)
	if err != nil {
		t.Fatalf("buildLoginClient(naukri) error = %v", err)
	}
	if source != domain.SourceNaukri {
		t.Errorf("got source %q, want %q", source, domain.SourceNaukri)
	}
	if _, ok := client.(*naukriclient.Client); !ok {
		t.Errorf("got %T, want *naukriclient.Client", client)
	}

	if _, _, err := buildLoginClient("linkedin", &out); err == nil {
		t.Error("expected an error for linkedin (not implemented until ticket 06)")
	}

	if _, _, err := buildLoginClient("bogus", &out); err == nil {
		t.Error("expected an error for an invalid platform name")
	}
}

// fakeClient is a minimal platform.Client double for testing runLogin's
// reporting/error-wrapping in isolation from any real client construction.
type fakeClient struct {
	loginErr error
}

func (f *fakeClient) Login() (platform.SessionState, error) { return platform.SessionState{}, f.loginErr }
func (f *fakeClient) Search(domain.Filters) ([]domain.Profile, error) {
	return nil, errors.New("not used")
}
func (f *fakeClient) DownloadResume(string) (domain.ResumeFile, error) {
	return domain.ResumeFile{}, errors.New("not used")
}

func TestRunLogin_Success(t *testing.T) {
	var out bytes.Buffer
	err := runLogin(&fakeClient{}, domain.SourceNaukri, &out)
	if err != nil {
		t.Fatalf("runLogin() error = %v", err)
	}
	if !strings.Contains(out.String(), "naukri session is ready") {
		t.Errorf("expected a ready message, got %q", out.String())
	}
}

func TestRunLogin_PropagatesLoginError(t *testing.T) {
	var out bytes.Buffer
	err := runLogin(&fakeClient{loginErr: errors.New("browser closed")}, domain.SourceNaukri, &out)
	if err == nil {
		t.Fatal("expected runLogin to return the underlying Login error")
	}
}
