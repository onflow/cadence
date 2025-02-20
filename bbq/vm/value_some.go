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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/interpreter"
)

type SomeValue struct {
	value Value
}

var _ Value = &SomeValue{}
var _ MemberAccessibleValue = &SomeValue{}
var _ ResourceKindedValue = &SomeValue{}
var _ EquatableValue = &SomeValue{}

func NewSomeValueNonCopying(value Value) *SomeValue {
	return &SomeValue{
		value: value,
	}
}

func (*SomeValue) isValue() {}

func (v *SomeValue) StaticType(config *Config) StaticType {
	innerType := v.value.StaticType(config)
	if innerType == nil {
		return nil
	}
	return interpreter.NewOptionalStaticType(
		config.MemoryGauge,
		innerType,
	)
}

func (v *SomeValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v *SomeValue) String() string {
	return v.value.String()
}
func (v *SomeValue) GetMember(config *Config, name string) Value {
	memberAccessibleValue := (v.value).(MemberAccessibleValue)
	return memberAccessibleValue.GetMember(config, name)
}

func (v *SomeValue) SetMember(config *Config, name string, value Value) {
	memberAccessibleValue := (v.value).(MemberAccessibleValue)
	memberAccessibleValue.SetMember(config, name, value)
}

func (v *SomeValue) IsResourceKinded() bool {
	resourceKinded, ok := v.value.(ResourceKindedValue)
	return ok && resourceKinded.IsResourceKinded()
}

func (v *SomeValue) Equal(other Value) BoolValue {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	equatableValue, ok := v.value.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(otherSome.value)
}
