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
	"github.com/onflow/cadence/runtime/sema"
)

type ConsoleREPL struct {
	lineIsContinuation bool
	code               string
	lineNumber         int
	errorPrettyPrinter pretty.ErrorPrettyPrinter
	repl               *runtime.REPL
	historyWriter      *csv.Writer
}

func NewConsoleREPL() (*ConsoleREPL, error) {
	consoleREPL := &ConsoleREPL{
		lineNumber:         1,
		errorPrettyPrinter: pretty.NewErrorPrettyPrinter(os.Stderr, true),
	}

	repl, err := runtime.NewREPL()
	if err != nil {
		return nil, err
	}

	repl.OnError = consoleREPL.onError
	repl.OnResult = consoleREPL.onResult

	consoleREPL.repl = repl

	return consoleREPL, nil
}

func (consoleREPL *ConsoleREPL) onError(err error, location common.Location, codes map[common.Location][]byte) {
	printErr := consoleREPL.errorPrettyPrinter.PrettyPrintError(err, location, codes)
	if printErr != nil {
		panic(printErr)
	}
}

func (consoleREPL *ConsoleREPL) onResult(value interpreter.Value) {
	fmt.Println(colorizeValue(value))
}

func (consoleREPL *ConsoleREPL) handleCommand(command string) {
	parts := strings.SplitN(command, " ", 2)
	for _, command := range commands {
		if command.name != parts[0][1:] {
			continue
		}

		var argument string
		if len(parts) > 1 {
			argument = parts[1]
		}

		command.handler(consoleREPL, argument)
		return
	}

	printError(fmt.Sprintf("Unknown command. %s", replAssistanceMessage))
}

func (consoleREPL *ConsoleREPL) exportVariable(name string) {
	repl := consoleREPL.repl

	global := repl.GetGlobal(name)
	if global == nil {
		printError(fmt.Sprintf("Undefined global: %s", name))
		return
	}

	value, err := repl.ExportValue(global)
	if err != nil {
		printError(fmt.Sprintf("Failed to export global %s: %s", name, err))
		return
	}

	json, err := jsoncdc.Encode(value)
	if err != nil {
		printError(fmt.Sprintf("Failed to encode global %s to JSON: %s", name, err))
		return
	}

	_, _ = os.Stdout.Write(prettyJSON.Color(prettyJSON.Pretty(json), nil))
}

func (consoleREPL *ConsoleREPL) showType(expression string) {
	repl := consoleREPL.repl

	oldOnExpressionType := repl.OnExpressionType
	repl.OnExpressionType = func(ty sema.Type) {
		fmt.Println(colorizeResult(string(ty.ID())))
	}
	defer func() {
		repl.OnExpressionType = oldOnExpressionType
	}()

	_, err := repl.Accept([]byte(expression+"\n"), false)
	if err == nil {
		consoleREPL.lineNumber++
	}
}

func (consoleREPL *ConsoleREPL) execute(line string) {
	if consoleREPL.code == "" && strings.HasPrefix(line, ".") {
		consoleREPL.handleCommand(line)
		consoleREPL.code = ""
		return
	}

	consoleREPL.code += line + "\n"

	inputIsComplete, err := consoleREPL.repl.Accept([]byte(consoleREPL.code), true)
	if err == nil {
		consoleREPL.lineNumber++

		if !inputIsComplete {
			consoleREPL.lineIsContinuation = true
			return
		}
	}

	err = consoleREPL.appendHistory()
	if err != nil {
		panic(err)
	}

	consoleREPL.lineIsContinuation = false
	consoleREPL.code = ""
}

func (consoleREPL *ConsoleREPL) suggest(d prompt.Document) []prompt.Suggest {
	wordBeforeCursor := d.GetWordBeforeCursor()

	if len(wordBeforeCursor) == 0 {
		return nil
	}

	var suggests []prompt.Suggest

	if wordBeforeCursor[0] == commandPrefix {
		commandLookupPrefix := wordBeforeCursor[1:]

		for _, command := range commands {
			if !strings.HasPrefix(command.name, commandLookupPrefix) {
				continue
			}
			suggests = append(suggests, prompt.Suggest{
				Text:        fmt.Sprintf("%c%s", commandPrefix, command.name),
				Description: command.description,
			})
		}

	} else {
		for _, suggestion := range consoleREPL.repl.Suggestions() {
			suggests = append(suggests, prompt.Suggest{
				Text:        suggestion.Name,
				Description: suggestion.Description,
			})
		}
	}

	return prompt.FilterHasPrefix(suggests, wordBeforeCursor, false)
}

