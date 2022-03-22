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
				interpreter.NewStringValue("42"),
				inter.Globals["y"].GetValue(),
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
			interpreter.NewStringValue("0x0000000000000042"),
			inter.Globals["y"].GetValue(),
		)
	})

	for _, ty := range sema.AllFixedPointTypes {

		t.Run(ty.String(), func(t *testing.T) {

			var literal string
			var expected interpreter.Value

			isSigned := sema.IsSubType(ty, sema.SignedFixedPointType)

			if isSigned {
				literal = "-12.34"
				expected = interpreter.NewStringValue("-12.34000000")
			} else {
				literal = "12.34"
				expected = interpreter.NewStringValue("12.34000000")
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
				inter.Globals["y"].GetValue(),
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
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.Address{},
				interpreter.UInt8Value(0x0),
				interpreter.UInt8Value(0x0),
				interpreter.UInt8Value(0x0),
				interpreter.UInt8Value(0x0),
				interpreter.UInt8Value(0x0),
				interpreter.UInt8Value(0x12),
				interpreter.UInt8Value(0x34),
				interpreter.UInt8Value(0x56),
			),
			inter.Globals["y"].GetValue(),
		)
	})
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
			"200": {200},
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
					inter.Globals["result"].GetValue(),
				)
			})
		}
	}
}
