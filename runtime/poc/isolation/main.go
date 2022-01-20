package main

import (
	"fmt"
	"log"
	"os"
)

const (
	expectedArgsCount = 1

	RunScriptFuncName     = "runScript"
	StackOverflowFuncName = "stackOverflow"
)

//go:generate go build ./main.go
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
	case StackOverflowFuncName:
		StackOverflow()
	default:
		log.Fatalf("unsupported operation '%s'", funcName)
		//panic("boom!")
	}
}

func RunScript() {
	fmt.Println("Hello, world!")
}

func StackOverflow() {
	StackOverflow()
}
