// Package store persists the routine state in one file, optionally encrypted,
// and is careful about the file living in a folder another machine writes to.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

// ErrNotFound means the state file does not exist yet.
var ErrNotFound = errors.New("state file does not exist")

// Store reads and writes the state file. A nil cipher means plain JSON.
//
// Two machines can share the file through a sync tool, so mutations go through
// Update, which re-reads the file right before changing it: whatever the other
// machine saved in the meantime is kept instead of overwritten.
type Store struct {
	path   string
	cipher *vault.Cipher

	// What the file looked like when we last read or wrote it.
	mod  time.Time
	size int64
}

func New(path string, cipher *vault.Cipher) *Store {
	return &Store{path: path, cipher: cipher}
}

func (s *Store) Path() string    { return s.path }
func (s *Store) Encrypted() bool { return s.cipher != nil }

// Cipher is the cipher in use (nil for plain JSON), for opening sibling files.
func (s *Store) Cipher() *vault.Cipher { return s.cipher }

func (s *Store) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Load reads the state file. A file that exists but cannot be read, decrypted or
// parsed is an error: callers must not carry on with defaults, or the next save
// would overwrite the user's data.
func (s *Store) Load() (routine.State, error) {
	info, _ := os.Stat(s.path)
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return routine.State{}, fmt.Errorf("%w: %s", ErrNotFound, s.path)
	}
	if err != nil {
		return routine.State{}, fmt.Errorf("reading %s: %w", s.path, err)
	}

	st, err := s.decode(data)
	if err != nil {
		return routine.State{}, fmt.Errorf("%s: %w", s.path, err)
	}
	if info != nil {
		s.mod, s.size = info.ModTime(), info.Size()
	}
	return st, nil
}

func (s *Store) decode(data []byte) (routine.State, error) {
	if vault.IsEncrypted(data) {
		if s.cipher == nil {
			return routine.State{}, errors.New("file is encrypted but encryption is turned off in the config")
		}
		plain, err := s.cipher.Open(data)
		if err != nil {
			return routine.State{}, err
		}
		data = plain
	}
	// A plaintext file with encryption on is accepted: it is how a file from
	// before encryption (or from `routine export`) gets migrated. The next save
	// encrypts it.

	var st routine.State
	if err := json.Unmarshal(data, &st); err != nil {
		return routine.State{}, fmt.Errorf("parsing: %w", err)
	}
	st.Normalize()
	return st, nil
}

// Save writes the state atomically (temp file in the same directory, then
// rename), so a crash, a yanked drive or a sync tool picking the file up midway
// cannot see a truncated file. The file is readable by the owner only.
func (s *Store) Save(st routine.State) error {
	var (
		data []byte
		err  error
	)
	if s.cipher != nil {
		if data, err = json.Marshal(st); err == nil {
			data, err = s.cipher.Seal(data)
		}
	} else if data, err = json.MarshalIndent(st, "", "  "); err == nil {
		data = append(data, '\n')
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".routine-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Sync(); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		return fail(err)
	}
	_ = os.Chmod(tmpName, 0o600)
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return err
	}

	if info, err := os.Stat(s.path); err == nil {
		s.mod, s.size = info.ModTime(), info.Size()
	}
	return nil
}

// Update re-reads the file, applies fn and saves the result. If fn fails
// nothing is written. It returns the state as saved.
func (s *Store) Update(fn func(*routine.State) error) (routine.State, error) {
	st, err := s.Load()
	if err != nil {
		return routine.State{}, err
	}
	if err := fn(&st); err != nil {
		return st, err
	}
	return st, s.Save(st)
}

// Changed reports whether the file was modified by someone else since we last
// read or wrote it (typically: the sync tool delivered the other machine's save).
func (s *Store) Changed() bool {
	info, err := os.Stat(s.path)
	if err != nil {
		return false
	}
	return !info.ModTime().Equal(s.mod) || info.Size() != s.size
}

// Conflicts lists copies a sync tool left next to the state file after two
// machines saved at the same time (Syncthing and Dropbox naming).
func (s *Store) Conflicts() []string {
	dir := filepath.Dir(s.path)
	base := filepath.Base(s.path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if name == base || !strings.HasPrefix(name, stem) {
			continue
		}
		if strings.Contains(name, ".sync-conflict-") || strings.Contains(name, "conflicted copy") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out
}
