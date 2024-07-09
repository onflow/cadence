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

const key = "key"
const value = "value"

func TestNewBiMap(t *testing.T) {
	actual := NewBiMap[string, string]()
	expected := &BiMap[string, string]{forward: make(map[string]string), backward: make(map[string]string)}
	assert.Equal(t, expected, actual, "They should be equal")
}

func TestBiMap_Insert(t *testing.T) {
	actual := NewBiMap[string, string]()
	actual.Insert(key, value)

	fwdExpected := make(map[string]string)
	invExpected := make(map[string]string)
	fwdExpected[key] = value
	invExpected[value] = key
	expected := &BiMap[string, string]{forward: fwdExpected, backward: invExpected}

	assert.Equal(t, expected, actual, "They should be equal")
}

func TestBiMap_InsertTwice(t *testing.T) {
	additionalValue := value + value

	actual := NewBiMap[string, string]()
	actual.Insert(key, value)
	actual.Insert(key, additionalValue)

	fwdExpected := make(map[string]string)
	invExpected := make(map[string]string)
	fwdExpected[key] = additionalValue

	invExpected[additionalValue] = key
	expected := &BiMap[string, string]{forward: fwdExpected, backward: invExpected}

	assert.Equal(t, expected, actual, "They should be equal")
}

func TestBiMap_Exists(t *testing.T) {
	actual := NewBiMap[string, string]()

	actual.Insert(key, value)
	assert.False(t, actual.Exists("ARBITRARY_KEY"), "Key should not exist")
	assert.True(t, actual.Exists(key), "Inserted key should exist")
}

func TestBiMap_InverseExists(t *testing.T) {
	actual := NewBiMap[string, string]()

	actual.Insert(key, value)
	assert.False(t, actual.ExistsInverse("ARBITRARY_VALUE"), "Value should not exist")
	assert.True(t, actual.ExistsInverse(value), "Inserted value should exist")
}

func TestBiMap_Get(t *testing.T) {
	actual := NewBiMap[string, string]()

	actual.Insert(key, value)

	actualVal, ok := actual.Get(key)

	assert.True(t, ok, "It should return true")
	assert.Equal(t, value, actualVal, "Value and returned val should be equal")

	actualVal, ok = actual.Get(value)

	assert.False(t, ok, "It should return false")
	assert.Empty(t, actualVal, "Actual val should be empty")
}

func TestBiMap_GetInverse(t *testing.T) {
	actual := NewBiMap[string, string]()

	actual.Insert(key, value)

	actualKey, ok := actual.GetInverse(value)

	assert.True(t, ok, "It should return true")
	assert.Equal(t, key, actualKey, "Key and returned key should be equal")

	actualKey, ok = actual.Get(value)

	assert.False(t, ok, "It should return false")
	assert.Empty(t, actualKey, "Actual key should be empty")
}

func TestBiMap_Size(t *testing.T) {
	actual := NewBiMap[string, string]()

	assert.Equal(t, 0, actual.Size(), "Length of empty bimap should be zero")

	actual.Insert(key, value)

	assert.Equal(t, 1, actual.Size(), "Length of bimap should be one")
}

func TestBiMap_Delete(t *testing.T) {
	actual := NewBiMap[string, string]()
	dummyKey := "DummyKey"
	dummyVal := "DummyVal"
	actual.Insert(key, value)
	actual.Insert(dummyKey, dummyVal)

	assert.Equal(t, 2, actual.Size(), "Size of bimap should be two")

	actual.Delete(dummyKey)

	fwdExpected := make(map[string]string)
	invExpected := make(map[string]string)
	fwdExpected[key] = value
	invExpected[value] = key

	expected := &BiMap[string, string]{forward: fwdExpected, backward: invExpected}

	assert.Equal(t, 1, actual.Size(), "Size of bimap should be two")
	assert.Equal(t, expected, actual, "They should be the same")

	actual.Delete(dummyKey)

	assert.Equal(t, 1, actual.Size(), "Size of bimap should be two")
	assert.Equal(t, expected, actual, "They should be the same")
}

func TestBiMap_InverseDelete(t *testing.T) {
	actual := NewBiMap[string, string]()
	dummyKey := "DummyKey"
	dummyVal := "DummyVal"
	actual.Insert(key, value)
	actual.Insert(dummyKey, dummyVal)

	assert.Equal(t, 2, actual.Size(), "Size of bimap should be two")

	actual.DeleteInverse(dummyVal)

	fwdExpected := make(map[string]string)
	invExpected := make(map[string]string)
	fwdExpected[key] = value
	invExpected[value] = key

	expected := &BiMap[string, string]{forward: fwdExpected, backward: invExpected}

	assert.Equal(t, 1, actual.Size(), "Size of bimap should be two")
	assert.Equal(t, expected, actual, "They should be the same")

	actual.DeleteInverse(dummyVal)

	assert.Equal(t, 1, actual.Size(), "Size of bimap should be two")
	assert.Equal(t, expected, actual, "They should be the same")
}

func TestBiMap_WithVaryingType(t *testing.T) {
	actual := NewBiMap[string, int]()
	dummyKey := "Dummy key"
	dummyVal := 3

	actual.Insert(dummyKey, dummyVal)

	res, _ := actual.Get(dummyKey)
	resVal, _ := actual.GetInverse(dummyVal)
	assert.Equal(t, dummyVal, res, "Get by string key should return integer val")
	assert.Equal(t, dummyKey, resVal, "Get by integer val should return string key")

}
