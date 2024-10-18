// Code generated from testdata/nested/test.cdc. DO NOT EDIT.
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

package nested

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

const FooTypeFooFunctionName = "foo"

var FooTypeFooFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const FooTypeFooFunctionDocString = `
foo
`

const FooTypeBarFieldName = "bar"

var FooTypeBarFieldType = Foo_BarType

const FooTypeBarFieldDocString = `
Bar
`

const Foo_BarTypeBarFunctionName = "bar"

var Foo_BarTypeBarFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const Foo_BarTypeBarFunctionDocString = `
bar
`

const Foo_BarTypeName = "Bar"

var Foo_BarType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         Foo_BarTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			Foo_BarType,
			sema.PrimitiveAccess(ast.AccessAll),
			Foo_BarTypeBarFunctionName,
			Foo_BarTypeBarFunctionType,
			Foo_BarTypeBarFunctionDocString,
		),
	}

	Foo_BarType.Members = sema.MembersAsMap(members)
	Foo_BarType.Fields = sema.MembersFieldNames(members)
}

const FooTypeName = "Foo"

var FooType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         FooTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	t.SetNestedType(Foo_BarTypeName, Foo_BarType)
	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			FooType,
			sema.PrimitiveAccess(ast.AccessAll),
			FooTypeFooFunctionName,
			FooTypeFooFunctionType,
			FooTypeFooFunctionDocString,
		),
		sema.NewUnmeteredFieldMember(
			FooType,
			sema.PrimitiveAccess(ast.AccessAll),
			ast.VariableKindConstant,
			FooTypeBarFieldName,
			FooTypeBarFieldType,
			FooTypeBarFieldDocString,
		),
	}

	FooType.Members = sema.MembersAsMap(members)
	FooType.Fields = sema.MembersFieldNames(members)
}
