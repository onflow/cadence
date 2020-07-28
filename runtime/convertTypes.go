/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

// exportType converts a runtime type to its corresponding Go representation.
func exportType(typ sema.Type) cadence.Type {
	switch t := typ.(type) {
	case *sema.AnyType:
		return cadence.AnyType{}
	case *sema.AnyStructType:
		return cadence.AnyStructType{}
	case *sema.AnyResourceType:
		return cadence.AnyResourceType{}
	case *sema.VoidType:
		return cadence.VoidType{}
	case *sema.NeverType:
		return cadence.NeverType{}
	case *sema.MetaType:
		return cadence.MetaType{}
	case *sema.OptionalType:
		return exportOptionalType(t)
	case *sema.BoolType:
		return cadence.BoolType{}
	case *sema.StringType:
		return cadence.StringType{}
	case *sema.CharacterType:
		return cadence.CharacterType{}
	case *sema.NumberType:
		return cadence.NumberType{}
	case *sema.SignedNumberType:
		return cadence.SignedNumberType{}
	case *sema.IntegerType:
		return cadence.IntegerType{}
	case *sema.SignedIntegerType:
		return cadence.SignedIntegerType{}
	case *sema.FixedPointType:
		return cadence.FixedPointType{}
	case *sema.SignedFixedPointType:
		return cadence.SignedFixedPointType{}
	case *sema.IntType:
		return cadence.IntType{}
	case *sema.Int8Type:
		return cadence.Int8Type{}
	case *sema.Int16Type:
		return cadence.Int16Type{}
	case *sema.Int32Type:
		return cadence.Int32Type{}
	case *sema.Int64Type:
		return cadence.Int64Type{}
	case *sema.Int128Type:
		return cadence.Int128Type{}
	case *sema.Int256Type:
		return cadence.Int256Type{}
	case *sema.UIntType:
		return cadence.UIntType{}
	case *sema.UInt8Type:
		return cadence.UInt8Type{}
	case *sema.UInt16Type:
		return cadence.UInt16Type{}
	case *sema.UInt32Type:
		return cadence.UInt32Type{}
	case *sema.UInt64Type:
		return cadence.UInt64Type{}
	case *sema.UInt128Type:
		return cadence.UInt128Type{}
	case *sema.UInt256Type:
		return cadence.UInt256Type{}
	case *sema.Word8Type:
		return cadence.Word8Type{}
	case *sema.Word16Type:
		return cadence.Word16Type{}
	case *sema.Word32Type:
		return cadence.Word32Type{}
	case *sema.Word64Type:
		return cadence.Word64Type{}
	case *sema.Fix64Type:
		return cadence.Fix64Type{}
	case *sema.UFix64Type:
		return cadence.UFix64Type{}
	case *sema.VariableSizedType:
		return exportVariableSizedType(t)
	case *sema.ConstantSizedType:
		return exportConstantSizedType(t)
	case *sema.CompositeType:
		return exportCompositeType(t)
	case *sema.InterfaceType:
		return exportInterfaceType(t)
	case *sema.DictionaryType:
		return exportDictionaryType(t)
	case *sema.FunctionType:
		return exportFunctionType(t)
	case *sema.AddressType:
		return cadence.AddressType{}
	case *sema.ReferenceType:
		return exportReferenceType(t)
	case *sema.RestrictedType:
		return exportRestrictedType(t)
	case *stdlib.BlockType:
		return cadence.BlockType{}
	case *sema.PathType:
		return cadence.PathType{}
	case *sema.CheckedFunctionType:
		return exportFunctionType(t.FunctionType)
	case *sema.CapabilityType:
		return exportCapabilityType(t)
	case *sema.AuthAccountType:
		return cadence.AuthAccountType{}
	case *sema.PublicAccountType:
		return cadence.PublicAccountType{}
	}

	panic(fmt.Sprintf("cannot export type of type %T", typ))
}

func exportOptionalType(t *sema.OptionalType) cadence.Type {
	convertedType := exportType(t.Type)

	return cadence.OptionalType{Type: convertedType}
}

