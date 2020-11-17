/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"io"
	"io/ioutil"
	"os"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func PrettyPrintError(writer io.Writer, err error, filename string, codes map[string]string) {
	i := 0
	printErr := func(err error, filename string) {
		if i > 0 {
			println()
		}
		_, writeErr := writer.Write([]byte(runtime.PrettyPrintError(err, filename, codes[filename], true)))
		if writeErr != nil {
			panic(writeErr)
		}
		i++
	}

	switch typedErr := err.(type) {
	case parser2.Error:
		for _, err := range typedErr.Errors {
			printErr(err, filename)
		}
	case *sema.CheckerError:
		for _, err := range typedErr.Errors {
			printErr(err, filename)
			if err, ok := err.(*sema.ImportedProgramError); ok {
				filename := importLocationFileName(err.ImportLocation)

				for _, nestedErr := range err.CheckerError.Errors {
					PrettyPrintError(writer, nestedErr, filename, codes)
				}
			}
		}
	case ast.HasImportLocation:
		location := typedErr.ImportLocation()
		if location != nil {
			filename = importLocationFileName(location)
		}
		printErr(err, filename)
	default:
		printErr(err, filename)
	}
}

func importLocationFileName(importLocation ast.Location) string {
	switch importLocation := importLocation.(type) {
	case ast.StringLocation:
		return string(importLocation)
	case ast.AddressLocation:
		return importLocation.Address.ShortHexWithPrefix()
	case ast.IdentifierLocation:
		return string(importLocation)
	case runtime.FileLocation:
		return string(importLocation)
	}
	return ""
}

func must(err error, filename string, codes map[string]string) {
	if err == nil {
		return
	}
	PrettyPrintError(os.Stderr, err, filename, codes)
	os.Exit(1)
}

func mustClosure(filename string, codes map[string]string) func(error) {
	return func(e error) {
		must(e, filename, codes)
	}
}

func PrepareProgramFromFile(filename string, codes map[string]string) (*ast.Program, func(error)) {
	codeBytes, err := ioutil.ReadFile(filename)

	program, must := PrepareProgram(string(codeBytes), filename, codes)
	must(err)

	return program, must
}

func PrepareProgram(code string, filename string, codes map[string]string) (*ast.Program, func(error)) {
	must := mustClosure(filename, codes)

	program, err := parser2.ParseProgram(code)
	codes[filename] = code
	must(err)

	return program, must
}

var valueDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.DefaultFlowBuiltinImpls()),
	stdlib.BuiltinFunctions...,
)

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

// PrepareChecker prepares and initializes a checker with a given code as a string,
// and a filename which is used for pretty-printing errors, if any
func PrepareChecker(program *ast.Program, filename string, codes map[string]string, must func(error)) (*sema.Checker, func(error)) {
	location := runtime.FileLocation(filename)
	checker, err := sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(valueDeclarations.ToValueDeclarations()),
		sema.WithPredeclaredTypes(typeDeclarations),
		sema.WithImportHandler(
			func(checker *sema.Checker, location ast.Location) (sema.Import, *sema.CheckerError) {
				stringLocation, ok := location.(ast.StringLocation)

				if !ok {
					return nil, &sema.CheckerError{
						Errors: []error{
							fmt.Errorf("cannot import `%s`. only files are supported", location),
						},
					}
				}

				importChecker, err := checker.EnsureLoaded(
					location,
					func() *ast.Program {
						filename := string(stringLocation)
						imported, _ := PrepareProgramFromFile(filename, codes)
						return imported
					},
				)
				if err != nil {
					return nil, err
				}

				return sema.CheckerImport{
					Checker: importChecker,
				}, nil
			},
		),
	)
	must(err)

	return checker, must
}

func PrepareInterpreter(filename string) (*interpreter.Interpreter, *sema.Checker, func(error)) {

	codes := map[string]string{}

	program, must := PrepareProgramFromFile(filename, codes)

	checker, must := PrepareChecker(program, filename, codes, must)

	must(checker.Check())

	var uuid uint64

	inter, err := interpreter.NewInterpreter(
		checker,
		interpreter.WithPredefinedValues(valueDeclarations.ToValues()),
		interpreter.WithUUIDHandler(func() uint64 {
			defer func() { uuid++ }()
			return uuid
		}),
	)
	must(err)

	must(inter.Interpret())

	return inter, checker, must
}

func ExitWithError(message string) {
	print(runtime.FormatErrorMessage(message, true))
	os.Exit(1)
}
