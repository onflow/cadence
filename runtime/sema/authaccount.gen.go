// Code generated from authaccount.cdc. DO NOT EDIT.
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

const AuthAccountTypeAddressFieldName = "address"

var AuthAccountTypeAddressFieldType = TheAddressType

const AuthAccountTypeAddressFieldDocString = `
The address of the account.
`

const AuthAccountTypeBalanceFieldName = "balance"

var AuthAccountTypeBalanceFieldType = UFix64Type

const AuthAccountTypeBalanceFieldDocString = `
The FLOW balance of the default vault of this account.
`

const AuthAccountTypeAvailableBalanceFieldName = "availableBalance"

var AuthAccountTypeAvailableBalanceFieldType = UFix64Type

const AuthAccountTypeAvailableBalanceFieldDocString = `
The FLOW balance of the default vault of this account that is available to be moved.
`

const AuthAccountTypeStorageUsedFieldName = "storageUsed"

var AuthAccountTypeStorageUsedFieldType = UInt64Type

const AuthAccountTypeStorageUsedFieldDocString = `
The current amount of storage used by the account in bytes.
`

const AuthAccountTypeStorageCapacityFieldName = "storageCapacity"

var AuthAccountTypeStorageCapacityFieldType = UInt64Type

const AuthAccountTypeStorageCapacityFieldDocString = `
The storage capacity of the account in bytes.
`

const AuthAccountTypeContractsFieldName = "contracts"

var AuthAccountTypeContractsFieldType = AuthAccountContractsType

const AuthAccountTypeContractsFieldDocString = `
The contracts deployed to the account.
`

const AuthAccountTypeKeysFieldName = "keys"

var AuthAccountTypeKeysFieldType = AuthAccountKeysType

const AuthAccountTypeKeysFieldDocString = `
The keys assigned to the account.
`

const AuthAccountTypeInboxFieldName = "inbox"

var AuthAccountTypeInboxFieldType = AuthAccountInboxType

const AuthAccountTypeInboxFieldDocString = `
The inbox allows bootstrapping (sending and receiving) capabilities.
`

const AuthAccountTypeCapabilitiesFieldName = "capabilities"

var AuthAccountTypeCapabilitiesFieldType = &ReferenceType{
	Type: AuthAccountCapabilitiesType,
}

const AuthAccountTypeCapabilitiesFieldDocString = `
The capabilities of the account.
`

const AuthAccountTypePublicPathsFieldName = "publicPaths"

var AuthAccountTypePublicPathsFieldType = &VariableSizedType{
	Type: PublicPathType,
}

const AuthAccountTypePublicPathsFieldDocString = `
All public paths of this account.
`

const AuthAccountTypePrivatePathsFieldName = "privatePaths"

var AuthAccountTypePrivatePathsFieldType = &VariableSizedType{
	Type: PrivatePathType,
}

const AuthAccountTypePrivatePathsFieldDocString = `
All private paths of this account.
`

const AuthAccountTypeStoragePathsFieldName = "storagePaths"

var AuthAccountTypeStoragePathsFieldType = &VariableSizedType{
	Type: StoragePathType,
}

const AuthAccountTypeStoragePathsFieldDocString = `
All storage paths of this account.
`

const AuthAccountTypeAddPublicKeyFunctionName = "addPublicKey"

var AuthAccountTypeAddPublicKeyFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "publicKey",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: UInt8Type,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const AuthAccountTypeAddPublicKeyFunctionDocString = `
**DEPRECATED**: Use ` + "`keys.add`" + ` instead.

Adds a public key to the account.

The public key must be encoded together with their signature algorithm, hashing algorithm and weight.
`

const AuthAccountTypeRemovePublicKeyFunctionName = "removePublicKey"

var AuthAccountTypeRemovePublicKeyFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "index",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const AuthAccountTypeRemovePublicKeyFunctionDocString = `
**DEPRECATED**: Use ` + "`keys.revoke`" + ` instead.

Revokes the key at the given index.
`

const AuthAccountTypeSaveFunctionName = "save"

var AuthAccountTypeSaveFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: StorableType,
}

var AuthAccountTypeSaveFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeSaveFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: NewTypeAnnotation(&GenericType{
				TypeParameter: AuthAccountTypeSaveFunctionTypeParameterT,
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

const AuthAccountTypeSaveFunctionDocString = `
Saves the given object into the account's storage at the given path.

Resources are moved into storage, and structures are copied.

If there is already an object stored under the given path, the program aborts.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const AuthAccountTypeTypeFunctionName = "type"

var AuthAccountTypeTypeFunctionType = &FunctionType{
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

const AuthAccountTypeTypeFunctionDocString = `
Reads the type of an object from the account's storage which is stored under the given path,
or nil if no object is stored under the given path.

If there is an object stored, the type of the object is returned without modifying the stored object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed.
`

const AuthAccountTypeLoadFunctionName = "load"

var AuthAccountTypeLoadFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: StorableType,
}

var AuthAccountTypeLoadFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeLoadFunctionTypeParameterT,
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
				TypeParameter: AuthAccountTypeLoadFunctionTypeParameterT,
			},
		},
	),
}

const AuthAccountTypeLoadFunctionDocString = `
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

const AuthAccountTypeCopyFunctionName = "copy"

var AuthAccountTypeCopyFunctionTypeParameterT = &TypeParameter{
	Name:      "T",
	TypeBound: AnyStructType,
}

var AuthAccountTypeCopyFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeCopyFunctionTypeParameterT,
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
				TypeParameter: AuthAccountTypeCopyFunctionTypeParameterT,
			},
		},
	),
}

const AuthAccountTypeCopyFunctionDocString = `
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

const AuthAccountTypeBorrowFunctionName = "borrow"

var AuthAccountTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeBorrowFunctionTypeParameterT,
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
				TypeParameter: AuthAccountTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const AuthAccountTypeBorrowFunctionDocString = `
Returns a reference to an object in storage without removing it from storage.

If no object is stored under the given path, the function returns nil.
If there is an object stored, a reference is returned as an optional,
provided it can be borrowed using the given type.
If the stored object cannot be borrowed using the given type, the function panics.

The given type must not necessarily be exactly the same as the type of the borrowed object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

const AuthAccountTypeLinkFunctionName = "link"

var AuthAccountTypeLinkFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountTypeLinkFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeLinkFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "newCapabilityPath",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
		{
			Identifier:     "target",
			TypeAnnotation: NewTypeAnnotation(PathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MustInstantiate(
				&CapabilityType{},
				&GenericType{
					TypeParameter: AuthAccountTypeLinkFunctionTypeParameterT,
				},
			),
		},
	),
}

const AuthAccountTypeLinkFunctionDocString = `
**DEPRECATED**: Instead, use ` + "`capabilities.storage.issue`" + `, and ` + "`capabilities.publish`" + ` if the path is public.

Creates a capability at the given public or private path,
which targets the given public, private, or storage path.

The target path leads to the object that will provide the functionality defined by this capability.

The given type defines how the capability can be borrowed, i.e., how the stored value can be accessed.

Returns nil if a link for the given capability path already exists, or the newly created capability if not.

It is not necessary for the target path to lead to a valid object; the target path could be empty,
or could lead to an object which does not provide the necessary type interface:
The link function does **not** check if the target path is valid/exists at the time the capability is created
and does **not** check if the target value conforms to the given type.

The link is latent.

The target value might be stored after the link is created,
and the target value might be moved out after the link has been created.
`

const AuthAccountTypeLinkAccountFunctionName = "linkAccount"

var AuthAccountTypeLinkAccountFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "newCapabilityPath",
			TypeAnnotation: NewTypeAnnotation(PrivatePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: MustInstantiate(
				&CapabilityType{},
				&ReferenceType{
					Type: AuthAccountType,
				},
			),
		},
	),
}

const AuthAccountTypeLinkAccountFunctionDocString = `
**DEPRECATED**: Use ` + "`capabilities.account.issue`" + ` instead.

Creates a capability at the given public or private path which targets this account.

Returns nil if a link for the given capability path already exists, or the newly created capability if not.
`

const AuthAccountTypeGetCapabilityFunctionName = "getCapability"

var AuthAccountTypeGetCapabilityFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountTypeGetCapabilityFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountTypeGetCapabilityFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		MustInstantiate(
			&CapabilityType{},
			&GenericType{
				TypeParameter: AuthAccountTypeGetCapabilityFunctionTypeParameterT,
			},
		),
	),
}

const AuthAccountTypeGetCapabilityFunctionDocString = `
**DEPRECATED**: Use ` + "`capabilities.get`" + ` instead.

Returns the capability at the given private or public path.
`

const AuthAccountTypeGetLinkTargetFunctionName = "getLinkTarget"

