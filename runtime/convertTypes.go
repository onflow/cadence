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
func ExportType(
	t sema.Type,
	results map[sema.TypeID]cadence.Type,
) cadence.Type {
	if t == nil {
		return nil
	}

	typeID := t.ID()
	if result, ok := results[typeID]; ok {
		return result
	}

	var gauge common.MemoryGauge = nil

	result := func() cadence.Type {
		switch t := t.(type) {
		case *sema.OptionalType:
			return exportOptionalType(gauge, t, results)
		case *sema.VariableSizedType:
			return exportVariableSizedType(gauge, t, results)
		case *sema.ConstantSizedType:
			return exportConstantSizedType(gauge, t, results)
		case *sema.CompositeType:
			return exportCompositeType(gauge, t, results)
		case *sema.InterfaceType:
			return exportInterfaceType(gauge, t, results)
		case *sema.DictionaryType:
			return exportDictionaryType(gauge, t, results)
		case *sema.FunctionType:
			return exportFunctionType(gauge, t, results)
		case *sema.AddressType:
			return cadence.NewMeteredAddressType(gauge)
		case *sema.ReferenceType:
			return exportReferenceType(gauge, t, results)
		case *sema.RestrictedType:
			return exportRestrictedType(gauge, t, results)
		case *sema.CapabilityType:
			return exportCapabilityType(gauge, t, results)
		}

		switch t {
		case sema.NumberType:
			return cadence.NewMeteredNumberType(gauge)
		case sema.SignedNumberType:
			return cadence.NewMeteredSignedNumberType(gauge)
		case sema.IntegerType:
			return cadence.NewMeteredIntegerType(gauge)
		case sema.SignedIntegerType:
			return cadence.NewMeteredSignedIntegerType(gauge)
		case sema.FixedPointType:
			return cadence.NewMeteredFixedPointType(gauge)
		case sema.SignedFixedPointType:
			return cadence.NewMeteredSignedFixedPointType(gauge)
		case sema.IntType:
			return cadence.NewMeteredIntType(gauge)
		case sema.Int8Type:
			return cadence.NewMeteredInt8Type(gauge)
		case sema.Int16Type:
			return cadence.NewMeteredInt16Type(gauge)
		case sema.Int32Type:
			return cadence.NewMeteredInt32Type(gauge)
		case sema.Int64Type:
			return cadence.NewMeteredInt64Type(gauge)
		case sema.Int128Type:
			return cadence.NewMeteredInt128Type(gauge)
		case sema.Int256Type:
			return cadence.NewMeteredInt256Type(gauge)
		case sema.UIntType:
			return cadence.NewMeteredUIntType(gauge)
		case sema.UInt8Type:
			return cadence.NewMeteredUInt8Type(gauge)
		case sema.UInt16Type:
			return cadence.NewMeteredUInt16Type(gauge)
		case sema.UInt32Type:
			return cadence.NewMeteredUInt32Type(gauge)
		case sema.UInt64Type:
			return cadence.NewMeteredUInt64Type(gauge)
		case sema.UInt128Type:
			return cadence.NewMeteredUInt128Type(gauge)
		case sema.UInt256Type:
			return cadence.NewMeteredUInt256Type(gauge)
		case sema.Word8Type:
			return cadence.NewMeteredWord8Type(gauge)
		case sema.Word16Type:
			return cadence.NewMeteredWord16Type(gauge)
		case sema.Word32Type:
			return cadence.NewMeteredWord32Type(gauge)
		case sema.Word64Type:
			return cadence.NewMeteredWord64Type(gauge)
		case sema.Fix64Type:
			return cadence.NewMeteredFix64Type(gauge)
		case sema.UFix64Type:
			return cadence.NewMeteredUFix64Type(gauge)
		case sema.PathType:
			return cadence.NewMeteredPathType(gauge)
		case sema.StoragePathType:
			return cadence.NewMeteredStoragePathType(gauge)
		case sema.PrivatePathType:
			return cadence.NewMeteredPrivatePathType(gauge)
		case sema.PublicPathType:
			return cadence.NewMeteredPublicPathType(gauge)
		case sema.CapabilityPathType:
			return cadence.NewMeteredCapabilityPathType(gauge)
		case sema.NeverType:
			return cadence.NewMeteredNeverType(gauge)
		case sema.VoidType:
			return cadence.NewMeteredVoidType(gauge)
		case sema.InvalidType:
			return nil
		case sema.MetaType:
			return cadence.NewMeteredMetaType(gauge)
		case sema.BoolType:
			return cadence.NewMeteredBoolType(gauge)
		case sema.CharacterType:
			return cadence.NewMeteredCharacterType(gauge)
		case sema.AnyType:
			return cadence.NewMeteredAnyType(gauge)
		case sema.AnyStructType:
			return cadence.NewMeteredAnyStructType(gauge)
		case sema.AnyResourceType:
			return cadence.NewMeteredAnyResourceType(gauge)
		case sema.BlockType:
			return cadence.NewMeteredBlockType(gauge)
		case sema.StringType:
			return cadence.NewMeteredStringType(gauge)
		case sema.AccountKeyType:
			return cadence.NewMeteredAccountKeyType(gauge)
		case sema.PublicAccountContractsType:
			return cadence.NewMeteredPublicAccountContractsType(gauge)
		case sema.AuthAccountContractsType:
			return cadence.NewMeteredAuthAccountContractsType(gauge)
		case sema.PublicAccountKeysType:
			return cadence.NewMeteredPublicAccountKeysType(gauge)
		case sema.AuthAccountKeysType:
			return cadence.NewMeteredAuthAccountKeysType(gauge)
		case sema.PublicAccountType:
			return cadence.NewMeteredPublicAccountType(gauge)
		case sema.AuthAccountType:
			return cadence.NewMeteredAuthAccountType(gauge)
		case sema.DeployedContractType:
			return cadence.NewMeteredDeployedContractType(gauge)
		}

		panic(fmt.Sprintf("cannot export type of type %T", t))
	}()

	results[typeID] = result

	return result
}

