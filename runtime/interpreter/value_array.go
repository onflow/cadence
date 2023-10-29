package interpreter

import (
	goerrors "errors"
	"time"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
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
	atreeIterator *atree.ArrayIterator
}

func (v *ArrayValue) Iterator(_ *Interpreter) ValueIterator {
	arrayIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return ArrayValueIterator{
		atreeIterator: arrayIterator,
	}
}

var _ ValueIterator = ArrayValueIterator{}

func (i ArrayValueIterator) Next(interpreter *Interpreter) Value {
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

func newArrayValueFromAtreeValue(
	array *atree.Array,
	staticType ArrayStaticType,
) *ArrayValue {
	return &ArrayValue{
		Type:  staticType,
		array: array,
	}
}

func newArrayValueFromConstructor(
	gauge common.MemoryGauge,
	staticType ArrayStaticType,
	countOverestimate uint64,
	constructor func() *atree.Array,
) (array *ArrayValue) {
	var elementSize uint
	if staticType != nil {
		elementSize = staticType.ElementType().elementSize()
	}
	baseUsage, elementUsage, dataSlabs, metaDataSlabs := common.NewArrayMemoryUsages(countOverestimate, elementSize)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, elementUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	array = newArrayValueFromAtreeValue(constructor(), staticType)
	array.elementSize = elementSize
	return
}

var _ Value = &ArrayValue{}
var _ atree.Value = &ArrayValue{}
var _ EquatableValue = &ArrayValue{}
var _ ValueIndexableValue = &ArrayValue{}
var _ MemberAccessibleValue = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}
var _ IterableValue = &ArrayValue{}

func (*ArrayValue) isValue() {}

func (v *ArrayValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitArrayValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(interpreter, func(element Value) {
		element.Accept(interpreter, visitor)
	})
}

func (v *ArrayValue) Iterate(interpreter *Interpreter, f func(element Value) (resume bool)) {
	iterate := func() {
		err := v.array.Iterate(func(element atree.Value) (resume bool, err error) {
			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			resume = f(MustConvertStoredValue(interpreter, element))

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

func (v *ArrayValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.Iterate(interpreter, func(element Value) (resume bool) {
		walkChild(element)
		return true
	})
}

func (v *ArrayValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *ArrayValue) IsImportable(inter *Interpreter) bool {
	importable := true
	v.Iterate(inter, func(element Value) (resume bool) {
		if !element.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *ArrayValue) checkInvalidatedResourceUse(interpreter *Interpreter, locationRange LocationRange) {
	if v.isDestroyed || (v.array == nil && v.IsResourceKinded(interpreter)) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *ArrayValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyArrayValue, 1)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

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

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			v.Walk(interpreter, func(element Value) {
				maybeDestroy(interpreter, locationRange, element)
			})
		},
	)

	v.isDestroyed = true

	if config.InvalidatedResourceValidationEnabled {
		v.array = nil
	}

	interpreter.updateReferencedResource(
		storageID,
		storageID,
		func(value ReferenceTrackedResourceKindedValue) {
			arrayValue, ok := value.(*ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			arrayValue.isDestroyed = true

			if config.InvalidatedResourceValidationEnabled {
				arrayValue.array = nil
			}
		},
	)
}

func (v *ArrayValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *ArrayValue) Concat(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) Value {

	first := true

	firstIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	secondIterator, err := other.array.Iterator()
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
			)
		},
	)
}

func (v *ArrayValue) GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

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

	storable, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}

	return StoredValue(interpreter, storable, interpreter.Storage())
}

func (v *ArrayValue) SetKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	index := key.(NumberValue).ToInt(locationRange)
	v.Set(interpreter, locationRange, index, value)
}

func (v *ArrayValue) Set(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

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
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	existingStorable, err := v.array.Set(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)

	existingValue := StoredValue(interpreter, existingStorable, interpreter.Storage())

	existingValue.DeepRemove(interpreter)

	interpreter.RemoveReferencedSlab(existingStorable)
}

