// Code generated from testdata/fields/test.cdc. DO NOT EDIT.
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

package fields

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

const TestTypeTestIntFieldName = "testInt"

var TestTypeTestIntFieldType = sema.UInt64Type

const TestTypeTestIntFieldDocString = `
This is a test integer.
`

const TestTypeTestOptIntFieldName = "testOptInt"

var TestTypeTestOptIntFieldType = &sema.OptionalType{
	Type: sema.UInt64Type,
}

const TestTypeTestOptIntFieldDocString = `
This is a test optional integer.
`

const TestTypeTestRefIntFieldName = "testRefInt"

var TestTypeTestRefIntFieldType = &sema.ReferenceType{
	Type:          sema.UInt64Type,
	Authorization: sema.UnauthorizedAccess,
}

const TestTypeTestRefIntFieldDocString = `
This is a test integer reference.
`

const TestTypeTestVarIntsFieldName = "testVarInts"

var TestTypeTestVarIntsFieldType = &sema.VariableSizedType{
	Type: sema.UInt64Type,
}

const TestTypeTestVarIntsFieldDocString = `
This is a test variable-sized integer array.
`

const TestTypeTestConstIntsFieldName = "testConstInts"

var TestTypeTestConstIntsFieldType = &sema.ConstantSizedType{
	Type: sema.UInt64Type,
	Size: 2,
}

const TestTypeTestConstIntsFieldDocString = `
This is a test constant-sized integer array.
`

const TestTypeTestIntDictFieldName = "testIntDict"

var TestTypeTestIntDictFieldType = &sema.DictionaryType{
	KeyType:   sema.UInt64Type,
	ValueType: sema.BoolType,
}

const TestTypeTestIntDictFieldDocString = `
This is a test integer dictionary.
`

const TestTypeTestParamFieldName = "testParam"

var TestTypeTestParamFieldType = sema.MustInstantiate(
	FooType,
	BarType,
)

const TestTypeTestParamFieldDocString = `
This is a test parameterized-type field.
`

const TestTypeTestAddressFieldName = "testAddress"

var TestTypeTestAddressFieldType = sema.TheAddressType

const TestTypeTestAddressFieldDocString = `
This is a test address field.
`

const TestTypeTestTypeFieldName = "testType"

var TestTypeTestTypeFieldType = sema.MetaType

const TestTypeTestTypeFieldDocString = `
This is a test type field.
`

const TestTypeTestCapFieldName = "testCap"

var TestTypeTestCapFieldType = &sema.CapabilityType{}

const TestTypeTestCapFieldDocString = `
This is a test unparameterized capability field.
`

const TestTypeTestCapIntFieldName = "testCapInt"

var TestTypeTestCapIntFieldType = sema.MustInstantiate(
	&sema.CapabilityType{},
	sema.IntType,
)

const TestTypeTestCapIntFieldDocString = `
This is a test parameterized capability field.
`

const TestTypeTestIntersectionWithoutTypeFieldName = "testIntersectionWithoutType"

var TestTypeTestIntersectionWithoutTypeFieldType = &sema.IntersectionType{
	Types: []*sema.InterfaceType{BarType, BazType},
}

const TestTypeTestIntersectionWithoutTypeFieldDocString = `
This is a test intersection type (without type) field.
`

const TestTypeName = "Test"

var TestType = &sema.SimpleType{
	Name:          TestTypeName,
	QualifiedName: TestTypeName,
	TypeID:        TestTypeName,
	TypeTag:       TestTypeTag,
	IsResource:    false,
	IsPrimitive:   false,
	Storable:      false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
	ContainFields: false,
}

func init() {
	TestType.Members = func(t *sema.SimpleType) map[string]sema.MemberResolver {
		return sema.MembersAsResolvers([]*sema.Member{
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestIntFieldName,
				TestTypeTestIntFieldType,
				TestTypeTestIntFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestOptIntFieldName,
				TestTypeTestOptIntFieldType,
				TestTypeTestOptIntFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestRefIntFieldName,
				TestTypeTestRefIntFieldType,
				TestTypeTestRefIntFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestVarIntsFieldName,
				TestTypeTestVarIntsFieldType,
				TestTypeTestVarIntsFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestConstIntsFieldName,
				TestTypeTestConstIntsFieldType,
				TestTypeTestConstIntsFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestIntDictFieldName,
				TestTypeTestIntDictFieldType,
				TestTypeTestIntDictFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestParamFieldName,
				TestTypeTestParamFieldType,
				TestTypeTestParamFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestAddressFieldName,
				TestTypeTestAddressFieldType,
				TestTypeTestAddressFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestTypeFieldName,
				TestTypeTestTypeFieldType,
				TestTypeTestTypeFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestCapFieldName,
				TestTypeTestCapFieldType,
				TestTypeTestCapFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestCapIntFieldName,
				TestTypeTestCapIntFieldType,
				TestTypeTestCapIntFieldDocString,
			),
			sema.NewUnmeteredFieldMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				ast.VariableKindConstant,
				TestTypeTestIntersectionWithoutTypeFieldName,
				TestTypeTestIntersectionWithoutTypeFieldType,
				TestTypeTestIntersectionWithoutTypeFieldDocString,
			),
		})
	}
}
