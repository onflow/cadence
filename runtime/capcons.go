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

package runtime

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

type AddressIterator interface {
	NextAddress() common.Address
	Reset()
}

type AddressSliceIterator struct {
	Addresses []common.Address
	index     int
}

var _ AddressIterator = &AddressSliceIterator{}

func (a *AddressSliceIterator) NextAddress() common.Address {
	index := a.index
	if index >= len(a.Addresses) {
		return common.ZeroAddress
	}
	address := a.Addresses[index]
	a.index++
	return address
}

func (a *AddressSliceIterator) Reset() {
	a.index = 0
}

type CapConsMigrationReporter interface {
	CapConsLinkMigrationReporter
	CapConsPathCapabilityMigrationReporter
}

type CapConsLinkMigrationReporter interface {
	MigratedLink(
		addressPath interpreter.AddressPath,
		capabilityID interpreter.UInt64Value,
	)
}

type CapConsPathCapabilityMigrationReporter interface {
	MigratedPathCapability(
		address common.Address,
		addressPath interpreter.AddressPath,
	)
	MissingCapabilityID(
		address common.Address,
		addressPath interpreter.AddressPath,
	)
}

type CapConsMigration struct {
	storage       *Storage
	interpreter   *interpreter.Interpreter
	capabilityIDs map[interpreter.AddressPath]interpreter.UInt64Value
}

func NewCapConsMigration(runtime Runtime, context Context) (*CapConsMigration, error) {
	storage, inter, err := runtime.Storage(context)
	if err != nil {
		return nil, err
	}

	return &CapConsMigration{
		storage:     storage,
		interpreter: inter,
	}, nil
}

// Migrate migrates the links to capability controllers,
// and all path capabilities and account capabilities to ID capabilities,
// in all accounts of the given iterator.
func (m *CapConsMigration) Migrate(
	addressIterator AddressIterator,
	accountIDGenerator stdlib.AccountIDGenerator,
	reporter CapConsMigrationReporter,
) error {
	m.capabilityIDs = make(map[interpreter.AddressPath]interpreter.UInt64Value)
	defer func() {
		m.capabilityIDs = nil
	}()
	m.migrateLinks(
		addressIterator,
		accountIDGenerator,
		reporter,
	)

	addressIterator.Reset()
	m.migratePathCapabilities(
		addressIterator,
		reporter,
	)

	return m.storage.Commit(m.interpreter, false)
}

// migrateLinks migrates the links to capability controllers
// in all accounts of the given iterator.
// It constructs a source path to capability ID mapping,
// which is later needed to path capabilities to ID capabilities.
func (m *CapConsMigration) migrateLinks(
	addressIterator AddressIterator,
	accountIDGenerator stdlib.AccountIDGenerator,
	reporter CapConsLinkMigrationReporter,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migrateLinksInAccount(
			address,
			accountIDGenerator,
			reporter,
		)
	}
}

// migrateLinksInAccount migrates the links in the given account to capability controllers
// It records an entry in the source path to capability ID mapping,
// which is later needed to migrate path capabilities to ID capabilities.
func (m *CapConsMigration) migrateLinksInAccount(
	address common.Address,
	accountIDGenerator stdlib.AccountIDGenerator,
	reporter CapConsLinkMigrationReporter,
) {

	migrateDomain := func(domain common.PathDomain) {
		m.migrateAccountLinksInAccountDomain(
			address,
			accountIDGenerator,
			domain,
			reporter,
		)
	}

	migrateDomain(common.PathDomainPublic)
	migrateDomain(common.PathDomainPrivate)
}

// migrateAccountLinksInAccountDomain migrates the links in the given account's storage domain
// to capability controllers.
// It records an entry in the source path to capability ID mapping,
// which is later needed to migrate path capabilities to ID capabilities.
func (m *CapConsMigration) migrateAccountLinksInAccountDomain(
	address common.Address,
	accountIDGenerator stdlib.AccountIDGenerator,
	domain common.PathDomain,
	reporter CapConsLinkMigrationReporter,
) {
	addressValue := interpreter.AddressValue(address)

	storageMap := m.storage.GetStorageMap(address, domain.Identifier(), false)
	if storageMap == nil {
		return
	}

	iterator := storageMap.Iterator(m.interpreter)

	count := storageMap.Count()
	if count > 0 {
		for key := iterator.NextKey(); key != nil; key = iterator.NextKey() {
			// TODO: unfortunately, the iterator only returns an atree.Value, not a StorageMapKey
			identifier := string(key.(interpreter.StringAtreeValue))

			pathValue := interpreter.NewUnmeteredPathValue(domain, identifier)

			m.migrateLink(
				addressValue,
				pathValue,
				accountIDGenerator,
				reporter,
			)
		}
	}
}

