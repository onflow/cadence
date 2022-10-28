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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package orderedmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Fruit struct {
	name  string
	color string
	price float32
}

// TestOrderedMapOperations tests the operations of the generic map.
func TestOrderedMapOperations(t *testing.T) {

	t.Parallel()

	t.Run("test constructor", func(t *testing.T) {
		om := OrderedMap[string, *Fruit]{}
		require.NotNil(t, om)
		require.Nil(t, om.list)
		require.Nil(t, om.pairs)
		assert.Equal(t, 0, om.Len())
	})

	t.Run("test map set", func(t *testing.T) {
		om := OrderedMap[string, *Fruit]{}
		require.NotNil(t, om)

		insertedValues := []*Fruit{
			{name: "orange", color: "orange", price: 1.0},
			{name: "apple", color: "red", price: 1.5},
			{name: "mango", color: "green", price: 1.3},
		}

		// Insert values
		for _, fruit := range insertedValues {
			oldValue, updated := om.Set(fruit.name, fruit)

			// Check the return value
			assert.False(t, updated)
			assert.Nil(t, oldValue)
		}

		require.NotNil(t, om.list)
		require.NotNil(t, om.pairs)
		require.Equal(t, len(insertedValues), len(om.pairs))
		require.Equal(t, len(insertedValues), om.list.Len())

		// Check map's internal values
		element := om.list.Front()
		for _, value := range insertedValues {
			name := value.name

			// check the internal map
			keyValuePair := om.pairs[name]
			assert.Equal(t, name, keyValuePair.Key)
			assert.Equal(t, value, keyValuePair.Value)

			// check the internal list
			assert.Equal(t, keyValuePair, element.Value)
			element = element.Next()
		}
	})

	t.Run("test map update", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)

		require.NotNil(t, om.list)
		require.NotNil(t, om.pairs)
		require.Equal(t, len(insertedValues), len(om.pairs))
		require.Equal(t, len(insertedValues), om.list.Len())

		const updateItemIndex = 1
		insertedOldValue := insertedValues[updateItemIndex]

		// Create a new value, and set it to the map, using an existing key
		newValue := &Fruit{insertedOldValue.name, "white", insertedOldValue.price}
		oldValue, updated := om.Set(newValue.name, newValue)

		// Check the return value
		assert.True(t, updated)
		assert.Equal(t, insertedOldValue, oldValue)

		element := om.list.Front()
		for i := 0; i < len(insertedValues); i++ {
			var value *Fruit
			if i == updateItemIndex {
				value = newValue
			} else {
				value = insertedValues[i]
			}

			name := value.name

			// check the internal map
			keyValuePair := om.pairs[name]
			assert.Equal(t, name, keyValuePair.Key)
			assert.Equal(t, value, keyValuePair.Value)

			// check the internal list
			assert.Equal(t, keyValuePair, element.Value)
			element = element.Next()
		}
	})

	t.Run("test map get", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)

		for _, insertedValue := range insertedValues {
			name := insertedValue.name
			value, ok := om.Get(name)
			require.True(t, ok)
			require.NotNil(t, value)
			require.IsType(t, &Fruit{}, value)
			assert.Equal(t, insertedValue, value)
		}
	})

	t.Run("test map get non existing key", func(t *testing.T) {
		om, _ := createAndPopulateMap(t)
		value, ok := om.Get("cat")
		require.False(t, ok)
		require.Nil(t, value)
	})

	t.Run("test map get pair", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)

		for _, insertedValue := range insertedValues {
			name := insertedValue.name
			pair := om.GetPair(name)
			require.NotNil(t, pair)
			require.IsType(t, &Pair[string, *Fruit]{}, pair)
			assert.Equal(t, name, pair.Key)
			assert.Equal(t, insertedValue, pair.Value)
		}
	})

	t.Run("test map get non existing pair", func(t *testing.T) {
		om, _ := createAndPopulateMap(t)
		pair := om.GetPair("dog")
		require.Nil(t, pair)
	})

	t.Run("test map length", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)
		assert.Equal(t, len(insertedValues), om.Len())
	})

	t.Run("test map delete", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)

		deleteItemIndex := 1
		deletedItem, ok := om.Delete(insertedValues[deleteItemIndex].name)

		require.True(t, ok)
		require.NotNil(t, deletedItem)
		require.IsType(t, &Fruit{}, deletedItem)
		assert.Equal(t, insertedValues[deleteItemIndex], deletedItem)

		require.Equal(t, len(insertedValues)-1, len(om.pairs))
		require.Equal(t, len(insertedValues)-1, om.list.Len())
		require.Equal(t, len(insertedValues)-1, om.Len())

		element := om.list.Front()
		for i := 0; i < len(insertedValues); i++ {
			if i == deleteItemIndex {
				continue
			}

			value := insertedValues[i]
			name := value.name

			// check the internal map
			keyValuePair := om.pairs[name]
			assert.Equal(t, name, keyValuePair.Key)
			assert.Equal(t, value, keyValuePair.Value)

			// check the internal list
			assert.Equal(t, keyValuePair, element.Value)
			element = element.Next()
		}
	})

	t.Run("test map delete non existing key", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)
		deletedItem, ok := om.Delete("cat")

		require.False(t, ok)
		require.Nil(t, deletedItem)
		require.Equal(t, len(insertedValues), len(om.pairs))
		require.Equal(t, len(insertedValues), om.list.Len())
		require.Equal(t, len(insertedValues), om.Len())

		element := om.list.Front()
		for _, insertedValue := range insertedValues {
			name := insertedValue.name

			// check the internal map
			keyValuePair := om.pairs[name]
			assert.Equal(t, name, keyValuePair.Key)
			assert.Equal(t, insertedValue, keyValuePair.Value)

			// check the internal list
			assert.Equal(t, keyValuePair, element.Value)
			element = element.Next()
		}
	})

	t.Run("test map get oldest", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)
		value := om.Oldest()

		require.NotNil(t, value)
		require.IsType(t, &Pair[string, *Fruit]{}, value)

		expected := insertedValues[0]
		assert.Equal(t, expected.name, value.Key)
		assert.Equal(t, expected, value.Value)
	})

	t.Run("test map get oldest for empty map", func(t *testing.T) {
		om := OrderedMap[string, *Fruit]{}
		require.Nil(t, om.Oldest())
	})

	t.Run("test map get newest", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)
		value := om.Newest()

		require.NotNil(t, value)
		require.IsType(t, &Pair[string, *Fruit]{}, value)

		expected := insertedValues[len(insertedValues)-1]
		assert.Equal(t, expected.name, value.Key)
		assert.Equal(t, expected, value.Value)
	})

	t.Run("test map get newest for empty map", func(t *testing.T) {
		om := OrderedMap[string, *Fruit]{}
		require.Nil(t, om.Newest())
	})

	t.Run("test map foreach", func(t *testing.T) {
		om, insertedValues := createAndPopulateMap(t)

		var loopResult []*Fruit
		om.Foreach(func(key string, value *Fruit) {
			loopResult = append(loopResult, value)
		})

		assert.Equal(t, insertedValues, loopResult)
	})
}

