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

package interpreter_test

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
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
	inter, err := parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{})
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMetering(t, code, options, nil)
}

func parseCheckAndInterpretWithMemoryMetering(
	t testing.TB,
	code string,
	memoryGauge common.MemoryGauge,
) *interpreter.Interpreter {
	inter, err := parseCheckAndInterpretWithOptionsAndMemoryMetering(
		t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.StandardLibraryFunctions{
						stdlib.PanicFunction,
					}.ToSemaValueDeclarations(),
				),
			},
		},
		memoryGauge,
	)
	require.NoError(t, err)
	return inter
}

func parseCheckAndInterpretWithOptionsAndMemoryMetering(
	t testing.TB,
	code string,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	checker, err := checker.ParseAndCheckWithOptionsAndMemoryMetering(t,
		code,
		checker.ParseAndCheckOptions{
			Options: options.CheckerOptions,
		},
		memoryGauge,
	)

	if options.HandleCheckerError != nil {
		options.HandleCheckerError(err)
	} else if !assert.NoError(t, err) {
		var sb strings.Builder
		location := checker.Location
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, location, map[common.Location]string{location: code})
		if printErr != nil {
			panic(printErr)
		}
		assert.Fail(t, sb.String())
		return nil, err
	}

	var uuid uint64 = 0

	interpreterOptions := append(
		[]interpreter.Option{
			interpreter.WithUUIDHandler(func() (uint64, error) {
				uuid++
				return uuid, nil
			}),
			interpreter.WithStorage(interpreter.NewInMemoryStorage(memoryGauge)),
			interpreter.WithAtreeValueValidationEnabled(true),
			interpreter.WithAtreeStorageValidationEnabled(true),
			interpreter.WithOnRecordTraceHandler(
				func(
					_ *interpreter.Interpreter,
					_ string,
					_ time.Duration,
					_ []attribute.KeyValue,
				) {
					// NO-OP
				},
			),
			interpreter.WithTracingEnabled(true),
		},
		options.Options...,
	)

	if memoryGauge != nil {
		interpreterOptions = append(
			interpreterOptions,
			interpreter.WithMemoryGauge(memoryGauge),
		)
	}

	inter, err = interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreterOptions...,
	)

	require.NoError(t, err)

	err = inter.Interpret()

	if err == nil {

		// recover internal panics and return them as an error
		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		// Contract declarations are evaluated lazily,
		// so force the contract value handler to be called

		for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
			if compositeDeclaration.CompositeKind != common.CompositeKindContract {
				continue
			}

			contractVariable := inter.Globals[compositeDeclaration.Identifier.Identifier]

			_ = contractVariable.GetValue()
		}
	}

	return inter, err
}

func newUnmeteredInMemoryStorage() interpreter.InMemoryStorage {
	return interpreter.NewInMemoryStorage(nil)
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
			constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
			invocationRange ast.Range,
		) *interpreter.CompositeValue {

			constructor := constructorGenerator(common.Address{})

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["y"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals["z"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals["b"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("123"),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("f")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)

	value, err = inter.Invoke("g")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
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

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

	a := interpreter.NewUnmeteredIntValueFromInt64(24)
	b := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("returnA", a, b)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, a, value)

	value, err = inter.Invoke("returnB", a, b)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, b, value)
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
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

	expectedArray := interpreter.NewArrayValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		},
		common.Address{},
		interpreter.NewUnmeteredIntValueFromInt64(0),
		interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	RequireValuesEqual(
		t,
		inter,
		expectedArray,
		actualArray,
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

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("a"),
		inter.Globals["x"].GetValue(),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("b"),
		inter.Globals["y"].GetValue(),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("c"),
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

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("\u00e9"),
		value,
	)

	value, err = inter.Invoke("testUnicodeB")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredCharacterValue("e\u0301"),
		value,
	)
}

func TestInterpretStringSlicing(t *testing.T) {

	t.Parallel()

	locationRange1 := interpreter.LocationRange{
		Location: TestLocation,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 116, Line: 4, Column: 31},
			EndPos:   ast.Position{Offset: 140, Line: 4, Column: 55},
		},
	}

	locationRange2 := interpreter.LocationRange{
		Location: TestLocation,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 116, Line: 4, Column: 31},
			EndPos:   ast.Position{Offset: 141, Line: 4, Column: 56},
		},
	}

	type test struct {
		str           string
		from          int
		to            int
		result        string
		expectedError error
	}

	tests := []test{
		{"abcdef", 0, 6, "abcdef", nil},
		{"abcdef", 0, 0, "", nil},
		{"abcdef", 0, 1, "a", nil},
		{"abcdef", 0, 2, "ab", nil},
		{"abcdef", 1, 2, "b", nil},
		{"abcdef", 2, 3, "c", nil},
		{"abcdef", 5, 6, "f", nil},
		{"abcdef", 1, 6, "bcdef", nil},
		// Invalid indices
		{"abcdef", -1, 0, "", interpreter.StringSliceIndicesError{
			FromIndex:     -1,
			UpToIndex:     0,
			Length:        6,
			LocationRange: locationRange2,
		}},
		{"abcdef", 0, -1, "", interpreter.StringSliceIndicesError{
			FromIndex:     0,
			UpToIndex:     -1,
			Length:        6,
			LocationRange: locationRange2,
		}},
		{"abcdef", 0, 10, "", interpreter.StringSliceIndicesError{
			FromIndex:     0,
			UpToIndex:     10,
			Length:        6,
			LocationRange: locationRange2,
		}},
		{"abcdef", 2, 1, "", interpreter.InvalidSliceIndexError{
			FromIndex:     2,
			UpToIndex:     1,
			LocationRange: locationRange1,
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
		{"cafe\\u{301}ba\\u{308}be", 3, 4, "e\u0301", nil},
		{"cafe\\u{301}ba\\u{308}be", 5, 6, "a\u0308", nil},
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

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredStringValue(test.result),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretReturns(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	value, err := inter.Invoke("returnEarly")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

			AssertValuesEqual(
				t,
				inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

			AssertValuesEqual(
				t,
				inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["x"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("testFalse")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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
		interpreter.NewUnmeteredIntValueFromInt64(14),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(377),
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
		interpreter.NewUnmeteredIntValueFromInt64(5),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(120),
		value,
	)
}

func TestInterpretUnaryIntegerNegation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = -2
      let y = -(-2)
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(-2),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["b"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["c"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["d"].GetValue(),
	)
}

func TestInterpretHostFunction(t *testing.T) {

	t.Parallel()

	const code = `
      pub let a = test(1, 2)
    `
	program, err := parser.ParseProgram(code, nil)

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
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			a := invocation.Arguments[0].(interpreter.IntValue).ToBigInt(nil)
			b := invocation.Arguments[1].(interpreter.IntValue).ToBigInt(nil)
			value := new(big.Int).Add(a, b)
			return interpreter.NewUnmeteredIntValueFromBigInt(value)
		},
	)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		false,
		sema.WithPredeclaredValues(
			[]sema.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithStorage(storage),
		interpreter.WithPredeclaredValues(
			[]interpreter.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals["a"].GetValue(),
	)
}

func TestInterpretHostFunctionWithVariableArguments(t *testing.T) {

	t.Parallel()

	const code = `
      pub let nothing = test(1, true, "test")
    `
	program, err := parser.ParseProgram(code, nil)

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
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			called = true

			require.Len(t, invocation.ArgumentTypes, 3)
			assert.IsType(t, sema.IntType, invocation.ArgumentTypes[0])
			assert.IsType(t, sema.BoolType, invocation.ArgumentTypes[1])
			assert.IsType(t, sema.StringType, invocation.ArgumentTypes[2])

			require.Len(t, invocation.Arguments, 3)

			inter := invocation.Interpreter

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(1),
				invocation.Arguments[0],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				invocation.Arguments[1],
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredStringValue("test"),
				invocation.Arguments[2],
			)

			return interpreter.VoidValue{}
		},
	)

	checker, err := sema.NewChecker(
		program,
		TestLocation,
		nil,
		false,
		sema.WithPredeclaredValues(
			[]sema.ValueDeclaration{
				testFunction,
			},
		),
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		interpreter.WithStorage(storage),
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

			inter, err := parseCheckAndInterpretWithOptions(t,
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
			require.NoError(t, err)

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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter, newValue, value)
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

	newValue := interpreter.NewUnmeteredIntValueFromInt64(42)

	value, err := inter.Invoke("test", newValue)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter, newValue, inter.Globals["value"].GetValue())
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		test.GetField(inter, interpreter.ReturnEmptyLocationRange, "foo"),
	)

	value, err := inter.Invoke("callTest")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		test.GetField(inter, interpreter.ReturnEmptyLocationRange, "foo"),
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
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeBool,
			},
			common.Address{},
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	four := interpreter.NewUnmeteredIntValueFromInt64(4)

	value, err := inter.Invoke("isEven", four)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("isOdd", four)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["tests"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals["tests"].GetValue(),
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals["tests"].GetValue(),
	)
}

