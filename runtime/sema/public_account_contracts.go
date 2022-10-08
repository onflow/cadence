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

const PublicAccountContractsTypeName = "Contracts"
const PublicAccountContractsTypeGetFunctionName = "get"
const PublicAccountContractsTypeNamesField = "names"

// PublicAccountContractsType represents the type `PublicAccount.Contracts`
var PublicAccountContractsType = func() *CompositeType {

	publicAccountContractsType := &CompositeType{
		Identifier: PublicAccountContractsTypeName,
		Kind:       common.CompositeKindStructure,
		importable: false,
	}

	var members = []*Member{
		NewUnmeteredPublicFunctionMember(
			publicAccountContractsType,
			PublicAccountContractsTypeGetFunctionName,
			publicAccountContractsTypeGetFunctionType,
			publicAccountContractsTypeGetFunctionDocString,
		),
		NewUnmeteredPublicConstantFieldMember(
			publicAccountContractsType,
			PublicAccountContractsTypeNamesField,
			&VariableSizedType{
				Type: StringType,
			},
			publicAccountContractsTypeNamesDocString,
		),
	}

	publicAccountContractsType.Members = GetMembersAsMap(members)
	publicAccountContractsType.Fields = GetFieldNames(members)
	return publicAccountContractsType
}()

func init() {
	// Set the container type after initializing the `PublicAccountContractsType`, to avoid initializing loop.
	PublicAccountContractsType.SetContainerType(PublicAccountType)
}

const publicAccountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

var publicAccountContractsTypeGetFunctionType = &FunctionType{
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

const publicAccountContractsTypeNamesDocString = `
Names of all contracts deployed in the account.
`
