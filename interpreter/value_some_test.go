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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestSomeValueUnwrapAtreeValue(t *testing.T) {

	const (
		cborTagSize                                   = 2
		someStorableWithMultipleNestedLevelsArraySize = 1
	)

	t.Parallel()

	t.Run("SomeValue(bool)", func(t *testing.T) {
		bv := interpreter.BoolValue(true)

		v := interpreter.NewUnmeteredSomeValueNonCopying(bv)

		unwrappedValue, wrapperSize := v.UnwrapAtreeValue()
		require.Equal(t, bv, unwrappedValue)
		require.Equal(t, uint64(cborTagSize), wrapperSize)
	})

	t.Run("SomeValue(SomeValue(bool))", func(t *testing.T) {
		bv := interpreter.BoolValue(true)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				bv))

		unwrappedValue, wrapperSize := v.UnwrapAtreeValue()
		require.Equal(t, bv, unwrappedValue)
		require.Equal(t, uint64(cborTagSize+someStorableWithMultipleNestedLevelsArraySize+1), wrapperSize)
	})

	t.Run("SomeValue(SomeValue(ArrayValue(...)))", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		values := []interpreter.Value{
			interpreter.NewUnmeteredUInt64Value(0),
			interpreter.NewUnmeteredUInt64Value(1),
		}

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				array))

		unwrappedValue, wrapperSize := v.UnwrapAtreeValue()
		require.IsType(t, &atree.Array{}, unwrappedValue)
		require.Equal(t, uint64(cborTagSize+someStorableWithMultipleNestedLevelsArraySize+1), wrapperSize)

		atreeArray := unwrappedValue.(*atree.Array)
		require.Equal(t, atree.Address(address), atreeArray.Address())
		require.Equal(t, uint64(len(values)), atreeArray.Count())

		for i, expectedValue := range values {
			v, err := atreeArray.Get(uint64(i))
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(DictionaryValue(...)))", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		values := []interpreter.Value{
			interpreter.NewUnmeteredUInt64Value(0),
			interpreter.NewUnmeteredStringValue("a"),
			interpreter.NewUnmeteredUInt64Value(1),
			interpreter.NewUnmeteredStringValue("b"),
		}

		dict := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				dict))

		unwrappedValue, wrapperSize := v.UnwrapAtreeValue()
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)
		require.Equal(t, uint64(cborTagSize+someStorableWithMultipleNestedLevelsArraySize+1), wrapperSize)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(values)/2), atreeMap.Count())

		valueComparator := func(
			storage atree.SlabStorage,
			atreeValue atree.Value,
			otherStorable atree.Storable,
		) (bool, error) {
			value := interpreter.MustConvertStoredValue(inter, atreeValue)
			otherValue := interpreter.StoredValue(inter, otherStorable, storage)
			return value.(interpreter.EquatableValue).Equal(inter, interpreter.EmptyLocationRange, otherValue), nil
		}

		hashInputProvider := func(
			value atree.Value,
			scratch []byte,
		) ([]byte, error) {
			hashInput := interpreter.MustConvertStoredValue(inter, value).(interpreter.HashableValue).
				HashInput(inter, interpreter.EmptyLocationRange, scratch)
			return hashInput, nil
		}

		for i := 0; i < len(values); i += 2 {
			key := values[i]
			expectedValue := values[i+1]

			v, err := atreeMap.Get(
				valueComparator,
				hashInputProvider,
				key,
			)
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(CompositeValue(...)))", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		identifier := "test"

		location := common.AddressLocation{
			Address: address,
			Name:    identifier,
		}

		kind := common.CompositeKindStructure

		fields := []interpreter.CompositeField{
			interpreter.NewUnmeteredCompositeField(
				"field1",
				interpreter.NewUnmeteredStringValue("a"),
			),
			interpreter.NewUnmeteredCompositeField(
				"field2",
				interpreter.NewUnmeteredStringValue("b"),
			),
		}

		composite := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			identifier,
			kind,
			fields,
			address,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				composite))

		unwrappedValue, wrapperSize := v.UnwrapAtreeValue()
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)
		require.Equal(t, uint64(cborTagSize+someStorableWithMultipleNestedLevelsArraySize+1), wrapperSize)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(fields)), atreeMap.Count())

		for _, f := range fields {
			v, err := atreeMap.Get(
				interpreter.StringAtreeValueComparator,
				interpreter.StringAtreeValueHashInput,
				interpreter.StringAtreeValue(f.Name),
			)
			require.NoError(t, err)
			require.Equal(t, f.Value, v)
		}
	})
}

