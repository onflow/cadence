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

package interpreter_test

import (
	"fmt"
	"go/types"
	"math"
	"math/big"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/onflow/atree"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func newTestCompositeValue(inter *Interpreter, owner common.Address) *CompositeValue {
	return NewCompositeValue(
		inter,
		EmptyLocationRange,
		TestLocation,
		"Test",
		common.CompositeKindStructure,
		nil,
		owner,
	)
}

var testCompositeValueType = &sema.CompositeType{
	Location:   TestLocation,
	Identifier: "Test",
	Kind:       common.CompositeKindStructure,
	Members:    &sema.StringMemberOrderedMap{},
}

func getMeterCompFuncWithExpectedKinds(
	t *testing.T,
	kinds []common.ComputationKind,
	intensities []uint64,
) computationGaugeFunc {
	if len(kinds) != len(intensities) {
		t.Fatal("size of kinds doesn't match size of intensities")
	}
	expectedCompKindsIndex := 0
	return func(usage common.ComputationUsage) error {
		if expectedCompKindsIndex >= len(kinds) {
			t.Fatal("received an extra meterComputation call")
		}
		assert.Equal(t, kinds[expectedCompKindsIndex], usage.Kind)
		assert.Equal(t, intensities[expectedCompKindsIndex], usage.Intensity)
		expectedCompKindsIndex++
		return nil
	}
}

func TestOwnerNewArray(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}

	value := newTestCompositeValue(inter, oldOwner)

	assert.Equal(t, oldOwner, value.GetOwner())

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		value,
	)

	value = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, common.ZeroAddress, array.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
}

func TestOwnerArrayDeepCopy(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{
			Storage: storage,
			ComputationGauge: getMeterCompFuncWithExpectedKinds(t,
				[]common.ComputationKind{
					common.ComputationKindCreateCompositeValue,
					common.ComputationKindCreateArrayValue,
					common.ComputationKindTransferCompositeValue,
					common.ComputationKindTransferArrayValue,
					common.ComputationKindTransferCompositeValue,
				},
				[]uint64{1, 1, 1, 1, 1},
			),
		},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		value,
	)

	arrayCopy := array.Transfer(
		inter,
		EmptyLocationRange,
		atree.Address(newOwner),
		false,
		nil,
		nil,
		true, // array is standalone.
	)
	array = arrayCopy.(*ArrayValue)

	value = array.Get(
		inter,
		EmptyLocationRange,
		0,
	).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayElement(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		value,
	)

	value = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArraySetIndex(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value1 := newTestCompositeValue(inter, oldOwner)
	value2 := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		value1,
	)

	value1 = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value1.GetOwner())
	assert.Equal(t, oldOwner, value2.GetOwner())

	array.Set(inter, EmptyLocationRange, 0, value2)

	value2 = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value1.GetOwner())
	assert.Equal(t, newOwner, value2.GetOwner())
}

