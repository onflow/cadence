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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end with default value of step.
func NewInclusiveRangeValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	rangeType InclusiveRangeStaticType,
) *CompositeValue {
	startComparable, startOk := start.(ComparableValue)
	endInclusiveComparable, endInclusiveOk := end.(ComparableValue)
	if !startOk || !endInclusiveOk {
		panic(errors.NewUnreachableError())
	}

	step := interpreter.GetValueForIntegerType(1, rangeType.ElementType)
	if startComparable.Greater(interpreter, endInclusiveComparable, locationRange) {
		elemSemaTy := interpreter.MustConvertStaticToSemaType(rangeType.ElementType)
		if _, ok := sema.AllUnsignedIntegerTypesSet[elemSemaTy]; ok {
			panic(InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message:       fmt.Sprintf("step value cannot be negative for unsigned integer type %s", elemSemaTy),
			})
		}

		negatedStep, ok := step.Negate(interpreter, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		step = negatedStep
	}

	return NewInclusiveRangeValueWithStep(interpreter, locationRange, start, end, step, rangeType)
}

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end & step.
func NewInclusiveRangeValueWithStep(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
) *CompositeValue {

	// Validate that the step is non-zero.
	if step.Equal(interpreter, locationRange, interpreter.GetValueForIntegerType(0, rangeType.ElementType)) {
		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message:       "step value cannot be zero",
		})
	}

	// Validate that the sequence is moving towards the end value.
	// If start < end, step must be > 0
	// If start > end, step must be < 0
	// If start == end, step doesn't matter.
	if (start.Less(interpreter, end, locationRange) && step.Less(interpreter, interpreter.GetValueForIntegerType(0, rangeType.ElementType), locationRange)) ||
		(start.Greater(interpreter, end, locationRange) && step.Greater(interpreter, interpreter.GetValueForIntegerType(0, rangeType.ElementType), locationRange)) {

		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message: fmt.Sprintf(
				"sequence is moving away from end: %s due to the value of step: %s and start: %s",
				end,
				step,
				start,
			),
		})
	}

	fields := []CompositeField{
		{
			Name:  sema.InclusiveRangeTypeStartFieldName,
			Value: start,
		},
		{
			Name:  sema.InclusiveRangeTypeEndFieldName,
			Value: end,
		},
		{
			Name:  sema.InclusiveRangeTypeStepFieldName,
			Value: step,
		},
	}

	rangeSemaType := getInclusiveRangeSemaType(interpreter, rangeType)

	rangeValue := NewCompositeValueWithStaticType(
		interpreter,
		locationRange,
		nil,
		rangeSemaType.QualifiedString(),
		common.CompositeKindStructure,
		fields,
		common.ZeroAddress,
		rangeType,
	)

	rangeValue.Functions = map[string]FunctionValue{
		sema.InclusiveRangeTypeContainsFunctionName: NewHostFunctionValue(
			interpreter,
			sema.InclusiveRangeContainsFunctionType(
				rangeSemaType.MemberType,
			),
			func(invocation Invocation) Value {
				needleInteger := convertAndAssertIntegerValue(invocation.Arguments[0])

				return rangeContains(
					rangeValue,
					rangeType,
					invocation.Interpreter,
					invocation.LocationRange,
					needleInteger,
				)
			},
		),
	}

	return rangeValue
}

func getInclusiveRangeSemaType(interpreter *Interpreter, rangeType InclusiveRangeStaticType) *sema.InclusiveRangeType {
	return interpreter.MustConvertStaticToSemaType(rangeType).(*sema.InclusiveRangeType)
}

func rangeContains(
	rangeValue *CompositeValue,
	rangeType InclusiveRangeStaticType,
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue IntegerValue,
) BoolValue {
	start := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeStartFieldName)
	endInclusive := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeEndFieldName)
	step := getFieldAsIntegerValue(rangeValue, interpreter, locationRange, sema.InclusiveRangeTypeStepFieldName)

	result := start.Equal(interpreter, locationRange, needleValue) ||
		endInclusive.Equal(interpreter, locationRange, needleValue)

	if !result {
		greaterThanStart := needleValue.Greater(interpreter, start, locationRange)
		greaterThanEndInclusive := needleValue.Greater(interpreter, endInclusive, locationRange)

		if greaterThanStart == greaterThanEndInclusive {
			// If needle is greater or smaller than both start & endInclusive, then it is outside the range.
			result = false
		} else {
			// needle is in between start and endInclusive.
			// start + k * step should be equal to needle i.e. (needle - start) mod step == 0.
			diff, ok := needleValue.Minus(interpreter, start, locationRange).(IntegerValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			zeroValue := interpreter.GetValueForIntegerType(0, rangeType.ElementType)
			result = diff.Mod(interpreter, step, locationRange).Equal(interpreter, locationRange, zeroValue)
		}
	}

	return AsBoolValue(result)
}

func getFieldAsIntegerValue(
	rangeValue *CompositeValue,
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) IntegerValue {
	return convertAndAssertIntegerValue(
		rangeValue.GetField(
			interpreter,
			locationRange,
			name,
		),
	)
}

func convertAndAssertIntegerValue(value Value) IntegerValue {
	integerValue, ok := value.(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return integerValue
}
