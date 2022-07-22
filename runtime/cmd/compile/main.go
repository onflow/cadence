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

package main

import (
	"os"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/cmd"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/compiler"
	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/compiler/wasm"
)

func main() {
	args := os.Args

	if len(args) < 2 {
		cmd.ExitWithError("no input file")
	}

	path := args[1]

	location := common.NewStringLocation(nil, path)

	codes := map[common.Location]string{}

	program, must := cmd.PrepareProgramFromFile(location, codes)

	checker, must := cmd.PrepareChecker(program, location, codes, nil, must)

	must(checker.Check())

	// Compile all functions

	comp := compiler.NewCompiler(checker)

	functionDeclarations := checker.Program.FunctionDeclarations()

	funcs := make([]*ir.Func, len(functionDeclarations))

	for i, functionDeclaration := range functionDeclarations {
		funcs[i] = functionDeclaration.Accept(comp).(*ir.Func)
	}

	// Generate a WebAssembly module for the functions

	module := compiler.GenerateWasm(funcs)

	// Export all public functions

	for i, functionDeclaration := range functionDeclarations {
		if functionDeclaration.Access != ast.AccessPublic {
			continue
		}

		module.Exports = append(module.Exports,
			&wasm.Export{
				Name: functionDeclaration.Identifier.Identifier,
				Descriptor: wasm.FunctionExport{
					FunctionIndex: uint32(i),
				},
			},
		)
	}

	// Generate WASM binary

	var buf wasm.Buffer
	w := wasm.NewWASMWriter(&buf)
	err := w.WriteModule(module)
	if err != nil {
		panic(nil)
	}

	// Write WASM binary to stdout

	_, err = os.Stdout.Write(buf.Bytes())
	if err != nil {
		panic(nil)
	}
}