func TestOwnerArrayAppend(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	array.Append(inter, EmptyLocationRange, value)

	value = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayInsert(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	value := newTestCompositeValue(inter, oldOwner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	array.Insert(inter, EmptyLocationRange, 0, value)

	value = array.Get(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, newOwner, array.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerArrayRemove(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	owner := common.Address{0x1}

	value := newTestCompositeValue(inter, owner)

	array := NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		owner,
		value,
	)

	assert.Equal(t, owner, array.GetOwner())
	assert.Equal(t, owner, value.GetOwner())

	value = array.Remove(inter, EmptyLocationRange, 0).(*CompositeValue)

	assert.Equal(t, owner, array.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
}

func TestOwnerNewDictionary(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	assert.Equal(t, oldOwner, value.GetOwner())

	dictionary := NewDictionaryValue(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		keyValue, value,
	)

	// NOTE: keyValue is string, has no owner

	queriedValue, _ := dictionary.Get(inter, EmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, common.ZeroAddress, dictionary.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
}

func TestOwnerDictionary(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value,
	)

	// NOTE: keyValue is string, has no owner

	queriedValue, _ := dictionary.Get(inter, EmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryCopy(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{
			Storage: storage,
			ComputationGauge: getMeterCompFuncWithExpectedKinds(t,
				[]common.ComputationKind{
					common.ComputationKindCreateCompositeValue,
					common.ComputationKindCreateDictionaryValue,
					common.ComputationKindTransferCompositeValue,
					common.ComputationKindTransferDictionaryValue,
					common.ComputationKindTransferCompositeValue,
				},
				[]uint64{1, 1, 1, 1, 1},
			),
		},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
		keyValue, value,
	)

	copyResult := dictionary.Transfer(
		inter,
		EmptyLocationRange,
		atree.Address{},
		false,
		nil,
		nil,
		true, // dictionary is standalone.
	)

	dictionaryCopy := copyResult.(*DictionaryValue)

	queriedValue, _ := dictionaryCopy.Get(
		inter,
		EmptyLocationRange,
		keyValue,
	)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, common.ZeroAddress, dictionaryCopy.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
}

func TestOwnerDictionarySetSome(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	dictionary.SetKey(
		inter,
		EmptyLocationRange,
		keyValue,
		NewUnmeteredSomeValueNonCopying(value),
	)

	queriedValue, _ := dictionary.Get(inter, EmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryInsertNonExisting(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeAnyStruct,
		},
		newOwner,
	)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, oldOwner, value.GetOwner())

	existingValue := dictionary.Insert(
		inter,
		EmptyLocationRange,
		keyValue,
		value,
	)
	assert.Equal(t, Nil, existingValue)

	queriedValue, _ := dictionary.Get(inter, EmptyLocationRange, keyValue)
	value = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, newOwner, value.GetOwner())
}

func TestOwnerDictionaryRemove(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value1 := newTestCompositeValue(inter, oldOwner)
	value2 := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
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
		EmptyLocationRange,
		keyValue,
		value2,
	)
	require.IsType(t, &SomeValue{}, existingValue)
	innerValue := existingValue.(*SomeValue).InnerValue()
	value1 = innerValue.(*CompositeValue)

	queriedValue, _ := dictionary.Get(inter, EmptyLocationRange, keyValue)
	value2 = queriedValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, common.ZeroAddress, value1.GetOwner())
	assert.Equal(t, newOwner, value2.GetOwner())
}

func TestOwnerDictionaryInsertExisting(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	elaboration := sema.NewElaboration(nil)
	elaboration.SetCompositeType(
		testCompositeValueType.ID(),
		testCompositeValueType,
	)

	inter, err := NewInterpreter(
		&Program{
			Elaboration: elaboration,
		},
		TestLocation,
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	oldOwner := common.Address{0x1}
	newOwner := common.Address{0x2}

	keyValue := NewUnmeteredStringValue("test")
	value := newTestCompositeValue(inter, oldOwner)

	dictionary := NewDictionaryValueWithAddress(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
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
		EmptyLocationRange,
		keyValue,
	)
	require.IsType(t, &SomeValue{}, existingValue)
	innerValue := existingValue.(*SomeValue).InnerValue()
	value = innerValue.(*CompositeValue)

	assert.Equal(t, newOwner, dictionary.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
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

	composite.SetMember(inter, EmptyLocationRange, fieldName, value)

	value = composite.GetMember(inter, EmptyLocationRange, fieldName).(*CompositeValue)

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
		EmptyLocationRange,
		fieldName,
		value,
	)

	composite = composite.Transfer(
		inter,
		EmptyLocationRange,
		atree.Address{},
		false,
		nil,
		nil,
		true, // composite is standalone.
	).(*CompositeValue)

	value = composite.GetMember(
		inter,
		EmptyLocationRange,
		fieldName,
	).(*CompositeValue)

	assert.Equal(t, common.ZeroAddress, composite.GetOwner())
	assert.Equal(t, common.ZeroAddress, value.GetOwner())
}

func TestStringer(t *testing.T) {

	t.Parallel()

	type testCase struct {
		value    func(*Interpreter) Value
		expected string
	}

	stringerTests := map[string]testCase{
		"UInt": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUIntValueFromUint64(10)
			},
			expected: "10",
		},
		"UInt8": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt8Value(8)
			},
			expected: "8",
		},
		"UInt16": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt16Value(16)
			},
			expected: "16",
		},
		"UInt32": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt32Value(32)
			},
			expected: "32",
		},
		"UInt64": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt64Value(64)
			},
			expected: "64",
		},
		"UInt128": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt128ValueFromUint64(128)
			},
			expected: "128",
		},
		"UInt256": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUInt256ValueFromUint64(256)
			},
			expected: "256",
		},
		"Int8": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt8Value(-8)
			},
			expected: "-8",
		},
		"Int16": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt16Value(-16)
			},
			expected: "-16",
		},
		"Int32": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt32Value(-32)
			},
			expected: "-32",
		},
		"Int64": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt64Value(-64)
			},
			expected: "-64",
		},
		"Int128": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt128ValueFromInt64(-128)
			},
			expected: "-128",
		},
		"Int256": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredInt256ValueFromInt64(-256)
			},
			expected: "-256",
		},
		"Word8": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord8Value(8)
			},
			expected: "8",
		},
		"Word16": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord16Value(16)
			},
			expected: "16",
		},
		"Word32": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord32Value(32)
			},
			expected: "32",
		},
		"Word64": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord64Value(64)
			},
			expected: "64",
		},
		"Word128": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord128ValueFromUint64(128)
			},
			expected: "128",
		},
		"Word256": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredWord256ValueFromUint64(256)
			},
			expected: "256",
		},
		"UFix64": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredUFix64ValueWithInteger(64, EmptyLocationRange)
			},
			expected: "64.00000000",
		},
		"Fix64": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredFix64ValueWithInteger(-32, EmptyLocationRange)
			},
			expected: "-32.00000000",
		},
		"Void": {
			value: func(_ *Interpreter) Value {
				return Void
			},
			expected: "()",
		},
		"true": {
			value: func(_ *Interpreter) Value {
				return TrueValue
			},
			expected: "true",
		},
		"false": {
			value: func(_ *Interpreter) Value {
				return FalseValue
			},
			expected: "false",
		},
		"some": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredSomeValueNonCopying(TrueValue)
			},
			expected: "true",
		},
		"nil": {
			value: func(_ *Interpreter) Value {
				return Nil
			},
			expected: "nil",
		},
		"String": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredStringValue("Flow ridah!")
			},
			expected: `"Flow ridah!"`,
		},
		"Character": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredCharacterValue("😀")
			},
			expected: `"\u{1f600}"`,
		},
		"Array": {
			value: func(inter *Interpreter) Value {
				return NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeAnyStruct,
					},
					common.ZeroAddress,
					NewUnmeteredIntValueFromInt64(10),
					NewUnmeteredStringValue("TEST"),
				)
			},
			expected: "[10, \"TEST\"]",
		},
		"Dictionary": {
			value: func(inter *Interpreter) Value {
				return NewDictionaryValue(
					inter,
					EmptyLocationRange,
					&DictionaryStaticType{
						KeyType:   PrimitiveStaticTypeString,
						ValueType: PrimitiveStaticTypeUInt8,
					},
					NewUnmeteredStringValue("a"), NewUnmeteredUInt8Value(42),
					NewUnmeteredStringValue("b"), NewUnmeteredUInt8Value(99),
				)
			},
			expected: `{"b": 99, "a": 42}`,
		},
		"Address": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredAddressValueFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 1})
			},
			expected: "0x0000000000000001",
		},
		"composite": {
			value: func(inter *Interpreter) Value {
				fields := []CompositeField{
					{
						Name:  "y",
						Value: NewUnmeteredStringValue("bar"),
					},
				}

				return NewCompositeValue(
					inter,
					EmptyLocationRange,
					TestLocation,
					"Foo",
					common.CompositeKindResource,
					fields,
					common.ZeroAddress,
				)
			},
			expected: "S.test.Foo(y: \"bar\")",
		},
		"composite with custom stringer": {
			value: func(inter *Interpreter) Value {

				fields := []CompositeField{
					{
						Name:  "y",
						Value: NewUnmeteredStringValue("bar"),
					},
				}

				compositeValue := NewCompositeValue(
					inter,
					EmptyLocationRange,
					TestLocation,
					"Foo",
					common.CompositeKindResource,
					fields,
					common.ZeroAddress,
				)

				compositeValue.Stringer = func(_ common.MemoryGauge, _ *CompositeValue, _ SeenReferences) string {
					return "y --> bar"
				}

				return compositeValue
			},
			expected: "y --> bar",
		},
		"Path": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredPathValue(
					common.PathDomainStorage,
					"foo",
				)
			},
			expected: "/storage/foo",
		},
		"Type": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredTypeValue(PrimitiveStaticTypeInt)
			},
			expected: "Type<Int>()",
		},
		"ID Capability": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredCapabilityValue(
					6,
					NewUnmeteredAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
					&ReferenceStaticType{
						Authorization:  UnauthorizedAccess,
						ReferencedType: PrimitiveStaticTypeInt,
					},
				)
			},
			expected: "Capability<&Int>(address: 0x0000000102030405, id: 6)",
		},
		"Path capability with borrow type": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredPathCapabilityValue( //nolint:staticcheck
					&ReferenceStaticType{
						Authorization:  UnauthorizedAccess,
						ReferencedType: PrimitiveStaticTypeInt,
					},
					NewUnmeteredAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
					NewUnmeteredPathValue(
						common.PathDomainStorage,
						"foo",
					),
				)
			},
			expected: "Capability<&Int>(address: 0x0000000102030405, path: /storage/foo)",
		},
		"Path capability without borrow type": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredPathCapabilityValue( //nolint:staticcheck
					nil,
					NewUnmeteredAddressValueFromBytes([]byte{1, 2, 3, 4, 5}),
					NewUnmeteredPathValue(
						common.PathDomainStorage,
						"foo",
					),
				)
			},
			expected: "Capability(address: 0x0000000102030405, path: /storage/foo)",
		},
		"Recursive ephemeral reference (array)": {
			value: func(inter *Interpreter) Value {

				array := NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeAnyStruct,
					},
					common.ZeroAddress,
				)
				arrayRef := NewUnmeteredEphemeralReferenceValue(
					inter,
					UnauthorizedAccess,
					array,
					&sema.VariableSizedType{
						Type: sema.AnyStructType,
					},
					EmptyLocationRange,
				)

				array.Insert(inter, EmptyLocationRange, 0, arrayRef)
				return array
			},
			expected: `[[...]]`,
		},
		"static host function": {
			value: func(_ *Interpreter) Value {
				return NewUnmeteredStaticHostFunctionValue(
					&sema.FunctionType{
						Parameters: []sema.Parameter{
							{
								Label:          "foo",
								Identifier:     "bar",
								TypeAnnotation: sema.IntTypeAnnotation,
							},
						},
						ReturnTypeAnnotation: sema.StringTypeAnnotation,
					},
					func(invocation Invocation) Value {
						return NewUnmeteredStringValue("hello")
					},
				)
			},
			expected: "fun(foo bar: Int): String",
		},
		"bound host function": {
			value: func(inter *Interpreter) Value {
				self := NewUnmeteredStringValue("self")

				return NewUnmeteredBoundHostFunctionValue(
					inter,
					self,
					&sema.FunctionType{
						Parameters: []sema.Parameter{
							{
								Label:          "foo",
								Identifier:     "bar",
								TypeAnnotation: sema.IntTypeAnnotation,
							},
						},
						ReturnTypeAnnotation: sema.StringTypeAnnotation,
					},
					func(invocation Invocation) Value {
						return NewUnmeteredStringValue("hello")
					},
				)
			},
			expected: "fun(foo bar: Int): String",
		},
		"ephemeral reference": {
			value: func(inter *Interpreter) Value {
				return NewUnmeteredEphemeralReferenceValue(
					inter,
					UnauthorizedAccess,
					NewUnmeteredStringValue("hello"),
					&sema.ReferenceType{
						Authorization: sema.UnauthorizedAccess,
						Type:          sema.StringType,
					},
					EmptyLocationRange,
				)
			},
			expected: `"hello"`,
		},
		"storage capability controller": {
			value: func(inter *Interpreter) Value {
				return NewUnmeteredStorageCapabilityControllerValue(
					&ReferenceStaticType{
						Authorization:  UnauthorizedAccess,
						ReferencedType: PrimitiveStaticTypeInt,
					},
					UInt64Value(42),
					NewUnmeteredPathValue(
						common.PathDomainStorage,
						"foo",
					),
				)
			},
			expected: "StorageCapabilityController(borrowType: Type<&Int>(), capabilityID: 42, target: /storage/foo)",
		},
		"account capability controller": {
			value: func(inter *Interpreter) Value {
				return NewUnmeteredAccountCapabilityControllerValue(
					&ReferenceStaticType{
						Authorization:  UnauthorizedAccess,
						ReferencedType: PrimitiveStaticTypeAccount,
					},
					UInt64Value(42),
				)
			},
			expected: "AccountCapabilityController(borrowType: Type<&Account>(), capabilityID: 42)",
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {

			t.Parallel()

			inter := newTestInterpreter(t)

			value := testCase.value(inter)
			assert.Equal(t,
				testCase.expected,
				value.String(),
			)

			assert.Equal(t,
				testCase.expected,
				value.MeteredString(inter, SeenReferences{}, EmptyLocationRange),
			)
		})
	}

	for name, testCase := range stringerTests {
		name = fmt.Sprintf("optional %s", name)
		innerValue := testCase.value
		testCase.value = func(interpreter *Interpreter) Value {
			value := innerValue(interpreter)
			return NewSomeValueNonCopying(nil, value)
		}
		stringerTests[name] = testCase
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
		IntValueVisitor: func(_ ValueVisitContext, _ IntValue) {
			intVisits++
		},
		StringValueVisitor: func(_ ValueVisitContext, _ *StringValue) {
			stringVisits++
		},
	}

	var value Value
	value = NewUnmeteredIntValueFromInt64(42)
	value = NewUnmeteredSomeValueNonCopying(value)
	value = NewArrayValue(
		inter,
		EmptyLocationRange,
		&VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		value,
	)

	value = NewDictionaryValue(
		inter,
		EmptyLocationRange,
		&DictionaryStaticType{
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
		EmptyLocationRange,
		TestLocation,
		"Foo",
		common.CompositeKindStructure,
		fields,
		common.ZeroAddress,
	)

	value.Accept(inter, visitor, EmptyLocationRange)

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
		"Word128": {
			value:    NewUnmeteredWord128ValueFromUint64(128),
			expected: []byte{byte(HashInputTypeWord128), 128},
		},
		"Word128 min": {
			value:    NewUnmeteredWord128ValueFromUint64(0),
			expected: append([]byte{byte(HashInputTypeWord128)}, 0),
		},
		"Word128 max": {
			value:    NewUnmeteredWord128ValueFromBigInt(sema.Word128TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeWord128)}, sema.Word128TypeMaxIntBig.Bytes()...),
		},
		"Word256": {
			value:    NewUnmeteredWord256ValueFromUint64(256),
			expected: []byte{byte(HashInputTypeWord256), 1, 0},
		},
		"Word256 min": {
			value:    NewUnmeteredWord256ValueFromUint64(0),
			expected: append([]byte{byte(HashInputTypeWord256)}, 0),
		},
		"Word256 max": {
			value:    NewUnmeteredWord256ValueFromBigInt(sema.Word256TypeMaxIntBig),
			expected: append([]byte{byte(HashInputTypeWord256)}, sema.Word256TypeMaxIntBig.Bytes()...),
		},
		"UFix64": {
			value:    NewUnmeteredUFix64ValueWithInteger(64, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeUFix64), 0x0, 0x0, 0x0, 0x1, 0x7d, 0x78, 0x40, 0x0},
		},
		"UFix64 min": {
			value:    NewUnmeteredUFix64ValueWithInteger(0, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeUFix64), 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		"UFix64 max": {
			value:    NewUnmeteredUFix64ValueWithInteger(sema.UFix64TypeMaxInt, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeUFix64), 0xff, 0xff, 0xff, 0xff, 0xff, 0x6e, 0x41, 0x0},
		},
		"Fix64": {
			value:    NewUnmeteredFix64ValueWithInteger(-32, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeFix64), 0xff, 0xff, 0xff, 0xff, 0x41, 0x43, 0xe0, 0x0},
		},
		"Fix64 min": {
			value:    NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMinInt, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeFix64), 0x80, 0x0, 0x0, 0x0, 0x03, 0x43, 0xd0, 0x0},
		},
		"Fix64 max": {
			value:    NewUnmeteredFix64ValueWithInteger(sema.Fix64TypeMaxInt, EmptyLocationRange),
			expected: []byte{byte(HashInputTypeFix64), 0x7f, 0xff, 0xff, 0xff, 0xfc, 0xbc, 0x30, 0x00},
		},
		"true": {
			value:    TrueValue,
			expected: []byte{byte(HashInputTypeBool), 1},
		},
		"false": {
			value:    FalseValue,
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
					EmptyLocationRange,
					TestLocation,
					"Foo",
					common.CompositeKindEnum,
					fields,
					common.ZeroAddress,
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
					EmptyLocationRange,
					TestLocation,
					strings.Repeat("a", 32),
					common.CompositeKindEnum,
					fields,
					common.ZeroAddress,
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
			value: NewUnmeteredPathValue(
				common.PathDomainStorage,
				"foo",
			),
			expected: []byte{
				byte(HashInputTypePath),
				// domain: storage
				0x1,
				// identifier: "foo"
				0x66, 0x6f, 0x6f,
			},
		},
		"Path long identifier": {
			value: NewUnmeteredPathValue(
				common.PathDomainStorage,
				strings.Repeat("a", 32),
			),
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

			actual := testCase.value.HashInput(inter, EmptyLocationRange, scratch[:])

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
		NewArrayValue(
			inter,
			EmptyLocationRange,
			ByteArrayStaticType,
			common.ZeroAddress,
		),
		NewUnmeteredUFix64ValueWithInteger(5, EmptyLocationRange),
	)

	// static type test

	assert.Equal(t,
		NewUnmeteredUFix64ValueWithInteger(5, EmptyLocationRange),
		block.Fields[sema.BlockTypeTimestampFieldName],
	)
}

