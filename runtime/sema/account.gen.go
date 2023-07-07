// Code generated from account.cdc. DO NOT EDIT.
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

const AccountTypeAddressFieldName = "address"

var AccountTypeAddressFieldType = TheAddressType

const AccountTypeAddressFieldDocString = `
The address of the account.
`

const AccountTypeBalanceFieldName = "balance"

var AccountTypeBalanceFieldType = UFix64Type

const AccountTypeBalanceFieldDocString = `
The FLOW balance of the default vault of this account.
`

const AccountTypeAvailableBalanceFieldName = "availableBalance"

var AccountTypeAvailableBalanceFieldType = UFix64Type

const AccountTypeAvailableBalanceFieldDocString = `
The FLOW balance of the default vault of this account that is available to be moved.
`

const AccountTypeStorageFieldName = "storage"

var AccountTypeStorageFieldType = Account_StorageType

const AccountTypeStorageFieldDocString = `
The storage of the account.
`

const AccountTypeContractsFieldName = "contracts"

var AccountTypeContractsFieldType = Account_ContractsType

const AccountTypeContractsFieldDocString = `
The contracts deployed to the account.
`

const AccountTypeKeysFieldName = "keys"

var AccountTypeKeysFieldType = Account_KeysType

const AccountTypeKeysFieldDocString = `
The keys assigned to the account.
`

const AccountTypeInboxFieldName = "inbox"

var AccountTypeInboxFieldType = Account_InboxType

const AccountTypeInboxFieldDocString = `
The inbox allows bootstrapping (sending and receiving) capabilities.
`

const AccountTypeCapabilitiesFieldName = "capabilities"

var AccountTypeCapabilitiesFieldType = Account_CapabilitiesType

const AccountTypeCapabilitiesFieldDocString = `
The capabilities of the account.
`

const Account_StorageTypeUsedFieldName = "used"

var Account_StorageTypeUsedFieldType = UInt64Type

const Account_StorageTypeUsedFieldDocString = `
The current amount of storage used by the account in bytes.
`

const Account_StorageTypeCapacityFieldName = "capacity"

var Account_StorageTypeCapacityFieldType = UInt64Type

const Account_StorageTypeCapacityFieldDocString = `
The storage capacity of the account in bytes.
`

const Account_StorageTypePublicPathsFieldName = "publicPaths"

var Account_StorageTypePublicPathsFieldType = &VariableSizedType{
	Type: PublicPathType,
}

const Account_StorageTypePublicPathsFieldDocString = `
All public paths of this account.
`

const Account_StorageTypeStoragePathsFieldName = "storagePaths"

var Account_StorageTypeStoragePathsFieldType = &VariableSizedType{
	Type: StoragePathType,
}

const Account_StorageTypeStoragePathsFieldDocString = `
All storage paths of this account.
`

const Account_StorageTypeSaveFunctionName = "save"

var Account_StorageTypeSaveFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: StorableType,
}

var Account_StorageTypeSaveFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_StorageTypeSaveFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: NewTypeAnnotation(&GenericType{
				TypeParameter: Account_StorageTypeSaveFunctionTypeParameterT,
			}),
		},
		{
			Identifier:     "to",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const Account_StorageTypeSaveFunctionDocString = `
Saves the given object into the account's storage at the given path.

Resources are moved into storage, and structures are copied.

If there is already an object stored under the given path, the program aborts.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const Account_StorageTypeTypeFunctionName = "type"

var Account_StorageTypeTypeFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          "at",
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MetaType,
		},
	),
}

const Account_StorageTypeTypeFunctionDocString = `
Reads the type of an object from the account's storage which is stored under the given path,
or nil if no object is stored under the given path.

If there is an object stored, the type of the object is returned without modifying the stored object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const Account_StorageTypeLoadFunctionName = "load"

var Account_StorageTypeLoadFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: StorableType,
}

var Account_StorageTypeLoadFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_StorageTypeLoadFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Identifier:     "from",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: Account_StorageTypeLoadFunctionTypeParameterT,
			},
		},
	),
}

const Account_StorageTypeLoadFunctionDocString = `
Loads an object from the account's storage which is stored under the given path,
or nil if no object is stored under the given path.

If there is an object stored,
the stored resource or structure is moved out of storage and returned as an optional.

When the function returns, the storage no longer contains an object under the given path.

The given type must be a supertype of the type of the loaded object.
If it is not, the function panics.

The given type must not necessarily be exactly the same as the type of the loaded object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const Account_StorageTypeCopyFunctionName = "copy"

var Account_StorageTypeCopyFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: AnyStructType,
}

var Account_StorageTypeCopyFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_StorageTypeCopyFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Identifier:     "from",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: Account_StorageTypeCopyFunctionTypeParameterT,
			},
		},
	),
}

const Account_StorageTypeCopyFunctionDocString = `
Returns a copy of a structure stored in account storage under the given path,
without removing it from storage,
or nil if no object is stored under the given path.

If there is a structure stored, it is copied.
The structure stays stored in storage after the function returns.

The given type must be a supertype of the type of the copied structure.
If it is not, the function panics.

The given type must not necessarily be exactly the same as the type of the copied structure.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const Account_StorageTypeBorrowFunctionName = "borrow"

