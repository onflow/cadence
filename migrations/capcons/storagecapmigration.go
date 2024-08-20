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
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

type StorageCapabilityMigrationReporter interface {
	MissingBorrowType(
		targetPath interpreter.AddressPath,
		storedPath interpreter.AddressPath,
	)
	IssuedStorageCapabilityController(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
		capabilityID interpreter.UInt64Value,
	)
	InferredMissingBorrowType(
		targetPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
		storedPath interpreter.AddressPath,
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
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
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
			storageKey,
			storageMapKey,
		)
	}

	return nil, nil
}

func (m *StorageCapMigration) CanSkip(valueType interpreter.StaticType) bool {
	return CanSkipCapabilityValueMigration(valueType)
}

func IssueAccountCapabilities(
	inter *interpreter.Interpreter,
	storage *runtime.Storage,
	reporter StorageCapabilityMigrationReporter,
	address common.Address,
	capabilities *AccountCapabilities,
	handler stdlib.CapabilityControllerIssueHandler,
	typedCapabilityMapping *PathTypeCapabilityMapping,
	untypedCapabilityMapping *PathCapabilityMapping,
	inferAuth func(valueType interpreter.StaticType) interpreter.Authorization,
) {

	storageMap := storage.GetStorageMap(
		address,
		common.PathDomainStorage.Identifier(),
		false,
	)

	capabilities.ForEachSorted(func(capability AccountCapability) bool {

		addressPath := interpreter.AddressPath{
			Address: address,
			Path:    capability.TargetPath,
		}

		capabilityBorrowType := capability.BorrowType
		hasBorrowType := capabilityBorrowType != nil

		var borrowType *interpreter.ReferenceStaticType

		if hasBorrowType {
			if _, ok := typedCapabilityMapping.Get(addressPath, capabilityBorrowType.ID()); ok {
				return true
			}

			borrowType = capabilityBorrowType.(*interpreter.ReferenceStaticType)

		} else {
			targetPath := interpreter.AddressPath{
				Address: address,
				Path: interpreter.PathValue{
					Identifier: capability.StoredPath.Path,
					Domain:     common.PathDomainFromIdentifier(capability.StoredPath.Domain),
				},
			}

			// Report for all caps with missing borrow type.
			reporter.MissingBorrowType(addressPath, targetPath)

			if _, _, ok := untypedCapabilityMapping.Get(addressPath); ok {
				return true
			}

			// If the borrow type is missing, then borrow it as the type of the value.
			path := capability.TargetPath.Identifier
			value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(path))

			// However, if there is no value at the target,
			//it is not possible to migrate this cap.
			if value == nil {
				return true
			}

			valueType := value.StaticType(inter)

			borrowType = interpreter.NewReferenceStaticType(
				nil,
				inferAuth(valueType),
				valueType,
			)

			reporter.InferredMissingBorrowType(
				addressPath,
				borrowType,
				targetPath,
			)
		}

		capabilityID := stdlib.IssueStorageCapabilityController(
			inter,
			interpreter.EmptyLocationRange,
			handler,
			address,
			borrowType,
			capability.TargetPath,
		)

		if hasBorrowType {
			typedCapabilityMapping.Record(
				addressPath,
				capabilityID,
				capabilityBorrowType.ID(),
			)
		} else {
			untypedCapabilityMapping.Record(
				addressPath,
				capabilityID,
				borrowType,
			)
		}

		reporter.IssuedStorageCapabilityController(
			address,
			addressPath,
			borrowType,
			capabilityID,
		)

		return true
	})
}
