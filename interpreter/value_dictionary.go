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
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {
	return NewDictionaryValueWithAddress(
		interpreter,
		locationRange,
		dictionaryType,
		common.ZeroAddress,
		keysAndValues...,
	)
}

func NewDictionaryValueWithAddress(
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	address common.Address,
	keysAndValues ...Value,
) *DictionaryValue {

	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			config.Tracer.ReportDictionaryValueConstructTrace(
				interpreter,
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
			config.Storage,
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
	v = newDictionaryValueFromConstructor(interpreter, dictionaryType, 0, constructor)

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		existingValue := v.Insert(interpreter, locationRange, key, value)
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			if v.IsResourceKinded(interpreter) {
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
	interpreter *Interpreter,
	locationRange LocationRange,
	staticType *DictionaryStaticType,
	count uint64,
	seed uint64,
	address common.Address,
	values func() (Value, Value),
) *DictionaryValue {
	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			config.Tracer.ReportDictionaryValueConstructTrace(
				interpreter,
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.OrderedMap {
		orderedMap, err := atree.NewMapFromBatchData(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			staticType,
			newValueComparator(interpreter, locationRange),
			newHashInputProvider(interpreter, locationRange),
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
	v = newDictionaryValueFromConstructor(interpreter, staticType, count, constructor)

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

	return NewDictionaryValueFromAtreeMap(
		gauge,
		staticType,
		elementSize,
		constructor(),
	)
}

func NewDictionaryValueFromAtreeMap(
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
var _ EquatableValue = &DictionaryValue{}
var _ ValueIndexableValue = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}
var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}

func (*DictionaryValue) isValue() {}

func (v *DictionaryValue) Accept(interpreter *Interpreter, visitor Visitor, locationRange LocationRange) {
	descend := visitor.VisitDictionaryValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(
		interpreter,
		func(value Value) {
			value.Accept(interpreter, visitor, locationRange)
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

	interpreter.withMutationPrevention(v.ValueID(), iterate)
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
	interpreter *Interpreter,
	locationRange LocationRange,
	f func(key, value Value) (resume bool),
) {
	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)
	iterate := func(fn atree.MapEntryIterationFunc) error {
		return v.dictionary.Iterate(
			valueComparator,
			hashInputProvider,
			fn,
		)
	}
	v.iterate(interpreter, iterate, f, locationRange)
}

// IterateReadOnlyLoaded iterates over all LOADED key-valye pairs of the array.
// DO NOT perform storage mutations in the callback!
func (v *DictionaryValue) IterateReadOnlyLoaded(
	interpreter *Interpreter,
	locationRange LocationRange,
	f func(key, value Value) (resume bool),
) {
	v.iterate(
		interpreter,
		v.dictionary.IterateReadOnlyLoadedValues,
		f,
		locationRange,
	)
}

func (v *DictionaryValue) iterate(
	interpreter *Interpreter,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(key Value, value Value) (resume bool),
	locationRange LocationRange,
) {
	iterate := func() {
		err := atreeIterate(func(key, value atree.Value) (resume bool, err error) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			keyValue := MustConvertStoredValue(interpreter, key)
			valueValue := MustConvertStoredValue(interpreter, value)

			interpreter.checkInvalidatedResourceOrResourceReference(keyValue, locationRange)
			interpreter.checkInvalidatedResourceOrResourceReference(valueValue, locationRange)

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

	interpreter.withMutationPrevention(v.ValueID(), iterate)
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

func (v *DictionaryValue) Walk(interpreter *Interpreter, walkChild func(Value), locationRange LocationRange) {
	v.Iterate(
		interpreter,
		locationRange,
		func(key, value Value) (resume bool) {
			walkChild(key)
			walkChild(value)
			return true
		},
	)
}

func (v *DictionaryValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *DictionaryValue) IsImportable(inter *Interpreter, locationRange LocationRange) bool {
	importable := true
	v.Iterate(
		inter,
		locationRange,
		func(key, value Value) (resume bool) {
			if !key.IsImportable(inter, locationRange) || !value.IsImportable(inter, locationRange) {
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

func (v *DictionaryValue) isInvalidatedResource(interpreter *Interpreter) bool {
	return v.isDestroyed || (v.dictionary == nil && v.IsResourceKinded(interpreter))
}

func (v *DictionaryValue) IsStaleResource(interpreter *Interpreter) bool {
	return v.dictionary == nil && v.IsResourceKinded(interpreter)
}

func (v *DictionaryValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyDictionaryValue, 1)

	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			config.Tracer.ReportDictionaryValueDestroyTrace(
				interpreter,
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
			v.Iterate(
				interpreter,
				locationRange,
				func(key, value Value) (resume bool) {
					// Resources cannot be keys at the moment, so should theoretically not be needed
					maybeDestroy(interpreter, locationRange, key)
					maybeDestroy(interpreter, locationRange, value)

					return true
				},
			)
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, locationRange)

	v.dictionary = nil
}

func (v *DictionaryValue) ForEachKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) {
	keyType := v.SemaType(interpreter).KeyType

	argumentTypes := []sema.Type{keyType}

	procedureFunctionType := procedure.FunctionType()
	parameterTypes := procedureFunctionType.ParameterTypes()
	returnType := procedureFunctionType.ReturnTypeAnnotation.Type

	iterate := func() {
		err := v.dictionary.IterateReadOnlyKeys(
			func(item atree.Value) (bool, error) {
				key := MustConvertStoredValue(interpreter, item)

				result := interpreter.invokeFunctionValue(
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

	interpreter.withMutationPrevention(v.ValueID(), iterate)
}

func (v *DictionaryValue) ContainsKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) BoolValue {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	exists, err := v.dictionary.Has(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return AsBoolValue(exists)
}

func (v *DictionaryValue) Get(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) (Value, bool) {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

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

	return MustConvertStoredValue(interpreter, storedValue), true
}

func (v *DictionaryValue) GetKey(interpreter *Interpreter, locationRange LocationRange, keyValue Value) Value {
	value, ok := v.Get(interpreter, locationRange, keyValue)
	if ok {
		return NewSomeValueNonCopying(interpreter, value)
	}

	return Nil
}

func (v *DictionaryValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
	value Value,
) {
	interpreter.validateMutation(v.ValueID(), locationRange)

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(
		&OptionalStaticType{ // intentionally unmetered
			Type: v.Type.ValueType,
		},
		value,
		locationRange,
	)

	var existingValue Value
	switch value := value.(type) {
	case *SomeValue:
		innerValue := value.InnerValue(interpreter, locationRange)
		existingValue = v.Insert(interpreter, locationRange, keyValue, innerValue)

	case NilValue:
		existingValue = v.Remove(interpreter, locationRange, keyValue)

	case placeholderValue:
		// NO-OP

	default:
		panic(errors.NewUnreachableError())
	}

	if existingValue != nil {
		interpreter.checkResourceLoss(existingValue, locationRange)
	}
}

func (v *DictionaryValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *DictionaryValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences, EmptyLocationRange)
}

func (v *DictionaryValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {

	pairs := make([]struct {
		Key   string
		Value string
	}, v.Count())

	index := 0

	v.Iterate(
		interpreter,
		locationRange,
		func(key, value Value) (resume bool) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			pairs[index] = struct {
				Key   string
				Value string
			}{
				Key:   key.MeteredString(interpreter, seenReferences, locationRange),
				Value: value.MeteredString(interpreter, seenReferences, locationRange),
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

	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(strLen))

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			config.Tracer.ReportDictionaryValueGetMemberTrace(
				interpreter,
				typeInfo,
				count,
				name,
				time.Since(startTime),
			)
		}()
	}

	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "keys":

		iterator, err := v.dictionary.ReadOnlyIterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.KeyType),
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

				return MustConvertStoredValue(interpreter, key).
					Transfer(
						interpreter,
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
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.ValueType),
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

				return MustConvertStoredValue(interpreter, value).
					Transfer(
						interpreter,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
						false, // value is an element of parent container because it is returned from iterator.
					)
			})

	case "remove":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.DictionaryRemoveFunctionType(
				v.SemaType(interpreter),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				keyValue := invocation.Arguments[0]

				return v.Remove(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
				)
			},
		)

	case "insert":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.DictionaryInsertFunctionType(
				v.SemaType(interpreter),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				keyValue := invocation.Arguments[0]
				newValue := invocation.Arguments[1]

				return v.Insert(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
					newValue,
				)
			},
		)

	case "containsKey":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.DictionaryContainsKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				return v.ContainsKey(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)
	case "forEachKey":
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.DictionaryForEachKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(v *DictionaryValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v.ForEachKey(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)

				return Void
			},
		)
	}

	return nil
}

func (v *DictionaryValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Dictionaries have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return int(v.dictionary.Count())
}

func (v *DictionaryValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	return v.Remove(interpreter, locationRange, key)
}

func (v *DictionaryValue) RemoveWithoutTransfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue atree.Value,
) (
	existingKeyStorable,
	existingValueStorable atree.Storable,
) {

	interpreter.validateMutation(v.ValueID(), locationRange)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

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

	interpreter.maybeValidateAtreeValue(v.dictionary)
	interpreter.maybeValidateAtreeStorage()

	return existingKeyStorable, existingValueStorable
}

func (v *DictionaryValue) Remove(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) OptionalValue {

	existingKeyStorable, existingValueStorable := v.RemoveWithoutTransfer(interpreter, locationRange, keyValue)

	if existingKeyStorable == nil {
		return NilOptionalValue
	}

	storage := interpreter.Storage()

	// Key

	existingKeyValue := StoredValue(interpreter, existingKeyStorable, storage)
	existingKeyValue.DeepRemove(interpreter, true) // existingValue is standalone because it was removed from parent container.
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existingValue := StoredValue(interpreter, existingValueStorable, storage).
		Transfer(
			interpreter,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
			true, // value is standalone because it was removed from parent container.
		)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

func (v *DictionaryValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key, value Value,
) {
	v.SetKey(interpreter, locationRange, key, value)
}

func (v *DictionaryValue) InsertWithoutTransfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue, value atree.Value,
) (existingValueStorable atree.Storable) {

	interpreter.validateMutation(v.ValueID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(v.dictionary.Count(), v.elementSize, false)
	common.UseMemory(interpreter, common.AtreeMapElementOverhead)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

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

	interpreter.maybeValidateAtreeValue(v.dictionary)
	interpreter.maybeValidateAtreeStorage()

	return existingValueStorable
}

func (v *DictionaryValue) Insert(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue, value Value,
) OptionalValue {

	address := v.dictionary.Address()

	preventTransfer := map[atree.ValueID]struct{}{
		v.ValueID(): {},
	}

	keyValue = keyValue.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
		true, // keyValue is standalone before it is inserted into parent container.
	)

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
		true, // value is standalone before it is inserted into parent container.
	)

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(v.Type.ValueType, value, locationRange)

	existingValueStorable := v.InsertWithoutTransfer(interpreter, locationRange, keyValue, value)

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
	keyValue.DeepRemove(interpreter, true)

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
		interpreter.RemoveReferencedSlab(atree.SlabIDStorable(keyCompositeSlabID))
	}

	storage := interpreter.Storage()

	existingValue := StoredValue(
		interpreter,
		existingValueStorable,
		storage,
	).Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		existingValueStorable,
		nil,
		true, // existingValueStorable is standalone after it is overwritten in parent container.
	)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			config.Tracer.ReportDictionaryValueConformsToStaticTypeTrace(
				interpreter,
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	staticType, ok := v.StaticType(interpreter).(*DictionaryStaticType)
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
		entryKey := MustConvertStoredValue(interpreter, key)

		if !interpreter.IsSubType(entryKey.StaticType(interpreter), keyType) {
			return false
		}

		if !entryKey.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}

		// Check the value

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryValue := MustConvertStoredValue(interpreter, value)

		if !interpreter.IsSubType(entryValue.StaticType(interpreter), valueType) {
			return false
		}

		if !entryValue.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}
	}
}

