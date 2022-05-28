/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package execute

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/pretty"
)

func RunREPL() {
	printReplWelcome()

	lineNumber := 1
	lineIsContinuation := false
	code := ""

	errorPrettyPrinter := pretty.NewErrorPrettyPrinter(os.Stderr, true)

	repl, err := runtime.NewREPL(
		func(err error, location common.Location, codes map[common.LocationID]string) {
			printErr := errorPrettyPrinter.PrettyPrintError(err, location, codes)
			if printErr != nil {
				panic(printErr)
			}
		},
		func(value interpreter.Value) {
			fmt.Println(formatValue(value))
		},
		nil,
		nil,
	)

	if err != nil {
		panic(err)
	}

	executor := func(line string) {
		defer func() {
			lineNumber++
		}()

		if code == "" && strings.HasPrefix(line, ".") {
			handleCommand(line)
			code = ""
			return
		}

		// Prefix the code with empty lines,
		// so that error messages match current line number

		for i := 1; i < lineNumber; i++ {
			code = "\n" + code
		}

		code += line + "\n"

		inputIsComplete := repl.Accept(code)
		if !inputIsComplete {
			lineIsContinuation = true
			return
		}

		lineIsContinuation = false
		code = ""
	}

	suggest := func(d prompt.Document) []prompt.Suggest {
		if len(d.GetWordBeforeCursor()) == 0 {
			return nil
		}

		suggests := []prompt.Suggest{}

		for _, suggestion := range repl.Suggestions() {
			suggests = append(suggests, prompt.Suggest{
				Text:        suggestion.Name,
				Description: suggestion.Description,
			})
		}

		return prompt.FilterHasPrefix(suggests, d.GetWordBeforeCursor(), false)
	}

	changeLivePrefix := func() (string, bool) {
		separator := '>'
		if lineIsContinuation {
			separator = '.'
		}

		return fmt.Sprintf("%d%c ", lineNumber, separator), true
	}

	options := []prompt.Option{
		prompt.OptionLivePrefix(changeLivePrefix),
	}
	prompt.New(executor, suggest, options...).Run()
}

const replHelpMessage = `
Enter declarations and statements to evaluate them.
Commands are prefixed with a dot. Valid commands are:

.exit     Exit the interpreter
.help     Print this help message

Press ^C to abort current expression, ^D to exit`

const replAssistanceMessage = `Type '.help' for assistance.`

func handleCommand(command string) {
	switch command {
	case ".exit":
		os.Exit(0)
	case ".help":
		fmt.Println(replHelpMessage)
	default:
		fmt.Println(colorizeError(fmt.Sprintf("Unknown command. %s", replAssistanceMessage)))
	}
}

func printReplWelcome() {
	fmt.Printf("Welcome to Cadence %s!\n%s\n\n", cadence.Version, replAssistanceMessage)
}
