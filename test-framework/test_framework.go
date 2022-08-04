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

package test

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/execution/testutil"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/fvm/state"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
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

const setupFunctionName = "setup"

const tearDownFunctionName = "tearDown"

var testScriptLocation = common.NewScriptLocation(nil, []byte("test"))

type Results []Result

type Result struct {
	testName string
	err      error
}

// ImportResolver is used to resolve and get the source code for imports.
// Must be provided by the user of the TestRunner.
//
type ImportResolver func(location common.Location) (string, error)

// TestRunner runs tests.
//
type TestRunner struct {
	importResolver ImportResolver
	testRuntime    *runtime.TestFrameworkRuntime
}

func NewTestRunner() *TestRunner {
	return &TestRunner{
		testRuntime: runtime.NewTestFrameworkRuntime(),
	}
}

func (r *TestRunner) WithImportResolver(importResolver ImportResolver) *TestRunner {
	r.importResolver = importResolver
	return r
}

// RunTest runs a single test in the provided test script.
//
func (r *TestRunner) RunTest(script string, funcName string) (result *Result, err error) {
	defer func() {
		recoverPanics(func(internalErr error) {
			err = internalErr
		})
	}()

	_, inter, err := r.parseCheckAndInterpret(script)
	if err != nil {
		return nil, err
	}

	// Run test `setup()` before running the test function.
	err = r.runTestSetup(inter)
	if err != nil {
		return nil, err
	}

	_, testResult := inter.Invoke(funcName)

	// Run test `tearDown()` once running all test functions are completed.
	err = r.runTestTearDown(inter)

	return &Result{
		testName: funcName,
		err:      testResult,
	}, err
}

// RunTests runs all the tests in the provided test script.
//
func (r *TestRunner) RunTests(script string) (results Results, err error) {
	defer func() {
		recoverPanics(func(internalErr error) {
			err = internalErr
		})
	}()

	program, inter, err := r.parseCheckAndInterpret(script)
	if err != nil {
		return nil, err
	}

	results = make(Results, 0)

	// Run test `setup()` before test functions
	err = r.runTestSetup(inter)
	if err != nil {
		return nil, err
	}

	for _, funcDecl := range program.Program.FunctionDeclarations() {
		funcName := funcDecl.Identifier.Identifier

		if !strings.HasPrefix(funcName, testFunctionPrefix) {
			continue
		}

		err := r.invokeTestFunction(inter, funcName)

		results = append(results, Result{
			testName: funcName,
			err:      err,
		})
	}

	// Run test `tearDown()` once running all test functions are completed.
	err = r.runTestTearDown(inter)

	return results, err
}

func (r *TestRunner) runTestSetup(inter *interpreter.Interpreter) error {
	if !hasSetup(inter) {
		return nil
	}

	return r.invokeTestFunction(inter, setupFunctionName)
}

func hasSetup(inter *interpreter.Interpreter) bool {
	_, ok := inter.Globals.Get(setupFunctionName)
	return ok
}

func (r *TestRunner) runTestTearDown(inter *interpreter.Interpreter) error {
	if !hasTearDown(inter) {
		return nil
	}

	return r.invokeTestFunction(inter, tearDownFunctionName)
}

func hasTearDown(inter *interpreter.Interpreter) bool {
	_, ok := inter.Globals.Get(tearDownFunctionName)
	return ok
}

func (r *TestRunner) invokeTestFunction(inter *interpreter.Interpreter, funcName string) (err error) {
	// Individually fail each test-case for any internal error.
	defer func() {
		recoverPanics(func(internalErr error) {
			err = internalErr
		})
	}()

	_, err = inter.Invoke(funcName)
	return err
}

func recoverPanics(onError func(error)) {
	r := recover()
	switch r := r.(type) {
	case nil:
		return
	case error:
		onError(r)
	default:
		onError(fmt.Errorf("%s", r))
	}
}

func (r *TestRunner) parseCheckAndInterpret(script string) (*interpreter.Program, *interpreter.Interpreter, error) {
	ctx := runtime.Context{
		Interface: newScriptEnvironment(),
		Location:  testScriptLocation,
	}

	var checkerOptions = r.checkerOptions(ctx)
	var interpreterOptions = r.interpreterOptions(ctx)

	program, err := r.testRuntime.ParseAndCheck([]byte(script), ctx, checkerOptions, interpreterOptions)
	if err != nil {
		return nil, nil, err
	}

	// TODO: validate test function signature
	//   e.g: no return values, no arguments, etc.

	memoryGauge, _ := ctx.Interface.(common.MemoryGauge)
	storage := runtime.NewStorage(ctx.Interface, memoryGauge)

	inter, err := r.testRuntime.Interpret(
		program,
		storage,
		ctx,
		checkerOptions,
		interpreterOptions,
	)
	if err != nil {
		return nil, nil, err
	}

	err = inter.Interpret()
	if err != nil {
		return nil, nil, err
	}

	return program, inter, nil
}

