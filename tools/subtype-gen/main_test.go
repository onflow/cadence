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

package subtype_gen

import (
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"
	"os"
	"testing"
)

func TestGen(t *testing.T) {
	// Read and parse YAML rules
	rules, err := ParseRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	const pkgPath = "github.com/onflow/cadence/sema"

	// Generate code using the comprehensive generator
	gen := NewSubTypeCheckGenerator(pkgPath)
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(pkgPath, resolver)

	packageName, err := resolver.ResolvePackage(pkgPath)
	if err != nil {
		panic(err)
	}

	err = restorer.Fprint(
		os.Stdout,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
}
