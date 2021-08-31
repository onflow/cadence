/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"

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
		storage,
		TestLocation,
		"TestStruct",
		common.CompositeKindStructure,
		NewStringValueOrderedMap(),
		testOwner,
	)

	require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
	require.NoError(t, err)
	require.True(t, ok)

	const fieldName = "test"

	value.SetMember(inter, ReturnEmptyLocationRange, fieldName, BoolValue(true))

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
	require.NoError(t, err)
	require.True(t, ok)

	storedValue, err := StoredValue(retrievedStorable, storage)
	require.NoError(t, err)

	require.IsType(t, storedValue, &CompositeValue{})
	storedComposite := storedValue.(*CompositeValue)

	RequireValuesEqual(t,
		BoolValue(true),
		storedComposite.GetField(fieldName),
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

		element := newTestCompositeValue(inter.Storage, common.Address{})

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: element.StaticType(),
			},
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID())

		// array + composite
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		require.False(t, bool(value.Contains(element)))

		value.Insert(
			inter,
			ReturnEmptyLocationRange,
			0,
			element,
		)

		require.True(t, bool(value.Contains(element)))

		// array + original composite element + new copy of composite element
		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		actual := storedArray.GetIndex(ReturnEmptyLocationRange, 0)

		RequireValuesEqual(t, element, actual)
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

		element := newTestCompositeValue(inter.Storage, common.Address{})

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		value := NewArrayValue(
			inter,
			VariableSizedStaticType{
				Type: element.StaticType(),
			},
			element,
		)

		require.True(t, bool(value.Contains(element)))

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

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &ArrayValue{})
		storedArray := storedValue.(*ArrayValue)

		require.False(t, bool(storedArray.Contains(element)))
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

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		entryKey := NewStringValue("test")
		entryValue := BoolValue(true)

		value.Set(
			inter,
			ReturnEmptyLocationRange,
			entryKey,
			NewSomeValueNonCopying(entryValue),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		actual, _, ok := storedDictionary.GetKey(entryKey)
		require.True(t, ok)

		RequireValuesEqual(t, entryValue, actual)

		require.True(t, bool(storedDictionary.Keys.Contains(entryKey)))
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

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Set(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NilValue{},
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		require.False(t, bool(storedDictionary.Keys.Contains(NewStringValue("test"))))
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

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		require.False(t, bool(storedDictionary.Keys.Contains(NewStringValue("test"))))
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

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Insert(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NewSomeValueNonCopying(BoolValue(true)),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		storedValue, err := StoredValue(retrievedStorable, storage)
		require.NoError(t, err)

		require.IsType(t, storedValue, &DictionaryValue{})
		storedDictionary := storedValue.(*DictionaryValue)

		require.True(t, bool(storedDictionary.Keys.Contains(NewStringValue("test"))))
	})
}
