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

package interpreter

import (
	goerrors "errors"
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// ArrayValue

type ArrayValue struct {
	Type             ArrayStaticType
	semaType         sema.ArrayType
	array            *atree.Array
	isResourceKinded *bool
	elementSize      uint
	isDestroyed      bool
}

type ArrayValueIterator struct {
	atreeIterator atree.ArrayIterator
}

func (v *ArrayValue) Iterator(_ *Interpreter, _ LocationRange) ValueIterator {
	arrayIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return ArrayValueIterator{
		atreeIterator: arrayIterator,
	}
}

var _ ValueIterator = ArrayValueIterator{}

func (i ArrayValueIterator) Next(interpreter *Interpreter, _ LocationRange) Value {
	atreeValue, err := i.atreeIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if atreeValue == nil {
		return nil
	}

	// atree.Array iterator returns low-level atree.Value,
	// convert to high-level interpreter.Value
	return MustConvertStoredValue(interpreter, atreeValue)
}

func NewArrayValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	arrayType ArrayStaticType,
	address common.Address,
	values ...Value,
) *ArrayValue {

	var index int
	count := len(values)

	return NewArrayValueWithIterator(
		interpreter,
		arrayType,
		address,
		uint64(count),
		func() Value {
			if index >= count {
				return nil
			}

			value := values[index]

			index++

			value = value.Transfer(
				interpreter,
				locationRange,
				atree.Address(address),
				true,
				nil,
				nil,
				true, // standalone value doesn't have parent container.
			)

			return value
		},
	)
}

