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

import "github.com/onflow/cadence/errors"

//go:generate go run ./type_check_gen subtype_check.gen.go

func IsResourceType(typ Type) bool {
	return typ.IsResourceType()
}

func PermitsAccess(superTypeAccess, subTypeAccess Access) bool {
	return superTypeAccess.PermitsAccess(subTypeAccess)
}

func IsIntersectionSubset(superType *IntersectionType, subType Type) bool {
	switch subType := subType.(type) {
	case *IntersectionType:
		return superType.EffectiveIntersectionSet().
			IsSubsetOf(subType.EffectiveIntersectionSet())
	case ConformingType:
		return superType.EffectiveIntersectionSet().
			IsSubsetOf(subType.EffectiveInterfaceConformanceSet())
	default:
		panic(errors.NewUnreachableError())
	}
}

func AreTypeParamsEqual(source, target *FunctionType) bool {
	if len(source.TypeParameters) != len(target.TypeParameters) {
		return false
	}

	for i, subTypeParameter := range source.TypeParameters {
		superTypeParameter := target.TypeParameters[i]
		if !subTypeParameter.TypeBoundEqual(superTypeParameter.TypeBound) {
			return false
		}
	}

	return true
}

func AreParamsContravariant(source, target *FunctionType) bool {
	// Parameter arity must be equivalent.
	if len(source.Parameters) != len(target.Parameters) {
		return false
	}

	if !source.ArityEqual(target.Arity) {
		return false
	}

	// Functions are contravariant in their parameter types
	for i, subParameter := range source.Parameters {
		superParameter := target.Parameters[i]
		if !IsSubType(
			superParameter.TypeAnnotation.Type,
			subParameter.TypeAnnotation.Type,
		) {
			return false
		}
	}

	return true
}

func AreReturnsCovariant(source, target *FunctionType) bool {
	// Functions are covariant in their return type
	if source.ReturnTypeAnnotation.Type != nil {
		if target.ReturnTypeAnnotation.Type == nil {
			return false
		}

		if !IsSubType(
			source.ReturnTypeAnnotation.Type,
			target.ReturnTypeAnnotation.Type,
		) {
			return false
		}
	} else if target.ReturnTypeAnnotation.Type != nil {
		return false
	}

	return true
}

func AreConstructorsEqual(source, target *FunctionType) bool {
	return source.IsConstructor == target.IsConstructor
}

func AreTypeArgumentsEqual(source, target ParameterizedType) bool {
	subTypeTypeArguments := source.TypeArguments()
	superTypeTypeArguments := target.TypeArguments()

	if len(subTypeTypeArguments) != len(superTypeTypeArguments) {
		return false
	}

	for i, superTypeTypeArgument := range superTypeTypeArguments {
		subTypeTypeArgument := subTypeTypeArguments[i]
		if !IsSubType(subTypeTypeArgument, superTypeTypeArgument) {
			return false
		}
	}

	return true
}