// migrateAccountLinksInAccountDomain migrates the links in the given account's storage domain
// to capability controllers.
// It constructs a source path to ID mapping,
// which is later needed to migrate path capabilities to ID capabilities.
func (m *CapConsMigration) migrateLink(
	address interpreter.AddressValue,
	path interpreter.PathValue,
	accountIDGenerator stdlib.AccountIDGenerator,
	reporter CapConsLinkMigrationReporter,
) {
	capabilityID := stdlib.MigrateLinkToCapabilityController(
		m.interpreter,
		interpreter.EmptyLocationRange,
		address,
		path,
		accountIDGenerator,
	)
	if capabilityID == 0 {
		return
	}

	// Record new capability ID in source path mapping.
	// The mapping is used later for migrating path capabilities to ID capabilities.

	addressPath := interpreter.AddressPath{
		Address: address.ToAddress(),
		Path:    path,
	}
	m.capabilityIDs[addressPath] = capabilityID

	if reporter != nil {
		reporter.MigratedLink(addressPath, capabilityID)
	}
}

// migratePathCapabilities migrates the path capabilities to ID capabilities
// in all accounts of the given iterator.
// It uses the source path to capability ID mapping which was constructed in migrateLinks.
func (m *CapConsMigration) migratePathCapabilities(
	addressIterator AddressIterator,
	reporter CapConsPathCapabilityMigrationReporter,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migratePathCapabilitiesInAccount(address, reporter)
	}
}

var pathDomainStorage = common.PathDomainStorage.Identifier()

func (m *CapConsMigration) migratePathCapabilitiesInAccount(address common.Address, reporter CapConsPathCapabilityMigrationReporter) {

	storageMap := m.storage.GetStorageMap(address, pathDomainStorage, false)
	if storageMap == nil {
		return
	}

	iterator := storageMap.Iterator(m.interpreter)

	count := storageMap.Count()
	if count > 0 {
		for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {

			newValue := m.migratePathCapability(
				address,
				value,
				reporter,
			)

			if newValue != nil {
				// TODO: unfortunately, the iterator only returns an atree.Value, not a StorageMapKey
				identifier := string(key.(interpreter.StringAtreeValue))
				storageMap.SetValue(
					m.interpreter,
					interpreter.StringStorageMapKey(identifier),
					newValue,
				)
			}
		}
	}
}

// migratePathCapability migrates a path capability to an ID capability in the given value.
// If a value is returned, the value must be updated with the replacement in the parent.
// If nil is returned, the value was not updated and no operation has to be performed.
func (m *CapConsMigration) migratePathCapability(
	address common.Address,
	value interpreter.Value,
	reporter CapConsPathCapabilityMigrationReporter,
) interpreter.Value {
	locationRange := interpreter.EmptyLocationRange

	switch value := value.(type) {
	case *interpreter.PathCapabilityValue:

		// Migrate the path capability to an ID capability

		oldCapability := value

		addressPath := oldCapability.AddressPath()
		capabilityID, ok := m.capabilityIDs[addressPath]
		if !ok {
			if reporter != nil {
				reporter.MissingCapabilityID(address, addressPath)
			}
			break
		}

		newCapability := interpreter.NewUnmeteredIDCapabilityValue(
			capabilityID,
			oldCapability.Address,
			oldCapability.BorrowType,
		)

		if reporter != nil {
			reporter.MigratedPathCapability(address, addressPath)
		}

		return newCapability

	case *interpreter.CompositeValue:
		composite := value

		// Migrate composite's fields

		composite.ForEachField(nil, func(fieldName string, fieldValue interpreter.Value) {
			newFieldValue := m.migratePathCapability(address, fieldValue, reporter)
			if newFieldValue != nil {
				composite.SetMember(
					m.interpreter,
					locationRange,
					fieldName,
					newFieldValue,
				)
			}
		})

		// The composite itself does not have to be replaced

		return nil

	case *interpreter.SomeValue:
		innerValue := value.InnerValue(m.interpreter, locationRange)
		newInnerValue := m.migratePathCapability(address, innerValue, reporter)
		if newInnerValue != nil {
			return interpreter.NewSomeValueNonCopying(m.interpreter, newInnerValue)
		}

		return nil

	case *interpreter.ArrayValue:
		array := value
		var index int

		// Migrate array's elements

		array.Iterate(m.interpreter, func(element interpreter.Value) (resume bool) {
			newElement := m.migratePathCapability(address, element, reporter)
			if newElement != nil {
				array.Set(
					m.interpreter,
					locationRange,
					index,
					newElement,
				)
			}

			index++

			return true
		})

		// The array itself does not have to be replaced

		return nil

	case *interpreter.DictionaryValue:
		dictionary := value

		// Migrate dictionary's values

		dictionary.Iterate(m.interpreter, func(key, value interpreter.Value) (resume bool) {

			// Keys cannot be capabilities at the moment,
			// so this should never occur in stored data

			if _, ok := key.(interpreter.CapabilityValue); ok {
				panic(errors.NewUnreachableError())
			}

			// Migrate the value of the key-value pair

			newValue := m.migratePathCapability(address, value, reporter)

			if newValue != nil {
				dictionary.Insert(
					m.interpreter,
					locationRange,
					key,
					newValue,
				)
			}

			return true
		})

		// The dictionary itself does not have to be replaced

		return nil

	case interpreter.NumberValue,
		*interpreter.StringValue,
		interpreter.CharacterValue,
		interpreter.BoolValue,
		interpreter.TypeValue,
		interpreter.PathValue,
		interpreter.NilValue:

		// Primitive values do not have to be updated,
		// as they do not contain path capabilities.

		return nil
	}

	panic(errors.NewUnexpectedError("unsupported value type: %T", value))
}
