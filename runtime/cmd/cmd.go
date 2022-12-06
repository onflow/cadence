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
	"os"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func must(err error, location common.Location, codes map[common.Location][]byte) {
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

func mustClosure(location common.Location, codes map[common.Location][]byte) func(error) {
	return func(e error) {
		must(e, location, codes)
	}
}

func PrepareProgramFromFile(location common.StringLocation, codes map[common.Location][]byte) (*ast.Program, func(error)) {
	code, err := os.ReadFile(string(location))

	program, must := PrepareProgram(code, location, codes)
	must(err)

	return program, must
}

func PrepareProgram(code []byte, location common.Location, codes map[common.Location][]byte) (*ast.Program, func(error)) {
	must := mustClosure(location, codes)

	program, err := parser.ParseProgram(nil, code, parser.Config{})
	codes[location] = code
	must(err)

	return program, must
}

var checkers = map[common.Location]*sema.Checker{}

type StandardOutputLogger struct{}

func (s StandardOutputLogger) ProgramLog(message string) error {
	fmt.Println(message)
	return nil
}

var _ stdlib.Logger = StandardOutputLogger{}

func DefaultCheckerConfig(
	checkers map[common.Location]*sema.Checker,
	codes map[common.Location][]byte,
) *sema.Config {
	// NOTE: declarations here only create a nil binding in the checker environment,
	// not a definition that the interpreter can follow. (see #2106 and #2109)
	// remember to also implement all definitions, e.g. for the `REPL` in `NewREPL`
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewLogFunction(StandardOutputLogger{}))

	return &sema.Config{
		BaseValueActivation: baseValueActivation,
		AccessCheckMode:     sema.AccessCheckModeStrict,
		ImportHandler: func(
			checker *sema.Checker,
			importedLocation common.Location,
			_ ast.Range,
		) (sema.Import, error) {
			cryptoChecker := stdlib.CryptoChecker()
			if importedLocation == cryptoChecker.Location {
				return sema.ElaborationImport{
					Elaboration: cryptoChecker.Elaboration,
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

			importedChecker, ok := checkers[importedLocation]
			if !ok {
				importedProgram, _ := PrepareProgramFromFile(stringLocation, codes)
				importedChecker, _ = checker.SubChecker(importedProgram, importedLocation)
				checkers[importedLocation] = importedChecker
			}

			return sema.ElaborationImport{
				Elaboration: importedChecker.Elaboration,
			}, nil
		},
	}
}

// PrepareChecker prepares and initializes a checker with a given code as a string,
// and a filename which is used for pretty-printing errors, if any
func PrepareChecker(
	program *ast.Program,
	location common.Location,
	codes map[common.Location][]byte,
	memberAccountAccess map[common.Location]map[common.Location]struct{},
	must func(error),
) (*sema.Checker, func(error)) {

	config := DefaultCheckerConfig(checkers, codes)

	config.MemberAccountAccessHandler = func(checker *sema.Checker, memberLocation common.Location) bool {
		if memberAccountAccess == nil {
			return false
		}

		targets, ok := memberAccountAccess[checker.Location]
		if !ok {
			return false
		}

		_, ok = targets[memberLocation]
		return ok
	}

	checker, err := sema.NewChecker(
		program,
		location,
		nil,
		config,
	)
	must(err)

	return checker, must
}

func PrepareInterpreter(filename string, debugger *interpreter.Debugger) (*interpreter.Interpreter, *sema.Checker, func(error)) {

	codes := map[common.Location][]byte{}

	// do not need to meter this as it's a one-off overhead
	location := common.NewStringLocation(nil, filename)

	program, must := PrepareProgramFromFile(location, codes)

	checker, must := PrepareChecker(program, location, codes, nil, must)

	must(checker.Check())

	var uuid uint64

	storage := interpreter.NewInMemoryStorage(nil)

	// NOTE: storage option must be provided *before* the predeclared values option,
	// as predeclared values may rely on storage

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.NewLogFunction(StandardOutputLogger{}))

	config := &interpreter.Config{
		BaseActivation: baseActivation,
		Storage:        storage,
		UUIDHandler: func() (uint64, error) {
			defer func() { uuid++ }()
			return uuid, nil
		},
		Debugger: debugger,
		ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			panic("Importing programs is not supported yet")
		},
	}

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		config,
	)
	must(err)

	must(inter.Interpret())

	return inter, checker, must
}

func ExitWithError(message string) {
	println(pretty.FormatErrorMessage(pretty.ErrorPrefix, message, true))
	os.Exit(1)
}
