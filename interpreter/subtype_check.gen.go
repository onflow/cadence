// Code generated from rules.yaml. DO NOT EDIT.
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

package interpreter

func checkSubTypeWithoutEquality_gen(typeConverter TypeConverter, subType StaticType, superType StaticType) bool {
	if subType == PrimitiveStaticTypeNever {
		return true
	}

	switch superType {
	case PrimitiveStaticTypeAny:
		return true

	case PrimitiveStaticTypeAnyStruct:
		return !(IsResourceType(subType)) &&
			subType != PrimitiveStaticTypeAny

	case PrimitiveStaticTypeAnyResource:
		return IsResourceType(subType)

	case PrimitiveStaticTypeAnyResourceAttachment:
		return isAttachmentType(subType) &&
			IsResourceType(subType)

	case PrimitiveStaticTypeAnyStructAttachment:
		return isAttachmentType(subType) &&
			!(IsResourceType(subType))

	case PrimitiveStaticTypeHashableStruct:
		return IsHashableStructType(typeConverter, subType)

	case PrimitiveStaticTypePath:
		return IsSubType(typeConverter, subType, PrimitiveStaticTypeStoragePath) ||
			IsSubType(typeConverter, subType, PrimitiveStaticTypeCapabilityPath)

	case PrimitiveStaticTypeCapabilityPath:
		switch subType {
		case PrimitiveStaticTypePrivatePath,
			PrimitiveStaticTypePublicPath:
			return true
		}

		return false

	case PrimitiveStaticTypeNumber:
		switch subType {
		case PrimitiveStaticTypeNumber,
			PrimitiveStaticTypeSignedNumber:
			return true
		}

		return IsSubType(typeConverter, subType, PrimitiveStaticTypeInteger) ||
			IsSubType(typeConverter, subType, PrimitiveStaticTypeFixedPoint)

	case PrimitiveStaticTypeSignedNumber:
		return subType == PrimitiveStaticTypeSignedNumber ||
			(IsSubType(typeConverter, subType, PrimitiveStaticTypeSignedInteger) ||
				IsSubType(typeConverter, subType, PrimitiveStaticTypeSignedFixedPoint))

	case PrimitiveStaticTypeInteger:
		switch subType {
		case PrimitiveStaticTypeInteger,
			PrimitiveStaticTypeSignedInteger,
			PrimitiveStaticTypeFixedSizeUnsignedInteger,
			PrimitiveStaticTypeUInt:
			return true
		}

		return IsSubType(typeConverter, subType, PrimitiveStaticTypeSignedInteger) ||
			IsSubType(typeConverter, subType, PrimitiveStaticTypeFixedSizeUnsignedInteger)

	case PrimitiveStaticTypeSignedInteger:
		switch subType {
		case PrimitiveStaticTypeSignedInteger,
			PrimitiveStaticTypeInt,
			PrimitiveStaticTypeInt8,
			PrimitiveStaticTypeInt16,
			PrimitiveStaticTypeInt32,
			PrimitiveStaticTypeInt64,
			PrimitiveStaticTypeInt128,
			PrimitiveStaticTypeInt256:
			return true
		}

		return false

	case PrimitiveStaticTypeFixedSizeUnsignedInteger:
		switch subType {
		case PrimitiveStaticTypeUInt8,
			PrimitiveStaticTypeUInt16,
			PrimitiveStaticTypeUInt32,
			PrimitiveStaticTypeUInt64,
			PrimitiveStaticTypeUInt128,
			PrimitiveStaticTypeUInt256,
			PrimitiveStaticTypeWord8,
			PrimitiveStaticTypeWord16,
			PrimitiveStaticTypeWord32,
			PrimitiveStaticTypeWord64,
			PrimitiveStaticTypeWord128,
			PrimitiveStaticTypeWord256:
			return true
		}

		return false

	case PrimitiveStaticTypeFixedPoint:
		switch subType {
		case PrimitiveStaticTypeFixedPoint,
			PrimitiveStaticTypeSignedFixedPoint,
			PrimitiveStaticTypeUFix64,
			PrimitiveStaticTypeUFix128:
			return true
		}

		return IsSubType(typeConverter, subType, PrimitiveStaticTypeSignedFixedPoint)

	case PrimitiveStaticTypeSignedFixedPoint:
		switch subType {
		case PrimitiveStaticTypeSignedFixedPoint,
			PrimitiveStaticTypeFix64,
			PrimitiveStaticTypeFix128:
			return true
		}

		return false

	}

	switch typedSuperType := superType.(type) {
	case *OptionalStaticType:
		switch typedSubType := subType.(type) {
		case *OptionalStaticType:
			return IsSubType(typeConverter, typedSubType.Type, typedSuperType.Type)
		}

		return IsSubType(typeConverter, subType, typedSuperType.Type)

	case *DictionaryStaticType:
		switch typedSubType := subType.(type) {
		case *DictionaryStaticType:
			return IsSubType(typeConverter, typedSubType.ValueType, typedSuperType.ValueType) &&
				IsSubType(typeConverter, typedSubType.KeyType, typedSuperType.KeyType)
		}

		return false

	case *VariableSizedStaticType:
		switch typedSubType := subType.(type) {
		case *VariableSizedStaticType:
			return IsSubType(typeConverter, typedSubType.ElementType(), typedSuperType.ElementType())
		}

		return false

	case *ConstantSizedStaticType:
		switch typedSubType := subType.(type) {
		case *ConstantSizedStaticType:
			return typedSuperType.Size == typedSubType.Size &&
				IsSubType(typeConverter, typedSubType.ElementType(), typedSuperType.ElementType())
		}

		return false

	case *ReferenceStaticType:
		switch typedSubType := subType.(type) {
		case *ReferenceStaticType:
			return PermitsAccess(typedSuperType.Authorization, typedSubType.Authorization) &&
				IsSubType(typeConverter, typedSubType.Type, typedSuperType.Type)
		}

		return false

	case *CompositeStaticType:
		switch typedSubType := subType.(type) {
		case *IntersectionStaticType:
			switch typedSubType.LegacyType {
			case nil,
				PrimitiveStaticTypeAnyResource,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAny:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeStaticType:
				return typedSubTypeLegacyType == typedSuperType
			}

		case *CompositeStaticType:
			return false
		}

		return false

	case *InterfaceStaticType:
		switch typedSubType := subType.(type) {
		case *CompositeStaticType:
			return typedSubType.Kind == typedSuperType.CompositeKind &&
				typedSubType.EffectiveInterfaceConformanceSet().Contains(typedSuperType)
		case *IntersectionStaticType:
			return typedSubType.EffectiveIntersectionSet().Contains(typedSuperType)
		case *InterfaceStaticType:
			return typedSubType.EffectiveInterfaceConformanceSet().Contains(typedSuperType)
		}

		return false

	case *IntersectionStaticType:
		switch typedSuperType.LegacyType {
		case nil,
			PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:
			switch subType {
			case PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			switch typedSubType := subType.(type) {
			case *IntersectionStaticType:
				if typedSubType.LegacyType == nil &&
					IsIntersectionSubset(typedSuperType, typedSubType) {
					return true
				}

				switch typedSubType.LegacyType {
				case PrimitiveStaticTypeAny,
					PrimitiveStaticTypeAnyStruct,
					PrimitiveStaticTypeAnyResource:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typeConverter, typedSubType.LegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubType)
				}

				switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
				case *CompositeStaticType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typeConverter, typedSubTypeLegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubTypeLegacyType)
				}

			case PrimitiveStaticTypeConforming:
				return (typedSuperType.LegacyType == nil ||
					IsSubType(typeConverter, typedSubType, typedSuperType.LegacyType)) &&
					IsIntersectionSubset(typedSuperType, typedSubType)
			}

		}

		switch typedSubType := subType.(type) {
		case *IntersectionStaticType:
			switch typedSubType.LegacyType {
			case nil,
				PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeStaticType:
				return typedSubTypeLegacyType == typedSuperType.LegacyType
			}

		case *CompositeStaticType:
			return IsSubType(typeConverter, typedSubType, typedSuperType.LegacyType)
		}

		switch subType {
		case PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:
			return false
		}

		return false

	}

	return false
}