func TestEphemeralReferenceTypeConformance(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	// Obtain a self referencing (cyclic) ephemeral reference value.

	code := `
        access(all) fun getEphemeralRef(): &Foo {
            var foo = Foo()
            var fooRef = &foo as &Foo

            // Create the cyclic reference
            fooRef.setBar(fooRef)

            return fooRef
        }

        access(all) struct Foo {

            access(all) var bar: &Foo?

			access(all) fun setBar(_ bar: &Foo) {
				self.bar = bar
			}

            init() {
                self.bar = nil
            }
        }`

	checker, err := ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{},
	)

	require.NoError(t, err)

	inter, err := NewInterpreter(
		ProgramFromChecker(checker),
		checker.Location,
		&Config{Storage: storage},
	)

	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("getEphemeralRef")
	require.NoError(t, err)
	require.IsType(t, &EphemeralReferenceValue{}, value)

	// Check the dynamic type conformance on a cyclic value.
	conforms := value.ConformsToStaticType(
		inter,
		EmptyLocationRange,
		TypeConformanceResults{},
	)
	assert.True(t, conforms)
}

func TestCapabilityValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal, borrow type", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewUnmeteredCapabilityValue(
				4,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				PrimitiveStaticTypeInt,
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredCapabilityValue(
					4,
					NewUnmeteredAddressValueFromBytes([]byte{0x1}),
					PrimitiveStaticTypeInt,
				),
			),
		)
	})

	t.Run("different addresses", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredCapabilityValue(
				4,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				PrimitiveStaticTypeInt,
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredCapabilityValue(
					4,
					NewUnmeteredAddressValueFromBytes([]byte{0x2}),
					PrimitiveStaticTypeInt,
				),
			),
		)
	})

	t.Run("different borrow types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredCapabilityValue(
				4,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				PrimitiveStaticTypeInt,
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredCapabilityValue(
					4,
					NewUnmeteredAddressValueFromBytes([]byte{0x1}),
					PrimitiveStaticTypeString,
				),
			),
		)
	})

	t.Run("different ID", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredCapabilityValue(
				4,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				PrimitiveStaticTypeInt,
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredCapabilityValue(
					5,
					NewUnmeteredAddressValueFromBytes([]byte{0x1}),
					PrimitiveStaticTypeInt,
				),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredCapabilityValue(
				4,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				PrimitiveStaticTypeInt,
			).Equal(
				inter,
				EmptyLocationRange,
				FalseValue,
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
			NewUnmeteredAddressValueFromBytes([]byte{0x1}).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredAddressValueFromBytes([]byte{0x1}).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredAddressValueFromBytes([]byte{0x2}),
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredAddressValueFromBytes([]byte{0x1}).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

// ensure () == ()
func TestVoidValue_Equal(t *testing.T) {
	t.Parallel()

	inter := newTestInterpreter(t)
	require.True(t,
		VoidValue{}.Equal(
			inter,
			EmptyLocationRange,
			VoidValue{},
		),
	)
}

func TestBoolValue_Equal(t *testing.T) {

	t.Parallel()

	t.Run("equal true", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			TrueValue.Equal(
				inter,
				EmptyLocationRange,
				TrueValue,
			),
		)
	})

	t.Run("equal false", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			FalseValue.Equal(
				inter,
				EmptyLocationRange,
				FalseValue,
			),
		)
	})

	t.Run("different", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			TrueValue.Equal(
				inter,
				EmptyLocationRange,
				FalseValue,
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			TrueValue.Equal(
				inter,
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
				Nil,
			),
		)
	})

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NilValue{}.Equal(
				inter,
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				EmptyLocationRange,
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
				NewUnmeteredPathValue(
					domain,
					"test",
				).Equal(
					inter,
					EmptyLocationRange,
					NewUnmeteredPathValue(
						domain,
						"test",
					),
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
					NewUnmeteredPathValue(
						domain,
						"test",
					).Equal(
						inter,
						EmptyLocationRange,
						NewUnmeteredPathValue(
							otherDomain,
							"test",
						),
					),
				)
			})
		}
	}

	for _, domain := range common.AllPathDomains {

		t.Run(fmt.Sprintf("different identifiers, %s", domain), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.False(t,
				NewUnmeteredPathValue(
					domain,
					"test1",
				).Equal(
					inter,
					EmptyLocationRange,
					NewUnmeteredPathValue(
						domain,
						"test2",
					),
				),
			)
		})
	}

	t.Run("different kind", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.False(t,
			NewUnmeteredPathValue(
				common.PathDomainStorage,
				"test",
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredStringValue("/storage/test"),
			),
		)
	})
}