func TestSomeStorableUnwrapAtreeStorable(t *testing.T) {

	t.Parallel()

	address := common.Address{'A'}

	t.Run("SomeValue(bool)", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.BoolValue(true))

		const maxInlineSize = 1024 / 4
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.Equal(t, interpreter.BoolValue(true), unwrappedStorable)
	})

	t.Run("SomeValue(SomeValue(bool))", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.BoolValue(true)))

		const maxInlineSize = 1024 / 4
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.Equal(t, interpreter.BoolValue(true), unwrappedStorable)
	})

	t.Run("SomeValue(SomeValue(ArrayValue(...))), small ArrayValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		values := []interpreter.Value{
			interpreter.NewUnmeteredUInt64Value(0),
			interpreter.NewUnmeteredUInt64Value(1),
		}

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				array))

		const maxInlineSize = 1024 / 4
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, &atree.ArrayDataSlab{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(*atree.ArrayDataSlab).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.Array{}, unwrappedValue)

		atreeArray := unwrappedValue.(*atree.Array)
		require.Equal(t, atree.Address(address), atreeArray.Address())
		require.Equal(t, uint64(len(values)), atreeArray.Count())

		for i, expectedValue := range values {
			v, err := atreeArray.Get(uint64(i))
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(ArrayValue(...))), large ArrayValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		const valuesCount = 40
		values := make([]interpreter.Value, valuesCount)
		for i := range valuesCount {
			values[i] = interpreter.NewUnmeteredUInt64Value(uint64(i))
		}

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				array))

		const maxInlineSize = 1024 / 8
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, atree.SlabIDStorable{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(atree.SlabIDStorable).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.Array{}, unwrappedValue)

		atreeArray := unwrappedValue.(*atree.Array)
		require.Equal(t, atree.Address(address), atreeArray.Address())
		require.Equal(t, uint64(len(values)), atreeArray.Count())

		for i, expectedValue := range values {
			v, err := atreeArray.Get(uint64(i))
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(DictionaryValue(...))), small DictionaryValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()
		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		values := []interpreter.Value{
			interpreter.NewUnmeteredUInt64Value(0),
			interpreter.NewUnmeteredStringValue("a"),
			interpreter.NewUnmeteredUInt64Value(1),
			interpreter.NewUnmeteredStringValue("b"),
		}

		dict := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				dict))

		const maxInlineSize = 1024 / 4
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, &atree.MapDataSlab{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(*atree.MapDataSlab).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(values)/2), atreeMap.Count())

		valueComparator := func(
			storage atree.SlabStorage,
			atreeValue atree.Value,
			otherStorable atree.Storable,
		) (bool, error) {
			value := interpreter.MustConvertStoredValue(inter, atreeValue)
			otherValue := interpreter.StoredValue(inter, otherStorable, storage)
			return value.(interpreter.EquatableValue).Equal(inter, interpreter.EmptyLocationRange, otherValue), nil
		}

		hashInputProvider := func(
			value atree.Value,
			scratch []byte,
		) ([]byte, error) {
			hashInput := interpreter.MustConvertStoredValue(inter, value).(interpreter.HashableValue).
				HashInput(inter, interpreter.EmptyLocationRange, scratch)
			return hashInput, nil
		}

		for i := 0; i < len(values); i += 2 {
			key := values[i]
			expectedValue := values[i+1]

			v, err := atreeMap.Get(
				valueComparator,
				hashInputProvider,
				key,
			)
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(DictionaryValue(...))), large DictionaryValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		const valuesCount = 20
		values := make([]interpreter.Value, valuesCount*2)

		char := 'a'
		for i := 0; i < len(values); i += 2 {
			values[i] = interpreter.NewUnmeteredUInt64Value(uint64(i))
			values[i+1] = interpreter.NewUnmeteredStringValue(string(char))
			char += 1
		}

		dict := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			address,
			values...,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				dict))

		const maxInlineSize = 1024 / 8
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, atree.SlabIDStorable{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(atree.SlabIDStorable).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(values)/2), atreeMap.Count())

		valueComparator := func(
			storage atree.SlabStorage,
			atreeValue atree.Value,
			otherStorable atree.Storable,
		) (bool, error) {
			value := interpreter.MustConvertStoredValue(inter, atreeValue)
			otherValue := interpreter.StoredValue(inter, otherStorable, storage)
			return value.(interpreter.EquatableValue).Equal(inter, interpreter.EmptyLocationRange, otherValue), nil
		}

		hashInputProvider := func(
			value atree.Value,
			scratch []byte,
		) ([]byte, error) {
			hashInput := interpreter.MustConvertStoredValue(inter, value).(interpreter.HashableValue).
				HashInput(inter, interpreter.EmptyLocationRange, scratch)
			return hashInput, nil
		}

		for i := 0; i < len(values); i += 2 {
			key := values[i]
			expectedValue := values[i+1]

			v, err := atreeMap.Get(
				valueComparator,
				hashInputProvider,
				key,
			)
			require.NoError(t, err)
			require.Equal(t, expectedValue, v)
		}
	})

	t.Run("SomeValue(SomeValue(CompositeValue(...))), small CompositeValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		identifier := "test"

		location := common.AddressLocation{
			Address: address,
			Name:    identifier,
		}

		kind := common.CompositeKindStructure

		fields := []interpreter.CompositeField{
			interpreter.NewUnmeteredCompositeField(
				"field1",
				interpreter.NewUnmeteredStringValue("a"),
			),
			interpreter.NewUnmeteredCompositeField(
				"field2",
				interpreter.NewUnmeteredStringValue("b"),
			),
		}

		composite := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			identifier,
			kind,
			fields,
			address,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				composite))

		const maxInlineSize = 1024 / 4
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, &atree.MapDataSlab{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(*atree.MapDataSlab).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(fields)), atreeMap.Count())

		for _, f := range fields {
			v, err := atreeMap.Get(
				interpreter.StringAtreeValueComparator,
				interpreter.StringAtreeValueHashInput,
				interpreter.StringAtreeValue(f.Name),
			)
			require.NoError(t, err)
			require.Equal(t, f.Value, v)
		}
	})

	t.Run("SomeValue(SomeValue(CompositeValue(...))), large CompositeValue", func(t *testing.T) {
		storage := newUnmeteredInMemoryStorage()

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Program:     ast.NewProgram(nil, []ast.Declaration{}),
				Elaboration: sema.NewElaboration(nil),
			},
			TestLocation,
			&interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					return interpreter.VirtualImport{
						Elaboration: inter.Program.Elaboration,
					}
				},
			},
		)
		require.NoError(t, err)

		address := common.Address{'A'}

		identifier := "test"

		location := common.AddressLocation{
			Address: address,
			Name:    identifier,
		}

		kind := common.CompositeKindStructure

		const fieldsCount = 20
		fields := make([]interpreter.CompositeField, fieldsCount)
		char := 'a'
		for i := range len(fields) {
			fields[i] = interpreter.NewUnmeteredCompositeField(
				fmt.Sprintf("field%d", i),
				interpreter.NewUnmeteredStringValue(string(char)),
			)
			char += 1
		}

		composite := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			identifier,
			kind,
			fields,
			address,
		)

		v := interpreter.NewUnmeteredSomeValueNonCopying(
			interpreter.NewUnmeteredSomeValueNonCopying(
				composite))

		const maxInlineSize = 1024 / 8
		storable, err := v.Storable(storage, atree.Address(address), maxInlineSize)
		require.NoError(t, err)
		require.IsType(t, interpreter.SomeStorable{}, storable)

		unwrappedStorable := storable.(interpreter.SomeStorable).UnwrapAtreeStorable()
		require.IsType(t, atree.SlabIDStorable{}, unwrappedStorable)

		unwrappedValue, err := unwrappedStorable.(atree.SlabIDStorable).StoredValue(storage)
		require.NoError(t, err)
		require.IsType(t, &atree.OrderedMap{}, unwrappedValue)

		// Verify unwrapped value
		atreeMap := unwrappedValue.(*atree.OrderedMap)
		require.Equal(t, atree.Address(address), atreeMap.Address())
		require.Equal(t, uint64(len(fields)), atreeMap.Count())

		for _, f := range fields {
			v, err := atreeMap.Get(
				interpreter.StringAtreeValueComparator,
				interpreter.StringAtreeValueHashInput,
				interpreter.StringAtreeValue(f.Name),
			)
			require.NoError(t, err)
			require.Equal(t, f.Value, v)
		}
	})
}
