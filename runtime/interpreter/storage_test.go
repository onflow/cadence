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
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
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

	require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
	require.NoError(t, err)
	require.True(t, ok)

	const fieldName = "test"

	value.SetMember(inter, EmptyLocationRange, fieldName, TrueValue)

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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

func TestInclusiveRangeStorage(t *testing.T) {

	t.Parallel()

	storage := newUnmeteredInMemoryStorage()

	inter, err := NewInterpreter(
		nil,
		common.AddressLocation{},
		&Config{Storage: storage},
	)
	require.NoError(t, err)

	value := NewInclusiveRangeValueWithStep(
		inter,
		EmptyLocationRange,
		NewUnmeteredInt16Value(1),
		NewUnmeteredInt16Value(100),
		NewUnmeteredInt16Value(5),
		NewInclusiveRangeStaticType(inter, PrimitiveStaticTypeInt16),
		sema.NewInclusiveRangeType(inter, sema.Int16Type),
	)

	require.NotEqual(t, atree.ValueID{}, value.ValueID())

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
	require.NoError(t, err)
	require.True(t, ok)

	// Ensure that updating a field (e.g. step) works
	const stepFieldName = "step"

	value.SetMember(inter, EmptyLocationRange, stepFieldName, NewUnmeteredInt16Value(10))

	require.Equal(t, 1, storage.BasicSlabStorage.Count())

	retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
	require.NoError(t, err)
	require.True(t, ok)

	storedValue := StoredValue(inter, retrievedStorable, storage)

	// InclusiveRange is stored as a CompositeValue.
	require.IsType(t, storedValue, &CompositeValue{})
	storedComposite := storedValue.(*CompositeValue)

	RequireValuesEqual(
		t,
		inter,
		NewUnmeteredInt16Value(10),
		storedComposite.GetField(inter, EmptyLocationRange, stepFieldName),
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
			&VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.ZeroAddress,
		)

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		// array + composite
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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

		// array + new copy of composite element
		// NOTE: original composite value is inlined in parent array.
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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
			&VariableSizedStaticType{
				Type: element.StaticType(inter),
			},
			common.ZeroAddress,
			element,
		)

		require.True(t, bool(value.Contains(inter, EmptyLocationRange, element)))

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		// array + new copy of composite element
		// NOTE: original composite value is inlined in parent array.
		require.Equal(t, 2, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			EmptyLocationRange,
			0,
		)

		require.Equal(t, 3, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
		)

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
		require.NoError(t, err)
		require.True(t, ok)

		value.SetKey(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
			Nil,
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Remove(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
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
			&DictionaryStaticType{
				KeyType:   PrimitiveStaticTypeString,
				ValueType: PrimitiveStaticTypeAnyStruct,
			},
		)

		require.NotEqual(t, atree.SlabIDUndefined, value.SlabID())

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		_, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
		require.NoError(t, err)
		require.True(t, ok)

		value.Insert(
			inter,
			EmptyLocationRange,
			NewUnmeteredStringValue("test"),
			NewUnmeteredSomeValueNonCopying(TrueValue),
		)

		require.Equal(t, 1, storage.BasicSlabStorage.Count())

		retrievedStorable, ok, err := storage.BasicSlabStorage.Retrieve(value.SlabID())
		require.NoError(t, err)
		require.True(t, ok)

		storedValue := StoredValue(inter, retrievedStorable, storage)

		require.IsType(t, storedValue, &DictionaryValue{})
	})
}

func TestStorageOverwriteAndRemove(t *testing.T) {

	t.Parallel()

	t.Run("overwrite inlined value with inlined value", func(t *testing.T) {

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
			&VariableSizedStaticType{
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
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			address,
			NewUnmeteredStringValue("second"),
		)

		storageMap.WriteValue(inter, storageMapKey, array2)

		// 1:
		// - storage map (atree ordered map)
		// NOTE: array (atree array) is inlined in storage map
		assert.Len(t, storage.Slabs, 1)

		// Writing nil is deletion and should delete any child slabs

		storageMap.WriteValue(inter, storageMapKey, nil)

		// 1:
		// - storage map (atree ordered map)
		assert.Len(t, storage.Slabs, 1)
	})

	// TODO: add subtests to
	// - overwrite inlined value with not inlined value
	// - overwrite not inlined value with not inlined value
	// - overwrite not inlined value with inlined value
}