func TestInterpretOptionalVariableDeclaration(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int?? = 2
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
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
		interpreter.NewUnmeteredIntValueFromInt64(2),
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
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

	value, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretOptionalNestingNil(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
     let x: Int?? = nil
   `)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingRightSubtype(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: Int? = nil ?? nil
    `)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretNilCoalescingNilLiteralInt(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let one = 1
      let x: Int = nil ?? one
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		inter.Globals["test"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals["z"].GetValue(),
	)
}

func TestInterpretNilsComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = nil == nil
   `)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["y"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredStringValue("42"),
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

		AssertValuesEqual(
			t,
			inter,
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

			inter, err := parseCheckAndInterpretWithOptions(t,
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
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				inter.Globals["y"].GetValue(),
			)

			AssertValuesEqual(
				t,
				inter,
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

			inter := parseCheckAndInterpret(t,
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

			inter, err := parseCheckAndInterpretWithOptions(t,
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
								interpreter.NewUnmeteredIntValueFromInt64(1),
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
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(1),
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

			inter, err := parseCheckAndInterpretWithOptions(t,
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
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(2),
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

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithStorage(storage),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		value,
	)
}

func TestInterpretImportError(t *testing.T) {

	t.Parallel()

	const importedLocation1 = common.StringLocation("imported1")
	const importedLocation2 = common.StringLocation("imported2")

	var importedChecker1, importedChecker2 *sema.Checker

	valueDeclarations :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}.ToSemaValueDeclarations()

	parseAndCheck := func(code string, location common.Location) *sema.Checker {
		checker, err := checker.ParseAndCheckWithOptions(t,
			code,
			checker.ParseAndCheckOptions{
				Location: location,
				Options: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
					sema.WithImportHandler(
						func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
							switch importedLocation {
							case importedLocation1:
								return sema.ElaborationImport{
									Elaboration: importedChecker1.Elaboration,
								}, nil
							case importedLocation2:
								return sema.ElaborationImport{
									Elaboration: importedChecker2.Elaboration,
								}, nil
							default:
								assert.FailNow(t, "invalid location")
								return nil, nil
							}
						},
					),
				},
			},
		)
		require.NoError(t, err)
		return checker
	}

	const importedCode1 = `
      pub fun realAnswer(): Int {
          return panic("?!")
      }
    `

	importedChecker1 = parseAndCheck(importedCode1, importedLocation1)

	const importedCode2 = `
	  import realAnswer from "imported1"

      pub fun answer(): Int {
          return realAnswer()
      }
    `

	importedChecker2 = parseAndCheck(importedCode2, importedLocation2)

	const code = `
      import answer from "imported2"

      pub fun test(): Int {
          return answer()
      }
    `

	mainChecker := parseAndCheck(code, TestLocation)

	values := stdlib.StandardLibraryFunctions{
		stdlib.PanicFunction,
	}.ToInterpreterValueDeclarations()

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(mainChecker),
		mainChecker.Location,
		interpreter.WithStorage(storage),
		interpreter.WithPredeclaredValues(values),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				var importedChecker *sema.Checker
				switch location {
				case importedLocation1:
					importedChecker = importedChecker1
				case importedLocation2:
					importedChecker = importedChecker2
				default:
					assert.FailNow(t, "invalid location")
				}

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

	var sb strings.Builder
	printErr := pretty.NewErrorPrettyPrinter(&sb, false).
		PrettyPrintError(
			err,
			mainChecker.Location,
			map[common.Location]string{
				TestLocation:      code,
				importedLocation1: importedCode1,
				importedLocation2: importedCode2,
			},
		)
	require.NoError(t, printErr)
	assert.Equal(t,
		" --> test:5:17\n"+
			"  |\n"+
			"5 |           return answer()\n"+
			"  |                  ^^^^^^^^\n"+
			"\n"+
			" --> imported2:5:17\n"+
			"  |\n"+
			"5 |           return realAnswer()\n"+
			"  |                  ^^^^^^^^^^^^\n"+
			"\n"+
			"error: panic: ?!\n"+
			" --> imported1:3:17\n"+
			"  |\n"+
			"3 |           return panic(\"?!\")\n"+
			"  |                  ^^^^^^^^^^^\n",
		sb.String(),
	)

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

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
		interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	actualDict := inter.Globals["x"].GetValue()

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)
}

func TestInterpretDictionaryInsertionOrder(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"c": 3, "a": 1, "b": 2}
    `)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("c"), interpreter.NewUnmeteredIntValueFromInt64(3),
		interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
		interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
	)

	actualDict := inter.Globals["x"].GetValue()

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)
}

func TestInterpretDictionaryIndexingString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = {"abc": 1, "def": 2}
      let a = x["abc"]
      let b = x["def"]
      let c = x["ghi"]
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
		),
		inter.Globals["b"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("a"),
		),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("b"),
		),
		inter.Globals["b"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["c"].GetValue(),
	)
}

