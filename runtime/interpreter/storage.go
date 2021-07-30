/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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
	"bytes"

	"github.com/fxamacker/atree"
	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/cadence/runtime/common"
)

type InMemoryStorageKey struct {
	Address common.Address
	Key     string
}

type InMemoryStorage struct {
	*atree.BasicSlabStorage
	Data map[InMemoryStorageKey]atree.Storable
}

func (i InMemoryStorage) Exists(_ *Interpreter, address common.Address, key string) bool {
	_, ok := i.Data[InMemoryStorageKey{Address: address, Key: key}]
	return ok
}

func (i InMemoryStorage) Read(_ *Interpreter, address common.Address, key string) OptionalValue {
	storable, ok := i.Data[InMemoryStorageKey{Address: address, Key: key}]
	if !ok {
		return nil
	}

	value, err := storable.Value(i.BasicSlabStorage)
	if err != nil {
		panic(err)
	}

	// TODO: embed atree.Value in Value and implement
	return NewSomeValueOwningNonCopying(value.(Value))
}

func (i InMemoryStorage) Write(_ *Interpreter, address common.Address, key string, value OptionalValue) {
	storageKey := InMemoryStorageKey{
		Address: address,
		Key:     key,
	}

	switch value := value.(type) {
	case *SomeValue:
		// TODO: embed atree.Value in Value and implement
		i.Data[storageKey] = value.InnerValue.(atree.Value).Storable()

	case NilValue:
		delete(i.Data, storageKey)
	}
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage() InMemoryStorage {
	return InMemoryStorage{
		BasicSlabStorage: atree.NewBasicSlabStorage(),
	}
}

func storableSize(v atree.Storable) uint32 {
	var buf bytes.Buffer
	enc := &atree.Encoder{
		Writer: &buf,
		CBOR:   cbor.NewStreamEncoder(&buf),
	}
	err := v.Encode(enc)
	if err != nil {
		panic(err)
	}
	err = enc.CBOR.Flush()
	if err != nil {
		panic(err)
	}
	return uint32(len(buf.Bytes()))
}
