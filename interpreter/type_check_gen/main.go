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
	"text/template"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"

	subtypegen "github.com/onflow/cadence/tools/subtype-gen"
)

const headerTemplate = `// Code generated from {{ . }}. DO NOT EDIT.
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

`

var parsedHeaderTemplate = template.Must(template.New("header").Parse(headerTemplate))

const interpreterPath = "github.com/onflow/cadence/interpreter"

var packagePathFlag = flag.String("pkg", interpreterPath, "target Go package name")

func main() {

	flag.Parse()
	argumentCount := flag.NArg()
	if argumentCount < 1 {
		panic("Missing path to input Cadence file")
	}

	outPath := flag.Arg(0)

	// Read and parse YAML rules
	rules, err := subtypegen.ParseRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	const (
		interpreterPath        = "github.com/onflow/cadence/interpreter"
		typeConverterParamName = "typeConverter"
		typeConverterTypeName  = "TypeConverter"
	)

	config := subtypegen.Config{
		SimpleTypePrefix:  "PrimitiveStaticType",
		ComplexTypeSuffix: "StaticType",
		ExtraParams: []subtypegen.ExtraParam{
			{
				Name:    typeConverterParamName,
				Type:    typeConverterTypeName,
				PkgPath: interpreterPath,
			},
		},
		SkipTypes: map[string]struct{}{
			subtypegen.TypePlaceholderStorable: {},
		},
	}

	// Generate code using the comprehensive generator
	gen := subtypegen.NewSubTypeCheckGenerator(config)
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	writeGoFile(outFile, decls, *packagePathFlag)
}

func writeGoFile(outFile *os.File, decls []dst.Decl, packagePath string) {
	err := parsedHeaderTemplate.Execute(outFile, nil)
	if err != nil {
		panic(err)
	}

	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(packagePath, resolver)

	packageName, err := resolver.ResolvePackage(packagePath)
	if err != nil {
		panic(err)
	}

	err = restorer.Fprint(
		outFile,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
	if err != nil {
		panic(err)
	}
}