func TestInterpretDictionaryIndexingType(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct TestStruct {}
      resource TestResource {}

      let x: {Type: String} = {
        Type<Int16>(): "a", 
        Type<String>(): "b", 
        Type<AnyStruct>(): "c",
        Type<@TestResource>(): "f"
      }

      let a = x[Type<Int16>()]
      let b = x[Type<String>()]
      let c = x[Type<AnyStruct>()]
      let d = x[Type<Int>()]
      let e = x[Type<TestStruct>()]
      let f = x[Type<@TestResource>()]
    `)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("a"),
		),
		inter.Globals["a"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("b"),
		),
		inter.Globals["b"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("c"),
		),
		inter.Globals["c"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["d"].GetValue(),
	)

	// types need to match exactly, subtypes won't cut it
	assert.Equal(t,
		interpreter.NilValue{},
		inter.Globals["e"].GetValue(),
	)

	assert.Equal(t,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredStringValue("f"),
		),
		inter.Globals["f"].GetValue(),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	actualValue := inter.Globals["x"].GetValue()
	actualDict := actualValue.(*interpreter.DictionaryValue)

	newValue := actualDict.GetKey(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewUnmeteredStringValue("abc"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(23)),
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
		},
		dictionaryKeyValues(inter, actualDict),
	)
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("def"), interpreter.NewUnmeteredIntValueFromInt64(42),
		interpreter.NewUnmeteredStringValue("abc"), interpreter.NewUnmeteredIntValueFromInt64(23),
	)

	actualDict := inter.Globals["x"].GetValue().(*interpreter.DictionaryValue)

	AssertValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.GetKey(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewUnmeteredStringValue("abc"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(23)),
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(42),
		},
		dictionaryKeyValues(inter, actualDict),
	)
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.VoidValue{},
		value,
	)

	expectedDict := interpreter.NewDictionaryValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeString,
			ValueType: interpreter.PrimitiveStaticTypeInt,
		},
		interpreter.NewUnmeteredStringValue("abc"), interpreter.NewUnmeteredIntValueFromInt64(23),
	)

	actualDict := inter.Globals["x"].GetValue().(*interpreter.DictionaryValue)

	RequireValuesEqual(
		t,
		inter,
		expectedDict,
		actualDict,
	)

	newValue := actualDict.GetKey(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.NewUnmeteredStringValue("def"),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		newValue,
	)

	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(23),
		},
		dictionaryKeyValues(inter, actualDict),
	)
}

func TestInterpretOptionalAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x: AnyStruct? = 42
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
		),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(23),
		),
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(23),
		inter.Globals["y"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(23),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["x"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals["y"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
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

		AssertValuesEqual(
			t,
			inter,
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

		// Inject a function that returns a storage reference value,
		// which is borrowed as:
		// - `&R{RI}` (unauthorized, if argument for parameter `authorized` == false)
		// - `auth &R{RI}` (authorized, if argument for parameter `authorized` == true)

		storageAddress := common.MustBytesToAddress([]byte{0x42})
		storagePath := interpreter.PathValue{
			Domain:     common.PathDomainStorage,
			Identifier: "test",
		}

		getStorageReferenceFunctionType := &sema.FunctionType{
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
		}

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				{
					Name: "getStorageReference",
					Type: getStorageReferenceFunctionType,
					Function: interpreter.NewUnmeteredHostFunctionValue(
						func(invocation interpreter.Invocation) interpreter.Value {

							authorized := bool(invocation.Arguments[0].(interpreter.BoolValue))

							riType := getType("RI").(*sema.InterfaceType)
							rType := getType("R")

							return &interpreter.StorageReferenceValue{
								Authorized:           authorized,
								TargetStorageAddress: storageAddress,
								TargetPath:           storagePath,
								BorrowedType: &sema.RestrictedType{
									Type: rType,
									Restrictions: []*sema.InterfaceType{
										riType,
									},
								},
							}
						},
						getStorageReferenceFunctionType,
					),
				},
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		storage := newUnmeteredInMemoryStorage()

		var err error
		inter, err = parseCheckAndInterpretWithOptions(t,
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
					interpreter.WithStorage(storage),
					interpreter.WithPredeclaredValues(values),
				},
			},
		)
		require.NoError(t, err)

		r, err := inter.Invoke("createR")
		require.NoError(t, err)

		r = r.Transfer(
			inter,
			interpreter.ReturnEmptyLocationRange,
			atree.Address(storageAddress),
			true,
			nil,
		)

		storageMap := storage.GetStorageMap(storageAddress, storagePath.Domain.Identifier(), true)
		storageMap.WriteValue(inter, storagePath.Identifier, r)

		result, err := inter.Invoke("testInvalidUnauthorized")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretStringLength(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = "cafe\u{301}".length
      let y = x
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		inter.Globals["x"].GetValue(),
	)
	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		inter.Globals["y"].GetValue(),
	)
}

func TestInterpretStructureFunctionBindingInside(t *testing.T) {

	t.Parallel()

	// TODO: replace AnyStruct return types with (X#(): X),
	//   and test case once bound function types are supported:
	//
	//   fun test(): X {
	//   	let x = X()
	//   	let bar = x.foo()
	//   	return bar()
	//   }

	inter := parseCheckAndInterpret(t, `
        struct X {
            fun foo(): AnyStruct {
                return self.bar
            }

            fun bar(): X {
                return self
            }
        }

        fun test(): AnyStruct {
            let x = X()
            return x.foo()
        }
    `)

	functionValue, err := inter.Invoke("test")
	require.NoError(t, err)

	value, err := inter.InvokeFunctionValue(
		functionValue.(interpreter.FunctionValue),
		nil,
		nil,
		nil,
		nil,
	)
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

	actualArray := inter.Globals["xs"].GetValue()

	arrayValue := actualArray.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		arrayElements(inter, arrayValue),
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
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			},
		},
		{
			name:  "middle",
			index: 1,
			expectedValues: []interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(100),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
			},
		},
		{
			name:  "end",
			index: 3,
			expectedValues: []interpreter.Value{
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredIntValueFromInt64(2),
				interpreter.NewUnmeteredIntValueFromInt64(3),
				interpreter.NewUnmeteredIntValueFromInt64(100),
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

			_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(int64(testCase.index)))
			require.NoError(t, err)

			actualArray := inter.Globals["x"].GetValue()

			require.IsType(t, &interpreter.ArrayValue{}, actualArray)

			AssertValueSlicesEqual(
				t,
				inter,

				testCase.expectedValues,
				arrayElements(inter, actualArray.(*interpreter.ArrayValue)),
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

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
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

	value := inter.Globals["x"].GetValue()

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		arrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

			indexValue := interpreter.NewUnmeteredIntValueFromInt64(int64(index))
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

	value := inter.Globals["x"].GetValue()

	arrayValue := value.(*interpreter.ArrayValue)
	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		arrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
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

	value := inter.Globals["x"].GetValue()

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		arrayElements(inter, arrayValue),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

func TestInterpretArraySlicing(t *testing.T) {

	t.Parallel()

	locationRange1 := interpreter.LocationRange{
		Location: TestLocation,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 125, Line: 4, Column: 31},
			EndPos:   ast.Position{Offset: 149, Line: 4, Column: 55},
		},
	}

	locationRange2 := interpreter.LocationRange{
		Location: TestLocation,
		Range: ast.Range{
			StartPos: ast.Position{Offset: 125, Line: 4, Column: 31},
			EndPos:   ast.Position{Offset: 150, Line: 4, Column: 56},
		},
	}

	type test struct {
		literal       string
		from          int
		to            int
		result        string
		expectedError error
	}

	tests := []test{
		{"[1, 2, 3, 4, 5, 6]", 0, 6, "[1, 2, 3, 4, 5, 6]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 0, "[]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 1, "[1]", nil},
		{"[1, 2, 3, 4, 5, 6]", 0, 2, "[1, 2]", nil},
		{"[1, 2, 3, 4, 5, 6]", 1, 2, "[2]", nil},
		{"[1, 2, 3, 4, 5, 6]", 2, 3, "[3]", nil},
		{"[1, 2, 3, 4, 5, 6]", 5, 6, "[6]", nil},
		{"[1, 2, 3, 4, 5, 6]", 1, 6, "[2, 3, 4, 5, 6]", nil},
		// Invalid indices
		{"[1, 2, 3, 4, 5, 6]", -1, 0, "", interpreter.ArraySliceIndicesError{
			FromIndex:     -1,
			UpToIndex:     0,
			Size:          6,
			LocationRange: locationRange2,
		}},
		{"[1, 2, 3, 4, 5, 6]", 0, -1, "", interpreter.ArraySliceIndicesError{
			FromIndex:     0,
			UpToIndex:     -1,
			Size:          6,
			LocationRange: locationRange2,
		}},
		{"[1, 2, 3, 4, 5, 6]", 0, 10, "", interpreter.ArraySliceIndicesError{
			FromIndex:     0,
			UpToIndex:     10,
			Size:          6,
			LocationRange: locationRange2,
		}},
		{"[1, 2, 3, 4, 5, 6]", 2, 1, "", interpreter.InvalidSliceIndexError{
			FromIndex:     2,
			UpToIndex:     1,
			LocationRange: locationRange1,
		}},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      fun test(): [Int] {
                        let s = %s
                        return s.slice(from: %d, upTo: %d)
                      }
                    `,
					test.literal,
					test.from,
					test.to,
				),
			)

			value, err := inter.Invoke("test")
			if test.expectedError == nil {
				require.NoError(t, err)

				assert.Equal(
					t,
					test.result,
					fmt.Sprint(value),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("doesNotContain")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		value,
	)

	value, err = inter.Invoke("doesNotContainKey")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("abcdef"),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("abcdef"),
		value,
	)
}

