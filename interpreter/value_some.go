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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// SomeValue

type SomeValue struct {
	value         Value
	valueStorable atree.Storable
	// TODO: Store isDestroyed in SomeStorable?
	isDestroyed bool
}

func NewSomeValueNonCopying(memoryGauge common.MemoryGauge, value Value) *SomeValue {
	common.UseMemory(memoryGauge, common.OptionalValueMemoryUsage)

	return NewUnmeteredSomeValueNonCopying(value)
}

func NewUnmeteredSomeValueNonCopying(value Value) *SomeValue {
	return &SomeValue{
		value: value,
	}
}

var _ Value = &SomeValue{}
var _ EquatableValue = &SomeValue{}
var _ MemberAccessibleValue = &SomeValue{}
var _ OptionalValue = &SomeValue{}
var _ atree.Value = &SomeValue{}
var _ atree.WrapperValue = &SomeValue{}

// UnwrapAtreeValue returns non-SomeValue and wrapper size.
func (v *SomeValue) UnwrapAtreeValue() (atree.Value, uint64) {
	// NOTE:
	// - non-SomeValue is the same as non-SomeValue in SomeValue.Storable()
	// - non-SomeValue wrapper size is the same as encoded wrapper size in SomeStorable.ByteSize().

	// Unwrap SomeValue(s)
	nonSomeValue, nestedLevels := v.nonSomeValue()

	// Get SomeValue(s) wrapper size
	someStorableEncodedPrefixSize := getSomeStorableEncodedPrefixSize(nestedLevels)

	// Unwrap nonSomeValue if needed
	switch nonSomeValue := nonSomeValue.(type) {
	case atree.WrapperValue:
		unwrappedValue, wrapperSize := nonSomeValue.UnwrapAtreeValue()
		return unwrappedValue, wrapperSize + uint64(someStorableEncodedPrefixSize)

	default:
		return nonSomeValue, uint64(someStorableEncodedPrefixSize)
	}
}

func (*SomeValue) IsValue() {}

func (v *SomeValue) Accept(context ValueVisitContext, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitSomeValue(context, v)
	if !descend {
		return
	}
	v.value.Accept(context, visitor, locationRange)
}

func (v *SomeValue) Walk(_ ValueWalkContext, walkChild func(Value), _ LocationRange) {
	walkChild(v.value)
}

func (v *SomeValue) StaticType(context ValueStaticTypeContext) StaticType {
	if v.isDestroyed {
		return nil
	}

	innerType := v.value.StaticType(context)
	if innerType == nil {
		return nil
	}
	return NewOptionalStaticType(
		context,
		innerType,
	)
}

func (v *SomeValue) IsImportable(context ValueImportableContext, locationRange LocationRange) bool {
	return v.value.IsImportable(context, locationRange)
}

func (*SomeValue) isOptionalValue() {}

func (v *SomeValue) forEach(f func(Value)) {
	f(v.value)
}

func (v *SomeValue) fmap(memoryGauge common.MemoryGauge, f func(Value) Value) OptionalValue {
	newValue := f(v.value)
	return NewSomeValueNonCopying(memoryGauge, newValue)
}

func (v *SomeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *SomeValue) Destroy(context ResourceDestructionContext, locationRange LocationRange) {
	innerValue := v.InnerValue()
	maybeDestroy(context, locationRange, innerValue)

	v.isDestroyed = true
	v.value = nil
}

func (v *SomeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SomeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.value.RecursiveString(seenReferences)
}

func (v *SomeValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
	return v.value.MeteredString(context, seenReferences, locationRange)
}

func (v *SomeValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v *SomeValue) GetMethod(
	context MemberAccessibleContext,
	_ LocationRange,
	name string,
) FunctionValue {
	switch name {
	case sema.OptionalTypeMapFunctionName:
		innerValueType := v.InnerValueType(context)
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.OptionalTypeMapFunctionType(
				innerValueType,
			),
			func(v *SomeValue, invocation Invocation) Value {
				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				transformFunction, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType := transformFunction.FunctionType(invocationContext)

				return OptionalValueMapFunction(
					invocationContext,
					v,
					transformFunctionType,
					transformFunction,
					innerValueType,
					locationRange,
				)
			},
		)
	}

	return nil
}

