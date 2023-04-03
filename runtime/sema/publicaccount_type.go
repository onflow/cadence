/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
const PublicAccountTypeAddressFieldName = "address"
const PublicAccountTypeBalanceFieldName = "balance"
const PublicAccountTypeAvailableBalanceFieldName = "availableBalance"
const PublicAccountTypeStorageUsedFieldName = "storageUsed"
const PublicAccountTypeStorageCapacityFieldName = "storageCapacity"
const PublicAccountTypeGetCapabilityFieldName = "getCapability"
const PublicAccountTypeGetTargetLinkFieldName = "getLinkTarget"
const PublicAccountTypeForEachPublicFieldName = "forEachPublic"
const PublicAccountTypeKeysFieldName = "keys"
const PublicAccountTypeContractsFieldName = "contracts"
const PublicAccountTypePathsFieldName = "publicPaths"

// PublicAccountType represents the publicly accessible portion of an account.
var PublicAccountType = func() *CompositeType {

	publicAccountType := &CompositeType{
		Identifier:         PublicAccountTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: true,
		importable:         false,
	}

	publicAccountType.SetNestedType(AccountKeysTypeName, PublicAccountKeysType)
	publicAccountType.SetNestedType(PublicAccountContractsTypeName, PublicAccountContractsType)

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeAddressFieldName,
			TheAddressType,
			accountTypeAddressFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeBalanceFieldName,
			UFix64Type,
			accountTypeAccountBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeAvailableBalanceFieldName,
			UFix64Type,
			accountTypeAccountAvailableBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeStorageUsedFieldName,
			UInt64Type,
			accountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeStorageCapacityFieldName,
			UInt64Type,
			accountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountType,
			PublicAccountTypeGetCapabilityFieldName,
			PublicAccountTypeGetCapabilityFunctionType,
			publicAccountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountType,
			PublicAccountTypeGetTargetLinkFieldName,
			AccountTypeGetLinkTargetFunctionType,
			accountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeKeysFieldName,
			PublicAccountKeysType,
			accountTypeKeysFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypeContractsFieldName,
			PublicAccountContractsType,
			accountTypeContractsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountType,
			PublicAccountTypePathsFieldName,
			PublicAccountPathsType,
			publicAccountTypePathsFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			publicAccountType,
			PublicAccountTypeForEachPublicFieldName,
			PublicAccountForEachPublicFunctionType,
			publicAccountForEachPublicDocString,
		),
	}

	publicAccountType.Members = GetMembersAsMap(members)
	publicAccountType.Fields = GetFieldNames(members)
	return publicAccountType
}()

var PublicAccountPathsType = &VariableSizedType{
	Type: PublicPathType,
}

const publicAccountTypePathsFieldDocString = `
All the public paths of an account
`

func AccountForEachFunctionType(pathType Type) *FunctionType {
	iterFunctionType := &FunctionType{
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "path",
				TypeAnnotation: NewTypeAnnotation(pathType),
			},
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "type",
				TypeAnnotation: NewTypeAnnotation(MetaType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
	}
	return &FunctionType{
		Parameters: []Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "function",
				TypeAnnotation: NewTypeAnnotation(iterFunctionType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}

const publicAccountForEachPublicDocString = `
Iterate over all the public paths in an account.

Takes two arguments: the first is the path (/domain/key) of the stored object, and the second is the runtime type of that object

The returned boolean of the supplied function indicates whether the iteration should continue; true will continue iterating onto the next element in storage, 
false will abort iteration.
`

var PublicAccountForEachPublicFunctionType = AccountForEachFunctionType(PublicPathType)

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
			AccountKeysTypeGetFunctionName,
			AccountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysTypeForEachFunctionName,
			AccountKeysTypeForEachFunctionType,
			accountKeysTypeForEachFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			accountKeys,
			AccountKeysTypeCountFieldName,
			AccountKeysTypeCountFieldType,
			accountKeysTypeCountFieldDocString,
		),
	}

	accountKeys.Members = GetMembersAsMap(members)
	accountKeys.Fields = GetFieldNames(members)
	return accountKeys
}()

var PublicAccountTypeGetCapabilityFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
		Name:     "T",
		Optional: true,
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
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
