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

package interpreter

import (
	"math/big"

	"github.com/onflow/cadence/runtime/errors"
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
			buf[i] = 255 // sign extend
		}
		for i := 0; i < len(buf)-offset; i++ {
			buf[i+offset] = ^bytes[i]
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
