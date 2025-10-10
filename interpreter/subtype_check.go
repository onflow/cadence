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

		return false
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

func PermitsAccess(superTypeAccess, subtypeAccess Authorization) bool {
	// TODO:
	return false
}

func IsIntersectionSubset(superType *IntersectionStaticType, subType StaticType) bool {
	// TODO:
	return false
}

func AreTypeParamsEqual(source, target FunctionStaticType) bool {
	return sema.AreTypeParamsEqual(source.Type, target.Type)
}

func AreParamsContravariant(source, target FunctionStaticType) bool {
	return sema.AreParamsContravariant(source.Type, target.Type)
}

func AreReturnsCovariant(source, target FunctionStaticType) bool {
	return sema.AreReturnsCovariant(source.Type, target.Type)
}

func AreConstructorsEqual(source, target FunctionStaticType) bool {
	return sema.AreConstructorsEqual(source.Type, target.Type)
}

func IsParameterizedSubType(source, target StaticType) bool {
	// TODO:
	return false
}
