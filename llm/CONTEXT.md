# GO-DITOR Context File for LLM Agents

## Project Overview

GO-DITOR is a terminal-based text editor written in Go, inspired by the kilo editor tutorial but significantly enhanced with modern Go practices and advanced features.

**Current Version**: 1.0.0  
**Language**: Go  
**Architecture**: Single-file terminal application with modular structure

## Key Features Implemented

### Core Editor Functionality

- **Terminal Raw Mode**: Full terminal control using `golang.org/x/term`
- **File Operations**: Open, edit, save files with proper error handling
- **Navigation**: Arrow keys, Home/End, Page Up/Down
- **Editing**: Insert, delete characters and lines
- **Search**: Find functionality with highlighting

### Advanced Syntax Highlighting System

- **Multi-language Support**: C/C++ and Go syntax highlighting
- **Extensible Architecture**: Easy to add new languages
- **Highlighting Types**: Keywords, strings, numbers, comments, control sequences
- **ANSI Graphics Integration**: Comprehensive color and style system

### Control Sequence Highlighting (Recent Implementation)

- **Purpose**: Highlights control sequences that appear in files but have no valid terminal actions
- **Detection**: Recognizes `^X` patterns (e.g., `^[`, `^A`, `^B`)
- **Extended Sequences**: Special handling for escape sequences like `^[B`, `^[2J`
- **Visual Feedback**: Red reverse video highlighting for easy identification

## File Structure

```
go-ditor/
├── main.go          # Main application file (~1300+ lines)
├── ansi.go         # ANSI escape sequences and constants
├── go.mod          # Go module definition
├── go.sum          # Module checksums
├── README.md       # Project documentation
├── TODO.md         # Development roadmap
└── CONTEXT.md      # This context file
```

## Architecture Deep Dive

### Data Structures

#### Core Types

```go
type editorConfig struct {
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
    originalState     *term.State   // Terminal state backup
}

type editorRow struct {
    idx           int      // Row index
    chars         []byte   // Original characters
    render        []byte   // Rendered characters (with tabs/control expanded)
    hl            []int    // Highlighting information
    hlOpenComment bool     // Multi-line comment state
}

type editorSyntax struct {
    filetype               string   // Language identifier
    filematch              []string // File extension patterns
    keywords               []string // Language keywords
    singlelineCommentStart string   // Single-line comment syntax
    multilineCommentStart  string   // Multi-line comment start
    multilineCommentEnd    string   // Multi-line comment end
    flags                  int      // Feature flags
}
```

### Key Subsystems

#### 1. Terminal Management (`/*** terminal ***/`)

- Raw mode enable/disable with proper cleanup
- Key reading with escape sequence parsing
- Window size detection
- Cross-platform compatibility considerations

#### 2. Syntax Highlighting (`/*** syntax highlighting ***/`)

- **editorUpdateSyntax()**: Core highlighting engine
- **Control Sequence Detection**: Latest feature for highlighting terminal sequences
- **Multi-state Parser**: Handles strings, comments, keywords simultaneously
- **Extensible Design**: Easy language addition through HLDB_ENTRIES

#### 3. Row Operations (`/*** row operations ***/`)

- **Cursor Mapping**: Convert between cursor position and render position
- **Text Rendering**: Handle tabs and control characters
- **Control Character Expansion**: Convert control chars to `^X` format

#### 4. File I/O (`/*** file i/o ***/`)

- **Efficient Loading**: Line-by-line file reading
- **Safe Saving**: Atomic write operations with error handling
- **Encoding Handling**: Proper text encoding management

## Recent Development History

### Major Refactoring (Recent)

1. **Architecture Modernization**:

   - Migrated from pointer-based to value-based editorRow
   - Improved memory management and slice operations
   - Enhanced method organization with receiver methods

2. **Control Sequence Highlighting Implementation**:

   - Added `HL_CONTROL` highlighting type
   - Implemented detection for `^X` patterns in `editorUpdateSyntax()`
   - Special handling for escape sequences (`^[B`, `^[2J`, etc.)
   - Integrated with ANSI graphics system for visual feedback

