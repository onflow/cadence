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
	"github.com/onflow/cadence/runtime/errors"
)

// ContractType represents the type `Contract`

type ContractType struct{}

func (*ContractType) IsType() {}

func (*ContractType) String() string {
	return "Contract"
}

func (*ContractType) QualifiedString() string {
	return "Contract"
}

func (*ContractType) ID() TypeID {
	return "Contract"
}

func (*ContractType) Equal(other Type) bool {
	_, ok := other.(*ContractType)
	return ok
}

func (*ContractType) IsResourceType() bool {
	return false
}

func (*ContractType) IsInvalidType() bool {
	return false
}

func (*ContractType) IsStorable(_ map[*Member]bool) bool {
	return false
}

func (*ContractType) IsEquatable() bool {
	// TODO:
	return false
}

func (*ContractType) TypeAnnotationState() TypeAnnotationState {
	return TypeAnnotationStateValid
}

func (t *ContractType) RewriteWithRestrictedTypes() (Type, bool) {
	return t, false
}

func (*ContractType) Unify(_ Type, _ map[*TypeParameter]Type, _ func(err error), _ ast.Range) bool {
	return false
}

func (t *ContractType) Resolve(_ map[*TypeParameter]Type) Type {
	return t
}

const contractTypeNameFieldDocString = `
The name of the contract
`

const contractTypeCodeFieldDocString = `
The code of the contract
`

func (t *ContractType) GetMembers() map[string]MemberResolver {
	return withBuiltinMembers(t, map[string]MemberResolver{
		"name": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicConstantFieldMember(
					t,
					identifier,
					&StringType{},
					contractTypeNameFieldDocString,
				)
			},
		},
		"code": {
			Kind: common.DeclarationKindField,
			Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
				return NewPublicConstantFieldMember(
					t,
					identifier,
					&VariableSizedType{
						Type: &UInt8Type{},
					},
					contractTypeCodeFieldDocString,
				)
			},
		},
	})
}

func init() {
	addressType := &ContractType{}
	typeName := addressType.String()

	// check type is not accidentally redeclared
	if _, ok := BaseValues[typeName]; ok {
		panic(errors.NewUnreachableError())
	}

	BaseValues[typeName] = baseFunction{
		name: typeName,
		invokableType: &FunctionType{
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
			ReturnTypeAnnotation: NewTypeAnnotation(addressType),
		},
	}
}
