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

const AuthAccountTypeName = "AuthAccount"
const AuthAccountTypeAddressFieldName = "address"
const AuthAccountTypeBalanceFieldName = "balance"
const AuthAccountTypeAvailableBalanceFieldName = "availableBalance"
const AuthAccountTypeStorageUsedFieldName = "storageUsed"
const AuthAccountTypeStorageCapacityFieldName = "storageCapacity"
const AuthAccountTypeAddPublicKeyFunctionName = "addPublicKey"
const AuthAccountTypeRemovePublicKeyFunctionName = "removePublicKey"
const AuthAccountTypeSaveFunctionName = "save"
const AuthAccountTypeLoadFunctionName = "load"
const AuthAccountTypeTypeFunctionName = "type"
const AuthAccountTypeCopyFunctionName = "copy"
const AuthAccountTypeBorrowFunctionName = "borrow"
const AuthAccountTypeLinkFunctionName = "link"
const AuthAccountTypeLinkAccountFunctionName = "linkAccount"
const AuthAccountTypeUnlinkFunctionName = "unlink"
const AuthAccountTypeGetCapabilityFunctionName = "getCapability"
const AuthAccountTypeGetLinkTargetFunctionName = "getLinkTarget"
const AuthAccountTypeForEachPublicFunctionName = "forEachPublic"
const AuthAccountTypeForEachPrivateFunctionName = "forEachPrivate"
const AuthAccountTypeForEachStoredFunctionName = "forEachStored"
const AuthAccountTypeContractsFieldName = "contracts"
const AuthAccountTypeKeysFieldName = "keys"
const AuthAccountTypeInboxFieldName = "inbox"
const AuthAccountTypePublicPathsFieldName = "publicPaths"
const AuthAccountTypePrivatePathsFieldName = "privatePaths"
const AuthAccountTypeStoragePathsFieldName = "storagePaths"
const AuthAccountTypeInboxPublishFunctionName = "publish"
const AuthAccountTypeInboxUnpublishFunctionName = "unpublish"
const AuthAccountTypeInboxClaimFunctionName = "claim"

// AuthAccountType represents the authorized access to an account.
// Access to an AuthAccount means having full access to its storage, public keys, and code.
// Only signed transactions can get the AuthAccount for an account.
var AuthAccountType = func() *CompositeType {

	authAccountType := &CompositeType{
		Identifier:         AuthAccountTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: true,
		importable:         false,
		NestedTypes: func() *StringTypeOrderedMap {
			nestedTypes := &StringTypeOrderedMap{}
			nestedTypes.Set(AuthAccountContractsTypeName, AuthAccountContractsType)
			nestedTypes.Set(AccountKeysTypeName, AuthAccountKeysType)
			nestedTypes.Set(AuthAccountInboxTypeName, AuthAccountInboxType)
			return nestedTypes
		}(),
	}

	var members = []*Member{
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeAddressFieldName,
			TheAddressType,
			accountTypeAddressFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeBalanceFieldName,
			UFix64Type,
			accountTypeAccountBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeAvailableBalanceFieldName,
			UFix64Type,
			accountTypeAccountAvailableBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeStorageUsedFieldName,
			UInt64Type,
			accountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeStorageCapacityFieldName,
			UInt64Type,
			accountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeAddPublicKeyFunctionName,
			AuthAccountTypeAddPublicKeyFunctionType,
			authAccountTypeAddPublicKeyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeRemovePublicKeyFunctionName,
			AuthAccountTypeRemovePublicKeyFunctionType,
			authAccountTypeRemovePublicKeyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeSaveFunctionName,
			AuthAccountTypeSaveFunctionType,
			authAccountTypeSaveFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeTypeFunctionName,
			AuthAccountTypeTypeFunctionType,
			authAccountTypeTypeFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeLoadFunctionName,
			AuthAccountTypeLoadFunctionType,
			authAccountTypeLoadFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeCopyFunctionName,
			AuthAccountTypeCopyFunctionType,
			authAccountTypeCopyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeBorrowFunctionName,
			AuthAccountTypeBorrowFunctionType,
			authAccountTypeBorrowFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeLinkFunctionName,
			AuthAccountTypeLinkFunctionType,
			authAccountTypeLinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeUnlinkFunctionName,
			AuthAccountTypeUnlinkFunctionType,
			authAccountTypeUnlinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeGetCapabilityFunctionName,
			AuthAccountTypeGetCapabilityFunctionType,
			authAccountTypeGetCapabilityFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeGetLinkTargetFunctionName,
			AccountTypeGetLinkTargetFunctionType,
			accountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeContractsFieldName,
			AuthAccountContractsType,
			accountTypeContractsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeKeysFieldName,
			AuthAccountKeysType,
			accountTypeKeysFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeInboxFieldName,
			AuthAccountInboxType,
			accountInboxDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypePublicPathsFieldName,
			AuthAccountPublicPathsType,
			authAccountTypePublicPathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypePrivatePathsFieldName,
			AuthAccountPrivatePathsType,
			authAccountTypePrivatePathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountTypeStoragePathsFieldName,
			AuthAccountStoragePathsType,
			authAccountTypeStoragePathsFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeForEachPublicFunctionName,
			AuthAccountForEachPublicFunctionType,
			authAccountForEachPublicDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeForEachPrivateFunctionName,
			AuthAccountForEachPrivateFunctionType,
			authAccountForEachPrivateDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeForEachStoredFunctionName,
			AuthAccountForEachStoredFunctionType,
			authAccountForEachStoredDocString,
		),
	}

	authAccountType.Members = GetMembersAsMap(members)
	authAccountType.Fields = GetFieldNames(members)
	return authAccountType
}()

