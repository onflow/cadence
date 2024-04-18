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

import (
	"sort"
	"strings"

	"github.com/onflow/cadence/runtime/common/orderedmap"
)

func disjunctionKey(disjunction *EntitlementOrderedSet) string {
	// Gather type IDs, sorted
	var typeIDs []string
	disjunction.Foreach(func(entitlementType *EntitlementType, _ struct{}) {
		typeIDs = append(typeIDs, string(entitlementType.ID()))
	})
	sort.Strings(typeIDs)

	// Join type IDs
	var sb strings.Builder
	for index, typeID := range typeIDs {
		if index > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(typeID)
	}
	return sb.String()
}

// DisjunctionOrderedSet is a set of entitlement disjunctions, keyed by disjunctionKey
type DisjunctionOrderedSet = orderedmap.OrderedMap[string, *EntitlementOrderedSet]

// EntitlementSet is a set (conjunction) of entitlements and entitlement disjunctions.
// e.g. {entitlements: A, B; disjunctions: (C | D), (E | F)}
type EntitlementSet struct {
	// Entitlements is a set of entitlements
	Entitlements *EntitlementOrderedSet
	// Disjunctions is a set of entitlement disjunctions, keyed by disjunctionKey
	Disjunctions *DisjunctionOrderedSet
}

// Add adds an entitlement to the set.
//
// NOTE: The resulting set is potentially not minimal:
// If the set contains a disjunction that contains the entitlement,
// then the disjunction is NOT discarded.
// Call Minimize to obtain a minimal set.
func (s *EntitlementSet) Add(entitlementType *EntitlementType) {
	if s.Entitlements == nil {
		s.Entitlements = orderedmap.New[EntitlementOrderedSet](1)
	}
	s.Entitlements.Set(entitlementType, struct{}{})
}

// AddDisjunction adds an entitlement disjunction to the set.
// If the set already contains an entitlement of the given disjunction,
// then the disjunction is discarded.
func (s *EntitlementSet) AddDisjunction(disjunction *EntitlementOrderedSet) {
	// If this set already contains an entitlement of the given disjunction,
	// there is no need to add the disjunction.
	if s.Entitlements != nil &&
		disjunction.ForAnyKey(s.Entitlements.Contains) {

		return
	}

	// If the disjunction already exists in the set,
	// there is no need to add the disjunction.
	key := disjunctionKey(disjunction)
	if s.Disjunctions != nil && s.Disjunctions.Contains(key) {
		return
	}

	if s.Disjunctions == nil {
		s.Disjunctions = orderedmap.New[DisjunctionOrderedSet](1)
	}
	s.Disjunctions.Set(key, disjunction)
}

// Merge merges the other entitlement set into this set.
// The result is the union of the entitlements and disjunctions of both sets.
//
// The result is not necessarily minimal:
// For example, if s contains a disjunction d,
// and other contains an entitlement e that is part of d,
// then the result will still contain d.
// See Add.
// Call Minimize to obtain a minimal set.
func (s *EntitlementSet) Merge(other *EntitlementSet) {
	if other.Entitlements != nil {
		other.Entitlements.Foreach(func(key *EntitlementType, _ struct{}) {
			s.Add(key)
		})
	}

	if other.Disjunctions != nil {
		other.Disjunctions.
			Foreach(func(_ string, disjunction *EntitlementOrderedSet) {
				s.AddDisjunction(disjunction)
			})
	}
}

// Minimize minimizes the entitlement set.
// It removes disjunctions that contain entitlements
// which are also in the entitlement set
func (s *EntitlementSet) Minimize() {
	// If there are no entitlements or no disjunctions,
	// there is nothing to minimize
	if s.Entitlements == nil || s.Disjunctions == nil {
		return
	}

	// Remove disjunctions that contain entitlements that are also in the entitlement set
	var keysToRemove []string
	s.Disjunctions.Foreach(func(key string, disjunction *EntitlementOrderedSet) {
		if disjunction.ForAnyKey(s.Entitlements.Contains) {
			keysToRemove = append(keysToRemove, key)
		}
	})

	for _, key := range keysToRemove {
		s.Disjunctions.Delete(key)
	}
}

// Access returns the access represented by the entitlement set.
// The set is minimized before the access is computed.
func (s *EntitlementSet) Access() Access {
	if s == nil {
		return UnauthorizedAccess
	}

	s.Minimize()

	var entitlements *EntitlementOrderedSet
	if s.Entitlements != nil && s.Entitlements.Len() > 0 {
		entitlements = orderedmap.New[EntitlementOrderedSet](s.Entitlements.Len())
		entitlements.SetAll(s.Entitlements)
	}

	if s.Disjunctions != nil && s.Disjunctions.Len() > 0 {
		if entitlements == nil {
			// If there are no entitlements, and there is only one disjunction,
			// then the access is the disjunction.
			if s.Disjunctions.Len() == 1 {
				onlyDisjunction := s.Disjunctions.Oldest().Value
				return EntitlementSetAccess{
					Entitlements: onlyDisjunction,
					SetKind:      Disjunction,
				}
			}

			// There are no entitlements, but disjunctions.
			// Allocate a new ordered map for all entitlements in the disjunctions
			// (at minimum there are two entitlements in each disjunction).
			entitlements = orderedmap.New[EntitlementOrderedSet](s.Disjunctions.Len() * 2)
		}

		// Add all entitlements in the disjunctions to the entitlements
		s.Disjunctions.Foreach(func(_ string, disjunction *EntitlementOrderedSet) {
			entitlements.SetAll(disjunction)
		})
	}

	if entitlements == nil {
		return UnauthorizedAccess
	}

	return EntitlementSetAccess{
		Entitlements: entitlements,
		SetKind:      Conjunction,
	}
}
