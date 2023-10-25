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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

// TODO: make these program args?
const containerMaxDepth = 3
const containerMaxSize = 100
const compositeMaxFields = 10

var runSmokeTests = flag.Bool("runSmokeTests", false, "Run smoke tests on values")
var validateAtree = flag.Bool("validateAtree", true, "Enable atree validation")
var smokeTestSeed = flag.Int64("smokeTestSeed", -1, "Seed for prng (-1 specifies current Unix time)")

func TestInterpretRandomMapOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	t.Parallel()

	r := newRandomValueGenerator()
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
			AtreeStorageValidationEnabled: *validateAtree,
			AtreeValueValidationEnabled:   *validateAtree,
		},
	)
	require.NoError(t, err)

	numberOfValues := r.randomInt(containerMaxSize)

	var testMap, copyOfTestMap *interpreter.DictionaryValue
	var storageSize, slabCounts int

	entries := newValueMap(numberOfValues)
	orgOwner := common.Address{'A'}

	t.Run("construction", func(t *testing.T) {
		keyValues := make([]interpreter.Value, numberOfValues*2)
		for i := 0; i < numberOfValues; i++ {
			key := r.randomHashableValue(inter)
			value := r.randomStorableValue(inter, 0)

			entries.put(inter, key, value)

			keyValues[i*2] = key
			keyValues[i*2+1] = value
		}

		testMap = interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			keyValues...,
		)

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		require.Equal(t, testMap.Count(), entries.size())

		entries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			exists := testMap.ContainsKey(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, bool(exists))

			value, found := testMap.Get(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)

			return false
		})

		owner := testMap.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("iterate", func(t *testing.T) {
		require.Equal(t, testMap.Count(), entries.size())

		testMap.Iterate(
			inter,
			interpreter.EmptyLocationRange,
			func(key, value interpreter.Value) (resume bool) {
				orgValue, ok := entries.get(inter, key)
				require.True(t, ok, "cannot find key: %v", key)

				utils.AssertValuesEqual(t, inter, orgValue, value)
				return true
			},
		)
	})

	t.Run("deep copy", func(t *testing.T) {
		newOwner := atree.Address{'B'}
		copyOfTestMap = testMap.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			false,
			nil,
			nil,
		).(*interpreter.DictionaryValue)

		require.Equal(t, entries.size(), copyOfTestMap.Count())

		entries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			exists := copyOfTestMap.ContainsKey(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, bool(exists))

			value, found := copyOfTestMap.Get(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)

			return false
		})

		owner := copyOfTestMap.GetOwner()
		assert.Equal(t, newOwner[:], owner[:])
	})

	t.Run("deep remove", func(t *testing.T) {
		copyOfTestMap.DeepRemove(inter)
		err = storage.Remove(copyOfTestMap.SlabID())
		require.NoError(t, err)

		// deep removal should clean up everything
		newStorageSize, newSlabCounts := getSlabStorageSize(t, storage)
		assert.Equal(t, slabCounts, newSlabCounts)
		assert.Equal(t, storageSize, newStorageSize)

		require.Equal(t, entries.size(), testMap.Count())

		// go over original values again and check no missing data (no side effect should be found)
		entries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			exists := testMap.ContainsKey(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, bool(exists))

			value, found := testMap.Get(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)

			return false
		})

		owner := testMap.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("insert", func(t *testing.T) {
		newEntries := newValueMap(numberOfValues)

		dictionary := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		// Insert
		for i := 0; i < numberOfValues; i++ {
			key := r.randomHashableValue(inter)
			value := r.randomStorableValue(inter, 0)

			newEntries.put(inter, key, value)

			_ = dictionary.Insert(inter, interpreter.EmptyLocationRange, key, value)
		}

		require.Equal(t, newEntries.size(), dictionary.Count())

		// Go over original values again and check no missing data (no side effect should be found)
		newEntries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			exists := dictionary.ContainsKey(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, bool(exists))

			value, found := dictionary.Get(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)

			return false
		})
	})

	t.Run("remove", func(t *testing.T) {
		newEntries := newValueMap(numberOfValues)

		keyValues := make([][2]interpreter.Value, numberOfValues)
		for i := 0; i < numberOfValues; i++ {
			key := r.randomHashableValue(inter)
			value := r.randomStorableValue(inter, 0)

			newEntries.put(inter, key, value)

			keyValues[i][0] = key
			keyValues[i][1] = value
		}

		dictionary := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, dictionary.Count())

		// Get the initial storage size before inserting values
		startingStorageSize, startingSlabCounts := getSlabStorageSize(t, storage)

		// Insert
		for _, keyValue := range keyValues {
			dictionary.Insert(inter, interpreter.EmptyLocationRange, keyValue[0], keyValue[1])
		}

		require.Equal(t, newEntries.size(), dictionary.Count())

		// Remove
		newEntries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			removedValue := dictionary.Remove(inter, interpreter.EmptyLocationRange, orgKey)

			assert.IsType(t, &interpreter.SomeValue{}, removedValue)
			someValue := removedValue.(*interpreter.SomeValue)

			// Removed value must be same as the original value
			innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
			utils.AssertValuesEqual(t, inter, orgValue, innerValue)

			return false
		})

		// Dictionary must be empty
		require.Equal(t, 0, dictionary.Count())

		storageSize, slabCounts := getSlabStorageSize(t, storage)

		// Storage size after removals should be same as the size before insertion.
		assert.Equal(t, startingStorageSize, storageSize)
		assert.Equal(t, startingSlabCounts, slabCounts)
	})

	t.Run("remove enum key", func(t *testing.T) {

		dictionary := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, dictionary.Count())

		// Get the initial storage size after creating empty dictionary
		startingStorageSize, startingSlabCounts := getSlabStorageSize(t, storage)

		newEntries := newValueMap(numberOfValues)

		keyValues := make([][2]interpreter.Value, numberOfValues)
		for i := 0; i < numberOfValues; i++ {
			// Create a random enum as key
			key := r.generateRandomHashableValue(inter, randomValueKindEnum)
			value := interpreter.Void

			newEntries.put(inter, key, value)

			keyValues[i][0] = key
			keyValues[i][1] = value
		}

		// Insert
		for _, keyValue := range keyValues {
			dictionary.Insert(inter, interpreter.EmptyLocationRange, keyValue[0], keyValue[1])
		}

		// Remove
		newEntries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			removedValue := dictionary.Remove(inter, interpreter.EmptyLocationRange, orgKey)

			assert.IsType(t, &interpreter.SomeValue{}, removedValue)
			someValue := removedValue.(*interpreter.SomeValue)

			// Removed value must be same as the original value
			innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
			utils.AssertValuesEqual(t, inter, orgValue, innerValue)

			return false
		})

		// Dictionary must be empty
		require.Equal(t, 0, dictionary.Count())

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		// Storage size after removals should be same as the size before insertion.
		assert.Equal(t, startingStorageSize, storageSize)
		assert.Equal(t, startingSlabCounts, slabCounts)
	})

	t.Run("random insert & remove", func(t *testing.T) {
		keyValues := make([][2]interpreter.Value, numberOfValues)
		for i := 0; i < numberOfValues; i++ {
			// Generate unique key
			var key interpreter.Value
			for {
				key = r.randomHashableValue(inter)

				var foundConflict bool
				for j := 0; j < i; j++ {
					existingKey := keyValues[j][0]
					if key.(interpreter.EquatableValue).Equal(inter, interpreter.EmptyLocationRange, existingKey) {
						foundConflict = true
						break
					}
				}
				if !foundConflict {
					break
				}
			}

			keyValues[i][0] = key
			keyValues[i][1] = r.randomStorableValue(inter, 0)
		}

		dictionary := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, dictionary.Count())

		// Get the initial storage size before inserting values
		startingStorageSize, startingSlabCounts := getSlabStorageSize(t, storage)

		insertCount := 0
		deleteCount := 0

		isInsert := func() bool {
			if dictionary.Count() == 0 {
				return true
			}

			if insertCount >= numberOfValues {
				return false
			}

			return r.randomInt(1) == 1
		}

		for insertCount < numberOfValues || dictionary.Count() > 0 {
			// Perform a random operation out of insert/remove
			if isInsert() {
				key := keyValues[insertCount][0]
				if _, ok := key.(*interpreter.CompositeValue); ok {
					key = key.Clone(inter)
				}

				value := keyValues[insertCount][1].Clone(inter)

				dictionary.Insert(
					inter,
					interpreter.EmptyLocationRange,
					key,
					value,
				)
				insertCount++
			} else {
				key := keyValues[deleteCount][0]
				orgValue := keyValues[deleteCount][1]

				removedValue := dictionary.Remove(inter, interpreter.EmptyLocationRange, key)

				assert.IsType(t, &interpreter.SomeValue{}, removedValue)
				someValue := removedValue.(*interpreter.SomeValue)

				// Removed value must be same as the original value
				innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
				utils.AssertValuesEqual(t, inter, orgValue, innerValue)

				deleteCount++
			}
		}

		// Dictionary must be empty
		require.Equal(t, 0, dictionary.Count())

		storageSize, slabCounts := getSlabStorageSize(t, storage)

		// Storage size after removals should be same as the size before insertion.
		assert.Equal(t, startingStorageSize, storageSize)
		assert.Equal(t, startingSlabCounts, slabCounts)
	})

	t.Run("move", func(t *testing.T) {
		newOwner := atree.Address{'B'}

		entries := newValueMap(numberOfValues)

		keyValues := make([]interpreter.Value, numberOfValues*2)
		for i := 0; i < numberOfValues; i++ {
			key := r.randomHashableValue(inter)
			value := r.randomStorableValue(inter, 0)

			entries.put(inter, key, value)

			keyValues[i*2] = key
			keyValues[i*2+1] = value
		}

		dictionary := interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			keyValues...,
		)

		require.Equal(t, entries.size(), dictionary.Count())

		movedDictionary := dictionary.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			true,
			nil,
			nil,
		).(*interpreter.DictionaryValue)

		require.Equal(t, entries.size(), movedDictionary.Count())

		// Cleanup the slab of original dictionary.
		err := storage.Remove(dictionary.SlabID())
		require.NoError(t, err)

		// Check the values
		entries.foreach(func(orgKey, orgValue interpreter.Value) (exit bool) {
			exists := movedDictionary.ContainsKey(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, bool(exists))

			value, found := movedDictionary.Get(inter, interpreter.EmptyLocationRange, orgKey)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)

			return false
		})

		owner := movedDictionary.GetOwner()
		assert.Equal(t, newOwner[:], owner[:])
	})
}

func TestInterpretRandomArrayOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	r := newRandomValueGenerator()
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
		},
	)
	require.NoError(t, err)

	numberOfValues := r.randomInt(containerMaxSize)

	var testArray, copyOfTestArray *interpreter.ArrayValue
	var storageSize, slabCounts int

	elements := make([]interpreter.Value, numberOfValues)
	orgOwner := common.Address{'A'}

	t.Run("construction", func(t *testing.T) {
		values := make([]interpreter.Value, numberOfValues)
		for i := 0; i < numberOfValues; i++ {
			value := r.randomStorableValue(inter, 0)
			elements[i] = value
			values[i] = value.Clone(inter)
		}

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			values...,
		)

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		require.Equal(t, len(elements), testArray.Count())

		for index, orgElement := range elements {
			element := testArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		owner := testArray.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("iterate", func(t *testing.T) {
		require.Equal(t, testArray.Count(), len(elements))

		index := 0
		testArray.Iterate(inter, func(element interpreter.Value) (resume bool) {
			orgElement := elements[index]
			utils.AssertValuesEqual(t, inter, orgElement, element)

			elementByIndex := testArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, element, elementByIndex)

			index++
			return true
		})
	})

	t.Run("deep copy", func(t *testing.T) {
		newOwner := atree.Address{'B'}
		copyOfTestArray = testArray.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			false,
			nil,
			nil,
		).(*interpreter.ArrayValue)

		require.Equal(t, len(elements), copyOfTestArray.Count())

		for index, orgElement := range elements {
			element := copyOfTestArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		owner := copyOfTestArray.GetOwner()
		assert.Equal(t, newOwner[:], owner[:])
	})

	t.Run("deep removal", func(t *testing.T) {
		copyOfTestArray.DeepRemove(inter)
		err = storage.Remove(copyOfTestArray.SlabID())
		require.NoError(t, err)

		// deep removal should clean up everything
		newStorageSize, newSlabCounts := getSlabStorageSize(t, storage)
		assert.Equal(t, slabCounts, newSlabCounts)
		assert.Equal(t, storageSize, newStorageSize)

		assert.Equal(t, len(elements), testArray.Count())

		// go over original elements again and check no missing data (no side effect should be found)
		for index, orgElement := range elements {
			element := testArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		owner := testArray.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("insert", func(t *testing.T) {
		newElements := make([]interpreter.Value, numberOfValues)

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, testArray.Count())

		for i := 0; i < numberOfValues; i++ {
			element := r.randomStorableValue(inter, 0)
			newElements[i] = element

			testArray.Insert(
				inter,
				interpreter.EmptyLocationRange,
				i,
				element.Clone(inter),
			)
		}

		require.Equal(t, len(newElements), testArray.Count())

		// Go over original values again and check no missing data (no side effect should be found)
		for index, element := range newElements {
			value := testArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, element, value)
		}
	})

	t.Run("append", func(t *testing.T) {
		newElements := make([]interpreter.Value, numberOfValues)

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, testArray.Count())

		for i := 0; i < numberOfValues; i++ {
			element := r.randomStorableValue(inter, 0)
			newElements[i] = element

			testArray.Append(
				inter,
				interpreter.EmptyLocationRange,
				element.Clone(inter),
			)
		}

		require.Equal(t, len(newElements), testArray.Count())

		// Go over original values again and check no missing data (no side effect should be found)
		for index, element := range newElements {
			value := testArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, element, value)
		}
	})

	t.Run("remove", func(t *testing.T) {
		newElements := make([]interpreter.Value, numberOfValues)

		for i := 0; i < numberOfValues; i++ {
			newElements[i] = r.randomStorableValue(inter, 0)
		}

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, testArray.Count())

		// Get the initial storage size before inserting values
		startingStorageSize, startingSlabCounts := getSlabStorageSize(t, storage)

		// Insert
		for index, element := range newElements {
			testArray.Insert(
				inter,
				interpreter.EmptyLocationRange,
				index,
				element.Clone(inter),
			)
		}

		require.Equal(t, len(newElements), testArray.Count())

		// Remove
		for _, element := range newElements {
			removedValue := testArray.Remove(inter, interpreter.EmptyLocationRange, 0)

			// Removed value must be same as the original value
			utils.AssertValuesEqual(t, inter, element, removedValue)
		}

		// Array must be empty
		require.Equal(t, 0, testArray.Count())

		storageSize, slabCounts := getSlabStorageSize(t, storage)

		// Storage size after removals should be same as the size before insertion.
		assert.Equal(t, startingStorageSize, storageSize)
		assert.Equal(t, startingSlabCounts, slabCounts)
	})

	t.Run("random insert & remove", func(t *testing.T) {
		elements := make([]interpreter.Value, numberOfValues)

		for i := 0; i < numberOfValues; i++ {
			elements[i] = r.randomStorableValue(inter, 0)
		}

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
		)

		require.Equal(t, 0, testArray.Count())

		// Get the initial storage size before inserting values
		startingStorageSize, startingSlabCounts := getSlabStorageSize(t, storage)

		insertCount := 0
		deleteCount := 0

		isInsert := func() bool {
			if testArray.Count() == 0 {
				return true
			}

			if insertCount >= numberOfValues {
				return false
			}

			return r.randomInt(1) == 1
		}

		for insertCount < numberOfValues || testArray.Count() > 0 {
			// Perform a random operation out of insert/remove
			if isInsert() {
				value := elements[insertCount].Clone(inter)

				testArray.Append(
					inter,
					interpreter.EmptyLocationRange,
					value,
				)
				insertCount++
			} else {
				orgValue := elements[deleteCount]
				removedValue := testArray.RemoveFirst(inter, interpreter.EmptyLocationRange)

				// Removed value must be same as the original value
				utils.AssertValuesEqual(t, inter, orgValue, removedValue)

				deleteCount++
			}
		}

		// Dictionary must be empty
		require.Equal(t, 0, testArray.Count())

		storageSize, slabCounts := getSlabStorageSize(t, storage)

		// Storage size after removals should be same as the size before insertion.
		assert.Equal(t, startingStorageSize, storageSize)
		assert.Equal(t, startingSlabCounts, slabCounts)
	})

	t.Run("move", func(t *testing.T) {
		values := make([]interpreter.Value, numberOfValues)
		elements := make([]interpreter.Value, numberOfValues)

		for i := 0; i < numberOfValues; i++ {
			value := r.randomStorableValue(inter, 0)
			elements[i] = value
			values[i] = value.Clone(inter)
		}

		array := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			values...,
		)

		require.Equal(t, len(elements), array.Count())

		owner := array.GetOwner()
		assert.Equal(t, orgOwner, owner)

		newOwner := atree.Address{'B'}
		movedArray := array.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			true,
			nil,
			nil,
		).(*interpreter.ArrayValue)

		require.Equal(t, len(elements), movedArray.Count())

		// Cleanup the slab of original array.
		err := storage.Remove(array.SlabID())
		require.NoError(t, err)

		// Check the elements
		for index, orgElement := range elements {
			element := movedArray.Get(inter, interpreter.EmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		owner = movedArray.GetOwner()
		assert.Equal(t, newOwner[:], owner[:])
	})
}

