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
	"github.com/rivo/uniseg"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

// CharacterType represents the character type
//
var CharacterType = &SimpleType{
	Name:                 "Character",
	QualifiedName:        "Character",
	TypeID:               "Character",
	tag:                  CharacterTypeTag,
	IsInvalid:            false,
	IsResource:           false,
	Storable:             true,
	Equatable:            true,
	ExternallyReturnable: true,
	Importable:           true,
}

func IsValidCharacter(s string) bool {
	graphemes := uniseg.NewGraphemes(s)
	// a valid character must have exactly one grapheme cluster
	return graphemes.Next() && !graphemes.Next()
}

func init() {
	CharacterType.Members = func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			ToStringFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						ToStringFunctionType,
						toStringFunctionDocString,
					)
				},
			},
		}
	}
}
