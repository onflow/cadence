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
	"testing"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

var bitwiseTestValueFunctions = map[string]func(int) interpreter.NumberValue{
	// Int*
	"Int": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredIntValueFromInt64(int64(v))
	},
	"Int8": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt8Value(int8(v))
	},
	"Int16": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt16Value(int16(v))
	},
	"Int32": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt32Value(int32(v))
	},
	"Int64": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt64Value(int64(v))
	},
	"Int128": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt128ValueFromInt64(int64(v))
	},
	"Int256": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredInt256ValueFromInt64(int64(v))
	},
	// UInt*
	"UInt": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUIntValueFromUint64(uint64(v))
	},
	"UInt8": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt8Value(uint8(v))
	},
	"UInt16": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt16Value(uint16(v))
	},
	"UInt32": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt32Value(uint32(v))
	},
	"UInt64": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt64Value(uint64(v))
	},
	"UInt128": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt128ValueFromUint64(uint64(v))
	},
	"UInt256": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredUInt256ValueFromUint64(uint64(v))
	},
	// Word*
	"Word8": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord8Value(uint8(v))
	},
	"Word16": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord16Value(uint16(v))
	},
	"Word32": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord32Value(uint32(v))
	},
	"Word64": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord64Value(uint64(v))
	},
}

func init() {

	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType:
			continue
		}

		if _, ok := bitwiseTestValueFunctions[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}
}

func TestInterpretBitwiseOr(t *testing.T) {

	t.Parallel()

	for ty, valueFunc := range bitwiseTestValueFunctions {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 0b00010001
                      let b: %[1]s = 0b00000100
                      let c = a | b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				valueFunc(0b00010101),
				inter.Globals["c"].GetValue(),
			)
		})
	}
}

func TestInterpretBitwiseXor(t *testing.T) {

	t.Parallel()

	for ty, valueFunc := range bitwiseTestValueFunctions {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 0b00010001
                      let b: %[1]s = 0b00010100
                      let c = a ^ b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				valueFunc(0b00000101),
				inter.Globals["c"].GetValue(),
			)
		})
	}
}

func TestInterpretBitwiseAnd(t *testing.T) {

	t.Parallel()

	for ty, valueFunc := range bitwiseTestValueFunctions {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 0b00010001
                      let b: %[1]s = 0b00010100
                      let c = a & b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				valueFunc(0b00010000),
				inter.Globals["c"].GetValue(),
			)
		})
	}
}

func TestInterpretBitwiseLeftShift(t *testing.T) {

	t.Parallel()

	for ty, valueFunc := range bitwiseTestValueFunctions {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 0b00001100
                      let b: %[1]s = 3
                      let c = a << b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				valueFunc(0b01100000),
				inter.Globals["c"].GetValue(),
			)
		})
	}
}

func TestInterpretBitwiseRightShift(t *testing.T) {

	t.Parallel()

	for ty, valueFunc := range bitwiseTestValueFunctions {

		t.Run(ty, func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let a: %[1]s = 0b01100000
                      let b: %[1]s = 3
                      let c = a >> b
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				valueFunc(0b00001100),
				inter.Globals["c"].GetValue(),
			)
		})
	}
}
