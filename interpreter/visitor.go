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
	if v.SimpleCompositeValueVisitor == nil {
		return
	}
	v.SimpleCompositeValueVisitor(context, value)
}

func (v EmptyVisitor) VisitTypeValue(context ValueVisitContext, value TypeValue) {
	if v.TypeValueVisitor == nil {
		return
	}
	v.TypeValueVisitor(context, value)
}

func (v EmptyVisitor) VisitVoidValue(context ValueVisitContext, value VoidValue) {
	if v.VoidValueVisitor == nil {
		return
	}
	v.VoidValueVisitor(context, value)
}

func (v EmptyVisitor) VisitBoolValue(context ValueVisitContext, value BoolValue) {
	if v.BoolValueVisitor == nil {
		return
	}
	v.BoolValueVisitor(context, value)
}

func (v EmptyVisitor) VisitStringValue(context ValueVisitContext, value *StringValue) {
	if v.StringValueVisitor == nil {
		return
	}
	v.StringValueVisitor(context, value)
}

func (v EmptyVisitor) VisitCharacterValue(context ValueVisitContext, value CharacterValue) {
	if v.StringValueVisitor == nil {
		return
	}
	v.CharacterValueVisitor(context, value)
}

func (v EmptyVisitor) VisitArrayValue(context ValueVisitContext, value *ArrayValue) bool {
	if v.ArrayValueVisitor == nil {
		return true
	}
	return v.ArrayValueVisitor(context, value)
}

