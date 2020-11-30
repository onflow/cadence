/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/logrusorgru/aurora"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const replFilename = "REPL"

func RunREPL() {
	printWelcome()

	lineNumber := 1
	lineIsContinuation := false
	code := ""

	codes := map[string]string{}

	repl, err := runtime.NewREPL(
		func(err error) {
			// TODO: handle imports
			cmd.PrettyPrintError(os.Stderr, err, replFilename, map[string]string{replFilename: code})
		},
		func(value interpreter.Value) {
			if _, isVoid := value.(*interpreter.VoidValue); isVoid || value == nil {
				return
			}

			println(colorizeResult(value))
		},
		[]sema.Option{
			sema.WithImportHandler(
				func(checker *sema.Checker, location common.Location) (sema.Import, *sema.CheckerError) {
					stringLocation, ok := location.(common.StringLocation)

					if !ok {
						return nil, &sema.CheckerError{
							Errors: []error{
								fmt.Errorf("cannot import `%s`. only files are supported", location),
							},
						}
					}

					importChecker, err := checker.EnsureLoaded(
						location,
						func() *ast.Program {
							filename := string(stringLocation)
							imported, _ := cmd.PrepareProgramFromFile(filename, codes)
							return imported
						},
					)
					if err != nil {
						return nil, err
					}

					return sema.CheckerImport{
						Checker: importChecker,
					}, nil
				},
			),
		},
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

const helpMessage = `
Enter declarations and statements to evaluate them.
Commands are prefixed with a dot. Valid commands are:

.exit     Exit the interpreter
.help     Print this help message

Press ^C to abort current expression, ^D to exit
`

const assistanceMessage = `Type '.help' for assistance.`

func handleCommand(command string) {
	switch command {
	case ".exit":
		os.Exit(0)
	case ".help":
		println(helpMessage)
	default:
		println(colorizeError(fmt.Sprintf("Unknown command. %s", assistanceMessage)))
	}
}

func printWelcome() {
	fmt.Printf("Welcome to Cadence %s!\n%s\n\n", cadence.Version, assistanceMessage)
}

func colorizeResult(value interpreter.Value) string {
	return aurora.Colorize(fmt.Sprint(value), aurora.YellowFg|aurora.BrightFg).String()
}

func colorizeError(message string) string {
	return aurora.Colorize(message, aurora.RedFg|aurora.BrightFg|aurora.BoldFm).String()
}
