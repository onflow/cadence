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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

type Variable interface {
	GetValue(ValueStaticTypeContext) Value
	SetValue(context ValueStaticTypeContext, locationRange LocationRange, value Value)
	InitializeWithValue(value Value)
	InitializeWithGetter(getter func() Value)
}

type SimpleVariable struct {
	value  Value
	getter func() Value
}

var _ Variable = &SimpleVariable{}

func (v *SimpleVariable) InitializeWithValue(value Value) {
	v.getter = nil
	v.value = value
}

func (v *SimpleVariable) InitializeWithGetter(getter func() Value) {
	v.getter = getter
}

func (v *SimpleVariable) GetValue(ValueStaticTypeContext) Value {
	if v.getter != nil {
		v.value = v.getter()
		v.getter = nil
	}
	return v.value
}

func (v *SimpleVariable) SetValue(context ValueStaticTypeContext, locationRange LocationRange, value Value) {
	existingValue := v.GetValue(context)
	if existingValue != nil {
		checkResourceLoss(context, existingValue, locationRange)
	}
	v.getter = nil
	v.value = value
}

var variableMemoryUsage = common.NewConstantMemoryUsage(common.MemoryKindVariable)

func NewVariableWithValue(gauge common.MemoryGauge, value Value) Variable {
	common.UseMemory(gauge, variableMemoryUsage)
	return &SimpleVariable{
		value: value,
	}
}

func NewVariableWithGetter(gauge common.MemoryGauge, getter func() Value) Variable {
	common.UseMemory(gauge, variableMemoryUsage)
	return &SimpleVariable{
		getter: getter,
	}
}

type SelfVariable struct {
	value   Value
	selfRef ReferenceValue
}

var _ Variable = &SelfVariable{}

func NewSelfVariableWithValue(interpreter *Interpreter, value Value, locationRange LocationRange) Variable {
	common.UseMemory(interpreter, variableMemoryUsage)

	semaType := MustSemaTypeOfValue(value, interpreter)

	// Create an explicit reference to represent the implicit reference behavior of 'self' value.
	// Authorization doesn't matter, we just need a reference to add to tracking.
	selfRef := NewEphemeralReferenceValue(interpreter, UnauthorizedAccess, value, semaType, locationRange)

	return &SelfVariable{
		value:   value,
		selfRef: selfRef,
	}
}

func (v *SelfVariable) InitializeWithValue(Value) {
	// self variable cannot re-initialized.
	panic(errors.NewUnreachableError())
}

func (v *SelfVariable) InitializeWithGetter(func() Value) {
	// self variable doesn't have getters.
	panic(errors.NewUnreachableError())
}

func (v *SelfVariable) GetValue(context ValueStaticTypeContext) Value {
	// TODO: pass proper location range
	checkInvalidatedResourceOrResourceReference(v.selfRef, EmptyLocationRange, context)
	return v.value
}

func (v *SelfVariable) SetValue(ValueStaticTypeContext, LocationRange, Value) {
	// self variable cannot be updated.
	panic(errors.NewUnreachableError())
}
