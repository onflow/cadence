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
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretToString(t *testing.T) {

	t.Parallel()

	for _, ty := range sema.AllIntegerTypes {

		t.Run(ty.String(), func(t *testing.T) {

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x: %s = 42
                      let y = x.toString()
                    `,
					ty,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredStringValue("42"),
				inter.Globals.Get("y").GetValue(),
			)
		})
	}

	t.Run("Address", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Address = 0x42
          let y = x.toString()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("0x0000000000000042"),
			inter.Globals.Get("y").GetValue(),
		)
	})

	for _, ty := range sema.AllFixedPointTypes {

		t.Run(ty.String(), func(t *testing.T) {

			var literal string
			var expected interpreter.Value

			isSigned := sema.IsSubType(ty, sema.SignedFixedPointType)

			if isSigned {
				literal = "-12.34"
				expected = interpreter.NewUnmeteredStringValue("-12.34000000")
			} else {
				literal = "12.34"
				expected = interpreter.NewUnmeteredStringValue("12.34000000")
			}

			inter := parseCheckAndInterpret(t,
				fmt.Sprintf(
					`
                      let x: %s = %s
                      let y = x.toString()
                    `,
					ty,
					literal,
				),
			)

			AssertValuesEqual(
				t,
				inter,
				expected,
				inter.Globals.Get("y").GetValue(),
			)
		})
	}
}

func TestInterpretToBytes(t *testing.T) {

	t.Parallel()

	t.Run("Address", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let x: Address = 0x123456
          let y = x.toBytes()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredUInt8Value(0x0),
				interpreter.NewUnmeteredUInt8Value(0x0),
				interpreter.NewUnmeteredUInt8Value(0x0),
				interpreter.NewUnmeteredUInt8Value(0x0),
				interpreter.NewUnmeteredUInt8Value(0x0),
				interpreter.NewUnmeteredUInt8Value(0x12),
				interpreter.NewUnmeteredUInt8Value(0x34),
				interpreter.NewUnmeteredUInt8Value(0x56),
			),
			inter.Globals.Get("y").GetValue(),
		)
	})
}

func TestInterpretAddressFromBytes(t *testing.T) {

	t.Parallel()

	runValidCase := func(t *testing.T, expected []byte, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
                  fun test(): Address {
                      return Address.fromBytes(%s)
                  }
            	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			addressVal, ok := res.(interpreter.AddressValue)
			require.True(t, ok)

			require.Equal(t, expected, addressVal.ToAddress().Bytes())
		})
	}

	runValidRoundTripCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
                  fun test(): Bool {
                    let address : Address = %s;
					return address == Address.fromBytes(address.toBytes());
                  }
            	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			boolVal, ok := res.(interpreter.BoolValue)
			require.True(t, ok)

			require.True(t, bool(boolVal))
		})
	}

	runInvalidCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
                  fun test(): Address {
                      return Address.fromBytes(%s)
                  }
            	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			_, err := inter.Invoke("test")

			RequireError(t, err)
			require.ErrorIs(t, err, common.AddressOverflowError)
		})
	}

	runValidCase(t, []byte{}, "[]")
	runValidCase(t, []byte{1}, "[1]")
	runValidCase(t, []byte{12, 34, 56}, "[12, 34, 56]")
	runValidCase(t, []byte{67, 97, 100, 101, 110, 99, 101, 33}, "[67, 97, 100, 101, 110, 99, 101, 33]")

	runValidRoundTripCase(t, "0x0")
	runValidRoundTripCase(t, "0x01")
	runValidRoundTripCase(t, "0x436164656E636521")
	runValidRoundTripCase(t, "0x46716465AE633188")

	runInvalidCase(t, "[12, 34, 56, 11, 22, 33, 44, 55, 66, 77, 88, 99, 111]")
}

func TestInterpretAddressFromString(t *testing.T) {

	t.Parallel()

	runValidCase := func(t *testing.T, expected string, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
                  fun test(): Address? {
                      return Address.fromString(%s)
                  }
            	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			addressOpt, ok := res.(*interpreter.SomeValue)
			require.True(t, ok)

			innerValue := addressOpt.InnerValue(inter, interpreter.EmptyLocationRange)
			addressVal, ok := innerValue.(interpreter.AddressValue)
			require.True(t, ok)
			require.Equal(t, expected, addressVal.ToAddress().HexWithPrefix())
		})
	}

	runValidRoundTripCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
	              fun test(): Bool {
	                let address : Address? = %s;
					return address == Address.fromString(address!.toString());
	              }
	        	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")

			require.NoError(t, err)

			boolVal, ok := res.(interpreter.BoolValue)
			require.True(t, ok)

			require.True(t, bool(boolVal))
		})
	}

	runInvalidCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(`
	              fun test(): Address? {
	                  return Address.fromString(%s)
	              }
	        	`,
				innerCode,
			)

			inter := parseCheckAndInterpret(t, code)
			res, err := inter.Invoke("test")
			require.NoError(t, err)

			_, ok := res.(interpreter.NilValue)
			require.True(t, ok)
		})
	}

	// Note: output of HexWithPrefix() lowercases the 'E'.
	runValidCase(t, "0x436164656e636521", "\"0x436164656E636521\"")
	runValidCase(t, "0x0000000000000000", "\"0x0\"")
	runValidCase(t, "0x0000000000000001", "\"0x01\"")

	runValidRoundTripCase(t, "0x0")
	runValidRoundTripCase(t, "0x01")
	runValidRoundTripCase(t, "0x46716465AE633188")

	runInvalidCase(t, "\"436164656E636521\"")
	runInvalidCase(t, "\"ZZZ\"")
	runInvalidCase(t, "\"0xZZZ\"")
	runInvalidCase(t, "\"0x436164656E63652146757265766572\"")
}

func TestInterpretToBigEndianBytes(t *testing.T) {

	t.Parallel()

	typeTests := map[string]map[string][]byte{
		// Int*
		"Int": {
			"0":                  {0},
			"42":                 {42},
			"127":                {127},
			"128":                {0, 128},
			"200":                {0, 200},
			"-1":                 {255},
			"-200":               {255, 56},
			"-10000000000000000": {220, 121, 13, 144, 63, 0, 0},
		},
		"Int8": {
			"0":    {0},
			"42":   {42},
			"127":  {127},
			"-1":   {255},
			"-127": {129},
			"-128": {128},
		},
		"Int16": {
			"0":      {0, 0},
			"42":     {0, 42},
			"32767":  {127, 255},
			"-1":     {255, 255},
			"-32767": {128, 1},
			"-32768": {128, 0},
		},
		"Int32": {
			"0":           {0, 0, 0, 0},
			"42":          {0, 0, 0, 42},
			"2147483647":  {127, 255, 255, 255},
			"-1":          {255, 255, 255, 255},
			"-2147483647": {128, 0, 0, 1},
			"-2147483648": {128, 0, 0, 0},
		},
		"Int64": {
			"0":                    {0, 0, 0, 0, 0, 0, 0, 0},
			"42":                   {0, 0, 0, 0, 0, 0, 0, 42},
			"9223372036854775807":  {127, 255, 255, 255, 255, 255, 255, 255},
			"-1":                   {255, 255, 255, 255, 255, 255, 255, 255},
			"-9223372036854775807": {128, 0, 0, 0, 0, 0, 0, 1},
			"-9223372036854775808": {128, 0, 0, 0, 0, 0, 0, 0},
		},
		"Int128": {
			"0":                  {0},
			"42":                 {42},
			"127":                {127},
			"128":                {0, 128},
			"200":                {0, 200},
			"-1":                 {255},
			"-200":               {255, 56},
			"-10000000000000000": {220, 121, 13, 144, 63, 0, 0},
		},
		"Int256": {
			"0":                  {0},
			"42":                 {42},
			"127":                {127},
			"128":                {0, 128},
			"200":                {0, 200},
			"-1":                 {255},
			"-200":               {255, 56},
			"-10000000000000000": {220, 121, 13, 144, 63, 0, 0},
		},
		// UInt*
		"UInt": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"200": {200},
		},
		"UInt8": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"255": {255},
		},
		"UInt16": {
			"0":     {0, 0},
			"42":    {0, 42},
			"32767": {127, 255},
			"32768": {128, 0},
			"65535": {255, 255},
		},
		"UInt32": {
			"0":          {0, 0, 0, 0},
			"42":         {0, 0, 0, 42},
			"2147483647": {127, 255, 255, 255},
			"2147483648": {128, 0, 0, 0},
			"4294967295": {255, 255, 255, 255},
		},
		"UInt64": {
			"0":                    {0, 0, 0, 0, 0, 0, 0, 0},
			"42":                   {0, 0, 0, 0, 0, 0, 0, 42},
			"9223372036854775807":  {127, 255, 255, 255, 255, 255, 255, 255},
			"9223372036854775808":  {128, 0, 0, 0, 0, 0, 0, 0},
			"18446744073709551615": {255, 255, 255, 255, 255, 255, 255, 255},
		},
		"UInt128": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"170141183460469231731687303715884105727": {127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			"170141183460469231731687303715884105728": {128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			"340282366920938463463374607431768211455": {255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		"UInt256": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"200": {200},
		},
		// Word*
		"Word8": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"255": {255},
		},
		"Word16": {
			"0":     {0, 0},
			"42":    {0, 42},
			"32767": {127, 255},
			"32768": {128, 0},
			"65535": {255, 255},
		},
		"Word32": {
			"0":          {0, 0, 0, 0},
			"42":         {0, 0, 0, 42},
			"2147483647": {127, 255, 255, 255},
			"2147483648": {128, 0, 0, 0},
			"4294967295": {255, 255, 255, 255},
		},
		"Word64": {
			"0":                    {0, 0, 0, 0, 0, 0, 0, 0},
			"42":                   {0, 0, 0, 0, 0, 0, 0, 42},
			"9223372036854775807":  {127, 255, 255, 255, 255, 255, 255, 255},
			"9223372036854775808":  {128, 0, 0, 0, 0, 0, 0, 0},
			"18446744073709551615": {255, 255, 255, 255, 255, 255, 255, 255},
		},
		"Word128": {
			"0":   {0},
			"42":  {42},
			"127": {127},
			"128": {128},
			"170141183460469231731687303715884105727": {127, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			"170141183460469231731687303715884105728": {128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			"340282366920938463463374607431768211455": {255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
		// Fix*
		"Fix64": {
			"0.0":   {0, 0, 0, 0, 0, 0, 0, 0},
			"42.0":  {0, 0, 0, 0, 250, 86, 234, 0},
			"42.24": {0, 0, 0, 0, 251, 197, 32, 0},
			"-1.0":  {255, 255, 255, 255, 250, 10, 31, 0},
		},
		// UFix*
		"UFix64": {
			"0.0":   {0, 0, 0, 0, 0, 0, 0, 0},
			"42.0":  {0, 0, 0, 0, 250, 86, 234, 0},
			"42.24": {0, 0, 0, 0, 251, 197, 32, 0},
		},
	}

	// Ensure the test cases are complete

	for _, integerType := range sema.AllNumberTypes {
		switch integerType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := typeTests[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test: missing %s", integerType))
		}
	}

	for ty, tests := range typeTests {

		for value, expected := range tests {

			t.Run(fmt.Sprintf("%s: %s", ty, value), func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                      let value: %s = %s
	                      let result = value.toBigEndianBytes()
	                    `,
						ty,
						value,
					),
				)

				AssertValuesEqual(
					t,
					inter,
					interpreter.ByteSliceToByteArrayValue(inter, expected),
					inter.Globals.Get("result").GetValue(),
				)
			})
		}
	}
}

