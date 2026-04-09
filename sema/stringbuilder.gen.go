// Code generated from stringbuilder.cdc. DO NOT EDIT.
/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

import "github.com/onflow/cadence/ast"

const StringBuilderTypeAppendFunctionName = "append"

var StringBuilderTypeAppendFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "string",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const StringBuilderTypeAppendFunctionDocString = `
Appends a string to the builder
`

const StringBuilderTypeAppendCharacterFunctionName = "appendCharacter"

var StringBuilderTypeAppendCharacterFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "character",
			TypeAnnotation: NewTypeAnnotation(CharacterType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const StringBuilderTypeAppendCharacterFunctionDocString = `
Appends a character to the builder
`

const StringBuilderTypeClearFunctionName = "clear"

var StringBuilderTypeClearFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const StringBuilderTypeClearFunctionDocString = `
Clears the builder, allowing it to be reused
`

const StringBuilderTypeToStringFunctionName = "toString"

var StringBuilderTypeToStringFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const StringBuilderTypeToStringFunctionDocString = `
Returns the built string
`

const StringBuilderTypeLengthFieldName = "length"

var StringBuilderTypeLengthFieldType = IntType

const StringBuilderTypeLengthFieldDocString = `
Returns the current length of the string being built
`

const StringBuilderTypeName = "StringBuilder"

var StringBuilderType = &SimpleType{
	Name:          StringBuilderTypeName,
	QualifiedName: StringBuilderTypeName,
	TypeID:        StringBuilderTypeName,
	TypeTag:       StringBuilderTypeTag,
	IsResource:    false,
	Storable:      false,
	Primitive:     false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
	ContainFields: false,
}

func init() {
	StringBuilderType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredFunctionMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				StringBuilderTypeAppendFunctionName,
				StringBuilderTypeAppendFunctionType,
				StringBuilderTypeAppendFunctionDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				StringBuilderTypeAppendCharacterFunctionName,
				StringBuilderTypeAppendCharacterFunctionType,
				StringBuilderTypeAppendCharacterFunctionDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				StringBuilderTypeClearFunctionName,
				StringBuilderTypeClearFunctionType,
				StringBuilderTypeClearFunctionDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				StringBuilderTypeToStringFunctionName,
				StringBuilderTypeToStringFunctionType,
				StringBuilderTypeToStringFunctionDocString,
			),
			NewUnmeteredFieldMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				StringBuilderTypeLengthFieldName,
				StringBuilderTypeLengthFieldType,
				StringBuilderTypeLengthFieldDocString,
			),
		})
	}
}