var Account_StorageTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_StorageTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_StorageTypeBorrowFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Identifier:     "from",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &GenericType{
				TypeParameter: Account_StorageTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const Account_StorageTypeBorrowFunctionDocString = `
Returns a reference to an object in storage without removing it from storage.

If no object is stored under the given path, the function returns nil.
If there is an object stored, a reference is returned as an optional,
provided it can be borrowed using the given type.
If the stored object cannot be borrowed using the given type, the function panics.

The given type must not necessarily be exactly the same as the type of the borrowed object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

const Account_StorageTypeForEachPublicFunctionName = "forEachPublic"

var Account_StorageTypeForEachPublicFunctionType = &FunctionType{
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

const Account_StorageTypeForEachPublicFunctionDocString = `
Iterate over all the public paths of an account,
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration is undefined.

If an object is stored under a new public path,
or an existing object is removed from a public path,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const Account_StorageTypeForEachStoredFunctionName = "forEachStored"

var Account_StorageTypeForEachStoredFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(StoragePathType),
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

const Account_StorageTypeForEachStoredFunctionDocString = `
Iterate over all the stored paths of an account,
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

If an object is stored under a new storage path,
or an existing object is removed from a storage path,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const Account_StorageTypeName = "Storage"

var Account_StorageType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_StorageTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_StorageTypeUsedFieldName,
			Account_StorageTypeUsedFieldType,
			Account_StorageTypeUsedFieldDocString,
		),
		NewUnmeteredFieldMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_StorageTypeCapacityFieldName,
			Account_StorageTypeCapacityFieldType,
			Account_StorageTypeCapacityFieldDocString,
		),
		NewUnmeteredFieldMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_StorageTypePublicPathsFieldName,
			Account_StorageTypePublicPathsFieldType,
			Account_StorageTypePublicPathsFieldDocString,
		),
		NewUnmeteredFieldMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_StorageTypeStoragePathsFieldName,
			Account_StorageTypeStoragePathsFieldType,
			Account_StorageTypeStoragePathsFieldDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			newEntitlementAccess(
				[]Type{SaveValueType},
				Conjunction,
			),
			Account_StorageTypeSaveFunctionName,
			Account_StorageTypeSaveFunctionType,
			Account_StorageTypeSaveFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			Account_StorageTypeTypeFunctionName,
			Account_StorageTypeTypeFunctionType,
			Account_StorageTypeTypeFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			newEntitlementAccess(
				[]Type{LoadValueType},
				Conjunction,
			),
			Account_StorageTypeLoadFunctionName,
			Account_StorageTypeLoadFunctionType,
			Account_StorageTypeLoadFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			Account_StorageTypeCopyFunctionName,
			Account_StorageTypeCopyFunctionType,
			Account_StorageTypeCopyFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			newEntitlementAccess(
				[]Type{BorrowValueType},
				Conjunction,
			),
			Account_StorageTypeBorrowFunctionName,
			Account_StorageTypeBorrowFunctionType,
			Account_StorageTypeBorrowFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			Account_StorageTypeForEachPublicFunctionName,
			Account_StorageTypeForEachPublicFunctionType,
			Account_StorageTypeForEachPublicFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageType,
			PrimitiveAccess(ast.AccessAll),
			Account_StorageTypeForEachStoredFunctionName,
			Account_StorageTypeForEachStoredFunctionType,
			Account_StorageTypeForEachStoredFunctionDocString,
		),
	}

	Account_StorageType.Members = MembersAsMap(members)
	Account_StorageType.Fields = MembersFieldNames(members)
}

const Account_ContractsTypeNamesFieldName = "names"

var Account_ContractsTypeNamesFieldType = &VariableSizedType{
	Type: StringType,
}

const Account_ContractsTypeNamesFieldDocString = `
The names of all contracts deployed in the account.
`

const Account_ContractsTypeAddFunctionName = "add"

var Account_ContractsTypeAddFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
		{
			Identifier: "code",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		DeployedContractType,
	),
}

const Account_ContractsTypeAddFunctionDocString = `
Adds the given contract to the account.

The ` + "`code`" + ` parameter is the UTF-8 encoded representation of the source code.
The code must contain exactly one contract or contract interface,
which must have the same name as the ` + "`name`" + ` parameter.

All additional arguments that are given are passed further to the initializer
of the contract that is being deployed.

The function fails if a contract/contract interface with the given name already exists in the account,
if the given code does not declare exactly one contract or contract interface,
or if the given name does not match the name of the contract/contract interface declaration in the code.

Returns the deployed contract.
`

const Account_ContractsTypeUpdateFunctionName = "update"

var Account_ContractsTypeUpdateFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
		{
			Identifier: "code",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		DeployedContractType,
	),
}

const Account_ContractsTypeUpdateFunctionDocString = `
Updates the code for the contract/contract interface in the account.

The ` + "`code`" + ` parameter is the UTF-8 encoded representation of the source code.
The code must contain exactly one contract or contract interface,
which must have the same name as the ` + "`name`" + ` parameter.

Does **not** run the initializer of the contract/contract interface again.
The contract instance in the world state stays as is.

Fails if no contract/contract interface with the given name exists in the account,
if the given code does not declare exactly one contract or contract interface,
or if the given name does not match the name of the contract/contract interface declaration in the code.

Returns the deployed contract for the updated contract.
`

const Account_ContractsTypeGetFunctionName = "get"

var Account_ContractsTypeGetFunctionType = &FunctionType{
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

const Account_ContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const Account_ContractsTypeRemoveFunctionName = "remove"

var Account_ContractsTypeRemoveFunctionType = &FunctionType{
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

const Account_ContractsTypeRemoveFunctionDocString = `
Removes the contract/contract interface from the account which has the given name, if any.

Returns the removed deployed contract, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const Account_ContractsTypeBorrowFunctionName = "borrow"

var Account_ContractsTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_ContractsTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_ContractsTypeBorrowFunctionTypeParameterT,
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
				TypeParameter: Account_ContractsTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const Account_ContractsTypeBorrowFunctionDocString = `
Returns a reference of the given type to the contract with the given name in the account, if any.

Returns nil if no contract with the given name exists in the account,
or if the contract does not conform to the given type.
`

const Account_ContractsTypeName = "Contracts"

var Account_ContractsType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_ContractsTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			Account_ContractsType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_ContractsTypeNamesFieldName,
			Account_ContractsTypeNamesFieldType,
			Account_ContractsTypeNamesFieldDocString,
		),
		NewUnmeteredFunctionMember(
			Account_ContractsType,
			newEntitlementAccess(
				[]Type{AddContractType},
				Conjunction,
			),
			Account_ContractsTypeAddFunctionName,
			Account_ContractsTypeAddFunctionType,
			Account_ContractsTypeAddFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_ContractsType,
			newEntitlementAccess(
				[]Type{UpdateContractType},
				Conjunction,
			),
			Account_ContractsTypeUpdateFunctionName,
			Account_ContractsTypeUpdateFunctionType,
			Account_ContractsTypeUpdateFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_ContractsType,
			PrimitiveAccess(ast.AccessAll),
			Account_ContractsTypeGetFunctionName,
			Account_ContractsTypeGetFunctionType,
			Account_ContractsTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_ContractsType,
			newEntitlementAccess(
				[]Type{RemoveContractType},
				Conjunction,
			),
			Account_ContractsTypeRemoveFunctionName,
			Account_ContractsTypeRemoveFunctionType,
			Account_ContractsTypeRemoveFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_ContractsType,
			PrimitiveAccess(ast.AccessAll),
			Account_ContractsTypeBorrowFunctionName,
			Account_ContractsTypeBorrowFunctionType,
			Account_ContractsTypeBorrowFunctionDocString,
		),
	}

	Account_ContractsType.Members = MembersAsMap(members)
	Account_ContractsType.Fields = MembersFieldNames(members)
}

