/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
const AuthAccountAddressField = "address"
const AuthAccountBalanceField = "balance"
const AuthAccountAvailableBalanceField = "availableBalance"
const AuthAccountStorageUsedField = "storageUsed"
const AuthAccountStorageCapacityField = "storageCapacity"
const AuthAccountAddPublicKeyField = "addPublicKey"
const AuthAccountRemovePublicKeyField = "removePublicKey"
const AuthAccountSaveField = "save"
const AuthAccountLoadField = "load"
const AuthAccountCopyField = "copy"
const AuthAccountBorrowField = "borrow"
const AuthAccountLinkField = "link"
const AuthAccountUnlinkField = "unlink"
const AuthAccountGetCapabilityField = "getCapability"
const AuthAccountGetLinkTargetField = "getLinkTarget"
const AuthAccountContractsField = "contracts"
const AuthAccountKeysField = "keys"

// AuthAccountType represents the authorized access to an account.
// Access to an AuthAccount means having full access to its storage, public keys, and code.
// Only signed transactions can get the AuthAccount for an account.
//
var AuthAccountType = func() *CompositeType {

	authAccountType := &CompositeType{
		Identifier:         AuthAccountTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: true,
		importable:         false,

		nestedTypes: func() *StringTypeOrderedMap {
			nestedTypes := NewStringTypeOrderedMap()
			nestedTypes.Set(AuthAccountContractsTypeName, AuthAccountContractsType)
			nestedTypes.Set(AccountKeysTypeName, AuthAccountKeysType)
			return nestedTypes
		}(),
	}

	var members = []*Member{
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountAddressField,
			&AddressType{},
			accountTypeAddressFieldDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountBalanceField,
			UFix64Type,
			accountTypeAccountBalanceFieldDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountAvailableBalanceField,
			UFix64Type,
			accountTypeAccountAvailableBalanceFieldDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountStorageUsedField,
			UInt64Type,
			accountTypeStorageUsedFieldDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountStorageCapacityField,
			UInt64Type,
			accountTypeStorageCapacityFieldDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountAddPublicKeyField,
			AuthAccountTypeAddPublicKeyFunctionType,
			authAccountTypeAddPublicKeyFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountRemovePublicKeyField,
			AuthAccountTypeRemovePublicKeyFunctionType,
			authAccountTypeRemovePublicKeyFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountSaveField,
			AuthAccountTypeSaveFunctionType,
			authAccountTypeSaveFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountLoadField,
			AuthAccountTypeLoadFunctionType,
			authAccountTypeLoadFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountCopyField,
			AuthAccountTypeCopyFunctionType,
			authAccountTypeCopyFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountBorrowField,
			AuthAccountTypeBorrowFunctionType,
			authAccountTypeBorrowFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountLinkField,
			AuthAccountTypeLinkFunctionType,
			authAccountTypeLinkFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountUnlinkField,
			AuthAccountTypeUnlinkFunctionType,
			authAccountTypeUnlinkFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountGetCapabilityField,
			AuthAccountTypeGetCapabilityFunctionType,
			authAccountTypeGetCapabilityFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountType,
			AuthAccountGetLinkTargetField,
			AccountTypeGetLinkTargetFunctionType,
			accountTypeGetLinkTargetFunctionDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountContractsField,
			AuthAccountContractsType,
			accountTypeContractsFieldDocString,
		),
		NewPublicConstantFieldMember(
			authAccountType,
			AuthAccountKeysField,
			AuthAccountKeysType,
			accountTypeKeysFieldDocString,
		),
	}

	authAccountType.Members = GetMembersAsMap(members)
	authAccountType.Fields = getFieldNames(members)
	return authAccountType
}()

var AuthAccountTypeAddPublicKeyFunctionType = &FunctionType{
	Parameters: []*Parameter{
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
	Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
	Parameters: []*Parameter{
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
		Parameters: []*Parameter{
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
	Parameters: []*Parameter{
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
		NewPublicFunctionMember(
			accountKeys,
			AccountKeysAddFunctionName,
			AuthAccountKeysTypeAddFunctionType,
			authAccountKeysTypeAddFunctionDocString,
		),
		NewPublicFunctionMember(
			accountKeys,
			AccountKeysGetFunctionName,
			AccountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
		NewPublicFunctionMember(
			accountKeys,
			AccountKeysRevokeFunctionName,
			AuthAccountKeysTypeRevokeFunctionType,
			authAccountKeysTypeRevokeFunctionDocString,
		),
	}

	accountKeys.Members = GetMembersAsMap(members)
	accountKeys.Fields = getFieldNames(members)
	return accountKeys
}()

var AuthAccountKeysTypeAddFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     AccountKeyPublicKeyField,
			TypeAnnotation: NewTypeAnnotation(PublicKeyType),
		},
		{
			Identifier:     AccountKeyHashAlgoField,
			TypeAnnotation: NewTypeAnnotation(HashAlgorithmType),
		},
		{
			Identifier:     AccountKeyWeightField,
			TypeAnnotation: NewTypeAnnotation(UFix64Type),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(AccountKeyType),
	RequiredArgumentCount: RequiredArgumentCount(3),
}

var AccountKeysTypeGetFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     AccountKeyKeyIndexField,
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(&OptionalType{Type: AccountKeyType}),
	RequiredArgumentCount: RequiredArgumentCount(1),
}

var AuthAccountKeysTypeRevokeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     AccountKeyKeyIndexField,
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
const AccountKeysAddFunctionName = "add"
const AccountKeysGetFunctionName = "get"
const AccountKeysRevokeFunctionName = "revoke"

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
