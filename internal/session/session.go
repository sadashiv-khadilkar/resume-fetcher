// Package session persists PlatformClient login sessions to local disk so a
// captured login (ticket 03) survives across CLI invocations. Only the
// resulting session is ever stored, never credentials (ADR-0003).
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"resumefetcher/internal/domain"
	"resumefetcher/internal/platform"
)

// ErrNotFound is returned by Store.Load when no session has been persisted
// yet for a Source.
var ErrNotFound = errors.New("no persisted session")

// Store persists one platform.SessionState per domain.Source as a file
// under Dir.
type Store struct {
	Dir string
}

// New builds a Store rooted at dir. Dir is created lazily by Save.
func New(dir string) *Store {
	return &Store{Dir: dir}
}

func (s *Store) path(source domain.Source) string {
	return filepath.Join(s.Dir, string(source)+".json")
}

// Load returns the persisted session for source, or ErrNotFound if none has
// been captured yet.
func (s *Store) Load(source domain.Source) (platform.SessionState, error) {
	data, err := os.ReadFile(s.path(source))
	if errors.Is(err, os.ErrNotExist) {
		return platform.SessionState{}, ErrNotFound
	}
	if err != nil {
		return platform.SessionState{}, fmt.Errorf("read session for %s: %w", source, err)
	}
	var state platform.SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return platform.SessionState{}, fmt.Errorf("parse session for %s: %w", source, err)
	}
	return state, nil
}

// Save persists state as the current session for source, overwriting
// whatever was previously saved.
func (s *Store) Save(source domain.Source, state platform.SessionState) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal session for %s: %w", source, err)
	}
	if err := os.WriteFile(s.path(source), data, 0o600); err != nil {
		return fmt.Errorf("write session for %s: %w", source, err)
	}
	return nil
}