func TestInterpretRandomCompositeValueOperations(t *testing.T) {
	if !*runSmokeTests {
		t.Skip("smoke tests are disabled")
	}

	r := newRandomValueGenerator()
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
		},
	)
	require.NoError(t, err)

	var testComposite, copyOfTestComposite *interpreter.CompositeValue
	var storageSize, slabCounts int
	var orgFields map[string]interpreter.Value

	fieldsCount := r.randomInt(compositeMaxFields)
	orgOwner := common.Address{'A'}

	t.Run("construction", func(t *testing.T) {
		testComposite, orgFields = r.randomCompositeValue(orgOwner, fieldsCount, inter, 0)

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		for fieldName, orgFieldValue := range orgFields {
			fieldValue := testComposite.GetField(inter, interpreter.EmptyLocationRange, fieldName)
			utils.AssertValuesEqual(t, inter, orgFieldValue, fieldValue)
		}

		owner := testComposite.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("iterate", func(t *testing.T) {
		fieldCount := 0
		testComposite.ForEachField(inter, func(name string, value interpreter.Value) (resume bool) {
			orgValue, ok := orgFields[name]
			require.True(t, ok)
			utils.AssertValuesEqual(t, inter, orgValue, value)
			fieldCount++

			// continue iteration
			return true
		})

		assert.Equal(t, len(orgFields), fieldCount)
	})

	t.Run("deep copy", func(t *testing.T) {
		newOwner := atree.Address{'B'}

		copyOfTestComposite = testComposite.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			false,
			nil,
			nil,
		).(*interpreter.CompositeValue)

		for name, orgValue := range orgFields {
			value := copyOfTestComposite.GetField(inter, interpreter.EmptyLocationRange, name)
			utils.AssertValuesEqual(t, inter, orgValue, value)
		}

		owner := copyOfTestComposite.GetOwner()
		assert.Equal(t, newOwner[:], owner[:])
	})

	t.Run("deep remove", func(t *testing.T) {
		copyOfTestComposite.DeepRemove(inter)
		err = storage.Remove(copyOfTestComposite.SlabID())
		require.NoError(t, err)

		// deep removal should clean up everything
		newStorageSize, newSlabCounts := getSlabStorageSize(t, storage)
		assert.Equal(t, slabCounts, newSlabCounts)
		assert.Equal(t, storageSize, newStorageSize)

		// go over original values again and check no missing data (no side effect should be found)
		for name, orgValue := range orgFields {
			value := testComposite.GetField(inter, interpreter.EmptyLocationRange, name)
			utils.AssertValuesEqual(t, inter, orgValue, value)
		}

		owner := testComposite.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})

	t.Run("remove field", func(t *testing.T) {
		newOwner := atree.Address{'c'}

		composite := testComposite.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			false,
			nil,
			nil,
		).(*interpreter.CompositeValue)

		require.NoError(t, err)

		for name := range orgFields {
			composite.RemoveField(inter, interpreter.EmptyLocationRange, name)
			value := composite.GetField(inter, interpreter.EmptyLocationRange, name)
			assert.Nil(t, value)
		}
	})

	t.Run("move", func(t *testing.T) {
		composite, fields := r.randomCompositeValue(orgOwner, fieldsCount, inter, 0)

		owner := composite.GetOwner()
		assert.Equal(t, orgOwner, owner)

		newOwner := atree.Address{'B'}
		movedComposite := composite.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			newOwner,
			true,
			nil,
			nil,
		).(*interpreter.CompositeValue)

		// Cleanup the slab of original composite.
		err := storage.Remove(composite.SlabID())
		require.NoError(t, err)

		// Check the elements
		for fieldName, orgFieldValue := range fields {
			fieldValue := movedComposite.GetField(inter, interpreter.EmptyLocationRange, fieldName)
			utils.AssertValuesEqual(t, inter, orgFieldValue, fieldValue)
		}

		owner = composite.GetOwner()
		assert.Equal(t, orgOwner, owner)
	})
}

