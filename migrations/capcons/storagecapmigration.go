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
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

type StorageCapabilityMigrationReporter interface {
	MissingBorrowType(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
	)
	IssuedStorageCapabilityController(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
		capabilityID interpreter.UInt64Value,
	)
}

// StorageCapMigration records path capabilities with storage domain target.
// It does not actually migrate any values.
type StorageCapMigration struct {
	StorageDomainCapabilities *AccountsCapabilities
}

var _ migrations.ValueMigration = &StorageCapMigration{}

func (*StorageCapMigration) Name() string {
	return "StorageCapMigration"
}

func (*StorageCapMigration) Domains() map[string]struct{} {
	return nil
}

// Migrate records path capabilities with storage domain target.
// It does not actually migrate any values.
func (m *StorageCapMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ migrations.ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	// Record path capabilities with storage domain target
	if pathCapabilityValue, ok := value.(*interpreter.PathCapabilityValue); ok && //nolint:staticcheck
		pathCapabilityValue.Path.Domain == common.PathDomainStorage {

		m.StorageDomainCapabilities.Record(
			pathCapabilityValue.AddressPath(),
			pathCapabilityValue.BorrowType,
		)
	}

	return nil, nil
}

func (m *StorageCapMigration) CanSkip(valueType interpreter.StaticType) bool {
	return CanSkipCapabilityValueMigration(valueType)
}

func IssueAccountCapabilities(
	inter *interpreter.Interpreter,
	reporter StorageCapabilityMigrationReporter,
	address common.Address,
	capabilities *AccountCapabilities,
	handler stdlib.CapabilityControllerIssueHandler,
	typedCapabilityMapping *PathTypeCapabilityMapping,
	untypedCapabilityMapping *PathCapabilityMapping,
	storage interpreter.Storage,
) {
	storageMap := storage.GetStorageMap(
		address,
		common.PathDomainStorage.Identifier(),
		false,
	)

	for _, capability := range capabilities.Capabilities {

		addressPath := interpreter.AddressPath{
			Address: address,
			Path:    capability.Path,
		}

		borrowStaticType := capability.BorrowType
		if borrowStaticType == nil {
			reporter.MissingBorrowType(address, addressPath)

			// If the borrow type is missing,
			// infer the borrow type from the value stored at the target path (if any)

			path := capability.Path.Identifier
			value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(path))
			if value == nil {
				continue
			}

			borrowStaticType = interpreter.NewReferenceStaticType(
				nil,
				// TODO: might want to add all entitlements via sema.AllSupportedEntitlements
				interpreter.UnauthorizedAccess,
				value.StaticType(inter),
			)
		}

		if _, ok := typedCapabilityMapping.Get(addressPath, borrowStaticType.ID()); ok {
			continue
		}

		borrowType := borrowStaticType.(*interpreter.ReferenceStaticType)

		capabilityID := stdlib.IssueStorageCapabilityController(
			inter,
			interpreter.EmptyLocationRange,
			handler,
			address,
			borrowType,
			capability.Path,
		)

		typedCapabilityMapping.Record(
			addressPath,
			capabilityID,
			borrowStaticType.ID(),
		)

		untypedCapabilityMapping.Record(
			addressPath,
			capabilityID,
			borrowType,
		)

		reporter.IssuedStorageCapabilityController(
			address,
			addressPath,
			borrowType,
			capabilityID,
		)
	}
}
