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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// DeployedContractType represents the type `DeployedContract`
var DeployedContractType = &SimpleType{
	Name:          "DeployedContract",
	QualifiedName: "DeployedContract",
	TypeID:        "DeployedContract",
	tag:           DeployedContractTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Exportable:    false,
	Importable:    false,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			DeployedContractTypeAddressFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						&AddressType{},
						deployedContractTypeAddressFieldDocString,
					)
				},
			},
			DeployedContractTypeNameFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						StringType,
						deployedContractTypeNameFieldDocString,
					)
				},
			},
			DeployedContractTypeCodeFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						ByteArrayType,
						deployedContractTypeCodeFieldDocString,
					)
				},
			},
		}
	},
}

var DeployedContractTypeAnnotation = NewTypeAnnotation(DeployedContractType)

const DeployedContractTypeAddressFieldName = "address"

const deployedContractTypeAddressFieldDocString = `
The address of the account where the contract is deployed at
`

const DeployedContractTypeNameFieldName = "name"

const deployedContractTypeNameFieldDocString = `
The name of the contract
`

const DeployedContractTypeCodeFieldName = "code"

const deployedContractTypeCodeFieldDocString = `
The code of the contract
`