func (r randomValueGenerator) randomCompositeValue(
	orgOwner common.Address,
	fieldsCount int,
	inter *interpreter.Interpreter,
	currentDepth int,
) (*interpreter.CompositeValue, map[string]interpreter.Value) {

	orgFields := make(map[string]interpreter.Value, fieldsCount)

	identifier := r.randomUTF8String()

	location := common.AddressLocation{
		Address: orgOwner,
		Name:    identifier,
	}

	fields := make([]interpreter.CompositeField, fieldsCount)

	fieldNames := make(map[string]any, fieldsCount)

	for i := 0; i < fieldsCount; {
		fieldName := r.randomUTF8String()

		// avoid duplicate field names
		if _, ok := fieldNames[fieldName]; ok {
			continue
		}
		fieldNames[fieldName] = struct{}{}

		field := interpreter.NewUnmeteredCompositeField(
			fieldName,
			r.randomStorableValue(inter, currentDepth+1),
		)

		fields[i] = field
		orgFields[field.Name] = field.Value.Clone(inter)

		i++
	}

	kind := common.CompositeKindStructure

	compositeType := &sema.CompositeType{
		Location:   location,
		Identifier: identifier,
		Kind:       kind,
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

	// Add the type to the elaboration, to short-circuit the type-lookup
	inter.Program.Elaboration.SetCompositeType(
		compositeType.ID(),
		compositeType,
	)

	testComposite := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		location,
		identifier,
		kind,
		fields,
		orgOwner,
	)
	return testComposite, orgFields
}

