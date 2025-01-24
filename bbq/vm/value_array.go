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

package vm

import (
	goerrors "errors"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type ArrayValue struct {
	Type             interpreter.ArrayStaticType
	array            *atree.Array
	isResourceKinded bool
	elementSize      uint
	isDestroyed      bool
}

var _ Value = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}

func NewArrayValue(
	config *Config,
	arrayType interpreter.ArrayStaticType,
	isResourceKinded bool,
	values ...Value,
) *ArrayValue {

	address := common.ZeroAddress

	var index int
	count := len(values)

	return NewArrayValueWithIterator(
		config,
		arrayType,
		isResourceKinded,
		address,
		func() Value {
			if index >= count {
				return nil
			}

			value := values[index]

			index++

			value = value.Transfer(
				config,
				atree.Address(address),
				true,
				nil,
			)

			return value
		},
	)
}

func NewArrayValueWithIterator(
	config *Config,
	arrayType interpreter.ArrayStaticType,
	isResourceKinded bool,
	address common.Address,
	values func() Value,
) *ArrayValue {
	constructor := func() *atree.Array {
		array, err := atree.NewArrayFromBatchData(
			config.Storage,
			atree.Address(address),
			arrayType,
			func() (atree.Value, error) {
				vmValue := values()
				value := VMValueToInterpreterValue(config, vmValue)
				return value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return array
	}

	return newArrayValueFromConstructor(arrayType, isResourceKinded, constructor)
}

func newArrayValueFromConstructor(
	staticType interpreter.ArrayStaticType,
	isResourceKinded bool,
	constructor func() *atree.Array,
) *ArrayValue {

	elementSize := interpreter.ArrayElementSize(staticType)

	return newArrayValueFromAtreeArray(
		staticType,
		isResourceKinded,
		elementSize,
		constructor(),
	)
}

func newArrayValueFromAtreeArray(
	staticType interpreter.ArrayStaticType,
	isResourceKinded bool,
	elementSize uint,
	atreeArray *atree.Array,
) *ArrayValue {
	return &ArrayValue{
		Type:             staticType,
		array:            atreeArray,
		elementSize:      elementSize,
		isResourceKinded: isResourceKinded,
	}
}

func (v *ArrayValue) isValue() {
	panic("implement me")
}

func (v *ArrayValue) StaticType(*Config) StaticType {
	return v.Type
}

func (v *ArrayValue) Transfer(config *Config, address atree.Address, remove bool, storable atree.Storable) Value {

	storage := config.Storage

	array := v.array

	//currentValueID := v.ValueID()

	//if preventTransfer == nil {
	//	preventTransfer = map[atree.ValueID]struct{}{}
	//} else if _, ok := preventTransfer[currentValueID]; ok {
	//	panic(RecursiveTransferError{
	//		LocationRange: locationRange,
	//	})
	//}
	//preventTransfer[currentValueID] = struct{}{}
	//defer delete(preventTransfer, currentValueID)

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded()

	if needsStoreTo || !isResourceKinded {

		// Use non-readonly iterator here because iterated
		// value can be removed if remove parameter is true.
		iterator, err := v.array.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		array, err = atree.NewArrayFromBatchData(
			config.Storage,
			address,
			v.array.Type(),
			func() (atree.Value, error) {
				value, err := iterator.Next()
				if err != nil {
					return nil, err
				}
				if value == nil {
					return nil, nil
				}

				element := interpreter.MustConvertStoredValue(config.MemoryGauge, value)

				// TODO: Transfer before returning.
				return element, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.array.PopIterate(func(valueStorable atree.Storable) {
				RemoveReferencedSlab(storage, valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			//interpreter.maybeValidateAtreeValue(v.array)
			//if hasNoParentContainer {
			//	interpreter.maybeValidateAtreeStorage()
			//}

			RemoveReferencedSlab(storage, storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		invalidateReferencedResources(config, v)

		v.array = nil
	}

	res := newArrayValueFromAtreeArray(
		v.Type,
		isResourceKinded,
		v.elementSize,
		array,
	)

	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *ArrayValue) String() string {
	panic("implement me")
}

func (v *ArrayValue) ValueID() atree.ValueID {
	return v.array.ValueID()
}

func (v *ArrayValue) SlabID() atree.SlabID {
	return v.array.SlabID()
}

func (v *ArrayValue) StorageAddress() atree.Address {
	return v.array.Address()
}

func (v *ArrayValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *ArrayValue) IsResourceKinded() bool {
	return v.isResourceKinded
}

func (v *ArrayValue) IsReferenceTrackedResourceKindedValue() {}

func (v *ArrayValue) IsStaleResource() bool {
	return v.array == nil && v.IsResourceKinded()
}

func (v *ArrayValue) Count() int {
	return int(v.array.Count())
}

func (v *ArrayValue) Get(config *Config, index int) Value {

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Get function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(interpreter.ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	storedValue, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)
		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(
		config.MemoryGauge,
		config.Storage,
		storedValue,
	)
}

func (v *ArrayValue) handleIndexOutOfBoundsError(err error, index int) {
	var indexOutOfBoundsError *atree.IndexOutOfBoundsError
	if goerrors.As(err, &indexOutOfBoundsError) {
		panic(interpreter.ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}
}

func (v *ArrayValue) Set(config *Config, index int, element Value) {

	// TODO:
	//interpreter.validateMutation(v.ValueID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Set function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(interpreter.ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	// TODO:
	//interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	element = element.Transfer(
		config,
		v.array.Address(),
		true,
		nil,
	)

	storableElement := VMValueToInterpreterValue(config, element)

	existingStorable, err := v.array.Set(uint64(index), storableElement)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}

	//interpreter.maybeValidateAtreeValue(v.array)
	//interpreter.maybeValidateAtreeStorage()

	existingValue := StoredValue(config.MemoryGauge, existingStorable, config.Storage)
	_ = existingValue

	//interpreter.checkResourceLoss(existingValue, locationRange)

	//existingValue.DeepRemove(interpreter, true) // existingValue is standalone because it was overwritten in parent container.

	RemoveReferencedSlab(config.Storage, existingStorable)
}

func (v *ArrayValue) Iterate(
	config *Config,
	f func(element Value) (resume bool),
	transferElements bool,
) {
	v.iterate(
		config,
		v.array.Iterate,
		f,
		transferElements,
	)
}

// IterateReadOnlyLoaded iterates over all LOADED elements of the array.
// DO NOT perform storage mutations in the callback!
func (v *ArrayValue) IterateReadOnlyLoaded(
	config *Config,
	f func(element Value) (resume bool),
) {
	const transferElements = false

	v.iterate(
		config,
		v.array.IterateReadOnlyLoadedValues,
		f,
		transferElements,
	)
}

func (v *ArrayValue) iterate(
	config *Config,
	atreeIterate func(fn atree.ArrayIterationFunc) error,
	f func(element Value) (resume bool),
	transferElements bool,
) {
	iterate := func() {
		err := atreeIterate(func(element atree.Value) (resume bool, err error) {
			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value
			elementValue := MustConvertStoredValue(config, config, element)
			checkInvalidatedResourceOrResourceReference(elementValue)

			if transferElements {
				// Each element must be transferred before passing onto the function.
				elementValue = elementValue.Transfer(
					config,
					atree.Address{},
					false,
					nil,
				)
			}

			resume = f(elementValue)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	iterate()
	//interpreter.withMutationPrevention(v.ValueID(), iterate)
}
