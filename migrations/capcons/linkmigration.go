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
	goerrors "errors"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type LinkMigrationReporter interface {
	MigratedLink(
		accountAddressPath interpreter.AddressPath,
		capabilityID interpreter.UInt64Value,
	)
	CyclicLink(err CyclicLinkError)
	MissingTarget(accountAddressPath interpreter.AddressPath)
}

// LinkValueMigration migrates all links to capability controllers.
type LinkValueMigration struct {
	CapabilityMapping *PathCapabilityMapping
	IssueHandler      stdlib.CapabilityControllerIssueHandler
	Handler           stdlib.CapabilityControllerHandler
	Reporter          LinkMigrationReporter
}

var _ migrations.ValueMigration = &LinkValueMigration{}

func (*LinkValueMigration) Name() string {
	return "LinkValueMigration"
}

func (m *LinkValueMigration) CanSkip(valueType interpreter.StaticType) bool {
	// Link values have a capability static type
	return CanSkipCapabilityValueMigration(valueType)
}

var linkValueMigrationDomains = map[string]struct{}{
	common.PathDomainPublic.Identifier():  {},
	common.PathDomainPrivate.Identifier(): {},
}

func (m *LinkValueMigration) Domains() map[string]struct{} {
	return linkValueMigrationDomains
}

func (m *LinkValueMigration) Migrate(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	value interpreter.Value,
	inter *interpreter.Interpreter,
	_ migrations.ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	pathValue, ok := storageKeyToPathValue(storageKey, storageMapKey)
	if !ok {
		return nil, nil
	}

	pathDomain := pathValue.Domain
	switch pathDomain {
	case common.PathDomainPublic, common.PathDomainPrivate:
		// migrate public and private domain
	default:
		// ignore other domains (e.g. storage)
		return nil, nil
	}

	accountAddress := storageKey.Address

	addressPath := interpreter.AddressPath{
		Address: accountAddress,
		Path:    pathValue,
	}

	reporter := m.Reporter
	issueHandler := m.IssueHandler

	locationRange := interpreter.EmptyLocationRange

	var borrowStaticType *interpreter.ReferenceStaticType

	switch readValue := value.(type) {
	case *interpreter.IDCapabilityValue:
		// Already migrated
		return nil, nil

	case interpreter.PathLinkValue: //nolint:staticcheck
		var ok bool
		borrowType := readValue.Type
		borrowStaticType, ok = borrowType.(*interpreter.ReferenceStaticType)
		if !ok {
			panic(errors.NewUnexpectedError("unexpected non-reference borrow type: %T", borrowType))
		}

	case interpreter.AccountLinkValue: //nolint:staticcheck
		borrowStaticType = interpreter.NewReferenceStaticType(
			nil,
			interpreter.FullyEntitledAccountAccess,
			interpreter.PrimitiveStaticTypeAccount,
		)

	default:
		panic(errors.NewUnexpectedError("unexpected value type: %T", value))
	}

	// Get target

	target, _, err := m.getPathCapabilityFinalTarget(
		inter,
		accountAddress,
		pathValue,
		// Use top-most type to follow link all the way to final target
		&sema.ReferenceType{
			Authorization: sema.UnauthorizedAccess,
			Type:          sema.AnyType,
		},
	)
	if err != nil {
		var cyclicLinkErr CyclicLinkError
		if goerrors.As(err, &cyclicLinkErr) {
			if reporter != nil {
				reporter.CyclicLink(cyclicLinkErr)
			}

			// TODO: really leave as-is? or still convert?
			return nil, nil
		}

		return nil, err
	}

	// Issue appropriate capability controller

	var capabilityID interpreter.UInt64Value

	switch target := target.(type) {
	case nil:
		if reporter != nil {
			reporter.MissingTarget(addressPath)
		}

		// TODO: really leave as-is? or still convert?
		return nil, nil

	case pathCapabilityTarget:

		targetPath := interpreter.PathValue(target)

		capabilityID = stdlib.IssueStorageCapabilityController(
			inter,
			locationRange,
			issueHandler,
			accountAddress,
			borrowStaticType,
			targetPath,
		)

	case accountCapabilityTarget:
		capabilityID = stdlib.IssueAccountCapabilityController(
			inter,
			locationRange,
			issueHandler,
			accountAddress,
			borrowStaticType,
		)

	default:
		panic(errors.NewUnexpectedError("unexpected target type: %T", target))
	}

	// Record new capability ID in source path mapping.
	// The mapping is used later for migrating path capabilities to ID capabilities,
	// see CapabilityMigration.
	m.CapabilityMapping.Record(addressPath, capabilityID, borrowStaticType)

	if reporter != nil {
		reporter.MigratedLink(addressPath, capabilityID)
	}

	addressValue := interpreter.AddressValue(addressPath.Address)

	return interpreter.NewCapabilityValue(
		inter,
		capabilityID,
		addressValue,
		borrowStaticType,
	), nil
}

