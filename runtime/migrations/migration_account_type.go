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

package migrations

import (
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type MigrationReporter interface {
	Report(address common.Address, key string, message string)
	ReportErrors(message string)
}

type AccountTypeMigration struct {
	storage       *runtime.Storage
	interpreter   *interpreter.Interpreter
	capabilityIDs map[interpreter.AddressPath]interpreter.UInt64Value
}

func NewCapConsMigration(runtime runtime.Runtime, context runtime.Context) (*AccountTypeMigration, error) {
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
	addressIterator AddressIterator,
	reporter MigrationReporter,
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
	reporter MigrationReporter,
) {

	accountStorage := AccountStorage{
		storage: m.storage,
		address: address,
	}

	accountStorage.ForEachValue(
		m.interpreter,
		common.AllPathDomains,
		m.migrateValue,
		reporter,
	)
}

func (m *AccountTypeMigration) migrateValue(value interpreter.Value) interpreter.Value {
	typeValue, ok := value.(*interpreter.TypeValue)
	if !ok {
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
		convertedTypes := make([]interpreter.StaticType, len(staticType.Types))

		converted := false

		for _, interfaceType := range staticType.Types {
			convertedInterfaceType := m.maybeConvertAccountType(interfaceType)

		}

	case *interpreter.OptionalStaticType:

	case *interpreter.ReferenceStaticType:

	case interpreter.FunctionStaticType:
		// Non-storable

	case *interpreter.CompositeStaticType,
		*interpreter.InterfaceStaticType:
		// Nothing to do

	default:
		// Is it safe to do so?
		switch staticType {
		case interpreter.PrimitiveStaticTypePublicAccount:
			return interpreter.NewReferenceStaticType(
				nil,
				nil,
				interpreter.PrimitiveStaticTypeAccount,
			)
		case interpreter.PrimitiveStaticTypeAuthAccount:
			auth := interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return authAccountEntitlements
				},
				0,
				sema.Conjunction,
			)
			return interpreter.NewReferenceStaticType(
				nil,
				auth,
				interpreter.PrimitiveStaticTypeAccount,
			)

		// TODO: What about these?
		case interpreter.PrimitiveStaticTypeAuthAccountCapabilities:
		case interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities:
		case interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities:
		case interpreter.PrimitiveStaticTypeAuthAccountContracts:
		case interpreter.PrimitiveStaticTypeAuthAccountKeys:
		case interpreter.PrimitiveStaticTypeAuthAccountInbox:

		case interpreter.PrimitiveStaticTypePublicAccountCapabilities:
		case interpreter.PrimitiveStaticTypePublicAccountContracts:
		case interpreter.PrimitiveStaticTypePublicAccountKeys:

		case interpreter.PrimitiveStaticTypeAccountKey:
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
