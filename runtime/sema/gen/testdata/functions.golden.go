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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

const TestTypeNothingFunctionName = "nothing"

var TestTypeNothingFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeNothingFunctionDocString = `This is a test function.
`

const TestTypeParamsFunctionName = "params"

var TestTypeParamsFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
		{
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeParamsFunctionDocString = `This is a test function with parameters.
`

const TestTypeReturnFunctionName = "return"

var TestTypeReturnFunctionType = &FunctionType{
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const TestTypeReturnFunctionDocString = `This is a test function with a return type.
`

const TestTypeParamsAndReturnFunctionName = "paramsAndReturn"

var TestTypeParamsAndReturnFunctionType = &FunctionType{
	Parameters: []Parameter{
		{
			TypeAnnotation: NewTypeAnnotation(IntType),
		},
		{
			TypeAnnotation: NewTypeAnnotation(StringType),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		BoolType,
	),
}

const TestTypeParamsAndReturnFunctionDocString = `This is a test function with parameters and a return type.
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

const TestTypeTypeParamFunctionDocString = `This is a test function with a type parameter.
`

const TestTypeTypeParamWithBoundFunctionName = "typeParamWithBound"

var TestTypeTypeParamWithBoundFunctionTypeParameterT = &TypeParameter{
	Name: "T",
	TypeBound: &ReferenceType{
		Type: AnyType,
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

const TestTypeTypeParamWithBoundFunctionDocString = `This is a test function with a type parameter and a type bound.
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
			TypeAnnotation: NewTypeAnnotation(&GenericType{
				TypeParameter: TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT,
			}),
		},
	},
	ReturnTypeAnnotation: NewTypeAnnotation(
		VoidType,
	),
}

const TestTypeTypeParamWithBoundAndParamFunctionDocString = `This is a test function with a type parameter and a parameter using it.
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
			TestTypeNothingFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeNothingFunctionType,
						TestTypeNothingFunctionDocString,
					)
				},
			},
			TestTypeParamsFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeParamsFunctionType,
						TestTypeParamsFunctionDocString,
					)
				},
			},
			TestTypeReturnFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeReturnFunctionType,
						TestTypeReturnFunctionDocString,
					)
				},
			},
			TestTypeParamsAndReturnFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeParamsAndReturnFunctionType,
						TestTypeParamsAndReturnFunctionDocString,
					)
				},
			},
			TestTypeTypeParamFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTypeParamFunctionType,
						TestTypeTypeParamFunctionDocString,
					)
				},
			},
			TestTypeTypeParamWithBoundFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTypeParamWithBoundFunctionType,
						TestTypeTypeParamWithBoundFunctionDocString,
					)
				},
			},
			TestTypeTypeParamWithBoundAndParamFunctionName: {
				Kind: common.DeclarationKindFunction,
				Resolve: func(memoryGauge common.MemoryGauge,
					identifier string,
					targetRange ast.Range,
					report func(error)) *Member {

					return NewPublicFunctionMember(
						memoryGauge,
						t,
						identifier,
						TestTypeTypeParamWithBoundAndParamFunctionType,
						TestTypeTypeParamWithBoundAndParamFunctionDocString,
					)
				},
			},
		}
	},
}
