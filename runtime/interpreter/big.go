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

	twoscomplement "github.com/ElrondNetwork/big-int-util/twos-complement"

	"github.com/onflow/cadence/runtime/errors"
)

func SignedBigIntToBigEndianBytes(bigInt *big.Int) []byte {
	bytes := twoscomplement.ToBytes(bigInt)
	if bigInt.Sign() == 0 {
		bytes = []byte{0}
	}
	return bytes
}

// Return the BigInt encoded as a big-endian byte array of size `sizeInBytes`.
// The value inside `bigInt` must fit inside 8^sizeInBytes bits.
func SignedBigIntToSizedBigEndianBytes(bigInt *big.Int, sizeInBytes int) []byte {
	res, _ := twoscomplement.ToBytesOfLength(bigInt, sizeInBytes)
	return res
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

func UnsignedBigIntToSizedBigEndianBytes(bigInt *big.Int, sizeInBytes int) []byte {
	buf := make([]byte, sizeInBytes)

	switch bigInt.Sign() {
	case -1:
		panic(errors.NewUnreachableError())

	case 0:
		return buf

	case 1:
		return bigInt.FillBytes(buf)

	default:
		panic(errors.NewUnreachableError())
	}
}