func (v *DictionaryValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {

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
				interpreter,
				locationRange,
				MustConvertStoredValue(interpreter, key),
			)

		if !otherValueExists {
			return false
		}

		equatableValue, ok := MustConvertStoredValue(interpreter, value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}
}

func (v *DictionaryValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

func (v *DictionaryValue) Transfer(
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
		common.ComputationKindTransferDictionaryValue,
		uint(v.Count()),
	)

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			config.Tracer.ReportDictionaryValueTransferTrace(
				interpreter,
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
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		valueComparator := newValueComparator(interpreter, locationRange)
		hashInputProvider := newHashInputProvider(interpreter, locationRange)

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
		common.UseMemory(interpreter, elementOverhead)
		common.UseMemory(interpreter, dataUse)
		common.UseMemory(interpreter, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(
			elementCount,
			v.elementSize,
		)
		common.UseMemory(config.MemoryGauge, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			config.Storage,
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

				key := MustConvertStoredValue(interpreter, atreeKey).
					Transfer(
						interpreter,
						locationRange,
						address,
						remove,
						nil,
						preventTransfer,
						false, // atreeKey has parent container because it is returned from iterator.
					)

				value := MustConvertStoredValue(interpreter, atreeValue).
					Transfer(
						interpreter,
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
				interpreter.RemoveReferencedSlab(keyStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			interpreter.maybeValidateAtreeValue(v.dictionary)
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

		v.dictionary = nil
	}

	res := NewDictionaryValueFromAtreeMap(
		interpreter,
		v.Type,
		v.elementSize,
		dictionary,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *DictionaryValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	valueComparator := newValueComparator(interpreter, EmptyLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, EmptyLocationRange)

	iterator, err := v.dictionary.ReadOnlyIterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	orderedMap, err := atree.NewMapFromBatchData(
		config.Storage,
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

			key := MustConvertStoredValue(interpreter, atreeKey).
				Clone(interpreter)

			value := MustConvertStoredValue(interpreter, atreeValue).
				Clone(interpreter)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	dictionary := NewDictionaryValueFromAtreeMap(
		interpreter,
		v.Type,
		v.elementSize,
		orderedMap,
	)

	dictionary.semaType = v.semaType
	dictionary.isResourceKinded = v.isResourceKinded
	dictionary.isDestroyed = v.isDestroyed

	return dictionary
}

func (v *DictionaryValue) DeepRemove(interpreter *Interpreter, hasNoParentContainer bool) {

	config := interpreter.SharedState.Config

	if config.Tracer.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			config.Tracer.ReportDictionaryValueDeepRemoveTrace(
				interpreter,
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {

		key := StoredValue(interpreter, keyStorable, storage)
		key.DeepRemove(interpreter, false) // key is an element of v.dictionary because it is from PopIterate() callback.
		interpreter.RemoveReferencedSlab(keyStorable)

		value := StoredValue(interpreter, valueStorable, storage)
		value.DeepRemove(interpreter, false) // value is an element of v.dictionary because it is from PopIterate() callback.
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	interpreter.maybeValidateAtreeValue(v.dictionary)
	if hasNoParentContainer {
		interpreter.maybeValidateAtreeStorage()
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

func (v *DictionaryValue) SemaType(interpreter *Interpreter) *sema.DictionaryType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = interpreter.MustConvertStaticToSemaType(v.Type).(*sema.DictionaryType)
	}
	return v.semaType
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *DictionaryValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
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