const Account_KeysTypeAddFunctionName = "add"

var Account_KeysTypeAddFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "publicKey",
			TypeAnnotation: NewTypeAnnotation(PublicKeyType),
		},
		{
			Identifier:     "hashAlgorithm",
			TypeAnnotation: NewTypeAnnotation(HashAlgorithmType),
		},
		{
			Identifier:     "weight",
			TypeAnnotation: NewTypeAnnotation(UFix64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		AccountKeyType,
	),
}

const Account_KeysTypeAddFunctionDocString = `
Adds a new key with the given hashing algorithm and a weight.

Returns the added key.
`

const Account_KeysTypeGetFunctionName = "get"

var Account_KeysTypeGetFunctionType = &FunctionType{
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

const Account_KeysTypeGetFunctionDocString = `
Returns the key at the given index, if it exists, or nil otherwise.

Revoked keys are always returned, but they have ` + "`isRevoked`" + ` field set to true.
`

const Account_KeysTypeRevokeFunctionName = "revoke"

var Account_KeysTypeRevokeFunctionType = &FunctionType{
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

const Account_KeysTypeRevokeFunctionDocString = `
Marks the key at the given index revoked, but does not delete it.

Returns the revoked key if it exists, or nil otherwise.
`

const Account_KeysTypeForEachFunctionName = "forEach"

var Account_KeysTypeForEachFunctionType = &FunctionType{
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

const Account_KeysTypeForEachFunctionDocString = `
Iterate over all unrevoked keys in this account,
passing each key in turn to the provided function.

Iteration is stopped early if the function returns ` + "`false`" + `.

The order of iteration is undefined.
`

const Account_KeysTypeCountFieldName = "count"

var Account_KeysTypeCountFieldType = UInt64Type

const Account_KeysTypeCountFieldDocString = `
The total number of unrevoked keys in this account.
`

const Account_KeysTypeName = "Keys"

var Account_KeysType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_KeysTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			Account_KeysType,
			newEntitlementAccess(
				[]Type{AddKeyType},
				Conjunction,
			),
			Account_KeysTypeAddFunctionName,
			Account_KeysTypeAddFunctionType,
			Account_KeysTypeAddFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_KeysType,
			PrimitiveAccess(ast.AccessAll),
			Account_KeysTypeGetFunctionName,
			Account_KeysTypeGetFunctionType,
			Account_KeysTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_KeysType,
			newEntitlementAccess(
				[]Type{RevokeKeyType},
				Conjunction,
			),
			Account_KeysTypeRevokeFunctionName,
			Account_KeysTypeRevokeFunctionType,
			Account_KeysTypeRevokeFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_KeysType,
			PrimitiveAccess(ast.AccessAll),
			Account_KeysTypeForEachFunctionName,
			Account_KeysTypeForEachFunctionType,
			Account_KeysTypeForEachFunctionDocString,
		),
		NewUnmeteredFieldMember(
			Account_KeysType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			Account_KeysTypeCountFieldName,
			Account_KeysTypeCountFieldType,
			Account_KeysTypeCountFieldDocString,
		),
	}

	Account_KeysType.Members = MembersAsMap(members)
	Account_KeysType.Fields = MembersFieldNames(members)
}

