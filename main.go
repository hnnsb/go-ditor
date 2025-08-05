package main

import (
	"os"

	"github.com/hnnsb/go-ditor/editor"
)

func main() {
	editor := editor.NewEditor()

	args := os.Args[1:]
	err := editor.EnableRawMode()
	if err != nil {
		editor.Die("enabling raw mode: %s", err.Error())
	}
	defer editor.RestoreTerminal()

	err = editor.Init()
	if err != nil {
		editor.Die("initializing editor: %s", err.Error())
	}

	editor.SetStatusMessage("HELP: Ctrl-S = save | Ctrl-Q = quit | Ctrl-F = find")

	if len(args) >= 1 {
		err = editor.Open(args[0])
		if err != nil {
			editor.ShowError("%v", err)
		}
	}

	for {
		editor.RefreshScreen()
		editor.ProcessKeypress()
	}
}
