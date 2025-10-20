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

package sema

func checkSubTypeWithoutEquality_gen(subType Type, superType Type) bool {
	if subType == NeverType {
		return true
	}

	switch superType {
	case AnyType:
		return true

	case AnyStructType:
		return !(IsResourceType(subType)) &&
			subType != AnyType

	case AnyResourceType:
		return IsResourceType(subType)

	case AnyResourceAttachmentType:
		return isAttachmentType(subType) &&
			IsResourceType(subType)

	case AnyStructAttachmentType:
		return isAttachmentType(subType) &&
			!(IsResourceType(subType))

	case HashableStructType:
		return IsHashableStructType(subType)

	case PathType:
		return IsSubType(subType, StoragePathType) ||
			IsSubType(subType, CapabilityPathType)

	case StorableType:
		return subType.IsStorable(map[*Member]bool{})

	case CapabilityPathType:
		switch subType {
		case PrivatePathType,
			PublicPathType:
			return true
		}

		return false

	case NumberType:
		switch subType {
		case NumberType,
			SignedNumberType:
			return true
		}

		return IsSubType(subType, IntegerType) ||
			IsSubType(subType, FixedPointType)

	case SignedNumberType:
		return subType == SignedNumberType ||
			(IsSubType(subType, SignedIntegerType) ||
				IsSubType(subType, SignedFixedPointType))

	case IntegerType:
		switch subType {
		case IntegerType,
			SignedIntegerType,
			FixedSizeUnsignedIntegerType,
			UIntType:
			return true
		}

		return IsSubType(subType, SignedIntegerType) ||
			IsSubType(subType, FixedSizeUnsignedIntegerType)

	case SignedIntegerType:
		switch subType {
		case SignedIntegerType,
			IntType,
			Int8Type,
			Int16Type,
			Int32Type,
			Int64Type,
			Int128Type,
			Int256Type:
			return true
		}

		return false

	case FixedSizeUnsignedIntegerType:
		switch subType {
		case UInt8Type,
			UInt16Type,
			UInt32Type,
			UInt64Type,
			UInt128Type,
			UInt256Type,
			Word8Type,
			Word16Type,
			Word32Type,
			Word64Type,
			Word128Type,
			Word256Type:
			return true
		}

		return false

	case FixedPointType:
		switch subType {
		case FixedPointType,
			SignedFixedPointType,
			UFix64Type,
			UFix128Type:
			return true
		}

		return IsSubType(subType, SignedFixedPointType)

	case SignedFixedPointType:
		switch subType {
		case SignedFixedPointType,
			Fix64Type,
			Fix128Type:
			return true
		}

		return false

	}

	switch typedSuperType := superType.(type) {
	case *OptionalType:
		switch typedSubType := subType.(type) {
		case *OptionalType:
			return IsSubType(typedSubType.Type, typedSuperType.Type)
		}

		return IsSubType(subType, typedSuperType.Type)

	case *DictionaryType:
		switch typedSubType := subType.(type) {
		case *DictionaryType:
			return IsSubType(typedSubType.ValueType, typedSuperType.ValueType) &&
				IsSubType(typedSubType.KeyType, typedSuperType.KeyType)
		}

		return false

	case *VariableSizedType:
		switch typedSubType := subType.(type) {
		case *VariableSizedType:
			return IsSubType(typedSubType.ElementType(false), typedSuperType.ElementType(false))
		}

		return false

	case *ConstantSizedType:
		switch typedSubType := subType.(type) {
		case *ConstantSizedType:
			return typedSuperType.Size == typedSubType.Size &&
				IsSubType(typedSubType.ElementType(false), typedSuperType.ElementType(false))
		}

		return false

	case *ReferenceType:
		switch typedSubType := subType.(type) {
		case *ReferenceType:
			return PermitsAccess(typedSuperType.Authorization, typedSubType.Authorization) &&
				IsSubType(typedSubType.Type, typedSuperType.Type)
		}

		return false

	case *CompositeType:
		switch typedSubType := subType.(type) {
		case *IntersectionType:
			switch typedSubType.LegacyType {
			case nil,
				AnyResourceType,
				AnyStructType,
				AnyType:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeType:
				return typedSubTypeLegacyType == typedSuperType
			}

		case *CompositeType:
			return false
		}

		return IsParameterizedSubType(subType, typedSuperType)

	case *InterfaceType:
		switch typedSubType := subType.(type) {
		case *CompositeType:
			return typedSubType.Kind == typedSuperType.CompositeKind &&
				typedSubType.EffectiveInterfaceConformanceSet().Contains(typedSuperType)
		case *IntersectionType:
			return typedSubType.EffectiveIntersectionSet().Contains(typedSuperType)
		case *InterfaceType:
			return typedSubType.EffectiveInterfaceConformanceSet().Contains(typedSuperType)
		}

		return IsParameterizedSubType(subType, typedSuperType)

	case *IntersectionType:
		switch typedSuperType.LegacyType {
		case nil,
			AnyType,
			AnyStructType,
			AnyResourceType:
			switch subType {
			case AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			switch typedSubType := subType.(type) {
			case *IntersectionType:
				if typedSubType.LegacyType == nil &&
					IsIntersectionSubset(typedSuperType, typedSubType) {
					return true
				}

				switch typedSubType.LegacyType {
				case AnyType,
					AnyStructType,
					AnyResourceType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typedSubType.LegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubType)
				}

				switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
				case *CompositeType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typedSubTypeLegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubTypeLegacyType)
				}

			case ConformingType:
				return (typedSuperType.LegacyType == nil ||
					IsSubType(typedSubType, typedSuperType.LegacyType)) &&
					IsIntersectionSubset(typedSuperType, typedSubType)
			}

		}

		switch typedSubType := subType.(type) {
		case *IntersectionType:
			switch typedSubType.LegacyType {
			case nil,
				AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeType:
				return typedSubTypeLegacyType == typedSuperType.LegacyType
			}

		case *CompositeType:
			return IsSubType(typedSubType, typedSuperType.LegacyType)
		}

		switch subType {
		case AnyType,
			AnyStructType,
			AnyResourceType:
			return false
		}

		return IsParameterizedSubType(subType, typedSuperType)

	case *FunctionType:
		switch typedSubType := subType.(type) {
		case *FunctionType:
			switch typedSubType.Purity {
			case typedSuperType.Purity,
				FunctionPurityView:
				typedSubTypeTypeParameters := typedSubType.TypeParameters
				typedSuperTypeTypeParameters := typedSuperType.TypeParameters
				if len(typedSubTypeTypeParameters) != len(typedSuperTypeTypeParameters) {
					return false
				}

				for i, source := range typedSubTypeTypeParameters {
					target := typedSuperTypeTypeParameters[i]
					if source != target {
						return false
					}
				}

				typedSubTypeParameters := typedSubType.Parameters
				typedSuperTypeParameters := typedSuperType.Parameters
				if len(typedSubTypeParameters) != len(typedSuperTypeParameters) {
					return false
				}

				for i, source := range typedSubTypeParameters {
					target := typedSuperTypeParameters[i]
					if !(IsSubType(target.TypeAnnotation.Type, source.TypeAnnotation.Type)) {
						return false
					}
				}

				return typedSubType.Arity == typedSuperType.Arity &&
					(AreReturnsCovariant(typedSubType, typedSuperType) &&
						typedSubType.IsConstructor == typedSuperType.IsConstructor)
			}

		}

		return false

	case ParameterizedType:
		if typedSuperType.BaseType() != nil {
			switch typedSubType := subType.(type) {
			case ParameterizedType:
				if typedSubType.BaseType() != nil {
					if IsSubType(typedSubType.BaseType(), typedSuperType.BaseType()) {
						typedSubTypeTypeArguments := typedSubType.TypeArguments()
						typedSuperTypeTypeArguments := typedSuperType.TypeArguments()
						if len(typedSubTypeTypeArguments) != len(typedSuperTypeTypeArguments) {
							return false
						}

						for i, source := range typedSubTypeTypeArguments {
							target := typedSuperTypeTypeArguments[i]
							if !(IsSubType(source, target)) {
								return false
							}
						}

						return true
					}

				}

			}

		}

		return IsParameterizedSubType(subType, typedSuperType)

	}

	return false
}
