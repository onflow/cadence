/*
* Cadence - The resource-oriented smart contract programming language
*
* Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/interpreter"
)

//	type Storage interface {
//		atree.SlabStorage
//		GetStorageMap(address common.Address, domain string, createIfNotExists bool) *StorageMap
//		CheckHealth() error
//	}
func StoredValue(gauge common.MemoryGauge, storable atree.Storable, storage atree.SlabStorage) Value {
	// Delegate
	value := interpreter.StoredValue(gauge, storable, storage)
	return InterpreterValueToVMValue(storage, value)
}

func WriteStored(
	gauge common.MemoryGauge,
	storage interpreter.Storage,
	storageAddress common.Address,
	domain string,
	identifier string,
	value Value,
) {
	accountStorage := storage.GetStorageMap(storageAddress, domain, true)
	interValue := VMValueToInterpreterValue(storage, value)
	accountStorage.WriteValue(inter(storage), identifier, interValue)
	//interpreter.recordStorageMutation()
}

//
//// InMemoryStorage
//type InMemoryStorage struct {
//	*atree.BasicSlabStorage
//	StorageMaps map[interpreter.StorageKey]*StorageMap
//	memoryGauge common.MemoryGauge
//}
//
//var _ Storage = InMemoryStorage{}
//
//func NewInMemoryStorage(memoryGauge common.MemoryGauge) InMemoryStorage {
//	decodeStorable := func(decoder *cbor.StreamDecoder, storableSlabStorageID atree.StorageID) (atree.Storable, error) {
//		return interpreter.DecodeStorable(decoder, storableSlabStorageID, memoryGauge)
//	}
//
//	decodeTypeInfo := func(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
//		return interpreter.DecodeTypeInfo(decoder, memoryGauge)
//	}
//
//	slabStorage := atree.NewBasicSlabStorage(
//		interpreter.CBOREncMode,
//		interpreter.CBORDecMode,
//		decodeStorable,
//		decodeTypeInfo,
//	)
//
//	return InMemoryStorage{
//		BasicSlabStorage: slabStorage,
//		StorageMaps:      make(map[interpreter.StorageKey]*StorageMap),
//		memoryGauge:      memoryGauge,
//	}
//}
//
//func (i InMemoryStorage) GetStorageMap(
//	address common.Address,
//	domain string,
//	createIfNotExists bool,
//) (
//	storageMap *StorageMap,
//) {
//	key := interpreter.NewStorageKey(i.memoryGauge, address, domain)
//	storageMap = i.StorageMaps[key]
//	if storageMap == nil && createIfNotExists {
//		storageMap = NewStorageMap(i.memoryGauge, i, atree.Address(address))
//		i.StorageMaps[key] = storageMap
//	}
//	return storageMap
//}
//
//func (i InMemoryStorage) CheckHealth() error {
//	_, err := atree.CheckStorageHealth(i, -1)
//	return err
//}
