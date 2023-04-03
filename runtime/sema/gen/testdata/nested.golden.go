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

import "github.com/onflow/cadence/runtime/common"

const FooTypeFooFunctionName = "foo"

var FooTypeFooFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const FooTypeFooFunctionDocString = `
foo
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
