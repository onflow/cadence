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

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

type VMInvokable struct {
	vmInstance *vm.VM
	*vm.Context
	elaboration *compiler.DesugaredElaboration

	// For slabs comparison
	interpreter *interpreter.Interpreter
}

var _ Invokable = &VMInvokable{}

func NewVMInvokable(
	vmInstance *vm.VM,
	elaboration *compiler.DesugaredElaboration,
) *VMInvokable {
	return &VMInvokable{
		vmInstance:  vmInstance,
		Context:     vmInstance.Context(),
		elaboration: elaboration,
	}
}

func (v *VMInvokable) Invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, vmError error) {
	vmResult, vmError := v.invoke(functionName, arguments...)

	// If the test was set up without the need for comparing slabs,
	// then return only the VM result.
	if v.interpreter == nil {
		return vmResult, vmError
	}

	// Invoke the interpreter, even if there are errors from VM.
	// This is because even an erroneous program could have side effects
	// on slab creation.
	_, interpreterError := v.interpreter.Invoke(functionName, arguments...)

	if vmError != nil {
		return nil, vmError
	}

	if interpreterError != nil {
		return nil, interpreterError
	}

	vmStorage := v.Storage()
	interpreterStorage := v.interpreter.Storage()

	// Collect VM slabs
	vmSlabs, vmError := sortedSlabIDsFromStorage(vmStorage)
	if vmError != nil {
		return nil, vmError
	}

	// Collect Interpreter slabs
	interpreterSlabs, vmError := sortedSlabIDsFromStorage(interpreterStorage)
	if vmError != nil {
		return nil, vmError
	}

	var slabComparisonError error

	equalSlab := func(slab1, slab2 atree.Slab) bool {
		switch slab1 := slab1.(type) {
		case *atree.MapDataSlab:
			if mapDataSlab2, ok := slab2.(*atree.MapDataSlab); ok {
				slabComparisonError = compareMapDataSlabs(slab1, mapDataSlab2)
				return slabComparisonError == nil
			}

		case *atree.ArrayDataSlab:
			if arrayDataSlab2, ok := slab2.(*atree.ArrayDataSlab); ok {
				slabComparisonError = compareArrayDataSlabs(slab1, arrayDataSlab2)
				return slabComparisonError == nil
			}

		default:
			return assert.ObjectsAreEqual(slab1, slab2)
		}

		return false
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

func (v *VMInvokable) invoke(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	value, err = v.vmInstance.InvokeExternally(functionName, arguments...)

	// Reset the VM after a function invocation,
	// so the same vm can be re-used for subsequent invocation.
	v.vmInstance.Reset()

	return
}

func (v *VMInvokable) InvokeTransaction(arguments []interpreter.Value, signers ...interpreter.Value) (err error) {
	err = v.vmInstance.InvokeTransaction(arguments, signers...)

	// Reset the VM after a function invocation,
	// so the same vm can be re-used for subsequent invocation.
	v.vmInstance.Reset()

	return
}

func (v *VMInvokable) GetGlobal(name string) interpreter.Value {
	return v.vmInstance.Global(name)
}

func (v *VMInvokable) GetGlobalType(name string) (*sema.Variable, bool) {
	return v.elaboration.GetGlobalType(name)
}

func (v *VMInvokable) InitializeContract(contractName string, arguments ...interpreter.Value) (*interpreter.CompositeValue, error) {
	return v.vmInstance.InitializeContract(contractName, arguments...)
}

func compareSlabs(a, b atree.Slab) int {
	return a.SlabID().Compare(b.SlabID())
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
