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
	"flag"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var defaultRandomValueLimits = randomValueLimits{
	containerMaxDepth:  3,
	containerMaxSize:   50,
	compositeMaxFields: 10,
}

var runSmokeTests = flag.Bool("runSmokeTests", false, "Run smoke tests on values")
var validateAtree = flag.Bool("validateAtree", false, "Enable atree validation")
var smokeTestSeed = flag.Int64("smokeTestSeed", -1, "Seed for prng (-1 specifies current Unix time)")

func newRandomValueTestInterpreter(t *testing.T) (inter *interpreter.Interpreter, resetStorage func()) {

	config := &interpreter.Config{
		ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			return interpreter.VirtualImport{
				Elaboration: inter.Program.Elaboration,
			}
		},
		AtreeStorageValidationEnabled: *validateAtree,
		AtreeValueValidationEnabled:   *validateAtree,
	}

	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Elaboration: sema.NewElaboration(nil),
		},
		utils.TestLocation,
		config,
	)
	require.NoError(t, err)

	ledger := NewTestLedger(nil, nil)

	resetStorage = func() {
		if config.Storage != nil {
			storage := config.Storage.(*runtime.Storage)
			err := storage.Commit(inter, false)
			require.NoError(t, err)
		}
		config.Storage = runtime.NewStorage(ledger, nil)
	}

	resetStorage()

	return inter, resetStorage
}

func importValue(t *testing.T, inter *interpreter.Interpreter, value cadence.Value) interpreter.Value {

	switch value := value.(type) {
	case cadence.Array:
		// Work around for "cannot import array: elements do not belong to the same type",
		// caused by import of array without expected type, which leads to inference of the element type:
		// Create an empty array with an expected type, then append imported elements to it.

		arrayResult, err := runtime.ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			cadence.Array{},
			sema.NewVariableSizedType(nil, sema.AnyStructType),
		)
		require.NoError(t, err)
		require.IsType(t, &interpreter.ArrayValue{}, arrayResult)
		array := arrayResult.(*interpreter.ArrayValue)

		for _, element := range value.Values {
			array.Append(
				inter,
				interpreter.EmptyLocationRange,
				importValue(t, inter, element),
			)
		}

		return array

	case cadence.Dictionary:
		// Work around for "cannot import dictionary: keys does not belong to the same type",
		// caused by import of dictionary without expected type, which leads to inference of the key type:
		// Create an empty dictionary with an expected type, then append imported key-value pairs to it.

		dictionaryResult, err := runtime.ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			cadence.Dictionary{},
			sema.NewDictionaryType(
				nil,
				sema.HashableStructType,
				sema.AnyStructType,
			),
		)
		require.NoError(t, err)
		require.IsType(t, &interpreter.DictionaryValue{}, dictionaryResult)
		dictionary := dictionaryResult.(*interpreter.DictionaryValue)

		for _, pair := range value.Pairs {
			dictionary.Insert(
				inter,
				interpreter.EmptyLocationRange,
				importValue(t, inter, pair.Key),
				importValue(t, inter, pair.Value),
			)
		}

		return dictionary

	case cadence.Struct:

		structResult, err := runtime.ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			cadence.Struct{
				StructType: value.StructType,
			},
			nil,
		)
		require.NoError(t, err)
		require.IsType(t, &interpreter.CompositeValue{}, structResult)
		composite := structResult.(*interpreter.CompositeValue)

		for fieldName, fieldValue := range value.FieldsMappedByName() {
			composite.SetMember(
				inter,
				interpreter.EmptyLocationRange,
				fieldName,
				importValue(t, inter, fieldValue),
			)
		}

		return composite

	case cadence.Optional:

		if value.Value == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewUnmeteredSomeValueNonCopying(
			importValue(t, inter, value.Value),
		)

	default:
		result, err := runtime.ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			nil,
		)
		require.NoError(t, err)
		return result
	}
}

func withoutAtreeStorageValidationEnabled[T any](inter *interpreter.Interpreter, f func() T) T {
	config := inter.SharedState.Config
	original := config.AtreeStorageValidationEnabled
	config.AtreeStorageValidationEnabled = false
	result := f()
	config.AtreeStorageValidationEnabled = original
	return result
}

func TestInterpretRandomDictionaryOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	t.Parallel()

	orgOwner := common.Address{'A'}

	const dictionaryStorageMapKey = interpreter.StringStorageMapKey("dictionary")

	createDictionary := func(
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.DictionaryValue,
		cadence.Dictionary,
		func() *interpreter.DictionaryValue,
	) {
		expectedValue := r.randomDictionaryValue(inter, 0)

		keyValues := make([]interpreter.Value, 2*len(expectedValue.Pairs))
		for i, pair := range expectedValue.Pairs {

			key := importValue(t, inter, pair.Key)
			value := importValue(t, inter, pair.Value)

			keyValues[i*2] = key
			keyValues[i*2+1] = value
		}

		// Construct a dictionary directly in the owner's account.
		// However, the dictionary is not referenced by the root of the storage yet
		// (a storage map), so atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab" errors.

		dictionary := withoutAtreeStorageValidationEnabled(
			inter,
			func() *interpreter.DictionaryValue {
				return interpreter.NewDictionaryValueWithAddress(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeHashableStruct,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					orgOwner,
					keyValues...,
				)
			},
		)

		// Store the dictionary in a storage map, so that the dictionary's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				dictionaryStorageMapKey,
				dictionary,
			)

		reloadDictionary := func() *interpreter.DictionaryValue {
			storageMap := inter.Storage().GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, dictionaryStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.DictionaryValue{}, readValue)
			return readValue.(*interpreter.DictionaryValue)
		}

		return dictionary, expectedValue, reloadDictionary
	}

	checkDictionary := func(
		inter *interpreter.Interpreter,
		dictionary *interpreter.DictionaryValue,
		expectedValue cadence.Dictionary,
		expectedOwner common.Address,
	) {
		require.Equal(t, len(expectedValue.Pairs), dictionary.Count())

		for _, pair := range expectedValue.Pairs {
			pairKey := importValue(t, inter, pair.Key)

			exists := dictionary.ContainsKey(inter, interpreter.EmptyLocationRange, pairKey)
			require.True(t, bool(exists))

			value, found := dictionary.Get(inter, interpreter.EmptyLocationRange, pairKey)
			require.True(t, found)

			pairValue := importValue(t, inter, pair.Value)
			utils.AssertValuesEqual(t, inter, pairValue, value)
		}

		owner := dictionary.GetOwner()
		assert.Equal(t, expectedOwner, owner)
	}

	doubleCheckDictionary := func(
		inter *interpreter.Interpreter,
		resetStorage func(),
		dictionary *interpreter.DictionaryValue,
		expectedValue cadence.Dictionary,
		expectedOwner common.Address,
	) {
		// Check the values of the dictionary.
		// Once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkDictionary(
				inter,
				dictionary,
				expectedValue,
				expectedOwner,
			)

			resetStorage()
		}
	}

	checkIteration := func(
		inter *interpreter.Interpreter,
		resetStorage func(),
		dictionary *interpreter.DictionaryValue,
		expectedValue cadence.Dictionary,
	) {
		// Index the expected key-value pairs for lookup during iteration

		indexedExpected := map[any]interpreter.DictionaryEntryValues{}
		for _, pair := range expectedValue.Pairs {
			pairKey := importValue(t, inter, pair.Key)

			mapKey := mapKey(inter, pairKey)

			require.NotContains(t, indexedExpected, mapKey)
			indexedExpected[mapKey] = interpreter.DictionaryEntryValues{
				Key:   pairKey,
				Value: importValue(t, inter, pair.Value),
			}
		}

		// Iterate over the values of the created dictionary.
		// Once right after construction, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			require.Equal(t, len(expectedValue.Pairs), dictionary.Count())

			var iterations int

			dictionary.Iterate(
				inter,
				interpreter.EmptyLocationRange,
				func(key, value interpreter.Value) (resume bool) {

					mapKey := mapKey(inter, key)
					require.Contains(t, indexedExpected, mapKey)

					pair := indexedExpected[mapKey]

					utils.AssertValuesEqual(t, inter, pair.Key, key)
					utils.AssertValuesEqual(t, inter, pair.Value, value)

					iterations += 1

					return true
				},
			)

			assert.Equal(t, len(expectedValue.Pairs), iterations)

			resetStorage()
		}
	}

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue, _ := createDictionary(&r, inter)

		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)
	})

	t.Run("iterate", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue, _ := createDictionary(&r, inter)

		checkIteration(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
		)
	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue, _ := createDictionary(&r, inter)

		resetStorage()

		// Transfer the dictionary to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		).(*interpreter.DictionaryValue)

		// Store the transferred dictionary in a storage map, so that the dictionary's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				newOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				interpreter.StringStorageMapKey("transferred_dictionary"),
				transferred,
			)

		// Both original and transferred dictionary should contain the expected values
		// Check once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkDictionary(
				inter,
				original,
				expectedValue,
				orgOwner,
			)

			checkDictionary(
				inter,
				transferred,
				expectedValue,
				newOwner,
			)

			resetStorage()
		}

		// Deep remove the original dictionary

		// TODO: is has no parent container = true correct?
		original.DeepRemove(inter, true)

		if !original.Inlined() {
			err := inter.Storage().Remove(original.SlabID())
			require.NoError(t, err)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// New dictionary should still be accessible

		doubleCheckDictionary(
			inter,
			resetStorage,
			transferred,
			expectedValue,
			newOwner,
		)

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue, reloadDictionary := createDictionary(&r, inter)

		// Check dictionary and reset storage
		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// Reload the dictionary after the reset

		dictionary = reloadDictionary()

		// Insert new values into the dictionary.
		// Atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab errors.

		numberOfValues := r.randomInt(r.containerMaxSize)

		for i := 0; i < numberOfValues; i++ {

			// Generate a unique key
			var key cadence.Value
			var importedKey interpreter.Value
			for {
				key = r.randomHashableValue(inter)
				importedKey = importValue(t, inter, key)

				if !dictionary.ContainsKey(
					inter,
					interpreter.EmptyLocationRange,
					importedKey,
				) {
					break
				}
			}

			value := r.randomStorableValue(inter, 0)
			importedValue := importValue(t, inter, value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			_ = withoutAtreeStorageValidationEnabled(inter, func() struct{} {

				existing := dictionary.Insert(
					inter,
					interpreter.EmptyLocationRange,
					importedKey,
					importedValue,
				)
				require.Equal(t,
					interpreter.NilOptionalValue,
					existing,
				)
				return struct{}{}
			})

			expectedValue.Pairs = append(
				expectedValue.Pairs,
				cadence.KeyValuePair{
					Key:   key,
					Value: value,
				},
			)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue, reloadDictionary := createDictionary(&r, inter)

		// Check dictionary and reset storage
		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// Reload the dictionary after the reset

		dictionary = reloadDictionary()

		// Remove
		for _, pair := range expectedValue.Pairs {

			key := importValue(t, inter, pair.Key)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			removedValue := withoutAtreeStorageValidationEnabled(inter, func() interpreter.OptionalValue {
				return dictionary.Remove(inter, interpreter.EmptyLocationRange, key)
			})

			require.IsType(t, &interpreter.SomeValue{}, removedValue)
			someValue := removedValue.(*interpreter.SomeValue)

			// TODO: panic: duplicate slab 0x4100000000000000.14 for seed 1736809620917220000
			value := importValue(t, inter, pair.Value)

			// Removed value must be same as the original value
			innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
			utils.AssertValuesEqual(t, inter, value, innerValue)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		expectedValue = cadence.Dictionary{}.
			WithType(expectedValue.Type().(*cadence.DictionaryType))

		// Dictionary must be empty
		require.Equal(t, 0, dictionary.Count())

		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// TODO: check storage size, slab count
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue, reloadDictionary := createDictionary(&r, inter)

		// Check dictionary and reset storage
		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// Reload the dictionary after the reset

		dictionary = reloadDictionary()

		elementCount := dictionary.Count()

		// Generate new values

		newValues := make([]cadence.Value, len(expectedValue.Pairs))
		for i := range expectedValue.Pairs {
			newValues[i] = r.randomStorableValue(inter, 0)
		}

		// Update
		for i, pair := range expectedValue.Pairs {

			key := importValue(t, inter, pair.Key)
			newValue := importValue(t, inter, newValues[i])

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			existingValue := withoutAtreeStorageValidationEnabled(inter, func() interpreter.OptionalValue {
				return dictionary.Insert(
					inter,
					interpreter.EmptyLocationRange,
					key,
					newValue,
				)
			})

			require.IsType(t, &interpreter.SomeValue{}, existingValue)
			someValue := existingValue.(*interpreter.SomeValue)

			value := importValue(t, inter, pair.Value)

			// Removed value must be same as the original value
			innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
			utils.AssertValuesEqual(t, inter, value, innerValue)

			expectedValue.Pairs[i].Value = newValues[i]
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// Dictionary must have same number of key-value pairs
		require.Equal(t, elementCount, dictionary.Count())

		doubleCheckDictionary(
			inter,
			resetStorage,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// TODO: check storage size, slab count
	})
}

func TestInterpretRandomCompositeOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	t.Parallel()

	orgOwner := common.Address{'A'}

	const compositeStorageMapKey = interpreter.StringStorageMapKey("composite")

	createComposite := func(
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.CompositeValue,
		cadence.Struct,
		func() *interpreter.CompositeValue,
	) {
		expectedValue := r.randomStructValue(inter, 0)

		fieldsMappedByName := expectedValue.FieldsMappedByName()
		fields := make([]interpreter.CompositeField, 0, len(fieldsMappedByName))
		for name, field := range fieldsMappedByName {

			value := importValue(t, inter, field)

			fields = append(fields, interpreter.CompositeField{
				Name:  name,
				Value: value,
			})
		}

		// Construct a composite directly in the owner's account.
		// However, the composite is not referenced by the root of the storage yet
		// (a storage map), so atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab" errors.

		composite := withoutAtreeStorageValidationEnabled(
			inter,
			func() *interpreter.CompositeValue {
				return interpreter.NewCompositeValue(
					inter,
					interpreter.EmptyLocationRange,
					expectedValue.StructType.Location,
					expectedValue.StructType.QualifiedIdentifier,
					common.CompositeKindStructure,
					fields,
					orgOwner,
				)
			},
		)

		// Store the composite in a storage map, so that the composite's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				compositeStorageMapKey,
				composite,
			)

		reloadComposite := func() *interpreter.CompositeValue {
			storageMap := inter.Storage().GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, compositeStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.CompositeValue{}, readValue)
			return readValue.(*interpreter.CompositeValue)
		}

		return composite, expectedValue, reloadComposite
	}

	checkComposite := func(
		inter *interpreter.Interpreter,
		composite *interpreter.CompositeValue,
		expectedValue cadence.Struct,
		expectedOwner common.Address,
	) {
		fieldsMappedByName := expectedValue.FieldsMappedByName()

		require.Equal(t, len(fieldsMappedByName), composite.FieldCount())

		for name, field := range fieldsMappedByName {

			value := composite.GetMember(inter, interpreter.EmptyLocationRange, name)

			fieldValue := importValue(t, inter, field)
			utils.AssertValuesEqual(t, inter, fieldValue, value)
		}

		owner := composite.GetOwner()
		assert.Equal(t, expectedOwner, owner)
	}

	doubleCheckComposite := func(
		inter *interpreter.Interpreter,
		resetStorage func(),
		composite *interpreter.CompositeValue,
		expectedValue cadence.Struct,
		expectedOwner common.Address,
	) {
		// Check the values of the composite.
		// Once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkComposite(
				inter,
				composite,
				expectedValue,
				expectedOwner,
			)

			resetStorage()
		}
	}

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		composite, expectedValue, _ := createComposite(&r, inter)

		doubleCheckComposite(
			inter,
			resetStorage,
			composite,
			expectedValue,
			orgOwner,
		)
	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue, _ := createComposite(&r, inter)

		resetStorage()

		// Transfer the composite to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		).(*interpreter.CompositeValue)

		// Store the transferred composite in a storage map, so that the composote's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				newOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				interpreter.StringStorageMapKey("transferred_composite"),
				transferred,
			)

		// Both original and transferred composite should contain the expected values
		// Check once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkComposite(
				inter,
				original,
				expectedValue,
				orgOwner,
			)

			checkComposite(
				inter,
				transferred,
				expectedValue,
				newOwner,
			)

			resetStorage()
		}

		// Deep remove the original composite

		// TODO: is has no parent container = true correct?
		original.DeepRemove(inter, true)

		if !original.Inlined() {
			err := inter.Storage().Remove(original.SlabID())
			require.NoError(t, err)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// New composite should still be accessible

		doubleCheckComposite(
			inter,
			resetStorage,
			transferred,
			expectedValue,
			newOwner,
		)

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		composite, expectedValue, reloadComposite := createComposite(&r, inter)

		// Check composite and reset storage
		doubleCheckComposite(
			inter,
			resetStorage,
			composite,
			expectedValue,
			orgOwner,
		)

		// Reload the composite after the reset

		composite = reloadComposite()

		typeID := expectedValue.StructType.Location.
			TypeID(nil, expectedValue.StructType.QualifiedIdentifier)
		compositeType := inter.Program.Elaboration.CompositeType(typeID)

		typeFieldCount := len(compositeType.Fields)
		require.Equal(t, typeFieldCount, len(expectedValue.FieldsMappedByName()))
		require.Equal(t, typeFieldCount, composite.FieldCount())

		// Generate new values

		newValues := make([]cadence.Value, typeFieldCount)

		for i := range compositeType.Fields {
			newValues[i] = r.randomStorableValue(inter, 0)
		}

		// Update
		for i, name := range compositeType.Fields {

			newValue := importValue(t, inter, newValues[i])

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			existed := withoutAtreeStorageValidationEnabled(inter, func() bool {
				return composite.SetMember(
					inter,
					interpreter.EmptyLocationRange,
					name,
					newValue,
				)
			})

			require.True(t, existed)
		}

		expectedValue = cadence.NewStruct(newValues).
			WithType(expectedValue.Type().(*cadence.StructType))

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// Composite must have same number of key-value pairs
		require.Equal(t, typeFieldCount, composite.FieldCount())

		doubleCheckComposite(
			inter,
			resetStorage,
			composite,
			expectedValue,
			orgOwner,
		)

		// TODO: check storage size, slab count
	})
}

func TestInterpretRandomArrayOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	t.Parallel()

	orgOwner := common.Address{'A'}

	const arrayStorageMapKey = interpreter.StringStorageMapKey("array")

	createArray := func(
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.ArrayValue,
		cadence.Array,
		func() *interpreter.ArrayValue,
	) {
		expectedValue := r.randomArrayValue(inter, 0)

		elements := make([]interpreter.Value, len(expectedValue.Values))
		for i, value := range expectedValue.Values {
			elements[i] = importValue(t, inter, value)
		}

		// Construct an array directly in the owner's account.
		// However, the array is not referenced by the root of the storage yet
		// (a storage map), so atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab" errors.

		array := withoutAtreeStorageValidationEnabled(
			inter,
			func() *interpreter.ArrayValue {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					orgOwner,
					elements...,
				)
			},
		)

		// Store the array in a storage map, so that the array's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				arrayStorageMapKey,
				array,
			)

		reloadArray := func() *interpreter.ArrayValue {
			storageMap := inter.Storage().GetStorageMap(
				orgOwner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, arrayStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.ArrayValue{}, readValue)
			return readValue.(*interpreter.ArrayValue)
		}

		return array, expectedValue, reloadArray
	}

	checkArray := func(
		inter *interpreter.Interpreter,
		array *interpreter.ArrayValue,
		expectedValue cadence.Array,
		expectedOwner common.Address,
	) {
		require.Equal(t, len(expectedValue.Values), array.Count())

		for i, value := range expectedValue.Values {
			value := importValue(t, inter, value)

			element := array.Get(inter, interpreter.EmptyLocationRange, i)

			utils.AssertValuesEqual(t, inter, value, element)
		}

		owner := array.GetOwner()
		assert.Equal(t, expectedOwner, owner)
	}

	doubleCheckArray := func(
		inter *interpreter.Interpreter,
		resetStorage func(),
		array *interpreter.ArrayValue,
		expectedValue cadence.Array,
		expectedOwner common.Address,
	) {
		// Check the values of the array.
		// Once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkArray(
				inter,
				array,
				expectedValue,
				expectedOwner,
			)

			resetStorage()
		}
	}

	checkIteration := func(
		inter *interpreter.Interpreter,
		resetStorage func(),
		array *interpreter.ArrayValue,
		expectedValue cadence.Array,
	) {

		// Iterate over the values of the created array.
		// Once right after construction, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			require.Equal(t, len(expectedValue.Values), array.Count())

			var iterations int

			array.Iterate(
				inter,
				func(element interpreter.Value) (resume bool) {
					value := importValue(t, inter, expectedValue.Values[iterations])

					utils.AssertValuesEqual(t, inter, value, element)

					iterations += 1

					return true
				},
				false,
				interpreter.EmptyLocationRange,
			)

			assert.Equal(t, len(expectedValue.Values), iterations)

			resetStorage()
		}
	}

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue, _ := createArray(&r, inter)

		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)
	})

	t.Run("iterate", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue, _ := createArray(&r, inter)

		checkIteration(
			inter,
			resetStorage,
			array,
			expectedValue,
		)
	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue, _ := createArray(&r, inter)

		resetStorage()

		// Transfer the array to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		).(*interpreter.ArrayValue)

		// Store the transferred array in a storage map, so that the array's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				newOwner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				interpreter.StringStorageMapKey("transferred_array"),
				transferred,
			)

		// Both original and transferred array should contain the expected values
		// Check once right away, and once after a reset (commit and reload) of the storage.

		for i := 0; i < 2; i++ {

			checkArray(
				inter,
				original,
				expectedValue,
				orgOwner,
			)

			checkArray(
				inter,
				transferred,
				expectedValue,
				newOwner,
			)

			resetStorage()
		}

		// Deep remove the original array

		// TODO: is has no parent container = true correct?
		original.DeepRemove(inter, true)

		if !original.Inlined() {
			err := inter.Storage().Remove(original.SlabID())
			require.NoError(t, err)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// New array should still be accessible

		doubleCheckArray(
			inter,
			resetStorage,
			transferred,
			expectedValue,
			newOwner,
		)

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue, reloadArray := createArray(&r, inter)

		// Check array and reset storage
		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)

		// Reload the array after the reset

		array = reloadArray()

		existingValueCount := len(expectedValue.Values)

		// Insert new values into the array.

		newValueCount := r.randomInt(r.containerMaxSize)

		for i := 0; i < newValueCount; i++ {

			value := r.randomStorableValue(inter, 0)
			importedValue := importValue(t, inter, value)

			// Generate a random index
			index := 0
			if existingValueCount > 0 {
				index = r.rand.Intn(existingValueCount)
			}

			expectedValue.Values = append(expectedValue.Values, nil)
			copy(expectedValue.Values[index+1:], expectedValue.Values[index:])
			expectedValue.Values[index] = value

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			_ = withoutAtreeStorageValidationEnabled(inter, func() struct{} {

				array.Insert(
					inter,
					interpreter.EmptyLocationRange,
					index,
					importedValue,
				)

				return struct{}{}
			})
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue, reloadArray := createArray(&r, inter)

		// Check array and reset storage
		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)

		// Reload the array after the reset

		array = reloadArray()

		// Random remove
		numberOfValues := len(expectedValue.Values)
		for i := 0; i < numberOfValues; i++ {

			index := r.rand.Intn(len(expectedValue.Values))

			value := importValue(t, inter, expectedValue.Values[index])

			expectedValue.Values = append(
				expectedValue.Values[:index],
				expectedValue.Values[index+1:]...,
			)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			removedValue := withoutAtreeStorageValidationEnabled(inter, func() interpreter.Value {
				return array.Remove(inter, interpreter.EmptyLocationRange, index)
			})

			// Removed value must be same as the original value
			utils.AssertValuesEqual(t, inter, value, removedValue)
		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// Array must be empty
		require.Equal(t, 0, array.Count())

		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)

		// TODO: check storage size, slab count
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue, reloadArray := createArray(&r, inter)

		// Check array and reset storage
		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)

		// Reload the array after the reset

		array = reloadArray()

		elementCount := array.Count()

		// Random update
		for i := 0; i < len(expectedValue.Values); i++ {

			index := r.rand.Intn(len(expectedValue.Values))

			expectedValue.Values[index] = r.randomStorableValue(inter, 0)
			newValue := importValue(t, inter, expectedValue.Values[index])

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				array.Set(
					inter,
					interpreter.EmptyLocationRange,
					index,
					newValue,
				)
				return struct{}{}
			})

		}

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// Array must have same number of key-value pairs
		require.Equal(t, elementCount, array.Count())

		doubleCheckArray(
			inter,
			resetStorage,
			array,
			expectedValue,
			orgOwner,
		)

		// TODO: check storage size, slab count
	})
}


