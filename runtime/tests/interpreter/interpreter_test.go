/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package interpreter_test

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/examples"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type ParseCheckAndInterpretOptions struct {
	Options            []interpreter.Option
	CheckerOptions     []sema.Option
	HandleCheckerError func(error)
}

func parseCheckAndInterpret(t testing.TB, code string) *interpreter.Interpreter {
	return parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{})
}

func parseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) *interpreter.Interpreter {

	checker, err := checker.ParseAndCheckWithOptions(t,
		code,
		checker.ParseAndCheckOptions{
			Options: options.CheckerOptions,
		},
	)

	if options.HandleCheckerError != nil {
		options.HandleCheckerError(err)
	} else if !assert.NoError(t, err) {
		var sb strings.Builder
		locationID := checker.Location.ID()
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, checker.Location, map[common.LocationID]string{locationID: code})
		if printErr != nil {
			panic(printErr)
		}
		assert.FailNow(t, sb.String())
		return nil
	}

	var uuid uint64 = 0

	interpreterOptions := append(
		[]interpreter.Option{
			interpreter.WithUUIDHandler(func() (uint64, error) {
				uuid++
				return uuid, nil
			}),
		},
		options.Options...,
	)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreterOptions...,
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter
}

func constructorArguments(compositeKind common.CompositeKind, arguments string) string {
	switch compositeKind {
	case common.CompositeKindContract:
		return ""
	case common.CompositeKindEnum:
		return ".a"
	default:
		return fmt.Sprintf("(%s)", arguments)
	}
}

// makeContractValueHandler creates an interpreter option which
// sets the ContractValueHandler.
// The handler immediately invokes the constructor with the given arguments.
//
func makeContractValueHandler(
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
) interpreter.Option {
	return interpreter.WithContractValueHandler(
		func(
			inter *interpreter.Interpreter,
			compositeType *sema.CompositeType,
			constructor interpreter.FunctionValue,
			invocationRange ast.Range,
		) *interpreter.CompositeValue {
			value, err := inter.InvokeFunctionValue(
				constructor,
				arguments,
				argumentTypes,
				parameterTypes,
				ast.Range{},
			)
			if err != nil {
				panic(err)
			}

			return value.(*interpreter.CompositeValue)
		},
	)
}

func TestInterpretConstantAndVariableDeclarations(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        let x = 1
        let y = true
        let z = 1 + 2
        var a = 3 == 3
        var b = [1, 2]
        let s = "123"
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		inter.Globals["z"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
		),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewStringValue("123"),
		inter.Globals["s"].GetValue(),
	)
}

func TestInterpretDeclarations(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun test(): Int {
            return 42
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		value,
	)
}

func TestInterpretInvalidUnknownDeclarationInvocation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, ``)

	_, err := inter.Invoke("test")
	assert.IsType(t, interpreter.NotDeclaredError{}, err)
}

func TestInterpretInvalidNonFunctionDeclarationInvocation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let test = 1
   `)

	_, err := inter.Invoke("test")
	assert.IsType(t, interpreter.NotInvokableError{}, err)
}

func TestInterpretLexicalScope(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 10

       fun f(): Int {
          // check resolution
          return x
       }

       fun g(): Int {
          // check scope is lexical, not dynamic
          let x = 20
          return f()
       }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(10),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("f")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(10),
		value,
	)

	value, err = inter.Invoke("g")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(10),
		value,
	)
}

func TestInterpretFunctionSideEffects(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var value = 0

       fun test(_ newValue: Int) {
           value = newValue
       }
    `)

	newValue := interpreter.NewIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	assert.Equal(t,
		newValue,
		inter.Globals["value"].GetValue(),
	)
}

func TestInterpretNoHoisting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 2

       fun test(): Int {
          if x == 0 {
              let x = 3
              return x
          }
          return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretFunctionExpressionsAndScope(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 10

       // check first-class functions and scope inside them
       let y = (fun (x: Int): Int { return x })(42)
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(10),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretVariableAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var x = 2
           x = 3
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}

func TestInterpretGlobalVariableAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var x = 2

       fun test(): Int {
           x = 3
           return x
       }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretConstantRedeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x = 2

       fun test(): Int {
           let x = 3
           return x
       }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}

func TestInterpretParameters(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun returnA(a: Int, b: Int): Int {
           return a
       }

       fun returnB(a: Int, b: Int): Int {
           return b
       }
    `)

	a := interpreter.NewIntValueFromInt64(24)
	b := interpreter.NewIntValueFromInt64(42)

	value, err := inter.Invoke("returnA", a, b)
	require.NoError(t, err)

	assert.Equal(t, a, value)

	value, err = inter.Invoke("returnB", a, b)
	require.NoError(t, err)

	assert.Equal(t, b, value)
}

func TestInterpretArrayIndexing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           let z = [0, 3]
           return z[1]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}

func TestInterpretInvalidArrayIndexing(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int): Int {
                   let z = [0, 3]
                   return z[index]
               }
            `)

			indexValue := interpreter.NewIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			require.Equal(t,
				interpreter.ArrayIndexOutOfBoundsError{
					Index: index,
					Size:  2,
					LocationRange: interpreter.LocationRange{
						Location: TestLocation,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 107, Line: 4, Column: 27},
							EndPos:   ast.Position{Offset: 113, Line: 4, Column: 33},
						},
					},
				},
				indexErr,
			)
		})
	}
}

func TestInterpretArrayIndexingAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let z = [0, 3]

       fun test() {
           z[1] = 2
       }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	actualArray := inter.Globals["z"].GetValue()

	expectedArray := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.NewIntValueFromInt64(0),
		interpreter.NewIntValueFromInt64(3),
	).Copy().(*interpreter.ArrayValue)
	expectedArray.SetIndex(1, interpreter.NewIntValueFromInt64(2), nil)

	require.Equal(t,
		expectedArray,
		actualArray,
	)

	assert.True(t, actualArray.IsModified())

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewIntValueFromInt64(0),
			interpreter.NewIntValueFromInt64(2),
		},
		actualArray.(*interpreter.ArrayValue).Elements(),
	)
}

func TestInterpretInvalidArrayIndexingAssignment(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int) {
                   let z = [0, 3]
                   z[index] = 1
               }
            `)

			indexValue := interpreter.NewIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			require.Equal(t,
				interpreter.ArrayIndexOutOfBoundsError{
					Index: index,
					Size:  2,
					LocationRange: interpreter.LocationRange{
						Location: TestLocation,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 95, Line: 4, Column: 20},
							EndPos:   ast.Position{Offset: 101, Line: 4, Column: 26},
						},
					},
				},
				indexErr,
			)
		})
	}
}

func TestInterpretStringIndexing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = "abc"
      let x = a[0]
      let y = a[1]
      let z = a[2]
    `)

	assert.Equal(t,
		interpreter.NewStringValue("a"),
		inter.Globals["x"].GetValue(),
	)
	assert.Equal(t,
		interpreter.NewStringValue("b"),
		inter.Globals["y"].GetValue(),
	)
	assert.Equal(t,
		interpreter.NewStringValue("c"),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretInvalidStringIndexing(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 2,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               fun test(_ index: Int) {
                   let x = "ab"
                   x[index]
               }
            `)

			indexValue := interpreter.NewIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)

			var indexErr interpreter.StringIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			require.Equal(t,
				interpreter.StringIndexOutOfBoundsError{
					Index:  index,
					Length: 2,
					LocationRange: interpreter.LocationRange{
						Location: TestLocation,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 93, Line: 4, Column: 20},
							EndPos:   ast.Position{Offset: 99, Line: 4, Column: 26},
						},
					},
				},
				indexErr,
			)
		})
	}
}

func TestInterpretStringIndexingUnicode(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testUnicodeA(): Character {
          let a = "caf\u{E9}"
          return a[3]
      }

      fun testUnicodeB(): Character {
        let b = "cafe\u{301}"
        return b[3]
      }
    `)

	value, err := inter.Invoke("testUnicodeA")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewStringValue("\u00e9"),
		value,
	)

	value, err = inter.Invoke("testUnicodeB")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewStringValue("e\u0301"),
		value,
	)
}

type stringSliceTest struct {
	str           string
	from          int
	to            int
	result        string
	expectedError error
}

func TestInterpretStringSlicing(t *testing.T) {

	t.Parallel()

	locationRange := interpreter.LocationRange{
		Location: TestLocation,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 116, Line: 4, Column: 31},
			EndPos:   ast.Position{Offset: 141, Line: 4, Column: 56},
		},
	}

	tests := []stringSliceTest{
		{"abcdef", 0, 6, "abcdef", nil},
		{"abcdef", 0, 0, "", nil},
		{"abcdef", 0, 1, "a", nil},
		{"abcdef", 0, 2, "ab", nil},
		{"abcdef", 1, 2, "b", nil},
		{"abcdef", 2, 3, "c", nil},
		{"abcdef", 5, 6, "f", nil},
		{"abcdef", 1, 6, "bcdef", nil},
		// Invalid indices
		{"abcdef", -1, 0, "", interpreter.StringIndexOutOfBoundsError{
			Index:         -1,
			Length:        6,
			LocationRange: locationRange,
		}},
		{"abcdef", 0, -1, "", interpreter.StringIndexOutOfBoundsError{
			Index:         -1,
			Length:        6,
			LocationRange: locationRange,
		}},
		{"abcdef", 0, 10, "", interpreter.StringIndexOutOfBoundsError{
			Index:         10,
			Length:        6,
			LocationRange: locationRange,
		}},
		// Unicode: indices are based on characters = grapheme clusters
		{"cafe\\u{301}b", 0, 5, "cafe\u0301b", nil},
		{"cafe\\u{301}ba\\u{308}", 0, 6, "cafe\u0301ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 0, 8, "cafe\u0301ba\u0308be", nil},
		{"cafe\\u{301}b", 3, 5, "e\u0301b", nil},
		{"cafe\\u{301}ba\\u{308}", 3, 6, "e\u0301ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 3, 8, "e\u0301ba\u0308be", nil},
		{"cafe\\u{301}b", 4, 5, "b", nil},
		{"cafe\\u{301}ba\\u{308}", 4, 6, "ba\u0308", nil},
		{"cafe\\u{301}ba\\u{308}be", 4, 8, "ba\u0308be", nil},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): String {
                        let s = "%s"
                        return s.slice(from: %d, upTo: %d)
                      }
                    `,
					test.str,
					test.from,
					test.to,
				),
			)

			value, err := inter.Invoke("test")
			if test.expectedError == nil {
				require.NoError(t, err)

				assert.Equal(t,
					interpreter.NewStringValue(test.result),
					value,
				)
			} else {
				require.IsType(t,
					interpreter.Error{},
					err,
				)
				err = err.(interpreter.Error).Unwrap()

				assert.Equal(t, test.expectedError, err)
			}
		})
	}
}

func TestInterpretReturnWithoutExpression(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun returnNothing() {
           return
       }
    `)

	value, err := inter.Invoke("returnNothing")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretReturns(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`
           pub fun returnEarly(): Int {
               return 2
               return 1
           }
        `,
		ParseCheckAndInterpretOptions{
			HandleCheckerError: func(err error) {
				errs := checker.ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.UnreachableStatementError{}, errs[0])
			},
		},
	)

	value, err := inter.Invoke("returnEarly")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)
}

func TestInterpretEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersUnequal(): Bool {
          return 5 == 3
      }

      fun testIntegersEqual(): Bool {
          return 3 == 3
      }

      fun testTrueAndTrue(): Bool {
          return true == true
      }

      fun testTrueAndFalse(): Bool {
          return true == false
      }

      fun testFalseAndTrue(): Bool {
          return false == true
      }

      fun testFalseAndFalse(): Bool {
          return false == false
      }

      fun testEqualStrings(): Bool {
          return "123" == "123"
      }

      fun testUnequalStrings(): Bool {
          return "123" == "abc"
      }

      fun testUnicodeStrings(): Bool {
          return "caf\u{E9}" == "cafe\u{301}"
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersUnequal": false,
		"testIntegersEqual":   true,
		"testTrueAndTrue":     true,
		"testTrueAndFalse":    false,
		"testFalseAndTrue":    false,
		"testFalseAndFalse":   true,
		"testEqualStrings":    true,
		"testUnequalStrings":  false,
		"testUnicodeStrings":  true,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretUnequalOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersUnequal(): Bool {
          return 5 != 3
      }

      fun testIntegersEqual(): Bool {
          return 3 != 3
      }

      fun testTrueAndTrue(): Bool {
          return true != true
      }

      fun testTrueAndFalse(): Bool {
          return true != false
      }

      fun testFalseAndTrue(): Bool {
          return false != true
      }

      fun testFalseAndFalse(): Bool {
          return false != false
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersUnequal": true,
		"testIntegersEqual":   false,
		"testTrueAndTrue":     false,
		"testTrueAndFalse":    true,
		"testFalseAndTrue":    true,
		"testFalseAndFalse":   false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretLessOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 < 3
      }

      fun testIntegersEqual(): Bool {
          return 3 < 3
      }

      fun testIntegersLess(): Bool {
          return 3 < 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": false,
		"testIntegersEqual":   false,
		"testIntegersLess":    true,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretLessEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 <= 3
      }

      fun testIntegersEqual(): Bool {
          return 3 <= 3
      }

      fun testIntegersLess(): Bool {
          return 3 <= 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": false,
		"testIntegersEqual":   true,
		"testIntegersLess":    true,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretGreaterOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 > 3
      }

      fun testIntegersEqual(): Bool {
          return 3 > 3
      }

      fun testIntegersLess(): Bool {
          return 3 > 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": true,
		"testIntegersEqual":   false,
		"testIntegersLess":    false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretGreaterEqualOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testIntegersGreater(): Bool {
          return 5 >= 3
      }

      fun testIntegersEqual(): Bool {
          return 3 >= 3
      }

      fun testIntegersLess(): Bool {
          return 3 >= 5
      }
    `)

	for name, expected := range map[string]bool{
		"testIntegersGreater": true,
		"testIntegersEqual":   true,
		"testIntegersLess":    false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretOrOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testTrueTrue(): Bool {
          return true || true
      }

      fun testTrueFalse(): Bool {
          return true || false
      }

      fun testFalseTrue(): Bool {
          return false || true
      }

      fun testFalseFalse(): Bool {
          return false || false
      }
    `)

	for name, expected := range map[string]bool{
		"testTrueTrue":   true,
		"testTrueFalse":  true,
		"testFalseTrue":  true,
		"testFalseFalse": false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretOrOperatorShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return true
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() || changeY()
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretOrOperatorShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return false
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() || changeY()
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretAndOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testTrueTrue(): Bool {
          return true && true
      }

      fun testTrueFalse(): Bool {
          return true && false
      }

      fun testFalseTrue(): Bool {
          return false && true
      }

      fun testFalseFalse(): Bool {
          return false && false
      }
    `)

	for name, expected := range map[string]bool{
		"testTrueTrue":   true,
		"testTrueFalse":  false,
		"testFalseTrue":  false,
		"testFalseFalse": false,
	} {
		t.Run(name, func(t *testing.T) {
			value, err := inter.Invoke(name)
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(expected),
				value,
			)
		})
	}
}

func TestInterpretAndOperatorShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return true
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() && changeY()
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretAndOperatorShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Bool {
          x = true
          return false
      }

      fun changeY(): Bool {
          y = true
          return true
      }

      let test = changeX() && changeY()
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretExpressionStatement(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var x = 0

       fun incX() {
           x = x + 2
       }

       fun test(): Int {
           incX()
           return x
       }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretConditionalOperator(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun testTrue(): Int {
           return true ? 2 : 3
       }

       fun testFalse(): Int {
            return false ? 2 : 3
       }
    `)

	value, err := inter.Invoke("testTrue")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("testFalse")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}

func TestInterpretFunctionBindingInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun foo(): AnyStruct {
          return foo
      }
  `)

	_, err := inter.Invoke("foo")
	require.NoError(t, err)
}

func TestInterpretRecursionFib(t *testing.T) {

	t.Parallel()

	// mainly tests that the function declaration identifier is bound
	// to the function inside the function and that the arguments
	// of the function calls are evaluated in the call-site scope

	inter := parseCheckAndInterpret(t, `
       fun fib(_ n: Int): Int {
           if n < 2 {
              return n
           }
           return fib(n - 1) + fib(n - 2)
       }
   `)

	value, err := inter.Invoke(
		"fib",
		interpreter.NewIntValueFromInt64(14),
	)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(377),
		value,
	)
}

func TestInterpretRecursionFactorial(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        fun factorial(_ n: Int): Int {
            if n < 1 {
               return 1
            }

            return n * factorial(n - 1)
        }
   `)

	value, err := inter.Invoke(
		"factorial",
		interpreter.NewIntValueFromInt64(5),
	)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(120),
		value,
	)
}

func TestInterpretUnaryIntegerNegation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = -2
      let y = -(-2)
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(-2),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretUnaryBooleanNegation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let a = !true
      let b = !(!true)
      let c = !false
      let d = !(!false)
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["d"].GetValue(),
	)
}

func TestInterpretHostFunction(t *testing.T) {

	t.Parallel()

	program, err := parser2.ParseProgram(`
      pub let a = test(1, 2)
    `)

	require.NoError(t, err)

	testFunction := stdlib.NewStandardLibraryFunction(
		"test",
		&sema.FunctionType{
			Parameters: []*sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "a",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "b",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.IntType,
			),
		},
		func(invocation interpreter.Invocation) interpreter.Value {
			a := invocation.Arguments[0].(interpreter.IntValue).ToBigInt()
			b := invocation.Arguments[1].(interpreter.IntValue).ToBigInt()
			value := new(big.Int).Add(a, b)
			return interpreter.NewIntValueFromBigInt(value)
		},
	)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		sema.WithPredeclaredValues(
			[]sema.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithPredeclaredValues(
			[]interpreter.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		inter.Globals["a"].GetValue(),
	)
}

func TestInterpretHostFunctionWithVariableArguments(t *testing.T) {

	t.Parallel()

	program, err := parser2.ParseProgram(`
      pub let nothing = test(1, true, "test")
    `)

	require.NoError(t, err)

	called := false

	testFunction := stdlib.NewStandardLibraryFunction(
		"test",
		&sema.FunctionType{
			Parameters: []*sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.VoidType,
			),
			RequiredArgumentCount: sema.RequiredArgumentCount(1),
		},
		func(invocation interpreter.Invocation) interpreter.Value {
			called = true

			require.Len(t, invocation.ArgumentTypes, 3)
			assert.IsType(t, sema.IntType, invocation.ArgumentTypes[0])
			assert.IsType(t, sema.BoolType, invocation.ArgumentTypes[1])
			assert.IsType(t, sema.StringType, invocation.ArgumentTypes[2])

			require.Len(t, invocation.Arguments, 3)
			assert.Equal(t, interpreter.NewIntValueFromInt64(1), invocation.Arguments[0])
			assert.Equal(t, interpreter.BoolValue(true), invocation.Arguments[1])
			assert.Equal(t, interpreter.NewStringValue("test"), invocation.Arguments[2])

			return interpreter.VoidValue{}
		},
	)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		sema.WithPredeclaredValues(
			[]sema.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithPredeclaredValues(
			[]interpreter.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	assert.True(t, called)
}

func TestInterpretCompositeDeclaration(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		t.Run(compositeKind.Name(), func(t *testing.T) {

			t.Parallel()

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                       pub %[1]s Test {}

                       pub fun test(): %[2]sTest {
                           return %[3]s %[4]s Test%[5]s
                       }
                    `,
					compositeKind.Keyword(),
					compositeKind.Annotation(),
					compositeKind.MoveOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(nil, nil, nil),
					},
				},
			)

			value, err := inter.Invoke("test")
			require.NoError(t, err)

			assert.IsType(t,
				&interpreter.CompositeValue{},
				value,
			)
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		switch compositeKind {
		case common.CompositeKindContract,
			common.CompositeKindEvent,
			common.CompositeKindEnum:

			continue
		}

		test(compositeKind)
	}
}

func TestInterpretStructureSelfUseInInitializer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          init() {
              self
          }
      }

      fun test() {
          Test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretStructureConstructorUseInInitializerAndFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          init() {
              Test
          }

          fun test(): Test {
              return Test()
          }
      }

      fun test(): Test {
          return Test()
      }

      fun test2(): Test {
          return Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	value, err = inter.Invoke("test2")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretStructureSelfUseInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {

          fun test() {
              self
          }
      }

      fun test() {
          Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretStructureConstructorUseInFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {

          fun test() {
              Test
          }
      }

      fun test() {
          Test().test()
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretStructureDeclarationWithField(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      struct Test {
          var test: Int

          init(_ test: Int) {
              self.test = test
          }
      }

      fun test(test: Int): Int {
          let test = Test(test)
          return test.test
      }
    `)

	newValue := interpreter.NewIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	assert.Equal(t, newValue, value)
}

func TestInterpretStructureDeclarationWithFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var value = 0

      struct Test {
          fun test(_ newValue: Int) {
              value = newValue
          }
      }

      fun test(newValue: Int) {
          let test = Test()
          test.test(newValue)
      }
    `)

	newValue := interpreter.NewIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	assert.Equal(t, newValue, inter.Globals["value"].GetValue())
}

func TestInterpretStructureFunctionCall(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          fun foo(): Int {
              return 42
          }

          fun bar(): Int {
              return self.foo()
          }
      }

      let value = Test().bar()
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		inter.Globals["value"].GetValue(),
	)
}

func TestInterpretStructureFieldAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          var foo: Int

          init() {
              self.foo = 1
              let alsoSelf = self
              alsoSelf.foo = 2
          }

          fun test() {
              self.foo = 3
              let alsoSelf = self
              alsoSelf.foo = 4
          }
      }

      let test = Test()

      fun callTest() {
          test.test()
      }
    `)

	test := inter.Globals["test"].GetValue().(*interpreter.CompositeValue)

	assert.True(t, test.IsModified())

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		test.GetField("foo"),
	)

	value, err := inter.Invoke("callTest")
	require.NoError(t, err)

	assert.True(t, test.IsModified())

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		test.GetField("foo"),
	)
}

func TestInterpretStructureInitializesConstant(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          let foo: Int

          init() {
              self.foo = 42
          }
      }

      let test = Test()
    `)

	actual := inter.Globals["test"].GetValue().(*interpreter.CompositeValue).
		GetMember(inter, interpreter.ReturnEmptyLocationRange, "foo")
	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		actual,
	)
}

func TestInterpretStructureFunctionMutatesSelf(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          var foo: Int

          init() {
              self.foo = 0
          }

          fun inc() {
              self.foo = self.foo + 1
          }
      }

      fun test(): Int {
          let test = Test()
          test.inc()
          test.inc()
          return test.foo
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)
}

func TestInterpretStructCopyOnDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          let cat = Cat()
          let kitty = cat
          kitty.wasFed = true
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(false),
			interpreter.BoolValue(true),
		),
		value,
	)
}

func TestInterpretStructCopyOnDeclarationModifiedWithStructFunction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }

          fun feed() {
              self.wasFed = true
          }
      }

      fun test(): [Bool] {
          let cat = Cat()
          let kitty = cat
          kitty.feed()
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(false),
			interpreter.BoolValue(true),
		),
		value,
	)
}

func TestInterpretStructCopyOnIdentifierAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          var cat = Cat()
          let kitty = Cat()
          cat = kitty
          kitty.wasFed = true
          return [cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(false),
			interpreter.BoolValue(true),
		),
		value,
	)
}

func TestInterpretStructCopyOnIndexingAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun test(): [Bool] {
          let cats = [Cat()]
          let kitty = Cat()
          cats[0] = kitty
          kitty.wasFed = true
          return [cats[0].wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(false),
			interpreter.BoolValue(true),
		),
		value,
	)
}

func TestInterpretStructCopyOnMemberAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      struct Carrier {
          var cat: Cat
          init(cat: Cat) {
              self.cat = cat
          }
      }

      fun test(): [Bool] {
          let carrier = Carrier(cat: Cat())
          let kitty = Cat()
          carrier.cat = kitty
          kitty.wasFed = true
          return [carrier.cat.wasFed, kitty.wasFed]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.BoolValue(false),
			interpreter.BoolValue(true),
		),
		value,
	)
}

func TestInterpretStructCopyOnPassing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Cat {
          var wasFed: Bool

          init() {
              self.wasFed = false
          }
      }

      fun feed(cat: Cat) {
          cat.wasFed = true
      }

      fun test(): Bool {
          let kitty = Cat()
          feed(cat: kitty)
          return kitty.wasFed
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(false),
		value,
	)
}

func TestInterpretArrayCopy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      fun change(_ numbers: [Int]): [Int] {
          numbers[0] = 1
          return numbers
      }

      fun test(): [Int] {
          let numbers = [0]
          let numbers2 = change(numbers)
          return [
              numbers[0],
              numbers2[0]
          ]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(0),
			interpreter.NewIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretStructCopyInArray(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Foo {
          var bar: Int
          init(bar: Int) {
              self.bar = bar
          }
      }

      fun test(): [Int] {
        let foo = Foo(bar: 1)
        let foos = [foo, foo]
        foo.bar = 2
        foos[0].bar = 3
        return [foo.bar, foos[0].bar, foos[1].bar]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretMutuallyRecursiveFunctions(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun isEven(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return isOdd(n - 1)
      }

      fun isOdd(_ n: Int): Bool {
          if n == 0 {
              return false
          }
          return isEven(n - 1)
      }
    `)

	four := interpreter.NewIntValueFromInt64(4)

	value, err := inter.Invoke("isEven", four)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("isOdd", four)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(false),
		value,
	)
}

func TestInterpretUseBeforeDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var tests = 0

      fun test(): Test {
          return Test()
      }

      struct Test {
         init() {
             tests = tests + 1
         }
      }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["tests"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["tests"].GetValue(),
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["tests"].GetValue(),
	)
}

func TestInterpretOptionalVariableDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 2
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretOptionalParameterInvokedExternal(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int??): Int?? {
          return x
      }
    `)

	value, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(2),
	)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalParameterInvokedInternal(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun testActual(x: Int??): Int?? {
          return x
      }

      fun test(): Int?? {
          return testActual(x: 2)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalReturn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int?? {
          return x
      }
    `)

	value, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(2))
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
		),
		value,
	)
}

func TestInterpretOptionalAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x: Int?? = 1

      fun test() {
          x = 2
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(2),
			),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = nil
   `)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretOptionalNestingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = nil
   `)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilReturnValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int?? {
         return nil
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NilValue{},
		value,
	)
}

func TestInterpretSomeReturnValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int? {
         let x: Int? = 1
         return x
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretSomeReturnValueFromDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     fun test(): Int? {
         let foo: {String: Int} = {"a": 1}
         return foo["a"]
     }
   `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretNilCoalescingNilIntToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int? = nil
      let x: Int? = none ?? one
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingNilIntToOptionals(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int?? = nil
      let x: Int? = none ?? one
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingNilIntToOptionalNilLiteral(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let x: Int? = nil ?? one
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingRightSubtype(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = nil ?? nil
    `)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingNilInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let none: Int? = nil
      let x: Int = none ?? one
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingNilLiteralInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let x: Int = nil ?? one
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingShortCircuitLeftSuccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Int? {
          x = true
          return 1
      }

      fun changeY(): Int {
          y = true
          return 2
      }

      let test = changeX() ?? changeY()
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNilCoalescingShortCircuitLeftFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = false
      var y = false

      fun changeX(): Int? {
          x = true
          return nil
      }

      fun changeY(): Int {
          y = true
          return 2
      }

      let test = changeX() ?? changeY()
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["test"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNilCoalescingOptionalAnyStructNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = nil
      let y = x ?? true
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNilCoalescingOptionalAnyStructSome(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 2
      let y = x ?? true
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNilCoalescingOptionalRightHandSide(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 1
      let y: Int? = 2
      let z = x ?? y
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNilCoalescingBothOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = 1
     let y: Int? = 2
     let z = x ?? y
   `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNilCoalescingBothOptionalLeftNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = nil
     let y: Int? = 2
     let z = x ?? y
   `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(2),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNilsComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = nil == nil
   `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNonOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int = 1
      let y = x == nil
      let z = nil == x
   `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = 1
     let y = x == nil
   `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNestedOptionalNilComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 1
      let y = x == nil
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretOptionalNilComparisonSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 1
      let y = nil == x
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNestedOptionalNilComparisonSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 1
      let y = nil == x
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretNestedOptionalComparisonNils(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = nil
      let y: Int?? = nil
      let z = x == y
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNestedOptionalComparisonValues(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 2
      let y: Int?? = 2
      let z = x == y
    `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNestedOptionalComparisonMixed(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = 2
      let y: Int?? = nil
      let z = x == y
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretOptionalSomeValueComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = 1
     let y = x == 1
   `)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretOptionalNilValueComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int? = nil
     let y = x == 1
   `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretOptionalMap(t *testing.T) {

	t.Parallel()

	t.Run("some", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          let one: Int? = 42
          let result = one.map(fun (v: Int): String {
              return v.toString()
          })
        `)

		assert.Equal(t,
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewStringValue("42"),
			),
			inter.Globals["result"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          let none: Int? = nil
          let result = none.map(fun (v: Int): String {
              return v.toString()
          })
        `)

		assert.Equal(t,
			interpreter.NilValue{},
			inter.Globals["result"].GetValue(),
		)
	})
}

func TestInterpretCompositeNilEquality(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		t.Run(compositeKind.Name(), func(t *testing.T) {

			t.Parallel()

			var setupCode, identifier string
			if compositeKind == common.CompositeKindContract {
				identifier = "X"
			} else {
				setupCode = fmt.Sprintf(
					`pub let x: %[1]sX? %[2]s %[3]s X%[4]s`,
					compositeKind.Annotation(),
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				)
				identifier = "x"
			}

			body := "{}"
			if compositeKind == common.CompositeKindEnum {
				body = "{ case a }"
			}

			conformances := ""
			if compositeKind == common.CompositeKindEnum {
				conformances = ": Int"
			}

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      pub %[1]s X%[2]s %[3]s

                      %[4]s

                      pub let y = %[5]s == nil
                      pub let z = nil == %[5]s
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(nil, nil, nil),
					},
				},
			)

			assert.Equal(t,
				interpreter.BoolValue(false),
				inter.Globals["y"].GetValue(),
			)

			assert.Equal(t,
				interpreter.BoolValue(false),
				inter.Globals["z"].GetValue(),
			)
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindEvent {
			continue
		}

		test(compositeKind)
	}
}

func TestInterpretInterfaceConformanceNoRequirements(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindContract {
			continue
		}

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		interfaceType := AsInterfaceType("Test", compositeKind)

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      pub %[1]s interface Test {}

                      pub %[1]s TestImpl: Test {}

                      pub let test: %[2]s%[3]s %[4]s %[5]s TestImpl%[6]s
                    `,
					compositeKind.Keyword(),
					compositeKind.Annotation(),
					interfaceType,
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind, ""),
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(nil, nil, nil),
					},
				},
			)

			assert.IsType(t,
				&interpreter.CompositeValue{},
				inter.Globals["test"].GetValue(),
			)
		})
	}
}

func TestInterpretInterfaceFieldUse(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		var setupCode, identifier string
		if compositeKind == common.CompositeKindContract {
			identifier = "TestImpl"
		} else {
			interfaceType := AsInterfaceType("Test", compositeKind)

			setupCode = fmt.Sprintf(
				`pub let test: %[1]s%[2]s %[3]s %[4]s TestImpl%[5]s`,
				compositeKind.Annotation(),
				interfaceType,
				compositeKind.TransferOperator(),
				compositeKind.ConstructionKeyword(),
				constructorArguments(compositeKind, "x: 1"),
			)
			identifier = "test"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      pub %[1]s interface Test {
                          pub x: Int
                      }

                      pub %[1]s TestImpl: Test {
                          pub var x: Int

                          init(x: Int) {
                              self.x = x
                          }
                      }

                      %[2]s

                      pub let x = %[3]s.x
                    `,
					compositeKind.Keyword(),
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(
							[]interpreter.Value{
								interpreter.NewIntValueFromInt64(1),
							},
							[]sema.Type{
								sema.IntType,
							},
							[]sema.Type{
								sema.IntType,
							},
						),
					},
				},
			)

			assert.Equal(t,
				interpreter.NewIntValueFromInt64(1),
				inter.Globals["x"].GetValue(),
			)
		})
	}
}

func TestInterpretInterfaceFunctionUse(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		var setupCode, identifier string
		if compositeKind == common.CompositeKindContract {
			identifier = "TestImpl"
		} else {
			interfaceType := AsInterfaceType("Test", compositeKind)

			setupCode = fmt.Sprintf(
				`pub let test: %[1]s %[2]s %[3]s %[4]s TestImpl%[5]s`,
				compositeKind.Annotation(),
				interfaceType,
				compositeKind.TransferOperator(),
				compositeKind.ConstructionKeyword(),
				constructorArguments(compositeKind, ""),
			)
			identifier = "test"
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      pub %[1]s interface Test {
                          pub fun test(): Int
                      }

                      pub %[1]s TestImpl: Test {
                          pub fun test(): Int {
                              return 2
                          }
                      }

                      %[2]s

                      pub let val = %[3]s.test()
                    `,
					compositeKind.Keyword(),
					setupCode,
					identifier,
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(nil, nil, nil),
					},
				},
			)

			assert.Equal(t,
				interpreter.NewIntValueFromInt64(2),
				inter.Globals["val"].GetValue(),
			)
		})
	}
}

func TestInterpretImport(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub fun answer(): Int {
              return 42
          }
        `,
		checker.ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import answer from "imported"

          pub fun test(): Int {
              return answer()
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						assert.Equal(t,
							ImportedLocation,
							importedLocation,
						)

						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)
	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		value,
	)
}

func TestInterpretImportError(t *testing.T) {

	t.Parallel()

	valueDeclarations :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}.ToSemaValueDeclarations()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          pub fun answer(): Int {
              return panic("?!")
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import answer from "imported"

          pub fun test(): Int {
              return answer()
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
				sema.WithImportHandler(
					func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						assert.Equal(t,
							ImportedLocation,
							importedLocation,
						)

						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)
	require.NoError(t, err)

	values := stdlib.StandardLibraryFunctions{
		stdlib.PanicFunction,
	}.ToInterpreterValueDeclarations()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithPredeclaredValues(values),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")

	var panicErr stdlib.PanicError
	require.ErrorAs(t, err, &panicErr)

	assert.Equal(t,
		"?!",
		panicErr.Message,
	)
}

func TestInterpretDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"a": 1, "b": 2}
    `)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("a"), interpreter.NewIntValueFromInt64(1),
		interpreter.NewStringValue("b"), interpreter.NewIntValueFromInt64(2),
	).Copy()

	actualDict := inter.Globals["x"].GetValue()

	assert.Equal(t,
		expectedDict,
		actualDict,
	)

	assert.True(t, actualDict.IsModified())
}

func TestInterpretDictionaryInsertionOrder(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"c": 3, "a": 1, "b": 2}
    `)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("c"), interpreter.NewIntValueFromInt64(3),
		interpreter.NewStringValue("a"), interpreter.NewIntValueFromInt64(1),
		interpreter.NewStringValue("b"), interpreter.NewIntValueFromInt64(2),
	).Copy()

	actualDict := inter.Globals["x"].GetValue()

	assert.Equal(t,
		expectedDict,
		actualDict,
	)

	assert.True(t, actualDict.IsModified())
}

func TestInterpretDictionaryIndexingString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"abc": 1, "def": 2}
      let a = x["abc"]
      let b = x["def"]
      let c = x["ghi"]
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(2),
		),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)
}

func TestInterpretDictionaryIndexingBool(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {true: 1, false: 2}
      let a = x[true]
      let b = x[false]
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(2),
		),
		inter.Globals["b"].GetValue(),
	)
}

func TestInterpretDictionaryIndexingInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {23: "a", 42: "b"}
      let a = x[23]
      let b = x[42]
      let c = x[100]
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewStringValue("a"),
		),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewStringValue("b"),
		),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)
}

func TestInterpretDictionaryIndexingAssignmentExisting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"abc": 42}
      fun test() {
          x["abc"] = 23
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("abc"), interpreter.NewIntValueFromInt64(42),
	).Copy().(*interpreter.DictionaryValue)
	expectedDict.Set(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewStringValue("abc"),
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(23),
		),
	)

	actualDict := inter.Globals["x"].GetValue().(*interpreter.DictionaryValue)

	require.Equal(t,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.
		Get(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("abc"))

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(interpreter.NewIntValueFromInt64(23)),
		newValue,
	)

	expectedEntries := interpreter.NewStringValueOrderedMap()
	expectedEntries.Set("abc", interpreter.NewIntValueFromInt64(23))

	assert.Equal(t,
		expectedEntries,
		actualDict.Entries,
	)

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewStringValue("abc"),
		},
		actualDict.Keys.Elements(),
	)

	assert.True(t, actualDict.IsModified())
}

func TestInterpretDictionaryIndexingAssignmentNew(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"def": 42}
      fun test() {
          x["abc"] = 23
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("def"), interpreter.NewIntValueFromInt64(42),
	).Copy().(*interpreter.DictionaryValue)
	expectedDict.Set(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewStringValue("abc"),
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(23),
		),
	)

	actualDict := inter.Globals["x"].GetValue().(*interpreter.DictionaryValue)

	require.Equal(t,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.
		Get(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("abc"))

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(interpreter.NewIntValueFromInt64(23)),
		newValue,
	)

	expectedEntries := interpreter.NewStringValueOrderedMap()
	expectedEntries.Set("def", interpreter.NewIntValueFromInt64(42))
	expectedEntries.Set("abc", interpreter.NewIntValueFromInt64(23))

	assert.Equal(t,
		expectedEntries,
		actualDict.Entries,
	)

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewStringValue("def"),
			interpreter.NewStringValue("abc"),
		},
		actualDict.Keys.Elements(),
	)

	assert.True(t, actualDict.IsModified())
}

func TestInterpretDictionaryIndexingAssignmentNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"def": 42, "abc": 23}
      fun test() {
          x["def"] = nil
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("def"), interpreter.NewIntValueFromInt64(42),
		interpreter.NewStringValue("abc"), interpreter.NewIntValueFromInt64(23),
	).Copy().(*interpreter.DictionaryValue)
	expectedDict.Set(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewStringValue("def"),
		interpreter.NilValue{},
	)

	actualDict := inter.Globals["x"].GetValue().(*interpreter.DictionaryValue)

	require.Equal(t,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.
		Get(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("def"))

	assert.Equal(t,
		interpreter.NilValue{},
		newValue,
	)

	expectedEntries := interpreter.NewStringValueOrderedMap()
	expectedEntries.Set("abc", interpreter.NewIntValueFromInt64(23))

	assert.Equal(t,
		expectedEntries,
		actualDict.Entries,
	)

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewStringValue("abc"),
		},
		actualDict.Keys.Elements(),
	)

	assert.True(t, actualDict.IsModified())
}

func TestInterpretOptionalAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 42
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretOptionalAnyStructFailableCasting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 42
      let y = (x ?? 23) as? Int
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretOptionalAnyStructFailableCastingInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 23
      let y = x ?? 42
      let z = y as? Int
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(23),
		),
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(23),
		inter.Globals["y"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(23),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretOptionalAnyStructFailableCastingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = nil
      let y = x ?? 42
      let z = y as? Int
    `)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		inter.Globals["y"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretReferenceFailableDowncasting(t *testing.T) {

	t.Parallel()

	t.Run("ephemeral", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource interface RI {}

          resource R: RI {}

          fun testInvalidUnauthorized(): &R? {
              let r  <- create R()
              let ref: AnyStruct = &r as &R{RI}
              let ref2 = ref as? &R
              destroy r
              return ref2
          }

          fun testValidAuthorized(): &R? {
              let r  <- create R()
              let ref: AnyStruct = &r as auth &R{RI}
              let ref2 = ref as? &R
              destroy r
              return ref2
          }

          fun testValidRestricted(): &R{RI}? {
              let r  <- create R()
              let ref: AnyStruct = &r as &R{RI}
              let ref2 = ref as? &R{RI}
              destroy r
              return ref2
          }
        `)

		result, err := inter.Invoke("testInvalidUnauthorized")
		require.NoError(t, err)

		assert.Equal(t,
			interpreter.NilValue{},
			result,
		)

		result, err = inter.Invoke("testValidAuthorized")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)

		result, err = inter.Invoke("testValidRestricted")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)
	})

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		var inter *interpreter.Interpreter

		getType := func(name string) sema.Type {
			variable, ok := inter.Program.Elaboration.GlobalTypes.Get(name)
			require.True(t, ok, "missing global type %s", name)
			return variable.Type
		}

		// Inject a function that return a storage reference value,
		// which is borrowed as:
		// - `&R{RI}` (unauthorized, if argument for parameter `authorized` == false)
		// - `auth &R{RI}` (authorized, if argument for parameter `authorized` == true)

		storageAddress := common.BytesToAddress([]byte{0x42})
		const storageKey = "test storage key"

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				{
					Name: "getStorageReference",
					Type: &sema.FunctionType{
						Parameters: []*sema.Parameter{
							{
								Label:      "authorized",
								Identifier: "authorized",
								TypeAnnotation: sema.NewTypeAnnotation(
									sema.BoolType,
								),
							},
						},
						ReturnTypeAnnotation: sema.NewTypeAnnotation(
							sema.AnyStructType,
						),
					},
					Function: interpreter.NewHostFunctionValue(
						func(invocation interpreter.Invocation) interpreter.Value {

							authorized := bool(invocation.Arguments[0].(interpreter.BoolValue))

							riType := getType("RI").(*sema.InterfaceType)
							rType := getType("R")

							return &interpreter.StorageReferenceValue{
								Authorized:           authorized,
								TargetStorageAddress: storageAddress,
								TargetKey:            storageKey,
								BorrowedType: &sema.RestrictedType{
									Type: rType,
									Restrictions: []*sema.InterfaceType{
										riType,
									},
								},
							}
						},
					),
				},
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		var r interpreter.Value

		inter = parseCheckAndInterpretWithOptions(t,
			`
              resource interface RI {}

              resource R: RI {}

              fun createR(): @R {
                  return <- create R()
              }

              fun testInvalidUnauthorized(): &R? {
                  let ref: AnyStruct = getStorageReference(authorized: false)
                  return ref as? &R
              }

              fun testValidAuthorized(): &R? {
                  let ref: AnyStruct = getStorageReference(authorized: true)
                  return ref as? &R
              }

              fun testValidRestricted(): &R{RI}? {
                  let ref: AnyStruct = getStorageReference(authorized: false)
                  return ref as? &R{RI}
              }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(values),
					interpreter.WithStorageReadHandler(
						func(
							inter *interpreter.Interpreter,
							address common.Address,
							key string,
							deferred bool,
						) interpreter.OptionalValue {

							if address != storageAddress || key != storageKey {
								return interpreter.NilValue{}
							}

							// When the storage reference is dereferenced,
							// return r (a resource of type R)

							return interpreter.NewSomeValueOwningNonCopying(r)
						},
					),
				},
			},
		)

		var err error
		r, err = inter.Invoke("createR")
		require.NoError(t, err)

		result, err := inter.Invoke("testInvalidUnauthorized")
		require.NoError(t, err)

		assert.Equal(t,
			interpreter.NilValue{},
			result,
		)

		result, err = inter.Invoke("testValidAuthorized")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)

		result, err = inter.Invoke("testValidRestricted")
		require.NoError(t, err)

		assert.IsType(t,
			&interpreter.SomeValue{},
			result,
		)
	})
}

func TestInterpretArrayLength(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let y = [1, 2, 3].length
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretStringLength(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = "cafe\u{301}".length
      let y = x
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(4),
		inter.Globals["x"].GetValue(),
	)
	assert.Equal(t,
		interpreter.NewIntValueFromInt64(4),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretStructureFunctionBindingInside(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        struct X {
            fun foo(): ((): X) {
                return self.bar
            }

            fun bar(): X {
                return self
            }
        }

        fun test(): X {
            let x = X()
            let bar = x.foo()
            return bar()
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretStructureFunctionBindingOutside(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        struct X {
            fun foo(): X {
                return self
            }
        }

        fun test(): X {
            let x = X()
            let bar = x.foo
            return bar()
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretArrayAppend(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test() {
          xs.append(4)
      }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	expectedArray := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.NewIntValueFromInt64(1),
		interpreter.NewIntValueFromInt64(2),
		interpreter.NewIntValueFromInt64(3),
	).Copy().(*interpreter.ArrayValue)
	expectedArray.Append(interpreter.NewIntValueFromInt64(4))

	actualArray := inter.Globals["xs"].GetValue()

	require.Equal(t,
		expectedArray,
		actualArray,
	)

	assert.True(t, actualArray.IsModified())

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		},
		actualArray.(*interpreter.ArrayValue).Elements(),
	)
}

func TestInterpretArrayAppendBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let x = [1, 2, 3]
          let y = x.append
          y(4)
          return x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		),
		value,
	)
}

func TestInterpretArrayAppendAll(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
		  a.appendAll([3, 4])
		  return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		),
		value,
	)
}

func TestInterpretArrayAppendAllBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          let b = a.appendAll
		  b([3, 4])
		  return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		),
		value,
	)
}

func TestInterpretArrayConcat(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          return a.concat([3, 4])
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		),
		value,
	)
}

func TestInterpretArrayConcatBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
          let b = a.concat
          return b([3, 4])
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(4),
		),
		value,
	)
}

func TestInterpretArrayConcatDoesNotModifyOriginalArray(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let a = [1, 2]
		  a.concat([3, 4])
		  return a
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretArrayInsert(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name           string
		index          int
		expectedValues []interpreter.Value
	}

	for _, testCase := range []testCase{
		{
			name:  "start",
			index: 0,
			expectedValues: []interpreter.Value{
				interpreter.NewIntValueFromInt64(100),
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewIntValueFromInt64(2),
				interpreter.NewIntValueFromInt64(3),
			},
		},
		{
			name:  "middle",
			index: 1,
			expectedValues: []interpreter.Value{
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewIntValueFromInt64(100),
				interpreter.NewIntValueFromInt64(2),
				interpreter.NewIntValueFromInt64(3),
			},
		},
		{
			name:  "end",
			index: 3,
			expectedValues: []interpreter.Value{
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewIntValueFromInt64(2),
				interpreter.NewIntValueFromInt64(3),
				interpreter.NewIntValueFromInt64(100),
			},
		},
	} {

		t.Run(testCase.name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
              let x = [1, 2, 3]

              fun test(_ index: Int) {
                  x.insert(at: index, 100)
              }
            `)

			_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(int64(testCase.index)))
			require.NoError(t, err)

			actualArray := inter.Globals["x"].GetValue()

			require.IsType(t, &interpreter.ArrayValue{}, actualArray)

			assert.True(t, actualArray.IsModified())

			assert.Equal(t,
				testCase.expectedValues,
				actualArray.(*interpreter.ArrayValue).Elements(),
			)
		})
	}
}

func TestInterpretInvalidArrayInsert(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 4,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               let x = [1, 2, 3]

               fun test(_ index: Int) {
                   x.insert(at: index, 4)
               }
            `)

			indexValue := interpreter.NewIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			require.Equal(t,
				interpreter.ArrayIndexOutOfBoundsError{
					Index: index,
					Size:  3,
					LocationRange: interpreter.LocationRange{
						Location: TestLocation,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 94, Line: 5, Column: 19},
							EndPos:   ast.Position{Offset: 115, Line: 5, Column: 40},
						},
					},
				},
				indexErr,
			)
		})
	}
}

func TestInterpretArrayRemove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = [1, 2, 3]
      let y = x.remove(at: 1)
    `)

	expectedArray := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.NewIntValueFromInt64(1),
		interpreter.NewIntValueFromInt64(2),
		interpreter.NewIntValueFromInt64(3),
	).Copy().(*interpreter.ArrayValue)
	expectedArray.Remove(1, nil)

	actualArray := inter.Globals["x"].GetValue()

	require.Equal(t,
		expectedArray,
		actualArray,
	)

	assert.True(t, actualArray.IsModified())

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(3),
		},
		actualArray.(*interpreter.ArrayValue).Elements(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretInvalidArrayRemove(t *testing.T) {

	t.Parallel()

	for name, index := range map[string]int{
		"negative":          -1,
		"larger than count": 3,
	} {

		t.Run(name, func(t *testing.T) {

			inter := parseCheckAndInterpret(t, `
               let x = [1, 2, 3]

               fun test(_ index: Int) {
                   x.remove(at: index)
               }
            `)

			indexValue := interpreter.NewIntValueFromInt64(int64(index))
			_, err := inter.Invoke("test", indexValue)

			var indexErr interpreter.ArrayIndexOutOfBoundsError
			require.ErrorAs(t, err, &indexErr)

			require.Equal(t,
				interpreter.ArrayIndexOutOfBoundsError{
					Index: index,
					Size:  3,
					LocationRange: interpreter.LocationRange{
						Location: TestLocation,
						Range: ast.Range{
							StartPos: ast.Position{Offset: 94, Line: 5, Column: 19},
							EndPos:   ast.Position{Offset: 112, Line: 5, Column: 37},
						},
					},
				},
				indexErr,
			)
		})
	}
}

func TestInterpretArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = [1, 2, 3]
      let y = x.removeFirst()
    `)

	expectedArray := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.NewIntValueFromInt64(1),
		interpreter.NewIntValueFromInt64(2),
		interpreter.NewIntValueFromInt64(3),
	).Copy().(*interpreter.ArrayValue)
	expectedArray.RemoveFirst(nil)

	actualArray := inter.Globals["x"].GetValue()

	require.Equal(t,
		expectedArray,
		actualArray,
	)

	assert.True(t, actualArray.IsModified())

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(3),
		},
		actualArray.(*interpreter.ArrayValue).Elements(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretInvalidArrayRemoveFirst(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x: [Int] = []

       fun test() {
           x.removeFirst()
       }
    `)

	_, err := inter.Invoke("test")

	var indexErr interpreter.ArrayIndexOutOfBoundsError
	require.ErrorAs(t, err, &indexErr)

	require.Equal(t,
		interpreter.ArrayIndexOutOfBoundsError{
			Index: 0,
			Size:  0,
			LocationRange: interpreter.LocationRange{
				Location: TestLocation,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 58, Line: 5, Column: 11},
					EndPos:   ast.Position{Offset: 72, Line: 5, Column: 25},
				},
			},
		},
		indexErr,
	)
}

func TestInterpretArrayRemoveLast(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
          let x = [1, 2, 3]
          let y = x.removeLast()
    `)

	expectedArray := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.NewIntValueFromInt64(1),
		interpreter.NewIntValueFromInt64(2),
		interpreter.NewIntValueFromInt64(3),
	).Copy().(*interpreter.ArrayValue)
	expectedArray.RemoveLast(nil)

	actualArray := inter.Globals["x"].GetValue()

	require.Equal(t,
		expectedArray,
		actualArray,
	)

	assert.True(t, actualArray.IsModified())

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
		},
		actualArray.(*interpreter.ArrayValue).Elements(),
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretInvalidArrayRemoveLast(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       let x: [Int] = []

       fun test() {
           x.removeLast()
       }
    `)

	_, err := inter.Invoke("test")

	var indexErr interpreter.ArrayIndexOutOfBoundsError
	require.ErrorAs(t, err, &indexErr)

	require.Equal(t,
		interpreter.ArrayIndexOutOfBoundsError{
			Index: -1,
			Size:  0,
			LocationRange: interpreter.LocationRange{
				Location: TestLocation,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 58, Line: 5, Column: 11},
					EndPos:   ast.Position{Offset: 71, Line: 5, Column: 24},
				},
			},
		},
		indexErr,
	)
}

func TestInterpretArrayContains(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun doesContain(): Bool {
          let a = [1, 2]
          return a.contains(1)
      }

      fun doesNotContain(): Bool {
          let a = [1, 2]
          return a.contains(3)
      }
    `)

	value, err := inter.Invoke("doesContain")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("doesNotContain")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(false),
		value,
	)
}

func TestInterpretDictionaryContainsKey(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun doesContainKey(): Bool {
		  let x = {
			  1: "one",
			  2: "two"
		  }
          return x.containsKey(1)
      }

      fun doesNotContainKey(): Bool {
		  let x = {
			  1: "one",
			  2: "two"
		  }
          return x.containsKey(3)
      }
    `)

	value, err := inter.Invoke("doesContainKey")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("doesNotContainKey")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(false),
		value,
	)
}

