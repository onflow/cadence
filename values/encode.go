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

package values

import (
	"math"
	"math/big"

	"github.com/onflow/atree"
)

// Cadence needs to encode different kinds of objects in CBOR, for instance,
// dictionaries, structs, resources, etc.
//
// However, CBOR only provides one native map type, and no support
// for directly representing e.g. structs or resources.
//
// To be able to encode/decode such semantically different values,
// we define custom CBOR tags.

// !!! *WARNING* !!!
//
// Only add new fields to encoded structs by
// appending new fields with the next highest key.
//
// DO *NOT* REPLACE EXISTING FIELDS!

const CBORTagBase = 128

// !!! *WARNING* !!!
//
// Only add new types by:
// - replacing existing placeholders (`_`) with new types
// - appending new types
//
// Only remove types by:
// - replace existing types with a placeholder `_`
//
// DO *NOT* REPLACE EXISTING TYPES!
// DO *NOT* ADD NEW TYPES IN BETWEEN!

const (
	CBORTagVoidValue = CBORTagBase + iota
	_                // DO *NOT* REPLACE. Previously used for dictionary values
	CBORTagSomeValue
	CBORTagAddressValue
	CBORTagCompositeValue
	CBORTagTypeValue
	_ // DO *NOT* REPLACE. Previously used for array values
	CBORTagStringValue
	CBORTagCharacterValue
	CBORTagSomeValueWithNestedLevels
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_
	_

	// Int*
	CBORTagIntValue
	CBORTagInt8Value
	CBORTagInt16Value
	CBORTagInt32Value
	CBORTagInt64Value
	CBORTagInt128Value
	CBORTagInt256Value
	_

	// UInt*
	CBORTagUIntValue
	CBORTagUInt8Value
	CBORTagUInt16Value
	CBORTagUInt32Value
	CBORTagUInt64Value
	CBORTagUInt128Value
	CBORTagUInt256Value
	_

	// Word*
	_
	CBORTagWord8Value
	CBORTagWord16Value
	CBORTagWord32Value
	CBORTagWord64Value
	CBORTagWord128Value
	CBORTagWord256Value
	_

	// Fix*
	_
	_ // future: Fix8
	_ // future: Fix16
	_ // future: Fix32
	CBORTagFix64Value
	_ // future: Fix128
	_ // future: Fix256
	_

	// UFix*
	_
	_ // future: UFix8
	_ // future: UFix16
	_ // future: UFix32
	CBORTagUFix64Value
	_ // future: UFix128
	_ // future: UFix256
	_

	// Locations
	CBORTagAddressLocation
	CBORTagStringLocation
	CBORTagIdentifierLocation
	CBORTagTransactionLocation
	CBORTagScriptLocation
	_
	_
	_

	// Storage

	CBORTagPathValue
	// Deprecated: CBORTagPathCapabilityValue
	CBORTagPathCapabilityValue
	_ // DO NOT REPLACE! used to be used for storage references
	// Deprecated: CBORTagPathLinkValue
	CBORTagPathLinkValue
	CBORTagPublishedValue
	// Deprecated: CBORTagAccountLinkValue
	CBORTagAccountLinkValue
	CBORTagStorageCapabilityControllerValue
	CBORTagAccountCapabilityControllerValue
	CBORTagCapabilityValue
	_
	_
	_

	// Static Types
	CBORTagPrimitiveStaticType
	CBORTagCompositeStaticType
	CBORTagInterfaceStaticType
	CBORTagVariableSizedStaticType
	CBORTagConstantSizedStaticType
	CBORTagDictionaryStaticType
	CBORTagOptionalStaticType
	CBORTagReferenceStaticType
	CBORTagIntersectionStaticType
	CBORTagCapabilityStaticType
	CBORTagUnauthorizedStaticAuthorization
	CBORTagEntitlementMapStaticAuthorization
	CBORTagEntitlementSetStaticAuthorization
	CBORTagInaccessibleStaticAuthorization

	_
	_
	_
	_

	CBORTagInclusiveRangeStaticType

	// !!! *WARNING* !!!
	// ADD NEW TYPES *BEFORE* THIS WARNING.
	// DO *NOT* ADD NEW TYPES AFTER THIS LINE!
	CBORTag_Count
)

const CBORTagSize = 2

func GetBigIntCBORSize(v *big.Int) uint32 {
	sign := v.Sign()
	if sign < 0 {
		v = new(big.Int).Abs(v)
		v.Sub(v, bigOne)
	}

	// tag number + bytes
	return 1 + GetBytesCBORSize(v.Bytes())
}

func GetIntCBORSize(v int64) uint32 {
	if v < 0 {
		return GetUintCBORSize(uint64(-v - 1))
	}
	return GetUintCBORSize(uint64(v))
}

func GetUintCBORSize(v uint64) uint32 {
	if v <= 23 {
		return 1
	}
	if v <= math.MaxUint8 {
		return 2
	}
	if v <= math.MaxUint16 {
		return 3
	}
	if v <= math.MaxUint32 {
		return 5
	}
	return 9
}

func GetBytesCBORSize(b []byte) uint32 {
	length := len(b)
	if length == 0 {
		return 1
	}
	return GetUintCBORSize(uint64(length)) + uint32(length)
}

// MaybeLargeImmutableStorable either returns the given immutable atree.Storable
// if it can be stored inline inside its parent container,
// or else stores it in a separate slab and returns an atree.SlabIDStorable.
func MaybeLargeImmutableStorable(
	storable atree.Storable,
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (
	atree.Storable,
	error,
) {

	if uint64(storable.ByteSize()) < maxInlineSize {
		return storable, nil
	}

	return atree.NewStorableSlab(storage, address, storable)
}
