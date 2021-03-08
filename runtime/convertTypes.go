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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// ExportType converts a runtime type to its corresponding Go representation.
func ExportType(t sema.Type, results map[sema.TypeID]cadence.Type) cadence.Type {

	typeID := t.ID()
	if result, ok := results[typeID]; ok {
		return result
	}

	result := func() cadence.Type {
		switch t := t.(type) {
		case *sema.OptionalType:
			return exportOptionalType(t, results)
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
			return exportVariableSizedType(t, results)
		case *sema.ConstantSizedType:
			return exportConstantSizedType(t, results)
		case *sema.CompositeType:
			return exportCompositeType(t, results)
		case *sema.InterfaceType:
			return exportInterfaceType(t, results)
		case *sema.DictionaryType:
			return exportDictionaryType(t, results)
		case *sema.FunctionType:
			return exportFunctionType(t, results)
		case *sema.AddressType:
			return cadence.AddressType{}
		case *sema.ReferenceType:
			return exportReferenceType(t, results)
		case *sema.RestrictedType:
			return exportRestrictedType(t, results)
		case *sema.CheckedFunctionType:
			return exportFunctionType(t.FunctionType, results)
		case *sema.CapabilityType:
			return exportCapabilityType(t, results)
		}

		switch t {
		case sema.PathType:
			return cadence.PathType{}
		case sema.StoragePathType:
			return cadence.StoragePathType{}
		case sema.PrivatePathType:
			return cadence.PrivatePathType{}
		case sema.PublicPathType:
			return cadence.PublicPathType{}
		case sema.CapabilityPathType:
			return cadence.CapabilityPathType{}
		case sema.NeverType:
			return cadence.NeverType{}
		case sema.VoidType:
			return cadence.VoidType{}
		case sema.InvalidType:
			return nil
		case sema.MetaType:
			return cadence.MetaType{}
		case sema.BoolType:
			return cadence.BoolType{}
		case sema.CharacterType:
			return cadence.CharacterType{}
		case sema.AnyType:
			return cadence.AnyType{}
		case sema.AnyStructType:
			return cadence.AnyStructType{}
		case sema.AnyResourceType:
			return cadence.AnyResourceType{}
		case sema.BlockType:
			return cadence.BlockType{}
		case sema.StringType:
			return cadence.StringType{}
		}

		panic(fmt.Sprintf("cannot export type of type %T", t))
	}()

	results[typeID] = result

	return result
}

func exportOptionalType(t *sema.OptionalType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedType := ExportType(t.Type, results)

	return cadence.OptionalType{
		Type: convertedType,
	}
}

func exportVariableSizedType(t *sema.VariableSizedType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedElement := ExportType(t.Type, results)

	return cadence.VariableSizedArrayType{
		ElementType: convertedElement,
	}
}

func exportConstantSizedType(t *sema.ConstantSizedType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedElement := ExportType(t.Type, results)

	return cadence.ConstantSizedArrayType{
		Size:        uint(t.Size),
		ElementType: convertedElement,
	}
}

func exportCompositeType(t *sema.CompositeType, results map[sema.TypeID]cadence.Type) (result cadence.CompositeType) {

	fieldMembers := make([]*sema.Member, 0, len(t.Fields))

	for _, identifier := range t.Fields {
		member, ok := t.Members.Get(identifier)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		if member.IgnoreInSerialization {
			continue
		}

		fieldMembers = append(fieldMembers, member)
	}

	fields := make([]cadence.Field, len(fieldMembers))

	switch t.Kind {
	case common.CompositeKindStructure:
		result = &cadence.StructType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindResource:
		result = &cadence.ResourceType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindEvent:
		result = &cadence.EventType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindContract:
		result = &cadence.ContractType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindEnum:
		result = &cadence.EnumType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
			RawType:             ExportType(t.EnumRawType, results),
		}

	default:
		panic(fmt.Sprintf("cannot export composite type %v of unknown kind %v", t, t.Kind))
	}

	// NOTE: ensure to set the result before recursively export field types

	results[t.ID()] = result

	for i, member := range fieldMembers {
		convertedFieldType := ExportType(member.TypeAnnotation.Type, results)

		fields[i] = cadence.Field{
			Identifier: member.Identifier.Identifier,
			Type:       convertedFieldType,
		}
	}

	return
}

