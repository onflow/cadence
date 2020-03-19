package stdlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/interpreter"
	"github.com/dapperlabs/cadence/runtime/sema"
	"github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestAssert(t *testing.T) {

	program := &ast.Program{}

	checker, err := sema.NewChecker(
		program,
		utils.TestLocation,
		sema.WithPredeclaredValues(BuiltinFunctions.ToValueDeclarations()),
	)
	require.Nil(t, err)

	inter, err := interpreter.NewInterpreter(
		checker,
		interpreter.WithPredefinedValues(BuiltinFunctions.ToValues()),
	)
	require.Nil(t, err)

	_, err = inter.Invoke("assert", false, "oops")
	assert.Equal(t,
		AssertionError{
			Message:       "oops",
			LocationRange: interpreter.LocationRange{},
		},
		err,
	)

	_, err = inter.Invoke("assert", false)
	assert.Equal(t,
		AssertionError{
			Message:       "",
			LocationRange: interpreter.LocationRange{},
		},
		err)

	_, err = inter.Invoke("assert", true, "oops")
	assert.NoError(t, err)

	_, err = inter.Invoke("assert", true)
	assert.NoError(t, err)
}

func TestPanic(t *testing.T) {

	checker, err := sema.NewChecker(
		&ast.Program{},
		utils.TestLocation,
		sema.WithPredeclaredValues(BuiltinFunctions.ToValueDeclarations()),
	)
	require.Nil(t, err)

	inter, err := interpreter.NewInterpreter(
		checker,
		interpreter.WithPredefinedValues(BuiltinFunctions.ToValues()),
	)
	require.Nil(t, err)

	_, err = inter.Invoke("panic", "oops")
	assert.Equal(t,
		PanicError{
			Message:       "oops",
			LocationRange: interpreter.LocationRange{},
		},
		err)
}
