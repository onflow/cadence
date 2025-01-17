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
	containerMaxDepth:  4,
	containerMaxSize:   40,
	compositeMaxFields: 10,
}

var runSmokeTests = flag.Bool("runSmokeTests", false, "Run smoke tests on values")
var validateAtree = flag.Bool("validateAtree", true, "Enable atree validation")
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

	writeDictionary := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
		dictionary *interpreter.DictionaryValue,
	) {
		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				storageMapKey,
				dictionary,
			)
	}

	readDictionary := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
	) *interpreter.DictionaryValue {
		storageMap := inter.Storage().GetStorageMap(
			owner,
			common.PathDomainStorage.Identifier(),
			false,
		)
		require.NotNil(t, storageMap)

		readValue := storageMap.ReadValue(inter, storageMapKey)
		require.NotNil(t, readValue)

		require.IsType(t, &interpreter.DictionaryValue{}, readValue)
		return readValue.(*interpreter.DictionaryValue)
	}

	createDictionary := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.DictionaryValue,
		cadence.Dictionary,
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

		writeDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
			dictionary,
		)

		return dictionary, expectedValue
	}

	checkDictionary := func(
		t *testing.T,
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

	checkIteration := func(
		t *testing.T,
		inter *interpreter.Interpreter,
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
	}

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}
	})

	t.Run("iterate", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		checkIteration(
			t,
			inter,
			dictionary,
			expectedValue,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		checkIteration(
			t,
			inter,
			dictionary,
			expectedValue,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}
	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		original = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		// Transfer the dictionary to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			false,
		).(*interpreter.DictionaryValue)

		// Store the transferred dictionary in a storage map, so that the dictionary's slab
		// is referenced by the root of the storage.

		const transferredStorageMapKey = interpreter.StringStorageMapKey("transferred")

		writeDictionary(
			inter,
			newOwner,
			transferredStorageMapKey,
			transferred,
		)

		withoutAtreeStorageValidationEnabled(inter, func() struct{} {
			inter.Storage().
				GetStorageMap(orgOwner, common.PathDomainStorage.Identifier(), false).
				RemoveValue(inter, dictionaryStorageMapKey)

			return struct{}{}
		})

		checkDictionary(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		transferred = readDictionary(
			inter,
			newOwner,
			transferredStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		// Insert new values into the dictionary.
		// Atree storage validation must be temporarily disabled
		// to not report any "unreferenced slab" errors.

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

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

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

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check storage size, slab count
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		dictionary, expectedValue := createDictionary(t, &r, inter)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

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

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		dictionary = readDictionary(
			inter,
			orgOwner,
			dictionaryStorageMapKey,
		)

		checkDictionary(
			t,
			inter,
			dictionary,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

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

	writeComposite := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
		composite *interpreter.CompositeValue,
	) {
		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				storageMapKey,
				composite,
			)
	}

	readComposite := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
	) *interpreter.CompositeValue {
		storageMap := inter.Storage().GetStorageMap(
			owner,
			common.PathDomainStorage.Identifier(),
			false,
		)
		require.NotNil(t, storageMap)

		readValue := storageMap.ReadValue(inter, storageMapKey)
		require.NotNil(t, readValue)

		require.IsType(t, &interpreter.CompositeValue{}, readValue)
		return readValue.(*interpreter.CompositeValue)
	}

	createComposite := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.CompositeValue,
		cadence.Struct,
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

		writeComposite(
			inter,
			orgOwner,
			compositeStorageMapKey,
			composite,
		)

		return composite, expectedValue
	}

	checkComposite := func(
		t *testing.T,
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

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		composite, expectedValue := createComposite(t, &r, inter)

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		composite = readComposite(
			inter,
			orgOwner,
			compositeStorageMapKey,
		)

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue := createComposite(t, &r, inter)

		checkComposite(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		original = readComposite(
			inter,
			orgOwner,
			compositeStorageMapKey,
		)

		checkComposite(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		// Transfer the composite to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			false,
		).(*interpreter.CompositeValue)

		// Store the transferred composite in a storage map, so that the composite's slab
		// is referenced by the root of the storage.

		const transferredStorageMapKey = interpreter.StringStorageMapKey("transferred")

		writeComposite(
			inter,
			newOwner,
			transferredStorageMapKey,
			transferred,
		)

		withoutAtreeStorageValidationEnabled(inter, func() struct{} {
			inter.Storage().
				GetStorageMap(orgOwner, common.PathDomainStorage.Identifier(), false).
				RemoveValue(inter, compositeStorageMapKey)

			return struct{}{}
		})

		checkComposite(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		transferred = readComposite(
			inter,
			newOwner,
			transferredStorageMapKey,
		)

		checkComposite(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		composite, expectedValue := createComposite(t, &r, inter)

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		composite = readComposite(
			inter,
			orgOwner,
			compositeStorageMapKey,
		)

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

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

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		composite = readComposite(
			inter,
			orgOwner,
			compositeStorageMapKey,
		)

		checkComposite(
			t,
			inter,
			composite,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

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

	writeArray := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
		array *interpreter.ArrayValue,
	) {
		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				storageMapKey,
				array,
			)
	}

	readArray := func(
		inter *interpreter.Interpreter,
		owner common.Address,
		storageMapKey interpreter.StorageMapKey,
	) *interpreter.ArrayValue {
		storageMap := inter.Storage().GetStorageMap(
			owner,
			common.PathDomainStorage.Identifier(),
			false,
		)
		require.NotNil(t, storageMap)

		readValue := storageMap.ReadValue(inter, storageMapKey)
		require.NotNil(t, readValue)

		require.IsType(t, &interpreter.ArrayValue{}, readValue)
		return readValue.(*interpreter.ArrayValue)
	}

	createArray := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
	) (
		*interpreter.ArrayValue,
		cadence.Array,
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

		writeArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
			array,
		)

		return array, expectedValue
	}

	checkArray := func(
		t *testing.T,
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

	checkIteration := func(
		t *testing.T,
		inter *interpreter.Interpreter,
		array *interpreter.ArrayValue,
		expectedValue cadence.Array,
	) {
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
	}

	t.Run("construction", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}
	})

	t.Run("iterate", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		checkIteration(
			t,
			inter,
			array,
			expectedValue,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		checkIteration(
			t,
			inter,
			array,
			expectedValue,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

	})

	t.Run("move (transfer and deep remove)", func(t *testing.T) {

		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		original, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		original = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			original,
			expectedValue,
			orgOwner,
		)

		// Transfer the array to a new owner

		newOwner := common.Address{'B'}

		transferred := original.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(newOwner),
			false,
			nil,
			nil,
			false,
		).(*interpreter.ArrayValue)

		// Store the transferred array in a storage map, so that the array's slab
		// is referenced by the root of the storage.

		const transferredStorageMapKey = interpreter.StringStorageMapKey("transferred")

		writeArray(
			inter,
			newOwner,
			transferredStorageMapKey,
			transferred,
		)

		withoutAtreeStorageValidationEnabled(inter, func() struct{} {
			inter.Storage().
				GetStorageMap(orgOwner, common.PathDomainStorage.Identifier(), false).
				RemoveValue(inter, arrayStorageMapKey)

			return struct{}{}
		})

		checkArray(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		transferred = readArray(
			inter,
			newOwner,
			transferredStorageMapKey,
		)

		checkArray(
			t,
			inter,
			transferred,
			expectedValue,
			newOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check deep removal cleaned up everything in original account (storage size, slab count)
	})

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

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

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

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

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check storage size, slab count
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		r := newRandomValueGenerator(*smokeTestSeed, defaultRandomValueLimits)
		t.Logf("seed: %d", r.seed)

		inter, resetStorage := newRandomValueTestInterpreter(t)

		array, expectedValue := createArray(t, &r, inter)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

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

		// Array must have same number of elements
		require.Equal(t, elementCount, array.Count())

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		resetStorage()

		array = readArray(
			inter,
			orgOwner,
			arrayStorageMapKey,
		)

		checkArray(
			t,
			inter,
			array,
			expectedValue,
			orgOwner,
		)

		if *validateAtree {
			err := inter.Storage().CheckHealth()
			require.NoError(t, err)
		}

		// TODO: check storage size, slab count
	})
}

func TestInterpretRandomNestedArrayOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	owner := common.Address{'A'}

	limits := randomValueLimits{
		containerMaxDepth:  6,
		containerMaxSize:   20,
		compositeMaxFields: 10,
	}

	const opCount = 5

	createValue := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
		predicate func(cadence.Array) bool,
	) (
		actualRootValue interpreter.Value,
		generatedValue cadence.Value,
		reloadActualRootValue func() interpreter.Value,
		getNestedArray func(rootValue interpreter.Value, owner common.Address) *interpreter.ArrayValue,
	) {

		// It does not matter what the root value is,
		// as long as it contains a nested array,
		// which it is nested inside an optional,
		// and it satisfies the given predicate.

		var path []pathElement
		for {
			generatedValue = r.randomArrayValue(inter, 0)

			path = findNestedValue(
				generatedValue,
				func(value cadence.Value, path []pathElement) bool {
					array, ok := value.(cadence.Array)
					if !ok {
						return false
					}

					if !predicate(array) {
						return false
					}

					var foundSome bool
					for _, element := range path {
						if _, ok := element.(somePathElement); ok {
							foundSome = true
							break
						}
					}
					return foundSome
				},
			)
			if path != nil {
				break
			}
		}

		actualRootValue = importValue(t, inter, generatedValue).Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(owner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		)

		const arrayStorageMapKey = interpreter.StringStorageMapKey("array")

		// Store the array in a storage map, so that the array's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				arrayStorageMapKey,
				actualRootValue,
			)

		reloadActualRootValue = func() interpreter.Value {
			storageMap := inter.Storage().GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, arrayStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.ArrayValue{}, readValue)
			return readValue.(*interpreter.ArrayValue)
		}

		getNestedArray = func(rootValue interpreter.Value, owner common.Address) *interpreter.ArrayValue {
			nestedValue := getNestedValue(t, inter, rootValue, path)
			require.IsType(t, &interpreter.ArrayValue{}, nestedValue)
			nestedArray := nestedValue.(*interpreter.ArrayValue)
			require.Equal(t, owner, nestedArray.GetOwner())
			return nestedArray
		}

		return
	}

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedArray :=
			createValue(
				t,
				&r,
				inter,
				// Accept any array, even empty ones,
				// given we're only inserting
				func(array cadence.Array) bool {
					return true
				},
			)

		actualNestedArray := getNestedArray(actualRootValue, owner)

		type insert struct {
			index int
			value cadence.Value
		}

		performInsert := func(array *interpreter.ArrayValue, insert insert) {

			newValue := importValue(t, inter, insert.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				array.Insert(
					inter,
					interpreter.EmptyLocationRange,
					insert.index,
					newValue,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var inserts []insert

		elementCount := actualNestedArray.Count()

		for i := 0; i < opCount; i++ {
			var index int
			elementCountAfterInserts := elementCount + i
			if elementCountAfterInserts > 0 {
				index = r.rand.Intn(elementCountAfterInserts)
			}

			inserts = append(
				inserts,
				insert{
					index: index,
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, insert := range inserts {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedArray = getNestedArray(actualRootValue, owner)

			performInsert(
				actualNestedArray,
				insert,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedArray := getNestedArray(expectedRootValue, common.ZeroAddress)

			for _, insert := range inserts[:i+1] {

				performInsert(
					expectedNestedArray,
					insert,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedArray :=
			createValue(
				t,
				&r,
				inter,
				// Generate a non-empty array,
				// so we have at least one element to update
				func(array cadence.Array) bool {
					return len(array.Values) > 0
				},
			)

		actualNestedArray := getNestedArray(actualRootValue, owner)

		elementCount := actualNestedArray.Count()
		require.Greater(t, elementCount, 0)

		type update struct {
			index int
			value cadence.Value
		}

		performUpdate := func(array *interpreter.ArrayValue, update update) {

			newValue := importValue(t, inter, update.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				array.Set(
					inter,
					interpreter.EmptyLocationRange,
					update.index,
					newValue,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}

			// Array must have same number of elements
			require.Equal(t, elementCount, array.Count())
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var updates []update

		for i := 0; i < opCount; i++ {
			updates = append(
				updates,
				update{
					index: r.rand.Intn(elementCount),
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, update := range updates {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedArray = getNestedArray(actualRootValue, owner)

			performUpdate(
				actualNestedArray,
				update,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedArray := getNestedArray(expectedRootValue, common.ZeroAddress)

			for _, update := range updates[:i+1] {

				performUpdate(
					expectedNestedArray,
					update,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedArray :=
			createValue(
				t,
				&r,
				inter,
				func(array cadence.Array) bool {
					return len(array.Values) >= opCount
				},
			)

		actualNestedArray := getNestedArray(actualRootValue, owner)
		elementCount := actualNestedArray.Count()
		require.GreaterOrEqual(t, elementCount, opCount)

		performRemove := func(array *interpreter.ArrayValue, index int) {

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				array.Remove(
					inter,
					interpreter.EmptyLocationRange,
					index,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var removes []int

		for i := 0; i < opCount; i++ {
			index := r.rand.Intn(elementCount - i)
			removes = append(removes, index)
		}

		for i, index := range removes {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedArray = getNestedArray(actualRootValue, owner)

			performRemove(
				actualNestedArray,
				index,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedArray := getNestedArray(expectedRootValue, common.ZeroAddress)

			for _, index := range removes[:i+1] {

				performRemove(
					expectedNestedArray,
					index,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})
}

func TestInterpretRandomNestedDictionaryOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	owner := common.Address{'A'}

	limits := randomValueLimits{
		containerMaxDepth:  6,
		containerMaxSize:   20,
		compositeMaxFields: 10,
	}

	const opCount = 5

	createValue := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
		predicate func(cadence.Dictionary) bool,
	) (
		actualRootValue interpreter.Value,
		generatedValue cadence.Value,
		reloadActualRootValue func() interpreter.Value,
		getNestedDictionary func(rootValue interpreter.Value, owner common.Address) *interpreter.DictionaryValue,
	) {

		// It does not matter what the root value is,
		// as long as it contains a nested dictionary,
		// which it is nested inside an optional,
		// and it satisfies the given predicate.

		var path []pathElement
		for {
			generatedValue = r.randomDictionaryValue(inter, 0)

			path = findNestedValue(
				generatedValue,
				func(value cadence.Value, path []pathElement) bool {
					dictionary, ok := value.(cadence.Dictionary)
					if !ok {
						return false
					}

					if !predicate(dictionary) {
						return false
					}

					var foundSome bool
					for _, element := range path {
						if _, ok := element.(somePathElement); ok {
							foundSome = true
							break
						}
					}
					return foundSome
				},
			)
			if path != nil {
				break
			}
		}

		actualRootValue = importValue(t, inter, generatedValue).Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(owner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		)

		const dictionaryStorageMapKey = interpreter.StringStorageMapKey("dictionary")

		// Store the dictionary in a storage map, so that the dictionary's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				dictionaryStorageMapKey,
				actualRootValue,
			)

		reloadActualRootValue = func() interpreter.Value {
			storageMap := inter.Storage().GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, dictionaryStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.DictionaryValue{}, readValue)
			return readValue.(*interpreter.DictionaryValue)
		}

		getNestedDictionary = func(rootValue interpreter.Value, owner common.Address) *interpreter.DictionaryValue {
			nestedValue := getNestedValue(t, inter, rootValue, path)
			require.IsType(t, &interpreter.DictionaryValue{}, nestedValue)
			nestedDictionary := nestedValue.(*interpreter.DictionaryValue)
			require.Equal(t, owner, nestedDictionary.GetOwner())
			return nestedDictionary
		}

		return
	}

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedDictionary :=
			createValue(
				t,
				&r,
				inter,
				// Accept any dictionary, even empty ones,
				// given we're only inserting
				func(dictionary cadence.Dictionary) bool {
					return true
				},
			)

		actualNestedDictionary := getNestedDictionary(actualRootValue, owner)

		type insert struct {
			key   cadence.Value
			value cadence.Value
		}

		performInsert := func(dictionary *interpreter.DictionaryValue, insert insert) {

			newKey := importValue(t, inter, insert.key)
			newValue := importValue(t, inter, insert.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				dictionary.Insert(
					inter,
					interpreter.EmptyLocationRange,
					newKey,
					newValue,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var inserts []insert
		insertSet := map[any]struct{}{}

		for i := 0; i < opCount; i++ {
			// Generate a unique key
			var key cadence.Value
			for {
				key = r.randomHashableValue(inter)

				importedKey := importValue(t, inter, key)
				if actualNestedDictionary.ContainsKey(
					inter,
					interpreter.EmptyLocationRange,
					importedKey,
				) {
					continue
				}

				mapKey := mapKey(inter, importedKey)
				if _, ok := insertSet[mapKey]; ok {
					continue
				}
				insertSet[mapKey] = struct{}{}

				break
			}

			inserts = append(
				inserts,
				insert{
					key:   key,
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, insert := range inserts {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedDictionary = getNestedDictionary(actualRootValue, owner)

			performInsert(
				actualNestedDictionary,
				insert,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedDictionary := getNestedDictionary(expectedRootValue, common.ZeroAddress)

			for _, insert := range inserts[:i+1] {

				performInsert(
					expectedNestedDictionary,
					insert,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedDictionary :=
			createValue(
				t,
				&r,
				inter,
				// Generate a non-empty dictionary,
				// so we have at least one element to update
				func(dictionary cadence.Dictionary) bool {
					return len(dictionary.Pairs) > 0
				},
			)

		actualNestedDictionary := getNestedDictionary(actualRootValue, owner)

		elementCount := actualNestedDictionary.Count()
		require.Greater(t, elementCount, 0)

		type update struct {
			key   cadence.Value
			value cadence.Value
		}

		performUpdate := func(dictionary *interpreter.DictionaryValue, update update) {

			key := importValue(t, inter, update.key)
			newValue := importValue(t, inter, update.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				dictionary.SetKey(
					inter,
					interpreter.EmptyLocationRange,
					key,
					interpreter.NewUnmeteredSomeValueNonCopying(newValue),
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}

			// Dictionary must have same number of elements
			require.Equal(t, elementCount, dictionary.Count())
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		keys := make([]cadence.Value, 0, elementCount)

		actualNestedDictionary.IterateKeys(
			inter,
			interpreter.EmptyLocationRange,
			func(key interpreter.Value) (resume bool) {
				cadenceKey, err := runtime.ExportValue(
					key,
					inter,
					interpreter.EmptyLocationRange,
				)
				require.NoError(t, err)

				keys = append(keys, cadenceKey)

				return true
			},
		)

		var updates []update

		for i := 0; i < opCount; i++ {
			index := r.rand.Intn(elementCount)

			updates = append(
				updates,
				update{
					key:   keys[index],
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, update := range updates {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedDictionary = getNestedDictionary(actualRootValue, owner)

			performUpdate(
				actualNestedDictionary,
				update,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedDictionary := getNestedDictionary(expectedRootValue, common.ZeroAddress)

			for _, update := range updates[:i+1] {

				performUpdate(
					expectedNestedDictionary,
					update,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedDictionary :=
			createValue(
				t,
				&r,
				inter,
				func(dictionary cadence.Dictionary) bool {
					return len(dictionary.Pairs) >= opCount
				},
			)

		actualNestedDictionary := getNestedDictionary(actualRootValue, owner)

		elementCount := actualNestedDictionary.Count()
		require.GreaterOrEqual(t, elementCount, opCount)

		performRemove := func(dictionary *interpreter.DictionaryValue, key cadence.Value) {

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				dictionary.Remove(
					inter,
					interpreter.EmptyLocationRange,
					importValue(t, inter, key),
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		keys := make([]interpreter.Value, 0, elementCount)

		actualNestedDictionary.IterateKeys(
			inter,
			interpreter.EmptyLocationRange,
			func(key interpreter.Value) (resume bool) {

				keys = append(keys, key)

				return true
			},
		)

		var removes []cadence.Value
		removeSet := map[any]struct{}{}

		for i := 0; i < opCount; i++ {
			// Find a unique key
			var key interpreter.Value
			for {
				key = keys[r.rand.Intn(elementCount)]

				mapKey := mapKey(inter, key)
				if _, ok := removeSet[mapKey]; ok {
					continue
				}
				removeSet[mapKey] = struct{}{}

				break
			}

			cadenceKey, err := runtime.ExportValue(
				key,
				inter,
				interpreter.EmptyLocationRange,
			)
			require.NoError(t, err)

			removes = append(removes, cadenceKey)
		}

		for i, index := range removes {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedDictionary = getNestedDictionary(actualRootValue, owner)

			performRemove(
				actualNestedDictionary,
				index,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedDictionary := getNestedDictionary(expectedRootValue, common.ZeroAddress)

			for _, index := range removes[:i+1] {

				performRemove(
					expectedNestedDictionary,
					index,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})
}

func TestInterpretRandomNestedCompositeOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	owner := common.Address{'A'}

	limits := randomValueLimits{
		containerMaxDepth:  6,
		containerMaxSize:   20,
		compositeMaxFields: 10,
	}

	const opCount = 5

	createValue := func(
		t *testing.T,
		r *randomValueGenerator,
		inter *interpreter.Interpreter,
		predicate func(cadence.Composite) bool,
	) (
		actualRootValue interpreter.Value,
		generatedValue cadence.Value,
		reloadActualRootValue func() interpreter.Value,
		getNestedComposite func(rootValue interpreter.Value, owner common.Address) *interpreter.CompositeValue,
	) {

		// It does not matter what the root value is,
		// as long as it contains a nested composite,
		// which it is nested inside an optional,
		// and it satisfies the given predicate.

		var path []pathElement
		for {
			generatedValue = r.randomStructValue(inter, 0)

			path = findNestedValue(
				generatedValue,
				func(value cadence.Value, path []pathElement) bool {
					composite, ok := value.(cadence.Struct)
					if !ok {
						return false
					}

					if !predicate(composite) {
						return false
					}

					var foundSome bool
					for _, element := range path {
						if _, ok := element.(somePathElement); ok {
							foundSome = true
							break
						}
					}
					return foundSome
				},
			)
			if path != nil {
				break
			}
		}

		actualRootValue = importValue(t, inter, generatedValue).Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(owner),
			false,
			nil,
			nil,
			// TODO: is has no parent container = true correct?
			true,
		)

		const compositeStorageMapKey = interpreter.StringStorageMapKey("composite")

		// Store the composite in a storage map, so that the composite's slab
		// is referenced by the root of the storage.

		inter.Storage().
			GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				true,
			).
			WriteValue(
				inter,
				compositeStorageMapKey,
				actualRootValue,
			)

		reloadActualRootValue = func() interpreter.Value {
			storageMap := inter.Storage().GetStorageMap(
				owner,
				common.PathDomainStorage.Identifier(),
				false,
			)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(inter, compositeStorageMapKey)
			require.NotNil(t, readValue)

			require.IsType(t, &interpreter.CompositeValue{}, readValue)
			return readValue.(*interpreter.CompositeValue)
		}

		getNestedComposite = func(rootValue interpreter.Value, owner common.Address) *interpreter.CompositeValue {
			nestedValue := getNestedValue(t, inter, rootValue, path)
			require.IsType(t, &interpreter.CompositeValue{}, nestedValue)
			nestedComposite := nestedValue.(*interpreter.CompositeValue)
			require.Equal(t, owner, nestedComposite.GetOwner())
			return nestedComposite
		}

		return
	}

	t.Run("insert", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedComposite :=
			createValue(
				t,
				&r,
				inter,
				// Accept any composite, even empty ones,
				// given we're only inserting
				func(composite cadence.Composite) bool {
					return true
				},
			)

		actualNestedComposite := getNestedComposite(actualRootValue, owner)

		type insert struct {
			name  string
			value cadence.Value
		}

		performInsert := func(composite *interpreter.CompositeValue, insert insert) {

			newValue := importValue(t, inter, insert.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				composite.SetMember(
					inter,
					interpreter.EmptyLocationRange,
					insert.name,
					newValue,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var inserts []insert
		insertSet := map[string]struct{}{}

		for i := 0; i < opCount; i++ {
			// Generate a unique name
			var name string
			for {
				name = r.randomUTF8String()

				if actualNestedComposite.GetMember(
					inter,
					interpreter.EmptyLocationRange,
					name,
				) != nil {
					continue
				}

				if _, ok := insertSet[name]; ok {
					continue
				}
				insertSet[name] = struct{}{}

				break
			}

			inserts = append(
				inserts,
				insert{
					name:  name,
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, insert := range inserts {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedComposite = getNestedComposite(actualRootValue, owner)

			performInsert(
				actualNestedComposite,
				insert,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedComposite := getNestedComposite(expectedRootValue, common.ZeroAddress)

			for _, insert := range inserts[:i+1] {

				performInsert(
					expectedNestedComposite,
					insert,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedComposite :=
			createValue(
				t,
				&r,
				inter,
				// Generate a non-empty composite,
				// so we have at least one element to update
				func(composite cadence.Composite) bool {
					return len(composite.FieldsMappedByName()) > 0
				},
			)

		actualNestedComposite := getNestedComposite(actualRootValue, owner)

		fieldCount := actualNestedComposite.FieldCount()
		require.Greater(t, fieldCount, 0)

		type update struct {
			name  string
			value cadence.Value
		}

		performUpdate := func(composite *interpreter.CompositeValue, update update) {

			newValue := importValue(t, inter, update.value)

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				composite.SetMember(
					inter,
					interpreter.EmptyLocationRange,
					update.name,
					interpreter.NewUnmeteredSomeValueNonCopying(newValue),
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}

			// Composite must have same number of elements
			require.Equal(t, fieldCount, composite.FieldCount())
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		var updates []update

		fieldNames := make([]string, 0, fieldCount)

		actualNestedComposite.ForEachFieldName(
			func(name string) (resume bool) {
				fieldNames = append(fieldNames, name)
				return true
			},
		)

		for i := 0; i < opCount; i++ {
			index := r.rand.Intn(fieldCount)

			updates = append(
				updates,
				update{
					name:  fieldNames[index],
					value: r.randomStorableValue(inter, 0),
				},
			)
		}

		for i, update := range updates {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedComposite = getNestedComposite(actualRootValue, owner)

			performUpdate(
				actualNestedComposite,
				update,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedComposite := getNestedComposite(expectedRootValue, common.ZeroAddress)

			for _, update := range updates[:i+1] {

				performUpdate(
					expectedNestedComposite,
					update,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		inter, resetStorage := newRandomValueTestInterpreter(t)

		r := newRandomValueGenerator(
			*smokeTestSeed,
			limits,
		)
		t.Logf("seed: %d", r.seed)

		actualRootValue, generatedValue, reloadActualRootValue, getNestedComposite :=
			createValue(
				t,
				&r,
				inter,
				func(composite cadence.Composite) bool {
					return len(composite.FieldsMappedByName()) >= opCount
				},
			)

		actualNestedComposite := getNestedComposite(actualRootValue, owner)

		fieldCount := actualNestedComposite.FieldCount()
		require.GreaterOrEqual(t, fieldCount, opCount)

		performRemove := func(composite *interpreter.CompositeValue, name string) {

			// Atree storage validation must be temporarily disabled
			// to not report any "unreferenced slab" errors.

			withoutAtreeStorageValidationEnabled(inter, func() struct{} {
				composite.RemoveMember(
					inter,
					interpreter.EmptyLocationRange,
					name,
				)
				return struct{}{}
			})

			if *validateAtree {
				err := inter.Storage().CheckHealth()
				require.NoError(t, err)
			}
		}

		// We use the generated value twice: once as the expected value, and once as the actual value.
		// We first perform mutations on the actual value, and then compare it to the expected value.
		// The actual value is stored in an account and reloaded.
		// The expected value is temporary (zero address), and is not stored in storage.
		// Given that the storage reset destroys the data for the expected value because it is temporary,
		// we re-import it each time and perform all operations on it from scratch.

		fieldNames := make([]string, 0, fieldCount)

		actualNestedComposite.ForEachFieldName(
			func(name string) (resume bool) {

				fieldNames = append(fieldNames, name)

				return true
			},
		)

		var removes []string
		removeSet := map[string]struct{}{}

		for i := 0; i < opCount; i++ {
			// Find a unique name
			var name string
			for {
				name = fieldNames[r.rand.Intn(fieldCount)]

				if _, ok := removeSet[name]; ok {
					continue
				}
				removeSet[name] = struct{}{}

				break
			}

			removes = append(removes, name)
		}

		for i, index := range removes {

			resetStorage()

			actualRootValue = reloadActualRootValue()
			actualNestedComposite = getNestedComposite(actualRootValue, owner)

			performRemove(
				actualNestedComposite,
				index,
			)

			// Re-create the expected value from scratch,
			// by importing the generated value, and performing all updates on it
			// that have been performed on the actual value so far.

			expectedRootValue := importValue(t, inter, generatedValue)
			expectedNestedComposite := getNestedComposite(expectedRootValue, common.ZeroAddress)

			for _, index := range removes[:i+1] {

				performRemove(
					expectedNestedComposite,
					index,
				)
			}
			utils.AssertValuesEqual(t, inter, expectedRootValue, actualRootValue)
		}
	})
}

func findNestedValue(
	value cadence.Value,
	predicate func(value cadence.Value, path []pathElement) bool,
) []pathElement {
	return findNestedRecursive(value, nil, predicate)
}

func findNestedRecursive(
	value cadence.Value,
	path []pathElement,
	predicate func(value cadence.Value, path []pathElement) bool,
) []pathElement {
	if predicate(value, path) {
		return path
	}

	switch value := value.(type) {
	case cadence.Array:
		for index, element := range value.Values {

			nestedPath := append(path, arrayPathElement{index})

			result := findNestedRecursive(element, nestedPath, predicate)
			if result != nil {
				return result
			}
		}

	case cadence.Dictionary:
		for _, pair := range value.Pairs {

			nestedPath := append(path, dictionaryPathElement{pair.Key})

			result := findNestedRecursive(pair.Value, nestedPath, predicate)
			if result != nil {
				return result
			}
		}

	case cadence.Struct:
		for name, field := range value.FieldsMappedByName() {

			nestedPath := append(path, structPathElement{name})

			result := findNestedRecursive(field, nestedPath, predicate)
			if result != nil {
				return result
			}
		}

	case cadence.Optional:
		nestedValue := value.Value
		if nestedValue == nil {
			break
		}

		nestedPath := append(path, somePathElement{})

		result := findNestedRecursive(nestedValue, nestedPath, predicate)
		if result != nil {
			return result
		}
	}

	return nil
}

func getNestedValue(
	t *testing.T,
	inter *interpreter.Interpreter,
	value interpreter.Value,
	path []pathElement,
) interpreter.Value {
	for i, element := range path {
		switch element := element.(type) {
		case arrayPathElement:
			require.IsType(
				t,
				&interpreter.ArrayValue{},
				value,
				"path: %v",
				path[:i],
			)
			array := value.(*interpreter.ArrayValue)

			value = array.Get(
				inter,
				interpreter.EmptyLocationRange,
				element.index,
			)

			require.NotNil(t,
				value,
				"missing value for array element %d (path: %v)",
				element.index,
				path[:i],
			)

		case dictionaryPathElement:
			require.IsType(
				t,
				&interpreter.DictionaryValue{},
				value,
				"path: %v",
				path[:i],
			)
			dictionary := value.(*interpreter.DictionaryValue)

			key := importValue(t, inter, element.key)

			var found bool
			value, found = dictionary.Get(
				inter,
				interpreter.EmptyLocationRange,
				key,
			)
			require.True(t,
				found,
				"missing value for dictionary key %s (path: %v)",
				element.key,
				path[:i],
			)
			require.NotNil(t,
				value,
				"missing value for dictionary key %s (path: %v)",
				element.key,
				path[:i],
			)

		case structPathElement:
			require.IsType(
				t,
				&interpreter.CompositeValue{},
				value,
				"path: %v",
				path[:i],
			)
			composite := value.(*interpreter.CompositeValue)

			value = composite.GetMember(
				inter,
				interpreter.EmptyLocationRange,
				element.name,
			)

			require.NotNil(t,
				value,
				"missing value for composite field %q (path: %v)",
				element.name,
				path[:i],
			)

		case somePathElement:
			require.IsType(
				t,
				&interpreter.SomeValue{},
				value,
				"path: %v",
				path[:i],
			)
			optional := value.(*interpreter.SomeValue)

			value = optional.InnerValue(inter, interpreter.EmptyLocationRange)

			require.NotNil(t,
				value,
				"missing value for optional (path: %v)",
				path[:i],
			)

		default:
			panic(errors.NewUnexpectedError("unsupported path element: %T", element))
		}
	}

	return value
}

type pathElement interface {
	isPathElement()
}

type arrayPathElement struct {
	index int
}

var _ pathElement = arrayPathElement{}

func (arrayPathElement) isPathElement() {}

type dictionaryPathElement struct {
	key cadence.Value
}

var _ pathElement = dictionaryPathElement{}

func (dictionaryPathElement) isPathElement() {}

type structPathElement struct {
	name string
}

var _ pathElement = structPathElement{}

func (structPathElement) isPathElement() {}

type somePathElement struct{}

var _ pathElement = somePathElement{}

func (somePathElement) isPathElement() {}

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
	var kind randomValueKind
	if currentDepth < r.containerMaxDepth {
		kind = r.randomValueKind(randomValueKindStruct)
	} else {
		kind = r.randomValueKind(randomValueKindCapability)
	}

	switch kind {

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
		return r.generateHashableValueOfKind(inter, kind)
	}
}

func (r randomValueGenerator) randomHashableValue(inter *interpreter.Interpreter) cadence.Value {
	return r.generateHashableValueOfKind(inter, r.randomValueKind(randomValueKindEnum))
}

func (r randomValueGenerator) generateHashableValueOfKind(inter *interpreter.Interpreter, kind randomValueKind) cadence.Value {
	switch kind {

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
		panic(fmt.Sprintf("unsupported: %d", kind))
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

func (r randomValueGenerator) cadenceIntegerType(kind randomValueKind) cadence.Type {
	switch kind {
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
		panic(fmt.Sprintf("unsupported kind: %d", kind))
	}
}

func (r randomValueGenerator) semaIntegerType(kind randomValueKind) sema.Type {
	switch kind {
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
		panic(fmt.Sprintf("unsupported kind: %d", kind))
	}
}

type randomValueKind uint8

const (
	// Hashable values
	// Int*
	randomValueKindInt randomValueKind = iota
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
	typ := r.randomValueKind(randomValueKindWord64)

	rawValue := r.generateHashableValueOfKind(inter, typ).(cadence.NumberValue)

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
		Fields: []string{
			sema.EnumRawValueFieldName,
		},
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

func (r randomValueGenerator) randomValueKind(kind randomValueKind) randomValueKind {
	return randomValueKind(r.randomInt(int(kind)))
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
		type stringValue string
		return stringValue(key.Str)

	case interpreter.CharacterValue:
		type characterValue string
		return characterValue(key.Str)

	case interpreter.TypeValue:
		type typeValue common.TypeID
		return typeValue(key.Type.ID())

	case *interpreter.CompositeValue:
		type enumKey struct {
			location            common.Location
			qualifiedIdentifier string
			kind                common.CompositeKind
			rawValue            string
		}
		return enumKey{
			location:            key.Location,
			qualifiedIdentifier: key.QualifiedIdentifier,
			kind:                key.Kind,
			rawValue: key.GetField(
				inter,
				interpreter.EmptyLocationRange,
				sema.EnumRawValueFieldName,
			).String(),
		}

	case interpreter.IntValue:
		type intValue string
		return intValue(key.String())

	case interpreter.UIntValue:
		type uintValue string
		return uintValue(key.String())

	case interpreter.Int8Value:
		type int8Value string
		return int8Value(key.String())

	case interpreter.UInt8Value:
		type uint8Value string
		return uint8Value(key.String())

	case interpreter.Int16Value:
		type int16Value string
		return int16Value(key.String())

	case interpreter.UInt16Value:
		type uint16Value string
		return uint16Value(key.String())

	case interpreter.Int32Value:
		type int32Value string
		return int32Value(key.String())

	case interpreter.UInt32Value:
		type uint32Value string
		return uint32Value(key.String())

	case interpreter.Int64Value:
		type int64Value string
		return int64Value(key.String())

	case interpreter.UInt64Value:
		type uint64Value string
		return uint64Value(key.String())

	case interpreter.Int128Value:
		type int128Value string
		return int128Value(key.String())

	case interpreter.UInt128Value:
		type uint128Value string
		return uint128Value(key.String())

	case interpreter.Int256Value:
		type int256Value string
		return int256Value(key.String())

	case interpreter.UInt256Value:
		type uint256Value string
		return uint256Value(key.String())

	case interpreter.Word8Value:
		type word8Value string
		return word8Value(key.String())

	case interpreter.Word16Value:
		type word16Value string
		return word16Value(key.String())

	case interpreter.Word32Value:
		type word32Value string
		return word32Value(key.String())

	case interpreter.Word64Value:
		type word64Value string
		return word64Value(key.String())

	case interpreter.Word128Value:
		type word128Value string
		return word128Value(key.String())

	case interpreter.Word256Value:
		type word256Value string
		return word256Value(key.String())

	case interpreter.PathValue:
		return key

	case interpreter.AddressValue:
		return key

	case interpreter.BoolValue:
		return key

	case interpreter.Fix64Value:
		type fix64Value string
		return fix64Value(key.String())

	case interpreter.UFix64Value:
		type ufix64Value string
		return ufix64Value(key.String())

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
