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

type TypeTag struct {
	block1 uint64
	block2 uint64
}

func NewTypeTag(flag uint64) TypeTag {
	if flag >= 127 {
		panic("flag too large")
	}

	if flag < 63 {
		return TypeTag{1 << flag, 0}
	}

	return TypeTag{0, 1 << (flag - 63)}
}

func (t TypeTag) Equals(tag TypeTag) bool {
	return t.block1 == tag.block1 && t.block2 == tag.block2
}

func (t TypeTag) And(tag TypeTag) TypeTag {
	return TypeTag{t.block1 & tag.block1, t.block2 & tag.block2}
}

func (t TypeTag) Or(tag TypeTag) TypeTag {
	return TypeTag{t.block1 | tag.block1, t.block2 | tag.block2}
}

func (t TypeTag) Not() TypeTag {
	return TypeTag{^t.block1, ^t.block2}
}

var (
	NeverTypeTag = TypeTag{0, 0}

	UInt8Tag   = NewTypeTag(0)
	UInt16Tag  = NewTypeTag(1)
	UInt32Tag  = NewTypeTag(2)
	UInt64Tag  = NewTypeTag(3)
	UInt128Tag = NewTypeTag(4)
	UInt256Tag = NewTypeTag(5)

	Int8Tag   = NewTypeTag(6)
	Int16Tag  = NewTypeTag(7)
	Int32Tag  = NewTypeTag(8)
	Int64Tag  = NewTypeTag(9)
	Int128Tag = NewTypeTag(10)
	Int256Tag = NewTypeTag(11)

	// reserve 10 bits for float and word
	IntTag         = NewTypeTag(21)
	UIntTag        = NewTypeTag(22)
	StringTag      = NewTypeTag(23)
	CharacterTag   = NewTypeTag(24)
	BoolTag        = NewTypeTag(25)
	NilTag         = NewTypeTag(26)
	VoidTag        = NewTypeTag(27)
	AddressTag     = NewTypeTag(28)
	MetaTag        = NewTypeTag(29)
	AnyStructTag   = NewTypeTag(30)
	AnyResourceTag = NewTypeTag(31)
	AnyTag         = NewTypeTag(32)

	PathTag           = NewTypeTag(33)
	StoragePathTag    = NewTypeTag(34)
	CapabilityPathTag = NewTypeTag(35)
	PublicPathTag     = NewTypeTag(36)
	PrivatePathTag    = NewTypeTag(37)

	ArrayTag      = NewTypeTag(38)
	DictionaryTag = NewTypeTag(39)
	CompositeTag  = NewTypeTag(40)
	ReferenceTag  = NewTypeTag(41)
	ResourceTag   = NewTypeTag(42)

	OptionalTag    = NewTypeTag(43)
	GenericTag     = NewTypeTag(44)
	FunctionTag    = NewTypeTag(45)
	InterfaceTag   = NewTypeTag(46)
	TransactionTag = NewTypeTag(47)
	RestrictedTag  = NewTypeTag(48)
	CapabilityTag  = NewTypeTag(49)

	InvalidTag = NewTypeTag(50)

	// Super types

	SignedIntTag = IntTag.Or(Int8Tag).Or(Int16Tag).Or(Int32Tag).Or(Int64Tag).Or(Int128Tag).Or(Int256Tag)

	UnsignedIntTag = UIntTag.Or(UInt8Tag).Or(UInt16Tag).Or(UInt32Tag).Or(UInt64Tag).Or(UInt128Tag).Or(UInt256Tag)

	IntSuperTypeTag = SignedIntTag.Or(UnsignedIntTag)

	AnyStructSuperTypeTag = AnyStructTag.Or(NeverTypeTag).Or(IntSuperTypeTag).Or(StringTag).Or(ArrayTag).
				Or(DictionaryTag).Or(CompositeTag).Or(ReferenceTag).Or(NilTag)

	AnyResourceSuperTypeTag = AnyResourceTag.Or(ResourceTag)

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
		panic("unsupported")
	}

	switch joinedTypeTag.block1 {
	case Int8Tag.block1:
		return Int8Type
	case Int16Tag.block1:
		return Int16Type
	case Int32Tag.block1:
		return Int32Type
	case Int64Tag.block1:
		return Int64Type
	case Int128Tag.block1:
		return Int128Type
	case Int256Tag.block1:
		return Int256Type

	case UInt8Tag.block1:
		return UInt8Type
	case UInt16Tag.block1:
		return UInt16Type
	case UInt32Tag.block1:
		return UInt32Type
	case UInt64Tag.block1:
		return UInt64Type
	case UInt128Tag.block1:
		return UInt128Type
	case UInt256Tag.block1:
		return UInt256Type

	case IntTag.block1:
		return IntType
	case UIntTag.block1:
		return UIntType
	case StringTag.block1:
		return StringType
	case NilTag.block1:
		return &OptionalType{
			Type: NeverType,
		}
	case AnyStructTag.block1:
		return AnyStructType
	case AnyResourceTag.block1:
		return AnyResourceType
	case NeverTypeTag.block1:
		return NeverType
	case ArrayTag.block1, DictionaryTag.block1:
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

	default:

		// Optional types.
		if joinedTypeTag.ContainsAny(OptionalTag) {
			// Get the type without the optional flag
			innerTypeTag := joinedTypeTag.And(OptionalTag.Not())
			innerType := getType(innerTypeTag)
			return &OptionalType{
				Type: innerType,
			}
		}

		// Any heterogeneous int subtypes goes here.
		if joinedTypeTag.BelongsTo(IntSuperTypeTag) {
			return IntType
		}

		if joinedTypeTag.ContainsAny(ArrayTag, DictionaryTag) {
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
}

func (t TypeTag) ContainsAny(typeTags ...TypeTag) bool {
	for _, tag := range typeTags {
		if t.And(tag).Equals(tag) {
			return true
		}
	}

	return false
}

func (t TypeTag) BelongsTo(targetTypeTag TypeTag) bool {
	return targetTypeTag.ContainsAny(t)
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
