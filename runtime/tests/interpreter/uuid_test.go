package interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/cmd"
	"github.com/dapperlabs/cadence/runtime/interpreter"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestInterpretResourceUUID(t *testing.T) {

	checkerImported, err := ParseAndCheck(t, `
      pub resource R {}

      pub fun createR(): @R {
          return <- create R()
      }
    `)
	require.NoError(t, err)

	checkerImporting, err := ParseAndCheckWithOptions(t,
		`
          import createR from "imported"

          pub resource R2 {}

          pub fun createRs(): @[AnyResource] {
              return <- [
                  <- (createR() as @AnyResource),
                  <- create R2()
              ]
          }
        `,
		ParseAndCheckOptions{
			ImportResolver: func(location ast.Location) (program *ast.Program, e error) {
				assert.Equal(t,
					ImportedLocation,
					location,
				)
				return checkerImported.Program, nil
			},
		},
	)

	if err != nil {
		cmd.PrettyPrintError(err, "", map[string]string{"": ""})
	}

	require.NoError(t, err)

	var uuid uint64

	inter, err := interpreter.NewInterpreter(checkerImporting,
		interpreter.WithUUIDHandler(func() uint64 {
			defer func() { uuid++ }()
			return uuid
		}),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("createRs")
	require.NoError(t, err)

	require.IsType(t, &interpreter.ArrayValue{}, value)

	array := value.(*interpreter.ArrayValue)

	const length = 2
	require.Len(t, array.Values, length)

	for i := 0; i < length; i++ {
		element := array.Values[i]

		require.IsType(t, &interpreter.CompositeValue{}, element)
		res := element.(*interpreter.CompositeValue)

		require.Equal(t,
			interpreter.UInt64Value(i),
			res.Fields[interpreter.ResourceUUIDMemberName],
		)
	}
}
