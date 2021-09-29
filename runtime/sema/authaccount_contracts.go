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
	"github.com/onflow/cadence/runtime/common"
)

const AuthAccountContractsTypeName = "Contracts"
const AuthAccountContractsTypeAddFunctionName = "add"
const AuthAccountContractsTypeGetFunctionName = "get"
const AuthAccountContractsTypeRemoveFunctionName = "remove"
const AuthAccountContractsTypeUpdateExperimentalFunctionName = "update__experimental"
const AuthAccountContractsTypeNamesField = "names"

// AuthAccountContractsType represents the type `AuthAccount.Contracts`
//
var AuthAccountContractsType = func() *CompositeType {

	authAccountContractsType := &CompositeType{
		Identifier: AuthAccountContractsTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	var members = []*Member{
		NewPublicFunctionMember(
			authAccountContractsType,
			AuthAccountContractsTypeAddFunctionName,
			AuthAccountContractsTypeAddFunctionType,
			authAccountContractsTypeAddFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountContractsType,
			AuthAccountContractsTypeUpdateExperimentalFunctionName,
			AuthAccountContractsTypeUpdateExperimentalFunctionType,
			authAccountContractsTypeUpdateExperimentalFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountContractsType,
			AuthAccountContractsTypeGetFunctionName,
			AuthAccountContractsTypeGetFunctionType,
			authAccountContractsTypeGetFunctionDocString,
		),
		NewPublicFunctionMember(
			authAccountContractsType,
			AuthAccountContractsTypeRemoveFunctionName,
			AuthAccountContractsTypeRemoveFunctionType,
			authAccountContractsTypeRemoveFunctionDocString,
		),
		NewPublicConstantFieldMember(
			authAccountContractsType,
			AuthAccountContractsTypeNamesField,
			&VariableSizedType{
				Type: StringType,
			},
			authAccountContractsTypeGetNamesDocString,
		),
	}

	authAccountContractsType.Members = GetMembersAsMap(members)
	authAccountContractsType.Fields = getFieldNames(members)
	return authAccountContractsType
}()

func init() {
	// Set the container type after initializing the `AuthAccountContractsType`, to avoid initializing loop.
	AuthAccountContractsType.SetContainerType(AuthAccountType)
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

var AuthAccountContractsTypeAddFunctionType = &FunctionType{
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
				ByteArrayType,
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

var AuthAccountContractsTypeUpdateExperimentalFunctionType = &FunctionType{
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
				ByteArrayType,
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

var AuthAccountContractsTypeGetFunctionType = &FunctionType{
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

var AuthAccountContractsTypeRemoveFunctionType = &FunctionType{
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

const authAccountContractsTypeGetNamesDocString = `
Names of all contracts deployed in the account.
`
