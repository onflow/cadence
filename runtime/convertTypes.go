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
	return ExportMeteredType(nil, t, results)
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
		switch t {
		case sema.NumberType:
			return cadence.NumberType
		case sema.SignedNumberType:
			return cadence.SignedNumberType
		case sema.IntegerType:
			return cadence.IntegerType
		case sema.SignedIntegerType:
			return cadence.SignedIntegerType
		case sema.FixedSizeUnsignedIntegerType:
			return cadence.FixedSizeUnsignedIntegerType
		case sema.FixedPointType:
			return cadence.FixedPointType
		case sema.SignedFixedPointType:
			return cadence.SignedFixedPointType
		case sema.IntType:
			return cadence.IntType
		case sema.Int8Type:
			return cadence.Int8Type
		case sema.Int16Type:
			return cadence.Int16Type
		case sema.Int32Type:
			return cadence.Int32Type
		case sema.Int64Type:
			return cadence.Int64Type
		case sema.Int128Type:
			return cadence.Int128Type
		case sema.Int256Type:
			return cadence.Int256Type
		case sema.UIntType:
			return cadence.UIntType
		case sema.UInt8Type:
			return cadence.UInt8Type
		case sema.UInt16Type:
			return cadence.UInt16Type
		case sema.UInt32Type:
			return cadence.UInt32Type
		case sema.UInt64Type:
			return cadence.UInt64Type
		case sema.UInt128Type:
			return cadence.UInt128Type
		case sema.UInt256Type:
			return cadence.UInt256Type
		case sema.Word8Type:
			return cadence.Word8Type
		case sema.Word16Type:
			return cadence.Word16Type
		case sema.Word32Type:
			return cadence.Word32Type
		case sema.Word64Type:
			return cadence.Word64Type
		case sema.Word128Type:
			return cadence.Word128Type
		case sema.Word256Type:
			return cadence.Word256Type
		case sema.Fix64Type:
			return cadence.Fix64Type
		case sema.UFix64Type:
			return cadence.UFix64Type
		case sema.PathType:
			return cadence.PathType
		case sema.StoragePathType:
			return cadence.StoragePathType
		case sema.PrivatePathType:
			return cadence.PrivatePathType
		case sema.PublicPathType:
			return cadence.PublicPathType
		case sema.CapabilityPathType:
			return cadence.CapabilityPathType
		case sema.NeverType:
			return cadence.NeverType
		case sema.VoidType:
			return cadence.VoidType
		case sema.InvalidType:
			return nil
		case sema.MetaType:
			return cadence.MetaType
		case sema.BoolType:
			return cadence.BoolType
		case sema.CharacterType:
			return cadence.CharacterType
		case sema.AnyType:
			return cadence.AnyType
		case sema.AnyStructType:
			return cadence.AnyStructType
		case sema.HashableStructType:
			return cadence.HashableStructType
		case sema.AnyResourceType:
			return cadence.AnyResourceType
		case sema.AnyStructAttachmentType:
			return cadence.AnyStructAttachmentType
		case sema.AnyResourceAttachmentType:
			return cadence.AnyResourceAttachmentType
		case sema.BlockType:
			return cadence.BlockType
		case sema.StringType:
			return cadence.StringType
		case sema.StorageCapabilityControllerType:
			return cadence.StorageCapabilityControllerType
		case sema.AccountCapabilityControllerType:
			return cadence.AccountCapabilityControllerType
		case sema.Account_StorageType:
			return cadence.Account_StorageType
		case sema.Account_ContractsType:
			return cadence.Account_ContractsType
		case sema.Account_KeysType:
			return cadence.Account_KeysType
		case sema.Account_InboxType:
			return cadence.Account_InboxType
		case sema.Account_CapabilitiesType:
			return cadence.Account_CapabilitiesType
		case sema.Account_StorageCapabilitiesType:
			return cadence.Account_StorageCapabilitiesType
		case sema.Account_AccountCapabilitiesType:
			return cadence.Account_AccountCapabilitiesType
		case sema.AccountType:
			return cadence.AccountType
		case sema.DeployedContractType:
			return cadence.DeployedContractType

		case sema.MutateType:
			return cadence.MutateType
		case sema.InsertType:
			return cadence.InsertType
		case sema.RemoveType:
			return cadence.RemoveType

		case sema.StorageType:
			return cadence.StorageType
		case sema.SaveValueType:
			return cadence.SaveValueType
		case sema.LoadValueType:
			return cadence.LoadValueType
		case sema.CopyValueType:
			return cadence.CopyValueType
		case sema.BorrowValueType:
			return cadence.BorrowValueType
		case sema.ContractsType:
			return cadence.ContractsType
		case sema.AddContractType:
			return cadence.AddContractType
		case sema.UpdateContractType:
			return cadence.UpdateContractType
		case sema.RemoveContractType:
			return cadence.RemoveContractType
		case sema.KeysType:
			return cadence.KeysType
		case sema.AddKeyType:
			return cadence.AddKeyType
		case sema.RevokeKeyType:
			return cadence.RevokeKeyType
		case sema.InboxType:
			return cadence.InboxType
		case sema.PublishInboxCapabilityType:
			return cadence.PublishInboxCapabilityType
		case sema.UnpublishInboxCapabilityType:
			return cadence.UnpublishInboxCapabilityType
		case sema.ClaimInboxCapabilityType:
			return cadence.ClaimInboxCapabilityType
		case sema.CapabilitiesType:
			return cadence.CapabilitiesType
		case sema.StorageCapabilitiesType:
			return cadence.StorageCapabilitiesType
		case sema.AccountCapabilitiesType:
			return cadence.AccountCapabilitiesType
		case sema.PublishCapabilityType:
			return cadence.PublishCapabilityType
		case sema.UnpublishCapabilityType:
			return cadence.UnpublishCapabilityType
		case sema.GetStorageCapabilityControllerType:
			return cadence.GetStorageCapabilityControllerType
		case sema.IssueStorageCapabilityControllerType:
			return cadence.IssueStorageCapabilityControllerType
		case sema.GetAccountCapabilityControllerType:
			return cadence.GetAccountCapabilityControllerType
		case sema.IssueAccountCapabilityControllerType:
			return cadence.IssueAccountCapabilityControllerType

		case sema.CapabilitiesMappingType:
			return cadence.CapabilitiesMappingType
		case sema.AccountMappingType:
			return cadence.AccountMappingType
		case sema.IdentityType:
			return cadence.IdentityType
		}

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
			return cadence.AddressType
		case *sema.ReferenceType:
			return exportReferenceType(gauge, t, results)
		case *sema.IntersectionType:
			return exportIntersectionType(gauge, t, results)
		case *sema.CapabilityType:
			return exportCapabilityType(gauge, t, results)
		case *sema.InclusiveRangeType:
			return exportInclusiveRangeType(gauge, t, results)
		}

		panic(fmt.Sprintf("cannot export type %s", t))
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

	case common.CompositeKindAttachment:
		result = cadence.NewMeteredAttachmentType(
			gauge,
			t.Location,
			ExportMeteredType(gauge, t.GetBaseType(), results),
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

func exportInclusiveRangeType(
	gauge common.MemoryGauge,
	t *sema.InclusiveRangeType,
	results map[sema.TypeID]cadence.Type,
) *cadence.InclusiveRangeType {
	convertedMemberType := ExportMeteredType(gauge, t.MemberType, results)

	return cadence.NewMeteredInclusiveRangeType(
		gauge,
		convertedMemberType,
	)
}

func exportFunctionType(
	gauge common.MemoryGauge,
	t *sema.FunctionType,
	results map[sema.TypeID]cadence.Type,
) cadence.Type {
	// Type parameters
	typeParameterCount := len(t.TypeParameters)
	common.UseMemory(gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceTypeParameter,
		Amount: uint64(typeParameterCount),
	})
	var convertedTypeParameters []cadence.TypeParameter
	if typeParameterCount > 0 {
		convertedTypeParameters = make([]cadence.TypeParameter, typeParameterCount)

		for i, typeParameter := range t.TypeParameters {

			typeBound := typeParameter.TypeBound
			var convertedParameterTypeBound cadence.Type
			if typeBound != nil {
				convertedParameterTypeBound = ExportMeteredType(gauge, typeBound, results)
			}

			// Metered above
			convertedTypeParameters[i] = cadence.NewTypeParameter(
				typeParameter.Name,
				convertedParameterTypeBound,
			)
		}
	}

	// Parameters
	parameterCount := len(t.Parameters)
	common.UseMemory(gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: uint64(parameterCount),
	})
	var convertedParameters []cadence.Parameter
	if parameterCount > 0 {
		convertedParameters = make([]cadence.Parameter, parameterCount)

		for i, parameter := range t.Parameters {
			convertedParameterType := ExportMeteredType(gauge, parameter.TypeAnnotation.Type, results)

			// Metered above
			convertedParameters[i] = cadence.NewParameter(
				parameter.Label,
				parameter.Identifier,
				convertedParameterType,
			)
		}
	}

	convertedReturnType := ExportMeteredType(gauge, t.ReturnTypeAnnotation.Type, results)

	return cadence.NewMeteredFunctionType(
		gauge,
		cadence.FunctionPurity(t.Purity),
		convertedTypeParameters,
		convertedParameters,
		convertedReturnType,
	)
}

