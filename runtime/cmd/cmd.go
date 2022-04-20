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

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func must(err error, location common.Location, codes map[common.LocationID]string) {
	if err == nil {
		return
	}
	printErr := pretty.NewErrorPrettyPrinter(os.Stderr, true).
		PrettyPrintError(err, location, codes)
	if printErr != nil {
		panic(printErr)
	}
	os.Exit(1)
}

func mustClosure(location common.Location, codes map[common.LocationID]string) func(error) {
	return func(e error) {
		must(e, location, codes)
	}
}

func PrepareProgramFromFile(location common.StringLocation, codes map[common.LocationID]string) (*ast.Program, func(error)) {
	codeBytes, err := ioutil.ReadFile(string(location))

	program, must := PrepareProgram(string(codeBytes), location, codes)
	must(err)

	return program, must
}

func PrepareProgram(code string, location common.Location, codes map[common.LocationID]string) (*ast.Program, func(error)) {
	must := mustClosure(location, codes)

	program, err := parser2.ParseProgram(code, nil)
	codes[location.ID()] = code
	must(err)

	return program, must
}

var checkers = map[common.LocationID]*sema.Checker{}

func DefaultCheckerInterpreterOptions(
	checkers map[common.LocationID]*sema.Checker,
	codes map[common.LocationID]string,
	impls stdlib.FlowBuiltinImpls,
) (
	[]sema.Option,
	[]interpreter.Option,
) {

	semaPredeclaredValues, interpreterPredeclaredValues :=
		stdlib.FlowDefaultPredeclaredValues(impls)

	return []sema.Option{
			sema.WithPredeclaredValues(semaPredeclaredValues),
			sema.WithPredeclaredTypes(stdlib.FlowDefaultPredeclaredTypes),
			sema.WithImportHandler(
				func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					if importedLocation == stdlib.CryptoChecker.Location {
						return sema.ElaborationImport{
							Elaboration: stdlib.CryptoChecker.Elaboration,
						}, nil
					}

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

					importedChecker, ok := checkers[importedLocation.ID()]
					if !ok {
						importedProgram, _ := PrepareProgramFromFile(stringLocation, codes)
						importedChecker, _ = checker.SubChecker(importedProgram, importedLocation)
						checkers[importedLocation.ID()] = importedChecker
					}

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			),
		},
		[]interpreter.Option{
			interpreter.WithPredeclaredValues(interpreterPredeclaredValues),
		}
}

// PrepareChecker prepares and initializes a checker with a given code as a string,
// and a filename which is used for pretty-printing errors, if any
func PrepareChecker(
	program *ast.Program,
	location common.Location,
	codes map[common.LocationID]string,
	memberAccountAccess map[common.LocationID]map[common.LocationID]struct{},
	must func(error),
) (*sema.Checker, func(error)) {

	defaultCheckerOptions, _ :=
		DefaultCheckerInterpreterOptions(
			checkers,
			codes,
			stdlib.FlowBuiltinImpls{},
		)

	defaultCheckerOptions = append(
		defaultCheckerOptions,
		sema.WithMemberAccountAccessHandler(func(checker *sema.Checker, memberLocation common.Location) bool {
			if memberAccountAccess == nil {
				return false
			}

			targets, ok := memberAccountAccess[checker.Location.ID()]
			if !ok {
				return false
			}

			_, ok = targets[memberLocation.ID()]
			return ok
		}),
	)

	checker, err := sema.NewChecker(
		program,
		location,
		nil,
		defaultCheckerOptions...,
	)
	must(err)

	return checker, must
}

func PrepareInterpreter(filename string, debugger *interpreter.Debugger) (*interpreter.Interpreter, *sema.Checker, func(error)) {

	codes := map[common.LocationID]string{}

	// do not need to meter this as it's a one-off overhead
	location := common.NewStringLocation(nil, filename)

	program, must := PrepareProgramFromFile(location, codes)

	checker, must := PrepareChecker(program, location, codes, nil, must)

	must(checker.Check())

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	_, defaultInterpreterOptions :=
		DefaultCheckerInterpreterOptions(
			checkers,
			codes,
			stdlib.DefaultFlowBuiltinImpls(),
		)

	defaultInterpreterOptions = append(
		defaultInterpreterOptions,
		interpreter.WithStorage(storage),
		interpreter.WithUUIDHandler(func() (uint64, error) {
			defer func() { uuid++ }()
			return uuid, nil
		}),
		interpreter.WithDebugger(debugger),
	)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		defaultInterpreterOptions...,
	)
	must(err)

	must(inter.Interpret())

	return inter, checker, must
}

func ExitWithError(message string) {
	fmt.Println(pretty.FormatErrorMessage(message, true))
	os.Exit(1)
}
