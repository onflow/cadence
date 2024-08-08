package capcons

import (
	"sync"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type AccountCapability struct {
	Path       interpreter.PathValue
	BorrowType interpreter.StaticType
}

type AccountCapabilities struct {
	Capabilities []AccountCapability
}

func (c *AccountCapabilities) Record(path interpreter.PathValue, borrowType interpreter.StaticType) {
	c.Capabilities = append(
		c.Capabilities,
		AccountCapability{
			Path:       path,
			BorrowType: borrowType,
		},
	)
}

type AccountsCapabilities struct {
	// accountCapabilities maps common.Address to *AccountCapabilities
	accountCapabilities sync.Map
}

func (m *AccountsCapabilities) Record(
	addressPath interpreter.AddressPath,
	borrowType interpreter.StaticType,
) {
	var accountCapabilities *AccountCapabilities
	rawAccountCapabilities, ok := m.accountCapabilities.Load(addressPath.Address)
	if ok {
		accountCapabilities = rawAccountCapabilities.(*AccountCapabilities)
	} else {
		accountCapabilities = &AccountCapabilities{}
		m.accountCapabilities.Store(addressPath.Address, accountCapabilities)
	}
	accountCapabilities.Record(addressPath.Path, borrowType)
}

func (m *AccountsCapabilities) ForEach(
	address common.Address,
	f func(AccountCapability) bool,
) {
	rawAccountCapabilities, ok := m.accountCapabilities.Load(address)
	if !ok {
		return
	}

	accountCapabilities := rawAccountCapabilities.(*AccountCapabilities)
	for _, accountCapability := range accountCapabilities.Capabilities {
		if !f(accountCapability) {
			return
		}
	}
}

func (m *AccountsCapabilities) Get(address common.Address) *AccountCapabilities {
	rawAccountCapabilities, ok := m.accountCapabilities.Load(address)
	if !ok {
		return nil
	}
	return rawAccountCapabilities.(*AccountCapabilities)
}