func NewArrayValueWithIterator(
	interpreter *Interpreter,
	arrayType ArrayStaticType,
	address common.Address,
	countOverestimate uint64,
	values func() Value,
) *ArrayValue {
	interpreter.ReportComputation(common.ComputationKindCreateArrayValue, 1)

	config := interpreter.SharedState.Config

	var v *ArrayValue

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function,
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			interpreter.reportArrayValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.Array {
		array, err := atree.NewArrayFromBatchData(
			config.Storage,
			atree.Address(address),
			arrayType,
			func() (atree.Value, error) {
				return values(), nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return array
	}
	// must assign to v here for tracing to work properly
	v = newArrayValueFromConstructor(interpreter, arrayType, countOverestimate, constructor)
	return v
}

func ArrayElementSize(staticType ArrayStaticType) uint {
	if staticType == nil {
		return 0
	}
	return staticType.ElementType().elementSize()
}

func newArrayValueFromConstructor(
	gauge common.MemoryGauge,
	staticType ArrayStaticType,
	countOverestimate uint64,
	constructor func() *atree.Array,
) *ArrayValue {

	elementSize := ArrayElementSize(staticType)

	elementUsage, dataSlabs, metaDataSlabs :=
		common.NewAtreeArrayMemoryUsages(countOverestimate, elementSize)
	common.UseMemory(gauge, elementUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	return newArrayValueFromAtreeArray(
		gauge,
		staticType,
		elementSize,
		constructor(),
	)
}

func newArrayValueFromAtreeArray(
	gauge common.MemoryGauge,
	staticType ArrayStaticType,
	elementSize uint,
	atreeArray *atree.Array,
) *ArrayValue {

	common.UseMemory(gauge, common.ArrayValueBaseMemoryUsage)

	return &ArrayValue{
		Type:        staticType,
		array:       atreeArray,
		elementSize: elementSize,
	}
}

var _ Value = &ArrayValue{}
var _ atree.Value = &ArrayValue{}
var _ EquatableValue = &ArrayValue{}
var _ ValueIndexableValue = &ArrayValue{}
var _ MemberAccessibleValue = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}
var _ IterableValue = &ArrayValue{}

func (*ArrayValue) isValue() {}

func (v *ArrayValue) Accept(interpreter *Interpreter, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitArrayValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(
		interpreter,
		func(element Value) {
			element.Accept(interpreter, visitor, locationRange)
		},
		locationRange,
	)
}

func (v *ArrayValue) Iterate(
	interpreter *Interpreter,
	f func(element Value) (resume bool),
	transferElements bool,
	locationRange LocationRange,
) {
	v.iterate(
		interpreter,
		v.array.Iterate,
		f,
		transferElements,
		locationRange,
	)
}

// IterateReadOnlyLoaded iterates over all LOADED elements of the array.
// DO NOT perform storage mutations in the callback!
func (v *ArrayValue) IterateReadOnlyLoaded(
	interpreter *Interpreter,
	f func(element Value) (resume bool),
	locationRange LocationRange,
) {
	const transferElements = false

	v.iterate(
		interpreter,
		v.array.IterateReadOnlyLoadedValues,
		f,
		transferElements,
		locationRange,
	)
}

func (v *ArrayValue) iterate(
	interpreter *Interpreter,
	atreeIterate func(fn atree.ArrayIterationFunc) error,
	f func(element Value) (resume bool),
	transferElements bool,
	locationRange LocationRange,
) {
	iterate := func() {
		err := atreeIterate(func(element atree.Value) (resume bool, err error) {
			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value
			elementValue := MustConvertStoredValue(interpreter, element)
			interpreter.checkInvalidatedResourceOrResourceReference(elementValue, locationRange)

			if transferElements {
				// Each element must be transferred before passing onto the function.
				elementValue = elementValue.Transfer(
					interpreter,
					locationRange,
					atree.Address{},
					false,
					nil,
					nil,
					false, // value has a parent container because it is from iterator.
				)
			}

			resume = f(elementValue)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	interpreter.withMutationPrevention(v.ValueID(), iterate)
}

func (v *ArrayValue) Walk(
	interpreter *Interpreter,
	walkChild func(Value),
	locationRange LocationRange,
) {
	v.Iterate(
		interpreter,
		func(element Value) (resume bool) {
			walkChild(element)
			return true
		},
		false,
		locationRange,
	)
}

func (v *ArrayValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *ArrayValue) IsImportable(inter *Interpreter, locationRange LocationRange) bool {
	importable := true
	v.Iterate(
		inter,
		func(element Value) (resume bool) {
			if !element.IsImportable(inter, locationRange) {
				importable = false
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
		false,
		locationRange,
	)

	return importable
}

func (v *ArrayValue) isInvalidatedResource(interpreter *Interpreter) bool {
	return v.isDestroyed || (v.array == nil && v.IsResourceKinded(interpreter))
}

func (v *ArrayValue) IsStaleResource(interpreter *Interpreter) bool {
	return v.array == nil && v.IsResourceKinded(interpreter)
}

func (v *ArrayValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyArrayValue, 1)

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueDestroyTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	valueID := v.ValueID()

	interpreter.withResourceDestruction(
		valueID,
		locationRange,
		func() {
			v.Walk(
				interpreter,
				func(element Value) {
					maybeDestroy(interpreter, locationRange, element)
				},
				locationRange,
			)
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, locationRange)

	v.array = nil
}

func (v *ArrayValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *ArrayValue) Concat(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) Value {

	first := true

	// Use ReadOnlyIterator here because new ArrayValue is created with elements copied (not removed) from original value.
	firstIterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	// Use ReadOnlyIterator here because new ArrayValue is created with elements copied (not removed) from original value.
	secondIterator, err := other.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	elementType := v.Type.ElementType()

	return NewArrayValueWithIterator(
		interpreter,
		v.Type,
		common.ZeroAddress,
		v.array.Count()+other.array.Count(),
		func() Value {

			// Meter computation for iterating the two arrays.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			var value Value

			if first {
				atreeValue, err := firstIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue == nil {
					first = false
				} else {
					value = MustConvertStoredValue(interpreter, atreeValue)
				}
			}

			if !first {
				atreeValue, err := secondIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue != nil {
					value = MustConvertStoredValue(interpreter, atreeValue)

					interpreter.checkContainerMutation(elementType, value, locationRange)
				}
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt(locationRange)
	return v.Get(interpreter, locationRange, index)
}

func (v *ArrayValue) handleIndexOutOfBoundsError(err error, index int, locationRange LocationRange) {
	var indexOutOfBoundsError *atree.IndexOutOfBoundsError
	if goerrors.As(err, &indexOutOfBoundsError) {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}
}

func (v *ArrayValue) Get(interpreter *Interpreter, locationRange LocationRange, index int) Value {

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Get function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	storedValue, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(interpreter, storedValue)
}

func (v *ArrayValue) SetKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt(locationRange)
	v.Set(interpreter, locationRange, index, value)
}

func (v *ArrayValue) Set(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	interpreter.validateMutation(v.ValueID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Set function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.ValueID]struct{}{
			v.ValueID(): {},
		},
		true, // standalone element doesn't have a parent container yet.
	)

	existingStorable, err := v.array.Set(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.array)
	interpreter.maybeValidateAtreeStorage()

	existingValue := StoredValue(interpreter, existingStorable, interpreter.Storage())
	interpreter.checkResourceLoss(existingValue, locationRange)
	existingValue.DeepRemove(interpreter, true) // existingValue is standalone because it was overwritten in parent container.

	interpreter.RemoveReferencedSlab(existingStorable)
}

func (v *ArrayValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *ArrayValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences, EmptyLocationRange)
}

func (v *ArrayValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
	// if n > 0:
	// len = open-bracket + close-bracket + ((n-1) comma+space)
	//     = 2 + 2n - 2
	//     = 2n
	// Always +2 to include empty array case (over estimate).
	// Each elements' string value is metered individually.
	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(v.Count()*2+2))

	values := make([]string, v.Count())

	i := 0

	v.Iterate(
		interpreter,
		func(value Value) (resume bool) {
			// ok to not meter anything created as part of this iteration, since we will discard the result
			// upon creating the string
			values[i] = value.MeteredString(interpreter, seenReferences, locationRange)
			i++
			return true
		},
		false,
		locationRange,
	)

	return format.Array(values)
}

func (v *ArrayValue) Append(interpreter *Interpreter, locationRange LocationRange, element Value) {

	interpreter.validateMutation(v.ValueID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)
	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.ValueID]struct{}{
			v.ValueID(): {},
		},
		true, // standalone element doesn't have a parent container yet.
	)

	err := v.array.Append(element)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.array)
	interpreter.maybeValidateAtreeStorage()
}

func (v *ArrayValue) AppendAll(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) {
	other.Walk(
		interpreter,
		func(value Value) {
			v.Append(interpreter, locationRange, value)
		},
		locationRange,
	)
}

func (v *ArrayValue) InsertKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt(locationRange)
	v.Insert(interpreter, locationRange, index, value)
}

func (v *ArrayValue) InsertWithoutTransfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	index int,
	element Value,
) {
	interpreter.validateMutation(v.ValueID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Insert function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)
	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
	interpreter.maybeValidateAtreeStorage()
}

func (v *ArrayValue) Insert(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	address := v.array.Address()

	preventTransfer := map[atree.ValueID]struct{}{
		v.ValueID(): {},
	}

	element = element.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
		true, // standalone element doesn't have a parent container yet.
	)

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	v.InsertWithoutTransfer(
		interpreter,
		locationRange,
		index,
		element,
	)
}

