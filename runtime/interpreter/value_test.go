/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func newTestCompositeValue(owner common.Address) *CompositeValue {
	return NewCompositeValue(
		utils.TestLocation,
		"Test",
		common.CompositeKindStructure,
		NewStringValueOrderedMap(),
		&owner,
	)
}

func TestOwnerNewArray(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, value.GetOwner())

	array := NewArrayValueUnownedNonCopying(value)

	assert.Nil(t, array.GetOwner())
	assert.Nil(t, value.GetOwner())
}

func TestSetOwnerArray(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	array := NewArrayValueUnownedNonCopying(value)

	array.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerArrayCopy(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	array := NewArrayValueUnownedNonCopying(value)

	array.SetOwner(&newOwner)

	arrayCopy := array.Copy().(*ArrayValue)
	valueCopy := arrayCopy.Values[0]

	assert.Nil(t, arrayCopy.GetOwner())
	assert.Nil(t, valueCopy.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerArraySetIndex(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value1 := newTestCompositeValue(oldOwner)
	value2 := newTestCompositeValue(oldOwner)

	array := NewArrayValueUnownedNonCopying(value1)
	array.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &newOwner, value1.GetOwner())
	assert.Equal(t, &oldOwner, value2.GetOwner())

	array.Set(nil, ReturnEmptyLocationRange, NewIntValueFromInt64(0), value2)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &newOwner, value1.GetOwner())
	assert.Equal(t, &newOwner, value2.GetOwner())
}

func TestSetOwnerArrayAppend(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	array := NewArrayValueUnownedNonCopying()
	array.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())

	array.Append(value)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerArrayInsert(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	array := NewArrayValueUnownedNonCopying()
	array.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())

	array.Insert(0, value)

	assert.Equal(t, &newOwner, array.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestOwnerNewDictionary(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}

	keyValue := NewStringValue("test")
	value := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, value.GetOwner())

	dictionary := NewDictionaryValueUnownedNonCopying(keyValue, value)

	assert.Nil(t, dictionary.GetOwner())
	// NOTE: keyValue is string, has no owner
	assert.Nil(t, value.GetOwner())
}