func getSlabStorageSize(t *testing.T, storage interpreter.InMemoryStorage) (totalSize int, slabCounts int) {
	slabs, err := storage.Encode()
	require.NoError(t, err)

	for id, slab := range slabs {
		if id.HasTempAddress() {
			continue
		}

		totalSize += len(slab)
		slabCounts++
	}

	return
}

type randomValueGenerator struct {
	seed int64
	rand *rand.Rand
}

func newRandomValueGenerator() randomValueGenerator {
	seed := *smokeTestSeed
	if seed == -1 {
		seed = time.Now().UnixNano()
	}

	return randomValueGenerator{
		seed: seed,
		rand: rand.New(rand.NewSource(seed)),
	}
}
func (r randomValueGenerator) randomStorableValue(inter *interpreter.Interpreter, currentDepth int) interpreter.Value {
	n := 0
	if currentDepth < containerMaxDepth {
		n = r.randomInt(randomValueKindComposite)
	} else {
		n = r.randomInt(randomValueKindCapability)
	}

	switch n {

	// Non-hashable
	case randomValueKindVoid:
		return interpreter.Void
	case randomValueKindNil:
		return interpreter.Nil
	case randomValueKindDictionaryVariant1,
		randomValueKindDictionaryVariant2:
		return r.randomDictionaryValue(inter, currentDepth)
	case randomValueKindArrayVariant1,
		randomValueKindArrayVariant2:
		return r.randomArrayValue(inter, currentDepth)
	case randomValueKindComposite:
		fieldsCount := r.randomInt(compositeMaxFields)
		v, _ := r.randomCompositeValue(common.ZeroAddress, fieldsCount, inter, currentDepth)
		return v
	case randomValueKindCapability:
		return interpreter.NewUnmeteredCapabilityValue(
			interpreter.UInt64Value(r.randomInt(math.MaxInt-1)),
			r.randomAddressValue(),
			&interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
		)
	case randomValueKindSome:
		return interpreter.NewUnmeteredSomeValueNonCopying(
			r.randomStorableValue(inter, currentDepth+1),
		)

	// Hashable
	default:
		return r.generateRandomHashableValue(inter, n)
	}
}

