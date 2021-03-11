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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// PublicAccountType represents the publicly accessible portion of an account.
//
var PublicAccountType = &SimpleType{
	Name:                 "PublicAccount",
	QualifiedName:        "PublicAccount",
	TypeID:               "PublicAccount",
	IsInvalid:            false,
	IsResource:           false,
	Storable:             false,
	Equatable:            false,
	ExternallyReturnable: false,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			"address": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						&AddressType{},
						accountTypeAddressFieldDocString,
					)
				},
			},
			"storageUsed": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						&UInt64Type{},
						accountTypeStorageUsedFieldDocString,
					)
				},
			},
			"storageCapacity": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						&UInt64Type{},
						accountTypeStorageCapacityFieldDocString,
					)
				},
			},
			"getCapability": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						t,
						identifier,
						publicAccountTypeGetCapabilityFunctionType,
						publicAccountTypeGetLinkTargetFunctionDocString,
					)
				},
			},
			"getLinkTarget": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						t,
						identifier,
						accountTypeGetLinkTargetFunctionType,
						accountTypeGetLinkTargetFunctionDocString,
					)
				},
			},
			"keys": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						PublicAccountKeysType,
						accountTypeKeysFieldDocString,
					)
				},
			},
		}
	},

	NestedTypes: func() *StringTypeOrderedMap {
		nestedTypes := NewStringTypeOrderedMap()
		nestedTypes.Set(AccountKeysTypeName, PublicAccountKeysType)
		return nestedTypes
	}(),
}

// PublicAccountKeysType represents the keys associated with a public account.
var PublicAccountKeysType = func() *CompositeType {

	accountKeys := &CompositeType{
		Identifier: AccountKeysTypeName,
		Kind:       common.CompositeKindStructure,
	}

	var members = []*Member{
		NewPublicFunctionMember(
			accountKeys,
			AccountKeysGetFunctionName,
			accountKeysTypeGetFunctionType,
			accountKeysTypeGetFunctionDocString,
		),
	}

	accountKeys.Members = GetMembersAsMap(members)
	accountKeys.Fields = getFieldNames(members)
	return accountKeys
}()

func init() {
	// Set the container type after initializing the AccountKeysTypes, to avoid initializing loop.
	PublicAccountKeysType.ContainerType = PublicAccountType
}

const publicAccountTypeGetLinkTargetFunctionDocString = `
Returns the capability at the given public path, or nil if it does not exist
`
