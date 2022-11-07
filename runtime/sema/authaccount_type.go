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
const AuthAccountAddressField = "address"
const AuthAccountBalanceField = "balance"
const AuthAccountAvailableBalanceField = "availableBalance"
const AuthAccountStorageUsedField = "storageUsed"
const AuthAccountStorageCapacityField = "storageCapacity"
const AuthAccountSaveField = "save"
const AuthAccountLoadField = "load"
const AuthAccountTypeField = "type"
const AuthAccountCopyField = "copy"
const AuthAccountBorrowField = "borrow"
const AuthAccountLinkField = "link"
const AuthAccountUnlinkField = "unlink"
const AuthAccountGetCapabilityField = "getCapability"
const AuthAccountGetLinkTargetField = "getLinkTarget"
const AuthAccountForEachPublicField = "forEachPublic"
const AuthAccountForEachPrivateField = "forEachPrivate"
const AuthAccountForEachStoredField = "forEachStored"
const AuthAccountContractsField = "contracts"
const AuthAccountKeysField = "keys"
const AuthAccountInboxField = "inbox"
const AuthAccountPublicPathsField = "publicPaths"
const AuthAccountPrivatePathsField = "privatePaths"
const AuthAccountStoragePathsField = "storagePaths"
const AuthAccountInboxPublishField = "publish"
const AuthAccountInboxUnpublishField = "unpublish"
const AuthAccountInboxClaimField = "claim"

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
			AuthAccountAddressField,
			&AddressType{},
			accountTypeAddressFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountBalanceField,
			UFix64Type,
			accountTypeAccountBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountAvailableBalanceField,
			UFix64Type,
			accountTypeAccountAvailableBalanceFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountStorageUsedField,
			UInt64Type,
			accountTypeStorageUsedFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountStorageCapacityField,
			UInt64Type,
			accountTypeStorageCapacityFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountSaveField,
			AuthAccountTypeSaveFunctionType,
			authAccountTypeSaveFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountTypeField,
			AuthAccountTypeTypeFunctionType,
			authAccountTypeTypeFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountLoadField,
			AuthAccountTypeLoadFunctionType,
			authAccountTypeLoadFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountCopyField,
			AuthAccountTypeCopyFunctionType,
			authAccountTypeCopyFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountBorrowField,
			AuthAccountTypeBorrowFunctionType,
			authAccountTypeBorrowFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountLinkField,
			AuthAccountTypeLinkFunctionType,
			authAccountTypeLinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountUnlinkField,
			AuthAccountTypeUnlinkFunctionType,
			authAccountTypeUnlinkFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountGetCapabilityField,
			AuthAccountTypeGetCapabilityFunctionType,
			authAccountTypeGetCapabilityFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountGetLinkTargetField,
			AccountTypeGetLinkTargetFunctionType,
			accountTypeGetLinkTargetFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountContractsField,
			AuthAccountContractsType,
			accountTypeContractsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountKeysField,
			AuthAccountKeysType,
			accountTypeKeysFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountInboxField,
			AuthAccountInboxType,
			accountInboxDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountPublicPathsField,
			AuthAccountPublicPathsType,
			authAccountTypePublicPathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountPrivatePathsField,
			AuthAccountPrivatePathsType,
			authAccountTypePrivatePathsFieldDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			authAccountType,
			AuthAccountStoragePathsField,
			AuthAccountStoragePathsType,
			authAccountTypeStoragePathsFieldDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountForEachPublicField,
			AuthAccountForEachPublicFunctionType,
			authAccountForEachPublicDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountForEachPrivateField,
			AuthAccountForEachPrivateFunctionType,
			authAccountForEachPrivateDocString,
		),
		NewUnmeteredPublicFunctionMember(
			authAccountType,
			AuthAccountForEachStoredField,
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

var AuthAccountTypeSaveFunctionType = func() *FunctionType {

	typeParameter := &TypeParameter{
		Name:      "T",
		TypeBound: StorableType,
	}

	return &FunctionType{
		Purity: FunctionPurityImpure,
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
		Purity: FunctionPurityImpure,
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

const authAccountTypeTypeFunctionDocString = `
Reads the type of an object from the account's storage which is stored under the given path, or nil if no object is stored under the given path.

If there is an object stored, the type of the object is returned without modifying the stored object. 

The path must be a storage path, i.e., only the domain ` + "`storage`" + ` is allowed
`

var AuthAccountTypeTypeFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	Parameters: []*Parameter{
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
		Purity: FunctionPurityView,
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
		Purity: FunctionPurityView,
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
		Purity: FunctionPurityImpure,
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
	Purity: FunctionPurityImpure,
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
		Purity: FunctionPurityView,
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
	Purity: FunctionPurityView,
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
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysAddFunctionName,
			AuthAccountKeysTypeAddFunctionType,
			authAccountKeysTypeAddFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysGetFunctionName,
			AccountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountKeys,
			AccountKeysRevokeFunctionName,
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
			AccountKeysCountFieldName,
			AccountKeysTypeCountFieldType,
			accountKeysTypeCountFieldDocString,
		),
	}

	accountKeys.Members = GetMembersAsMap(members)
	accountKeys.Fields = GetFieldNames(members)
	return accountKeys
}()

var AuthAccountKeysTypeAddFunctionType = &FunctionType{
	Purity: FunctionPurityImpure,
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
	Purity: FunctionPurityView,
	Parameters: []*Parameter{
		{
			Identifier:     AccountKeyKeyIndexField,
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation:  NewTypeAnnotation(&OptionalType{Type: AccountKeyType}),
	RequiredArgumentCount: RequiredArgumentCount(1),
}

// fun keys.forEach(_ function: ((AccountKey): Bool)): Void
var AccountKeysTypeForEachFunctionType = func() *FunctionType {
	const functionPurity = FunctionPurityImpure

	// ((AccountKey): Bool)
	iterFunctionType := &FunctionType{
		Purity: functionPurity,
		Parameters: []*Parameter{
			{
				TypeAnnotation: NewTypeAnnotation(AccountKeyType),
			},
		},
		ReturnTypeAnnotation: NewTypeAnnotation(BoolType),
	}

	return &FunctionType{
		Purity: functionPurity,
		Parameters: []*Parameter{
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
	Purity: FunctionPurityImpure,
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
const AccountKeysTypeForEachFunctionName = "forEach"
const AccountKeysRevokeFunctionName = "revoke"
const AccountKeysCountFieldName = "count"

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
	Purity: FunctionPurityImpure,
	Parameters: []*Parameter{
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
			TypeAnnotation: NewTypeAnnotation(&AddressType{}),
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
		Purity: FunctionPurityImpure,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []*Parameter{
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
		Purity: FunctionPurityImpure,
		TypeParameters: []*TypeParameter{
			typeParameter,
		},
		Parameters: []*Parameter{
			{
				Label:          ArgumentLabelNotRequired,
				Identifier:     "name",
				TypeAnnotation: NewTypeAnnotation(StringType),
			},
			{
				Identifier:     "provider",
				TypeAnnotation: NewTypeAnnotation(&AddressType{}),
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
			AuthAccountInboxClaimField,
			AuthAccountTypeInboxClaimFunctionType,
			authAccountTypeInboxClaimFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountInbox,
			AuthAccountInboxPublishField,
			AuthAccountTypeInboxPublishFunctionType,
			authAccountTypeInboxPublishFunctionDocString,
		),
		NewUnmeteredPublicFunctionMember(
			accountInbox,
			AuthAccountInboxUnpublishField,
			AuthAccountTypeInboxUnpublishFunctionType,
			authAccountTypeInboxUnpublishFunctionDocString,
		),
	}

	accountInbox.Members = GetMembersAsMap(members)
	accountInbox.Fields = GetFieldNames(members)
	return accountInbox
}()
