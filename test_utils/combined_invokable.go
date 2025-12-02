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

package test_utils

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
	"github.com/stretchr/testify/assert"
)

var (
	compareSlabs = func(a, b atree.Slab) int {
		return a.SlabID().Compare(b.SlabID())
	}
)

type CombinedInvokable struct {
	interpreter *interpreter.Interpreter
	*VMInvokable
}

var _ Invokable = &CombinedInvokable{}

func NewCombinedInvokable(
	interpreterInvokable *interpreter.Interpreter,
	vmInvokable *VMInvokable,
) *CombinedInvokable {
	return &CombinedInvokable{
		interpreter: interpreterInvokable,
		VMInvokable: vmInvokable,
	}
}

func (i *CombinedInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	vmResult, err := i.VMInvokable.Invoke(functionName, arguments...)
	if err != nil {
		return nil, err
	}

	// If the test was testup without the need for comparing slabs,
	// then return only the VM result.
	if i.interpreter == nil {
		return vmResult, nil
	}

	_, err = i.interpreter.Invoke(functionName, arguments...)
	if err != nil {
		return nil, err
	}

	vmStorage := i.VMInvokable.Storage()
	interpreterStorage := i.interpreter.Storage()

	if vmStorage.Count() != interpreterStorage.Count() {
		return nil, fmt.Errorf(
			"storage count mismatch. interpreter: %d, vm: %d",
			interpreterStorage.Count(),
			vmStorage.Count(),
		)
	}

	// Collect VM slabs

	vmSlabs, err := sortedSlabIDsFromStorage(vmStorage)
	if err != nil {
		return nil, err
	}

	// Collect Interpreter slabs
	interpreterSlabs, err := sortedSlabIDsFromStorage(interpreterStorage)
	if err != nil {
		return nil, err
	}

	var slabComparisonError error

	equalSlab := func(slab1, slab2 atree.Slab) bool {
		switch slab1 := slab1.(type) {
		case *atree.MapDataSlab:
			if mapDataSlab2, ok := slab2.(*atree.MapDataSlab); ok {
				slabComparisonError = compareMapDataSlabs(slab1, mapDataSlab2)
				if slabComparisonError != nil {
					return false
				}
			}

		case *atree.ArrayDataSlab:
			if arrayDataSlab2, ok := slab2.(*atree.ArrayDataSlab); ok {
				slabComparisonError = compareArrayDataSlabs(slab1, arrayDataSlab2)
				if slabComparisonError != nil {
					return false
				}
			}

		default:
			return assert.ObjectsAreEqual(slab1, slab2)
		}

		return true
	}

	if !slices.EqualFunc(vmSlabs, interpreterSlabs, equalSlab) {
		return nil, fmt.Errorf(
			"slabs do not match!\ninterpreter: %v\nvm: %v.\n%w",
			interpreterSlabs,
			vmSlabs,
			slabComparisonError,
		)
	}

	return vmResult, nil
}

func (i *CombinedInvokable) InvokeWithoutComparison(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	return i.VMInvokable.Invoke(functionName, arguments...)
}

func sortedSlabIDsFromStorage(storage interpreter.Storage) ([]atree.Slab, error) {
	storageItr, err := storage.SlabIterator()
	if err != nil {
		return nil, err
	}

	var slabs []atree.Slab
	for {
		_, slab := storageItr()
		if slab == nil {
			break
		}
		slabs = append(slabs, slab)
	}
	slices.SortFunc(slabs, compareSlabs)
	return slabs, nil
}

func compareMapDataSlabs(slab1 *atree.MapDataSlab, slab2 *atree.MapDataSlab) error {
	if slab1.Count() != slab2.Count() {
		fmt.Printf("Different count: %d vs %d\n", slab1.Count(), slab2.Count())
	}

	entries1 := make(map[string]mapEntry)
	err := slab1.Iterate(nil, newMapDataSlabIndexer(entries1))
	if err != nil {
		return fmt.Errorf("Error iterating slab 1: %w\n", err)
	}

	entries2 := make(map[string]mapEntry)
	err = slab2.Iterate(nil, newMapDataSlabIndexer(entries2))
	if err != nil {
		return fmt.Errorf("Error iterating slab 2: %w\n", err)
	}

	for encodedKey, entry1 := range entries1 { //nolint:maprange
		entry2, ok := entries2[encodedKey]
		if !ok {
			return fmt.Errorf("Key %q missing in slab 2\n", entry1.key)
		}

		value1 := entry1.value
		value2 := entry2.value

		err := compareStorable(value1, value2, entry1.key)
		if err != nil {
			return err
		}
	}

	for encodedKey, entry2 := range entries2 { //nolint:maprange
		_, ok := entries1[encodedKey]
		if !ok {
			return fmt.Errorf("Key %q missing in slab 1\n", entry2.key)
		}
	}

	return nil
}

func compareArrayDataSlabs(slab1 *atree.ArrayDataSlab, slab2 *atree.ArrayDataSlab) error {
	elements1 := slab1.ChildStorables()
	elements2 := slab2.ChildStorables()

	if len(elements1) != len(elements2) {
		return fmt.Errorf(
			"Different count: %d vs %d\n",
			len(elements1),
			len(elements2),
		)
	}

	for index, element1 := range elements1 { //nolint:maprange
		element2 := elements2[index]
		err := compareStorable(element1, element2, index)
		if err != nil {
			return err
		}
	}

	return nil
}

func compareStorable(value1 atree.Storable, value2 atree.Storable, index any) error {
	mismatchError := func() error {
		return fmt.Errorf(
			"Different value for key %q: %q vs %q\n",
			index,
			value1,
			value2,
		)
	}

	switch value1 := value1.(type) {
	case *atree.MapDataSlab:
		if mapDataSlabValue2, ok := value2.(*atree.MapDataSlab); ok {
			err := compareMapDataSlabs(value1, mapDataSlabValue2)
			if err != nil {
				return err
			}
		}

	case *atree.ArrayDataSlab:
		if arrayDataSlab2, ok := value2.(*atree.ArrayDataSlab); ok {
			err := compareArrayDataSlabs(value1, arrayDataSlab2)
			if err != nil {
				return err
			}
		}

	case interpreter.NonStorable:
		// Non-storables doesn't get stored. So no need to compare.
		return nil

	default:
		if !bytes.Equal(encodeStorable(value1), encodeStorable(value2)) {
			return mismatchError()
		}
	}

	return nil
}

func encodeStorable(storable atree.Storable) []byte {
	var buf bytes.Buffer
	encoder := atree.NewEncoder(&buf, interpreter.CBOREncMode)
	err := storable.Encode(encoder)
	if err != nil {
		panic(fmt.Errorf("failed to encode storable: %w", err))
	}

	err = encoder.CBOR.Flush()
	if err != nil {
		panic(fmt.Errorf("failed to flush encoder: %w", err))
	}

	return buf.Bytes()
}

type mapEntry struct {
	key   atree.MapKey
	value atree.MapValue
}

func newMapDataSlabIndexer(entries map[string]mapEntry) func(key atree.MapKey, value atree.MapValue) error {
	return func(key atree.MapKey, value atree.MapValue) error {
		entries[string(encodeStorable(key))] = mapEntry{
			key:   key,
			value: value,
		}
		return nil
	}
}
