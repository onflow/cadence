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

package vm

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/common"
)

type SimpleCompositeValue struct {
	fields     map[string]Value
	typeID     common.TypeID
	staticType StaticType
	Kind       common.CompositeKind

	// metadata is a property bag to carry internal data
	// that are not visible to cadence users.
	// TODO: any better way to pass down information?
	metadata map[string]any
}

var _ Value = &CompositeValue{}

func NewSimpleCompositeValue(
	kind common.CompositeKind,
	typeID common.TypeID,
	fields map[string]Value,
) *SimpleCompositeValue {

	return &SimpleCompositeValue{
		Kind:   kind,
		typeID: typeID,
		fields: fields,
	}
}

func (*SimpleCompositeValue) isValue() {}

func (v *SimpleCompositeValue) StaticType(memoryGauge common.MemoryGauge) StaticType {
	return v.staticType
}

func (v *SimpleCompositeValue) GetMember(_ *Config, name string) Value {
	return v.fields[name]
}

func (v *SimpleCompositeValue) SetMember(_ *Config, name string, value Value) {
	v.fields[name] = value
}

func (v *SimpleCompositeValue) IsResourceKinded() bool {
	return v.Kind == common.CompositeKindResource
}

func (v *SimpleCompositeValue) String() string {
	//TODO implement me
	panic("implement me")
}

func (v *SimpleCompositeValue) Transfer(
	conf *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	return v
}

func (v *SimpleCompositeValue) Destroy(*Config) {}