func TestInterpretDictionaryRemove(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = {"abc": 1, "def": 2}
      let removed = xs.remove(key: "abc")
    `)

	actualValue := inter.Globals["xs"].GetValue()

	require.IsType(t, actualValue, &interpreter.DictionaryValue{})
	actualDict := actualValue.(*interpreter.DictionaryValue)

	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		dictionaryKeyValues(inter, actualDict),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	actualValue := inter.Globals["xs"].GetValue()

	require.IsType(t, actualValue, &interpreter.DictionaryValue{})
	actualDict := actualValue.(*interpreter.DictionaryValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		dictionaryKeyValues(inter, actualDict),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,

		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue("abc"),
			interpreter.NewUnmeteredStringValue("def"),
			interpreter.NewUnmeteredStringValue("a"),
		},
		arrayElements(inter, arrayValue),
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

	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		arrayElements(inter, arrayValue),
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

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
				inter.Globals["v"].GetValue(),
			)
		})
	}
}

func TestInterpretPathToString(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"Path":           `/storage/a`,
		"StoragePath":    `/storage/a`,
		"PublicPath":     `/public/a`,
		"PrivatePath":    `/private/a`,
		"CapabilityPath": `/private/a`,
	}

	for ty, val := range tests {
		t.Run(ty, func(t *testing.T) {
			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
					  let x: %s = %s
					  let y: String = x.toString()
					`,
					ty,
					val,
				))

			assert.Equal(t,
				interpreter.NewUnmeteredStringValue(val),
				inter.Globals["y"].GetValue(),
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destroys"].GetValue(),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destroys"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		value,
	)

	value, err = inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
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

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithStorage(storage),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructor"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructorA"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(false),
		inter.Globals["ranDestructorB"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.BoolValue(true),
		inter.Globals["ranDestructorA"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
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

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
		inter.Globals["destructionCount"].GetValue(),
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(0),
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

	var actualEvents []interpreter.Value

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
		func(
			_ *interpreter.Interpreter,
			_ func() interpreter.LocationRange,
			event *interpreter.CompositeValue,
			eventType *sema.CompositeType,
		) error {
			actualEvents = append(actualEvents, event)
			return nil
		},
	)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	transferEventType := checker.RequireGlobalType(t, inter.Program.Elaboration, "Transfer")
	transferAmountEventType := checker.RequireGlobalType(t, inter.Program.Elaboration, "TransferAmount")

	fields1 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(2),
		},
	}

	fields2 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(4),
		},
	}

	fields3 := []interpreter.CompositeField{
		{
			Name:  "to",
			Value: interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		{
			Name:  "from",
			Value: interpreter.NewUnmeteredIntValueFromInt64(2),
		},
		{
			Name:  "amount",
			Value: interpreter.NewUnmeteredIntValueFromInt64(100),
		},
	}

	expectedEvents := []interpreter.Value{
		interpreter.NewCompositeValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			fields1,
			common.Address{},
		),
		interpreter.NewCompositeValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferEventType.ID()),
			common.CompositeKindEvent,
			fields2,
			common.Address{},
		),
		interpreter.NewCompositeValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			TestLocation,
			TestLocation.QualifiedIdentifier(transferAmountEventType.ID()),
			common.CompositeKindEvent,
			fields3,
			common.Address{},
		),
	}

	for _, event := range expectedEvents {
		event.(*interpreter.CompositeValue).InitializeFunctions(inter)
	}

	AssertValueSlicesEqual(
		t,
		inter,

		expectedEvents,
		actualEvents,
	)
}

type testValue struct {
	value              interpreter.Value
	ty                 sema.Type
	literal            string
	notAsDictionaryKey bool
}

func (v testValue) String() string {
	if v.literal == "" {
		return v.value.String()
	}
	return v.literal
}

