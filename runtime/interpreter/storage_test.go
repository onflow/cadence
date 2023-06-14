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

package interpreter_test

import (
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestCompositeStorage(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		common.AddressLocation{},
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	value := NewCompositeValue(
		inter,
		EmptyLocationRange,
		TestLocation,
		"TestStruct",
		common.CompositeKindStructure,
		nil,
		testOwner,
	)

	require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
	require.NoError(t, err)
	require.True(t, ok)

	const fieldName = "test"

	value.SetMember(inter, EmptyLocationRange, fieldName, TrueValue)

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
	require.NoError(t, err)
	require.True(t, ok)

	storedValue := StoredValue(inter, retrievedStorable, storage)

	require.IsType(t, storedValue, &CompositeValue{})
	storedComposite := storedValue.(*CompositeValue)

	RequireValuesEqual(
		t,
		inter,
		TrueValue,
		storedComposite.GetField(inter, EmptyLocationRange, fieldName),
	)
}

func TestArrayStorage(t *testing.T) {

	t.Parallel()

	importLocationHandlerFunc := func(inter *Interpreter, location common.Location) Import {
		elaboration := sema.NewElaboration(nil)
		elaboration.SetCompositeType(
			testCompositeValueType.ID(),
			testCompositeValueType,
		)
		return VirtualImport{Elaboration: elaboration}
	}

	t.Run("insert", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{
				Storage:               storage,
				ImportLocationHandler: importLocationHandlerFunc,
			},
		)
		require.NoError(t, err)

		element := newTestCompositeValue(inter, common.ZeroAddress)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			EmptyLocationRange,
			VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.ZeroAddress,
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		// array + composite
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		require.False(t, bool(value.Contains(inter, EmptyLocationRange, element)))

		value.Insert(
			inter,
			EmptyLocationRange,
			0,
			element,
		)

		require.True(t, bool(value.Contains(inter, EmptyLocationRange, element)))

		// array + original composite element + new copy of composite element
		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		actual := storedArray.Get(inter, EmptyLocationRange, 0)

		RequireValuesEqual(t, inter, element, actual)
	})

	t.Run("remove", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{
				Storage:               storage,
				ImportLocationHandler: importLocationHandlerFunc,
			},
		)
		require.NoError(t, err)

		element := newTestCompositeValue(inter, common.ZeroAddress)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			EmptyLocationRange,
			VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.ZeroAddress,
			element,
		)

		require.True(t, bool(value.Contains(inter, EmptyLocationRange, element)))

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		// array + original composite element + new copy of composite element
		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			EmptyLocationRange,
			0,
		)

		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		require.False(t, bool(storedArray.Contains(inter, EmptyLocationRange, element)))
	})
}

func TestDictionaryStorage(t *testing.T) {

	t.Parallel()

	t.Run("set some", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{Storage: storage},
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			EmptyLocationRange,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		entryKey := NewUnmeteredStringValue("test")
		entryValue := TrueValue

		value.SetKey(
			inter,
			EmptyLocationRange,
			entryKey,
			NewUnmeteredSomeValueNonCopying(entryValue),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		actual, ok := storedDictionary.Get(inter, EmptyLocationRange, entryKey)
		require.True(t, ok)

		RequireValuesEqual(t, inter, entryValue, actual)
	})

	t.Run("set nil", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{Storage: storage},
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			EmptyLocationRange,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.SetKey(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
			Nil,
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})

	t.Run("remove", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{Storage: storage},
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			EmptyLocationRange,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})

	t.Run("insert", func(t *testing.T) {

		t.Parallel()

		storage := newUnmeteredInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			&Config{Storage: storage},
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			EmptyLocationRange,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Insert(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})
}

func TestInterpretStorageOverwriteAndRemove(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		common.AddressLocation{},
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	address := common.ZeroAddress

	array1 := NewArrayValue(
		inter,
		EmptyLocationRange,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		address,
		NewUnmeteredStringValue("first"),
	)

	const storageMapKey = StringStorageMapKey("test")

	storageMap := storage.GetStorageMap(address, "storage", true)
	storageMap.WriteValue(inter, storageMapKey, array1)

	// Overwriting delete any existing child slabs

	array2 := NewArrayValue(
		inter,
		EmptyLocationRange,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		address,
		NewUnmeteredStringValue("second"),
	)

	storageMap.WriteValue(inter, storageMapKey, array2)

	// 2:
	// - storage map (atree ordered map)
	// - array (atree array)
	assert.Len(t, storage.Slabs, 2)

	// Writing nil is deletion and should delete any child slabs

	storageMap.WriteValue(inter, storageMapKey, nil)

	// 1:
	// - storage map (atree ordered map)
	assert.Len(t, storage.Slabs, 1)
}