func TestInterpretStringConcat(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): String {
          let a = "abc"
          return a.concat("def")
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewStringValue("abcdef"),
		value,
	)
}

func TestInterpretStringConcatBound(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): String {
          let a = "abc"
          let b = a.concat
          return b("def")
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewStringValue("abcdef"),
		value,
	)
}

func TestInterpretDictionaryRemove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = {"abc": 1, "def": 2}
      let removed = xs.remove(key: "abc")
    `)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("abc"), interpreter.NewIntValueFromInt64(1),
		interpreter.NewStringValue("def"), interpreter.NewIntValueFromInt64(2),
	).Copy().(*interpreter.DictionaryValue)
	expectedDict.Remove(nil, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("abc"))

	actualDict := inter.Globals["xs"].GetValue().(*interpreter.DictionaryValue)

	assert.Equal(t,
		expectedDict,
		actualDict,
	)

	expectedEntries := interpreter.NewStringValueOrderedMap()
	expectedEntries.Set("def", interpreter.NewIntValueFromInt64(2))

	assert.Equal(t,
		expectedEntries,
		actualDict.Entries,
	)

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewStringValue("def"),
		},
		actualDict.Keys.Elements(),
	)

	assert.True(t, actualDict.IsModified())

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["removed"].GetValue(),
	)
}

func TestInterpretDictionaryInsert(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = {"abc": 1, "def": 2}
      let inserted = xs.insert(key: "abc", 3)
    `)

	expectedDict := interpreter.NewDictionaryValueUnownedNonCopying(
		interpreter.NewStringValue("abc"), interpreter.NewIntValueFromInt64(1),
		interpreter.NewStringValue("def"), interpreter.NewIntValueFromInt64(2),
	).Copy().(*interpreter.DictionaryValue)
	expectedDict.Insert(
		nil,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewStringValue("abc"),
		interpreter.NewIntValueFromInt64(3),
	)

	actualDict := inter.Globals["xs"].GetValue().(*interpreter.DictionaryValue)

	require.Equal(t,
		expectedDict,
		actualDict,
	)

	expectedEntries := interpreter.NewStringValueOrderedMap()
	expectedEntries.Set("abc", interpreter.NewIntValueFromInt64(3))
	expectedEntries.Set("def", interpreter.NewIntValueFromInt64(2))

	assert.Equal(t,
		expectedEntries,
		actualDict.Entries,
	)

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewStringValue("abc"),
			interpreter.NewStringValue("def"),
		},
		actualDict.Keys.Elements(),
	)

	assert.True(t, actualDict.IsModified())

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(1),
		),
		inter.Globals["inserted"].GetValue(),
	)
}

func TestInterpretDictionaryKeys(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [String] {
          let dict = {"def": 2, "abc": 1}
          dict.insert(key: "a", 3)
          return dict.keys
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewStringValue("def"),
			interpreter.NewStringValue("abc"),
			interpreter.NewStringValue("a"),
		),
		value,
	)
}

func TestInterpretDictionaryValues(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Int] {
          let dict = {"def": 2, "abc": 1}
          dict.insert(key: "a", 3)
          return dict.values
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(3),
		),
		value,
	)
}

func TestInterpretDictionaryKeyTypes(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"String":         `"abc"`,
		"Character":      `"X"`,
		"Address":        `0x1`,
		"Bool":           `true`,
		"Path":           `/storage/a`,
		"StoragePath":    `/storage/a`,
		"PublicPath":     `/public/a`,
		"PrivatePath":    `/private/a`,
		"CapabilityPath": `/private/a`,
	}

	for _, integerType := range sema.AllIntegerTypes {
		tests[integerType.String()] = `42`
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		var literal string

		if sema.IsSubType(fixedPointType, sema.SignedFixedPointType) {
			literal = "-1.23"
		} else {
			literal = "1.23"
		}

		tests[fixedPointType.String()] = literal
	}

	for ty, code := range tests {
		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let k: %s = %s
                      let xs = {k: "test"}
                      let v = xs[k]
                    `,
					ty,
					code,
				),
			)

			assert.Equal(t,
				interpreter.NewSomeValueOwningNonCopying(
					interpreter.NewStringValue("test"),
				),
				inter.Globals["v"].GetValue(),
			)
		})
	}
}

func TestInterpretIndirectDestroy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test() {
          let x <- create X()
          destroy x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretUnaryMove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun foo(x: @X): @X {
          return <-x
      }

      fun bar() {
          let x <- foo(x: <-create X())
          destroy x
      }
    `)

	value, err := inter.Invoke("bar")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretResourceMoveInArrayAndDestroy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destroys = 0

      resource Foo {
          var bar: Int

          init(bar: Int) {
              self.bar = bar
          }

          destroy() {
              destroys = destroys + 1
          }
      }

      fun test(): Int {
          let foo1 <- create Foo(bar: 1)
          let foo2 <- create Foo(bar: 2)
          let foos <- [<-foo1, <-foo2]
          let bar = foos[1].bar
          destroy foos
          return bar
      }
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destroys"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["destroys"].GetValue(),
	)
}

func TestInterpretResourceMoveInDictionaryAndDestroy(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destroys = 0

      resource Foo {
          var bar: Int

          init(bar: Int) {
              self.bar = bar
          }

          destroy() {
              destroys = destroys + 1
          }
      }

      fun test() {
          let foo1 <- create Foo(bar: 1)
          let foo2 <- create Foo(bar: 2)
          let foos <- {"foo1": <-foo1, "foo2": <-foo2}
          destroy foos
      }
    `)

	require.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destroys"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["destroys"].GetValue(),
	)
}

func TestInterpretClosure(t *testing.T) {

	t.Parallel()

	// Create a closure that increments and returns
	// a variable each time it is invoked.

	inter := parseCheckAndInterpret(t, `
        fun makeCounter(): ((): Int) {
            var count = 0
            return fun (): Int {
                count = count + 1
                return count
            }
        }

        let test = makeCounter()
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}

// TestInterpretCompositeFunctionInvocationFromImportingProgram checks
// that member functions of imported composites can be invoked from an importing program.
// See https://github.com/dapperlabs/flow-go/issues/838
//
func TestInterpretCompositeFunctionInvocationFromImportingProgram(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          // function must have arguments
          pub fun x(x: Int) {}

          // invocation must be in composite
          pub struct Y {

              pub fun x() {
                  x(x: 1)
              }
          }
        `,
		checker.ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import Y from "imported"

          pub fun test() {
              // get member must bind using imported interpreter
              Y().x()
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						assert.Equal(t,
							ImportedLocation,
							importedLocation,
						)

						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)
	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretSwapVariables(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): [Int] {
           var x = 2
           var y = 3
           x <-> y
           return [x, y]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(3),
			interpreter.NewIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretSwapArrayAndField(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       struct Foo {
           var bar: Int

           init(bar: Int) {
               self.bar = bar
           }
       }

       fun test(): [Int] {
           let foo = Foo(bar: 1)
           let nums = [2]
           foo.bar <-> nums[0]
           return [foo.bar, nums[0]]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretResourceDestroyExpressionNoDestructor(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       resource R {}

       fun test() {
           let r <- create R()
           destroy r
       }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)
}

func TestInterpretResourceDestroyExpressionDestructor(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       var ranDestructor = false

       resource R {
           destroy() {
               ranDestructor = true
           }
       }

       fun test() {
           let r <- create R()
           destroy r
       }
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructor"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["ranDestructor"].GetValue(),
	)
}

func TestInterpretResourceDestroyExpressionNestedResources(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var ranDestructorA = false
      var ranDestructorB = false

      resource B {
          destroy() {
              ranDestructorB = true
          }
      }

      resource A {
          let b: @B

          init(b: @B) {
              self.b <- b
          }

          destroy() {
              ranDestructorA = true
              destroy self.b
          }
      }

      fun test() {
          let b <- create B()
          let a <- create A(b: <-b)
          destroy a
      }
    `)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructorA"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructorB"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["ranDestructorA"].GetValue(),
	)

	assert.Equal(t,
		interpreter.BoolValue(true),
		inter.Globals["ranDestructorB"].GetValue(),
	)
}

func TestInterpretResourceDestroyArray(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destructionCount = 0

      resource R {
          destroy() {
              destructionCount = destructionCount + 1
          }
      }

      fun test() {
          let rs <- [<-create R(), <-create R()]
          destroy rs
      }
    `)

	require.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["destructionCount"].GetValue(),
	)
}

func TestInterpretResourceDestroyDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destructionCount = 0

      resource R {
          destroy() {
              destructionCount = destructionCount + 1
          }
      }

      fun test() {
          let rs <- {"r1": <-create R(), "r2": <-create R()}
          destroy rs
      }
    `)

	require.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(2),
		inter.Globals["destructionCount"].GetValue(),
	)
}

func TestInterpretResourceDestroyOptionalSome(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destructionCount = 0

      resource R {
          destroy() {
              destructionCount = destructionCount + 1
          }
      }

      fun test() {
          let maybeR: @R? <- create R()
          destroy maybeR
      }
    `)

	require.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(1),
		inter.Globals["destructionCount"].GetValue(),
	)
}

func TestInterpretResourceDestroyOptionalNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var destructionCount = 0

      resource R {
          destroy() {
              destructionCount = destructionCount + 1
          }
      }

      fun test() {
          let maybeR: @R? <- nil
          destroy maybeR
      }
    `)

	require.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)
}