func exportAuthorization(
	gauge common.MemoryGauge,
	access sema.Access,
) cadence.Authorization {
	switch access := access.(type) {
	case sema.PrimitiveAccess:
		if access.Equal(sema.UnauthorizedAccess) {
			return cadence.UnauthorizedAccess
		}
	case *sema.EntitlementMapAccess:
		common.UseMemory(gauge, common.NewConstantMemoryUsage(common.MemoryKindCadenceEntitlementMapAccess))
		return cadence.EntitlementMapAuthorization{
			TypeID: access.Type.ID(),
		}
	case sema.EntitlementSetAccess:
		common.UseMemory(gauge, common.MemoryUsage{
			Kind:   common.MemoryKindCadenceEntitlementSetAccess,
			Amount: uint64(access.Entitlements.Len()),
		})
		var entitlements []common.TypeID
		access.Entitlements.Foreach(func(key *sema.EntitlementType, _ struct{}) {
			entitlements = append(entitlements, key.ID())
		})
		return &cadence.EntitlementSetAuthorization{
			Entitlements: entitlements,
			Kind:         access.SetKind,
		}
	}
	panic(fmt.Sprintf("cannot export authorization with access %T", access))
}

func exportReferenceType(
	gauge common.MemoryGauge,
	t *sema.ReferenceType,
	results map[sema.TypeID]cadence.Type,
) *cadence.ReferenceType {
	convertedType := ExportMeteredType(gauge, t.Type, results)

	return cadence.NewMeteredReferenceType(
		gauge,
		exportAuthorization(gauge, t.Authorization),
		convertedType,
	)
}

