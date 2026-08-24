//go:build !windows

package provision

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// IsStdinTerminal reports whether standard input is an interactive terminal.
func IsStdinTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PromptHidden reads one line from the terminal with echo disabled. It fails
// closed when stdin is not a terminal.
func PromptHidden(prompt string) (string, error) {
	if !IsStdinTerminal() {
		return "", ErrNotTerminal
	}
	fmt.Fprint(os.Stderr, prompt)
	line, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("provision: read secret: %w", err)
	}
	return strings.TrimSpace(string(line)), nil
}

// PromptVisible reads one visible line (used for non-secret confirmations).
func PromptVisible(prompt string) (string, error) {
	if !IsStdinTerminal() {
		return "", ErrNotTerminal
	}
	fmt.Fprint(os.Stderr, prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("provision: read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// Confirm asks a yes/no question and reports whether the user typed "yes".
func Confirm(prompt string) (bool, error) {
	ans, err := PromptVisible(prompt + " [yes/no]: ")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(ans) {
	case "y", "yes":
		return true, nil
	default:
		return false, errors.New("provision: confirmation declined")
	}
}
