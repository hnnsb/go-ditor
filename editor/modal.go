package editor

// ModalScreen represents a modal screen interface that can be displayed in the editor
type ModalScreen interface {
	// GetContent returns the content rows to display
	GetContent() []editorRow

	// GetTitle returns the title for the modal screen
	GetTitle() string

	// GetStatusMessage returns the status message for the modal screen
	GetStatusMessage() string

	// HandleKey processes a key press and returns true if the modal should close
	// The second return value indicates whether to restore the previous state (true) or keep current state (false)
	HandleKey(key int, e *Editor) (bool, bool)

	// Initialize sets up the initial cursor position and any other screen-specific setup
	Initialize(e *Editor)
}

// handles the common logic for modal screens
type ModalManager struct {
	savedState EditorState
	screen     ModalScreen
	editor     *Editor
}

// creates a new modal manager
func NewModalManager(editor *Editor, screen ModalScreen) *ModalManager {
	return &ModalManager{
		savedState: editor.getEditorState(),
		screen:     screen,
		editor:     editor,
	}
}

// displays the modal screen and handles the interaction loop
func (m *ModalManager) Show(mode int) {
	content := m.screen.GetContent()
	m.setupModalDisplay(content, mode)

	// Let the screen initialize itself (e.g., set cursor position)
	m.screen.Initialize(m.editor)

	// Main interaction loop
	for {
		m.editor.RefreshScreen()

		key, err := readKey()
		if err != nil {
			m.editor.ShowError("%v", err)
			continue
		}

		shouldClose, shouldRestore := m.screen.HandleKey(key, m.editor)
		if shouldClose {
			if shouldRestore {
				m.restoreState()
			}
			break // Screen requested to close
		}
	}
}

// configures the editor for modal display
func (m *ModalManager) setupModalDisplay(content []editorRow, mode int) {
	m.editor.mode = mode
	m.editor.row = content
	m.editor.totalRows = len(content)
	m.editor.cx = 0
	m.editor.cy = 0
	m.editor.colOffset = 0
	m.editor.rowOffset = 0
	m.editor.SetStatusMessage("%s", m.screen.GetStatusMessage())
}

// restores the editor to its previous state
func (m *ModalManager) restoreState() {
	m.editor.setEditorState(m.savedState)
	m.editor.SetStatusMessage("Returned to editor")
}