func exportInterfaceType(t *sema.InterfaceType, results map[sema.TypeID]cadence.Type) (result cadence.InterfaceType) {

	fieldMembers := make([]*sema.Member, 0, len(t.Fields))

	for _, identifier := range t.Fields {
		member, ok := t.Members.Get(identifier)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		if member.IgnoreInSerialization {
			continue
		}

		fieldMembers = append(fieldMembers, member)
	}

	fields := make([]cadence.Field, len(fieldMembers))

	switch t.CompositeKind {
	case common.CompositeKindStructure:
		result = &cadence.StructInterfaceType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindResource:
		result = &cadence.ResourceInterfaceType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	case common.CompositeKindContract:
		result = &cadence.ContractInterfaceType{
			Location:            t.Location,
			QualifiedIdentifier: t.QualifiedIdentifier(),
			Fields:              fields,
		}

	default:
		panic(fmt.Sprintf("cannot export interface type %v of unknown kind %v", t, t.CompositeKind))
	}

	// NOTE: ensure to set the result before recursively export field types

	results[t.ID()] = result

	for i, member := range fieldMembers {
		convertedFieldType := ExportType(member.TypeAnnotation.Type, results)

		fields[i] = cadence.Field{
			Identifier: member.Identifier.Identifier,
			Type:       convertedFieldType,
		}
	}

	return
}

func exportDictionaryType(t *sema.DictionaryType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedKeyType := ExportType(t.KeyType, results)
	convertedElementType := ExportType(t.ValueType, results)

	return cadence.DictionaryType{
		KeyType:     convertedKeyType,
		ElementType: convertedElementType,
	}
}

func exportFunctionType(t *sema.FunctionType, results map[sema.TypeID]cadence.Type) cadence.Type {

	convertedParameters := make([]cadence.Parameter, len(t.Parameters))

	for i, parameter := range t.Parameters {
		convertedParameterType := ExportType(parameter.TypeAnnotation.Type, results)

		convertedParameters[i] = cadence.Parameter{
			Label:      parameter.Label,
			Identifier: parameter.Identifier,
			Type:       convertedParameterType,
		}
	}

	convertedReturnType := ExportType(t.ReturnTypeAnnotation.Type, results)

	return cadence.Function{
		Parameters: convertedParameters,
		ReturnType: convertedReturnType,
	}.WithID(string(t.ID()))
}

func exportReferenceType(t *sema.ReferenceType, results map[sema.TypeID]cadence.Type) cadence.ReferenceType {
	convertedType := ExportType(t.Type, results)

	return cadence.ReferenceType{
		Authorized: t.Authorized,
		Type:       convertedType,
	}.WithID(string(t.ID()))
}

func exportRestrictedType(t *sema.RestrictedType, results map[sema.TypeID]cadence.Type) cadence.RestrictedType {

	convertedType := ExportType(t.Type, results)

	restrictions := make([]cadence.Type, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = ExportType(restriction, results)
	}

	return cadence.RestrictedType{
		Type:         convertedType,
		Restrictions: restrictions,
	}.WithID(string(t.ID()))
}

func exportCapabilityType(t *sema.CapabilityType, results map[sema.TypeID]cadence.Type) cadence.CapabilityType {

	var borrowType cadence.Type
	if t.BorrowType != nil {
		borrowType = ExportType(t.BorrowType, results)
	}

	return cadence.CapabilityType{
		BorrowType: borrowType,
	}.WithID(string(t.ID()))
}
