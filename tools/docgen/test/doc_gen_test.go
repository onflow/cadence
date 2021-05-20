package test

import (
	"os"
	"path"
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/tools/docgen/gen"
)

// This is a convenient method to generate the doc files given a cadence file.
// Can be used to generate the assert files needed for doc tests.
func TestDocGen(t *testing.T) {

	// Don't run this as a test
	t.Skip()

	content, err := ioutil.ReadFile("samples/sample2.cdc")
	require.NoError(t, err)

	err = os.MkdirAll("outputs", os.ModePerm)
	require.NoError(t, err)

	docGen := gen.NewDocGenerator()

	err = docGen.Generate(string(content), "outputs")
	require.NoError(t, err)
}

func TestDocGenForMultiDeclarationFile(t *testing.T) {
	content, err := ioutil.ReadFile("samples/sample1.cdc")
	require.NoError(t, err)

	docGen := gen.NewDocGenerator()

	docFiles, err := docGen.GenerateInMemory(string(content))
	require.NoError(t, err)

	require.Len(t, docFiles, 5)

	for fileName, fileContent := range docFiles {
		expectedContent, err := ioutil.ReadFile(path.Join("outputs", fileName))
		require.NoError(t, err)
		assert.Equal(t, expectedContent, fileContent)
	}
}

func TestDocGenForSingleContractFile(t *testing.T) {

	content, err := ioutil.ReadFile("samples/sample2.cdc")
	require.NoError(t, err)

	docGen := gen.NewDocGenerator()

	docFiles, err := docGen.GenerateInMemory(string(content))
	require.NoError(t, err)

	require.Len(t, docFiles, 5)

	for fileName, fileContent := range docFiles {
		expectedContent, err := ioutil.ReadFile(path.Join("outputs", fileName))
		require.NoError(t, err)
		assert.Equal(t, expectedContent, fileContent)
	}
}

func TestDocGenErrors(t *testing.T) {

	t.Parallel()

	t.Run("syntax error", func(t *testing.T) {
		docGen := gen.NewDocGenerator()

		code := `
            fun foo() {
	    `
		_, err := docGen.GenerateInMemory(code)

		require.Error(t, err)
		assert.IsType(t, err, parser2.Error{})
	})

	t.Run("invalid output path", func(t *testing.T) {
		docGen := gen.NewDocGenerator()

		code := `
            fun foo() {
            }
        `
		err := docGen.Generate(code, "non-existing-dir")

		require.Error(t, err)
		assert.IsType(t, err, &os.PathError{})
	})
}
