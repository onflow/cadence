/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

const typeIdentifierDocString = `
The fully-qualified identifier of the type
`

// MetaType represents the type of a type.
//
var MetaType = &SimpleType{
	Name:                 "Type",
	QualifiedName:        "Type",
	TypeID:               "Type",
	tag:                  MetaTypeTag,
	IsInvalid:            false,
	IsResource:           false,
	Storable:             true,
	Equatable:            true,
	ExternallyReturnable: true,
	Importable:           true,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			"identifier": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						StringType,
						typeIdentifierDocString,
					)
				},
			},
		}
	},
}
