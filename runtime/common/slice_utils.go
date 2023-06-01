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

// uses the provided function `f` to generate a slice with no duplicates.
// This is equivalent to adding all the values produced by `f` to a map,
// and then returning `maps.Keys` of that map, except without the nondeterminism.
// K must be comparable in order to have a consistent meaning of the concept of a "duplicate"
package common

func GenerateSliceWithNoDuplicates[K comparable](generator func() *K) (slice []K) {
	m := make(map[K]struct{})

	next := generator()

	for next != nil {
		k := *next
		if _, exists := m[k]; !exists {
			m[k] = struct{}{}
			slice = append(slice, k)
		}
		next = generator()
	}

	return
}

func MappedSliceWithNoDuplicates[T any, K comparable](ts []T, f func(T) K) []K {

	index := 0
	generator := func() *K {
		if index >= len(ts) {
			return nil
		}
		nextK := f(ts[index])
		index++
		return &nextK
	}

	return GenerateSliceWithNoDuplicates(generator)
}
