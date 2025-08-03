package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

/*** helper ***/

const (
	GO_DITOR_VERSION = "0.0.1"
	TAB_STOP         = 4
)

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
type erow struct {
	size   int
	chars  []byte
	rsize  int
	render []byte
}

type editorConfig struct {
	cx, cy            int
	rx                int
	rowOffset         int
	colOffset         int
	screenRows        int
	screenCols        int
	numrows           int
	row               []erow // Changed from *erow to []erow (slice)
	filename          *string
	statusMessage     string
	statusMessageTime time.Time
	originalState     *term.State
}

var (
	E editorConfig // Global editor configuration
)

/*** terminal ***/

// die prints an error message and exits the program, similar to the C version
func die(s string) {
	restoreTerminal()                     // Restore terminal before output
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
		E.originalState = nil // Prevent multiple restoration attempts
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

/*** row operations ***/

func editorRowCxToRx(row *erow, cx int) int {
	rx := 0
	for j := 0; j < cx; j++ {
		if row.chars[j] == '\t' {
			rx += TAB_STOP - (rx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else {
			rx++
		}
	}
	return rx
}

func editorUpdateRow(row *erow) {
	tabs := 0
	for j := 0; j < row.size; j++ {
		if row.chars[j] == '\t' {
			tabs++
		}
	}

	// Size: for worst case tab expansion
	row.render = make([]byte, row.size+tabs*(TAB_STOP-1))

	idx := 0
	for j := 0; j < row.size; j++ {
		if row.chars[j] == '\t' {
			row.render[idx] = ' '
			idx++
			// Add spaces until we reach the next TAB_STOP boundary
			for idx%TAB_STOP != 0 {
				row.render[idx] = ' '
				idx++
			}
		} else {
			row.render[idx] = row.chars[j]
			idx++
		}
	}

	row.rsize = idx
}

func editorAppendRow(s []byte, rowlen int) {
	// Equivalent to realloc in C - grow the slice
	E.row = append(E.row, erow{})
	at := E.numrows
	E.row[at].size = rowlen
	E.row[at].chars = make([]byte, rowlen)
	copy(E.row[at].chars, s)

	E.row[at].rsize = 0
	E.row[at].render = nil

	editorUpdateRow(&E.row[at])
	E.numrows++
}

/*** file i/o ***/

func editorOpen(filename *string) {
	E.filename = filename
	file, err := os.Open(*filename)
	if err != nil {
		die("fopen")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing newlines and carriage returns
		for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
			line = line[:len(line)-1]
		}

		editorAppendRow([]byte(line), len(line))
	}

	if err := scanner.Err(); err != nil {
		die("reading file")
	}
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

func editorScroll() {
	E.rx = 0
	if E.cy < E.numrows {
		E.rx = editorRowCxToRx(&E.row[E.cy], E.cx)
	}

	if E.cy < E.rowOffset {
		E.rowOffset = E.cy
	}
	if E.cy >= E.rowOffset+E.screenRows {
		E.rowOffset = E.cy - E.screenRows + 1
	}

	if E.rx < E.colOffset {
		E.colOffset = E.rx
	}
	if E.rx >= E.colOffset+E.screenCols {
		E.colOffset = E.rx - E.screenCols + 1
	}
}

func editorDrawRows(abuf *appendBuffer) {
	for y := range E.screenRows {
		filerow := y + E.rowOffset
		if filerow >= E.numrows {
			if E.numrows == 0 && y == E.screenRows/3 {
				welcome := "GO-DITOR editor -- version " + GO_DITOR_VERSION
				welcomelen := len(welcome)
				if welcomelen > E.screenCols {
					welcomelen = E.screenCols
				}
				padding := (E.screenCols - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte("~"))
					padding--
				}
				for padding > 0 {
					abuf.append([]byte(" "))
					padding--
				}
				abuf.append([]byte(welcome[:welcomelen]))
			} else {
				abuf.append([]byte("~"))
			}
		} else {
			lineLen := min(max(E.row[filerow].rsize-E.colOffset, 0), E.screenCols)
			if lineLen > 0 {
				start := E.colOffset
				end := E.colOffset + lineLen
				abuf.append(E.row[filerow].render[start:end])
			}
		}

		abuf.append([]byte(CLEAR_LINE)) // Clear line
		abuf.append([]byte("\r\n"))
	}
}

func editorDrawStatusBar(abuf *appendBuffer) {
	abuf.append([]byte(COLORS_INVERT)) // Invert colors for status bar

	var status string
	var rstatus string
	filename := "[No Name]"
	if E.filename != nil {
		filename = *E.filename
		// Truncate filename to 20 characters if needed
		if len(filename) > 20 {
			filename = filename[:20]
		}
	}
	status = fmt.Sprintf("%.20s - %d lines", filename, E.numrows)

	statusLen := min(len(status), E.screenCols)
	rstatus = fmt.Sprintf("%d/%d", E.cy+1, E.numrows)
	rstatusLen := len(rstatus)
	abuf.append([]byte(status[:statusLen]))

	for statusLen < E.screenCols {
		if E.screenCols-statusLen == rstatusLen {
			abuf.append([]byte(rstatus))
			break
		} else {
			abuf.append([]byte(" "))
			statusLen++
		}
	}

	abuf.append([]byte(COLORS_RESET)) // Reset colors
	abuf.append([]byte("\r\n"))       // New line
}

func editorDrawMessageBar(abuf *appendBuffer) {
	abuf.append([]byte(CLEAR_LINE))
	messageLen := min(len(E.statusMessage), E.screenCols)
	if time.Since(E.statusMessageTime) < 5*time.Second {
		abuf.append([]byte(E.statusMessage[:messageLen]))
	}
}

func editorRefreshScreen() {
	editorScroll()

	var abuf appendBuffer

	abuf.append([]byte(CURSOR_HIDE)) // Hide the cursor
	abuf.append([]byte(CURSOR_HOME)) // Move cursor to the top-left corner

	editorDrawRows(&abuf)
	editorDrawStatusBar(&abuf)
	editorDrawMessageBar(&abuf)

	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, E.cy-E.rowOffset+1, E.rx-E.colOffset+1)) // Move cursor to the current position

	abuf.append([]byte(CURSOR_SHOW)) // Show the cursor again

	os.Stdout.Write(abuf.b)
	abuf.free()
}