type randomValueLimits struct {
	containerMaxDepth  int
	containerMaxSize   int
	compositeMaxFields int
}

type randomValueGenerator struct {
	seed int64
	rand *rand.Rand
	randomValueLimits
}

func newRandomValueGenerator(seed int64, limits randomValueLimits) randomValueGenerator {
	if seed == -1 {
		seed = time.Now().UnixNano()
	}

	return randomValueGenerator{
		seed:              seed,
		rand:              rand.New(rand.NewSource(seed)),
		randomValueLimits: limits,
	}
}
func (r randomValueGenerator) randomStorableValue(inter *interpreter.Interpreter, currentDepth int) cadence.Value {
	n := 0
	if currentDepth < r.containerMaxDepth {
		n = r.randomInt(randomValueKindStruct)
	} else {
		n = r.randomInt(randomValueKindCapability)
	}

	switch n {

	// Non-hashable
	case randomValueKindVoid:
		return cadence.Void{}

	case randomValueKindNil:
		return cadence.NewOptional(nil)

	case randomValueKindDictionaryVariant1,
		randomValueKindDictionaryVariant2:
		return r.randomDictionaryValue(inter, currentDepth)

	case randomValueKindArrayVariant1,
		randomValueKindArrayVariant2:
		return r.randomArrayValue(inter, currentDepth)

	case randomValueKindStruct:
		return r.randomStructValue(inter, currentDepth)

	case randomValueKindCapability:
		return r.randomCapabilityValue()

	case randomValueKindSome:
		return cadence.NewOptional(
			r.randomStorableValue(inter, currentDepth+1),
		)

	// Hashable
	default:
		return r.generateHashableValueOfType(inter, n)
	}
}