const Account_InboxTypePublishFunctionName = "publish"

var Account_InboxTypePublishFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: NewTypeAnnotation(&CapabilityType{}),
		},
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
		{
			Identifier:     "recipient",
			TypeAnnotation: NewTypeAnnotation(TheAddressType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const Account_InboxTypePublishFunctionDocString = `
Publishes a new Capability under the given name,
to be claimed by the specified recipient.
`

const Account_InboxTypeUnpublishFunctionName = "unpublish"

var Account_InboxTypeUnpublishFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_InboxTypeUnpublishFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_InboxTypeUnpublishFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MustInstantiate(
				&CapabilityType{},
				&GenericType{
					TypeParameter: Account_InboxTypeUnpublishFunctionTypeParameterT,
				},
			),
		},
	),
}

const Account_InboxTypeUnpublishFunctionDocString = `
Unpublishes a Capability previously published by this account.

Returns ` + "`nil`" + ` if no Capability is published under the given name.

Errors if the Capability under that name does not match the provided type.
`

const Account_InboxTypeClaimFunctionName = "claim"

var Account_InboxTypeClaimFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_InboxTypeClaimFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_InboxTypeClaimFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
		{
			Identifier:     "provider",
			TypeAnnotation: NewTypeAnnotation(TheAddressType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MustInstantiate(
				&CapabilityType{},
				&GenericType{
					TypeParameter: Account_InboxTypeClaimFunctionTypeParameterT,
				},
			),
		},
	),
}

const Account_InboxTypeClaimFunctionDocString = `
Claims a Capability previously published by the specified provider.

Returns ` + "`nil`" + ` if no Capability is published under the given name,
or if this account is not its intended recipient.

Errors if the Capability under that name does not match the provided type.
`

const Account_InboxTypeName = "Inbox"

var Account_InboxType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_InboxTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			Account_InboxType,
			newEntitlementAccess(
				[]Type{PublishInboxCapabilityType},
				Conjunction,
			),
			Account_InboxTypePublishFunctionName,
			Account_InboxTypePublishFunctionType,
			Account_InboxTypePublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_InboxType,
			newEntitlementAccess(
				[]Type{UnpublishInboxCapabilityType},
				Conjunction,
			),
			Account_InboxTypeUnpublishFunctionName,
			Account_InboxTypeUnpublishFunctionType,
			Account_InboxTypeUnpublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_InboxType,
			newEntitlementAccess(
				[]Type{ClaimInboxCapabilityType},
				Conjunction,
			),
			Account_InboxTypeClaimFunctionName,
			Account_InboxTypeClaimFunctionType,
			Account_InboxTypeClaimFunctionDocString,
		),
	}

	Account_InboxType.Members = MembersAsMap(members)
	Account_InboxType.Fields = MembersFieldNames(members)
}

const Account_CapabilitiesTypeStorageFieldName = "storage"

var Account_CapabilitiesTypeStorageFieldType = Account_StorageCapabilitiesType

const Account_CapabilitiesTypeStorageFieldDocString = `
The storage capabilities of the account.
`

const Account_CapabilitiesTypeAccountFieldName = "account"

var Account_CapabilitiesTypeAccountFieldType = Account_AccountCapabilitiesType

const Account_CapabilitiesTypeAccountFieldDocString = `
The account capabilities of the account.
`

const Account_CapabilitiesTypeGetFunctionName = "get"

var Account_CapabilitiesTypeGetFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_CapabilitiesTypeGetFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_CapabilitiesTypeGetFunctionTypeParameterT,
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
					TypeParameter: Account_CapabilitiesTypeGetFunctionTypeParameterT,
				},
			),
		},
	),
}

const Account_CapabilitiesTypeGetFunctionDocString = `
Returns the capability at the given public path.
Returns nil if the capability does not exist,
or if the given type is not a supertype of the capability's borrow type.
`

const Account_CapabilitiesTypeBorrowFunctionName = "borrow"

var Account_CapabilitiesTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_CapabilitiesTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_CapabilitiesTypeBorrowFunctionTypeParameterT,
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
				TypeParameter: Account_CapabilitiesTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const Account_CapabilitiesTypeBorrowFunctionDocString = `
Borrows the capability at the given public path.
Returns nil if the capability does not exist, or cannot be borrowed using the given type.
The function is equivalent to ` + "`get(path)?.borrow()`" + `.
`

const Account_CapabilitiesTypePublishFunctionName = "publish"

var Account_CapabilitiesTypePublishFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "capability",
			TypeAnnotation: NewTypeAnnotation(&CapabilityType{}),
		},
		{
			Identifier:     "at",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const Account_CapabilitiesTypePublishFunctionDocString = `
Publish the capability at the given public path.

If there is already a capability published under the given path, the program aborts.

The path must be a public path, i.e., only the domain ` + "`public`" + ` is allowed.
`

const Account_CapabilitiesTypeUnpublishFunctionName = "unpublish"

var Account_CapabilitiesTypeUnpublishFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(PublicPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &CapabilityType{},
		},
	),
}

const Account_CapabilitiesTypeUnpublishFunctionDocString = `
Unpublish the capability published at the given path.

Returns the capability if one was published at the path.
Returns nil if no capability was published at the path.
`

const Account_CapabilitiesTypeName = "Capabilities"