// TestInterpretResourceDestroyExpressionResourceInterfaceCondition tests that
// the resource interface's destructor is called, even if the conforming resource
// does not have an destructor
//
func TestInterpretResourceDestroyExpressionResourceInterfaceCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource interface I {
          destroy() {
              pre { false }
          }
      }

      resource R: I {}

      fun test() {
          let r <- create R()
          destroy r
      }
    `)

	_, err := inter.Invoke("test")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

// TestInterpretInterfaceInitializer tests that the interface's initializer
// is called, even if the conforming composite does not have an initializer
//
func TestInterpretInterfaceInitializer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct interface I {
          init(a a1: Bool) {
              pre { a1 }
          }
      }

      struct S: I {
          init(a a2: Bool) {}
      }

      fun test() {
          S(a: false)
      }
    `)

	_, err := inter.Invoke("test")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretEmitEvent(t *testing.T) {

	t.Parallel()

	var actualEvents []*interpreter.CompositeValue

	inter := parseCheckAndInterpret(t,
		`
          event Transfer(to: Int, from: Int)
          event TransferAmount(to: Int, from: Int, amount: Int)

          fun test() {
              emit Transfer(to: 1, from: 2)
              emit Transfer(to: 3, from: 4)
              emit TransferAmount(to: 1, from: 2, amount: 100)
          }
        `,
	)

	inter.SetOnEventEmittedHandler(
		func(_ *interpreter.Interpreter, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
			actualEvents = append(actualEvents, event)
			return nil
		},
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	transferEventType := checker.RequireGlobalType(t, inter.Program.Elaboration, "Transfer")
	transferAmountEventType := checker.RequireGlobalType(t, inter.Program.Elaboration, "TransferAmount")

	members1 := interpreter.NewStringValueOrderedMap()
	members1.Set("to", interpreter.NewIntValueFromInt64(1))
	members1.Set("from", interpreter.NewIntValueFromInt64(2))

	members2 := interpreter.NewStringValueOrderedMap()
	members2.Set("to", interpreter.NewIntValueFromInt64(3))
	members2.Set("from", interpreter.NewIntValueFromInt64(4))

	members3 := interpreter.NewStringValueOrderedMap()
	members3.Set("to", interpreter.NewIntValueFromInt64(1))
	members3.Set("from", interpreter.NewIntValueFromInt64(2))
	members3.Set("amount", interpreter.NewIntValueFromInt64(100))

	expectedEvents := []*interpreter.CompositeValue{
		interpreter.NewCompositeValue(
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			members1,
			nil,
		),
		interpreter.NewCompositeValue(
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			members2,
			nil,
		),
		interpreter.NewCompositeValue(
			TestLocation,
			TestLocation.QualifiedIdentifier(transferAmountEventType.ID()),
			common.CompositeKindEvent,
			members3,
			nil,
		),
	}

	for _, event := range expectedEvents {
		event.InitializeFunctions(inter)
	}

	assert.Equal(t, expectedEvents, actualEvents)
}

type testValue struct {
	value              interpreter.Value
	literal            string
	notAsDictionaryKey bool
}

func (v testValue) String() string {
	if v.literal == "" {
		return v.value.String(interpreter.StringResults{})
	}
	return v.literal
}

func TestInterpretEmitEventParameterTypes(t *testing.T) {

	t.Parallel()

	validTypes := map[string]testValue{
		"String":    {value: interpreter.NewStringValue("test")},
		"Character": {value: interpreter.NewStringValue("X")},
		"Bool":      {value: interpreter.BoolValue(true)},
		"Address": {
			literal: `0x1`,
			value:   interpreter.NewAddressValueFromBytes([]byte{0x1}),
		},
		// Int*
		"Int":    {value: interpreter.NewIntValueFromInt64(42)},
		"Int8":   {value: interpreter.Int8Value(42)},
		"Int16":  {value: interpreter.Int16Value(42)},
		"Int32":  {value: interpreter.Int32Value(42)},
		"Int64":  {value: interpreter.Int64Value(42)},
		"Int128": {value: interpreter.NewInt128ValueFromInt64(42)},
		"Int256": {value: interpreter.NewInt256ValueFromInt64(42)},
		// UInt*
		"UInt":    {value: interpreter.NewUIntValueFromUint64(42)},
		"UInt8":   {value: interpreter.UInt8Value(42)},
		"UInt16":  {value: interpreter.UInt16Value(42)},
		"UInt32":  {value: interpreter.UInt32Value(42)},
		"UInt64":  {value: interpreter.UInt64Value(42)},
		"UInt128": {value: interpreter.NewUInt128ValueFromUint64(42)},
		"UInt256": {value: interpreter.NewUInt256ValueFromUint64(42)},
		// Word*
		"Word8":  {value: interpreter.Word8Value(42)},
		"Word16": {value: interpreter.Word16Value(42)},
		"Word32": {value: interpreter.Word32Value(42)},
		"Word64": {value: interpreter.Word64Value(42)},
		// Fix*
		"Fix64": {value: interpreter.Fix64Value(123000000)},
		// UFix*
		"UFix64": {value: interpreter.UFix64Value(123000000)},
		// Struct
		"S": {
			literal: `S()`,
			value: func() interpreter.Value {
				v := interpreter.NewCompositeValue(
					TestLocation,
					"S",
					common.CompositeKindStructure,
					interpreter.NewStringValueOrderedMap(),
					nil,
				)
				v.Functions = map[string]interpreter.FunctionValue{}
				return v
			}(),
			notAsDictionaryKey: true,
		},
	}

	for _, integerType := range sema.AllIntegerTypes {

		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		if _, ok := validTypes[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		switch fixedPointType {
		case sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := validTypes[fixedPointType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", fixedPointType))
		}
	}

	tests := map[string]testValue{}

	for validType, testCase := range validTypes {
		tests[validType] = testCase

		tests[fmt.Sprintf("%s?", validType)] =
			testValue{
				value:   interpreter.NewSomeValueOwningNonCopying(testCase.value),
				literal: testCase.literal,
			}

		tests[fmt.Sprintf("[%s]", validType)] =
			testValue{
				value:   interpreter.NewArrayValueUnownedNonCopying(testCase.value),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		tests[fmt.Sprintf("[%s; 1]", validType)] =
			testValue{
				value:   interpreter.NewArrayValueUnownedNonCopying(testCase.value),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		if !testCase.notAsDictionaryKey {

			tests[fmt.Sprintf("{%[1]s: %[1]s}", validType)] =
				testValue{
					value:   interpreter.NewDictionaryValueUnownedNonCopying(testCase.value, testCase.value).Copy(),
					literal: fmt.Sprintf("{%[1]s as %[2]s: %[1]s as %[2]s}", testCase, validType),
				}
		}
	}

	for ty, value := range tests {

		t.Run(ty, func(t *testing.T) {

			code := fmt.Sprintf(
				`
                  struct S {}

                  event Test(_ value: %[1]s)

                  fun test() {
                      emit Test(%[2]s as %[1]s)
                  }
                `,
				ty,
				value,
			)

			inter := parseCheckAndInterpret(t, code)

			var actualEvents []*interpreter.CompositeValue

			inter.SetOnEventEmittedHandler(
				func(_ *interpreter.Interpreter, event *interpreter.CompositeValue, eventType *sema.CompositeType) error {
					actualEvents = append(actualEvents, event)
					return nil
				},
			)

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			testType := checker.RequireGlobalType(t, inter.Program.Elaboration, "Test")

			members := interpreter.NewStringValueOrderedMap()
			members.Set("value", value.value)

			expectedEvents := []*interpreter.CompositeValue{
				interpreter.NewCompositeValue(
					TestLocation,
					TestLocation.QualifiedIdentifier(testType.ID()),
					common.CompositeKindEvent,
					members,
					nil,
				),
			}

			for _, event := range expectedEvents {
				event.InitializeFunctions(inter)
			}

			AssertEqualWithDiff(t,
				expectedEvents,
				actualEvents,
			)
		})
	}
}

func TestInterpretSwapResourceDictionaryElementReturnSwapped(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @X? {
          let xs: @{String: X} <- {}
          var x: @X? <- create X()
          xs["foo"] <-> x
          destroy xs
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NilValue{},
		value,
	)
}

func TestInterpretSwapResourceDictionaryElementReturnDictionary(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @{String: X} {
          let xs: @{String: X} <- {}
          var x: @X? <- create X()
          xs["foo"] <-> x
          destroy x
          return <-xs
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.DictionaryValue{},
		value,
	)

	foo := value.(*interpreter.DictionaryValue).
		Get(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewStringValue("foo"))

	require.IsType(t,
		&interpreter.SomeValue{},
		foo,
	)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		foo.(*interpreter.SomeValue).Value,
	)
}

func TestInterpretSwapResourceDictionaryElementRemoveUsingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource X {}

      fun test(): @X? {
          let xs: @{String: X} <- {"foo": <-create X()}
          var x: @X? <- nil
          xs["foo"] <-> x
          destroy xs
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.SomeValue{},
		value,
	)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value.(*interpreter.SomeValue).Value,
	)
}

func TestInterpretReferenceExpression(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      pub resource R {}

      pub fun test(): &R {
          let r <- create R()
          let ref = &r as &R
          destroy r
          return ref
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.EphemeralReferenceValue{},
		value,
	)
}

func TestInterpretReferenceUse(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      pub resource R {
          pub(set) var x: Int

          init() {
              self.x = 0
          }

          pub fun setX(_ newX: Int) {
              self.x = newX
          }
      }

      pub fun test(): [Int] {
          let r <- create R()

          let ref1 = &r as &R
          let ref2 = &r as &R

          ref1.x = 1
          let x1 = ref1.x
          ref1.setX(2)
          let x2 = ref1.x

          let x3 = ref2.x
          let res = [x1, x2, x3]
          destroy r
          return res
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
			interpreter.NewIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretReferenceUseAccess(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      pub resource R {
          pub(set) var x: Int

          init() {
              self.x = 0
          }

          pub fun setX(_ newX: Int) {
              self.x = newX
          }
      }

      pub fun test(): [Int] {
          let rs <- [<-create R()]
          let ref = &rs as &[R]
          let x0 = ref[0].x
          ref[0].x = 1
          let x1 = ref[0].x
          ref[0].setX(2)
          let x2 = ref[0].x
          let res = [x0, x1, x2]
          destroy rs
          return res
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(0),
			interpreter.NewIntValueFromInt64(1),
			interpreter.NewIntValueFromInt64(2),
		),
		value,
	)
}

func TestInterpretReferenceDereferenceFailure(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      pub resource R {
          pub fun foo() {}
      }

      pub fun test() {
          let r <- create R()
          let ref = &r as &R
          destroy r
          ref.foo()
      }
    `)

	_, err := inter.Invoke("test")

	require.ErrorAs(t, err, &interpreter.DestroyedCompositeError{})
}

func TestInterpretInvalidForwardReferenceCall(t *testing.T) {

	t.Parallel()

	// TODO: improve:
	//   - call to `g` should succeed, but access to `y` should fail with error
	//   - maybe make this a static error

	assert.Panics(t, func() {
		_ = parseCheckAndInterpret(t, `
          fun f(): Int {
             return g()
          }

          let x = f()
          let y = 0

          fun g(): Int {
              return y
          }
        `)
	})
}

func TestInterpretVariableDeclarationSecondValue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {
          let id: Int
          init(id: Int) {
              self.id = id
          }
      }

      fun test(): @[R?] {
          let x <- create R(id: 1)
          var ys <- {"r": <-create R(id: 2)}
          // NOTE: nested move is valid here
          let z <- ys["r"] <- x

          // NOTE: nested move is invalid here
          let r <- ys.remove(key: "r")

          destroy ys

          return <-[<-z, <-r]
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t,
		&interpreter.ArrayValue{},
		value,
	)

	values := value.(*interpreter.ArrayValue).Elements()

	require.IsType(t,
		&interpreter.SomeValue{},
		values[0],
	)

	firstValue := values[0].(*interpreter.SomeValue).Value

	require.IsType(t,
		&interpreter.CompositeValue{},
		firstValue,
	)

	firstResource := firstValue.(*interpreter.CompositeValue)

	assert.Equal(t,
		firstResource.GetField("id"),
		interpreter.NewIntValueFromInt64(2),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		values[1],
	)

	secondValue := values[1].(*interpreter.SomeValue).Value

	require.IsType(t,
		&interpreter.CompositeValue{},
		secondValue,
	)

	secondResource := secondValue.(*interpreter.CompositeValue)

	assert.Equal(t,
		secondResource.GetField("id"),
		interpreter.NewIntValueFromInt64(1),
	)
}

func TestInterpretCastingIntLiteralToInt8(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int8
    `)

	assert.Equal(t,
		interpreter.Int8Value(42),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCastingIntLiteralToAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as AnyStruct
    `)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCastingIntLiteralToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int?
    `)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(interpreter.NewIntValueFromInt64(42)),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCastingResourceToAnyResource(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}

      fun test(): @AnyResource {
          let r <- create R()
          let x <- r as @AnyResource
          return <-x
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)
}

func TestInterpretOptionalChainingFieldRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct Test {
              let x: Int

              init(x: Int) {
                  self.x = x
              }
          }

          let test1: Test? = nil
          let x1 = test1?.x

          let test2: Test? = Test(x: 42)
          let x2 = test2?.x
        `,
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["x2"].GetValue(),
	)
}

func TestInterpretOptionalChainingFunctionRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct Test {
              fun x(): Int {
                  return 42
              }
          }

          let test1: Test? = nil
          let x1 = test1?.x

          let test2: Test? = Test()
          let x2 = test2?.x
        `,
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		inter.Globals["x2"].GetValue(),
	)

	assert.IsType(t,
		interpreter.BoundFunctionValue{},
		inter.Globals["x2"].GetValue().(*interpreter.SomeValue).Value,
	)
}

func TestInterpretOptionalChainingFunctionCall(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
         struct Test {
             fun x(): Int {
                 return 42
             }
         }

         let test1: Test? = nil
         let x1 = test1?.x()

         let test2: Test? = Test()
         let x2 = test2?.x()
       `,
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewSomeValueOwningNonCopying(
			interpreter.NewIntValueFromInt64(42),
		),
		inter.Globals["x2"].GetValue(),
	)
}

func TestInterpretOptionalChainingFieldReadAndNilCoalescing(t *testing.T) {

	t.Parallel()

	standardLibraryFunctions :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}

	valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
	values := standardLibraryFunctions.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          struct Test {
              let x: Int

              init(x: Int) {
                  self.x = x
              }
          }

          let test: Test? = Test(x: 42)
          let x = test?.x ?? panic("nil")
        `,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(values),
			},
		},
	)

	assert.Equal(t,
		inter.Globals["x"].GetValue(),
		interpreter.NewIntValueFromInt64(42),
	)
}

