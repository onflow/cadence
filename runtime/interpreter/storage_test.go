/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	storage := NewInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		common.AddressLocation{},
		WithStorage(storage),
	)
	require.NoError(t, err)

	value := NewCompositeValue(
		inter,
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

	value.SetMember(inter, ReturnEmptyLocationRange, fieldName, BoolValue(true))

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
	require.NoError(t, err)
	require.True(t, ok)

	storedValue := StoredValue(retrievedStorable, storage)

	require.IsType(t, storedValue, &CompositeValue{})
	storedComposite := storedValue.(*CompositeValue)

	RequireValuesEqual(
		t,
		inter,
		BoolValue(true),
		storedComposite.GetField(inter, ReturnEmptyLocationRange, fieldName),
	)
}

func TestArrayStorage(t *testing.T) {

	t.Parallel()

	importLocationHandlerFunc := func(inter *Interpreter, location common.Location) Import {
		elaboration := sema.NewElaboration()
		elaboration.CompositeTypes[testCompositeValueType.ID()] = testCompositeValueType
		return VirtualImport{Elaboration: elaboration}
	}

	t.Run("insert", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
			WithImportLocationHandler(importLocationHandlerFunc),
		)
		require.NoError(t, err)

		element := newTestCompositeValue(inter, common.Address{})

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.Address{},
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		// array + composite
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		require.False(t, bool(value.Contains(nil, nil, element)))

		value.Insert(
			inter,
			ReturnEmptyLocationRange,
			0,
			element,
		)

		require.True(t, bool(value.Contains(nil, nil, element)))

		// array + original composite element + new copy of composite element
		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		actual := storedArray.Get(inter, ReturnEmptyLocationRange, 0)

		RequireValuesEqual(t, inter, element, actual)
	})

	t.Run("remove", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
			WithImportLocationHandler(importLocationHandlerFunc),
		)
		require.NoError(t, err)

		element := newTestCompositeValue(inter, common.Address{})

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.Address{},
			element,
		)

		require.True(t, bool(value.Contains(nil, nil, element)))

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		// array + original composite element + new copy of composite element
		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			ReturnEmptyLocationRange,
			0,
		)

		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		require.False(t, bool(storedArray.Contains(nil, nil, element)))
	})
}

func TestDictionaryStorage(t *testing.T) {

	t.Parallel()

	t.Run("set some", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
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

		entryKey := NewStringValue("test")
		entryValue := BoolValue(true)

		value.SetKey(
			inter,
			ReturnEmptyLocationRange,
			entryKey,
			NewSomeValueNonCopying(entryValue),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		actual, ok := storedDictionary.Get(inter, ReturnEmptyLocationRange, entryKey)
		require.True(t, ok)

		RequireValuesEqual(t, inter, entryValue, actual)
	})

	t.Run("set nil", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewStringValue("test"),
			NewSomeValueNonCopying(BoolValue(true)),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.SetKey(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NilValue{},
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})

	t.Run("remove", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewStringValue("test"),
			NewSomeValueNonCopying(BoolValue(true)),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})

	t.Run("insert", func(t *testing.T) {

		t.Parallel()

		storage := NewInMemoryStorage()

		inter, err := NewInterpreter(
			nil,
			common.AddressLocation{},
			WithStorage(storage),
		)
		require.NoError(t, err)

		value := NewDictionaryValue(
			inter,
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
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NewSomeValueNonCopying(BoolValue(true)),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})
}

func TestStorageOverwriteAndRemove(t *testing.T) {

	t.Parallel()

	storage := NewInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		common.AddressLocation{},
		WithStorage(storage),
	)
	require.NoError(t, err)

	address := common.Address{}

	array1 := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		address,
		NewStringValue("first"),
	)

	const identifier = "test"

	storageMap := storage.GetStorageMap(address, "storage", true)
	storageMap.WriteValue(inter, identifier, array1)

	// Overwriting delete any existing child slabs

	array2 := NewArrayValue(
		inter,
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		address,
		NewStringValue("second"),
	)

	storageMap.WriteValue(inter, identifier, array2)

	// 2:
	// - storage map (atree ordered map)
	// - array (atree array)
	assert.Len(t, storage.Slabs, 2)

	// Writing nil is deletion and should delete any child slabs

	storageMap.WriteValue(inter, identifier, nil)

	// 1:
	// - storage map (atree ordered map)
	assert.Len(t, storage.Slabs, 1)
}
