package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretForStatement(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var sum = 0
           for y in [1, 2, 3, 4] {
               sum = sum + y
           }
           return sum
       }

    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(10),
		value,
	)
}

func TestInterpretForStatementWithReturn(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           for x in [1, 2, 3, 4, 5] {
               if x > 3 {
                   return x
               }
           }
           return -1
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(4),
		value,
	)
}

func TestInterpretForStatementWithContinue(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): [Int] {
           var xs: [Int] = []
           for x in [1, 2, 3, 4, 5] {
               if x <= 3 {
                   continue
               }
               xs.append(x)
           }
           return xs
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValue(4),
			interpreter.NewIntValue(5),
		),
		value,
	)
}

func TestInterpretForStatementWithBreak(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var y = 0
           for x in [1, 2, 3, 4] {
               y = x
               if x > 3 {
                   break
               }
           }
           return y
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValue(4),
		value,
	)
}

func TestInterpretForStatementEmpty(t *testing.T) {

	inter := parseCheckAndInterpret(t, `
       fun test(): Bool {
           var x = false
           for y in [] {
               x = true
           }
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(false),
		value,
	)
}
