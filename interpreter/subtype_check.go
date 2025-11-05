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

//go:generate go run ./type_check_gen subtype_check.gen.go

var FunctionPurityView = sema.FunctionPurityView

func isAttachmentType(t StaticType) bool {
	switch t {
	case PrimitiveStaticTypeAnyResourceAttachment, PrimitiveStaticTypeAnyStructAttachment:
		return true
	default:
		_, ok := t.(*CompositeStaticType)
		if !ok {
			return false
		}

		// TODO:
		//return compositeType.Kind == common.CompositeKindAttachment
		return true
	}
}

func IsHashableStructType(typeConverter TypeConverter, typ StaticType) bool {
	switch typ {
	case PrimitiveStaticTypeNever,
		PrimitiveStaticTypeBool,
		PrimitiveStaticTypeCharacter,
		PrimitiveStaticTypeString,
		PrimitiveStaticTypeMetaType,
		PrimitiveStaticTypeHashableStruct:
		return true
	default:
		_, ok := typ.(*CompositeStaticType)
		if !ok {
			// TODO:
			//return  compositeType.Kind == common.CompositeKindEnum
			return false
		}

		return IsSubType(typeConverter, typ, PrimitiveStaticTypeNumber) ||
			IsSubType(typeConverter, typ, PrimitiveStaticTypePath)
	}
}

func IsResourceType(typ StaticType) bool {
	switch typ := typ.(type) {
	case PrimitiveStaticType:
		// Primitive static type to sema type conversion is just a switch case.
		// So not much overhead there.
		// TODO: Maybe have these precomputed.
		return typ.SemaType().IsResourceType()
	default:
		// TODO:
		return false
	}
}

func PermitsAccess(superTypeAccess, subTypeAccess Authorization) bool {
	// TODO:
	return true
}

func IsIntersectionSubset(superType *IntersectionStaticType, subType StaticType) bool {
	// TODO
	return true
}

func AreReturnsCovariant(source, target FunctionStaticType) bool {
	return AreReturnsCovariant(source, target)
}

func IsParameterizedSubType(typeConverter TypeConverter, subType StaticType, superType StaticType) bool {
	typedSubType, ok := subType.(ParameterizedType)
	if !ok {
		return false
	}

	if baseType := typedSubType.BaseType(); baseType != nil {
		return IsSubType(typeConverter, baseType, superType)
	}

	return false
}

type Equatable[T any] interface {
	comparable
	Equal(other T) bool
}

func deepEquals[T Equatable[T]](source, target T) bool {
	var empty T
	if source == empty {
		return target == empty
	}

	return source.Equal(target)
}