func TestInterpretFromBigEndianBytes(t *testing.T) {

	t.Parallel()

	validTestsWithRoundtrip := map[string]map[string]interpreter.Value{
		// Int*
		"Int": {
			"[0]":                           interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(0)),
			"[42]":                          interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(42)),
			"[127]":                         interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(127)),
			"[0, 128]":                      interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(128)),
			"[0, 200]":                      interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(200)),
			"[255]":                         interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(-1)),
			"[255, 56]":                     interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(-200)),
			"[220, 121, 13, 144, 63, 0, 0]": interpreter.NewUnmeteredIntValueFromBigInt(big.NewInt(-10000000000000000)),
		},
		"Int8": {
			"[0]":   interpreter.NewUnmeteredInt8Value(0),
			"[42]":  interpreter.NewUnmeteredInt8Value(42),
			"[127]": interpreter.NewUnmeteredInt8Value(127),
			"[255]": interpreter.NewUnmeteredInt8Value(-1),
			"[129]": interpreter.NewUnmeteredInt8Value(-127),
			"[128]": interpreter.NewUnmeteredInt8Value(-128),
		},
		"Int16": {
			"[0, 0]":     interpreter.NewUnmeteredInt16Value(0),
			"[42]":       interpreter.NewUnmeteredInt16Value(42),
			"[0, 42]":    interpreter.NewUnmeteredInt16Value(42),
			"[127, 255]": interpreter.NewUnmeteredInt16Value(32767),
			"[255, 255]": interpreter.NewUnmeteredInt16Value(-1),
			"[128, 1]":   interpreter.NewUnmeteredInt16Value(-32767),
			"[128, 0]":   interpreter.NewUnmeteredInt16Value(-32768),
		},
		"Int32": {
			"[0, 0, 0, 0]":         interpreter.NewUnmeteredInt32Value(0),
			"[42]":                 interpreter.NewUnmeteredInt32Value(42),
			"[0, 0, 0, 42]":        interpreter.NewUnmeteredInt32Value(42),
			"[127, 255, 255, 255]": interpreter.NewUnmeteredInt32Value(2147483647),
			"[255, 255, 255, 255]": interpreter.NewUnmeteredInt32Value(-1),
			"[128, 0, 0, 1]":       interpreter.NewUnmeteredInt32Value(-2147483647),
			"[128, 0, 0, 0]":       interpreter.NewUnmeteredInt32Value(-2147483648),
		},
		"Int64": {
			"[0, 0, 0, 0, 0, 0, 0, 0]":  interpreter.NewUnmeteredInt64Value(0),
			"[42]":                      interpreter.NewUnmeteredInt64Value(42),
			"[0, 0, 0, 0, 0, 0, 0, 42]": interpreter.NewUnmeteredInt64Value(42),
			"[127, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredInt64Value(9223372036854775807),
			"[255, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredInt64Value(-1),
			"[128, 0, 0, 0, 0, 0, 0, 1]":               interpreter.NewUnmeteredInt64Value(-9223372036854775807),
			"[128, 0, 0, 0, 0, 0, 0, 0]":               interpreter.NewUnmeteredInt64Value(-9223372036854775808),
		},
		"Int128": {
			"[0]":                           interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(0)),
			"[42]":                          interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(42)),
			"[127]":                         interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(127)),
			"[0, 128]":                      interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(128)),
			"[0, 200]":                      interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(200)),
			"[255]":                         interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(-1)),
			"[255, 56]":                     interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(-200)),
			"[220, 121, 13, 144, 63, 0, 0]": interpreter.NewUnmeteredInt128ValueFromBigInt(big.NewInt(-10000000000000000)),
		},
		"Int256": {
			"[0]":                           interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(0)),
			"[42]":                          interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(42)),
			"[127]":                         interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(127)),
			"[0, 128]":                      interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(128)),
			"[0, 200]":                      interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(200)),
			"[255]":                         interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(-1)),
			"[255, 56]":                     interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(-200)),
			"[220, 121, 13, 144, 63, 0, 0]": interpreter.NewUnmeteredInt256ValueFromBigInt(big.NewInt(-10000000000000000)),
		},
		// UInt*
		"UInt": {
			"[0]":   interpreter.NewUnmeteredUIntValueFromUint64(0),
			"[42]":  interpreter.NewUnmeteredUIntValueFromUint64(42),
			"[127]": interpreter.NewUnmeteredUIntValueFromUint64(127),
			"[128]": interpreter.NewUnmeteredUIntValueFromUint64(128),
			"[200]": interpreter.NewUnmeteredUIntValueFromUint64(200),
		},
		"UInt8": {
			"[0]":   interpreter.NewUnmeteredUInt8Value(0),
			"[42]":  interpreter.NewUnmeteredUInt8Value(42),
			"[127]": interpreter.NewUnmeteredUInt8Value(127),
			"[128]": interpreter.NewUnmeteredUInt8Value(128),
			"[255]": interpreter.NewUnmeteredUInt8Value(255),
		},
		"UInt16": {
			"[0, 0]":     interpreter.NewUnmeteredUInt16Value(0),
			"[42]":       interpreter.NewUnmeteredUInt16Value(42),
			"[0, 42]":    interpreter.NewUnmeteredUInt16Value(42),
			"[127, 255]": interpreter.NewUnmeteredUInt16Value(32767),
			"[128, 0]":   interpreter.NewUnmeteredUInt16Value(32768),
			"[255, 255]": interpreter.NewUnmeteredUInt16Value(65535),
		},
		"UInt32": {
			"[0, 0, 0, 0]":         interpreter.NewUnmeteredUInt32Value(0),
			"[42]":                 interpreter.NewUnmeteredUInt32Value(42),
			"[0, 0, 0, 42]":        interpreter.NewUnmeteredUInt32Value(42),
			"[127, 255, 255, 255]": interpreter.NewUnmeteredUInt32Value(2147483647),
			"[128, 0, 0, 0]":       interpreter.NewUnmeteredUInt32Value(2147483648),
			"[255, 255, 255, 255]": interpreter.NewUnmeteredUInt32Value(4294967295),
		},
		"UInt64": {
			"[0, 0, 0, 0, 0, 0, 0, 0]":  interpreter.NewUnmeteredUInt64Value(0),
			"[42]":                      interpreter.NewUnmeteredUInt64Value(42),
			"[0, 0, 0, 0, 0, 0, 0, 42]": interpreter.NewUnmeteredUInt64Value(42),
			"[127, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredUInt64Value(9223372036854775807),
			"[128, 0, 0, 0, 0, 0, 0, 0]":               interpreter.NewUnmeteredUInt64Value(9223372036854775808),
			"[255, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredUInt64Value(18446744073709551615),
		},
		"UInt128": {
			"[0]":   interpreter.NewUnmeteredUInt128ValueFromBigInt(big.NewInt(0)),
			"[42]":  interpreter.NewUnmeteredUInt128ValueFromBigInt(big.NewInt(42)),
			"[127]": interpreter.NewUnmeteredUInt128ValueFromBigInt(big.NewInt(127)),
			"[128]": interpreter.NewUnmeteredUInt128ValueFromBigInt(big.NewInt(128)),
			"[200]": interpreter.NewUnmeteredUInt128ValueFromBigInt(big.NewInt(200)),
		},
		"UInt256": {
			"[0]":   interpreter.NewUnmeteredUInt256ValueFromBigInt(big.NewInt(0)),
			"[42]":  interpreter.NewUnmeteredUInt256ValueFromBigInt(big.NewInt(42)),
			"[127]": interpreter.NewUnmeteredUInt256ValueFromBigInt(big.NewInt(127)),
			"[128]": interpreter.NewUnmeteredUInt256ValueFromBigInt(big.NewInt(128)),
			"[200]": interpreter.NewUnmeteredUInt256ValueFromBigInt(big.NewInt(200)),
		},
		// Word*
		"Word8": {
			"[0]":   interpreter.NewUnmeteredWord8Value(0),
			"[42]":  interpreter.NewUnmeteredWord8Value(42),
			"[127]": interpreter.NewUnmeteredWord8Value(127),
			"[128]": interpreter.NewUnmeteredWord8Value(128),
			"[255]": interpreter.NewUnmeteredWord8Value(255),
		},
		"Word16": {
			"[0, 0]":     interpreter.NewUnmeteredWord16Value(0),
			"[42]":       interpreter.NewUnmeteredWord16Value(42),
			"[0, 42]":    interpreter.NewUnmeteredWord16Value(42),
			"[127, 255]": interpreter.NewUnmeteredWord16Value(32767),
			"[128, 0]":   interpreter.NewUnmeteredWord16Value(32768),
			"[255, 255]": interpreter.NewUnmeteredWord16Value(65535),
		},
		"Word32": {
			"[0, 0, 0, 0]":         interpreter.NewUnmeteredWord32Value(0),
			"[42]":                 interpreter.NewUnmeteredWord32Value(42),
			"[0, 0, 0, 42]":        interpreter.NewUnmeteredWord32Value(42),
			"[127, 255, 255, 255]": interpreter.NewUnmeteredWord32Value(2147483647),
			"[128, 0, 0, 0]":       interpreter.NewUnmeteredWord32Value(2147483648),
			"[255, 255, 255, 255]": interpreter.NewUnmeteredWord32Value(4294967295),
		},
		"Word64": {
			"[0, 0, 0, 0, 0, 0, 0, 0]":  interpreter.NewUnmeteredWord64Value(0),
			"[42]":                      interpreter.NewUnmeteredWord64Value(42),
			"[0, 0, 0, 0, 0, 0, 0, 42]": interpreter.NewUnmeteredWord64Value(42),
			"[127, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredWord64Value(9223372036854775807),
			"[128, 0, 0, 0, 0, 0, 0, 0]":               interpreter.NewUnmeteredWord64Value(9223372036854775808),
			"[255, 255, 255, 255, 255, 255, 255, 255]": interpreter.NewUnmeteredWord64Value(18446744073709551615),
		},
		// Fix*
		"Fix64": {
			"[0, 0, 0, 0, 0, 0, 0, 0]":             interpreter.NewUnmeteredFix64Value(0),
			"[250, 86, 234, 0]":                    interpreter.NewUnmeteredFix64Value(42 * sema.Fix64Factor), // 42.0 with padding
			"[0, 0, 0, 0, 250, 86, 234, 0]":        interpreter.NewUnmeteredFix64Value(42 * sema.Fix64Factor), // 42.0
			"[0, 0, 0, 0, 251, 197, 32, 0]":        interpreter.NewUnmeteredFix64Value(4224_000_000),          // 42.24
			"[255, 255, 255, 255, 250, 10, 31, 0]": interpreter.NewUnmeteredFix64Value(-1 * sema.Fix64Factor), // -1.0
		},
		// UFix*
		"UFix64": {
			"[0, 0, 0, 0, 0, 0, 0, 0]":      interpreter.NewUnmeteredUFix64Value(0),
			"[250, 86, 234, 0]":             interpreter.NewUnmeteredUFix64Value(42 * sema.Fix64Factor), // 42.0 with padding
			"[0, 0, 0, 0, 250, 86, 234, 0]": interpreter.NewUnmeteredUFix64Value(42 * sema.Fix64Factor), // 42.0
			"[0, 0, 0, 0, 251, 197, 32, 0]": interpreter.NewUnmeteredUFix64Value(4224_000_000),          // 42.24
		},
	}

	invalidTests := map[string][]string{
		// Int*
		"Int": {}, // No overflow
		"Int8": {
			"[0, 0]",
			"[0, 22]",
		},
		"Int16": {
			"[0, 0, 0]",
			"[0, 22, 0]",
		},
		"Int32": {
			"[0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0]",
		},
		"Int64": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0]",
		},
		"Int128": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
		},
		"Int256": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
		},
		// UInt*
		"UInt": {}, // No overflow
		"UInt8": {
			"[0, 0]",
			"[0, 22]",
		},
		"UInt16": {
			"[0, 0, 0]",
			"[0, 22, 0]",
		},
		"UInt32": {
			"[0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0]",
		},
		"UInt64": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0]",
		},
		"UInt128": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
		},
		"UInt256": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]",
		},
		// Word*
		"Word8": {
			"[0, 0]",
			"[0, 22]",
		},
		"Word16": {
			"[0, 0, 0]",
			"[0, 22, 0]",
		},
		"Word32": {
			"[0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0]",
		},
		"Word64": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0]",
		},
		// Fix*
		"Fix64": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0]",
		},
		// UFix*
		"UFix64": {
			"[0, 0, 0, 0, 0, 0, 0, 0, 0]",
			"[0, 22, 0, 0, 0, 0, 0, 0, 0]",
		},
	}

	// Ensure the test cases are complete

	for _, integerType := range sema.AllNumberTypes {
		switch integerType {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue
		}

		if _, ok := validTestsWithRoundtrip[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test for valid cases: missing %s", integerType))
		}

		if _, ok := invalidTests[integerType.String()]; !ok {
			panic(fmt.Sprintf("broken test for invalid cases: missing %s", integerType))
		}
	}

	for ty, tests := range validTestsWithRoundtrip {
		for value, expected := range tests {
			t.Run(fmt.Sprintf("%s: %s", ty, value), func(t *testing.T) {
				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                      let resultOpt: %s? = %s.fromBigEndianBytes(%s)
						  let result: %s = resultOpt!
						  let roundTripEqual = result == %s.fromBigEndianBytes(result.toBigEndianBytes())!
	                    `,
						ty,
						ty,
						value,
						ty,
						ty,
					),
				)

				AssertValuesEqual(
					t,
					inter,
					expected,
					inter.Globals.Get("result").GetValue(),
				)
				AssertValuesEqual(
					t,
					inter,
					interpreter.TrueValue,
					inter.Globals.Get("roundTripEqual").GetValue(),
				)
			})
		}
	}

	for ty, tests := range invalidTests {
		for _, value := range tests {
			t.Run(fmt.Sprintf("%s: %s", ty, value), func(t *testing.T) {
				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
	                      let result: %s? = %s.fromBigEndianBytes(%s)
	                    `,
						ty,
						ty,
						value,
					),
				)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NilValue{},
					inter.Globals.Get("result").GetValue(),
				)
			})
		}
	}
}
