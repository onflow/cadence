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

import "github.com/onflow/cadence/runtime/common"

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
Iterate over all the public paths of an account.
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration, as well as the behavior of adding or removing objects from storage during iteration,
is undefined.
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
Iterate over all the private paths of an account.
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration, as well as the behavior of adding or removing objects from storage during iteration,
is undefined.
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
Iterate over all the stored paths of an account.
passing each path and type in turn to the provided callback function.

The callback function takes two arguments:
1. The path of the stored object
2. The runtime type of that object

Iteration is stopped early if the callback function returns ` + "`false`" + `.

The order of iteration, as well as the behavior of adding or removing objects from storage during iteration,
is undefined.
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
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountContractsType,
			AuthAccountContractsTypeNamesFieldName,
			AuthAccountContractsTypeNamesFieldType,
			AuthAccountContractsTypeNamesFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountContractsType,
			AuthAccountContractsTypeAddFunctionName,
			AuthAccountContractsTypeAddFunctionType,
			AuthAccountContractsTypeAddFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountContractsType,
			AuthAccountContractsTypeUpdate__experimentalFunctionName,
			AuthAccountContractsTypeUpdate__experimentalFunctionType,
			AuthAccountContractsTypeUpdate__experimentalFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountContractsType,
			AuthAccountContractsTypeGetFunctionName,
			AuthAccountContractsTypeGetFunctionType,
			AuthAccountContractsTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountContractsType,
			AuthAccountContractsTypeRemoveFunctionName,
			AuthAccountContractsTypeRemoveFunctionType,
			AuthAccountContractsTypeRemoveFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountContractsType,
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
		NewUnmeteredPublicFunctionMember(
			AuthAccountKeysType,
			AuthAccountKeysTypeAddFunctionName,
			AuthAccountKeysTypeAddFunctionType,
			AuthAccountKeysTypeAddFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountKeysType,
			AuthAccountKeysTypeGetFunctionName,
			AuthAccountKeysTypeGetFunctionType,
			AuthAccountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountKeysType,
			AuthAccountKeysTypeRevokeFunctionName,
			AuthAccountKeysTypeRevokeFunctionType,
			AuthAccountKeysTypeRevokeFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountKeysType,
			AuthAccountKeysTypeForEachFunctionName,
			AuthAccountKeysTypeForEachFunctionType,
			AuthAccountKeysTypeForEachFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountKeysType,
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
		NewUnmeteredPublicFunctionMember(
			AuthAccountInboxType,
			AuthAccountInboxTypePublishFunctionName,
			AuthAccountInboxTypePublishFunctionType,
			AuthAccountInboxTypePublishFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountInboxType,
			AuthAccountInboxTypeUnpublishFunctionName,
			AuthAccountInboxTypeUnpublishFunctionType,
			AuthAccountInboxTypeUnpublishFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountInboxType,
			AuthAccountInboxTypeClaimFunctionName,
			AuthAccountInboxTypeClaimFunctionType,
			AuthAccountInboxTypeClaimFunctionDocString,
		),
	}

	AuthAccountInboxType.Members = MembersAsMap(members)
	AuthAccountInboxType.Fields = MembersFieldNames(members)
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
	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeAddressFieldName,
			AuthAccountTypeAddressFieldType,
			AuthAccountTypeAddressFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeBalanceFieldName,
			AuthAccountTypeBalanceFieldType,
			AuthAccountTypeBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeAvailableBalanceFieldName,
			AuthAccountTypeAvailableBalanceFieldType,
			AuthAccountTypeAvailableBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeStorageUsedFieldName,
			AuthAccountTypeStorageUsedFieldType,
			AuthAccountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeStorageCapacityFieldName,
			AuthAccountTypeStorageCapacityFieldType,
			AuthAccountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeContractsFieldName,
			AuthAccountTypeContractsFieldType,
			AuthAccountTypeContractsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeKeysFieldName,
			AuthAccountTypeKeysFieldType,
			AuthAccountTypeKeysFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeInboxFieldName,
			AuthAccountTypeInboxFieldType,
			AuthAccountTypeInboxFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypePublicPathsFieldName,
			AuthAccountTypePublicPathsFieldType,
			AuthAccountTypePublicPathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypePrivatePathsFieldName,
			AuthAccountTypePrivatePathsFieldType,
			AuthAccountTypePrivatePathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			AuthAccountType,
			AuthAccountTypeStoragePathsFieldName,
			AuthAccountTypeStoragePathsFieldType,
			AuthAccountTypeStoragePathsFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeAddPublicKeyFunctionName,
			AuthAccountTypeAddPublicKeyFunctionType,
			AuthAccountTypeAddPublicKeyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeRemovePublicKeyFunctionName,
			AuthAccountTypeRemovePublicKeyFunctionType,
			AuthAccountTypeRemovePublicKeyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeSaveFunctionName,
			AuthAccountTypeSaveFunctionType,
			AuthAccountTypeSaveFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeTypeFunctionName,
			AuthAccountTypeTypeFunctionType,
			AuthAccountTypeTypeFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeLoadFunctionName,
			AuthAccountTypeLoadFunctionType,
			AuthAccountTypeLoadFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeCopyFunctionName,
			AuthAccountTypeCopyFunctionType,
			AuthAccountTypeCopyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeBorrowFunctionName,
			AuthAccountTypeBorrowFunctionType,
			AuthAccountTypeBorrowFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeLinkFunctionName,
			AuthAccountTypeLinkFunctionType,
			AuthAccountTypeLinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeLinkAccountFunctionName,
			AuthAccountTypeLinkAccountFunctionType,
			AuthAccountTypeLinkAccountFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeGetCapabilityFunctionName,
			AuthAccountTypeGetCapabilityFunctionType,
			AuthAccountTypeGetCapabilityFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeGetLinkTargetFunctionName,
			AuthAccountTypeGetLinkTargetFunctionType,
			AuthAccountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeUnlinkFunctionName,
			AuthAccountTypeUnlinkFunctionType,
			AuthAccountTypeUnlinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeForEachPublicFunctionName,
			AuthAccountTypeForEachPublicFunctionType,
			AuthAccountTypeForEachPublicFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeForEachPrivateFunctionName,
			AuthAccountTypeForEachPrivateFunctionType,
			AuthAccountTypeForEachPrivateFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			AuthAccountType,
			AuthAccountTypeForEachStoredFunctionName,
			AuthAccountTypeForEachStoredFunctionType,
			AuthAccountTypeForEachStoredFunctionDocString,
		),
	}

	AuthAccountType.Members = MembersAsMap(members)
	AuthAccountType.Fields = MembersFieldNames(members)
}