var AuthAccountTypeGetLinkTargetFunctionType = &FunctionType{
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

const AuthAccountTypeGetLinkTargetFunctionDocString = `
**DEPRECATED**: Use ` + "`capabilities.storage.getController`" + ` and ` + "`StorageCapabilityController.target()`" + `.

Returns the target path of the capability at the given public or private path,
or nil if there exists no capability at the given path.
`

const AuthAccountTypeUnlinkFunctionName = "unlink"

var AuthAccountTypeUnlinkFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const AuthAccountTypeUnlinkFunctionDocString = `
**DEPRECATED**: Use ` + "`capabilities.unpublish`" + ` instead if the path is public.

Removes the capability at the given public or private path.
`

const AuthAccountTypeForEachPublicFunctionName = "forEachPublic"

var AuthAccountTypeForEachPublicFunctionType = &FunctionType{
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

const AuthAccountTypeForEachPublicFunctionDocString = `
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

const AuthAccountTypeForEachPrivateFunctionName = "forEachPrivate"

var AuthAccountTypeForEachPrivateFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(PrivatePathType),
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

const AuthAccountTypeForEachPrivateFunctionDocString = `
Iterate over all the private paths of an account,
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration is undefined.

If an object is stored under a new private path,
or an existing object is removed from a private path,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const AuthAccountTypeForEachStoredFunctionName = "forEachStored"

var AuthAccountTypeForEachStoredFunctionType = &FunctionType{
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

const AuthAccountTypeForEachStoredFunctionDocString = `
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

const AuthAccountContractsTypeNamesFieldName = "names"

var AuthAccountContractsTypeNamesFieldType = &VariableSizedType{
	Type: StringType,
}

const AuthAccountContractsTypeNamesFieldDocString = `
The names of all contracts deployed in the account.
`

const AuthAccountContractsTypeAddFunctionName = "add"

var AuthAccountContractsTypeAddFunctionType = &FunctionType{
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

const AuthAccountContractsTypeAddFunctionDocString = `
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

const AuthAccountContractsTypeUpdate__experimentalFunctionName = "update__experimental"

var AuthAccountContractsTypeUpdate__experimentalFunctionType = &FunctionType{
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

const AuthAccountContractsTypeUpdate__experimentalFunctionDocString = `
**Experimental**

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

const AuthAccountContractsTypeGetFunctionName = "get"

var AuthAccountContractsTypeGetFunctionType = &FunctionType{
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

const AuthAccountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const AuthAccountContractsTypeRemoveFunctionName = "remove"

var AuthAccountContractsTypeRemoveFunctionType = &FunctionType{
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

const AuthAccountContractsTypeRemoveFunctionDocString = `
Removes the contract/contract interface from the account which has the given name, if any.

Returns the removed deployed contract, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const AuthAccountContractsTypeBorrowFunctionName = "borrow"

var AuthAccountContractsTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountContractsTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountContractsTypeBorrowFunctionTypeParameterT,
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
				TypeParameter: AuthAccountContractsTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const AuthAccountContractsTypeBorrowFunctionDocString = `
Returns a reference of the given type to the contract with the given name in the account, if any.

Returns nil if no contract with the given name exists in the account,
or if the contract does not conform to the given type.
`

const AuthAccountContractsTypeName = "Contracts"

var AuthAccountContractsType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountContractsTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountContractsTypeNamesFieldName,
			AuthAccountContractsTypeNamesFieldType,
			AuthAccountContractsTypeNamesFieldDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			AuthAccountContractsTypeAddFunctionName,
			AuthAccountContractsTypeAddFunctionType,
			AuthAccountContractsTypeAddFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			AuthAccountContractsTypeUpdate__experimentalFunctionName,
			AuthAccountContractsTypeUpdate__experimentalFunctionType,
			AuthAccountContractsTypeUpdate__experimentalFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			AuthAccountContractsTypeGetFunctionName,
			AuthAccountContractsTypeGetFunctionType,
			AuthAccountContractsTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			AuthAccountContractsTypeRemoveFunctionName,
			AuthAccountContractsTypeRemoveFunctionType,
			AuthAccountContractsTypeRemoveFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountContractsType,
			ast.AccessPublic,
			AuthAccountContractsTypeBorrowFunctionName,
			AuthAccountContractsTypeBorrowFunctionType,
			AuthAccountContractsTypeBorrowFunctionDocString,
		),
	}

	AuthAccountContractsType.Members = MembersAsMap(members)
	AuthAccountContractsType.Fields = MembersFieldNames(members)
}

const AuthAccountKeysTypeAddFunctionName = "add"

var AuthAccountKeysTypeAddFunctionType = &FunctionType{
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

const AuthAccountKeysTypeAddFunctionDocString = `
Adds a new key with the given hashing algorithm and a weight.

Returns the added key.
`

const AuthAccountKeysTypeGetFunctionName = "get"

var AuthAccountKeysTypeGetFunctionType = &FunctionType{
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

const AuthAccountKeysTypeGetFunctionDocString = `
Returns the key at the given index, if it exists, or nil otherwise.

Revoked keys are always returned, but they have ` + "`isRevoked`" + ` field set to true.
`

const AuthAccountKeysTypeRevokeFunctionName = "revoke"

var AuthAccountKeysTypeRevokeFunctionType = &FunctionType{
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

const AuthAccountKeysTypeRevokeFunctionDocString = `
Marks the key at the given index revoked, but does not delete it.

Returns the revoked key if it exists, or nil otherwise.
`

const AuthAccountKeysTypeForEachFunctionName = "forEach"

var AuthAccountKeysTypeForEachFunctionType = &FunctionType{
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

const AuthAccountKeysTypeForEachFunctionDocString = `
Iterate over all unrevoked keys in this account,
passing each key in turn to the provided function.

Iteration is stopped early if the function returns ` + "`false`" + `.

The order of iteration is undefined.
`

const AuthAccountKeysTypeCountFieldName = "count"

var AuthAccountKeysTypeCountFieldType = UInt64Type

const AuthAccountKeysTypeCountFieldDocString = `
The total number of unrevoked keys in this account.
`

const AuthAccountKeysTypeName = "Keys"

var AuthAccountKeysType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountKeysTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			AuthAccountKeysType,
			ast.AccessPublic,
			AuthAccountKeysTypeAddFunctionName,
			AuthAccountKeysTypeAddFunctionType,
			AuthAccountKeysTypeAddFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountKeysType,
			ast.AccessPublic,
			AuthAccountKeysTypeGetFunctionName,
			AuthAccountKeysTypeGetFunctionType,
			AuthAccountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountKeysType,
			ast.AccessPublic,
			AuthAccountKeysTypeRevokeFunctionName,
			AuthAccountKeysTypeRevokeFunctionType,
			AuthAccountKeysTypeRevokeFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountKeysType,
			ast.AccessPublic,
			AuthAccountKeysTypeForEachFunctionName,
			AuthAccountKeysTypeForEachFunctionType,
			AuthAccountKeysTypeForEachFunctionDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountKeysType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountKeysTypeCountFieldName,
			AuthAccountKeysTypeCountFieldType,
			AuthAccountKeysTypeCountFieldDocString,
		),
	}

	AuthAccountKeysType.Members = MembersAsMap(members)
	AuthAccountKeysType.Fields = MembersFieldNames(members)
}

const AuthAccountInboxTypePublishFunctionName = "publish"

var AuthAccountInboxTypePublishFunctionType = &FunctionType{
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

const AuthAccountInboxTypePublishFunctionDocString = `
Publishes a new Capability under the given name,
to be claimed by the specified recipient.
`

const AuthAccountInboxTypeUnpublishFunctionName = "unpublish"

var AuthAccountInboxTypeUnpublishFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountInboxTypeUnpublishFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountInboxTypeUnpublishFunctionTypeParameterT,
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
					TypeParameter: AuthAccountInboxTypeUnpublishFunctionTypeParameterT,
				},
			),
		},
	),
}