func storageKeyToPathValue(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
) (
	interpreter.PathValue,
	bool,
) {
	domain := common.PathDomainFromIdentifier(storageKey.Key)
	if domain == common.PathDomainUnknown {
		return interpreter.PathValue{}, false
	}
	stringStorageMapKey, ok := storageMapKey.(interpreter.StringStorageMapKey)
	if !ok {
		return interpreter.PathValue{}, false
	}
	identifier := string(stringStorageMapKey)
	return interpreter.NewUnmeteredPathValue(domain, identifier), true
}

var unauthorizedAccountReferenceStaticType = interpreter.NewReferenceStaticType(
	nil,
	interpreter.UnauthorizedAccess,
	interpreter.PrimitiveStaticTypeAccount,
)

func (m *LinkValueMigration) getPathCapabilityFinalTarget(
	inter *interpreter.Interpreter,
	accountAddress common.Address,
	pathValue interpreter.PathValue,
	wantedBorrowType *sema.ReferenceType,
) (
	target capabilityTarget,
	authorization interpreter.Authorization,
	err error,
) {
	handler := m.Handler

	locationRange := interpreter.EmptyLocationRange

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
					inter,
					wantedBorrowType.Authorization,
				),
				nil

		case common.PathDomainPublic,
			common.PathDomainPrivate:

			value := inter.ReadStored(accountAddress, domain, storageMapKey)
			if value == nil {
				return nil, interpreter.UnauthorizedAccess, nil
			}

			switch value := value.(type) {
			case interpreter.PathLinkValue: //nolint:staticcheck
				allowedType := inter.MustConvertStaticToSemaType(value.Type)

				if !sema.IsSubType(allowedType, wantedBorrowType) {
					return nil, interpreter.UnauthorizedAccess, nil
				}

				targetPath := value.TargetPath
				paths = append(paths, targetPath)
				pathValue = targetPath

			case interpreter.AccountLinkValue: //nolint:staticcheck
				allowedType := unauthorizedAccountReferenceStaticType

				if !inter.IsSubTypeOfSemaType(allowedType, wantedBorrowType) {
					return nil, interpreter.UnauthorizedAccess, nil
				}

				return accountCapabilityTarget(accountAddress),
					interpreter.UnauthorizedAccess,
					nil

			case *interpreter.IDCapabilityValue:

				// Follow ID capability values which are published in the public or private domain.
				// This is needed for two reasons:
				// 1. Support for migrating path capabilities to ID capabilities was already enabled on Testnet
				// 2. During the migration of a whole link chain,
				//    the order of the migration of the individual links is undefined,
				//    so it's possible that a capability value is encountered when determining the final target,
				//    when a part of the full link chain was already previously migrated.

				convertedBorrowType := inter.MustConvertStaticToSemaType(value.BorrowType)
				capabilityBorrowType, ok := convertedBorrowType.(*sema.ReferenceType)
				if !ok {
					panic(errors.NewUnexpectedError(
						"unexpected non-reference borrow type: %T",
						convertedBorrowType,
					))
				}

				// Do not borrow final target (i.e. do not require target to exist),
				// just get target address/path
				reference := stdlib.GetCheckedCapabilityControllerReference(
					inter,
					locationRange,
					value.Address(),
					value.ID,
					wantedBorrowType,
					capabilityBorrowType,
					handler,
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
				panic(errors.NewUnexpectedError("unexpected value type: %T", value))
			}
		}
	}
}
