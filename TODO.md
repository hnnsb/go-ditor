# Bugs

# Improvements

- Config via config file
- beautify explorer screen
  - sort dirs first and files second
- [x] clear up []byte vs int vs string usage, switch to runes
  - [x] allows umlauts ä, ö, ü etc. (Currently are displayed but cursors is messed up) (runes?)

## Go Idiomatic Refactoring Opportunities

### 1. Error Handling & Recovery

- [x] Replace `die()` function with proper Go error handling patterns
- [x] Return errors from functions instead of calling `die()` internally
- [x] Create custom error types for different failure modes
- [x] Implement graceful error recovery without terminating the program

### 2. Global State Management

- [x] Eliminate global variable `E` (editorConfig)
- [x] Create an Editor struct and pass it around instead of global access
- [x] Use dependency injection patterns
- [x] Separate editor state from terminal/display state
- [x] Convert all functions to methods or accept Editor parameter

### 3. Constants and Naming Conventions

- [x] Group related constants using typed constants and iota
- [x] Create enum-like types for keys, colors, and styles
- [x] Use consistent naming throughout (keeping current SCREAMING_SNAKE_CASE style)
  - GO idiomatic would be PascalCase
- [x] Constants and naming conventions are well-organized

### 4. Function Organization and Interfaces

- [ ] Define interfaces for terminal operations, file operations, and rendering
- [ ] Convert package-level functions to methods on appropriate structs
- [ ] Split monolithic file into multiple packages (terminal, editor, syntax, etc.)
- [ ] Rename functions to follow Go conventions (editorReadKey → (e \*Editor) ReadKey)

### 5. Memory Management and Slices

- [ ] Simplify complex slice operations with Go idioms
- [ ] Use proper slice initialization patterns
- [ ] Replace manual buffer management with strings.Builder or bytes.Buffer
- [ ] Remove manual length tracking where Go handles it automatically

### 6. Type System Improvements

- [ ] Create proper types for keys, colors, and styles instead of using `int`
- [ ] Implement enum-like types for syntax highlighting
- [ ] Add methods to types like `editorRow` and `editorSyntax`
- [ ] Review and optimize pointer vs value receivers

### 7. Concurrency and Channels

- [ ] Use goroutines and channels for non-blocking input handling
- [ ] Make file saving operations non-blocking
- [ ] Implement proper Go signal handling
- [ ] Consider background operations for syntax highlighting

### 8. Configuration and Initialization

- [ ] Create a proper configuration system
- [ ] Use builder pattern for complex initialization
- [ ] Implement struct literal defaults
- [ ] Add configuration validation

### 9. File I/O and Resource Management

- [ ] Use more idiomatic file operations
- [ ] Ensure proper resource cleanup with defer statements
- [ ] Add context support for cancellable operations
- [ ] Consider using embed for built-in configurations

### 10. Testing and Testability

- [ ] Make code more testable through dependency injection
- [ ] Create mockable interfaces
- [ ] Implement table-driven tests
- [ ] Add test helpers for common scenarios

### 11. String and Byte Handling

- [x] Use strings.Builder for string concatenation
- [x] Ensure consistent string/byte usage
- [x] Implement proper Unicode support
- [x] Use more efficient string manipulation methods

### 12. Syntax Highlighting Architecture

- [ ] Design extensible syntax highlighting system
- [ ] Implement registry pattern for syntax definitions
- [ ] Consider streaming approach for large files
- [ ] Move towards structured syntax representation

## Priority Order

1. **High Priority**: Error handling, global state elimination, naming conventions
2. **Medium Priority**: Function organization, type system improvements, configuration
3. **Low Priority**: Concurrency, advanced architecture changes

## Implementation Strategy

- Start with simple changes (naming conventions, error handling)
- Gradually eliminate global state
- Refactor into proper Go packages
- Add interfaces and improve testability
- Implement advanced features (concurrency, plugins)
