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

const MetaTypeIdentifierFieldName = "identifier"

const metaTypeIdentifierFieldDocString = `
The fully-qualified identifier of the type
`

const MetaTypeIsSubtypeFunctionName = "isSubtype"

const metaTypeIsSubtypeFunctionDocString = `
Returns true if this type is a subtype of the given type at run-time
`

const MetaTypeName = "Type"

// MetaType represents the type of a type.
var MetaType = &SimpleType{
	Name:          MetaTypeName,
	QualifiedName: MetaTypeName,
	TypeID:        MetaTypeName,
	tag:           MetaTypeTag,
	IsResource:    false,
	Storable:      true,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var MetaTypeIsSubtypeFunctionType = &FunctionType{
	Parameters: []Parameter{
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
		return MembersAsResolvers([]*Member{
			NewUnmeteredPublicConstantFieldMember(
				t,
				MetaTypeIdentifierFieldName,
				StringType,
				metaTypeIdentifierFieldDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				MetaTypeIsSubtypeFunctionName,
				MetaTypeIsSubtypeFunctionType,
				metaTypeIsSubtypeFunctionDocString,
			),
		})
	}
}
