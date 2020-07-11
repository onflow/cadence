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

	if parserError, ok := err.(parser2.Error); ok {
		for _, err := range parserError.Errors {
			printErr(err, filename)
		}
	} else if checkerError, ok := err.(*sema.CheckerError); ok {
		for _, err := range checkerError.Errors {
			printErr(err, filename)
			if err, ok := err.(*sema.ImportedProgramError); ok {
				filename := string(err.ImportLocation.(ast.StringLocation))
				for _, err := range err.CheckerError.Errors {
					PrettyPrintError(writer, err, filename, codes)
				}
			}
		}
	} else if locatedErr, ok := err.(ast.HasImportLocation); ok {
		location := locatedErr.ImportLocation()
		if location != nil {
			filename = string(location.(ast.StringLocation))
		}
		printErr(err, filename)
	} else {
		printErr(err, filename)
	}
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

func PrepareProgramFromFile(filename string) (*ast.Program, func(error)) {
	codeBytes, err := ioutil.ReadFile(filename)

	program, _, must := PrepareProgram(string(codeBytes), filename)
	must(err)

	return program, must
}

func PrepareProgram(code string, filename string) (*ast.Program, map[string]string, func(error)) {
	codes := map[string]string{}

	must := mustClosure(filename, codes)

	program, err := parser2.ParseProgram(code)
	codes[filename] = code
	must(err)

	err = program.ResolveImports(func(location ast.Location) (program *ast.Program, err error) {
		switch location := location.(type) {
		case ast.StringLocation:
			filename := string(location)
			imported, code, err := parser2.ParseProgramFromFile(filename)
			codes[filename] = code
			must(err)
			return imported, nil

		default:
			return nil, fmt.Errorf("cannot import `%s`. only files are supported", location)
		}
	})
	must(err)
	return program, codes, must
}

var valueDeclarations = append(
	stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{}),
	stdlib.BuiltinFunctions...,
)

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

// PrepareChecker prepares and initializes a checker with a given code as a string,
// and a filename which is used for pretty-printing errors, if any
func PrepareChecker(program *ast.Program, filename string, must func(error)) (*sema.Checker, func(error)) {
	location := runtime.FileLocation(filename)
	checker, err := sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(valueDeclarations.ToValueDeclarations()),
		sema.WithPredeclaredTypes(typeDeclarations),
	)
	must(err)

	return checker, must
}

func PrepareInterpreter(filename string) (*interpreter.Interpreter, *sema.Checker, func(error)) {

	program, must := PrepareProgramFromFile(filename)

	checker, must := PrepareChecker(program, filename, must)

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
