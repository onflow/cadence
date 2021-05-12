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

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

/*


   ┌────────────────────────┐
   │                        │
   │       Resources:       │      ┌─────────────────┐                ┏━━━━━━━━━━┓
   │                        │      │                 │                ▼          ┃
   │  map[interface{}]Info ━╋━━━━━▶│      Info:      │       ┌────────────────┐  ┃
   │                        │      │                 │       │                │  ┃
   └────────────────────────┘      │  Invalidations ━╋━━━━━━▶│ Invalidations: │  ┃
                                   │                 │       │                │  ┃
                │                  │      Uses ━━━━━━╋━━━┓   └────────────────┘  ┃
                                   │                 │   ┃                       ┃          ┏━━━━━━━━┓
                │                  └─────────────────┘   ┃            │          ┃          ▼        ┃
                                                         ┃                       ┃    ┌───────────┐  ┃
             Clone                          │            ┃            │          ┃    │           │  ┃
                                                         ┗━━━━━━━━━━━━━━━━━━━━━━━╋━━━▶│   Uses:   │  ┃
                │                        Clone                        │          ┃    │           │  ┃
                                                                                 ┃    └───────────┘  ┃
                ▼                           │                      Clone         ┃                   ┃
   ┌────────────────────────┐                                                    ┃          │        ┃
   │                        │               ▼                         │          ┃                   ┃
   │       Resources:       │      ┌─────────────────┐                           ┃          │        ┃
   │                        │      │                 │                ▼          ┃                   ┃
   │  map[interface{}]Info ━╋━━━━━▶│      Info:      │       ┌────────────────┐  ┃       Clone       ┃
   │                        │      │                 │       │                │  ┃                   ┃
   └────────────────────────┘      │  Invalidations ━╋━━━━━━▶│ Invalidations: │  ┃          │        ┃
                                   │                 │       │                │  ┃                   ┃
                                   │      Uses ━━━━━━╋━━━┓   │     Parent ━━━━╋━━┛          │        ┃
                                   │                 │   ┃   │                │                      ┃
                                   └─────────────────┘   ┃   └────────────────┘             ▼        ┃
                                                         ┃                            ┌───────────┐  ┃
                                                         ┃                            │   Uses:   │  ┃
                                                         ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━▶│           │  ┃
                                                                                      │  Parent ━━╋━━┛
                                                                                      └───────────┘
*/

// ResourceInfo is the info for a resource.
//
type ResourceInfo struct {
	// DefinitivelyInvalidated is true if the invalidation of the resource
	// can be considered definitive
	DefinitivelyInvalidated bool
	// Invalidations is the set of invalidations of the resource
	Invalidations ResourceInvalidations
	// UsePositions is the set of uses of the resource
	UsePositions ResourceUses
}

func (ri ResourceInfo) Clone() ResourceInfo {
	return ResourceInfo{
		DefinitivelyInvalidated: ri.DefinitivelyInvalidated,
		Invalidations:           ri.Invalidations.Clone(),
		UsePositions:            ri.UsePositions.Clone(),
	}
}

// Resources is a map which contains invalidation info for resources.
//
type Resources struct {
	resources *InterfaceResourceInfoOrderedMap
	Returns   bool
}

func NewResources() *Resources {
	return &Resources{
		resources: NewInterfaceResourceInfoOrderedMap(),
	}
}

func (ris *Resources) String() string {
	var builder strings.Builder
	builder.WriteString("Resources:")
	ris.ForEach(func(resource interface{}, info ResourceInfo) {
		builder.WriteString("- ")
		builder.WriteString(fmt.Sprint(resource))
		builder.WriteString(": ")
		builder.WriteString(fmt.Sprint(info))
		builder.WriteRune('\n')
	})
	return builder.String()
}

func (ris *Resources) Get(resource interface{}) ResourceInfo {
	info, _ := ris.resources.Get(resource)
	return info
}

// AddInvalidation adds the given invalidation to the set of invalidations for the given resource.
// If the invalidation is not temporary, marks the resource to be definitely invalidated.
//
func (ris *Resources) AddInvalidation(resource interface{}, invalidation ResourceInvalidation) {
	info, _ := ris.resources.Get(resource)
	info.Invalidations.Add(invalidation)
	if invalidation.Kind.IsDefinite() {
		info.DefinitivelyInvalidated = true
	}
	ris.resources.Set(resource, info)
}

