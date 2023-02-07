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

const AccountContractsTypeGetFunctionName = "get"
const AccountContractsTypeBorrowFunctionName = "borrow"
const AccountContractsTypeNamesFieldName = "names"

const accountContractsTypeGetFunctionDocString = `
Returns the deployed contract for the contract/contract interface with the given name in the account, if any.

Returns nil if no contract/contract interface with the given name exists in the account.
`

var AccountContractsTypeGetFunctionType = &FunctionType{
	Parameters: []Parameter{
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

const accountContractsTypeBorrowFunctionDocString = `
Returns a reference of the given type to the contract with the given name in the account, if any.

Returns nil if no contract with the given name exists in the account, or if the contract does not conform to the given type.
`

var AccountContractsTypeBorrowFunctionType = func() *FunctionType {

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
		Parameters: []Parameter{
			{
				Identifier:     "name",
				TypeAnnotation: NewTypeAnnotation(StringType),
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

var accountContractsTypeNamesFieldType = &VariableSizedType{
	Type: StringType,
}

const accountContractsTypeNamesFieldDocString = `
Names of all contracts deployed in the account.
`
