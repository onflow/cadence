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

type Visitor interface {
	VisitSimpleCompositeValue(context ValueVisitContext, value *SimpleCompositeValue)
	VisitTypeValue(context ValueVisitContext, value TypeValue)
	VisitVoidValue(context ValueVisitContext, value VoidValue)
	VisitBoolValue(context ValueVisitContext, value BoolValue)
	VisitStringValue(context ValueVisitContext, value *StringValue)
	VisitCharacterValue(context ValueVisitContext, value CharacterValue)
	VisitArrayValue(context ValueVisitContext, value *ArrayValue) bool
	VisitIntValue(context ValueVisitContext, value IntValue)
	VisitInt8Value(context ValueVisitContext, value Int8Value)
	VisitInt16Value(context ValueVisitContext, value Int16Value)
	VisitInt32Value(context ValueVisitContext, value Int32Value)
	VisitInt64Value(context ValueVisitContext, value Int64Value)
	VisitInt128Value(context ValueVisitContext, value Int128Value)
	VisitInt256Value(context ValueVisitContext, value Int256Value)
	VisitUIntValue(context ValueVisitContext, value UIntValue)
	VisitUInt8Value(context ValueVisitContext, value UInt8Value)
	VisitUInt16Value(context ValueVisitContext, value UInt16Value)
	VisitUInt32Value(context ValueVisitContext, value UInt32Value)
	VisitUInt64Value(context ValueVisitContext, value UInt64Value)
	VisitUInt128Value(context ValueVisitContext, value UInt128Value)
	VisitUInt256Value(context ValueVisitContext, value UInt256Value)
	VisitWord8Value(context ValueVisitContext, value Word8Value)
	VisitWord16Value(context ValueVisitContext, value Word16Value)
	VisitWord32Value(context ValueVisitContext, value Word32Value)
	VisitWord64Value(context ValueVisitContext, value Word64Value)
	VisitWord128Value(context ValueVisitContext, value Word128Value)
	VisitWord256Value(context ValueVisitContext, value Word256Value)
	VisitFix64Value(context ValueVisitContext, value Fix64Value)
	VisitFix128Value(context ValueVisitContext, v Fix128Value)
	VisitUFix64Value(context ValueVisitContext, value UFix64Value)
	VisitCompositeValue(context ValueVisitContext, value *CompositeValue) bool
	VisitDictionaryValue(context ValueVisitContext, value *DictionaryValue) bool
	VisitNilValue(context ValueVisitContext, value NilValue)
	VisitSomeValue(context ValueVisitContext, value *SomeValue) bool
	VisitStorageReferenceValue(context ValueVisitContext, value *StorageReferenceValue)
	VisitEphemeralReferenceValue(context ValueVisitContext, value *EphemeralReferenceValue)
	VisitAddressValue(context ValueVisitContext, value AddressValue)
	VisitPathValue(context ValueVisitContext, value PathValue)
	VisitCapabilityValue(context ValueVisitContext, value *IDCapabilityValue)
	VisitPublishedValue(context ValueVisitContext, value *PublishedValue)
	VisitInterpretedFunctionValue(context ValueVisitContext, value *InterpretedFunctionValue)
	VisitHostFunctionValue(context ValueVisitContext, value *HostFunctionValue)
	VisitBoundFunctionValue(context ValueVisitContext, value BoundFunctionValue)
	VisitStorageCapabilityControllerValue(context ValueVisitContext, v *StorageCapabilityControllerValue)
	VisitAccountCapabilityControllerValue(context ValueVisitContext, v *AccountCapabilityControllerValue)
}

