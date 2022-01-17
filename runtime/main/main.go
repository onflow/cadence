package main

import (
	"fmt"
	"log"
	"os"
)

const (
	expectedArgsCount = 1

	RunScriptFuncName = "runScript"
)

func main() {

	programArgsCount := len(os.Args) - 1
	if programArgsCount < expectedArgsCount {
		log.Fatalf("Not enough arguments: expected %d, found %d", expectedArgsCount, programArgsCount)
	}

	if programArgsCount > expectedArgsCount {
		log.Fatalf("Too many arguments: expected %d, found %d", expectedArgsCount, programArgsCount)
	}

	funcName := os.Args[1]

	switch funcName {
	case RunScriptFuncName:
		RunScript()
	default:
		fmt.Println(fmt.Errorf("unsopported operation '%s'", funcName))
		panic("boom!")
	}
}

func RunScript() {
	//fmt.Println("Hello, world!")
	RunScript()
}