func TestInterpretEmitEventParameterTypes(t *testing.T) {

	t.Parallel()

	sType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		nil,
		TestLocation,
		interpreter.WithStorage(storage),
	)
	require.NoError(t, err)

	sValue := interpreter.NewCompositeValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		TestLocation,
		"S",
		common.CompositeKindStructure,
		nil,
		common.Address{},
	)
	sValue.Functions = map[string]interpreter.FunctionValue{}

	validTypes := map[string]testValue{
		"String": {
			value: interpreter.NewUnmeteredStringValue("test"),
			ty:    sema.StringType,
		},
		"Character": {
			value: interpreter.NewUnmeteredCharacterValue("X"),
			ty:    sema.CharacterType,
		},
		"Bool": {
			value: interpreter.BoolValue(true),
			ty:    sema.BoolType,
		},
		"Address": {
			literal: `0x1`,
			value:   interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			ty:      &sema.AddressType{},
		},
		// Int*
		"Int": {
			value: interpreter.NewUnmeteredIntValueFromInt64(42),
			ty:    sema.IntType,
		},
		"Int8": {
			value: interpreter.NewUnmeteredInt8Value(42),
			ty:    sema.Int8Type,
		},
		"Int16": {
			value: interpreter.NewUnmeteredInt16Value(42),
			ty:    sema.Int16Type,
		},
		"Int32": {
			value: interpreter.NewUnmeteredInt32Value(42),
			ty:    sema.Int32Type,
		},
		"Int64": {
			value: interpreter.NewUnmeteredInt64Value(42),
			ty:    sema.Int64Type,
		},
		"Int128": {
			value: interpreter.NewUnmeteredInt128ValueFromInt64(42),
			ty:    sema.Int128Type,
		},
		"Int256": {
			value: interpreter.NewUnmeteredInt256ValueFromInt64(42),
			ty:    sema.Int256Type,
		},
		// UInt*
		"UInt": {
			value: interpreter.NewUnmeteredUIntValueFromUint64(42),
			ty:    sema.UIntType,
		},
		"UInt8": {
			value: interpreter.NewUnmeteredUInt8Value(42),
			ty:    sema.UInt8Type,
		},
		"UInt16": {
			value: interpreter.NewUnmeteredUInt16Value(42),
			ty:    sema.UInt16Type,
		},
		"UInt32": {
			value: interpreter.NewUnmeteredUInt32Value(42),
			ty:    sema.UInt32Type,
		},
		"UInt64": {
			value: interpreter.NewUnmeteredUInt64Value(42),
			ty:    sema.UInt64Type,
		},
		"UInt128": {
			value: interpreter.NewUnmeteredUInt128ValueFromUint64(42),
			ty:    sema.UInt128Type,
		},
		"UInt256": {
			value: interpreter.NewUnmeteredUInt256ValueFromUint64(42),
			ty:    sema.UInt256Type,
		},
		// Word*
		"Word8": {
			value: interpreter.NewUnmeteredWord8Value(42),
			ty:    sema.Word8Type,
		},
		"Word16": {
			value: interpreter.NewUnmeteredWord16Value(42),
			ty:    sema.Word16Type,
		},
		"Word32": {
			value: interpreter.NewUnmeteredWord32Value(42),
			ty:    sema.Word32Type,
		},
		"Word64": {
			value: interpreter.NewUnmeteredWord64Value(42),
			ty:    sema.Word64Type,
		},
		// Fix*
		"Fix64": {
			value: interpreter.NewUnmeteredFix64Value(123000000),
			ty:    sema.Fix64Type,
		},
		// UFix*
		"UFix64": {
			value: interpreter.NewUnmeteredUFix64Value(123000000),
			ty:    sema.UFix64Type,
		},
		// TODO:
		//// Struct
		//"S": {
		//	literal:            `s`,
		//	ty:                 sType,
		//	notAsDictionaryKey: true,
		//},
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
				value:   interpreter.NewUnmeteredSomeValueNonCopying(testCase.value),
				literal: testCase.literal,
			}

		tests[fmt.Sprintf("[%s]", validType)] =
			testValue{
				value: interpreter.NewArrayValue(
					inter,
					interpreter.ReturnEmptyLocationRange,
					interpreter.VariableSizedStaticType{
						Type: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
					},
					common.Address{},
					testCase.value,
				),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		tests[fmt.Sprintf("[%s; 1]", validType)] =
			testValue{
				value: interpreter.NewArrayValue(
					inter,
					interpreter.ReturnEmptyLocationRange,
					interpreter.ConstantSizedStaticType{
						Type: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
						Size: 1,
					},
					common.Address{},
					testCase.value,
				),
				literal: fmt.Sprintf("[%s as %s]", testCase, validType),
			}

		if !testCase.notAsDictionaryKey {

			value := interpreter.NewDictionaryValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.ConvertSemaToStaticType(nil, testCase.ty),
					ValueType: interpreter.ConvertSemaToStaticType(nil, testCase.ty),
				},
				testCase.value, testCase.value,
			)

			tests[fmt.Sprintf("{%[1]s: %[1]s}", validType)] =
				testValue{
					value:   value,
					literal: fmt.Sprintf("{%[1]s as %[2]s: %[1]s as %[2]s}", testCase, validType),
				}
		}
	}

	for ty, testCase := range tests {

		t.Run(ty, func(t *testing.T) {

			code := fmt.Sprintf(
				`
                  event Test(_ value: %[1]s)

                  fun test() {
                      emit Test(%[2]s as %[1]s)
                  }
                `,
				ty,
				testCase.String(),
			)

			valueDeclarations := stdlib.StandardLibraryValues{
				{
					Name: "s",
					Type: sType,
					ValueFactory: func(i *interpreter.Interpreter) interpreter.Value {
						return sValue
					},
					Kind: common.DeclarationKindConstant,
				},
			}

			inter, err := parseCheckAndInterpretWithOptions(
				t, code, ParseCheckAndInterpretOptions{
					CheckerOptions: []sema.Option{
						sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
						sema.WithPredeclaredTypes([]sema.TypeDeclaration{
							stdlib.StandardLibraryType{
								Name: "S",
								Type: sType,
								Kind: common.DeclarationKindStructure,
							},
						}),
					},
					Options: []interpreter.Option{
						interpreter.WithStorage(storage),
					},
				})
			require.NoError(t, err)

			var actualEvents []interpreter.Value

			inter.SetOnEventEmittedHandler(
				func(
					_ *interpreter.Interpreter,
					_ func() interpreter.LocationRange,
					event *interpreter.CompositeValue,
					eventType *sema.CompositeType,
				) error {
					actualEvents = append(actualEvents, event)
					return nil
				},
			)

			_, err = inter.Invoke("test")
			require.NoError(t, err)

			testType := checker.RequireGlobalType(t, inter.Program.Elaboration, "Test")

			fields := []interpreter.CompositeField{
				{
					Name:  "value",
					Value: testCase.value,
				},
			}

			expectedEvents := []interpreter.Value{
				interpreter.NewCompositeValue(
					inter,
					interpreter.ReturnEmptyLocationRange,
					TestLocation,
					TestLocation.QualifiedIdentifier(testType.ID()),
					common.CompositeKindEvent,
					fields,
					common.Address{},
				),
			}

			for _, event := range expectedEvents {
				event.(*interpreter.CompositeValue).InitializeFunctions(inter)
			}

			AssertValueSlicesEqual(
				t,
				inter,

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

	AssertValuesEqual(
		t,
		inter,
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
		GetKey(inter, interpreter.ReturnEmptyLocationRange, interpreter.NewUnmeteredStringValue("foo"))

	require.IsType(t,
		&interpreter.SomeValue{},
		foo,
	)

	assert.IsType(t,
		&interpreter.CompositeValue{},
		foo.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange),
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
		value.(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange),
	)
}

func TestInterpretReferenceExpression(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      resource R {}

      fun test(): &R {
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(2),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(0),
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
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
	require.Error(t, err)

	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
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

	values := arrayElements(inter, value.(*interpreter.ArrayValue))

	require.IsType(t,
		&interpreter.SomeValue{},
		values[0],
	)

	firstValue := values[0].(*interpreter.SomeValue).
		InnerValue(inter, interpreter.ReturnEmptyLocationRange)

	require.IsType(t,
		&interpreter.CompositeValue{},
		firstValue,
	)

	firstResource := firstValue.(*interpreter.CompositeValue)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(2),
		firstResource.GetField(inter, interpreter.ReturnEmptyLocationRange, "id"),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		values[1],
	)

	secondValue := values[1].(*interpreter.SomeValue).
		InnerValue(inter, interpreter.ReturnEmptyLocationRange)

	require.IsType(t,
		&interpreter.CompositeValue{},
		secondValue,
	)

	secondResource := secondValue.(*interpreter.CompositeValue)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(1),
		secondResource.GetField(inter, interpreter.ReturnEmptyLocationRange, "id"),
	)
}

func TestInterpretResourceMovingAndBorrowing(t *testing.T) {

	t.Parallel()

	t.Run("stack to stack", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                var r2: @R2?

                init() {
                    self.r2 <- nil
                }

                destroy() {
                    destroy self.r2
                }

                fun moveToStack_Borrow_AndMoveBack(): &R2 {
                    // The second assignment should not lead to the resource being cleared
                    let optR2 <- self.r2 <- nil
                    let r2 <- optR2!
                    let ref = &r2 as &R2
                    self.r2 <-! r2
                    return ref
                }
            }

            fun test(): [String?] {
                let r2 <- create R2()
                let r1 <- create R1()
                r1.r2 <-! r2
                let ref = r1.moveToStack_Borrow_AndMoveBack()
                let value = r1.r2?.value
                let refValue = ref.value
                destroy r1
                return [value, refValue]
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeString,
					},
				},
				common.Address{},
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
			),
			value,
		)

	})

	t.Run("from account to stack and back", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            resource R2 {
                let value: String

                init() {
                    self.value = "test"
                }
            }

            resource R1 {
                var r2: @R2?

                init() {
                    self.r2 <- nil
                }

                destroy() {
                    destroy self.r2
                }

                fun moveToStack_Borrow_AndMoveBack(): &R2 {
                    // The second assignment should not lead to the resource being cleared
                    let optR2 <- self.r2 <- nil
                    let r2 <- optR2!
                    let ref = &r2 as &R2
                    self.r2 <-! r2
                    return ref
                }
            }

            fun createR1(): @R1 {
                return <- create R1()
            }

            fun test(r1: &R1): [String?] {
                let r2 <- create R2()
                r1.r2 <-! r2
                let ref = r1.moveToStack_Borrow_AndMoveBack()
                let value = r1.r2?.value
                let refValue = ref.value
                return [value, refValue]
            }
        `)

		r1, err := inter.Invoke("createR1")
		require.NoError(t, err)

		r1 = r1.Transfer(inter, interpreter.ReturnEmptyLocationRange, atree.Address{1}, false, nil)

		r1Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R1")

		ref := &interpreter.EphemeralReferenceValue{
			Value:        r1,
			BorrowedType: r1Type,
		}

		value, err := inter.Invoke("test", ref)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeString,
					},
				},
				common.Address{},
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
				interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue("test"),
				),
			),
			value,
		)

		var permanentSlabs []atree.Slab

		for _, slab := range inter.Storage.(interpreter.InMemoryStorage).Slabs {
			if slab.ID().Address == (atree.Address{}) {
				continue
			}

			permanentSlabs = append(permanentSlabs, slab)
		}

		require.Equal(t, 2, len(permanentSlabs))

		sort.Slice(permanentSlabs, func(i, j int) bool {
			a := permanentSlabs[i].ID()
			b := permanentSlabs[j].ID()
			return a.Compare(b) < 0
		})

		var storedValues []string

		for _, slab := range permanentSlabs {
			storedValue := interpreter.StoredValue(inter, slab, inter.Storage)
			storedValues = append(storedValues, storedValue.String())
		}

		require.Equal(t,
			[]string{
				`S.test.R1(r2: S.test.R2(value: "test", uuid: 2), uuid: 1)`,
				`S.test.R2(value: "test", uuid: 2)`,
			},
			storedValues,
		)
	})
}

func TestInterpretCastingIntLiteralToInt8(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int8
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredInt8Value(42),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCastingIntLiteralToAnyStruct(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as AnyStruct
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCastingIntLiteralToOptional(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let x = 42 as Int?
    `)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(interpreter.NewUnmeteredIntValueFromInt64(42)),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	require.IsType(t,
		&interpreter.SomeValue{},
		inter.Globals["x2"].GetValue(),
	)

	assert.IsType(t,
		interpreter.BoundFunctionValue{},
		inter.Globals["x2"].GetValue().(*interpreter.SomeValue).
			InnerValue(inter, interpreter.ReturnEmptyLocationRange),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["x1"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(42),
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

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals["x"].GetValue(),
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

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretCompositeDeclarationNestedTypeScopingOuterInner(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

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

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

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

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: 2,
			},
			common.Address{},
			interpreter.NewUnmeteredIntValueFromInt64(40),
			interpreter.NewUnmeteredIntValueFromInt64(60),
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

	inter, err := parseCheckAndInterpretWithOptions(t, code,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
				interpreter.WithInjectedCompositeFieldsHandler(
					func(
						inter *interpreter.Interpreter,
						_ common.Location,
						_ string,
						_ common.CompositeKind,
					) map[string]interpreter.Value {
						return map[string]interpreter.Value{
							"account": newTestAuthAccountValue(inter, addressValue),
						}
					},
				),
			},
		},
	)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		addressValue,
		inter.Globals["address1"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
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

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithStorage(storage),
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

	inter, err := parseCheckAndInterpretWithOptions(t, `
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
	require.NoError(t, err)

	i := inter.Globals["C"].GetValue().(interpreter.MemberAccessibleValue).
		GetMember(inter, interpreter.ReturnEmptyLocationRange, "i")

	require.IsType(t,
		interpreter.NewUnmeteredIntValueFromInt64(2),
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

              let nftRef = (&resources[1] as &NFT?)!
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

	AssertValuesEqual(
		t,
		inter, interpreter.NewUnmeteredIntValueFromInt64(3), value)
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

	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
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
              let ref = (&xs["yes"] as &Foo?)!
              let name = ref.name
              destroy xs
              return name
          }

          fun testNil(): String {
              let xs: @{String: Foo} <- {}
              let ref = (&xs["no"] as &Foo?)!
              let name = ref.name
              destroy xs
              return name
          }
        `,
	)
	t.Run("some", func(t *testing.T) {
		value, err := inter.Invoke("testSome")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter, interpreter.NewUnmeteredStringValue("YES"), value)
	})

	t.Run("nil", func(t *testing.T) {
		_, err := inter.Invoke("testNil")
		require.Error(t, err)
		require.ErrorAs(t, err, &interpreter.ForceNilError{})
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUFix64Value(78_900_123_010),
		inter.Globals["a"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUFix64Value(123_405_600_000),
		inter.Globals["b"].GetValue(),
	)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-1_234_500_678_900),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredFix64Value(-121000000),
		inter.Globals["a"].GetValue(),
	)
}

