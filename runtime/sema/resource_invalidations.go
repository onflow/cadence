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

type ResourceInvalidations struct {
	invalidations map[ResourceInvalidation]struct{}
}

// ForEach calls the given function for each resource invalidation in the set.
// It can be used to iterate over all invalidations.
//
func (ris ResourceInvalidations) ForEach(cb func(invalidation ResourceInvalidation) error) error {
	for invalidation := range ris.invalidations {
		err := cb(invalidation)
		if err != nil {
			return err
		}
	}
	return nil
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

// Insert adds the given resource invalidation to this set.
//
func (ris *ResourceInvalidations) Insert(invalidation ResourceInvalidation) {
	if ris.invalidations == nil {
		ris.invalidations = map[ResourceInvalidation]struct{}{}
	}
	ris.invalidations[invalidation] = struct{}{}
}

// Delete removes the given resource invalidation from this set.
//
func (ris *ResourceInvalidations) Delete(invalidation ResourceInvalidation) {
	if ris.invalidations == nil {
		return
	}
	delete(ris.invalidations, invalidation)
}

// Merge adds the resource invalidations of the given set to this set.
//
func (ris *ResourceInvalidations) Merge(other ResourceInvalidations) {
	_ = other.ForEach(func(invalidation ResourceInvalidation) error {
		ris.Insert(invalidation)

		// NOTE: when changing this function to return an error,
		// also return it from the outer function,
		// as the outer error is currently ignored!
		return nil
	})
}

// Size returns the number of resource invalidations in this set.
//
func (ris ResourceInvalidations) Size() int {
	return len(ris.invalidations)
}

// IsEmpty returns true if this set contains no resource invalidations.
//
func (ris ResourceInvalidations) IsEmpty() bool {
	return ris.Size() == 0
}
