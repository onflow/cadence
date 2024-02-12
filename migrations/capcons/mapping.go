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
)

type CapabilityIDMapping struct {
	// CapabilityIDs' maps common.Address to map[interpreter.PathValue]interpreter.UInt64Value
	capabilityIDs sync.Map
}

func (m *CapabilityIDMapping) Record(addressPath interpreter.AddressPath, capabilityID interpreter.UInt64Value) {
	var accountCapabilityIDs map[interpreter.PathValue]interpreter.UInt64Value
	rawAccountCapabilityIDs, ok := m.capabilityIDs.Load(addressPath.Address)
	if ok {
		accountCapabilityIDs = rawAccountCapabilityIDs.(map[interpreter.PathValue]interpreter.UInt64Value)
	} else {
		accountCapabilityIDs = map[interpreter.PathValue]interpreter.UInt64Value{}
		m.capabilityIDs.Store(addressPath.Address, accountCapabilityIDs)
	}
	accountCapabilityIDs[addressPath.Path] = capabilityID
}

func (m *CapabilityIDMapping) Get(addressPath interpreter.AddressPath) (interpreter.UInt64Value, bool) {
	rawAccountCapabilityIDs, ok := m.capabilityIDs.Load(addressPath.Address)
	if !ok {
		return 0, false
	}
	accountCapabilityIDs := rawAccountCapabilityIDs.(map[interpreter.PathValue]interpreter.UInt64Value)
	capabilityID, ok := accountCapabilityIDs[addressPath.Path]
	return capabilityID, ok
}
