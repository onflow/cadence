/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package sema

import "github.com/raviqqe/hamt"

type ResourceInvalidations struct {
	invalidations hamt.Set
}

func (ris ResourceInvalidations) All() (result []ResourceInvalidation) {
	_ = ris.invalidations.ForEach(func(entry hamt.Entry) error {
		invalidation := entry.(ResourceInvalidationEntry).ResourceInvalidation
		result = append(result, invalidation)

		// NOTE: when changing this function to return an error,
		// also return it from the outer function,
		// as the outer error is currently ignored!
		return nil
	})
	return
}

func (ris ResourceInvalidations) Include(invalidation ResourceInvalidation) bool {
	return ris.invalidations.Include(ResourceInvalidationEntry{
		ResourceInvalidation: invalidation,
	})
}

func (ris ResourceInvalidations) Insert(invalidation ResourceInvalidation) ResourceInvalidations {
	entry := ResourceInvalidationEntry{invalidation}
	newInvalidations := ris.invalidations.Insert(entry)
	return ResourceInvalidations{newInvalidations}
}

func (ris ResourceInvalidations) Delete(invalidation ResourceInvalidation) ResourceInvalidations {
	entry := ResourceInvalidationEntry{invalidation}
	newInvalidations := ris.invalidations.Delete(entry)
	return ResourceInvalidations{newInvalidations}
}

func (ris ResourceInvalidations) Merge(other ResourceInvalidations) ResourceInvalidations {
	newInvalidations := ris.invalidations.Merge(other.invalidations)
	return ResourceInvalidations{newInvalidations}
}

func (ris ResourceInvalidations) Size() int {
	return ris.invalidations.Size()
}

func (ris ResourceInvalidations) IsEmpty() bool {
	return ris.Size() == 0
}
