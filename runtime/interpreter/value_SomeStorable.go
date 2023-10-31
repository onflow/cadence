package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// SomeValue

type SomeValue struct {
	value         Value
	valueStorable atree.Storable
	// TODO: Store isDestroyed in SomeStorable?
	isDestroyed bool
}

func NewSomeValueNonCopying(interpreter *Interpreter, value Value) *SomeValue {
	common.UseMemory(interpreter, common.OptionalValueMemoryUsage)

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

func (v *SomeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitSomeValue(interpreter, v)
	if !descend {
		return
	}
	v.value.Accept(interpreter, visitor)
}

func (v *SomeValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.value)
}

func (v *SomeValue) StaticType(inter *Interpreter) StaticType {
	if v.isDestroyed {
		return nil
	}

	innerType := v.value.StaticType(inter)
	if innerType == nil {
		return nil
	}
	return NewOptionalStaticType(
		inter,
		innerType,
	)
}

func (v *SomeValue) IsImportable(inter *Interpreter) bool {
	return v.value.IsImportable(inter)
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
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

	innerValue := v.InnerValue(interpreter, locationRange)

	maybeDestroy(interpreter, locationRange, innerValue)
	v.isDestroyed = true

	if config.InvalidatedResourceValidationEnabled {
		v.value = nil
	}
}

func (v *SomeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SomeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.value.RecursiveString(seenReferences)
}

func (v *SomeValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	return v.value.MeteredString(memoryGauge, seenReferences)
}

func (v *SomeValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}
	switch name {
	case sema.OptionalTypeMapFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.OptionalTypeMapFunctionType(
				interpreter.MustConvertStaticToSemaType(
					v.value.StaticType(interpreter),
				),
			),
			func(invocation Invocation) Value {

				transformFunction, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType, ok := invocation.ArgumentTypes[0].(*sema.FunctionType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				valueType := transformFunctionType.Parameters[0].TypeAnnotation.Type

				f := func(v Value) Value {
					transformInvocation := NewInvocation(
						invocation.Interpreter,
						nil,
						nil,
						[]Value{v},
						[]sema.Type{valueType},
						nil,
						invocation.LocationRange,
					)
					return transformFunction.invoke(transformInvocation)
				}

				return v.fmap(invocation.Interpreter, f)
			},
		)
	}

	return nil
}

func (v *SomeValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

	panic(errors.NewUnreachableError())
}

func (v *SomeValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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

	innerValue := v.InnerValue(interpreter, locationRange)

	return innerValue.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)
}

func (v *SomeValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	innerValue := v.InnerValue(interpreter, locationRange)

	equatableValue, ok := innerValue.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(interpreter, locationRange, otherSome.value)
}

func (v *SomeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {

	if v.valueStorable == nil {
		var err error
		v.valueStorable, err = v.value.Storable(
			storage,
			address,
			maxInlineSize,
		)
		if err != nil {
			return nil, err
		}
	}

	return maybeLargeImmutableStorable(
		SomeStorable{
			Storable: v.valueStorable,
		},
		storage,
		address,
		maxInlineSize,
	)
}

func (v *SomeValue) NeedsStoreTo(address atree.Address) bool {
	return v.value.NeedsStoreTo(address)
}

func (v *SomeValue) IsResourceKinded(interpreter *Interpreter) bool {
	return v.value.IsResourceKinded(interpreter)
}

func (v *SomeValue) checkInvalidatedResourceUse(locationRange LocationRange) {
	if v.isDestroyed || v.value == nil {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *SomeValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

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
		)

		if remove {
			interpreter.RemoveReferencedSlab(v.valueStorable)
			interpreter.RemoveReferencedSlab(storable)
		}
	}

	var res *SomeValue

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		if config.InvalidatedResourceValidationEnabled {
			v.value = nil
		} else {
			v.value = innerValue
			v.valueStorable = nil
			res = v
		}

	}

	if res == nil {
		res = NewSomeValueNonCopying(interpreter, innerValue)
		res.valueStorable = nil
		res.isDestroyed = v.isDestroyed
	}

	return res
}

func (v *SomeValue) Clone(interpreter *Interpreter) Value {
	innerValue := v.value.Clone(interpreter)
	return NewUnmeteredSomeValueNonCopying(innerValue)
}

func (v *SomeValue) DeepRemove(interpreter *Interpreter) {
	v.value.DeepRemove(interpreter)
	if v.valueStorable != nil {
		interpreter.RemoveReferencedSlab(v.valueStorable)
	}
}

func (v *SomeValue) InnerValue(interpreter *Interpreter, locationRange LocationRange) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(locationRange)
	}

	return v.value
}

type SomeStorable struct {
	gauge    common.MemoryGauge
	Storable atree.Storable
}

var _ atree.Storable = SomeStorable{}

func (s SomeStorable) ByteSize() uint32 {
	return cborTagSize + s.Storable.ByteSize()
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
