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
	if startComparable.Greater(context, endComparable) {
		elemSemaTy := context.SemaTypeFromStaticType(rangeStaticType.ElementType)
		if elemSemaTy.Tag().BelongsTo(sema.UnsignedIntegerTypeTag) {
			panic(&InclusiveRangeConstructionError{
				Message: fmt.Sprintf(
					"step value cannot be negative for unsigned integer type %s",
					elemSemaTy,
				),
			})
		}

		negatedStep, ok := step.Negate(context).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		step = negatedStep
	}

	return createInclusiveRange(
		context,
		start,
		end,
		step,
		rangeStaticType,
		rangeSemaType,
	)
}

// NewInclusiveRangeValueWithStep constructs an InclusiveRange value with the provided start, end & step.
// NOTE: Assumes that the values start, end and step are of the same static type.
func NewInclusiveRangeValueWithStep(
	context MemberAccessibleContext,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	rangeType InclusiveRangeStaticType,
	rangeSemaType *sema.InclusiveRangeType,
) *CompositeValue {

	zeroValue := GetSmallIntegerValue(0, start.StaticType(context))

	// Validate that the step is non-zero.
	if step.Equal(context, zeroValue) {
		panic(&InclusiveRangeConstructionError{
			Message: "step value cannot be zero",
		})
	}

	// Validate that the sequence is moving towards the end value.
	// If start < end, step must be > 0
	// If start > end, step must be < 0
	// If start == end, step doesn't matter.
	if isSequenceMovingAwayFromEnd(context, start, end, step, zeroValue) {

		panic(&InclusiveRangeConstructionError{
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
		start,
		end,
		step,
		rangeType,
		rangeSemaType,
	)
}

var NativeInclusiveRangeContainsFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		rangeValue := AssertValueOfType[*CompositeValue](receiver)
		needleInteger := convertAndAssertIntegerValue(args[0])
		rangeType, ok := rangeValue.StaticType(context).(InclusiveRangeStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		return InclusiveRangeContains(
			rangeValue,
			rangeType,
			context,
			needleInteger,
		)
	},
)

func createInclusiveRange(
	context MemberAccessibleContext,
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
			NativeInclusiveRangeContainsFunction,
		),
	)

	return rangeValue
}

func InclusiveRangeContains(
	rangeValue *CompositeValue,
	rangeType InclusiveRangeStaticType,
	context ValueComparisonContext,
	needleValue IntegerValue,
) BoolValue {
	start := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeStartFieldName)
	end := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeEndFieldName)
	step := getFieldAsIntegerValue(context, rangeValue, sema.InclusiveRangeTypeStepFieldName)

	result := start.Equal(context, needleValue) ||
		end.Equal(context, needleValue)

	if result {
		return TrueValue
	}

	// Exclusive check since we already checked for boundaries above.
	if !isNeedleBetweenStartEndExclusive(context, needleValue, start, end) {
		result = false
	} else {
		// needle is in between start and end.
		// start + k * step should be equal to needle i.e. (needle - start) mod step == 0.
		diff, ok := needleValue.Minus(context, start).(IntegerValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		zeroValue := GetSmallIntegerValue(0, rangeType.ElementType)
		mod := diff.Mod(context, step)
		result = mod.Equal(context, zeroValue)
	}

	return BoolValue(result)
}

func getFieldAsIntegerValue(gauge common.Gauge, rangeValue *CompositeValue, name string) IntegerValue {
	return convertAndAssertIntegerValue(
		rangeValue.GetField(gauge, name),
	)
}

func isNeedleBetweenStartEndExclusive(
	context ValueComparisonContext,
	needleValue IntegerValue,
	start IntegerValue,
	end IntegerValue,
) bool {
	greaterThanStart := needleValue.Greater(context, start)
	greaterThanEnd := needleValue.Greater(context, end)

	// needle is in between start and end values if is greater than one and smaller than the other.
	return bool(greaterThanStart) != bool(greaterThanEnd)
}

func isSequenceMovingAwayFromEnd(
	comparisonContext ValueComparisonContext,
	start IntegerValue,
	end IntegerValue,
	step IntegerValue,
	zeroValue IntegerValue,
) BoolValue {
	return (start.Less(comparisonContext, end) && step.Less(comparisonContext, zeroValue)) ||
		(start.Greater(comparisonContext, end) && step.Greater(comparisonContext, zeroValue))
}

func convertAndAssertIntegerValue(value Value) IntegerValue {
	integerValue, ok := value.(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return integerValue
}
