package main

import (
	"testing"
)

func TestEditorRowDeleteChar(t *testing.T) {
	// Create a test row
	row := &editorRow{
		idx:           0,
		size:          5,
		chars:         []byte("hello"),
		rsize:         0,
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Initialize the render and hl slices
	row.update()

	// Test deleting a character
	row.deleteChar(1) // Delete 'e' from "hello"

	// Check if the character was deleted correctly
	expected := "hllo"
	actual := string(row.chars[:row.size])

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}

	if row.size != 4 {
		t.Errorf("Expected size 4, got %d", row.size)
	}

	if len(row.chars) != 4 {
		t.Errorf("Expected chars slice length 4, got %d", len(row.chars))
	}
}

func TestEditorRowDeleteCharMultiple(t *testing.T) {
	// Create a test row
	row := &editorRow{
		idx:           0,
		size:          3,
		chars:         []byte("abc"),
		rsize:         0,
		render:        nil,
		hl:            nil,
		hlOpenComment: false,
	}

	// Initialize the render and hl slices
	row.update()

	// Test deleting multiple characters
	row.deleteChar(0) // Delete 'a' from "abc" -> "bc"
	row.deleteChar(0) // Delete 'b' from "bc" -> "c"

	// Check if the characters were deleted correctly
	expected := "c"
	actual := string(row.chars[:row.size])

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}

	if row.size != 1 {
		t.Errorf("Expected size 1, got %d", row.size)
	}

	if len(row.chars) != 1 {
		t.Errorf("Expected chars slice length 1, got %d", len(row.chars))
	}
}
