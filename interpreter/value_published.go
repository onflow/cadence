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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
)

// PublishedValue

type PublishedValue struct {
	// NB: If `publish` and `claim` are ever extended to support arbitrary values, rather than just capabilities,
	// this will need to be changed to `Value`, and more storage-related operations must be implemented for `PublishedValue`
	Value     CapabilityValue
	Recipient AddressValue
}

func NewPublishedValue(memoryGauge common.MemoryGauge, recipient AddressValue, value CapabilityValue) *PublishedValue {
	common.UseMemory(memoryGauge, common.PublishedValueMemoryUsage)
	return &PublishedValue{
		Recipient: recipient,
		Value:     value,
	}
}

var _ Value = &PublishedValue{}
var _ atree.Value = &PublishedValue{}
var _ EquatableValue = &PublishedValue{}

func (*PublishedValue) isValue() {}

func (v *PublishedValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitPublishedValue(interpreter, v)
}

func (v *PublishedValue) StaticType(context ValueStaticTypeContext) StaticType {
	// checking the static type of a published value should show us the
	// static type of the underlying value
	return v.Value.StaticType(context)
}

func (*PublishedValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (v *PublishedValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *PublishedValue) RecursiveString(seenReferences SeenReferences) string {
	return fmt.Sprintf(
		"PublishedValue<%s>(%s)",
		v.Recipient.RecursiveString(seenReferences),
		v.Value.RecursiveString(seenReferences),
	)
}

func (v *PublishedValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
	common.UseMemory(interpreter, common.PublishedValueStringMemoryUsage)

	return fmt.Sprintf(
		"PublishedValue<%s>(%s)",
		v.Recipient.MeteredString(interpreter, seenReferences, locationRange),
		v.Value.MeteredString(interpreter, seenReferences, locationRange),
	)
}

func (v *PublishedValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
	walkChild(v.Recipient)
	walkChild(v.Value)
}

func (v *PublishedValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return false
}

func (v *PublishedValue) Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool {
	otherValue, ok := other.(*PublishedValue)
	if !ok {
		return false
	}

	return otherValue.Recipient.Equal(context, locationRange, v.Recipient) &&
		otherValue.Value.Equal(context, locationRange, v.Value)
}

func (*PublishedValue) IsStorable() bool {
	return true
}

func (v *PublishedValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (v *PublishedValue) NeedsStoreTo(address atree.Address) bool {
	return v.Value.NeedsStoreTo(address)
}

func (*PublishedValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *PublishedValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {
	// NB: if the inner value of a PublishedValue can be a resource,
	// we must perform resource-related checks here as well

	if v.NeedsStoreTo(address) {

		innerValue := v.Value.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
			hasNoParentContainer,
		).(*IDCapabilityValue)

		addressValue := v.Recipient.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
			hasNoParentContainer,
		).(AddressValue)

		if remove {
			interpreter.RemoveReferencedSlab(storable)
		}

		return NewPublishedValue(interpreter, addressValue, innerValue)
	}

	return v

}

func (v *PublishedValue) Clone(interpreter *Interpreter) Value {
	return &PublishedValue{
		Recipient: v.Recipient,
		Value:     v.Value.Clone(interpreter).(*IDCapabilityValue),
	}
}

func (*PublishedValue) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v *PublishedValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *PublishedValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *PublishedValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Recipient,
		v.Value,
	}
}