var Account_CapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_CapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			Account_CapabilitiesType,
			newEntitlementAccess(
				[]Type{CapabilitiesMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			Account_CapabilitiesTypeStorageFieldName,
			Account_CapabilitiesTypeStorageFieldType,
			Account_CapabilitiesTypeStorageFieldDocString,
		),
		NewUnmeteredFieldMember(
			Account_CapabilitiesType,
			newEntitlementAccess(
				[]Type{CapabilitiesMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			Account_CapabilitiesTypeAccountFieldName,
			Account_CapabilitiesTypeAccountFieldType,
			Account_CapabilitiesTypeAccountFieldDocString,
		),
		NewUnmeteredFunctionMember(
			Account_CapabilitiesType,
			PrimitiveAccess(ast.AccessAll),
			Account_CapabilitiesTypeGetFunctionName,
			Account_CapabilitiesTypeGetFunctionType,
			Account_CapabilitiesTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_CapabilitiesType,
			PrimitiveAccess(ast.AccessAll),
			Account_CapabilitiesTypeBorrowFunctionName,
			Account_CapabilitiesTypeBorrowFunctionType,
			Account_CapabilitiesTypeBorrowFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_CapabilitiesType,
			newEntitlementAccess(
				[]Type{PublishCapabilityType},
				Conjunction,
			),
			Account_CapabilitiesTypePublishFunctionName,
			Account_CapabilitiesTypePublishFunctionType,
			Account_CapabilitiesTypePublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_CapabilitiesType,
			newEntitlementAccess(
				[]Type{UnpublishCapabilityType},
				Conjunction,
			),
			Account_CapabilitiesTypeUnpublishFunctionName,
			Account_CapabilitiesTypeUnpublishFunctionType,
			Account_CapabilitiesTypeUnpublishFunctionDocString,
		),
	}

	Account_CapabilitiesType.Members = MembersAsMap(members)
	Account_CapabilitiesType.Fields = MembersFieldNames(members)
}

const Account_StorageCapabilitiesTypeGetControllerFunctionName = "getController"

var Account_StorageCapabilitiesTypeGetControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "byCapabilityID",
			TypeAnnotation: NewTypeAnnotation(UInt64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &ReferenceType{
				Type:          StorageCapabilityControllerType,
				Authorization: UnauthorizedAccess,
			},
		},
	),
}

const Account_StorageCapabilitiesTypeGetControllerFunctionDocString = `
Get the storage capability controller for the capability with the specified ID.

Returns nil if the ID does not reference an existing storage capability.
`

const Account_StorageCapabilitiesTypeGetControllersFunctionName = "getControllers"

var Account_StorageCapabilitiesTypeGetControllersFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "forPath",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: &ReferenceType{
				Type:          StorageCapabilityControllerType,
				Authorization: UnauthorizedAccess,
			},
		},
	),
}

const Account_StorageCapabilitiesTypeGetControllersFunctionDocString = `
Get all storage capability controllers for capabilities that target this storage path
`

const Account_StorageCapabilitiesTypeForEachControllerFunctionName = "forEachController"

var Account_StorageCapabilitiesTypeForEachControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "forPath",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(&ReferenceType{
							Type:          StorageCapabilityControllerType,
							Authorization: UnauthorizedAccess,
						}),
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

const Account_StorageCapabilitiesTypeForEachControllerFunctionDocString = `
Iterate over all storage capability controllers for capabilities that target this storage path,
passing a reference to each controller to the provided callback function.

Iteration is stopped early if the callback function returns ` + "`false`" + `.

If a new storage capability controller is issued for the path,
an existing storage capability controller for the path is deleted,
or a storage capability controller is retargeted from or to the path,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const Account_StorageCapabilitiesTypeIssueFunctionName = "issue"

var Account_StorageCapabilitiesTypeIssueFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var Account_StorageCapabilitiesTypeIssueFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_StorageCapabilitiesTypeIssueFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		MustInstantiate(
			&CapabilityType{},
			&GenericType{
				TypeParameter: Account_StorageCapabilitiesTypeIssueFunctionTypeParameterT,
			},
		),
	),
}

const Account_StorageCapabilitiesTypeIssueFunctionDocString = `
Issue/create a new storage capability.
`

const Account_StorageCapabilitiesTypeName = "StorageCapabilities"

var Account_StorageCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_StorageCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			Account_StorageCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetStorageCapabilityControllerType},
				Conjunction,
			),
			Account_StorageCapabilitiesTypeGetControllerFunctionName,
			Account_StorageCapabilitiesTypeGetControllerFunctionType,
			Account_StorageCapabilitiesTypeGetControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetStorageCapabilityControllerType},
				Conjunction,
			),
			Account_StorageCapabilitiesTypeGetControllersFunctionName,
			Account_StorageCapabilitiesTypeGetControllersFunctionType,
			Account_StorageCapabilitiesTypeGetControllersFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetStorageCapabilityControllerType},
				Conjunction,
			),
			Account_StorageCapabilitiesTypeForEachControllerFunctionName,
			Account_StorageCapabilitiesTypeForEachControllerFunctionType,
			Account_StorageCapabilitiesTypeForEachControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_StorageCapabilitiesType,
			newEntitlementAccess(
				[]Type{IssueStorageCapabilityControllerType},
				Conjunction,
			),
			Account_StorageCapabilitiesTypeIssueFunctionName,
			Account_StorageCapabilitiesTypeIssueFunctionType,
			Account_StorageCapabilitiesTypeIssueFunctionDocString,
		),
	}

	Account_StorageCapabilitiesType.Members = MembersAsMap(members)
	Account_StorageCapabilitiesType.Fields = MembersFieldNames(members)
}

const Account_AccountCapabilitiesTypeGetControllerFunctionName = "getController"

var Account_AccountCapabilitiesTypeGetControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "byCapabilityID",
			TypeAnnotation: NewTypeAnnotation(UInt64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &ReferenceType{
				Type:          AccountCapabilityControllerType,
				Authorization: UnauthorizedAccess,
			},
		},
	),
}

const Account_AccountCapabilitiesTypeGetControllerFunctionDocString = `
Get capability controller for capability with the specified ID.

