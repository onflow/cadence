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

const metaTypeIdentifierDocString = `
The fully-qualified identifier of the type
`

const metaTypeSubtypeDocString = `
Returns true if this type is a subtype of the given type at run-time
`

const MetaTypeName = "Type"

// MetaType represents the type of a type.
var MetaType = &SimpleType{
	Name:                 MetaTypeName,
	QualifiedName:        MetaTypeName,
	TypeID:               MetaTypeName,
	tag:                  MetaTypeTag,
	IsResource:           false,
	Storable:             true,
	Equatable:            true,
	ExternallyReturnable: true,
	Importable:           true,
}

var MetaTypeIsSubtypeFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          "of",
			Identifier:     "otherType",
			TypeAnnotation: NewTypeAnnotation(MetaType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

func init() {
	MetaType.Members = func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			"identifier": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						StringType,
						metaTypeIdentifierDocString,
					)
				},
			},
			"isSubtype": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						MetaTypeIsSubtypeFunctionType,
						metaTypeSubtypeDocString,
					)
				},
			},
		}
	}
}