func TestInterpretHexDecode(t *testing.T) {

	t.Parallel()

	expected := []interpreter.Value{
		interpreter.NewUnmeteredUInt8Value(71),
		interpreter.NewUnmeteredUInt8Value(111),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(87),
		interpreter.NewUnmeteredUInt8Value(105),
		interpreter.NewUnmeteredUInt8Value(116),
		interpreter.NewUnmeteredUInt8Value(104),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(116),
		interpreter.NewUnmeteredUInt8Value(104),
		interpreter.NewUnmeteredUInt8Value(101),
		interpreter.NewUnmeteredUInt8Value(32),
		interpreter.NewUnmeteredUInt8Value(70),
		interpreter.NewUnmeteredUInt8Value(108),
		interpreter.NewUnmeteredUInt8Value(111),
		interpreter.NewUnmeteredUInt8Value(119),
	}

	t.Run("in Cadence", func(t *testing.T) {

		standardLibraryFunctions :=
			stdlib.StandardLibraryFunctions{
				stdlib.PanicFunction,
			}

		valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
		values := standardLibraryFunctions.ToInterpreterValueDeclarations()

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              fun hexDecode(_ s: String): [UInt8] {
                  if s.length % 2 != 0 {
                      panic("Input must have even number of characters")
                  }
                  let table: {String: UInt8} = {
                          "0" : 0,
                          "1" : 1,
                          "2" : 2,
                          "3" : 3,
                          "4" : 4,
                          "5" : 5,
                          "6" : 6,
                          "7" : 7,
                          "8" : 8,
                          "9" : 9,
                          "a" : 10,
                          "A" : 10,
                          "b" : 11,
                          "B" : 11,
                          "c" : 12,
                          "C" : 12,
                          "d" : 13,
                          "D" : 13,
                          "e" : 14,
                          "E" : 14,
                          "f" : 15,
                          "F" : 15
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
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, result, &interpreter.ArrayValue{})
		arrayValue := result.(*interpreter.ArrayValue)

		AssertValueSlicesEqual(
			t,
			inter,

			expected,
			arrayElements(inter, arrayValue),
		)
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

		require.IsType(t, result, &interpreter.ArrayValue{})
		arrayValue := result.(*interpreter.ArrayValue)

		AssertValueSlicesEqual(
			t,
			inter,

			expected,
			arrayElements(inter, arrayValue),
		)
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		inter.Globals["x"].GetValue(),
	)
}

func TestInterpretReferenceUseAfterCopy(t *testing.T) {

	t.Parallel()

	t.Run("resource, field write", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test() {
              let r <- create R(name: "1")
              let ref = &r as &R
              let container <- [<-r]
              ref.name = "2"
              destroy container
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

	})

	t.Run("resource, field read", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test(): String {
              let r <- create R(name: "1")
              let ref = &r as &R
              let container <- [<-r]
              let name = ref.name
              destroy container
              return name
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource array, insert", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              let container <- [<-rs]
              ref.insert(at: 1, <-create R())
              destroy container
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource array, append", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              let container <- [<-rs]
              ref.append(<-create R())
              destroy container
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource array, get/set", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              let container <- [<-rs]
              var r <- create R()
              ref[0] <-> r
              destroy container
              destroy r
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource array, remove", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              let container <- [<-rs]
              let r <- ref.remove(at: 0)
              destroy container
              destroy r
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource dictionary, insert", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- {0: <-create R()}
              let ref = &rs as &{Int: R}
              let container <- [<-rs]
              ref[1] <-! create R()
              destroy container
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource dictionary, remove", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          resource R {}

          fun test() {
              let rs <- {0: <-create R()}
              let ref = &rs as &{Int: R}
              let container <- [<-rs]
              let r <- ref.remove(key: 0)
              destroy container
              destroy r
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("struct, field write and read", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test(): [String] {
              let s = S(name: "1")
              let ref = &s as &S
              let container = [s]
              ref.name = "2"
              container[0].name = "3"
              let s2 = container.remove(at: 0)
              return [s.name, s2.name]
          }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeString,
				},
				common.Address{},
				interpreter.NewUnmeteredStringValue("2"),
				interpreter.NewUnmeteredStringValue("3"),
			),
			result,
		)
	})
}

func TestInterpretResourceOwnerFieldUse(t *testing.T) {

	t.Parallel()

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
	// `authAccount`

	address := common.Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "account",
		Type: sema.AuthAccountType,
		ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
			return newTestAuthAccountValue(inter, interpreter.AddressValue(address))
		},
		Kind: common.DeclarationKindConstant,
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
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
				interpreter.WithPublicAccountHandler(
					newTestPublicAccountValue,
				),
			},
		},
	)
	require.NoError(t, err)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NilValue{},
			interpreter.NewUnmeteredSomeValueNonCopying(interpreter.AddressValue(address)),
		},
		arrayElements(inter, result.(*interpreter.ArrayValue)),
	)
}