func OptionalValueMapFunction(
	invocationContext InvocationContext,
	optionalValue OptionalValue,
	transformFunctionType *sema.FunctionType,
	transformFunction FunctionValue,
	innerValueType sema.Type,
	locationRange LocationRange,
) Value {
	parameterTypes := transformFunctionType.ParameterTypes()
	returnType := transformFunctionType.ReturnTypeAnnotation.Type

	return optionalValue.fmap(
		invocationContext,
		func(v Value) Value {
			return invokeFunctionValue(
				invocationContext,
				transformFunction,
				[]Value{v},
				nil,
				[]sema.Type{innerValueType},
				parameterTypes,
				returnType,
				nil,
				locationRange,
			)
		},
	)
}

func (v *SomeValue) InnerValueType(context ValueStaticTypeContext) sema.Type {
	return MustConvertStaticToSemaType(
		v.value.StaticType(context),
		context,
	)
}

func (v *SomeValue) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	// NOTE: value does not have static type information on its own,
	// SomeValue.StaticType builds type from inner value (if available),
	// so no need to check it

	innerValue := v.InnerValue()

	return innerValue.ConformsToStaticType(
		context,
		locationRange,
		results,
	)
}

func (v *SomeValue) Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	innerValue := v.InnerValue()

	equatableValue, ok := innerValue.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(context, locationRange, otherSome.value)
}