func TestInterpretOptionalChainingFunctionCallAndNilCoalescing(t *testing.T) {

	t.Parallel()

	standardLibraryFunctions :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}

	valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
	values := standardLibraryFunctions.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          struct Test {
              fun x(): Int {
                  return 42
              }
          }

          let test: Test? = Test()
          let x = test?.x() ?? panic("nil")
        `,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(values),
			},
		},
	)

	assert.Equal(t,
		inter.Globals["x"].GetValue(),
		interpreter.NewIntValueFromInt64(42),
	)
}

func TestInterpretCompositeDeclarationNestedTypeScopingOuterInner(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          pub contract Test {

              pub struct X {

                  pub fun test(): X {
                     return Test.x()
                  }
              }

              pub fun x(): X {
                 return X()
              }
          }

          pub let x1 = Test.x()
          pub let x2 = x1.test()
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	x1 := inter.Globals["x1"].GetValue()
	x2 := inter.Globals["x2"].GetValue()

	require.IsType(t,
		&interpreter.CompositeValue{},
		x1,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x1.(*interpreter.CompositeValue).TypeID(),
	)

	require.IsType(t,
		&interpreter.CompositeValue{},
		x2,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x2.(*interpreter.CompositeValue).TypeID(),
	)
}

func TestInterpretCompositeDeclarationNestedConstructor(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          pub contract Test {

              pub struct X {}
          }

          pub let x = Test.X()
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	x := inter.Globals["x"].GetValue()

	require.IsType(t,
		&interpreter.CompositeValue{},
		x,
	)

	assert.Equal(t,
		sema.TypeID("S.test.Test.X"),
		x.(*interpreter.CompositeValue).TypeID(),
	)
}

func TestInterpretFungibleTokenContract(t *testing.T) {

	t.Parallel()

	code := strings.Join(
		[]string{
			examples.FungibleTokenContractInterface,
			examples.ExampleFungibleTokenContract,
			`
              pub fun test(): [Int; 2] {

                  let publisher <- ExampleToken.sprout(balance: 100)
                  let receiver <- ExampleToken.sprout(balance: 0)

                  let withdrawn <- publisher.withdraw(amount: 60)
                  receiver.deposit(vault: <-withdrawn)

                  let publisherBalance = publisher.balance
                  let receiverBalance = receiver.balance

                  destroy publisher
                  destroy receiver

                  return [publisherBalance, receiverBalance]
              }
            `,
		},
		"\n",
	)

	standardLibraryFunctions :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}

	valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
	values := standardLibraryFunctions.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(values),
				makeContractValueHandler(nil, nil, nil),
			},
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
		},
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewIntValueFromInt64(40),
			interpreter.NewIntValueFromInt64(60),
		),
		value,
	)
}

func TestInterpretContractAccountFieldUse(t *testing.T) {

	t.Parallel()

	code := `
      pub contract Test {
          pub let address: Address

          init() {
              // field 'account' can be used, as it is considered initialized
              self.address = self.account.address
          }

          pub fun test(): Address {
              return self.account.address
          }
      }

      pub let address1 = Test.address
      pub let address2 = Test.test()
    `

	addressValue := interpreter.AddressValue{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	inter := parseCheckAndInterpretWithOptions(t, code,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
				interpreter.WithInjectedCompositeFieldsHandler(
					func(
						_ *interpreter.Interpreter,
						_ common.Location,
						_ string,
						_ common.CompositeKind,
					) *interpreter.StringValueOrderedMap {

						panicFunction := interpreter.NewHostFunctionValue(
							func(invocation interpreter.Invocation) interpreter.Value {
								panic(errors.NewUnreachableError())
							},
						)

						injectedMembers := interpreter.NewStringValueOrderedMap()
						injectedMembers.Set(
							"account",
							interpreter.NewAuthAccountValue(
								addressValue,
								returnZeroUFix64,
								returnZeroUFix64,
								func(interpreter *interpreter.Interpreter) interpreter.UInt64Value {
									return 0
								},
								returnZeroUInt64,
								panicFunction,
								panicFunction,
								&interpreter.CompositeValue{},
								&interpreter.CompositeValue{},
							),
						)
						return injectedMembers
					},
				),
			},
		},
	)

	assert.Equal(t,
		addressValue,
		inter.Globals["address1"].GetValue(),
	)

	assert.Equal(t,
		addressValue,
		inter.Globals["address2"].GetValue(),
	)
}

func TestInterpretConformToImportedInterface(t *testing.T) {

	t.Parallel()

	importedChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          struct interface Foo {
              fun check(answer: Int) {
                  pre {
                      answer == 42
                  }
              }
          }
	    `,
		checker.ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import Foo from "imported"

          struct Bar: Foo {
              fun check(answer: Int) {}
          }

          fun test() {
              let bar = Bar()
              bar.check(answer: 1)
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						assert.Equal(t,
							ImportedLocation,
							importedLocation,
						)

						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)
	require.NoError(t, err)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				assert.Equal(t,
					ImportedLocation,
					location,
				)

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretContractUseInNestedDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t, `
          pub contract C {

              pub var i: Int

              pub struct S {

                  init() {
                      C.i = C.i + 1
                  }
              }

              init () {
                  self.i = 0
                  S()
                  S()
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	i := inter.Globals["C"].GetValue().(interpreter.MemberAccessibleValue).
		GetMember(inter, interpreter.ReturnEmptyLocationRange, "i")

	require.IsType(t,
		interpreter.NewIntValueFromInt64(2),
		i,
	)
}

func TestInterpretNonStorageReference(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          resource NFT {
              var id: Int

              init(id: Int) {
                  self.id = id
              }
          }

          fun test(): Int {
              let resources <- [
                  <-create NFT(id: 1),
                  <-create NFT(id: 2)
              ]

              let nftRef = &resources[1] as &NFT
              let nftRef2 = nftRef
              nftRef2.id = 3

              let nft <- resources.remove(at: 1)
              destroy resources
              let newID = nft.id
              destroy nft

              return newID
          }
        `,
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t, interpreter.NewIntValueFromInt64(3), value)
}

func TestInterpretNonStorageReferenceAfterDestruction(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          resource NFT {
              var id: Int

              init(id: Int) {
                  self.id = id
              }
          }

          fun test(): Int {
              let nft <- create NFT(id: 1)
              let nftRef = &nft as &NFT
              destroy nft
              return nftRef.id
          }
        `,
	)

	_, err := inter.Invoke("test")
	require.Error(t, err)

	require.ErrorAs(t, err, &interpreter.DestroyedCompositeError{})
}

func TestInterpretNonStorageReferenceToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          resource Foo {
              let name: String

              init(name: String) {
                  self.name = name
              }
          }


          fun testSome(): String {
              let xs: @{String: Foo} <- {"yes": <-create Foo(name: "YES")}
              let ref = &xs["yes"] as &Foo
              let name = ref.name
              destroy xs
              return name
          }

          fun testNil(): String {
              let xs: @{String: Foo} <- {}
              let ref = &xs["no"] as &Foo
              let name = ref.name
              destroy xs
              return name
          }
        `,
	)

	t.Run("some", func(t *testing.T) {
		value, err := inter.Invoke("testSome")
		require.NoError(t, err)

		assert.Equal(t, interpreter.NewStringValue("YES"), value)
	})

	t.Run("nil", func(t *testing.T) {
		_, err := inter.Invoke("testNil")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})
}

func TestInterpretFix64(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let a = 789.00123010
          let b = 1234.056
          let c = -12345.006789
        `,
	)

	assert.Equal(t,
		interpreter.UFix64Value(78_900_123_010),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.UFix64Value(123_405_600_000),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.Fix64Value(-1_234_500_678_900),
		inter.Globals["c"].GetValue(),
	)
}

func TestInterpretFix64Mul(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let a = Fix64(1.1) * -1.1
        `,
	)

	assert.Equal(t,
		interpreter.Fix64Value(-121000000),
		inter.Globals["a"].GetValue(),
	)
}

func TestInterpretHexDecode(t *testing.T) {

	t.Parallel()

	expected := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.UInt8Value(71),
		interpreter.UInt8Value(111),
		interpreter.UInt8Value(32),
		interpreter.UInt8Value(87),
		interpreter.UInt8Value(105),
		interpreter.UInt8Value(116),
		interpreter.UInt8Value(104),
		interpreter.UInt8Value(32),
		interpreter.UInt8Value(116),
		interpreter.UInt8Value(104),
		interpreter.UInt8Value(101),
		interpreter.UInt8Value(32),
		interpreter.UInt8Value(70),
		interpreter.UInt8Value(108),
		interpreter.UInt8Value(111),
		interpreter.UInt8Value(119),
	)

	t.Run("in Cadence", func(t *testing.T) {

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.PanicFunction,
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter := parseCheckAndInterpretWithOptions(t,
			`
              fun hexDecode(_ s: String): [UInt8] {
                  if s.length % 2 != 0 {
                      panic("Input must have even number of characters")
                  }
                  let table: {String: UInt8} = {
                          "0" : 0 as UInt8,
                          "1" : 1 as UInt8,
                          "2" : 2 as UInt8,
                          "3" : 3 as UInt8,
                          "4" : 4 as UInt8,
                          "5" : 5 as UInt8,
                          "6" : 6 as UInt8,
                          "7" : 7 as UInt8,
                          "8" : 8 as UInt8,
                          "9" : 9 as UInt8,
                          "a" : 10 as UInt8,
                          "A" : 10 as UInt8,
                          "b" : 11 as UInt8,
                          "B" : 11 as UInt8,
                          "c" : 12 as UInt8,
                          "C" : 12 as UInt8,
                          "d" : 13 as UInt8,
                          "D" : 13 as UInt8,
                          "e" : 14 as UInt8,
                          "E" : 14 as UInt8,
                          "f" : 15 as UInt8,
                          "F" : 15 as UInt8
                      }
                  let length = s.length / 2
                  var i = 0
                  var res: [UInt8] = []
                  while i < length {
                      let c = s.slice(from: i * 2, upTo: i * 2 + 1)
                      let in = table[c] ?? panic("Invalid character ".concat(c))
                      let c2 = s.slice(from: i * 2 + 1, upTo: i * 2 + 2)
                      let in2 = table[c2] ?? panic("Invalid character ".concat(c2))
                      res.append((16 as UInt8) * in + in2)
                      i = i + 1
                  }
                  return res
              }

              fun test(): [UInt8] {
                  return hexDecode("476F20576974682074686520466C6F77")
              }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues(values),
				},
			},
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		assert.Equal(t, expected, result)
	})

	t.Run("native", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
              fun test(): [UInt8] {
                  return "476F20576974682074686520466C6F77".decodeHex()
              }
            `,
		)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		assert.Equal(t, expected, result)
	})

}

