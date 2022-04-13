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
	"os"
	"path"
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/tools/docgen"
)

// This is a convenient method to generate the doc files given a cadence file.
// Can be used to generate the assert files needed for doc tests.
func TestDocGen(t *testing.T) {

	// Don't run this as a test
	t.Skip()

	content, err := ioutil.ReadFile(path.Join("samples", "sample2.cdc"))
	require.NoError(t, err)

	err = os.MkdirAll("outputs", os.ModePerm)
	require.NoError(t, err)

	docGen := docgen.NewDocGenerator()

	err = docGen.Generate(string(content), "outputs")
	require.NoError(t, err)
}

func TestDocGenForMultiDeclarationFile(t *testing.T) {
	content, err := ioutil.ReadFile(path.Join("samples", "sample1.cdc"))
	require.NoError(t, err)

	docGen := docgen.NewDocGenerator()

	docFiles, err := docGen.GenerateInMemory(string(content))
	require.NoError(t, err)

	require.Len(t, docFiles, 6)

	for fileName, fileContent := range docFiles {
		expectedContent, err := ioutil.ReadFile(path.Join("outputs", fileName))
		require.NoError(t, err)
		assert.Equal(t, string(expectedContent), string(fileContent))
	}
}

func TestDocGenForSingleContractFile(t *testing.T) {

	content, err := ioutil.ReadFile(path.Join("samples", "sample2.cdc"))
	require.NoError(t, err)

	docGen := docgen.NewDocGenerator()

	docFiles, err := docGen.GenerateInMemory(string(content))
	require.NoError(t, err)

	require.Len(t, docFiles, 5)

	for fileName, fileContent := range docFiles {
		expectedContent, err := ioutil.ReadFile(path.Join("outputs", fileName))
		require.NoError(t, err)
		assert.Equal(t, string(expectedContent), string(fileContent))
	}
}

func TestDocGenErrors(t *testing.T) {

	t.Parallel()

	t.Run("syntax error", func(t *testing.T) {
		docGen := docgen.NewDocGenerator()

		code := `
            fun foo() {
	    `
		_, err := docGen.GenerateInMemory(code)

		require.Error(t, err)
		assert.IsType(t, err, parser2.Error{})
	})

	t.Run("invalid output path", func(t *testing.T) {
		docGen := docgen.NewDocGenerator()

		code := `
            fun foo() {
            }
        `
		err := docGen.Generate(code, "non-existing-dir")

		require.Error(t, err)
		assert.IsType(t, err, &os.PathError{})
	})
}

func TestFunctionDocFormatting(t *testing.T) {

	content, err := ioutil.ReadFile(path.Join("samples", "sample3.cdc"))
	require.NoError(t, err)

	docGen := docgen.NewDocGenerator()

	docFiles, err := docGen.GenerateInMemory(string(content))
	require.NoError(t, err)
	require.Len(t, docFiles, 1)

	expectedContent, err := ioutil.ReadFile(path.Join("outputs", "sample3_output.md"))

	assert.Equal(t, string(expectedContent), string(docFiles["index.md"]))
}
