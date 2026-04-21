package common

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func EnsureInteractive(in io.Reader, out io.Writer, modeName string) error {
	if !isTerminalReader(in) || !isTerminalWriter(out) {
		return fmt.Errorf("%s requires an interactive terminal; use explicit non-TUI output flags in scripts", modeName)
	}
	return nil
}

func isTerminalReader(in io.Reader) bool {
	f, ok := in.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func isTerminalWriter(out io.Writer) bool {
	f, ok := out.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