func TestNestedContainerMutationAfterMove(t *testing.T) {

	t.Parallel()

	testStructType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "TestStruct",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	testResourceType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "TestResource",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	const fieldName = "test"

	for _, testCompositeType := range []*sema.CompositeType{
		testStructType,
		testResourceType,
	} {
		fieldMember := sema.NewFieldMember(
			nil,
			testCompositeType,
			sema.UnauthorizedAccess,
			ast.VariableKindVariable,
			fieldName,
			sema.UInt8Type,
			"",
		)
		testCompositeType.Members.Set(fieldName, fieldMember)
	}

	importLocationHandlerFunc := func(inter *Interpreter, location common.Location) Import {
		elaboration := sema.NewElaboration(nil)
		elaboration.SetCompositeType(
			testStructType.ID(),
			testStructType,
		)
		elaboration.SetCompositeType(
			testResourceType.ID(),
			testResourceType,
		)
		return VirtualImport{Elaboration: elaboration}
	}

	t.Run("struct, move from array to array", func(t *testing.T) {

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

		containerValue1 := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		containerValue2 := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		newChildValue := func(value uint8) *CompositeValue {
			return NewCompositeValue(
				inter,
				EmptyLocationRange,
				TestLocation,
				"TestStruct",
				common.CompositeKindStructure,
				[]CompositeField{
					{
						Name:  fieldName,
						Value: NewUnmeteredUInt8Value(value),
					},
				},
				common.ZeroAddress,
			)
		}

		childValue1 := newChildValue(0)

		require.Equal(t, "[]", containerValue1.String())
		require.Equal(t, "[]", containerValue2.String())

		containerValue1.Append(inter, EmptyLocationRange, NewUnmeteredUInt8Value(1))
		containerValue2.Append(inter, EmptyLocationRange, NewUnmeteredUInt8Value(2))

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())

		require.Equal(t, "S.test.TestStruct(test: 0)", childValue1.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(3))

		require.Equal(t, "S.test.TestStruct(test: 3)", childValue1.String())

		childValue2 := childValue1.Transfer(
			inter,
			EmptyLocationRange,
			atree.Address{},
			false,
			nil,
			map[atree.ValueID]struct{}{},
			true, // childValue1 is standalone before being inserted into containerValue1.
		).(*CompositeValue)

		containerValue1.Append(inter, EmptyLocationRange, childValue1)
		// Append invalidated, get again
		childValue1 = containerValue1.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestStruct(test: 3)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 3)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 3)", childValue2.String())

		childValue2.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(4))

		require.Equal(t, "[1, S.test.TestStruct(test: 3)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 3)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(5))

		require.Equal(t, "[1, S.test.TestStruct(test: 5)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())

		childValue3 := containerValue1.Remove(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		// TODO: fix
		require.Equal(t, "S.test.TestStruct()", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(6))

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 6)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())

		childValue4 := newChildValue(7)

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 6)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 7)", childValue4.String())

		containerValue1.Append(inter, EmptyLocationRange, childValue4)
		// Append invalidated, get again
		childValue4 = containerValue1.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestStruct(test: 7)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 6)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 7)", childValue4.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(8))

		require.Equal(t, "[1, S.test.TestStruct(test: 7)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 8)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 7)", childValue4.String())

		childValue4.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(9))

		require.Equal(t, "[1, S.test.TestStruct(test: 9)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 8)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 9)", childValue4.String())

		containerValue2.Append(inter, EmptyLocationRange, childValue3)
		// Append invalidated, get again
		childValue3 = containerValue2.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestStruct(test: 9)]", containerValue1.String())
		require.Equal(t, "[2, S.test.TestStruct(test: 5)]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 8)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 5)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 9)", childValue4.String())

		childValue3.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(10))

		require.Equal(t, "[1, S.test.TestStruct(test: 9)]", containerValue1.String())
		require.Equal(t, "[2, S.test.TestStruct(test: 10)]", containerValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 8)", childValue1.String())
		require.Equal(t, "S.test.TestStruct(test: 4)", childValue2.String())
		require.Equal(t, "S.test.TestStruct(test: 10)", childValue3.String())
		require.Equal(t, "S.test.TestStruct(test: 9)", childValue4.String())
	})

	t.Run("resource, move from array to array", func(t *testing.T) {

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

		containerValue1 := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		containerValue2 := NewArrayValue(
			inter,
			EmptyLocationRange,
			&VariableSizedStaticType{
				Type: PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		newChildValue := func(value uint8) *CompositeValue {
			return NewCompositeValue(
				inter,
				EmptyLocationRange,
				TestLocation,
				"TestResource",
				common.CompositeKindResource,
				[]CompositeField{
					{
						Name:  fieldName,
						Value: NewUnmeteredUInt8Value(value),
					},
				},
				common.ZeroAddress,
			)
		}

		childValue1 := newChildValue(0)

		require.Equal(t, "[]", containerValue1.String())
		require.Equal(t, "[]", containerValue2.String())

		containerValue1.Append(inter, EmptyLocationRange, NewUnmeteredUInt8Value(1))
		containerValue2.Append(inter, EmptyLocationRange, NewUnmeteredUInt8Value(2))

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())

		require.Equal(t, "S.test.TestResource(test: 0)", childValue1.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(3))

		require.Equal(t, "S.test.TestResource(test: 3)", childValue1.String())

		ref1 := NewEphemeralReferenceValue(
			inter,
			UnauthorizedAccess,
			childValue1,
			testResourceType,
			EmptyLocationRange,
		)

		containerValue1.Append(inter, EmptyLocationRange, childValue1)
		// Append invalidated, get again
		childValue1 = containerValue1.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestResource(test: 3)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 3)", childValue1.String())
		require.Nil(t, ref1.Value)

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(4))

		require.Equal(t, "[1, S.test.TestResource(test: 4)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 4)", childValue1.String())
		require.Nil(t, ref1.Value)

		// Cannot use ref1, as it's invalidated
		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(5))

		require.Equal(t, "[1, S.test.TestResource(test: 5)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 5)", childValue1.String())
		require.Nil(t, ref1.Value)

		childValue2 := containerValue1.Remove(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 5)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 5)", childValue2.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(6))

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 6)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 6)", childValue2.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(7))

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 7)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 7)", childValue2.String())

		// TODO: rename childValue4 to childValue3
		childValue4 := newChildValue(8)

		require.Equal(t, "[1]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 7)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 7)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 8)", childValue4.String())

		containerValue1.Append(inter, EmptyLocationRange, childValue4)
		// Append invalidated, get again
		childValue4 = containerValue1.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestResource(test: 8)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 7)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 7)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 8)", childValue4.String())

		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(9))

		require.Equal(t, "[1, S.test.TestResource(test: 8)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 9)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 9)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 8)", childValue4.String())

		// Cannot use ref1, as it's invalidated
		childValue1.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(10))

		require.Equal(t, "[1, S.test.TestResource(test: 8)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 10)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 10)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 8)", childValue4.String())

		childValue4.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(11))

		require.Equal(t, "[1, S.test.TestResource(test: 11)]", containerValue1.String())
		require.Equal(t, "[2]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 10)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 10)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 11)", childValue4.String())

		containerValue2.Append(inter, EmptyLocationRange, childValue2)
		// Append invalidated, get again
		childValue2 = containerValue2.Get(inter, EmptyLocationRange, 1).(*CompositeValue)

		require.Equal(t, "[1, S.test.TestResource(test: 11)]", containerValue1.String())
		require.Equal(t, "[2, S.test.TestResource(test: 10)]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 10)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 10)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 11)", childValue4.String())

		childValue2.SetMember(inter, EmptyLocationRange, fieldName, NewUnmeteredUInt8Value(12))

		require.Equal(t, "[1, S.test.TestResource(test: 11)]", containerValue1.String())
		require.Equal(t, "[2, S.test.TestResource(test: 12)]", containerValue2.String())
		require.Equal(t, "S.test.TestResource(test: 12)", childValue1.String())
		require.Nil(t, ref1.Value)
		require.Equal(t, "S.test.TestResource(test: 12)", childValue2.String())
		require.Equal(t, "S.test.TestResource(test: 11)", childValue4.String())
	})
}
