package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

const planHeader = `# Your plan. Save and close the editor to apply it; nothing changes if it has errors.
#
# micro_minutes   length of a micro block ("M" in the app)
# [categories]    colour (#RRGGBB) of each category; any category name works
# [[activity]]    something you do in a sitting and tick off. target_per_week is how
#                 many times it sits on the queue. id is optional (made from the name).
#                 Changing target_per_week adds/removes queue entries; removing an
#                 activity drops it from the queue but keeps its history.
# [[track]]       something you spend time on (work, studying...), by timer or by
#                 logging it. period is "day" or "week"; target_minutes is the goal for
#                 that period (0 = no goal); days (mon..sun) says which days a daily goal
#                 counts on (empty = every day). Removing a track keeps its history.
# [[rule]]        a contingency: when <trigger>, do <instruction>

`

func encodePlan(p routine.Plan) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(planHeader)
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	if len(p.Rules) == 0 {
		buf.WriteString("\n# No rules yet. Uncomment and edit this to add one:\n" +
			"# [[rule]]\n# trigger = \"I slept badly\"\n# instruction = \"Only do the lightest activity and rest.\"\n")
	}
	return buf.Bytes(), nil
}

func decodePlan(data []byte) (routine.Plan, error) {
	var p routine.Plan
	meta, err := toml.Decode(string(data), &p)
	if err != nil {
		return p, err
	}
	if un := meta.Undecoded(); len(un) > 0 {
		return p, fmt.Errorf("unknown key %q", un[0].String())
	}
	return p, nil
}

func (s *session) printPlan() error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	data, err := encodePlan(st.Plan())
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}

func (s *session) export() error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(st)
}

// edit opens the plan in $EDITOR. The plan is decrypted into a private temp
// file for the duration, in RAM-backed storage where the OS has it.
func (s *session) edit() error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	data, err := encodePlan(st.Plan())
	if err != nil {
		return err
	}

	dir, err := privateTempDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "plan.toml")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		return err
	}

	for {
		if err := runEditor(file); err != nil {
			return err
		}
		edited, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if bytes.Equal(edited, data) {
			fmt.Fprintln(os.Stderr, "No changes.")
			return nil
		}

		plan, err := decodePlan(edited)
		if err == nil {
			_, err = s.store.Update(func(st *routine.State) error { return st.ApplyPlan(plan) })
		}
		if err == nil {
			fmt.Fprintln(os.Stderr, "Plan updated.")
			return nil
		}
		fmt.Fprintf(os.Stderr, "Not applied: %v\n", err)
		if !confirm("Edit again?") {
			return errors.New("plan not changed")
		}
		data = edited
	}
}

func privateTempDir() (string, error) {
	if runtime.GOOS == "linux" {
		if dir, err := os.MkdirTemp("/dev/shm", "routine-edit-"); err == nil {
			return dir, nil
		}
	}
	return os.MkdirTemp("", "routine-edit-")
}

func runEditor(file string) error {
	editor := firstNonEmpty(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
		if runtime.GOOS == "windows" {
			editor = "notepad"
		}
	}
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], file)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q: %w", editor, err)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func confirm(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [Y/n] ", question)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "" || line == "y" || line == "yes"
}

// merge folds other copies of the state into this one: the conflict files a
// sync tool leaves when two machines saved at the same time, or an old export.
// The files are not deleted.
func (s *session) merge(files []string) error {
	if len(files) == 0 {
		files = s.store.Conflicts()
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No conflict copies found next to the state file.")
		return nil
	}

	for _, f := range files {
		other, err := store.New(f, s.store.Cipher()).Load()
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		var logs, sessions int
		if _, err := s.store.Update(func(st *routine.State) error {
			logs, sessions = st.Merge(other)
			return nil
		}); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%s: added %d log entries and %d work slices\n", filepath.Base(f), logs, sessions)
	}
	fmt.Fprintln(os.Stderr, "Merged. The copies were left in place: delete them once you are happy.")
	return nil
}

func (s *session) passwd() error {
	if !s.cfg.Encrypt {
		return errors.New("encryption is off (encrypt = false in the config)")
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	pass, err := prompt(true)
	if err != nil {
		return err
	}
	if err := store.New(s.store.Path(), vault.New(pass)).Save(st); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Passphrase changed. Other machines need the new one (and an updated passphrase_command, if you use one).")
	return nil
}

func (s *session) where() error {
	mode := "encrypted"
	if !s.store.Encrypted() {
		mode = "plain JSON"
	}
	fmt.Printf("config: %s\nstate:  %s (%s)\n", s.cfgPath, s.store.Path(), mode)
	for _, c := range s.store.Conflicts() {
		fmt.Printf("conflict copy: %s (run \"routine merge\")\n", c)
	}
	return nil
}
