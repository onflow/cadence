// Code generated from testdata/docstrings/test.cdc. DO NOT EDIT.
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

package docstrings

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

const DocstringsTypeOwoFieldName = "owo"

var DocstringsTypeOwoFieldType = sema.IntType

const DocstringsTypeOwoFieldDocString = `
This is a 1-line docstring.
`

const DocstringsTypeUwuFieldName = "uwu"

var DocstringsTypeUwuFieldType = &sema.VariableSizedType{
	Type: sema.IntType,
}

const DocstringsTypeUwuFieldDocString = `
This is a 2-line docstring.
This is the second line.
`

const DocstringsTypeNwnFunctionName = "nwn"

var DocstringsTypeNwnFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Identifier:     "x",
			TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.StringType,
		},
	),
}

const DocstringsTypeNwnFunctionDocString = `
This is a 3-line docstring for a function.
This is the second line.
And the third line!
`

const DocstringsTypeWithBlanksFieldName = "withBlanks"

var DocstringsTypeWithBlanksFieldType = sema.IntType

const DocstringsTypeWithBlanksFieldDocString = `
This is a multiline docstring.

There should be two newlines before this line!
`

const DocstringsTypeIsSmolBeanFunctionName = "isSmolBean"

var DocstringsTypeIsSmolBeanFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.BoolType,
	),
}

const DocstringsTypeIsSmolBeanFunctionDocString = `
The function ` + "`isSmolBean`" + ` has docstrings with backticks.
These should be handled accordingly.
`

const DocstringsTypeRunningOutOfIdeasFunctionName = "runningOutOfIdeas"

var DocstringsTypeRunningOutOfIdeasFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.OptionalType{
			Type: sema.UInt64Type,
		},
	),
}

const DocstringsTypeRunningOutOfIdeasFunctionDocString = `
A function with a docstring.
This docstring is ` + "`cool`" + ` because it has inline backticked expressions.
Look, I did it ` + "`again`" + `, wowie!!
`

const DocstringsTypeName = "Docstrings"

var DocstringsType = &sema.SimpleType{
	Name:          DocstringsTypeName,
	QualifiedName: DocstringsTypeName,
	TypeID:        DocstringsTypeName,
	TypeTag:       DocstringsTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
	ContainFields: false,
}

func init() {
	DocstringsType.Members = func(t *sema.SimpleType) map[string]sema.MemberResolver {
		return sema.MembersAsResolvers([]*sema.Member{
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				DocstringsTypeOwoFieldName,
				DocstringsTypeOwoFieldType,
				DocstringsTypeOwoFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				DocstringsTypeUwuFieldName,
				DocstringsTypeUwuFieldType,
				DocstringsTypeUwuFieldDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				DocstringsTypeNwnFunctionName,
				DocstringsTypeNwnFunctionType,
				DocstringsTypeNwnFunctionDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				DocstringsTypeWithBlanksFieldName,
				DocstringsTypeWithBlanksFieldType,
				DocstringsTypeWithBlanksFieldDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				DocstringsTypeIsSmolBeanFunctionName,
				DocstringsTypeIsSmolBeanFunctionType,
				DocstringsTypeIsSmolBeanFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				DocstringsTypeRunningOutOfIdeasFunctionName,
				DocstringsTypeRunningOutOfIdeasFunctionType,
				DocstringsTypeRunningOutOfIdeasFunctionDocString,
			),
		})
	}
}
