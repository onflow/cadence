/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package sema

import (
	"fmt"

	"github.com/onflow/cadence/runtime/errors"
)

// TypeTag is a bitmask representation for types.
// Each type has a unique dedicated bit in the bitMask.
// The mask consist of two sections: lowerMask and the upperMask.
// Each section can represent 64-types. Currently only the lower mask is used.
//
type TypeTag struct {
	lowerMask uint64
	upperMask uint64
}

var allTypeTags = map[TypeTag]bool{}

func newTypeTagFromLowerMask(mask uint64) TypeTag {
	typeTag := TypeTag{
		lowerMask: mask,
		upperMask: 0,
	}

	if _, ok := allTypeTags[typeTag]; ok {
		panic(fmt.Errorf("duplicate type tag: %v", typeTag))
	}
	allTypeTags[typeTag] = true

	return typeTag
}

func (t TypeTag) Equals(tag TypeTag) bool {
	return t.lowerMask == tag.lowerMask && t.upperMask == tag.upperMask
}

func (t TypeTag) And(tag TypeTag) TypeTag {
	return TypeTag{
		lowerMask: t.lowerMask & tag.lowerMask,
		upperMask: t.upperMask & tag.upperMask,
	}
}

func (t TypeTag) Or(tag TypeTag) TypeTag {
	return TypeTag{
		lowerMask: t.lowerMask | tag.lowerMask,
		upperMask: t.upperMask | tag.upperMask,
	}
}

func (t TypeTag) Not() TypeTag {
	return TypeTag{
		lowerMask: ^t.lowerMask,
		upperMask: ^t.upperMask,
	}
}

func (t TypeTag) ContainsAny(typeTags ...TypeTag) bool {
	for _, tag := range typeTags {
		if t.And(tag).Equals(tag) {
			return true
		}
	}

	return false
}

func (t TypeTag) BelongsTo(typeTag TypeTag) bool {
	return typeTag.ContainsAny(t)
}

const neverTypeMask = 0

const (
	uint8TypeMask uint64 = 1 << iota
	uint16TypeMask
	uint32TypeMask
	uint64TypeMask
	uint128TypeMask
	uint256TypeMask

	int8TypeMask
	int16TypeMask
	int32TypeMask
	int64TypeMask
	int128TypeMask
	int256TypeMask

	word8TypeMask
	word16TypeMask
	word32TypeMask
	word64TypeMask

	fix64TypeMask
	ufix64TypeMask

	intTypeMask
	uIntTypeMask
	stringTypeMask
	characterTypeMask
	boolTypeMask
	nilTypeMask
	voidTypeMask
	addressTypeMask
	metaTypeMask
	anyStructTypeMask
	anyResourceTypeMask
	anyTypeMask

	pathTypeMask
	storagePathTypeMask
	capabilityPathTypeMask
	publicPathTypeMask
	privatePathTypeMask

	arrayTypeMask
	dictionaryTypeMask
	compositeTypeMask
	referenceTypeMask
	resourceTypeMask

	optionalTypeMask
	genericTypeMask
	functionTypeMask
	interfaceTypeMask
	transactionTypeMask
	restrictedTypeMask
	capabilityTypeMask

	invalidTypeMask
)

