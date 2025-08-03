package main

// ANSI escape sequences for terminal control

const (
	// Screen control
	CLEAR_SCREEN = "\x1b[2J" // Clear entire screen
	CLEAR_LINE   = "\x1b[K"  // Clear line from cursor to end
	CURSOR_HOME  = "\x1b[H"  // Move cursor to top-left (1,1)

	// Cursor visibility
	CURSOR_HIDE = "\x1b[?25l" // Hide cursor
	CURSOR_SHOW = "\x1b[?25h" // Show cursor

	// Cursor positioning
	CURSOR_BOTTOM_RIGHT = "\x1b[999;999H" // Move cursor to bottom-right corner
	CURSOR_GET_POSITION = "\x1b[6n"       // Request cursor position

	// Format strings for dynamic positioning
	CURSOR_POSITION_FORMAT = "\x1b[%d;%dH" // Format for moving cursor to specific row;col
	CURSOR_RESPONSE_FORMAT = "\x1b[%d;%dR" // Format for parsing cursor position response

	// Text formatting
	COLORS_RESET  = "\x1b[m" // Reset all text formatting
	COLORS_INVERT = "\x1b[7m"
)
