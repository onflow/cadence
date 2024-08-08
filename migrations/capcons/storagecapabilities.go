package capcons

import (
	"sync"

	"github.com/onflow/cadence/runtime/interpreter"
)

type AddressPaths struct {
	// set is a "set" of common.AddressPath (Go map of common.AddressPath ((address, path) pairs) to struct{})
	set sync.Map
}

func (m *AddressPaths) Record(
	addressPath interpreter.AddressPath,
) {
	_, ok := m.set.Load(addressPath.Address)
	if !ok {
		m.set.Store(addressPath, struct{}{})
	}
}

func (m *AddressPaths) ForEach(f func(addressPath interpreter.AddressPath) bool) {
	m.set.Range(func(key, _ interface{}) bool {
		addressPath := key.(interpreter.AddressPath)
		return f(addressPath)
	})
}