func (v *ArrayValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *ArrayValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

func (v *ArrayValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	// if n > 0:
	// len = open-bracket + close-bracket + ((n-1) comma+space)
	//     = 2 + 2n - 2
	//     = 2n
	// Always +2 to include empty array case (over estimate).
	// Each elements' string value is metered individually.
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(v.Count()*2+2))

	values := make([]string, v.Count())

	i := 0

	_ = v.array.Iterate(func(element atree.Value) (resume bool, err error) {
		// ok to not meter anything created as part of this iteration, since we will discard the result
		// upon creating the string
		values[i] = MustConvertUnmeteredStoredValue(element).MeteredString(memoryGauge, seenReferences)
		i++
		return true, nil
	})

	return format.Array(values)
}

func (v *ArrayValue) Append(interpreter *Interpreter, locationRange LocationRange, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

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
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	err := v.array.Append(element)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) AppendAll(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) {
	other.Walk(interpreter, func(value Value) {
		v.Append(interpreter, locationRange, value)
	})
}

func (v *ArrayValue) InsertKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	index := key.(NumberValue).ToInt(locationRange)
	v.Insert(interpreter, locationRange, index, value)
}

func (v *ArrayValue) Insert(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

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

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) RemoveKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	index := key.(NumberValue).ToInt(locationRange)
	return v.Remove(interpreter, locationRange, index)
}

func (v *ArrayValue) Remove(interpreter *Interpreter, locationRange LocationRange, index int) Value {

	interpreter.validateMutation(v.StorageID(), locationRange)

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

	value := StoredValue(interpreter, storable, interpreter.Storage())

	return value.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		storable,
		nil,
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
	v.Iterate(interpreter, func(element Value) (resume bool) {
		if needleEquatable.Equal(interpreter, locationRange, element) {
			result = true
			// stop iteration
			return false
		}
		counter++
		// continue iteration
		return true
	})

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
	v.Iterate(interpreter, func(element Value) (resume bool) {
		if needleEquatable.Equal(interpreter, locationRange, element) {
			result = true
			// stop iteration
			return false
		}
		// continue iteration
		return true
	})

	return AsBoolValue(result)
}

