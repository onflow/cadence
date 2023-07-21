// Code generated from publicaccount.cdc. DO NOT EDIT.
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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const PublicAccountTypeAddressFieldName = "address"

var PublicAccountTypeAddressFieldType = TheAddressType

const PublicAccountTypeAddressFieldDocString = `
The address of the account.
`

const PublicAccountTypeBalanceFieldName = "balance"

var PublicAccountTypeBalanceFieldType = UFix64Type

const PublicAccountTypeBalanceFieldDocString = `
The FLOW balance of the default vault of this account.
`

const PublicAccountTypeAvailableBalanceFieldName = "availableBalance"

var PublicAccountTypeAvailableBalanceFieldType = UFix64Type

const PublicAccountTypeAvailableBalanceFieldDocString = `
The FLOW balance of the default vault of this account that is available to be moved.
`

const PublicAccountTypeStorageUsedFieldName = "storageUsed"

var PublicAccountTypeStorageUsedFieldType = UInt64Type

const PublicAccountTypeStorageUsedFieldDocString = `
The current amount of storage used by the account in bytes.
`

const PublicAccountTypeStorageCapacityFieldName = "storageCapacity"

var PublicAccountTypeStorageCapacityFieldType = UInt64Type

const PublicAccountTypeStorageCapacityFieldDocString = `
The storage capacity of the account in bytes.
`

const PublicAccountTypeContractsFieldName = "contracts"

var PublicAccountTypeContractsFieldType = PublicAccountContractsType

const PublicAccountTypeContractsFieldDocString = `
The contracts deployed to the account.
`

const PublicAccountTypeKeysFieldName = "keys"

var PublicAccountTypeKeysFieldType = PublicAccountKeysType

const PublicAccountTypeKeysFieldDocString = `
The keys assigned to the account.
`

const PublicAccountTypeCapabilitiesFieldName = "capabilities"

var PublicAccountTypeCapabilitiesFieldType = PublicAccountCapabilitiesType

const PublicAccountTypeCapabilitiesFieldDocString = `
The capabilities of the account.
`

const PublicAccountTypePublicPathsFieldName = "publicPaths"

var PublicAccountTypePublicPathsFieldType = &VariableSizedType{
	Type: PublicPathType,
}

const PublicAccountTypePublicPathsFieldDocString = `
All public paths of this account.
`

const PublicAccountTypeGetCapabilityFunctionName = "getCapability"

var PublicAccountTypeGetCapabilityFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var PublicAccountTypeGetCapabilityFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		PublicAccountTypeGetCapabilityFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		MustInstantiate(
			&CapabilityType{},
			&GenericType{
				TypeParameter: PublicAccountTypeGetCapabilityFunctionTypeParameterT,
			},
		),
	),
}

const PublicAccountTypeGetCapabilityFunctionDocString = `
Returns the capability at the given public path.
`

const PublicAccountTypeGetLinkTargetFunctionName = "getLinkTarget"

var PublicAccountTypeGetLinkTargetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: PathType,
		},
	),
}

const PublicAccountTypeGetLinkTargetFunctionDocString = `
Returns the target path of the capability at the given public or private path,
or nil if there exists no capability at the given path.
`

const PublicAccountTypeForEachPublicFunctionName = "forEachPublic"

var PublicAccountTypeForEachPublicFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(PublicPathType),
					},
					{
						TypeAnnotation: NewTypeAnnotation(MetaType),
					},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(
					BoolType,
				),
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const PublicAccountTypeForEachPublicFunctionDocString = `
Iterate over all the public paths of an account.
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration, as well as the behavior of adding or removing objects from storage during iteration,
is undefined.
`

const PublicAccountContractsTypeNamesFieldName = "names"

var PublicAccountContractsTypeNamesFieldType = &VariableSizedType{
	Type: StringType,
}

const PublicAccountContractsTypeNamesFieldDocString = `
The names of all contracts deployed in the account.
`

