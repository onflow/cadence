// Code generated from testdata/functions.cdc. DO NOT EDIT.
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

const TestTypeNothingFunctionName = "nothing"

var TestTypeNothingFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeNothingFunctionDocString = `
This is a test function.
`

const TestTypeParamsFunctionName = "params"

var TestTypeParamsFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "a",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "b",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeParamsFunctionDocString = `
This is a test function with parameters.
`

const TestTypeReturnBoolFunctionName = "returnBool"

var TestTypeReturnBoolFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const TestTypeReturnBoolFunctionDocString = `
This is a test function with a return type.
`

const TestTypeParamsAndReturnFunctionName = "paramsAndReturn"

var TestTypeParamsAndReturnFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			Identifier:     "a",
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
		{
			Label:          ArgumentLabelNotRequired,
			Identifier:     "b",
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const TestTypeParamsAndReturnFunctionDocString = `
This is a test function with parameters and a return type.
`

const TestTypeTypeParamFunctionName = "typeParam"

var TestTypeTypeParamFunctionTypeParameterT = &TypeParameter{
	Name: "T",
}

var TestTypeTypeParamFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		TestTypeTypeParamFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeTypeParamFunctionDocString = `
This is a test function with a type parameter.
`

const TestTypeTypeParamWithBoundFunctionName = "typeParamWithBound"

var TestTypeTypeParamWithBoundFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type:          AnyType,
		Authorization: UnauthorizedAccess,
	},
}

var TestTypeTypeParamWithBoundFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		TestTypeTypeParamWithBoundFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeTypeParamWithBoundFunctionDocString = `
This is a test function with a type parameter and a type bound.
`

const TestTypeTypeParamWithBoundAndParamFunctionName = "typeParamWithBoundAndParam"

var TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT = &TypeParameter{
	Name: "T",
}

var TestTypeTypeParamWithBoundAndParamFunctionType = &FunctionType{
	TypeParameters: []*TypeParameter{
		TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT,
	},
	Parameters: []Parameter{
		{
			Identifier: "t",
			TypeAnnotation: NewTypeAnnotation(&GenericType{
				TypeParameter: TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeTypeParamWithBoundAndParamFunctionDocString = `
This is a test function with a type parameter and a parameter using it.
`

const TestTypeViewFunctionFunctionName = "viewFunction"

var TestTypeViewFunctionFunctionType = &FunctionType{
	Purity: FunctionPurityView,
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeViewFunctionFunctionDocString = `
This is a function with 'view' modifier
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
}

func init() {
	TestType.Members = func(t *SimpleType) map[string]MemberResolver {
		return MembersAsResolvers([]*Member{
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeNothingFunctionName,
				TestTypeNothingFunctionType,
				TestTypeNothingFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeParamsFunctionName,
				TestTypeParamsFunctionType,
				TestTypeParamsFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeReturnBoolFunctionName,
				TestTypeReturnBoolFunctionType,
				TestTypeReturnBoolFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeParamsAndReturnFunctionName,
				TestTypeParamsAndReturnFunctionType,
				TestTypeParamsAndReturnFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeTypeParamFunctionName,
				TestTypeTypeParamFunctionType,
				TestTypeTypeParamFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeTypeParamWithBoundFunctionName,
				TestTypeTypeParamWithBoundFunctionType,
				TestTypeTypeParamWithBoundFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeTypeParamWithBoundAndParamFunctionName,
				TestTypeTypeParamWithBoundAndParamFunctionType,
				TestTypeTypeParamWithBoundAndParamFunctionDocString,
			),
			NewUnmeteredPublicFunctionMember(
				t,
				TestTypeViewFunctionFunctionName,
				TestTypeViewFunctionFunctionType,
				TestTypeViewFunctionFunctionDocString,
			),
		})
	}
}
