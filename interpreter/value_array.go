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

	// valueID is the atree value ID captured at construction time.
	// It is used for reference tracking and invalidation, and must remain
	// stable even if the array's runtime root slab ID changes.
	// See the equivalent field on CompositeValue for the underlying scenario.
	valueID atree.ValueID
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
) (v *ArrayValue) {

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindCreateArrayValue,
			Intensity: 1,
		},
	)

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

	return newArrayValueFromConstructor(
		context,
		arrayType,
		countOverestimate,
		func() (array *atree.Array) {

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

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeArrayBatchConstruction,
					Intensity: countOverestimate,
				},
			)

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
		},
	)
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
		valueID:     atreeArray.ValueID(),
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

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeArrayReadIteration,
					Intensity: 1,
				},
			)

			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value
			elementValue := MustConvertStoredValue(context, element)
			CheckInvalidatedValueOrValueReference(elementValue, context)

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

func (v *ArrayValue) Iterator(context ValueStaticTypeContext) ValueIterator {
	return NewArrayIterator(context, v)
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

func (v *ArrayValue) Concat(
	context ValueTransferContext,
	other *ArrayValue,
	accessedType sema.Type,
) Value {

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

	// `other`'s elements are checked against the receiver's declared element
	// type (param type stayed at the declared type — the caller provided it).
	otherElementType := v.Type.ElementType()

	// Cascade outer authorization into the result element type, matching
	// sema's ArrayConcatFunctionType.
	resultElementSemaType := v.SemaType(context).ElementType(false)
	resultElementSemaType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, resultElementSemaType, false)
	resultElementStaticType := ConvertSemaToStaticType(context, resultElementSemaType)

	newCount := v.array.Count() + other.array.Count()

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: newCount,
		},
	)

	return NewArrayValueWithIterator(
		context,
		NewVariableSizedStaticType(context, resultElementStaticType),
		common.ZeroAddress,
		newCount,
		func() Value {

			var value Value

			if first {
				// Computation was already metered above

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
				// Computation was already metered above

				atreeValue, err := secondIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue != nil {
					value = MustConvertStoredValue(context, atreeValue)

					checkContainerMutation(context, otherElementType, value)
				}
			}

			if value == nil {
				return nil
			}

			if asReference {
				value = getReferenceValue(context, value, resultElementSemaType)
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

func (v *ArrayValue) GetKey(context ContainerReadContext, key Value) Value {
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

func (v *ArrayValue) Get(context ContainerReadContext, index int) Value {

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Get function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(&ArrayIndexOutOfBoundsError{
			Index: index,
			Size:  v.Count(),
		})
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayGet,
			Intensity: 1,
		},
	)

	storedValue, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}

	result := MustConvertStoredValue(context, storedValue)

	return result
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

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArraySet,
			Intensity: 1,
		},
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

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayAppend,
			Intensity: 1,
		},
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
	v.InsertKeyWithMutationCheck(
		context,
		key,
		value,
		true,
	)
}

func (v *ArrayValue) InsertKeyWithMutationCheck(
	context ContainerMutationContext,
	key Value,
	value Value,
	checkMutation bool,
) {
	index := key.(NumberValue).ToInt()
	v.InsertWithMutationCheck(
		context,
		index,
		value,
		checkMutation,
	)
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

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayInsert,
			Intensity: 1,
		},
	)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index)

		panic(errors.NewExternalError(err))
	}
	context.MaybeValidateAtreeValue(v.array)
	context.MaybeValidateAtreeStorage()
}

func (v *ArrayValue) Insert(context ContainerMutationContext, index int, element Value) {
	v.InsertWithMutationCheck(
		context,
		index,
		element,
		true,
	)
}

func (v *ArrayValue) InsertWithMutationCheck(
	context ContainerMutationContext,
	index int,
	element Value,
	checkMutation bool,
) {

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

	if checkMutation {
		checkContainerMutation(context, v.Type.ElementType(), element)
	}

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

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayRemove,
			Intensity: 1,
		},
	)

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

func (v *ArrayValue) GetMember(
	context MemberAccessibleContext,
	name string,
	memberKind common.DeclarationKind,
	accessedReference ReferenceValue,
) Value {
	return GetMember(
		context,
		v,
		accessedReference,
		name,
		memberKind,
		func() Value {
			switch name {
			case "length":
				return NewIntValueFromInt64(context, int64(v.Count()))
			}
			return nil
		},
	)
}