const AuthAccountInboxTypeUnpublishFunctionDocString = `
Unpublishes a Capability previously published by this account.

Returns ` + "`nil`" + ` if no Capability is published under the given name.

Errors if the Capability under that name does not match the provided type.
`

const AuthAccountInboxTypeClaimFunctionName = "claim"

var AuthAccountInboxTypeClaimFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountInboxTypeClaimFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountInboxTypeClaimFunctionTypeParameterT,
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
					TypeParameter: AuthAccountInboxTypeClaimFunctionTypeParameterT,
				},
			),
		},
	),
}

const AuthAccountInboxTypeClaimFunctionDocString = `
Claims a Capability previously published by the specified provider.

Returns ` + "`nil`" + ` if no Capability is published under the given name,
or if this account is not its intended recipient.

Errors if the Capability under that name does not match the provided type.
`

const AuthAccountInboxTypeName = "Inbox"

var AuthAccountInboxType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountInboxTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			AuthAccountInboxType,
			ast.AccessPublic,
			AuthAccountInboxTypePublishFunctionName,
			AuthAccountInboxTypePublishFunctionType,
			AuthAccountInboxTypePublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountInboxType,
			ast.AccessPublic,
			AuthAccountInboxTypeUnpublishFunctionName,
			AuthAccountInboxTypeUnpublishFunctionType,
			AuthAccountInboxTypeUnpublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountInboxType,
			ast.AccessPublic,
			AuthAccountInboxTypeClaimFunctionName,
			AuthAccountInboxTypeClaimFunctionType,
			AuthAccountInboxTypeClaimFunctionDocString,
		),
	}

	AuthAccountInboxType.Members = MembersAsMap(members)
	AuthAccountInboxType.Fields = MembersFieldNames(members)
}