func (r *TestRunner) checkerOptions(ctx runtime.Context) []sema.Option {
	return []sema.Option{
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				case stdlib.TestContractLocation:
					elaboration = stdlib.TestContractChecker.Elaboration

				default:
					importedProgram, importedElaboration, err := r.parseAndCheckImport(importedLocation, ctx)
					if err != nil {
						return nil, err
					}

					elaboration = importedElaboration

					contractDecl := importedProgram.SoleContractDeclaration()
					compositeType := elaboration.CompositeDeclarationTypes[contractDecl]

					constructorType, constructorArgumentLabels :=
						sema.CompositeConstructorType(importedElaboration, contractDecl, compositeType)

					// Remove the contract variable, and instead declare a constructor.
					elaboration.GlobalValues.Delete(compositeType.Identifier)

					// Declare a constructor
					_, err = checker.ValueActivations.Declare(sema.VariableDeclaration{
						Identifier:               contractDecl.Identifier.Identifier,
						Type:                     constructorType,
						DocString:                contractDecl.DocString,
						Access:                   contractDecl.Access,
						Kind:                     contractDecl.DeclarationKind(),
						Pos:                      contractDecl.Identifier.Pos,
						IsConstant:               true,
						ArgumentLabels:           constructorArgumentLabels,
						AllowOuterScopeShadowing: false,
					})

					if err != nil {
						return nil, err
					}
				}

				return sema.ElaborationImport{
					Elaboration: elaboration,
				}, nil
			},
		),
	}
}

func (r *TestRunner) interpreterOptions(ctx runtime.Context) []interpreter.Option {
	return []interpreter.Option{
		// TODO: The default injected fields handler only supports 'address' locations.
		//   However, during tests, it is possible to get non-address locations. e.g: file paths.
		//   Thus, need to properly handle them. Make this nil for now.
		interpreter.WithInjectedCompositeFieldsHandler(nil),

		interpreter.WithTestFramework(NewEmulatorBackend()),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
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
					importedProgram, importedElaboration, err := r.parseAndCheckImport(location, ctx)
					if err != nil {
						panic(err)
					}

					program := &interpreter.Program{
						Program:     importedProgram,
						Elaboration: importedElaboration,
					}

					subInterpreter, err := inter.NewSubInterpreter(program, location)
					if err != nil {
						panic(err)
					}

					return interpreter.InterpreterImport{
						Interpreter: subInterpreter,
					}
				}
			},
		),
		interpreter.WithContractValueHandler(func(
			inter *interpreter.Interpreter,
			compositeType *sema.CompositeType,
			constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
			invocationRange ast.Range,
		) interpreter.Value {

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
		}),
	}
}

// newScriptEnvironment creates an environment for test scripts to run.
// Leverages the functionality of FVM.
//
func newScriptEnvironment() *fvm.ScriptEnv {
	vm := fvm.NewVirtualMachine(runtime.NewTestFrameworkRuntime())
	ctx := fvm.NewContext(zerolog.Nop())
	emptyPrograms := programs.NewEmptyPrograms()

	view := testutil.RootBootstrappedLedger(vm, ctx)
	v := view.NewChild()

	st := state.NewState(v, state.WithMaxInteractionSizeAllowed(math.MaxUint64))
	sth := state.NewStateHolder(st)

	return fvm.NewScriptEnvironment(context.Background(), ctx, vm, sth, emptyPrograms)
}

func (r *TestRunner) parseAndCheckImport(location common.Location, startCtx runtime.Context) (*ast.Program, *sema.Elaboration, error) {
	if r.importResolver == nil {
		return nil, nil, ImportResolverNotProvidedError{}
	}

	code, err := r.importResolver(location)
	if err != nil {
		return nil, nil, err
	}

	ctx := startCtx.WithLocation(location)

	// Use separate checker-options and interpreter-options for the imports.
	// For e.g: imports are not supported for imported programs (i.e: nested imports are not supported).
	var checkerOptions = []sema.Option{
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {
				return nil, fmt.Errorf("nested imports are not supported")
			},
		),
	}

	var interpreterOptions = []interpreter.Option{
		interpreter.WithInjectedCompositeFieldsHandler(nil),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				panic(fmt.Errorf("nested imports are not supported"))
			},
		),
	}

	program, err := r.testRuntime.ParseAndCheck([]byte(code), ctx, checkerOptions, interpreterOptions)
	if err != nil {
		return nil, nil, err
	}

	return program.Program, program.Elaboration, nil
}

// PrettyPrintResults is a utility function to pretty print the test results.
//
func PrettyPrintResults(results Results) string {
	var sb strings.Builder
	sb.WriteString("Test Results\n")
	for _, result := range results {
		sb.WriteString(PrettyPrintResult(result.testName, result.err))
	}
	return sb.String()
}

func PrettyPrintResult(funcName string, err error) string {
	if err == nil {
		return fmt.Sprintf("- PASS: %s\n", funcName)
	}

	return fmt.Sprintf("- FAIL: %s\n\t\t%s\n", funcName, err.Error())
}
