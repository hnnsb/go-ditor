package editor

import "fmt"

// HelpScreen implements the ModalScreen interface for the help display
type HelpScreen struct {
	content []editorRow
}

// NewHelpScreen creates a new help screen
func NewHelpScreen(editor *Editor) *HelpScreen {
	helpContent := []string{
		"=== KIGO HELP ===",
		"",
		"NAVIGATION:",
		"  Arrow Keys       - Move cursor",
		"  Page Up/Down     - Scroll by page",
		"  Home/End         - Move to line start/end",
		"",
		"EDITING:",
		"  Ctrl+S           - Save file",
		"  Ctrl+Q           - Quit (with confirmation if unsaved)",
		"  Delete/Backspace - Delete characters",
		"",
		"SEARCH:",
		"  Ctrl+F           - Find text",
		"  Arrow Up/Down    - Navigate search results",
		"  Escape           - Cancel search",
		"",
		"FILE OPERATIONS:",
		"  Ctrl+E           - Open file explorer",
		"",
		"OTHER:",
		"  Ctrl+H           - Show this help",
		"  Ctrl+R           - Redraw screen",
		"",
		"About KIGO:",
		fmt.Sprintf("  Version: %s", KIGO_VERSION),
		"  A simple terminal-based text editor written in Go",
		"",
		"Press 'q' or Escape to close this help screen.",
	}

	// Convert help content to editor rows
	content := make([]editorRow, len(helpContent))
	for i, line := range helpContent {
		content[i] = editorRow{
			idx:   i,
			chars: []rune(line),
		}
		content[i].Update(editor)
	}

	return &HelpScreen{
		content: content,
	}
}

// GetContent returns the help content rows
func (h *HelpScreen) GetContent() []editorRow {
	return h.content
}

// GetTitle returns the help screen title
func (h *HelpScreen) GetTitle() string {
	return "Help"
}

// GetStatusMessage returns the status message for the help screen
func (h *HelpScreen) GetStatusMessage() string {
	return "Help Screen - Use Arrow Keys to scroll, 'q' or Escape to exit"
}

// Initialize sets up the initial cursor position for the help screen
func (h *HelpScreen) Initialize(e *Editor) {
	// Help screen starts at the top
	e.cy = 0
	e.rowOffset = 0
}

// HandleKey processes key presses for the help screen
func (h *HelpScreen) HandleKey(key int, e *Editor) (bool, bool) {
	switch key {
	case 'q', 'Q', '\x1b': // ESC or 'q' to quit
		return true, true // Close modal and restore previous state

	case ARROW_UP:
		if e.cy > 0 {
			e.cy--
		} else if e.rowOffset > 0 {
			e.rowOffset--
		}

	case ARROW_DOWN:
		maxCy := len(h.content) - 1
		if e.cy < e.screenRows-1 && e.cy < maxCy {
			e.cy++
		} else if e.rowOffset+e.screenRows < len(h.content) {
			e.rowOffset++
		}

	case PAGE_UP:
		for i := 0; i < e.screenRows && (e.cy > 0 || e.rowOffset > 0); i++ {
			if e.cy > 0 {
				e.cy--
			} else if e.rowOffset > 0 {
				e.rowOffset--
			}
		}

	case PAGE_DOWN:
		for i := 0; i < e.screenRows && e.rowOffset+e.cy < len(h.content)-1; i++ {
			maxCy := len(h.content) - 1
			if e.cy < e.screenRows-1 && e.cy < maxCy {
				e.cy++
			} else if e.rowOffset+e.screenRows < len(h.content) {
				e.rowOffset++
			}
		}

	case HOME_KEY:
		e.cy = 0
		e.rowOffset = 0

	case END_KEY:
		maxRows := len(h.content)
		if maxRows <= e.screenRows {
			e.cy = maxRows - 1
			e.rowOffset = 0
		} else {
			e.cy = e.screenRows - 1
			e.rowOffset = maxRows - e.screenRows
		}
	}

	return false, false // Don't close modal
}

// Help displays the help screen
func (e *Editor) Help() {
	helpScreen := NewHelpScreen(e)
	modalManager := NewModalManager(e, helpScreen)
	modalManager.Show(HELP_MODE)
}
