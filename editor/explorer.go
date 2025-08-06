package editor

import (
	"fmt"
	"os"
	"strings"
)

// EditorState represents the saved state of the editor
type EditorState struct {
	rows      []editorRow
	totalRows int
	cx, cy    int
	colOffset int
	rowOffset int
}

// getEditorState creates a snapshot of the current editor state
func (e *Editor) getEditorState() EditorState {
	return EditorState{
		rows:      e.row,
		totalRows: e.totalRows,
		cx:        e.cx,
		cy:        e.cy,
		colOffset: e.colOffset,
		rowOffset: e.rowOffset,
	}
}

// setEditorState restores the editor to a previously saved state
func (e *Editor) setEditorState(state EditorState) {
	e.row = state.rows
	e.totalRows = state.totalRows
	e.cx = state.cx
	e.cy = state.cy
	e.colOffset = state.colOffset
	e.rowOffset = state.rowOffset
	e.mode = EDIT_MODE
}

// ExplorerScreen implements the ModalScreen interface for file exploration
type ExplorerScreen struct {
	currentDir   string
	files        []os.DirEntry
	hasParentDir bool
	content      []editorRow
	editor       *Editor
}

// NewExplorerScreen creates a new explorer screen
func NewExplorerScreen(editor *Editor, startDir string) *ExplorerScreen {
	explorer := &ExplorerScreen{
		currentDir: startDir,
		editor:     editor,
	}
	err := explorer.refreshContent()
	if err != nil {
		editor.ShowError("Failed to read directory: %v", err)
		return nil
	}
	return explorer
}

// refreshContent updates the explorer content for the current directory
func (ex *ExplorerScreen) refreshContent() error {
	// Read current directory contents
	files, err := os.ReadDir(ex.currentDir)
	if err != nil {
		return err
	}

	ex.files = files
	ex.hasParentDir = ex.currentDir != "." && ex.currentDir != "/"

	// Create content rows
	ex.content = ex.createExplorerRows(files, ex.currentDir)

	return nil
}

// createExplorerRows creates all the display rows for the file explorer
func (ex *ExplorerScreen) createExplorerRows(files []os.DirEntry, currentDir string) []editorRow {
	explorerRows := make([]editorRow, 0, len(files)+2)

	// Add header
	headerText := fmt.Sprintf("=== File Explorer: %s ===", currentDir)
	headerRow := editorRow{
		idx:   0,
		chars: []rune(headerText),
	}
	headerRow.Update(ex.editor)
	explorerRows = append(explorerRows, headerRow)

	// Add parent directory option (unless we're at root)
	if ex.hasParentDir {
		parentText := "📂 .. (parent directory)"
		parentRow := editorRow{
			idx:   1,
			chars: []rune(parentText),
		}
		parentRow.Update(ex.editor)
		explorerRows = append(explorerRows, parentRow)
	}

	// Add files
	for i, file := range files {
		fileRow := ex.createFileDisplayRow(i, file)
		fileRow.Update(ex.editor)
		explorerRows = append(explorerRows, fileRow)
	}

	return explorerRows
}

