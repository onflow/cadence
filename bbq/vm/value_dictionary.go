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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type DictionaryValue struct {
	dictionary  map[Value]Value
	Type        *interpreter.DictionaryStaticType
	elementSize uint
	address     common.Address
}

var _ Value = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}

//var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}

func NewDictionaryValue(
	config *Config,
	dictionaryType *interpreter.DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {

	keysAndValuesCount := len(keysAndValues)
	if keysAndValuesCount%2 != 0 {
		panic("uneven number of keys and values")
	}

	dictionary := make(map[Value]Value)

	// values are added to the dictionary after creation, not here

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]

		existingValue, _ := dictionary[key]
		dictionary[key] = value
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			//if v.IsResourceKinded() {
			//	panic(interpreter.DuplicateKeyInResourceDictionaryError{})
			//}
		}
	}

	return &DictionaryValue{
		Type:        dictionaryType,
		dictionary:  dictionary,
		elementSize: uint(len(dictionary)),
		address:     common.ZeroAddress,
	}
}

//func newDictionaryValueFromConstructor(
//	staticType *interpreter.DictionaryStaticType,
//	constructor func() *atree.OrderedMap,
//) *DictionaryValue {
//
//	elementSize := interpreter.DictionaryElementSize(staticType)
//	return newDictionaryValueFromAtreeMap(
//		staticType,
//		elementSize,
//		constructor(),
//	)
//}

func newDictionaryValueFromAtreeMap(
	staticType *interpreter.DictionaryStaticType,
	elementSize uint,
	atreeOrderedMap map[Value]Value,
) *DictionaryValue {

	return &DictionaryValue{
		Type:        staticType,
		dictionary:  atreeOrderedMap,
		elementSize: elementSize,
	}
}

func (*DictionaryValue) isValue() {}

func (v *DictionaryValue) StaticType(*Config) StaticType {
	return v.Type
}

func (v *DictionaryValue) GetMember(config *Config, name string) Value {
	// TODO:
	return nil
}

func (v *DictionaryValue) SetMember(conf *Config, name string, value Value) {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Insert(
	conf *Config,
	keyValue, value Value,
) Value /* TODO: OptionalValue*/ {

	address := atree.Address(v.address)

	keyValue = keyValue.Transfer(
		conf,
		address,
		true,
		nil,
	)

	value = value.Transfer(
		conf,
		address,
		true,
		nil,
	)

	//interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	//interpreter.checkContainerMutation(v.Type.ValueType, value, locationRange)

	existingValue := v.InsertWithoutTransfer(conf, keyValue, value)

	if existingValue == nil {
		return Nil // TODO: NilOptionalValue
	}

	existingValue = existingValue.Transfer(
		conf,
		atree.Address{},
		true,
		nil,
	)

	return NewSomeValueNonCopying(existingValue)
}

func (v *DictionaryValue) InsertWithoutTransfer(config *Config, key, value Value) (existingValue Value) {

	//interpreter.validateMutation(v.StorageID(), locationRange)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	existingValue, _ = v.dictionary[key]
	v.dictionary[key] = value

	//interpreter.maybeValidateAtreeValue(v.dictionary)

	return existingValue
}

//func (v *DictionaryValue) SlabID() atree.SlabID {
//	return v.dictionary.SlabID()
//}

func (v *DictionaryValue) IsResourceKinded() bool {
	// TODO:
	return false
}

func (v *DictionaryValue) String() string {
	//TODO implement me
	panic("implement me")
}

func (v *DictionaryValue) Transfer(
	config *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded()

	if needsStoreTo || !isResourceKinded {
		newDictionary := make(map[Value]Value, len(dictionary))

		for key, value := range dictionary {
			newDictionary[key] = value.Transfer(config, address, remove, storable)
		}

		dictionary = newDictionary
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		//invalidateReferencedResources(config, v)

		v.dictionary = nil
	}

	res := newDictionaryValueFromAtreeMap(
		v.Type,
		v.elementSize,
		dictionary,
	)

	res.address = common.Address(address)

	return res
}

func (v *DictionaryValue) Destroy(*Config) {
	v.dictionary = nil
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *DictionaryValue) StorageAddress() atree.Address {
	return atree.Address(v.address)
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

//func (v *DictionaryValue) ValueID() atree.ValueID {
//	return v.dictionary.ValueID()
//}

func (v *DictionaryValue) IsStaleResource() bool {
	return v.dictionary == nil && v.IsResourceKinded()
}

func (v *DictionaryValue) Iterate(
	config *Config,
	f func(key, value Value) (resume bool),
) {
	for key, value := range v.dictionary {
		if !f(key, value) {
			break
		}
	}
}

//// IterateReadOnlyLoaded iterates over all LOADED key-valye pairs of the array.
//// DO NOT perform storage mutations in the callback!
//func (v *DictionaryValue) IterateReadOnlyLoaded(
//	config *Config,
//	f func(key, value Value) (resume bool),
//) {
//	v.iterate(
//		config,
//		v.dictionary.IterateReadOnlyLoadedValues,
//		f,
//	)
//}

//func (v *DictionaryValue) iterate(
//	config *Config,
//	atreeIterate func(fn atree.MapEntryIterationFunc) error,
//	f func(key Value, value Value) (resume bool),
//) {
//	iterate := func() {
//		err := atreeIterate(func(key, value atree.Value) (resume bool, err error) {
//			// atree.OrderedMap iteration provides low-level atree.Value,
//			// convert to high-level interpreter.Value
//
//			keyValue := MustConvertStoredValue(config.MemoryGauge, config.Storage, key)
//			valueValue := MustConvertStoredValue(config.MemoryGauge, config.Storage, value)
//
//			//checkInvalidatedResourceOrResourceReference(keyValue)
//			//checkInvalidatedResourceOrResourceReference(valueValue)
//
//			resume = f(
//				keyValue,
//				valueValue,
//			)
//
//			return resume, nil
//		})
//		if err != nil {
//			panic(errors.NewExternalError(err))
//		}
//	}
//
//	// TODO:
//	//interpreter.withMutationPrevention(v.ValueID(), iterate)
//	iterate()
//}

func newValueComparator(conf *Config) atree.ValueComparator {
	return func(storage atree.SlabStorage, atreeValue atree.Value, otherStorable atree.Storable) (bool, error) {
		inter := conf.interpreter()
		locationRange := interpreter.EmptyLocationRange
		value := interpreter.MustConvertStoredValue(inter, atreeValue)
		otherValue := interpreter.StoredValue(inter, otherStorable, storage)
		return value.(interpreter.EquatableValue).Equal(inter, locationRange, otherValue), nil
	}
}

func newHashInputProvider(conf *Config) atree.HashInputProvider {
	return func(value atree.Value, scratch []byte) ([]byte, error) {
		inter := conf.interpreter()
		locationRange := interpreter.EmptyLocationRange
		hashInput := interpreter.MustConvertStoredValue(inter, value).(interpreter.HashableValue).
			HashInput(inter, locationRange, scratch)
		return hashInput, nil
	}
}
