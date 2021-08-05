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

	"github.com/fxamacker/atree"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
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
		utils.TestLocation,
		"TestStruct",
		common.CompositeKindStructure,
		NewStringValueOrderedMap(),
		testOwner,
	)

	require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	storable1, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
	require.NoError(t, err)
	require.True(t, ok)

	value.SetMember(inter, ReturnEmptyLocationRange, "test", BoolValue(true))

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	storable2, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
	require.NoError(t, err)
	require.True(t, ok)

	require.NotEqual(t, storable1, storable2)
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

		value := NewDictionaryValueUnownedNonCopying(
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			storage,
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable1, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Set(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NewSomeValueOwningNonCopying(BoolValue(true)),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable2, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		require.NotEqual(t, dictStorable1, dictStorable2)

		require.True(t, bool(value.Keys.Contains(NewStringValue("test"))))
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

		value := NewDictionaryValueUnownedNonCopying(
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			storage,
			NewStringValue("test"),
			NewSomeValueOwningNonCopying(BoolValue(true)),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable1, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Set(
			inter,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NilValue{},
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable2, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		require.NotEqual(t, dictStorable1, dictStorable2)

		require.False(t, bool(value.Keys.Contains(NewStringValue("test"))))
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

		value := NewDictionaryValueUnownedNonCopying(
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			storage,
			NewStringValue("test"),
			NewSomeValueOwningNonCopying(BoolValue(true)),
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable1, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter.Storage,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable2, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		require.NotEqual(t, dictStorable1, dictStorable2)

		require.False(t, bool(value.Keys.Contains(NewStringValue("test"))))
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

		value := NewDictionaryValueUnownedNonCopying(
			DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			storage,
		)

		require.NotEqual(t, atree.StorageIDUndefined, value.StorageID)

		// nested keys array + dictionary itself
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable1, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		value.Insert(
			inter.Storage,
			ReturnEmptyLocationRange,
			NewStringValue("test"),
			NewSomeValueOwningNonCopying(BoolValue(true)),
		)

		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		dictStorable2, ok, err := storage.BasicSlabStorage.Retrieve(value.StorageID)
		require.NoError(t, err)
		require.True(t, ok)

		require.NotEqual(t, dictStorable1, dictStorable2)

		require.True(t, bool(value.Keys.Contains(NewStringValue("test"))))
	})
}