type EmptyVisitor struct {
	SimpleCompositeValueVisitor             func(context ValueVisitContext, value *SimpleCompositeValue)
	TypeValueVisitor                        func(context ValueVisitContext, value TypeValue)
	VoidValueVisitor                        func(context ValueVisitContext, value VoidValue)
	BoolValueVisitor                        func(context ValueVisitContext, value BoolValue)
	CharacterValueVisitor                   func(context ValueVisitContext, value CharacterValue)
	StringValueVisitor                      func(context ValueVisitContext, value *StringValue)
	ArrayValueVisitor                       func(context ValueVisitContext, value *ArrayValue) bool
	IntValueVisitor                         func(context ValueVisitContext, value IntValue)
	Int8ValueVisitor                        func(context ValueVisitContext, value Int8Value)
	Int16ValueVisitor                       func(context ValueVisitContext, value Int16Value)
	Int32ValueVisitor                       func(context ValueVisitContext, value Int32Value)
	Int64ValueVisitor                       func(context ValueVisitContext, value Int64Value)
	Int128ValueVisitor                      func(context ValueVisitContext, value Int128Value)
	Int256ValueVisitor                      func(context ValueVisitContext, value Int256Value)
	UIntValueVisitor                        func(context ValueVisitContext, value UIntValue)
	UInt8ValueVisitor                       func(context ValueVisitContext, value UInt8Value)
	UInt16ValueVisitor                      func(context ValueVisitContext, value UInt16Value)
	UInt32ValueVisitor                      func(context ValueVisitContext, value UInt32Value)
	UInt64ValueVisitor                      func(context ValueVisitContext, value UInt64Value)
	UInt128ValueVisitor                     func(context ValueVisitContext, value UInt128Value)
	UInt256ValueVisitor                     func(context ValueVisitContext, value UInt256Value)
	Word8ValueVisitor                       func(context ValueVisitContext, value Word8Value)
	Word16ValueVisitor                      func(context ValueVisitContext, value Word16Value)
	Word32ValueVisitor                      func(context ValueVisitContext, value Word32Value)
	Word64ValueVisitor                      func(context ValueVisitContext, value Word64Value)
	Word128ValueVisitor                     func(context ValueVisitContext, value Word128Value)
	Word256ValueVisitor                     func(context ValueVisitContext, value Word256Value)
	Fix64ValueVisitor                       func(context ValueVisitContext, value Fix64Value)
	Fix128ValueVisitor                      func(context ValueVisitContext, value Fix128Value)
	UFix64ValueVisitor                      func(context ValueVisitContext, value UFix64Value)
	CompositeValueVisitor                   func(context ValueVisitContext, value *CompositeValue) bool
	DictionaryValueVisitor                  func(context ValueVisitContext, value *DictionaryValue) bool
	NilValueVisitor                         func(context ValueVisitContext, value NilValue)
	SomeValueVisitor                        func(context ValueVisitContext, value *SomeValue) bool
	StorageReferenceValueVisitor            func(context ValueVisitContext, value *StorageReferenceValue)
	EphemeralReferenceValueVisitor          func(context ValueVisitContext, value *EphemeralReferenceValue)
	AddressValueVisitor                     func(context ValueVisitContext, value AddressValue)
	PathValueVisitor                        func(context ValueVisitContext, value PathValue)
	CapabilityValueVisitor                  func(context ValueVisitContext, value *IDCapabilityValue)
	PublishedValueVisitor                   func(context ValueVisitContext, value *PublishedValue)
	InterpretedFunctionValueVisitor         func(context ValueVisitContext, value *InterpretedFunctionValue)
	HostFunctionValueVisitor                func(context ValueVisitContext, value *HostFunctionValue)
	BoundFunctionValueVisitor               func(context ValueVisitContext, value BoundFunctionValue)
	StorageCapabilityControllerValueVisitor func(context ValueVisitContext, value *StorageCapabilityControllerValue)
	AccountCapabilityControllerValueVisitor func(context ValueVisitContext, value *AccountCapabilityControllerValue)
}

var _ Visitor = &EmptyVisitor{}

