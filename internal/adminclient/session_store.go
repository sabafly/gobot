/*
 * gobot -- a useful discord bot
 *
 * Copyright (C) 2024 Sabafly Developers
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package adminclient

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// StoredSession holds a persisted TUI operator session token for a specific admin server URL.
type StoredSession struct {
	BaseURL     string    `json:"base_url"`
	Token       string    `json:"token"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	ExpiresAt   time.Time `json:"expires_at"`
	SavedAt     time.Time `json:"saved_at"`
}

type sessionFile struct {
	Sessions map[string]StoredSession `json:"sessions"`
}

// FileSessionStore manages local session persistence with 0600 file permissions.
type FileSessionStore struct {
	path string
}

// DefaultSessionPath returns the default per-user session file path ($XDG_CONFIG_HOME/gobot/tui_session.json).
func DefaultSessionPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "gobot", "tui_session.json"), nil
}

// NewFileSessionStore constructs a FileSessionStore backed by path (or DefaultSessionPath if empty).
func NewFileSessionStore(path string) (*FileSessionStore, error) {
	if path == "" {
		var err error
		path, err = DefaultSessionPath()
		if err != nil {
			return nil, err
		}
	}
	return &FileSessionStore{path: path}, nil
}

// Path returns the underlying session file path.
func (s *FileSessionStore) Path() string {
	return s.path
}

// Get returns the stored session for baseURL, if present.
func (s *FileSessionStore) Get(baseURL string) (StoredSession, bool, error) {
	doc, err := s.readDoc()
	if err != nil {
		return StoredSession{}, false, err
	}
	key := NormalizeBaseURL(baseURL)
	sess, ok := doc.Sessions[key]
	return sess, ok, nil
}

// Save writes or updates the stored session for baseURL atomically with 0600 permissions.
func (s *FileSessionStore) Save(baseURL string, sess StoredSession) error {
	doc, err := s.readDoc()
	if err != nil {
		return err
	}
	key := NormalizeBaseURL(baseURL)
	sess.BaseURL = key
	if sess.SavedAt.IsZero() {
		sess.SavedAt = time.Now().UTC()
	}
	doc.Sessions[key] = sess
	return s.writeDoc(doc)
}

// Delete removes any stored session for baseURL.
func (s *FileSessionStore) Delete(baseURL string) error {
	doc, err := s.readDoc()
	if err != nil {
		return err
	}
	key := NormalizeBaseURL(baseURL)
	if _, ok := doc.Sessions[key]; !ok {
		return nil
	}
	delete(doc.Sessions, key)
	return s.writeDoc(doc)
}

func (s *FileSessionStore) readDoc() (sessionFile, error) {
	doc := sessionFile{Sessions: make(map[string]StoredSession)}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doc, nil
		}
		return doc, err
	}
	if len(raw) == 0 {
		return doc, nil
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return sessionFile{Sessions: make(map[string]StoredSession)}, err
	}
	if doc.Sessions == nil {
		doc.Sessions = make(map[string]StoredSession)
	}
	return doc, nil
}

func (s *FileSessionStore) writeDoc(doc sessionFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(dir, ".tui_session_*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
