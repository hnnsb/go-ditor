# GO-DITOR Context File for LLM Agents

## Project Overview

GO-DITOR is a terminal-based text editor written in Go, inspired by the kilo editor tutorial but significantly enhanced with modern Go practices and advanced features. The project has undergone major refactoring to follow Go idioms and best practices.

**Current Version**: 1.0.0  
**Language**: Go  
**Architecture**: Single-file terminal application with object-oriented design

## Recent Major Refactoring (2024)

### Completed Modernization Efforts

1. **✅ Error Handling & Recovery**

   - Replaced `die()` function with proper Go error handling patterns
   - File operations now return errors instead of terminating
   - Graceful error recovery for non-fatal operations
   - User-friendly error messages in status bar

2. **✅ Global State Elimination**

   - Converted all functions to methods on `Editor` struct
   - Eliminated global variable access patterns
   - Implemented dependency injection with constructor functions
   - Separated terminal and editor state management

3. **✅ Go Idiomatic Patterns**
   - Method receivers on `*Editor` for all operations
   - Proper error handling with early returns
   - Constructor pattern with `NewEditor()` and `NewTerminal()`
   - Clean separation of concerns

## Key Features Implemented

### Core Editor Functionality

- **Terminal Raw Mode**: Full terminal control using `golang.org/x/term`
- **File Operations**: Open, edit, save files with proper error handling
- **Navigation**: Arrow keys, Home/End, Page Up/Down
- **Editing**: Insert, delete characters and lines
- **Search**: Find functionality with highlighting

### Advanced Syntax Highlighting System

- **Multi-language Support**: C/C++ and Go syntax highlighting
- **Extensible Architecture**: Easy to add new languages via `HLDB_ENTRIES`
- **Highlighting Types**: Keywords, strings, numbers, comments, control sequences
- **ANSI Graphics Integration**: Comprehensive color and style system

### Control Sequence Highlighting

- **Purpose**: Highlights control sequences in files (e.g., `^[`, `^A`, `^B`)
- **Detection**: Recognizes patterns with special handling for escape sequences
- **Visual Feedback**: Red reverse video highlighting for easy identification

## File Structure

```
go-ditor/
├── go_ditor.go     # Main application file (~1300+ lines, fully refactored)
├── ansi.go         # ANSI escape sequences and constants
├── go.mod          # Go module definition
├── go.sum          # Module checksums
├── README.md       # Project documentation
├── TODO.md         # Development roadmap (mostly completed)
└── llm/
    └── CONTEXT.md  # This context file
```

## Architecture Deep Dive

### Core Data Structures (Refactored)

```go
// Terminal handles terminal-specific operations
type Terminal struct {
    originalState *term.State
}

// Editor represents the text editor state
type Editor struct {
    cx, cy            int           // Cursor position
    rx                int           // Render X position
    rowOffset         int           // Vertical scroll offset
    colOffset         int           // Horizontal scroll offset
    screenRows        int           // Terminal dimensions
    screenCols        int
    totalRows         int           // Number of rows in file
    row               []editorRow   // File content
    dirty             int           // Modification counter
    filename          string        // Current filename
    statusMessage     string        // Status bar message
    statusMessageTime time.Time     // Message timestamp
    syntax            *editorSyntax // Current syntax highlighting
    terminal          *Terminal     // Terminal interface
}

type editorRow struct {
    idx           int      // Row index
    chars         []byte   // Original characters
    render        []byte   // Rendered characters (with tabs/control expanded)
    hl            []int    // Highlighting information
    hlOpenComment bool     // Multi-line comment state
}
```

### Method-Based Architecture

All operations are now methods on the appropriate structs:

#### Editor Methods

```go
// Core operations
func (e *Editor) InsertChar(c int)
func (e *Editor) DeleteChar()
func (e *Editor) InsertNewline()

// File operations
func (e *Editor) Open(filename string) error
func (e *Editor) Save()

// Navigation and display
func (e *Editor) MoveCursor(key int)
func (e *Editor) RefreshScreen()
func (e *Editor) ProcessKeypress()

// Terminal control
func (e *Editor) EnableRawMode() error
func (e *Editor) RestoreTerminal()

// Error handling
func (e *Editor) Die(format string, args ...any)
func (e *Editor) ShowError(format string, args ...any)
```

#### Row Methods (with Editor Context)

```go
func (row *editorRow) InsertChar(e *Editor, at int, c int)
func (row *editorRow) Update(e *Editor)
func (row *editorRow) UpdateSyntax(e *Editor)
```

### Key Subsystems

#### 1. Terminal Management

- Raw mode enable/disable with proper cleanup via `*Terminal` struct
- Key reading with escape sequence parsing
- Window size detection
- Proper state restoration on exit

