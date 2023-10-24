// Code generated from testdata/constructor/test.cdc. DO NOT EDIT.
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

package constructor

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

var FooTypeConstructorType = &sema.FunctionType{
	IsConstructor: true,
	Parameters: []sema.Parameter{
		{
			Identifier:     "bar",
			TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		FooType,
	),
}

const FooTypeConstructorDocString = `
Constructs a new Foo
`

const FooTypeName = "Foo"

var FooType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         FooTypeName,
		Kind:               common.CompositeKindStructure,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{}

	FooType.Members = sema.MembersAsMap(members)
	FooType.Fields = sema.MembersFieldNames(members)
	FooType.ConstructorParameters = FooTypeConstructorType.Parameters
}
