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

package interpreter_test

import (
	"fmt"
	"go/types"
	"math"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	checkerUtils "github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func newTestCompositeValue(inter *Interpreter, owner common.Address) *CompositeValue {
	return NewCompositeValue(
		inter,
		utils.TestLocation,
		"Test",
		common.CompositeKindStructure,
		nil,
		owner,
	)
}

var testCompositeValueType = &sema.CompositeType{
	Location:   utils.TestLocation,
	Identifier: "Test",
	Kind:       common.CompositeKindStructure,
	Members:    sema.NewStringMemberOrderedMap(),
}

func getMeterCompFuncWithExpectedKinds(
	t *testing.T,
	kinds []common.ComputationKind,
	intensities []uint,
) OnMeterComputationFunc {
	if len(kinds) != len(intensities) {
		t.Fatal("size of kinds doesn't match size of intensitites")
	}
	expectedCompKindsIndex := 0
	return func(compKind common.ComputationKind, intensity uint) {
		if expectedCompKindsIndex >= len(kinds) {
			t.Fatal("received an extra meterComputation call")
		}
		assert.Equal(t, kinds[expectedCompKindsIndex], compKind)
		assert.Equal(t, intensities[expectedCompKindsIndex], intensity)
		expectedCompKindsIndex++
	}
}

func TestOwnerNewArray(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(inter, oldOwner)

	assert.Equal(t, oldOwner, value.GetOwner())

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.Address{},
		value,
	)

	value = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, common.Address{}, array.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestOwnerArrayDeepCopy(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
		WithOnMeterComputationFuncHandler(
			getMeterCompFuncWithExpectedKinds(t,
				[]common.ComputationKind{
					common.ComputationKindCreateCompositeValue,
					common.ComputationKindCreateArrayValue,
					common.ComputationKindTransferCompositeValue,
					common.ComputationKindTransferArrayValue,
					common.ComputationKindTransferCompositeValue,
				},
				[]uint{1, 1, 1, 1, 1},
			),
		),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.Address{},
		value,
	)

	arrayCopy := array.Transfer(
		inter,
		ReturnEmptyLocationRange,
		atree.Address(newOwner),
		false,
		nil,
	)
	array = arrayCopy.(*ArrayValue)

	value = array.Get(
		inter,
		ReturnEmptyLocationRange,
		0,
	).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayElement(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		value,
	)

	value = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArraySetIndex(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value1 := newTestCompositeValue(inter, oldOwner)
	value2 := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		value1,
	)

	value1 = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value1.GetOwner())
	assert.Equal(t, oldOwner, value2.GetOwner())

	array.Set(inter, ReturnEmptyLocationRange, 0, value2)

	value2 = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value1.GetOwner())
	assert.Equal(t, newOwner, value2.GetOwner())
}

