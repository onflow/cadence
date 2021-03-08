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

const AuthAccountContractsTypeName = "Contracts"

// AuthAccountContractsType represents the type `AuthAccount.Contracts`
//
var AuthAccountContractsType = &SimpleType{
	Name:                 AuthAccountContractsTypeName,
	QualifiedName:        "AuthAccount.Contracts",
	TypeID:               "AuthAccount.Contracts",
	IsInvalid:            false,
	IsResource:           false,
	Storable:             false,
	Equatable:            false,
	ExternallyReturnable: false,
	IsSuperTypeOf:        nil,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return withBuiltinMembers(t, map[string]MemberResolver{
			AuthAccountContractsTypeAddFunctionName: {
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
			AuthAccountContractsTypeUpdateExperimentalFunctionName: {
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
			AuthAccountContractsTypeGetFunctionName: {
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
			AuthAccountContractsTypeRemoveFunctionName: {
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
	},
}

const authAccountContractsTypeAddFunctionDocString = `
Adds the given contract to the account.

The ` + "`code`" + ` parameter is the UTF-8 encoded representation of the source code.
The code must contain exactly one contract or contract interface,
which must have the same name as the ` + "`name`" + ` parameter.

All additional arguments that are given are passed further to the initializer
of the contract that is being deployed.

Fails if a contract/contract interface with the given name already exists in the account,
if the given code does not declare exactly one contract or contract interface,
or if the given name does not match the name of the contract/contract interface declaration in the code.

Returns the deployed contract.
`

const AuthAccountContractsTypeAddFunctionName = "add"

var authAccountContractsTypeAddFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier: "name",
			TypeAnnotation: NewTypeAnnotation(
				StringType,
			),
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
	ReturnTypeAnnotation: NewTypeAnnotation(
		DeployedContractType,
	),
	// additional arguments are passed to the contract initializer
	RequiredArgumentCount: RequiredArgumentCount(2),
}

const authAccountContractsTypeUpdateExperimentalFunctionDocString = `
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

const AuthAccountContractsTypeUpdateExperimentalFunctionName = "update__experimental"

var authAccountContractsTypeUpdateExperimentalFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier: "name",
			TypeAnnotation: NewTypeAnnotation(
				StringType,
			),
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
	ReturnTypeAnnotation: NewTypeAnnotation(
		DeployedContractType,
	),
}

const authAccountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const AuthAccountContractsTypeGetFunctionName = "get"

var authAccountContractsTypeGetFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier: "name",
			TypeAnnotation: NewTypeAnnotation(
				StringType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: DeployedContractType,
		},
	),
}

const authAccountContractsTypeRemoveFunctionDocString = `
Removes the contract/contract interface from the account which has the given name, if any.

Returns the removed deployed contract, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

const AuthAccountContractsTypeRemoveFunctionName = "remove"

var authAccountContractsTypeRemoveFunctionType = &FunctionType{
	Parameters: []*Parameter{
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
