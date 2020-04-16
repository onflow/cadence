package runtime

import (
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/errors"
	"github.com/dapperlabs/cadence/runtime/interpreter"
)

type storageKey struct {
	storageIdentifier string
	key               string
}

type interpreterRuntimeStorage struct {
	runtimeInterface Interface
	cache            map[storageKey]interpreter.Value
}

func newInterpreterRuntimeStorage(runtimeInterface Interface) *interpreterRuntimeStorage {
	return &interpreterRuntimeStorage{
		runtimeInterface: runtimeInterface,
		cache:            map[storageKey]interpreter.Value{},
	}
}

// valueExists is the StorageExistenceHandlerFunc for the interpreter.
//
// It checks the cache for values which were already previously loaded/deserialized
// from storage (through the runtime interface) and returns true if the cached value exists.
//
// If there is a cache miss, the key is read from storage through the runtime interface,
// places in the cache, and returned.
//
func (s *interpreterRuntimeStorage) valueExists(
	storageIdentifier string,
	key string,
) bool {

	storageKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Check cache

	if cachedValue, ok := s.cache[storageKey]; ok {
		return cachedValue != nil
	}

	// Cache miss: Ask interface

	var exists bool
	if runtimeInterfaceV2, ok := s.runtimeInterface.(InterfaceV2); ok {
		var err error
		// TODO: fix controller
		exists, err = runtimeInterfaceV2.ValueExists([]byte(storageIdentifier), []byte{}, []byte(key))
		if err != nil {
			panic(err)
		}
	} else {
		value, err := s.runtimeInterface.GetValue([]byte(storageIdentifier), []byte{}, []byte(key))
		if err != nil {
			panic(err)
		}
		exists = len(value) > 0
	}

	if !exists {
		s.cache[storageKey] = nil
	}

	return exists
}

// readValue is the StorageReadHandlerFunc for the interpreter.
//
// It checks the cache for values which were already previously loaded/deserialized
// from storage (through the runtime interface) and returns the cached value if it exists.
//
// If there is a cache miss, the key is read from storage through the runtime interface,
// places in the cache, and returned.
//
func (s *interpreterRuntimeStorage) readValue(
	storageIdentifier string,
	key string,
) interpreter.OptionalValue {

	storageKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Check cache. Return cached value, if any

	if cachedValue, ok := s.cache[storageKey]; ok {
		if cachedValue == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewSomeValueOwningNonCopying(cachedValue)
	}

	// Cache miss: Load and deserialize the stored value (if any)
	// through the runtime interface

	// TODO: fix controller
	storedData, err := s.runtimeInterface.GetValue([]byte(storageIdentifier), []byte{}, []byte(key))
	if err != nil {
		panic(err)
	}

	if len(storedData) == 0 {
		s.cache[storageKey] = nil
		return interpreter.NilValue{}
	}

	var address *common.Address
	address.SetBytes([]byte(storageIdentifier))

	storedValue, err := interpreter.DecodeValue(storedData, address)
	if err != nil {
		panic(err)
	}

	s.cache[storageKey] = storedValue
	return interpreter.NewSomeValueOwningNonCopying(storedValue)
}

// writeValue is the StorageWriteHandlerFunc for the interpreter.
//
// It only places the written value in the cache.
//
// It does *not* serialize/save the value in  storage (through the runtime interface).
// (The Cache is finally written back through the runtime interface in `writeCached`.)
//
func (s *interpreterRuntimeStorage) writeValue(
	storageIdentifier string,
	key string,
	value interpreter.OptionalValue,
) {
	storageKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Only write the value to the cache.
	// The Cache is finally written back through the runtime interface in `writeCached`

	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		s.cache[storageKey] = typedValue.Value
	case interpreter.NilValue:
		s.cache[storageKey] = nil
	default:
		panic(errors.NewUnreachableError())
	}
}

// writeCached serializes/saves all values in the cache in storage (through the runtime interface).
//
func (s *interpreterRuntimeStorage) writeCached() {

	for storageKey, value := range s.cache {

		var newData []byte
		if value != nil {
			encodedValue, err := interpreter.EncodeValue(value)
			if err != nil {
				panic(err)
			}
			newData = encodedValue
		}

		// TODO: fix controller
		err := s.runtimeInterface.SetValue(
			[]byte(storageKey.storageIdentifier),
			[]byte{},
			[]byte(storageKey.key),
			newData,
		)
		if err != nil {
			panic(err)
		}
	}
}
