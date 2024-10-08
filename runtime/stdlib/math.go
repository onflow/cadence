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

package stdlib

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// SqrtFunction

const mathTypeSqrtFunctionDocString = `
Computes the square root of the value and returns it.
Available on all Number types.
Panics with error if the provided value is < 0.
`

const mathTypeSqrtFunctionName = "Sqrt"

var mathTypeSqrtFunctionType = func() *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		Name:      "T",
		TypeBound: sema.NumberType,
	}

	typeAnnotation := sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.UFix64Type),
	}
}()

var mathSqrtFunction = interpreter.NewUnmeteredHostFunctionValue(
	mathTypeSqrtFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		value, ok := invocation.Arguments[0].(interpreter.NumberValue)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		return value.Sqrt(invocation.Interpreter, invocation.LocationRange)
	},
)

// Math Contract

const MathTypeName = "Math"

var MathType = func() *sema.CompositeType {
	var t = &sema.CompositeType{
		Identifier:         MathTypeName,
		Kind:               common.CompositeKindContract,
		ImportableBuiltin:  false,
		HasComputedMembers: true,
	}

	return t
}()

func init() {
	var members = []*sema.Member{
		sema.NewUnmeteredFunctionMember(
			MathType,
			sema.PrimitiveAccess(ast.AccessAll),
			mathTypeSqrtFunctionName,
			mathTypeSqrtFunctionType,
			mathTypeSqrtFunctionDocString,
		),
	}

	MathType.Members = sema.MembersAsMap(members)
	MathType.Fields = sema.MembersFieldNames(members)
}

var mathContractFields = map[string]interpreter.Value{
	mathTypeSqrtFunctionName: mathSqrtFunction,
}

var MathTypeStaticType = interpreter.ConvertSemaToStaticType(nil, MathType)

var mathContractValue = interpreter.NewSimpleCompositeValue(
	nil,
	MathType.ID(),
	MathTypeStaticType,
	nil,
	mathContractFields,
	nil,
	nil,
	nil,
)

var MathContract = StandardLibraryValue{
	Name:  MathTypeName,
	Type:  MathType,
	Value: mathContractValue,
	Kind:  common.DeclarationKindContract,
}
