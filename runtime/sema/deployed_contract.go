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

// DeployedContractType represents the type `DeployedContract`

type DeployedContractType struct{}

func (*DeployedContractType) IsType() {}

func (*DeployedContractType) String() string {
	return "DeployedContract"
}

func (*DeployedContractType) QualifiedString() string {
	return "DeployedContract"
}

func (*DeployedContractType) ID() TypeID {
	return "DeployedContract"
}

func (*DeployedContractType) Equal(other Type) bool {
	_, ok := other.(*DeployedContractType)
	return ok
}

func (*DeployedContractType) IsResourceType() bool {
	return false
}

func (*DeployedContractType) IsInvalidType() bool {
	return false
}

func (*DeployedContractType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*DeployedContractType) IsEquatable() bool {
	// TODO:
	return false
}

func (*DeployedContractType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *DeployedContractType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (*DeployedContractType) Unify(_ Type, _ map[*TypeParameter]Type, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *DeployedContractType) Resolve(_ map[*TypeParameter]Type) Type {
	return t
}

const deployedContractTypeContractFieldDocString = `
The deployed contract
`

const deployedContractTypeAddressFieldDocString = `
The address of the account where the contract is deployed at
`

func (t *DeployedContractType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, map[string]MemberResolver{
		"address": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicConstantFieldMember(
					t,
					identifier,
					&AddressType{},
					deployedContractTypeAddressFieldDocString,
				)
			},
		},
		"contract": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicConstantFieldMember(
					t,
					identifier,
					&ContractType{},
					deployedContractTypeContractFieldDocString,
				)
			},
		},
	})
}
