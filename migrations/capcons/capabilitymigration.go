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
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityMigrationReporter interface {
	MigratedPathCapability(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
	)
	MissingCapabilityID(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
	)
}

type CapabilityMigration struct {
	CapabilityIDs map[interpreter.AddressPath]interpreter.UInt64Value
	Reporter      CapabilityMigrationReporter
}

var _ migrations.Migration = &CapabilityMigration{}

func (*CapabilityMigration) Name() string {
	return "CapabilityMigration"
}

var fullyEntitledAccountReferenceStaticType = interpreter.ConvertSemaReferenceTypeToStaticReferenceType(
	nil,
	sema.FullyEntitledAccountReferenceType,
)

// Migrate migrates a path capability to an ID capability in the given value.
// If a value is returned, the value must be updated with the replacement in the parent.
// If nil is returned, the value was not updated and no operation has to be performed.
func (m *CapabilityMigration) Migrate(
	addressPath interpreter.AddressPath,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) interpreter.Value {
	reporter := m.Reporter

	switch value := value.(type) {
	case *interpreter.PathCapabilityValue: //nolint:staticcheck

		// Migrate the path capability to an ID capability

		oldCapability := value

		capabilityAddressPath := oldCapability.AddressPath()
		capabilityID, ok := m.CapabilityIDs[capabilityAddressPath]
		if !ok {
			if reporter != nil {
				reporter.MissingCapabilityID(
					addressPath.Address,
					capabilityAddressPath,
				)
			}
			break
		}

		newBorrowType, ok := oldCapability.BorrowType.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Convert the old AuthAccount type to the new fully-entitled Account type
		if newBorrowType.ReferencedType == interpreter.PrimitiveStaticTypeAuthAccount { //nolint:staticcheck
			newBorrowType = fullyEntitledAccountReferenceStaticType
		}

		newCapability := interpreter.NewUnmeteredCapabilityValue(
			capabilityID,
			oldCapability.Address,
			newBorrowType,
		)

		if reporter != nil {
			reporter.MigratedPathCapability(
				addressPath.Address,
				capabilityAddressPath,
				newBorrowType,
			)
		}

		return newCapability
	}

	return nil
}
