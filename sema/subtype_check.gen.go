// Code generated from <no value>. DO NOT EDIT.
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

package sema

func checkSubTypeWithoutEquality_gen(subType Type, superType Type) bool {
	if subType == NeverType {
		return true
	}
	switch superType {
	case AnyType:
		return true
	case AnyStructType:
		return !(subType.IsResourceType()) && !(subType == AnyType)
	case AnyResourceType:
		return subType.IsResourceType()
	case AnyResourceAttachmentType:
		return isAttachmentType(subType) && subType.IsResourceType()
	case AnyStructAttachmentType:
		return isAttachmentType(subType) && !(subType.IsResourceType())
	case HashableStructType:
		return IsHashableStructType(subType)
	case PathType:
		return IsSubType(subType, StoragePathType) || IsSubType(subType, CapabilityPathType)
	case StorableType:
		return subType.IsStorable(map[*Member]bool{})
	case CapabilityPathType:
		switch subType {
		case PrivatePathType, PublicPathType:
			return true
		}
		return false
	case NumberType:
		switch subType {
		case NumberType, SignedNumberType:
			return true
		}
		return IsSubType(subType, IntegerType) || IsSubType(subType, FixedPointType)
	case SignedNumberType:
		return subType == SignedNumberType || (IsSubType(subType, SignedIntegerType) || IsSubType(subType, SignedFixedPointType))
	case IntegerType:
		switch subType {
		case IntegerType, SignedIntegerType, FixedSizeUnsignedIntegerType, UIntType:
			return true
		}
		return IsSubType(subType, SignedIntegerType) || IsSubType(subType, FixedSizeUnsignedIntegerType)
	case SignedIntegerType:
		switch subType {
		case SignedIntegerType, IntType, Int8Type, Int16Type, Int32Type, Int64Type, Int128Type, Int256Type:
			return true
		}
		return false
	case FixedSizeUnsignedIntegerType:
		switch subType {
		case UInt8Type, UInt16Type, UInt32Type, UInt64Type, UInt128Type, UInt256Type, Word8Type, Word16Type, Word32Type, Word64Type, Word128Type, Word256Type:
			return true
		}
		return false
	case FixedPointType:
		switch subType {
		case FixedPointType, SignedFixedPointType, UFix64Type, UFix128Type:
			return true
		}
		return IsSubType(subType, SignedFixedPointType)
	case SignedFixedPointType:
		switch subType {
		case SignedFixedPointType, Fix64Type, Fix128Type:
			return true
		}
		return false
	}
	return false
}
