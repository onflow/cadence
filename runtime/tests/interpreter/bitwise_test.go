/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"testing"

	"github.com/stretchr/testify/require"

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
	"Word128": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord128ValueFromUint64(uint64(v))
	},
	"Word256": func(v int) interpreter.NumberValue {
		return interpreter.NewUnmeteredWord256ValueFromUint64(uint64(v))
	},
}

func init() {

	for _, integerType := range sema.AllIntegerTypes {
		switch integerType {
		case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
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
				inter.Globals.Get("c").GetValue(inter),
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
				inter.Globals.Get("c").GetValue(inter),
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
				inter.Globals.Get("c").GetValue(inter),
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
				inter.Globals.Get("c").GetValue(inter),
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
				inter.Globals.Get("c").GetValue(inter),
			)
		})
	}
}

func TestInterpretBitwiseLeftShift8(t *testing.T) {

	t.Parallel()

	t.Run("Int8 << 9 (zero result)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int8 = 0x7f
			let b: Int8 = 9
			let c = a << b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt8Value(0),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int8 << 1 (positive to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int8 = 5
				let b: Int8 = 1
				let c = a << b
			  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt8Value(10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int8 << 1 (negative to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int8 = -5  // 0b1111_1011
				let b: Int8 = 1
				let c = a << b    // 0b1111_0110  --> -10
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt8Value(-10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int8 << 1 (positive to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int8 = 5  // 0b0000_0101
				let b: Int8 = 7
				let c = a << b    // 0b1000_0000  --> -128
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt8Value(-128),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int8 << 1 (negative to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
					let a: Int8 = -5  // 0b1111_1011
					let b: Int8 = 5
					let c = a << b    // 0b0110_0000  --> 96 
						`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt8Value(0x60), // or 96
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int8 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int8 = 0x7f
				let b: Int8 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("Int16 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int16 = 0x7f
				let b: Int16 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("Int32 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int32 = 0x7f
				let b: Int32 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("Int64 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int64 = 0x7f
				let b: Int64 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("Int128 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int128 = 0x7f
				let b: Int128 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("Int256 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int256 = 0x7f
				let b: Int256 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("UInt8 << 9", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			    let a: UInt8 = 0x7f
				let b: UInt8 = 9
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt8Value(0),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("UInt8 << 1", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: UInt8 = 0xff
				let b: UInt8 = 1
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt8Value(0xfe),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word8 << 9", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word8 = 0xff
				let b: Word8 = 9
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord8Value(0),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word8 << 1", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word8 = 0xff
				let b: Word8 = 1
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord8Value(0xfe),
			inter.Globals.Get("c").GetValue(inter),
		)
	})
}

func TestInterpretBitwiseLeftShift128(t *testing.T) {

	t.Parallel()

	t.Run("Int128 << 130 (zero result)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int128 = 0x7fff_ffff_ffff_ffff_ffff_ffff_ffff_ffff
			let b: Int128 = 130
			let c = a << b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromInt64(int64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 << 1 (positive to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int128 = 5
			let b: Int128 = 1
			let c = a << b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromInt64(10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 << 1 (negative to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int128 = -5
				let b: Int128 = 1
				let c = a << b
					`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromInt64(-10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 << 127 (positive to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int128 = 5 		// 0b0000_0101
				let b: Int128 = 127
				let c = a << b			// 0b1000_0000_..._0000  --> -2^127
			  	`,
		)

		bigInt, _ := big.NewInt(0).SetString("-0x80000000_00000000_00000000_00000000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 << 125 (negative to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int128 = -5 // 0b1111_1111_..._1111_1011
				let b: Int128 = 125
				let c = a << b    // 0b0110_0000_..._0000
			  	`,
		)

		bigInt, _ := big.NewInt(0).SetString("0x60000000_00000000_00000000_00000000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int128 = 0x7fff_ffff
				let b: Int128 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("UInt128 << 130", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			    let a: UInt128 = 0x7fff_ffff
				let b: UInt128 = 130
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt128ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("UInt128 << 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: UInt128 = 0xffff_ffff_ffff_ffff_ffff_ffff_ffff_ffff
				let b: UInt128 = 32
				let c = a << b
				`,
		)

		bigInt, _ := big.NewInt(0).SetString("0xffff_ffff_ffff_ffff_ffff_ffff_0000_0000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt128ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word128 << 130", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word128 = 0xffff_ffff_ffff_ffff
				let b: Word128 = 130
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord128ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word128 << 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word128 = 0xffff_ffff_ffff_ffff_ffff_ffff_ffff_ffff
				let b: Word128 = 32
				let c = a << b
				`,
		)

		bigInt, _ := big.NewInt(0).SetString("0xffff_ffff_ffff_ffff_ffff_ffff_0000_0000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord128ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})
}

func TestInterpretBitwiseLeftShift256(t *testing.T) {

	t.Parallel()

	t.Run("Int256 << 260 (zero result)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int256 = 0x7fff_ffff
			let b: Int256 = 260
			let c = a << b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromInt64(int64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 << 1 (positive to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Int256 = 5
				let b: Int256 = 1
				let c = a << b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromInt64(10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 << 1 (negative to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
					let a: Int256 = -5
					let b: Int256 = 1
					let c = a << b
						`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromInt64(-10),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 << 255 (positive to negative)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
					let a: Int256 = 5 		// 0b0000_0101
					let b: Int256 = 255
					let c = a << b			// 0b1000_0000_..._0000  --> -2^127
				  	`,
		)

		bigInt, _ := big.NewInt(0).SetString("-0x80000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 << 253 (negative to positive)", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
					let a: Int256 = -5 // 0b1111_1111_..._1111_1011
					let b: Int256 = 253
					let c = a << b    // 0b0110_0000_..._0000
				  	`,
		)

		bigInt, _ := big.NewInt(0).SetString("0x60000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000", 0)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromBigInt(bigInt),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 << -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int256 = 0x7fff_ffff
				let b: Int256 = -3
				let c = a << b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("UInt256 << 260", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: UInt256 = 0x7fff_ffff
			let b: UInt256 = 260
			let c = a << b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt256ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("UInt256 << 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: UInt256 = 0x7fff_ffff
			let b: UInt256 = 32
			let c = a << b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt256ValueFromUint64(uint64(0x7fff_ffff_0000_0000)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word256 << 260", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Word256 = 0x7fff_ffff
			let b: Word256 = 260
			let c = a << b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord256ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word256 << 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Word256 = 0x7fff_ffff
			let b: Word256 = 32
			let c = a << b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord256ValueFromUint64(uint64(0x7fff_ffff_0000_0000)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})
}

func TestInterpretBitwiseRightShift128(t *testing.T) {

	t.Parallel()

	t.Run("Int128 >> 130", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int128 = 0x7fff_ffff_ffff_ffff_ffff_ffff_ffff_ffff
			let b: Int128 = 130
			let c = a >> b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromInt64(int64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int128 = 0x7fff_ffff_0000_0000
			let b: Int128 = 32
			let c = a >> b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt128ValueFromInt64(int64(0x7fff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int128 >> -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int128 = 0x7fff_ffff
				let b: Int128 = -3
				let c = a >> b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("UInt128 >> 130", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			    let a: UInt128 = 0x7fff_ffff
				let b: UInt128 = 130
				let c = a >> b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt128ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("UInt128 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: UInt128 = 0xffff_ffff_0000_0000
				let b: UInt128 = 32
				let c = a >> b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt128ValueFromUint64(uint64(0xffff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word128 >> 130", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word128 = 0xffff_ffff_ffff_ffff
				let b: Word128 = 130
				let c = a >> b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord128ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word128 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
				let a: Word128 = 0xffff_ffff_0000_0000
				let b: Word128 = 32
				let c = a >> b
				`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord128ValueFromUint64(uint64(0xffff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})
}

func TestInterpretBitwiseRightShift256(t *testing.T) {

	t.Parallel()

	t.Run("Int256 >> 260", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int256 = 0x7fff_ffff
			let b: Int256 = 260
			let c = a >> b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromInt64(int64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Int256 = 0x7fff_ffff_0000_0000
			let b: Int256 = 32
			let c = a >> b
		  	`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredInt256ValueFromInt64(int64(0x7fff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Int256 >> -3", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			fun test() {
				let a: Int256 = 0x7fff_ffff
				let b: Int256 = -3
				let c = a >> b
				}
		   `)
		_, err := inter.Invoke("test")
		RequireError(t, err)

		var shiftErr interpreter.NegativeShiftError
		require.ErrorAs(t, err, &shiftErr)
	})

	t.Run("UInt256 >> 260", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: UInt256 = 0x7fff_ffff
			let b: UInt256 = 260
			let c = a >> b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt256ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("UInt256 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: UInt256 = 0x7fff_ffff_0000_0000
			let b: UInt256 = 32
			let c = a >> b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredUInt256ValueFromUint64(uint64(0x7fff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word256 >> 260", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Word256 = 0x7fff_ffff
			let b: Word256 = 260
			let c = a >> b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord256ValueFromUint64(uint64(0)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})

	t.Run("Word256 >> 32", func(t *testing.T) {

		inter := parseCheckAndInterpret(t,
			`
			let a: Word256 = 0x7fff_ffff_0000_0000
			let b: Word256 = 32
			let c = a >> b
			`,
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredWord256ValueFromUint64(uint64(0x7fff_ffff)),
			inter.Globals.Get("c").GetValue(inter),
		)
	})
}
