// Code generated from testdata/nested.cdc. DO NOT EDIT.
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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const FooTypeFooFunctionName = "foo"

var FooTypeFooFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const FooTypeFooFunctionDocString = `
foo
`

const FooTypeBarFieldName = "bar"

var FooTypeBarFieldType = FooBarType

const FooTypeBarFieldDocString = `
Bar
`

const FooBarTypeBarFunctionName = "bar"

var FooBarTypeBarFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const FooBarTypeBarFunctionDocString = `
bar
`

const FooBarTypeName = "Bar"

var FooBarType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         FooBarTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			FooBarType,
			ast.AccessAll,
			FooBarTypeBarFunctionName,
			FooBarTypeBarFunctionType,
			FooBarTypeBarFunctionDocString,
		),
	}

	FooBarType.Members = MembersAsMap(members)
	FooBarType.Fields = MembersFieldNames(members)
}

const FooTypeName = "Foo"

var FooType = func() *CompositeType {
	var t = &CompositeType{
		Identifier:         FooTypeName,
		Kind:               common.CompositeKindStructure,
		importable:         false,
		hasComputedMembers: true,
	}

	t.SetNestedType(FooBarTypeName, FooBarType)
	return t
}()

func init() {
	var members = []*Member{
		NewUnmeteredFunctionMember(
			FooType,
			ast.AccessAll,
			FooTypeFooFunctionName,
			FooTypeFooFunctionType,
			FooTypeFooFunctionDocString,
		),
		NewUnmeteredFieldMember(
			FooType,
			ast.AccessAll,
			ast.VariableKindConstant,
			FooTypeBarFieldName,
			FooTypeBarFieldType,
			FooTypeBarFieldDocString,
		),
	}

	FooType.Members = MembersAsMap(members)
	FooType.Fields = MembersFieldNames(members)
}