func TestArrayValue_Equal(t *testing.T) {

	t.Parallel()

	uint8ArrayStaticType := &VariableSizedStaticType{
		Type: PrimitiveStaticTypeUInt8,
	}

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewArrayValue(
				inter,
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint8ArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint8ArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
				NewUnmeteredUInt8Value(1),
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint8ArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredUInt8Value(2),
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint8ArrayStaticType,
					common.ZeroAddress,
					NewUnmeteredUInt8Value(1),
				),
			),
		)
	})

	t.Run("different types", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		uint16ArrayStaticType := &VariableSizedStaticType{
			Type: PrimitiveStaticTypeUInt16,
		}

		require.False(t,
			NewArrayValue(
				inter,
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint16ArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				nil,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					uint8ArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					nil,
					common.ZeroAddress,
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
				EmptyLocationRange,
				nil,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					nil,
					common.ZeroAddress,
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
				EmptyLocationRange,
				uint8ArrayStaticType,
				common.ZeroAddress,
				NewUnmeteredUInt8Value(1),
			).Equal(
				inter,
				EmptyLocationRange,
				NewUnmeteredUInt8Value(1),
			),
		)
	})
}

func TestDictionaryValue_Equal(t *testing.T) {

	t.Parallel()

	byteStringDictionaryType := &DictionaryStaticType{
		KeyType:   PrimitiveStaticTypeUInt8,
		ValueType: PrimitiveStaticTypeString,
	}

	t.Run("equal", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		require.True(t,
			NewDictionaryValue(
				inter,
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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

		stringByteDictionaryStaticType := &DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeUInt8,
		}

		require.False(t,
			NewDictionaryValue(
				inter,
				EmptyLocationRange,
				byteStringDictionaryType,
			).Equal(
				inter,
				EmptyLocationRange,
				NewDictionaryValue(
					inter,
					EmptyLocationRange,
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
				EmptyLocationRange,
				byteStringDictionaryType,
				NewUnmeteredUInt8Value(1),
				NewUnmeteredStringValue("1"),
				NewUnmeteredUInt8Value(2),
				NewUnmeteredStringValue("2"),
			).Equal(
				inter,
				EmptyLocationRange,
				NewArrayValue(
					inter,
					EmptyLocationRange,
					ByteArrayStaticType,
					common.ZeroAddress,
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
				EmptyLocationRange,
				TestLocation,
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					TestLocation,
					"X",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("B"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("A"),
					"Y",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindStructure,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
				NewCompositeValue(
					inter,
					EmptyLocationRange,
					common.IdentifierLocation("A"),
					"X",
					common.CompositeKindResource,
					fields2,
					common.ZeroAddress,
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
				EmptyLocationRange,
				common.IdentifierLocation("A"),
				"X",
				common.CompositeKindStructure,
				fields1,
				common.ZeroAddress,
			).Equal(
				inter,
				EmptyLocationRange,
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
		"Word128": NewUnmeteredWord128ValueFromUint64(128),
		"Word256": NewUnmeteredWord256ValueFromUint64(256),
		"UFix64":  NewUnmeteredUFix64ValueWithInteger(64, EmptyLocationRange),
		"Fix64":   NewUnmeteredFix64ValueWithInteger(-32, EmptyLocationRange),
	}

	for name, value := range testValues {

		t.Run(fmt.Sprintf("equal, %s", name), func(t *testing.T) {

			inter := newTestInterpreter(t)

			require.True(t,
				value.Equal(
					inter,
					EmptyLocationRange,
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
						EmptyLocationRange,
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
					EmptyLocationRange,
					NewUnmeteredAddressValueFromBytes([]byte{0x1}),
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
			TestLocation,
			&Config{
				Storage: storage,
			},
		)
		require.NoError(t, err)

		publicKey := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			NewUnmeteredIntValueFromInt64(1),
			NewUnmeteredIntValueFromInt64(7),
			NewUnmeteredIntValueFromInt64(3),
		)

		sigAlgo := stdlib.NewSignatureAlgorithmCase(
			UInt8Value(sema.SignatureAlgorithmECDSA_secp256k1.RawValue()),
		)

		key := NewPublicKeyValue(
			inter,
			EmptyLocationRange,
			publicKey,
			sigAlgo,
			func(context PublicKeyValidationContext, locationRange LocationRange, publicKey *CompositeValue) error {
				return nil
			},
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
			TestLocation,
			&Config{
				Storage: storage,
			},
		)
		require.NoError(t, err)

		publicKeyBytes := []byte{1, 7, 3}

		publicKey := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeInt,
			},
			common.ZeroAddress,
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[0])),
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[1])),
			NewUnmeteredIntValueFromInt64(int64(publicKeyBytes[2])),
		)

		sigAlgo := stdlib.NewSignatureAlgorithmCase(
			UInt8Value(sema.SignatureAlgorithmECDSA_secp256k1.RawValue()),
		)

		func() {
			defer func() {
				r := recover()
				assert.Equal(
					t,
					&InvalidPublicKeyError{PublicKey: publicKey, Err: fakeError},
					r,
				)
			}()

			_ = NewPublicKeyValue(
				inter,
				EmptyLocationRange,
				publicKey,
				sigAlgo,
				func(context PublicKeyValidationContext, locationRange LocationRange, publicKey *CompositeValue) error {
					return fakeError
				},
			)
		}()
	})
}

func TestHashable(t *testing.T) {
	t.Parallel()

	// Assert that all Value implementations are hashable

	pkgs, err := packages.Load(
		&packages.Config{
			// https://github.com/golang/go/issues/45218
			Mode: packages.NeedImports | packages.NeedDeps | packages.NeedTypes,
		},
		"github.com/onflow/cadence/interpreter",
	)
	require.NoError(t, err)

	pkg := pkgs[0]
	scope := pkg.Types.Scope()

	test := func(interfaceName string) {

		t.Run(interfaceName, func(t *testing.T) {

			object := scope.Lookup(interfaceName)
			ty := object.Type()
			interfaceType, ok := ty.Underlying().(*types.Interface)
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
		TestLocation,
		&Config{
			Storage:                       storage,
			AtreeValueValidationEnabled:   true,
			AtreeStorageValidationEnabled: true,
		},
	)
	require.NoError(tb, err)

	return inter
}

func TestNonStorable(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	code := `
      access(all) struct Foo {

          let bar: &Int?

          init() {
              self.bar = &1 as &Int
          }
      }

      fun foo(): &Int? {
          return Foo().bar
      }
    `

	checker, err := ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{},
	)

	require.NoError(t, err)

	inter, err := NewInterpreter(
		ProgramFromChecker(checker),
		checker.Location,
		&Config{Storage: storage},
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

func TestNumberValueIntegerConversion(t *testing.T) {

	t.Parallel()

	type converter struct {
		convert func(NumberValue) (result any, convertible bool)
		check   func(t *testing.T, result any) bool
	}

	test := func(
		t *testing.T,
		numericType sema.Type,
		testValue NumberValue,
		converter converter,
	) {

		result, convertible := converter.convert(testValue)
		if !convertible {
			return
		}
		converter.check(t, result)
	}

	testValues := map[*sema.NumericType]NumberValue{
		sema.IntType:     NewUnmeteredIntValueFromInt64(42),
		sema.UIntType:    NewUnmeteredUIntValueFromUint64(42),
		sema.UInt8Type:   NewUnmeteredUInt8Value(42),
		sema.UInt16Type:  NewUnmeteredUInt16Value(42),
		sema.UInt32Type:  NewUnmeteredUInt32Value(42),
		sema.UInt64Type:  NewUnmeteredUInt64Value(42),
		sema.UInt128Type: NewUnmeteredUInt128ValueFromUint64(42),
		sema.UInt256Type: NewUnmeteredUInt256ValueFromUint64(42),
		sema.Word8Type:   NewUnmeteredWord8Value(42),
		sema.Word16Type:  NewUnmeteredWord16Value(42),
		sema.Word32Type:  NewUnmeteredWord32Value(42),
		sema.Word64Type:  NewUnmeteredWord64Value(42),
		sema.Word128Type: NewUnmeteredWord128ValueFromUint64(42),
		sema.Word256Type: NewUnmeteredWord256ValueFromUint64(42),
		sema.Int8Type:    NewUnmeteredInt8Value(42),
		sema.Int16Type:   NewUnmeteredInt16Value(42),
		sema.Int32Type:   NewUnmeteredInt32Value(42),
		sema.Int64Type:   NewUnmeteredInt64Value(42),
		sema.Int128Type:  NewUnmeteredInt128ValueFromInt64(42),
		sema.Int256Type:  NewUnmeteredInt256ValueFromInt64(42),
	}

	for _, ty := range sema.AllIntegerTypes {
		// Only test leaf types
		switch ty {
		case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
			continue
		}

		_, ok := testValues[ty.(*sema.NumericType)]
		require.True(t, ok, "missing expected value for type %s", ty.String())
	}

	converters := map[string]converter{
		"ToInt": {
			convert: func(value NumberValue) (any, bool) {
				return value.ToInt(EmptyLocationRange), true
			},
			check: func(t *testing.T, result any) bool {
				return assert.Equal(t, 42, result)
			},
		},
		"ToBigInt": {
			convert: func(value NumberValue) (any, bool) {
				bigNumberValue, ok := value.(BigNumberValue)
				if !ok {
					return nil, false
				}
				return bigNumberValue.ToBigInt(nil), true
			},
			check: func(t *testing.T, result any) bool {
				return assert.Equal(t, big.NewInt(42), result)
			},
		},
	}

	for numericType, testValue := range testValues {

		t.Run(numericType.String(), func(t *testing.T) {

			for converterName, converter := range converters {

				t.Run(converterName, func(t *testing.T) {
					test(t, numericType, testValue, converter)
				})

			}
		})
	}
}

func TestValue_ConformsToStaticType(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	newCompositeValue := func(inter *Interpreter, fields []CompositeField) *CompositeValue {
		return NewCompositeValue(
			inter,
			EmptyLocationRange,
			TestLocation,
			"Test",
			common.CompositeKindStructure,
			fields,
			testAddress,
		)
	}

	newInvalidCompositeValue := func(inter *Interpreter) *CompositeValue {
		return newCompositeValue(inter, []CompositeField{})
	}

	test := func(valueFactory func(*Interpreter) Value, expected bool) {

		storage := newUnmeteredInMemoryStorage()

		members := &sema.StringMemberOrderedMap{}

		compositeType := &sema.CompositeType{
			Location:   TestLocation,
			Identifier: "Test",
			Kind:       common.CompositeKindStructure,
			Members:    members,
			Fields:     []string{"foo"},
		}

		fooField := sema.NewPublicConstantFieldMember(
			nil,
			compositeType,
			"foo",
			sema.BoolType,
			"",
		)
		members.Set("foo", fooField)

		elaboration := sema.NewElaboration(nil)
		elaboration.SetCompositeType(
			compositeType.ID(),
			compositeType,
		)

		inter, err := NewInterpreter(
			&Program{
				Elaboration: elaboration,
			},
			TestLocation,
			&Config{Storage: storage},
		)
		require.NoError(t, err)

		storageMap := storage.GetDomainStorageMap(inter, testAddress, common.StorageDomainPathStorage, true)
		storageMap.WriteValue(inter, StringStorageMapKey("test"), TrueValue)

		value := valueFactory(inter)

		result := value.ConformsToStaticType(
			inter,
			EmptyLocationRange,
			TypeConformanceResults{},
		)
		if expected {
			assert.True(t, result)
		} else {
			assert.False(t, result)
		}
	}

	t.Run("function values", func(t *testing.T) {

		t.Parallel()

		functionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			[]sema.Parameter{
				{
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			sema.BoolTypeAnnotation,
		)

		for name, f := range map[string]Value{
			"InterpretedFunctionValue": &InterpretedFunctionValue{
				Type: functionType,
			},
			"HostFunctionValue": &HostFunctionValue{
				Type: functionType,
			},
			"BoundFunctionValue": &BoundFunctionValue{
				Function: &InterpretedFunctionValue{
					Type: functionType,
				},
			},
		} {
			t.Run(name, func(t *testing.T) {
				test(
					func(_ *Interpreter) Value {
						return f
					},
					true,
				)
			})
		}
	})

	t.Run("BoolValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return TrueValue
			},
			true,
		)
	})

	t.Run("StringValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredStringValue("test")
			},
			true,
		)
	})

	t.Run("AddressValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredAddressValueFromBytes([]byte{0x1})
			},
			true,
		)
	})

	t.Run("TypeValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredTypeValue(PrimitiveStaticTypeInt)
			},
			true,
		)
	})

	t.Run("VoidValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return Void
			},
			true,
		)
	})

	t.Run("CharacterValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredCharacterValue("t")
			},
			true,
		)
	})

	t.Run("NilValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return Nil
			},
			true,
		)
	})

	t.Run("SomeValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(interpreter *Interpreter) Value {
				return NewUnmeteredSomeValueNonCopying(
					TrueValue,
				)
			},
			true,
		)
	})

	t.Run("PathValue", func(t *testing.T) {

		t.Parallel()

		for _, domain := range common.AllPathDomains {
			t.Run(domain.Identifier(), func(t *testing.T) {
				test(
					func(interpreter *Interpreter) Value {
						return NewUnmeteredPathValue(domain, "test")
					},
					true,
				)
			})
		}
	})

	t.Run("integer values", func(t *testing.T) {

		t.Parallel()

		testCases := map[*sema.NumericType]NumberValue{
			sema.IntType:     NewUnmeteredIntValueFromInt64(42),
			sema.UIntType:    NewUnmeteredUIntValueFromUint64(42),
			sema.UInt8Type:   NewUnmeteredUInt8Value(42),
			sema.UInt16Type:  NewUnmeteredUInt16Value(42),
			sema.UInt32Type:  NewUnmeteredUInt32Value(42),
			sema.UInt64Type:  NewUnmeteredUInt64Value(42),
			sema.UInt128Type: NewUnmeteredUInt128ValueFromUint64(42),
			sema.UInt256Type: NewUnmeteredUInt256ValueFromUint64(42),
			sema.Word8Type:   NewUnmeteredWord8Value(42),
			sema.Word16Type:  NewUnmeteredWord16Value(42),
			sema.Word32Type:  NewUnmeteredWord32Value(42),
			sema.Word64Type:  NewUnmeteredWord64Value(42),
			sema.Word128Type: NewUnmeteredWord128ValueFromUint64(42),
			sema.Word256Type: NewUnmeteredWord256ValueFromUint64(42),
			sema.Int8Type:    NewUnmeteredInt8Value(42),
			sema.Int16Type:   NewUnmeteredInt16Value(42),
			sema.Int32Type:   NewUnmeteredInt32Value(42),
			sema.Int64Type:   NewUnmeteredInt64Value(42),
			sema.Int128Type:  NewUnmeteredInt128ValueFromInt64(42),
			sema.Int256Type:  NewUnmeteredInt256ValueFromInt64(42),
		}

		for _, ty := range sema.AllIntegerTypes {
			// Only test leaf types
			switch ty {
			case sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType:
				continue
			}

			_, ok := testCases[ty.(*sema.NumericType)]
			require.True(t, ok, "missing case for type %s", ty.String())
		}

		for ty, v := range testCases {
			t.Run(ty.String(), func(t *testing.T) {
				test(
					func(_ *Interpreter) Value {
						return v
					},
					true,
				)
			})
		}
	})

	t.Run("fixed-point values", func(t *testing.T) {

		t.Parallel()

		testCases := map[*sema.FixedPointNumericType]NumberValue{
			sema.UFix64Type: NewUnmeteredUFix64ValueWithInteger(42, EmptyLocationRange),
			sema.Fix64Type:  NewUnmeteredFix64ValueWithInteger(42, EmptyLocationRange),
		}

		for _, ty := range sema.AllFixedPointTypes {
			// Only test leaf types
			switch ty {
			case sema.FixedPointType, sema.SignedFixedPointType:
				continue
			}

			_, ok := testCases[ty.(*sema.FixedPointNumericType)]
			require.True(t, ok, "missing case for type %s", ty.String())
		}

		for ty, v := range testCases {
			t.Run(ty.String(), func(t *testing.T) {
				test(
					func(_ *Interpreter) Value {
						return v
					},
					true,
				)
			})
		}
	})

	t.Run("EphemeralReferenceValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(inter *Interpreter) Value {
				return NewUnmeteredEphemeralReferenceValue(
					inter,
					UnauthorizedAccess,
					TrueValue,
					sema.BoolType,
					EmptyLocationRange,
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewUnmeteredEphemeralReferenceValue(
					inter,
					UnauthorizedAccess,
					TrueValue,
					sema.StringType,
					EmptyLocationRange,
				)
			},
			false,
		)
	})

	t.Run("StorageReferenceValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredStorageReferenceValue(
					UnauthorizedAccess,
					testAddress,
					NewUnmeteredPathValue(common.PathDomainStorage, "test"),
					sema.BoolType,
				)
			},
			true,
		)

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredStorageReferenceValue(
					UnauthorizedAccess,
					testAddress,
					NewUnmeteredPathValue(common.PathDomainStorage, "test"),
					sema.StringType,
				)
			},
			false,
		)
	})

	t.Run("CapabilityValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(_ *Interpreter) Value {
				return NewUnmeteredCapabilityValue(
					NewUnmeteredUInt64Value(4),
					NewUnmeteredAddressValueFromBytes(testAddress.Bytes()),
					&ReferenceStaticType{
						Authorization:  UnauthorizedAccess,
						ReferencedType: PrimitiveStaticTypeBool,
					},
				)
			},
			true,
		)
	})

	t.Run("ArrayValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(inter *Interpreter) Value {
				return NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeNumber,
					},
					testAddress,
					NewUnmeteredInt8Value(2),
					NewUnmeteredFix64Value(3),
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeAnyStruct,
					},
					testAddress,
					NewUnmeteredInt8Value(2),
					NewUnmeteredFix64Value(3),
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeInteger,
					},
					testAddress,
					NewUnmeteredInt8Value(2),
					NewUnmeteredFix64Value(3),
				)
			},
			false,
		)

		test(
			func(inter *Interpreter) Value {
				return NewArrayValue(
					inter,
					EmptyLocationRange,
					&VariableSizedStaticType{
						Type: PrimitiveStaticTypeAnyStruct,
					},
					testAddress,
					newInvalidCompositeValue(inter),
				)
			},
			false,
		)
	})

	t.Run("DictionaryValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(inter *Interpreter) Value {
				return NewDictionaryValueWithAddress(
					inter,
					EmptyLocationRange,
					&DictionaryStaticType{
						KeyType:   PrimitiveStaticTypeString,
						ValueType: PrimitiveStaticTypeNumber,
					},
					testAddress,
					NewUnmeteredStringValue("a"),
					NewUnmeteredInt8Value(2),
					NewUnmeteredStringValue("b"),
					NewUnmeteredFix64Value(3),
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewDictionaryValueWithAddress(
					inter,
					EmptyLocationRange,
					&DictionaryStaticType{
						KeyType:   PrimitiveStaticTypeString,
						ValueType: PrimitiveStaticTypeAnyStruct,
					},
					testAddress,
					NewUnmeteredStringValue("a"),
					NewUnmeteredInt8Value(2),
					NewUnmeteredStringValue("b"),
					NewUnmeteredFix64Value(3),
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewDictionaryValueWithAddress(
					inter,
					EmptyLocationRange,
					&DictionaryStaticType{
						KeyType:   PrimitiveStaticTypeAnyStruct,
						ValueType: PrimitiveStaticTypeNumber,
					},
					testAddress,
					NewUnmeteredStringValue("a"),
					NewUnmeteredInt8Value(2),
					NewUnmeteredStringValue("b"),
					NewUnmeteredFix64Value(3),
				)
			},
			true,
		)

		// TODO: cannot test due to container mutation check. import instead?

		//test(
		//	NewDictionaryValueWithAddress(
		//		inter,
		//		&DictionaryStaticTypeX{
		//			KeyType:   PrimitiveStaticTypeInt,
		//			ValueType: PrimitiveStaticTypeNumber,
		//		},
		//		testAddress,
		//		NewUnmeteredStringValue("a"),
		//		NewUnmeteredInt8Value(2),
		//		NewUnmeteredStringValue("b"),
		//		NewUnmeteredFix64Value(3),
		//	),
		//	false,
		//)
		//
		//test(
		//	NewDictionaryValueWithAddress(
		//		inter,
		//		&DictionaryStaticTypeX{
		//			KeyType:   PrimitiveStaticTypeAnyStruct,
		//			ValueType: PrimitiveStaticTypeInteger,
		//		},
		//		testAddress,
		//		NewUnmeteredStringValue("a"),
		//		NewUnmeteredInt8Value(2),
		//		NewUnmeteredStringValue("b"),
		//		NewUnmeteredFix64Value(3),
		//	),
		//	false,
		//)

		test(
			func(inter *Interpreter) Value {
				return NewDictionaryValueWithAddress(
					inter,
					EmptyLocationRange,
					&DictionaryStaticType{
						KeyType:   PrimitiveStaticTypeAnyStruct,
						ValueType: PrimitiveStaticTypeAnyStruct,
					},
					testAddress,
					NewUnmeteredStringValue("a"),
					newInvalidCompositeValue(inter),
				)
			},
			false,
		)
	})

	t.Run("CompositeValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(inter *Interpreter) Value {
				return newCompositeValue(inter, []CompositeField{
					{
						Name:  "foo",
						Value: TrueValue,
					},
				})
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return newCompositeValue(inter, []CompositeField{
					{
						Name:  "foo",
						Value: NewUnmeteredStringValue("test"),
					},
				})
			},
			false,
		)

		test(
			func(inter *Interpreter) Value {
				return newInvalidCompositeValue(inter)
			},
			false,
		)
	})

	t.Run("SimpleCompositeValue", func(t *testing.T) {

		t.Parallel()

		test(
			func(inter *Interpreter) Value {
				return NewSimpleCompositeValue(
					inter,
					PrimitiveStaticTypeBlock.SemaType().ID(),
					PrimitiveStaticTypeBlock,
					[]string{"height"},
					map[string]Value{
						"height": NewUnmeteredInt64Value(1),
					},
					nil,
					nil,
					nil,
					nil,
				)
			},
			true,
		)

		test(
			func(inter *Interpreter) Value {
				return NewSimpleCompositeValue(
					inter,
					PrimitiveStaticTypeBlock.SemaType().ID(),
					PrimitiveStaticTypeBlock,
					[]string{"foo"},
					map[string]Value{
						"foo": newInvalidCompositeValue(inter),
					},
					nil,
					nil,
					nil,
					nil,
				)
			},
			false,
		)
	})

}

