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
	COLORS_RESET  = "\x1b[m"
	COLORS_INVERT = "\x1b[7m"
)

// ANSI Graphics Mode Constants
const (
	ANSI_RESET_ALL     = 0
	ANSI_BOLD          = 1
	ANSI_DIM           = 2
	ANSI_ITALIC        = 3
	ANSI_UNDERLINE     = 4
	ANSI_BLINK         = 5
	ANSI_REVERSE       = 7
	ANSI_STRIKETHROUGH = 9

	// Reset codes for specific styles
	ANSI_RESET_BOLD          = 22
	ANSI_RESET_DIM           = 22
	ANSI_RESET_ITALIC        = 23
	ANSI_RESET_UNDERLINE     = 24
	ANSI_RESET_BLINK         = 25
	ANSI_RESET_REVERSE       = 27
	ANSI_RESET_STRIKETHROUGH = 29

	// Color codes
	ANSI_COLOR_RED     = 31
	ANSI_COLOR_GREEN   = 32
	ANSI_COLOR_YELLOW  = 33
	ANSI_COLOR_BLUE    = 34
	ANSI_COLOR_MAGENTA = 35
	ANSI_COLOR_CYAN    = 36
	ANSI_COLOR_WHITE   = 37
	ANSI_COLOR_DEFAULT = 39
)

// Style reset lookup table
var styleResetCodes = map[int]int{
	ANSI_BOLD:          ANSI_RESET_BOLD,
	ANSI_DIM:           ANSI_RESET_DIM,
	ANSI_ITALIC:        ANSI_RESET_ITALIC,
	ANSI_UNDERLINE:     ANSI_RESET_UNDERLINE,
	ANSI_BLINK:         ANSI_RESET_BLINK,
	ANSI_REVERSE:       ANSI_RESET_REVERSE,
	ANSI_STRIKETHROUGH: ANSI_RESET_STRIKETHROUGH,
	0:                  0, // Normal style has no reset needed
}