Returns nil if the ID does not reference an existing account capability.
`

const Account_AccountCapabilitiesTypeGetControllersFunctionName = "getControllers"

var Account_AccountCapabilitiesTypeGetControllersFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: &ReferenceType{
				Type:          AccountCapabilityControllerType,
				Authorization: UnauthorizedAccess,
			},
		},
	),
}

const Account_AccountCapabilitiesTypeGetControllersFunctionDocString = `
Get all capability controllers for all account capabilities.
`

const Account_AccountCapabilitiesTypeForEachControllerFunctionName = "forEachController"

var Account_AccountCapabilitiesTypeForEachControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(&ReferenceType{
							Type:          AccountCapabilityControllerType,
							Authorization: UnauthorizedAccess,
						}),
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

const Account_AccountCapabilitiesTypeForEachControllerFunctionDocString = `
Iterate over all account capability controllers for all account capabilities,
passing a reference to each controller to the provided callback function.

Iteration is stopped early if the callback function returns ` + "`false`" + `.

If a new account capability controller is issued for the account,
or an existing account capability controller for the account is deleted,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const Account_AccountCapabilitiesTypeIssueFunctionName = "issue"

var Account_AccountCapabilitiesTypeIssueFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: &RestrictedType{
			Type: AccountType,
		},
		Authorization: UnauthorizedAccess,
	},
}

var Account_AccountCapabilitiesTypeIssueFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		Account_AccountCapabilitiesTypeIssueFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		MustInstantiate(
			&CapabilityType{},
			&GenericType{
				TypeParameter: Account_AccountCapabilitiesTypeIssueFunctionTypeParameterT,
			},
		),
	),
}

const Account_AccountCapabilitiesTypeIssueFunctionDocString = `
Issue/create a new account capability.
`

const Account_AccountCapabilitiesTypeName = "AccountCapabilities"

var Account_AccountCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         Account_AccountCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			Account_AccountCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetAccountCapabilityControllerType},
				Conjunction,
			),
			Account_AccountCapabilitiesTypeGetControllerFunctionName,
			Account_AccountCapabilitiesTypeGetControllerFunctionType,
			Account_AccountCapabilitiesTypeGetControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_AccountCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetAccountCapabilityControllerType},
				Conjunction,
			),
			Account_AccountCapabilitiesTypeGetControllersFunctionName,
			Account_AccountCapabilitiesTypeGetControllersFunctionType,
			Account_AccountCapabilitiesTypeGetControllersFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_AccountCapabilitiesType,
			newEntitlementAccess(
				[]Type{GetAccountCapabilityControllerType},
				Conjunction,
			),
			Account_AccountCapabilitiesTypeForEachControllerFunctionName,
			Account_AccountCapabilitiesTypeForEachControllerFunctionType,
			Account_AccountCapabilitiesTypeForEachControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			Account_AccountCapabilitiesType,
			newEntitlementAccess(
				[]Type{IssueAccountCapabilityControllerType},
				Conjunction,
			),
			Account_AccountCapabilitiesTypeIssueFunctionName,
			Account_AccountCapabilitiesTypeIssueFunctionType,
			Account_AccountCapabilitiesTypeIssueFunctionDocString,
		),
	}

	Account_AccountCapabilitiesType.Members = MembersAsMap(members)
	Account_AccountCapabilitiesType.Fields = MembersFieldNames(members)
}

const AccountTypeName = "Account"

