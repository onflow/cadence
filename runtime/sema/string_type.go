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

import (
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

const StringTypeJoinFunctionName = "join"
const StringTypeJoinFunctionDocString = `
Returns a string after joining the array of strings with the provided separator.
`

const StringTypeSplitFunctionName = "split"
const StringTypeSplitFunctionDocString = `
Returns a variable-sized array of strings after splitting the string on the delimiter.
`

// StringType represents the string type
var StringType = &SimpleType{
	Name:          "String",
	QualifiedName: "String",
	TypeID:        "String",
	TypeTag:       StringTypeTag,
	IsResource:    false,
	Storable:      true,
	Primitive:     true,
	Equatable:     true,
	Comparable:    true,
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

var StringTypeAnnotation = NewTypeAnnotation(StringType)

func init() {
	StringType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeConcatFunctionName,
				StringTypeConcatFunctionType,
				stringTypeConcatFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeSliceFunctionName,
				StringTypeSliceFunctionType,
				stringTypeSliceFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeDecodeHexFunctionName,
				StringTypeDecodeHexFunctionType,
				stringTypeDecodeHexFunctionDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				StringTypeUtf8FieldName,
				ByteArrayType,
				stringTypeUtf8FieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				StringTypeLengthFieldName,
				IntType,
				stringTypeLengthFieldDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeToLowerFunctionName,
				StringTypeToLowerFunctionType,
				stringTypeToLowerFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeSplitFunctionName,
				StringTypeSplitFunctionType,
				StringTypeSplitFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				StringTypeReplaceAllFunctionName,
				StringTypeReplaceAllFunctionType,
				StringTypeReplaceAllFunctionDocString,
			),
		})
	}
}

var StringTypeConcatFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "other",
			TypeAnnotation: StringTypeAnnotation,
		},
	},
	StringTypeAnnotation,
)

const StringTypeConcatFunctionName = "concat"

const stringTypeConcatFunctionDocString = `
Returns a new string which contains the given string concatenated to the end of the original string, but does not modify the original string
`

var StringTypeSliceFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "from",
			TypeAnnotation: IntTypeAnnotation,
		},
		{
			Identifier:     "upTo",
			TypeAnnotation: IntTypeAnnotation,
		},
	},
	StringTypeAnnotation,
)

const StringTypeSliceFunctionName = "slice"

const stringTypeSliceFunctionDocString = `
Returns a new string containing the slice of the characters in the given string from start index ` + "`from`" + ` up to, but not including, the end index ` + "`upTo`" + `.

This function creates a new string whose length is ` + "`upTo - from`" + `.
It does not modify the original string.
If either of the parameters are out of the bounds of the string, or the indices are invalid (` + "`from > upTo`" + `), then the function will fail
`

const StringTypeReplaceAllFunctionName = "replaceAll"
const StringTypeReplaceAllFunctionDocString = `
Returns a new string after replacing all the occurrences of parameter ` + "`of` with the parameter `with`" + `.

If ` + "`with`" + ` is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding k+1 replacements for a string of length k.
`

// ByteArrayType represents the type [UInt8]
var ByteArrayType = &VariableSizedType{
	Type: UInt8Type,
}

var ByteArrayTypeAnnotation = NewTypeAnnotation(ByteArrayType)

// ByteArrayArrayType represents the type [[UInt8]]
var ByteArrayArrayType = &VariableSizedType{
	Type: ByteArrayType,
}

var ByteArrayArrayTypeAnnotation = NewTypeAnnotation(ByteArrayArrayType)

var StringTypeDecodeHexFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	ByteArrayTypeAnnotation,
)

const StringTypeDecodeHexFunctionName = "decodeHex"

const stringTypeDecodeHexFunctionDocString = `
Returns an array containing the bytes represented by the given hexadecimal string.

The given string must only contain hexadecimal characters and must have an even length.
If the string is malformed, the program aborts
`

const StringTypeLengthFieldName = "length"

const stringTypeLengthFieldDocString = `
The number of characters in the string
`

const StringTypeUtf8FieldName = "utf8"

const stringTypeUtf8FieldDocString = `
The byte array of the UTF-8 encoding
`

var StringTypeToLowerFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	nil,
	StringTypeAnnotation,
)

const StringTypeToLowerFunctionName = "toLower"

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

	functionType := NewSimpleFunctionType(
		FunctionPurityView,
		nil,
		StringTypeAnnotation,
	)

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

	addMember(NewUnmeteredPublicFunctionMember(
		functionType,
		StringTypeJoinFunctionName,
		StringTypeJoinFunctionType,
		StringTypeJoinFunctionDocString,
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

var StringTypeEncodeHexFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "data",
			TypeAnnotation: ByteArrayTypeAnnotation,
		},
	},
	StringTypeAnnotation,
)

var StringTypeFromUtf8FunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "bytes",
			TypeAnnotation: ByteArrayTypeAnnotation,
		},
	},
	NewTypeAnnotation(
		&OptionalType{
			Type: StringType,
		},
	),
)

var StringTypeFromCharactersFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "characters",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: CharacterType,
			}),
		},
	},
	StringTypeAnnotation,
)

var StringTypeJoinFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:      ArgumentLabelNotRequired,
			Identifier: "strings",
			TypeAnnotation: NewTypeAnnotation(&VariableSizedType{
				Type: StringType,
			}),
		},
		{
			Identifier:     "separator",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	StringTypeAnnotation,
)

var StringTypeSplitFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "separator",
			TypeAnnotation: StringTypeAnnotation,
		},
	},
	NewTypeAnnotation(
		&VariableSizedType{
			Type: StringType,
		},
	),
)

var StringTypeReplaceAllFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Identifier:     "of",
			TypeAnnotation: StringTypeAnnotation,
		},
		{
			Identifier:     "with",
			TypeAnnotation: StringTypeAnnotation,
		},
	},
	StringTypeAnnotation,
)