func exportIntersectionType(
	gauge common.MemoryGauge,
	t *sema.IntersectionType,
	results map[sema.TypeID]cadence.Type,
) *cadence.IntersectionType {

	intersectionTypes := make([]cadence.Type, len(t.Types))

	for i, typ := range t.Types {
		intersectionTypes[i] = ExportMeteredType(gauge, typ, results)
	}

	return cadence.NewMeteredIntersectionType(
		gauge,
		intersectionTypes,
	)
}

func exportCapabilityType(
	gauge common.MemoryGauge,
	t *sema.CapabilityType,
	results map[sema.TypeID]cadence.Type,
) *cadence.CapabilityType {

	var borrowType cadence.Type
	if t.BorrowType != nil {
		borrowType = ExportMeteredType(gauge, t.BorrowType, results)
	}

	return cadence.NewMeteredCapabilityType(
		gauge,
		borrowType,
	)
}

func importInterfaceType(memoryGauge common.MemoryGauge, t cadence.InterfaceType) *interpreter.InterfaceStaticType {
	return interpreter.NewInterfaceStaticTypeComputeTypeID(
		memoryGauge,
		t.InterfaceTypeLocation(),
		t.InterfaceTypeQualifiedIdentifier(),
	)
}

func importCompositeType(memoryGauge common.MemoryGauge, t cadence.CompositeType) interpreter.StaticType {
	location := t.CompositeTypeLocation()
	qualifiedIdentifier := t.CompositeTypeQualifiedIdentifier()

	typeID := common.NewTypeIDFromQualifiedName(
		memoryGauge,
		location,
		qualifiedIdentifier,
	)

	if location == nil {
		primitiveStaticType := interpreter.PrimitiveStaticTypeFromTypeID(typeID)

		if primitiveStaticType != interpreter.PrimitiveStaticTypeUnknown &&
			!primitiveStaticType.IsDeprecated() { //nolint:staticcheck

			return primitiveStaticType
		}
	}

	return interpreter.NewCompositeStaticType(
		memoryGauge,
		location,
		qualifiedIdentifier,
		typeID,
	)
}