func TestInterpretOptionalChainingOptionalFieldRead(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct Test {
          let x: Int?

          init(x: Int?) {
              self.x = x
          }
      }

      let test: Test? = Test(x: 1)
      let x = test?.x
    `)

	assert.Equal(t,
		&interpreter.SomeValue{
			Value: interpreter.NewIntValueFromInt64(1),
		},
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretResourceOwnerFieldUse(t *testing.T) {

	t.Parallel()

	storedValues := map[string]interpreter.OptionalValue{}

	// NOTE: Getter and Setter are very naive for testing purposes and don't remove nil values
	//

	checker := func(_ *interpreter.Interpreter, _ common.Address, key string) bool {
		_, ok := storedValues[key]
		return ok
	}

	getter := func(_ *interpreter.Interpreter, _ common.Address, key string, _ bool) interpreter.OptionalValue {
		value, ok := storedValues[key]
		if !ok {
			return interpreter.NilValue{}
		}
		return value
	}

	setter := func(_ *interpreter.Interpreter, _ common.Address, key string, value interpreter.OptionalValue) {
		storedValues[key] = value
	}

	address := common.Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	addressValue := interpreter.AddressValue(address)

	code := `
      pub resource R {}

      pub fun test(): [Address?] {
          let addresses: [Address?] = []

          let r <- create R()
          addresses.append(r.owner?.address)

          account.save(<-r, to: /storage/r)

          let ref = account.borrow<&R>(from: /storage/r)
          addresses.append(ref?.owner?.address)

          return addresses
      }
    `

	panicFunction := interpreter.NewHostFunctionValue(func(invocation interpreter.Invocation) interpreter.Value {
		panic(errors.NewUnreachableError())
	})

	// `authAccount`

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "account",
		Type: sema.AuthAccountType,
		Value: interpreter.NewAuthAccountValue(
			addressValue,
			returnZeroUFix64,
			returnZeroUFix64,
			func(interpreter *interpreter.Interpreter) interpreter.UInt64Value {
				return 0
			},
			returnZeroUInt64,
			panicFunction,
			panicFunction,
			&interpreter.CompositeValue{},
			&interpreter.CompositeValue{},
		),
		Kind: common.DeclarationKindConstant,
	}

	inter := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues([]sema.ValueDeclaration{
					valueDeclaration,
				}),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues([]interpreter.ValueDeclaration{
					valueDeclaration,
				}),
				interpreter.WithStorageExistenceHandler(checker),
				interpreter.WithStorageReadHandler(getter),
				interpreter.WithStorageWriteHandler(setter),
				interpreter.WithAccountHandlerFunc(
					func(address interpreter.AddressValue) *interpreter.CompositeValue {
						return interpreter.NewPublicAccountValue(
							address,
							returnZeroUFix64,
							returnZeroUFix64,
							func(interpreter *interpreter.Interpreter) interpreter.UInt64Value {
								panic(errors.NewUnreachableError())
							},
							func() interpreter.UInt64Value {
								panic(errors.NewUnreachableError())
							},
							interpreter.NewPublicAccountKeysValue(
								panicFunction,
							),
						)
					},
				),
			},
		},
	)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NilValue{},
			interpreter.NewSomeValueOwningNonCopying(interpreter.AddressValue(address)),
		),
		result,
	)
}

func TestInterpretResourceAssignmentForceTransfer(t *testing.T) {

	t.Parallel()

	t.Run("new to nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          resource X {}

          fun test() {
              var x: @X? <- nil
              x <-! create X()
              destroy x
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("new to non-nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
	     resource X {}

	     fun test() {
	         var x: @X? <- create X()
	         x <-! create X()
	         destroy x
	     }
	   `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.ForceAssignmentToNonNilResourceError{})
	})

	t.Run("existing to nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
	     resource X {}

	     fun test() {
	         let x <- create X()
	         var x2: @X? <- nil
	         x2 <-! x
	         destroy x2
	     }
	   `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("existing to non-nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
	     resource X {}

	     fun test() {
	         let x <- create X()
	         var x2: @X? <- create X()
	         x2 <-! x
	         destroy x2
	     }
	   `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.ForceAssignmentToNonNilResourceError{})
	})

	t.Run("force-assignment initialization", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
	     resource X {}

	     resource Y {

             var x: @X?

             init() {
                 self.x <-! create X()
             }

             destroy() {
                 destroy self.x
             }
         }

	     fun test() {
	         let y <- create Y()
	         destroy y
	     }
	   `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

}

func TestInterpretForce(t *testing.T) {

	t.Parallel()

	t.Run("non-nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          let x: Int? = 1
          let y = x!
        `)

		assert.Equal(t,
			interpreter.NewSomeValueOwningNonCopying(
				interpreter.NewIntValueFromInt64(1),
			),
			inter.Globals["x"].GetValue(),
		)

		assert.Equal(t,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["y"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          let x: Int? = nil

          fun test(): Int {
              return x!
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.ForceNilError{})
	})

	t.Run("non-optional", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          let x: Int = 1
          let y = x!
        `)

		assert.Equal(t,
			interpreter.NewIntValueFromInt64(1),
			inter.Globals["y"].GetValue(),
		)
	})
}

func permutations(xs [][]byte) (res [][][]byte) {
	var f func([][]byte, int)
	f = func(a [][]byte, k int) {
		if k == len(a) {
			res = append(res, append([][]byte{}, a...))
		} else {
			for i := k; i < len(xs); i++ {
				a[k], a[i] = a[i], a[k]
				f(a, k+1)
				a[k], a[i] = a[i], a[k]
			}
		}
	}

	f(xs, 0)

	return res
}

func TestInterpretCompositeValueFieldEncodingOrder(t *testing.T) {

	t.Parallel()

	fieldValues := map[byte]byte{
		'a': 1,
		'b': 2,
		'c': 3,
	}

	// prepare initialization statements

	initializations := make([][]byte, 0, len(fieldValues))
	expectedEncodings := make([][]byte, 0, len(fieldValues))

	for name, value := range fieldValues {
		initialization := fmt.Sprintf("self.%c = %d", name, value)
		initializations = append(initializations, []byte(initialization))

		expectedEncodings = append(expectedEncodings, []byte{
			// UTF-8 string, length 1
			0x61,
			name,
			// tag
			0xD8, 0x98,
			// - positive bignum
			0xc2,
			// - byte string, length 1
			0x41,
			value,
		})
	}

	allInitializations := permutations(initializations)
	allExpectedEncodings := permutations(expectedEncodings)

	expectedPrefix := []byte{
		// tag
		0xd8, 0x84,
		// array, 5 items follow
		0x85,

		// tag
		0xd8, 0xc1,
		// UTF-8 string, length 4
		0x64,
		// t, e, s, t
		0x74, 0x65, 0x73, 0x74,

		// nil
		0xf6,

		// positive integer 1
		0x1,

		// array, 6 items follow
		0x86,
	}

	expectedSuffix := []byte{
		// UTF-8 string, length 4
		0x64,
		0x54, 0x65, 0x73, 0x74,
	}

	for i, initialization := range allInitializations {

		var codeBuilder strings.Builder
		codeBuilder.WriteString(`
          struct Test {
              let a: Int
              let b: Int
              let c: Int

              init() {
        `)

		for _, statement := range initialization {
			codeBuilder.Write(statement)
			codeBuilder.WriteRune('\n')
		}

		codeBuilder.WriteString(`
              }
          }

          let test = Test()
        `)

		inter := parseCheckAndInterpret(t, codeBuilder.String())

		test := inter.Globals["test"].GetValue().(*interpreter.CompositeValue)

		test.SetOwner(&common.Address{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
		})

		encoded, _, err := interpreter.EncodeValue(test, nil, false, nil)
		require.NoError(t, err)

		expected := expectedPrefix[:]

		for _, expectedEncoding := range allExpectedEncodings[i] {
			expected = append(expected, expectedEncoding...)
		}

		expected = append(expected, expectedSuffix...)

		assert.Equal(t, expected, encoded)
	}
}

func TestInterpretDictionaryValueEncodingOrder(t *testing.T) {

	t.Parallel()

	fieldValues := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	initializations := make([][]byte, 0, len(fieldValues))

	for name, value := range fieldValues {
		initialization := fmt.Sprintf(`xs["%s"] = %d`, name, value)
		initializations = append(initializations, []byte(initialization))
	}

	for _, initialization := range permutations(initializations) {

		var codeBuilder strings.Builder
		codeBuilder.WriteString(`
          fun construct(): {String: Int} {
              let xs: {String: Int} = {}
        `)

		for _, statement := range initialization {
			codeBuilder.Write(statement)
			codeBuilder.WriteRune('\n')
		}

		codeBuilder.WriteString(`
              return xs
          }

          let test = construct()
        `)

		inter := parseCheckAndInterpret(t, codeBuilder.String())

		test := inter.Globals["test"].GetValue().(*interpreter.DictionaryValue)

		owner := &common.Address{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
		}

		test.SetOwner(owner)

		var path []string = nil
		encoded, _, err := interpreter.EncodeValue(test, path, false, nil)
		require.NoError(t, err)

		decoder, err := interpreter.NewDecoder(
			bytes.NewReader(encoded),
			owner,
			interpreter.CurrentEncodingVersion,
			nil,
		)
		require.NoError(t, err)

		decoded, err := decoder.Decode(path)
		require.NoError(t, err)

		test.SetModified(false)
		test.Keys.SetModified(false)
		for _, key := range test.Keys.Elements() {
			stringKey := key.(*interpreter.StringValue)
			stringKey.SetModified(false)
		}

		require.Equal(t, test, decoded)
	}
}

func TestInterpretEphemeralReferenceToOptional(t *testing.T) {

	t.Parallel()

	_ = parseCheckAndInterpretWithOptions(t,
		`
          contract C {

              var rs: @{Int: R}

              resource R {
                  pub let id: Int

                  init(id: Int) {
                      self.id = id
                  }
              }

              fun borrow(id: Int): &R {
                  return &C.rs[id] as &R
              }

              init() {
                  self.rs <- {}
                  self.rs[1] <-! create R(id: 1)
                  let ref = self.borrow(id: 1)
                  ref.id
              }
          }
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)
}

func TestInterpretNestedDeclarationOrder(t *testing.T) {

	t.Parallel()

	t.Run("A, B", func(t *testing.T) {
		_ = parseCheckAndInterpretWithOptions(t,
			`
              pub contract Test {

                  pub resource A {

                      pub fun b(): @B {
                          return <-create B()
                      }
                  }

                  pub resource B {}

                  init() {
                      let a <- create A()
                      let b <- a.b()
                      destroy a
                      destroy b
                  }
              }
            `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					makeContractValueHandler(nil, nil, nil),
				},
			},
		)
	})

	t.Run("B, A", func(t *testing.T) {

		_ = parseCheckAndInterpretWithOptions(t,
			`
              pub contract Test {

                  pub resource B {}

                  pub resource A {

                      pub fun b(): @B {
                          return <-create B()
                      }
                  }

                  init() {
                      let a <- create A()
                      let b <- a.b()
                      destroy a
                      destroy b
                  }
              }
            `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					makeContractValueHandler(nil, nil, nil),
				},
			},
		)
	})
}

func TestInterpretCountDigits256(t *testing.T) {

	t.Parallel()

	type test struct {
		Type    sema.Type
		Literal string
		Count   int
	}

	for _, test := range []test{
		{
			sema.Int256Type,
			"676983016644359394637212096269997871684197836659065544033845082275068334",
			72,
		},
		{
			sema.UInt256Type,
			"676983016644359394637212096269997871684197836659065544033845082275068334",
			72,
		},
		{
			sema.Int128Type,
			"676983016644359394637212096269997871",
			36,
		},
		{
			sema.UInt128Type,
			"676983016644359394637212096269997871",
			36,
		},
	} {

		t.Run(test.Type.String(), func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun countDigits(_ x: %[2]s): UInt8 {
                          var count: UInt8 = UInt8(0)
                          var input = x
                          while input != %[2]s(0) {
                              count = count + UInt8(1)
                              input = input / %[2]s(10)
                          }
                          return count
                      }

                      let number: %[2]s = %[1]s
                      let result1 = countDigits(%[1]s)
                      let result2 = countDigits(%[2]s(%[1]s))
                      let result3 = countDigits(number)
                    `,
					test.Literal,
					test.Type,
				),
			)

			bigInt, ok := new(big.Int).SetString(test.Literal, 10)
			require.True(t, ok)

			assert.Equal(t,
				bigInt,
				inter.Globals["number"].GetValue().(interpreter.BigNumberValue).ToBigInt(),
			)

			expected := interpreter.UInt8Value(test.Count)

			for i := 1; i <= 3; i++ {
				variableName := fmt.Sprintf("result%d", i)
				assert.Equal(t,
					expected,
					inter.Globals[variableName].GetValue(),
				)
			}
		})
	}
}

func TestInterpretFailableCastingCompositeTypeConfusion(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          contract A {
              struct S {}
          }

          contract B {
              struct S {}
          }

          let s = A.S() as? B.S
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["s"].GetValue(),
	)
}

func TestInterpretNestedDestroy(t *testing.T) {

	var logs []string

	logFunction := stdlib.NewStandardLibraryFunction(
		"log",
		&sema.FunctionType{
			Parameters: []*sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
				},
			},
			ReturnTypeAnnotation: sema.NewTypeAnnotation(
				sema.VoidType,
			),
		},
		func(invocation interpreter.Invocation) interpreter.Value {
			message := invocation.Arguments[0].String(interpreter.StringResults{})
			logs = append(logs, message)
			return interpreter.VoidValue{}
		},
	)

	valueDeclarations :=
		stdlib.StandardLibraryFunctions{
			logFunction,
		}.ToSemaValueDeclarations()

	values := stdlib.StandardLibraryFunctions{
		logFunction,
	}.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          resource B {
              let id: Int

              init(_ id: Int){
                  self.id = id
              }

              destroy(){
                  log("destroying B with id:")
                  log(self.id)
              }
          }

          resource A {
              let id: Int
              let bs: @[B]

              init(_ id: Int){
                  self.id = id
                  self.bs <- []
              }

              fun add(_ b: @B){
                  self.bs.append(<-b)
              }

              destroy() {
                  log("destroying A with id:")
                  log(self.id)
                  destroy self.bs
              }
          }

          fun test() {
              let a <- create A(1)
              a.add(<- create B(2))
              a.add(<- create B(3))
              a.add(<- create B(4))

              destroy a
          }
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(values),
			},
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
			HandleCheckerError: nil,
		},
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)

	assert.Equal(t,
		[]string{
			`"destroying A with id:"`,
			"1",
			`"destroying B with id:"`,
			"2",
			`"destroying B with id:"`,
			"3",
			`"destroying B with id:"`,
			"4",
		},
		logs,
	)
}

// TestInterpretInternalAssignment ensures that a modification of an "internal" value
// is not possible, because the value that is assigned into is a copy
//
func TestInterpretInternalAssignment(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       struct S {
           priv let xs: {String: Int}

           init() {
               self.xs = {"a": 1}
           }

           fun getXS(): {String: Int} {
               return self.xs
           }
       }

       fun test(): [{String: Int}] {
           let s = S()
           let xs = s.getXS()
           xs["b"] = 2
           return [xs, s.getXS()]
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewArrayValueUnownedNonCopying(
			interpreter.NewDictionaryValueUnownedNonCopying(
				interpreter.NewStringValue("a"),
				interpreter.NewIntValueFromInt64(1),
				interpreter.NewStringValue("b"),
				interpreter.NewIntValueFromInt64(2),
			),
			interpreter.NewDictionaryValueUnownedNonCopying(
				interpreter.NewStringValue("a"),
				interpreter.NewIntValueFromInt64(1),
			),
		),
		value,
	)
}

func TestInterpretCopyOnReturn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          let xs: {String: String} = {}

          fun returnXS(): {String: String} {
              return xs
          }

          fun test(): {String: String} {
              returnXS().insert(key: "foo", "bar")
              return xs
          }
        `,
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewDictionaryValueUnownedNonCopying(),
		value,
	)
}

func BenchmarkInterpretRecursionFib(b *testing.B) {

	inter := parseCheckAndInterpret(b, `
       fun fib(_ n: Int): Int {
           if n < 2 {
              return n
           }
           return fib(n - 1) + fib(n - 2)
       }
   `)

	expected := interpreter.NewIntValueFromInt64(377)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		result, err := inter.Invoke(
			"fib",
			interpreter.NewIntValueFromInt64(14),
		)
		require.NoError(b, err)
		require.Equal(b, expected, result)
	}
}

func TestInterpretMissingMember(t *testing.T) {

	// prepare type `struct X { let y: Int }`

	const typeName = "X"
	ty := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: typeName,
	}

	members := sema.NewStringMemberOrderedMap()
	ty.Members = members

	const fieldName = "y"
	fieldMember := sema.NewPublicConstantFieldMember(
		ty,
		fieldName,
		sema.IntType,
		"",
	)
	members.Set(fieldName, fieldMember)

	// prepare value of type X,
	// which is missing field `y`!

	value := interpreter.NewCompositeValue(
		TestLocation,
		typeName,
		common.CompositeKindStructure,
		interpreter.NewStringValueOrderedMap(),
		nil,
	)

	predeclaredValues :=
		stdlib.StandardLibraryValues{
			{
				Name:  "x",
				Type:  ty,
				Value: value,
				Kind:  common.DeclarationKindStructure,
			},
		}

	valueDeclarations := predeclaredValues.ToSemaValueDeclarations()
	values := predeclaredValues.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          fun test() {
              // access missing field y
              x.y
          }
        `,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(values),
			},
		},
	)

	_, err := inter.Invoke("test")
	require.Error(t, err)

	var missingMemberError interpreter.MissingMemberValueError
	require.ErrorAs(t, err, &missingMemberError)

	require.Equal(t, "y", missingMemberError.Name)
}
