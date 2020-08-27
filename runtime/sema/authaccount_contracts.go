/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

// AuthAccountContractsType represents the type `AuthAccount.Contracts`

type AuthAccountContractsType struct{}

func (*AuthAccountContractsType) IsType() {}

func (*AuthAccountContractsType) String() string {
	return "Contracts"
}

func (*AuthAccountContractsType) QualifiedString() string {
	return "AuthAccount.Contracts"
}

func (*AuthAccountContractsType) ID() TypeID {
	return "AuthAccount.Contracts"
}

func (*AuthAccountContractsType) Equal(other Type) bool {
	_, ok := other.(*AuthAccountContractsType)
	return ok
}

func (*AuthAccountContractsType) IsResourceType() bool {
	return false
}

func (*AuthAccountContractsType) IsInvalidType() bool {
	return false
}

func (*AuthAccountContractsType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*AuthAccountContractsType) IsEquatable() bool {
	// TODO:
	return false
}

func (*AuthAccountContractsType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *AuthAccountContractsType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (*AuthAccountContractsType) Unify(_ Type, _ map[*TypeParameter]Type, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *AuthAccountContractsType) Resolve(_ map[*TypeParameter]Type) Type {
	return t
}

const authAccountContractsTypeAddFunctionDocString = `
Adds the given contract to the account.

Additional arguments arguments are passed to the initializer of the contract.

Fails if a contract/contract interface with the given name already exists in the account,
if the given code does not declare exactly one contract or contract interface,
or if the given name does not match the name of the contract/contract interface declaration in the code.

Returns the deployed contract.
`

var authAccountContractsTypeAddFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(&StringType{}),
		},
		{
			Identifier: "code",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: &UInt8Type{},
				},
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&DeployedContractType{}),
	// additional arguments are passed to the contract initializer
	RequiredArgumentCount: RequiredArgumentCount(2),
}

const authAccountContractsTypeUpdateExperimentalFunctionDocString = `
**Experimental**

Updates the code for the contract/contract interface  in the account.

Does **not** run the initializer of the contract/contract interface again. 
The contract instance in the world state stays as is.

Fails if no contract/contract interface with the given name exists in the account,
if the given code does not declare exactly one contract or contract interface,
or if the given name does not match the name of the contract/contract interface declaration in the code.

Returns the deployed contract for the updated contract.
`

const authAccountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

var authAccountContractsTypeGetFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(&StringType{}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &DeployedContractType{},
		},
	),
}

const authAccountContractsTypeRemoveFunctionDocString = `
Removes the contract/contract interface from the account which has the given name, if any.

Returns the deleted deployed contract, if any.

Returns nil if no contract/contract interface with the given name exist in the account.
`

var authAccountContractsTypeRemoveFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(&StringType{}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: &DeployedContractType{},
		},
	),
}

var authAccountContractsTypeUpdateExperimentalFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     "name",
			TypeAnnotation: NewTypeAnnotation(&StringType{}),
		},
		{
			Identifier: "code",
			TypeAnnotation: NewTypeAnnotation(
				&VariableSizedType{
					Type: &UInt8Type{},
				},
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(&DeployedContractType{}),
}

func (t *AuthAccountContractsType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, map[string]MemberResolver{
		"add": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					t,
					identifier,
					authAccountContractsTypeAddFunctionType,
					authAccountContractsTypeAddFunctionDocString,
				)
			},
		},
		"update__experimental": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					t,
					identifier,
					authAccountContractsTypeUpdateExperimentalFunctionType,
					authAccountContractsTypeUpdateExperimentalFunctionDocString,
				)
			},
		},
		"get": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					t,
					identifier,
					authAccountContractsTypeGetFunctionType,
					authAccountContractsTypeGetFunctionDocString,
				)
			},
		},
		"remove": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicFunctionMember(
					t,
					identifier,
					authAccountContractsTypeRemoveFunctionType,
					authAccountContractsTypeRemoveFunctionDocString,
				)
			},
		},
	})
}
