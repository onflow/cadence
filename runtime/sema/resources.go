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

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

/*


   ┌────────────────────────┐              ┏━━━━━━━━━━━━┓
   │                        │              ▼            ┃
   │       Resources:       │      ┌─────────────────┐  ┃
   │                        │      │                 │  ┃
   │    map[Resource]Info  ━╋━━━━━▶│      Info:      │  ┃
   │                        │      │                 │  ┃
   └────────────────────────┘      │                 │  ┃
                                   │                 │  ┃
                │                  └─────────────────┘  ┃
                                                        ┃
             Clone                          │           ┃
                                                        ┃
                │                        Clone          ┃
                                                        ┃
                ▼                           │           ┃
   ┌────────────────────────┐                           ┃
   │                        │               ▼           ┃
   │       Resources:       │      ┌─────────────────┐  ┃
   │                        │      │                 │  ┃
   │    map[Resource]Info  ━╋━━━━━▶│      Info:      │  ┃
   │                        │      │                 │  ┃
   └────────────────────────┘      │      Parent  ━━━╋━━┛
                                   │                 │
                                   └─────────────────┘

*/

// A Resource is a variable or a member
type Resource struct {
	*Variable
	*Member
}

// Resources is a map which contains invalidation info for resources.
type Resources struct {
	resources *orderedmap.OrderedMap[Resource, ResourceInfo]
}

func NewResources() *Resources {
	return &Resources{
		resources: &orderedmap.OrderedMap[Resource, ResourceInfo]{},
	}
}

func (ris *Resources) String() string {
	var builder strings.Builder
	builder.WriteString("Resources:")
	ris.ForEach(func(resource Resource, info ResourceInfo) {
		builder.WriteString("- ")
		builder.WriteString(fmt.Sprint(resource))
		builder.WriteString(": ")
		builder.WriteString(fmt.Sprint(info))
		builder.WriteRune('\n')
	})
	return builder.String()
}

func (ris *Resources) Get(resource Resource) ResourceInfo {
	info, _ := ris.resources.Get(resource)
	return info
}

// MaybeRecordInvalidation records the given resource invalidation,
// if no invalidation has yet been recorded for the given resource.
func (ris *Resources) MaybeRecordInvalidation(resource Resource, invalidation ResourceInvalidation) {
	info, _ := ris.resources.Get(resource)
	info.MaybeRecordInvalidation(invalidation)
	ris.resources.Set(resource, info)
}

// RemoveTemporaryMoveInvalidation removes the given invalidation
// from the set of invalidations for the given resource.
//
func (ris *Resources) RemoveTemporaryMoveInvalidation(resource Resource, invalidation ResourceInvalidation) {
	if invalidation.Kind != ResourceInvalidationKindMoveTemporary {
		panic(errors.NewUnreachableError())
	}

	info, _ := ris.resources.Get(resource)
	info.DeleteLocally(invalidation)
	ris.resources.Set(resource, info)
}

func (ris *Resources) Clone() *Resources {
	// TODO: optimize
	result := NewResources()
	for pair := ris.resources.Oldest(); pair != nil; pair = pair.Next() {
		resource := pair.Key
		info := pair.Value

		result.resources.Set(resource, info.Clone())
	}
	return result
}

func (ris *Resources) Size() int {
	return ris.resources.Len()
}

func (ris *Resources) ForEach(f func(resource Resource, info ResourceInfo)) {
	ris.resources.Foreach(f)
}

// MergeBranches merges the given resources from two branches into these resources.
// Invalidations occurring in both branches are considered definitive,
// other new invalidations are only considered potential.
// The else resources are optional.
//
func (ris *Resources) MergeBranches(
	thenResources *Resources,
	thenReturnInfo *ReturnInfo,
	elseResources *Resources,
	elseReturnInfo *ReturnInfo,
) {

	merged := make(map[Resource]struct{})

	merge := func(resource Resource) {

		// Only merge each resource once.
		// We iterate over the resources of the then-branch
		// and the else-branch (if it exists)

		if _, ok := merged[resource]; ok {
			return
		}
		defer func() {
			merged[resource] = struct{}{}
		}()

		// Get the resource info in this outer scope

		info := ris.Get(resource)

		// If the resource is already invalidated in the outer scope,
		// then there is nothing to do.

		if info.Invalidation() != nil {
			return
		}

		// Get the resource info in the then-branch,
		// and from the else-branch, if any.

		thenInfo := thenResources.Get(resource)
		var elseInfo ResourceInfo
		if elseResources != nil {
			elseInfo = elseResources.Get(resource)
		}

		// The resource can be considered definitively invalidated
		// if it was definitely invalidated in both branches.

		// TODO:
		// A halting branch should also be considered resulting in a definitive invalidation,
		// to support e.g.
		//
		//     let r <- create R()
		//     if false {
		//         f(<-r)
		//     } else {
		//         panic("")
		//     }

		info.invalidation = mergeResourceInfos(thenInfo, elseInfo)

		ris.resources.Set(resource, info)
	}

	// Merge the resource info of all resources in the then-branch

	thenResources.ForEach(func(resource Resource, _ ResourceInfo) {
		merge(resource)
	})

	// If there is an else-branch,
	// then merge the resource info of all resources in it

	if elseResources != nil {
		elseResources.ForEach(func(resource Resource, _ ResourceInfo) {
			merge(resource)
		})
	}
}

func mergeResourceInfos(thenInfo, elseInfo ResourceInfo) (invalidation *ResourceInvalidation) {
	thenInvalidation := thenInfo.Invalidation()
	elseInvalidation := elseInfo.Invalidation()

	if thenInvalidation != nil {
		invalidation = thenInvalidation

		if !(elseInvalidation != nil &&
			elseInvalidation.Kind.IsDefinite() &&
			thenInvalidation.Kind.IsDefinite()) {

			invalidation.Kind = invalidation.Kind.AsPotential()
		}
	} else if elseInvalidation != nil {
		invalidation = elseInvalidation
		invalidation.Kind = invalidation.Kind.AsPotential()
	}

	return
}
