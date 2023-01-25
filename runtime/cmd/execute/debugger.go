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
	"text/tabwriter"

	"github.com/c-bata/go-prompt"

	"github.com/onflow/cadence/runtime/interpreter"
)

const commandShortHelp = "h"
const commandLongHelp = "help"
const commandShortContinue = "c"
const commandLongContinue = "continue"
const commandShortNext = "n"
const commandLongNext = "next"
const commandLongExit = "exit"
const commandShortShow = "s"
const commandLongShow = "show"
const commandShortWhere = "w"
const commandLongWhere = "where"

var debuggerCommandSuggestions = []prompt.Suggest{
	{Text: commandLongContinue, Description: "Continue"},
	{Text: commandLongNext, Description: "Next / step"},
	{Text: commandLongWhere, Description: "Location info"},
	{Text: commandLongShow, Description: "Show variable(s)"},
	{Text: commandLongExit, Description: "Exit"},
	{Text: commandLongHelp, Description: "Help"},
}

type InteractiveDebugger struct {
	debugger *interpreter.Debugger
	stop     interpreter.Stop
}

func NewInteractiveDebugger(debugger *interpreter.Debugger, stop interpreter.Stop) *InteractiveDebugger {
	return &InteractiveDebugger{
		debugger: debugger,
		stop:     stop,
	}
}

func (d *InteractiveDebugger) Continue() {
	d.debugger.Continue()
}

func (d *InteractiveDebugger) Next() {
	d.stop = d.debugger.Next()
}

// Show shows the values for the variables with the given names.
// If no names are given, lists all non-base variables
func (d *InteractiveDebugger) Show(names []string) {
	current := d.debugger.CurrentActivation(d.stop.Interpreter)
	switch len(names) {
	case 0:
		for name := range current.FunctionValues() { //nolint:maprange
			fmt.Println(name)
		}

	case 1:
		name := names[0]
		variable := current.Find(name)
		if variable == nil {
			fmt.Println(colorizeError(fmt.Sprintf("error: variable '%s' is not in scope", name)))
			return
		}

		fmt.Println(formatValue(variable.GetValue()))

	default:
		for _, name := range names {
			variable := current.Find(name)
			if variable == nil {
				continue
			}

			fmt.Printf(
				"%s = %s\n",
				name,
				formatValue(variable.GetValue()),
			)
		}
	}
}

func (d *InteractiveDebugger) Run() {

	executor := func(in string) {
		in = strings.TrimSpace(in)

		parts := strings.Split(in, " ")

		command, arguments := parts[0], parts[1:]

		switch command {
		case "":
			break
		case commandShortContinue, commandLongContinue:
			d.Continue()
		case commandShortNext, commandLongNext:
			d.Next()
		case commandShortShow, commandLongShow:
			d.Show(arguments)
		case commandShortWhere, commandLongWhere:
			d.Where()
		case commandShortHelp, commandLongHelp:
			d.Help()
		case commandLongExit:
			os.Exit(0)
		default:
			message := fmt.Sprintf("error: '%s' is not a valid command.\n", in)
			fmt.Println(colorizeError(message))
		}
	}

	suggest := func(d prompt.Document) []prompt.Suggest {
		wordBeforeCursor := d.GetWordBeforeCursor()
		if len(wordBeforeCursor) == 0 {
			return nil
		}

		return prompt.FilterHasPrefix(debuggerCommandSuggestions, wordBeforeCursor, true)
	}

	exitChecker := func(in string, breakline bool) bool {
		switch in {
		case commandShortContinue, commandLongContinue:
			return breakline
		}
		return false
	}

	fmt.Println()

	prompt.New(
		executor,
		suggest,
		prompt.OptionPrefix("(cdb) "),
		prompt.OptionSetExitCheckerOnInput(exitChecker),
	).Run()
}

func (d *InteractiveDebugger) Help() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, suggestion := range debuggerCommandSuggestions {
		_, _ = fmt.Fprintf(w,
			"%s\t\t%s\n",
			suggestion.Text,
			suggestion.Description,
		)
	}
	_ = w.Flush()
}

func (d *InteractiveDebugger) Where() {
	fmt.Printf(
		"%s @ %d\n",
		d.stop.Interpreter.Location,
		d.stop.Statement.StartPosition().Line,
	)
}
