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

// StringType represents the string type
//
var StringType = &SimpleType{
	Name:                 "String",
	QualifiedName:        "String",
	TypeID:               "String",
	IsInvalid:            false,
	IsResource:           false,
	Storable:             true,
	Equatable:            true,
	ExternallyReturnable: true,
	ValueIndexingInfo: ValueIndexingInfo{
		IsValueIndexableType:          true,
		AllowsValueIndexingAssignment: false,
		ElementType: func(_ bool) Type {
			return CharacterType
		},
		IndexingType: IntegerType,
	},
}

func init() {
	StringType.Members = func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			"concat": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						t,
						identifier,
						stringTypeConcatFunctionType,
						stringTypeConcatFunctionDocString,
					)
				},
			},
			"slice": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						t,
						identifier,
						stringTypeSliceFunctionType,
						stringTypeSliceFunctionDocString,
					)
				},
			},
			"decodeHex": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						t,
						identifier,
						stringTypeDecodeHexFunctionType,
						stringTypeDecodeHexFunctionDocString,
					)
				},
			},
			"length": {
				Kind: common.DeclarationKindField,
				Resolve: func(identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						t,
						identifier,
						IntType,
						stringTypeLengthFieldDocString,
					)
				},
			},
		}
	}
}

var stringTypeConcatFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "other",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const stringTypeConcatFunctionDocString = `
Returns a new string which contains the given string concatenated to the end of the original string, but does not modify the original string
`

var stringTypeSliceFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Identifier:     "from",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
		{
			Identifier:     "upTo",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const stringTypeSliceFunctionDocString = `
Returns a new string containing the slice of the characters in the given string from start index ` + "`from`" + ` up to, but not including, the end index ` + "`upTo`" + `.

This function creates a new string whose length is ` + "`upTo - from`" + `.
It does not modify the original string.
If either of the parameters are out of the bounds of the string, the function will fail
`

var stringTypeDecodeHexFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		&VariableSizedType{
			Type: UInt8Type,
		},
	),
}

const stringTypeDecodeHexFunctionDocString = `
Returns an array containing the bytes represented by the given hexadecimal string.

The given string must only contain hexadecimal characters and must have an even length.
If the string is malformed, the program aborts
`

const stringTypeLengthFieldDocString = `
The number of characters in the string
`
