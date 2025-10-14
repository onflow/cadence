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

func NewArrayValue(
	context ArrayCreationContext,
	arrayType ArrayStaticType,
	address common.Address,
	values ...Value,
) *ArrayValue {

	var index int
	count := len(values)

	return NewArrayValueWithIterator(
		context,
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
				context,
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
	context ArrayCreationContext,
	arrayType ArrayStaticType,
	address common.Address,
	countOverestimate uint64,
	values func() Value,
) *ArrayValue {
	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindCreateArrayValue,
			Intensity: 1,
		},
	)

	var v *ArrayValue

	if TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function,
			// if there was no error during construction
			if v == nil {
				return
			}

			valueID := v.ValueID().String()
			var typeID string
			if v.Type != nil {
				typeID = string(v.Type.ID())
			}

			context.ReportArrayValueConstructTrace(
				valueID,
				typeID,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() (array *atree.Array) {

		if TracingEnabled {
			startTime := time.Now()

			defer func() {
				// NOTE: in defer, as array is only initialized at the end of the function,
				// if there was no error during construction
				if array == nil {
					return
				}

				valueID := array.ValueID().String()
				var typeID string
				if arrayType != nil {
					typeID = string(arrayType.ID())
				}

				context.ReportAtreeNewArrayFromBatchDataTrace(
					valueID,
					typeID,
					time.Since(startTime),
				)
			}()
		}

		var err error
		array, err = atree.NewArrayFromBatchData(
			context.Storage(),
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
	v = newArrayValueFromConstructor(context, arrayType, countOverestimate, constructor)
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
var _ atree.WrapperValue = &ArrayValue{}
var _ EquatableValue = &ArrayValue{}
var _ ValueIndexableValue = &ArrayValue{}
var _ MemberAccessibleValue = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}
var _ IterableValue = &ArrayValue{}
var _ atreeContainerBackedValue = &ArrayValue{}

func (*ArrayValue) IsValue() {}

func (*ArrayValue) isAtreeContainerBackedValue() {}

func (v *ArrayValue) Accept(context ValueVisitContext, visitor Visitor) {
	descend := visitor.VisitArrayValue(context, v)
	if !descend {
		return
	}

	v.Walk(
		context,
		func(element Value) {
			element.Accept(context, visitor)
		},
	)
}

func (v *ArrayValue) Iterate(
	context ValueTransferContext,
	f func(element Value) (resume bool),
	transferElements bool,
) {
	v.iterate(
		context,
		v.array.Iterate,
		f,
		transferElements,
	)
}

// IterateReadOnlyLoaded iterates over all LOADED elements of the array.
// DO NOT perform storage mutations in the callback!
func (v *ArrayValue) IterateReadOnlyLoaded(
	context ValueTransferContext,
	f func(element Value) (resume bool),
) {
	const transferElements = false

	v.iterate(
		context,
		v.array.IterateReadOnlyLoadedValues,
		f,
		transferElements,
	)
}

func (v *ArrayValue) iterate(
	context ValueTransferContext,
	atreeIterate func(fn atree.ArrayIterationFunc) error,
	f func(element Value) (resume bool),
	transferElements bool,
) {
	iterate := func() {
		err := atreeIterate(func(element atree.Value) (resume bool, err error) {
			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value
			elementValue := MustConvertStoredValue(context, element)
			CheckInvalidatedResourceOrResourceReference(elementValue, context)

			if transferElements {
				// Each element must be transferred before passing onto the function.
				elementValue = elementValue.Transfer(
					context,
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

	context.WithContainerMutationPrevention(v.ValueID(), iterate)
}

func (v *ArrayValue) Iterator(_ ValueStaticTypeContext) ValueIterator {
	valueID := v.array.ValueID()

	arrayIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &ArrayIterator{
		valueID:       valueID,
		atreeIterator: arrayIterator,
	}
}

func (v *ArrayValue) Walk(context ValueWalkContext, walkChild func(Value)) {
	v.Iterate(
		context,
		func(element Value) (resume bool) {
			walkChild(element)
			return true
		},
		false,
	)
}

func (v *ArrayValue) StaticType(_ ValueStaticTypeContext) StaticType {
	// TODO meter
	return v.Type
}

func (v *ArrayValue) IsImportable(context ValueImportableContext) bool {
	importable := true
	v.Iterate(
		context,
		func(element Value) (resume bool) {
			if !element.IsImportable(context) {
				importable = false
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
		false,
	)

	return importable
}

func (v *ArrayValue) isInvalidatedResource(context ValueStaticTypeContext) bool {
	return v.isDestroyed || (v.array == nil && v.IsResourceKinded(context))
}

func (v *ArrayValue) IsStaleResource(context ValueStaticTypeContext) bool {
	return v.array == nil && v.IsResourceKinded(context)
}

func (v *ArrayValue) Destroy(context ResourceDestructionContext) {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindDestroyArrayValue,
			Intensity: 1,
		},
	)

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.Type.ID())

		defer func() {
			context.ReportArrayValueDestroyTrace(
				valueID,
				typeID,
				time.Since(startTime),
			)
		}()
	}

	valueID := v.ValueID()

	context.WithResourceDestruction(
		valueID,
		func() {
			v.Walk(
				context,
				func(element Value) {
					maybeDestroy(context, element)
				},
			)
		},
	)

	v.isDestroyed = true

	InvalidateReferencedResources(context, v)

	v.array = nil
}

func (v *ArrayValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *ArrayValue) Concat(context ValueTransferContext, other *ArrayValue) Value {

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
		context,
		v.Type,
		common.ZeroAddress,
		v.array.Count()+other.array.Count(),
		func() Value {

			// Meter computation for iterating the two arrays.
			common.UseComputation(
				context,
				common.LoopComputationUsage,
			)

			var value Value

			if first {
				atreeValue, err := firstIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue == nil {
					first = false
				} else {
					value = MustConvertStoredValue(context, atreeValue)
				}
			}

			if !first {
				atreeValue, err := secondIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue != nil {
					value = MustConvertStoredValue(context, atreeValue)

					checkContainerMutation(context, elementType, value)
				}
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				context,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) GetKey(context ValueComparisonContext, key Value) Value {
	index := key.(NumberValue).ToInt()
	return v.Get(context, index)
}

func (v *ArrayValue) handleIndexOutOfBoundsError(err error, index int) {
	var indexOutOfBoundsError *atree.IndexOutOfBoundsError
	if goerrors.As(err, &indexOutOfBoundsError) {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}
}

func (v *ArrayValue) Get(gauge common.MemoryGauge, index int) Value {

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Get function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	storedValue, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(gauge, storedValue)
}

func (v *ArrayValue) SetKey(context ContainerMutationContext, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	v.Set(context, index, value)
}

func (v *ArrayValue) Set(context ContainerMutationContext, index int, element Value) {

	context.ValidateContainerMutation(v.ValueID())

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Set function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	checkContainerMutation(context, v.Type.ElementType(), element)

	common.UseMemory(context, common.AtreeArrayElementOverhead)

	element = element.Transfer(
		context,
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
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.array)
	context.MaybeValidateAtreeStorage()

	existingValue := StoredValue(context, existingStorable, context.Storage())
	CheckResourceLoss(context, existingValue)
	existingValue.DeepRemove(context, true) // existingValue is standalone because it was overwritten in parent container.

	RemoveReferencedSlab(context, existingStorable)
}

func (v *ArrayValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *ArrayValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(NoOpStringContext{}, seenReferences)
}

func (v *ArrayValue) MeteredString(
	context ValueStringContext,
	seenReferences SeenReferences,
) string {
	// if n > 0:
	// len = open-bracket + close-bracket + ((n-1) comma+space)
	//     = 2 + 2n - 2
	//     = 2n
	// Always +2 to include empty array case (over estimate).
	// Each elements' string value is metered individually.
	common.UseMemory(context, common.NewRawStringMemoryUsage(v.Count()*2+2))

	values := make([]string, v.Count())

	i := 0

	v.Iterate(
		context,
		func(value Value) (resume bool) {
			// ok to not meter anything created as part of this iteration, since we will discard the result
			// upon creating the string
			values[i] = value.MeteredString(context, seenReferences)
			i++
			return true
		},
		false,
	)

	return format.Array(values)
}

func (v *ArrayValue) Append(context ValueTransferContext, element Value) {

	context.ValidateContainerMutation(v.ValueID())

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(context, dataSlabs)
	common.UseMemory(context, metaDataSlabs)
	common.UseMemory(context, common.AtreeArrayElementOverhead)

	checkContainerMutation(context, v.Type.ElementType(), element)

	element = element.Transfer(
		context,
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

	context.MaybeValidateAtreeValue(v.array)
	context.MaybeValidateAtreeStorage()
}

func (v *ArrayValue) AppendAll(context ValueTransferContext, other *ArrayValue) {
	other.Walk(
		context,
		func(value Value) {
			v.Append(context, value)
		},
	)
}

func (v *ArrayValue) InsertKey(context ContainerMutationContext, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	v.Insert(context, index, value)
}

func (v *ArrayValue) InsertWithoutTransfer(
	context ContainerMutationContext,
	index int,
	element Value,
) {
	context.ValidateContainerMutation(v.ValueID())

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Insert function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(context, dataSlabs)
	common.UseMemory(context, metaDataSlabs)
	common.UseMemory(context, common.AtreeArrayElementOverhead)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}
	context.MaybeValidateAtreeValue(v.array)
	context.MaybeValidateAtreeStorage()
}

func (v *ArrayValue) Insert(context ContainerMutationContext, index int, element Value) {

	address := v.array.Address()

	preventTransfer := map[atree.ValueID]struct{}{
		v.ValueID(): {},
	}

	element = element.Transfer(
		context,
		address,
		true,
		nil,
		preventTransfer,
		true, // standalone element doesn't have a parent container yet.
	)

	checkContainerMutation(context, v.Type.ElementType(), element)

	v.InsertWithoutTransfer(
		context,
		index,
		element,
	)
}

func (v *ArrayValue) RemoveKey(context ContainerMutationContext, key Value) Value {
	index := key.(NumberValue).ToInt()
	return v.Remove(context, index)
}

func (v *ArrayValue) RemoveWithoutTransfer(
	context ContainerMutationContext,
	index int,
) atree.Storable {

	context.ValidateContainerMutation(v.ValueID())

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Remove function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	storable, err := v.array.Remove(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.array)
	context.MaybeValidateAtreeStorage()

	return storable
}

func (v *ArrayValue) Remove(context ContainerMutationContext, index int) Value {
	storable := v.RemoveWithoutTransfer(context, index)

	value := StoredValue(context, storable, context.Storage())

	return value.Transfer(
		context,
		atree.Address{},
		true,
		storable,
		nil,
		true, // value is standalone because it was removed from parent container.
	)
}

func (v *ArrayValue) RemoveFirst(context ContainerMutationContext) Value {
	return v.Remove(context, 0)
}

func (v *ArrayValue) RemoveLast(context ContainerMutationContext) Value {
	return v.Remove(context, v.Count()-1)
}

func (v *ArrayValue) FirstIndex(interpreter ContainerMutationContext, needleValue Value) OptionalValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var counter int64
	var result bool
	v.Iterate(
		interpreter,
		func(element Value) (resume bool) {
			if needleEquatable.Equal(interpreter, element) {
				result = true
				// stop iteration
				return false
			}
			counter++
			// continue iteration
			return true
		},
		false,
	)

	if result {
		value := NewIntValueFromInt64(interpreter, counter)
		return NewSomeValueNonCopying(interpreter, value)
	}
	return NilOptionalValue
}

func (v *ArrayValue) Contains(
	context ContainerMutationContext,
	needleValue Value,
) BoolValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var result bool
	v.Iterate(
		context,
		func(element Value) (resume bool) {
			if needleEquatable.Equal(context, element) {
				result = true
				// stop iteration
				return false
			}
			// continue iteration
			return true
		},
		false,
	)

	return BoolValue(result)
}

func (v *ArrayValue) GetMember(context MemberAccessibleContext, name string) Value {
	switch name {
	case "length":
		return NewIntValueFromInt64(context, int64(v.Count()))
	}

	return context.GetMethod(v, name)
}

func (v *ArrayValue) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	switch name {
	case sema.ArrayTypeAppendFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayAppendFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayAppendFunction,
		)

	case sema.ArrayTypeAppendAllFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayAppendAllFunctionType(
				v.SemaType(context),
			),
			NativeArrayAppendAllFunction,
		)

	case sema.ArrayTypeConcatFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayConcatFunctionType(
				v.SemaType(context),
			),
			NativeArrayConcatFunction,
		)

	case sema.ArrayTypeInsertFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayInsertFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayInsertFunction,
		)

	case sema.ArrayTypeRemoveFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayRemoveFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayRemoveFunction,
		)

	case sema.ArrayTypeRemoveFirstFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayRemoveFirstFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayRemoveFirstFunction,
		)

	case sema.ArrayTypeRemoveLastFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayRemoveLastFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayRemoveLastFunction,
		)

	case sema.ArrayTypeFirstIndexFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayFirstIndexFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayFirstIndexFunction,
		)

	case sema.ArrayTypeContainsFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayContainsFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayContainsFunction,
		)

	case sema.ArrayTypeSliceFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArraySliceFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArraySliceFunction,
		)

	case sema.ArrayTypeReverseFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayReverseFunctionType(
				v.SemaType(context),
			),
			NativeArrayReverseFunction,
		)

	case sema.ArrayTypeFilterFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayFilterFunctionType(
				context,
				v.SemaType(context).ElementType(false),
			),
			NativeArrayFilterFunction,
		)

	case sema.ArrayTypeMapFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayMapFunctionType(
				context,
				v.SemaType(context),
			),
			NativeArrayMapFunction,
		)

	case sema.ArrayTypeReduceFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayReduceFunctionType(
				context,
				v.SemaType(context).ElementType(false),
			),
			NativeArrayReduceFunction,
		)

	case sema.ArrayTypeToVariableSizedFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayToVariableSizedFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayToVariableSizedFunction,
		)

	case sema.ArrayTypeToConstantSizedFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.ArrayToConstantSizedFunctionType(
				v.SemaType(context).ElementType(false),
			),
			NativeArrayToConstantSizedFunction,
		)
	}

	return nil
}

