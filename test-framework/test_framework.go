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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package test

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const testFunctionPrefix = "test"

type Results map[string]error

func RunTest(script string, funcName string) error {
	_, inter := parseCheckAndInterpret(script)
	_, err := inter.Invoke(funcName)
	return err
}

func RunTests(script string) Results {
	program, inter := parseCheckAndInterpret(script)

	results := make(Results)

	for _, funcDecl := range program.FunctionDeclarations() {
		funcName := funcDecl.Identifier.Identifier

		if strings.HasPrefix(funcName, testFunctionPrefix) {
			_, err := inter.Invoke(funcName)
			results[funcName] = err
		}
	}

	return results
}

func parseCheckAndInterpret(script string) (*ast.Program, *interpreter.Interpreter) {
	program, err := parser.ParseProgram(script, nil)

	checker, err := newChecker(program)
	if err != nil {
		panic(err)
	}

	err = checker.Check()
	if err != nil {
		panic(err)
	}

	// TODO: validate test function signature
	//   e.g: no return values, no arguments, etc.

	inter, err := newInterpreterFromChecker(checker)
	if err != nil {
		panic(err)
	}

	err = inter.Interpret()
	if err != nil {
		panic(err)
	}

	return program, inter
}

func newInterpreterFromChecker(checker *sema.Checker) (*interpreter.Interpreter, error) {
	predeclaredInterpreterValues := stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.BuiltinValues.ToInterpreterValueDeclarations()...)

	return interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithStorage(interpreter.NewInMemoryStorage(nil)),
		interpreter.WithTestFramework(NewEmulatorBackend()),
		interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
		interpreter.WithImportLocationHandler(func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			switch location {
			case stdlib.CryptoChecker.Location:
				program := interpreter.ProgramFromChecker(stdlib.CryptoChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			case stdlib.TestContractLocation:
				program := interpreter.ProgramFromChecker(stdlib.TestContractChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			default:
				panic(errors.NewUnexpectedError("importing programs not implemented"))
			}
		}),
	)
}

func newChecker(program *ast.Program) (*sema.Checker, error) {
	predeclaredSemaValues := stdlib.BuiltinFunctions.ToSemaValueDeclarations()
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)

	return sema.NewChecker(
		program,
		utils.TestLocation,
		nil,
		true,
		sema.WithPredeclaredValues(predeclaredSemaValues),
		sema.WithPredeclaredTypes(stdlib.FlowDefaultPredeclaredTypes),
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				case stdlib.TestContractLocation:
					elaboration = stdlib.TestContractChecker.Elaboration

				default:
					return nil, errors.NewUnexpectedError("importing programs not implemented")
				}

				return sema.ElaborationImport{
					Elaboration: elaboration,
				}, nil
			},
		),
	)
}

func PrettyPrintResults(results Results) string {
	var sb strings.Builder
	sb.WriteString("Test Results\n")
	for funcName, err := range results {
		sb.WriteString(PrettyPrintResult(funcName, err))
	}
	return sb.String()
}

func PrettyPrintResult(funcName string, err error) string {
	if err == nil {
		return fmt.Sprintf("- PASS: %s\n", funcName)
	}

	return fmt.Sprintf("- FAIL: %s\n\t\t%s\n", funcName, err.Error())
}