func TestOwnerArrayAppend(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	array.Append(inter, ReturnEmptyLocationRange, value)

	value = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayInsert(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	array.Insert(inter, ReturnEmptyLocationRange, 0, value)

	value = array.Get(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayRemove(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	owner := common.Address{0x1}

	value := newTestCompositeValue(inter, owner)

	array := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		owner,
		value,
	)

	assert.Equal(t, owner, array.GetOwner())
	assert.Equal(t, owner, value.GetOwner())

	value = array.Remove(inter, ReturnEmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, owner, array.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestOwnerNewDictionary(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	assert.Equal(t, oldOwner, value.GetOwner())

	dictionary := NewDictionaryValue(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		keyValue, value,
	)

	// NOTE: keyValue is string, has no owner

	queriedValue, _ := dictionary.Get(inter, ReturnEmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, common.Address{}, dictionary.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestOwnerDictionary(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value,
	)

	// NOTE: keyValue is string, has no owner

	queriedValue, _ := dictionary.Get(inter, ReturnEmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryCopy(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
		WithOnMeterComputationFuncHandler(
			getMeterCompFuncWithExpectedKinds(t,
				[]common.ComputationKind{
					common.ComputationKindCreateCompositeValue,
					common.ComputationKindCreateDictionaryValue,
					common.ComputationKindTransferCompositeValue,
					common.ComputationKindTransferDictionaryValue,
					common.ComputationKindTransferCompositeValue,
				},
				[]uint{1, 1, 1, 1, 1},
			),
		),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value,
	)

	copyResult := dictionary.Transfer(
		inter,
		ReturnEmptyLocationRange,
		atree.Address{},
		false,
		nil,
	)

	dictionaryCopy := copyResult.(*DictionaryValue)

	queriedValue, _ := dictionaryCopy.Get(
		inter,
		ReturnEmptyLocationRange,
		keyValue,
	)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, common.Address{}, dictionaryCopy.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestOwnerDictionarySetSome(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	dictionary.SetKey(
		inter,
		ReturnEmptyLocationRange,
		keyValue,
		NewUnmeteredSomeValueNonCopying(value),
	)

	queriedValue, _ := dictionary.Get(inter, ReturnEmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryInsertNonExisting(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	existingValue := dictionary.Insert(
		inter,
		ReturnEmptyLocationRange,
		keyValue,
		value,
	)
	assert.Equal(t, NilValue{}, existingValue)

	queriedValue, _ := dictionary.Get(inter, ReturnEmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryRemove(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value1 := newTestCompositeValue(inter, oldOwner)
	value2 := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value1,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value1.GetOwner())
	assert.Equal(t, oldOwner, value2.GetOwner())

	existingValue := dictionary.Insert(
		inter,
		ReturnEmptyLocationRange,
		keyValue,
		value2,
	)
	require.IsType(t, &SomeValue{}, existingValue)
	innerValue := existingValue.(*SomeValue).InnerValue(inter, ReturnEmptyLocationRange)
	value1 = innerValue.(*CompositeValue)

	queriedValue, _ := dictionary.Get(inter, ReturnEmptyLocationRange, keyValue)
	value2 = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, common.Address{}, value1.GetOwner())
	assert.Equal(t, newOwner, value2.GetOwner())
}

func TestOwnerDictionaryInsertExisting(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration()
	elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		WithStorage(storage),
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	existingValue := dictionary.Remove(
		inter,
		ReturnEmptyLocationRange,
		keyValue,
	)
	require.IsType(t, &SomeValue{}, existingValue)
	innerValue := existingValue.(*SomeValue).InnerValue(inter, ReturnEmptyLocationRange)
	value = innerValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestOwnerNewComposite(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	oldOwner := common.Address{0x1}

	composite := newTestCompositeValue(inter, oldOwner)

	assert.Equal(t, oldOwner, composite.GetOwner())
}

func TestOwnerCompositeSet(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)
	composite := newTestCompositeValue(inter, newOwner)

	assert.Equal(t, oldOwner, value.GetOwner())
	assert.Equal(t, newOwner, composite.GetOwner())

	const fieldName = "test"

	composite.SetMember(inter, ReturnEmptyLocationRange, fieldName, value)

	value = composite.GetMember(inter, ReturnEmptyLocationRange, fieldName).(*CompositeValue)

	assert.Equal(t, newOwner, composite.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerCompositeCopy(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(inter, oldOwner)
	composite := newTestCompositeValue(inter, oldOwner)

	const fieldName = "test"

	composite.SetMember(
		inter,
		ReturnEmptyLocationRange,
		fieldName,
		value,
	)

	composite = composite.Transfer(
		inter,
		ReturnEmptyLocationRange,
		atree.Address{},
		false,
		nil,
	).(*CompositeValue)

	value = composite.GetMember(
		inter,
		ReturnEmptyLocationRange,
		fieldName,
	).(*CompositeValue)

	assert.Equal(t, common.Address{}, composite.GetOwner())
	assert.Equal(t, common.Address{}, value.GetOwner())
}

func TestStringer(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    Value
		expected string
	}

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUnmeteredUIntValueFromUint64(10),
			expected: "10",
		},
		"UInt8": {
			value:    NewUnmeteredUInt8Value(8),
			expected: "8",
		},
		"UInt16": {
			value:    NewUnmeteredUInt16Value(16),
			expected: "16",
		},
		"UInt32": {
			value:    NewUnmeteredUInt32Value(32),
			expected: "32",
		},
		"UInt64": {
			value:    NewUnmeteredUInt64Value(64),
			expected: "64",
		},
		"UInt128": {
			value:    NewUnmeteredUInt128ValueFromUint64(128),
			expected: "128",
		},
		"UInt256": {
			value:    NewUnmeteredUInt256ValueFromUint64(256),
			expected: "256",
		},
		"Int8": {
			value:    NewUnmeteredInt8Value(-8),
			expected: "-8",
		},
		"Int16": {
			value:    NewUnmeteredInt16Value(-16),
			expected: "-16",
		},
		"Int32": {
			value:    NewUnmeteredInt32Value(-32),
			expected: "-32",
		},
		"Int64": {
			value:    NewUnmeteredInt64Value(-64),
			expected: "-64",
		},
		"Int128": {
			value:    NewUnmeteredInt128ValueFromInt64(-128),
			expected: "-128",
		},
		"Int256": {
			value:    NewUnmeteredInt256ValueFromInt64(-256),
			expected: "-256",
		},
		"Word8": {
			value:    NewUnmeteredWord8Value(8),
			expected: "8",
		},
		"Word16": {
			value:    NewUnmeteredWord16Value(16),
			expected: "16",
		},
		"Word32": {
			value:    NewUnmeteredWord32Value(32),
			expected: "32",
		},
		"Word64": {
			value:    NewUnmeteredWord64Value(64),
			expected: "64",
		},
		"UFix64": {
			value:    NewUnmeteredUFix64ValueWithInteger(64),
			expected: "64.00000000",
		},
		"Fix64": {
			value:    NewUnmeteredFix64ValueWithInteger(-32),
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
			value:    NewUnmeteredSomeValueNonCopying(BoolValue(true)),
			expected: "true",
		},
		"nil": {
			value:    NilValue{},
			expected: "nil",
		},
		"String": {
			value:    NewUnmeteredStringValue("Flow ridah!"),
			expected: "\"Flow ridah!\"",
		},
		"Array": {
			value: NewArrayValue(
				newTestInterpreter(t),
				VariableSizedStaticType{
					Type: PrimitiveStaticTypeAnyStruct,
				},
				common.Address{},
				NewUnmeteredIntValueFromInt64(10),
				NewUnmeteredStringValue("TEST"),
			),
			expected: "[10, \"TEST\"]",
		},
		"Dictionary": {
			value: NewDictionaryValue(
				newTestInterpreter(t),
				DictionaryStaticType{
					KeyType:   PrimitiveStaticTypeString,
					ValueType: PrimitiveStaticTypeUInt8,
				},
				NewUnmeteredStringValue("a"), NewUnmeteredUInt8Value(42),
				NewUnmeteredStringValue("b"), NewUnmeteredUInt8Value(99),
			),
			expected: `{"b": 99, "a": 42}`,
		},
		"Address": {
			value:    NewUnmeteredAddressValueFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: "0x0000000000000001",
		},
		"composite": {
			value: func() Value {
				inter := newTestInterpreter(t)

				fields := []CompositeField{
					{
						Name:  "y",
						Value: NewUnmeteredStringValue("bar"),
					},
				}

				return NewCompositeValue(
					inter,
					utils.TestLocation,
					"Foo",
					common.CompositeKindResource,
					fields,
					common.Address{},
				)
			}(),
			expected: "S.test.Foo(y: \"bar\")",
		},
		"composite with custom stringer": {
			value: func() Value {
				inter := newTestInterpreter(t)

				fields := []CompositeField{
					{
						Name:  "y",
						Value: NewUnmeteredStringValue("bar"),
					},
				}

				compositeValue := NewCompositeValue(
					inter,
					utils.TestLocation,
					"Foo",
					common.CompositeKindResource,
					fields,
					common.Address{},
				)

				compositeValue.Stringer = func(_ *CompositeValue, _ SeenReferences) string {
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
			value: &CapabilityValue{
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Address:    NewUnmeteredAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
				BorrowType: PrimitiveStaticTypeInt,
			},
			expected: "Capability<Int>(address: 0x0000000102030405, path: /storage/foo)",
		},
		"Capability without borrow type": {
			value: &CapabilityValue{
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Address: NewUnmeteredAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
			},
			expected: "Capability(address: 0x0000000102030405, path: /storage/foo)",
		},
		"Recursive ephemeral reference (array)": {
			value: func() Value {
				array := NewArrayValue(
					newTestInterpreter(t),
					VariableSizedStaticType{
						Type: PrimitiveStaticTypeAnyStruct,
					},
					common.Address{},
				)
				arrayRef := &EphemeralReferenceValue{Value: array}
				array.Insert(newTestInterpreter(t), ReturnEmptyLocationRange, 0, arrayRef)
				return array
			}(),
			expected: `[[...]]`,
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

	inter := newTestInterpreter(t)

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
	value = NewUnmeteredIntValueFromInt64(42)
	value = NewUnmeteredSomeValueNonCopying(value)
	value = NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.Address{},
		value,
	)

	value = NewDictionaryValue(
		inter,
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAny,
		},
		NewUnmeteredStringValue("42"), value,
	)

	fields := []CompositeField{
		{
			Name:  "foo",
			Value: value,
		},
	}

	value = NewCompositeValue(
		inter,
		utils.TestLocation,
		"Foo",
		common.CompositeKindStructure,
		fields,
		common.Address{},
	)

	value.Accept(inter, visitor)

	require.Equal(t, 1, intVisits)
	require.Equal(t, 1, stringVisits)
}

func TestGetHashInput(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    HashableValue
		expected []byte
	}

	stringerTests := map[string]testCase{
		"UInt": {
			value:    NewUnmeteredUIntValueFromUint64(10),
			expected: []byte{byte(HashInputTypeUInt), 10},
		},
		"UInt min": {
			value:    NewUnmeteredUIntValueFromUint64(0),
			expected: []byte{byte(HashInputTypeUInt), 0},
		},
		"UInt large": {
			value:    NewUnmeteredUIntValueFromBigInt(sema.UInt256TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeUInt)}, sema.UInt256TypeMaxIntBig.Bytes()...),
		},
		"UInt8": {
			value:    NewUnmeteredUInt8Value(8),
			expected: []byte{byte(HashInputTypeUInt8), 8},
		},
		"UInt8 min": {
			value:    NewUnmeteredUInt8Value(0),
			expected: []byte{byte(HashInputTypeUInt8), 0},
		},
		"UInt8 max": {
			value:    NewUnmeteredUInt8Value(math.MaxUint8),
			expected: []byte{byte(HashInputTypeUInt8), 0xff},
		},
		"UInt16": {
			value:    NewUnmeteredUInt16Value(16),
			expected: []byte{byte(HashInputTypeUInt16), 0, 16},
		},
		"UInt16 min": {
			value:    NewUnmeteredUInt16Value(0),
			expected: []byte{byte(HashInputTypeUInt16), 0, 0},
		},
		"UInt16 max": {
			value:    NewUnmeteredUInt16Value(math.MaxUint16),
			expected: []byte{byte(HashInputTypeUInt16), 0xff, 0xff},
		},
		"UInt32": {
			value:    NewUnmeteredUInt32Value(32),
			expected: []byte{byte(HashInputTypeUInt32), 0, 0, 0, 32},
		},
		"UInt32 min": {
			value:    NewUnmeteredUInt32Value(0),
			expected: []byte{byte(HashInputTypeUInt32), 0, 0, 0, 0},
		},
		"UInt32 max": {
			value:    NewUnmeteredUInt32Value(math.MaxUint32),
			expected: []byte{byte(HashInputTypeUInt32), 0xff, 0xff, 0xff, 0xff},
		},
		"UInt64": {
			value:    NewUnmeteredUInt64Value(64),
			expected: []byte{byte(HashInputTypeUInt64), 0, 0, 0, 0, 0, 0, 0, 64},
		},
		"UInt64 min": {
			value:    NewUnmeteredUInt64Value(0),
			expected: []byte{byte(HashInputTypeUInt64), 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"UInt64 max": {
			value:    NewUnmeteredUInt64Value(math.MaxUint64),
			expected: []byte{byte(HashInputTypeUInt64), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		"UInt128": {
			value:    NewUnmeteredUInt128ValueFromUint64(128),
			expected: []byte{byte(HashInputTypeUInt128), 128},
		},
		"UInt128 min": {
			value:    NewUnmeteredUInt128ValueFromUint64(0),
			expected: append([]byte{byte(HashInputTypeUInt128)}, 0),
		},
		"UInt128 max": {
			value:    NewUnmeteredUInt128ValueFromBigInt(sema.UInt128TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeUInt128)}, sema.UInt128TypeMaxIntBig.Bytes()...),
		},
		"UInt256": {
			value:    NewUnmeteredUInt256ValueFromUint64(256),
			expected: []byte{byte(HashInputTypeUInt256), 1, 0},
		},
		"UInt256 min": {
			value:    NewUnmeteredUInt256ValueFromUint64(0),
			expected: append([]byte{byte(HashInputTypeUInt256)}, 0),
		},
		"UInt256 max": {
			value:    NewUnmeteredUInt256ValueFromBigInt(sema.UInt256TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeUInt256)}, sema.UInt256TypeMaxIntBig.Bytes()...),
		},
		"Int": {
			value:    NewUnmeteredIntValueFromInt64(10),
			expected: []byte{byte(HashInputTypeInt), 10},
		},
		"Int small": {
			value:    NewUnmeteredIntValueFromBigInt(sema.Int256TypeMinIntBig),
			expected: append([]byte{byte(HashInputTypeInt)}, sema.Int256TypeMinIntBig.Bytes()...),
		},
		"Int large": {
			value:    NewUnmeteredIntValueFromBigInt(sema.Int256TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeInt)}, sema.Int256TypeMaxIntBig.Bytes()...),
		},
		"Int8": {
			value:    NewUnmeteredInt8Value(-8),
			expected: []byte{byte(HashInputTypeInt8), 0xf8},
		},
		"Int8 min": {
			value:    NewUnmeteredInt8Value(math.MinInt8),
			expected: []byte{byte(HashInputTypeInt8), 0x80},
		},
		"Int8 max": {
			value:    NewUnmeteredInt8Value(math.MaxInt8),
			expected: []byte{byte(HashInputTypeInt8), 0x7f},
		},
		"Int16": {
			value:    NewUnmeteredInt16Value(-16),
			expected: []byte{byte(HashInputTypeInt16), 0xff, 0xf0},
		},
		"Int16 min": {
			value:    NewUnmeteredInt16Value(math.MinInt16),
			expected: []byte{byte(HashInputTypeInt16), 0x80, 0x00},
		},
		"Int16 max": {
			value:    NewUnmeteredInt16Value(math.MaxInt16),
			expected: []byte{byte(HashInputTypeInt16), 0x7f, 0xff},
		},
		"Int32": {
			value:    NewUnmeteredInt32Value(-32),
			expected: []byte{byte(HashInputTypeInt32), 0xff, 0xff, 0xff, 0xe0},
		},
		"Int32 min": {
			value:    NewUnmeteredInt32Value(math.MinInt32),
			expected: []byte{byte(HashInputTypeInt32), 0x80, 0x00, 0x00, 0x00},
		},
		"Int32 max": {
			value:    NewUnmeteredInt32Value(math.MaxInt32),
			expected: []byte{byte(HashInputTypeInt32), 0x7f, 0xff, 0xff, 0xff},
		},
		"Int64": {
			value:    NewUnmeteredInt64Value(-64),
			expected: []byte{byte(HashInputTypeInt64), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xc0},
		},
		"Int64 min": {
			value:    NewUnmeteredInt64Value(math.MinInt64),
			expected: []byte{byte(HashInputTypeInt64), 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		"Int64 max": {
			value:    NewUnmeteredInt64Value(math.MaxInt64),
			expected: []byte{byte(HashInputTypeInt64), 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		"Int128": {
			value:    NewUnmeteredInt128ValueFromInt64(-128),
			expected: []byte{byte(HashInputTypeInt128), 0x80},
		},
		"Int128 min": {
			value:    NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMinIntBig),
			expected: append([]byte{byte(HashInputTypeInt128)}, sema.Int128TypeMinIntBig.Bytes()...),
		},
		"Int128 max": {
			value:    NewUnmeteredInt128ValueFromBigInt(sema.Int128TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeInt128)}, sema.Int128TypeMaxIntBig.Bytes()...),
		},
		"Int256": {
			value:    NewUnmeteredInt256ValueFromInt64(-256),
			expected: []byte{byte(HashInputTypeInt256), 0xff, 0x0},
		},
		"Int256 min": {
			value:    NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMinIntBig),
			expected: append([]byte{byte(HashInputTypeInt256)}, sema.Int256TypeMinIntBig.Bytes()...),
		},
		"Int256 max": {
			value:    NewUnmeteredInt256ValueFromBigInt(sema.Int256TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeInt256)}, sema.Int256TypeMaxIntBig.Bytes()...),
		},
		"Word8": {
			value:    NewUnmeteredWord8Value(8),
			expected: []byte{byte(HashInputTypeWord8), 8},
		},
		"Word8 min": {
			value:    NewUnmeteredWord8Value(0),
			expected: []byte{byte(HashInputTypeWord8), 0},
		},
		"Word8 max": {
			value:    NewUnmeteredWord8Value(255),
			expected: []byte{byte(HashInputTypeWord8), 0xff},
		},
		"Word16": {
			value:    NewUnmeteredWord16Value(16),
			expected: []byte{byte(HashInputTypeWord16), 0, 16},
		},
		"Word16 min": {
			value:    NewUnmeteredWord16Value(0),
			expected: []byte{byte(HashInputTypeWord16), 0, 0},
		},
		"Word16 max": {
			value:    NewUnmeteredWord16Value(math.MaxUint16),
			expected: []byte{byte(HashInputTypeWord16), 0xff, 0xff},
		},
		"Word32": {
			value:    NewUnmeteredWord32Value(32),
			expected: []byte{byte(HashInputTypeWord32), 0, 0, 0, 32},
		},
		"Word32 min": {
			value:    NewUnmeteredWord32Value(0),
			expected: []byte{byte(HashInputTypeWord32), 0, 0, 0, 0},
		},
		"Word32 max": {
			value:    NewUnmeteredWord32Value(math.MaxUint32),
			expected: []byte{byte(HashInputTypeWord32), 0xff, 0xff, 0xff, 0xff},
		},
		"Word64": {
			value:    NewUnmeteredWord64Value(64),
			expected: []byte{byte(HashInputTypeWord64), 0, 0, 0, 0, 0, 0, 0, 64},
		},
		"Word64 min": {
			value:    NewUnmeteredWord64Value(0),
			expected: []byte{byte(HashInputTypeWord64), 0, 0, 0, 0, 0, 0, 0, 0},
		},
		"Word64 max": {
			value:    NewUnmeteredWord64Value(math.MaxUint64),
			expected: []byte{byte(HashInputTypeWord64), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		"UFix64": {
			value:    NewUnmeteredUFix64ValueWithInteger(64),
			expected: []byte{byte(HashInputTypeUFix64), 0x0, 0x0, 0x0, 0x1, 0x7d, 0x78, 0x40, 0x0},
		},
		"UFix64 min": {
			value:    NewUnmeteredUFix64ValueWithInteger(0),
			expected: []byte{byte(HashInputTypeUFix64), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"UFix64 max": {
			value:    NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMaxInt),
			expected: []byte{byte(HashInputTypeUFix64), 0xff, 0xff, 0xff, 0xff, 0xff, 0x6e, 0x41, 0x0},
		},
		"Fix64": {
			value:    NewUnmeteredFix64ValueWithInteger(-32),
			expected: []byte{byte(HashInputTypeFix64), 0xff, 0xff, 0xff, 0xff, 0x41, 0x43, 0xe0, 0x0},
		},
		"Fix64 min": {
			value:    NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMinInt),
			expected: []byte{byte(HashInputTypeFix64), 0x80, 0x0, 0x0, 0x0, 0x03, 0x43, 0xd0, 0x0},
		},
		"Fix64 max": {
			value:    NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMaxInt),
			expected: []byte{byte(HashInputTypeFix64), 0x7f, 0xff, 0xff, 0xff, 0xfc, 0xbc, 0x30, 0x00},
		},
		"true": {
			value:    BoolValue(true),
			expected: []byte{byte(HashInputTypeBool), 1},
		},
		"false": {
			value:    BoolValue(false),
			expected: []byte{byte(HashInputTypeBool), 0},
		},
		"String": {
			value: NewUnmeteredStringValue("Flow ridah!"),
			expected: []byte{
				byte(HashInputTypeString),
				0x46, 0x6c, 0x6f, 0x77, 0x20, 0x72, 0x69, 0x64, 0x61, 0x68, 0x21,
			},
		},
		"String long": {
			value: NewUnmeteredStringValue(strings.Repeat("a", 32)),
			expected: append([]byte{byte(HashInputTypeString)},
				[]byte(strings.Repeat("a", 32))...,
			),
		},
		"Character": {
			value: NewUnmeteredCharacterValue("ᄀᄀᄀ각ᆨᆨ"),
			expected: []byte{
				byte(HashInputTypeCharacter),
				0xe1, 0x84, 0x80, 0xe1, 0x84, 0x80, 0xe1, 0x84, 0x80, 0xea, 0xb0, 0x81, 0xe1, 0x86, 0xa8, 0xe1, 0x86, 0xa8,
			},
		},
		"Address": {
			value:    NewUnmeteredAddressValueFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			expected: []byte{byte(HashInputTypeAddress), 0, 0, 0, 0, 0, 0, 0, 1},
		},
		"enum": {
			value: func() HashableValue {
				inter := newTestInterpreter(t)

				fields := []CompositeField{
					{
						Name:  "rawValue",
						Value: NewUnmeteredUInt8Value(42),
					},
				}
				return NewCompositeValue(
					inter,
					utils.TestLocation,
					"Foo",
					common.CompositeKindEnum,
					fields,
					common.Address{},
				)
			}(),
			expected: []byte{
				byte(HashInputTypeEnum),
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				byte(HashInputTypeUInt8),
				42,
			},
		},
		"enum long identifier": {
			value: func() HashableValue {
				inter := newTestInterpreter(t)

				fields := []CompositeField{
					{
						Name:  "rawValue",
						Value: NewUnmeteredUInt8Value(42),
					},
				}
				return NewCompositeValue(
					inter,
					utils.TestLocation,
					strings.Repeat("a", 32),
					common.CompositeKindEnum,
					fields,
					common.Address{},
				)
			}(),
			expected: append(
				append([]byte{byte(HashInputTypeEnum)},
					append(
						// S.test.
						[]byte{0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e},
						// identifier
						[]byte(strings.Repeat("a", 32))...,
					)...),
				byte(HashInputTypeUInt8),
				42,
			),
		},
		"Path": {
			value: PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			expected: []byte{
				byte(HashInputTypePath),
				// domain: storage
				0x1,
				// identifier: "foo"
				0x66, 0x6f, 0x6f,
			},
		},
		"Path long identifier": {
			value: PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: strings.Repeat("a", 32),
			},
			expected: append(
				[]byte{byte(HashInputTypePath),
					// domain: storage
					0x1},
				// identifier: aaa...
				[]byte(strings.Repeat("a", 32))...,
			),
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			var scratch [32]byte

			inter := newTestInterpreter(t)

			actual := testCase.value.HashInput(inter, ReturnEmptyLocationRange, scratch[:])

			assert.Equal(t,
				testCase.expected,
				actual,
			)
		})
	}

	for name, testCase := range stringerTests {
		test(name, testCase)
	}
}

func TestBlockValue(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	block := NewBlockValue(
		inter,
		4,
		5,
		NewArrayValue(inter, ByteArrayStaticType, common.Address{}),
		5.0,
	)

	// static type test
	var actualTs = block.Fields[sema.BlockTypeTimestampFieldName]
	const expectedTs UFix64Value = 5.0
	assert.Equal(t, expectedTs, actualTs)
}

func TestEphemeralReferenceTypeConformance(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	// Obtain a self referencing (cyclic) ephemeral reference value.

	code := `
        pub fun getEphemeralRef(): &Foo {
            var foo = Foo()
            var fooRef = &foo as &Foo

            // Create the cyclic reference
            fooRef.bar = fooRef

            return fooRef
        }

        pub struct Foo {

            pub(set) var bar: &Foo?

            init() {
                self.bar = nil
            }
        }`

	checker, err := checkerUtils.ParseAndCheckWithOptions(t,
		code,
		checkerUtils.ParseAndCheckOptions{},
	)

	require.NoError(t, err)

	inter, err := NewInterpreter(
		ProgramFromChecker(checker),
		checker.Location,
		WithStorage(storage),
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("getEphemeralRef")
	require.NoError(t, err)
	require.IsType(t, &EphemeralReferenceValue{}, value)

	dynamicType := value.DynamicType(inter, SeenReferences{})

	// Check the dynamic type conformance on a cyclic value.
	conforms := value.ConformsToDynamicType(
		inter,
		ReturnEmptyLocationRange,
		dynamicType,
		TypeConformanceResults{},
	)
	assert.True(t, conforms)

	// Check against a non-conforming type
	conforms = value.ConformsToDynamicType(
		inter,
		ReturnEmptyLocationRange,
		EphemeralReferenceDynamicType{},
		TypeConformanceResults{},
	)
	assert.False(t, conforms)
}

func TestCapabilityValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal, borrow type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				BorrowType: PrimitiveStaticTypeInt,
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				&CapabilityValue{
					Address: AddressValue{0x1},
					Path: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
					BorrowType: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("equal, no borrow type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				&CapabilityValue{
					Address: AddressValue{0x1},
					Path: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
				},
			),
		)
	})

	t.Run("different paths", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test1",
				},
				BorrowType: PrimitiveStaticTypeInt,
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				&CapabilityValue{
					Address: AddressValue{0x1},
					Path: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test2",
					},
					BorrowType: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different addresses", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				BorrowType: PrimitiveStaticTypeInt,
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				&CapabilityValue{
					Address: AddressValue{0x2},
					Path: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
					BorrowType: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different borrow types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				BorrowType: PrimitiveStaticTypeInt,
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				&CapabilityValue{
					Address: AddressValue{0x1},
					Path: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
					BorrowType: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			(&CapabilityValue{
				Address: AddressValue{0x1},
				Path: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				BorrowType: PrimitiveStaticTypeInt,
			}).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("test"),
			),
		)
	})
}

func TestAddressValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			AddressValue{0x1}.Equal(
				inter,
				ReturnEmptyLocationRange,
				AddressValue{0x1},
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			AddressValue{0x1}.Equal(
				inter,
				ReturnEmptyLocationRange,
				AddressValue{0x2},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			AddressValue{0x1}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestBoolValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal true", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			BoolValue(true).Equal(
				inter,
				ReturnEmptyLocationRange,
				BoolValue(true),
			),
		)
	})

	t.Run("equal false", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			BoolValue(false).Equal(
				inter,
				ReturnEmptyLocationRange,
				BoolValue(false),
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			BoolValue(true).Equal(
				inter,
				ReturnEmptyLocationRange,
				BoolValue(false),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			BoolValue(true).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestStringValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewUnmeteredStringValue("test").Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("test"),
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredStringValue("test").Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("foo"),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredStringValue("1").Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestNilValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NilValue{}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NilValue{},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NilValue{}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(0),
			),
		)
	})
}

func TestSomeValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewUnmeteredSomeValueNonCopying(NewUnmeteredStringValue("test")).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredSomeValueNonCopying(NewUnmeteredStringValue("test")),
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredSomeValueNonCopying(NewUnmeteredStringValue("test")).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredSomeValueNonCopying(NewUnmeteredStringValue("foo")),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredSomeValueNonCopying(NewUnmeteredStringValue("1")).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestTypeValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			TypeValue{
				Type: PrimitiveStaticTypeString,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				TypeValue{
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			TypeValue{
				Type: PrimitiveStaticTypeString,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				TypeValue{
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			TypeValue{
				Type: PrimitiveStaticTypeString,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("String"),
			),
		)
	})
}

