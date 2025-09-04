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

package constant

import (
	"github.com/onflow/cadence/sema"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=Kind

type Kind uint8

const (
	Unknown Kind = iota
	String
	Character
	Address
	_
	_
	_
	_

	// Int*
	Int
	Int8
	Int16
	Int32
	Int64
	Int128
	Int256
	_

	// UInt*
	UInt
	UInt8
	UInt16
	UInt32
	UInt64
	UInt128
	UInt256
	_

	// Word*
	_
	Word8
	Word16
	Word32
	Word64
	Word128
	Word256
	_

	// Fix*
	_
	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	Fix64
	Fix128
	_ // future: Fix256
	_

	// UFix*
	_
	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	UFix64
	UFix128
	_ // future: UFix256
)

func FromSemaType(ty sema.Type) Kind {
	switch ty {
	case sema.StringType:
		return String
	case sema.CharacterType:
		return Character

	case sema.TheAddressType:
		return Address

	// Int*
	case sema.IntType, sema.IntegerType, sema.SignedIntegerType:
		return Int
	case sema.Int8Type:
		return Int8
	case sema.Int16Type:
		return Int16
	case sema.Int32Type:
		return Int32
	case sema.Int64Type:
		return Int64
	case sema.Int128Type:
		return Int128
	case sema.Int256Type:
		return Int256

	// UInt*
	case sema.UIntType:
		return UInt
	case sema.UInt8Type:
		return UInt8
	case sema.UInt16Type:
		return UInt16
	case sema.UInt32Type:
		return UInt32
	case sema.UInt64Type:
		return UInt64
	case sema.UInt128Type:
		return UInt128
	case sema.UInt256Type, sema.FixedSizeUnsignedIntegerType:
		return UInt256

	// Word*
	case sema.Word8Type:
		return Word8
	case sema.Word16Type:
		return Word16
	case sema.Word32Type:
		return Word32
	case sema.Word64Type:
		return Word64
	case sema.Word128Type:
		return Word128
	case sema.Word256Type:
		return Word256

	// Fix*
	case sema.Fix64Type:
		return Fix64
	case sema.Fix128Type:
		return Fix128

	case sema.UFix64Type:
		return UFix64
	case sema.UFix128Type:
		return UFix128

	default:
		return Unknown
	}
}
