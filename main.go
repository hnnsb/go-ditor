package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

/*** helper ***/

// Config Constants
const (
	GO_DITOR_VERSION = "0.0.1"
	TAB_STOP         = 4
	QUIT_TIMES       = 3
)

// Key aliase
const (
	BACKSPACE  = 127 // ASCII backspace
	ARROW_LEFT = iota + 1000
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DELETE_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
)

// Syntax highlighting types
const (
	HL_NORMAL = iota
	HL_COMMENT
	HL_MLCOMMENT
	HL_KEYWORD1
	HL_KEYWORD2
	HL_STRING
	HL_NUMBER
	HL_MATCH
)

// Syntax highlighting flags
const (
	HL_HIGHLIGHT_NUMBERS = 1 << 0
	HL_HIGHLIGHT_STRINGS = 1 << 1
)

// Check if the byte is a control character
func isControl(c byte) bool {
	return c < 32 || c == 127
}

// Check if the byte is a digit character
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// Convert a character to its control key equivalent
func withControlKey(c int) int {
	return c & 0x1f // 0x1f is 31 in decimal, which is the control character range
}

/*** data ***/

type editorSyntax struct {
	filetype               string
	filematch              []string
	keywords               []string
	singlelineCommentStart string
	multilineCommentStart  string
	multilineCommentEnd    string
	flags                  int
}

type erow struct {
	idx           int
	size          int
	chars         []byte
	rsize         int
	render        []byte
	hl            []int
	hlOpenComment bool
}

type editorConfig struct {
	cx, cy            int
	rx                int
	rowOffset         int
	colOffset         int
	screenRows        int
	screenCols        int
	totalRows         int
	row               []erow
	dirty             int // captures if and how much edits are made
	filename          *string
	statusMessage     string
	statusMessageTime time.Time
	syntax            *editorSyntax
	originalState     *term.State
}

var (
	E editorConfig // Global editor configuration
)

/*** filetypes ***/

var HLDB_ENTRIES = []editorSyntax{
	{
		filetype:  "c",
		filematch: []string{".c", ".h", ".cpp"},
		keywords: []string{
			"switch", "if", "while", "for", "break", "continue", "return", "else",
			"struct", "union", "typedef", "static", "enum", "class", "case",
			"int|", "long|", "double|", "float|", "char|", "unsigned|", "signed|",
			"void|"},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		flags:                  HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
	},
	{
		filetype:  "go",
		filematch: []string{".go", ".mod", ".sum"},
		keywords: []string{
			"break", "case", "chan", "const", "continue", "default", "defer", "else",
			"fallthrough", "for", "func|", "go", "goto", "if", "import", "interface",
			"map", "package", "range", "return", "select", "struct", "switch", "type",
			"var"},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		flags:                  HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
	},
}

/*** terminal ***/

// die prints an error message and exits the program
func die(s string) {
	restoreTerminal()
	os.Stdout.Write([]byte(CLEAR_SCREEN))
	os.Stdout.Write([]byte(CURSOR_HOME))
	fmt.Fprintf(os.Stderr, "Error: %s\n", s)
	os.Exit(1)
}

// Enable raw mode for terminal input.
// This allows us to read every input key and positions the cursor freely
func enableRawMode() error {
	var err error
	E.originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	return err
}

// Restore the original terminal state, disabling raw mode.
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

/*** syntax highlighting ***/

// Check if the character is a separator (whitespace, null, or punctuation)
func isSeparator(c int) bool {
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' || c == 0 {
		return true
	}
	// Check for common programming punctuation separators
	separators := ",.()+-/*=~%<>[];"
	for _, sep := range separators {
		if c == int(sep) {
			return true
		}
	}
	return false
}

