/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type DictionaryValue struct {
	dictionary  *atree.OrderedMap
	Type        *interpreter.DictionaryStaticType
	elementSize uint
}

var _ Value = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}

func NewDictionaryValue(
	config *Config,
	dictionaryType *interpreter.DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {

	address := common.ZeroAddress

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
	v := newDictionaryValueFromConstructor(dictionaryType, constructor)

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		existingValue := v.Insert(config, key, value)
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			if v.IsResourceKinded() {
				panic(interpreter.DuplicateKeyInResourceDictionaryError{})
			}
		}
	}

	return v
}

func newDictionaryValueFromConstructor(
	staticType *interpreter.DictionaryStaticType,
	constructor func() *atree.OrderedMap,
) *DictionaryValue {

	elementSize := interpreter.DictionaryElementSize(staticType)
	return newDictionaryValueFromAtreeMap(
		staticType,
		elementSize,
		constructor(),
	)
}

func newDictionaryValueFromAtreeMap(
	staticType *interpreter.DictionaryStaticType,
	elementSize uint,
	atreeOrderedMap *atree.OrderedMap,
) *DictionaryValue {

	return &DictionaryValue{
		Type:        staticType,
		dictionary:  atreeOrderedMap,
		elementSize: elementSize,
	}
}

func (*DictionaryValue) isValue() {}

func (v *DictionaryValue) StaticType(memoryGauge common.MemoryGauge) StaticType {
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

	address := v.dictionary.Address()

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

	existingValueStorable := v.InsertWithoutTransfer(conf, keyValue, value)

	if existingValueStorable == nil {
		return Nil // TODO: NilOptionalValue
	}

	existingValue := StoredValue(
		conf,
		existingValueStorable,
		conf.Storage,
	).Transfer(
		conf,
		atree.Address{},
		true,
		existingValueStorable,
	)

	return NewSomeValueNonCopying(existingValue)
}

func (v *DictionaryValue) InsertWithoutTransfer(conf *Config, key, value Value) (existingValueStorable atree.Storable) {

	//interpreter.validateMutation(v.StorageID(), locationRange)

	valueComparator := newValueComparator(conf)
	hashInputProvider := newHashInputProvider(conf)

	keyInterpreterValue := VMValueToInterpreterValue(key)
	valueInterpreterValue := VMValueToInterpreterValue(value)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	var err error
	existingValueStorable, err = v.dictionary.Set(
		valueComparator,
		hashInputProvider,
		keyInterpreterValue,
		valueInterpreterValue,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	//interpreter.maybeValidateAtreeValue(v.dictionary)

	return existingValueStorable
}

func (v *DictionaryValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *DictionaryValue) IsResourceKinded() bool {
	// TODO:
	return false
}

func (v *DictionaryValue) String() string {
	//TODO implement me
	panic("implement me")
}

func (v *DictionaryValue) Transfer(
	conf *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	currentStorageID := v.StorageID()
	currentAddress := currentStorageID.Address

	dictionary := v.dictionary

	needsStoreTo := address != currentAddress
	isResourceKinded := v.IsResourceKinded()

	if needsStoreTo || !isResourceKinded {
		valueComparator := newValueComparator(conf)
		hashInputProvider := newHashInputProvider(conf)

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		dictionary, err = atree.NewMapFromBatchData(
			conf.Storage,
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

				key := interpreter.MustConvertStoredValue(conf.MemoryGauge, atreeValue)
				// TODO: converted value is unused
				vmKey := InterpreterValueToVMValue(key)
				vmKey = vmKey.Transfer(conf, address, remove, nil)

				value := interpreter.MustConvertStoredValue(conf.MemoryGauge, atreeValue)
				// TODO: converted value is unused
				vmValue := InterpreterValueToVMValue(value)
				vmValue = vmValue.Transfer(conf, address, remove, nil)

				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
				RemoveReferencedSlab(conf.Storage, keyStorable)
				RemoveReferencedSlab(conf.Storage, valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			//interpreter.maybeValidateAtreeValue(v.dictionary)

			RemoveReferencedSlab(conf.Storage, storable)
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

		// TODO:
		//interpreter.invalidateReferencedResources(v, locationRange)

		v.dictionary = nil
	}

	res := newDictionaryValueFromAtreeMap(
		v.Type,
		v.elementSize,
		dictionary,
	)

	//res.semaType = v.semaType
	//res.isResourceKinded = v.isResourceKinded
	//res.isDestroyed = v.isDestroyed

	return res
}

func (v *DictionaryValue) Destroy(*Config) {
	v.dictionary = nil
}

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
