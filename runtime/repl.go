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

package runtime

import (
	"bytes"
	"fmt"
	goRuntime "runtime"
	"sort"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type REPL struct {
	checker  *sema.Checker
	inter    *interpreter.Interpreter
	onError  func(err error, location Location, codes map[Location][]byte)
	onResult func(interpreter.Value)
	codes    map[Location][]byte
}

func NewREPL(
	onError func(err error, location Location, codes map[Location][]byte),
	onResult func(interpreter.Value),
) (*REPL, error) {

	checkers := map[Location]*sema.Checker{}
	codes := map[Location][]byte{}

	checkerConfig := cmd.DefaultCheckerConfig(checkers, codes)
	checkerConfig.AccessCheckMode = sema.AccessCheckModeNotSpecifiedUnrestricted

	checker, err := sema.NewChecker(
		nil,
		common.REPLLocation{},
		nil,
		checkerConfig,
	)
	if err != nil {
		return nil, err
	}

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	// necessary now due to log being looked up in the
	// interpreter's activations instead of the checker
	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.NewLogFunction(cmd.StandardOutputLogger{}))

	interpreterConfig := &interpreter.Config{
		Storage: storage,
		UUIDHandler: func() (uint64, error) {
			defer func() { uuid++ }()
			return uuid, nil
		},
		BaseActivation: baseActivation,
	}

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreterConfig,
	)
	if err != nil {
		return nil, err
	}

	repl := &REPL{
		checker:  checker,
		inter:    inter,
		onError:  onError,
		onResult: onResult,
		codes:    codes,
	}
	return repl, nil
}

func (r *REPL) handleCheckerError() error {
	err := r.checker.CheckerError()
	if err == nil {
		return nil
	}
	if r.onError != nil {
		r.onError(err, r.checker.Location, r.codes)
	}
	return err
}

var lineSep = []byte{'\n'}

func (r *REPL) Accept(code []byte) (inputIsComplete bool, err error) {

	// We need two codes:
	//
	// 1. The code used for parsing and type checking (`code`).
	//
	//    This is only the code that was just entered in the REPL,
	//    as we do not want to re-check and re-run the whole program already previously entered into the REPL –
	//    the checker's and interpreter's state are kept, and they already have the previously entered declarations.
	//
	//    However, just parsing the entered code would result in an AST with wrong position information,
	//    the line number would be always 1. To adjust the line information, we prepend the new code with empty lines.
	//
	// 2. The code used for error pretty printing (`codes`).
	//
	//    We temporarily update the full code of the whole program to include the new code.
	//    This allows the error pretty printer to properly refer to previous code (instead of empty lines),
	//    as well as the new code.
	//    However, if an error occurs, we revert the addition of the new code
	//    and leave the program code as it was before.

	// Append the new code to the existing code (used for error reporting),
	// temporarily, so that errors for the new code can be reported

	currentCode := r.codes[r.checker.Location]

	r.codes[r.checker.Location] = append(currentCode[:], code...)

	defer func() {
		if panicResult := recover(); panicResult != nil {

			var err error

			switch panicResult := panicResult.(type) {
			case goRuntime.Error:
				// don't recover Go or external panics
				panic(panicResult)
			case error:
				err = panicResult
				break
			default:
				err = fmt.Errorf("%s", panicResult)
				break
			}

			r.onError(err, r.checker.Location, r.codes)
		}
	}()

	// If the new code results in a parsing or checking error,
	// reset the code
	defer func() {
		if err != nil {
			r.codes[r.checker.Location] = currentCode
		}
	}()

	// Only parse the new code, and ignore the existing code.
	//
	// Prefix the new code with empty lines,
	// so that the line number is correct in error messages

	lineSepCount := bytes.Count(currentCode, lineSep)

	if lineSepCount > 0 {
		prefixedCode := make([]byte, lineSepCount+len(code))

		for i := 0; i < lineSepCount; i++ {
			prefixedCode[i] = '\n'
		}
		copy(prefixedCode[lineSepCount:], code)

		code = prefixedCode
	}

	// TODO: detect if the input is complete
	inputIsComplete = true

	result, errs := parser.ParseStatements(nil, code, parser.Config{})
	if len(errs) > 0 {
		err = parser.Error{
			Code:   code,
			Errors: errs,
		}
	}

	if !inputIsComplete {
		return
	}

	if err != nil {
		r.onError(err, r.checker.Location, r.codes)
		return
	}

	r.checker.ResetErrors()

	for _, element := range result {

		switch element := element.(type) {
		case ast.Declaration:
			program := ast.NewProgram(nil, []ast.Declaration{element})

			r.checker.CheckProgram(program)
			err = r.handleCheckerError()
			if err != nil {
				return
			}

			r.inter.VisitProgram(program)

		case ast.Statement:
			r.checker.Program = nil

			r.checker.CheckStatement(element)

			err = r.handleCheckerError()
			if err != nil {
				return
			}

			result := ast.AcceptStatement[interpreter.StatementResult](element, r.inter)

			onResult := r.onResult
			if result, ok := result.(interpreter.ExpressionResult); ok && onResult != nil {
				onResult(result)
			}

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return
}

type REPLSuggestion struct {
	Name, Description string
}

func (r *REPL) Suggestions() (result []REPLSuggestion) {
	names := map[string]string{}

	r.checker.Elaboration.GlobalValues.Foreach(func(name string, variable *sema.Variable) {
		if names[name] != "" {
			return
		}
		names[name] = variable.Type.String()
	})

	// Iterating over the dictionary of names is safe,
	// as the suggested entries are sorted afterwards

	for name, description := range names { //nolint:maprangecheck
		result = append(result, REPLSuggestion{
			Name:        name,
			Description: description,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]
		return a.Name < b.Name
	})

	return
}

func (r *REPL) GetGlobal(name string) interpreter.Value {
	variable := r.inter.Globals.Get(name)
	if variable == nil {
		return nil
	}
	return variable.GetValue()
}

func (r *REPL) ExportValue(value interpreter.Value) (cadence.Value, error) {
	return ExportValue(
		value, r.inter,
		interpreter.LocationRange{
			Location: r.checker.Location,
			// TODO: hasPosition
		},
	)
}
