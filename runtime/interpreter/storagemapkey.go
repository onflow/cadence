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

package interpreter

import "github.com/onflow/atree"

type StorageMapKey interface {
	isStorageMapKey()
	AtreeValue() atree.Value
	AtreeValueHashInput(v atree.Value, _ []byte) ([]byte, error)
	AtreeValueCompare(storage atree.SlabStorage, value atree.Value, otherStorable atree.Storable) (bool, error)
}

// StringStorageMapKey is a StorageMapKey backed by a simple StringAtreeValue
type StringStorageMapKey StringAtreeValue

var _ StorageMapKey = StringStorageMapKey("")

func (StringStorageMapKey) isStorageMapKey() {}

func (StringStorageMapKey) AtreeValueHashInput(v atree.Value, scratch []byte) ([]byte, error) {
	return StringAtreeValueHashInput(v, scratch)
}

func (StringStorageMapKey) AtreeValueCompare(
	slabStorage atree.SlabStorage,
	value atree.Value,
	otherStorable atree.Storable,
) (bool, error) {
	return StringAtreeValueComparator(slabStorage, value, otherStorable)
}

func (k StringStorageMapKey) AtreeValue() atree.Value {
	return StringAtreeValue(k)
}

// Uint64StorageMapKey is a StorageMapKey backed by a simple Uint64AtreeValue
type Uint64StorageMapKey Uint64AtreeValue

var _ StorageMapKey = Uint64StorageMapKey(0)

func (Uint64StorageMapKey) isStorageMapKey() {}

func (Uint64StorageMapKey) AtreeValueHashInput(v atree.Value, scratch []byte) ([]byte, error) {
	return Uint64AtreeValueHashInput(v, scratch)
}

func (Uint64StorageMapKey) AtreeValueCompare(
	slabStorage atree.SlabStorage,
	value atree.Value,
	otherStorable atree.Storable,
) (bool, error) {
	return Uint64AtreeValueComparator(slabStorage, value, otherStorable)
}

func (k Uint64StorageMapKey) AtreeValue() atree.Value {
	return Uint64AtreeValue(k)
}