const AuthAccountCapabilitiesTypeStorageFieldName = "storage"

var AuthAccountCapabilitiesTypeStorageFieldType = &ReferenceType{
	Type: AuthAccountStorageCapabilitiesType,
}

const AuthAccountCapabilitiesTypeStorageFieldDocString = `
The storage capabilities of the account.
`

const AuthAccountCapabilitiesTypeAccountFieldName = "account"

var AuthAccountCapabilitiesTypeAccountFieldType = &ReferenceType{
	Type: AuthAccountAccountCapabilitiesType,
}

const AuthAccountCapabilitiesTypeAccountFieldDocString = `
The account capabilities of the account.
`

const AuthAccountCapabilitiesTypeGetFunctionName = "get"

var AuthAccountCapabilitiesTypeGetFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountCapabilitiesTypeGetFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountCapabilitiesTypeGetFunctionTypeParameterT,
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
					TypeParameter: AuthAccountCapabilitiesTypeGetFunctionTypeParameterT,
				},
			),
		},
	),
}

const AuthAccountCapabilitiesTypeGetFunctionDocString = `
Returns the capability at the given public path.
Returns nil if the capability does not exist,
or if the given type is not a supertype of the capability's borrow type.
`

const AuthAccountCapabilitiesTypeBorrowFunctionName = "borrow"

var AuthAccountCapabilitiesTypeBorrowFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountCapabilitiesTypeBorrowFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountCapabilitiesTypeBorrowFunctionTypeParameterT,
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
				TypeParameter: AuthAccountCapabilitiesTypeBorrowFunctionTypeParameterT,
			},
		},
	),
}

const AuthAccountCapabilitiesTypeBorrowFunctionDocString = `
Borrows the capability at the given public path.
Returns nil if the capability does not exist, or cannot be borrowed using the given type.
The function is equivalent to ` + "`get(path)?.borrow()`" + `.
`

const AuthAccountCapabilitiesTypePublishFunctionName = "publish"

var AuthAccountCapabilitiesTypePublishFunctionType = &FunctionType{
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

const AuthAccountCapabilitiesTypePublishFunctionDocString = `
Publish the capability at the given public path.

If there is already a capability published under the given path, the program aborts.

The path must be a public path, i.e., only the domain ` + "`public`" + ` is allowed.
`

const AuthAccountCapabilitiesTypeUnpublishFunctionName = "unpublish"

var AuthAccountCapabilitiesTypeUnpublishFunctionType = &FunctionType{
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

const AuthAccountCapabilitiesTypeUnpublishFunctionDocString = `
Unpublish the capability published at the given path.

Returns the capability if one was published at the path.
Returns nil if no capability was published at the path.
`

const AuthAccountCapabilitiesTypeMigrateLinkFunctionName = "migrateLink"

var AuthAccountCapabilitiesTypeMigrateLinkFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "newCapabilityPath",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: UInt64Type,
		},
	),
}

const AuthAccountCapabilitiesTypeMigrateLinkFunctionDocString = `
**DEPRECATED**: This function only exists temporarily to aid in the migration of links.
This function will not be part of the final Capability Controller API.

Migrates the link at the given path to a capability controller.

Does not migrate intermediate links of the chain.
`

const AuthAccountCapabilitiesTypeName = "Capabilities"

var AuthAccountCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountCapabilitiesTypeStorageFieldName,
			AuthAccountCapabilitiesTypeStorageFieldType,
			AuthAccountCapabilitiesTypeStorageFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountCapabilitiesTypeAccountFieldName,
			AuthAccountCapabilitiesTypeAccountFieldType,
			AuthAccountCapabilitiesTypeAccountFieldDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountCapabilitiesTypeGetFunctionName,
			AuthAccountCapabilitiesTypeGetFunctionType,
			AuthAccountCapabilitiesTypeGetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountCapabilitiesTypeBorrowFunctionName,
			AuthAccountCapabilitiesTypeBorrowFunctionType,
			AuthAccountCapabilitiesTypeBorrowFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountCapabilitiesTypePublishFunctionName,
			AuthAccountCapabilitiesTypePublishFunctionType,
			AuthAccountCapabilitiesTypePublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountCapabilitiesTypeUnpublishFunctionName,
			AuthAccountCapabilitiesTypeUnpublishFunctionType,
			AuthAccountCapabilitiesTypeUnpublishFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountCapabilitiesTypeMigrateLinkFunctionName,
			AuthAccountCapabilitiesTypeMigrateLinkFunctionType,
			AuthAccountCapabilitiesTypeMigrateLinkFunctionDocString,
		),
	}

	AuthAccountCapabilitiesType.Members = MembersAsMap(members)
	AuthAccountCapabilitiesType.Fields = MembersFieldNames(members)
}

const AuthAccountStorageCapabilitiesTypeGetControllerFunctionName = "getController"

var AuthAccountStorageCapabilitiesTypeGetControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "byCapabilityID",
			TypeAnnotation: NewTypeAnnotation(UInt64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &ReferenceType{
				Type: StorageCapabilityControllerType,
			},
		},
	),
}

const AuthAccountStorageCapabilitiesTypeGetControllerFunctionDocString = `
Get the storage capability controller for the capability with the specified ID.

Returns nil if the ID does not reference an existing storage capability.
`

const AuthAccountStorageCapabilitiesTypeGetControllersFunctionName = "getControllers"

var AuthAccountStorageCapabilitiesTypeGetControllersFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "forPath",
			TypeAnnotation: NewTypeAnnotation(StoragePathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: &ReferenceType{
				Type: StorageCapabilityControllerType,
			},
		},
	),
}

const AuthAccountStorageCapabilitiesTypeGetControllersFunctionDocString = `
Get all storage capability controllers for capabilities that target this storage path
`

const AuthAccountStorageCapabilitiesTypeForEachControllerFunctionName = "forEachController"

