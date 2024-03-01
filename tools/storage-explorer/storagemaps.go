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
package main

import (
	"encoding/json"
	"strconv"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

type KnownStorageMap struct {
	Domain      string
	KeyAsString func(key atree.Value) string
}

var knownStorageMaps = map[string]KnownStorageMap{}

func addKnownStorageMap(storageMap KnownStorageMap) {
	knownStorageMaps[storageMap.Domain] = storageMap
}

func StringAtreeValueAsString(key atree.Value) string {
	return string(key.(interpreter.StringAtreeValue))
}

func Uint64AtreeValueAsString(key atree.Value) string {
	return strconv.FormatUint(uint64(key.(interpreter.Uint64AtreeValue)), 10)
}

func init() {
	for _, domain := range common.AllPathDomains {
		addKnownStorageMap(KnownStorageMap{
			Domain:      domain.Identifier(),
			KeyAsString: StringAtreeValueAsString,
		})
	}

	addKnownStorageMap(KnownStorageMap{
		Domain:      stdlib.InboxStorageDomain,
		KeyAsString: StringAtreeValueAsString,
	})

	addKnownStorageMap(KnownStorageMap{
		Domain:      runtime.StorageDomainContract,
		KeyAsString: StringAtreeValueAsString,
	})

	addKnownStorageMap(KnownStorageMap{
		Domain:      stdlib.CapabilityControllerStorageDomain,
		KeyAsString: Uint64AtreeValueAsString,
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
