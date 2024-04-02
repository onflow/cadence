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

//go:generate go run ./gen account.cdc account.gen.go

var AccountTypeAnnotation = NewTypeAnnotation(AccountType)

var AccountReferenceType = &ReferenceType{
	Authorization: UnauthorizedAccess,
	Type:          AccountType,
}

var AccountReferenceTypeAnnotation = NewTypeAnnotation(AccountReferenceType)

// FullyEntitledAccountAccess represents
//
//	auth(Storage, Contracts, Keys, Inbox, Capabilities)
var FullyEntitledAccountAccess = NewEntitlementSetAccess(
	[]*EntitlementType{
		StorageType,
		ContractsType,
		KeysType,
		InboxType,
		CapabilitiesType,
	},
	Conjunction,
)

// FullyEntitledAccountReferenceType represents the type
//
//	auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account
var FullyEntitledAccountReferenceType = &ReferenceType{
	Authorization: FullyEntitledAccountAccess,
	Type:          AccountType,
}

var FullyEntitledAccountReferenceTypeAnnotation = NewTypeAnnotation(FullyEntitledAccountReferenceType)

func init() {
	Account_ContractsTypeAddFunctionType.Arity = &Arity{Min: 2}

	// capabilities.get has a strict supertype requirement that its type argument is not `Never`,
	// but we can't yet express this in source syntax.
	// TODO: if we add support for arbitrary logical type bounds to the source language, move this
	// into the generator
	Account_CapabilitiesTypeGetFunctionType.TypeParameters[0].TypeBound =
		NewConjunctionTypeBound(
			[]TypeBound{
				Account_CapabilitiesTypeGetFunctionType.TypeParameters[0].TypeBound,
				NewStrictSupertypeTypeBound(NeverType),
			},
		)

	addToBaseActivation(AccountMappingType)
	addToBaseActivation(CapabilitiesMappingType)
	addToBaseActivation(StorageType)
	addToBaseActivation(SaveValueType)
	addToBaseActivation(LoadValueType)
	addToBaseActivation(CopyValueType)
	addToBaseActivation(BorrowValueType)
	addToBaseActivation(ContractsType)
	addToBaseActivation(AddContractType)
	addToBaseActivation(UpdateContractType)
	addToBaseActivation(RemoveContractType)
	addToBaseActivation(KeysType)
	addToBaseActivation(AddKeyType)
	addToBaseActivation(RevokeKeyType)
	addToBaseActivation(InboxType)
	addToBaseActivation(PublishInboxCapabilityType)
	addToBaseActivation(UnpublishInboxCapabilityType)
	addToBaseActivation(ClaimInboxCapabilityType)
	addToBaseActivation(CapabilitiesType)
	addToBaseActivation(StorageCapabilitiesType)
	addToBaseActivation(AccountCapabilitiesType)
	addToBaseActivation(PublishCapabilityType)
	addToBaseActivation(UnpublishCapabilityType)
	addToBaseActivation(GetStorageCapabilityControllerType)
	addToBaseActivation(IssueStorageCapabilityControllerType)
	addToBaseActivation(GetAccountCapabilityControllerType)
	addToBaseActivation(IssueAccountCapabilityControllerType)
}
