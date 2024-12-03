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
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
)

type ReferenceValue interface {
	Value
	//AuthorizedValue
	isReference()
	ReferencedValue(config *Config, errorOnFailedDereference bool) *Value
	BorrowType() interpreter.StaticType
}

type EphemeralReferenceValue struct {
	Value Value
	// BorrowedType is the T in &T
	BorrowedType  interpreter.StaticType
	Authorization interpreter.Authorization
}

var _ Value = &EphemeralReferenceValue{}
var _ MemberAccessibleValue = &EphemeralReferenceValue{}
var _ ReferenceValue = &EphemeralReferenceValue{}

func NewEphemeralReferenceValue(
	conf *Config,
	value Value,
	authorization interpreter.Authorization,
	borrowedType interpreter.StaticType,
) *EphemeralReferenceValue {
	ref := &EphemeralReferenceValue{
		Value:         value,
		Authorization: authorization,
		BorrowedType:  borrowedType,
	}

	//maybeTrackReferencedResourceKindedValue(conf, ref)

	return ref
}

func (*EphemeralReferenceValue) isValue() {}

func (v *EphemeralReferenceValue) isReference() {}

func (v *EphemeralReferenceValue) ReferencedValue(*Config, bool) *Value {
	return &v.Value
}

func (v *EphemeralReferenceValue) BorrowType() interpreter.StaticType {
	return v.BorrowedType
}

func (v *EphemeralReferenceValue) StaticType(config *Config) StaticType {
	return interpreter.NewReferenceStaticType(
		config.MemoryGauge,
		v.Authorization,
		v.Value.StaticType(config),
	)
}

func (v *EphemeralReferenceValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v *EphemeralReferenceValue) String() string {
	return format.StorageReference
}

func (v *EphemeralReferenceValue) GetMember(config *Config, name string) Value {
	memberAccessibleValue := v.Value.(MemberAccessibleValue)
	return memberAccessibleValue.GetMember(config, name)
}

func (v *EphemeralReferenceValue) SetMember(config *Config, name string, value Value) {
	memberAccessibleValue := v.Value.(MemberAccessibleValue)
	memberAccessibleValue.SetMember(config, name, value)
}
