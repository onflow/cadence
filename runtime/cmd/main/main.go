package main

import (
	"os"

	"github.com/onflow/cadence/runtime/cmd/execute"
)

func main() {
	if len(os.Args) > 1 {
		execute.Execute(os.Args[1:])
	} else {
		execute.RunREPL()
	}
}
