package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/DiegoAndradeD/dynamic-routine/internal/config"
	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

// session is an opened routine: config resolved, passphrase known, state file
// readable (or just created).
type session struct {
	cfgPath string
	cfg     config.Config
	store   *store.Store
	command string // passphrase_command, kept for `passwd`
	created bool   // the state file did not exist and was created
}

// open resolves the config and opens the state, creating the config file and a
// fresh state from the default plan on first use.
func open(cfgPath string) (*session, error) {
	if cfgPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return nil, err
		}
		cfgPath = p
	}
	cfg, exists, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := config.WriteTemplate(cfgPath); err != nil {
			return nil, fmt.Errorf("creating %s: %w", cfgPath, err)
		}
		fmt.Fprintf(os.Stderr, "Created %s\n", cfgPath)
	}

	s := &session{cfgPath: cfgPath, cfg: cfg, command: cfg.PassphraseCommand}
	path := cfg.StatePath(cfgPath)

	info, statErr := os.Stat(path)
	exists = statErr == nil
	if exists && info.IsDir() {
		return nil, fmt.Errorf("%s is a directory: state_file must be a file", path)
	}

	if !cfg.Encrypt {
		s.store = store.New(path, nil)
		return s, s.ensure(exists)
	}

	// A file that is not encrypted yet (new, or from before encryption) gets a
	// new passphrase, asked twice; an encrypted one gets asked until it opens.
	header, _ := os.ReadFile(path)
	fresh := !exists || !vault.IsEncrypted(header)
	for attempt := 0; ; attempt++ {
		pass, prompted, err := passphrase(cfg.PassphraseCommand, fresh)
		if err != nil {
			return nil, err
		}
		s.store = store.New(path, vault.New(pass))
		if err = s.ensure(exists); err == nil {
			return s, nil
		}
		if errors.Is(err, vault.ErrWrongPassphrase) && prompted && attempt < 2 {
			fmt.Fprintln(os.Stderr, "Wrong passphrase, try again.")
			continue
		}
		return nil, err
	}
}

// ensure creates the state from the default plan if the file is missing,
// otherwise checks that it can be read.
func (s *session) ensure(exists bool) error {
	if exists {
		_, err := s.store.Load()
		return err
	}
	st, err := routine.NewState(routine.DefaultPlan())
	if err != nil {
		return err
	}
	s.created = true
	return s.store.Save(st)
}
