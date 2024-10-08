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

package interpreter

import (
	"math/big"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

func SignedBigIntToBigEndianBytes(bigInt *big.Int) []byte {

	switch bigInt.Sign() {
	case -1:
		// Encode as two's complement
		twosComplement := new(big.Int).Neg(bigInt)
		twosComplement.Sub(twosComplement, bigOne)
		bytes := twosComplement.Bytes()
		for i := range bytes {
			bytes[i] ^= 0xff
		}
		// Pad with 0xFF to prevent misinterpretation as positive
		if len(bytes) == 0 || bytes[0]&0x80 == 0 {
			return append([]byte{0xff}, bytes...)
		}
		return bytes

	case 0:
		return []byte{0}

	case 1:
		bytes := bigInt.Bytes()
		// Pad with 0x0 to prevent misinterpretation as negative
		if len(bytes) > 0 && bytes[0]&0x80 != 0 {
			return append([]byte{0x0}, bytes...)
		}
		return bytes

	default:
		panic(errors.NewUnreachableError())
	}
}

func SignedBigIntToSizedBigEndianBytes(bigInt *big.Int, sizeInBytes uint) []byte {
	// todo use uint64 for fewer iterations?
	buf := make([]byte, sizeInBytes)

	switch bigInt.Sign() {
	case -1:
		increm := big.NewInt(0)
		increm = increm.Add(bigInt, bigOne)
		bytes := increm.Bytes()
		offset := len(buf) - len(bytes)
		for i := 0; i < offset; i++ {
			buf[i] = 0xff // sign extend
		}

		offsetBuf := buf[offset:]
		for i := 0; i < len(bytes); i++ {
			offsetBuf[i] = ^bytes[i]
		}
	case 0:
		break
	case 1:
		bigInt.FillBytes(buf)
	default:
		panic(errors.NewUnreachableError())
	}
	return buf
}

func UnsignedBigIntToBigEndianBytes(bigInt *big.Int) []byte {

	switch bigInt.Sign() {
	case 0:
		return []byte{0}

	case 1:
		return bigInt.Bytes()

	default:
		panic(errors.NewUnexpectedError("Negative sign on big.Int with unsigned constraint"))
	}
}

func UnsignedBigIntToSizedBigEndianBytes(bigInt *big.Int, sizeInBytes uint) []byte {
	buf := make([]byte, sizeInBytes)
	switch bigInt.Sign() {
	case 0:
		return buf
	case 1:
		bigInt.FillBytes(buf)
		return buf
	default:
		panic(errors.NewUnexpectedError("Negative sign on big.Int with unsigned constraint"))
	}
}

func BigEndianBytesToSignedBigInt(b []byte) *big.Int {
	// Check for special cases of 0 and 1
	if len(b) == 1 && b[0] == 0 {
		return big.NewInt(0)
	} else if len(b) == 1 && b[0] <= 0x7f {
		return big.NewInt(int64(b[0]))
	}

	// Check if number is negative (high bit set)
	isNegative := b[0]&0x80 != 0

	// Perform two's complement transformation if negative
	if isNegative {
		for i := range b {
			b[i] ^= 0xff
		}
		result := new(big.Int).SetBytes(b)
		result.Add(result, big.NewInt(1)).Neg(result)
		return result
	}

	// Positive number
	return new(big.Int).SetBytes(b)
}

func BigEndianBytesToUnsignedBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

func BigIntSqrt(interpreter *Interpreter, value *big.Int, locationRange LocationRange) UFix64Value {
	if value.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if value.Cmp(sema.MaxSquareIntegerBig) == 1 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	// Once we reach here, Cadence integer values are guaranteed to fit into
	// floating-point values with 256 bit precision _without_ truncation.
	// This is because of the above check with sema.MaxSquareIntegerBig.
	valueFloat := new(big.Float).SetPrec(256).SetInt(value)
	res := new(big.Float).SetPrec(256).SetMode(big.ToZero).Sqrt(valueFloat)
	res.Mul(res, new(big.Float).SetPrec(256).SetInt(sema.Fix64FactorBig))

	// Converting the result to a fixed-point number, we are conceptually converting it to an integer
	// IEEE 754 specifies different rounding modes https:	//en.wikipedia.org/wiki/IEEE_754#Rounding_rules
	// We follow the "Rationale for International Standard -- Programming Languages -- C", Revision 5.10, April-2003:
	//   > Section 6.3.1.5 Real floating types:
	//   > When a finite value of real floating type is converted to an integer type other than Bool,
	//   > the fractional part is discarded (i.e., the value is truncated toward zero). If the value
	//   > of the integral part cannot be represented by the integer type, the behavior is undefined.
	// For details, see
	// https:	//wiki.sei.cmu.edu/confluence/display/c/FLP34-C.+Ensure+that+floating-point+conversions+are+within+range+of+the+new+type
	resInt := new(big.Int)
	res.Int(resInt)
	if !resInt.IsUint64() {
		// We checked for overflow above, so we shouldn't hit this.
		panic(errors.NewUnreachableError())
	}

	valueGetter := func() uint64 {
		return resInt.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}
