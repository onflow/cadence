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
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

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
	lineNumber := 1
	var lineIsContinuation bool
	var code string
	var history []string

	printReplWelcome()

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

		history = append(history, code)
		err = writeHistory(history)
		if err != nil {
			panic(err)
		}

		lineIsContinuation = false
		code = ""
	}

	suggest := func(d prompt.Document) []prompt.Suggest {
		if len(d.GetWordBeforeCursor()) == 0 {
			return nil
		}

		var suggests []prompt.Suggest

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

	history, _ = readHistory()

	options := []prompt.Option{
		prompt.OptionLivePrefix(changeLivePrefix),
		prompt.OptionHistory(history),
	}
	prompt.New(executor, suggest, options...).Run()
}

func cadenceDirPath() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userCacheDir, "cadence"), nil
}

func historyFilePath() (string, error) {
	cadenceDirPath, err := cadenceDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cadenceDirPath, "replHistory"), nil
}

func readHistory() ([]string, error) {
	path, err := historyFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine cadence directory path: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open history path: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	var result []string

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read history: %w", err)
		}

		if len(row) <= 0 {
			return nil, fmt.Errorf("failed to read history: invalid row %d", len(result)+1)
		}

		result = append(result, row[0])
	}

	return result, nil
}

func writeHistory(history []string) error {
	cadenceDirPath, err := cadenceDirPath()
	if err != nil {
		return fmt.Errorf("failed to determine cadence directory path: %w", err)
	}

	err = os.MkdirAll(cadenceDirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create cadence directory: %w", err)
	}

	path, err := historyFilePath()
	if err != nil {
		return fmt.Errorf("failed to determine history path: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create history: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	for _, code := range history {
		err = writer.Write([]string{strings.TrimRightFunc(code, unicode.IsSpace)})
		if err != nil {
			return fmt.Errorf("failed to write history: %w", err)
		}
	}

	writer.Flush()

	return nil
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
