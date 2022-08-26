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
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
)

type REPL struct {
	checker  *sema.Checker
	inter    *interpreter.Interpreter
	onError  func(err error, location Location, codes map[Location]string)
	onResult func(interpreter.Value)
	codes    map[Location]string
}

func NewREPL(
	onError func(err error, location Location, codes map[Location]string),
	onResult func(interpreter.Value),
) (*REPL, error) {

	checkers := map[Location]*sema.Checker{}
	codes := map[Location]string{}

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

	// NOTE: storage option must be provided *before* the predeclared values option,
	// as predeclared values may rely on storage

	interpreterConfig := &interpreter.Config{
		Storage: storage,
		UUIDHandler: func() (uint64, error) {
			defer func() { uuid++ }()
			return uuid, nil
		},
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

func (r *REPL) handleCheckerError() bool {
	err := r.checker.CheckerError()
	if err == nil {
		return true
	}
	if r.onError != nil {
		r.onError(err, r.checker.Location, r.codes)
	}
	return false
}

func (r *REPL) handleResult(value interpreter.Value) {
	if r.onResult == nil {
		return
	}
	r.onResult(value)
}

func (r *REPL) Accept(code string) (inputIsComplete bool) {

	r.codes[r.checker.Location] = code

	// TODO: detect if the input is complete
	inputIsComplete = true

	var err error
	result, errs := parser.ParseStatements(code, nil)
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
			if !r.handleCheckerError() {
				return
			}

			// TODO:
			//r.inter.visitProgram(program)

		case ast.Statement:
			r.checker.Program = nil

			r.checker.CheckStatement(element)

			if !r.handleCheckerError() {
				return
			}

			// TODO:
			//result := ast.AcceptStatement[interpreter.Value](element, r.inter)
			//if result, ok := result.(interpreter.ExpressionStatementResult); ok {
			//	r.handleResult(result.Value)
			//}

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
