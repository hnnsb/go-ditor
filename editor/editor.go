package editor

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

/*** helper ***/

// Config Constants
const (
	KIGO_VERSION           = "1.0.0"
	TAB_STOP               = 4
	CONTROL_SEQUENCE_WIDTH = 2
	QUIT_TIMES             = 3
)

// getLineEnding returns the appropriate line ending for the current OS
func getLineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

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
	HL_CONTROL
)

// Syntax highlighting flags
const (
	HL_HIGHLIGHT_NUMBERS = 1 << 0
	HL_HIGHLIGHT_STRINGS = 1 << 1
)

// Editor modes
const (
	EDIT_MODE = iota
	EXPLORER_MODE
	SEARCH_MODE
	SAVE_MODE
	HELP_MODE
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
	keywords               [][]string
	singlelineCommentStart string
	multilineCommentStart  string
	multilineCommentEnd    string
	flags                  int
}

type editorRow struct {
	idx           int
	chars         []byte
	render        []byte
	hl            []int
	hlOpenComment bool
}

// Terminal handles terminal-specific operations
type Terminal struct {
	originalState *term.State
}

// Editor represents the text editor state
type Editor struct {
	cx, cy            int
	rx                int
	rowOffset         int
	colOffset         int
	screenRows        int
	screenCols        int
	totalRows         int
	row               []editorRow
	dirty             int // captures if and how much edits are made
	filename          string
	statusMessage     string
	statusMessageTime time.Time
	syntax            *editorSyntax
	mode              int // e.g., "insert", "normal", "visual"
	terminal          *Terminal
}

/*** filetypes ***/

var HLDB_ENTRIES = []editorSyntax{
	{
		filetype:  "c",
		filematch: []string{".c", ".h", ".cpp"},
		keywords: [][]string{
			{"switch", "if", "while", "for", "break", "continue", "return", "else",
				"struct", "union", "typedef", "static", "enum", "class", "case"},
			{"int", "long", "double", "float", "char", "unsigned", "signed", "void"},
		},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		flags:                  HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
	},
	{
		filetype:  "go",
		filematch: []string{".go", ".mod", ".sum"},
		keywords: [][]string{
			{"break", "case", "chan", "const", "continue", "default", "defer", "else",
				"fallthrough", "for", "go", "goto", "if", "import", "map", "package",
				"range", "return", "select", "struct", "switch", "type", "var"},
			{"interface", "func"},
		},
		singlelineCommentStart: "//",
		multilineCommentStart:  "/*",
		multilineCommentEnd:    "*/",
		flags:                  HL_HIGHLIGHT_NUMBERS | HL_HIGHLIGHT_STRINGS,
	},
}

/*** terminal ***/

// Die restores terminal, prints an error message and exits the program
func (e *Editor) Die(format string, args ...any) {
	e.RestoreTerminal()
	os.Stdout.Write([]byte(CLEAR_SCREEN))
	os.Stdout.Write([]byte(CURSOR_HOME))
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// ShowError displays an error message in the status bar instead of terminating
func (e *Editor) ShowError(format string, args ...any) {
	e.SetStatusMessage("Warn: "+format, args...)
}

// Enable raw mode for terminal input.
// This allows us to read every input key and positions the cursor freely
func (e *Editor) EnableRawMode() error {
	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("not running in a terminal")
	}

	var err error
	e.terminal.originalState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return errors.New("enabling terminal raw mode: " + err.Error())
	}
	return nil
}

// Restore the original terminal state, disabling raw mode.
func (e *Editor) RestoreTerminal() {
	if e.terminal != nil && e.terminal.originalState != nil {
		term.Restore(int(os.Stdin.Fd()), e.terminal.originalState)
		e.terminal.originalState = nil // Prevent multiple restoration attempts
	}
}

