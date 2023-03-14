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

package ccf

import (
	"sort"
	"sync"

	"github.com/onflow/cadence"
)

// cachedSortedFieldIndex contains sorted field index of Cadence composite types.
var cachedSortedFieldIndex sync.Map // key: cadence.CompositeType, value: []int

func getSortedFieldIndex(t cadence.CompositeType) []int {
	if v, _ := cachedSortedFieldIndex.Load(t); v != nil {
		return v.([]int)
	}

	// NOTE: bytewiseFieldIdentifierSorter doesn't sort fields in place.
	// bytewiseFieldIdentifierSorter.indexes is used as sorted fieldTypes
	// index.
	sorter := newBytewiseFieldSorter(t.CompositeFields())

	sort.Sort(sorter)

	cachedSortedFieldIndex.Store(t, sorter.indexes)
	return sorter.indexes
}
