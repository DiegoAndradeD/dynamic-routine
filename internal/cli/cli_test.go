package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/DiegoAndradeD/dynamic-routine/internal/config"
	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

// setup points the app at a throwaway config and passphrase.
func setup(t *testing.T) (cfgPath string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(EnvPassphrase, "test passphrase")
	t.Setenv("HOME", dir)
	return filepath.Join(dir, "cfg", "config.toml")
}

func TestFirstRunCreatesConfigAndEncryptedState(t *testing.T) {
	cfgPath := setup(t)
	s, err := open(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !s.created || !s.store.Encrypted() {
		t.Fatalf("created=%v encrypted=%v", s.created, s.store.Encrypted())
	}
	raw, err := os.ReadFile(s.store.Path())
	if err != nil || !vault.IsEncrypted(raw) {
		t.Fatalf("state file should exist and be encrypted: %v", err)
	}
	if _, exists, _ := config.Load(cfgPath); !exists {
		t.Fatal("config file should have been written")
	}

	// second run: opens, does not recreate
	again, err := open(cfgPath)
	if err != nil || again.created {
		t.Fatalf("second open: %v created=%v", err, again.created)
	}
}

func TestWrongPassphraseIsRefused(t *testing.T) {
	cfgPath := setup(t)
	if _, err := open(cfgPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvPassphrase, "something else")
	if _, err := open(cfgPath); err == nil {
		t.Fatal("must not open with the wrong passphrase")
	}
}

func TestPlaintextModeAndMigrationToEncrypted(t *testing.T) {
	cfgPath := setup(t)
	os.MkdirAll(filepath.Dir(cfgPath), 0o700)
	os.WriteFile(cfgPath, []byte("encrypt = false\n"), 0o600)
	s, err := open(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(s.store.Path()); vault.IsEncrypted(raw) || !strings.Contains(string(raw), "activities") {
		t.Fatal("expected plain JSON")
	}

	// turn encryption on, pointing at the same file
	os.WriteFile(cfgPath, []byte("encrypt = true\nstate_file = \""+filepath.ToSlash(s.store.Path())+"\"\n"), 0o600)
	s2, err := open(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.store.Update(func(*routine.State) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(s2.store.Path()); !vault.IsEncrypted(raw) {
		t.Fatal("first save after enabling encryption must encrypt the file")
	}
}

func TestEditAppliesThePlan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as the editor")
	}
	cfgPath := setup(t)
	script := filepath.Join(t.TempDir(), "editor.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nsed -i 's/name = \"Go for a walk\"/name = \"Climbing\"/' \"$1\"\n"), 0o700)
	t.Setenv("EDITOR", script)
	t.Setenv("VISUAL", "")

	s, err := open(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.edit(); err != nil {
		t.Fatal(err)
	}
	st, _ := s.store.Load()
	if st.Activities["walk"].Name != "Climbing" {
		t.Fatalf("name = %q", st.Activities["walk"].Name)
	}
}

func TestPlanEncodeDecodeRoundTrip(t *testing.T) {
	p := routine.DefaultPlan()
	data, err := encodePlan(p)
	if err != nil {
		t.Fatal(err)
	}
	back, err := decodePlan(data)
	if err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
	st1, err1 := routine.NewState(p)
	st2, err2 := routine.NewState(back)
	if err1 != nil || err2 != nil || len(st1.Activities) != len(st2.Activities) || len(st2.Rules) != len(p.Rules) {
		t.Fatalf("round trip lost data: %v %v", err1, err2)
	}
	if _, err := decodePlan([]byte(data2typo(string(data)))); err == nil {
		t.Fatal("an unknown key must be rejected")
	}
}

func data2typo(s string) string {
	return strings.Replace(s, "target_per_week =", "target_per_wek =", 1)
}

func TestMergeConflictCopy(t *testing.T) {
	cfgPath := setup(t)
	s, err := open(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// the other machine's copy: same file plus one completion
	other, _ := s.store.Load()
	other.Complete(routine.SessionFull, time.Now())
	conflict := strings.TrimSuffix(s.store.Path(), ".rtn") + ".sync-conflict-20261101-100000-ABCDEFG.rtn"
	if err := store.New(conflict, s.store.Cipher()).Save(other); err != nil {
		t.Fatal(err)
	}

	if got := s.store.Conflicts(); len(got) != 1 {
		t.Fatalf("conflicts = %v", got)
	}
	if err := s.merge(nil); err != nil {
		t.Fatal(err)
	}
	st, _ := s.store.Load()
	if len(st.Logs) != 1 {
		t.Fatalf("logs after merge = %d", len(st.Logs))
	}
	if err := s.merge(nil); err != nil { // idempotent
		t.Fatal(err)
	}
	if st, _ = s.store.Load(); len(st.Logs) != 1 {
		t.Fatal("merging again duplicated history")
	}
}

func TestVersion(t *testing.T) {
	if err := Run([]string{"version"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(version(), "routine ") {
		t.Fatalf("version = %q", version())
	}
}

func TestUnknownCommand(t *testing.T) {
	if err := Run([]string{"frobnicate"}); err == nil {
		t.Fatal("expected an error")
	}
}