func (r randomValueGenerator) randomHashableValue(inter *interpreter.Interpreter) cadence.Value {
	return r.generateHashableValueOfType(inter, r.randomInt(randomValueKindEnum))
}

func (r randomValueGenerator) generateHashableValueOfType(inter *interpreter.Interpreter, n int) cadence.Value {
	switch n {

	// Int*
	case randomValueKindInt:
		// TODO: generate larger numbers
		return cadence.NewInt(r.randomSign() * int(r.rand.Int63()))
	case randomValueKindInt8:
		return cadence.NewInt8(int8(r.randomInt(math.MaxUint8)))
	case randomValueKindInt16:
		return cadence.NewInt16(int16(r.randomInt(math.MaxUint16)))
	case randomValueKindInt32:
		return cadence.NewInt32(int32(r.randomSign()) * r.rand.Int31())
	case randomValueKindInt64:
		return cadence.NewInt64(int64(r.randomSign()) * r.rand.Int63())
	case randomValueKindInt128:
		// TODO: generate larger numbers
		return cadence.NewInt128(r.randomSign() * int(r.rand.Int63()))
	case randomValueKindInt256:
		// TODO: generate larger numbers
		return cadence.NewInt256(r.randomSign() * int(r.rand.Int63()))

	// UInt*
	case randomValueKindUInt:
		// TODO: generate larger numbers
		return cadence.NewUInt(uint(r.rand.Uint64()))
	case randomValueKindUInt8:
		return cadence.NewUInt8(uint8(r.randomInt(math.MaxUint8)))
	case randomValueKindUInt16:
		return cadence.NewUInt16(uint16(r.randomInt(math.MaxUint16)))
	case randomValueKindUInt32:
		return cadence.NewUInt32(r.rand.Uint32())
	case randomValueKindUInt64Variant1,
		randomValueKindUInt64Variant2,
		randomValueKindUInt64Variant3,
		randomValueKindUInt64Variant4: // should be more common
		return cadence.NewUInt64(r.rand.Uint64())
	case randomValueKindUInt128:
		// TODO: generate larger numbers
		return cadence.NewUInt128(uint(r.rand.Uint64()))
	case randomValueKindUInt256:
		// TODO: generate larger numbers
		return cadence.NewUInt256(uint(r.rand.Uint64()))

	// Word*
	case randomValueKindWord8:
		return cadence.NewWord8(uint8(r.randomInt(math.MaxUint8)))
	case randomValueKindWord16:
		return cadence.NewWord16(uint16(r.randomInt(math.MaxUint16)))
	case randomValueKindWord32:
		return cadence.NewWord32(r.rand.Uint32())
	case randomValueKindWord64:
		return cadence.NewWord64(r.rand.Uint64())
	case randomValueKindWord128:
		// TODO: generate larger numbers
		return cadence.NewWord128(uint(r.rand.Uint64()))
	case randomValueKindWord256:
		// TODO: generate larger numbers
		return cadence.NewWord256(uint(r.rand.Uint64()))

	// (U)Fix*
	case randomValueKindFix64:
		return cadence.Fix64(
			int64(r.randomSign()) * r.rand.Int63n(sema.Fix64TypeMaxInt),
		)
	case randomValueKindUFix64:
		return cadence.UFix64(
			uint64(r.rand.Int63n(int64(sema.UFix64TypeMaxInt))),
		)

	// String
	case randomValueKindStringVariant1,
		randomValueKindStringVariant2,
		randomValueKindStringVariant3,
		randomValueKindStringVariant4: // small string - should be more common
		size := r.randomInt(255)
		return cadence.String(r.randomUTF8StringOfSize(size))
	case randomValueKindStringVariant5: // large string
		size := r.randomInt(4048) + 255
		return cadence.String(r.randomUTF8StringOfSize(size))

	case randomValueKindBoolVariantTrue:
		return cadence.NewBool(true)
	case randomValueKindBoolVariantFalse:
		return cadence.NewBool(false)

	case randomValueKindAddress:
		return r.randomAddressValue()

	case randomValueKindPath:
		return r.randomPathValue()

	case randomValueKindEnum:
		return r.randomEnumValue(inter)

	default:
		panic(fmt.Sprintf("unsupported: %d", n))
	}
}

