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
	"fmt"
	"sort"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type REPL struct {
	checker  *sema.Checker
	inter    *interpreter.Interpreter
	onError  func(err error, location common.Location, codes map[common.Location]string)
	onResult func(interpreter.Value)
	codes    map[common.Location]string
}

func NewREPL(
	onError func(err error, location common.Location, codes map[common.Location]string),
	onResult func(interpreter.Value),
	checkerOptions []sema.Option,
	interpreterOptions []interpreter.Option,
) (*REPL, error) {

	valueDeclarations := append(
		stdlib.FlowBuiltInFunctions(stdlib.DefaultFlowBuiltinImpls()),
		stdlib.BuiltinFunctions...,
	)

	checkers := map[common.Location]*sema.Checker{}
	codes := map[common.Location]string{}

	var newChecker func(program *ast.Program, location common.Location) (*sema.Checker, error)

	checkerOptions = append(
		[]sema.Option{
			sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
			sema.WithPredeclaredTypes(typeDeclarations),
			sema.WithAccessCheckMode(sema.AccessCheckModeNotSpecifiedUnrestricted),
			sema.WithImportHandler(
				func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					stringLocation, ok := importedLocation.(common.StringLocation)

					if !ok {
						return nil, &sema.CheckerError{
							Location: checker.Location,
							Codes:    codes,
							Errors: []error{
								fmt.Errorf("cannot import `%s`. only files are supported", importedLocation),
							},
						}
					}

					importedChecker, ok := checkers[importedLocation]
					if !ok {
						importedProgram, _ := cmd.PrepareProgramFromFile(stringLocation, codes)
						importedChecker, _ = newChecker(importedProgram, importedLocation)
						checkers[importedLocation] = importedChecker
					}

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			),
		},
		checkerOptions...,
	)

	newChecker = func(program *ast.Program, location common.Location) (*sema.Checker, error) {
		return sema.NewChecker(
			program,
			common.REPLLocation{},
			nil,
			checkerOptions...,
		)
	}

	checker, err := newChecker(nil, common.REPLLocation{})
	if err != nil {
		return nil, err
	}

	values := valueDeclarations.ToInterpreterValueDeclarations()

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	interpreterOptions = append(
		[]interpreter.Option{
			interpreter.WithStorage(storage),
			interpreter.WithPredeclaredValues(values),
			interpreter.WithUUIDHandler(func() (uint64, error) {
				defer func() { uuid++ }()
				return uuid, nil
			}),
		},
		interpreterOptions...,
	)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreterOptions...,
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

func (r *REPL) execute(element ast.Element) {
	result := element.Accept(r.inter)
	expStatementRes, ok := result.(interpreter.ExpressionStatementResult)
	if !ok {
		return
	}
	if r.onResult == nil {
		return
	}
	r.onResult(expStatementRes.Value)
}

func (r *REPL) check(element ast.Element, code string) bool {
	element.Accept(r.checker)
	r.codes[r.checker.Location] = code
	return r.handleCheckerError()
}

func (r *REPL) Accept(code string) (inputIsComplete bool) {

	// TODO: detect if the input is complete
	inputIsComplete = true

	var err error
	result, errs := parser2.ParseStatements(code, nil)
	if len(errs) > 0 {
		err = parser2.Error{
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
	r.checker.ResetHints()

	for _, element := range result {

		switch typedElement := element.(type) {
		case ast.Declaration:
			program := ast.NewProgram(nil, []ast.Declaration{typedElement})

			if !r.check(program, code) {
				return
			}

			r.execute(typedElement)

		case ast.Statement:
			r.checker.Program = nil

			if !r.check(typedElement, code) {
				return
			}

			r.execute(typedElement)

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