func newTestAuthAccountValue(
	inter *interpreter.Interpreter,
	addressValue interpreter.AddressValue,
) interpreter.Value {

	panicFunction := interpreter.NewHostFunctionValue(
		inter,
		func(invocation interpreter.Invocation) interpreter.Value {
			panic(errors.NewUnreachableError())
		},
		stdlib.PanicFunction.Type,
	)

	return interpreter.NewAuthAccountValue(
		inter,
		addressValue,
		returnZeroUFix64,
		returnZeroUFix64,
		returnZeroUInt64,
		returnZeroUInt64,
		panicFunction,
		panicFunction,
		func() interpreter.Value {
			return interpreter.NewAuthAccountContractsValue(
				inter,
				addressValue,
				panicFunction,
				panicFunction,
				panicFunction,
				panicFunction,
				func(
					inter *interpreter.Interpreter,
					getLocationRange func() interpreter.LocationRange,
				) *interpreter.ArrayValue {
					return interpreter.NewArrayValue(
						inter,
						getLocationRange,
						interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
						common.Address{},
					)
				},
			)
		},
		func() interpreter.Value {
			return interpreter.NewAuthAccountKeysValue(
				inter,
				addressValue,
				panicFunction,
				panicFunction,
				panicFunction,
			)
		},
	)
}

func newTestPublicAccountValue(
	inter *interpreter.Interpreter,
	addressValue interpreter.AddressValue,
) interpreter.Value {

	panicFunction := interpreter.NewHostFunctionValue(
		inter,
		func(invocation interpreter.Invocation) interpreter.Value {
			panic(errors.NewUnreachableError())
		},
		stdlib.PanicFunction.Type,
	)

	return interpreter.NewPublicAccountValue(
		inter,
		addressValue,
		returnZeroUFix64,
		returnZeroUFix64,
		returnZeroUInt64,
		returnZeroUInt64,
		func() interpreter.Value {
			return interpreter.NewPublicAccountKeysValue(
				inter,
				addressValue,
				panicFunction,
			)
		},
		func() interpreter.Value {
			return interpreter.NewPublicAccountContractsValue(
				inter,
				addressValue,
				panicFunction,
				func(
					inter *interpreter.Interpreter,
					getLocationRange func() interpreter.LocationRange,
				) *interpreter.ArrayValue {
					return interpreter.NewArrayValue(
						inter,
						interpreter.ReturnEmptyLocationRange,
						interpreter.VariableSizedStaticType{
							Type: interpreter.PrimitiveStaticTypeString,
						},
						common.Address{},
					)
				},
			)
		},
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

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Int? = 1
          let y = x!
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(1),
			),
			inter.Globals["x"].GetValue(),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals["y"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

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

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Int = 1
          let y = x!
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(1),
			inter.Globals["y"].GetValue(),
		)
	})
}

