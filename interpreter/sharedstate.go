/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type AddressPath struct {
	Address common.Address
	Path    PathValue
}

type SharedState struct {
	attachmentIterationMap map[*CompositeValue]bool
	typeCodes              TypeCodes
	Config                 *Config
	allInterpreters        map[common.Location]*Interpreter
	callStack              *CallStack
	// TODO: ideally this would be a weak map, but Go has no weak references
	referencedResourceKindedValues              ReferencedResourceKindedValues
	resourceVariables                           map[ResourceKindedValue]Variable
	inStorageIteration                          bool
	storageMutatedDuringIteration               bool
	CapabilityControllerIterations              map[AddressPath]int
	MutationDuringCapabilityControllerIteration bool
	containerValueIteration                     map[atree.ValueID]struct{}
	destroyedResources                          map[atree.ValueID]struct{}
	// canonicalAtreeContainers deduplicates the Cadence-level wrappers
	// (ArrayValue, DictionaryValue, CompositeValue) created for atree
	// containers, keyed by their atree value ID. The first wrapper created
	// for a given value ID via `ConvertStoredValue` becomes canonical;
	// subsequent retrievals reuse the same wrapper instance. The wrapper's
	// `array`/`dictionary` field is updated on each retrieval to the
	// `*atree.Array`/`*atree.OrderedMap` instance that atree just handed
	// out (and on which it set up the parent updater), so operations
	// through the canonical wrapper notify the parent correctly.
	//
	// Without this, atree.Array.Get / OrderedMap.Get returns a fresh
	// `*atree.Array`/`*atree.OrderedMap` per call - two `&outer[0]`
	// evaluations would hold separate wrappers around separate atree
	// instances over the same slab. An atree slab split through one
	// wrapper reassigns root pointers on that instance only; the others go
	// stale and mutations through them silently corrupt the container.
	//
	// Entries are removed when a wrapper is invalidated by Destroy or
	// Transfer (its `array`/`dictionary` is nilled). Subsequent retrievals
	// for the same value ID create a fresh wrapper, leaving the held
	// reference to the invalidated wrapper unaffected.
	canonicalAtreeContainers map[atree.ValueID]Value
}

func NewSharedState(config *Config) *SharedState {
	return &SharedState{
		Config:          config,
		allInterpreters: map[common.Location]*Interpreter{},
		callStack:       &CallStack{},
		typeCodes: TypeCodes{
			CompositeCodes: map[sema.TypeID]CompositeTypeCode{},
			InterfaceCodes: map[sema.TypeID]WrapperCode{},
		},
		inStorageIteration:             false,
		storageMutatedDuringIteration:  false,
		referencedResourceKindedValues: ReferencedResourceKindedValues{},
		resourceVariables:              map[ResourceKindedValue]Variable{},
		CapabilityControllerIterations: map[AddressPath]int{},
		containerValueIteration:        map[atree.ValueID]struct{}{},
		destroyedResources:             map[atree.ValueID]struct{}{},
		canonicalAtreeContainers:       map[atree.ValueID]Value{},
	}
}

func (s *SharedState) inAttachmentIteration(base *CompositeValue) bool {
	return s.attachmentIterationMap[base]
}

func (s *SharedState) setAttachmentIteration(base *CompositeValue, b bool) {
	if s.attachmentIterationMap == nil {
		s.attachmentIterationMap = map[*CompositeValue]bool{}
	}
	s.attachmentIterationMap[base] = b
}
