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
		// EncodeType as two's complement
		twosComplement := new(big.Int).Neg(bigInt)
		twosComplement.Sub(twosComplement, big.NewInt(1))
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

func UnsignedBigIntToBigEndianBytes(bigInt *big.Int) []byte {

	switch bigInt.Sign() {
	case -1:
		panic(errors.NewUnreachableError())

	case 0:
		return []byte{0}

	case 1:
		return bigInt.Bytes()

	default:
		panic(errors.NewUnreachableError())
	}
}
