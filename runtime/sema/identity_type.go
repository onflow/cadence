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

const IdentityTypeName = "Identity"
const IdentityAddressField = "address"

// IdentityType represents the Identity of an Auth Account
// Identity can only be fetch as a reference from AuthAccount so it guarantees that the address in it comes from one of the signers.
// Only signed transactions can get the Identity for an account.
//
var IdentityType = func() *CompositeType {

	identityType := &CompositeType{
		Identifier:         IdentityTypeName,
		Kind:               common.CompositeKindStructure,
		hasComputedMembers: false,
		importable:         false,
	}

	var members = []*Member{
		NewPublicConstantFieldMember(
			identityType,
			IdentityAddressField,
			&AddressType{},
			accountTypeAddressFieldDocString,
		),
	}

	identityType.Members = GetMembersAsMap(members)
	identityType.Fields = getFieldNames(members)
	return identityType
}()

func init() {
	// Set the container type after initializing the AccountKeysTypes, to avoid initializing loop.
}

const identitytTypeAddressFieldDocString = `
The address of the identity
`