var AuthAccountStorageCapabilitiesTypeForEachControllerFunctionType = &FunctionType{
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
							Type: StorageCapabilityControllerType,
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

const AuthAccountStorageCapabilitiesTypeForEachControllerFunctionDocString = `
Iterate over all storage capability controllers for capabilities that target this storage path,
passing a reference to each controller to the provided callback function.

Iteration is stopped early if the callback function returns ` + "`false`" + `.

If a new storage capability controller is issued for the path,
an existing storage capability controller for the path is deleted,
or a storage capability controller is retargeted from or to the path,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const AuthAccountStorageCapabilitiesTypeIssueFunctionName = "issue"

var AuthAccountStorageCapabilitiesTypeIssueFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
	},
}

var AuthAccountStorageCapabilitiesTypeIssueFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountStorageCapabilitiesTypeIssueFunctionTypeParameterT,
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
				TypeParameter: AuthAccountStorageCapabilitiesTypeIssueFunctionTypeParameterT,
			},
		),
	),
}

const AuthAccountStorageCapabilitiesTypeIssueFunctionDocString = `
Issue/create a new storage capability.
`

const AuthAccountStorageCapabilitiesTypeName = "StorageCapabilities"

var AuthAccountStorageCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountStorageCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			AuthAccountStorageCapabilitiesType,
			ast.AccessPublic,
			AuthAccountStorageCapabilitiesTypeGetControllerFunctionName,
			AuthAccountStorageCapabilitiesTypeGetControllerFunctionType,
			AuthAccountStorageCapabilitiesTypeGetControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountStorageCapabilitiesType,
			ast.AccessPublic,
			AuthAccountStorageCapabilitiesTypeGetControllersFunctionName,
			AuthAccountStorageCapabilitiesTypeGetControllersFunctionType,
			AuthAccountStorageCapabilitiesTypeGetControllersFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountStorageCapabilitiesType,
			ast.AccessPublic,
			AuthAccountStorageCapabilitiesTypeForEachControllerFunctionName,
			AuthAccountStorageCapabilitiesTypeForEachControllerFunctionType,
			AuthAccountStorageCapabilitiesTypeForEachControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountStorageCapabilitiesType,
			ast.AccessPublic,
			AuthAccountStorageCapabilitiesTypeIssueFunctionName,
			AuthAccountStorageCapabilitiesTypeIssueFunctionType,
			AuthAccountStorageCapabilitiesTypeIssueFunctionDocString,
		),
	}

	AuthAccountStorageCapabilitiesType.Members = MembersAsMap(members)
	AuthAccountStorageCapabilitiesType.Fields = MembersFieldNames(members)
}

const AuthAccountAccountCapabilitiesTypeGetControllerFunctionName = "getController"

var AuthAccountAccountCapabilitiesTypeGetControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "byCapabilityID",
			TypeAnnotation: NewTypeAnnotation(UInt64Type),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &ReferenceType{
				Type: AccountCapabilityControllerType,
			},
		},
	),
}

const AuthAccountAccountCapabilitiesTypeGetControllerFunctionDocString = `
Get capability controller for capability with the specified ID.

Returns nil if the ID does not reference an existing account capability.
`

const AuthAccountAccountCapabilitiesTypeGetControllersFunctionName = "getControllers"

var AuthAccountAccountCapabilitiesTypeGetControllersFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: &ReferenceType{
				Type: AccountCapabilityControllerType,
			},
		},
	),
}

const AuthAccountAccountCapabilitiesTypeGetControllersFunctionDocString = `
Get all capability controllers for all account capabilities.
`

const AuthAccountAccountCapabilitiesTypeForEachControllerFunctionName = "forEachController"

var AuthAccountAccountCapabilitiesTypeForEachControllerFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "function",
			TypeAnnotation: NewTypeAnnotation(&FunctionType{
				Parameters: []Parameter{
					{
						TypeAnnotation: NewTypeAnnotation(&ReferenceType{
							Type: AccountCapabilityControllerType,
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

const AuthAccountAccountCapabilitiesTypeForEachControllerFunctionDocString = `
Iterate over all account capability controllers for all account capabilities,
passing a reference to each controller to the provided callback function.

Iteration is stopped early if the callback function returns ` + "`false`" + `.

If a new account capability controller is issued for the account,
or an existing account capability controller for the account is deleted,
then the callback must stop iteration by returning false.
Otherwise, iteration aborts.
`

const AuthAccountAccountCapabilitiesTypeIssueFunctionName = "issue"

var AuthAccountAccountCapabilitiesTypeIssueFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: &RestrictedType{
			Type: AuthAccountType,
		},
	},
}

var AuthAccountAccountCapabilitiesTypeIssueFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		AuthAccountAccountCapabilitiesTypeIssueFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		MustInstantiate(
			&CapabilityType{},
			&GenericType{
				TypeParameter: AuthAccountAccountCapabilitiesTypeIssueFunctionTypeParameterT,
			},
		),
	),
}

const AuthAccountAccountCapabilitiesTypeIssueFunctionDocString = `
Issue/create a new account capability.
`

const AuthAccountAccountCapabilitiesTypeName = "AccountCapabilities"

var AuthAccountAccountCapabilitiesType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountAccountCapabilitiesTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			AuthAccountAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountAccountCapabilitiesTypeGetControllerFunctionName,
			AuthAccountAccountCapabilitiesTypeGetControllerFunctionType,
			AuthAccountAccountCapabilitiesTypeGetControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountAccountCapabilitiesTypeGetControllersFunctionName,
			AuthAccountAccountCapabilitiesTypeGetControllersFunctionType,
			AuthAccountAccountCapabilitiesTypeGetControllersFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountAccountCapabilitiesTypeForEachControllerFunctionName,
			AuthAccountAccountCapabilitiesTypeForEachControllerFunctionType,
			AuthAccountAccountCapabilitiesTypeForEachControllerFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountAccountCapabilitiesType,
			ast.AccessPublic,
			AuthAccountAccountCapabilitiesTypeIssueFunctionName,
			AuthAccountAccountCapabilitiesTypeIssueFunctionType,
			AuthAccountAccountCapabilitiesTypeIssueFunctionDocString,
		),
	}

	AuthAccountAccountCapabilitiesType.Members = MembersAsMap(members)
	AuthAccountAccountCapabilitiesType.Fields = MembersFieldNames(members)
}

const AuthAccountTypeName = "AuthAccount"

var AuthAccountType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         AuthAccountTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	t.SetNestedType(AuthAccountContractsTypeName, AuthAccountContractsType)
	t.SetNestedType(AuthAccountKeysTypeName, AuthAccountKeysType)
	t.SetNestedType(AuthAccountInboxTypeName, AuthAccountInboxType)
	t.SetNestedType(AuthAccountCapabilitiesTypeName, AuthAccountCapabilitiesType)
	t.SetNestedType(AuthAccountStorageCapabilitiesTypeName, AuthAccountStorageCapabilitiesType)
	t.SetNestedType(AuthAccountAccountCapabilitiesTypeName, AuthAccountAccountCapabilitiesType)
	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeAddressFieldName,
			AuthAccountTypeAddressFieldType,
			AuthAccountTypeAddressFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeBalanceFieldName,
			AuthAccountTypeBalanceFieldType,
			AuthAccountTypeBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeAvailableBalanceFieldName,
			AuthAccountTypeAvailableBalanceFieldType,
			AuthAccountTypeAvailableBalanceFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeStorageUsedFieldName,
			AuthAccountTypeStorageUsedFieldType,
			AuthAccountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeStorageCapacityFieldName,
			AuthAccountTypeStorageCapacityFieldType,
			AuthAccountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeContractsFieldName,
			AuthAccountTypeContractsFieldType,
			AuthAccountTypeContractsFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeKeysFieldName,
			AuthAccountTypeKeysFieldType,
			AuthAccountTypeKeysFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeInboxFieldName,
			AuthAccountTypeInboxFieldType,
			AuthAccountTypeInboxFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeCapabilitiesFieldName,
			AuthAccountTypeCapabilitiesFieldType,
			AuthAccountTypeCapabilitiesFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypePublicPathsFieldName,
			AuthAccountTypePublicPathsFieldType,
			AuthAccountTypePublicPathsFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypePrivatePathsFieldName,
			AuthAccountTypePrivatePathsFieldType,
			AuthAccountTypePrivatePathsFieldDocString,
		),
		NewUnmeteredFieldMember(
			AuthAccountType,
			ast.AccessPublic,
			ast.VariableKindConstant,
			AuthAccountTypeStoragePathsFieldName,
			AuthAccountTypeStoragePathsFieldType,
			AuthAccountTypeStoragePathsFieldDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeAddPublicKeyFunctionName,
			AuthAccountTypeAddPublicKeyFunctionType,
			AuthAccountTypeAddPublicKeyFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeRemovePublicKeyFunctionName,
			AuthAccountTypeRemovePublicKeyFunctionType,
			AuthAccountTypeRemovePublicKeyFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeSaveFunctionName,
			AuthAccountTypeSaveFunctionType,
			AuthAccountTypeSaveFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeTypeFunctionName,
			AuthAccountTypeTypeFunctionType,
			AuthAccountTypeTypeFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeLoadFunctionName,
			AuthAccountTypeLoadFunctionType,
			AuthAccountTypeLoadFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeCopyFunctionName,
			AuthAccountTypeCopyFunctionType,
			AuthAccountTypeCopyFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeBorrowFunctionName,
			AuthAccountTypeBorrowFunctionType,
			AuthAccountTypeBorrowFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeLinkFunctionName,
			AuthAccountTypeLinkFunctionType,
			AuthAccountTypeLinkFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeLinkAccountFunctionName,
			AuthAccountTypeLinkAccountFunctionType,
			AuthAccountTypeLinkAccountFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeGetCapabilityFunctionName,
			AuthAccountTypeGetCapabilityFunctionType,
			AuthAccountTypeGetCapabilityFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeGetLinkTargetFunctionName,
			AuthAccountTypeGetLinkTargetFunctionType,
			AuthAccountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeUnlinkFunctionName,
			AuthAccountTypeUnlinkFunctionType,
			AuthAccountTypeUnlinkFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeForEachPublicFunctionName,
			AuthAccountTypeForEachPublicFunctionType,
			AuthAccountTypeForEachPublicFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeForEachPrivateFunctionName,
			AuthAccountTypeForEachPrivateFunctionType,
			AuthAccountTypeForEachPrivateFunctionDocString,
		),
		NewUnmeteredFunctionMember(
			AuthAccountType,
			ast.AccessPublic,
			AuthAccountTypeForEachStoredFunctionName,
			AuthAccountTypeForEachStoredFunctionType,
			AuthAccountTypeForEachStoredFunctionDocString,
		),
	}

	AuthAccountType.Members = MembersAsMap(members)
	AuthAccountType.Fields = MembersFieldNames(members)
}
