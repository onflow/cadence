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

func getRandomBytes(buffer []byte, generator RandomGenerator) {
	var err error
	errors.WrapPanic(func() {
		err = generator.ReadRandom(buffer)
	})
	if err != nil {
		panic(interpreter.WrappedExternalError(err))
	}
}

var ZeroModuloError = errors.NewDefaultUserError("modulo argument cannot be zero")

func NewRevertibleRandomFunction(generator RandomGenerator) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"revertibleRandom",
		revertibleRandomFunctionType,
		revertibleRandomFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			returnIntegerType := invocation.TypeParameterTypes.Oldest().Value

			// arguments should be 0 or 1 at this point
			var moduloValue interpreter.Value
			if len(invocation.Arguments) == 1 {
				moduloValue = invocation.Arguments[0]
			}

			switch returnIntegerType {
			// UInt*
			case sema.UInt8Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt8Value(
					inter,
					func() uint8 {
						return uint8(randomUint64)
					},
				)
			case sema.UInt16Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt16Value(
					inter,
					func() uint16 {
						return uint16(randomUint64)
					},
				)
			case sema.UInt32Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt32Value(
					inter,
					func() uint32 {
						return uint32(randomUint64)
					},
				)
			case sema.UInt64Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt64Value(
					inter,
					func() uint64 {
						return randomUint64
					},
				)
			case sema.UInt128Type:
				randomBig := getBigRandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt128ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)
			case sema.UInt256Type:
				randomBig := getBigRandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewUInt256ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)

			// Word*
			case sema.Word8Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord8Value(
					inter,
					func() uint8 {
						return uint8(randomUint64)
					},
				)
			case sema.Word16Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord16Value(
					inter,
					func() uint16 {
						return uint16(randomUint64)
					},
				)
			case sema.Word32Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord32Value(
					inter,
					func() uint32 {
						return uint32(randomUint64)
					},
				)
			case sema.Word64Type:
				randomUint64 := getUint64RandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord64Value(
					inter,
					func() uint64 {
						return randomUint64
					},
				)
			case sema.Word128Type:
				randomBig := getBigRandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord128ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)
			case sema.Word256Type:
				randomBig := getBigRandomNumber(generator, returnIntegerType, moduloValue)
				return interpreter.NewWord256ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)

			default:
				// Checker should prevent this.
				panic(errors.NewUnreachableError())
			}
		},
	)
}

// map for a quick type to byte-size lookup
var typeToBytes = map[sema.Type]int{
	sema.UInt8Type: 1, sema.UInt16Type: 2, sema.UInt32Type: 4, sema.UInt64Type: 8,
	sema.Word8Type: 1, sema.Word16Type: 2, sema.Word32Type: 4, sema.Word64Type: 8,
	sema.UInt128Type: 16, sema.UInt256Type: 32, sema.Word128Type: 16, sema.Word256Type: 32,
}

// cases of a random number of size 8 bytes or less can be all treated
// by the same function, based on the uint64 type.
// Although the final output is a `uint64`, it can be safely
// casted into the desired output type because the extra bytes are guaranteed
// to be zeros.
func getUint64RandomNumber(
	generator RandomGenerator,
	ty sema.Type,
	moduloArg interpreter.Value,
) uint64 {

	// buffer to get random bytes from the generator
	// 8 is the size of the largest type supported
	const bufferSize = 8
	var buffer [bufferSize]byte

	// case where no modulo argument was provided
	if moduloArg == nil {
		bytes := typeToBytes[ty]
		getRandomBytes(buffer[bufferSize-bytes:], generator)
		return binary.BigEndian.Uint64(buffer[:])
	}

	var ok bool
	var modulo uint64

	switch ty {
	case sema.UInt8Type:
		var moduloVal interpreter.UInt8Value
		moduloVal, ok = moduloArg.(interpreter.UInt8Value)
		modulo = uint64(moduloVal)
	case sema.UInt16Type:
		var moduloVal interpreter.UInt16Value
		moduloVal, ok = moduloArg.(interpreter.UInt16Value)
		modulo = uint64(moduloVal)
	case sema.UInt32Type:
		var moduloVal interpreter.UInt32Value
		moduloVal, ok = moduloArg.(interpreter.UInt32Value)
		modulo = uint64(moduloVal)
	case sema.UInt64Type:
		var moduloVal interpreter.UInt64Value
		moduloVal, ok = moduloArg.(interpreter.UInt64Value)
		modulo = uint64(moduloVal)
	case sema.Word8Type:
		var moduloVal interpreter.Word8Value
		moduloVal, ok = moduloArg.(interpreter.Word8Value)
		modulo = uint64(moduloVal)
	case sema.Word16Type:
		var moduloVal interpreter.Word16Value
		moduloVal, ok = moduloArg.(interpreter.Word16Value)
		modulo = uint64(moduloVal)
	case sema.Word32Type:
		var moduloVal interpreter.Word32Value
		moduloVal, ok = moduloArg.(interpreter.Word32Value)
		modulo = uint64(moduloVal)
	case sema.Word64Type:
		var moduloVal interpreter.Word64Value
		moduloVal, ok = moduloArg.(interpreter.Word64Value)
		modulo = uint64(moduloVal)
	default:
		// sanity check: shouldn't reach here
		panic(errors.NewUnreachableError())
	}

	if !ok {
		// checker should prevent this
		panic(errors.NewUnreachableError())
	}

	// user error if modulo is zero
	if modulo == 0 {
		panic(ZeroModuloError)
	}

	// `max` is the maximum value that can be returned
	max := modulo - 1
	// get a bit mask (0b11..11) that covers all `max` bits,
	// and count the byte size of `max`
	mask := uint64(0)
	bitSize := 0
	for max&mask != max {
		bitSize++
		mask = (mask << 1) | 1
	}
	byteSize := (bitSize + 7) >> 3

	// Generate a number less or equal than `max`.
	// use the reject-sample method to avoid the modulo bias.
	// the function isn't constant-time in this case and may take longer than computing
	// a modular reduction.
	// However, sampling exactly the size of `max` in bits makes the loop return fast:
	// loop returns after (k) iterations with a probability of at most 1-(1/2)^k.
	//
	// (a different approach would be to pull 128 bits more bits than the size of `max`
	// from the random generator and use big number reduction by `modulo`)
	random := modulo
	for random > max {
		// only generate `byteSize` random bytes
		getRandomBytes(buffer[bufferSize-byteSize:], generator)
		// big endianness must be used in this case
		random = binary.BigEndian.Uint64(buffer[:])
		// adjust to the size of max in bits
		random &= mask
	}
	return random
}