var AuthAccountPublicPathsType = &VariableSizedType{
	Type: PublicPathType,
}

var AuthAccountPrivatePathsType = &VariableSizedType{
	Type: PrivatePathType,
}

var AuthAccountStoragePathsType = &VariableSizedType{
	Type: StoragePathType,
}

const authAccountTypeStoragePathsFieldDocString = `
All the storage paths of an account
`

const authAccountTypePublicPathsFieldDocString = `
All the public paths of an account
`

const authAccountTypePrivatePathsFieldDocString = `
All the private paths of an account
`

const authAccountForEachPublicDocString = `
Iterate over all the public paths of an account. Takes one argument: the function to be applied to each public path. 

This function parameter takes two arguments: the first is the path (/domain/key) of the stored object, and the second is the runtime type of that object.

The function parameter returns a bool indicating whether the iteration should continue; true will continue iterating onto the next element in storage, 
false will abort iteration.

The order of iteration, as well as the behavior of adding or removing keys from storage during iteration, is undefined. 
`

const authAccountForEachPrivateDocString = `
Iterate over all the private paths of an account. Takes one argument: the function to be applied to each private path. 

This function parameter takes two arguments: the first is the path (/domain/key) of the stored object, and the second is the runtime type of that object.

The function parameter returns a bool indicating whether the iteration should continue; true will continue iterating onto the next element in storage, 
false will abort iteration.

The order of iteration, as well as the behavior of adding or removing keys from storage during iteration, is undefined. 
`

const authAccountForEachStoredDocString = `
Iterate over all the storage paths of an account. Takes one argument: the function to be applied to each storage path. 

This function parameter takes two arguments: the first is the path (/domain/key) of the stored object, and the second is the runtime type of that object.

The function parameter returns a bool indicating whether the iteration should continue; true will continue iterating onto the next element in storage, 
false will abort iteration.

The order of iteration, as well as the behavior of adding or removing keys from storage during iteration, is undefined. 
`

var AuthAccountForEachPublicFunctionType = AccountForEachFunctionType(PublicPathType)

var AuthAccountForEachPrivateFunctionType = AccountForEachFunctionType(PrivatePathType)

var AuthAccountForEachStoredFunctionType = AccountForEachFunctionType(StoragePathType)

var AuthAccountTypeAddPublicKeyFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "key",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const authAccountTypeAddPublicKeyFunctionDocString = `
Adds the given byte representation of a public key to the account's keys
`

var AuthAccountTypeRemovePublicKeyFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "index",
			TypeAnnotation: NewTypeAnnotation(
				IntType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const authAccountTypeRemovePublicKeyFunctionDocString = `
Removes the public key at the given index from the account's keys
`

var AuthAccountTypeSaveFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		Name:      "T",
		TypeBound: StorableType,
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:      ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: NewTypeAnnotation(
					&GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
			{
				Label:          "to",
				Identifier:     "path",
				TypeAnnotation: NewTypeAnnotation(StoragePathType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
	}
}()

const authAccountTypeSaveFunctionDocString = `
Saves the given object into the account's storage at the given path.
Resources are moved into storage, and structures are copied.

If there is already an object stored under the given path, the program aborts.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

var AuthAccountTypeLoadFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		Name:      "T",
		TypeBound: StorableType,
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:          "from",
				Identifier:     "path",
				TypeAnnotation: NewTypeAnnotation(StoragePathType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: &GenericType{
					TypeParameter: typeParameter,
				},
			},
		),
	}
}()

const authAccountTypeTypeFunctionDocString = `
Reads the type of an object from the account's storage which is stored under the given path, or nil if no object is stored under the given path.

If there is an object stored, the type of the object is returned without modifying the stored object. 

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

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

const authAccountTypeLoadFunctionDocString = `
Loads an object from the account's storage which is stored under the given path, or nil if no object is stored under the given path.

If there is an object stored, the stored resource or structure is moved out of storage and returned as an optional.

When the function returns, the storage no longer contains an object under the given path.

The given type must be a supertype of the type of the loaded object.
If it is not, the function returns nil.
The given type must not necessarily be exactly the same as the type of the loaded object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

var AuthAccountTypeCopyFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		Name:      "T",
		TypeBound: AnyStructType,
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:          "from",
				Identifier:     "path",
				TypeAnnotation: NewTypeAnnotation(StoragePathType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: &GenericType{
					TypeParameter: typeParameter,
				},
			},
		),
	}
}()

const authAccountTypeCopyFunctionDocString = `
Returns a copy of a structure stored in account storage under the given path, without removing it from storage, or nil if no object is stored under the given path.

If there is a structure stored, it is copied.
The structure stays stored in storage after the function returns.

The given type must be a supertype of the type of the copied structure.
If it is not, the function returns nil.
The given type must not necessarily be exactly the same as the type of the copied structure.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

var AuthAccountTypeBorrowFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
		Name: "T",
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []Parameter{
			{
				Label:          "from",
				Identifier:     "path",
				TypeAnnotation: NewTypeAnnotation(StoragePathType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(
			&OptionalType{
				Type: &GenericType{
					TypeParameter: typeParameter,
				},
			},
		),
	}
}()

const authAccountTypeBorrowFunctionDocString = `
Returns a reference to an object in storage without removing it from storage.

If no object is stored under the given path, the function returns nil.
If there is an object stored, a reference is returned as an optional.

The given type must be a reference type.
It must be possible to create the given reference type for the borrowed object.
If it is not, the function returns nil.

The given type must not necessarily be exactly the same as the type of the borrowed object.

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

var AuthAccountTypeLinkFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
		Name: "T",
	}

	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
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
				Type: &CapabilityType{
					BorrowType: &GenericType{
						TypeParameter: typeParameter,
					},
				},
			},
		),
	}
}()

const authAccountTypeLinkFunctionDocString = `
Creates a capability at the given public or private path which targets the given public, private, or storage path.
The target path leads to the object that will provide the functionality defined by this capability.

The given type defines how the capability can be borrowed, i.e., how the stored value can be accessed.

Returns nil if a link for the given capability path already exists, or the newly created capability if not.

It is not necessary for the target path to lead to a valid object; the target path could be empty, or could lead to an object which does not provide the necessary type interface:
The link function does **not** check if the target path is valid/exists at the time the capability is created and does **not** check if the target value conforms to the given type.
The link is latent. The target value might be stored after the link is created, and the target value might be moved out after the link has been created.
`

var AuthAccountTypeUnlinkFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "capabilityPath",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(VoidType),
}

const authAccountTypeUnlinkFunctionDocString = `
Removes the capability at the given public or private path
`

var AuthAccountTypeGetCapabilityFunctionType = func() *FunctionType {

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
				TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
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

const authAccountTypeGetCapabilityFunctionDocString = `
Returns the capability at the given private or public path, or nil if it does not exist
`

var AccountTypeGetLinkTargetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "capabilityPath",
			TypeAnnotation: NewTypeAnnotation(CapabilityPathType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: PathType,
		},
	),
}

// AuthAccountKeysType represents the keys associated with an auth account.
var AuthAccountKeysType = func() *CompositeType {

	accountKeys := &CompositeType{
		Identifier: AccountKeysTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	var members = []*Member{
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysTypeAddFunctionName,
			AuthAccountKeysTypeAddFunctionType,
			authAccountKeysTypeAddFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysTypeGetFunctionName,
			AccountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysTypeRevokeFunctionName,
			AuthAccountKeysTypeRevokeFunctionType,
			authAccountKeysTypeRevokeFunctionDocString,
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

var AuthAccountKeysTypeAddFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     AccountKeyPublicKeyFieldName,
			TypeAnnotation: NewTypeAnnotation(PublicKeyType),
		},
		{
			Identifier:     AccountKeyHashAlgoFieldName,
			TypeAnnotation: NewTypeAnnotation(HashAlgorithmType),
		},
		{
			Identifier:     AccountKeyWeightFieldName,
			TypeAnnotation: NewTypeAnnotation(UFix64Type),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(AccountKeyType),
	RequiredArgumentCount: RequiredArgumentCount(3),
}

var AccountKeysTypeGetFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     AccountKeyKeyIndexFieldName,
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(&OptionalType{Type: AccountKeyType}),
	RequiredArgumentCount: RequiredArgumentCount(1),
}

// fun keys.forEach(_ function: ((AccountKey): Bool)): Void
var AccountKeysTypeForEachFunctionType = func() *FunctionType {
	// ((AccountKey): Bool)
	iterFunctionType := &FunctionType{
		Parameters: []Parameter{
			{
				TypeAnnotation: NewTypeAnnotation(AccountKeyType),
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
}()

var AccountKeysTypeCountFieldType = UInt64Type

var AuthAccountKeysTypeRevokeFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     AccountKeyKeyIndexFieldName,
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(&OptionalType{Type: AccountKeyType}),
	RequiredArgumentCount: RequiredArgumentCount(1),
}

func init() {
	// Set the container type after initializing the AccountKeysTypes, to avoid initializing loop.
	AuthAccountKeysType.SetContainerType(AuthAccountType)
}

const AccountKeysTypeName = "Keys"
const AccountKeysTypeAddFunctionName = "add"
const AccountKeysTypeGetFunctionName = "get"
const AccountKeysTypeForEachFunctionName = "forEach"
const AccountKeysTypeRevokeFunctionName = "revoke"
const AccountKeysTypeCountFieldName = "count"

const accountTypeGetLinkTargetFunctionDocString = `
Returns the target path of the capability at the given public or private path, or nil if there exists no capability at the given path.
`

const accountTypeAddressFieldDocString = `
The address of the account
`

const accountTypeContractsFieldDocString = `
The contracts of the account
`

const accountTypeAccountBalanceFieldDocString = `
The FLOW balance of the default vault of this account
`

const accountTypeAccountAvailableBalanceFieldDocString = `
The FLOW balance of the default vault of this account that is available to be moved
`

const accountTypeStorageUsedFieldDocString = `
The current amount of storage used by the account in bytes
`

const accountTypeStorageCapacityFieldDocString = `
The storage capacity of the account in bytes
`

const accountTypeKeysFieldDocString = `
The keys associated with the account
`

const authAccountKeysTypeAddFunctionDocString = `
Adds the given key to the keys list of the account.
`

const accountKeysTypeGetFunctionDocString = `
Retrieves the key at the given index of the account.
`

const authAccountKeysTypeRevokeFunctionDocString = `
Revokes the key at the given index of the account.
`
const accountKeysTypeForEachFunctionDocString = `
Iterates through all the keys of this account, passing each key to the provided function and short-circuiting if the function returns false. 

The order of iteration is undefined.
`

const accountKeysTypeCountFieldDocString = `
The number of keys associated with this account.
`

const authAccountTypeInboxPublishFunctionDocString = `
Publishes the argument value under the given name, to be later claimed by the specified recipient
`

var AuthAccountTypeInboxPublishFunctionType = &FunctionType{
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

const authAccountTypeInboxUnpublishFunctionDocString = `
Unpublishes the value specified by the argument string
`

var AuthAccountTypeInboxUnpublishFunctionType = func() *FunctionType {
	typeParameter := &TypeParameter{
		Name: "T",
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
	}
	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
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
				Type: &CapabilityType{
					BorrowType: &GenericType{
						TypeParameter: typeParameter,
					},
				},
			},
		),
	}
}()

const authAccountTypeInboxClaimFunctionDocString = `
Claims the value specified by the argument string from the account specified as the provider
`

var AuthAccountTypeInboxClaimFunctionType = func() *FunctionType {
	typeParameter := &TypeParameter{
		Name: "T",
		TypeBound: &ReferenceType{
			Type: AnyType,
		},
	}
	return &FunctionType{
		TypeParameters: []*TypeParameter{
			typeParameter,
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
				Type: &CapabilityType{
					BorrowType: &GenericType{
						TypeParameter: typeParameter,
					},
				},
			},
		),
	}
}()

var AuthAccountInboxTypeName = "Inbox"

var accountInboxDocString = "an inbox for sending and receiving capabilities"

// AuthAccountInboxType represents the account's inbox.
var AuthAccountInboxType = func() *CompositeType {

	accountInbox := &CompositeType{
		Identifier: AuthAccountInboxTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	var members = []*Member{
		NewUnmeteredPublicFunctionMember(
			accountInbox,
			AuthAccountTypeInboxClaimFunctionName,
			AuthAccountTypeInboxClaimFunctionType,
			authAccountTypeInboxClaimFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountInbox,
			AuthAccountTypeInboxPublishFunctionName,
			AuthAccountTypeInboxPublishFunctionType,
			authAccountTypeInboxPublishFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountInbox,
			AuthAccountTypeInboxUnpublishFunctionName,
			AuthAccountTypeInboxUnpublishFunctionType,
			authAccountTypeInboxUnpublishFunctionDocString,
		),
	}

	accountInbox.Members = GetMembersAsMap(members)
	accountInbox.Fields = GetFieldNames(members)
	return accountInbox
}()
