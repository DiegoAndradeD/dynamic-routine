// Package config reads the small, machine-local settings file: where the state
// lives and how it is protected. Everything about the routine itself (activities,
// work quota, rules) is the plan inside the state, so it follows you between
// machines; this file does not.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// EnvConfig overrides the config file location.
const EnvConfig = "ROUTINE_CONFIG"

type Config struct {
	// StateFile is where the state is stored. Point it into a synced folder
	// (Syncthing, Dropbox, OneDrive...) to use the same routine on several
	// machines. "~" and $VARS are expanded; a relative path is relative to the
	// config file. Empty means next to the config file.
	StateFile string `toml:"state_file"`

	// Encrypt keeps the state encrypted with a passphrase.
	Encrypt bool `toml:"encrypt"`

	// PassphraseCommand, if set, prints the passphrase on stdout, e.g.
	// `pass show routine` or `secret-tool lookup app routine`. Without it (and
	// without $ROUTINE_PASSPHRASE) the passphrase is asked on start.
	PassphraseCommand string `toml:"passphrase_command"`
}

func Default() Config { return Config{Encrypt: true} }

// DefaultPath is $ROUTINE_CONFIG, or config.toml in the user's config directory
// (~/.config/dynamic-routine on Linux, %AppData%\dynamic-routine on Windows).
func DefaultPath() (string, error) {
	if p := os.Getenv(EnvConfig); p != "" {
		return filepath.Abs(p)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding the config directory: %w (set %s)", err, EnvConfig)
	}
	return filepath.Join(dir, "dynamic-routine", "config.toml"), nil
}

// Load reads the config file. exists is false (with defaults returned) when
// there is none yet. Unknown keys are an error, so a typo does not silently
// turn encryption off.
func Load(path string) (cfg Config, exists bool, err error) {
	cfg = Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, false, nil
	}
	if err != nil {
		return cfg, false, err
	}
	meta, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return cfg, true, fmt.Errorf("%s: %w", path, err)
	}
	if un := meta.Undecoded(); len(un) > 0 {
		return cfg, true, fmt.Errorf("%s: unknown setting %q", path, un[0].String())
	}
	return cfg, true, nil
}

// StatePath resolves where the state file is, given where the config is.
func (c Config) StatePath(configPath string) string {
	dir := filepath.Dir(configPath)
	p := os.ExpandEnv(c.StateFile)
	if p == "" {
		name := "state.rtn"
		if !c.Encrypt {
			name = "state.json"
		}
		return filepath.Join(dir, name)
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p)
}

const template = `# dynamic-routine: machine-local settings.
# Your activities, work quota and rules live in the state itself: run
# "routine edit" to change them.

# Where the state is stored. To use the same routine on several machines, put it
# in a folder your sync tool already shares (Syncthing, Dropbox, OneDrive,
# Google Drive...), for example:
#   state_file = "~/Sync/routine/state.rtn"
# Empty means next to this file.
state_file = ""

# Keep the state encrypted (XChaCha20-Poly1305, key from scrypt). Strongly
# recommended whenever the file is in a synced folder or a cloud drive.
encrypt = true

# Optional: a command that prints the passphrase, so you are not asked on every
# start. It runs through the shell. Examples:
#   passphrase_command = "pass show routine"
#   passphrase_command = "secret-tool lookup app routine"
# $ROUTINE_PASSPHRASE works too. Leave empty to be asked.
passphrase_command = ""
`

// WriteTemplate creates a commented config file with the defaults.
func WriteTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(template), 0o600)
}