// ExportMeteredType converts a runtime type to its corresponding Go representation.
func ExportMeteredType(
	gauge common.MemoryGauge,
	t sema.Type,
	results map[sema.TypeID]cadence.Type,
) cadence.Type {
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
			return exportOptionalType(gauge, t, results)
		case *sema.VariableSizedType:
			return exportVariableSizedType(gauge, t, results)
		case *sema.ConstantSizedType:
			return exportConstantSizedType(gauge, t, results)
		case *sema.CompositeType:
			return exportCompositeType(gauge, t, results)
		case *sema.InterfaceType:
			return exportInterfaceType(gauge, t, results)
		case *sema.DictionaryType:
			return exportDictionaryType(gauge, t, results)
		case *sema.FunctionType:
			return exportFunctionType(gauge, t, results)
		case *sema.AddressType:
			return cadence.NewMeteredAddressType(gauge)
		case *sema.ReferenceType:
			return exportReferenceType(gauge, t, results)
		case *sema.RestrictedType:
			return exportRestrictedType(gauge, t, results)
		case *sema.CapabilityType:
			return exportCapabilityType(gauge, t, results)
		}

		switch t {
		case sema.NumberType:
			return cadence.NewMeteredNumberType(gauge)
		case sema.SignedNumberType:
			return cadence.NewMeteredSignedNumberType(gauge)
		case sema.IntegerType:
			return cadence.NewMeteredIntegerType(gauge)
		case sema.SignedIntegerType:
			return cadence.NewMeteredSignedIntegerType(gauge)
		case sema.FixedPointType:
			return cadence.NewMeteredFixedPointType(gauge)
		case sema.SignedFixedPointType:
			return cadence.NewMeteredSignedFixedPointType(gauge)
		case sema.IntType:
			return cadence.NewMeteredIntType(gauge)
		case sema.Int8Type:
			return cadence.NewMeteredInt8Type(gauge)
		case sema.Int16Type:
			return cadence.NewMeteredInt16Type(gauge)
		case sema.Int32Type:
			return cadence.NewMeteredInt32Type(gauge)
		case sema.Int64Type:
			return cadence.NewMeteredInt64Type(gauge)
		case sema.Int128Type:
			return cadence.NewMeteredInt128Type(gauge)
		case sema.Int256Type:
			return cadence.NewMeteredInt256Type(gauge)
		case sema.UIntType:
			return cadence.NewMeteredUIntType(gauge)
		case sema.UInt8Type:
			return cadence.NewMeteredUInt8Type(gauge)
		case sema.UInt16Type:
			return cadence.NewMeteredUInt16Type(gauge)
		case sema.UInt32Type:
			return cadence.NewMeteredUInt32Type(gauge)
		case sema.UInt64Type:
			return cadence.NewMeteredUInt64Type(gauge)
		case sema.UInt128Type:
			return cadence.NewMeteredUInt128Type(gauge)
		case sema.UInt256Type:
			return cadence.NewMeteredUInt256Type(gauge)
		case sema.Word8Type:
			return cadence.NewMeteredWord8Type(gauge)
		case sema.Word16Type:
			return cadence.NewMeteredWord16Type(gauge)
		case sema.Word32Type:
			return cadence.NewMeteredWord32Type(gauge)
		case sema.Word64Type:
			return cadence.NewMeteredWord64Type(gauge)
		case sema.Fix64Type:
			return cadence.NewMeteredFix64Type(gauge)
		case sema.UFix64Type:
			return cadence.NewMeteredUFix64Type(gauge)
		case sema.PathType:
			return cadence.NewMeteredPathType(gauge)
		case sema.StoragePathType:
			return cadence.NewMeteredStoragePathType(gauge)
		case sema.PrivatePathType:
			return cadence.NewMeteredPrivatePathType(gauge)
		case sema.PublicPathType:
			return cadence.NewMeteredPublicPathType(gauge)
		case sema.CapabilityPathType:
			return cadence.NewMeteredCapabilityPathType(gauge)
		case sema.NeverType:
			return cadence.NewMeteredNeverType(gauge)
		case sema.VoidType:
			return cadence.NewMeteredVoidType(gauge)
		case sema.InvalidType:
			return nil
		case sema.MetaType:
			return cadence.NewMeteredMetaType(gauge)
		case sema.BoolType:
			return cadence.NewMeteredBoolType(gauge)
		case sema.CharacterType:
			return cadence.NewMeteredCharacterType(gauge)
		case sema.AnyType:
			return cadence.NewMeteredAnyType(gauge)
		case sema.AnyStructType:
			return cadence.NewMeteredAnyStructType(gauge)
		case sema.AnyResourceType:
			return cadence.NewMeteredAnyResourceType(gauge)
		case sema.BlockType:
			return cadence.NewMeteredBlockType(gauge)
		case sema.StringType:
			return cadence.NewMeteredStringType(gauge)
		case sema.AccountKeyType:
			return cadence.NewMeteredAccountKeyType(gauge)
		case sema.PublicAccountContractsType:
			return cadence.NewMeteredPublicAccountContractsType(gauge)
		case sema.AuthAccountContractsType:
			return cadence.NewMeteredAuthAccountContractsType(gauge)
		case sema.PublicAccountKeysType:
			return cadence.NewMeteredPublicAccountKeysType(gauge)
		case sema.AuthAccountKeysType:
			return cadence.NewMeteredAuthAccountKeysType(gauge)
		case sema.PublicAccountType:
			return cadence.NewMeteredPublicAccountType(gauge)
		case sema.AuthAccountType:
			return cadence.NewMeteredAuthAccountType(gauge)
		case sema.DeployedContractType:
			return cadence.NewMeteredDeployedContractType(gauge)
		}

		panic(fmt.Sprintf("cannot export type of type %T", t))
	}()

	results[typeID] = result

	return result
}

