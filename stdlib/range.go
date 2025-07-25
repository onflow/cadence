/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
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
		Purity: sema.FunctionPurityView,
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
		TypeArgumentsCheck: func(
			memoryGauge common.MemoryGauge,
			typeArguments *sema.TypeParameterTypeOrderedMap,
			astTypeArguments []*ast.TypeAnnotation,
			invocationRange ast.HasPosition,
			report func(error),
		) {
			memberType, ok := typeArguments.Get(typeParameter)
			if !ok || memberType == nil {
				// Invalid, already reported by checker
				return
			}

			// memberType must only be a leaf integer type.
			for _, ty := range sema.AllNonLeafIntegerTypes {
				if memberType != ty {
					continue
				}

				// If type argument was provided, use its range otherwise fallback to invocation range.
				errorRange := invocationRange
				if len(astTypeArguments) > 0 {
					errorRange = astTypeArguments[0]
				}

				report(&sema.InvalidTypeArgumentError{
					TypeArgumentName: typeParameter.Name,
					Range:            ast.NewRangeFromPositioned(memoryGauge, errorRange),
					Details:          fmt.Sprintf("Creation of InclusiveRange<%s> is disallowed", memberType),
				})

				break
			}
		},
	}
}()

var InterpreterInclusiveRangeConstructor = NewInterpreterStandardLibraryStaticFunction(
	"InclusiveRange",
	inclusiveRangeConstructorFunctionType,
	inclusiveRangeConstructorFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		invocationContext := invocation.InvocationContext
		locationRange := invocation.LocationRange

		start, ok := invocation.Arguments[0].(interpreter.IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		end, ok := invocation.Arguments[1].(interpreter.IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		var step interpreter.IntegerValue
		if len(invocation.Arguments) > 2 {
			step, ok = invocation.Arguments[2].(interpreter.IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
		}

		return NewInclusiveRange(
			invocationContext,
			locationRange,
			start,
			end,
			step,
		)
	},
)

var VMInclusiveRangeConstructor = NewVMStandardLibraryStaticFunction(
	"InclusiveRange",
	inclusiveRangeConstructorFunctionType,
	inclusiveRangeConstructorFunctionDocString,
	func(context *vm.Context, typeArguments []bbq.StaticType, _ vm.Value, arguments ...vm.Value) vm.Value {

		start, ok := arguments[0].(interpreter.IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		end, ok := arguments[1].(interpreter.IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		var step interpreter.IntegerValue
		if len(arguments) > 2 {
			step, ok = arguments[2].(interpreter.IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
		}

		return NewInclusiveRange(
			context,
			interpreter.EmptyLocationRange,
			start,
			end,
			step,
		)
	},
)

func NewInclusiveRange(
	invocationContext interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
	start interpreter.IntegerValue,
	end interpreter.IntegerValue,
	step interpreter.IntegerValue,
) interpreter.Value {

	startStaticType := start.StaticType(invocationContext)
	endStaticType := end.StaticType(invocationContext)
	if !startStaticType.Equal(endStaticType) {
		panic(&interpreter.InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message: fmt.Sprintf(
				"start and end are of different types. start: %s and end: %s",
				startStaticType,
				endStaticType,
			),
		})
	}

	rangeStaticType := interpreter.NewInclusiveRangeStaticType(invocationContext, startStaticType)
	rangeSemaType := interpreter.MustConvertStaticToSemaType(
		rangeStaticType,
		invocationContext,
	).(*sema.InclusiveRangeType)

	if step != nil {

		stepStaticType := step.StaticType(invocationContext)
		if stepStaticType != startStaticType {
			panic(&interpreter.InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message: fmt.Sprintf(
					"step must be of the same type as start and end. start/end: %s and step: %s",
					startStaticType,
					stepStaticType,
				),
			})
		}

		return interpreter.NewInclusiveRangeValueWithStep(
			invocationContext,
			locationRange,
			start,
			end,
			step,
			rangeStaticType,
			rangeSemaType,
		)
	}

	return interpreter.NewInclusiveRangeValue(
		invocationContext,
		locationRange,
		start,
		end,
		rangeStaticType,
		rangeSemaType,
	)
}