func TestStringIsGraphemeBoundaryStart(t *testing.T) {

	t.Parallel()

	test := func(s string, i int, expected bool) {

		name := fmt.Sprintf("%s, %d", s, i)

		t.Run(name, func(t *testing.T) {
			str := NewUnmeteredStringValue(s)
			assert.Equal(t, expected, str.IsGraphemeBoundaryStart(i))
		})
	}

	test("", 0, false)
	test("a", 0, true)
	test("a", 1, false)
	test("ab", 1, true)

	// 🇪🇸🇪🇪 ("ES", "EE")
	flagESflagEE := "\U0001F1EA\U0001F1F8\U0001F1EA\U0001F1EA"
	require.Len(t, flagESflagEE, 16)
	test(flagESflagEE, 0, true)
	test(flagESflagEE, 1, false)
	test(flagESflagEE, 2, false)
	test(flagESflagEE, 3, false)
	test(flagESflagEE, 4, false)
	test(flagESflagEE, 5, false)
	test(flagESflagEE, 6, false)
	test(flagESflagEE, 7, false)

	test(flagESflagEE, 8, true)
	test(flagESflagEE, 9, false)
	test(flagESflagEE, 10, false)
	test(flagESflagEE, 11, false)
	test(flagESflagEE, 12, false)
	test(flagESflagEE, 13, false)
	test(flagESflagEE, 14, false)
	test(flagESflagEE, 15, false)
}

