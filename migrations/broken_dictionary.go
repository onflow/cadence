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

package migrations

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/interpreter"
)

// ShouldFixBrokenCompositeKeyedDictionary returns true if the given value is a dictionary with a composite key type.
//
// It is useful for use with atree's PersistentSlabStorage.FixLoadedBrokenReferences.
//
// NOTE: The intended use case is to enable migration programs in onflow/flow-go to fix broken references.
// As of April 2024, only 10 registers in testnet (not mainnet) were found to have broken references,
// and they seem to have resulted from a bug that was fixed 2 years ago by https://github.com/onflow/cadence/pull/1565.
func ShouldFixBrokenCompositeKeyedDictionary(atreeValue atree.Value) bool {
	orderedMap, ok := atreeValue.(*atree.OrderedMap)
	if !ok {
		return false
	}

	dictionaryStaticType, ok := orderedMap.Type().(interpreter.DictionaryStaticType)
	if !ok {
		return false
	}

	_, ok = dictionaryStaticType.KeyType.(interpreter.CompositeStaticType)
	return ok
}
