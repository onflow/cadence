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

var InclusiveRangeConstructorFunctionType = func() *sema.FunctionType {

	typeAnnotation := sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: sema.InclusiveRangeConstructorFunctionTypeParameter,
		},
	)

	return &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			sema.InclusiveRangeConstructorFunctionTypeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "start",
				TypeAnnotation: typeAnnotation,
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "end",
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
		// `step` parameter is optional
		Arity: &sema.Arity{Min: 2, Max: 3},
	}
}()

var InclusiveRangeConstructorFunction = NewStandardLibraryFunction(
	"InclusiveRange",
	InclusiveRangeConstructorFunctionType,
	inclusiveRangeConstructorFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		start, startOk := invocation.Arguments[0].(interpreter.IntegerValue)
		end, endOk := invocation.Arguments[1].(interpreter.IntegerValue)

		if !startOk || !endOk {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter
		locationRange := invocation.LocationRange

		startStaticType := start.StaticType(inter)
		endStaticType := end.StaticType(inter)
		if !startStaticType.Equal(endStaticType) {
			panic(interpreter.InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message: fmt.Sprintf(
					"start and end are of different types. start: %s and end: %s",
					startStaticType,
					endStaticType,
				),
			})
		}

		rangeStaticType := interpreter.NewInclusiveRangeStaticType(invocation.Interpreter, startStaticType)
		rangeSemaType := sema.NewInclusiveRangeType(invocation.Interpreter, invocation.ArgumentTypes[0])

		if len(invocation.Arguments) > 2 {
			step, ok := invocation.Arguments[2].(interpreter.IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			stepStaticType := step.StaticType(inter)
			if stepStaticType != startStaticType {
				panic(interpreter.InclusiveRangeConstructionError{
					LocationRange: locationRange,
					Message: fmt.Sprintf(
						"step must be of the same type as start and end. start/end: %s and step: %s",
						startStaticType,
						stepStaticType,
					),
				})
			}

			return interpreter.NewInclusiveRangeValueWithStep(
				inter,
				locationRange,
				start,
				end,
				step,
				rangeStaticType,
				rangeSemaType,
			)
		}

		return interpreter.NewInclusiveRangeValue(
			inter,
			locationRange,
			start,
			end,
			rangeStaticType,
			rangeSemaType,
		)
	},
)
