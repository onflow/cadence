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

package sema

import (
	"github.com/onflow/cadence/runtime/common"
)

const PublicAccountTypeName = "PublicAccount"
const PublicAccountAddressField = "address"
const PublicAccountBalanceField = "balance"
const PublicAccountAvailableBalanceField = "availableBalance"
const PublicAccountStorageUsedField = "storageUsed"
const PublicAccountStorageCapacityField = "storageCapacity"
const PublicAccountGetCapabilityField = "getCapability"
const PublicAccountGetTargetLinkField = "getLinkTarget"
const PublicAccountKeysField = "keys"
const PublicAccountContractsField = "contracts"

// PublicAccountType represents the publicly accessible portion of an account.
//
var PublicAccountType = func() *CompositeType {

	publicAccountType := &CompositeType{
		Identifier:         PublicAccountTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: true,
		importable:         false,
		nestedTypes: func() *StringTypeOrderedMap {
			nestedTypes := &StringTypeOrderedMap{}
			nestedTypes.Set(AccountKeysTypeName, PublicAccountKeysType)
			nestedTypes.Set(PublicAccountContractsTypeName, PublicAccountContractsType)
			return nestedTypes
		}(),
	}

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountAddressField,
			&AddressType{},
			accountTypeAddressFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountBalanceField,
			UFix64Type,
			accountTypeAccountBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountAvailableBalanceField,
			UFix64Type,
			accountTypeAccountAvailableBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountStorageUsedField,
			UInt64Type,
			accountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountStorageCapacityField,
			UInt64Type,
			accountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountType,
			PublicAccountGetCapabilityField,
			PublicAccountTypeGetCapabilityFunctionType,
			publicAccountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountType,
			PublicAccountGetTargetLinkField,
			AccountTypeGetLinkTargetFunctionType,
			accountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountKeysField,
			PublicAccountKeysType,
			accountTypeKeysFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountContractsField,
			PublicAccountContractsType,
			accountTypeContractsFieldDocString,
		),
	}

	publicAccountType.Members = GetMembersAsMap(members)
	publicAccountType.Fields = getFieldNames(members)
	return publicAccountType
}()

// PublicAccountKeysType represents the keys associated with a public account.
var PublicAccountKeysType = func() *CompositeType {

	accountKeys := &CompositeType{
		Identifier: AccountKeysTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	var members = []*Member{
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysGetFunctionName,
			AccountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
	}

	accountKeys.Members = GetMembersAsMap(members)
	accountKeys.Fields = getFieldNames(members)
	return accountKeys
}()

func init() {
	// Set the container type after initializing the AccountKeysTypes, to avoid initializing loop.
	PublicAccountKeysType.SetContainerType(PublicAccountType)
}

var PublicAccountTypeGetCapabilityFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
		Name:     "T",
		Optional: true,
	}

	return &FunctionType{
		Purity: ViewFunction,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "capabilityPath",
				TypeAnnotation: NewTypeAnnotation(PublicPathType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&CapabilityType{
				BorrowType: &GenericType{
					TypeParameter: typeParameter,
				},
			},
		),
	}
}()

const publicAccountTypeGetLinkTargetFunctionDocString = `
Returns the capability at the given public path, or nil if it does not exist
`
