package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

/*** helper ***/
// Check if the byte is a control character
func iscontrol(c byte) bool {
	return c < 32 || c == 127
}

func ctrlKey(c byte) byte {
	// Convert a character to its control key equivalent
	return c & 0x1f // 0x1f is 31 in decimal, which is the control character range
}

/*** data ***/
var (
	originalState *term.State
)

/*** terminal ***/

// die prints an error message and exits the program, similar to the C version
func die(s string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", s)
	os.Exit(1)
}

func enableRawMode() error {
	var err error
	originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	return err
}

func restoreTerminal() {
	if originalState != nil {
		term.Restore(int(os.Stdin.Fd()), originalState)
	}
}

func editorReadKey() (byte, error) {
	buf := make([]byte, 1)
	for {
		nread, err := os.Stdin.Read(buf)
		if nread == 1 {
			return buf[0], nil
		}
		if err != nil {
			return 0, err
		}
	}
}

/*** output ***/

/*** input ***/

func editorProcessKeypress() {
	char, err := editorReadKey()
	if err != nil {
		die("reading key")
	}

	switch char {
	case ctrlKey('q'):
		os.Exit(0)
	default:
		if iscontrol(char) {
			fmt.Printf("Control character: %d\r\n", char)
		} else {
			fmt.Printf("Character: %c\r\n", char)
		}
	}
}

/*** init ***/

func main() {
	// Enable raw mode for terminal input
	// and ensure it is reset on exit
	err := enableRawMode()
	if err != nil {
		die("enabling raw mode")
	}
	defer restoreTerminal()

	for {
		editorProcessKeypress()
	}

}