func TestStringIsGraphemeBoundaryEnd(t *testing.T) {

	t.Parallel()

	test := func(s string, i int, expected bool) {

		name := fmt.Sprintf("%s, %d", s, i)

		t.Run(name, func(t *testing.T) {
			str := NewUnmeteredStringValue(s)
			assert.Equal(t, expected, str.IsGraphemeBoundaryEnd(i))
		})
	}

	test("", 0, false)
	test("a", 0, false)
	test("a", 1, true)
	test("ab", 1, true)

	// 🇪🇸🇪🇪 ("ES", "EE")
	flagESflagEE := "\U0001F1EA\U0001F1F8\U0001F1EA\U0001F1EA"
	require.Len(t, flagESflagEE, 16)
	test(flagESflagEE, 0, false)
	test(flagESflagEE, 1, false)
	test(flagESflagEE, 2, false)
	test(flagESflagEE, 3, false)
	test(flagESflagEE, 4, false)
	test(flagESflagEE, 5, false)
	test(flagESflagEE, 6, false)
	test(flagESflagEE, 7, false)

	test(flagESflagEE, 8, true)
	test(flagESflagEE, 9, false)
	test(flagESflagEE, 10, false)
	test(flagESflagEE, 11, false)
	test(flagESflagEE, 12, false)
	test(flagESflagEE, 13, false)
	test(flagESflagEE, 14, false)
	test(flagESflagEE, 15, false)

	test(flagESflagEE, 16, true)

}