func readKey() (int, error) {
	buf := make([]byte, 1)
	var nread int
	var err error

	for nread, err = os.Stdin.Read(buf); nread != 1; {
		if nread == -1 && err != nil {
			return 0, errors.New("reading keyboard input")
		}
		if err != nil {
			return 0, errors.New("reading keyboard input")
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

func getWindowsSize() (int, int, error) {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	return rows, cols, err
}

func (e *Editor) Redraw() {
	var err error
	e.screenRows, e.screenCols, err = getWindowsSize()
	if err != nil {
		e.ShowError("%v", err)
	}
	e.screenRows -= 2 // Adjust for status bar and message bar
	e.RefreshScreen()
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

func (row *editorRow) UpdateSyntax(e *Editor) {
	row.hl = make([]int, len(row.render))

	if e.syntax == nil {
		return
	}

	keywords := e.syntax.keywords

	scs := e.syntax.singlelineCommentStart
	mcs := e.syntax.multilineCommentStart
	mce := e.syntax.multilineCommentEnd

	scsBytes := []byte(scs)
	mcsBytes := []byte(mcs)
	mceBytes := []byte(mce)

	scsLen := len(scs)
	mcsLen := len(mcs)
	mceLen := len(mce)

	prevSep := true
	var inString byte = 0
	var inComment bool = row.idx > 0 && row.idx-1 < len(e.row) && e.row[row.idx-1].hlOpenComment

	for i := 0; i < len(row.render); {
		c := row.render[i]
		prevHl := HL_NORMAL
		if i > 0 {
			prevHl = row.hl[i-1]
		}

		// Highlight control sequences like ^[ ^A ^B etc.
		if inString == 0 && !inComment && c == '^' && i+1 < len(row.render) {
			row.hl[i] = HL_CONTROL
			row.hl[i+1] = HL_CONTROL

			if i+1 < len(row.render) && row.render[i+1] == '[' {
				j := i + 2
				for j < len(row.render) {
					ch := row.render[j]
					row.hl[j] = HL_CONTROL
					j++
					// Final character (letter) ends the sequence
					if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
						break
					}
					if ch == '~' || ch == 'm' || ch == 'H' || ch == 'J' || ch == 'K' {
						break
					}
				}
				i = j
			} else {
				i += 2 // Skip both characters for simple control sequences
			}
			prevSep = true
			continue
		}

		if scsLen > 0 && inString == 0 && !inComment {
			if bytes.HasPrefix(row.render[i:], scsBytes) {
				for j := i; j < len(row.render); j++ {
					row.hl[j] = HL_COMMENT
				}
				break
			}
		}

		if mcsLen > 0 && mceLen > 0 && inString == 0 {
			if inComment {
				row.hl[i] = HL_MLCOMMENT
				if bytes.HasPrefix(row.render[i:], mceBytes) {
					for j := range mceLen {
						if i+j < len(row.render) {
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
			} else if bytes.HasPrefix(row.render[i:], mcsBytes) {
				inComment = true
				for j := range mcsLen {
					if i+j < len(row.render) {
						row.hl[i+j] = HL_MLCOMMENT
					} else {
						break // Avoid out of bounds
					}
				}
				i += mcsLen // Skip the start of the multiline comment
				continue
			}
		}

		if e.syntax.flags&HL_HIGHLIGHT_STRINGS != 0 {
			if inString != 0 {
				row.hl[i] = HL_STRING
				if c == '\\' && i+1 < len(row.render) {
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

		if e.syntax.flags&HL_HIGHLIGHT_NUMBERS != 0 {
			if (isDigit(c) && (prevSep || prevHl == HL_NUMBER)) || (c == '.' && prevHl == HL_NUMBER) {
				row.hl[i] = HL_NUMBER
				i++
				prevSep = false
				continue
			}
		}

		if prevSep {
			// we entered a new word
			for j, sublist := range keywords {
				for _, keyword := range sublist {
					klen := len(keyword)
					if bytes.HasPrefix(row.render[i:], []byte(keyword)) {
						for k := range klen {
							row.hl[i+k] = HL_KEYWORD1 + j
						}
					}
				}
			}
			// No keyword found
			prevSep = false
		} else {
			prevSep = isSeparator(int(c))
		}
		i++
	}

	changed := row.hlOpenComment != inComment
	row.hlOpenComment = inComment
	if changed && row.idx+1 < e.totalRows {
		e.row[row.idx+1].UpdateSyntax(e)
	}
}

func syntaxToGraphics(hl int) (int, int) {
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
	case HL_CONTROL:
		return ANSI_COLOR_RED, ANSI_REVERSE
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

func (e *Editor) SelectSyntaxHighlight() {
	e.syntax = nil
	if e.filename == "" {
		return
	}

	filename := e.filename
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
				e.syntax = s

				for filerow := range e.totalRows {
					e.row[filerow].UpdateSyntax(e)
				}
				return
			}
		}
	}
}

/*** row operations ***/

// Convert cursor X to render X, since rendered characters may differ from original characters (e.g., tabs)
func (row *editorRow) cxToRx(cx int) int {
	rx := 0
	for j := range cx {
		if row.chars[j] == '\t' {
			rx += TAB_STOP - (rx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else if isControl(row.chars[j]) {
			rx += CONTROL_SEQUENCE_WIDTH
		} else {
			rx++
		}
	}
	return rx
}

func (row *editorRow) rxToCx(rx int) int {
	curRx := 0
	var cx int
	for cx = 0; cx < len(row.chars); cx++ {
		if row.chars[cx] == '\t' {
			curRx += (TAB_STOP - 1) - (curRx % TAB_STOP) // Expand tab to next TAB_STOP boundary
		} else if isControl(row.chars[cx]) {
			curRx += CONTROL_SEQUENCE_WIDTH
		}
		curRx++

		if curRx > rx {
			return cx
		}
	}
	return cx
}

func (row *editorRow) Update(e *Editor) {
	tabs := 0
	controlSequences := 0
	for _, char := range row.chars {
		if char == '\t' {
			tabs++
		} else if isControl(char) {
			controlSequences++
		}
	}

	// Size: for worst case tab expansion
	row.render = make([]byte, len(row.chars)+tabs*(TAB_STOP-1)+controlSequences*(CONTROL_SEQUENCE_WIDTH-1))

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
		} else if isControl(char) {
			row.render[idx] = '^'
			idx++
			switch char {
			case 127: // DEL character
				row.render[idx] = '?'
			case '\x1b': // ESC character
				row.render[idx] = '['
			default:
				row.render[idx] = char + '@' // Convert control character to printable
			}
			idx++
		} else {
			row.render[idx] = char
			idx++
		}
	}

	row.render = row.render[:idx] // Truncate to actual size
	row.UpdateSyntax(e)
}

func (e *Editor) InsertRow(at int, s []byte, rowlen int) {
	if at < 0 || at > e.totalRows {
		return
	}

	// Create new row
	newRow := editorRow{
		idx:           at,
		chars:         slices.Clone(s[:rowlen]), // Create copy of s with specified length
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Insert row using slice operations
	e.row = append(e.row[:at], append([]editorRow{newRow}, e.row[at:]...)...)

	// Update indices for rows that were shifted
	for j := at + 1; j < e.totalRows+1; j++ {
		e.row[j].idx = j
	}

	e.row[at].Update(e)
	e.totalRows++
	e.dirty++
}

func (e *Editor) DeleteRow(at int) {
	if at < 0 || at >= e.totalRows {
		return
	}

	// Delete row using slice operations
	e.row = append(e.row[:at], e.row[at+1:]...)

	// Update indices for remaining rows
	for j := at; j < len(e.row); j++ {
		e.row[j].idx = j
	}

	e.totalRows--
	e.dirty++
}

func (row *editorRow) InsertChar(e *Editor, at int, c int) {
	if at < 0 || at > len(row.chars) {
		at = len(row.chars)
	}

	// Insert character at position using slices
	row.chars = append(row.chars[:at], append([]byte{byte(c)}, row.chars[at:]...)...)

	row.Update(e)
	e.dirty++
}

func (row *editorRow) appendBytes(e *Editor, s []byte) {
	row.chars = append(row.chars, s...)

	row.Update(e)
	e.dirty++
}

func (row *editorRow) deleteChar(e *Editor, at int) {
	if at < 0 || at >= len(row.chars) {
		return
	}

	// Delete character using slice operations
	row.chars = slices.Delete(row.chars, at, at+1)

	row.Update(e)
	e.dirty++
}

/*** editor operations ***/

func (e *Editor) InsertChar(c int) {
	if e.cy == e.totalRows {
		e.InsertRow(e.totalRows, []byte(""), 0)
	}
	e.row[e.cy].InsertChar(e, e.cx, c)
	e.cx++
}

func (e *Editor) InsertNewline() {
	if e.cx == 0 {
		e.InsertRow(e.cy, []byte(""), 0)
	} else {
		row := &e.row[e.cy]

		// Insert new row with text from cursor to end of line
		remainingText := make([]byte, len(row.chars)-e.cx)
		copy(remainingText, row.chars[e.cx:])
		e.InsertRow(e.cy+1, remainingText, len(row.chars)-e.cx)

		// Truncate current row to text before cursor
		row = &e.row[e.cy]
		row.chars = row.chars[:e.cx]
		row.Update(e)
	}
	e.cy++
	e.cx = 0
}

func (e *Editor) DeleteChar() {
	if e.cy == e.totalRows {
		return
	}
	if e.cx == 0 && e.cy == 0 {
		return
	}

	row := &e.row[e.cy]
	if e.cx > 0 {
		row.deleteChar(e, e.cx-1)
		e.cx--
	} else {
		e.cx = len(e.row[e.cy-1].chars)
		e.row[e.cy-1].appendBytes(e, row.chars)
		e.DeleteRow(e.cy) // Delete the current row after appending its content to the previous row
		e.cy--            // Move cursor up to the previous row
	}
}

/*** file i/o ***/

func (e *Editor) RowsToString() ([]byte, int) {
	var buf strings.Builder
	lineEnding := getLineEnding()

	// Pre-calculate total size for efficiency
	totalSize := 0
	for _, row := range e.row {
		totalSize += len(row.chars) + len(lineEnding) // +len(lineEnding) for line ending
	}
	buf.Grow(totalSize)

	for _, row := range e.row {
		buf.Write(row.chars)
		buf.WriteString(lineEnding)
	}

	result := buf.String()
	return []byte(result), len(result)
}

func (e *Editor) Open(filename string) error {
	e.filename = filename
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open file '%s'", filename)
	}
	defer file.Close()

	// Reset editor state, because we are opening a new file
	e.row = make([]editorRow, 0)
	e.totalRows = 0
	e.cx = 0
	e.cy = 0
	e.rowOffset = 0
	e.colOffset = 0
	e.rx = 0
	e.SelectSyntaxHighlight()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing newlines and carriage returns
		for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
			line = line[:len(line)-1]
		}

		e.InsertRow(e.totalRows, []byte(line), len(line))
	}

	if err := scanner.Err(); err != nil {
		e.Die("reading file: " + err.Error())
	}
	e.dirty = 0
	return nil
}

func (e *Editor) Save() {
	if e.filename == "" {
		e.filename = e.Prompt("Save as: %s (ESC to cancel)", nil)
		if e.filename == "" {
			e.SetStatusMessage("Save aborted")
			return
		}
		e.SelectSyntaxHighlight()
	}

	buf, length := e.RowsToString()

	// Open file for read/write, create if not exists (equivalent to O_RDWR | O_CREAT, 0644)
	file, err := os.OpenFile(e.filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		e.SetStatusMessage("Can't save! I/O error: %v", err)
		return
	}
	defer file.Close()

	// Truncate file to exact length (equivalent to ftruncate(fd, len))
	err = file.Truncate(int64(length))
	if err != nil {
		e.SetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	// Write buffer to file (equivalent to write(fd, buf, len))
	bytesWritten, err := file.Write(buf)
	if err != nil {
		e.SetStatusMessage("Can't save! I/O error: %v", err)
		return
	}

	// Check if all bytes were written
	if bytesWritten != length {
		e.SetStatusMessage("Can't save! Partial write: %d/%d bytes", bytesWritten, length)
		return
	}

	// Success message with byte count (equivalent to C version's success case)
	e.SetStatusMessage("%d bytes written to disk", length)
	e.dirty = 0 // Reset dirty flag after successful save
}

/*** find ***/

var (
	lastMatch   = -1
	direction   = 1
	savedHlLine int
	savedHl     []int = nil
)

func (e *Editor) FindCallback(query []byte, key int) {

	if savedHl != nil {
		// Restore previous highlights
		copy(e.row[savedHlLine].hl, savedHl)
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

	for range e.totalRows {
		current += direction
		if current == -1 {
			current = e.totalRows - 1
		} else if current == e.totalRows {
			current = 0
		}

		row := &e.row[current]
		match := bytes.Index(row.render, query)
		if match != -1 {
			lastMatch = current
			e.cy = current
			e.cx = row.rxToCx(match)
			e.rowOffset = e.totalRows

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

func (e *Editor) Find() {
	savedCx := e.cx
	savedCy := e.cy
	savedColOffset := e.colOffset
	savedRowOffset := e.rowOffset

	query := e.Prompt("Search: %s (Use ESC/Arrows/Enter)", e.FindCallback)

	if query == "" {
		e.cx = savedCx
		e.cy = savedCy
		e.colOffset = savedColOffset
		e.rowOffset = savedRowOffset
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

/*** output ***/

func (e *Editor) Scroll() {
	e.rx = 0
	if e.cy < e.totalRows {
		e.rx = e.row[e.cy].cxToRx(e.cx)
	}

	if e.cy < e.rowOffset {
		e.rowOffset = e.cy
	}
	if e.cy >= e.rowOffset+e.screenRows {
		e.rowOffset = e.cy - e.screenRows + 1
	}

	if e.rx < e.colOffset {
		e.colOffset = e.rx
	}
	if e.rx >= e.colOffset+e.screenCols {
		e.colOffset = e.rx - e.screenCols + 1
	}
}

func (e *Editor) DrawRows(abuf *appendBuffer) {
	for y := range e.screenRows {
		filerow := y + e.rowOffset
		if filerow >= e.totalRows {
			if e.totalRows == 0 && y == e.screenRows/3 {
				welcome := "KIGO editor -- version " + KIGO_VERSION
				welcomelen := min(len(welcome), e.screenCols)
				padding := (e.screenCols - welcomelen) / 2
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
			lineLen := min(max(len(e.row[filerow].render)-e.colOffset, 0), e.screenCols)
			// Character-by-character rendering with syntax highlighting
			start := e.colOffset
			hl := e.row[filerow].hl
			render := e.row[filerow].render
			currentColor := -1
			currentStyle := 0
			for j := range lineLen {
				c := render[start+j]
				h := hl[start+j]
				if h == HL_NORMAL {
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
					color, style := syntaxToGraphics(h)

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

func (e *Editor) DrawStatusBar(abuf *appendBuffer) {
	abuf.append([]byte(COLORS_INVERT)) // Invert colors for status bar

	var status string
	var rstatus string
	filename := "[No Name]"
	if e.filename != "" {
		filename = e.filename
		// Truncate filename to 20 characters if needed
		if len(filename) > 20 {
			filename = filename[:20]
		}
	}
	dirtyFlag := ""
	if e.dirty > 0 {
		dirtyFlag = "(modified)"
	}
	switch e.mode {
	case EXPLORER_MODE:
		status = fmt.Sprintf("Explorer - %s %s", filename, dirtyFlag)
	default:
		status = fmt.Sprintf("%.20s - %d lines %s %d", filename, e.totalRows, dirtyFlag, e.dirty)
	}
	statusLen := min(len(status), e.screenCols)

	filetype := "no ft"
	if e.syntax != nil {
		filetype = e.syntax.filetype
	}
	rstatus = fmt.Sprintf("%s | %d/%d", filetype, e.cy+1, e.totalRows)
	rstatusLen := len(rstatus)
	abuf.append([]byte(status[:statusLen]))

	for statusLen < e.screenCols {
		if e.screenCols-statusLen == rstatusLen {
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

func (e *Editor) DrawMessageBar(abuf *appendBuffer) {
	abuf.append([]byte(CLEAR_LINE))
	messageLen := min(len(e.statusMessage), e.screenCols)
	if time.Since(e.statusMessageTime) < 5*time.Second {
		abuf.append([]byte(e.statusMessage[:messageLen]))
	}
}

func (e *Editor) RefreshScreen() {
	e.Scroll()

	var abuf appendBuffer

	abuf.append([]byte(CURSOR_HIDE))
	abuf.append([]byte(CURSOR_HOME)) // Move cursor to the top-left corner

	e.DrawRows(&abuf)
	e.DrawStatusBar(&abuf)
	e.DrawMessageBar(&abuf)

	abuf.append(fmt.Appendf(nil, CURSOR_POSITION_FORMAT, e.cy-e.rowOffset+1, e.rx-e.colOffset+1))

	abuf.append([]byte(CURSOR_SHOW))

	os.Stdout.Write(abuf.b)
}

func (e *Editor) SetStatusMessage(format string, args ...any) {
	e.statusMessage = fmt.Sprintf(format, args...)
	e.statusMessageTime = time.Now()
}

/*** input ***/

func (e *Editor) Prompt(prompt string, callback func([]byte, int)) string {
	bufSize := 128
	buf := make([]byte, 0, bufSize)

	for {
		e.SetStatusMessage(prompt, string(buf))
		e.RefreshScreen()

		key, err := readKey()
		if err != nil {
			e.ShowError("%v", err)
			continue // Try again instead of terminating
		}

		switch key {
		case DELETE_KEY, BACKSPACE, withControlKey('h'):
			if len(buf) != 0 {
				buf = buf[:len(buf)-1]
			}

		case '\x1b':
			e.SetStatusMessage("")
			if callback != nil {
				callback(buf, key)
			}
			return ""

		case '\r':
			if len(buf) != 0 {
				e.SetStatusMessage("")
				if callback != nil {
					callback(buf, key)
				}
				return string(buf)
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

func (e *Editor) MoveCursor(key int) {
	var row *editorRow
	if e.cy >= e.totalRows {
		row = nil
	} else {
		row = &e.row[e.cy]
	}

	switch key {
	case ARROW_LEFT:
		if e.cx != 0 {
			e.cx--
		} else if e.cy > 0 {
			e.cy--
			e.cx = len(e.row[e.cy].chars)
		}
	case ARROW_RIGHT:
		if row != nil && e.cx < len(row.chars) {
			e.cx++
		} else if row != nil && e.cx == len(row.chars) {
			e.cy++
			e.cx = 0
		}
	case ARROW_UP:
		if e.cy != 0 {
			e.cy--
		}
	case ARROW_DOWN:
		if e.cy < e.totalRows {
			e.cy++
		}
	}

	if e.cy >= e.totalRows {
		row = nil
	} else {
		row = &e.row[e.cy]
	}
	rowlen := 0
	if row != nil {
		rowlen = len(row.chars)
	}
	if e.cx > rowlen {
		e.cx = rowlen
	}
}

var quitTimes = QUIT_TIMES

func (e *Editor) ProcessKeypress() {

	key, err := readKey()
	if err != nil {
		e.ShowError("%v", err)
		return // Skip this keypress and continue
	}

	switch key {
	case '\r':
		e.InsertNewline()

	case withControlKey('q'):
		if e.dirty > 0 && quitTimes > 0 {
			e.SetStatusMessage("WARNING: File has unsaved changes. Press Ctrl-Q %d more times to quit.", quitTimes)
			quitTimes--
			return
		}

		e.RestoreTerminal()
		os.Stdout.Write([]byte(CLEAR_SCREEN))
		os.Stdout.Write([]byte(CURSOR_HOME))
		fmt.Println("Exiting KIGO editor")
		os.Exit(0)

	case withControlKey('s'):
		e.Save()

	case HOME_KEY:
		e.cx = 0

	case END_KEY:
		if e.cy < e.totalRows {
			e.cx = len(e.row[e.cy].chars)
		}

	case withControlKey('e'):
		e.Explorer()
		e.mode = EDIT_MODE

	case withControlKey('f'):
		e.Find()

	case withControlKey('r'):
		e.Redraw()

	case withControlKey('h'):
		e.Help()

	case BACKSPACE, DELETE_KEY:
		if key == DELETE_KEY {
			e.MoveCursor(ARROW_RIGHT)
		}
		e.DeleteChar()

	case PAGE_UP:
		e.cy = e.rowOffset
		for range e.screenRows {
			e.MoveCursor(ARROW_UP)
		}

	case PAGE_DOWN:
		e.cy = min(e.rowOffset+e.screenRows-1, e.totalRows)
		for range e.screenRows {
			e.MoveCursor(ARROW_DOWN)
		}

	case ARROW_LEFT, ARROW_RIGHT, ARROW_UP, ARROW_DOWN:
		e.MoveCursor(key)

	case withControlKey('l'):
	case '\x1b':
		break

	default:
		e.InsertChar(key)
	}

	quitTimes = QUIT_TIMES // Reset quit times after processing a key
}

/*** init ***/

// NewTerminal creates a new Terminal instance
func NewTerminal() *Terminal {
	return &Terminal{}
}

// NewEditor creates a new Editor instance with proper initialization
func NewEditor() Editor {
	return Editor{
		terminal: NewTerminal(),
	}
}

func (e *Editor) Init() error {
	e.cx, e.cy = 0, 0
	e.rx = 0
	e.rowOffset = 0
	e.colOffset = 0
	e.totalRows = 0
	e.row = make([]editorRow, 0)
	e.dirty = 0
	e.filename = ""
	e.statusMessage = ""
	e.statusMessageTime = time.Time{}
	e.syntax = nil
	e.mode = EDIT_MODE

	var err error
	e.screenRows, e.screenCols, err = getWindowsSize()
	if err != nil {
		return errors.New("getting window size")
	}
	e.screenRows -= 2
	return nil
}