func (r randomValueGenerator) randomSign() int {
	if r.randomInt(1) == 1 {
		return 1
	}

	return -1
}

func (r randomValueGenerator) randomAddressValue() (address cadence.Address) {
	r.rand.Read(address[:])
	return address
}

func (r randomValueGenerator) randomPathValue() cadence.Path {
	randomDomain := r.rand.Intn(len(common.AllPathDomains))
	identifier := r.randomUTF8String()

	return cadence.Path{
		Domain:     common.AllPathDomains[randomDomain],
		Identifier: identifier,
	}
}

func (r randomValueGenerator) randomCapabilityValue() cadence.Capability {
	return cadence.NewCapability(
		cadence.UInt64(r.randomInt(math.MaxInt-1)),
		r.randomAddressValue(),
		cadence.NewReferenceType(
			cadence.UnauthorizedAccess,
			cadence.AnyStructType,
		),
	)
}

func (r randomValueGenerator) randomDictionaryValue(inter *interpreter.Interpreter, currentDepth int) cadence.Dictionary {

	entryCount := r.randomInt(r.containerMaxSize)
	keyValues := make([]cadence.KeyValuePair, entryCount)

	existingKeys := map[string]struct{}{}

	for i := 0; i < entryCount; i++ {

		// generate a unique key
		var key cadence.Value
		for {
			key = r.randomHashableValue(inter)
			keyStr := key.String()

			// avoid duplicate keys
			_, exists := existingKeys[keyStr]
			if !exists {
				existingKeys[keyStr] = struct{}{}
				break
			}
		}

		keyValues[i] = cadence.KeyValuePair{
			Key:   key,
			Value: r.randomStorableValue(inter, currentDepth+1),
		}
	}

	return cadence.NewDictionary(keyValues).
		WithType(
			cadence.NewDictionaryType(
				cadence.HashableStructType,
				cadence.AnyStructType,
			),
		)
}

func (r randomValueGenerator) randomInt(upperBound int) int {
	return r.rand.Intn(upperBound + 1)
}

func (r randomValueGenerator) randomArrayValue(inter *interpreter.Interpreter, currentDepth int) cadence.Array {
	elementsCount := r.randomInt(r.containerMaxSize)
	elements := make([]cadence.Value, elementsCount)

	for i := 0; i < elementsCount; i++ {
		elements[i] = r.randomStorableValue(inter, currentDepth+1)
	}

	return cadence.NewArray(elements).
		WithType(cadence.NewVariableSizedArrayType(cadence.AnyStructType))
}

func (r randomValueGenerator) randomStructValue(inter *interpreter.Interpreter, currentDepth int) cadence.Struct {
	fieldsCount := r.randomInt(r.compositeMaxFields)

	fields := make([]cadence.Field, fieldsCount)
	fieldValues := make([]cadence.Value, fieldsCount)

	existingFieldNames := make(map[string]any, fieldsCount)

	for i := 0; i < fieldsCount; i++ {
		// generate a unique field name
		var fieldName string
		for {
			fieldName = r.randomUTF8String()

			// avoid duplicate field names
			_, exists := existingFieldNames[fieldName]
			if !exists {
				existingFieldNames[fieldName] = struct{}{}
				break
			}
		}

		fields[i] = cadence.NewField(fieldName, cadence.AnyStructType)
		fieldValues[i] = r.randomStorableValue(inter, currentDepth+1)
	}

	identifier := fmt.Sprintf("S%d", r.rand.Uint64())

	address := r.randomAddressValue()

	location := common.AddressLocation{
		Address: common.Address(address),
		Name:    identifier,
	}

	kind := common.CompositeKindStructure

	compositeType := &sema.CompositeType{
		Location:   location,
		Identifier: identifier,
		Kind:       kind,
		Members:    &sema.StringMemberOrderedMap{},
	}

	fieldNames := make([]string, fieldsCount)

	for i := 0; i < fieldsCount; i++ {
		fieldName := fields[i].Identifier
		compositeType.Members.Set(
			fieldName,
			sema.NewUnmeteredPublicConstantFieldMember(
				compositeType,
				fieldName,
				sema.AnyStructType,
				"",
			),
		)
		fieldNames[i] = fieldName
	}
	compositeType.Fields = fieldNames

	// Add the type to the elaboration, to short-circuit the type-lookup.
	inter.Program.Elaboration.SetCompositeType(
		compositeType.ID(),
		compositeType,
	)

	return cadence.NewStruct(fieldValues).WithType(
		cadence.NewStructType(
			location,
			identifier,
			fields,
			nil,
		),
	)
}

func (r randomValueGenerator) cadenceIntegerType(n int) cadence.Type {
	switch n {
	// Int
	case randomValueKindInt:
		return cadence.IntType
	case randomValueKindInt8:
		return cadence.Int8Type
	case randomValueKindInt16:
		return cadence.Int16Type
	case randomValueKindInt32:
		return cadence.Int32Type
	case randomValueKindInt64:
		return cadence.Int64Type
	case randomValueKindInt128:
		return cadence.Int128Type
	case randomValueKindInt256:
		return cadence.Int256Type

	// UInt
	case randomValueKindUInt:
		return cadence.UIntType
	case randomValueKindUInt8:
		return cadence.UInt8Type
	case randomValueKindUInt16:
		return cadence.UInt16Type
	case randomValueKindUInt32:
		return cadence.UInt32Type
	case randomValueKindUInt64Variant1,
		randomValueKindUInt64Variant2,
		randomValueKindUInt64Variant3,
		randomValueKindUInt64Variant4:
		return cadence.UInt64Type
	case randomValueKindUInt128:
		return cadence.UInt128Type
	case randomValueKindUInt256:
		return cadence.UInt256Type

	// Word
	case randomValueKindWord8:
		return cadence.Word8Type
	case randomValueKindWord16:
		return cadence.Word16Type
	case randomValueKindWord32:
		return cadence.Word32Type
	case randomValueKindWord64:
		return cadence.Word64Type
	case randomValueKindWord128:
		return cadence.Word128Type
	case randomValueKindWord256:
		return cadence.Word256Type

	default:
		panic(fmt.Sprintf("unsupported: %d", n))
	}
}