var (
	NeverTypeTag = newTypeTagFromLowerMask(neverTypeMask)

	UInt8TypeTag   = newTypeTagFromLowerMask(uint8TypeMask)
	UInt16TypeTag  = newTypeTagFromLowerMask(uint16TypeMask)
	UInt32TypeTag  = newTypeTagFromLowerMask(uint32TypeMask)
	UInt64TypeTag  = newTypeTagFromLowerMask(uint64TypeMask)
	UInt128TypeTag = newTypeTagFromLowerMask(uint128TypeMask)
	UInt256TypeTag = newTypeTagFromLowerMask(uint256TypeMask)

	Int8TypeTag   = newTypeTagFromLowerMask(int8TypeMask)
	Int16TypeTag  = newTypeTagFromLowerMask(int16TypeMask)
	Int32TypeTag  = newTypeTagFromLowerMask(int32TypeMask)
	Int64TypeTag  = newTypeTagFromLowerMask(int64TypeMask)
	Int128TypeTag = newTypeTagFromLowerMask(int128TypeMask)
	Int256TypeTag = newTypeTagFromLowerMask(int256TypeMask)

	Word8TypeTag  = newTypeTagFromLowerMask(word8TypeMask)
	Word16TypeTag = newTypeTagFromLowerMask(word16TypeMask)
	Word32TypeTag = newTypeTagFromLowerMask(word32TypeMask)
	Word64TypeTag = newTypeTagFromLowerMask(word64TypeMask)

	Fix64TypeTag  = newTypeTagFromLowerMask(fix64TypeMask)
	UFix64TypeTag = newTypeTagFromLowerMask(ufix64TypeMask)

	IntTypeTag       = newTypeTagFromLowerMask(intTypeMask)
	UIntTypeTag      = newTypeTagFromLowerMask(uIntTypeMask)
	StringTypeTag    = newTypeTagFromLowerMask(stringTypeMask)
	CharacterTypeTag = newTypeTagFromLowerMask(characterTypeMask)
	BoolTypeTag      = newTypeTagFromLowerMask(boolTypeMask)
	NilTypeTag       = newTypeTagFromLowerMask(nilTypeMask)
	VoidTypeTag      = newTypeTagFromLowerMask(voidTypeMask)
	AddressTypeTag   = newTypeTagFromLowerMask(addressTypeMask)
	MetaTypeTag      = newTypeTagFromLowerMask(metaTypeMask)

	PathTypeTag           = newTypeTagFromLowerMask(pathTypeMask)
	StoragePathTypeTag    = newTypeTagFromLowerMask(storagePathTypeMask)
	CapabilityPathTypeTag = newTypeTagFromLowerMask(capabilityPathTypeMask)
	PublicPathTypeTag     = newTypeTagFromLowerMask(publicPathTypeMask)
	PrivatePathTypeTag    = newTypeTagFromLowerMask(privatePathTypeMask)

	ArrayTypeTag      = newTypeTagFromLowerMask(arrayTypeMask)
	DictionaryTypeTag = newTypeTagFromLowerMask(dictionaryTypeMask)
	CompositeTypeTag  = newTypeTagFromLowerMask(compositeTypeMask)
	ReferenceTypeTag  = newTypeTagFromLowerMask(referenceTypeMask)
	ResourceTypeTag   = newTypeTagFromLowerMask(resourceTypeMask)

	OptionalTypeTag    = newTypeTagFromLowerMask(optionalTypeMask)
	GenericTypeTag     = newTypeTagFromLowerMask(genericTypeMask)
	FunctionTypeTag    = newTypeTagFromLowerMask(functionTypeMask)
	InterfaceTypeTag   = newTypeTagFromLowerMask(interfaceTypeMask)
	TransactionTypeTag = newTypeTagFromLowerMask(transactionTypeMask)
	RestrictedTypeTag  = newTypeTagFromLowerMask(restrictedTypeMask)
	CapabilityTypeTag  = newTypeTagFromLowerMask(capabilityTypeMask)

	InvalidTypeTag = newTypeTagFromLowerMask(invalidTypeMask)

	// Super types

	SignedIntTypeTag = IntTypeTag.
				Or(Int8TypeTag).
				Or(Int16TypeTag).
				Or(Int32TypeTag).
				Or(Int64TypeTag).
				Or(Int128TypeTag).
				Or(Int256TypeTag)

	UnsignedIntTypeTag = UIntTypeTag.
				Or(UInt8TypeTag).
				Or(UInt16TypeTag).
				Or(UInt32TypeTag).
				Or(UInt64TypeTag).
				Or(UInt128TypeTag).
				Or(UInt256TypeTag)

	IntegerTypeTag = SignedIntTypeTag.Or(UnsignedIntTypeTag)

	AnyStructTypeTag = newTypeTagFromLowerMask(anyStructTypeMask).
				Or(NeverTypeTag).
				Or(IntegerTypeTag).
				Or(StringTypeTag).
				Or(ArrayTypeTag).
				Or(DictionaryTypeTag).
				Or(CompositeTypeTag).
				Or(ReferenceTypeTag).
				Or(NilTypeTag)

	AnyResourceTypeTag = newTypeTagFromLowerMask(anyResourceTypeMask).
				Or(ResourceTypeTag)

	AnyTypeTag = newTypeTagFromLowerMask(anyTypeMask).
			Or(AnyStructTypeTag).
			Or(AnyResourceTypeTag)
)