func exportOptionalType(gauge common.MemoryGauge, t *sema.OptionalType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedType := ExportMeteredType(gauge, t.Type, results)

	return cadence.NewMeteredOptionalType(
		gauge,
		convertedType,
	)
}

func exportVariableSizedType(gauge common.MemoryGauge, t *sema.VariableSizedType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedElement := ExportMeteredType(gauge, t.Type, results)

	return cadence.NewMeteredVariableSizedArrayType(gauge, convertedElement)
}

func exportConstantSizedType(gauge common.MemoryGauge, t *sema.ConstantSizedType, results map[sema.TypeID]cadence.Type) cadence.Type {
	convertedElement := ExportMeteredType(gauge, t.Type, results)

	return cadence.NewMeteredConstantSizedArrayType(
		gauge,
		uint(t.Size),
		convertedElement,
	)
}

func exportCompositeType(
	gauge common.MemoryGauge,
	t *sema.CompositeType,
	results map[sema.TypeID]cadence.Type,
) (result cadence.CompositeType) {

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
		result = cadence.NewMeteredStructType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindResource:
		result = cadence.NewMeteredResourceType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindEvent:
		result = cadence.NewMeteredEventType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindContract:
		result = cadence.NewMeteredContractType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindEnum:
		result = cadence.NewMeteredEnumType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			ExportMeteredType(gauge, t.EnumRawType, results),
			fields,
			nil,
		)

	default:
		panic(fmt.Sprintf("cannot export composite type %v of unknown kind %v", t, t.Kind))
	}

	// NOTE: ensure to set the result before recursively export field types

	results[t.ID()] = result

	for i, member := range fieldMembers {
		convertedFieldType := ExportMeteredType(gauge, member.TypeAnnotation.Type, results)

		fields[i] = cadence.Field{
			Identifier: member.Identifier.Identifier,
			Type:       convertedFieldType,
		}
	}

	return
}

