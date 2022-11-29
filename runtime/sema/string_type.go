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
	"github.com/onflow/cadence/runtime/errors"
)

const StringTypeEncodeHexFunctionName = "encodeHex"
const StringTypeEncodeHexFunctionDocString = `
Returns a hexadecimal string for the given byte array
`

const StringTypeFromUtf8FunctionName = "fromUTF8"
const StringTypeFromUtf8FunctionDocString = `
Attempt to decode the input as a UTF-8 encoded string. Returns nil if the input bytes are malformed UTF-8
`

const StringTypeFromCharactersFunctionName = "fromCharacters"
const StringTypeFromCharactersFunctionDocString = `
Returns a string from the given array of characters
`

// StringType represents the string type
var StringType = &SimpleType{
	Name:          "String",
	QualifiedName: "String",
	TypeID:        "String",
	tag:           StringTypeTag,
	IsResource:    false,
	Storable:      true,
	Equatable:     true,
	Exportable:    true,
	Importable:    true,
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
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						StringTypeConcatFunctionType,
						stringTypeConcatFunctionDocString,
					)
				},
			},
			"slice": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						StringTypeSliceFunctionType,
						stringTypeSliceFunctionDocString,
					)
				},
			},
			"decodeHex": {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						StringTypeDecodeHexFunctionType,
						stringTypeDecodeHexFunctionDocString,
					)
				},
			},
			"utf8": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						ByteArrayType,
						stringTypeUtf8FieldDocString,
					)
				},
			},
			"length": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						IntType,
						stringTypeLengthFieldDocString,
					)
				},
			},
			"toLower": {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge, identifier string, _ ast.Range, _ func(error)) *Member {
					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						StringTypeToLowerFunctionType,
						stringTypeToLowerFunctionDocString,
					)
				},
			},
		}
	}
}

var StringTypeConcatFunctionType = &FunctionType{
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

var StringTypeSliceFunctionType = &FunctionType{
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
If either of the parameters are out of the bounds of the string, or the indices are invalid (` + "`from > upTo`" + `), then the function will fail
`

// ByteArrayType represents the type [UInt8]
var ByteArrayType = &VariableSizedType{
	Type: UInt8Type,
}

// ByteArrayArrayType represents the type [[UInt8]]
var ByteArrayArrayType = &VariableSizedType{
	Type: ByteArrayType,
}

var StringTypeDecodeHexFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(ByteArrayType),
}

const stringTypeDecodeHexFunctionDocString = `
Returns an array containing the bytes represented by the given hexadecimal string.

The given string must only contain hexadecimal characters and must have an even length.
If the string is malformed, the program aborts
`

const stringTypeLengthFieldDocString = `
The number of characters in the string
`

const stringTypeUtf8FieldDocString = `
The byte array of the UTF-8 encoding
`

var StringTypeToLowerFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(StringType),
}

const stringTypeToLowerFunctionDocString = `
Returns the string with upper case letters replaced with lowercase
`

const stringFunctionDocString = "Creates an empty string"

var StringFunctionType = func() *FunctionType {
	// Declare a function for the string type.
	// For now, it has no parameters and creates an empty string

	typeName := StringType.String()

	// Check that the function is not accidentally redeclared

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	functionType := &FunctionType{
		ReturnTypeAnnotation: NewTypeAnnotation(StringType),
	}

	addMember := func(member *Member) {
		if functionType.Members == nil {
			functionType.Members = &StringMemberOrderedMap{}
		}
		name := member.Identifier.Identifier
		if functionType.Members.Contains(name) {
			panic(errors.NewUnreachableError())
		}
		functionType.Members.Set(name, member)
	}

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		StringTypeEncodeHexFunctionName,
		StringTypeEncodeHexFunctionType,
		StringTypeEncodeHexFunctionDocString,
	))

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		StringTypeFromUtf8FunctionName,
		StringTypeFromUtf8FunctionType,
		StringTypeFromUtf8FunctionDocString,
	))

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		StringTypeFromCharactersFunctionName,
		StringTypeFromCharactersFunctionType,
		StringTypeFromCharactersFunctionDocString,
	))

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			functionType,
			stringFunctionDocString,
		),
	)

	return functionType
}()

var StringTypeEncodeHexFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "data",
			TypeAnnotation: NewTypeAnnotation(
				ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

var StringTypeFromUtf8FunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "bytes",
			TypeAnnotation: NewTypeAnnotation(ByteArrayType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		&OptionalType{
			Type: StringType,
		},
	),
}

var StringTypeFromCharactersFunctionType = &FunctionType{
	Parameters: []*Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "characters",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: CharacterType,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}
