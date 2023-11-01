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

package account_type

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type AccountTypeMigration struct {
	storage     *runtime.Storage
	interpreter *interpreter.Interpreter
}

func NewAccountTypeMigration(runtime runtime.Runtime, context runtime.Context) (*AccountTypeMigration, error) {
	storage, inter, err := runtime.Storage(context)
	if err != nil {
		return nil, err
	}

	return &AccountTypeMigration{
		storage:     storage,
		interpreter: inter,
	}, nil
}

func (m *AccountTypeMigration) Migrate(
	addressIterator migrations.AddressIterator,
	reporter migrations.Reporter,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migrateTypeValuesInAccount(
			address,
			reporter,
		)
	}
}

// migrateTypeValuesInAccount migrates `AuthAccount` and `PublicAccount` types in a given account
// to the account reference type (&Account).
func (m *AccountTypeMigration) migrateTypeValuesInAccount(
	address common.Address,
	reporter migrations.Reporter,
) {

	accountStorage := migrations.NewAccountStorage(m.storage, address)

	accountStorage.ForEachValue(
		m.interpreter,
		common.AllPathDomains,
		m.migrateValue,
		reporter,
	)
}

func (m *AccountTypeMigration) migrateValue(value interpreter.Value) interpreter.Value {
	typeValue, ok := value.(interpreter.TypeValue)
	if !ok {
		// TODO: support migration for type-values nested inside other values.
		return nil
	}

	innerType := typeValue.Type

	convertedType := m.maybeConvertAccountType(innerType)
	if convertedType == nil {
		return nil
	}

	return interpreter.NewTypeValue(nil, convertedType)
}

func (m *AccountTypeMigration) maybeConvertAccountType(staticType interpreter.StaticType) interpreter.StaticType {
	switch staticType := staticType.(type) {
	case *interpreter.ConstantSizedStaticType:
		convertedType := m.maybeConvertAccountType(staticType.Type)
		if convertedType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedType, staticType.Size)
		}

	case *interpreter.VariableSizedStaticType:
		convertedType := m.maybeConvertAccountType(staticType.Type)
		if convertedType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedType)
		}

	case *interpreter.DictionaryStaticType:
		convertedKeyType := m.maybeConvertAccountType(staticType.KeyType)
		convertedValueType := m.maybeConvertAccountType(staticType.ValueType)
		if convertedKeyType != nil && convertedValueType != nil {
			return interpreter.NewDictionaryStaticType(nil, staticType.KeyType, staticType.ValueType)
		}
		if convertedKeyType != nil {
			return interpreter.NewDictionaryStaticType(nil, convertedKeyType, staticType.ValueType)
		}
		if convertedValueType != nil {
			return interpreter.NewDictionaryStaticType(nil, staticType.KeyType, convertedValueType)
		}

	case *interpreter.CapabilityStaticType:
		convertedBorrowType := m.maybeConvertAccountType(staticType.BorrowType)
		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType)
		}

	case *interpreter.IntersectionStaticType:
		// Nothing to do. Inner types can only be interfaces.

	case *interpreter.OptionalStaticType:
		convertedInnerType := m.maybeConvertAccountType(staticType.Type)
		if convertedInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, convertedInnerType)
		}

	case *interpreter.ReferenceStaticType:
		// TODO: Reference of references must not be allowed?
		convertedReferencedType := m.maybeConvertAccountType(staticType.ReferencedType)
		if convertedReferencedType != nil {
			switch convertedReferencedType {

			// If the converted type is already an account reference, then return as-is.
			// i.e: Do not create reference to a reference.
			case authAccountReferenceType,
				unauthorizedAccountReferenceType:
				return convertedReferencedType

			default:
				return interpreter.NewReferenceStaticType(nil, staticType.Authorization, convertedReferencedType)
			}
		}

	case interpreter.FunctionStaticType:
		// Non-storable

	case *interpreter.CompositeStaticType,
		*interpreter.InterfaceStaticType:
		// Nothing to do

	default:
		// Is it safe to do so?
		switch staticType {
		case interpreter.PrimitiveStaticTypePublicAccount:
			return unauthorizedAccountReferenceType
		case interpreter.PrimitiveStaticTypeAuthAccount:
			return authAccountReferenceType

		case interpreter.PrimitiveStaticTypeAuthAccountCapabilities,
			interpreter.PrimitiveStaticTypePublicAccountCapabilities:
			return interpreter.PrimitiveStaticTypeAccount_Capabilities

		case interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities:
			return interpreter.PrimitiveStaticTypeAccount_AccountCapabilities

		case interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities:
			return interpreter.PrimitiveStaticTypeAccount_StorageCapabilities

		case interpreter.PrimitiveStaticTypeAuthAccountContracts,
			interpreter.PrimitiveStaticTypePublicAccountContracts:
			return interpreter.PrimitiveStaticTypeAccount_Contracts

		case interpreter.PrimitiveStaticTypeAuthAccountKeys,
			interpreter.PrimitiveStaticTypePublicAccountKeys:
			return interpreter.PrimitiveStaticTypeAccount_Keys

		case interpreter.PrimitiveStaticTypeAuthAccountInbox:
			return interpreter.PrimitiveStaticTypeAccount_Inbox

		case interpreter.PrimitiveStaticTypeAccountKey:
			return interpreter.AccountKeyStaticType
		}
	}

	return nil
}

var authAccountEntitlements = []common.TypeID{
	sema.StorageType.ID(),
	sema.ContractsType.ID(),
	sema.KeysType.ID(),
	sema.InboxType.ID(),
	sema.CapabilitiesType.ID(),
}

var authAccountReferenceType = func() *interpreter.ReferenceStaticType {
	auth := interpreter.NewEntitlementSetAuthorization(
		nil,
		func() []common.TypeID {
			return authAccountEntitlements
		},
		len(authAccountEntitlements),
		sema.Conjunction,
	)
	return interpreter.NewReferenceStaticType(
		nil,
		auth,
		interpreter.PrimitiveStaticTypeAccount,
	)
}()

var unauthorizedAccountReferenceType = interpreter.NewReferenceStaticType(
	nil,
	interpreter.UnauthorizedAccess,
	interpreter.PrimitiveStaticTypeAccount,
)
