/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// ExportType converts a runtime type to its corresponding Go representation.
func ExportType(t sema.Type, results map[sema.TypeID]cadence.Type) cadence.Type {
	if t == nil {
		return nil
	}

	typeID := t.ID()
	if result, ok := results[typeID]; ok {
		return result
	}

	result := func() cadence.Type {
		switch t := t.(type) {
		case *sema.OptionalType:
			return exportOptionalType(t, results)
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
		case *sema.CapabilityType:
			return exportCapabilityType(t, results)
		}

		switch t {
		case sema.NumberType:
			return cadence.NumberType{}
		case sema.SignedNumberType:
			return cadence.SignedNumberType{}
		case sema.IntegerType:
			return cadence.IntegerType{}
		case sema.SignedIntegerType:
			return cadence.SignedIntegerType{}
		case sema.FixedPointType:
			return cadence.FixedPointType{}
		case sema.SignedFixedPointType:
			return cadence.SignedFixedPointType{}
		case sema.IntType:
			return cadence.IntType{}
		case sema.Int8Type:
			return cadence.Int8Type{}
		case sema.Int16Type:
			return cadence.Int16Type{}
		case sema.Int32Type:
			return cadence.Int32Type{}
		case sema.Int64Type:
			return cadence.Int64Type{}
		case sema.Int128Type:
			return cadence.Int128Type{}
		case sema.Int256Type:
			return cadence.Int256Type{}
		case sema.UIntType:
			return cadence.UIntType{}
		case sema.UInt8Type:
			return cadence.UInt8Type{}
		case sema.UInt16Type:
			return cadence.UInt16Type{}
		case sema.UInt32Type:
			return cadence.UInt32Type{}
		case sema.UInt64Type:
			return cadence.UInt64Type{}
		case sema.UInt128Type:
			return cadence.UInt128Type{}
		case sema.UInt256Type:
			return cadence.UInt256Type{}
		case sema.Word8Type:
			return cadence.Word8Type{}
		case sema.Word16Type:
			return cadence.Word16Type{}
		case sema.Word32Type:
			return cadence.Word32Type{}
		case sema.Word64Type:
			return cadence.Word64Type{}
		case sema.Fix64Type:
			return cadence.Fix64Type{}
		case sema.UFix64Type:
			return cadence.UFix64Type{}
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
		case sema.AccountKeyType:
			return cadence.AccountKeyType{}
		case sema.PublicAccountContractsType:
			return cadence.PublicAccountContractsType{}
		case sema.AuthAccountContractsType:
			return cadence.AuthAccountContractsType{}
		case sema.PublicAccountKeysType:
			return cadence.PublicAccountKeysType{}
		case sema.AuthAccountKeysType:
			return cadence.AuthAccountKeysType{}
		case sema.PublicAccountType:
			return cadence.PublicAccountType{}
		case sema.AuthAccountType:
			return cadence.AuthAccountType{}
		case sema.DeployedContractType:
			return cadence.DeployedContractType{}
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

	return cadence.FunctionType{
		Parameters: convertedParameters,
		ReturnType: convertedReturnType,
	}.WithID(string(t.ID()))
}

func exportReferenceType(t *sema.ReferenceType, results map[sema.TypeID]cadence.Type) cadence.ReferenceType {
	convertedType := ExportType(t.Type, results)

	return cadence.ReferenceType{
		Authorized: t.Authorized,
		Type:       convertedType,
	}
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
	}
}

func importInterfaceType(memoryGauge common.MemoryGauge, t cadence.InterfaceType) interpreter.InterfaceStaticType {
	return interpreter.NewInterfaceStaticType(
		memoryGauge,
		t.InterfaceTypeLocation(),
		t.InterfaceTypeQualifiedIdentifier(),
	)
}

func importCompositeType(memoryGauge common.MemoryGauge, t cadence.CompositeType) interpreter.CompositeStaticType {
	return interpreter.NewCompositeStaticType(
		memoryGauge,
		t.CompositeTypeLocation(),
		t.CompositeTypeQualifiedIdentifier(),
		"", // intentionally empty
	)
}

func ImportType(memoryGauge common.MemoryGauge, t cadence.Type) interpreter.StaticType {
	switch t := t.(type) {
	case cadence.AnyType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAny)
	case cadence.AnyStructType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAnyStruct)
	case cadence.AnyResourceType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAnyResource)
	case cadence.OptionalType:
		return interpreter.NewOptionalStaticType(memoryGauge, ImportType(memoryGauge, t.Type))
	case cadence.MetaType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeMetaType)
	case cadence.VoidType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeVoid)
	case cadence.NeverType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeNever)
	case cadence.BoolType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeBool)
	case cadence.StringType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeString)
	case cadence.CharacterType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeCharacter)
	case cadence.AddressType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAddress)
	case cadence.NumberType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeNumber)
	case cadence.SignedNumberType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeSignedNumber)
	case cadence.IntegerType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInteger)
	case cadence.SignedIntegerType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeSignedInteger)
	case cadence.FixedPointType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeFixedPoint)
	case cadence.SignedFixedPointType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeSignedFixedPoint)
	case cadence.IntType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt)
	case cadence.Int8Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt8)
	case cadence.Int16Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt16)
	case cadence.Int32Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt32)
	case cadence.Int64Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt64)
	case cadence.Int128Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt128)
	case cadence.Int256Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeInt256)
	case cadence.UIntType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt)
	case cadence.UInt8Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt8)
	case cadence.UInt16Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt16)
	case cadence.UInt32Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt32)
	case cadence.UInt64Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt64)
	case cadence.UInt128Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt128)
	case cadence.UInt256Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUInt256)
	case cadence.Word8Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeWord8)
	case cadence.Word16Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeWord16)
	case cadence.Word32Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeWord32)
	case cadence.Word64Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeWord64)
	case cadence.Fix64Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeFix64)
	case cadence.UFix64Type:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeUFix64)
	case cadence.VariableSizedArrayType:
		return interpreter.NewVariableSizedStaticType(memoryGauge, ImportType(memoryGauge, t.ElementType))
	case cadence.ConstantSizedArrayType:
		return interpreter.NewConstantSizedStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.ElementType),
			int64(t.Size),
		)
	case cadence.DictionaryType:
		return interpreter.NewDictionaryStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.KeyType),
			ImportType(memoryGauge, t.ElementType),
		)
	case *cadence.StructType,
		*cadence.ResourceType,
		*cadence.EventType,
		*cadence.ContractType,
		*cadence.EnumType:
		return importCompositeType(memoryGauge, t.(cadence.CompositeType))
	case *cadence.StructInterfaceType,
		*cadence.ResourceInterfaceType,
		*cadence.ContractInterfaceType:
		return importInterfaceType(memoryGauge, t.(cadence.InterfaceType))
	case cadence.ReferenceType:
		return interpreter.NewReferenceStaticType(
			memoryGauge,
			t.Authorized,
			ImportType(memoryGauge, t.Type),
			nil,
		)
	case cadence.RestrictedType:
		restrictions := make([]interpreter.InterfaceStaticType, 0, len(t.Restrictions))
		for _, restriction := range t.Restrictions {
			intf, ok := restriction.(cadence.InterfaceType)
			if !ok {
				panic(fmt.Sprintf("cannot export type of type %T", t))
			}
			restrictions = append(restrictions, importInterfaceType(memoryGauge, intf))
		}
		return interpreter.NewRestrictedStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.Type),
			restrictions,
		)
	case cadence.BlockType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeBlock)
	case cadence.CapabilityPathType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeCapabilityPath)
	case cadence.StoragePathType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeStoragePath)
	case cadence.PublicPathType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypePublicPath)
	case cadence.PrivatePathType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypePrivatePath)
	case cadence.CapabilityType:
		return interpreter.NewCapabilityStaticType(memoryGauge, ImportType(memoryGauge, t.BorrowType))
	case cadence.AccountKeyType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAccountKey)
	case cadence.AuthAccountContractsType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAuthAccountContracts)
	case cadence.AuthAccountKeysType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAuthAccountKeys)
	case cadence.AuthAccountType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeAuthAccount)
	case cadence.PublicAccountContractsType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypePublicAccountContracts)
	case cadence.PublicAccountKeysType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypePublicAccountKeys)
	case cadence.PublicAccountType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypePublicAccount)
	case cadence.DeployedContractType:
		return interpreter.NewPrimitiveStaticType(memoryGauge, interpreter.PrimitiveStaticTypeDeployedContract)
	default:
		panic(fmt.Sprintf("cannot export type of type %T", t))
	}
}
