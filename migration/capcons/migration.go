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
	goerrors "errors"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
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

type MigrationReporter interface {
	LinkMigrationReporter
	PathCapabilityMigrationReporter
}

type LinkMigrationReporter interface {
	MigratedLink(
		accountAddressPath interpreter.AddressPath,
		capabilityID interpreter.UInt64Value,
	)
	CyclicLink(err CyclicLinkError)
	MissingTarget(
		accountAddress interpreter.AddressValue,
		path interpreter.PathValue,
	)
}

type PathCapabilityMigrationReporter interface {
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

type Migration struct {
	storage            *runtime.Storage
	interpreter        *interpreter.Interpreter
	capabilityIDs      map[interpreter.AddressPath]interpreter.UInt64Value
	addressIterator    AddressIterator
	accountIDGenerator stdlib.AccountIDGenerator
}

func NewMigration(
	storage *runtime.Storage,
	interpreter *interpreter.Interpreter,
	addressIterator AddressIterator,
	accountIDGenerator stdlib.AccountIDGenerator,
) (*Migration, error) {

	return &Migration{
		storage:            storage,
		interpreter:        interpreter,
		addressIterator:    addressIterator,
		accountIDGenerator: accountIDGenerator,
	}, nil
}

// Migrate migrates the links to capability controllers,
// and all path capabilities and account capabilities to ID capabilities,
// in all accounts of the given iterator.
func (m *Migration) Migrate(
	reporter MigrationReporter,
) error {
	m.capabilityIDs = make(map[interpreter.AddressPath]interpreter.UInt64Value)
	defer func() {
		m.capabilityIDs = nil
	}()
	m.migrateLinks(reporter)

	m.addressIterator.Reset()
	m.migratePathCapabilities(reporter)

	return m.storage.Commit(m.interpreter, false)
}

// migrateLinks migrates the links to capability controllers
// in all accounts of the given iterator.
// It constructs a source path to capability ID mapping,
// which is later needed to path capabilities to ID capabilities.
func (m *Migration) migrateLinks(
	reporter LinkMigrationReporter,
) {
	for {
		address := m.addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migrateLinksInAccount(
			address,
			reporter,
		)
	}
}

// migrateLinksInAccount migrates the links in the given account to capability controllers
// It records an entry in the source path to capability ID mapping,
// which is later needed to migrate path capabilities to ID capabilities.
func (m *Migration) migrateLinksInAccount(
	accountAddress common.Address,
	reporter LinkMigrationReporter,
) {

	migrateDomain := func(domain common.PathDomain) {
		m.migrateAccountLinksInAccountDomain(
			accountAddress,
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
func (m *Migration) migrateAccountLinksInAccountDomain(
	accountAddress common.Address,
	domain common.PathDomain,
	reporter LinkMigrationReporter,
) {
	accountAddressValue := interpreter.AddressValue(accountAddress)

	storageMap := m.storage.GetStorageMap(accountAddress, domain.Identifier(), false)
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
				accountAddressValue,
				pathValue,
				reporter,
			)
		}
	}
}

// migrateAccountLinksInAccountDomain migrates the links in the given account's storage domain
// to capability controllers.
// It constructs a source path to ID mapping,
// which is later needed to migrate path capabilities to ID capabilities.
func (m *Migration) migrateLink(
	accountAddressValue interpreter.AddressValue,
	pathValue interpreter.PathValue,
	reporter LinkMigrationReporter,
) {
	capabilityID := m.migrateLinkToCapabilityController(accountAddressValue, pathValue, reporter)
	if capabilityID == 0 {
		return
	}

	// Record new capability ID in source path mapping.
	// The mapping is used later for migrating path capabilities to ID capabilities.

	accountAddress := accountAddressValue.ToAddress()
	accountAddressPath := interpreter.AddressPath{
		Address: accountAddress,
		Path:    pathValue,
	}

	m.capabilityIDs[accountAddressPath] = capabilityID

	if reporter != nil {
		reporter.MigratedLink(accountAddressPath, capabilityID)
	}
}

// migratePathCapabilities migrates the path capabilities to ID capabilities
// in all accounts of the given iterator.
// It uses the source path to capability ID mapping which was constructed in migrateLinks.
func (m *Migration) migratePathCapabilities(
	reporter PathCapabilityMigrationReporter,
) {
	for {
		address := m.addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migratePathCapabilitiesInAccount(address, reporter)
	}
}

var pathDomainStorage = common.PathDomainStorage.Identifier()

func (m *Migration) migratePathCapabilitiesInAccount(
	accountAddress common.Address,
	reporter PathCapabilityMigrationReporter,
) {

	storageMap := m.storage.GetStorageMap(accountAddress, pathDomainStorage, false)
	if storageMap == nil {
		return
	}

	iterator := storageMap.Iterator(m.interpreter)

	count := storageMap.Count()
	if count > 0 {
		for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {

			newValue := m.migratePathCapability(
				accountAddress,
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

var fullyEntitledAccountReferenceStaticType = interpreter.ConvertSemaReferenceTypeToStaticReferenceType(
	nil,
	sema.FullyEntitledAccountReferenceType,
)

// migratePathCapability migrates a path capability to an ID capability in the given value.
// If a value is returned, the value must be updated with the replacement in the parent.
// If nil is returned, the value was not updated and no operation has to be performed.
func (m *Migration) migratePathCapability(
	accountAddress common.Address,
	value interpreter.Value,
	reporter PathCapabilityMigrationReporter,
) interpreter.Value {
	locationRange := interpreter.EmptyLocationRange

	switch value := value.(type) {
	case *interpreter.PathCapabilityValue: //nolint:staticcheck

		// Migrate the path capability to an ID capability

		oldCapability := value

		addressPath := oldCapability.AddressPath()
		capabilityID, ok := m.capabilityIDs[addressPath]
		if !ok {
			if reporter != nil {
				reporter.MissingCapabilityID(accountAddress, addressPath)
			}
			break
		}

		newBorrowType, ok := oldCapability.BorrowType.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		if newBorrowType.ReferencedType == interpreter.PrimitiveStaticTypeAuthAccount {
			newBorrowType = fullyEntitledAccountReferenceStaticType
		}

		newCapability := interpreter.NewUnmeteredCapabilityValue(
			capabilityID,
			oldCapability.Address,
			newBorrowType,
		)

		if reporter != nil {
			reporter.MigratedPathCapability(
				accountAddress,
				addressPath,
				newBorrowType,
			)
		}

		return newCapability

	case *interpreter.CompositeValue:
		composite := value

		// Migrate composite's fields

		composite.ForEachField(nil, func(fieldName string, fieldValue interpreter.Value) (resume bool) {
			newFieldValue := m.migratePathCapability(accountAddress, fieldValue, reporter)
			if newFieldValue != nil {
				composite.SetMember(
					m.interpreter,
					locationRange,
					fieldName,
					newFieldValue,
				)
			}

			// continue iteration
			return true
		})

		// The composite itself does not have to be replaced

		return nil

	case *interpreter.SomeValue:
		innerValue := value.InnerValue(m.interpreter, locationRange)
		newInnerValue := m.migratePathCapability(accountAddress, innerValue, reporter)
		if newInnerValue != nil {
			return interpreter.NewSomeValueNonCopying(m.interpreter, newInnerValue)
		}

		return nil

	case *interpreter.ArrayValue:
		array := value
		var index int

		// Migrate array's elements

		array.Iterate(m.interpreter, func(element interpreter.Value) (resume bool) {
			newElement := m.migratePathCapability(accountAddress, element, reporter)
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

			switch key.(type) {
			case *interpreter.CapabilityValue,
				*interpreter.PathCapabilityValue: //nolint:staticcheck

				panic(errors.NewUnreachableError())
			}

			// Migrate the value of the key-value pair

			newValue := m.migratePathCapability(accountAddress, value, reporter)

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

	case *interpreter.CapabilityValue:
		// Already migrated
		return nil

	default:
		panic(errors.NewUnexpectedError("unsupported value type: %T", value))
	}

	return nil
}

func (m *Migration) migrateLinkToCapabilityController(
	accountAddressValue interpreter.AddressValue,
	pathValue interpreter.PathValue,
	reporter LinkMigrationReporter,
) interpreter.UInt64Value {

	locationRange := interpreter.EmptyLocationRange

	address := accountAddressValue.ToAddress()

	domain := pathValue.Domain.Identifier()
	identifier := pathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	readValue := m.interpreter.ReadStored(address, domain, storageMapKey)
	if readValue == nil {
		return 0
	}

	var borrowStaticType *interpreter.ReferenceStaticType

	switch readValue := readValue.(type) {
	case *interpreter.CapabilityValue:
		// Already migrated
		return 0

	case interpreter.PathLinkValue: //nolint:staticcheck
		var ok bool
		borrowStaticType, ok = readValue.Type.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}

	case interpreter.AccountLinkValue: //nolint:staticcheck
		borrowStaticType = interpreter.NewReferenceStaticType(
			nil,
			interpreter.FullyEntitledAccountAccess,
			interpreter.PrimitiveStaticTypeAccount,
		)

	default:
		panic(errors.NewUnreachableError())
	}

	borrowType, ok := m.interpreter.MustConvertStaticToSemaType(borrowStaticType).(*sema.ReferenceType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Get target

	target, _, err := m.getPathCapabilityFinalTarget(
		address,
		pathValue,
		// TODO:
		// Use top-most type to follow link all the way to final target
		&sema.ReferenceType{
			Authorization: sema.UnauthorizedAccess,
			Type:          sema.AnyType,
		},
	)
	if err != nil {
		var cyclicLinkErr CyclicLinkError
		if goerrors.As(err, &cyclicLinkErr) {
			reporter.CyclicLink(cyclicLinkErr)
			return 0
		}
		panic(err)
	}

	// Issue appropriate capability controller

	var capabilityID interpreter.UInt64Value

	switch target := target.(type) {
	case nil:
		reporter.MissingTarget(accountAddressValue, pathValue)
		return 0

	case pathCapabilityTarget:

		targetPath := interpreter.PathValue(target)

		capabilityID, _ = stdlib.IssueStorageCapabilityController(
			m.interpreter,
			locationRange,
			m.accountIDGenerator,
			address,
			borrowType,
			targetPath,
		)

	case accountCapabilityTarget:
		capabilityID, _ = stdlib.IssueAccountCapabilityController(
			m.interpreter,
			locationRange,
			m.accountIDGenerator,
			address,
			borrowType,
		)

	default:
		panic(errors.NewUnreachableError())
	}

	// Publish: overwrite link value with capability

	capabilityValue := interpreter.NewCapabilityValue(
		m.interpreter,
		capabilityID,
		accountAddressValue,
		borrowStaticType,
	)

	capabilityValue, ok = capabilityValue.Transfer(
		m.interpreter,
		locationRange,
		atree.Address(address),
		true,
		nil,
		nil,
	).(*interpreter.CapabilityValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	m.interpreter.WriteStored(
		address,
		domain,
		storageMapKey,
		capabilityValue,
	)

	return capabilityID
}

var authAccountReferenceStaticType = interpreter.NewReferenceStaticType(
	nil,
	interpreter.UnauthorizedAccess,
	interpreter.PrimitiveStaticTypeAuthAccount, //nolint:staticcheck
)

func (m *Migration) getPathCapabilityFinalTarget(
	accountAddress common.Address,
	pathValue interpreter.PathValue,
	wantedBorrowType *sema.ReferenceType,
) (
	target capabilityTarget,
	authorization interpreter.Authorization,
	err error,
) {

	seenPaths := map[interpreter.PathValue]struct{}{}
	paths := []interpreter.PathValue{pathValue}

	for {
		// Detect cyclic links

		if _, ok := seenPaths[pathValue]; ok {
			return nil,
				interpreter.UnauthorizedAccess,
				CyclicLinkError{
					Address: accountAddress,
					Paths:   paths,
				}
		} else {
			seenPaths[pathValue] = struct{}{}
		}

		domain := pathValue.Domain.Identifier()
		identifier := pathValue.Identifier

		storageMapKey := interpreter.StringStorageMapKey(identifier)

		switch pathValue.Domain {
		case common.PathDomainStorage:

			return pathCapabilityTarget(pathValue),
				interpreter.ConvertSemaAccessToStaticAuthorization(
					m.interpreter,
					wantedBorrowType.Authorization,
				),
				nil

		case common.PathDomainPublic,
			common.PathDomainPrivate:

			value := m.interpreter.ReadStored(accountAddress, domain, storageMapKey)
			if value == nil {
				return nil, interpreter.UnauthorizedAccess, nil
			}

			switch value := value.(type) {
			case interpreter.PathLinkValue: //nolint:staticcheck
				allowedType := m.interpreter.MustConvertStaticToSemaType(value.Type)

				if !sema.IsSubType(allowedType, wantedBorrowType) {
					return nil, interpreter.UnauthorizedAccess, nil
				}

				targetPath := value.TargetPath
				paths = append(paths, targetPath)
				pathValue = targetPath

			case interpreter.AccountLinkValue: //nolint:staticcheck
				if !m.interpreter.IsSubTypeOfSemaType(
					authAccountReferenceStaticType,
					wantedBorrowType,
				) {
					return nil, interpreter.UnauthorizedAccess, nil
				}

				return accountCapabilityTarget(accountAddress),
					interpreter.UnauthorizedAccess,
					nil

			case *interpreter.CapabilityValue:

				// For backwards-compatibility, follow ID capability values
				// which are published in the public or private domain

				capabilityBorrowType, ok :=
					m.interpreter.MustConvertStaticToSemaType(value.BorrowType).(*sema.ReferenceType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Do not borrow final target (i.e. do not require target to exist),
				// just get target address/path
				reference := stdlib.GetCheckedCapabilityControllerReference(
					m.interpreter,
					value.Address,
					value.ID,
					wantedBorrowType,
					capabilityBorrowType,
				)
				if reference == nil {
					return nil, interpreter.UnauthorizedAccess, nil
				}

				switch reference := reference.(type) {
				case *interpreter.StorageReferenceValue:
					accountAddress = reference.TargetStorageAddress
					targetPath := reference.TargetPath
					paths = append(paths, targetPath)
					pathValue = targetPath

				case *interpreter.EphemeralReferenceValue:
					accountValue := reference.Value.(*interpreter.SimpleCompositeValue)
					address := accountValue.Fields[sema.AccountTypeAddressFieldName].(interpreter.AddressValue)

					return accountCapabilityTarget(address),
						interpreter.UnauthorizedAccess,
						nil

				default:
					return nil, interpreter.UnauthorizedAccess, nil
				}

			default:
				panic(errors.NewUnreachableError())
			}
		}
	}
}
