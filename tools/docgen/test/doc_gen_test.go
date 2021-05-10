package test

import (
	"os"
	"testing"

	"io/ioutil"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/tools/docgen/gen"
)

func TestDocGen(t *testing.T) {
	content, err := ioutil.ReadFile("samples/sample3.cdc")
	require.NoError(t, err)

	err = os.RemoveAll("generated")
	require.NoError(t, err)

	err = os.MkdirAll("generated", os.ModePerm)
	require.NoError(t, err)

	docGen := gen.NewDocGenerator()
	err = docGen.Generate(string(content), "generated")
	require.NoError(t, err)
}