func editorUpdateSyntax(row *erow) {
	row.hl = make([]int, row.rsize)
	for i := range row.hl {
		row.hl[i] = HL_NORMAL // Default to normal highlighting
	}

	if E.syntax == nil {
		return
	}

	keywords := E.syntax.keywords

	scs := E.syntax.singlelineCommentStart
	mcs := E.syntax.multilineCommentStart
	mce := E.syntax.multilineCommentEnd

	scsLen := len(scs)
	mcsLen := len(mcs)
	mceLen := len(mce)

	prevSep := true
	var inString byte = 0
	var inComment bool = row.idx > 0 && E.row[row.idx-1].hlOpenComment

	for i := 0; i < row.rsize; {
		c := row.render[i]
		prevHl := HL_NORMAL
		if i > 0 {
			prevHl = row.hl[i-1]
		}

		if scsLen > 0 && inString == 0 && !inComment {
			if bytes.HasPrefix(row.render[i:], []byte(scs)) {
				for j := i; j < row.rsize; j++ {
					row.hl[j] = HL_COMMENT
				}
				break
			}
		}

		if mcsLen > 0 && mceLen > 0 && inString == 0 {
			if inComment {
				row.hl[i] = HL_MLCOMMENT
				if bytes.HasPrefix(row.render[i:], []byte(mce)) {
					for j := range mceLen {
						if i+j < row.rsize {
							row.hl[i+j] = HL_MLCOMMENT
						} else {
							break
						}
					}
					inComment = false
					i += mceLen // Skip the end of the multiline comment
					continue
				}
				i++ // Continue in the multiline comment
				continue
			} else if bytes.HasPrefix(row.render[i:], []byte(mcs)) {
				inComment = true
				for j := range mcsLen {
					if i+j < row.rsize {
						row.hl[i+j] = HL_MLCOMMENT
					} else {
						break // Avoid out of bounds
					}
				}
				i += mcsLen // Skip the start of the multiline comment
				continue
			}
		}

		if E.syntax.flags&HL_HIGHLIGHT_STRINGS != 0 {
			if inString != 0 {
				row.hl[i] = HL_STRING
				if c == '\\' && i+1 < row.rsize {
					row.hl[i+1] = HL_STRING
					i += 2
					continue
				}
				if c == inString {
					inString = 0 // End of string
				}
				i++
				prevSep = true
				continue
			} else {
				if c == '"' || c == '\'' {
					inString = c
					row.hl[i] = HL_STRING
					i++
					continue
				}
			}
		}

		if E.syntax.flags&HL_HIGHLIGHT_NUMBERS != 0 {
			if (isDigit(c) && (prevSep || prevHl == HL_NUMBER)) || (c == '.' && prevHl == HL_NUMBER) {
				row.hl[i] = HL_NUMBER
				i++
				prevSep = false
				continue
			}
		}

		if prevSep {
			// we entered a new word
			j := 0
			for j < len(keywords) {
				klen := len(keywords[j])
				isKw2 := false
				if klen > 0 && keywords[j][klen-1] == '|' {
					isKw2 = true
					klen-- // Exclude the trailing '|'
				}

				if klen > 0 && i+klen <= row.rsize &&
					bytes.Equal(row.render[i:i+klen], []byte(keywords[j][:klen])) &&
					(i+klen >= row.rsize || isSeparator(int(row.render[i+klen]))) {
					for k := range klen {
						if isKw2 {
							row.hl[i+k] = HL_KEYWORD2
						} else {
							row.hl[i+k] = HL_KEYWORD1
						}
					}
					i += klen
					break
				}
				j++
			}
			if j < len(keywords) {
				prevSep = false
				continue
			}
		}

		prevSep = isSeparator(int(c))
		i++
	}

	changed := row.hlOpenComment != inComment
	row.hlOpenComment = inComment
	if changed && row.idx+1 < E.totalRows {
		editorUpdateSyntax(&E.row[row.idx+1])
	}
}

func editorSyntaxToGraphics(hl int) (int, int) {
	switch hl {
	case HL_COMMENT, HL_MLCOMMENT:
		return ANSI_COLOR_CYAN, 0
	case HL_KEYWORD1:
		return ANSI_COLOR_YELLOW, 0
	case HL_KEYWORD2:
		return ANSI_COLOR_GREEN, 0
	case HL_STRING:
		return ANSI_COLOR_MAGENTA, 0
	case HL_NUMBER:
		return ANSI_COLOR_RED, 0
	case HL_MATCH:
		return ANSI_COLOR_BLUE, ANSI_REVERSE
	default:
		return ANSI_COLOR_DEFAULT, 0
	}
}

// Get the appropriate reset code for a given style
func getStyleResetCode(style int) int {
	if resetCode, exists := styleResetCodes[style]; exists {
		return resetCode
	}
	return 0
}

func editorSelectSyntaxHighlight() {
	E.syntax = nil
	if E.filename == nil {
		return
	}

	filename := *E.filename
	var ext string
	if lastDot := strings.LastIndex(filename, "."); lastDot != -1 {
		ext = filename[lastDot:]
	}

	for j := range HLDB_ENTRIES {
		s := &HLDB_ENTRIES[j]
		for i := range s.filematch {
			pattern := s.filematch[i]

			isExt := pattern[0] == '.'
			if (isExt && ext != "" && ext == pattern) ||
				(!isExt && strings.Contains(filename, pattern)) {
				E.syntax = s

				for filerow := range E.totalRows {
					editorUpdateSyntax(&E.row[filerow])
				}
				return
			}
		}
	}
}

