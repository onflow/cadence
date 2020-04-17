package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretWhileStatement(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var x = 0
           while x < 5 {
               x = x + 2
           }
           return x
       }

    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(6),
		value,
	)
}

func TestInterpretWhileStatementWithReturn(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var x = 0
           while x < 10 {
               x = x + 2
               if x > 5 {
                   return x
               }
           }
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(6),
		value,
	)
}

func TestInterpretWhileStatementWithContinue(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var i = 0
           var x = 0
           while i < 10 {
               i = i + 1
               if i < 5 {
                   continue
               }
               x = x + 1
           }
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(6),
		value,
	)
}

func TestInterpretWhileStatementWithBreak(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var x = 0
           while x < 10 {
               x = x + 1
               if x == 5 {
                   break
               }
           }
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(5),
		value,
	)
}
