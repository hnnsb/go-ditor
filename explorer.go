package main

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

// createFileDisplayRow creates a formatted display row for a file or directory
func createFileDisplayRow(index int, file os.DirEntry) editorRow {
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
		idx:   index + 2, // +2 to account for header and parent dir option
		chars: []byte(fileInfo),
	}
}

// createExplorerRows creates all the display rows for the file explorer
func (e *Editor) createExplorerRows(files []os.DirEntry, currentDir string) []editorRow {
	explorerRows := make([]editorRow, 0, len(files)+2)

	// Add header
	headerText := fmt.Sprintf("=== File Explorer: %s ===", currentDir)
	headerRow := editorRow{
		idx:   0,
		chars: []byte(headerText),
	}
	headerRow.Update(e)
	explorerRows = append(explorerRows, headerRow)

	// Add parent directory option (unless we're at root)
	if currentDir != "." && currentDir != "/" {
		parentText := "📂 .. (parent directory)"
		parentRow := editorRow{
			idx:   1,
			chars: []byte(parentText),
		}
		parentRow.Update(e)
		explorerRows = append(explorerRows, parentRow)
	}

	// Add files
	for i, file := range files {
		fileRow := createFileDisplayRow(i, file)
		fileRow.Update(e)
		explorerRows = append(explorerRows, fileRow)
	}

	return explorerRows
}

// setupExplorerDisplay configures the editor for file explorer display
func (e *Editor) setupExplorerDisplay(explorerRows []editorRow, fileCount int, currentDir string, hasParentDir bool) {
	e.mode = EXPLORER_MODE
	e.row = explorerRows
	e.totalRows = len(explorerRows)
	e.cx = 0
	// Start at first file (skip header and optionally parent dir)
	if hasParentDir {
		e.cy = 2 // Skip header and parent dir option
	} else {
		e.cy = 1 // Skip only header
	}
	e.colOffset = 0
	e.rowOffset = 0
	e.SetStatusMessage("File Explorer: %s - %d items (Enter=open/navigate, ESC/q=quit)", currentDir, fileCount)
}

// highlightSelectedFile highlights the currently selected file in the explorer
func (e *Editor) highlightSelectedFile() {
	if e.cy <= 0 || e.cy >= len(e.row) {
		return
	}

	// Reset all highlights first
	for i := 1; i < len(e.row); i++ {
		for j := range e.row[i].hl {
			e.row[i].hl[j] = HL_NORMAL
		}
	}

	// Highlight current selection
	for j := range e.row[e.cy].hl {
		e.row[e.cy].hl[j] = HL_MATCH
	}
}

// handleExplorerNavigation handles arrow key navigation in the explorer
func (e *Editor) handleExplorerNavigation(key int, maxItems int, hasParentDir bool) {
	minCy := 1 // Start after header
	if hasParentDir {
		minCy = 1 // Can navigate to parent dir option
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

// openSelectedFile attempts to open the currently selected file or navigate to directory
func (e *Editor) openSelectedFile(files []os.DirEntry, savedState EditorState, currentDir string, hasParentDir bool) (bool, string) {
	selectedIndex := e.cy - 1 // -1 to account for header

	// Handle parent directory navigation
	if hasParentDir && selectedIndex == 0 {
		// Navigate to parent directory
		parentDir := ".."
		if currentDir != "." {
			// Get actual parent path
			if lastSlash := strings.LastIndex(currentDir, "/"); lastSlash != -1 {
				parentDir = currentDir[:lastSlash]
				if parentDir == "" {
					parentDir = "."
				}
			} else {
				parentDir = "."
			}
		}
		return false, parentDir
	}

	// Adjust index if parent dir option is present
	if hasParentDir {
		selectedIndex--
	}

	if selectedIndex < 0 || selectedIndex >= len(files) {
		return false, currentDir
	}

	selectedFile := files[selectedIndex]

	if selectedFile.IsDir() {
		// Navigate into directory
		newDir := selectedFile.Name()
		if currentDir != "." {
			newDir = currentDir + "/" + newDir
		}
		return false, newDir
	}

	if e.dirty > 0 {
		e.SetStatusMessage("File has unsaved changes")
		return false, currentDir
	}

	// Open regular file
	e.setEditorState(savedState)

	filePath := selectedFile.Name()
	if currentDir != "." {
		filePath = currentDir + "/" + filePath
	}

	err := e.Open(filePath)
	if err != nil {
		e.ShowError("Failed to open file: %v", err)
	}

	return true, currentDir
}

// runs the main interaction loop for the file explorer
func (e *Editor) runExplorerLoop(savedState EditorState, startDir string) {
	currentDir := startDir

	for {
		// Read current directory contents
		files, err := os.ReadDir(currentDir)
		if err != nil {
			e.ShowError("Failed to read directory: %v", err)
			e.setEditorState(savedState)
			e.SetStatusMessage("File explorer closed")
			return
		}

		// Check if we have a parent directory option
		hasParentDir := currentDir != "." && currentDir != "/"

		// Create and setup explorer display
		explorerRows := e.createExplorerRows(files, currentDir)
		e.setupExplorerDisplay(explorerRows, len(files), currentDir, hasParentDir)

		// Navigation loop for current directory
		directoryChanged := false
		for !directoryChanged {
			e.highlightSelectedFile()
			e.RefreshScreen()

			key, err := readKey()
			if err != nil {
				e.ShowError("%v", err)
				continue
			}

			switch key {
			case ARROW_UP, ARROW_DOWN:
				totalItems := len(files)
				if hasParentDir {
					totalItems++ // Add parent dir option
				}
				e.handleExplorerNavigation(key, totalItems, hasParentDir)

			case '\r': // Enter key
				opened, newDir := e.openSelectedFile(files, savedState, currentDir, hasParentDir)
				if opened {
					return // File was opened successfully
				}
				if newDir != currentDir {
					currentDir = newDir
					directoryChanged = true // Break inner loop to refresh directory listing
				}

			case 'q', 'Q', '\x1b': // ESC or 'q' to quit
				e.setEditorState(savedState)
				e.SetStatusMessage("File explorer closed")
				return
			}
		}
	}
}

// Explorer opens the file explorer interface
func (e *Editor) Explorer() {
	savedState := e.getEditorState()
	startDir := "."
	e.runExplorerLoop(savedState, startDir)
}
