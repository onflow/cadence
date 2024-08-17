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

const MetaTypeName = "Type"

// MetaType represents the type of a type.
var MetaType = &SimpleType{
	Name:          MetaTypeName,
	QualifiedName: MetaTypeName,
	TypeID:        MetaTypeName,
	TypeTag:       MetaTypeTag,
	IsResource:    false,
	Storable:      true,
	Primitive:     false,
	Equatable:     true,
	Comparable:    false,
	Exportable:    true,
	Importable:    true,
}

var MetaTypeAnnotation = NewTypeAnnotation(MetaType)

const MetaTypeIdentifierFieldName = "identifier"

const metaTypeIdentifierFieldDocString = `
The fully-qualified identifier of the type
`

var MetaTypeIsSubtypeFunctionType = NewSimpleFunctionType(
	FunctionPurityView,
	[]Parameter{
		{
			Label:          "of",
			Identifier:     "otherType",
			TypeAnnotation: MetaTypeAnnotation,
		},
	},
	BoolTypeAnnotation,
)

const MetaTypeIsSubtypeFunctionName = "isSubtype"

const metaTypeIsSubtypeFunctionDocString = `
Returns true if this type is a subtype of the given type at run-time
`

const MetaTypeIsRecoveredFieldName = "isRecovered"

var MetaTypeIsRecoveredFieldType = BoolType

const metaTypeIsRecoveredFieldDocString = `
The type was defined through a recovered program
`

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
			NewUnmeteredPublicConstantFieldMember(
				t,
				MetaTypeIsRecoveredFieldName,
				MetaTypeIsRecoveredFieldType,
				metaTypeIsRecoveredFieldDocString,
			),
		})
	}
}