// Methods

func LeastCommonSuperType(types ...Type) Type {
	join := NeverTypeTag

	for _, typ := range types {
		join = join.Or(typ.Tag())
	}

	return findCommonSupperType(join, types...)
}

func findCommonSupperType(joinedTypeTag TypeTag, types ...Type) Type {
	if joinedTypeTag.upperMask != 0 {
		// All existing types can be represented using 64-bits.
		// Hence upperMask is unused for now.
		panic(errors.NewUnreachableError())
	}

	switch joinedTypeTag.lowerMask {

	case uint8TypeMask:
		return UInt8Type
	case uint16TypeMask:
		return UInt16Type
	case uint32TypeMask:
		return UInt32Type
	case uint64TypeMask:
		return UInt64Type
	case uint128TypeMask:
		return UInt128Type
	case uint256TypeMask:
		return UInt256Type

	case int8TypeMask:
		return Int8Type
	case int16TypeMask:
		return Int16Type
	case int32TypeMask:
		return Int32Type
	case int64TypeMask:
		return Int64Type
	case int128TypeMask:
		return Int128Type
	case int256TypeMask:
		return Int256Type

	case word8TypeMask:
		return Word8Type
	case word16TypeMask:
		return Word16Type
	case word32TypeMask:
		return Word32Type
	case word64TypeMask:
		return Word64Type

	case fix64TypeMask:
		return Fix64Type
	case ufix64TypeMask:
		return UFix64Type

	case intTypeMask:
		return IntType
	case uIntTypeMask:
		return UIntType
	case stringTypeMask:
		return StringType
	case nilTypeMask:
		return &OptionalType{
			Type: NeverType,
		}
	case anyStructTypeMask:
		return AnyStructType
	case anyResourceTypeMask:
		return AnyResourceType
	case neverTypeMask:
		return NeverType
	case arrayTypeMask, dictionaryTypeMask, compositeTypeMask:
		// Contains only arrays/dictionaries/composites.
		var prevType Type
		for _, typ := range types {
			if prevType == nil {
				prevType = typ
				continue
			}

			if !typ.Equal(prevType) {
				return commonSuperTypeOfHeterogeneousTypes(types)
			}
		}

		return prevType
	}

	// Optional types.
	if joinedTypeTag.ContainsAny(OptionalTypeTag) {
		// Get the type without the optional flag
		innerTypeTag := joinedTypeTag.And(OptionalTypeTag.Not())
		supperType := findCommonSupperType(innerTypeTag)

		// If the common supertype of the rest of types contain nil,
		// then do not wrap with optional again.
		if supperType.Tag().ContainsAny(NilTypeTag) {
			return supperType
		}

		return &OptionalType{
			Type: supperType,
		}
	}

	// Any heterogeneous int subtypes goes here.
	if joinedTypeTag.BelongsTo(IntegerTypeTag) {
		// Cadence currently doesn't support implicit casting to int supertypes.
		// Therefore any heterogeneous integer types should belong to AnyStruct.
		return AnyStructType
	}

	if joinedTypeTag.ContainsAny(
		ArrayTypeTag,
		DictionaryTypeTag,
		CompositeTypeTag,
	) {
		// At this point, the types contains arrays/dictionaries/composites along with other types.
		// So the common supertype could only be AnyStruct, AnyResource or none (both)
		return commonSuperTypeOfHeterogeneousTypes(types)
	}

	if joinedTypeTag.BelongsTo(AnyStructTypeTag) {
		return AnyStructType
	}

	if joinedTypeTag.BelongsTo(AnyResourceTypeTag) {
		return AnyResourceType
	}

	// If nothing works, then there's no common supertype.
	return NeverType
}

func commonSuperTypeOfHeterogeneousTypes(types []Type) Type {
	var hasStructs, hasResources bool
	for _, typ := range types {
		isResource := typ.IsResourceType()
		hasResources = hasResources || isResource
		hasStructs = hasStructs || !isResource
	}

	if hasResources {
		if hasStructs {
			// If the types has both structs and resources,
			// then there's no common super type.
			return NeverType
		}

		return AnyResourceType
	}

	return AnyStructType
}