func exportVariableSizedType(t *sema.VariableSizedType) cadence.Type {
	convertedElement := exportType(t.Type)

	return cadence.VariableSizedArrayType{ElementType: convertedElement}
}

func exportConstantSizedType(t *sema.ConstantSizedType) cadence.Type {
	convertedElement := exportType(t.Type)

	return cadence.ConstantSizedArrayType{
		Size:        uint(t.Size),
		ElementType: convertedElement,
	}
}

func exportCompositeType(t *sema.CompositeType) cadence.CompositeType {

	fields := make([]cadence.Field, 0, len(t.Fields))

	for _, identifier := range t.Fields {
		member := t.Members[identifier]

		if member.IgnoreInSerialization {
			continue
		}

		convertedFieldType := exportType(member.TypeAnnotation.Type)

		fields = append(fields, cadence.Field{
			Identifier: identifier,
			Type:       convertedFieldType,
		})
	}

	id := string(t.ID())

	switch t.Kind {
	case common.CompositeKindStructure:
		return cadence.StructType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindResource:
		return cadence.ResourceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindEvent:
		return cadence.EventType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindContract:
		return cadence.ContractType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	}

	panic(fmt.Sprintf("cannot export composite type %v of unknown kind %v", t, t.Kind))
}

func exportInterfaceType(t *sema.InterfaceType) cadence.InterfaceType {

	fields := make([]cadence.Field, 0, len(t.Members))

	for _, identifier := range t.Fields {
		member := t.Members[identifier]

		if member.IgnoreInSerialization {
			continue
		}

		convertedFieldType := exportType(member.TypeAnnotation.Type)

		fields = append(fields, cadence.Field{
			Identifier: identifier,
			Type:       convertedFieldType,
		})
	}

	id := string(t.ID())

	switch t.CompositeKind {
	case common.CompositeKindStructure:
		return cadence.StructInterfaceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindResource:
		return cadence.ResourceInterfaceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	case common.CompositeKindContract:
		return cadence.ContractInterfaceType{
			TypeID:     id,
			Identifier: t.Identifier,
			Fields:     fields,
		}
	}

	panic(fmt.Sprintf("cannot export interface type %v of unknown kind %v", t, t.CompositeKind))
}

func exportDictionaryType(t *sema.DictionaryType) cadence.Type {
	convertedKeyType := exportType(t.KeyType)
	convertedElementType := exportType(t.ValueType)

	return cadence.DictionaryType{
		KeyType:     convertedKeyType,
		ElementType: convertedElementType,
	}
}

func exportFunctionType(t *sema.FunctionType) cadence.Type {
	convertedReturnType := exportType(t.ReturnTypeAnnotation.Type)

	parameters := make([]cadence.Parameter, len(t.Parameters))

	for i, parameter := range t.Parameters {
		convertedParameterType := exportType(parameter.TypeAnnotation.Type)

		parameters[i] = cadence.Parameter{
			Label:      parameter.Label,
			Identifier: parameter.Identifier,
			Type:       convertedParameterType,
		}
	}

	return cadence.Function{
		Parameters: parameters,
		ReturnType: convertedReturnType,
	}.WithID(string(t.ID()))
}

func exportReferenceType(t *sema.ReferenceType) cadence.ReferenceType {
	return cadence.ReferenceType{
		Authorized: t.Authorized,
		Type:       exportType(t.Type),
	}.WithID(string(t.ID()))
}

func exportRestrictedType(t *sema.RestrictedType) cadence.RestrictedType {

	restrictions := make([]cadence.Type, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = exportType(restriction)
	}

	return cadence.RestrictedType{
		Type:         exportType(t.Type),
		Restrictions: restrictions,
	}.WithID(string(t.ID()))
}

func exportCapabilityType(t *sema.CapabilityType) cadence.CapabilityType {

	var borrowType cadence.Type
	if t.BorrowType != nil {
		borrowType = exportType(t.BorrowType)
	}

	return cadence.CapabilityType{
		BorrowType: borrowType,
	}.WithID(string(t.ID()))
}
