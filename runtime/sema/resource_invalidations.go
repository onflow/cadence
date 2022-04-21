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

package sema

type ResourceInvalidations struct {
	Parent        *ResourceInvalidations
	invalidations *ResourceInvalidationStructOrderedMap
}

// ForEach calls the given function for each resource invalidation in the set.
// It can be used to iterate over all invalidations.
//
func (ris *ResourceInvalidations) ForEach(cb func(invalidation ResourceInvalidation) error) error {

	resourceInvalidations := ris

	for resourceInvalidations != nil {

		if resourceInvalidations.invalidations != nil {
			for pair := resourceInvalidations.invalidations.Oldest(); pair != nil; pair = pair.Next() {
				invalidation := pair.Key

				err := cb(invalidation)
				if err != nil {
					return err
				}
			}
		}

		resourceInvalidations = resourceInvalidations.Parent
	}

	return nil
}

// Contains returns true if the given resource use position exists in the set.
//
func (ris ResourceInvalidations) Contains(invalidation ResourceInvalidation) bool {
	if ris.invalidations != nil {
		_, ok := ris.invalidations.Get(invalidation)
		if ok {
			return true
		}
	}

	if ris.Parent != nil {
		return ris.Parent.Contains(invalidation)
	}

	return false
}

// All returns a slice with all resource invalidations in the set.
//
func (ris ResourceInvalidations) All() (result []ResourceInvalidation) {
	_ = ris.ForEach(func(invalidation ResourceInvalidation) error {
		result = append(result, invalidation)

		// NOTE: when changing this function to return an error,
		// also return it from the outer function,
		// as the outer error is currently ignored!
		return nil
	})
	return
}

// Add adds the given resource invalidation to this set.
//
func (ris *ResourceInvalidations) Add(invalidation ResourceInvalidation) {
	if ris.Contains(invalidation) {
		return
	}
	if ris.invalidations == nil {
		ris.invalidations = NewResourceInvalidationStructOrderedMap()
	}
	ris.invalidations.Set(invalidation, struct{}{})
}

// DeleteLocally removes the given resource invalidation from this current set.
//
// NOTE: the invalidation still might exist in a parent afterwards,
// i.e. call to Contains might still return true!
//
func (ris *ResourceInvalidations) DeleteLocally(invalidation ResourceInvalidation) {
	if ris.invalidations == nil {
		return
	}
	ris.invalidations.Delete(invalidation)
}

// Merge adds the resource invalidations of the given set to this set.
//
func (ris *ResourceInvalidations) Merge(other ResourceInvalidations) {
	_ = other.ForEach(func(invalidation ResourceInvalidation) error {
		ris.Add(invalidation)

		// NOTE: when changing this function to return an error,
		// also return it from the outer function,
		// as the outer error is currently ignored!
		return nil
	})
}

// Size returns the number of resource invalidations in this set.
//
func (ris ResourceInvalidations) Size() int {
	var size int
	if ris.Parent != nil {
		size = ris.Parent.Size()
	}
	if ris.invalidations == nil {
		return size
	}
	return size + ris.invalidations.Len()
}

// IsEmpty returns true if this set contains no resource invalidations.
//
func (ris ResourceInvalidations) IsEmpty() bool {
	return ris.Size() == 0
}

// Clone returns a new child resource invalidation set that contains all entries of this parent set.
// Changes to the returned set will only be applied in the returned set, not the parent.
//
func (ris *ResourceInvalidations) Clone() ResourceInvalidations {
	return ResourceInvalidations{
		Parent: ris,
	}
}
