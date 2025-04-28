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

// DictionaryValue

type DictionaryValue struct {
	Type             *DictionaryStaticType
	semaType         *sema.DictionaryType
	isResourceKinded *bool
	dictionary       *atree.OrderedMap
	isDestroyed      bool
	elementSize      uint
}

func NewDictionaryValue(
	context DictionaryCreationContext,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {
	return NewDictionaryValueWithAddress(
		context,
		locationRange,
		dictionaryType,
		common.ZeroAddress,
		keysAndValues...,
	)
}

func NewDictionaryValueWithAddress(
	context DictionaryCreationContext,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	address common.Address,
	keysAndValues ...Value,
) *DictionaryValue {

	context.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	if context.TracingEnabled() {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			context.ReportDictionaryValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	keysAndValuesCount := len(keysAndValues)
	if keysAndValuesCount%2 != 0 {
		panic("uneven number of keys and values")
	}

	constructor := func() *atree.OrderedMap {
		dictionary, err := atree.NewMap(
			context.Storage(),
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			dictionaryType,
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	// values are added to the dictionary after creation, not here
	v = newDictionaryValueFromConstructor(context, dictionaryType, 0, constructor)

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		existingValue := v.Insert(context, locationRange, key, value)
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			if v.IsResourceKinded(context) {
				panic(DuplicateKeyInResourceDictionaryError{
					LocationRange: locationRange,
				})
			}
		}
	}

	return v
}

func DictionaryElementSize(staticType *DictionaryStaticType) uint {
	keySize := staticType.KeyType.elementSize()
	valueSize := staticType.ValueType.elementSize()
	if keySize == 0 || valueSize == 0 {
		return 0
	}
	return keySize + valueSize
}

func newDictionaryValueWithIterator(
	context DictionaryCreationContext,
	locationRange LocationRange,
	staticType *DictionaryStaticType,
	count uint64,
	seed uint64,
	address common.Address,
	values func() (Value, Value),
) *DictionaryValue {
	context.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	if context.TracingEnabled() {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			context.ReportDictionaryValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.OrderedMap {
		orderedMap, err := atree.NewMapFromBatchData(
			context.Storage(),
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			staticType,
			newValueComparator(context, locationRange),
			newHashInputProvider(context, locationRange),
			seed,
			func() (atree.Value, atree.Value, error) {
				key, value := values()
				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return orderedMap
	}

	// values are added to the dictionary after creation, not here
	v = newDictionaryValueFromConstructor(context, staticType, count, constructor)

	return v
}

func newDictionaryValueFromConstructor(
	gauge common.MemoryGauge,
	staticType *DictionaryStaticType,
	count uint64,
	constructor func() *atree.OrderedMap,
) *DictionaryValue {

	elementSize := DictionaryElementSize(staticType)

	overheadUsage, dataSlabs, metaDataSlabs :=
		common.NewAtreeMapMemoryUsages(count, elementSize)
	common.UseMemory(gauge, overheadUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	return newDictionaryValueFromAtreeMap(
		gauge,
		staticType,
		elementSize,
		constructor(),
	)
}

func newDictionaryValueFromAtreeMap(
	gauge common.MemoryGauge,
	staticType *DictionaryStaticType,
	elementSize uint,
	atreeOrderedMap *atree.OrderedMap,
) *DictionaryValue {

	common.UseMemory(gauge, common.DictionaryValueBaseMemoryUsage)

	return &DictionaryValue{
		Type:        staticType,
		dictionary:  atreeOrderedMap,
		elementSize: elementSize,
	}
}

var _ Value = &DictionaryValue{}
var _ atree.Value = &DictionaryValue{}
var _ atree.WrapperValue = &DictionaryValue{}
var _ EquatableValue = &DictionaryValue{}
var _ ValueIndexableValue = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}
var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}
var _ atreeContainerBackedValue = &DictionaryValue{}

func (*DictionaryValue) IsValue() {}

func (*DictionaryValue) isAtreeContainerBackedValue() {}

func (v *DictionaryValue) Accept(context ValueVisitContext, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitDictionaryValue(context, v)
	if !descend {
		return
	}

	v.Walk(
		context,
		func(value Value) {
			value.Accept(context, visitor, locationRange)
		},
		locationRange,
	)
}

func (v *DictionaryValue) IterateKeys(
	interpreter *Interpreter,
	locationRange LocationRange,
	f func(key Value) (resume bool),
) {
	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)
	iterate := func(fn atree.MapElementIterationFunc) error {
		// Use NonReadOnlyIterator because we are not sure if f in
		// all uses of DictionaryValue.IterateKeys are always read-only.
		// TODO: determine if all uses of f are read-only.
		return v.dictionary.IterateKeys(
			valueComparator,
			hashInputProvider,
			fn,
		)
	}
	v.iterateKeys(interpreter, iterate, f)
}

func (v *DictionaryValue) iterateKeys(
	interpreter *Interpreter,
	atreeIterate func(fn atree.MapElementIterationFunc) error,
	f func(key Value) (resume bool),
) {
	iterate := func() {
		err := atreeIterate(func(key atree.Value) (resume bool, err error) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			resume = f(
				MustConvertStoredValue(interpreter, key),
			)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	interpreter.WithMutationPrevention(v.ValueID(), iterate)
}

func (v *DictionaryValue) IterateReadOnly(
	interpreter *Interpreter,
	locationRange LocationRange,
	f func(key, value Value) (resume bool),
) {
	iterate := func(fn atree.MapEntryIterationFunc) error {
		return v.dictionary.IterateReadOnly(
			fn,
		)
	}
	v.iterate(interpreter, iterate, f, locationRange)
}

func (v *DictionaryValue) Iterate(
	context ContainerMutationContext,
	locationRange LocationRange,
	f func(key, value Value) (resume bool),
) {
	valueComparator := newValueComparator(context, locationRange)
	hashInputProvider := newHashInputProvider(context, locationRange)
	iterate := func(fn atree.MapEntryIterationFunc) error {
		return v.dictionary.Iterate(
			valueComparator,
			hashInputProvider,
			fn,
		)
	}
	v.iterate(context, iterate, f, locationRange)
}

// IterateReadOnlyLoaded iterates over all LOADED key-value pairs of the array.
// DO NOT perform storage mutations in the callback!
func (v *DictionaryValue) IterateReadOnlyLoaded(
	context ContainerMutationContext,
	locationRange LocationRange,
	f func(key, value Value) (resume bool),
) {
	v.iterate(
		context,
		v.dictionary.IterateReadOnlyLoadedValues,
		f,
		locationRange,
	)
}

func (v *DictionaryValue) iterate(
	context ContainerMutationContext,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(key Value, value Value) (resume bool),
	locationRange LocationRange,
) {
	iterate := func() {
		err := atreeIterate(func(key, value atree.Value) (resume bool, err error) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			keyValue := MustConvertStoredValue(context, key)
			valueValue := MustConvertStoredValue(context, value)

			checkInvalidatedResourceOrResourceReference(keyValue, locationRange, context)
			checkInvalidatedResourceOrResourceReference(valueValue, locationRange, context)

			resume = f(
				keyValue,
				valueValue,
			)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	context.WithMutationPrevention(v.ValueID(), iterate)
}

type DictionaryKeyIterator struct {
	mapIterator atree.MapIterator
}

func (i DictionaryKeyIterator) NextKeyUnconverted() atree.Value {
	atreeValue, err := i.mapIterator.NextKey()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return atreeValue
}

func (i DictionaryKeyIterator) NextKey(gauge common.MemoryGauge) Value {
	atreeValue := i.NextKeyUnconverted()
	if atreeValue == nil {
		return nil
	}
	return MustConvertStoredValue(gauge, atreeValue)
}

func (i DictionaryKeyIterator) Next(gauge common.MemoryGauge) (Value, Value) {
	atreeKeyValue, atreeValue, err := i.mapIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	if atreeKeyValue == nil {
		return nil, nil
	}
	return MustConvertStoredValue(gauge, atreeKeyValue),
		MustConvertStoredValue(gauge, atreeValue)
}

func (v *DictionaryValue) Iterator() DictionaryKeyIterator {
	mapIterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return DictionaryKeyIterator{
		mapIterator: mapIterator,
	}
}

func (v *DictionaryValue) Walk(context ValueWalkContext, walkChild func(Value), locationRange LocationRange) {
	v.Iterate(
		context,
		locationRange,
		func(key, value Value) (resume bool) {
			walkChild(key)
			walkChild(value)
			return true
		},
	)
}

func (v *DictionaryValue) StaticType(_ ValueStaticTypeContext) StaticType {
	// TODO meter
	return v.Type
}

func (v *DictionaryValue) IsImportable(context ValueImportableContext, locationRange LocationRange) bool {
	importable := true
	v.Iterate(
		context,
		locationRange,
		func(key, value Value) (resume bool) {
			if !key.IsImportable(context, locationRange) || !value.IsImportable(context, locationRange) {
				importable = false
				// stop iteration
				return false
			}

			// continue iteration
			return true
		},
	)

	return importable
}

func (v *DictionaryValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *DictionaryValue) isInvalidatedResource(context ValueStaticTypeContext) bool {
	return v.isDestroyed || (v.dictionary == nil && v.IsResourceKinded(context))
}

func (v *DictionaryValue) IsStaleResource(context ValueStaticTypeContext) bool {
	return v.dictionary == nil && v.IsResourceKinded(context)
}

func (v *DictionaryValue) Destroy(context ResourceDestructionContext, locationRange LocationRange) {

	context.ReportComputation(common.ComputationKindDestroyDictionaryValue, 1)

	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			context.ReportDictionaryValueDestroyTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	valueID := v.ValueID()

	context.WithResourceDestruction(
		valueID,
		locationRange,
		func() {
			v.Iterate(
				context,
				locationRange,
				func(key, value Value) (resume bool) {
					// Resources cannot be keys at the moment, so should theoretically not be needed
					maybeDestroy(context, locationRange, key)
					maybeDestroy(context, locationRange, value)

					return true
				},
			)
		},
	)

	v.isDestroyed = true

	InvalidateReferencedResources(context, v, locationRange)

	v.dictionary = nil
}

func (v *DictionaryValue) ForEachKey(
	context InvocationContext,
	locationRange LocationRange,
	procedure FunctionValue,
) {
	keyType := v.SemaType(context).KeyType

	argumentTypes := []sema.Type{keyType}

	procedureFunctionType := procedure.FunctionType(context)
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	iterate := func() {
		err := v.dictionary.IterateReadOnlyKeys(
			func(item atree.Value) (bool, error) {
				key := MustConvertStoredValue(context, item)

				result := invokeFunctionValue(
					context,
					procedure,
					[]Value{key},
					nil,
					argumentTypes,
					parameterTypes,
					returnType,
					nil,
					locationRange,
				)

				shouldContinue, ok := result.(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return bool(shouldContinue), nil
			},
		)

		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	context.WithMutationPrevention(v.ValueID(), iterate)
}

func (v *DictionaryValue) ContainsKey(
	context ValueComparisonContext,
	locationRange LocationRange,
	keyValue Value,
) BoolValue {

	valueComparator := newValueComparator(context, locationRange)
	hashInputProvider := newHashInputProvider(context, locationRange)

	exists, err := v.dictionary.Has(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return BoolValue(exists)
}

func (v *DictionaryValue) Get(
	context ValueComparisonContext,
	locationRange LocationRange,
	keyValue Value,
) (Value, bool) {

	valueComparator := newValueComparator(context, locationRange)
	hashInputProvider := newHashInputProvider(context, locationRange)

	storedValue, err := v.dictionary.Get(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil, false
		}
		panic(errors.NewExternalError(err))
	}

	return MustConvertStoredValue(context, storedValue), true
}

func (v *DictionaryValue) GetKey(context ValueComparisonContext, locationRange LocationRange, keyValue Value) Value {
	value, ok := v.Get(context, locationRange, keyValue)
	if ok {
		return NewSomeValueNonCopying(context, value)
	}

	return Nil
}

func (v *DictionaryValue) SetKey(context ContainerMutationContext, locationRange LocationRange, keyValue Value, value Value) {
	context.ValidateMutation(v.ValueID(), locationRange)

	checkContainerMutation(context, v.Type.KeyType, keyValue, locationRange)
	checkContainerMutation(
		context,
		&OptionalStaticType{ // intentionally unmetered
			Type: v.Type.ValueType,
		},
		value,
		locationRange,
	)

	var existingValue Value
	switch value := value.(type) {
	case *SomeValue:
		innerValue := value.InnerValue()
		existingValue = v.Insert(context, locationRange, keyValue, innerValue)

	case NilValue:
		existingValue = v.Remove(context, locationRange, keyValue)

	case placeholderValue:
		// NO-OP

	default:
		panic(errors.NewUnreachableError())
	}

	if existingValue != nil {
		checkResourceLoss(context, existingValue, locationRange)
	}
}

func (v *DictionaryValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *DictionaryValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(NoOpStringContext{}, seenReferences, EmptyLocationRange)
}

func (v *DictionaryValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {

	pairs := make([]struct {
		Key   string
		Value string
	}, v.Count())

	index := 0

	v.Iterate(
		context,
		locationRange,
		func(key, value Value) (resume bool) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			pairs[index] = struct {
				Key   string
				Value string
			}{
				Key:   key.MeteredString(context, seenReferences, locationRange),
				Value: value.MeteredString(context, seenReferences, locationRange),
			}
			index++
			return true
		},
	)

	// len = len(open-brace) + len(close-brace) + (n times colon+space) + ((n-1) times comma+space)
	//     = 2 + 2n + 2n - 2
	//     = 4n + 2 - 2
	//
	// Since (-2) only occurs if its non-empty (i.e: n>0), ignore the (-2). i.e: overestimate
	//    len = 4n + 2
	//
	// String of each key and value are metered separately.
	strLen := len(pairs)*4 + 2

	common.UseMemory(context, common.NewRawStringMemoryUsage(strLen))

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			context.ReportDictionaryValueGetMemberTrace(
				typeInfo,
				count,
				name,
				time.Since(startTime),
			)
		}()
	}

	switch name {
	case "length":
		return NewIntValueFromInt64(context, int64(v.Count()))

	case "keys":

		iterator, err := v.dictionary.ReadOnlyIterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			context,
			NewVariableSizedStaticType(context, v.Type.KeyType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				key, err := iterator.NextKey()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if key == nil {
					return nil
				}

				return MustConvertStoredValue(context, key).
					Transfer(
						context,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
						false, // value is an element of parent container because it is returned from iterator.
					)
			},
		)

	case "values":

		// Use ReadOnlyIterator here because new ArrayValue is created with copied elements (not removed) from original.
		iterator, err := v.dictionary.ReadOnlyIterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			context,
			NewVariableSizedStaticType(context, v.Type.ValueType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				value, err := iterator.NextValue()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if value == nil {
					return nil
				}

				return MustConvertStoredValue(context, value).
					Transfer(
						context,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
						false, // value is an element of parent container because it is returned from iterator.
					)
			},
		)
	}

	return context.GetMethod(v, name, locationRange)
}

func (v *DictionaryValue) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	switch name {
	case sema.DictionaryTypeRemoveFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.DictionaryRemoveFunctionType(
				v.SemaType(context),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				keyValue := invocation.Arguments[0]

				return v.Remove(
					invocation.InvocationContext,
					invocation.LocationRange,
					keyValue,
				)
			},
		)

	case sema.DictionaryTypeInsertFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.DictionaryInsertFunctionType(
				v.SemaType(context),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				keyValue := invocation.Arguments[0]
				newValue := invocation.Arguments[1]

				return v.Insert(
					invocation.InvocationContext,
					invocation.LocationRange,
					keyValue,
					newValue,
				)
			},
		)

	case sema.DictionaryTypeContainsKeyFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.DictionaryContainsKeyFunctionType(
				v.SemaType(context),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				return v.ContainsKey(
					invocation.InvocationContext,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)
	case sema.DictionaryTypeForEachKeyFunctionName:
		return NewBoundHostFunctionValue(
			context,
			v,
			sema.DictionaryForEachKeyFunctionType(
				v.SemaType(context),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				invocationContext := invocation.InvocationContext

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v.ForEachKey(
					invocationContext,
					invocation.LocationRange,
					funcArgument,
				)

				return Void
			},
		)
	}

	return nil
}

func (v *DictionaryValue) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Dictionaries have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return int(v.dictionary.Count())
}

func (v *DictionaryValue) RemoveKey(context ContainerMutationContext, locationRange LocationRange, key Value) Value {
	return v.Remove(context, locationRange, key)
}

func (v *DictionaryValue) RemoveWithoutTransfer(
	context ContainerMutationContext,
	locationRange LocationRange,
	keyValue atree.Value,
) (
	existingKeyStorable,
	existingValueStorable atree.Storable,
) {

	context.ValidateMutation(v.ValueID(), locationRange)

	valueComparator := newValueComparator(context, locationRange)
	hashInputProvider := newHashInputProvider(context, locationRange)

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	var err error
	existingKeyStorable, existingValueStorable, err = v.dictionary.Remove(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil, nil
		}
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	context.MaybeValidateAtreeStorage()

	return existingKeyStorable, existingValueStorable
}

func (v *DictionaryValue) Remove(
	context ContainerMutationContext,
	locationRange LocationRange,
	keyValue Value,
) OptionalValue {

	existingKeyStorable, existingValueStorable := v.RemoveWithoutTransfer(context, locationRange, keyValue)

	if existingKeyStorable == nil {
		return NilOptionalValue
	}

	storage := context.Storage()

	// Key

	existingKeyValue := StoredValue(context, existingKeyStorable, storage)
	existingKeyValue.DeepRemove(context, true) // existingValue is standalone because it was removed from parent container.
	RemoveReferencedSlab(context, existingKeyStorable)

	// Value

	existingValue := StoredValue(context, existingValueStorable, storage).
		Transfer(
			context,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
			true, // value is standalone because it was removed from parent container.
		)

	return NewSomeValueNonCopying(context, existingValue)
}

func (v *DictionaryValue) InsertKey(context ContainerMutationContext, locationRange LocationRange, key Value, value Value) {
	v.SetKey(context, locationRange, key, value)
}

func (v *DictionaryValue) InsertWithoutTransfer(
	context ContainerMutationContext,
	locationRange LocationRange,
	keyValue, value atree.Value,
) (existingValueStorable atree.Storable) {

	context.ValidateMutation(v.ValueID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(v.dictionary.Count(), v.elementSize, false)
	common.UseMemory(context, common.AtreeMapElementOverhead)
	common.UseMemory(context, dataSlabs)
	common.UseMemory(context, metaDataSlabs)

	valueComparator := newValueComparator(context, locationRange)
	hashInputProvider := newHashInputProvider(context, locationRange)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	var err error
	existingValueStorable, err = v.dictionary.Set(
		valueComparator,
		hashInputProvider,
		keyValue,
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	context.MaybeValidateAtreeStorage()

	return existingValueStorable
}

func (v *DictionaryValue) Insert(
	context ContainerMutationContext,
	locationRange LocationRange,
	keyValue, value Value,
) OptionalValue {

	address := v.dictionary.Address()

	preventTransfer := map[atree.ValueID]struct{}{
		v.ValueID(): {},
	}

	keyValue = keyValue.Transfer(
		context,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
		true, // keyValue is standalone before it is inserted into parent container.
	)

	value = value.Transfer(
		context,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
		true, // value is standalone before it is inserted into parent container.
	)

	checkContainerMutation(context, v.Type.KeyType, keyValue, locationRange)
	checkContainerMutation(context, v.Type.ValueType, value, locationRange)

	existingValueStorable := v.InsertWithoutTransfer(context, locationRange, keyValue, value)

	if existingValueStorable == nil {
		return NilOptionalValue
	}

	// At this point, existingValueStorable is not nil, which means previous op updated existing
	// dictionary element (instead of inserting new element).

	// When existing dictionary element is updated, atree.OrderedMap reuses existing stored key
	// so new key isn't stored or referenced in atree.OrderedMap.  This aspect of atree cannot change
	// without API changes in atree to return existing key storable for updated element.

	// Given this, remove the transferred key used to update existing dictionary element to
	// prevent transferred key (in owner address) remaining in storage when it isn't
	// referenced from dictionary.

	// Remove content of transferred keyValue.
	keyValue.DeepRemove(context, true)

	// Remove slab containing transferred keyValue from storage if needed.
	// Currently, we only need to handle enum composite type because it is the only type that:
	// - can be used as dictionary key (hashable) and
	// - is transferred to its own slab.
	if keyComposite, ok := keyValue.(*CompositeValue); ok {

		// Get SlabID of transferred enum value.
		keyCompositeSlabID := keyComposite.SlabID()

		if keyCompositeSlabID == atree.SlabIDUndefined {
			// It isn't possible for transferred enum value to be inlined in another container
			// (SlabID as SlabIDUndefined) because it is transferred from stack by itself.
			panic(errors.NewUnexpectedError("transferred enum value as dictionary key should not be inlined"))
		}

		// Remove slab containing transferred enum value from storage.
		RemoveReferencedSlab(context, atree.SlabIDStorable(keyCompositeSlabID))
	}

	storage := context.Storage()

	existingValue := StoredValue(
		context,
		existingValueStorable,
		storage,
	).Transfer(
		context,
		locationRange,
		atree.Address{},
		true,
		existingValueStorable,
		nil,
		true, // existingValueStorable is standalone after it is overwritten in parent container.
	)

	return NewSomeValueNonCopying(context, existingValue)
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			context.ReportDictionaryValueConformsToStaticTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	staticType, ok := v.StaticType(context).(*DictionaryStaticType)
	if !ok {
		return false
	}

	keyType := staticType.KeyType
	valueType := staticType.ValueType

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		// Check the key

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryKey := MustConvertStoredValue(context, key)

		if !IsSubType(context, entryKey.StaticType(context), keyType) {
			return false
		}

		if !entryKey.ConformsToStaticType(
			context,
			locationRange,
			results,
		) {
			return false
		}

		// Check the value

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryValue := MustConvertStoredValue(context, value)

		if !IsSubType(context, entryValue.StaticType(context), valueType) {
			return false
		}

		if !entryValue.ConformsToStaticType(
			context,
			locationRange,
			results,
		) {
			return false
		}
	}
}

func (v *DictionaryValue) Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool {

	otherDictionary, ok := other.(*DictionaryValue)
	if !ok {
		return false
	}

	if v.Count() != otherDictionary.Count() {
		return false
	}

	if !v.Type.Equal(otherDictionary.Type) {
		return false
	}

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		// Do NOT use an iterator, as other value may be stored in another account,
		// leading to a different iteration order, as the storage ID is used in the seed
		otherValue, otherValueExists :=
			otherDictionary.Get(
				context,
				locationRange,
				MustConvertStoredValue(context, key),
			)

		if !otherValueExists {
			return false
		}

		equatableValue, ok := MustConvertStoredValue(context, value).(EquatableValue)
		if !ok || !equatableValue.Equal(context, locationRange, otherValue) {
			return false
		}
	}
}

func (v *DictionaryValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	// NOTE: Need to change DictionaryValue.UnwrapAtreeValue()
	// if DictionaryValue is stored with wrapping.
	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *DictionaryValue) UnwrapAtreeValue() (atree.Value, uint64) {
	// Wrapper size is 0 because DictionaryValue is stored as
	// atree.OrderedMap without any physical wrapping (see DictionaryValue.Storable()).
	return v.dictionary, 0
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

func (v *DictionaryValue) Transfer(
	context ValueTransferContext,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.ValueID]struct{},
	hasNoParentContainer bool,
) Value {

	context.ReportComputation(
		common.ComputationKindTransferDictionaryValue,
		uint(v.Count()),
	)

	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			context.ReportDictionaryValueTransferTrace(
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

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(context)

	if needsStoreTo || !isResourceKinded {

		valueComparator := newValueComparator(context, locationRange)
		hashInputProvider := newHashInputProvider(context, locationRange)

		// Use non-readonly iterator here because iterated
		// value can be removed if remove parameter is true.
		iterator, err := v.dictionary.Iterator(valueComparator, hashInputProvider)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementCount := v.dictionary.Count()

		elementOverhead, dataUse, metaDataUse := common.NewAtreeMapMemoryUsages(
			elementCount,
			v.elementSize,
		)
		common.UseMemory(context, elementOverhead)
		common.UseMemory(context, dataUse)
		common.UseMemory(context, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(
			elementCount,
			v.elementSize,
		)
		common.UseMemory(context, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			context.Storage(),
			address,
			atree.NewDefaultDigesterBuilder(),
			v.dictionary.Type(),
			valueComparator,
			hashInputProvider,
			v.dictionary.Seed(),
			func() (atree.Value, atree.Value, error) {

				atreeKey, atreeValue, err := iterator.Next()
				if err != nil {
					return nil, nil, err
				}
				if atreeKey == nil || atreeValue == nil {
					return nil, nil, nil
				}

				key := MustConvertStoredValue(context, atreeKey).
					Transfer(
						context,
						locationRange,
						address,
						remove,
						nil,
						preventTransfer,
						false, // atreeKey has parent container because it is returned from iterator.
					)

				value := MustConvertStoredValue(context, atreeValue).
					Transfer(
						context,
						locationRange,
						address,
						remove,
						nil,
						preventTransfer,
						false, // atreeValue has parent container because it is returned from iterator.
					)

				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
				RemoveReferencedSlab(context, keyStorable)
				RemoveReferencedSlab(context, valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			context.MaybeValidateAtreeValue(v.dictionary)
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

		InvalidateReferencedResources(context, v, locationRange)

		v.dictionary = nil
	}

	res := newDictionaryValueFromAtreeMap(
		context,
		v.Type,
		v.elementSize,
		dictionary,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *DictionaryValue) Clone(context ValueCloneContext) Value {
	valueComparator := newValueComparator(context, EmptyLocationRange)
	hashInputProvider := newHashInputProvider(context, EmptyLocationRange)

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	orderedMap, err := atree.NewMapFromBatchData(
		context.Storage(),
		v.StorageAddress(),
		atree.NewDefaultDigesterBuilder(),
		v.dictionary.Type(),
		valueComparator,
		hashInputProvider,
		v.dictionary.Seed(),
		func() (atree.Value, atree.Value, error) {

			atreeKey, atreeValue, err := iterator.Next()
			if err != nil {
				return nil, nil, err
			}
			if atreeKey == nil || atreeValue == nil {
				return nil, nil, nil
			}

			key := MustConvertStoredValue(context, atreeKey).
				Clone(context)

			value := MustConvertStoredValue(context, atreeValue).
				Clone(context)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	dictionary := newDictionaryValueFromAtreeMap(
		context,
		v.Type,
		v.elementSize,
		orderedMap,
	)

	dictionary.semaType = v.semaType
	dictionary.isResourceKinded = v.isResourceKinded
	dictionary.isDestroyed = v.isDestroyed

	return dictionary
}

func (v *DictionaryValue) DeepRemove(context ValueRemoveContext, hasNoParentContainer bool) {

	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			context.ReportDictionaryValueDeepRemoveTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {

		key := StoredValue(context, keyStorable, storage)
		key.DeepRemove(context, false) // key is an element of v.dictionary because it is from PopIterate() callback.
		RemoveReferencedSlab(context, keyStorable)

		value := StoredValue(context, valueStorable, storage)
		value.DeepRemove(context, false) // value is an element of v.dictionary because it is from PopIterate() callback.
		RemoveReferencedSlab(context, valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(v.dictionary)
	if hasNoParentContainer {
		context.MaybeValidateAtreeStorage()
	}
}

func (v *DictionaryValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *DictionaryValue) SlabID() atree.SlabID {
	return v.dictionary.SlabID()
}

func (v *DictionaryValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *DictionaryValue) ValueID() atree.ValueID {
	return v.dictionary.ValueID()
}

func (v *DictionaryValue) SemaType(typeConverter TypeConverter) *sema.DictionaryType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = MustConvertStaticToSemaType(v.Type, typeConverter).(*sema.DictionaryType)
	}
	return v.semaType
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *DictionaryValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(context).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

func (v *DictionaryValue) SetType(staticType *DictionaryStaticType) {
	v.Type = staticType
	err := v.dictionary.SetType(staticType)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (v *DictionaryValue) AtreeMap() *atree.OrderedMap {
	return v.dictionary
}

func (v *DictionaryValue) ElementSize() uint {
	return v.elementSize
}

func (v *DictionaryValue) Inlined() bool {
	return v.dictionary.Inlined()
}