func (consoleREPL *ConsoleREPL) changeLivePrefix() (string, bool) {
	separator := '>'
	if consoleREPL.lineIsContinuation {
		separator = '.'
	}

	return fmt.Sprintf("%d%c ", consoleREPL.lineNumber, separator), true
}

func (consoleREPL *ConsoleREPL) Run() {

	consoleREPL.printWelcome()

	history, _ := consoleREPL.readHistory()
	err := consoleREPL.openHistoryWriter()
	if err != nil {
		panic(err)
	}

	prompt.New(
		consoleREPL.execute,
		consoleREPL.suggest,
		prompt.OptionLivePrefix(consoleREPL.changeLivePrefix),
		prompt.OptionHistory(history),
	).Run()
}

func printError(message string) {
	println(colorizeError(message))
}

const commandPrefix = '.'

func (consoleREPL *ConsoleREPL) cadenceDirPath() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userCacheDir, "cadence"), nil
}

func (consoleREPL *ConsoleREPL) historyFilePath() (string, error) {
	cadenceDirPath, err := consoleREPL.cadenceDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(cadenceDirPath, "replHistory"), nil
}

func (consoleREPL *ConsoleREPL) readHistory() ([]string, error) {
	path, err := consoleREPL.historyFilePath()
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

		if len(row) == 0 {
			return nil, fmt.Errorf("failed to read history: invalid row %d", len(result)+1)
		}

		result = append(result, row[0])
	}

	return result, nil
}

func (consoleREPL *ConsoleREPL) openHistoryWriter() error {
	cadenceDirPath, err := consoleREPL.cadenceDirPath()
	if err != nil {
		return fmt.Errorf("failed to determine cadence directory path: %w", err)
	}

	err = os.MkdirAll(cadenceDirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create cadence directory: %w", err)
	}

	path, err := consoleREPL.historyFilePath()
	if err != nil {
		return fmt.Errorf("failed to determine history path: %w", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to create history: %w", err)
	}

	consoleREPL.historyWriter = csv.NewWriter(f)

	return nil
}

func (consoleREPL *ConsoleREPL) appendHistory() error {

	writer := consoleREPL.historyWriter

	err := writer.Write([]string{strings.TrimRightFunc(consoleREPL.code, unicode.IsSpace)})
	if err != nil {
		return fmt.Errorf("failed to write history: %w", err)
	}

	writer.Flush()

	return nil
}

const replAssistanceMessage = `Type '.help' for assistance.`

const replHelpMessagePrefix = `
Enter declarations and statements to evaluate them.
Commands are prefixed with a dot. Valid commands are:
`

const replHelpMessageSuffix = `
Press ^C to abort current expression, ^D to exit
`

func (consoleREPL *ConsoleREPL) printHelp() {
	println(replHelpMessagePrefix)

	for _, command := range commands {
		fmt.Printf(
			"%c%s\t%s\n",
			commandPrefix,
			command.name,
			command.description,
		)
	}

	println(replHelpMessageSuffix)
}

type command struct {
	name        string
	description string
	handler     func(repl *ConsoleREPL, argument string)
}

var commands []command

func init() {
	commands = []command{
		{
			name:        "exit",
			description: "Exit the interpreter",
			handler: func(_ *ConsoleREPL, _ string) {
				os.Exit(0)
			},
		},
		{
			name:        "help",
			description: "Show help",
			handler: func(consoleREPL *ConsoleREPL, _ string) {
				consoleREPL.printHelp()
			},
		},
		{
			name:        "export",
			description: "Export variable",
			handler: func(consoleREPL *ConsoleREPL, argument string) {
				name := strings.TrimSpace(argument)
				if len(name) == 0 {
					printError("Missing name")
					return
				}
				consoleREPL.exportVariable(name)
			},
		},
		{
			name:        "type",
			description: "Show type of expression",
			handler: func(consoleREPL *ConsoleREPL, argument string) {
				if len(argument) == 0 {
					printError("Missing expression")
					return
				}

				consoleREPL.showType(argument)
			},
		},
	}
}

func (consoleREPL *ConsoleREPL) printWelcome() {
	fmt.Printf("Welcome to Cadence %s!\n%s\n\n", cadence.Version, replAssistanceMessage)
}
