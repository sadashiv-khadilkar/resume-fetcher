package session_test

import (
	"errors"
	"path/filepath"
	"testing"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
	"resumefetcher/internal/session"
)

func TestStore_LoadWithoutSave_ReturnsErrNotFound(t *testing.T) {
	store := session.New(filepath.Join(t.TempDir(), "sessions"))

	_, err := store.Load(domain.SourceNaukri)
	if !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestStore_SaveThenLoad_RoundTrips(t *testing.T) {
	store := session.New(filepath.Join(t.TempDir(), "sessions"))
	want := platform.SessionState{Data: []byte("cookie-jar")}

	if err := store.Save(domain.SourceNaukri, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load(domain.SourceNaukri)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(got.Data) != string(want.Data) {
		t.Errorf("Load() = %q, want %q", got.Data, want.Data)
	}
}

func TestStore_SessionsAreScopedPerSource(t *testing.T) {
	store := session.New(filepath.Join(t.TempDir(), "sessions"))
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("naukri")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := store.Load(domain.SourceLinkedIn)
	if !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("Load(linkedin) error = %v, want ErrNotFound (naukri's session must not leak)", err)
	}
}

func TestStore_SaveOverwritesPreviousSession(t *testing.T) {
	store := session.New(filepath.Join(t.TempDir(), "sessions"))
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("old")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := store.Save(domain.SourceNaukri, platform.SessionState{Data: []byte("new")}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load(domain.SourceNaukri)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(got.Data) != "new" {
		t.Errorf("Load() = %q, want %q", got.Data, "new")
	}
}
