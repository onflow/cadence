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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

type sharedState struct {
	allInterpreters map[common.Location]*Interpreter
	callStack       *CallStack
	typeCodes       TypeCodes
	// TODO: ideally this would be a weak map, but Go has no weak references
	referencedResourceKindedValues ReferencedResourceKindedValues
}

func newSharedState() *sharedState {
	return &sharedState{
		allInterpreters: map[common.Location]*Interpreter{},
		callStack:       &CallStack{},
		typeCodes: TypeCodes{
			CompositeCodes:       map[sema.TypeID]CompositeTypeCode{},
			InterfaceCodes:       map[sema.TypeID]WrapperCode{},
			TypeRequirementCodes: map[sema.TypeID]WrapperCode{},
		},
		referencedResourceKindedValues: map[atree.StorageID]map[ReferenceTrackedResourceKindedValue]struct{}{},
	}
}
