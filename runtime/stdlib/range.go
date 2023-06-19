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
	"fmt"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// InclusiveRangeConstructorFunction

const inclusiveRangeConstructorFunctionDocString = `
 Constructs a Range covering from start to end.
 
 The step argument is optional and determines the step size. 
 If not provided, the value of +1 or -1 is used based on the values of start and end. 
 `

var inclusiveRangeConstructorFunctionType = func() *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		Name:      "T",
		TypeBound: sema.IntegerType,
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
				Identifier:     "start",
				TypeAnnotation: typeAnnotation,
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "endInclusive",
				TypeAnnotation: typeAnnotation,
			},
			{
				Identifier:     "step",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.InclusiveRangeType{
				MemberType: typeAnnotation.Type,
			},
		),
		RequiredArgumentCount: sema.RequiredArgumentCount(2),
	}
}()

var InclusiveRangeConstructorFunction = NewStandardLibraryFunction(
	"InclusiveRange",
	inclusiveRangeConstructorFunctionType,
	inclusiveRangeConstructorFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		start, startOk := invocation.Arguments[0].(interpreter.IntegerValue)
		endInclusive, endInclusiveOk := invocation.Arguments[1].(interpreter.IntegerValue)

		if !startOk || !endInclusiveOk {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter
		locationRange := invocation.LocationRange

		leftStaticType := start.StaticType(inter)
		rightStaticType := endInclusive.StaticType(inter)
		if leftStaticType != rightStaticType {
			// Checker would only allow same type for both start & endInclusive.
			panic(errors.NewUnreachableError())
		}

		rangeStaticType := interpreter.InclusiveRangeStaticType{ElementType: leftStaticType}

		if len(invocation.Arguments) > 2 {
			step, ok := invocation.Arguments[2].(interpreter.IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return interpreter.NewInclusiveRangeValueWithStep(inter, locationRange, start, endInclusive, step, rangeStaticType)
		} else {
			return interpreter.NewInclusiveRangeValue(inter, locationRange, start, endInclusive, rangeStaticType)
		}
	},
)

// InclusiveRangeConstructionError

type InclusiveRangeConstructionError struct {
	interpreter.LocationRange
	Message string
}

var _ errors.UserError = InclusiveRangeConstructionError{}

func (InclusiveRangeConstructionError) IsUserError() {}

func (e InclusiveRangeConstructionError) Error() string {
	const message = "InclusiveRange construction failed"
	if e.Message == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", message, e.Message)
}
