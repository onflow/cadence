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

// TypeTag is a bitmask representation for types.
// Each type has a unique dedicated bit in the bitMaskBit.
//
type TypeTag struct {
	block1 uint64
	block2 uint64
}

func NewTypeTag(flag uint64) TypeTag {
	if flag > 127 {
		panic("flag out of range")
	}

	if flag < 64 {
		return TypeTag{
			block1: 1 << flag,
			block2: 0,
		}
	}

	return TypeTag{
		block1: 0,
		block2: 1 << (flag - 64),
	}
}

func (t TypeTag) Equals(tag TypeTag) bool {
	return t.block1 == tag.block1 && t.block2 == tag.block2
}

func (t TypeTag) And(tag TypeTag) TypeTag {
	return TypeTag{
		block1: t.block1 & tag.block1,
		block2: t.block2 & tag.block2,
	}
}

func (t TypeTag) Or(tag TypeTag) TypeTag {
	return TypeTag{
		block1: t.block1 | tag.block1,
		block2: t.block2 | tag.block2,
	}
}

func (t TypeTag) Not() TypeTag {
	return TypeTag{
		block1: ^t.block1,
		block2: ^t.block2,
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

const (
	uint8TypeMaskBit uint64 = iota
	uint16TypeMaskBit
	uint32TypeMaskBit
	uint64TypeMaskBit
	uint128TypeMaskBit
	uint256TypeMaskBit

	int8TypeMaskBit
	int16TypeMaskBit
	int32TypeMaskBit
	int64TypeMaskBit
	int128TypeMaskBit
	int256TypeMaskBit

	word8TypeMaskBit
	word16TypeMaskBit
	word32TypeMaskBit
	word64TypeMaskBit

	fix64TypeMaskBit
	ufix64TypeMaskBit

	intTypeMaskBit
	uIntTypeMaskBit
	stringTypeMaskBit
	characterTypeMaskBit
	boolTypeMaskBit
	nilTypeMaskBit
	voidTypeMaskBit
	addressTypeMaskBit
	metaTypeMaskBit
	anyStructTypeMaskBit
	anyResourceTypeMaskBit
	anyTypeMaskBit

	pathTypeMaskBit
	storagePathTypeMaskBit
	capabilityPathTypeMaskBit
	publicPathTypeMaskBit
	privatePathTypeMaskBit

	arrayTypeMaskBit
	dictionaryTypeMaskBit
	compositeTypeMaskBit
	referenceTypeMaskBit
	resourceTypeMaskBit

	optionalTypeMaskBit
	genericTypeMaskBit
	functionTypeMaskBit
	interfaceTypeMaskBit
	transactionTypeMaskBit
	restrictedTypeMaskBit
	capabilityTypeMaskBit

	invalidTypeMaskBit
)

var (
	NeverTypeTag = TypeTag{0, 0}

	UInt8TypeTag   = NewTypeTag(uint8TypeMaskBit)
	UInt16TypeTag  = NewTypeTag(uint16TypeMaskBit)
	UInt32TypeTag  = NewTypeTag(uint32TypeMaskBit)
	UInt64TypeTag  = NewTypeTag(uint64TypeMaskBit)
	UInt128TypeTag = NewTypeTag(uint128TypeMaskBit)
	UInt256TypeTag = NewTypeTag(uint256TypeMaskBit)

	Int8TypeTag   = NewTypeTag(int8TypeMaskBit)
	Int16TypeTag  = NewTypeTag(int16TypeMaskBit)
	Int32TypeTag  = NewTypeTag(int32TypeMaskBit)
	Int64TypeTag  = NewTypeTag(int64TypeMaskBit)
	Int128TypeTag = NewTypeTag(int128TypeMaskBit)
	Int256TypeTag = NewTypeTag(int256TypeMaskBit)

	Word8TypeTag  = NewTypeTag(word8TypeMaskBit)
	Word16TypeTag = NewTypeTag(word16TypeMaskBit)
	Word32TypeTag = NewTypeTag(word32TypeMaskBit)
	Word64TypeTag = NewTypeTag(word64TypeMaskBit)

	Fix64TypeTag  = NewTypeTag(fix64TypeMaskBit)
	UFix64TypeTag = NewTypeTag(ufix64TypeMaskBit)

	IntTypeTag         = NewTypeTag(intTypeMaskBit)
	UIntTypeTag        = NewTypeTag(uIntTypeMaskBit)
	StringTypeTag      = NewTypeTag(stringTypeMaskBit)
	CharacterTypeTag   = NewTypeTag(characterTypeMaskBit)
	BoolTypeTag        = NewTypeTag(boolTypeMaskBit)
	NilTypeTag         = NewTypeTag(nilTypeMaskBit)
	VoidTypeTag        = NewTypeTag(voidTypeMaskBit)
	AddressTypeTag     = NewTypeTag(addressTypeMaskBit)
	MetaTypeTag        = NewTypeTag(metaTypeMaskBit)
	AnyStructTypeTag   = NewTypeTag(anyStructTypeMaskBit)
	AnyResourceTypeTag = NewTypeTag(anyResourceTypeMaskBit)
	AnyTypeTag         = NewTypeTag(anyTypeMaskBit)

	PathTypeTag           = NewTypeTag(pathTypeMaskBit)
	StoragePathTypeTag    = NewTypeTag(storagePathTypeMaskBit)
	CapabilityPathTypeTag = NewTypeTag(capabilityPathTypeMaskBit)
	PublicPathTypeTag     = NewTypeTag(publicPathTypeMaskBit)
	PrivatePathTypeTag    = NewTypeTag(privatePathTypeMaskBit)

	ArrayTypeTag      = NewTypeTag(arrayTypeMaskBit)
	DictionaryTypeTag = NewTypeTag(dictionaryTypeMaskBit)
	CompositeTypeTag  = NewTypeTag(compositeTypeMaskBit)
	ReferenceTypeTag  = NewTypeTag(referenceTypeMaskBit)
	ResourceTypeTag   = NewTypeTag(resourceTypeMaskBit)

	OptionalTypeTag    = NewTypeTag(optionalTypeMaskBit)
	GenericTypeTag     = NewTypeTag(genericTypeMaskBit)
	FunctionTypeTag    = NewTypeTag(functionTypeMaskBit)
	InterfaceTypeTag   = NewTypeTag(interfaceTypeMaskBit)
	TransactionTypeTag = NewTypeTag(transactionTypeMaskBit)
	RestrictedTypeTag  = NewTypeTag(restrictedTypeMaskBit)
	CapabilityTypeTag  = NewTypeTag(capabilityTypeMaskBit)

	InvalidTypeTag = NewTypeTag(invalidTypeMaskBit)

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

	IntSuperTypeTag = SignedIntTypeTag.Or(UnsignedIntTypeTag)

	AnyStructSuperTypeTag = AnyStructTypeTag.
				Or(NeverTypeTag).
				Or(IntSuperTypeTag).
				Or(StringTypeTag).
				Or(ArrayTypeTag).
				Or(DictionaryTypeTag).
				Or(CompositeTypeTag).
				Or(ReferenceTypeTag).
				Or(NilTypeTag)

	AnyResourceSuperTypeTag = AnyResourceTypeTag.Or(ResourceTypeTag)

	AnySuperTypeTag = AnyResourceSuperTypeTag.Or(AnyStructSuperTypeTag)
)

// Methods

func CommonSuperType(types ...Type) Type {
	join := NeverTypeTag

	for _, typ := range types {
		join = join.Or(typ.Tag())
	}

	return getType(join, types...)
}

func getType(joinedTypeTag TypeTag, types ...Type) Type {
	if joinedTypeTag.block2 > 0 {
		// All existing types can be represented using 64-bits.
		// Hence block2 is unused for now.
		panic("unsupported")
	}

	switch joinedTypeTag.block1 {

	case UInt8TypeTag.block1:
		return UInt8Type
	case UInt16TypeTag.block1:
		return UInt16Type
	case UInt32TypeTag.block1:
		return UInt32Type
	case UInt64TypeTag.block1:
		return UInt64Type
	case UInt128TypeTag.block1:
		return UInt128Type
	case UInt256TypeTag.block1:
		return UInt256Type

	case Int8TypeTag.block1:
		return Int8Type
	case Int16TypeTag.block1:
		return Int16Type
	case Int32TypeTag.block1:
		return Int32Type
	case Int64TypeTag.block1:
		return Int64Type
	case Int128TypeTag.block1:
		return Int128Type
	case Int256TypeTag.block1:
		return Int256Type

	case Word8TypeTag.block1:
		return Word8Type
	case Word16TypeTag.block1:
		return Word16Type
	case Word32TypeTag.block1:
		return Word32Type
	case Word64TypeTag.block1:
		return Word64Type

	case Fix64TypeTag.block1:
		return Fix64Type
	case UFix64TypeTag.block1:
		return UFix64Type

	case IntTypeTag.block1:
		return IntType
	case UIntTypeTag.block1:
		return UIntType
	case StringTypeTag.block1:
		return StringType
	case NilTypeTag.block1:
		return &OptionalType{
			Type: NeverType,
		}
	case AnyStructTypeTag.block1:
		return AnyStructType
	case AnyResourceTypeTag.block1:
		return AnyResourceType
	case NeverTypeTag.block1:
		return NeverType
	case ArrayTypeTag.block1, DictionaryTypeTag.block1:
		// Contains only arrays or only dictionaries.
		var prevType Type
		for _, typ := range types {
			if prevType == nil {
				prevType = typ
				continue
			}

			if !typ.Equal(prevType) {
				return commonSupertypeOfHeterogeneousTypes(types)
			}
		}

		return prevType
	}

	// Optional types.
	if joinedTypeTag.ContainsAny(OptionalTypeTag) {
		// Get the type without the optional flag
		innerTypeTag := joinedTypeTag.And(OptionalTypeTag.Not())
		innerType := getType(innerTypeTag)
		return &OptionalType{
			Type: innerType,
		}
	}

	// Any heterogeneous int subtypes goes here.
	if joinedTypeTag.BelongsTo(IntSuperTypeTag) {
		return IntType
	}

	if joinedTypeTag.ContainsAny(ArrayTypeTag, DictionaryTypeTag) {
		// At this point, the types contains arrays/dictionaries along with other types.
		// So the common supertype could only be AnyStruct, AnyResource or none (both)
		return commonSupertypeOfHeterogeneousTypes(types)
	}

	if joinedTypeTag.BelongsTo(AnyStructSuperTypeTag) {
		return AnyStructType
	}

	if joinedTypeTag.BelongsTo(AnyResourceSuperTypeTag) {
		return AnyResourceType
	}

	// If nothing works, then there's no common supertype.
	return NeverType
}

func commonSupertypeOfHeterogeneousTypes(types []Type) Type {
	var hasStructs, hasResources bool
	for _, typ := range types {
		isResource := typ.IsResourceType()
		hasResources = hasResources || isResource
		hasStructs = hasStructs || !isResource
	}

	if hasResources {
		if hasStructs {
			// If the types has both structs and resources,
			// then there no common super type.
			return NeverType
		}

		return AnyResourceType
	}

	return AnyStructType
}