// createFileDisplayRow creates a formatted display row for a file or directory
func (ex *ExplorerScreen) createFileDisplayRow(index int, file os.DirEntry) editorRow {
	var fileInfo string
	if file.IsDir() {
		fileInfo = fmt.Sprintf("📁 %s/", file.Name())
	} else {
		info, _ := file.Info()
		size := ""
		if info != nil {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		fileInfo = fmt.Sprintf("📄 %s%s", file.Name(), size)
	}

	return editorRow{
		idx:   index + 2, // +2 to account for header and potential parent dir option
		chars: []rune(fileInfo),
	}
}

// GetContent returns the explorer content rows
func (ex *ExplorerScreen) GetContent() []editorRow {
	return ex.content
}

// GetTitle returns the explorer screen title
func (ex *ExplorerScreen) GetTitle() string {
	return "File Explorer"
}

// GetStatusMessage returns the status message for the explorer screen
func (ex *ExplorerScreen) GetStatusMessage() string {
	return fmt.Sprintf("File Explorer: %s - %d items (Enter=open/navigate, ESC/q=quit)", ex.currentDir, len(ex.files))
}

// Initialize sets up the initial cursor position for the explorer
func (ex *ExplorerScreen) Initialize(e *Editor) {
	// Start at first file (skip header and optionally parent dir)
	if ex.hasParentDir {
		e.cy = 2 // Skip header and parent dir option
	} else {
		e.cy = 1 // Skip only header
	}
	ex.highlightSelectedFile(e)
}

// HandleKey processes key presses for the explorer screen
func (ex *ExplorerScreen) HandleKey(key int, e *Editor) (bool, bool) {
	switch key {
	case 'q', 'Q', '\x1b': // ESC or 'q' to quit
		return true, true // Close modal and restore previous state

	case ARROW_UP, ARROW_DOWN:
		ex.handleExplorerNavigation(key, e)
		ex.highlightSelectedFile(e)

	case '\r': // Enter key
		opened := ex.openSelectedFile(e)
		if opened {
			return true, false // Close modal but keep new file state (don't restore)
		}
		// Directory was changed, update display with new cursor position
		if ex.hasParentDir {
			e.cy = 2 // Skip header and parent dir option
		} else {
			e.cy = 1 // Skip only header
		}
		e.rowOffset = 0
		// Update the editor's row content with new directory content
		e.row = ex.content
		e.totalRows = len(ex.content)
		// Update status message
		e.SetStatusMessage("%s", ex.GetStatusMessage())
	}

	return false, false // Don't close modal
}

// handleExplorerNavigation handles arrow key navigation in the explorer
func (ex *ExplorerScreen) handleExplorerNavigation(key int, e *Editor) {
	minCy := 1 // Start after header
	if ex.hasParentDir {
		minCy = 1 // Can navigate to parent dir option
	}

	maxItems := len(ex.files)
	if ex.hasParentDir {
		maxItems++ // Add parent dir option
	}

	switch key {
	case ARROW_UP:
		if e.cy > minCy {
			e.cy--
		}
	case ARROW_DOWN:
		if e.cy < maxItems {
			e.cy++
		}
	}
}

// highlightSelectedFile highlights the currently selected file in the explorer
func (ex *ExplorerScreen) highlightSelectedFile(e *Editor) {
	if e.cy <= 0 || e.cy >= len(ex.content) {
		return
	}

	// Reset all highlights first
	for i := 1; i < len(ex.content); i++ {
		for j := range ex.content[i].hl {
			ex.content[i].hl[j] = HL_NORMAL
		}
	}

	// Highlight current selection
	for j := range ex.content[e.cy].hl {
		ex.content[e.cy].hl[j] = HL_MATCH
	}

	// Update the editor's content reference
	e.row = ex.content
}

// openSelectedFile attempts to open the currently selected file or navigate to directory
func (ex *ExplorerScreen) openSelectedFile(e *Editor) bool {
	selectedIndex := e.cy - 1 // -1 to account for header

	// Handle parent directory navigation
	if ex.hasParentDir && selectedIndex == 0 {
		// Navigate to parent directory
		parentDir := ".."
		if ex.currentDir != "." {
			// Get actual parent path
			if lastSlash := strings.LastIndex(ex.currentDir, "/"); lastSlash != -1 {
				parentDir = ex.currentDir[:lastSlash]
				if parentDir == "" {
					parentDir = "."
				}
			} else {
				parentDir = "."
			}
		}
		ex.currentDir = parentDir
		err := ex.refreshContent()
		if err != nil {
			e.ShowError("Failed to read directory: %v", err)
			return false
		}
		return false // Directory changed, don't close explorer
	}

	// Adjust index if parent dir option is present
	if ex.hasParentDir {
		selectedIndex--
	}

	if selectedIndex < 0 || selectedIndex >= len(ex.files) {
		return false
	}

	selectedFile := ex.files[selectedIndex]

	if selectedFile.IsDir() {
		// Navigate into directory
		newDir := selectedFile.Name()
		if ex.currentDir != "." {
			newDir = ex.currentDir + "/" + newDir
		}
		ex.currentDir = newDir
		err := ex.refreshContent()
		if err != nil {
			e.ShowError("Failed to read directory: %v", err)
			return false
		}
		return false // Directory changed, don't close explorer
	}

	if e.dirty > 0 {
		e.SetStatusMessage("File has unsaved changes")
		return false
	}

	// Open regular file
	filePath := selectedFile.Name()
	if ex.currentDir != "." {
		filePath = ex.currentDir + "/" + filePath
	}

	err := e.Open(filePath)
	if err != nil {
		e.ShowError("Failed to open file: %v", err)
		return false
	}

	return true // File opened successfully
}

// Explorer opens the file explorer interface using the modal system
func (e *Editor) Explorer() {
	explorerScreen := NewExplorerScreen(e, ".")
	if explorerScreen == nil {
		return // Error already shown
	}
	modalManager := NewModalManager(e, explorerScreen)
	modalManager.Show(EXPLORER_MODE)
}
