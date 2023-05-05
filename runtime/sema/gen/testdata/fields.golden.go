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

const TestTypeTestIntFieldName = "testInt"

var TestTypeTestIntFieldType = UInt64Type

const TestTypeTestIntFieldDocString = `
This is a test integer.
`

const TestTypeTestOptIntFieldName = "testOptInt"

var TestTypeTestOptIntFieldType = &OptionalType{
	Type: UInt64Type,
}

const TestTypeTestOptIntFieldDocString = `
This is a test optional integer.
`

const TestTypeTestRefIntFieldName = "testRefInt"

var TestTypeTestRefIntFieldType = &ReferenceType{
	Type:          UInt64Type,
	Authorization: UnauthorizedAccess,
}

const TestTypeTestRefIntFieldDocString = `
This is a test integer reference.
`

const TestTypeTestVarIntsFieldName = "testVarInts"

var TestTypeTestVarIntsFieldType = &VariableSizedType{
	Type: UInt64Type,
}

const TestTypeTestVarIntsFieldDocString = `
This is a test variable-sized integer array.
`

const TestTypeTestConstIntsFieldName = "testConstInts"

var TestTypeTestConstIntsFieldType = &ConstantSizedType{
	Type: UInt64Type,
	Size: 2,
}

const TestTypeTestConstIntsFieldDocString = `
This is a test constant-sized integer array.
`

const TestTypeTestParamFieldName = "testParam"

var TestTypeTestParamFieldType = MustInstantiate(
	FooType,
	BarType,
)

const TestTypeTestParamFieldDocString = `
This is a test parameterized-type field.
`

const TestTypeTestAddressFieldName = "testAddress"

var TestTypeTestAddressFieldType = TheAddressType

const TestTypeTestAddressFieldDocString = `
This is a test address field.
`

const TestTypeTestTypeFieldName = "testType"

var TestTypeTestTypeFieldType = MetaType

const TestTypeTestTypeFieldDocString = `
This is a test type field.
`

const TestTypeTestCapFieldName = "testCap"

var TestTypeTestCapFieldType = &CapabilityType{}

const TestTypeTestCapFieldDocString = `
This is a test capability field.
`

const TestTypeTestCapIntFieldName = "testCapInt"

var TestTypeTestCapIntFieldType = MustInstantiate(
	&CapabilityType{},
	IntType,
)

const TestTypeTestCapIntFieldDocString = `
This is a test specific capability field.
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
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
}

func init() {
	TestType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestIntFieldName,
				TestTypeTestIntFieldType,
				TestTypeTestIntFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestOptIntFieldName,
				TestTypeTestOptIntFieldType,
				TestTypeTestOptIntFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestRefIntFieldName,
				TestTypeTestRefIntFieldType,
				TestTypeTestRefIntFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestVarIntsFieldName,
				TestTypeTestVarIntsFieldType,
				TestTypeTestVarIntsFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestConstIntsFieldName,
				TestTypeTestConstIntsFieldType,
				TestTypeTestConstIntsFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestParamFieldName,
				TestTypeTestParamFieldType,
				TestTypeTestParamFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestAddressFieldName,
				TestTypeTestAddressFieldType,
				TestTypeTestAddressFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestTypeFieldName,
				TestTypeTestTypeFieldType,
				TestTypeTestTypeFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestCapFieldName,
				TestTypeTestCapFieldType,
				TestTypeTestCapFieldDocString,
			),
			NewUnmeteredPublicConstantFieldMember(
				t,
				TestTypeTestCapIntFieldName,
				TestTypeTestCapIntFieldType,
				TestTypeTestCapIntFieldDocString,
			),
		})
	}
}