func (v *ArrayValue) RemoveMember(_ ValueTransferContext, _ string) Value {
	// Arrays have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) SetMember(_ ValueTransferContext, _ string, _ Value) bool {
	// Arrays have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) Count() int {
	return int(v.array.Count())
}

func (v *ArrayValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
) bool {

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.Type.ID())

		defer func() {
			context.ReportArrayValueConformsToStaticTypeTrace(
				valueID,
				typeID,
				time.Since(startTime),
			)
		}()
	}

	var elementType StaticType
	switch staticType := v.StaticType(context).(type) {
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
		context,
		func(element Value) (resume bool) {

			if !IsSubType(context, element.StaticType(context), elementType) {
				elementMismatch = true
				// stop iteration
				return false
			}

			if !element.ConformsToStaticType(context, results) {
				elementMismatch = true
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
		false,
	)

	return !elementMismatch
}

func (v *ArrayValue) Equal(context ValueComparisonContext, other Value) bool {
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
		value := v.Get(context, i)
		otherValue := otherArray.Get(context, i)

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(context, otherValue) {
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
	// NOTE: Need to change ArrayValue.UnwrapAtreeValue()
	// if ArrayValue is stored with wrapping.
	return v.array.Storable(storage, address, maxInlineSize)
}

func (v *ArrayValue) UnwrapAtreeValue() (atree.Value, uint64) {
	// Wrapper size is 0 because ArrayValue is stored as
	// atree.Array without any physical wrapping (see ArrayValue.Storable()).
	return v.array, 0
}

func (v *ArrayValue) IsReferenceTrackedResourceKindedValue() {}

func (v *ArrayValue) Transfer(
	context ValueTransferContext,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindTransferArrayValue,
			Intensity: uint64(v.Count()),
		},
	)

	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.Type.ID())

		defer func() {
			context.ReportArrayValueTransferTrace(
				valueID,
				typeID,
				time.Since(startTime),
			)
		}()
	}

	currentValueID := v.ValueID()

	if preventTransfer == nil {
		preventTransfer = map[atree.ValueID]struct{}{}
	} else if _, ok := preventTransfer[currentValueID]; ok {
		panic(&RecursiveTransferError{})
	}
	preventTransfer[currentValueID] = struct{}{}
	defer delete(preventTransfer, currentValueID)

	array := v.array

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(context)

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
		common.UseMemory(context, elementUsage)
		common.UseMemory(context, dataSlabs)
		common.UseMemory(context, metaDataSlabs)

		func() {

			if TracingEnabled {
				startTime := time.Now()

				defer func() {
					valueID := array.ValueID().String()
					typeID := string(v.Type.ID())

					context.ReportAtreeNewArrayFromBatchDataTrace(
						valueID,
						typeID,
						time.Since(startTime),
					)
				}()
			}

			array, err = atree.NewArrayFromBatchData(
				context.Storage(),
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

					element := MustConvertStoredValue(context, value).
						Transfer(
							context,
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
		}()

		if remove {
			err = v.array.PopIterate(func(storable atree.Storable) {
				RemoveReferencedSlab(context, storable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			context.MaybeValidateAtreeValue(v.array)
			if hasNoParentContainer {
				context.MaybeValidateAtreeStorage()
			}

			RemoveReferencedSlab(context, storable)
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

		InvalidateReferencedResources(context, v)

		v.array = nil
	}

	res := newArrayValueFromAtreeArray(
		context,
		v.Type,
		v.elementSize,
		array,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *ArrayValue) Clone(context ValueCloneContext) Value {
	array := newArrayValueFromConstructor(
		context,
		v.Type,
		v.array.Count(),
		func() *atree.Array {
			iterator, err := v.array.ReadOnlyIterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			array, err := atree.NewArrayFromBatchData(
				context.Storage(),
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

					element := MustConvertStoredValue(context, value).
						Clone(context)

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

func (v *ArrayValue) DeepRemove(context ValueRemoveContext, hasNoParentContainer bool) {
	if TracingEnabled {
		startTime := time.Now()

		valueID := v.ValueID().String()
		typeID := string(v.Type.ID())

		defer func() {
			context.ReportArrayValueDeepRemoveTrace(
				valueID,
				typeID,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.array.Storage

	err := v.array.PopIterate(func(storable atree.Storable) {
		value := StoredValue(context, storable, storage)
		value.DeepRemove(context, false) // existingValue is an element of v.array because it is from PopIterate() callback.
		RemoveReferencedSlab(context, storable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.array)
	if hasNoParentContainer {
		context.MaybeValidateAtreeStorage()
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

func (v *ArrayValue) SemaType(typeConverter TypeConverter) sema.ArrayType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = MustConvertStaticToSemaType(v.Type, typeConverter).(sema.ArrayType)
	}
	return v.semaType
}

func (v *ArrayValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *ArrayValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(context).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

func (v *ArrayValue) Slice(
	context ArrayCreationContext,
	from IntValue,
	to IntValue,
) Value {
	fromIndex := from.ToInt()
	toIndex := to.ToInt()

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.RangeIterator function will check the upper bound and report an atree.SliceOutOfBoundsError

	if fromIndex < 0 || toIndex < 0 {
		panic(&ArraySliceIndicesError{
			FromIndex: fromIndex,
			UpToIndex: toIndex,
			Size:      v.Count(),
		})
	}

	// Use ReadOnlyIterator here because new ArrayValue is created from elements copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyRangeIterator(uint64(fromIndex), uint64(toIndex))
	if err != nil {

		var sliceOutOfBoundsError *atree.SliceOutOfBoundsError
		if goerrors.As(err, &sliceOutOfBoundsError) {
			panic(&ArraySliceIndicesError{
				FromIndex: fromIndex,
				UpToIndex: toIndex,
				Size:      v.Count(),
			})
		}

		var invalidSliceIndexError *atree.InvalidSliceIndexError
		if goerrors.As(err, &invalidSliceIndexError) {
			panic(&InvalidSliceIndexError{
				FromIndex: fromIndex,
				UpToIndex: toIndex,
			})
		}

		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		context,
		NewVariableSizedStaticType(context, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(toIndex-fromIndex),
		func() Value {

			// Meter computation for iterating the array.
			common.UseComputation(context, common.LoopComputationUsage)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			var value Value
			if atreeValue != nil {
				value = MustConvertStoredValue(context, atreeValue)
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				context,
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
	context ArrayCreationContext,
) Value {
	count := v.Count()
	index := count - 1

	return NewArrayValueWithIterator(
		context,
		v.Type,
		common.ZeroAddress,
		uint64(count),
		func() Value {
			if index < 0 {
				return nil
			}

			// Meter computation for iterating the array.
			common.UseComputation(
				context,
				common.LoopComputationUsage,
			)

			value := v.Get(context, index)
			index--

			return value.Transfer(
				context,
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
	context InvocationContext,
	procedure FunctionValue,
) Value {

	elementType := v.SemaType(context).ElementType(false)

	argumentTypes := []sema.Type{elementType}

	procedureFunctionType := procedure.FunctionType(context)
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	// TODO: Use ReadOnlyIterator here if procedure doesn't change array elements.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		context,
		NewVariableSizedStaticType(context, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(v.Count()), // worst case estimation.
		func() Value {

			var value Value

			for {
				// Meter computation for iterating the array.
				common.UseComputation(
					context,
					common.LoopComputationUsage,
				)

				atreeValue, err := iterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				// Also handles the end of array case since iterator.Next() returns nil for that.
				if atreeValue == nil {
					return nil
				}

				value = MustConvertStoredValue(context, atreeValue)
				if value == nil {
					return nil
				}

				result := invokeFunctionValue(
					context,
					procedure,
					[]Value{value},
					argumentTypes,
					parameterTypes,
					returnType,
					nil,
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
				context,
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
	context InvocationContext,
	procedure FunctionValue,
) Value {

	elementType := v.SemaType(context).ElementType(false)

	argumentTypes := []sema.Type{elementType}

	procedureFunctionType := procedure.FunctionType(context)
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	returnStaticType := ConvertSemaToStaticType(context, returnType)

	var returnArrayStaticType ArrayStaticType
	switch v.Type.(type) {
	case *VariableSizedStaticType:
		returnArrayStaticType = NewVariableSizedStaticType(
			context,
			returnStaticType,
		)
	case *ConstantSizedStaticType:
		returnArrayStaticType = NewConstantSizedStaticType(
			context,
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
		context,
		returnArrayStaticType,
		common.ZeroAddress,
		uint64(v.Count()),
		func() Value {

			// Meter computation for iterating the array.
			common.UseComputation(
				context,
				common.LoopComputationUsage,
			)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(context, atreeValue)

			result := invokeFunctionValue(
				context,
				procedure,
				[]Value{value},
				argumentTypes,
				parameterTypes,
				returnType,
				nil,
			)

			return result.Transfer(
				context,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		},
	)
}

func (v *ArrayValue) Reduce(
	context InvocationContext,
	initial Value,
	procedure FunctionValue,
) Value {

	elementType := v.SemaType(context).ElementType(false)
	
	argumentTypes := []sema.Type{MustSemaTypeOfValue(initial, context), elementType}
	
	procedureFunctionType := procedure.FunctionType(context)
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	accumulator := initial

	// TODO: Use ReadOnlyIterator here if procedure doesn't change array values.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		// Meter computation for iterating the array.
		common.UseComputation(
			context,
			common.LoopComputationUsage,
		)

		atreeValue, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if atreeValue == nil {
			break
		}

		value := MustConvertStoredValue(context, atreeValue)

		result := invokeFunctionValue(
			context,
			procedure,
			[]Value{accumulator, value},
			argumentTypes,
			parameterTypes,
			returnType,
			nil,
		)

		accumulator = result.Transfer(
			context,
			atree.Address{},
			false,
			nil,
			nil,
			false, // value has a parent container because it is from iterator.
		)
	}

	return accumulator
}

func (v *ArrayValue) ForEach(
	context IterableValueForeachContext,
	_ sema.Type,
	function func(value Value) (resume bool),
	transferElements bool,
) {
	v.Iterate(context, function, transferElements)
}

func (v *ArrayValue) ToVariableSized(
	context ArrayCreationContext,
) Value {

	// Convert the constant-sized array type to a variable-sized array type.

	constantSizedType, ok := v.Type.(*ConstantSizedStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	variableSizedType := NewVariableSizedStaticType(
		context,
		constantSizedType.Type,
	)

	// Convert the array to a variable-sized array.

	// Use ReadOnlyIterator here because ArrayValue elements are copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		context,
		variableSizedType,
		common.ZeroAddress,
		uint64(v.Count()),
		func() Value {

			// Meter computation for iterating the array.
			common.UseComputation(
				context,
				common.LoopComputationUsage,
			)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(context, atreeValue)

			return value.Transfer(
				context,
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
	context ArrayCreationContext,
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
		context,
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
		context,
		constantSizedType,
		common.ZeroAddress,
		uint64(count),
		func() Value {

			// Meter computation for iterating the array.
			common.UseComputation(
				context,
				common.LoopComputationUsage,
			)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(context, atreeValue)

			return value.Transfer(
				context,
				atree.Address{},
				false,
				nil,
				nil,
				false,
			)
		},
	)

	// Return the constant-sized array as an optional value.

	return NewSomeValueNonCopying(context, constantSizedArray)
}

func (v *ArrayValue) SetType(staticType ArrayStaticType) {
	v.Type = staticType
	err := v.array.SetType(staticType)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (v *ArrayValue) Inlined() bool {
	return v.array.Inlined()
}

// Array iterator

type ArrayIterator struct {
	valueID       atree.ValueID
	atreeIterator atree.ArrayIterator
	next          atree.Value
}

var _ ValueIterator = &ArrayIterator{}

func (i *ArrayIterator) HasNext() bool {
	if i.next != nil {
		return true
	}

	var err error
	i.next, err = i.atreeIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return i.next != nil
}

func (i *ArrayIterator) Next(context ValueIteratorContext) Value {
	var atreeValue atree.Value
	if i.next != nil {
		// If there's already a `next` (i.e: `hasNext()` was called before this)
		// then use that.
		atreeValue = i.next

		// Clear the cached `next`.
		i.next = nil
	} else {
		var err error
		atreeValue, err = i.atreeIterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	if atreeValue == nil {
		return nil
	}

	// atree.Array iterator returns low-level atree.Value,
	// convert to high-level interpreter.Value
	return MustConvertStoredValue(context, atreeValue)
}

func (i *ArrayIterator) ValueID() (atree.ValueID, bool) {
	return i.valueID, true
}

// define all native functions for array type
var NativeArrayAppendFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		element := args[0]

		thisArray.Append(context, element)
		return Void
	},
)

var NativeArrayAppendAllFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		otherArray := AssertValueOfType[*ArrayValue](args[0])

		thisArray.AppendAll(context, otherArray)
		return Void
	},
)

var NativeArrayConcatFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		otherArray := AssertValueOfType[*ArrayValue](args[0])

		return thisArray.Concat(context, otherArray)
	},
)

var NativeArrayInsertFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		index := AssertValueOfType[NumberValue](args[0])
		element := args[1]

		thisArray.Insert(context, index.ToInt(), element)
		return Void
	},
)

var NativeArrayRemoveFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		index := AssertValueOfType[NumberValue](args[0])

		return thisArray.Remove(context, index.ToInt())
	},
)

var NativeArrayContainsFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		element := args[0]

		return thisArray.Contains(context, element)
	},
)

var NativeArraySliceFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		fromValue := AssertValueOfType[IntValue](args[0])
		toValue := AssertValueOfType[IntValue](args[1])

		return thisArray.Slice(context, fromValue, toValue)
	},
)

var NativeArrayReverseFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		return thisArray.Reverse(context)
	},
)

var NativeArrayFilterFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		funcValue := AssertValueOfType[FunctionValue](args[0])

		return thisArray.Filter(context, funcValue)
	},
)

var NativeArrayMapFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		funcValue := AssertValueOfType[FunctionValue](args[0])

		return thisArray.Map(context, funcValue)
	},
)

var NativeArrayReduceFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		initial := args[0]
		funcValue := AssertValueOfType[FunctionValue](args[1])

		return thisArray.Reduce(context, initial, funcValue)
	},
)

var NativeArrayToVariableSizedFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)

		return thisArray.ToVariableSized(context)
	},
)

var NativeArrayToConstantSizedFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		constantSizedArrayType, ok := typeArguments.NextStatic().(*ConstantSizedStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return thisArray.ToConstantSized(context, constantSizedArrayType.Size)
	},
)

var NativeArrayFirstIndexFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)
		element := args[0]

		return thisArray.FirstIndex(context, element)
	},
)

var NativeArrayRemoveFirstFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)

		return thisArray.RemoveFirst(context)
	},
)

var NativeArrayRemoveLastFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := AssertValueOfType[*ArrayValue](receiver)

		return thisArray.RemoveLast(context)
	},
)