/*** row operations ***/

// Convert cursor X to render X, since rendered characters may differ from original characters (e.g., tabs)
func editorRowCxToRx(row *erow, cx int) int {
	rx := 0
	for j := range cx {
		if row.chars[j] == '\t' {
			rx += TAB_STOP - (rx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else {
			rx++
		}
	}
	return rx
}

func editorRowRxToCx(row *erow, rx int) int {
	curRx := 0
	var cx int
	for cx = 0; cx < row.size; cx++ {
		if row.chars[cx] == '\t' {
			curRx += (TAB_STOP - 1) - (curRx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		}
		curRx++

		if curRx > rx {
			return cx
		}
	}
	return cx
}

func editorUpdateRow(row *erow) {
	tabs := 0
	for _, char := range row.chars {
		if char == '\t' {
			tabs++
		}
	}

	// Size: for worst case tab expansion
	row.render = make([]byte, row.size+tabs*(TAB_STOP-1))

	idx := 0
	for _, char := range row.chars {
		if char == '\t' {
			row.render[idx] = ' '
			idx++
			// Add spaces until we reach the next TAB_STOP boundary
			for idx%TAB_STOP != 0 {
				row.render[idx] = ' '
				idx++
			}
		} else {
			row.render[idx] = char
			idx++
		}
	}

	row.rsize = idx
	editorUpdateSyntax(row)
}

func editorInsertRow(at int, s []byte, rowlen int) {
	if at < 0 || at > E.totalRows {
		return
	}

	E.row = append(E.row, erow{})
	copy(E.row[at+1:], E.row[at:E.totalRows])
	for j := at + 1; j < E.totalRows; j++ {
		E.row[j].idx++
	}

	E.row[at].idx = at

	E.row[at].size = rowlen
	E.row[at].chars = make([]byte, rowlen)
	copy(E.row[at].chars, s)

	E.row[at].rsize = 0
	E.row[at].render = nil
	E.row[at].hl = nil
	E.row[at].hlOpenComment = false

	editorUpdateRow(&E.row[at])
	E.totalRows++
	E.dirty++
}

func editorFreeRow(erow *erow) {
	if erow == nil {
		return
	}
	erow.chars = nil
	erow.render = nil
	erow.hl = nil
}

func editorDeleteRow(at int) {
	if at < 0 || at >= E.totalRows {
		return
	}

	// Free the row's resources
	editorFreeRow(&E.row[at])

	// Shift rows down to fill the gap
	copy(E.row[at:], E.row[at+1:E.totalRows])
	E.row = E.row[:E.totalRows-1] // Resize the slice

	for j := at; j < E.totalRows-1; j++ {
		E.row[j].idx--
	}

	E.totalRows--
	E.dirty++
}

func editorRowInsertChar(erow *erow, at int, c int) {
	if at < 0 || at > erow.size {
		at = erow.size
	}

	// Grow the slice to accommodate the new character
	erow.chars = append(erow.chars, 0) // Add space for one more character

	// Shift characters to the right to make room for insertion
	copy(erow.chars[at+1:], erow.chars[at:erow.size])

	erow.chars[at] = byte(c)
	erow.size++

	editorUpdateRow(erow)
	E.dirty++
}

func editorRowAppendString(erow *erow, s []byte, slen int) {
	newSize := erow.size + slen
	newChars := make([]byte, newSize)

	copy(newChars[:erow.size], erow.chars[:erow.size])
	copy(newChars[erow.size:], s[:slen])

	erow.chars = newChars
	erow.size = newSize

	editorUpdateRow(erow)
	E.dirty++
}

func editorRowDeleteChar(erow *erow, at int) {
	if at < 0 || at >= erow.size {
		return
	}

	// Shift characters to the left to overwrite the deleted character
	copy(erow.chars[at:], erow.chars[at+1:erow.size])
	erow.size--

	editorUpdateRow(erow)
	E.dirty++
}

/*** editor operations ***/

func editorInsertChar(c int) {
	if E.cy == E.totalRows {
		editorInsertRow(E.totalRows, []byte(""), 0)
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
		row = &E.row[E.cy]
		row.size = E.cx
		row.chars = row.chars[:E.cx]
		editorUpdateRow(row)
	}
	E.cy++
	E.cx = 0
}

func editorDeleteChar() {
	if E.cy == E.totalRows {
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
	for _, row := range E.row {
		totalLength += row.size + 1 // +1 for newline character
	}
	*bufLen = totalLength

	buf := make([]byte, totalLength)
	p := 0

	for _, row := range E.row {
		copy(buf[p:p+row.size], row.chars[:row.size])
		p += row.size
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

	editorSelectSyntaxHighlight()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing newlines and carriage returns
		for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
			line = line[:len(line)-1]
		}

		editorInsertRow(E.totalRows, []byte(line), len(line))
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
		editorSelectSyntaxHighlight()
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
	lastMatch   = -1
	direction   = 1
	savedHlLine int
	savedHl     []int = nil
)

func editorFindCallback(query []byte, key int) {

	if savedHl != nil {
		// Restore previous highlights
		copy(E.row[savedHlLine].hl, savedHl)
		savedHl = nil
	}

	switch key {
	case '\r', '\x1b':
		lastMatch = -1
		direction = 1
		return
	case ARROW_RIGHT, ARROW_DOWN:
		direction = 1
	case ARROW_LEFT, ARROW_UP:
		direction = -1
	default:
		lastMatch = -1
		direction = 1
	}

	if lastMatch == -1 {
		direction = 1
	}
	current := lastMatch

	for range E.totalRows {
		current += direction
		if current == -1 {
			current = E.totalRows - 1
		} else if current == E.totalRows {
			current = 0
		}

		row := &E.row[current]
		match := bytes.Index(row.render, query)
		if match != -1 {
			lastMatch = current
			E.cy = current
			E.cx = editorRowRxToCx(row, match)
			E.rowOffset = E.totalRows

			savedHlLine = current
			savedHl = make([]int, len(row.hl))
			copy(savedHl, row.hl)
			// Highlight the match
			for k := match; k < match+len(query) && k < len(row.hl); k++ {
				row.hl[k] = HL_MATCH
			}
			break
		}
	}
}

func editorFind() {
	savedCx := E.cx
	savedCy := E.cy
	savedColOffset := E.colOffset
	savedRowOffset := E.rowOffset

	query := editorPrompt("Search: %s (Use ESC/Arrows/Enter)", editorFindCallback)

	if query == nil {
		E.cx = savedCx
		E.cy = savedCy
		E.colOffset = savedColOffset
		E.rowOffset = savedRowOffset
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
	if E.cy < E.totalRows {
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
		if filerow >= E.totalRows {
			if E.totalRows == 0 && y == E.screenRows/3 {
				welcome := "GO-DITOR editor -- version " + GO_DITOR_VERSION
				welcomelen := min(len(welcome), E.screenCols)
				padding := (E.screenCols - welcomelen) / 2
				if padding > 0 {
					abuf.append([]byte("~"))
					padding--
				}
				for range padding {
					abuf.append([]byte(" "))
				}
				abuf.append([]byte(welcome[:welcomelen]))
			} else {
				abuf.append([]byte("~"))
			}
		} else {
			lineLen := min(max(E.row[filerow].rsize-E.colOffset, 0), E.screenCols)
			// Character-by-character rendering with syntax highlighting
			start := E.colOffset
			hl := E.row[filerow].hl
			render := E.row[filerow].render
			currentColor := -1
			currentStyle := 0
			for j := range lineLen {
				c := render[start+j]
				h := hl[start+j]
				if isControl(c) {
					sym := "?"
					if c <= 26 {
						sym = "@" + string(c+'A'-1) // Convert control character to symbol
					}
					abuf.append([]byte("\x1b[7m"))
					abuf.append([]byte(sym))
					abuf.append([]byte("\x1b[m"))
					if currentColor != -1 {
						abuf.append(fmt.Appendf(nil, "\x1b[%dm", currentColor))
					}
				} else if h == HL_NORMAL {
					// Reset both color and style for normal text
					if currentColor != -1 {
						abuf.append(fmt.Appendf(nil, "\x1b[%dm", ANSI_COLOR_DEFAULT))
						currentColor = -1
					}
					if currentStyle != 0 {
						resetCode := getStyleResetCode(currentStyle)
						if resetCode != 0 {
							abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
						}
						currentStyle = 0
					}
					abuf.append([]byte{c})
				} else {
					// Get both color and style from the combined function
					color, style := editorSyntaxToGraphics(h)

					// Apply style if different from current
					if currentStyle != style {
						// Reset previous style if it was set and not normal
						if currentStyle != 0 {
							resetCode := getStyleResetCode(currentStyle)
							if resetCode != 0 {
								abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
							}
						}
						// Apply new style if not normal
						if style != 0 {
							abuf.append(fmt.Appendf(nil, "\x1b[%dm", style))
						}
						currentStyle = style
					}

					// Apply color if different from current
					if color != currentColor {
						currentColor = color
						abuf.append(fmt.Appendf(nil, "\x1b[%dm", color))
					}
					abuf.append([]byte{c})
				}
			}
			// Reset all formatting at end of line
			abuf.append(fmt.Appendf(nil, "\x1b[%dm", ANSI_COLOR_DEFAULT))
			if currentStyle != 0 {
				resetCode := getStyleResetCode(currentStyle)
				if resetCode != 0 {
					abuf.append(fmt.Appendf(nil, "\x1b[%dm", resetCode))
				}
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
	status = fmt.Sprintf("%.20s - %d lines %s %d", filename, E.totalRows, dirtyFlag, E.dirty)
	statusLen := min(len(status), E.screenCols)

	filetype := "no ft"
	if E.syntax != nil {
		filetype = E.syntax.filetype
	}
	rstatus = fmt.Sprintf("%s | %d/%d", filetype, E.cy+1, E.totalRows)
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

	abuf.append([]byte(COLORS_RESET))
	abuf.append([]byte("\r\n"))
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

	abuf.append([]byte(CURSOR_HIDE))
	abuf.append([]byte(CURSOR_HOME)) // Move cursor to the top-left corner

	editorDrawRows(&abuf)
	editorDrawStatusBar(&abuf)
	editorDrawMessageBar(&abuf)

	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, E.cy-E.rowOffset+1, E.rx-E.colOffset+1))

	abuf.append([]byte(CURSOR_SHOW))

	os.Stdout.Write(abuf.b)
	abuf.free()
}

func editorSetStatusMessage(format string, args ...any) {
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
		case DELETE_KEY, BACKSPACE, withControlKey('h'):
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
	var row *erow
	if E.cy >= E.totalRows {
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
		if E.cy < E.totalRows {
			E.cy++
		}
	}

	if E.cy >= E.totalRows {
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

var quitTimes = QUIT_TIMES

func editorProcessKeypress() {

	key, err := editorReadKey()
	if err != nil {
		die("reading key")
	}

	switch key {
	case '\r':
		editorInsertNewline()

	case withControlKey('q'):
		if E.dirty > 0 && quitTimes > 0 {
			editorSetStatusMessage("WARNING: File has unsaved changes. Press Ctrl-Q %d more times to quit.", quitTimes)
			quitTimes--
			return
		}

		restoreTerminal()
		os.Stdout.Write([]byte(CLEAR_SCREEN))
		os.Stdout.Write([]byte(CURSOR_HOME))
		fmt.Println("Exiting GO-DITOR editor")
		os.Exit(0)

	case withControlKey('s'):
		editorSave()

	case HOME_KEY:
		E.cx = 0

	case END_KEY:
		if E.cy < E.totalRows {
			E.cx = E.row[E.cy].size
		}

	case withControlKey('f'):
		editorFind()

	case BACKSPACE, withControlKey('h'), DELETE_KEY:
		if key == DELETE_KEY {
			editorMoveCursor(ARROW_RIGHT)
		}
		editorDeleteChar()

	case PAGE_UP:
		E.cy = E.rowOffset
		for range E.screenRows {
			editorMoveCursor(ARROW_UP)
		}

	case PAGE_DOWN:
		E.cy = min(E.rowOffset+E.screenRows-1, E.totalRows)
		for range E.screenRows {
			editorMoveCursor(ARROW_DOWN)
		}

	case ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ARROW_DOWN:
		editorMoveCursor(key)

	case withControlKey('l'):
	case '\x1b':
		break

	default:
		editorInsertChar(key)
	}

	quitTimes = QUIT_TIMES // Reset quit times after processing a key
}

/*** init ***/

func initEditor() {
	E.cx, E.cy = 0, 0
	E.rx = 0
	E.rowOffset = 0
	E.colOffset = 0
	E.totalRows = 0
	E.row = make([]erow, 0)
	E.dirty = 0
	E.filename = nil
	E.statusMessage = ""
	E.statusMessageTime = time.Time{}
	E.syntax = nil

	if getWindowsSize(&E.screenRows, &E.screenCols) != nil {
		die("getting window size")
	}
	E.screenRows -= 2
}

func main() {
	args := os.Args[1:]
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
