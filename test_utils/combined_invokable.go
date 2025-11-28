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

	equalSlab = func(slab1, slab2 atree.Slab) bool {
		if mapDataSlab1, ok := slab1.(*atree.MapDataSlab); ok {
			if mapDataSlab2, ok := slab2.(*atree.MapDataSlab); ok {
				compareMapDataSlabs(mapDataSlab1, mapDataSlab2)
				return true
			}
		}

		return assert.ObjectsAreEqual(slab1, slab2)
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

	_, _ = i.interpreter.Invoke(functionName, arguments...)

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

	if !slices.EqualFunc(vmSlabs, interpreterSlabs, equalSlab) {
		return nil, fmt.Errorf(
			"slab IDs does not match!\ninterpreter: %v\nvm: %v",
			interpreterSlabs,
			vmSlabs,
		)
	}

	return vmResult, nil
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

func (i *CombinedInvokable) InvokeWithoutComparison(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	return i.VMInvokable.Invoke(functionName, arguments...)
}

func compareMapDataSlabs(slab1 *atree.MapDataSlab, slab2 *atree.MapDataSlab) {

	if slab1.Count() != slab2.Count() {
		fmt.Printf("Different count: %d vs %d\n", slab1.Count(), slab2.Count())
	}

	entries1 := make(map[string]mapEntry)
	err := slab1.Iterate(nil, newMapDataSlabIndexer(entries1))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 1: %w\n", err))
	}

	entries2 := make(map[string]mapEntry)
	err = slab2.Iterate(nil, newMapDataSlabIndexer(entries2))
	if err != nil {
		panic(fmt.Errorf("Error iterating slab 2: %w\n", err))
	}

	for encodedKey, entry1 := range entries1 { //nolint:maprange
		entry2, ok := entries2[encodedKey]
		if !ok {
			fmt.Printf("Key %q missing in slab 2\n", entry1.key)
			continue
		}

		value1 := entry1.value
		value2 := entry2.value

		if mapDataSlabValue1, ok := value1.(*atree.MapDataSlab); ok {
			if mapDataSlabValue2, ok := value2.(*atree.MapDataSlab); ok {
				compareMapDataSlabs(mapDataSlabValue1, mapDataSlabValue2)
				continue
			}
		}

		if !bytes.Equal(encodeStorable(value1), encodeStorable(value2)) {
			fmt.Printf(
				"Different value for key %q: %q vs %q\n",
				entry1.key,
				value1,
				value2,
			)
		}
	}

	for encodedKey, entry2 := range entries2 { //nolint:maprange
		_, ok := entries1[encodedKey]
		if !ok {
			fmt.Printf("Key %q missing in slab 1\n", entry2.key)
		}
	}
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
