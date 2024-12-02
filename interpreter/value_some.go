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

func (*SomeValue) isValue() {}

func (v *SomeValue) Accept(interpreter *Interpreter, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitSomeValue(interpreter, v)
	if !descend {
		return
	}
	v.value.Accept(interpreter, visitor, locationRange)
}

func (v *SomeValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
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

func (v *SomeValue) IsImportable(inter *Interpreter, locationRange LocationRange) bool {
	return v.value.IsImportable(inter, locationRange)
}

func (*SomeValue) isOptionalValue() {}

func (v *SomeValue) forEach(f func(Value)) {
	f(v.value)
}

func (v *SomeValue) fmap(inter *Interpreter, f func(Value) Value) OptionalValue {
	newValue := f(v.value)
	return NewSomeValueNonCopying(inter, newValue)
}

func (v *SomeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *SomeValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {
	innerValue := v.InnerValue()
	maybeDestroy(interpreter, locationRange, innerValue)

	v.isDestroyed = true
	v.value = nil
}

func (v *SomeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SomeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.value.RecursiveString(seenReferences)
}

func (v *SomeValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
	return v.value.MeteredString(interpreter, seenReferences, locationRange)
}

func (v *SomeValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.OptionalTypeMapFunctionName:
		innerValueType := interpreter.MustConvertStaticToSemaType(
			v.value.StaticType(interpreter),
		)
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.OptionalTypeMapFunctionType(
				innerValueType,
			),
			func(v *SomeValue, invocation Invocation) Value {
				inter := invocation.Interpreter
				locationRange := invocation.LocationRange

				transformFunction, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType := transformFunction.FunctionType()
				parameterTypes := transformFunctionType.ParameterTypes()
				returnType := transformFunctionType.ReturnTypeAnnotation.Type

				return v.fmap(
					inter,
					func(v Value) Value {
						return inter.invokeFunctionValue(
							transformFunction,
							[]Value{v},
							nil,
							[]sema.Type{innerValueType},
							parameterTypes,
							returnType,
							invocation.TypeParameterTypes,
							locationRange,
						)
					},
				)
			},
		)
	}

	return nil
}

func (v *SomeValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	// NOTE: value does not have static type information on its own,
	// SomeValue.StaticType builds type from inner value (if available),
	// so no need to check it

	innerValue := v.InnerValue()

	return innerValue.ConformsToStaticType(
		interpreter,
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

	if v.valueStorable == nil {

		nonSomeValue, nestedLevels := v.nonSomeValue()

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

func (v *SomeValue) IsResourceKinded(interpreter *Interpreter) bool {
	// If the inner value is `nil`, then this is an invalidated resource.
	if v.value == nil {
		return true
	}

	return v.value.IsResourceKinded(interpreter)
}

func (v *SomeValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {
	innerValue := v.value

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		innerValue = v.value.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
			hasNoParentContainer,
		)

		if remove {
			interpreter.RemoveReferencedSlab(v.valueStorable)
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

		// we don't need to invalidate referenced resources if this resource was moved
		// to storage, as the earlier transfer will have done this already
		if !needsStoreTo {
			interpreter.invalidateReferencedResources(v.value, locationRange)
		}
		v.value = nil
	}

	res := NewSomeValueNonCopying(interpreter, innerValue)
	res.valueStorable = nil
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *SomeValue) Clone(interpreter *Interpreter) Value {
	innerValue := v.value.Clone(interpreter)
	return NewUnmeteredSomeValueNonCopying(innerValue)
}

func (v *SomeValue) DeepRemove(interpreter *Interpreter, hasNoParentContainer bool) {
	v.value.DeepRemove(interpreter, hasNoParentContainer)
	if v.valueStorable != nil {
		interpreter.RemoveReferencedSlab(v.valueStorable)
	}
}

func (v *SomeValue) InnerValue() Value {
	return v.value
}

func (v *SomeValue) isInvalidatedResource(_ *Interpreter) bool {
	return v.value == nil || v.IsDestroyed()
}

type SomeStorable struct {
	gauge    common.MemoryGauge
	Storable atree.Storable
}

var _ atree.ContainerStorable = SomeStorable{}

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
		return cborTagSize
	}
	return cborTagSize + someStorableWithMultipleNestedlevelsArraySize + getUintCBORSize(nestedLevels)
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