func (v *ArrayValue) GetMethod(
	context MemberAccessibleContext,
	name string,
	accessedReference ReferenceValue,
) FunctionValue {

	arrayType := v.SemaType(context)

	var accessedType sema.Type
	if accessedReference != nil {
		accessedType = MustSemaTypeOfValue(accessedReference, context)
	} else {
		accessedType = arrayType
	}

	switch name {
	case sema.ArrayTypeAppendFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayAppendFunctionType(
				arrayType.ElementType(false),
			),
			NativeArrayAppendFunction,
		)

	case sema.ArrayTypeAppendAllFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayAppendAllFunctionType(
				arrayType,
			),
			NativeArrayAppendAllFunction,
		)

	case sema.ArrayTypeConcatFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayConcatFunctionType(
				context,
				accessedType,
				arrayType,
			),
			NativeArrayConcatFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function can read accessedType from it and cascade the
			// outer reference's authorization into the result element
			// type, matching sema.ArrayConcatFunctionType's return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeInsertFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayInsertFunctionType(
				arrayType.ElementType(false),
			),
			NativeArrayInsertFunction,
		)

	case sema.ArrayTypeRemoveFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayRemoveFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayRemoveFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function and any BBQ VM derivation can read accessedType
			// from the un-dereferenced receiver and apply the inner-
			// reference intersection in sema.ArrayRemoveFunctionType's
			// return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeRemoveFirstFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayRemoveFirstFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayRemoveFirstFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function and any BBQ VM derivation can read accessedType
			// from the un-dereferenced receiver and apply the inner-
			// reference intersection in
			// sema.ArrayRemoveFirstFunctionType's return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeRemoveLastFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayRemoveLastFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayRemoveLastFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function and any BBQ VM derivation can read accessedType
			// from the un-dereferenced receiver and apply the inner-
			// reference intersection in
			// sema.ArrayRemoveLastFunctionType's return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeFirstIndexFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayFirstIndexFunctionType(
				arrayType.ElementType(false),
			),
			NativeArrayFirstIndexFunction,
		)

	case sema.ArrayTypeContainsFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayContainsFunctionType(
				arrayType.ElementType(false),
			),
			NativeArrayContainsFunction,
		)

	case sema.ArrayTypeSliceFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArraySliceFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArraySliceFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function can read accessedType from it and cascade the
			// outer reference's authorization into the result element
			// type, matching sema.ArraySliceFunctionType's return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeReverseFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayReverseFunctionType(
				context,
				accessedType,
				arrayType,
			),
			NativeArrayReverseFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function can read accessedType from it and cascade the
			// outer reference's authorization into the result element
			// type, matching sema.ArrayReverseFunctionType's return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeFilterFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayFilterFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayFilterFunction,
		).
			// Filter function's parameter-type depends on whether
			// the receiver is a reference or a concrete array.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeMapFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayMapFunctionType(
				context,
				accessedType,
				arrayType,
			),
			NativeArrayMapFunction,
		).
			// Map function's parameter-type depends on whether
			// the receiver is a reference or a concrete array.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeToVariableSizedFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayToVariableSizedFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayToVariableSizedFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function can read accessedType from it and cascade the
			// outer reference's authorization into the result element
			// type, matching sema.ArrayToVariableSizedFunctionType's
			// return.
			WithDereferenceReceiver(false)

	case sema.ArrayTypeToConstantSizedFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			accessedReference,
			sema.ArrayToConstantSizedFunctionType(
				context,
				accessedType,
				arrayType.ElementType(false),
			),
			NativeArrayToConstantSizedFunction,
		).
			// Receiver is kept as-is (not dereferenced) so the native
			// function can read accessedType from it and cascade the
			// outer reference's authorization into the result element
			// type, matching sema.ArrayToConstantSizedFunctionType's
			// return.
			WithDereferenceReceiver(false)
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
	maxInlineSize uint32,
) (atree.Storable, error) {
	// NOTE: Need to change ArrayValue.UnwrapAtreeValue()
	// if ArrayValue is stored with wrapping.
	return v.array.Storable(storage, address, maxInlineSize)
}

