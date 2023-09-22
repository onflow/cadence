// Code generated from testdata/contract/test.cdc. DO NOT EDIT.
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

package contract

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

var Test_FooTypeConstructorType = &sema.FunctionType{
	IsConstructor: true,
	Parameters: []sema.Parameter{
		{
			Identifier:     "bar",
			TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		Test_FooType,
	),
}

const Test_FooTypeConstructorDocString = `
Constructs a new Foo
`

const Test_FooTypeName = "Foo"

var Test_FooType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         Test_FooTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{}

	Test_FooType.Members = sema.MembersAsMap(members)
	Test_FooType.Fields = sema.MembersFieldNames(members)
	Test_FooType.ConstructorParameters = Test_FooTypeConstructorType.Parameters
}

const TestTypeName = "Test"

var TestType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         TestTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	t.SetNestedType(Test_FooTypeName, Test_FooType)
	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredConstructorMember(
			TestType,
			sema.PrimitiveAccess(ast.AccessAll),
			Test_FooTypeName,
			Test_FooTypeConstructorType,
			Test_FooTypeConstructorDocString,
		),
	}

	TestType.Members = sema.MembersAsMap(members)
	TestType.Fields = sema.MembersFieldNames(members)
}
