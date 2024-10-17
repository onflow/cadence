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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end with default value of step.
// NOTE: Assumes that the values start and end are of the same static type.
func NewInclusiveRangeValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	rangeStaticType InclusiveRangeStaticType,
	rangeSemaType *sema.InclusiveRangeType,
) *CompositeValue {
	startComparable, startOk := start.(ComparableValue)
	endComparable, endOk := end.(ComparableValue)
	if !startOk || !endOk {
		panic(errors.NewUnreachableError())
	}

	step := GetSmallIntegerValue(1, rangeStaticType.ElementType)
	if startComparable.Greater(interpreter, endComparable, locationRange) {
		elemSemaTy := interpreter.MustConvertStaticToSemaType(rangeStaticType.ElementType)
		if elemSemaTy.Tag().BelongsTo(sema.UnsignedIntegerTypeTag) {
			panic(InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message: fmt.Sprintf(
					"step value cannot be negative for unsigned integer type %s",
					elemSemaTy,
				),
			})
		}

		negatedStep, ok := step.Negate(interpreter, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		step = negatedStep
	}

	return createInclusiveRange(
		interpreter,
		locationRange,
		start,
		end,
		step,
		rangeStaticType,
		rangeSemaType,
	)
}

// NewInclusiveRangeValue constructs an InclusiveRange value with the provided start, end & step.
// NOTE: Assumes that the values start, end and step are of the same static type.
func NewInclusiveRangeValueWithStep(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
	rangeSemaType *sema.InclusiveRangeType,
) *CompositeValue {

	zeroValue := GetSmallIntegerValue(0, start.StaticType(interpreter))

	// Validate that the step is non-zero.
	if step.Equal(interpreter, locationRange, zeroValue) {
		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message:       "step value cannot be zero",
		})
	}

	// Validate that the sequence is moving towards the end value.
	// If start < end, step must be > 0
	// If start > end, step must be < 0
	// If start == end, step doesn't matter.
	if isSequenceMovingAwayFromEnd(interpreter, locationRange, start, end, step, zeroValue) {

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

	return createInclusiveRange(
		interpreter,
		locationRange,
		start,
		end,
		step,
		rangeType,
		rangeSemaType,
	)
}

func createInclusiveRange(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
	rangeSemaType *sema.InclusiveRangeType,
) *CompositeValue {
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

	rangeValue.Functions = orderedmap.New[FunctionOrderedMap](1)

	rangeValue.Functions.Set(
		sema.InclusiveRangeTypeContainsFunctionName,
		NewBoundHostFunctionValue(
			interpreter,
			rangeValue,
			sema.InclusiveRangeContainsFunctionType(
				rangeSemaType.MemberType,
			),
			func(rangeValue *CompositeValue, invocation Invocation) Value {
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
	)

	return rangeValue
}

func rangeContains(
	rangeValue *CompositeValue,
	rangeType InclusiveRangeStaticType,
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue IntegerValue,
) BoolValue {
	start := getFieldAsIntegerValue(interpreter, rangeValue, locationRange, sema.InclusiveRangeTypeStartFieldName)
	end := getFieldAsIntegerValue(interpreter, rangeValue, locationRange, sema.InclusiveRangeTypeEndFieldName)
	step := getFieldAsIntegerValue(interpreter, rangeValue, locationRange, sema.InclusiveRangeTypeStepFieldName)

	result := start.Equal(interpreter, locationRange, needleValue) ||
		end.Equal(interpreter, locationRange, needleValue)

	if result {
		return TrueValue
	}

	// Exclusive check since we already checked for boundaries above.
	if !isNeedleBetweenStartEndExclusive(interpreter, locationRange, needleValue, start, end) {
		result = false
	} else {
		// needle is in between start and end.
		// start + k * step should be equal to needle i.e. (needle - start) mod step == 0.
		diff, ok := needleValue.Minus(interpreter, start, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		zeroValue := GetSmallIntegerValue(0, rangeType.ElementType)
		mod := diff.Mod(interpreter, step, locationRange)
		result = mod.Equal(interpreter, locationRange, zeroValue)
	}

	return AsBoolValue(result)
}

func getFieldAsIntegerValue(
	interpreter *Interpreter,
	rangeValue *CompositeValue,
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

func isNeedleBetweenStartEndExclusive(
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue IntegerValue,
	start IntegerValue,
	end IntegerValue,
) bool {
	greaterThanStart := needleValue.Greater(interpreter, start, locationRange)
	greaterThanEnd := needleValue.Greater(interpreter, end, locationRange)

	// needle is in between start and end values if is greater than one and smaller than the other.
	return bool(greaterThanStart) != bool(greaterThanEnd)
}

func isSequenceMovingAwayFromEnd(
	interpreter *Interpreter,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	zeroValue IntegerValue,
) BoolValue {
	return (start.Less(interpreter, end, locationRange) && step.Less(interpreter, zeroValue, locationRange)) ||
		(start.Greater(interpreter, end, locationRange) && step.Greater(interpreter, zeroValue, locationRange))
}

func convertAndAssertIntegerValue(value Value) IntegerValue {
	integerValue, ok := value.(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return integerValue
}