// cases of a random number of size larger than 8 bytes can be all treated
// by the same function, based on the big.Int type.
func getBigRandomNumber(
	generator RandomGenerator,
	ty sema.Type,
	moduloArg interpreter.Value,
) *big.Int {

	// buffer to get random bytes from the generator
	// 32 is the size of the largest type supported
	const bufferSize = 32
	var buffer [bufferSize]byte
	// case where no modulo argument was provided
	if moduloArg == nil {
		bytes := typeToBytes[ty]
		getRandomBytes(buffer[:bytes], generator)
		// SetBytes considers big endianness (although little endian could be used too)
		return new(big.Int).SetBytes(buffer[:bytes])
	}

	var ok bool
	var modulo *big.Int

	switch ty {
	case sema.UInt128Type:
		var moduloVal interpreter.UInt128Value
		moduloVal, ok = moduloArg.(interpreter.UInt128Value)
		modulo = moduloVal.BigInt
	case sema.UInt256Type:
		var moduloVal interpreter.UInt256Value
		moduloVal, ok = moduloArg.(interpreter.UInt256Value)
		modulo = moduloVal.BigInt
	case sema.Word128Type:
		var moduloVal interpreter.Word128Value
		moduloVal, ok = moduloArg.(interpreter.Word128Value)
		modulo = moduloVal.BigInt
	case sema.Word256Type:
		var moduloVal interpreter.Word256Value
		moduloVal, ok = moduloArg.(interpreter.Word256Value)
		modulo = moduloVal.BigInt
	default:
		// sanity check: shouldn't reach here
		panic(errors.NewUnreachableError())
	}

	if !ok {
		// checker should prevent this
		panic(errors.NewUnreachableError())
	}

	// user error if modulo is zero
	if modulo.Sign() == 0 {
		panic(ZeroModuloError)
	}

	// `max` is the maximum value that can be returned (modulo - 1)
	one := big.NewInt(1)
	max := new(big.Int).Sub(modulo, one)
	// count the byte size of `max`
	bitSize := max.BitLen()
	byteSize := (bitSize + 7) >> 3
	// get a bit mask (0b11..11) that covers all `max`'s bits:
	// `mask` can be computed as:   (1 << bitSize) -1
	mask := new(big.Int).Lsh(one, uint(bitSize))
	mask.Sub(mask, one)

	// Generate a number less or equal than `max`
	// use the reject-sample method to avoid the modulo bias.
	// the function isn't constant-time in this case and may take longer than computing
	// a modular reduction.
	// However, sampling exactly the size of `max` in bits makes the loop return fast:
	// loop returns after (k) iterations with a probability of at most 1-(1/2)^k.
	//
	// (a different approach would be to pull 128 bits more bits than the size of `max`
	// from the random generator and use big number reduction by `modulo`)
	random := new(big.Int).Set(modulo)
	for random.Cmp(max) > 0 {
		// only generate `byteSize` random bytes
		getRandomBytes(buffer[:byteSize], generator)
		// big endianness is used for consistency (but little can be used too)
		random.SetBytes(buffer[:byteSize])
		// adjust to the size of max in bits
		random.And(random, mask)
	}
	return random
}