const PublicAccountContractsTypeGetFunctionName = "get"

var PublicAccountContractsTypeGetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: DeployedContractType,
		},
	),
}

const PublicAccountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const PublicAccountContractsTypeBorrowFunctionName = "borrow"

var PublicAccountContractsTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var PublicAccountContractsTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		PublicAccountContractsTypeBorrowFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: PublicAccountContractsTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const PublicAccountContractsTypeBorrowFunctionDocString = `
Returns a reference of the given type to the contract with the given name in the account, if any.

Returns nil if no contract with the given name exists in the account,
or if the contract does not conform to the given type.
`

const PublicAccountContractsTypeName = "Contracts"

var PublicAccountContractsType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         PublicAccountContractsTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			PublicAccountContractsType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountContractsTypeNamesFieldName,
			PublicAccountContractsTypeNamesFieldType,
			PublicAccountContractsTypeNamesFieldDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountContractsType,
			ast.AccessPublic,
			PublicAccountContractsTypeGetFunctionName,
			PublicAccountContractsTypeGetFunctionType,
			PublicAccountContractsTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountContractsType,
			ast.AccessPublic,
			PublicAccountContractsTypeBorrowFunctionName,
			PublicAccountContractsTypeBorrowFunctionType,
			PublicAccountContractsTypeBorrowFunctionDocString,
		),
	}

	PublicAccountContractsType.Members = MembersAsMap(members)
	PublicAccountContractsType.Fields = MembersFieldNames(members)
}

const PublicAccountKeysTypeGetFunctionName = "get"

var PublicAccountKeysTypeGetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "keyIndex",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: AccountKeyType,
		},
	),
}

const PublicAccountKeysTypeGetFunctionDocString = `
Returns the key at the given index, if it exists, or nil otherwise.

Revoked keys are always returned, but they have ` + "`isRevoked`" + ` field set to true.
`

const PublicAccountKeysTypeForEachFunctionName = "forEach"

var PublicAccountKeysTypeForEachFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(AccountKeyType),
					},
				},
				ReturnTypeAnnotation: NewTypeAnnotation(
					BoolType,
				),
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const PublicAccountKeysTypeForEachFunctionDocString = `
Iterate over all unrevoked keys in this account,
passing each key in turn to the provided function.

Iteration is stopped early if the function returns ` + "`false`" + `.
The order of iteration is undefined.
`

const PublicAccountKeysTypeCountFieldName = "count"

var PublicAccountKeysTypeCountFieldType = UInt64Type

const PublicAccountKeysTypeCountFieldDocString = `
The total number of unrevoked keys in this account.
`

const PublicAccountKeysTypeName = "Keys"

var PublicAccountKeysType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         PublicAccountKeysTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			PublicAccountKeysType,
			ast.AccessPublic,
			PublicAccountKeysTypeGetFunctionName,
			PublicAccountKeysTypeGetFunctionType,
			PublicAccountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountKeysType,
			ast.AccessPublic,
			PublicAccountKeysTypeForEachFunctionName,
			PublicAccountKeysTypeForEachFunctionType,
			PublicAccountKeysTypeForEachFunctionDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountKeysType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountKeysTypeCountFieldName,
			PublicAccountKeysTypeCountFieldType,
			PublicAccountKeysTypeCountFieldDocString,
		),
	}

	PublicAccountKeysType.Members = MembersAsMap(members)
	PublicAccountKeysType.Fields = MembersFieldNames(members)
}

const PublicAccountCapabilitiesTypeGetFunctionName = "get"

var PublicAccountCapabilitiesTypeGetFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var PublicAccountCapabilitiesTypeGetFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		PublicAccountCapabilitiesTypeGetFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MustInstantiate(
				&CapabilityType{},
				&GenericType{
					TypeParameter: PublicAccountCapabilitiesTypeGetFunctionTypeParameterT,
				},
			),
		},
	),
}