func (v EmptyVisitor) VisitSimpleCompositeValue(context ValueVisitContext, value *SimpleCompositeValue) {
	visitor := v.SimpleCompositeValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitTypeValue(context ValueVisitContext, value TypeValue) {
	visitor := v.TypeValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitVoidValue(context ValueVisitContext, value VoidValue) {
	visitor := v.VoidValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitBoolValue(context ValueVisitContext, value BoolValue) {
	visitor := v.BoolValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitStringValue(context ValueVisitContext, value *StringValue) {
	visitor := v.StringValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitCharacterValue(context ValueVisitContext, value CharacterValue) {
	visitor := v.CharacterValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitArrayValue(context ValueVisitContext, value *ArrayValue) bool {
	visitor := v.ArrayValueVisitor
	if visitor == nil {
		return true
	}
	return visitor(context, value)
}

func (v EmptyVisitor) VisitIntValue(context ValueVisitContext, value IntValue) {
	visitor := v.IntValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt8Value(context ValueVisitContext, value Int8Value) {
	visitor := v.Int8ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt16Value(context ValueVisitContext, value Int16Value) {
	visitor := v.Int16ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt32Value(context ValueVisitContext, value Int32Value) {
	visitor := v.Int32ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt64Value(context ValueVisitContext, value Int64Value) {
	visitor := v.Int64ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt128Value(context ValueVisitContext, value Int128Value) {
	visitor := v.Int128ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInt256Value(context ValueVisitContext, value Int256Value) {
	visitor := v.Int256ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUIntValue(context ValueVisitContext, value UIntValue) {
	visitor := v.UIntValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt8Value(context ValueVisitContext, value UInt8Value) {
	visitor := v.UInt8ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt16Value(context ValueVisitContext, value UInt16Value) {
	visitor := v.UInt16ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt32Value(context ValueVisitContext, value UInt32Value) {
	visitor := v.UInt32ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt64Value(context ValueVisitContext, value UInt64Value) {
	visitor := v.UInt64ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt128Value(context ValueVisitContext, value UInt128Value) {
	visitor := v.UInt128ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUInt256Value(context ValueVisitContext, value UInt256Value) {
	visitor := v.UInt256ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord8Value(context ValueVisitContext, value Word8Value) {
	visitor := v.Word8ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord16Value(context ValueVisitContext, value Word16Value) {
	visitor := v.Word16ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord32Value(context ValueVisitContext, value Word32Value) {
	visitor := v.Word32ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord64Value(context ValueVisitContext, value Word64Value) {
	visitor := v.Word64ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord128Value(context ValueVisitContext, value Word128Value) {
	visitor := v.Word128ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitWord256Value(context ValueVisitContext, value Word256Value) {
	visitor := v.Word256ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitFix64Value(context ValueVisitContext, value Fix64Value) {
	visitor := v.Fix64ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitFix128Value(context ValueVisitContext, value Fix128Value) {
	visitor := v.Fix128ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitUFix64Value(context ValueVisitContext, value UFix64Value) {
	visitor := v.UFix64ValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitCompositeValue(context ValueVisitContext, value *CompositeValue) bool {
	visitor := v.CompositeValueVisitor
	if visitor == nil {
		return true
	}
	return visitor(context, value)
}

func (v EmptyVisitor) VisitDictionaryValue(context ValueVisitContext, value *DictionaryValue) bool {
	visitor := v.DictionaryValueVisitor
	if visitor == nil {
		return true
	}
	return visitor(context, value)
}

func (v EmptyVisitor) VisitNilValue(context ValueVisitContext, value NilValue) {
	visitor := v.NilValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitSomeValue(context ValueVisitContext, value *SomeValue) bool {
	visitor := v.SomeValueVisitor
	if visitor == nil {
		return true
	}
	return visitor(context, value)
}

func (v EmptyVisitor) VisitStorageReferenceValue(context ValueVisitContext, value *StorageReferenceValue) {
	visitor := v.StorageReferenceValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitEphemeralReferenceValue(context ValueVisitContext, value *EphemeralReferenceValue) {
	visitor := v.EphemeralReferenceValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitAddressValue(context ValueVisitContext, value AddressValue) {
	visitor := v.AddressValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitPathValue(context ValueVisitContext, value PathValue) {
	visitor := v.PathValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitCapabilityValue(context ValueVisitContext, value *IDCapabilityValue) {
	visitor := v.CapabilityValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitPublishedValue(context ValueVisitContext, value *PublishedValue) {
	visitor := v.PublishedValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitInterpretedFunctionValue(context ValueVisitContext, value *InterpretedFunctionValue) {
	visitor := v.InterpretedFunctionValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitHostFunctionValue(context ValueVisitContext, value *HostFunctionValue) {
	visitor := v.HostFunctionValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitBoundFunctionValue(context ValueVisitContext, value BoundFunctionValue) {
	visitor := v.BoundFunctionValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitStorageCapabilityControllerValue(context ValueVisitContext, value *StorageCapabilityControllerValue) {
	visitor := v.StorageCapabilityControllerValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}

func (v EmptyVisitor) VisitAccountCapabilityControllerValue(context ValueVisitContext, value *AccountCapabilityControllerValue) {
	visitor := v.AccountCapabilityControllerValueVisitor
	if visitor == nil {
		return
	}
	visitor(context, value)
}