func TestPathValue_Equal(t *testing.T) {

	t.Parallel()

	for _, domain := range common.AllPathDomains {

		t.Run(fmt.Sprintf("equal, %s", domain), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.True(t,
				PathValue{
					Domain:     domain,
					Identifier: "test",
				}.Equal(
					inter,
					ReturnEmptyLocationRange,
					PathValue{
						Domain:     domain,
						Identifier: "test",
					},
				),
			)
		})
	}

	for _, domain := range common.AllPathDomains {
		for _, otherDomain := range common.AllPathDomains {

			if domain == otherDomain {
				continue
			}

			t.Run(fmt.Sprintf("different domains %s %s", domain, otherDomain), func(t *testing.T) {

				inter := newTestInterpreter(t)

				require.False(t,
					PathValue{
						Domain:     domain,
						Identifier: "test",
					}.Equal(
						inter,
						ReturnEmptyLocationRange,
						PathValue{
							Domain:     otherDomain,
							Identifier: "test",
						},
					),
				)
			})
		}
	}

	for _, domain := range common.AllPathDomains {

		t.Run(fmt.Sprintf("different identifiers, %s", domain), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.False(t,
				PathValue{
					Domain:     domain,
					Identifier: "test1",
				}.Equal(
					inter,
					ReturnEmptyLocationRange,
					PathValue{
						Domain:     domain,
						Identifier: "test2",
					},
				),
			)
		})
	}

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("/storage/test"),
			),
		)
	})
}

func TestLinkValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal, borrow type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			LinkValue{
				TargetPath: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				LinkValue{
					TargetPath: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different paths", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			LinkValue{
				TargetPath: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test1",
				},
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				LinkValue{
					TargetPath: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test2",
					},
					Type: PrimitiveStaticTypeInt,
				},
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			LinkValue{
				TargetPath: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				LinkValue{
					TargetPath: PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "test",
					},
					Type: PrimitiveStaticTypeString,
				},
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			LinkValue{
				TargetPath: PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				Type: PrimitiveStaticTypeInt,
			}.Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("test"),
			),
		)
	})
}

func TestArrayValue_Equal(t *testing.T) {

	t.Parallel()

	uint8ArrayStaticType := VariableSizedStaticType{
		Type: PrimitiveStaticTypeUInt8,
	}

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint8ArrayStaticType,
					common.Address{},
					NewUnmeteredUInt8Value(1),
					NewUnmeteredUInt8Value(2),
				),
			),
		)
	})

	t.Run("different elements", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint8ArrayStaticType,
					common.Address{},
					NewUnmeteredUInt8Value(2),
					NewUnmeteredUInt8Value(3),
				),
			),
		)
	})

	t.Run("more elements", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
				NewUnmeteredUInt8Value(1),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint8ArrayStaticType,
					common.Address{},
					NewUnmeteredUInt8Value(1),
					NewUnmeteredUInt8Value(2),
				),
			),
		)
	})

	t.Run("fewer elements", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint8ArrayStaticType,
					common.Address{},
					NewUnmeteredUInt8Value(1),
				),
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		uint16ArrayStaticType := VariableSizedStaticType{
			Type: PrimitiveStaticTypeUInt16,
		}

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint16ArrayStaticType,
					common.Address{},
				),
			),
		)
	})

	t.Run("no type, type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				nil,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					uint8ArrayStaticType,
					common.Address{},
				),
			),
		)
	})

	t.Run("type, no type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					nil,
					common.Address{},
				),
			),
		)
	})

	t.Run("no types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewArrayValue(
				inter,
				nil,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					nil,
					common.Address{},
				),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewArrayValue(
				inter,
				uint8ArrayStaticType,
				common.Address{},
				NewUnmeteredUInt8Value(1),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestDictionaryValue_Equal(t *testing.T) {

	t.Parallel()

	byteStringDictionaryType := DictionaryStaticType{
		KeyType:   PrimitiveStaticTypeUInt8,
		ValueType: PrimitiveStaticTypeString,
	}

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					byteStringDictionaryType,
					NewUnmeteredUInt8Value(1),
					NewUnmeteredStringValue("1"),
					NewUnmeteredUInt8Value(2),
					NewUnmeteredStringValue("2"),
				),
			),
		)
	})

	t.Run("different keys", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					byteStringDictionaryType,
					NewUnmeteredUInt8Value(2),
					NewUnmeteredStringValue("1"),
					NewUnmeteredUInt8Value(3),
					NewUnmeteredStringValue("2"),
				),
			),
		)
	})

	t.Run("different values", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					byteStringDictionaryType,
					NewUnmeteredUInt8Value(1),
					NewUnmeteredStringValue("2"),
					NewUnmeteredUInt8Value(2),
					NewUnmeteredStringValue("3"),
				),
			),
		)
	})

	t.Run("more elements", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					byteStringDictionaryType,
					NewUnmeteredUInt8Value(1),
					NewUnmeteredStringValue("1"),
					NewUnmeteredUInt8Value(2),
					NewUnmeteredStringValue("2"),
				),
			),
		)
	})

	t.Run("fewer elements", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					byteStringDictionaryType,
					NewUnmeteredUInt8Value(1),
					NewUnmeteredStringValue("1"),
				),
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		stringByteDictionaryStaticType := DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeUInt8,
		}

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewDictionaryValue(
					inter,
					stringByteDictionaryStaticType,
				),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewDictionaryValue(
				inter,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewArrayValue(
					inter,
					ByteArrayStaticType,
					common.Address{},
					NewUnmeteredUInt8Value(1),
					NewUnmeteredUInt8Value(2),
				),
			),
		)
	})
}

func TestCompositeValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.True(t,
			NewCompositeValue(
				inter,
				utils.TestLocation,
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					utils.TestLocation,
					"X",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("different location", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("B"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("different identifier", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("A"),
					"Y",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("different fields", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("b"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("more fields", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
			{
				Name:  "b",
				Value: NewUnmeteredStringValue("b"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("fewer fields", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
			{
				Name:  "b",
				Value: NewUnmeteredStringValue("b"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("different composite kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		fields2 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewCompositeValue(
					inter,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindResource,
					fields2,
					common.Address{},
				),
			),
		)
	})

	t.Run("different composite kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		fields1 := []CompositeField{
			{
				Name:  "a",
				Value: NewUnmeteredStringValue("a"),
			},
		}

		require.False(t,
			NewCompositeValue(
				inter,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.Address{},
			).Equal(
				inter,
				ReturnEmptyLocationRange,
				NewUnmeteredStringValue("test"),
			),
		)
	})
}

func TestNumberValue_Equal(t *testing.T) {

	t.Parallel()

	testValues := map[string]EquatableValue{
		"UInt":    NewUnmeteredUIntValueFromUint64(10),
		"UInt8":   NewUnmeteredUInt8Value(8),
		"UInt16":  NewUnmeteredUInt16Value(16),
		"UInt32":  NewUnmeteredUInt32Value(32),
		"UInt64":  NewUnmeteredUInt64Value(64),
		"UInt128": NewUnmeteredUInt128ValueFromUint64(128),
		"UInt256": NewUnmeteredUInt256ValueFromUint64(256),
		"Int8":    NewUnmeteredInt8Value(-8),
		"Int16":   NewUnmeteredInt16Value(-16),
		"Int32":   NewUnmeteredInt32Value(-32),
		"Int64":   NewUnmeteredInt64Value(-64),
		"Int128":  NewUnmeteredInt128ValueFromInt64(-128),
		"Int256":  NewUnmeteredInt256ValueFromInt64(-256),
		"Word8":   NewUnmeteredWord8Value(8),
		"Word16":  NewUnmeteredWord16Value(16),
		"Word32":  NewUnmeteredWord32Value(32),
		"Word64":  NewUnmeteredWord64Value(64),
		"UFix64":  NewUnmeteredUFix64ValueWithInteger(64),
		"Fix64":   NewUnmeteredFix64ValueWithInteger(-32),
	}

	for name, value := range testValues {

		t.Run(fmt.Sprintf("equal, %s", name), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.True(t,
				value.Equal(
					inter,
					ReturnEmptyLocationRange,
					value,
				),
			)
		})
	}

	for name, value := range testValues {
		for otherName, otherValue := range testValues {

			if name == otherName {
				continue
			}

			t.Run(fmt.Sprintf("unequal, %s %s", name, otherName), func(t *testing.T) {

				inter := newTestInterpreter(t)

				require.False(t,
					value.Equal(
						inter,
						ReturnEmptyLocationRange,
						otherValue,
					),
				)
			})
		}
	}

	for name, value := range testValues {

		t.Run(fmt.Sprintf("different kind, %s", name), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.False(t,
				value.Equal(
					inter,
					ReturnEmptyLocationRange,
					AddressValue{0x1},
				),
			)
		})
	}
}

func TestPublicKeyValue(t *testing.T) {

	t.Parallel()

	t.Run("Stringer output includes public key value", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			utils.TestLocation,
			WithStorage(storage),
			WithPublicKeyValidationHandler(runtime.DoNotValidatePublicKey),
		)
		require.NoError(t, err)

		publicKey := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
			common.Address{},
			NewUnmeteredIntValueFromInt64(1),
			NewUnmeteredIntValueFromInt64(7),
			NewUnmeteredIntValueFromInt64(3),
		)

		sigAlgo := stdlib.NewSignatureAlgorithmCase(
			inter,
			sema.SignatureAlgorithmECDSA_secp256k1.RawValue(),
		)

		key := NewPublicKeyValue(
			inter,
			ReturnEmptyLocationRange,
			publicKey,
			sigAlgo,
			inter.PublicKeyValidationHandler,
		)

		require.Equal(t,
			"PublicKey(publicKey: [1, 7, 3], signatureAlgorithm: SignatureAlgorithm(rawValue: 2))",
			key.String(),
		)
	})

	t.Run("Panics when PublicKey is invalid", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		fakeError := fakeError{}

		inter, err := NewInterpreter(
			nil,
			utils.TestLocation,
			WithStorage(storage),
			WithPublicKeyValidationHandler(
				func(_ *Interpreter, _ func() LocationRange, _ *CompositeValue) error {
					return fakeError
				},
			),
		)
		require.NoError(t, err)

		publicKeyBytes := []byte{1, 7, 3}

		publicKey := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
			common.Address{},
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[0])),
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[1])),
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[2])),
		)

		sigAlgo := stdlib.NewSignatureAlgorithmCase(
			inter,
			sema.SignatureAlgorithmECDSA_secp256k1.RawValue(),
		)

		assert.PanicsWithError(t,
			(InvalidPublicKeyError{PublicKey: publicKey, Err: fakeError}).Error(),
			func() {
				_ = NewPublicKeyValue(
					inter,
					ReturnEmptyLocationRange,
					publicKey,
					sigAlgo,
					inter.PublicKeyValidationHandler,
				)
			})
	})
}