var AccountType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AccountTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	t.SetNestedType(Account_StorageTypeName, Account_StorageType)
	t.SetNestedType(Account_ContractsTypeName, Account_ContractsType)
	t.SetNestedType(Account_KeysTypeName, Account_KeysType)
	t.SetNestedType(Account_InboxTypeName, Account_InboxType)
	t.SetNestedType(Account_CapabilitiesTypeName, Account_CapabilitiesType)
	t.SetNestedType(Account_StorageCapabilitiesTypeName, Account_StorageCapabilitiesType)
	t.SetNestedType(Account_AccountCapabilitiesTypeName, Account_AccountCapabilitiesType)
	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			AccountType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			AccountTypeAddressFieldName,
			AccountTypeAddressFieldType,
			AccountTypeAddressFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			AccountTypeBalanceFieldName,
			AccountTypeBalanceFieldType,
			AccountTypeBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			AccountTypeAvailableBalanceFieldName,
			AccountTypeAvailableBalanceFieldType,
			AccountTypeAvailableBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			newEntitlementAccess(
				[]Type{AccountMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			AccountTypeStorageFieldName,
			AccountTypeStorageFieldType,
			AccountTypeStorageFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			newEntitlementAccess(
				[]Type{AccountMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			AccountTypeContractsFieldName,
			AccountTypeContractsFieldType,
			AccountTypeContractsFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			newEntitlementAccess(
				[]Type{AccountMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			AccountTypeKeysFieldName,
			AccountTypeKeysFieldType,
			AccountTypeKeysFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			newEntitlementAccess(
				[]Type{AccountMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			AccountTypeInboxFieldName,
			AccountTypeInboxFieldType,
			AccountTypeInboxFieldDocString,
		),
		NewUnmeteredFieldMember(
			AccountType,
			newEntitlementAccess(
				[]Type{AccountMappingType},
				Conjunction,
			),
			ast.VariableKindConstant,
			AccountTypeCapabilitiesFieldName,
			AccountTypeCapabilitiesFieldType,
			AccountTypeCapabilitiesFieldDocString,
		),
	}

	AccountType.Members = MembersAsMap(members)
	AccountType.Fields = MembersFieldNames(members)
}

var StorageType = &EntitlementType{
	Identifier: "Storage",
}

var SaveValueType = &EntitlementType{
	Identifier: "SaveValue",
}

var LoadValueType = &EntitlementType{
	Identifier: "LoadValue",
}

var BorrowValueType = &EntitlementType{
	Identifier: "BorrowValue",
}

var ContractsType = &EntitlementType{
	Identifier: "Contracts",
}

var AddContractType = &EntitlementType{
	Identifier: "AddContract",
}

var UpdateContractType = &EntitlementType{
	Identifier: "UpdateContract",
}

var RemoveContractType = &EntitlementType{
	Identifier: "RemoveContract",
}

var KeysType = &EntitlementType{
	Identifier: "Keys",
}

var AddKeyType = &EntitlementType{
	Identifier: "AddKey",
}

var RevokeKeyType = &EntitlementType{
	Identifier: "RevokeKey",
}

var InboxType = &EntitlementType{
	Identifier: "Inbox",
}

var PublishInboxCapabilityType = &EntitlementType{
	Identifier: "PublishInboxCapability",
}

var UnpublishInboxCapabilityType = &EntitlementType{
	Identifier: "UnpublishInboxCapability",
}

var ClaimInboxCapabilityType = &EntitlementType{
	Identifier: "ClaimInboxCapability",
}

var CapabilitiesType = &EntitlementType{
	Identifier: "Capabilities",
}

var StorageCapabilitiesType = &EntitlementType{
	Identifier: "StorageCapabilities",
}

var AccountCapabilitiesType = &EntitlementType{
	Identifier: "AccountCapabilities",
}

var PublishCapabilityType = &EntitlementType{
	Identifier: "PublishCapability",
}

var UnpublishCapabilityType = &EntitlementType{
	Identifier: "UnpublishCapability",
}

var GetStorageCapabilityControllerType = &EntitlementType{
	Identifier: "GetStorageCapabilityController",
}

var IssueStorageCapabilityControllerType = &EntitlementType{
	Identifier: "IssueStorageCapabilityController",
}

var GetAccountCapabilityControllerType = &EntitlementType{
	Identifier: "GetAccountCapabilityController",
}

var IssueAccountCapabilityControllerType = &EntitlementType{
	Identifier: "IssueAccountCapabilityController",
}

var AccountMappingType = &EntitlementMapType{
	Identifier: "AccountMapping",
	Relations: []EntitlementRelation{
		EntitlementRelation{
			Input:  SaveValueType,
			Output: SaveValueType,
		},
		EntitlementRelation{
			Input:  LoadValueType,
			Output: LoadValueType,
		},
		EntitlementRelation{
			Input:  BorrowValueType,
			Output: BorrowValueType,
		},
		EntitlementRelation{
			Input:  AddContractType,
			Output: AddContractType,
		},
		EntitlementRelation{
			Input:  UpdateContractType,
			Output: UpdateContractType,
		},
		EntitlementRelation{
			Input:  RemoveContractType,
			Output: RemoveContractType,
		},
		EntitlementRelation{
			Input:  AddKeyType,
			Output: AddKeyType,
		},
		EntitlementRelation{
			Input:  RevokeKeyType,
			Output: RevokeKeyType,
		},
		EntitlementRelation{
			Input:  PublishInboxCapabilityType,
			Output: PublishInboxCapabilityType,
		},
		EntitlementRelation{
			Input:  UnpublishInboxCapabilityType,
			Output: UnpublishInboxCapabilityType,
		},
		EntitlementRelation{
			Input:  StorageCapabilitiesType,
			Output: StorageCapabilitiesType,
		},
		EntitlementRelation{
			Input:  AccountCapabilitiesType,
			Output: AccountCapabilitiesType,
		},
		EntitlementRelation{
			Input:  GetStorageCapabilityControllerType,
			Output: GetStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  IssueStorageCapabilityControllerType,
			Output: IssueStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  GetAccountCapabilityControllerType,
			Output: GetAccountCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  IssueAccountCapabilityControllerType,
			Output: IssueAccountCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  StorageType,
			Output: SaveValueType,
		},
		EntitlementRelation{
			Input:  StorageType,
			Output: LoadValueType,
		},
		EntitlementRelation{
			Input:  StorageType,
			Output: BorrowValueType,
		},
		EntitlementRelation{
			Input:  ContractsType,
			Output: AddContractType,
		},
		EntitlementRelation{
			Input:  ContractsType,
			Output: UpdateContractType,
		},
		EntitlementRelation{
			Input:  ContractsType,
			Output: RemoveContractType,
		},
		EntitlementRelation{
			Input:  KeysType,
			Output: AddKeyType,
		},
		EntitlementRelation{
			Input:  KeysType,
			Output: RevokeKeyType,
		},
		EntitlementRelation{
			Input:  InboxType,
			Output: PublishInboxCapabilityType,
		},
		EntitlementRelation{
			Input:  InboxType,
			Output: UnpublishInboxCapabilityType,
		},
		EntitlementRelation{
			Input:  InboxType,
			Output: ClaimInboxCapabilityType,
		},
		EntitlementRelation{
			Input:  CapabilitiesType,
			Output: StorageCapabilitiesType,
		},
		EntitlementRelation{
			Input:  CapabilitiesType,
			Output: AccountCapabilitiesType,
		},
	},
}

var CapabilitiesMappingType = &EntitlementMapType{
	Identifier: "CapabilitiesMapping",
	Relations: []EntitlementRelation{
		EntitlementRelation{
			Input:  GetStorageCapabilityControllerType,
			Output: GetStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  IssueStorageCapabilityControllerType,
			Output: IssueStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  GetAccountCapabilityControllerType,
			Output: GetAccountCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  IssueAccountCapabilityControllerType,
			Output: IssueAccountCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  StorageCapabilitiesType,
			Output: GetStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  StorageCapabilitiesType,
			Output: IssueStorageCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  AccountCapabilitiesType,
			Output: GetAccountCapabilityControllerType,
		},
		EntitlementRelation{
			Input:  AccountCapabilitiesType,
			Output: IssueAccountCapabilityControllerType,
		},
	},
}

func init() {
	BuiltinEntitlementMappings[AccountMappingType.Identifier] = AccountMappingType
	addToBaseActivation(AccountMappingType)
	BuiltinEntitlementMappings[CapabilitiesMappingType.Identifier] = CapabilitiesMappingType
	addToBaseActivation(CapabilitiesMappingType)
	BuiltinEntitlements[StorageType.Identifier] = StorageType
	addToBaseActivation(StorageType)
	BuiltinEntitlements[SaveValueType.Identifier] = SaveValueType
	addToBaseActivation(SaveValueType)
	BuiltinEntitlements[LoadValueType.Identifier] = LoadValueType
	addToBaseActivation(LoadValueType)
	BuiltinEntitlements[BorrowValueType.Identifier] = BorrowValueType
	addToBaseActivation(BorrowValueType)
	BuiltinEntitlements[ContractsType.Identifier] = ContractsType
	addToBaseActivation(ContractsType)
	BuiltinEntitlements[AddContractType.Identifier] = AddContractType
	addToBaseActivation(AddContractType)
	BuiltinEntitlements[UpdateContractType.Identifier] = UpdateContractType
	addToBaseActivation(UpdateContractType)
	BuiltinEntitlements[RemoveContractType.Identifier] = RemoveContractType
	addToBaseActivation(RemoveContractType)
	BuiltinEntitlements[KeysType.Identifier] = KeysType
	addToBaseActivation(KeysType)
	BuiltinEntitlements[AddKeyType.Identifier] = AddKeyType
	addToBaseActivation(AddKeyType)
	BuiltinEntitlements[RevokeKeyType.Identifier] = RevokeKeyType
	addToBaseActivation(RevokeKeyType)
	BuiltinEntitlements[InboxType.Identifier] = InboxType
	addToBaseActivation(InboxType)
	BuiltinEntitlements[PublishInboxCapabilityType.Identifier] = PublishInboxCapabilityType
	addToBaseActivation(PublishInboxCapabilityType)
	BuiltinEntitlements[UnpublishInboxCapabilityType.Identifier] = UnpublishInboxCapabilityType
	addToBaseActivation(UnpublishInboxCapabilityType)
	BuiltinEntitlements[ClaimInboxCapabilityType.Identifier] = ClaimInboxCapabilityType
	addToBaseActivation(ClaimInboxCapabilityType)
	BuiltinEntitlements[CapabilitiesType.Identifier] = CapabilitiesType
	addToBaseActivation(CapabilitiesType)
	BuiltinEntitlements[StorageCapabilitiesType.Identifier] = StorageCapabilitiesType
	addToBaseActivation(StorageCapabilitiesType)
	BuiltinEntitlements[AccountCapabilitiesType.Identifier] = AccountCapabilitiesType
	addToBaseActivation(AccountCapabilitiesType)
	BuiltinEntitlements[PublishCapabilityType.Identifier] = PublishCapabilityType
	addToBaseActivation(PublishCapabilityType)
	BuiltinEntitlements[UnpublishCapabilityType.Identifier] = UnpublishCapabilityType
	addToBaseActivation(UnpublishCapabilityType)
	BuiltinEntitlements[GetStorageCapabilityControllerType.Identifier] = GetStorageCapabilityControllerType
	addToBaseActivation(GetStorageCapabilityControllerType)
	BuiltinEntitlements[IssueStorageCapabilityControllerType.Identifier] = IssueStorageCapabilityControllerType
	addToBaseActivation(IssueStorageCapabilityControllerType)
	BuiltinEntitlements[GetAccountCapabilityControllerType.Identifier] = GetAccountCapabilityControllerType
	addToBaseActivation(GetAccountCapabilityControllerType)
	BuiltinEntitlements[IssueAccountCapabilityControllerType.Identifier] = IssueAccountCapabilityControllerType
	addToBaseActivation(IssueAccountCapabilityControllerType)
}
