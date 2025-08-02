package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

/*** helper ***/

const GO_DITOR_VERSION = "0.0.1"

const (
	ARROW_LEFT  = iota + 1000
	ARROW_RIGHT = iota + 1000
	ARROW_UP    = iota + 1000
	ARROW_DOWN  = iota + 1000
	DELETE_KEY  = iota + 1000
	HOME_KEY    = iota + 1000
	END_KEY     = iota + 1000
	PAGE_UP     = iota + 1000
	PAGE_DOWN   = iota + 1000
)

// Check if the byte is a control character
func isControl(c byte) bool {
	return c < 32 || c == 127
}

func ctrlKey(c int) int {
	// Convert a character to its control key equivalent
	return c & 0x1f // 0x1f is 31 in decimal, which is the control character range
}

/*** data ***/

type editorConfig struct {
	cx, cy        int
	screenRows    int
	screenCols    int
	originalState *term.State
}

var (
	E editorConfig // Global editor configuration
)

/*** terminal ***/

// die prints an error message and exits the program, similar to the C version
func die(s string) {
	os.Stdout.Write([]byte(CLEAR_SCREEN)) // Clear the screen
	os.Stdout.Write([]byte(CURSOR_HOME))  // Move cursor to the top-left corner

	fmt.Fprintf(os.Stderr, "Error: %s\n", s)
	os.Exit(1)
}

func enableRawMode() error {
	var err error
	E.originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	return err
}

func restoreTerminal() {
	if E.originalState != nil {
		term.Restore(int(os.Stdin.Fd()), E.originalState)
	}
}

func editorReadKey() (int, error) {
	buf := make([]byte, 1)
	var nread int
	var err error

	for nread, err = os.Stdin.Read(buf); nread != 1; {
		if nread == -1 && err != nil {
			die("read key")
		}
		if err != nil {
			return 0, err
		}
	}

	c := buf[0]
	if c == '\x1b' {
		seq := make([]byte, 3)
		if nread, err := os.Stdin.Read(seq[0:1]); nread != 1 || err != nil {
			return '\x1b', nil
		}
		if nread, err := os.Stdin.Read(seq[1:2]); nread != 1 || err != nil {
			return '\x1b', nil
		}

		switch seq[0] {
		case '[':
			if seq[1] >= '0' && seq[1] <= '9' {
				if nread, err := os.Stdin.Read(seq[2:3]); nread != 1 || err != nil {
					return '\x1b', nil
				}
				if seq[2] == '~' {
					switch seq[1] {
					case '1':
						return HOME_KEY, nil
					case '3':
						return DELETE_KEY, nil
					case '4':
						return END_KEY, nil
					case '5':
						return PAGE_UP, nil
					case '6':
						return PAGE_DOWN, nil
					case '7':
						return HOME_KEY, nil
					case '8':
						return END_KEY, nil
					}
				}
			} else {
				switch seq[1] {
				case 'A':
					return ARROW_UP, nil
				case 'B':
					return ARROW_DOWN, nil
				case 'C':
					return ARROW_RIGHT, nil
				case 'D':
					return ARROW_LEFT, nil
				case 'H':
					return HOME_KEY, nil
				case 'F':
					return END_KEY, nil
				}
			}
		case 'O':
			switch seq[1] {
			case 'H':
				return HOME_KEY, nil
			case 'F':
				return END_KEY, nil
			}
		}
		return '\x1b', nil
	} else {
		return int(c), nil
	}

}

func getWindowsSize(rows *int, cols *int) error {
	var err error
	*cols, *rows, err = term.GetSize(int(os.Stdout.Fd()))
	return err
}

func getCursorPosition(row *int, col *int) error {
	// Move cursor to the bottom-right corner and read the cursor position
	os.Stdout.Write([]byte(CURSOR_BOTTOM_RIGHT + CURSOR_GET_POSITION))
	var buf [32]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return err
	}

	// Parse the response
	var r, c int
	fmt.Sscanf(string(buf[:n]), CURSOR_RESPONSE_FORMAT, &r, &c)
	*row = r - 1 // Convert to zero-based index
	*col = c - 1 // Convert to zero-based index
	return nil
}

/*** append buffer ***/

type appendBuffer struct {
	b   []byte
	len int
}

func (ab *appendBuffer) append(s []byte) {
	ab.b = append(ab.b, s...)
	ab.len += len(s)
}

func (ab *appendBuffer) free() {
	ab.b = nil
	ab.len = 0
}

/*** output ***/

func editorDrawRows(abuf *appendBuffer) {
	fmt.Println("DrawRows") // Move cursor to the top-left corner
	for y := range E.screenRows {
		if y == E.screenRows/3 {
			welcome := "GO-DITOR editor -- version " + GO_DITOR_VERSION
			if len(welcome) > E.screenCols {
				welcome = welcome[:E.screenCols]
			}
			padding := (E.screenCols - len(welcome)) / 2
			if padding > 0 {
				abuf.append([]byte("~"))
			}

			for range padding - 1 {
				abuf.append([]byte(" "))
			}
			abuf.append([]byte(welcome))
		} else {
			abuf.append([]byte("~"))
		}

		abuf.append([]byte(CLEAR_LINE)) // Clear line
		if y < E.screenRows-1 {
			abuf.append([]byte("\r\n"))
		}
	}
}

func editorRefreshScreen() {
	var abuf appendBuffer

	abuf.append([]byte(CURSOR_HIDE)) // Hide the cursor
	abuf.append([]byte(CURSOR_HOME)) // Move cursor to the top-left corner

	editorDrawRows(&abuf)

	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, E.cy+1, E.cx+1)) // Move cursor to the current position

	abuf.append([]byte(CURSOR_SHOW)) // Show the cursor again

	os.Stdout.Write(abuf.b)
	abuf.free()
}

/*** input ***/

func editorMoveCursor(key int) {
	switch key {
	case ARROW_LEFT:
		if E.cx > 0 {
			E.cx--
		}
	case ARROW_RIGHT:
		if E.cx < E.screenCols-1 {
			E.cx++
		}
	case ARROW_UP:
		if E.cy > 0 {
			E.cy--
		}
	case ARROW_DOWN:
		if E.cy < E.screenRows-1 {
			E.cy++
		}
	}
}

func editorProcessKeypress() {
	key, err := editorReadKey()
	if err != nil {
		die("reading key")
	}

	switch key {
	case ctrlKey('q'):
		os.Stdout.Write([]byte(CLEAR_SCREEN)) // Clear the screen
		os.Stdout.Write([]byte(CURSOR_HOME))  // Move cursor to the top-left corner
		fmt.Println("Exiting GO-DITOR editor")
		os.Exit(0)

	case HOME_KEY:
		E.cx = 0

	case END_KEY:
		E.cx = E.screenCols - 1

	case PAGE_UP, PAGE_DOWN:
		times := E.screenRows
		for times > 0 {
			if key == PAGE_UP {
				editorMoveCursor(ARROW_UP)
			} else {
				editorMoveCursor(ARROW_DOWN)
			}
			times--
		}

	case ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ARROW_DOWN:
		editorMoveCursor(key)
	}
}

/*** init ***/

func initEditor() {
	E.cx, E.cy = 0, 0
	if getWindowsSize(&E.screenRows, &E.screenCols) != nil {
		die("getting window size")
	}
}

func main() {
	// Enable raw mode for terminal input
	// and ensure it is reset on exit
	err := enableRawMode()
	if err != nil {
		die("enabling raw mode")
	}
	defer restoreTerminal()

	initEditor()

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}

}
