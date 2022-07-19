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

// This Provides utility methods to easily run test-scripts.
// Example use-case:
//   - To run all tests in a script:
//         RunTests("source code")
//   - To run a single test method in a script:
//         RunTest("source code", "testMethodName")
//
// It is assumed that all test methods start with the 'test' prefix.

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

func init() {
	// TODO: find a better way to do this.
	// 	Option I: Move this logic behind a 'test' flag
	// 	Option II: Virtually move the 'EmulatorBackend' native struct inside the 'Test' contract
	//	Any other?
	sema.NativeCompositeTypes[stdlib.EmulatorBackendType.QualifiedIdentifier()] = stdlib.EmulatorBackendType
}

func newInterpreterFromChecker(checker *sema.Checker) (*interpreter.Interpreter, error) {
	predeclaredInterpreterValues := stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.BuiltinValues.ToInterpreterValueDeclarations()...)
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.HelperFunctions.ToInterpreterValueDeclarations()...)

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
		interpreter.WithContractValueHandler(
			func(inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
				invocationRange ast.Range) *interpreter.CompositeValue {

				switch compositeType.Location {
				case stdlib.CryptoChecker.Location:
					contract, err := stdlib.NewCryptoContract(
						inter,
						constructorGenerator(common.Address{}),
						invocationRange,
					)
					if err != nil {
						panic(err)
					}
					return contract

				case stdlib.TestContractLocation:
					contract, err := stdlib.NewTestContract(
						inter,
						constructorGenerator(common.Address{}),
						invocationRange,
					)
					if err != nil {
						panic(err)
					}
					return contract

				default:
					panic("importing other contracts not supported yet")
				}
			},
		),
	)
}

func newChecker(program *ast.Program) (*sema.Checker, error) {
	predeclaredSemaValues := stdlib.BuiltinFunctions.ToSemaValueDeclarations()
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.HelperFunctions.ToSemaValueDeclarations()...)

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