func TestOverwriteDictionaryValueWhereKeyIsStoredInSeparateAtreeSlab(t *testing.T) {

	t.Parallel()

	owner := common.Address{0x1}

	t.Run("enum as dict key", func(t *testing.T) {

		newEnumValue := func(inter *Interpreter) Value {
			return NewCompositeValue(
				inter,
				EmptyLocationRange,
				TestLocation,
				"Test",
				common.CompositeKindEnum,
				[]CompositeField{
					{
						Name:  "rawValue",
						Value: NewUnmeteredUInt8Value(42),
					},
				},
				common.ZeroAddress,
			)
		}

		storage := newUnmeteredInMemoryStorage()

		elaboration := sema.NewElaboration(nil)
		elaboration.SetCompositeType(
			testCompositeValueType.ID(),
			testCompositeValueType,
		)

		inter, err := NewInterpreter(
			&Program{
				Elaboration: elaboration,
			},
			TestLocation,
			&Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   true,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		// Create empty dictionary
		dictionary := NewDictionaryValueWithAddress(
			inter,
			EmptyLocationRange,
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeAnyStruct,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.Equal(t, 0, dictionary.Count())

		// Insert new key-value pair (enum as key) to dictionary
		existingValue := dictionary.Insert(
			inter,
			EmptyLocationRange,
			newEnumValue(inter),
			NewUnmeteredInt64Value(int64(1)),
		)
		require.Equal(t, NilOptionalValue, existingValue)
		require.Equal(t, 1, dictionary.Count())

		// Test inserted dictionary element
		v, found := dictionary.Get(
			inter,
			EmptyLocationRange,
			newEnumValue(inter),
		)
		require.True(t, found)
		require.Equal(t, Int64Value(1), v)

		// Update existing key with new value
		existingValue = dictionary.Insert(
			inter,
			EmptyLocationRange,
			newEnumValue(inter),
			NewUnmeteredInt64Value(int64(2)),
		)
		require.NotEqual(t, Int64Value(1), existingValue)
		require.Equal(t, 1, dictionary.Count())

		// Check updated dictionary element
		v, found = dictionary.Get(
			inter,
			EmptyLocationRange,
			newEnumValue(inter),
		)
		require.True(t, found)
		require.Equal(t, Int64Value(2), v)

		// Check storage containing only one root slab (dictionary root)
		checkRootSlabIDsInStorage(t, storage, []atree.SlabID{dictionary.SlabID()})
	})

	t.Run("large string as dict key", func(t *testing.T) {
		newStringValue := func() Value {
			return NewUnmeteredStringValue(strings.Repeat("a", 1024))
		}

		storage := newUnmeteredInMemoryStorage()

		elaboration := sema.NewElaboration(nil)

		inter, err := NewInterpreter(
			&Program{
				Elaboration: elaboration,
			},
			TestLocation,
			&Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   true,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		// Create empty dictionary
		dictionary := NewDictionaryValueWithAddress(
			inter,
			EmptyLocationRange,
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeAnyStruct,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			owner,
		)
		require.Equal(t, 0, dictionary.Count())

		// Insert new key-value pair to dictionary
		// Key is a large string which is stored in its own slab.
		existingValue := dictionary.Insert(
			inter,
			EmptyLocationRange,
			newStringValue(),
			NewUnmeteredInt64Value(int64(1)),
		)
		require.Equal(t, NilOptionalValue, existingValue)
		require.Equal(t, 1, dictionary.Count())

		// Check new dictionary element
		v, found := dictionary.Get(
			inter,
			EmptyLocationRange,
			newStringValue(),
		)
		require.True(t, found)
		require.Equal(t, Int64Value(1), v)

		// Update existing key with new value
		existingValue = dictionary.Insert(
			inter,
			EmptyLocationRange,
			newStringValue(),
			NewUnmeteredInt64Value(int64(2)),
		)
		require.NotEqual(t, Int64Value(1), existingValue)
		require.Equal(t, 1, dictionary.Count())

		// Check updated dictionary element
		v, found = dictionary.Get(
			inter,
			EmptyLocationRange,
			newStringValue(),
		)
		require.True(t, found)
		require.Equal(t, Int64Value(2), v)

		// Check storage containing only one root slab (dictionary root)
		checkRootSlabIDsInStorage(t, storage, []atree.SlabID{dictionary.SlabID()})
	})
}

func checkRootSlabIDsInStorage(t *testing.T, storage atree.SlabStorage, expectedRootSlabIDs []atree.SlabID) {
	rootSlabIDs, err := atree.CheckStorageHealth(storage, -1)
	require.NoError(t, err)

	// Get non-temp address slab IDs from rootSlabIDs
	nontempSlabIDs := make([]atree.SlabID, 0, len(rootSlabIDs))
	for rootSlabID := range rootSlabIDs {
		if !rootSlabID.HasTempAddress() {
			nontempSlabIDs = append(nontempSlabIDs, rootSlabID)
		}
	}

	require.ElementsMatch(t, expectedRootSlabIDs, nontempSlabIDs)
}
