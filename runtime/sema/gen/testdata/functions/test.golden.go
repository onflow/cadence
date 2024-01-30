// Code generated from testdata/functions/test.cdc. DO NOT EDIT.
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

package functions

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

const TestTypeNothingFunctionName = "nothing"

var TestTypeNothingFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeNothingFunctionDocString = `
This is a test function.
`

const TestTypeParamsFunctionName = "params"

var TestTypeParamsFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Identifier:     "a",
			TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "b",
			TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeParamsFunctionDocString = `
This is a test function with parameters.
`

const TestTypeReturnBoolFunctionName = "returnBool"

var TestTypeReturnBoolFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.BoolType,
	),
}

const TestTypeReturnBoolFunctionDocString = `
This is a test function with a return type.
`

const TestTypeParamsAndReturnFunctionName = "paramsAndReturn"

var TestTypeParamsAndReturnFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Identifier:     "a",
			TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
		},
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "b",
			TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.BoolType,
	),
}

const TestTypeParamsAndReturnFunctionDocString = `
This is a test function with parameters and a return type.
`

const TestTypeTypeParamFunctionName = "typeParam"

var TestTypeTypeParamFunctionTypeParameterT = &sema.TypeParameter{
	Name: "T",
}

var TestTypeTypeParamFunctionType = &sema.FunctionType{
	TypeParameters: []*sema.TypeParameter{
		TestTypeTypeParamFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeTypeParamFunctionDocString = `
This is a test function with a type parameter.
`

const TestTypeTypeParamWithBoundFunctionName = "typeParamWithBound"

var TestTypeTypeParamWithBoundFunctionTypeParameterT = &sema.TypeParameter{
	Name: "T",
	TypeBound: sema.SubtypeTypeBound{
		Type: &sema.ReferenceType{
			Type:          sema.AnyType,
			Authorization: sema.UnauthorizedAccess,
		},
	},
}

var TestTypeTypeParamWithBoundFunctionType = &sema.FunctionType{
	TypeParameters: []*sema.TypeParameter{
		TestTypeTypeParamWithBoundFunctionTypeParameterT,
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeTypeParamWithBoundFunctionDocString = `
This is a test function with a type parameter and a type bound.
`

const TestTypeTypeParamWithBoundAndParamFunctionName = "typeParamWithBoundAndParam"

var TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT = &sema.TypeParameter{
	Name: "T",
}

var TestTypeTypeParamWithBoundAndParamFunctionType = &sema.FunctionType{
	TypeParameters: []*sema.TypeParameter{
		TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT,
	},
	Parameters: []sema.Parameter{
		{
			Identifier: "t",
			TypeAnnotation: sema.NewTypeAnnotation(&sema.GenericType{
				TypeParameter: TestTypeTypeParamWithBoundAndParamFunctionTypeParameterT,
			}),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeTypeParamWithBoundAndParamFunctionDocString = `
This is a test function with a type parameter and a parameter using it.
`

const TestTypeViewFunctionFunctionName = "viewFunction"

var TestTypeViewFunctionFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

const TestTypeViewFunctionFunctionDocString = `
This is a function with 'view' modifier
`

const TestTypeName = "Test"

var TestType = &sema.SimpleType{
	Name:          TestTypeName,
	QualifiedName: TestTypeName,
	TypeID:        TestTypeName,
	TypeTag:       TestTypeTag,
	IsResource:    false,
	Storable:      false,
	Primitive:     false,
	Equatable:     false,
	Comparable:    false,
	Exportable:    false,
	Importable:    false,
	ContainFields: false,
}

func init() {
	TestType.Members = func(t *sema.SimpleType) map[string]sema.MemberResolver {
		return sema.MembersAsResolvers([]*sema.Member{
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeNothingFunctionName,
				TestTypeNothingFunctionType,
				TestTypeNothingFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeParamsFunctionName,
				TestTypeParamsFunctionType,
				TestTypeParamsFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeReturnBoolFunctionName,
				TestTypeReturnBoolFunctionType,
				TestTypeReturnBoolFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeParamsAndReturnFunctionName,
				TestTypeParamsAndReturnFunctionType,
				TestTypeParamsAndReturnFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeTypeParamFunctionName,
				TestTypeTypeParamFunctionType,
				TestTypeTypeParamFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeTypeParamWithBoundFunctionName,
				TestTypeTypeParamWithBoundFunctionType,
				TestTypeTypeParamWithBoundFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeTypeParamWithBoundAndParamFunctionName,
				TestTypeTypeParamWithBoundAndParamFunctionType,
				TestTypeTypeParamWithBoundAndParamFunctionDocString,
			),
			sema.NewUnmeteredFunctionMember(
				t,
				sema.PrimitiveAccess(ast.AccessAll),
				TestTypeViewFunctionFunctionName,
				TestTypeViewFunctionFunctionType,
				TestTypeViewFunctionFunctionDocString,
			),
		})
	}
}
