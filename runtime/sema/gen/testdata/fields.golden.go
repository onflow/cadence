// Code generated from testdata/fields.cdc. DO NOT EDIT.
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

const TestTypeTestIntFieldName = "testInt"

var TestTypeTestIntFieldType = UInt64Type

const TestTypeTestIntFieldDocString = `This is a test integer.
`

const TestTypeTestOptIntFieldName = "testOptInt"

var TestTypeTestOptIntFieldType = &OptionalType{
	Type: UInt64Type,
}

const TestTypeTestOptIntFieldDocString = `This is a test optional integer.
`

const TestTypeTestRefIntFieldName = "testRefInt"

var TestTypeTestRefIntFieldType = &ReferenceType{
	Type: UInt64Type,
}

const TestTypeTestRefIntFieldDocString = `This is a test integer reference.
`

const TestTypeTestVarIntsFieldName = "testVarInts"

var TestTypeTestVarIntsFieldType = &VariableSizedType{
	Type: UInt64Type,
}

const TestTypeTestVarIntsFieldDocString = `This is a test variable-sized integer array.
`

const TestTypeTestConstIntsFieldName = "testConstInts"

var TestTypeTestConstIntsFieldType = &ConstantSizedType{
	Type: UInt64Type,
	Size: 2,
}

const TestTypeTestConstIntsFieldDocString = `This is a test constant-sized integer array.
`

const TestTypeTestParamFieldName = "testParam"

var TestTypeTestParamFieldType = FooType.Instantiate([]Type{BarType}, panicUnexpected)

const TestTypeTestParamFieldDocString = `This is a test parameterized-type field.
`

const TestTypeName = "Test"

var TestType = &SimpleType{
	Name:          TestTypeName,
	QualifiedName: TestTypeName,
	TypeID:        TestTypeName,
	tag:           TestTypeTag,
	IsResource:    false,
	Storable:      false,
	Equatable:     false,
	Exportable:    false,
	Importable:    false,
	Members: func(t *SimpleType) map[string]MemberResolver {
		return map[string]MemberResolver{
			TestTypeTestIntFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestIntFieldType,
						TestTypeTestIntFieldDocString,
					)
				},
			},
			TestTypeTestOptIntFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestOptIntFieldType,
						TestTypeTestOptIntFieldDocString,
					)
				},
			},
			TestTypeTestRefIntFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestRefIntFieldType,
						TestTypeTestRefIntFieldDocString,
					)
				},
			},
			TestTypeTestVarIntsFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestVarIntsFieldType,
						TestTypeTestVarIntsFieldDocString,
					)
				},
			},
			TestTypeTestConstIntsFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestConstIntsFieldType,
						TestTypeTestConstIntsFieldDocString,
					)
				},
			},
			TestTypeTestParamFieldName: {
				Kind: common.DeclarationKindField,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicConstantFieldMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTestParamFieldType,
						TestTypeTestParamFieldDocString,
					)
				},
			},
		}
	},
}