func (v *ArrayValue) UnwrapAtreeValue() (atree.Value, uint32) {
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

	count := v.Count()

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindTransferArrayValue,
			Intensity: uint64(count),
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

		elementUsage, dataSlabs, metaDataSlabs := common.NewAtreeArrayMemoryUsages(
			v.array.Count(),
			v.elementSize,
		)
		common.UseMemory(context, elementUsage)
		common.UseMemory(context, dataSlabs)
		common.UseMemory(context, metaDataSlabs)

		// Check if atree.Array can be copied using v.array.CopyNonRefSimple():
		// - Use the fast path that looks at the array element type by calling canCopyNonRefSimpleForType().
		// - If the fast path fails, then look at the array element data by calling v.array.CanCopyNonRefSimple().

		isSingleSlabCopyableArrayType := v.array.IsWithinSingleSlab() && canCopyNonRefSimpleForType(v.Type.ElementType())
		canCopyNonRefSimple := isSingleSlabCopyableArrayType || v.array.CanCopyNonRefSimple()

		if canCopyNonRefSimple {

			func() {

				if TracingEnabled {
					startTime := time.Now()

					defer func() {
						valueID := array.ValueID().String()
						typeID := string(v.Type.ID())

						context.ReportAtreeNewArraySingleSlabTrace(
							valueID,
							typeID,
							time.Since(startTime),
						)
					}()
				}

				common.UseComputation(
					context,
					common.ComputationUsage{
						Kind:      common.ComputationKindAtreeArraySingleSlabConstruction,
						Intensity: uint64(count),
					},
				)

				copiedArray, err := v.array.CopyNonRefSimple(address)
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				array = copiedArray
			}()

		} else {
			// Use non-readonly iterator here because iterated
			// value can be removed if remove parameter is true.
			iterator, err := v.array.Iterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

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

				common.UseComputation(
					context,
					common.ComputationUsage{
						Kind:      common.ComputationKindAtreeArrayBatchConstruction,
						Intensity: uint64(count),
					},
				)

				common.UseComputation(
					context,
					common.ComputationUsage{
						Kind:      common.ComputationKindAtreeArrayReadIteration,
						Intensity: uint64(count),
					},
				)

				array, err = atree.NewArrayFromBatchData(
					context.Storage(),
					address,
					v.array.Type(),
					func() (atree.Value, error) {

						// Computation was already metered above

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
		}

		if remove {

			common.UseComputation(
				context,
				common.ComputationUsage{
					Kind:      common.ComputationKindAtreeArrayPopIteration,
					Intensity: v.array.Count(),
				},
			)

			err := v.array.PopIterate(func(storable atree.Storable) {
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

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayPopIteration,
			Intensity: v.array.Count(),
		},
	)

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
	return v.valueID
}

// LiveValueID returns the underlying atree array's current value ID.
// In contrast to ValueID, which returns a stable value ID cached at
// construction, LiveValueID reflects mutations to the atree array's root,
// including slab ID reassignments caused by splits triggered through other
// ArrayValue instances wrapping the same underlying atree array.
// Intended for testing only; production code must use ValueID for resource
// tracking and invalidation.
func (v *ArrayValue) LiveValueID() atree.ValueID {
	return v.array.ValueID()
}

// isStaleAtreeView reports whether this wrapper has been displaced by a
// structural change (slab split/merge/promotion) that was performed through
// a sibling wrapper sharing the same underlying slab tree. See the
// `AtreeBackedValue` interface and `InvalidatedContainerViewError` for the full
// context. Detected uses of a stale wrapper are rejected centrally in
// `CheckInvalidatedValueOrValueReference`.
func (v *ArrayValue) isStaleAtreeView() bool {
	return v.array.ValueID() != v.valueID
}

func (v *ArrayValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *ArrayValue) SemaType(typeConverter TypeConverter) sema.ArrayType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = typeConverter.SemaTypeFromStaticType(v.Type).(sema.ArrayType)
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
	accessedType sema.Type,
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

	newCount := uint64(toIndex - fromIndex)

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: newCount,
		},
	)

	// Cascade outer authorization into the result element type, matching
	// sema's ArraySliceFunctionType: when sliced through a reference,
	// elements are exposed as references (with auths intersected). Without
	// this, the new array's declared element type would mismatch its actual
	// contents.
	elementType := v.SemaType(context).ElementType(false)
	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)
	resultElementStaticType := ConvertSemaToStaticType(context, elementType)

	return NewArrayValueWithIterator(
		context,
		NewVariableSizedStaticType(context, resultElementStaticType),
		common.ZeroAddress,
		newCount,
		func() Value {

			// Computation was already metered above

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

			if asReference {
				value = getReferenceValue(context, value, elementType)
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
	accessedType sema.Type,
) Value {
	count := v.Count()
	index := count - 1

	// Cascade outer authorization into the result element type, matching
	// sema's ArrayReverseFunctionType.
	elementType := v.SemaType(context).ElementType(false)
	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)

	// reverse() preserves the array shape (variable- or constant-sized),
	// so reuse v.Type's shape but with the cascaded element type.
	resultElementStaticType := ConvertSemaToStaticType(context, elementType)
	var resultStaticType ArrayStaticType
	switch t := v.Type.(type) {
	case *VariableSizedStaticType:
		resultStaticType = NewVariableSizedStaticType(context, resultElementStaticType)
	case *ConstantSizedStaticType:
		resultStaticType = NewConstantSizedStaticType(context, resultElementStaticType, t.Size)
	default:
		panic(errors.NewUnreachableError())
	}

	return NewArrayValueWithIterator(
		context,
		resultStaticType,
		common.ZeroAddress,
		uint64(count),
		func() Value {
			if index < 0 {
				return nil
			}

			value := v.Get(context, index)
			index--

			if asReference {
				value = getReferenceValue(context, value, elementType)
			}

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
	accessedType sema.Type,
) Value {

	elementType := v.SemaType(context).ElementType(false)

	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)

	argumentType := elementType
	argumentTypes := []sema.Type{argumentType}

	procedureFunctionType := procedure.FunctionType(context)
	parameterTypes := procedureFunctionType.ParameterTypes()

	// TODO: Use ReadOnlyIterator here if procedure doesn't change array elements.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	resultElementStaticType := ConvertSemaToStaticType(context, argumentType)

	// Block mutations to v through any sibling wrapper while the lazy
	// iterator is being consumed by the user procedure. See ArrayValue.Map.
	var result Value
	context.WithContainerMutationPrevention(v.ValueID(), func() {
		result = NewArrayValueWithIterator(
			context,
			// NOTE: result is NOT v.Type, which could be a constant-sized array type.
			// Instead, result is always a variable-sized array type,
			// because filtering can change the number of elements in the array.
			NewVariableSizedStaticType(context, resultElementStaticType),
			common.ZeroAddress,
			uint64(v.Count()), // worst case estimation.
			func() Value {

				var value Value

				for {
					common.UseComputation(
						context,
						common.ComputationUsage{
							Kind:      common.ComputationKindAtreeArrayReadIteration,
							Intensity: 1,
						},
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

					if asReference {
						value = getReferenceValue(
							context,
							value,
							argumentType,
						)
					}

					procResult := invokeFunctionValue(
						context,
						procedure,
						[]Value{value},
						argumentTypes,
						parameterTypes,
						sema.BoolType,
						nil,
					)

					shouldInclude, ok := procResult.(BoolValue)
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
	})
	return result
}

func (v *ArrayValue) Map(
	context InvocationContext,
	procedure FunctionValue,
	accessedType sema.Type,
) Value {
	count := v.Count()

	elementType := v.SemaType(context).ElementType(false)

	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)

	argumentType := elementType
	argumentTypes := []sema.Type{argumentType}

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
			int64(count),
		)
	default:
		panic(errors.NewUnreachableError())
	}

	// TODO: Use ReadOnlyIterator here if procedure doesn't change map values.
	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: uint64(count),
		},
	)

	// Block mutations to v through any sibling wrapper while the lazy
	// iterator is being consumed by the user procedure. Without this,
	// a callback can mutate a sibling wrapper into a slab split/promote
	// and the iterator continues walking the orphaned root silently.
	var result Value
	context.WithContainerMutationPrevention(v.ValueID(), func() {
		result = NewArrayValueWithIterator(
			context,
			returnArrayStaticType,
			common.ZeroAddress,
			uint64(count),
			func() Value {

				// Computation was already metered above

				atreeValue, err := iterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue == nil {
					return nil
				}

				value := MustConvertStoredValue(context, atreeValue)
				if asReference {
					value = getReferenceValue(
						context,
						value,
						argumentType,
					)
				}

				mapped := invokeFunctionValue(
					context,
					procedure,
					[]Value{value},
					argumentTypes,
					parameterTypes,
					returnType,
					nil,
				)

				return mapped.Transfer(
					context,
					atree.Address{},
					false,
					nil,
					nil,
					false, // value has a parent container because it is from iterator.
				)
			},
		)
	})
	return result
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
	accessedType sema.Type,
) Value {
	count := v.Count()

	// Convert the constant-sized array type to a variable-sized array type.

	if _, ok := v.Type.(*ConstantSizedStaticType); !ok {
		panic(errors.NewUnreachableError())
	}

	// Cascade outer authorization into the result element type, matching
	// sema's ArrayToVariableSizedFunctionType.
	elementType := v.SemaType(context).ElementType(false)
	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)
	resultElementStaticType := ConvertSemaToStaticType(context, elementType)
	variableSizedType := NewVariableSizedStaticType(context, resultElementStaticType)

	// Convert the array to a variable-sized array.

	// Use ReadOnlyIterator here because ArrayValue elements are copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: uint64(count),
		},
	)

	return NewArrayValueWithIterator(
		context,
		variableSizedType,
		common.ZeroAddress,
		uint64(count),
		func() Value {

			// Computation was already metered above

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(context, atreeValue)

			if asReference {
				value = getReferenceValue(context, value, elementType)
			}

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
	accessedType sema.Type,
) OptionalValue {

	// Ensure the array has the expected size.

	count := v.Count()

	if int64(count) != expectedConstantSizedArraySize {
		return NilOptionalValue
	}

	// Convert the variable-sized array type to a constant-sized array type.

	if _, ok := v.Type.(*VariableSizedStaticType); !ok {
		panic(errors.NewUnreachableError())
	}

	// Cascade outer authorization into the result element type, matching
	// sema's ArrayToConstantSizedFunctionType.
	elementType := v.SemaType(context).ElementType(false)
	elementType, asReference := sema.GetDescendantTypeForAccess(context, accessedType, elementType, false)
	resultElementStaticType := ConvertSemaToStaticType(context, elementType)
	constantSizedType := NewConstantSizedStaticType(
		context,
		resultElementStaticType,
		expectedConstantSizedArraySize,
	)

	// Convert the array to a constant-sized array.

	// Use ReadOnlyIterator here because ArrayValue elements are copied (not removed) from original ArrayValue.
	iterator, err := v.array.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: uint64(count),
		},
	)

	constantSizedArray := NewArrayValueWithIterator(
		context,
		constantSizedType,
		common.ZeroAddress,
		uint64(count),
		func() Value {

			// Computation was already metered above

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(context, atreeValue)

			if asReference {
				value = getReferenceValue(context, value, elementType)
			}

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
	elementType   StaticType
	valueID       atree.ValueID
	atreeIterator atree.ArrayIterator
	next          atree.Value
}

var _ ValueIterator = &ArrayIterator{}

func NewArrayIterator(gauge common.MemoryGauge, v *ArrayValue) ValueIterator {
	common.UseMemory(
		gauge,
		common.MemoryUsage{
			Kind:   common.MemoryKindArrayIterator,
			Amount: 1,
		},
	)

	valueID := v.ValueID()

	arrayIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &ArrayIterator{
		elementType:   v.Type.ElementType(),
		valueID:       valueID,
		atreeIterator: arrayIterator,
	}
}

func (i *ArrayIterator) HasNext(context ValueIteratorContext) bool {
	if i.next != nil {
		return true
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindAtreeArrayReadIteration,
			Intensity: 1,
		},
	)

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
		common.UseComputation(
			context,
			common.ComputationUsage{
				Kind:      common.ComputationKindAtreeArrayReadIteration,
				Intensity: 1,
			},
		)

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
	result := MustConvertStoredValue(context, atreeValue)
	return result
}