3. **ANSI Graphics System Enhancement**:
   - Comprehensive constants in `ansi.go`
   - Lookup table for style reset codes
   - Dynamic color and style application
   - Eliminated hardcoded escape sequences

### Implementation Details: Control Sequence Highlighting

The control sequence highlighting was the most recent significant feature addition:

**Problem**: Files containing terminal control sequences (like `^[B` for arrow down) were not visually distinguished, making them hard to identify.

**Solution**:

- Modified `editorUpdateRow()` to convert control characters to visible `^X` format
- Enhanced `editorUpdateSyntax()` to detect these patterns
- Added special logic for extended escape sequences
- Integrated with existing syntax highlighting system

**Key Code Locations**:

- Control char conversion: `(row *editorRow) update()` method
- Syntax detection: `editorUpdateSyntax()` function
- Visual styling: `editorSyntaxToGraphics()` function

## Constants and Configuration

### Display Constants

```go
const (
    GO_DITOR_VERSION       = "1.0.0"
    TAB_STOP               = 4
    CONTROL_SEQUENCE_WIDTH = 2  // Width of ^X sequences
    QUIT_TIMES             = 3  // Confirmation prompts
)
```

### Syntax Highlighting Types

```go
const (
    HL_NORMAL = iota    // Default text
    HL_COMMENT          // Single-line comments
    HL_MLCOMMENT        // Multi-line comments
    HL_KEYWORD1         // Primary keywords
    HL_KEYWORD2         // Type keywords (ending with |)
    HL_STRING           // String literals
    HL_NUMBER           // Numeric literals
    HL_MATCH            // Search matches
    HL_CONTROL          // Control sequences (NEW)
)
```

## Dependencies

### External Packages

- `golang.org/x/term`: Terminal control and raw mode
- Standard library: `bufio`, `bytes`, `fmt`, `os`, `strings`, `time`

### Internal Architecture

- Single-file design for simplicity
- Modular section organization with clear separation
- Method-based operations on data structures

## Build and Run Instructions

```bash
# Build the editor
go build -o go-ditor.exe

# Run with a file
./go-ditor.exe filename.txt

# Run without arguments (creates new file)
./go-ditor.exe
```

## Key Bindings

- `Ctrl+Q`: Quit (with unsaved changes confirmation)
- `Ctrl+S`: Save file
- `Ctrl+F`: Find/search
- `Ctrl+L`: Refresh screen
- Arrow keys: Navigation
- Page Up/Down: Scroll by screen
- Home/End: Line beginning/end
- Backspace/Delete: Character deletion

## Development Guidelines for Future Agents

### Code Style

1. **Section Organization**: Code is organized in clearly marked sections with `/***/` comments
2. **Method Receivers**: Use value receivers for editorRow methods
3. **Error Handling**: Proper error checking with informative messages
4. **Memory Management**: Efficient slice operations, avoid unnecessary allocations

### Adding New Features

1. **Syntax Highlighting**: Add new language support through HLDB_ENTRIES
2. **Key Bindings**: Extend editorProcessKeypress() for new commands
3. **Display Elements**: Modify drawing functions for UI changes
4. **File Operations**: Enhance I/O functions for new file formats

### Testing Considerations

- Terminal compatibility across platforms
- File encoding handling
- Memory usage with large files
- Syntax highlighting performance

## Known Areas for Enhancement

1. **Multiple File Support**: Currently single-file editor
2. **Configuration System**: Hardcoded settings could be configurable
3. **Plugin Architecture**: Could support external syntax definitions
4. **Performance Optimization**: Large file handling improvements
5. **Undo/Redo System**: Currently not implemented

## Debugging and Troubleshooting

### Common Issues

1. **Terminal State**: Always ensure proper terminal restoration
2. **Cursor Positioning**: Watch for render vs. cursor position mismatches
3. **Syntax Highlighting**: Check for infinite loops in highlighting logic
4. **File Operations**: Handle edge cases in file I/O

### Debug Approaches

- Use status messages for runtime debugging
- Check terminal dimensions on start
- Validate slice bounds in row operations
- Monitor memory usage with large files

This context should provide comprehensive understanding for any LLM agent working on this codebase.
