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
		err = generator.ReadRandom(buffer[:])
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

			var randomUint64 uint64
			if returnIntegerType == sema.UInt8Type || returnIntegerType == sema.UInt16Type ||
				returnIntegerType == sema.UInt32Type || returnIntegerType == sema.UInt64Type ||
				returnIntegerType == sema.Word8Type || returnIntegerType == sema.Word16Type ||
				returnIntegerType == sema.Word32Type || returnIntegerType == sema.Word64Type {
				randomUint64 = getUint64RandomNumber(generator, returnIntegerType, moduloValue)
			}
			var randomBig *big.Int
			if returnIntegerType == sema.UInt128Type || returnIntegerType == sema.UInt256Type ||
				returnIntegerType == sema.Word128Type || returnIntegerType == sema.Word256Type {
				randomBig = getBigRandomNumber(generator, returnIntegerType, moduloValue)
			}

			switch returnIntegerType {
			// UInt*
			case sema.UInt8Type:
				return interpreter.NewUInt8Value(
					inter,
					func() uint8 {
						return uint8(randomUint64)
					},
				)
			case sema.UInt16Type:
				return interpreter.NewUInt16Value(
					inter,
					func() uint16 {
						return uint16(randomUint64)
					},
				)
			case sema.UInt32Type:
				return interpreter.NewUInt32Value(
					inter,
					func() uint32 {
						return uint32(randomUint64)
					},
				)
			case sema.UInt64Type:
				return interpreter.NewUInt64Value(
					inter,
					func() uint64 {
						return randomUint64
					},
				)
			case sema.UInt128Type:
				return interpreter.NewUInt128ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)
			case sema.UInt256Type:
				return interpreter.NewUInt256ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)

			// Word*
			case sema.Word8Type:
				return interpreter.NewWord8Value(
					inter,
					func() uint8 {
						return uint8(randomUint64)
					},
				)
			case sema.Word16Type:
				return interpreter.NewWord16Value(
					inter,
					func() uint16 {
						return uint16(randomUint64)
					},
				)
			case sema.Word32Type:
				return interpreter.NewWord32Value(
					inter,
					func() uint32 {
						return uint32(randomUint64)
					},
				)
			case sema.Word64Type:
				return interpreter.NewWord64Value(
					inter,
					func() uint64 {
						return randomUint64
					},
				)
			case sema.Word128Type:
				return interpreter.NewWord128ValueFromBigInt(
					inter,
					func() *big.Int {
						return randomBig
					},
				)
			case sema.Word256Type:
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

// cases of a random number of size 1, 2, 4 or 8 bytes can be all treated
// by the same function, based on the uint64 type.
// Although the final output is a `uint64`, it can be safely
// casted into the desired output type because the extra bytes are guaranteed
// to be zeros.
func getUint64RandomNumber(
	generator RandomGenerator,
	ty sema.Type,
	moduloArg interpreter.Value,
) uint64 {
	// map for a quick type to byte-size lookup
	typeToBytes := map[sema.Type]int{
		sema.UInt8Type:  1,
		sema.UInt16Type: 2,
		sema.UInt32Type: 4,
		sema.UInt64Type: 8,
		sema.Word8Type:  1,
		sema.Word16Type: 2,
		sema.Word32Type: 4,
		sema.Word64Type: 8,
	}

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

	if modulo == 0 {
		panic(ZeroModuloError)
	}

	// `max` is the maximum value that can be returned
	max := modulo - 1
	// get a bit mask that covers all `max` bits,
	// and count the byte size of `max`
	mask := uint64(0)
	bitSize := 0
	for max&mask != max {
		bitSize++
		mask = (mask << 1) | 1
	}
	byteSize := (bitSize + 7) >> 3
	// use the reject-sample method to avoid the modulo bias.
	// the function isn't constant-time in this case and may take longer than computing
	// a modular reduction.
	// However, sampling exactly the size of `max` in bits makes the loop return fast:
	// loop returns after (k) iterations with a probability at most 1-(1/2)^k.
	//
	// (a different approach would be to pull 128 bits extra bits from the random source
	// and use big number modular reduction by `modulo`)
	random := modulo
	for random > max {
		// only generate `byteSize` random bytes
		getRandomBytes(buffer[bufferSize-byteSize:], generator)
		// big endianness must be used in this case
		random = binary.BigEndian.Uint64(buffer[:])
		random &= mask // adjust to the size of max in bits
	}
	return random
}

func getBigRandomNumber(
	generator RandomGenerator,
	ty sema.Type,
	moduloArg interpreter.Value,
) *big.Int {
	// map for a quick type to bytes lookup
	typeToBytes := map[sema.Type]int{
		sema.UInt128Type: 16,
		sema.UInt256Type: 32,
		sema.Word128Type: 16,
		sema.Word256Type: 32,
	}

	// buffer to get random bytes from the generator
	// 32 is the size of the largest type supported
	const bufferSize = 32
	var buffer [bufferSize]byte
	// case where no modulo argument was provided
	if moduloArg == nil {
		getRandomBytes(buffer[:typeToBytes[ty]], generator)
		// SetBytes considers big endianness
		return new(big.Int).SetBytes(buffer[:typeToBytes[ty]])
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

	if modulo.Sign() == 0 {
		panic(ZeroModuloError)
	}

	// `max` is the maximum value that can be returned (modulo - 1)
	one := big.NewInt(1)
	max := new(big.Int).Sub(modulo, one)
	// count the byte size of `max`
	bitSize := max.BitLen()
	byteSize := (bitSize + 7) >> 3
	// get a bit mask that covers all `max` (1 << bitSize) -1
	mask := new(big.Int).Lsh(one, uint(bitSize))
	mask.Sub(mask, one)

	// use the reject-sample method to avoid the modulo bias.
	// the function isn't constant-time in this case and may take longer than computing
	// a modular reduction.
	// However, sampling exactly the size of `max` in bits makes the loop return fast:
	// loop returns after (k) iterations with a probability at most 1-(1/2)^k.
	//
	// (a different approach would be to pull 128 bits extra bits from the random source
	// and use big number modular reduction by `modulo`)
	random := new(big.Int).Set(modulo)
	for random.Cmp(max) > 0 {
		// only generate `byteSize` random bytes
		getRandomBytes(buffer[32-byteSize:], generator)
		// big endianness must be used in this case
		random.SetBytes(buffer[:])
		random.And(random, mask) // adjust to the size of max in bits
	}
	return random
}