func (v *SomeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {

	// SomeStorable returned from this function can be encoded in two ways:
	// - if non-SomeStorable is too large, non-SomeStorable is encoded in a separate slab
	//   while SomeStorable wrapper is encoded inline with reference to slab containing
	//   non-SomeStorable.
	// - otherwise, SomeStorable with non-SomeStorable is encoded inline.
	//
	// The above applies to both immutable non-SomeValue (such as StringValue),
	// and mutable non-SomeValue (such as ArrayValue).

	// NOTE:
	// - If SomeValue's inner value is a value with atree.Array or atree.OrderedMap,
	//   we MUST NOT cache SomeStorable because we need to call nonSomeValue.Storable()
	//   to trigger container inlining or un-inlining.
	// - Otherwise, we need to cache SomeStorable because nonSomeValue.Storable() can
	//   create registers in storage, such as large string.

	nonSomeValue, nestedLevels := v.nonSomeValue()

	_, isContainerValue := nonSomeValue.(atreeContainerBackedValue)

	if v.valueStorable == nil || isContainerValue {

		someStorableEncodedPrefixSize := getSomeStorableEncodedPrefixSize(nestedLevels)

		// Reduce maxInlineSize for non-SomeValue to make sure
		// that SomeStorable wrapper is always encoded inline.
		maxInlineSize -= uint64(someStorableEncodedPrefixSize)

		nonSomeValueStorable, err := nonSomeValue.Storable(
			storage,
			address,
			maxInlineSize,
		)
		if err != nil {
			return nil, err
		}

		valueStorable := nonSomeValueStorable
		for i := 1; i < int(nestedLevels); i++ {
			valueStorable = SomeStorable{
				Storable: valueStorable,
			}
		}
		v.valueStorable = valueStorable
	}

	// No need to call maybeLargeImmutableStorable() here for SomeStorable because:
	// - encoded SomeStorable size = someStorableEncodedPrefixSize + non-SomeValueStorable size
	// - non-SomeValueStorable size < maxInlineSize - someStorableEncodedPrefixSize
	return SomeStorable{
		Storable: v.valueStorable,
	}, nil
}

// nonSomeValue returns a non-SomeValue and nested levels of SomeValue reached
// by traversing nested SomeValue (SomeValue containing SomeValue, etc.)
// until it reaches a non-SomeValue.
// For example,
//   - `SomeValue{true}` has non-SomeValue `true`, and nested levels 1
//   - `SomeValue{SomeValue{1}}` has non-SomeValue `1` and nested levels 2
//   - `SomeValue{SomeValue{[SomeValue{SomeValue{SomeValue{1}}}]}} has
//     non-SomeValue `[SomeValue{SomeValue{SomeValue{1}}}]` and nested levels 2
func (v *SomeValue) nonSomeValue() (atree.Value, uint64) {
	nestedLevels := uint64(1)
	for {
		switch value := v.value.(type) {
		case *SomeValue:
			nestedLevels++
			v = value

		default:
			return value, nestedLevels
		}
	}
}

func (v *SomeValue) NeedsStoreTo(address atree.Address) bool {
	return v.value.NeedsStoreTo(address)
}

func (v *SomeValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	// If the inner value is `nil`, then this is an invalidated resource.
	if v.value == nil {
		return true
	}

	return v.value.IsResourceKinded(context)
}

func (v *SomeValue) Transfer(
	context ValueTransferContext,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {
	innerValue := v.value

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(context)

	if needsStoreTo || !isResourceKinded {

		innerValue = v.value.Transfer(
			context,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
			hasNoParentContainer,
		)

		if remove {
			RemoveReferencedSlab(context, v.valueStorable)
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

		// we don't need to invalidate referenced resources if this resource was moved
		// to storage, as the earlier transfer will have done this already
		if !needsStoreTo {
			InvalidateReferencedResources(context, v.value, locationRange)
		}
		v.value = nil
	}

	res := NewSomeValueNonCopying(context, innerValue)
	res.valueStorable = nil
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *SomeValue) Clone(context ValueCloneContext) Value {
	innerValue := v.value.Clone(context)
	return NewUnmeteredSomeValueNonCopying(innerValue)
}

func (v *SomeValue) DeepRemove(context ValueRemoveContext, hasNoParentContainer bool) {
	v.value.DeepRemove(context, hasNoParentContainer)
	if v.valueStorable != nil {
		RemoveReferencedSlab(context, v.valueStorable)
	}
}

func (v *SomeValue) InnerValue() Value {
	return v.value
}

func (v *SomeValue) isInvalidatedResource(_ ValueStaticTypeContext) bool {
	return v.value == nil || v.IsDestroyed()
}

type SomeStorable struct {
	gauge    common.MemoryGauge
	Storable atree.Storable
}

var _ atree.ContainerStorable = SomeStorable{}
var _ atree.WrapperStorable = SomeStorable{}

func (s SomeStorable) UnwrapAtreeStorable() atree.Storable {
	storable := s.Storable

	switch storable := storable.(type) {
	case atree.WrapperStorable:
		return storable.UnwrapAtreeStorable()

	default:
		return storable
	}
}

// WrapAtreeStorable() wraps storable as innermost wrapped value and
// returns new wrapped storable.
func (s SomeStorable) WrapAtreeStorable(storable atree.Storable) atree.Storable {
	_, nestedLevels := s.nonSomeStorable()

	newStorable := SomeStorable{Storable: storable}
	for i := 1; i < int(nestedLevels); i++ {
		newStorable = SomeStorable{Storable: newStorable}
	}
	return newStorable
}

func (s SomeStorable) HasPointer() bool {
	switch cs := s.Storable.(type) {
	case atree.ContainerStorable:
		return cs.HasPointer()
	default:
		return false
	}
}

func getSomeStorableEncodedPrefixSize(nestedLevels uint64) uint32 {
	if nestedLevels == 1 {
		return values.CBORTagSize
	}
	return values.CBORTagSize +
		someStorableWithMultipleNestedlevelsArraySize +
		values.GetUintCBORSize(nestedLevels)
}

func (s SomeStorable) ByteSize() uint32 {
	nonSomeStorable, nestedLevels := s.nonSomeStorable()
	return getSomeStorableEncodedPrefixSize(nestedLevels) + nonSomeStorable.ByteSize()
}

// nonSomeStorable returns a non-SomeStorable and nested levels of SomeStorable reached
// by traversing nested SomeStorable (SomeStorable containing SomeStorable, etc.)
// until it reaches a non-SomeStorable.
// For example,
//   - `SomeStorable{true}` has non-SomeStorable `true`, and nested levels 1
//   - `SomeStorable{SomeStorable{1}}` has non-SomeStorable `1` and nested levels 2
//   - `SomeStorable{SomeStorable{[SomeStorable{SomeStorable{SomeStorable{1}}}]}} has
//     non-SomeStorable `[SomeStorable{SomeStorable{SomeStorable{1}}}]` and nested levels 2
func (s SomeStorable) nonSomeStorable() (atree.Storable, uint64) {
	nestedLevels := uint64(1)
	for {
		switch storable := s.Storable.(type) {
		case SomeStorable:
			nestedLevels++
			s = storable

		default:
			return storable, nestedLevels
		}
	}
}

func (s SomeStorable) StoredValue(storage atree.SlabStorage) (atree.Value, error) {
	value := StoredValue(s.gauge, s.Storable, storage)

	return &SomeValue{
		value:         value,
		valueStorable: s.Storable,
	}, nil
}

func (s SomeStorable) ChildStorables() []atree.Storable {
	return []atree.Storable{
		s.Storable,
	}
}