func (v *ArrayValue) RemoveKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt(locationRange)
	return v.Remove(interpreter, locationRange, index)
}

func (v *ArrayValue) RemoveWithoutTransfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	index int,
) atree.Storable {

	interpreter.validateMutation(v.ValueID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Remove function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	storable, err := v.array.Remove(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.array)
	interpreter.maybeValidateAtreeStorage()

	return storable
}

func (v *ArrayValue) Remove(interpreter *Interpreter, locationRange LocationRange, index int) Value {
	storable := v.RemoveWithoutTransfer(interpreter, locationRange, index)

	value := StoredValue(interpreter, storable, interpreter.Storage())

	return value.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		storable,
		nil,
		true, // value is standalone because it was removed from parent container.
	)
}

func (v *ArrayValue) RemoveFirst(interpreter *Interpreter, locationRange LocationRange) Value {
	return v.Remove(interpreter, locationRange, 0)
}

func (v *ArrayValue) RemoveLast(interpreter *Interpreter, locationRange LocationRange) Value {
	return v.Remove(interpreter, locationRange, v.Count()-1)
}

func (v *ArrayValue) FirstIndex(interpreter *Interpreter, locationRange LocationRange, needleValue Value) OptionalValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var counter int64
	var result bool
	v.Iterate(
		interpreter,
		func(element Value) (resume bool) {
			if needleEquatable.Equal(interpreter, locationRange, element) {
				result = true
				// stop iteration
				return false
			}
			counter++
			// continue iteration
			return true
		},
		false,
		locationRange,
	)

	if result {
		value := NewIntValueFromInt64(interpreter, counter)
		return NewSomeValueNonCopying(interpreter, value)
	}
	return NilOptionalValue
}