// RemoveTemporaryMoveInvalidation removes the given invalidation
// from the set of invalidations for the given resource.
//
func (ris *Resources) RemoveTemporaryMoveInvalidation(resource interface{}, invalidation ResourceInvalidation) {
	if invalidation.Kind != ResourceInvalidationKindMoveTemporary {
		panic(errors.NewUnreachableError())
	}

	info, _ := ris.resources.Get(resource)
	info.Invalidations.DeleteLocally(invalidation)
	ris.resources.Set(resource, info)
}

// AddUse adds the given use position to the set of use positions for the given resource.
//
func (ris *Resources) AddUse(resource interface{}, use ast.Position) {
	info, _ := ris.resources.Get(resource)
	info.UsePositions.Add(use)
	ris.resources.Set(resource, info)
}

func (ris *Resources) MarkUseAfterInvalidationReported(resource interface{}, pos ast.Position) {
	info, _ := ris.resources.Get(resource)
	info.UsePositions.MarkUseAfterInvalidationReported(pos)
	ris.resources.Set(resource, info)
}

func (ris *Resources) IsUseAfterInvalidationReported(resource interface{}, pos ast.Position) bool {
	info, _ := ris.resources.Get(resource)
	return info.UsePositions.IsUseAfterInvalidationReported(pos)
}

func (ris *Resources) Clone() *Resources {
	result := NewResources()
	result.Returns = ris.Returns
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

func (ris *Resources) ForEach(f func(resource interface{}, info ResourceInfo)) {
	ris.resources.Foreach(f)
}

// MergeBranches merges the given resources from two branches into these resources.
// Invalidations occurring in both branches are considered definitive,
// other new invalidations are only considered potential.
// The else resources are optional.
//
func (ris *Resources) MergeBranches(thenResources *Resources, elseResources *Resources) {

	elseReturns := false
	if elseResources != nil {
		elseReturns = elseResources.Returns
	}

	merged := make(map[interface{}]struct{})

	merge := func(resource interface{}) {

		// Only merge each resource once.
		// We iterate over the resources of the then-branch
		// and the else-branch (if it exists)

		if _, ok := merged[resource]; ok {
			return
		}
		defer func() {
			merged[resource] = struct{}{}
		}()

		// Get the resource info in this outer scope,
		// in the then-branch,
		// and if there is an else-branch, from it.

		info := ris.Get(resource)
		thenInfo := thenResources.Get(resource)
		var elseInfo ResourceInfo
		if elseResources != nil {
			elseInfo = elseResources.Get(resource)
		}

		// The resource can be considered definitely invalidated in both branches
		// if in both branches, there were invalidations or the branch returned.
		//
		// The assumption that a returning branch results in a definitive invalidation
		// can be made, because we check at the point of the return if the resource
		// was invalidated.

		definitelyInvalidatedInBranches :=
			(!thenInfo.Invalidations.IsEmpty() || thenResources.Returns) &&
				(!elseInfo.Invalidations.IsEmpty() || elseReturns)

		// The resource can be considered definitively invalidated if it was already invalidated,
		// or the resource was invalidated in both branches

		info.DefinitivelyInvalidated =
			info.DefinitivelyInvalidated ||
				definitelyInvalidatedInBranches

		// If the a branch returns, the invalidations and uses won't have occurred in the outer scope,
		// so only merge invalidations and uses if the branch did not return

		if !thenResources.Returns {
			info.Invalidations.Merge(thenInfo.Invalidations)
			info.UsePositions.Merge(thenInfo.UsePositions)
		}

		if !elseReturns {
			info.Invalidations.Merge(elseInfo.Invalidations)
			info.UsePositions.Merge(elseInfo.UsePositions)
		}

		ris.resources.Set(resource, info)
	}

	// Merge the resource info of all resources in the then-branch

	thenResources.ForEach(func(resource interface{}, _ ResourceInfo) {
		merge(resource)
	})

	// If there is an else-branch,
	// then merge the resource info of all resources in it

	if elseResources != nil {
		elseResources.ForEach(func(resource interface{}, _ ResourceInfo) {
			merge(resource)
		})
	}

	ris.Returns = ris.Returns ||
		(thenResources.Returns && elseReturns)
}
