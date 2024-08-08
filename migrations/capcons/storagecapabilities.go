package capcons

import (
	"sync"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// AccountStorageCapabilities maps path to borrow type
type AccountStorageCapabilities map[interpreter.PathValue]interpreter.StaticType

type StorageCapabilities struct {
	// accountStorageCapabilities maps common.Address to AccountStorageCapabilities
	accountStorageCapabilities sync.Map
}

func (m *StorageCapabilities) Record(
	addressPath interpreter.AddressPath,
	borrowType interpreter.StaticType,
) {
	var capabilityEntryMap AccountStorageCapabilities
	rawCapabilityEntryMap, ok := m.accountStorageCapabilities.Load(addressPath.Address)
	if ok {
		capabilityEntryMap = rawCapabilityEntryMap.(AccountStorageCapabilities)
	} else {
		capabilityEntryMap = AccountStorageCapabilities{}
		m.accountStorageCapabilities.Store(addressPath.Address, capabilityEntryMap)
	}
	capabilityEntryMap[addressPath.Path] = borrowType
}

func (m *StorageCapabilities) ForEach(
	address common.Address,
	f func(path interpreter.PathValue, borrowType interpreter.StaticType) bool,
) {
	rawCapabilityEntryMap, ok := m.accountStorageCapabilities.Load(address)
	if !ok {
		return
	}

	capabilityEntryMap := rawCapabilityEntryMap.(AccountStorageCapabilities)
	for path, borrowType := range capabilityEntryMap {
		if !f(path, borrowType) {
			return
		}
	}
}
