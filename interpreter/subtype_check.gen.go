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

import "github.com/onflow/cadence/sema"

func checkSubTypeWithoutEquality_gen(typeConverter TypeConverter, subType StaticType, superType StaticType) bool {
	if subType == PrimitiveStaticTypeNever {
		return true
	}

	switch superType {
	case PrimitiveStaticTypeAny:
		return true

	case PrimitiveStaticTypeAnyStruct:
		return !(IsResourceType(typeConverter, subType)) &&
			subType != PrimitiveStaticTypeAny

	case PrimitiveStaticTypeAnyResource:
		return IsResourceType(typeConverter, subType)

	case PrimitiveStaticTypeAnyResourceAttachment:
		return isAttachmentType(typeConverter, subType) &&
			IsResourceType(typeConverter, subType)

	case PrimitiveStaticTypeAnyStructAttachment:
		return isAttachmentType(typeConverter, subType) &&
			!(IsResourceType(typeConverter, subType))

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

		// TODO: Maybe remove since these predicates only need to check for strict-subtyping, without the "equality".
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

		// Optionals are covariant: T? <: U? if T <: U
		switch typedSubType := subType.(type) {
		case *OptionalStaticType:
			return IsSubType(typeConverter, typedSubType.Type, typedSuperType.Type)
		}

		// T <: U? if T <: U
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

			// The authorization of the subtype reference must be usable in all situations where the supertype reference is usable.
			return PermitsAccess(typeConverter, typedSuperType.Authorization, typedSubType.Authorization) &&
				// References are covariant in their referenced type
				IsSubType(typeConverter, typedSubType.ReferencedType, typedSuperType.ReferencedType)
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

			return false
		case *CompositeStaticType:
			return false
		}

		return false

	case *InterfaceStaticType:
		switch typedSubType := subType.(type) {
		case *CompositeStaticType:
			typedSemaSuperType := typeConverter.SemaTypeFromStaticType(typedSuperType).(*sema.InterfaceType)
			typedSemaSubType := typeConverter.SemaTypeFromStaticType(typedSubType).(*sema.CompositeType)
			return typedSemaSubType.Kind == typedSemaSuperType.CompositeKind &&
				typedSemaSubType.EffectiveInterfaceConformanceSet().Contains(typedSemaSuperType)
		case *IntersectionStaticType:
			typedSemaSuperType := typeConverter.SemaTypeFromStaticType(typedSuperType).(*sema.InterfaceType)
			typedSemaSubType := typeConverter.SemaTypeFromStaticType(typedSubType).(*sema.IntersectionType)
			return typedSemaSubType.EffectiveIntersectionSet().Contains(typedSemaSuperType)
		case *InterfaceStaticType:
			typedSemaSuperType := typeConverter.SemaTypeFromStaticType(typedSuperType).(*sema.InterfaceType)
			typedSemaSubType := typeConverter.SemaTypeFromStaticType(typedSubType).(*sema.InterfaceType)
			return typedSemaSubType.EffectiveInterfaceConformanceSet().Contains(typedSemaSuperType)
		}

		return false

	case *IntersectionStaticType:
		switch typedSuperType.LegacyType {
		case nil,
			PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:

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
			case PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			// An intersection type `T{Us}`
			// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
			switch typedSubType := subType.(type) {
			case *IntersectionStaticType:
				// An intersection type `{Us}` is a subtype of an intersection type `{Vs}` / `{Vs}` / `{Vs}`:
				// when `Vs` is a subset of `Us`.
				if typedSubType.LegacyType == nil &&
					IsIntersectionSubset(typeConverter, typedSuperType, typedSubType) {
					return true
				}

				// When `T == AnyResource || T == AnyStruct || T == Any`:
				// if the intersection type of the subtype
				// is a subtype of the intersection supertype,
				// and `Vs` is a subset of `Us`.
				switch typedSubType.LegacyType {
				case PrimitiveStaticTypeAny,
					PrimitiveStaticTypeAnyStruct,
					PrimitiveStaticTypeAnyResource:

					// Below two combination is repeated several times below.
					// Maybe combine them to produce a single predicate.
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typeConverter, typedSubType.LegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typeConverter, typedSuperType, typedSubType)
				}

				// When `T != AnyResource && T != AnyStruct && T != Any`:
				// if the intersection type of the subtype
				// is a subtype of the intersection supertype,
				// and `T` conforms to `Vs`.
				// `Us` and `Vs` do *not* have to be subsets.
				switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
				case *CompositeStaticType:
					return (typedSuperType.LegacyType == nil ||
						IsSubType(typeConverter, typedSubTypeLegacyType, typedSuperType.LegacyType)) &&
						IsIntersectionSubset(typeConverter, typedSuperType, typedSubTypeLegacyType)
				}

				return false
			case ConformingStaticType:
				return (typedSuperType.LegacyType == nil ||
					IsSubType(typeConverter, typedSubType, typedSuperType.LegacyType)) &&
					IsIntersectionSubset(typeConverter, typedSuperType, typedSubType)
			}

			return false
		}

		// A type `T`
		// is a subtype of an intersection type `AnyResource{Vs}` / `AnyStruct{Vs}` / `Any{Vs}`:
		// not statically.
		switch subType {
		case PrimitiveStaticTypeAny,
			PrimitiveStaticTypeAnyStruct,
			PrimitiveStaticTypeAnyResource:
			return false
		}

		// An intersection type `T{Us}`
		// is a subtype of an intersection type `V{Ws}`:
		switch typedSubType := subType.(type) {
		case *IntersectionStaticType:

			// When `T == AnyResource || T == AnyStruct || T == Any`:
			// not statically.
			switch typedSubType.LegacyType {
			case nil,
				PrimitiveStaticTypeAny,
				PrimitiveStaticTypeAnyStruct,
				PrimitiveStaticTypeAnyResource:
				return false
			}

			switch typedSubTypeLegacyType := typedSubType.LegacyType.(type) {
			case *CompositeStaticType:

				// When `T != AnyResource && T != AnyStructType && T != Any`: if `T == V`.
				// `Us` and `Ws` do *not* have to be subsets:
				// The owner may freely restrict and unrestrict.
				return typedSubTypeLegacyType == typedSuperType.LegacyType
			}

			return false
		case *CompositeStaticType:
			return IsSubType(typeConverter, typedSubType, typedSuperType.LegacyType)
		}

		return false

	case FunctionStaticType:
		switch typedSubType := subType.(type) {
		case FunctionStaticType:

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
					!(sema.IsSubType(target.TypeAnnotation.Type, source.TypeAnnotation.Type)) {
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

	case ParameterizedStaticType:
		switch typedSubType := subType.(type) {
		case ParameterizedStaticType:
			if typedSubType.BaseType() != nil {
				if typedSuperType.BaseType() != nil {
					if IsSubType(typeConverter, typedSubType.BaseType(), typedSuperType.BaseType()) {
						typedSubTypeTypeArguments := typedSubType.TypeArguments()
						typedSuperTypeTypeArguments := typedSuperType.TypeArguments()
						if len(typedSubTypeTypeArguments) != len(typedSuperTypeTypeArguments) {
							return false
						}

						for i, source := range typedSubTypeTypeArguments {
							target := typedSuperTypeTypeArguments[i]
							if !(IsSubType(typeConverter, source, target)) {
								return false
							}
						}

						return true
					}

					return false
				}

				return IsSubType(typeConverter, typedSubType.BaseType(), typedSuperType)
			}

		}

		return false

	}

	return false
}
