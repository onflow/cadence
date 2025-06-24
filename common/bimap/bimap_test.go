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
 *
 */

package bimap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testKey = "a"
const testValue = 1

func TestNewBiMap(t *testing.T) {
	actual := NewBiMap[string, int]()
	expected := &BiMap[string, int]{
		forward:  map[string]int{},
		backward: map[int]string{},
	}
	assert.Equal(t, expected, actual)
}

func TestBiMap_Insert(t *testing.T) {
	actual := NewBiMap[string, int]()
	actual.Insert(testKey, testValue)

	expected := &BiMap[string, int]{
		forward: map[string]int{
			testKey: testValue,
		},
		backward: map[int]string{
			testValue: testKey,
		},
	}

	assert.Equal(t, expected, actual)
}

func TestBiMap_InsertTwiceSameKey(t *testing.T) {
	const otherValue = 2

	actual := NewBiMap[string, int]()
	actual.Insert(testKey, testValue)
	actual.Insert(testKey, otherValue)

	expected := &BiMap[string, int]{
		forward: map[string]int{
			testKey: otherValue,
		},
		backward: map[int]string{
			otherValue: testKey,
		},
	}

	assert.Equal(t, expected, actual)
}

func TestBiMap_InsertTwiceSameValue(t *testing.T) {
	const otherKey = "b"

	actual := NewBiMap[string, int]()
	actual.Insert(testKey, testValue)
	actual.Insert(otherKey, testValue)

	inverse, ok := actual.GetInverse(testValue)
	assert.True(t, ok)
	assert.Equal(t, otherKey, inverse)

	expected := &BiMap[string, int]{
		forward: map[string]int{
			otherKey: testValue,
		},
		backward: map[int]string{
			testValue: otherKey,
		},
	}
	assert.Equal(t, expected, actual)
}

func TestBiMap_Exists(t *testing.T) {
	const otherKey = "b"

	actual := NewBiMap[string, int]()

	actual.Insert(testKey, testValue)

	assert.False(t, actual.Exists(otherKey))
	assert.True(t, actual.Exists(testKey))
}

func TestBiMap_InverseExists(t *testing.T) {
	const otherValue = 2

	actual := NewBiMap[string, int]()

	actual.Insert(testKey, testValue)

	assert.False(t, actual.ExistsInverse(otherValue))
	assert.True(t, actual.ExistsInverse(testValue))
}

func TestBiMap_Get(t *testing.T) {
	actual := NewBiMap[string, int]()

	actual.Insert(testKey, testValue)

	actualValue, ok := actual.Get(testKey)

	assert.True(t, ok)
	assert.Equal(t, testValue, actualValue)
}

func TestBiMap_GetInverse(t *testing.T) {
	actual := NewBiMap[string, int]()

	actual.Insert(testKey, testValue)

	actualKey, ok := actual.GetInverse(testValue)

	assert.True(t, ok)
	assert.Equal(t, testKey, actualKey)
}

func TestBiMap_Size(t *testing.T) {
	actual := NewBiMap[string, int]()

	assert.Equal(t, 0, actual.Size())

	actual.Insert(testKey, testValue)

	assert.Equal(t, 1, actual.Size())
}

func TestBiMap_Delete(t *testing.T) {
	const otherKey = "b"
	const otherValue = 2

	actual := NewBiMap[string, int]()
	actual.Insert(testKey, testValue)
	actual.Insert(otherKey, otherValue)

	assert.Equal(t, 2, actual.Size())

	actual.Delete(otherKey)

	expected := &BiMap[string, int]{
		forward: map[string]int{
			testKey: testValue,
		},
		backward: map[int]string{
			testValue: testKey,
		},
	}

	assert.Equal(t, 1, actual.Size())
	assert.Equal(t, expected, actual)

	actual.Delete(otherKey)

	assert.Equal(t, 1, actual.Size())
	assert.Equal(t, expected, actual)

	actual.Delete(testKey)

	expected = &BiMap[string, int]{
		forward:  map[string]int{},
		backward: map[int]string{},
	}

	assert.Equal(t, 0, actual.Size())
	assert.Equal(t, expected, actual)
}

func TestBiMap_InverseDelete(t *testing.T) {
	const otherKey = "b"
	const otherValue = 2

	actual := NewBiMap[string, int]()
	actual.Insert(testKey, testValue)
	actual.Insert(otherKey, otherValue)

	assert.Equal(t, 2, actual.Size())

	actual.DeleteInverse(otherValue)

	expected := &BiMap[string, int]{
		forward: map[string]int{
			testKey: testValue,
		},
		backward: map[int]string{
			testValue: testKey,
		},
	}

	assert.Equal(t, 1, actual.Size())
	assert.Equal(t, expected, actual)

	actual.DeleteInverse(otherValue)

	assert.Equal(t, 1, actual.Size())
	assert.Equal(t, expected, actual)

	actual.DeleteInverse(testValue)

	expected = &BiMap[string, int]{
		forward:  map[string]int{},
		backward: map[int]string{},
	}
	assert.Equal(t, 0, actual.Size())
	assert.Equal(t, expected, actual)
}
