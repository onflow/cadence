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
	resources map[interface{}]ResourceInfo
	Returns   bool
}

func NewResources() *Resources {
	return &Resources{
		resources: make(map[interface{}]ResourceInfo),
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
	return ris.resources[resource]
}

// AddInvalidation adds the given invalidation to the set of invalidations for the given resource.
// If the invalidation is not temporary, marks the resource to be definitely invalidated.
//
func (ris *Resources) AddInvalidation(resource interface{}, invalidation ResourceInvalidation) {
	info := ris.resources[resource]
	info.Invalidations.Add(invalidation)
	if invalidation.Kind.IsDefinite() {
		info.DefinitivelyInvalidated = true
	}
	ris.resources[resource] = info
}

// RemoveTemporaryMoveInvalidation removes the given invalidation
// from the set of invalidations for the given resource.
//
func (ris *Resources) RemoveTemporaryMoveInvalidation(resource interface{}, invalidation ResourceInvalidation) {
	if invalidation.Kind != ResourceInvalidationKindMoveTemporary {
		panic(errors.NewUnreachableError())
	}

	info := ris.resources[resource]
	info.Invalidations.DeleteLocally(invalidation)
	ris.resources[resource] = info
}

// AddUse adds the given use position to the set of use positions for the given resource.
//
func (ris *Resources) AddUse(resource interface{}, use ast.Position) {
	info := ris.resources[resource]
	info.UsePositions.Add(use)
	ris.resources[resource] = info
}

func (ris *Resources) MarkUseAfterInvalidationReported(resource interface{}, pos ast.Position) {
	info := ris.resources[resource]
	info.UsePositions.MarkUseAfterInvalidationReported(pos)
	ris.resources[resource] = info
}

func (ris *Resources) IsUseAfterInvalidationReported(resource interface{}, pos ast.Position) bool {
	info := ris.resources[resource]
	return info.UsePositions.IsUseAfterInvalidationReported(pos)
}

func (ris *Resources) Clone() *Resources {
	result := NewResources()
	result.Returns = ris.Returns
	for resource, info := range ris.resources {
		result.resources[resource] = info.Clone()
	}
	return result
}

func (ris *Resources) Size() int {
	return len(ris.resources)
}

func (ris *Resources) ForEach(f func(resource interface{}, info ResourceInfo)) {
	for resource, info := range ris.resources {
		f(resource, info)
	}
}

// MergeBranches merges the given resources from two branches into these resources.
// Invalidations occurring in both branches are considered definitive,
// other new invalidations are only considered potential.
// The else resources are optional.
//
func (ris *Resources) MergeBranches(thenResources *Resources, elseResources *Resources) {

	infoTuples := NewBranchesResourceInfos(thenResources, elseResources)

	elseReturns := false
	if elseResources != nil {
		elseReturns = elseResources.Returns
	}

	for resource, infoTuple := range infoTuples {
		info := ris.resources[resource]

		// The resource can be considered definitely invalidated in both branches
		// if in both branches, there were invalidations or the branch returned.
		//
		// The assumption that a returning branch results in a definitive invalidation
		// can be made, because we check at the point of the return if the resource
		// was invalidated.

		definitelyInvalidatedInBranches :=
			(!infoTuple.thenInfo.Invalidations.IsEmpty() || thenResources.Returns) &&
				(!infoTuple.elseInfo.Invalidations.IsEmpty() || elseReturns)

		// The resource can be considered definitively invalidated if it was already invalidated,
		// or the resource was invalidated in both branches

		info.DefinitivelyInvalidated =
			info.DefinitivelyInvalidated ||
				definitelyInvalidatedInBranches

		// If the a branch returns, the invalidations and uses won't have occurred in the outer scope,
		// so only merge invalidations and uses if the branch did not return

		if !thenResources.Returns {
			info.Invalidations.Merge(infoTuple.thenInfo.Invalidations)
			info.UsePositions.Merge(infoTuple.thenInfo.UsePositions)
		}

		if !elseReturns {
			info.Invalidations.Merge(infoTuple.elseInfo.Invalidations)
			info.UsePositions.Merge(infoTuple.elseInfo.UsePositions)
		}

		ris.resources[resource] = info
	}

	ris.Returns = ris.Returns ||
		(thenResources.Returns && elseReturns)
}

type BranchesResourceInfo struct {
	thenInfo ResourceInfo
	elseInfo ResourceInfo
}

type BranchesResourceInfos map[interface{}]BranchesResourceInfo

func (infos BranchesResourceInfos) Add(
	resources *Resources,
	setValue func(*BranchesResourceInfo, ResourceInfo),
) {
	resources.ForEach(func(resource interface{}, info ResourceInfo) {
		branchesResourceInfo := infos[resource]
		setValue(&branchesResourceInfo, info)
		infos[resource] = branchesResourceInfo
	})
}

func NewBranchesResourceInfos(thenResources *Resources, elseResources *Resources) BranchesResourceInfos {
	infoTuples := make(BranchesResourceInfos)
	infoTuples.Add(
		thenResources,
		func(branchesResourceInfo *BranchesResourceInfo, resourceInfo ResourceInfo) {
			branchesResourceInfo.thenInfo = resourceInfo
		},
	)
	if elseResources != nil {
		infoTuples.Add(
			elseResources,
			func(branchesResourceInfo *BranchesResourceInfo, resourceInfo ResourceInfo) {
				branchesResourceInfo.elseInfo = resourceInfo
			},
		)
	}
	return infoTuples
}