func TestSetOwnerDictionary(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewStringValue("test")
	value := newTestCompositeValue(oldOwner)

	dictionary := NewDictionaryValueUnownedNonCopying(keyValue, value)

	dictionary.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, dictionary.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerDictionaryCopy(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewStringValue("test")
	value := newTestCompositeValue(oldOwner)

	dictionary := NewDictionaryValueUnownedNonCopying(keyValue, value)
	dictionary.SetOwner(&newOwner)

	dictionaryCopy := dictionary.Copy().(*DictionaryValue)
	valueCopy, _ := dictionaryCopy.Entries.Get(keyValue.KeyString())

	assert.Nil(t, dictionaryCopy.GetOwner())
	assert.Nil(t, valueCopy.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerDictionarySetIndex(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewStringValue("test")
	value := newTestCompositeValue(oldOwner)

	dictionary := NewDictionaryValueUnownedNonCopying()
	dictionary.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, dictionary.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())

	dictionary.Set(
		nil,
		ReturnEmptyLocationRange,
		keyValue,
		NewSomeValueOwningNonCopying(value),
	)

	assert.Equal(t, &newOwner, dictionary.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerDictionaryInsert(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewStringValue("test")
	value := newTestCompositeValue(oldOwner)

	dictionary := NewDictionaryValueUnownedNonCopying()
	dictionary.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, dictionary.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())

	dictionary.Insert(nil, ReturnEmptyLocationRange, keyValue, value)

	assert.Equal(t, &newOwner, dictionary.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestOwnerNewSome(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, value.GetOwner())

	any := NewSomeValueOwningNonCopying(value)

	assert.Equal(t, &oldOwner, any.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())
}

func TestSetOwnerSome(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, value.GetOwner())

	any := NewSomeValueOwningNonCopying(value)

	any.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, any.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerSomeCopy(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, value.GetOwner())

	some := NewSomeValueOwningNonCopying(value)
	some.SetOwner(&newOwner)

	someCopy := some.Copy().(*SomeValue)
	valueCopy := someCopy.Value

	assert.Nil(t, someCopy.GetOwner())
	assert.Nil(t, valueCopy.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestOwnerNewComposite(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}

	composite := newTestCompositeValue(oldOwner)

	assert.Equal(t, &oldOwner, composite.GetOwner())
}

func TestSetOwnerComposite(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)
	composite := newTestCompositeValue(oldOwner)

	const fieldName = "test"

	composite.Fields.Set(fieldName, value)

	composite.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, composite.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestSetOwnerCompositeCopy(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(oldOwner)
	composite := newTestCompositeValue(oldOwner)

	const fieldName = "test"

	composite.Fields.Set(fieldName, value)

	compositeCopy := composite.Copy().(*CompositeValue)
	valueCopy, _ := compositeCopy.Fields.Get(fieldName)

	assert.Nil(t, compositeCopy.GetOwner())
	assert.Nil(t, valueCopy.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())
}

func TestSetOwnerCompositeSetMember(t *testing.T) {

	t.Parallel()

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(oldOwner)
	composite := newTestCompositeValue(oldOwner)

	const fieldName = "test"

	composite.SetOwner(&newOwner)

	assert.Equal(t, &newOwner, composite.GetOwner())
	assert.Equal(t, &oldOwner, value.GetOwner())

	composite.SetMember(
		nil,
		ReturnEmptyLocationRange,
		fieldName,
		value,
	)

	assert.Equal(t, &newOwner, composite.GetOwner())
	assert.Equal(t, &newOwner, value.GetOwner())
}

func TestStringer(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    Value
		expected string
	}

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUIntValueFromUint64(10),
			expected: "10",
		},
		"UInt8": {
			value:    UInt8Value(8),
			expected: "8",
		},
		"UInt16": {
			value:    UInt16Value(16),
			expected: "16",
		},
		"UInt32": {
			value:    UInt32Value(32),
			expected: "32",
		},
		"UInt64": {
			value:    UInt64Value(64),
			expected: "64",
		},
		"UInt128": {
			value:    NewUInt128ValueFromUint64(128),
			expected: "128",
		},
		"UInt256": {
			value:    NewUInt256ValueFromUint64(256),
			expected: "256",
		},
		"Int8": {
			value:    Int8Value(-8),
			expected: "-8",
		},
		"Int16": {
			value:    Int16Value(-16),
			expected: "-16",
		},
		"Int32": {
			value:    Int32Value(-32),
			expected: "-32",
		},
		"Int64": {
			value:    Int64Value(-64),
			expected: "-64",
		},
		"Int128": {
			value:    NewInt128ValueFromInt64(-128),
			expected: "-128",
		},
		"Int256": {
			value:    NewInt256ValueFromInt64(-256),
			expected: "-256",
		},
		"Word8": {
			value:    Word8Value(8),
			expected: "8",
		},
		"Word16": {
			value:    Word16Value(16),
			expected: "16",
		},
		"Word32": {
			value:    Word32Value(32),
			expected: "32",
		},
		"Word64": {
			value:    Word64Value(64),
			expected: "64",
		},
		"UFix64": {
			value:    NewUFix64ValueWithInteger(64),
			expected: "64.00000000",
		},
		"Fix64": {
			value:    NewFix64ValueWithInteger(-32),
			expected: "-32.00000000",
		},
		"Void": {
			value:    VoidValue{},
			expected: "()",
		},
		"true": {
			value:    BoolValue(true),
			expected: "true",
		},
		"false": {
			value:    BoolValue(false),
			expected: "false",
		},
		"some": {
			value:    NewSomeValueOwningNonCopying(BoolValue(true)),
			expected: "true",
		},
		"nil": {
			value:    NilValue{},
			expected: "nil",
		},
		"String": {
			value:    NewStringValue("Flow ridah!"),
			expected: "\"Flow ridah!\"",
		},
		"Array": {
			value: NewArrayValueUnownedNonCopying(
				NewIntValueFromInt64(10),
				NewStringValue("TEST"),
			),
			expected: "[10, \"TEST\"]",
		},
		"Dictionary": {
			value: NewDictionaryValueUnownedNonCopying(
				NewStringValue("key"),
				NewStringValue("value"),
			),
			expected: "{\"key\": \"value\"}",
		},
		"Address": {
			value:    NewAddressValue(common.Address{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "0x1",
		},
		"composite": {
			value: func() Value {
				members := NewStringValueOrderedMap()
				members.Set("y", NewStringValue("bar"))

				return NewCompositeValue(
					utils.TestLocation,
					"Foo",
					common.CompositeKindResource,
					members,
					nil,
				)
			}(),
			expected: "S.test.Foo(y: \"bar\")",
		},
		"composite with custom stringer": {
			value: func() Value {
				members := NewStringValueOrderedMap()
				members.Set("y", NewStringValue("bar"))

				compositeValue := NewCompositeValue(
					utils.TestLocation,
					"Foo",
					common.CompositeKindResource,
					members,
					nil,
				)

				compositeValue.stringer = func() string {
					return "y --> bar"
				}

				return compositeValue
			}(),
			expected: "y --> bar",
		},
		"Link": {
			value: LinkValue{
				TargetPath: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Type: PrimitiveStaticTypeInt,
			},
			expected: "Link<Int>(/storage/foo)",
		},
		"Path": {
			value: PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			expected: "/storage/foo",
		},
		"Type": {
			value:    TypeValue{Type: PrimitiveStaticTypeInt},
			expected: "Type<Int>()",
		},
		"Capability with borrow type": {
			value: CapabilityValue{
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Address:    NewAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
				BorrowType: PrimitiveStaticTypeInt,
			},
			expected: "Capability<Int>(address: 0x102030405, path: /storage/foo)",
		},
		"Capability without borrow type": {
			value: CapabilityValue{
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Address: NewAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
			},
			expected: "Capability(address: 0x102030405, path: /storage/foo)",
		},
		"Dictionary with non-deferred values": {
			value: NewDictionaryValueUnownedNonCopying(
				NewStringValue("a"), UInt8Value(42),
				NewStringValue("b"), UInt8Value(99),
			),
			expected: `{"a": 42, "b": 99}`,
		},
		"Dictionary with deferred value": {
			value: func() Value {
				entries := NewStringValueOrderedMap()
				entries.Set(
					NewStringValue("a").KeyString(),
					UInt8Value(42),
				)
				return &DictionaryValue{
					Keys: NewArrayValueUnownedNonCopying(
						NewStringValue("a"),
						NewStringValue("b"),
					),
					Entries: entries,
				}
			}(),
			expected: `{"a": 42, "b": ...}`,
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			assert.Equal(t,
				testCase.expected,
				testCase.value.String(),
			)
		})
	}

	for name, testCase := range stringerTests {
		test(name, testCase)
	}
}

func TestVisitor(t *testing.T) {

	t.Parallel()

	var intVisits, stringVisits int

	visitor := EmptyVisitor{
		IntValueVisitor: func(interpreter *Interpreter, value IntValue) {
			intVisits++
		},
		StringValueVisitor: func(interpreter *Interpreter, value *StringValue) {
			stringVisits++
		},
	}

	var value Value
	value = NewIntValueFromInt64(42)
	value = NewSomeValueOwningNonCopying(value)
	value = NewArrayValueUnownedNonCopying(value)
	value = NewDictionaryValueUnownedNonCopying(NewStringValue("42"), value)
	members := NewStringValueOrderedMap()
	members.Set("foo", value)
	value = NewCompositeValue(
		utils.TestLocation,
		"Foo",
		common.CompositeKindStructure,
		members,
		nil,
	)

	value.Accept(nil, visitor)

	require.Equal(t, 1, intVisits)
	require.Equal(t, 1, stringVisits)
}

func TestKeyString(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    HasKeyString
		expected string
	}

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUIntValueFromUint64(10),
			expected: "10",
		},
		"UInt8": {
			value:    UInt8Value(8),
			expected: "8",
		},
		"UInt16": {
			value:    UInt16Value(16),
			expected: "16",
		},
		"UInt32": {
			value:    UInt32Value(32),
			expected: "32",
		},
		"UInt64": {
			value:    UInt64Value(64),
			expected: "64",
		},
		"UInt128": {
			value:    NewUInt128ValueFromUint64(128),
			expected: "128",
		},
		"UInt256": {
			value:    NewUInt256ValueFromUint64(256),
			expected: "256",
		},
		"Int8": {
			value:    Int8Value(-8),
			expected: "-8",
		},
		"Int16": {
			value:    Int16Value(-16),
			expected: "-16",
		},
		"Int32": {
			value:    Int32Value(-32),
			expected: "-32",
		},
		"Int64": {
			value:    Int64Value(-64),
			expected: "-64",
		},
		"Int128": {
			value:    NewInt128ValueFromInt64(-128),
			expected: "-128",
		},
		"Int256": {
			value:    NewInt256ValueFromInt64(-256),
			expected: "-256",
		},
		"Word8": {
			value:    Word8Value(8),
			expected: "8",
		},
		"Word16": {
			value:    Word16Value(16),
			expected: "16",
		},
		"Word32": {
			value:    Word32Value(32),
			expected: "32",
		},
		"Word64": {
			value:    Word64Value(64),
			expected: "64",
		},
		"UFix64": {
			value:    NewUFix64ValueWithInteger(64),
			expected: "64.00000000",
		},
		"Fix64": {
			value:    NewFix64ValueWithInteger(-32),
			expected: "-32.00000000",
		},
		"true": {
			value:    BoolValue(true),
			expected: "true",
		},
		"false": {
			value:    BoolValue(false),
			expected: "false",
		},
		"String": {
			value:    NewStringValue("Flow ridah!"),
			expected: "Flow ridah!",
		},
		"Address": {
			value:    NewAddressValue(common.Address{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "0x1",
		},
		"enum": {
			value: func() HasKeyString {
				members := NewStringValueOrderedMap()
				members.Set("rawValue", UInt8Value(42))
				return NewCompositeValue(
					utils.TestLocation,
					"Foo",
					common.CompositeKindEnum,
					members,
					nil,
				)
			}(),
			expected: "42",
		},
		"Path": {
			value: PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			// NOTE: this is an unfortunate mistake,
			// the KeyString function should have been using Domain.Identifier()
			expected: "/PathDomainStorage/foo",
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			assert.Equal(t,
				testCase.expected,
				testCase.value.KeyString(),
			)
		})
	}

	for name, testCase := range stringerTests {
		test(name, testCase)
	}
}

func TestBlockValue(t *testing.T) {

	t.Parallel()

	block := BlockValue{
		Height:    4,
		View:      5,
		ID:        NewArrayValueUnownedNonCopying(),
		Timestamp: 5.0,
	}

	// static type test
	var actualTs = block.Timestamp
	const expectedTs UFix64Value = 5.0
	assert.Equal(t, expectedTs, actualTs)
}
