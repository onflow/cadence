package main

import (
	"github.com/onflow/cadence/languageserver/server"

	"os"

	"github.com/mattn/go-isatty"
)

func main() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		print(
			"This program implements the Language Server Protocol for Cadence.\n"+
				"Please check the documentation on how to run it.\n" +
				"It does nothing in a terminal, it should be run with an editor/IDE.\n",
		)
		os.Exit(1)
	}

	server.NewServer().Start()
}
