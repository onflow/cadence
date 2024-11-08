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

package common

import (
	"fmt"

	"github.com/onflow/cadence/errors"
)

type StorageDomain uint8

const (
	StorageDomainUnknown StorageDomain = iota

	StorageDomainPathStorage

	StorageDomainPathPrivate

	StorageDomainPathPublic

	StorageDomainContract

	StorageDomainInbox

	// StorageDomainCapabilityController is the storage domain which stores
	// capability controllers by capability ID
	StorageDomainCapabilityController

	// StorageDomainCapabilityControllerTag is the storage domain which stores
	// capability controller tags by capability ID
	StorageDomainCapabilityControllerTag

	// StorageDomainPathCapability is the storage domain which stores
	// capability ID dictionaries (sets) by storage path identifier
	StorageDomainPathCapability

	// StorageDomainAccountCapability is the storage domain which
	// records active account capability controller IDs
	StorageDomainAccountCapability
)

var AllStorageDomains = []StorageDomain{
	StorageDomainPathStorage,
	StorageDomainPathPrivate,
	StorageDomainPathPublic,
	StorageDomainContract,
	StorageDomainInbox,
	StorageDomainCapabilityController,
	StorageDomainCapabilityControllerTag,
	StorageDomainPathCapability,
	StorageDomainAccountCapability,
}

var AllStorageDomainsByIdentifier = map[string]StorageDomain{}

var allStorageDomainsSet = map[StorageDomain]struct{}{}

func init() {
	for _, domain := range AllStorageDomains {
		identifier := domain.Identifier()
		AllStorageDomainsByIdentifier[identifier] = domain

		allStorageDomainsSet[domain] = struct{}{}
	}
}

func StorageDomainFromIdentifier(domain string) (StorageDomain, bool) {
	result, ok := AllStorageDomainsByIdentifier[domain]
	if !ok {
		return StorageDomainUnknown, false
	}
	return result, true
}

func StorageDomainFromUint64(i uint64) (StorageDomain, error) {
	d := StorageDomain(i)
	_, exists := allStorageDomainsSet[d]
	if !exists {
		return StorageDomainUnknown, fmt.Errorf("failed to convert %d to StorageDomain", i)
	}
	return d, nil
}

func (d StorageDomain) Identifier() string {
	switch d {
	case StorageDomainPathStorage:
		return PathDomainStorage.Identifier()

	case StorageDomainPathPrivate:
		return PathDomainPrivate.Identifier()

	case StorageDomainPathPublic:
		return PathDomainPublic.Identifier()

	case StorageDomainContract:
		return "contract"

	case StorageDomainInbox:
		return "inbox"

	case StorageDomainCapabilityController:
		return "cap_con"

	case StorageDomainCapabilityControllerTag:
		return "cap_tag"

	case StorageDomainPathCapability:
		return "path_cap"

	case StorageDomainAccountCapability:
		return "acc_cap"
	}

	panic(errors.NewUnreachableError())
}