func TestHashable(t *testing.T) {

	// Assert that all Value and DynamicType implementations are hashable

	pkgs, err := packages.Load(
		&packages.Config{
			// https://github.com/golang/go/issues/45218
			Mode: packages.NeedImports | packages.NeedTypes,
		},
		"github.com/onflow/cadence/runtime/interpreter",
	)
	require.NoError(t, err)

	pkg := pkgs[0]
	scope := pkg.Types.Scope()

	test := func(interfaceName string) {

		t.Run(interfaceName, func(t *testing.T) {

			interfaceType, ok := scope.Lookup(interfaceName).Type().Underlying().(*types.Interface)
			require.True(t, ok)

			for _, name := range scope.Names() {
				object := scope.Lookup(name)
				_, ok := object.(*types.TypeName)
				if !ok {
					continue
				}

				implementationType := object.Type()
				if !types.Implements(implementationType, interfaceType) {
					continue
				}

				err := checkHashable(implementationType)
				if !assert.NoError(t,
					err,
					"%s implementation is not hashable: %s",
					interfaceType.String(),
					implementationType,
				) {
					continue
				}
			}
		})
	}

	test("Value")
	test("DynamicType")
}

func checkHashable(ty types.Type) error {

	// TODO: extend the notion of unhashable types,
	//  see https://github.com/golang/go/blob/a22e3172200d4bdd0afcbbe6564dbb67fea4b03a/src/runtime/alg.go#L144

	switch ty := ty.(type) {
	case *types.Basic:
		switch ty.Kind() {
		case types.Bool,
			types.Int,
			types.Int8,
			types.Int16,
			types.Int32,
			types.Int64,
			types.Uint,
			types.Uint8,
			types.Uint16,
			types.Uint32,
			types.Uint64,
			types.Float32,
			types.Float64,
			types.String:
			return nil
		}
	case *types.Pointer,
		*types.Array,
		*types.Interface:
		return nil

	case *types.Struct:
		numFields := ty.NumFields()
		for i := 0; i < numFields; i++ {
			field := ty.Field(i)
			fieldTy := field.Type()
			err := checkHashable(fieldTy)
			if err != nil {
				return fmt.Errorf(
					"struct type has unhashable field %s: %w",
					field.Name(),
					err,
				)
			}
		}
		return nil

	case *types.Named:
		return checkHashable(ty.Underlying())
	}

	return fmt.Errorf(
		"type %s is potentially not hashable",
		ty.String(),
	)
}

func newTestInterpreter(tb testing.TB) *Interpreter {

	storage := newUnmeteredInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		utils.TestLocation,
		WithStorage(storage),
		WithAtreeValueValidationEnabled(true),
		WithAtreeStorageValidationEnabled(true),
	)
	require.NoError(tb, err)

	return inter
}

func TestNonStorable(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	code := `
      pub struct Foo {

          let bar: &Int?

          init() {
              self.bar = &1 as &Int
          }
      }

      fun foo(): &Int? {
          return Foo().bar
      }
    `

	checker, err := checkerUtils.ParseAndCheckWithOptions(t,
		code,
		checkerUtils.ParseAndCheckOptions{},
	)

	require.NoError(t, err)

	inter, err := NewInterpreter(
		ProgramFromChecker(checker),
		checker.Location,
		WithStorage(storage),
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("foo")
	require.NoError(t, err)

}

type fakeError struct{}

func (fakeError) Error() string {
	return "fake error for testing"
}