func TestInterpretEphemeralReferenceToOptional(t *testing.T) {

	t.Parallel()

	_, err := parseCheckAndInterpretWithOptions(t,
		`
          contract C {

              var rs: @{Int: R}

              resource R {
                  pub let id: Int

                  init(id: Int) {
                      self.id = id
                  }
              }

              fun borrow(id: Int): &R? {
                  return &C.rs[id] as &R?
              }

              init() {
                  self.rs <- {}
                  self.rs[1] <-! create R(id: 1)
                  let ref = self.borrow(id: 1)!
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
	require.NoError(t, err)
}

func TestInterpretNestedDeclarationOrder(t *testing.T) {

	t.Parallel()

	t.Run("A, B", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
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
		require.NoError(t, err)
	})

	t.Run("B, A", func(t *testing.T) {

		t.Parallel()

		_, err := parseCheckAndInterpretWithOptions(t,
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
		require.NoError(t, err)
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
				inter.Globals["number"].GetValue().(interpreter.BigNumberValue).ToBigInt(nil),
			)

			expected := interpreter.NewUnmeteredUInt8Value(uint8(test.Count))

			for i := 1; i <= 3; i++ {
				variableName := fmt.Sprintf("result%d", i)
				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals[variableName].GetValue(),
				)
			}
		})
	}
}

func TestInterpretFailableCastingCompositeTypeConfusion(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		inter.Globals["s"].GetValue(),
	)
}

func TestInterpretNestedDestroy(t *testing.T) {

	t.Parallel()

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
		``,
		func(invocation interpreter.Invocation) interpreter.Value {
			message := invocation.Arguments[0].String()
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

	inter, err := parseCheckAndInterpretWithOptions(t,
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
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
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

	stringIntDictionaryStaticType := interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: interpreter.PrimitiveStaticTypeInt,
	}

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: stringIntDictionaryStaticType,
			},
			common.Address{},
			interpreter.NewDictionaryValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				stringIntDictionaryStaticType,
				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredStringValue("b"),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			interpreter.NewDictionaryValue(
				inter,
				interpreter.ReturnEmptyLocationRange,
				stringIntDictionaryStaticType,
				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewUnmeteredIntValueFromInt64(1),
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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewDictionaryValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeString,
			},
		),
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

	expected := interpreter.NewUnmeteredIntValueFromInt64(377)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		result, err := inter.Invoke(
			"fib",
			interpreter.NewUnmeteredIntValueFromInt64(14),
		)
		require.NoError(b, err)
		RequireValuesEqual(b, inter, expected, result)
	}
}

func TestInterpretMissingMember(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          struct X {
              let y: Int

              init() {
                  self.y = 1
              }
          }

          let x = X()

          fun test() {
              // access missing field y
              x.y
          }
        `,
	)

	// Remove field `y`
	compositeValue := inter.Globals["x"].GetValue().(*interpreter.CompositeValue)
	compositeValue.RemoveField(inter, interpreter.ReturnEmptyLocationRange, "y")

	_, err := inter.Invoke("test")
	require.Error(t, err)

	var missingMemberError interpreter.MissingMemberValueError
	require.ErrorAs(t, err, &missingMemberError)

	require.Equal(t, "y", missingMemberError.Name)
}

func BenchmarkNewInterpreter(b *testing.B) {

	b.Run("new interpreter", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := interpreter.NewInterpreter(nil, nil)
			require.NoError(b, err)
		}
	})

	b.Run("new sub-interpreter", func(b *testing.B) {
		b.ReportAllocs()

		inter, err := interpreter.NewInterpreter(nil, nil)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := inter.NewSubInterpreter(nil, nil)
			require.NoError(b, err)
		}
	})
}

func TestHostFunctionStaticType(t *testing.T) {

	t.Parallel()

	t.Run("toString function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let x = 5
            let y = x.toString
        `)

		value := inter.Globals["y"].GetValue()
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			value.StaticType(inter),
		)
	})

	t.Run("Type function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let x = Type
            let y = x<Int8>()
        `)

		value := inter.Globals["x"].GetValue()
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(
				nil,
				&sema.FunctionType{
					ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.MetaType),
				},
			),
			value.StaticType(inter),
		)

		value = inter.Globals["y"].GetValue()
		assert.Equal(
			t,
			interpreter.PrimitiveStaticTypeMetaType,
			value.StaticType(inter),
		)

		require.IsType(t, interpreter.TypeValue{}, value)
		typeValue := value.(interpreter.TypeValue)
		assert.Equal(t, interpreter.PrimitiveStaticTypeInt8, typeValue.Type)
	})

	t.Run("toString function", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            let a: Int8 = 5
            let b: Fix64 = 4.0

            let x = a.toString
            let y = b.toString
        `)

		// Both `x` and `y` are two functions that returns a string.
		// Hence, their types are equal. i.e: Receivers shouldn't matter.

		xValue := inter.Globals["x"].GetValue()
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			xValue.StaticType(inter),
		)

		yValue := inter.Globals["y"].GetValue()
		assert.Equal(
			t,
			interpreter.ConvertSemaToStaticType(nil, sema.ToStringFunctionType),
			yValue.StaticType(inter),
		)

		assert.Equal(t, xValue.StaticType(inter), yValue.StaticType(inter))
	})
}

func TestInterpretArrayTypeInference(t *testing.T) {

	t.Parallel()

	t.Run("anystruct with empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): Type {
                let x: AnyStruct = []
                return x.getType()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TypeValue{
				Type: interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			},
			value,
		)
	})

	t.Run("anystruct with numeric array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): Type {
                let x: AnyStruct = [1, 2, 3]
                return x.getType()
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TypeValue{
				Type: interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			value,
		)
	})
}

func TestInterpretArrayFirstIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test(): Int? {
          return xs.firstIndex(of: 2)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredIntValueFromInt64(1),
		),
		value,
	)
}

func TestInterpretArrayFirstIndexDoesNotExist(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      let xs = [1, 2, 3]

      fun test(): Int? {
      return xs.firstIndex(of: 5)
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NilValue{},
		value,
	)
}

func TestInterpretOptionalReference(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          fun present(): &Int {
              let x: Int? = 1
              let y = &x as &Int?
              return y!
          }

          fun absent(): &Int {
              let x: Int? = nil
              let y = &x as &Int?
              return y!
          }
        `,
	)

	value, err := inter.Invoke("present")
	require.NoError(t, err)
	require.Equal(
		t,
		&interpreter.EphemeralReferenceValue{
			Value:        interpreter.NewUnmeteredIntValueFromInt64(1),
			BorrowedType: sema.IntType,
		},
		value,
	)

	_, err = inter.Invoke("absent")
	var forceNilError interpreter.ForceNilError
	require.ErrorAs(t, err, &forceNilError)
}

func TestInterpretCastingBoxing(t *testing.T) {

	t.Parallel()

	t.Run("failable cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as? Int?!)?.getType()
        `)

		variable, ok := inter.Globals.Get("a")
		require.True(t, ok)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(),
		)
	})

	t.Run("force cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as! Int?)?.getType()
        `)

		variable, ok := inter.Globals.Get("a")
		require.True(t, ok)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(),
		)
	})

	t.Run("cast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let a = (1 as Int?)?.getType()
        `)

		variable, ok := inter.Globals.Get("a")
		require.True(t, ok)

		require.Equal(
			t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			),
			variable.GetValue(),
		)
	})
}

func TestInterpretNilCoalesceReference(t *testing.T) {

	t.Parallel()

	standardLibraryFunctions :=
		stdlib.StandardLibraryFunctions{
			stdlib.PanicFunction,
		}

	valueDeclarations := standardLibraryFunctions.ToSemaValueDeclarations()
	values := standardLibraryFunctions.ToInterpreterValueDeclarations()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          let xs = {"a": 2}
          let ref = &xs["a"] as &Int? ?? panic("no a")
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
	require.NoError(t, err)

	variable, ok := inter.Globals.Get("ref")
	require.True(t, ok)

	require.Equal(
		t,
		&interpreter.EphemeralReferenceValue{
			Value:        interpreter.NewUnmeteredIntValueFromInt64(2),
			BorrowedType: sema.IntType,
		},
		variable.GetValue(),
	)
}

func TestInterpretDictionaryDuplicateKey(t *testing.T) {

	t.Parallel()

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

          struct S {}

          fun test() {
              let s1 = S()
              let s2 = S()
              {"a": s1, "a": s2}
          }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `

          resource R {}

          fun test() {
              let r1 <- create R()
              let r2 <- create R()
              let rs <- {"a": <-r1, "a": <-r2}
              destroy rs
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		require.ErrorAs(t, err, &interpreter.DuplicateKeyInResourceDictionaryError{})

	})
}