func (i *ArrayIterator) ValueID() (atree.ValueID, bool) {
	return i.valueID, true
}

// define all native functions for array type
var NativeArrayAppendFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
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
		_ ArgumentTypesIterator,
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
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)
		otherArray := AssertValueOfType[*ArrayValue](args[0])

		return thisArray.Concat(context, otherArray, accessedType)
	},
)

var NativeArrayInsertFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
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
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		index := AssertValueOfType[NumberValue](args[0])

		return thisArray.Remove(context, index.ToInt())
	},
)

var NativeArrayContainsFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
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
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)
		fromValue := AssertValueOfType[IntValue](args[0])
		toValue := AssertValueOfType[IntValue](args[1])

		return thisArray.Slice(context, fromValue, toValue, accessedType)
	},
)

var NativeArrayReverseFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)
		return thisArray.Reverse(context, accessedType)
	},
)

var NativeArrayFilterFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		array := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)

		funcValue := AssertValueOfType[FunctionValue](args[0])

		return array.Filter(context, funcValue, accessedType)
	},
)

func arrayValueFromReceiver(context ValueStaticTypeContext, receiver Value) *ArrayValue {
	switch receiver := receiver.(type) {
	case *ArrayValue:
		return receiver

	case *StorageReferenceValue:
		referencedValue := receiver.MustReferencedValue(context)
		return AssertValueOfType[*ArrayValue](referencedValue)

	case *EphemeralReferenceValue:
		referencedValue := receiver.Value
		return AssertValueOfType[*ArrayValue](referencedValue)

	default:
		panic(errors.NewUnreachableError())
	}
}

var NativeArrayMapFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		array := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)

		funcValue := AssertValueOfType[FunctionValue](args[0])

		return array.Map(context, funcValue, accessedType)
	},
)

var NativeArrayToVariableSizedFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)

		return thisArray.ToVariableSized(context, accessedType)
	},
)

var NativeArrayToConstantSizedFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		typeArguments TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		args []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)
		accessedType := MustSemaTypeOfValue(receiver, context)
		constantSizedArrayType, ok := typeArguments.NextStatic().(*ConstantSizedStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return thisArray.ToConstantSized(context, constantSizedArrayType.Size, accessedType)
	},
)

var NativeArrayFirstIndexFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
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
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)

		return thisArray.RemoveFirst(context)
	},
)

var NativeArrayRemoveLastFunction = NativeFunction(
	func(
		context NativeFunctionContext,
		_ TypeArgumentsIterator,
		_ ArgumentTypesIterator,
		receiver Value,
		_ []Value,
	) Value {
		thisArray := arrayValueFromReceiver(context, receiver)

		return thisArray.RemoveLast(context)
	},
)

// canCopyNonRefSimpleForType returns true if CopyNonRefSimple()
// always returns true for the given type's storable.
func canCopyNonRefSimpleForType(t StaticType) bool {
	pt, ok := t.(PrimitiveStaticType)
	if !ok {
		return false
	}
	switch pt {
	case PrimitiveStaticTypeBool:
		return true
	case PrimitiveStaticTypeAddress:
		return true
	case PrimitiveStaticTypeCharacter:
		return true
	case PrimitiveStaticTypeInt8,
		PrimitiveStaticTypeInt16,
		PrimitiveStaticTypeInt32,
		PrimitiveStaticTypeInt64,
		PrimitiveStaticTypeInt128,
		PrimitiveStaticTypeInt256:
		return true
	case PrimitiveStaticTypeUInt8,
		PrimitiveStaticTypeUInt16,
		PrimitiveStaticTypeUInt32,
		PrimitiveStaticTypeUInt64,
		PrimitiveStaticTypeUInt128,
		PrimitiveStaticTypeUInt256:
		return true
	case PrimitiveStaticTypeWord8,
		PrimitiveStaticTypeWord16,
		PrimitiveStaticTypeWord32,
		PrimitiveStaticTypeWord64,
		PrimitiveStaticTypeWord128,
		PrimitiveStaticTypeWord256:
		return true
	case PrimitiveStaticTypeFix64, PrimitiveStaticTypeFix128:
		return true
	case PrimitiveStaticTypeUFix64, PrimitiveStaticTypeUFix128:
		return true
	}
	return false
}
