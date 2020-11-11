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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func newTestCompositeValue(owner common.Address) *CompositeValue {
	return NewCompositeValue(
		utils.TestLocation,
		"S.test.Test",
		common.CompositeKindStructure,
		map[string]Value{},
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

	array.Set(nil, LocationRange{}, NewIntValueFromInt64(0), value2)

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
	valueCopy := dictionaryCopy.Entries[keyValue.KeyString()]

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
		LocationRange{},
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

	dictionary.Insert(nil, LocationRange{}, keyValue, value)

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

	composite.Fields[fieldName] = value

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

	composite.Fields[fieldName] = value

	compositeCopy := composite.Copy().(*CompositeValue)
	valueCopy := compositeCopy.Fields[fieldName]

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
		LocationRange{},
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
			value: NewCompositeValue(
				ast.StringLocation("test"),
				"S.test.Foo",
				common.CompositeKindResource,
				map[string]Value{
					"y": NewStringValue("bar"),
				},
				nil,
			),
			expected: "S.test.Foo(y: \"bar\")",
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
		"Capability": {
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
	value = NewCompositeValue(
		utils.TestLocation,
		"S.test.Foo",
		common.CompositeKindStructure,
		map[string]Value{
			"foo": value,
		},
		nil,
	)

	value.Accept(nil, visitor)

	require.Equal(t, 1, intVisits)
	require.Equal(t, 1, stringVisits)
}
