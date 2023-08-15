// Code generated from deployedcontract.cdc. DO NOT EDIT.
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

import "github.com/onflow/cadence/runtime/ast"

const DeployedContractTypeAddressFieldName = "address"

var DeployedContractTypeAddressFieldType = TheAddressType

const DeployedContractTypeAddressFieldDocString = `
The address of the account where the contract is deployed at.
`

const DeployedContractTypeNameFieldName = "name"

var DeployedContractTypeNameFieldType = StringType

const DeployedContractTypeNameFieldDocString = `
The name of the contract.
`

const DeployedContractTypeCodeFieldName = "code"

var DeployedContractTypeCodeFieldType = &VariableSizedType{
	Type: UInt8Type,
}

const DeployedContractTypeCodeFieldDocString = `
The code of the contract.
`

const DeployedContractTypePublicTypesFunctionName = "publicTypes"

var DeployedContractTypePublicTypesFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: MetaType,
		},
	),
}

const DeployedContractTypePublicTypesFunctionDocString = `
Returns an array of ` + "`Type`" + ` objects representing all the public type declarations in this contract
(e.g. structs, resources, enums).

For example, given a contract
` + `
contract Foo {
access(all) struct Bar {...}
access(all) resource Qux {...}
}
` + `
then ` + "`.publicTypes()`" + ` will return an array equivalent to the expression ` + "`[Type<Bar>(), Type<Qux>()]`" + `
`

const DeployedContractTypeName = "DeployedContract"

var DeployedContractType = &SimpleType{
	Name:          DeployedContractTypeName,
	QualifiedName: DeployedContractTypeName,
	TypeID:        DeployedContractTypeName,
	tag:           DeployedContractTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
	ContainFields: true,
}

func init() {
	DeployedContractType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				DeployedContractTypeAddressFieldName,
				DeployedContractTypeAddressFieldType,
				DeployedContractTypeAddressFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				DeployedContractTypeNameFieldName,
				DeployedContractTypeNameFieldType,
				DeployedContractTypeNameFieldDocString,
			),
			NewUnmeteredFieldMember(
				t,
				ast.AccessAll,
				ast.VariableKindConstant,
				DeployedContractTypeCodeFieldName,
				DeployedContractTypeCodeFieldType,
				DeployedContractTypeCodeFieldDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				ast.AccessAll,
				DeployedContractTypePublicTypesFunctionName,
				DeployedContractTypePublicTypesFunctionType,
				DeployedContractTypePublicTypesFunctionDocString,
			),
		})
	}
}
