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
	"bytes"

	"github.com/onflow/cadence"
)

// bytewiseFieldSorter

// bytewiseFieldSorter is used to sort fields by identifier.
// NOTE: in order to avoid making a copy of fields for sorting,
//   - create sorter by calling newBytewiseFieldIdentifierSorter()
//   - sort by calling sort.Sort(sorter)
//   - iterate sorted fields by
//     for _, index := range sorter.indexes {
//     // process sorted field at fields[index]
//     }
type bytewiseFieldSorter struct {
	// NOTE: DON'T sort fields in place because it isn't a copy.
	// Instead, sort indexes by field identifier.
	fields []cadence.Field
	// indexes represents sorted indexes of fields
	indexes []int
}

func newBytewiseFieldSorter(types []cadence.Field) bytewiseFieldSorter {
	indexes := make([]int, len(types))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = i
	}
	return bytewiseFieldSorter{fields: types, indexes: indexes}
}

func (x bytewiseFieldSorter) Len() int {
	return len(x.indexes)
}

func (x bytewiseFieldSorter) Swap(i, j int) {
	x.indexes[i], x.indexes[j] = x.indexes[j], x.indexes[i]
}

func (x bytewiseFieldSorter) Less(i, j int) bool {
	i = x.indexes[i]
	j = x.indexes[j]

	iIdentifier := x.fields[i].Identifier
	jIdentifier := x.fields[j].Identifier

	if len(iIdentifier) != len(jIdentifier) {
		return len(iIdentifier) < len(jIdentifier)
	}
	return iIdentifier <= jIdentifier
}

// bytewiseKeyValuePairSorter

type encodedKeyValuePair struct {
	encodedKey            []byte
	encodedPair           []byte
	keyLength, pairLength int
}

type bytewiseKeyValuePairSorter []encodedKeyValuePair

func (x bytewiseKeyValuePairSorter) Len() int {
	return len(x)
}

func (x bytewiseKeyValuePairSorter) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x bytewiseKeyValuePairSorter) Less(i, j int) bool {
	return bytes.Compare(x[i].encodedKey, x[j].encodedKey) <= 0
}

// bytewiseCadenceTypeInPlaceSorter

// bytewiseCadenceTypeInPlaceSorter is used to sort Cadence types by Cadence type ID.
type bytewiseCadenceTypeInPlaceSorter []cadence.Type

func (t bytewiseCadenceTypeInPlaceSorter) Len() int {
	return len(t)
}

func (t bytewiseCadenceTypeInPlaceSorter) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t bytewiseCadenceTypeInPlaceSorter) Less(i, j int) bool {
	iID := t[i].ID()
	jID := t[j].ID()
	if len(iID) != len(jID) {
		return len(iID) < len(jID)
	}
	return iID <= jID
}

// bytewiseCadenceTypeSorter

// bytewiseCadenceTypeSorter is used to sort Cadence types by Cadence type ID.
type bytewiseCadenceTypeSorter struct {
	// NOTE: DON'T sort types in place because it isn't a copy.
	// Instead, sort indexes by Cadence type ID.
	types []cadence.Type
	// indexes represents sorted indexes of fields
	indexes []int
}

func newBytewiseCadenceTypeSorter(types []cadence.Type) bytewiseCadenceTypeSorter {
	indexes := make([]int, len(types))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = i
	}
	return bytewiseCadenceTypeSorter{types: types, indexes: indexes}
}

func (t bytewiseCadenceTypeSorter) Len() int {
	return len(t.indexes)
}

func (t bytewiseCadenceTypeSorter) Swap(i, j int) {
	t.indexes[i], t.indexes[j] = t.indexes[j], t.indexes[i]
}

func (t bytewiseCadenceTypeSorter) Less(i, j int) bool {
	i = t.indexes[i]
	j = t.indexes[j]

	iID := t.types[i].ID()
	jID := t.types[j].ID()

	if len(iID) != len(jID) {
		return len(iID) < len(jID)
	}
	return iID <= jID
}

// Utility sort functions

func stringsAreSortedBytewise(s1, s2 string) bool {
	return len(s1) < len(s2) ||
		(len(s1) == len(s2) && s1 < s2)
}

func bytesAreSortedBytewise(b1, b2 []byte) bool {
	return bytes.Compare(b1, b2) <= 0
}