#### 2. Error Handling Strategy

- **Fatal Errors**: Terminal initialization failures (terminate with `Die()`)
- **Recoverable Errors**: File operations (show error in status bar with `ShowError()`)
- **Transient Errors**: Input errors (brief status message, continue operation)

#### 3. Syntax Highlighting Engine

- Real-time highlighting with `UpdateSyntax()` method
- Multi-language support through `HLDB_ENTRIES`
- Control sequence detection and highlighting
- Efficient character-by-character rendering

#### 4. Object-Oriented Design

- No global state access in any methods
- Clear ownership and responsibility separation
- Dependency injection through constructor pattern
- Testable architecture with method receivers

## Constructor Pattern

### Initialization

```go
func NewTerminal() *Terminal
func NewEditor() Editor
func (e *Editor) Init() error

// Usage in main:
editor := NewEditor()
err := editor.EnableRawMode()
// ... error handling
defer editor.RestoreTerminal()
```

## Constants and Configuration

### Display Constants

```go
const (
    GO_DITOR_VERSION       = "1.0.0"
    TAB_STOP               = 4
    CONTROL_SEQUENCE_WIDTH = 2
    QUIT_TIMES             = 3
)
```

### Well-Organized Enums with iota

```go
// Key constants with proper iota usage
const (
    BACKSPACE = 127
    ARROW_LEFT = iota + 1000
    ARROW_RIGHT
    ARROW_UP
    ARROW_DOWN
    // ...
)

// Syntax highlighting types
const (
    HL_NORMAL = iota
    HL_COMMENT
    HL_MLCOMMENT
    // ...
    HL_CONTROL
)
```

## Dependencies

### External Packages

- `golang.org/x/term`: Terminal control and raw mode
- Standard library: `bufio`, `bytes`, `errors`, `fmt`, `os`, `slices`, `strings`, `time`

### Internal Architecture

- Single-file design with clear method organization
- Object-oriented patterns with proper encapsulation
- No global state dependencies

## Development Status

### Completed Refactoring (TODO Items)

- ✅ **Error Handling & Recovery**: Proper Go error patterns implemented
- ✅ **Global State Management**: Eliminated global variables, method-based design
- ✅ **Constants and Naming**: Well-organized with iota, consistent naming
- ⏳ **Function Organization**: Partially complete - methods implemented, interfaces pending
- ⏳ **Package Structure**: Considering modular split for larger codebase

### Current Architecture Quality

- **Testable**: Methods can be called on editor instances
- **Go Idiomatic**: Proper error handling, method receivers, constructor pattern
- **Maintainable**: Clear separation of concerns, no global state
- **Extensible**: Easy to add new features through method addition

## Key Bindings (Unchanged)

- `Ctrl+Q`: Quit (with unsaved changes confirmation)
- `Ctrl+S`: Save file
- `Ctrl+F`: Find/search
- Arrow keys: Navigation
- Page Up/Down: Scroll by screen
- Home/End: Line beginning/end
- Backspace/Delete: Character deletion

## Development Guidelines for Future Agents

### Code Architecture

1. **Method-First Design**: All operations should be methods on appropriate structs
2. **Error Handling**: Return errors from methods, use `ShowError()` for recoverable issues
3. **No Global State**: Pass editor context explicitly where needed
4. **Constructor Pattern**: Use `NewEditor()` for initialization

### Adding New Features

1. **New Methods**: Add methods to `Editor` struct for new functionality
2. **Row Operations**: Use `(row *editorRow) method(e *Editor, ...)` pattern
3. **Error Strategy**: Determine if errors should be fatal or recoverable
4. **Syntax Highlighting**: Extend through `HLDB_ENTRIES` for new languages

### Code Quality Standards

- Use method receivers appropriately (`*Editor` for mutations)
- Implement proper error handling with context
- Maintain clear separation between terminal and editor concerns
- Follow existing naming conventions consistently

## Future Enhancement Opportunities

1. **Interface Definition**: Create interfaces for terminal, file operations
2. **Package Splitting**: Consider breaking into multiple packages when appropriate
3. **Multiple File Support**: Extend to handle multiple open files
4. **Configuration System**: Add configurable settings
5. **Undo/Redo System**: Implement operation history

## Testing and Debugging

### Current Testability

- Editor instances can be created independently
- Methods can be tested in isolation
- No global state interference
- Clear error handling for test validation

### Debug Approaches

- Use `ShowError()` for runtime debugging messages
- Create multiple editor instances for testing scenarios
- Monitor method call patterns and error flows
- Validate state changes through method calls

This refactored architecture provides a solid foundation for future development while maintaining the simplicity and educational value of the original design.
