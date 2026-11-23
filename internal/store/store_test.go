package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

func newState(t *testing.T) routine.State {
	t.Helper()
	st, err := routine.NewState(routine.DefaultPlan())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestMissingFileIsNotFound(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "state.json"), nil)
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCorruptFileFailsAndKeepsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path, nil).Load(); err == nil {
		t.Fatal("expected error for corrupt file")
	}
	if got, _ := os.ReadFile(path); string(got) != "{not json" {
		t.Fatal("corrupt file must not be overwritten")
	}
}

func TestSaveLoadRoundTripAndNoTempLeft(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "state.json"), nil)
	st := newState(t)
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	st.StartTimer("focus", now)
	st.StopTimer(now.Add(90 * time.Minute))
	if _, err := st.LogTime("focus", now, 15, "", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(st); err != nil {
		t.Fatal(err)
	}
	back, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := back.WorkedOn("focus", now, now); got != 105*time.Minute {
		t.Fatalf("worked after round trip = %v, want 1h45m", got)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestEncryptedFileIsOpaqueAndRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.rtn")
	st := newState(t)
	st.Rules = append(st.Rules, routine.Rule{ID: "r-1", Trigger: "very private trigger", Instruction: "x"})

	s := New(path, vault.New([]byte("pw")))
	if err := s.Save(st); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, leak := range []string{"private", "Go for a walk", "activities"} {
		if bytes.Contains(raw, []byte(leak)) {
			t.Fatalf("encrypted file leaks %q", leak)
		}
	}

	// another machine, same passphrase
	back, err := New(path, vault.New([]byte("pw"))).Load()
	if err != nil || back.Rules[0].Trigger != "very private trigger" {
		t.Fatalf("round trip: %v %+v", err, back.Rules)
	}
	if _, err := New(path, vault.New([]byte("nope"))).Load(); !errors.Is(err, vault.ErrWrongPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}
	if _, err := New(path, nil).Load(); err == nil {
		t.Fatal("encrypted file must not load with encryption off")
	}
}

func TestPlaintextFileIsEncryptedOnNextSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	if err := New(path, nil).Save(newState(t)); err != nil {
		t.Fatal(err)
	}
	s := New(path, vault.New([]byte("pw")))
	if _, err := s.Update(func(*routine.State) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); !vault.IsEncrypted(raw) {
		t.Fatal("file should be encrypted after the first save")
	}
}

// The reason Update exists: a save from another machine, delivered by the sync
// tool after we loaded, must survive our next change.
func TestUpdateKeepsChangesMadeElsewhere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	laptop, desktop := New(path, nil), New(path, nil)
	if err := laptop.Save(newState(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := desktop.Load(); err != nil {
		t.Fatal(err)
	}

	// desktop logs a completion...
	if _, err := desktop.Update(func(st *routine.State) error {
		st.Complete(routine.SessionFull, time.Now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !laptop.Changed() {
		t.Fatal("laptop should notice the file changed under it")
	}
	// ...and the laptop, which never reloaded, logs another one.
	st, err := laptop.Update(func(st *routine.State) error {
		st.Complete(routine.SessionMicro, time.Now().Add(time.Second))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Logs) != 2 {
		t.Fatalf("logs = %d, want both completions kept", len(st.Logs))
	}
	if laptop.Changed() {
		t.Fatal("our own write must not count as an external change")
	}
}

func TestUpdateWritesNothingWhenFnFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path, nil)
	s.Save(newState(t))
	before, _ := os.ReadFile(path)
	_, err := s.Update(func(st *routine.State) error {
		st.Complete(routine.SessionFull, time.Now())
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Fatal("failed update must not write")
	}
}

func TestConflicts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.rtn")
	for _, n := range []string{"state.rtn", "state.sync-conflict-20261101-101010-ABC.rtn", "state (conflicted copy).rtn", "other.rtn"} {
		os.WriteFile(filepath.Join(dir, n), nil, 0o600)
	}
	if got := New(path, nil).Conflicts(); len(got) != 2 {
		t.Fatalf("conflicts = %v", got)
	}
}