func (r randomValueGenerator) semaIntegerType(n int) sema.Type {
	switch n {
	// Int
	case randomValueKindInt:
		return sema.IntType
	case randomValueKindInt8:
		return sema.Int8Type
	case randomValueKindInt16:
		return sema.Int16Type
	case randomValueKindInt32:
		return sema.Int32Type
	case randomValueKindInt64:
		return sema.Int64Type
	case randomValueKindInt128:
		return sema.Int128Type
	case randomValueKindInt256:
		return sema.Int256Type

	// UInt
	case randomValueKindUInt:
		return sema.UIntType
	case randomValueKindUInt8:
		return sema.UInt8Type
	case randomValueKindUInt16:
		return sema.UInt16Type
	case randomValueKindUInt32:
		return sema.UInt32Type
	case randomValueKindUInt64Variant1,
		randomValueKindUInt64Variant2,
		randomValueKindUInt64Variant3,
		randomValueKindUInt64Variant4:
		return sema.UInt64Type
	case randomValueKindUInt128:
		return sema.UInt128Type
	case randomValueKindUInt256:
		return sema.UInt256Type

	// Word
	case randomValueKindWord8:
		return sema.Word8Type
	case randomValueKindWord16:
		return sema.Word16Type
	case randomValueKindWord32:
		return sema.Word32Type
	case randomValueKindWord64:
		return sema.Word64Type
	case randomValueKindWord128:
		return sema.Word128Type
	case randomValueKindWord256:
		return sema.Word256Type

	default:
		panic(fmt.Sprintf("unsupported: %d", n))
	}
}

const (
	// Hashable values
	// Int*
	randomValueKindInt = iota
	randomValueKindInt8
	randomValueKindInt16
	randomValueKindInt32
	randomValueKindInt64
	randomValueKindInt128
	randomValueKindInt256

	// UInt*
	randomValueKindUInt
	randomValueKindUInt8
	randomValueKindUInt16
	randomValueKindUInt32
	randomValueKindUInt64Variant1
	randomValueKindUInt64Variant2
	randomValueKindUInt64Variant3
	randomValueKindUInt64Variant4
	randomValueKindUInt128
	randomValueKindUInt256

	// Word*
	randomValueKindWord8
	randomValueKindWord16
	randomValueKindWord32
	randomValueKindWord64
	randomValueKindWord128
	randomValueKindWord256

	// (U)Fix*
	randomValueKindFix64
	randomValueKindUFix64

	// String
	randomValueKindStringVariant1
	randomValueKindStringVariant2
	randomValueKindStringVariant3
	randomValueKindStringVariant4
	randomValueKindStringVariant5

	randomValueKindBoolVariantTrue
	randomValueKindBoolVariantFalse
	randomValueKindPath
	randomValueKindAddress
	randomValueKindEnum

	// Non-hashable values
	randomValueKindVoid
	randomValueKindNil // `Never?`
	randomValueKindCapability

	// Containers
	randomValueKindSome
	randomValueKindArrayVariant1
	randomValueKindArrayVariant2
	randomValueKindDictionaryVariant1
	randomValueKindDictionaryVariant2
	randomValueKindStruct
)

func (r randomValueGenerator) randomUTF8String() string {
	return r.randomUTF8StringOfSize(8)
}

func (r randomValueGenerator) randomUTF8StringOfSize(size int) string {
	identifier := make([]byte, size)
	r.rand.Read(identifier)
	return strings.ToValidUTF8(string(identifier), "$")
}

func (r randomValueGenerator) randomEnumValue(inter *interpreter.Interpreter) cadence.Enum {
	// Get a random integer subtype to be used as the raw-type of enum
	typ := r.randomInt(randomValueKindWord64)

	rawValue := r.generateHashableValueOfType(inter, typ).(cadence.NumberValue)

	identifier := fmt.Sprintf("E%d", r.rand.Uint64())

	address := r.randomAddressValue()

	location := common.AddressLocation{
		Address: common.Address(address),
		Name:    identifier,
	}

	semaRawType := r.semaIntegerType(typ)

	semaEnumType := &sema.CompositeType{
		Identifier:  identifier,
		EnumRawType: semaRawType,
		Kind:        common.CompositeKindEnum,
		Location:    location,
		Members:     &sema.StringMemberOrderedMap{},
	}

	semaEnumType.Members.Set(
		sema.EnumRawValueFieldName,
		sema.NewUnmeteredPublicConstantFieldMember(
			semaEnumType,
			sema.EnumRawValueFieldName,
			semaRawType,
			"",
		),
	)

	// Add the type to the elaboration, to short-circuit the type-lookup.
	inter.Program.Elaboration.SetCompositeType(
		semaEnumType.ID(),
		semaEnumType,
	)

	rawType := r.cadenceIntegerType(typ)

	fields := []cadence.Value{
		rawValue,
	}

	return cadence.NewEnum(fields).WithType(
		cadence.NewEnumType(
			location,
			identifier,
			rawType,
			[]cadence.Field{
				{
					Identifier: sema.EnumRawValueFieldName,
					Type:       rawType,
				},
			},
			nil,
		),
	)
}

func TestRandomValueGeneration(t *testing.T) {

	inter, _ := newRandomValueTestInterpreter(t)

	limits := defaultRandomValueLimits

	// Generate random values
	for i := 0; i < 1000; i++ {
		r1 := newRandomValueGenerator(int64(i), limits)
		v1 := r1.randomStorableValue(inter, 0)

		r2 := newRandomValueGenerator(int64(i), limits)
		v2 := r2.randomStorableValue(inter, 0)

		// Check if the generated values are equal
		assert.Equal(t, v1, v2)
	}
}

func mapKey(inter *interpreter.Interpreter, key interpreter.Value) any {
	switch key := key.(type) {
	case *interpreter.StringValue:
		return *key

	case *interpreter.CompositeValue:
		type enumKey struct {
			location            common.Location
			qualifiedIdentifier string
			kind                common.CompositeKind
			rawValue            interpreter.Value
		}
		return enumKey{
			location:            key.Location,
			qualifiedIdentifier: key.QualifiedIdentifier,
			kind:                key.Kind,
			rawValue:            key.GetField(inter, interpreter.EmptyLocationRange, sema.EnumRawValueFieldName),
		}

	case interpreter.Value:
		return key

	default:
		panic(errors.NewUnexpectedError("unsupported map key type: %T", key))
	}
}

