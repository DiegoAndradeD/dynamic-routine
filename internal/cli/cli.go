// Package cli implements the routine command: the TUI and the few commands
// around it (edit the plan, merge conflict copies, change the passphrase...).
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/DiegoAndradeD/dynamic-routine/internal/ui"
)

const usage = `routine: a dynamic routine in your terminal.

Usage:
  routine [--config FILE] [command]

Commands:
  (none)         open the app
  edit           edit your activities, work quota and rules in $EDITOR
  plan           print the plan (activities, quota, rules) as TOML
  export         print the whole state as JSON (decrypted: handle with care)
  merge [FILE..] fold conflict copies left by your sync tool into the state
  passwd         change the passphrase
  where          show where the config and the state are
  version        show the version
  help           show this text

The config file is ~/.config/dynamic-routine/config.toml (%AppData% on
Windows), or $ROUTINE_CONFIG. See README.md.
`

// Run executes the command line (without the program name).
func Run(args []string) error {
	cfgPath := ""
	var rest []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--config" || args[i] == "-c":
			if i+1 >= len(args) {
				return fmt.Errorf("%s needs a file", args[i])
			}
			cfgPath = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--config="):
			cfgPath = strings.TrimPrefix(args[i], "--config=")
		default:
			rest = append(rest, args[i])
		}
	}

	cmd := "run"
	if len(rest) > 0 {
		cmd, rest = rest[0], rest[1:]
	}

	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version", "-v", "--version":
		fmt.Println(version())
		return nil
	case "run", "edit", "plan", "export", "merge", "passwd", "where":
	default:
		return fmt.Errorf("unknown command %q (try: routine help)", cmd)
	}

	s, err := open(cfgPath)
	if err != nil {
		return err
	}
	if s.created && cmd != "run" {
		fmt.Fprintf(os.Stderr, "Created %s with an example plan.\n", s.store.Path())
	}

	switch cmd {
	case "run":
		return ui.Run(s.store, s.created)
	case "edit":
		return s.edit()
	case "plan":
		return s.printPlan()
	case "export":
		return s.export()
	case "merge":
		return s.merge(rest)
	case "passwd":
		return s.passwd()
	default:
		return s.where()
	}
}
