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
	"fmt"
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type AccountCapability struct {
	TargetPath interpreter.PathValue
	BorrowType interpreter.StaticType
	StoredPath Path
}

type Path struct {
	Domain string
	Path   string
}

type AccountCapabilities struct {
	Capabilities []AccountCapability
}

func (c *AccountCapabilities) Record(
	path interpreter.PathValue,
	borrowType interpreter.StaticType,
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
) {
	c.Capabilities = append(
		c.Capabilities,
		AccountCapability{
			TargetPath: path,
			BorrowType: borrowType,
			StoredPath: Path{
				Domain: storageKey.Key,
				Path:   fmt.Sprintf("%s", storageMapKey),
			},
		},
	)
}

func (c *AccountCapabilities) ForEach(
	f func(AccountCapability) bool,
) {
	slices.SortFunc(
		c.Capabilities,
		func(a, b AccountCapability) int {
			pathA := a.TargetPath
			pathB := b.TargetPath

			if pathA.Domain == pathB.Domain {
				return strings.Compare(pathA.Identifier, pathB.Identifier)
			}

			if pathA.Domain < pathB.Domain {
				return -1
			}

			return +1
		},
	)

	for _, accountCapability := range c.Capabilities {
		if !f(accountCapability) {
			return
		}
	}
}

type AccountsCapabilities struct {
	// accountCapabilities maps common.Address to *AccountCapabilities
	accountCapabilities sync.Map
}

func (m *AccountsCapabilities) Record(
	addressPath interpreter.AddressPath,
	borrowType interpreter.StaticType,
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
) {
	var accountCapabilities *AccountCapabilities
	rawAccountCapabilities, ok := m.accountCapabilities.Load(addressPath.Address)
	if ok {
		accountCapabilities = rawAccountCapabilities.(*AccountCapabilities)
	} else {
		accountCapabilities = &AccountCapabilities{}
		m.accountCapabilities.Store(addressPath.Address, accountCapabilities)
	}
	accountCapabilities.Record(
		addressPath.Path,
		borrowType,
		storageKey,
		storageMapKey,
	)
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

	accountCapabilities.ForEach(f)
}

func (m *AccountsCapabilities) Get(address common.Address) *AccountCapabilities {
	rawAccountCapabilities, ok := m.accountCapabilities.Load(address)
	if !ok {
		return nil
	}
	return rawAccountCapabilities.(*AccountCapabilities)
}
