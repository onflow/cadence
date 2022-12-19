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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
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
			common.Address{},
			interpreter.NewUnmeteredCharacterValue("👪"),
			interpreter.NewUnmeteredCharacterValue("❤️"),
		),
		value,
	)
}

func TestInterpretForStatementCapturing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(): [Int] {
           let fs: [fun(): Int] = []
           for x in [1, 2, 3] {
               fs.append(fun (): Int {
                   return x
               })
           }

           let values: [Int] = []
           for f in fs {
              values.append(f())
           }
           return values
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t, value, &interpreter.ArrayValue{})
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
