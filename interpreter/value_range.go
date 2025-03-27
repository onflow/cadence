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
	context MemberAccessibleContext,
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
	if startComparable.Greater(context, endComparable, locationRange) {
		elemSemaTy := MustConvertStaticToSemaType(rangeStaticType.ElementType, context)
		if elemSemaTy.Tag().BelongsTo(sema.UnsignedIntegerTypeTag) {
			panic(InclusiveRangeConstructionError{
				LocationRange: locationRange,
				Message: fmt.Sprintf(
					"step value cannot be negative for unsigned integer type %s",
					elemSemaTy,
				),
			})
		}

		negatedStep, ok := step.Negate(context, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		step = negatedStep
	}

	return createInclusiveRange(
		context,
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
	context MemberAccessibleContext,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
	rangeSemaType *sema.InclusiveRangeType,
) *CompositeValue {

	zeroValue := GetSmallIntegerValue(0, start.StaticType(context))

	// Validate that the step is non-zero.
	if step.Equal(context, locationRange, zeroValue) {
		panic(InclusiveRangeConstructionError{
			LocationRange: locationRange,
			Message:       "step value cannot be zero",
		})
	}

	// Validate that the sequence is moving towards the end value.
	// If start < end, step must be > 0
	// If start > end, step must be < 0
	// If start == end, step doesn't matter.
	if isSequenceMovingAwayFromEnd(context, locationRange, start, end, step, zeroValue) {

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
		context,
		locationRange,
		start,
		end,
		step,
		rangeType,
		rangeSemaType,
	)
}

func createInclusiveRange(
	context MemberAccessibleContext,
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
		context,
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
			context,
			rangeValue,
			sema.InclusiveRangeContainsFunctionType(
				rangeSemaType.MemberType,
			),
			func(rangeValue *CompositeValue, invocation Invocation) Value {
				needleInteger := convertAndAssertIntegerValue(invocation.Arguments[0])

				return rangeContains(
					rangeValue,
					rangeType,
					invocation.InvocationContext,
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
	context ValueComparisonContext,
	locationRange LocationRange,
	needleValue IntegerValue,
) BoolValue {
	start := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeStartFieldName)
	end := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeEndFieldName)
	step := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeStepFieldName)

	result := start.Equal(context, locationRange, needleValue) ||
		end.Equal(context, locationRange, needleValue)

	if result {
		return TrueValue
	}

	// Exclusive check since we already checked for boundaries above.
	if !isNeedleBetweenStartEndExclusive(context, locationRange, needleValue, start, end) {
		result = false
	} else {
		// needle is in between start and end.
		// start + k * step should be equal to needle i.e. (needle - start) mod step == 0.
		diff, ok := needleValue.Minus(context, start, locationRange).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		zeroValue := GetSmallIntegerValue(0, rangeType.ElementType)
		mod := diff.Mod(context, step, locationRange)
		result = mod.Equal(context, locationRange, zeroValue)
	}

	return BoolValue(result)
}

func getFieldAsIntegerValue(memoryGauge common.MemoryGauge, rangeValue *CompositeValue, name string) IntegerValue {
	return convertAndAssertIntegerValue(
		rangeValue.GetField(memoryGauge, name),
	)
}

func isNeedleBetweenStartEndExclusive(
	context ValueComparisonContext,
	locationRange LocationRange,
	needleValue IntegerValue,
	start IntegerValue,
	end IntegerValue,
) bool {
	greaterThanStart := needleValue.Greater(context, start, locationRange)
	greaterThanEnd := needleValue.Greater(context, end, locationRange)

	// needle is in between start and end values if is greater than one and smaller than the other.
	return bool(greaterThanStart) != bool(greaterThanEnd)
}

func isSequenceMovingAwayFromEnd(
	comparisonContext ValueComparisonContext,
	locationRange LocationRange,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	zeroValue IntegerValue,
) BoolValue {
	return (start.Less(comparisonContext, end, locationRange) && step.Less(comparisonContext, zeroValue, locationRange)) ||
		(start.Greater(comparisonContext, end, locationRange) && step.Greater(comparisonContext, zeroValue, locationRange))
}

func convertAndAssertIntegerValue(value Value) IntegerValue {
	integerValue, ok := value.(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return integerValue
}