func (v EmptyVisitor) VisitIntValue(context ValueVisitContext, value IntValue) {
	if v.IntValueVisitor == nil {
		return
	}
	v.IntValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt8Value(context ValueVisitContext, value Int8Value) {
	if v.Int8ValueVisitor == nil {
		return
	}
	v.Int8ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt16Value(context ValueVisitContext, value Int16Value) {
	if v.Int16ValueVisitor == nil {
		return
	}
	v.Int16ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt32Value(context ValueVisitContext, value Int32Value) {
	if v.Int32ValueVisitor == nil {
		return
	}
	v.Int32ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt64Value(context ValueVisitContext, value Int64Value) {
	if v.Int64ValueVisitor == nil {
		return
	}
	v.Int64ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt128Value(context ValueVisitContext, value Int128Value) {
	if v.Int128ValueVisitor == nil {
		return
	}
	v.Int128ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInt256Value(context ValueVisitContext, value Int256Value) {
	if v.Int256ValueVisitor == nil {
		return
	}
	v.Int256ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUIntValue(context ValueVisitContext, value UIntValue) {
	if v.UIntValueVisitor == nil {
		return
	}
	v.UIntValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt8Value(context ValueVisitContext, value UInt8Value) {
	if v.UInt8ValueVisitor == nil {
		return
	}
	v.UInt8ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt16Value(context ValueVisitContext, value UInt16Value) {
	if v.UInt16ValueVisitor == nil {
		return
	}
	v.UInt16ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt32Value(context ValueVisitContext, value UInt32Value) {
	if v.UInt32ValueVisitor == nil {
		return
	}
	v.UInt32ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt64Value(context ValueVisitContext, value UInt64Value) {
	if v.UInt64ValueVisitor == nil {
		return
	}
	v.UInt64ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt128Value(context ValueVisitContext, value UInt128Value) {
	if v.UInt128ValueVisitor == nil {
		return
	}
	v.UInt128ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUInt256Value(context ValueVisitContext, value UInt256Value) {
	if v.UInt256ValueVisitor == nil {
		return
	}
	v.UInt256ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord8Value(context ValueVisitContext, value Word8Value) {
	if v.Word8ValueVisitor == nil {
		return
	}
	v.Word8ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord16Value(context ValueVisitContext, value Word16Value) {
	if v.Word16ValueVisitor == nil {
		return
	}
	v.Word16ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord32Value(context ValueVisitContext, value Word32Value) {
	if v.Word32ValueVisitor == nil {
		return
	}
	v.Word32ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord64Value(context ValueVisitContext, value Word64Value) {
	if v.Word64ValueVisitor == nil {
		return
	}
	v.Word64ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord128Value(context ValueVisitContext, value Word128Value) {
	if v.Word128ValueVisitor == nil {
		return
	}
	v.Word128ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitWord256Value(context ValueVisitContext, value Word256Value) {
	if v.Word256ValueVisitor == nil {
		return
	}
	v.Word256ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitFix64Value(context ValueVisitContext, value Fix64Value) {
	if v.Fix64ValueVisitor == nil {
		return
	}
	v.Fix64ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitUFix64Value(context ValueVisitContext, value UFix64Value) {
	if v.UFix64ValueVisitor == nil {
		return
	}
	v.UFix64ValueVisitor(context, value)
}

func (v EmptyVisitor) VisitCompositeValue(context ValueVisitContext, value *CompositeValue) bool {
	if v.CompositeValueVisitor == nil {
		return true
	}
	return v.CompositeValueVisitor(context, value)
}

func (v EmptyVisitor) VisitDictionaryValue(context ValueVisitContext, value *DictionaryValue) bool {
	if v.DictionaryValueVisitor == nil {
		return true
	}
	return v.DictionaryValueVisitor(context, value)
}

func (v EmptyVisitor) VisitNilValue(context ValueVisitContext, value NilValue) {
	if v.NilValueVisitor == nil {
		return
	}
	v.NilValueVisitor(context, value)
}

func (v EmptyVisitor) VisitSomeValue(context ValueVisitContext, value *SomeValue) bool {
	if v.SomeValueVisitor == nil {
		return true
	}
	return v.SomeValueVisitor(context, value)
}

func (v EmptyVisitor) VisitStorageReferenceValue(context ValueVisitContext, value *StorageReferenceValue) {
	if v.StorageReferenceValueVisitor == nil {
		return
	}
	v.StorageReferenceValueVisitor(context, value)
}

func (v EmptyVisitor) VisitEphemeralReferenceValue(context ValueVisitContext, value *EphemeralReferenceValue) {
	if v.EphemeralReferenceValueVisitor == nil {
		return
	}
	v.EphemeralReferenceValueVisitor(context, value)
}

func (v EmptyVisitor) VisitAddressValue(context ValueVisitContext, value AddressValue) {
	if v.AddressValueVisitor == nil {
		return
	}
	v.AddressValueVisitor(context, value)
}

func (v EmptyVisitor) VisitPathValue(context ValueVisitContext, value PathValue) {
	if v.PathValueVisitor == nil {
		return
	}
	v.PathValueVisitor(context, value)
}

func (v EmptyVisitor) VisitCapabilityValue(context ValueVisitContext, value *IDCapabilityValue) {
	if v.CapabilityValueVisitor == nil {
		return
	}
	v.CapabilityValueVisitor(context, value)
}

func (v EmptyVisitor) VisitPublishedValue(context ValueVisitContext, value *PublishedValue) {
	if v.PublishedValueVisitor == nil {
		return
	}
	v.PublishedValueVisitor(context, value)
}

func (v EmptyVisitor) VisitInterpretedFunctionValue(context ValueVisitContext, value *InterpretedFunctionValue) {
	if v.InterpretedFunctionValueVisitor == nil {
		return
	}
	v.InterpretedFunctionValueVisitor(context, value)
}

func (v EmptyVisitor) VisitHostFunctionValue(context ValueVisitContext, value *HostFunctionValue) {
	if v.HostFunctionValueVisitor == nil {
		return
	}
	v.HostFunctionValueVisitor(context, value)
}

func (v EmptyVisitor) VisitBoundFunctionValue(context ValueVisitContext, value BoundFunctionValue) {
	if v.BoundFunctionValueVisitor == nil {
		return
	}
	v.BoundFunctionValueVisitor(context, value)
}

func (v EmptyVisitor) VisitStorageCapabilityControllerValue(context ValueVisitContext, value *StorageCapabilityControllerValue) {
	if v.StorageCapabilityControllerValueVisitor == nil {
		return
	}
	v.StorageCapabilityControllerValueVisitor(context, value)
}

func (v EmptyVisitor) VisitAccountCapabilityControllerValue(context ValueVisitContext, value *AccountCapabilityControllerValue) {
	if v.AccountCapabilityControllerValueVisitor == nil {
		return
	}
	v.AccountCapabilityControllerValueVisitor(context, value)
}
