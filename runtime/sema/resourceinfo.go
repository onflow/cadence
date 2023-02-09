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

package sema

type ResourceInfo struct {
	Parent       *ResourceInfo
	invalidation *ResourceInvalidation
}

// MaybeRecordInvalidation records the given resource invalidation,
// if no invalidation has yet been recorded for the given resource.
func (ris *ResourceInfo) MaybeRecordInvalidation(invalidation ResourceInvalidation) ResourceInvalidation {
	if ris.invalidation != nil {
		return *ris.invalidation
	}
	ris.invalidation = &invalidation
	return invalidation
}

// DeleteLocally removes the given resource invalidation from this current set.
//
// NOTE: the invalidation still might exist in a parent afterwards,
// i.e. call to Contains might still return true!
func (ris *ResourceInfo) DeleteLocally(invalidation ResourceInvalidation) {
	if ris.invalidation == nil ||
		*ris.invalidation != invalidation {

		return
	}
	ris.invalidation = nil
}

// Clone returns a new child resource invalidation set that contains all entries of this parent set.
// Changes to the returned set will only be applied in the returned set, not the parent.
func (ris *ResourceInfo) Clone() ResourceInfo {
	return ResourceInfo{
		Parent: ris,
	}
}

func (ris ResourceInfo) Invalidation() *ResourceInvalidation {
	current := &ris
	for current != nil {
		invalidation := current.invalidation
		if invalidation != nil {
			return invalidation
		}
		current = current.Parent
	}
	return nil
}

func (ris ResourceInfo) DefinitivelyInvalidated() bool {
	invalidation := ris.Invalidation()
	return invalidation != nil &&
		invalidation.Kind.IsDefinite()
}
