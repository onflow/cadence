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

package stdlib

import (
	"encoding/binary"
	"math/big"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const revertibleRandomFunctionDocString = `
Returns a pseudo-random number.

NOTE: The use of this function is unsafe if not used correctly.

Follow best practices to prevent security issues when using this function
`

var revertibleRandomFunctionType = func() *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		Name:      "T",
		TypeBound: sema.FixedSizeUnsignedIntegerType,
	}

	typeAnnotation := sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Identifier:     "modulo",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: typeAnnotation,
		// `modulo` parameter is optional
		Arity: &sema.Arity{Min: 0, Max: 1},
	}
}()

type RandomGenerator interface {
	// ReadRandom reads pseudo-random bytes into the input slice, using distributed randomness.
	// The number of bytes read is equal to the length of input slice.
	ReadRandom([]byte) error
}

func getRandomBytes(generator RandomGenerator, numBytes int) []byte {
	buffer := make([]byte, numBytes)

	var err error
	errors.WrapPanic(func() {
		err = generator.ReadRandom(buffer[:])
	})
	if err != nil {
		panic(interpreter.WrappedExternalError(err))
	}

	return buffer
}

func NewRevertibleRandomFunction(generator RandomGenerator) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"revertibleRandom",
		revertibleRandomFunctionType,
		revertibleRandomFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			// TODO: Check if invocation has an argument and implement modulo operation.

			returnIntegerType := invocation.TypeParameterTypes.Oldest().Value
			returnIntegerStaticType := interpreter.ConvertSemaToStaticType(inter, returnIntegerType)

			switch returnIntegerStaticType {
			// UInt*
			case interpreter.PrimitiveStaticTypeUInt8:
				return interpreter.NewUInt8Value(
					inter,
					func() uint8 {
						return getRandomBytes(generator, 1)[0]
					},
				)
			case interpreter.PrimitiveStaticTypeUInt16:
				return interpreter.NewUInt16Value(
					inter,
					func() uint16 {
						return binary.LittleEndian.Uint16(getRandomBytes(generator, 2))
					},
				)
			case interpreter.PrimitiveStaticTypeUInt32:
				return interpreter.NewUInt32Value(
					inter,
					func() uint32 {
						return binary.LittleEndian.Uint32(getRandomBytes(generator, 4))
					},
				)
			case interpreter.PrimitiveStaticTypeUInt64:
				return interpreter.NewUInt64Value(
					inter,
					func() uint64 {
						return binary.LittleEndian.Uint64(getRandomBytes(generator, 8))
					},
				)
			case interpreter.PrimitiveStaticTypeUInt128:
				return interpreter.NewUInt128ValueFromBigInt(
					inter,
					func() *big.Int {
						buffer := getRandomBytes(generator, 16)
						return interpreter.LittleEndianBytesToUnsignedBigInt(buffer)
					},
				)
			case interpreter.PrimitiveStaticTypeUInt256:
				return interpreter.NewUInt256ValueFromBigInt(
					inter,
					func() *big.Int {
						buffer := getRandomBytes(generator, 32)
						return interpreter.LittleEndianBytesToUnsignedBigInt(buffer)
					},
				)

			// Word*
			case interpreter.PrimitiveStaticTypeWord8:
				return interpreter.NewWord8Value(
					inter,
					func() uint8 {
						return getRandomBytes(generator, 1)[0]
					},
				)
			case interpreter.PrimitiveStaticTypeWord16:
				return interpreter.NewWord16Value(
					inter,
					func() uint16 {
						return binary.LittleEndian.Uint16(getRandomBytes(generator, 2))
					},
				)
			case interpreter.PrimitiveStaticTypeWord32:
				return interpreter.NewWord32Value(
					inter,
					func() uint32 {
						return binary.LittleEndian.Uint32(getRandomBytes(generator, 4))
					},
				)
			case interpreter.PrimitiveStaticTypeWord64:
				return interpreter.NewWord64Value(
					inter,
					func() uint64 {
						return binary.LittleEndian.Uint64(getRandomBytes(generator, 8))
					},
				)
			case interpreter.PrimitiveStaticTypeWord128:
				return interpreter.NewWord128ValueFromBigInt(
					inter,
					func() *big.Int {
						buffer := getRandomBytes(generator, 16)
						return interpreter.LittleEndianBytesToUnsignedBigInt(buffer)
					},
				)
			case interpreter.PrimitiveStaticTypeWord256:
				return interpreter.NewWord256ValueFromBigInt(
					inter,
					func() *big.Int {
						buffer := getRandomBytes(generator, 32)
						return interpreter.LittleEndianBytesToUnsignedBigInt(buffer)
					},
				)

			default:
				// Checker should prevent this.
				panic(errors.NewUnreachableError())
			}
		},
	)
}