// This test is a reproducer for "slab was not reachable from leaves" false alarm.
// https://github.com/onflow/cadence/pull/2882#issuecomment-1781298107
// In this test, storage.CheckHealth() should be called after array.DeepRemove(),
// not in the middle of array.DeepRemove().
// CheckHealth() is called in the middle of array.DeepRemove() when:
// - array.DeepRemove() calls childArray1 and childArray2 DeepRemove()
// - DeepRemove() calls maybeValidateAtreeValue()
// - maybeValidateAtreeValue() calls CheckHealth()
func TestCheckStorageHealthInMiddleOfDeepRemove(t *testing.T) {

	storage := newUnmeteredInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Program:     ast.NewProgram(nil, []ast.Declaration{}),
			Elaboration: sema.NewElaboration(nil),
		},
		utils.TestLocation,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				return interpreter.VirtualImport{
					Elaboration: inter.Program.Elaboration,
				}
			},
			AtreeStorageValidationEnabled: true,
			AtreeValueValidationEnabled:   true,
		},
	)
	require.NoError(t, err)

	owner := common.Address{'A'}

	// Create a small child array which will be inlined in parent container.
	childArray1 := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		owner,
		interpreter.NewUnmeteredStringValue("a"),
	)

	size := int(atree.MaxInlineArrayElementSize()) - 10

	// Create a large child array which will NOT be inlined in parent container.
	childArray2 := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		owner,
		interpreter.NewUnmeteredStringValue(strings.Repeat("b", size)),
		interpreter.NewUnmeteredStringValue(strings.Repeat("c", size)),
	)

	// Create an array with childArray1 and childArray2.
	array := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		owner,
		childArray1, // inlined
		childArray2, // not inlined
	)

	// DeepRemove removes all elements (childArray1 and childArray2) recursively in array.
	array.DeepRemove(inter, true)

	// As noted earlier in comments at the top of this test:
	// storage.CheckHealth() should be called after array.DeepRemove(), not in the middle of array.DeepRemove().
	// This happens when:
	// - array.DeepRemove() calls childArray1 and childArray2 DeepRemove()
	// - DeepRemove() calls maybeValidateAtreeValue()
	// - maybeValidateAtreeValue() calls CheckHealth()
}

// This test is a reproducer for "slab was not reachable from leaves" false alarm.
// https://github.com/onflow/cadence/pull/2882#issuecomment-1796381227
// In this test, storage.CheckHealth() should be called after DictionaryValue.Transfer()
// with remove flag, not in the middle of DictionaryValue.Transfer().
func TestCheckStorageHealthInMiddleOfTransferAndRemove(t *testing.T) {

	t.Parallel()

	r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
	t.Logf("seed: %d", r.seed)

	storage := newUnmeteredInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Program:     ast.NewProgram(nil, []ast.Declaration{}),
			Elaboration: sema.NewElaboration(nil),
		},
		utils.TestLocation,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				return interpreter.VirtualImport{
					Elaboration: inter.Program.Elaboration,
				}
			},
			AtreeStorageValidationEnabled: true,
			AtreeValueValidationEnabled:   true,
		},
	)
	require.NoError(t, err)

	// Create large array value with zero address which will not be inlined.
	gchildArray := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		interpreter.NewUnmeteredStringValue(strings.Repeat("b", int(atree.MaxInlineArrayElementSize())-10)),
		interpreter.NewUnmeteredStringValue(strings.Repeat("c", int(atree.MaxInlineArrayElementSize())-10)),
	)

	// Create small composite value with zero address which will be inlined.
	identifier := "test"

	location := common.AddressLocation{
		Address: common.ZeroAddress,
		Name:    identifier,
	}

	compositeType := &sema.CompositeType{
		Location:   location,
		Identifier: identifier,
		Kind:       common.CompositeKindStructure,
	}

	fields := []interpreter.CompositeField{
		interpreter.NewUnmeteredCompositeField("a", interpreter.NewUnmeteredUInt64Value(0)),
		interpreter.NewUnmeteredCompositeField("b", interpreter.NewUnmeteredUInt64Value(1)),
		interpreter.NewUnmeteredCompositeField("c", interpreter.NewUnmeteredUInt64Value(2)),
	}

	compositeType.Members = &sema.StringMemberOrderedMap{}
	for _, field := range fields {
		compositeType.Members.Set(
			field.Name,
			sema.NewUnmeteredPublicConstantFieldMember(
				compositeType,
				field.Name,
				sema.AnyStructType,
				"",
			),
		)
	}

	// Add the type to the elaboration, to short-circuit the type-lookup.
	inter.Program.Elaboration.SetCompositeType(
		compositeType.ID(),
		compositeType,
	)

	gchildComposite := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		location,
		identifier,
		common.CompositeKindStructure,
		fields,
		common.ZeroAddress,
	)

	// Create large dictionary with zero address with 2 data slabs containing:
	// - SomeValue(SlabID) as first physical element in the first data slab
	// - inlined CompositeValue as last physical element in the second data slab

	numberOfValues := 10
	firstElementIndex := 7 // index of first physical element in the first data slab
	lastElementIndex := 8  // index of last physical element in the last data slab
	keyValues := make([]interpreter.Value, numberOfValues*2)
	for i := 0; i < numberOfValues; i++ {
		key := interpreter.NewUnmeteredUInt64Value(uint64(i))

		var value interpreter.Value
		switch i {
		case firstElementIndex:
			value = interpreter.NewUnmeteredSomeValueNonCopying(gchildArray)

		case lastElementIndex:
			value = gchildComposite

		default:
			// Other values are inlined random strings.
			const size = 235
			value = interpreter.NewUnmeteredStringValue(r.randomUTF8StringOfSize(size))
		}

		keyValues[i*2] = key
		keyValues[i*2+1] = value
	}

	childMap := interpreter.NewDictionaryValueWithAddress(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
			ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		keyValues...,
	)

	// Create dictionary with non-zero address containing child dictionary.
	owner := common.Address{'A'}
	m := interpreter.NewDictionaryValueWithAddress(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
			ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		owner,
		interpreter.NewUnmeteredUInt64Value(0),
		childMap,
	)

	inter.ValidateAtreeValue(m)

	require.NoError(t, storage.CheckHealth())
}
