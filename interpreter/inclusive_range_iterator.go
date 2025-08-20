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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

type InclusiveRangeIterator struct {
	rangeValue *CompositeValue
	next       IntegerValue

	// Cached values
	stepNegative bool
	step         IntegerValue
	end          IntegerValue
}

var _ ValueIterator = &InclusiveRangeIterator{}

type InclusiveRangeIteratorContext interface {
	common.MemoryGauge
	NumberValueArithmeticContext
}

func NewInclusiveRangeIterator(
	context InclusiveRangeIteratorContext,
	locationRange LocationRange,
	v *CompositeValue,
	typ InclusiveRangeStaticType,
) *InclusiveRangeIterator {
	startValue := getFieldAsIntegerValue(context, v, sema.InclusiveRangeTypeStartFieldName)

	zeroValue := GetSmallIntegerValue(0, typ.ElementType)
	endValue := getFieldAsIntegerValue(context, v, sema.InclusiveRangeTypeEndFieldName)

	stepValue := getFieldAsIntegerValue(context, v, sema.InclusiveRangeTypeStepFieldName)
	stepNegative := stepValue.Less(context, zeroValue, locationRange)

	i := &InclusiveRangeIterator{
		rangeValue:   v,
		stepNegative: bool(stepNegative),
		step:         stepValue,
		end:          endValue,
	}
	i.next = i.validate(startValue, context, locationRange)

	return i
}

func (i *InclusiveRangeIterator) Next(context ValueIteratorContext, locationRange LocationRange) Value {
	valueToReturn := i.next
	if valueToReturn == nil {
		return nil
	}

	// Update the next value.
	nextValueToReturn, ok := valueToReturn.Plus(context, i.step, locationRange).(IntegerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	i.next = i.validate(nextValueToReturn, context, locationRange)

	return valueToReturn
}

func (i *InclusiveRangeIterator) validate(
	element IntegerValue,
	context ValueIteratorContext,
	locationRange LocationRange,
) IntegerValue {
	// Ensure that element is within the bounds.
	if i.stepNegative && bool(element.Less(context, i.end, locationRange)) {
		return nil
	} else if !i.stepNegative && bool(element.Greater(context, i.end, locationRange)) {
		return nil
	}

	return element
}

func (i *InclusiveRangeIterator) HasNext() bool {
	return i.next != nil
}

func (*InclusiveRangeIterator) ValueID() (atree.ValueID, bool) {
	return atree.ValueID{}, false
}