func (r randomValueGenerator) randomHashableValue(interpreter *interpreter.Interpreter) interpreter.Value {
	return r.generateRandomHashableValue(interpreter, r.randomInt(randomValueKindEnum))
}

func (r randomValueGenerator) generateRandomHashableValue(inter *interpreter.Interpreter, n int) interpreter.Value {
	switch n {

	// Int*
	case randomValueKindInt:
		return interpreter.NewUnmeteredIntValueFromInt64(int64(r.randomSign()) * r.rand.Int63())
	case randomValueKindInt8:
		return interpreter.NewUnmeteredInt8Value(int8(r.randomInt(math.MaxUint8)))
	case randomValueKindInt16:
		return interpreter.NewUnmeteredInt16Value(int16(r.randomInt(math.MaxUint16)))
	case randomValueKindInt32:
		return interpreter.NewUnmeteredInt32Value(int32(r.randomSign()) * r.rand.Int31())
	case randomValueKindInt64:
		return interpreter.NewUnmeteredInt64Value(int64(r.randomSign()) * r.rand.Int63())
	case randomValueKindInt128:
		return interpreter.NewUnmeteredInt128ValueFromInt64(int64(r.randomSign()) * r.rand.Int63())
	case randomValueKindInt256:
		return interpreter.NewUnmeteredInt256ValueFromInt64(int64(r.randomSign()) * r.rand.Int63())

	// UInt*
	case randomValueKindUInt:
		return interpreter.NewUnmeteredUIntValueFromUint64(r.rand.Uint64())
	case randomValueKindUInt8:
		return interpreter.NewUnmeteredUInt8Value(uint8(r.randomInt(math.MaxUint8)))
	case randomValueKindUInt16:
		return interpreter.NewUnmeteredUInt16Value(uint16(r.randomInt(math.MaxUint16)))
	case randomValueKindUInt32:
		return interpreter.NewUnmeteredUInt32Value(r.rand.Uint32())
	case randomValueKindUInt64Variant1,
		randomValueKindUInt64Variant2,
		randomValueKindUInt64Variant3,
		randomValueKindUInt64Variant4: // should be more common
		return interpreter.NewUnmeteredUInt64Value(r.rand.Uint64())
	case randomValueKindUInt128:
		return interpreter.NewUnmeteredUInt128ValueFromUint64(r.rand.Uint64())
	case randomValueKindUInt256:
		return interpreter.NewUnmeteredUInt256ValueFromUint64(r.rand.Uint64())

	// Word*
	case randomValueKindWord8:
		return interpreter.NewUnmeteredWord8Value(uint8(r.randomInt(math.MaxUint8)))
	case randomValueKindWord16:
		return interpreter.NewUnmeteredWord16Value(uint16(r.randomInt(math.MaxUint16)))
	case randomValueKindWord32:
		return interpreter.NewUnmeteredWord32Value(r.rand.Uint32())
	case randomValueKindWord64:
		return interpreter.NewUnmeteredWord64Value(r.rand.Uint64())
	case randomValueKindWord128:
		return interpreter.NewUnmeteredWord128ValueFromUint64(r.rand.Uint64())
	case randomValueKindWord256:
		return interpreter.NewUnmeteredWord256ValueFromUint64(r.rand.Uint64())

	// (U)Fix*
	case randomValueKindFix64:
		return interpreter.NewUnmeteredFix64ValueWithInteger(
			int64(r.randomSign())*r.rand.Int63n(sema.Fix64TypeMaxInt),
			interpreter.EmptyLocationRange,
		)
	case randomValueKindUFix64:
		return interpreter.NewUnmeteredUFix64ValueWithInteger(
			uint64(r.rand.Int63n(
				int64(sema.UFix64TypeMaxInt),
			)),
			interpreter.EmptyLocationRange,
		)

	// String
	case randomValueKindStringVariant1,
		randomValueKindStringVariant2,
		randomValueKindStringVariant3,
		randomValueKindStringVariant4: // small string - should be more common
		size := r.randomInt(255)
		return interpreter.NewUnmeteredStringValue(r.randomUTF8StringOfSize(size))
	case randomValueKindStringVariant5: // large string
		size := r.randomInt(4048) + 255
		return interpreter.NewUnmeteredStringValue(r.randomUTF8StringOfSize(size))

	case randomValueKindBoolVariantTrue:
		return interpreter.TrueValue
	case randomValueKindBoolVariantFalse:
		return interpreter.FalseValue

	case randomValueKindAddress:
		return r.randomAddressValue()

	case randomValueKindPath:
		return r.randomPathValue()

	case randomValueKindEnum:
		// Get a random integer subtype to be used as the raw-type of enum
		typ := r.randomInt(randomValueKindWord64)

		rawValue := r.generateRandomHashableValue(inter, typ).(interpreter.NumberValue)

		identifier := r.randomUTF8String()

		address := r.randomAddressValue()

		location := common.AddressLocation{
			Address: common.Address(address),
			Name:    identifier,
		}

		enumType := &sema.CompositeType{
			Identifier:  identifier,
			EnumRawType: r.intSubtype(typ),
			Kind:        common.CompositeKindEnum,
			Location:    location,
		}

		inter.Program.Elaboration.SetCompositeType(
			enumType.ID(),
			enumType,
		)

		enum := interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			location,
			enumType.QualifiedIdentifier(),
			enumType.Kind,
			[]interpreter.CompositeField{
				{
					Name:  sema.EnumRawValueFieldName,
					Value: rawValue,
				},
			},
			common.ZeroAddress,
		)

		if enum.GetField(inter, interpreter.EmptyLocationRange, sema.EnumRawValueFieldName) == nil {
			panic("enum without raw value")
		}

		return enum

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

func (r randomValueGenerator) randomAddressValue() interpreter.AddressValue {
	data := make([]byte, 8)
	r.rand.Read(data)
	return interpreter.NewUnmeteredAddressValueFromBytes(data)
}

func (r randomValueGenerator) randomPathValue() interpreter.PathValue {
	randomDomain := r.rand.Intn(len(common.AllPathDomains))
	identifier := r.randomUTF8String()

	return interpreter.PathValue{
		Domain:     common.AllPathDomains[randomDomain],
		Identifier: identifier,
	}
}

func (r randomValueGenerator) randomDictionaryValue(
	inter *interpreter.Interpreter,
	currentDepth int,
) interpreter.Value {

	entryCount := r.randomInt(containerMaxSize)
	keyValues := make([]interpreter.Value, entryCount*2)

	for i := 0; i < entryCount; i++ {
		key := r.randomHashableValue(inter)
		value := r.randomStorableValue(inter, currentDepth+1)
		keyValues[i*2] = key
		keyValues[i*2+1] = value
	}

	return interpreter.NewDictionaryValueWithAddress(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.DictionaryStaticType{
			KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
			ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		keyValues...,
	)
}

func (r randomValueGenerator) randomInt(upperBound int) int {
	return r.rand.Intn(upperBound + 1)
}

func (r randomValueGenerator) randomArrayValue(inter *interpreter.Interpreter, currentDepth int) interpreter.Value {
	elementsCount := r.randomInt(containerMaxSize)
	elements := make([]interpreter.Value, elementsCount)

	for i := 0; i < elementsCount; i++ {
		value := r.randomStorableValue(inter, currentDepth+1)
		elements[i] = value.Clone(inter)
	}

	return interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		&interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		common.ZeroAddress,
		elements...,
	)
}

func (r randomValueGenerator) intSubtype(n int) sema.Type {
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
	randomValueKindComposite
)

func (r randomValueGenerator) randomUTF8String() string {
	return r.randomUTF8StringOfSize(8)
}

func (r randomValueGenerator) randomUTF8StringOfSize(size int) string {
	identifier := make([]byte, size)
	r.rand.Read(identifier)
	return strings.ToValidUTF8(string(identifier), "$")
}

type valueMap struct {
	values map[any]interpreter.Value
	keys   map[any]interpreter.Value
}

func newValueMap(size int) *valueMap {
	return &valueMap{
		values: make(map[any]interpreter.Value, size),
		keys:   make(map[any]interpreter.Value, size),
	}
}

type enumKey struct {
	location            common.Location
	qualifiedIdentifier string
	kind                common.CompositeKind
	rawValue            interpreter.Value
}

func (m *valueMap) put(inter *interpreter.Interpreter, key, value interpreter.Value) {
	internalKey := m.internalKey(inter, key)

	// Deep copy enum keys. This should be fine since we use an internal key for enums.
	// Deep copying other values would mess key-lookup.
	if _, ok := key.(*interpreter.CompositeValue); ok {
		key = key.Clone(inter)
	}

	m.keys[internalKey] = key
	m.values[internalKey] = value.Clone(inter)
}

func (m *valueMap) get(inter *interpreter.Interpreter, key interpreter.Value) (interpreter.Value, bool) {
	internalKey := m.internalKey(inter, key)
	value, ok := m.values[internalKey]
	return value, ok
}

func (m *valueMap) foreach(apply func(key, value interpreter.Value) (exit bool)) {
	for internalKey, key := range m.keys {
		value := m.values[internalKey]
		exit := apply(key, value)

		if exit {
			return
		}
	}
}

func (m *valueMap) internalKey(inter *interpreter.Interpreter, key interpreter.Value) any {
	switch key := key.(type) {
	case *interpreter.StringValue:
		return *key
	case *interpreter.CompositeValue:
		return enumKey{
			location:            key.Location,
			qualifiedIdentifier: key.QualifiedIdentifier,
			kind:                key.Kind,
			rawValue:            key.GetField(inter, interpreter.EmptyLocationRange, sema.EnumRawValueFieldName),
		}
	case interpreter.Value:
		return key
	default:
		panic("unreachable")
	}
}

func (m *valueMap) size() int {
	return len(m.keys)
}