func exportInterfaceType(
	gauge common.MemoryGauge,
	t *sema.InterfaceType,
	results map[sema.TypeID]cadence.Type,
) (result cadence.InterfaceType) {

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
		result = cadence.NewMeteredStructInterfaceType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindResource:
		result = cadence.NewMeteredResourceInterfaceType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	case common.CompositeKindContract:
		result = cadence.NewMeteredContractInterfaceType(
			gauge,
			t.Location,
			t.QualifiedIdentifier(),
			fields,
			nil,
		)

	default:
		panic(fmt.Sprintf("cannot export interface type %v of unknown kind %v", t, t.CompositeKind))
	}

	// NOTE: ensure to set the result before recursively export field types

	results[t.ID()] = result

	for i, member := range fieldMembers {
		convertedFieldType := ExportMeteredType(gauge, member.TypeAnnotation.Type, results)

		fields[i] = cadence.Field{
			Identifier: member.Identifier.Identifier,
			Type:       convertedFieldType,
		}
	}

	return
}

func exportDictionaryType(
	gauge common.MemoryGauge,
	t *sema.DictionaryType,
	results map[sema.TypeID]cadence.Type,
) cadence.Type {
	convertedKeyType := ExportMeteredType(gauge, t.KeyType, results)
	convertedElementType := ExportMeteredType(gauge, t.ValueType, results)

	return cadence.NewMeteredDictionaryType(
		gauge,
		convertedKeyType,
		convertedElementType,
	)
}

func exportFunctionType(
	gauge common.MemoryGauge,
	t *sema.FunctionType,
	results map[sema.TypeID]cadence.Type,
) cadence.Type {
	common.UseMemory(gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: uint64(len(t.Parameters)),
	})
	convertedParameters := make([]cadence.Parameter, len(t.Parameters))

	for i, parameter := range t.Parameters {
		convertedParameterType := ExportMeteredType(gauge, parameter.TypeAnnotation.Type, results)

		// Metered above
		convertedParameters[i] = cadence.NewParameter(
			parameter.Label,
			parameter.Identifier,
			convertedParameterType,
		)
	}

	convertedReturnType := ExportMeteredType(gauge, t.ReturnTypeAnnotation.Type, results)

	return cadence.NewMeteredFunctionType(
		gauge,
		"",
		convertedParameters,
		convertedReturnType,
	).WithID(string(t.ID()))
}

func exportReferenceType(
	gauge common.MemoryGauge,
	t *sema.ReferenceType,
	results map[sema.TypeID]cadence.Type,
) cadence.ReferenceType {
	convertedType := ExportMeteredType(gauge, t.Type, results)

	return cadence.NewMeteredReferenceType(
		gauge,
		t.Authorized,
		convertedType,
	)
}

func exportRestrictedType(
	gauge common.MemoryGauge,
	t *sema.RestrictedType,
	results map[sema.TypeID]cadence.Type,
) cadence.RestrictedType {

	convertedType := ExportMeteredType(gauge, t.Type, results)

	restrictions := make([]cadence.Type, len(t.Restrictions))

	for i, restriction := range t.Restrictions {
		restrictions[i] = ExportMeteredType(gauge, restriction, results)
	}

	return cadence.NewMeteredRestrictedType(
		gauge,
		"",
		convertedType,
		restrictions,
	).WithID(string(t.ID()))
}

func exportCapabilityType(
	gauge common.MemoryGauge,
	t *sema.CapabilityType,
	results map[sema.TypeID]cadence.Type,
) cadence.CapabilityType {

	var borrowType cadence.Type
	if t.BorrowType != nil {
		borrowType = ExportMeteredType(gauge, t.BorrowType, results)
	}

	return cadence.NewMeteredCapabilityType(
		gauge,
		borrowType,
	)
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
