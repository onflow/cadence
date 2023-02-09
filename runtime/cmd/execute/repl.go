/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	prettyJSON "github.com/tidwall/pretty"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
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
		func(err error, location common.Location, codes map[common.Location][]byte) {
			printErr := errorPrettyPrinter.PrettyPrintError(err, location, codes)
			if printErr != nil {
				panic(printErr)
			}
		},
		func(value interpreter.Value) {
			fmt.Println(formatValue(value))
		},
	)

	if err != nil {
		panic(err)
	}

	executor := func(line string) {
		if code == "" && strings.HasPrefix(line, ".") {
			handleCommand(repl, line)
			code = ""
			return
		}

		code += line + "\n"

		inputIsComplete, err := repl.Accept([]byte(code))
		if err == nil {
			lineNumber++

			if !inputIsComplete {
				lineIsContinuation = true
				return
			}
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

.exit                 Exit the interpreter
.help                 Print this help message
.export variable      Export variable

Press ^C to abort current expression, ^D to exit`

const replAssistanceMessage = `Type '.help' for assistance.`

func handleCommand(repl *runtime.REPL, command string) {
	parts := strings.SplitN(command, " ", 2)
	switch parts[0] {
	case ".exit":
		os.Exit(0)
	case ".help":
		fmt.Println(replHelpMessage)
	case ".export":
		name := strings.TrimSpace(parts[1])
		global := repl.GetGlobal(name)
		if global == nil {
			fmt.Println(colorizeError(fmt.Sprintf("Undefined global: %s", name)))
			return
		}

		value, err := repl.ExportValue(global)
		if err != nil {
			fmt.Println(colorizeError(fmt.Sprintf("Failed to export global %s: %s", name, err.Error())))
			return
		}

		json, err := jsoncdc.Encode(value)
		if err != nil {
			fmt.Println(colorizeError(fmt.Sprintf("Failed to encode global %s to JSON: %s", name, err.Error())))
			return
		}
		_, _ = os.Stdout.Write(prettyJSON.Color(prettyJSON.Pretty(json), nil))

	default:
		fmt.Println(colorizeError(fmt.Sprintf("Unknown command. %s", replAssistanceMessage)))
	}
}

func printReplWelcome() {
	fmt.Printf("Welcome to Cadence %s!\n%s\n\n", cadence.Version, replAssistanceMessage)
}
