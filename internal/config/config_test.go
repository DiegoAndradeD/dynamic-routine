package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTemplateIsValidAndEncryptedByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	if err := WriteTemplate(path); err != nil {
		t.Fatal(err)
	}
	cfg, exists, err := Load(path)
	if err != nil || !exists || !cfg.Encrypt {
		t.Fatalf("cfg=%+v exists=%v err=%v", cfg, exists, err)
	}
	if got := cfg.StatePath(path); got != filepath.Join(filepath.Dir(path), "state.rtn") {
		t.Fatalf("default state path = %s", got)
	}
}

func TestMissingConfigGivesDefaults(t *testing.T) {
	cfg, exists, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || exists || !cfg.Encrypt {
		t.Fatalf("cfg=%+v exists=%v err=%v", cfg, exists, err)
	}
}

func TestUnknownKeyIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("encrpyt = false\n"), 0o600) // typo
	if _, _, err := Load(path); err == nil || !strings.Contains(err.Error(), "encrpyt") {
		t.Fatalf("err = %v", err)
	}
}

func TestStatePathExpansion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the expected paths are POSIX paths")
	}
	t.Setenv("HOME", "/home/u")
	t.Setenv("SYNC", "/mnt/sync")
	cfgPath := filepath.Join("/etc", "r", "config.toml")
	for in, want := range map[string]string{
		"~/Sync/s.rtn": "/home/u/Sync/s.rtn",
		"$SYNC/s.rtn":  "/mnt/sync/s.rtn",
		"data/s.rtn":   "/etc/r/data/s.rtn",
		"/abs/s.rtn":   "/abs/s.rtn",
	} {
		got := Config{StateFile: in, Encrypt: true}.StatePath(cfgPath)
		if filepath.ToSlash(got) != want {
			t.Errorf("%q -> %s, want %s", in, got, want)
		}
	}
	if got := (Config{}).StatePath(cfgPath); filepath.Base(got) != "state.json" {
		t.Fatalf("unencrypted default = %s", got)
	}
}
