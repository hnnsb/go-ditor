package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

/*** helper ***/

const (
	GO_DITOR_VERSION = "0.0.1"
	TAB_STOP         = 4
	QUIT_TIMES       = 3
)

const (
	BACKSPACE   = 127 // ASCII backspace
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
type row struct {
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
	row               []row // Changed from *erow to []erow (slice)
	dirty             int
	filename          *string
	statusMessage     string
	statusMessageTime time.Time
	originalState     *term.State
}

var (
	E editorConfig // Global editor configuration
)

/*** terminal ***/

// die prints an error message and exits the program
func die(s string) {
	restoreTerminal()
	os.Stdout.Write([]byte(CLEAR_SCREEN))
	os.Stdout.Write([]byte(CURSOR_HOME))
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

func editorRowCxToRx(row *row, cx int) int {
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

func editorRowRxToCx(row *row, rx int) int {
	cur_rx := 0
	var cx int
	for cx = 0; cx < row.size; cx++ {
		if row.chars[cx] == '\t' {
			cur_rx += (TAB_STOP - 1) - (cur_rx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		}
		cur_rx++

		if cur_rx > rx {
			return cx
		}
	}
	return cx
}

func editorUpdateRow(row *row) {
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

func editorInsertRow(at int, s []byte, rowlen int) {
	if at < 0 || at > E.numrows {
		return
	}

	E.row = append(E.row, row{})
	copy(E.row[at+1:], E.row[at:E.numrows])

	E.row[at].size = rowlen
	E.row[at].chars = make([]byte, rowlen)
	copy(E.row[at].chars, s)

	E.row[at].rsize = 0
	E.row[at].render = nil

	editorUpdateRow(&E.row[at])
	E.numrows++
	E.dirty++
}

func editorFreeRow(erow *row) {
	if erow == nil {
		return
	}
	erow.chars = nil
	erow.render = nil
}

func editorDeleteRow(at int) {
	if at < 0 || at >= E.numrows {
		return
	}

	// Free the row's resources
	editorFreeRow(&E.row[at])

	// Shift rows down to fill the gap
	copy(E.row[at:], E.row[at+1:E.numrows])
	E.row = E.row[:E.numrows-1] // Resize the slice

	E.numrows--
	E.dirty++
}

func editorRowInsertChar(erow *row, at int, c int) {
	if at < 0 || at > erow.size {
		at = erow.size
	}

	// Grow the slice to accommodate the new character
	erow.chars = append(erow.chars, 0) // Add space for one more character

	// Shift characters to the right to make room for insertion
	// This is equivalent to memmove(&row->chars[at + 1], &row->chars[at], row->size - at + 1)
	copy(erow.chars[at+1:], erow.chars[at:erow.size])

	// Insert the new character
	erow.chars[at] = byte(c)
	erow.size++

	editorUpdateRow(erow)
	E.dirty++
}

func editorRowAppendString(erow *row, s []byte, slen int) {
	newSize := erow.size + slen
	newChars := make([]byte, newSize)

	copy(newChars[:erow.size], erow.chars[:erow.size])
	copy(newChars[erow.size:], s[:slen])

	erow.chars = newChars
	erow.size = newSize

	editorUpdateRow(erow)
	E.dirty++
}

func editorRowDeleteChar(erow *row, at int) {
	if at < 0 || at >= erow.size {
		return
	}

	// Shift characters to the left to overwrite the deleted character
	// This is equivalent to memmove(&row->chars[at], &row->chars[at + 1], row->size - at - 1)
	copy(erow.chars[at:], erow.chars[at+1:erow.size])
	erow.size--

	editorUpdateRow(erow)
	E.dirty++
}

/*** editor operations ***/

func editorInsertChar(c int) {
	if E.cy == E.numrows {
		editorInsertRow(E.numrows, []byte(""), 0)
	}
	editorRowInsertChar(&E.row[E.cy], E.cx, c)
	E.cx++
}

func editorInsertNewline() {
	if E.cx == 0 {
		editorInsertRow(E.cy, []byte(""), 0)
	} else {
		row := &E.row[E.cy]

		// Insert new row with text from cursor to end of line
		remainingText := make([]byte, row.size-E.cx)
		copy(remainingText, row.chars[E.cx:row.size])
		editorInsertRow(E.cy+1, remainingText, row.size-E.cx)

		// Truncate current row to text before cursor
		row = &E.row[E.cy] // Re-get pointer after slice may have been reallocated
		row.size = E.cx
		row.chars = row.chars[:E.cx]
		editorUpdateRow(row)
	}
	E.cy++
	E.cx = 0
}

func editorDeleteChar() {
	if E.cy == E.numrows {
		return
	}
	if E.cx == 0 && E.cy == 0 {
		return
	}

	row := &E.row[E.cy]
	if E.cx > 0 {
		editorRowDeleteChar(row, E.cx-1)
		E.cx--
	} else {
		E.cx = E.row[E.cy-1].size
		editorRowAppendString(&E.row[E.cy-1], row.chars, row.size)
		editorDeleteRow(E.cy) // Delete the current row after appending its content to the previous row
		E.cy--                // Move cursor up to the previous row
	}
}

/*** file i/o ***/

func editorRowsToString(bufLen *int) []byte {
	totalLength := 0
	for j := 0; j < E.numrows; j++ {
		totalLength += E.row[j].size + 1 // +1 for newline character
	}
	*bufLen = totalLength

	buf := make([]byte, totalLength)
	p := 0

	for j := 0; j < E.numrows; j++ {
		copy(buf[p:p+E.row[j].size], E.row[j].chars[:E.row[j].size])
		p += E.row[j].size
		buf[p] = '\n'
		p++
	}

	return buf
}

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

		editorInsertRow(E.numrows, []byte(line), len(line))
	}

	if err := scanner.Err(); err != nil {
		die("reading file")
	}
	E.dirty = 0
}

func editorSave() {
	if E.filename == nil {
		E.filename = editorPrompt("Save as: %s (ESC to cancel)", nil)
		if E.filename == nil {
			editorSetStatusMessage("Save aborted")
			return
		}
	}

	var length int
	buf := editorRowsToString(&length)

	// Open file for read/write, create if not exists (equivalent to O_RDWR | O_CREAT, 0644)
	file, err := os.OpenFile(*E.filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		editorSetStatusMessage("Can't save! I/O error: %v", err)
		return
	}
	defer file.Close()

	// Truncate file to exact length (equivalent to ftruncate(fd, len))
	err = file.Truncate(int64(length))
	if err != nil {
		editorSetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	// Write buffer to file (equivalent to write(fd, buf, len))
	bytesWritten, err := file.Write(buf)
	if err != nil {
		editorSetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	// Check if all bytes were written
	if bytesWritten != length {
		editorSetStatusMessage("Can't save! Partial write: %d/%d bytes", bytesWritten, length)
		return
	}

	// Success message with byte count (equivalent to C version's success case)
	editorSetStatusMessage("%d bytes written to disk", length)
	E.dirty = 0 // Reset dirty flag after successful save
}

/*** find ***/

var (
	last_match = -1
	direction  = 1
)

func editorFindCallback(query []byte, key int) {
	switch key {
	case '\r', '\x1b':
		last_match = -1
		direction = 1
		return
	case ARROW_RIGHT, ARROW_DOWN:
		direction = 1
	case ARROW_LEFT, ARROW_UP:
		direction = -1
	default:
		last_match = -1
		direction = 1
	}

	if last_match == -1 {
		direction = 1
	}
	current := last_match

	for i := 0; i < E.numrows; i++ {
		current += direction
		if current == -1 {
			current = E.numrows - 1
		} else if current == E.numrows {
			current = 0
		}

		row := &E.row[current]
		match := bytes.Index(row.render, query)
		if match != -1 {
			last_match = current
			E.cy = current
			E.cx = editorRowRxToCx(row, match)
			E.rowOffset = E.numrows
			break
		}
	}
}

func editorFind() {
	saved_cx := E.cx
	saved_cy := E.cy
	saved_colOffset := E.colOffset
	saved_rowOffset := E.rowOffset

	query := editorPrompt("Search: %s (Use ESC/Arrows/Enter)", editorFindCallback)

	if query == nil {
		E.cx = saved_cx
		E.cy = saved_cy
		E.colOffset = saved_colOffset
		E.rowOffset = saved_rowOffset
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
	dirtyFlag := ""
	if E.dirty > 0 {
		dirtyFlag = "(modified)"
	}
	status = fmt.Sprintf("%.20s - %d lines %s %d", filename, E.numrows, dirtyFlag, E.dirty)

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

func editorPrompt(prompt string, callback func([]byte, int)) *string {
	bufSize := 128
	buf := make([]byte, 0, bufSize)

	for {
		editorSetStatusMessage(prompt, string(buf))
		editorRefreshScreen()

		key, err := editorReadKey()
		if err != nil {
			die("reading key")
		}

		switch key {
		case DELETE_KEY, BACKSPACE, ctrlKey('h'):
			if len(buf) != 0 {
				buf = buf[:len(buf)-1]
			}

		case '\x1b':
			editorSetStatusMessage("")
			if callback != nil {
				callback(buf, key)
			}
			return nil

		case '\r':
			if len(buf) != 0 {
				editorSetStatusMessage("")
				if callback != nil {
					callback(buf, key)
				}
				result := string(buf)
				return &result
			}

		default:
			if !isControl(byte(key)) && key < 128 {
				if len(buf) == bufSize-1 {
					bufSize *= 2
					newBuf := make([]byte, len(buf), bufSize)
					copy(newBuf, buf)
					buf = newBuf
				}
				buf = append(buf, byte(key))
			}
		}
		if callback != nil {
			callback(buf, key)
		}
	}
}

func editorMoveCursor(key int) {
	var row *row
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

var quit_times = QUIT_TIMES

func editorProcessKeypress() {

	key, err := editorReadKey()
	if err != nil {
		die("reading key")
	}

	switch key {
	case '\r':
		editorInsertNewline()

	case ctrlKey('q'):
		if E.dirty > 0 && quit_times > 0 {
			editorSetStatusMessage("WARNING: File has unsaved changes. Press Ctrl-Q %d more times to quit.", quit_times)
			quit_times--
			return
		}

		restoreTerminal()                     // Restore terminal before clearing screen
		os.Stdout.Write([]byte(CLEAR_SCREEN)) // Clear the screen
		os.Stdout.Write([]byte(CURSOR_HOME))  // Move cursor to the top-left corner
		fmt.Println("Exiting GO-DITOR editor")
		os.Exit(0)

	case ctrlKey('s'):
		editorSave()

	case HOME_KEY:
		E.cx = 0

	case END_KEY:
		if E.cy < E.numrows {
			E.cx = E.row[E.cy].size
		}

	case ctrlKey('f'):
		editorFind()

	case BACKSPACE, ctrlKey('h'), DELETE_KEY:
		if key == DELETE_KEY {
			editorMoveCursor(ARROW_RIGHT)
		}
		editorDeleteChar()

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

	case ctrlKey('l'):
	case '\x1b':
		break

	default:
		editorInsertChar(key)
	}

	quit_times = QUIT_TIMES // Reset quit times after processing a key
}

/*** init ***/

func initEditor() {
	E.cx, E.cy = 0, 0
	E.rx = 0
	E.rowOffset = 0
	E.colOffset = 0
	E.numrows = 0
	E.row = make([]row, 0)
	E.dirty = 0
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

	editorSetStatusMessage("HELP: Ctrl-S = save | Ctrl-Q = quit | Ctrl-F = find")

	for {
		editorRefreshScreen()
		editorProcessKeypress()
	}

}
