/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"fmt"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

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

	Account_CapabilitiesTypeGetFunctionType.TypeArgumentsCheck =
		func(memoryGauge common.MemoryGauge,
			typeArguments *TypeParameterTypeOrderedMap,
			_ []*ast.TypeAnnotation,
			invocationRange ast.HasPosition,
			report func(err error),
		) {
			typeArg, ok := typeArguments.Get(Account_CapabilitiesTypeGetFunctionTypeParameterT)
			if !ok || typeArg == nil {
				// Invalid, already reported by checker
				return
			}
			if typeArg == NeverType {
				report(&InvalidTypeArgumentError{
					TypeArgumentName: Account_CapabilitiesTypeGetFunctionTypeParameterT.Name,
					Range:            ast.NewRangeFromPositioned(memoryGauge, invocationRange),
					Details: fmt.Sprintf(
						"Type argument for `%s` cannot be `%s`",
						Account_CapabilitiesTypeGetFunctionName,
						NeverType,
					),
				})
			}
		}

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
