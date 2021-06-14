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

const IntTag = Int256Tag << 1
const StringTag = IntTag << 1
const NilTag = StringTag << 1
const AnyStructTag = NilTag << 1
const AnyResourceTag = AnyStructTag << 1

const ArrayTag = AnyResourceTag << 1
const DictionaryTag = ArrayTag << 1
const CompositeTag = DictionaryTag << 1
const ReferenceTag = CompositeTag << 1
const ResourceTag = ReferenceTag << 1

//const OptionalTag = <actual_type> | NilTag
// Should be dynamically created

// Super types

const SignedIntTag = IntTag | Int8Tag | Int16Tag | Int32Tag | Int64Tag | Int128Tag | Int256Tag

const UnsignedIntTag = UInt8Tag | UInt16Tag | UInt32Tag | UInt64Tag | UInt128Tag | UInt256Tag

const IntSuperTypeTag = SignedIntTag | UnsignedIntTag

const AnyStructSuperTypeTag = AnyStructTag | NeverTypeTag | IntSuperTypeTag | StringTag | ArrayTag |
	DictionaryTag | CompositeTag | ReferenceTag | NilTag

const AnyResourceSuperTypeTag = AnyResourceTag | ResourceTag

// Methods

func CommonSuperType(typeTags ...TypeTag) Type {
	join := NeverTypeTag

	for _, typeTag := range typeTags {
		join |= typeTag
	}

	return getType(join)
}

func getType(tag TypeTag) Type {
	switch tag {
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

	default:

		// Optional types.
		// If the joined type contains nil, and iff that nil didn't
		// come from AnyStruct, then treat it as optional type.
		if isSuperType(tag, NilTag) && !isSuperType(tag, AnyStructSuperTypeTag) {
			// Type without the nil
			innerTypeTag := tag & (^NilTag)
			innerType := getType(innerTypeTag)

			return &OptionalType{
				Type: innerType,
			}
		}

		if isSubType(tag, UnsignedIntTag) {
			return UIntType
		}

		// SignedIntTag also included here
		if isSubType(tag, IntSuperTypeTag) {
			return IntType
		}

		if isSubType(tag, AnyStructSuperTypeTag) {
			return AnyStructType
		}

		if isSubType(tag, AnyResourceSuperTypeTag) {
			return AnyResourceType
		}

		// If nothing works, then there's no common supertype.
		return NeverType
	}
}

func isSubType(typeTag, superTypeTag TypeTag) bool {
	return (typeTag & superTypeTag) == typeTag
}

func isSuperType(typeTag, subTypeTag TypeTag) bool {
	return isSubType(subTypeTag, typeTag)
}
