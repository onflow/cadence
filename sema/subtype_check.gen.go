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
		return subType ==// TODO: Maybe remove since these predicates only need to check for strict-subtyping, without the "equality".
		SignedNumberType ||
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

		// Optionals are covariant: T? <: U? if T <: U
		switch typedSubType := subType.(type) {
		case *OptionalType:
			return IsSubType(typedSubType.Type, typedSuperType.Type)
		}

		// T <: U? if T <: U
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

			// The authorization of the subtype reference must be usable in all situations where the supertype reference is usable.
			return PermitsAccess(typedSuperType.Authorization, typedSubType.Authorization) &&
				// References are covariant in their referenced type
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

			return false
		case *CompositeType:
			return false
		}

		return false

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

		return false

	case *IntersectionType:
		switch typedSuperType.LegacyType {
		case nil,
			AnyType,
			AnyStructType,
			AnyResourceType:

			// `Any` is a subtype of an intersection type
			//  - `Any{Us}: not statically.`
			//  - `AnyStruct{Us}`: never;
			//  - `AnyResource{Us}`: never;
			//
			// `AnyStruct` is a subtype of an intersection type
			//  - `AnyStruct{Us}`: not statically.
			//  - `AnyResource{Us}`: never;
			//  - `Any{Us}`: not statically.
			//
			// `AnyResource` is a subtype of an intersection type
			//  - `AnyResource{Us}`: not statically;
			//  - `AnyStruct{Us}`: never.
			//  - `Any{Us}`: not statically;
			switch subType {
			case AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			// An intersection type `T{Us}`
			// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
			switch typedSubType := subType.(type) {
			case *IntersectionType:
				// An intersection type `{Us}` is a subtype of an intersection type `{Vs}` / `{Vs}` / `{Vs}`:
				// when `Vs` is a subset of `Us`.
				if typedSubType.LegacyType == nil &&
					IsIntersectionSubset(typedSuperType, typedSubType) {
					return true
				}

				// When `T == AnyResource || T == AnyStruct || T == Any`:
				// if the intersection type of the subtype
				// is a subtype of the intersection supertype,
				// and `Vs` is a subset of `Us`.
				switch typedSubType.LegacyType {
				case AnyType,
					AnyStructType,
					AnyResourceType:

					// Below two combination is repeated several times below.
					// Maybe combine them to produce a single predicate.
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typedSubType.LegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubType)
				}

				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if the intersection type of the subtype
				// is a subtype of the intersection supertype,
				// and `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.
				switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
				case *CompositeType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typedSubTypeLegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typedSuperType, typedSubTypeLegacyType)
				}

				return false
			case ConformingType:
				return (typedSuperType.LegacyType == nil ||
					IsSubType(typedSubType, typedSuperType.LegacyType)) &&
					IsIntersectionSubset(typedSuperType, typedSubType)
			}

			return false
		}

		// A type `T`
		// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
		// not statically.
		switch subType {
		case AnyType,
			AnyStructType,
			AnyResourceType:
			return false
		}

		// An intersection type `T{Us}`
		// is a subtype of an intersection type `V{Ws}`:
		switch typedSubType := subType.(type) {
		case *IntersectionType:

			// When `T == AnyResource || T == AnyStruct || T == Any`:
			// not statically.
			switch typedSubType.LegacyType {
			case nil,
				AnyType,
				AnyStructType,
				AnyResourceType:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeType:

				// When `T != AnyResource && T != AnyStructType && T != Any`: if `T == V`.
				// `Us` and `Ws` do *not* have to be subsets:
				// The owner may freely restrict and unrestrict.
				return typedSubTypeLegacyType == typedSuperType.LegacyType
			}

			return false
		case *CompositeType:
			return IsSubType(typedSubType, typedSuperType.LegacyType)
		}

		return false

	case *FunctionType:
		switch typedSubType := subType.(type) {
		case *FunctionType:

			// View functions are subtypes of impure functions
			switch typedSubType.Purity {
			case typedSuperType.Purity,
				FunctionPurityView:

				// Type parameters must be equivalent. This is because for subtyping of functions,
				// parameters must be *contravariant/supertypes*, whereas, return types must be *covariant/subtypes*.
				// Since type parameters can be used in both parameters and return types, inorder to satisfies both above
				// conditions, bound type of type parameters can only be strictly equal, but not subtypes/supertypes of one another.
				typedSubTypeTypeParameters := typedSubType.TypeParameters
				typedSuperTypeTypeParameters := typedSuperType.TypeParameters
				if len(typedSubTypeTypeParameters) != len(typedSuperTypeTypeParameters) {
					return false
				}

				for i, source := range typedSubTypeTypeParameters {
					target := typedSuperTypeTypeParameters[i]
					if !(deepEquals(source.TypeBound, target.TypeBound)) {
						return false
					}
				}

				// Functions are contravariant in their parameter types.
				typedSubTypeParameters := typedSubType.Parameters
				typedSuperTypeParameters := typedSuperType.Parameters
				if len(typedSubTypeParameters) != len(typedSuperTypeParameters) {
					return false
				}

				for i, source := range typedSubTypeParameters {
					target := typedSuperTypeParameters[i]
					if

					// Note the super-type is the subtype's parameter
					// because the parameters are contravariant.
					!(IsSubType(target.TypeAnnotation.Type, source.TypeAnnotation.Type)) {
						return false
					}
				}

				return deepEquals(typedSubType.Arity, typedSuperType.Arity) &&
					// Functions are covariant in their return type.
					(AreReturnsCovariant(typedSubType, typedSuperType) &&
						typedSubType.IsConstructor == typedSuperType.IsConstructor)
			}

			return false
		}

		return false

	case ParameterizedType:
		switch typedSubType := subType.(type) {
		case ParameterizedType:
			if typedSubType.BaseType() != nil {
				if typedSuperType.BaseType() != nil {
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

					return false
				}

				return IsSubType(typedSubType.BaseType(), typedSuperType)
			}

		}

		return false

	}

	return false
}
