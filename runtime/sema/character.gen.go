// Code generated from character.cdc. DO NOT EDIT.
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

import "github.com/onflow/cadence/runtime/ast"

const CharacterTypeUtf8FieldName = "utf8"

var CharacterTypeUtf8FieldType = &VariableSizedType{
	Type: UInt8Type,
}

const CharacterTypeUtf8FieldDocString = `
The byte array of the UTF-8 encoding.
`

const CharacterTypeToStringFunctionName = "toString"

var CharacterTypeToStringFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	ReturnTypeAnnotation: NewTypeAnnotation(
		StringType,
	),
}

const CharacterTypeToStringFunctionDocString = `
Returns this character as a String.
`

const CharacterTypeName = "Character"

var CharacterType = &SimpleType{
	Name:          CharacterTypeName,
	QualifiedName: CharacterTypeName,
	TypeID:        CharacterTypeName,
	TypeTag:       CharacterTypeTag,
	IsResource:    false,
	Storable:      true,
	Primitive:     true,
	Equatable:     true,
	Comparable:    true,
	Exportable:    true,
	Importable:    true,
	ContainFields: false,
	conformances:  []*InterfaceType{StructStringerType},
}

func init() {
	CharacterType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredFieldMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				CharacterTypeUtf8FieldName,
				CharacterTypeUtf8FieldType,
				CharacterTypeUtf8FieldDocString,
			),
			NewUnmeteredFunctionMember(
				t,
				PrimitiveAccess(ast.AccessAll),
				CharacterTypeToStringFunctionName,
				CharacterTypeToStringFunctionType,
				CharacterTypeToStringFunctionDocString,
			),
		})
	}
}
