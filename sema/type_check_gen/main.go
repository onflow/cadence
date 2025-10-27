/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"flag"
	"fmt"
	"os"

	subtypegen "github.com/onflow/cadence/tools/subtype-gen"
)

const semaPath = "github.com/onflow/cadence/sema"

var packagePathFlag = flag.String("pkg", semaPath, "target Go package name")

func main() {

	flag.Parse()
	argumentCount := flag.NArg()
	if argumentCount < 1 {
		panic("Missing path to output Go file")
	}

	outPath := flag.Arg(0)

	// Read and parse YAML rules
	rules, err := subtypegen.ParseRules()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	config := subtypegen.Config{
		SimpleTypeSuffix:  "Type",
		ComplexTypeSuffix: "Type",

		ArrayElementTypeMethodArgs: []any{
			false,
		},

		NonPointerTypes: map[string]struct{}{
			subtypegen.TypePlaceholderParameterized: {},
			subtypegen.TypePlaceholderConforming:    {},
		},

		NameMapping: map[string]string{
			subtypegen.FieldNameReferencedType: "Type",
		},
	}

	// Generate code using the comprehensive generator
	gen := subtypegen.NewSubTypeCheckGenerator(config)
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	// Write output
	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	subtypegen.WriteGoFile(outFile, decls, *packagePathFlag)
}
