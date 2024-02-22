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

package capcons

import (
	"sync"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityEntry struct {
	CapabilityID interpreter.UInt64Value
	BorrowType   *sema.ReferenceType
}

type CapabilityEntryMap map[interpreter.PathValue]CapabilityEntry

type CapabilityMapping struct {
	// capabilityEntries maps common.Address to CapabilityEntryMap
	capabilityEntries sync.Map
}

func (m *CapabilityMapping) Record(
	addressPath interpreter.AddressPath,
	capabilityID interpreter.UInt64Value,
	borrowType *sema.ReferenceType,
) {
	var capabilityEntryMap CapabilityEntryMap
	rawCapabilityEntryMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if ok {
		capabilityEntryMap = rawCapabilityEntryMap.(CapabilityEntryMap)
	} else {
		capabilityEntryMap = CapabilityEntryMap{}
		m.capabilityEntries.Store(addressPath.Address, capabilityEntryMap)
	}
	capabilityEntryMap[addressPath.Path] = CapabilityEntry{
		CapabilityID: capabilityID,
		BorrowType:   borrowType,
	}
}

func (m *CapabilityMapping) Get(addressPath interpreter.AddressPath) (interpreter.UInt64Value, sema.Type, bool) {
	rawCapabilityEntryMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if !ok {
		return 0, nil, false
	}
	capabilityEntryMap := rawCapabilityEntryMap.(CapabilityEntryMap)
	capabilityEntry, ok := capabilityEntryMap[addressPath.Path]
	return capabilityEntry.CapabilityID, capabilityEntry.BorrowType, ok
}