// TestGeneratedMapOperations tests the basic functionality of a generated map.
// This is to make sure any update to the generator would not cause any regression issues.
func TestGeneratedMapOperations(t *testing.T) {

	t.Parallel()

	fruits := OrderedMap[string, *Fruit]{}
	require.NotNil(t, fruits)

	apple := &Fruit{name: "apple", color: "red", price: 1.5}
	oldValue, updated := fruits.Set(apple.name, apple)
	assert.False(t, updated)
	assert.Nil(t, oldValue)

	mango := &Fruit{name: "mango", color: "green", price: 1.3}
	oldValue, updated = fruits.Set(mango.name, mango)
	assert.False(t, updated)
	assert.Nil(t, oldValue)

	orange := &Fruit{name: "orange", color: "orange", price: 1.0}
	oldValue, updated = fruits.Set(orange.name, orange)
	assert.False(t, updated)
	assert.Nil(t, oldValue)

	assert.Equal(t, 3, fruits.Len())

	newApple := &Fruit{name: "apple", color: "red", price: 1.8}
	oldValue, updated = fruits.Set(newApple.name, newApple)
	assert.True(t, updated)
	assert.Equal(t, apple, oldValue)

	assert.Equal(t, 3, fruits.Len())

	var loopResult []*Fruit
	fruits.Foreach(func(key string, value *Fruit) {
		loopResult = append(loopResult, value)
	})

	assert.Equal(t, []*Fruit{newApple, mango, orange}, loopResult)

	deleted, ok := fruits.Delete("orange")
	require.True(t, ok)
	require.NotNil(t, deleted)
	assert.Equal(t, orange, deleted)

	assert.Equal(t, 2, fruits.Len())

}

// Utility functions

func createAndPopulateMap(t *testing.T) (OrderedMap[string, *Fruit], []*Fruit) {
	om := OrderedMap[string, *Fruit]{}
	require.NotNil(t, om)

	fruits := []*Fruit{
		{name: "orange", color: "orange", price: 1.0},
		{name: "apple", color: "red", price: 1.5},
		{name: "mango", color: "green", price: 1.3},
	}

	for _, fruit := range fruits {
		om.Set(fruit.name, fruit)
	}

	require.NotNil(t, om.list)
	require.NotNil(t, om.pairs)
	require.Equal(t, 3, len(om.pairs))
	require.Equal(t, 3, om.list.Len())

	return om, fruits
}
