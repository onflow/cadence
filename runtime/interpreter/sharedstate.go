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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
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
	resourceVariables                           map[ResourceKindedValue]*Variable
	inStorageIteration                          bool
	storageMutatedDuringIteration               bool
	CapabilityControllerIterations              map[AddressPath]int
	MutationDuringCapabilityControllerIteration bool
	containerValueIteration                     map[atree.StorageID]struct{}
	resourceDestruction                         map[atree.StorageID]struct{}
}

func NewSharedState(config *Config) *SharedState {
	return &SharedState{
		Config:          config,
		allInterpreters: map[common.Location]*Interpreter{},
		callStack:       &CallStack{},
		typeCodes: TypeCodes{
			CompositeCodes:       map[sema.TypeID]CompositeTypeCode{},
			InterfaceCodes:       map[sema.TypeID]WrapperCode{},
			TypeRequirementCodes: map[sema.TypeID]WrapperCode{},
		},
		inStorageIteration:             false,
		storageMutatedDuringIteration:  false,
		referencedResourceKindedValues: map[atree.StorageID]map[ReferenceTrackedResourceKindedValue]struct{}{},
		resourceVariables:              map[ResourceKindedValue]*Variable{},
		CapabilityControllerIterations: map[AddressPath]int{},
		containerValueIteration:        map[atree.StorageID]struct{}{},
		resourceDestruction:            map[atree.StorageID]struct{}{},
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
