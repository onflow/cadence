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

package main

import (
	"encoding/json"
	"strconv"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/stdlib"
)

type KnownStorageMap struct {
	Domain      string
	KeyAsString func(key atree.Value) string
	StringAsKey func(identifier string) (interpreter.StorageMapKey, error)
}

var knownStorageMaps = map[string]KnownStorageMap{}

func addKnownStorageMap(storageMap KnownStorageMap) {
	knownStorageMaps[storageMap.Domain] = storageMap
}

func StringAtreeValueAsString(key atree.Value) string {
	return string(key.(interpreter.StringAtreeValue))
}

func StringAsStringAtreeValue(identifier string) (interpreter.StorageMapKey, error) {
	return interpreter.StringStorageMapKey(identifier), nil
}

func Uint64AtreeValueAsString(key atree.Value) string {
	return strconv.FormatUint(uint64(key.(interpreter.Uint64AtreeValue)), 10)
}

func StringAsUint64AtreeValue(identifier string) (interpreter.StorageMapKey, error) {
	num, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		return nil, err
	}
	return interpreter.Uint64StorageMapKey(num), nil
}

func init() {
	for _, domain := range common.AllPathDomains {
		addKnownStorageMap(KnownStorageMap{
			Domain:      domain.Identifier(),
			KeyAsString: StringAtreeValueAsString,
			StringAsKey: StringAsStringAtreeValue,
		})
	}

	addKnownStorageMap(KnownStorageMap{
		Domain:      stdlib.InboxStorageDomain,
		KeyAsString: StringAtreeValueAsString,
		StringAsKey: StringAsStringAtreeValue,
	})

	addKnownStorageMap(KnownStorageMap{
		Domain:      runtime.StorageDomainContract,
		KeyAsString: StringAtreeValueAsString,
		StringAsKey: StringAsStringAtreeValue,
	})

	addKnownStorageMap(KnownStorageMap{
		Domain:      stdlib.CapabilityControllerStorageDomain,
		KeyAsString: Uint64AtreeValueAsString,
		StringAsKey: StringAsUint64AtreeValue,
	})
}

func knownStorageMapsJSON() []byte {
	knownStorageMapDomains := make([]string, 0, len(knownStorageMaps))
	for _, knownStorageMap := range knownStorageMaps {
		knownStorageMapDomains = append(
			knownStorageMapDomains,
			knownStorageMap.Domain,
		)
	}
	encoded, err := json.Marshal(knownStorageMapDomains)
	if err != nil {
		panic(err)
	}
	return encoded
}