func importAuthorization(memoryGauge common.MemoryGauge, auth cadence.Authorization) interpreter.Authorization {
	switch auth := auth.(type) {
	case cadence.Unauthorized:
		return interpreter.UnauthorizedAccess
	case cadence.EntitlementMapAuthorization:
		return interpreter.NewEntitlementMapAuthorization(memoryGauge, auth.TypeID)
	case *cadence.EntitlementSetAuthorization:
		return interpreter.NewEntitlementSetAuthorization(
			memoryGauge,
			func() []common.TypeID { return auth.Entitlements },
			len(auth.Entitlements),
			auth.Kind,
		)
	}
	panic(fmt.Sprintf("cannot import authorization of type %T", auth))
}

func ImportType(memoryGauge common.MemoryGauge, t cadence.Type) interpreter.StaticType {
	switch t := t.(type) {
	case cadence.PrimitiveType:
		return interpreter.NewPrimitiveStaticType(
			memoryGauge,
			interpreter.PrimitiveStaticType(t),
		)

	case *cadence.OptionalType:
		return interpreter.NewOptionalStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.Type),
		)

	case *cadence.VariableSizedArrayType:
		return interpreter.NewVariableSizedStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.ElementType),
		)

	case *cadence.ConstantSizedArrayType:
		return interpreter.NewConstantSizedStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.ElementType),
			int64(t.Size),
		)

	case *cadence.DictionaryType:
		return interpreter.NewDictionaryStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.KeyType),
			ImportType(memoryGauge, t.ElementType),
		)
	case *cadence.InclusiveRangeType:
		return interpreter.NewInclusiveRangeStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.ElementType),
		)
	case *cadence.StructType,
		*cadence.ResourceType,
		*cadence.EventType,
		*cadence.ContractType,
		*cadence.EnumType:
		return importCompositeType(
			memoryGauge,
			t.(cadence.CompositeType),
		)

	case *cadence.StructInterfaceType,
		*cadence.ResourceInterfaceType,
		*cadence.ContractInterfaceType:
		return importInterfaceType(
			memoryGauge,
			t.(cadence.InterfaceType),
		)

	case *cadence.ReferenceType:
		return interpreter.NewReferenceStaticType(
			memoryGauge,
			importAuthorization(memoryGauge, t.Authorization),
			ImportType(memoryGauge, t.Type),
		)

	case *cadence.IntersectionType:
		types := make([]*interpreter.InterfaceStaticType, 0, len(t.Types))
		for _, typ := range t.Types {
			intf, ok := typ.(cadence.InterfaceType)
			if !ok {
				panic(fmt.Sprintf("cannot export type of type %T", t))
			}
			types = append(types, importInterfaceType(memoryGauge, intf))
		}
		return interpreter.NewIntersectionStaticType(
			memoryGauge,
			types,
		)

	case *cadence.CapabilityType:
		if t.BorrowType == nil {
			return interpreter.PrimitiveStaticTypeCapability
		}

		return interpreter.NewCapabilityStaticType(
			memoryGauge,
			ImportType(memoryGauge, t.BorrowType),
		)

	default:
		panic(fmt.Sprintf("cannot import type of type %T", t))
	}
}
