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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/dst"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsingRules(t *testing.T) {
	t.Parallel()

	rules, err := ParseRules()
	require.NoError(t, err)
	assert.Len(t, rules.Rules, 26)
}

func TestGeneratedCodeStructure(t *testing.T) {
	t.Parallel()

	rules, err := ParseRules()
	require.NoError(t, err)

	gen := NewSubTypeCheckGenerator(Config{})
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	require.Len(t, decls, 1)
	decl := decls[0]

	require.IsType(t, &dst.FuncDecl{}, decl)
	funcDecl := decl.(*dst.FuncDecl)

	// Assert function name
	assert.Equal(t, subtypeCheckFuncName, funcDecl.Name.Name)

	// Assert function body
	statements := funcDecl.Body.List
	require.Len(t, statements, 4)

	// If check for never type
	require.IsType(t, &dst.IfStmt{}, statements[0])
	// Switch statement for simple types
	require.IsType(t, &dst.SwitchStmt{}, statements[1])
	// Type-switch for complex types
	require.IsType(t, &dst.TypeSwitchStmt{}, statements[2])
	// The final return
	require.IsType(t, &dst.ReturnStmt{}, statements[3])
}

// Go treats directories named "testdata" specially
const testDataDirectory = "testdata"

// TestCodeGeneration finds all `.yaml` files in the `testdata` directory.
// Each file turns into a test case.
// Each input file is expected to have a "golden output" file,
// with the same path, except the `.yaml` extension is replaced by `.golden.go`.
func TestCodeGeneration(t *testing.T) {

	t.Parallel()

	test := func(inputPath string) {
		// The test name is the file name without the extension
		_, testName := filepath.Split(inputPath)
		fileExt := filepath.Ext(testName)
		testName = strings.TrimSuffix(testName, fileExt)

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			// Read input yaml file.
			yaml, err := os.ReadFile(inputPath)
			require.NoError(t, err)

			// Parse the yaml.
			rules, err := ParseRulesFromBytes(yaml)
			require.NoError(t, err)

			// Generate code.
			config := Config{
				SimpleTypeSuffix:  "Type",
				ComplexTypeSuffix: "Type",

				ArrayElementTypeMethodArgs: []any{
					false,
				},

				NonPointerTypes: map[string]struct{}{
					TypePlaceholderParameterized: {},
					TypePlaceholderConforming:    {},
				},

				NameMapping: map[string]string{
					FieldNameReferencedType: "Type",
				},
			}
			gen := NewSubTypeCheckGenerator(config)
			decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

			// Write output.
			outFile, err := os.CreateTemp(t.TempDir(), "gen.*.go")
			require.NoError(t, err)
			defer outFile.Close()
			WriteGoFile(outFile, decls, "github.com/onflow/cadence/sema")

			// Read the expected output file.
			goldenPath := strings.ReplaceAll(inputPath, fileExt, ".golden.go")
			expected, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			_, err = outFile.Seek(0, io.SeekStart)
			require.NoError(t, err)

			actual, err := io.ReadAll(outFile)
			require.NoError(t, err)

			// Compare
			require.Equal(t, string(expected), string(actual))
		})
	}

	pathPattern := filepath.Join(testDataDirectory, "*.yaml")
	paths, err := filepath.Glob(pathPattern)
	require.NoError(t, err)

	for _, path := range paths {
		test(path)
	}
}