func (v *ArrayValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}
	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "append":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayAppendFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				v.Append(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
				return Void
			},
		)

	case "appendAll":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayAppendAllFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
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
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayConcatFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
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
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayInsertFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				element := invocation.Arguments[1]

				v.Insert(
					invocation.Interpreter,
					invocation.LocationRange,
					index,
					element,
				)
				return Void
			},
		)

	case "remove":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				return v.Remove(
					invocation.Interpreter,
					invocation.LocationRange,
					index,
				)
			},
		)

	case "removeFirst":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveFirstFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.RemoveFirst(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "removeLast":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveLastFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.RemoveLast(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "firstIndex":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayFirstIndexFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.FirstIndex(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "contains":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayContainsFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.Contains(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "slice":
		return NewHostFunctionValue(
			interpreter,
			sema.ArraySliceFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
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
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayReverseFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				return v.Reverse(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeFilterFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayFilterFunctionType(
				interpreter,
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
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
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayMapFunctionType(
				interpreter,
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType, ok := invocation.ArgumentTypes[0].(*sema.FunctionType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Map(
					interpreter,
					invocation.LocationRange,
					funcArgument,
					transformFunctionType,
				)
			},
		)
	}

	return nil
}

func (v *ArrayValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	// Arrays have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

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
	case ConstantSizedStaticType:
		elementType = staticType.ElementType()
		if v.Count() != int(staticType.Size) {
			return false
		}
	case VariableSizedStaticType:
		elementType = staticType.ElementType()
	default:
		return false
	}

	var elementMismatch bool

	v.Iterate(interpreter, func(element Value) (resume bool) {

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
	})

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
	preventTransfer map[atree.StorageID]struct{},
) Value {
	baseUsage, elementUsage, dataSlabs, metaDataSlabs := common.NewArrayMemoryUsages(v.array.Count(), v.elementSize)
	common.UseMemory(interpreter, baseUsage)
	common.UseMemory(interpreter, elementUsage)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)

	config := interpreter.SharedState.Config

	if config.InvalidatedResourceValidationEnabled {
		v.checkInvalidatedResourceUse(interpreter, locationRange)
	}

	interpreter.ReportComputation(common.ComputationKindTransferArrayValue, uint(v.Count()))

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

	currentStorageID := v.StorageID()

	if preventTransfer == nil {
		preventTransfer = map[atree.StorageID]struct{}{}
	} else if _, ok := preventTransfer[currentStorageID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentStorageID] = struct{}{}
	defer delete(preventTransfer, currentStorageID)

	array := v.array

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

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

				element := MustConvertStoredValue(interpreter, value).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				return element, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.array.PopIterate(func(storable atree.Storable) {
				interpreter.RemoveReferencedSlab(storable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			interpreter.maybeValidateAtreeValue(v.array)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	var res *ArrayValue

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		if config.InvalidatedResourceValidationEnabled {
			v.array = nil
		} else {
			v.array = array
			res = v
		}

		newStorageID := array.StorageID()

		interpreter.updateReferencedResource(
			currentStorageID,
			newStorageID,
			func(value ReferenceTrackedResourceKindedValue) {
				arrayValue, ok := value.(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				arrayValue.array = array
			},
		)
	}

	if res == nil {
		res = newArrayValueFromAtreeValue(array, v.Type)
		res.elementSize = v.elementSize
		res.semaType = v.semaType
		res.isResourceKinded = v.isResourceKinded
		res.isDestroyed = v.isDestroyed
	}

	return res
}

func (v *ArrayValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	baseUsage, elementUsage, dataSlabs, metaDataSlabs := common.NewArrayMemoryUsages(v.array.Count(), v.elementSize)
	common.UseMemory(interpreter, baseUsage)
	common.UseMemory(interpreter, elementUsage)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)

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
	return &ArrayValue{
		Type:             v.Type,
		semaType:         v.semaType,
		isResourceKinded: v.isResourceKinded,
		array:            array,
		isDestroyed:      v.isDestroyed,
	}
}

func (v *ArrayValue) DeepRemove(interpreter *Interpreter) {
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
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) StorageID() atree.StorageID {
	return v.array.StorageID()
}

func (v *ArrayValue) StorageAddress() atree.Address {
	return v.array.Address()
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

	iterator, err := v.array.RangeIterator(uint64(fromIndex), uint64(toIndex))
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

			var value Value

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

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
			)
		},
	)
}

func (v *ArrayValue) Filter(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) Value {

	elementTypeSlice := []sema.Type{v.semaType.ElementType(false)}
	iterationInvocation := func(arrayElement Value) Invocation {
		invocation := NewInvocation(
			interpreter,
			nil,
			nil,
			[]Value{arrayElement},
			elementTypeSlice,
			nil,
			locationRange,
		)
		return invocation
	}

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

				shouldInclude, ok := procedure.invoke(iterationInvocation(value)).(BoolValue)
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
			)
		},
	)
}

func (v *ArrayValue) Map(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
	transformFunctionType *sema.FunctionType,
) Value {

	elementTypeSlice := []sema.Type{v.semaType.ElementType(false)}
	iterationInvocation := func(arrayElement Value) Invocation {
		return NewInvocation(
			interpreter,
			nil,
			nil,
			[]Value{arrayElement},
			elementTypeSlice,
			nil,
			locationRange,
		)
	}

	procedureStaticType, ok := ConvertSemaToStaticType(interpreter, transformFunctionType).(FunctionStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	returnType := procedureStaticType.ReturnType(interpreter)

	var returnArrayStaticType ArrayStaticType
	switch v.Type.(type) {
	case VariableSizedStaticType:
		returnArrayStaticType = NewVariableSizedStaticType(
			interpreter,
			returnType,
		)
	case ConstantSizedStaticType:
		returnArrayStaticType = NewConstantSizedStaticType(
			interpreter,
			returnType,
			int64(v.Count()),
		)
	default:
		panic(errors.NewUnreachableError())
	}

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

			mappedValue := procedure.invoke(iterationInvocation(value))
			return mappedValue.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}