const PublicAccountCapabilitiesTypeGetFunctionDocString = `
get returns the storage capability at the given path, if one was stored there.
`

const PublicAccountCapabilitiesTypeBorrowFunctionName = "borrow"

var PublicAccountCapabilitiesTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var PublicAccountCapabilitiesTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		PublicAccountCapabilitiesTypeBorrowFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: PublicAccountCapabilitiesTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const PublicAccountCapabilitiesTypeBorrowFunctionDocString = `
borrow gets the storage capability at the given path, and borrows the capability if it exists.

Returns nil if the capability does not exist or cannot be borrowed using the given type.
The function is equivalent to ` + "`get(path)?.borrow()`" + `.
`

const PublicAccountCapabilitiesTypeName = "Capabilities"

var PublicAccountCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         PublicAccountCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			PublicAccountCapabilitiesType,
			ast.AccessPublic,
			PublicAccountCapabilitiesTypeGetFunctionName,
			PublicAccountCapabilitiesTypeGetFunctionType,
			PublicAccountCapabilitiesTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountCapabilitiesType,
			ast.AccessPublic,
			PublicAccountCapabilitiesTypeBorrowFunctionName,
			PublicAccountCapabilitiesTypeBorrowFunctionType,
			PublicAccountCapabilitiesTypeBorrowFunctionDocString,
		),
	}

	PublicAccountCapabilitiesType.Members = MembersAsMap(members)
	PublicAccountCapabilitiesType.Fields = MembersFieldNames(members)
}

const PublicAccountTypeName = "PublicAccount"

var PublicAccountType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         PublicAccountTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	t.SetNestedType(PublicAccountContractsTypeName, PublicAccountContractsType)
	t.SetNestedType(PublicAccountKeysTypeName, PublicAccountKeysType)
	t.SetNestedType(PublicAccountCapabilitiesTypeName, PublicAccountCapabilitiesType)
	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeAddressFieldName,
			PublicAccountTypeAddressFieldType,
			PublicAccountTypeAddressFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeBalanceFieldName,
			PublicAccountTypeBalanceFieldType,
			PublicAccountTypeBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeAvailableBalanceFieldName,
			PublicAccountTypeAvailableBalanceFieldType,
			PublicAccountTypeAvailableBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeStorageUsedFieldName,
			PublicAccountTypeStorageUsedFieldType,
			PublicAccountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeStorageCapacityFieldName,
			PublicAccountTypeStorageCapacityFieldType,
			PublicAccountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeContractsFieldName,
			PublicAccountTypeContractsFieldType,
			PublicAccountTypeContractsFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeKeysFieldName,
			PublicAccountTypeKeysFieldType,
			PublicAccountTypeKeysFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypeCapabilitiesFieldName,
			PublicAccountTypeCapabilitiesFieldType,
			PublicAccountTypeCapabilitiesFieldDocString,
		),
		NewUnmeteredFieldMember(
			PublicAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			PublicAccountTypePublicPathsFieldName,
			PublicAccountTypePublicPathsFieldType,
			PublicAccountTypePublicPathsFieldDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountType,
			ast.AccessPublic,
			PublicAccountTypeGetCapabilityFunctionName,
			PublicAccountTypeGetCapabilityFunctionType,
			PublicAccountTypeGetCapabilityFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountType,
			ast.AccessPublic,
			PublicAccountTypeGetLinkTargetFunctionName,
			PublicAccountTypeGetLinkTargetFunctionType,
			PublicAccountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			PublicAccountType,
			ast.AccessPublic,
			PublicAccountTypeForEachPublicFunctionName,
			PublicAccountTypeForEachPublicFunctionType,
			PublicAccountTypeForEachPublicFunctionDocString,
		),
	}

	PublicAccountType.Members = MembersAsMap(members)
	PublicAccountType.Fields = MembersFieldNames(members)
}
