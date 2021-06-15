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

type TypeTag uint64

const NeverTypeTag TypeTag = 0

const UInt8Tag TypeTag = 1
const UInt16Tag = UInt8Tag << 1
const UInt32Tag = UInt16Tag << 1
const UInt64Tag = UInt32Tag << 1
const UInt128Tag = UInt64Tag << 1
const UInt256Tag = UInt128Tag << 1

const Int8Tag = UInt256Tag << 1
const Int16Tag = Int8Tag << 1
const Int32Tag = Int16Tag << 1
const Int64Tag = Int32Tag << 1
const Int128Tag = Int64Tag << 1
const Int256Tag = Int128Tag << 1

// reserve 10 bits for float and word
const IntTag = Int256Tag << 10
const UIntTag = IntTag << 1
const StringTag = UIntTag << 1
const CharacterTag = StringTag << 1
const BoolTag = CharacterTag << 1
const NilTag = BoolTag << 1
const VoidTag = NilTag << 1
const AddressTag = VoidTag << 1
const MetaTag = AddressTag << 1
const AnyStructTag = MetaTag << 1
const AnyResourceTag = AnyStructTag << 1
const AnyTag = AnyResourceTag << 1

const PathTag = AnyTag << 1
const StoragePathTag = PathTag << 1
const CapabilityPathTag = StoragePathTag << 1
const PublicPathTag = CapabilityPathTag << 1
const PrivatePathTag = PublicPathTag << 1

const ArrayTag = PrivatePathTag << 1
const DictionaryTag = ArrayTag << 1
const CompositeTag = DictionaryTag << 1
const ReferenceTag = CompositeTag << 1
const ResourceTag = ReferenceTag << 1

const OptionalTag = ResourceTag << 1
const GenericTag = OptionalTag << 1
const FunctionTag = GenericTag << 1
const InterfaceTag = FunctionTag << 1
const TransactionTag = InterfaceTag << 1
const RestrictedTag = TransactionTag << 1
const CapabilityTag = RestrictedTag << 1

const InvalidTag TypeTag = 1 << 62

// Super types

const SignedIntTag = IntTag | Int8Tag | Int16Tag | Int32Tag | Int64Tag | Int128Tag | Int256Tag

const UnsignedIntTag = UIntTag | UInt8Tag | UInt16Tag | UInt32Tag | UInt64Tag | UInt128Tag | UInt256Tag

const IntSuperTypeTag = SignedIntTag | UnsignedIntTag

const AnyStructSuperTypeTag = AnyStructTag | NeverTypeTag | IntSuperTypeTag | StringTag | ArrayTag |
	DictionaryTag | CompositeTag | ReferenceTag | NilTag

const AnyResourceSuperTypeTag = AnyResourceTag | ResourceTag

const AnySuperTypeTag = AnyResourceSuperTypeTag | AnyStructSuperTypeTag

// Methods

func CommonSuperType(types ...Type) Type {
	join := NeverTypeTag

	for _, typ := range types {
		join |= typ.Tag()
	}

	return getType(join, types...)
}

func getType(joinedTypeTag TypeTag, types ...Type) Type {
	switch joinedTypeTag {
	case Int8Tag:
		return Int8Type
	case Int16Tag:
		return Int16Type
	case Int32Tag:
		return Int32Type
	case Int64Tag:
		return Int64Type
	case Int128Tag:
		return Int128Type
	case Int256Tag:
		return Int256Type

	case UInt8Tag:
		return UInt8Type
	case UInt16Tag:
		return UInt16Type
	case UInt32Tag:
		return UInt32Type
	case UInt64Tag:
		return UInt64Type
	case UInt128Tag:
		return UInt128Type
	case UInt256Tag:
		return UInt256Type

	case IntTag:
		return IntType
	case UIntTag:
		return UIntType
	case StringTag:
		return StringType
	case NilTag:
		return &OptionalType{
			Type: NeverType,
		}
	case AnyStructTag:
		return AnyStructType
	case AnyResourceTag:
		return AnyResourceType
	case NeverTypeTag:
		return NeverType
	case ArrayTag:
		// Contains only arrays.
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
		if joinedTypeTag.contains(OptionalTag) {
			// Get the type without the optional flag
			innerTypeTag := joinedTypeTag & (^OptionalTag)
			innerType := getType(innerTypeTag)

			return &OptionalType{
				Type: innerType,
			}
		}

		// Any heterogeneous int subtypes goes here.
		if joinedTypeTag.belongsTo(IntSuperTypeTag) {
			return IntType
		}

		if joinedTypeTag.contains(ArrayTag) {
			// At this point, the types contains arrays and other types.
			// So the common supertype could only be AnyStruct, AnyResource or none (both)
			return commonSupertypeOfHeterogeneousTypes(types)
		}

		if joinedTypeTag.belongsTo(AnyStructSuperTypeTag) {
			return AnyStructType
		}

		if joinedTypeTag.belongsTo(AnyResourceSuperTypeTag) {
			return AnyResourceType
		}

		// If nothing works, then there's no common supertype.
		return NeverType
	}
}

func (t TypeTag) contains(typeTag TypeTag) bool {
	return (t & typeTag) == typeTag
}

func (t TypeTag) belongsTo(typeTag TypeTag) bool {
	return typeTag.contains(t)
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
