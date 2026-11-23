package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/x/term"
)

// EnvPassphrase supplies the passphrase non-interactively.
const EnvPassphrase = "ROUTINE_PASSPHRASE"

// passphrase finds the passphrase: $ROUTINE_PASSPHRASE, then the configured
// command, then a prompt. prompted tells the caller that a person typed it, so
// a wrong one is worth asking again. confirm asks twice (for a new passphrase).
func passphrase(command string, confirm bool) (pass []byte, prompted bool, err error) {
	if v := os.Getenv(EnvPassphrase); v != "" {
		return []byte(v), false, nil
	}
	if command != "" {
		out, err := runShell(command)
		if err != nil {
			return nil, false, fmt.Errorf("passphrase_command: %w", err)
		}
		if out = strings.TrimRight(out, "\r\n"); out == "" {
			return nil, false, errors.New("passphrase_command printed nothing")
		}
		return []byte(out), false, nil
	}

	pass, err = prompt(confirm)
	return pass, true, err
}

// prompt asks for a passphrase on the terminal (twice if confirm).
func prompt(confirm bool) ([]byte, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return nil, fmt.Errorf("no terminal to ask the passphrase on: set %s or passphrase_command", EnvPassphrase)
	}
	label := "Passphrase: "
	if confirm {
		label = "New passphrase: "
	}
	first, err := readSecret(label)
	if err != nil {
		return nil, err
	}
	if len(first) == 0 {
		return nil, errors.New("the passphrase cannot be empty")
	}
	if confirm {
		again, err := readSecret("Repeat it: ")
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(first, again) {
			return nil, errors.New("the two passphrases differ")
		}
	}
	return first, nil
}

func readSecret(label string) ([]byte, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Fprintln(os.Stderr)
	return b, err
}

func runShell(command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}
