/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretForStatement(t *testing.T) {

	t.Parallel()

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)
}

func TestInterpretForStatementWithIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           var sum = 0
           for x, y in [1, 2, 3, 4] {
               sum = sum + x
           }
           return sum
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretForStatementWithStoredIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): Int {
           let arr: [Int] = []
           for x, y in [1, 2, 3, 4] {
               arr.append(x)
           }
           var sum = 0
           for z in arr {
              sum = sum + z
           }
           return sum
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretForStatementWithReturn(t *testing.T) {

	t.Parallel()

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		value,
	)
}

func TestInterpretForStatementWithContinue(t *testing.T) {

	t.Parallel()

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

	require.IsType(t, &interpreter.ArrayValue{}, value)
	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(4),
			interpreter.NewUnmeteredIntValueFromInt64(5),
		},
		arrayElements(inter, arrayValue),
	)
}

func TestInterpretForStatementWithBreak(t *testing.T) {

	t.Parallel()

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		value,
	)
}

func TestInterpretForStatementEmpty(t *testing.T) {

	t.Parallel()

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretForString(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(): [Character] {
          let characters: [Character] = []
          let hello = "👪❤️"
          for c in hello {
              characters.append(c)
          }
          return characters
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	RequireValuesEqual(
		t,
		inter,
		interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeCharacter,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredCharacterValue("👪"),
			interpreter.NewUnmeteredCharacterValue("❤️"),
		),
		value,
	)
}

type inclusiveRangeForInLoopTest struct {
	start, end, step int8
	loopElements     []int
}

func TestInclusiveRangeForInLoop(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InclusiveRangeConstructorFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.InclusiveRangeConstructorFunction)

	unsignedTestCases := []inclusiveRangeForInLoopTest{
		{
			start:        0,
			end:          10,
			step:         1,
			loopElements: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			start:        0,
			end:          10,
			step:         2,
			loopElements: []int{0, 2, 4, 6, 8, 10},
		},
	}

	signedTestCases := []inclusiveRangeForInLoopTest{
		{
			start:        10,
			end:          -10,
			step:         -2,
			loopElements: []int{10, 8, 6, 4, 2, 0, -2, -4, -6, -8, -10},
		},
	}

	runTestCase := func(t *testing.T, typ sema.Type, testCase inclusiveRangeForInLoopTest) {
		t.Run(typ.String(), func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(
				`
					fun test(): [%[1]s] {
						let start : %[1]s = %[2]d
						let end : %[1]s = %[3]d
						let step : %[1]s = %[4]d
						let range: InclusiveRange<%[1]s> = InclusiveRange(start, end, step: step)

						var elements : [%[1]s] = []
						for element in range {
							elements.append(element)
						}
						return elements
					}
				`,
				typ.String(),
				testCase.start,
				testCase.end,
				testCase.step,
			)

			inter, err := parseCheckAndInterpretWithOptions(t, code,
				ParseCheckAndInterpretOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivation: baseValueActivation,
					},
					Config: &interpreter.Config{
						BaseActivation: baseActivation,
					},
				},
			)

			require.NoError(t, err)
			loopElements, err := inter.Invoke("test")
			require.NoError(t, err)

			integerStaticType := interpreter.ConvertSemaToStaticType(
				nil,
				typ,
			)

			count := 0
			iterator := (loopElements).(*interpreter.ArrayValue).Iterator(inter)
			for {
				elem := iterator.Next(inter)
				if elem == nil {
					break
				}

				AssertValuesEqual(
					t,
					inter,
					interpreter.GetSmallIntegerValue(
						int8(testCase.loopElements[count]),
						integerStaticType,
					),
					elem,
				)

				count += 1
			}

			assert.Equal(t, len(testCase.loopElements), count)
		})
	}

	for _, typ := range sema.AllIntegerTypes {
		// Only test leaf types
		switch typ {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		for _, testCase := range unsignedTestCases {
			runTestCase(t, typ, testCase)
		}
	}

	for _, typ := range sema.AllSignedIntegerTypes {
		// Only test leaf types
		switch typ {
		case sema.SignedIntegerType:
			continue
		}

		for _, testCase := range signedTestCases {
			runTestCase(t, typ, testCase)
		}
	}
}