func editorSetStatusMessage(format string, args ...interface{}) {
	E.statusMessage = fmt.Sprintf(format, args...)
	E.statusMessageTime = time.Now()
}

/*** input ***/

func editorMoveCursor(key int) {
	var row *erow
	if E.cy >= E.numrows {
		row = nil
	} else {
		row = &E.row[E.cy]
	}

	switch key {
	case ARROW_LEFT:
		if E.cx != 0 {
			E.cx--
		} else if E.cy > 0 {
			E.cy--
			E.cx = E.row[E.cy].size
		}
	case ARROW_RIGHT:
		if row != nil && E.cx < row.size {
			E.cx++
		} else if row != nil && E.cx == row.size {
			E.cy++
			E.cx = 0
		}
	case ARROW_UP:
		if E.cy != 0 {
			E.cy--
		}
	case ARROW_DOWN:
		if E.cy < E.numrows {
			E.cy++
		}
	}

	if E.cy >= E.numrows {
		row = nil
	} else {
		row = &E.row[E.cy]
	}
	rowlen := 0
	if row != nil {
		rowlen = row.size
	}
	if E.cx > rowlen {
		E.cx = rowlen
	}
}

func editorProcessKeypress() {
	key, err := editorReadKey()
	if err != nil {
		die("reading key")
	}

	switch key {
	case ctrlKey('q'):
		restoreTerminal()                     // Restore terminal before clearing screen
		os.Stdout.Write([]byte(CLEAR_SCREEN)) // Clear the screen
		os.Stdout.Write([]byte(CURSOR_HOME))  // Move cursor to the top-left corner
		fmt.Println("Exiting GO-DITOR editor")
		os.Exit(0)

	case HOME_KEY:
		E.cx = 0

	case END_KEY:
		if E.cy < E.numrows {
			E.cx = E.row[E.cy].size
		}

	case PAGE_UP:
		E.cy = E.rowOffset
		times := E.screenRows
		for times > 0 {
			editorMoveCursor(ARROW_UP)
			times--
		}

	case PAGE_DOWN:
		E.cy = min(E.rowOffset+E.screenRows-1, E.numrows)
		times := E.screenRows
		for times > 0 {
			editorMoveCursor(ARROW_DOWN)
			times--
		}

	case ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ARROW_DOWN:
		editorMoveCursor(key)
	}
}

/*** init ***/

func initEditor() {
	E.cx, E.cy = 0, 0
	E.rx = 0
	E.rowOffset = 0
	E.colOffset = 0
	E.numrows = 0
	E.row = make([]erow, 0)
	E.filename = nil
	E.statusMessage = ""
	E.statusMessageTime = time.Time{}

	if getWindowsSize(&E.screenRows, &E.screenCols) != nil {
		die("getting window size")
	}
	E.screenRows -= 2
}

func main() {
	args := os.Args[1:]
	// Enable raw mode for terminal input
	// and ensure it is reset on exit
	err := enableRawMode()
	if err != nil {
		die("enabling raw mode")
	}
	defer restoreTerminal()

	initEditor()
	if len(args) >= 1 {
		editorOpen(&args[0])
	}

	editorSetStatusMessage("HELP: Ctrl-Q = quit")

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}

}
