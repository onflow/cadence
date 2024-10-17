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

package capcons

import (
	"sync"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
)

// Path capability mappings map an address and path to a capability ID and borrow type

type PathCapabilityEntry struct {
	CapabilityID interpreter.UInt64Value
	BorrowType   *interpreter.ReferenceStaticType
}

type PathCapabilityEntryMap map[interpreter.PathValue]PathCapabilityEntry

type PathCapabilityMapping struct {
	// capabilityEntries maps common.Address to PathCapabilityEntryMap
	capabilityEntries sync.Map
}

func (m *PathCapabilityMapping) Record(
	addressPath interpreter.AddressPath,
	capabilityID interpreter.UInt64Value,
	borrowType *interpreter.ReferenceStaticType,
) {
	var capMap PathCapabilityEntryMap
	rawCapMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if ok {
		capMap = rawCapMap.(PathCapabilityEntryMap)
	} else {
		capMap = PathCapabilityEntryMap{}
		m.capabilityEntries.Store(addressPath.Address, capMap)
	}
	capMap[addressPath.Path] = PathCapabilityEntry{
		CapabilityID: capabilityID,
		BorrowType:   borrowType,
	}
}

func (m *PathCapabilityMapping) Get(addressPath interpreter.AddressPath) (interpreter.UInt64Value, *interpreter.ReferenceStaticType, bool) {
	rawCapabilityEntryMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if !ok {
		return 0, nil, false
	}
	capabilityEntryMap := rawCapabilityEntryMap.(PathCapabilityEntryMap)
	capabilityEntry, ok := capabilityEntryMap[addressPath.Path]
	return capabilityEntry.CapabilityID, capabilityEntry.BorrowType, ok
}

// Path/Type mappings map an address, path, and borrow type to a capability ID

type PathTypeCapabilityKey struct {
	Path       interpreter.PathValue
	BorrowType common.TypeID
}

type PathTypeCapabilityEntryMap map[PathTypeCapabilityKey]interpreter.UInt64Value

type PathTypeCapabilityMapping struct {
	// capabilityEntries maps common.Address to PathTypeCapabilityEntryMap
	capabilityEntries sync.Map
}

func (m *PathTypeCapabilityMapping) Record(
	addressPath interpreter.AddressPath,
	capabilityID interpreter.UInt64Value,
	borrowType common.TypeID,
) {
	var capMap PathTypeCapabilityEntryMap
	rawCapMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if ok {
		capMap = rawCapMap.(PathTypeCapabilityEntryMap)
	} else {
		capMap = PathTypeCapabilityEntryMap{}
		m.capabilityEntries.Store(addressPath.Address, capMap)
	}
	key := PathTypeCapabilityKey{
		Path:       addressPath.Path,
		BorrowType: borrowType,
	}
	capMap[key] = capabilityID
}

func (m *PathTypeCapabilityMapping) Get(
	addressPath interpreter.AddressPath,
	borrowType common.TypeID,
) (interpreter.UInt64Value, bool) {
	rawCapMap, ok := m.capabilityEntries.Load(addressPath.Address)
	if !ok {
		return 0, false
	}
	capMap := rawCapMap.(PathTypeCapabilityEntryMap)
	key := PathTypeCapabilityKey{
		Path:       addressPath.Path,
		BorrowType: borrowType,
	}
	capabilityID, ok := capMap[key]
	return capabilityID, ok
}