func (v *ArrayValue) Contains(
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue Value,
) BoolValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var result bool
	v.Iterate(
		interpreter,
		func(element Value) (resume bool) {
			if needleEquatable.Equal(interpreter, locationRange, element) {
				result = true
				// stop iteration
				return false
			}
			// continue iteration
			return true
		},
		false,
		locationRange,
	)

	return AsBoolValue(result)
}

func (v *ArrayValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "append":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayAppendFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				v.Append(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
				return Void
			},
		)

	case "appendAll":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayAppendAllFunctionType(
				v.SemaType(interpreter),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				otherArray, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				v.AppendAll(
					invocation.Interpreter,
					invocation.LocationRange,
					otherArray,
				)
				return Void
			},
		)

	case "concat":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayConcatFunctionType(
				v.SemaType(interpreter),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				otherArray, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.Concat(
					invocation.Interpreter,
					invocation.LocationRange,
					otherArray,
				)
			},
		)

	case "insert":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayInsertFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				inter := invocation.Interpreter
				locationRange := invocation.LocationRange

				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				element := invocation.Arguments[1]

				v.Insert(
					inter,
					locationRange,
					index,
					element,
				)
				return Void
			},
		)

	case "remove":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayRemoveFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				inter := invocation.Interpreter
				locationRange := invocation.LocationRange

				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				return v.Remove(
					inter,
					locationRange,
					index,
				)
			},
		)

	case "removeFirst":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayRemoveFirstFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				return v.RemoveFirst(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "removeLast":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayRemoveLastFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				return v.RemoveLast(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "firstIndex":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayFirstIndexFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				return v.FirstIndex(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "contains":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayContainsFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				return v.Contains(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "slice":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArraySliceFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				from, ok := invocation.Arguments[0].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				to, ok := invocation.Arguments[1].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Slice(
					invocation.Interpreter,
					from,
					to,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeReverseFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayReverseFunctionType(
				v.SemaType(interpreter),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				return v.Reverse(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeFilterFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayFilterFunctionType(
				interpreter,
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Filter(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)
			},
		)

	case sema.ArrayTypeMapFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayMapFunctionType(
				interpreter,
				v.SemaType(interpreter),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Map(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)
			},
		)

	case sema.ArrayTypeToVariableSizedFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayToVariableSizedFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				return v.ToVariableSized(
					interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeToConstantSizedFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ArrayToConstantSizedFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(v *ArrayValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				typeParameterPair := invocation.TypeParameterTypes.Oldest()
				if typeParameterPair == nil {
					panic(errors.NewUnreachableError())
				}

				ty := typeParameterPair.Value

				constantSizedArrayType, ok := ty.(*sema.ConstantSizedType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.ToConstantSized(
					interpreter,
					invocation.LocationRange,
					constantSizedArrayType.Size,
				)
			},
		)
	}

	return nil
}

func (v *ArrayValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	// Arrays have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	// Arrays have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) Count() int {
	return int(v.array.Count())
}

func (v *ArrayValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	config := interpreter.SharedState.Config

	count := v.Count()

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportArrayValueConformsToStaticTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	var elementType StaticType
	switch staticType := v.StaticType(interpreter).(type) {
	case *ConstantSizedStaticType:
		elementType = staticType.ElementType()
		if v.Count() != int(staticType.Size) {
			return false
		}
	case *VariableSizedStaticType:
		elementType = staticType.ElementType()
	default:
		return false
	}

	var elementMismatch bool

	v.Iterate(
		interpreter,
		func(element Value) (resume bool) {

			if !interpreter.IsSubType(element.StaticType(interpreter), elementType) {
				elementMismatch = true
				// stop iteration
				return false
			}

			if !element.ConformsToStaticType(
				interpreter,
				locationRange,
				results,
			) {
				elementMismatch = true
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
		false,
		locationRange,
	)

	return !elementMismatch
}

func (v *ArrayValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherArray, ok := other.(*ArrayValue)
	if !ok {
		return false
	}

	count := v.Count()

	if count != otherArray.Count() {
		return false
	}

	if v.Type == nil {
		if otherArray.Type != nil {
			return false
		}
	} else if otherArray.Type == nil ||
		!v.Type.Equal(otherArray.Type) {

		return false
	}

	for i := 0; i < count; i++ {
		value := v.Get(interpreter, locationRange, i)
		otherValue := otherArray.Get(interpreter, locationRange, i)

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}

	return true
}

func (v *ArrayValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return v.array.Storable(storage, address, maxInlineSize)
}

func (v *ArrayValue) IsReferenceTrackedResourceKindedValue() {}

func (v *ArrayValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {

	config := interpreter.SharedState.Config

	interpreter.ReportComputation(
		common.ComputationKindTransferArrayValue,
		uint(v.Count()),
	)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueTransferTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	currentValueID := v.ValueID()

	if preventTransfer == nil {
		preventTransfer = map[atree.ValueID]struct{}{}
	} else if _, ok := preventTransfer[currentValueID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentValueID] = struct{}{}
	defer delete(preventTransfer, currentValueID)

	array := v.array

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		// Use non-readonly iterator here because iterated
		// value can be removed if remove parameter is true.
		iterator, err := v.array.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementUsage, dataSlabs, metaDataSlabs := common.NewAtreeArrayMemoryUsages(
			v.array.Count(),
			v.elementSize,
		)
		common.UseMemory(interpreter, elementUsage)
		common.UseMemory(interpreter, dataSlabs)
		common.UseMemory(interpreter, metaDataSlabs)

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

				element := MustConvertStoredValue(interpreter, value).
					Transfer(
						interpreter,
						locationRange,
						address,
						remove,
						nil,
						preventTransfer,
						false, // value has a parent container because it is from iterator.
					)

				return element, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.array.PopIterate(interpreter.RemoveReferencedSlab)
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			interpreter.maybeValidateAtreeValue(v.array)
			if hasNoParentContainer {
				interpreter.maybeValidateAtreeStorage()
			}

			interpreter.RemoveReferencedSlab(storable)
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

		interpreter.invalidateReferencedResources(v, locationRange)

		v.array = nil
	}

	res := newArrayValueFromAtreeArray(
		interpreter,
		v.Type,
		v.elementSize,
		array,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *ArrayValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	array := newArrayValueFromConstructor(
		interpreter,
		v.Type,
		v.array.Count(),
		func() *atree.Array {
			iterator, err := v.array.ReadOnlyIterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			array, err := atree.NewArrayFromBatchData(
				config.Storage,
				v.StorageAddress(),
				v.array.Type(),
				func() (atree.Value, error) {
					value, err := iterator.Next()
					if err != nil {
						return nil, err
					}
					if value == nil {
						return nil, nil
					}

					element := MustConvertStoredValue(interpreter, value).
						Clone(interpreter)

					return element, nil
				},
			)
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			return array
		},
	)

	array.semaType = v.semaType
	array.isResourceKinded = v.isResourceKinded
	array.isDestroyed = v.isDestroyed

	return array
}

func (v *ArrayValue) DeepRemove(interpreter *Interpreter, hasNoParentContainer bool) {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueDeepRemoveTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.array.Storage

	err := v.array.PopIterate(func(storable atree.Storable) {
		value := StoredValue(interpreter, storable, storage)
		value.DeepRemove(interpreter, false) // existingValue is an element of v.array because it is from PopIterate() callback.
		interpreter.RemoveReferencedSlab(storable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.array)
	if hasNoParentContainer {
		interpreter.maybeValidateAtreeStorage()
	}
}

func (v *ArrayValue) SlabID() atree.SlabID {
	return v.array.SlabID()
}

func (v *ArrayValue) StorageAddress() atree.Address {
	return v.array.Address()
}

func (v *ArrayValue) ValueID() atree.ValueID {
	return v.array.ValueID()
}

func (v *ArrayValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *ArrayValue) SemaType(interpreter *Interpreter) sema.ArrayType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = interpreter.MustConvertStaticToSemaType(v.Type).(sema.ArrayType)
	}
	return v.semaType
}

func (v *ArrayValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *ArrayValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

func (v *ArrayValue) Slice(
	interpreter *Interpreter,
	from IntValue,
	to IntValue,
	locationRange LocationRange,
) Value {
	fromIndex := from.ToInt(locationRange)
	toIndex := to.ToInt(locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.RangeIterator function will check the upper bound and report an atree.SliceOutOfBoundsError

	if fromIndex < 0 || toIndex < 0 {
		panic(ArraySliceIndicesError{
			FromIndex:     fromIndex,
			UpToIndex:     toIndex,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	// Use ReadOnlyIterator here because new ArrayValue is created from elements copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyRangeIterator(uint64(fromIndex), uint64(toIndex))
	if err != nil {

		var sliceOutOfBoundsError *atree.SliceOutOfBoundsError
		if goerrors.As(err, &sliceOutOfBoundsError) {
			panic(ArraySliceIndicesError{
				FromIndex:     fromIndex,
				UpToIndex:     toIndex,
				Size:          v.Count(),
				LocationRange: locationRange,
			})
		}

		var invalidSliceIndexError *atree.InvalidSliceIndexError
		if goerrors.As(err, &invalidSliceIndexError) {
			panic(InvalidSliceIndexError{
				FromIndex:     fromIndex,
				UpToIndex:     toIndex,
				LocationRange: locationRange,
			})
		}

		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		NewVariableSizedStaticType(interpreter, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(toIndex-fromIndex),
		func() Value {

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			var value Value
			if atreeValue != nil {
				value = MustConvertStoredValue(interpreter, atreeValue)
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) Reverse(
	interpreter *Interpreter,
	locationRange LocationRange,
) Value {
	count := v.Count()
	index := count - 1

	return NewArrayValueWithIterator(
		interpreter,
		v.Type,
		common.ZeroAddress,
		uint64(count),
		func() Value {
			if index < 0 {
				return nil
			}

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			value := v.Get(interpreter, locationRange, index)
			index--

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is returned by Get().
			)
		},
	)
}

func (v *ArrayValue) Filter(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) Value {

	elementType := v.semaType.ElementType(false)

	argumentTypes := []sema.Type{elementType}

	procedureFunctionType := procedure.FunctionType()
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	// TODO: Use ReadOnlyIterator here if procedure doesn't change array elements.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		NewVariableSizedStaticType(interpreter, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(v.Count()), // worst case estimation.
		func() Value {

			var value Value

			for {
				// Meter computation for iterating the array.
				interpreter.ReportComputation(common.ComputationKindLoop, 1)

				atreeValue, err := iterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				// Also handles the end of array case since iterator.Next() returns nil for that.
				if atreeValue == nil {
					return nil
				}

				value = MustConvertStoredValue(interpreter, atreeValue)
				if value == nil {
					return nil
				}

				result := interpreter.invokeFunctionValue(
					procedure,
					[]Value{value},
					nil,
					argumentTypes,
					parameterTypes,
					returnType,
					nil,
					locationRange,
				)

				shouldInclude, ok := result.(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// We found the next entry of the filtered array.
				if shouldInclude {
					break
				}
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) Map(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) Value {

	elementType := v.semaType.ElementType(false)

	argumentTypes := []sema.Type{elementType}

	procedureFunctionType := procedure.FunctionType()
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	returnStaticType := ConvertSemaToStaticType(interpreter, returnType)

	var returnArrayStaticType ArrayStaticType
	switch v.Type.(type) {
	case *VariableSizedStaticType:
		returnArrayStaticType = NewVariableSizedStaticType(
			interpreter,
			returnStaticType,
		)
	case *ConstantSizedStaticType:
		returnArrayStaticType = NewConstantSizedStaticType(
			interpreter,
			returnStaticType,
			int64(v.Count()),
		)
	default:
		panic(errors.NewUnreachableError())
	}

	// TODO: Use ReadOnlyIterator here if procedure doesn't change map values.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		returnArrayStaticType,
		common.ZeroAddress,
		uint64(v.Count()),
		func() Value {

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(interpreter, atreeValue)

			result := interpreter.invokeFunctionValue(
				procedure,
				[]Value{value},
				nil,
				argumentTypes,
				parameterTypes,
				returnType,
				nil,
				locationRange,
			)

			return result.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) ForEach(
	interpreter *Interpreter,
	_ sema.Type,
	function func(value Value) (resume bool),
	transferElements bool,
	locationRange LocationRange,
) {
	v.Iterate(interpreter, function, transferElements, locationRange)
}

func (v *ArrayValue) ToVariableSized(
	interpreter *Interpreter,
	locationRange LocationRange,
) Value {

	// Convert the constant-sized array type to a variable-sized array type.

	constantSizedType, ok := v.Type.(*ConstantSizedStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	variableSizedType := NewVariableSizedStaticType(
		interpreter,
		constantSizedType.Type,
	)

	// Convert the array to a variable-sized array.

	// Use ReadOnlyIterator here because ArrayValue elements are copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		variableSizedType,
		common.ZeroAddress,
		uint64(v.Count()),
		func() Value {

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(interpreter, atreeValue)

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false,
			)
		},
	)
}

func (v *ArrayValue) ToConstantSized(
	interpreter *Interpreter,
	locationRange LocationRange,
	expectedConstantSizedArraySize int64,
) OptionalValue {

	// Ensure the array has the expected size.

	count := v.Count()

	if int64(count) != expectedConstantSizedArraySize {
		return NilOptionalValue
	}

	// Convert the variable-sized array type to a constant-sized array type.

	variableSizedType, ok := v.Type.(*VariableSizedStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	constantSizedType := NewConstantSizedStaticType(
		interpreter,
		variableSizedType.Type,
		expectedConstantSizedArraySize,
	)

	// Convert the array to a constant-sized array.

	// Use ReadOnlyIterator here because ArrayValue elements are copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	constantSizedArray := NewArrayValueWithIterator(
		interpreter,
		constantSizedType,
		common.ZeroAddress,
		uint64(count),
		func() Value {

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(interpreter, atreeValue)

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false,
			)
		},
	)

	// Return the constant-sized array as an optional value.

	return NewSomeValueNonCopying(interpreter, constantSizedArray)
}

func (v *ArrayValue) SetType(staticType ArrayStaticType) {
	v.Type = staticType
	err := v.array.SetType(staticType)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}
