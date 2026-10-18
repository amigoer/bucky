package cli

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

func readPassword(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", coreInvalidTerminal()
	}
	fmt.Fprint(os.Stderr, prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("empty credential")
	}
	return string(data), nil
}
