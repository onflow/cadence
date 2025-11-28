package test_utils

import (
	"fmt"
	"slices"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

var (
	compareSlabID = func(a, b atree.SlabID) int {
		return a.Compare(b)
	}

	equalSlabID = func(a, b atree.SlabID) bool {
		return a.Compare(b) == 0
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

	vmStorage := i.interpreter.Storage()
	interpreterStorage := i.interpreter.Storage()

	if vmStorage.Count() != interpreterStorage.Count() {
		return nil, fmt.Errorf(
			"storage count mismatch. interpreter: %d, vm: %d",
			interpreterStorage.Count(),
			vmStorage.Count(),
		)
	}

	// Collect VM slabs

	vmSlabIDs, err := sortedSlabIDsFromStorage(vmStorage)
	if err != nil {
		return nil, err
	}

	// Collect Interpreter slabs
	interpreterSlabIDs, err := sortedSlabIDsFromStorage(interpreterStorage)
	if err != nil {
		return nil, err
	}

	if !slices.EqualFunc(vmSlabIDs, interpreterSlabIDs, equalSlabID) {
		return nil, fmt.Errorf(
			"slab IDs does not match!\ninterpreter: %v\nvm: %v",
			interpreterSlabIDs,
			vmSlabIDs,
		)
	}

	return vmResult, nil
}

func sortedSlabIDsFromStorage(storage interpreter.Storage) ([]atree.SlabID, error) {
	storageItr, err := storage.SlabIterator()
	if err != nil {
		return nil, err
	}

	var slabIDs []atree.SlabID
	for {
		slabId, slab := storageItr()
		if slab == nil {
			break
		}
		slabIDs = append(slabIDs, slabId)
	}
	slices.SortFunc(slabIDs, compareSlabID)
	return slabIDs, nil
}

func (i *CombinedInvokable) InvokeWithoutComparison(functionName string, arguments ...interpreter.Value) (value interpreter.Value, err error) {
	return i.VMInvokable.Invoke(functionName, arguments...)
}
