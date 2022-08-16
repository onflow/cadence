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
	"github.com/onflow/cadence/runtime/tests/utils"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
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

// ImportResolver is used to resolve and get the source code for imports.
// Must be provided by the user of the TestRunner.
//
type ImportResolver func(location common.Location) (string, error)

// TestRunner runs tests.
//
type TestRunner struct {
	importResolver ImportResolver
}

func NewTestRunner() *TestRunner {
	return &TestRunner{}
}

func (r *TestRunner) WithImportResolver(importResolver ImportResolver) *TestRunner {
	r.importResolver = importResolver
	return r
}

// RunTest runs a single test in the provided test script.
//
func (r *TestRunner) RunTest(script string, funcName string) error {
	_, inter, err := r.parseCheckAndInterpret(script)
	if err != nil {
		return err
	}

	_, err = inter.Invoke(funcName)
	return err
}

// RunTests runs all the tests in the provided test script.
//
func (r *TestRunner) RunTests(script string) (Results, error) {
	program, inter, err := r.parseCheckAndInterpret(script)
	if err != nil {
		return nil, err
	}

	results := make(Results)

	for _, funcDecl := range program.FunctionDeclarations() {
		funcName := funcDecl.Identifier.Identifier

		if strings.HasPrefix(funcName, testFunctionPrefix) {
			_, err := inter.Invoke(funcName)
			results[funcName] = err
		}
	}

	return results, nil
}

func (r *TestRunner) parseCheckAndInterpret(script string) (*ast.Program, *interpreter.Interpreter, error) {
	program, err := parser.ParseProgram(script, nil)
	if err != nil {
		return nil, nil, err
	}

	checker, err := r.newChecker(program, utils.TestLocation)
	if err != nil {
		return nil, nil, err
	}

	err = checker.Check()
	if err != nil {
		return nil, nil, err
	}

	// TODO: validate test function signature
	//   e.g: no return values, no arguments, etc.

	inter, err := r.newInterpreterFromChecker(checker)
	if err != nil {
		return nil, nil, err
	}

	err = inter.Interpret()
	if err != nil {
		return nil, nil, err
	}

	return program, inter, nil
}

func (r *TestRunner) newInterpreterFromChecker(checker *sema.Checker) (*interpreter.Interpreter, error) {
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
				importedChecker, err := r.parseAndCheckImport(location)
				if err != nil {
					panic(err)
				}

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			}
		}),
		interpreter.WithContractValueHandler(func(
			inter *interpreter.Interpreter,
			compositeType *sema.CompositeType,
			constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
			invocationRange ast.Range,
		) interpreter.ContractValue {

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
				// During tests, imported contracts can be constructed using the constructor,
				// similar to structs. Therefore, generate a constructor function.
				return constructorGenerator(common.Address{})
			}
		},
		),
	)
}

func (r *TestRunner) newChecker(program *ast.Program, location common.Location) (*sema.Checker, error) {
	predeclaredSemaValues := stdlib.BuiltinFunctions.ToSemaValueDeclarations()
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.BuiltinValues.ToSemaValueDeclarations()...)
	predeclaredSemaValues = append(predeclaredSemaValues, stdlib.HelperFunctions.ToSemaValueDeclarations()...)

	return sema.NewChecker(
		program,
		location,
		nil,
		true,
		sema.WithPredeclaredValues(predeclaredSemaValues),
		sema.WithPredeclaredTypes(stdlib.FlowDefaultPredeclaredTypes),
		sema.WithContractVariableHandler(func(
			checker *sema.Checker,
			declaration *ast.CompositeDeclaration,
			compositeType *sema.CompositeType,
		) sema.VariableDeclaration {

			constructorType, constructorArgumentLabels := checker.CompositeConstructorType(declaration, compositeType)

			return sema.VariableDeclaration{
				Identifier:               declaration.Identifier.Identifier,
				Type:                     constructorType,
				DocString:                declaration.DocString,
				Access:                   declaration.Access,
				Kind:                     declaration.DeclarationKind(),
				Pos:                      declaration.Identifier.Pos,
				IsConstant:               true,
				ArgumentLabels:           constructorArgumentLabels,
				AllowOuterScopeShadowing: false,
			}
		}),

		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				case stdlib.TestContractLocation:
					elaboration = stdlib.TestContractChecker.Elaboration

				default:
					importedChecker, err := r.parseAndCheckImport(importedLocation)
					if err != nil {
						return nil, err
					}

					elaboration = importedChecker.Elaboration
				}

				return sema.ElaborationImport{
					Elaboration: elaboration,
				}, nil
			},
		),
	)
}

func (r *TestRunner) parseAndCheckImport(location common.Location) (*sema.Checker, error) {
	if r.importResolver == nil {
		return nil, ImportResolverNotProvidedError{}
	}

	code, err := r.importResolver(location)
	if err != nil {
		return nil, err
	}

	program, err := parser.ParseProgram(code, nil)
	if err != nil {
		return nil, err
	}

	checker, err := r.newChecker(program, location)
	if err != nil {
		return nil, err
	}

	err = checker.Check()
	if err != nil {
		return nil, err
	}

	return checker, nil
}

// PrettyPrintResults is a utility function to pretty print the test results.
//
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
