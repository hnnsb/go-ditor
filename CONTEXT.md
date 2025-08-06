# KIGO - Terminal Text Editor Context for LLM Agents

## Project Overview

**KIGO** is a terminal-based text editor written in Go, inspired by the Kilo editor tutorial. It's a learning project designed to understand Go programming while implementing a functional text editor with features like syntax highlighting, file exploration, and modal interfaces.

## Technical Stack

- **Language**: Go 1.24.2
- **Dependencies**:
  - `golang.org/x/term` v0.33.0 (terminal control)
  - `golang.org/x/sys` v0.34.0 (system calls)
- **Repository**: `github.com/hnnsb/kigo`

## Project Structure

```
kigo/
├── main.go                 # Entry point and main loop
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── kigo.exe               # Compiled binary (Windows)
├── README.md              # Project documentation
├── TODO.md                # Development roadmap and refactoring tasks
└── editor/                # Core editor package
    ├── editor.go          # Main editor logic and data structures
    ├── editor_test.go     # Unit tests
    ├── ansi.go           # ANSI escape sequences and terminal control
    ├── explorer.go       # File explorer modal functionality
    ├── help.go           # Help screen modal
    └── modal.go          # Modal interface and management system
```

## Core Architecture

### Main Components

1. **Editor Struct** (`editor/editor.go`):

   - Central state management for the editor
   - Handles file operations, cursor movement, and screen rendering
   - Manages editor modes (EDIT, EXPLORER, SEARCH, SAVE, HELP)

2. **Modal System** (`editor/modal.go`):

   - Interface-based modal screens (help, explorer, search)
   - State preservation when switching between modals
   - Consistent key handling across different modal types

3. **Terminal Management** (`editor/ansi.go`):

   - ANSI escape sequence constants
   - Raw mode terminal control
   - Cross-platform terminal handling (Windows/Unix)

4. **File Explorer** (`editor/explorer.go`):
   - Directory navigation modal
   - File selection and opening
   - State management for editor context switching

### Key Data Structures

```go
// Main editor state
type Editor struct {
    cx, cy           int              // Cursor position
    rx               int              // Render x position
    rowOffset        int              // Vertical scroll
    colOffset        int              // Horizontal scroll
    screenRows       int              // Terminal dimensions
    screenCols       int
    totalRows        int              // Number of text rows
    row             []editorRow       // Text content rows
    dirty           int               // Unsaved changes counter
    filename        string            // Current file
    statusmsg       string            // Status bar message
    statusmsgTime   time.Time         // Status message timestamp
    syntax          *editorSyntax     // Syntax highlighting rules
    mode            int               // Current editor mode
    terminal        Terminal          // Terminal state
}

// Individual text row
type editorRow struct {
    idx           int       // Row index
    chars         []byte    // Raw characters
    render        []byte    // Rendered characters (tabs expanded)
    hl            []byte    // Syntax highlighting data
    hlOpenComment bool      // Multi-line comment state
}

// Modal interface for screens
type ModalScreen interface {
    GetContent() []editorRow
    GetTitle() string
    GetStatusMessage() string
    HandleKey(key int, e *Editor) (bool, bool)
    Initialize(e *Editor)
}
```

## Key Features

### Editor Functionality

- **File Operations**: Open, save, new file creation
- **Navigation**: Arrow keys, Page Up/Down, Home/End
- **Editing**: Insert, delete, backspace with undo/redo tracking
- **Search**: Text search with highlighting and navigation
- **Syntax Highlighting**: Configurable syntax rules for different file types

### Modal Screens

- **File Explorer** (Ctrl+E): Navigate and open files
- **Help Screen** (Ctrl+H): Display keyboard shortcuts and commands
- **Search Mode** (Ctrl+F): Find and replace functionality
- **Save Mode**: File saving with name prompts

### Terminal Features

- **Raw Mode**: Direct keyboard input handling
- **Cross-platform**: Windows and Unix terminal support
- **ANSI Control**: Screen clearing, cursor positioning, text formatting
- **Responsive**: Dynamic screen size handling

## Key Constants and Enums

```go
// Editor modes
const (
    EDIT_MODE = iota
    EXPLORER_MODE
    SEARCH_MODE
    SAVE_MODE
    HELP_MODE
)

// Special keys
const (
    BACKSPACE = 127
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
```

## Development Status

### Completed Refactoring (from TODO.md)

- ✅ Replaced global state with Editor struct
- ✅ Improved error handling patterns
- ✅ Organized constants and naming conventions
- ✅ Dependency injection for editor state

### Pending Improvements

- 🔄 Interface definitions for terminal/file operations
- 🔄 Memory management optimization
- 🔄 Multi-package organization
- 🔄 Unicode/UTF-8 support (umlauts: ä, ö, ü)
- 🔄 Configuration file support
- 🔄 Enhanced file explorer UI

## Testing

- **Unit Tests**: `editor_test.go` with basic functionality tests
- **Test Coverage**: Limited to core editor row operations
- **Testing Strategy**: Focused on data structure manipulation

## Usage Patterns

### Key Bindings

- **Ctrl+S**: Save file
- **Ctrl+Q**: Quit (with confirmation)
- **Ctrl+F**: Find/Search
- **Ctrl+E**: File Explorer
- **Ctrl+H**: Help screen
- **Ctrl+R**: Redraw screen

### Command Line

```bash
./kigo [filename]    # Open specific file
./kigo              # Start with empty file
```

## Architecture Decisions

1. **Single Package for Core Logic**: Editor functionality concentrated in `editor/` package
2. **Modal Interface Pattern**: Consistent screen management across different editor modes
3. **State Preservation**: Save/restore editor state when switching between modals
4. **Raw Terminal Control**: Direct ANSI escape sequence handling for performance
5. **Go Idiomatic Patterns**: Struct methods, interface-based design, proper error handling

## File Handling

- **Auto-detection**: File type detection for syntax highlighting
- **Cross-platform**: Handles different line endings (Windows: \r\n, Unix: \n)
- **Dirty Tracking**: Monitors unsaved changes with confirmation prompts
- **Explorer Integration**: Modal file browser for easy file navigation

This context provides LLM agents with comprehensive understanding of KIGO's architecture, current implementation status, and development patterns for effective code assistance and modification suggestions.
