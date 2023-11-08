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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type AccountTypeMigration struct {
	storage     *runtime.Storage
	interpreter *interpreter.Interpreter
}

func NewAccountTypeMigration(
	interpreter *interpreter.Interpreter,
	storage *runtime.Storage,
) *AccountTypeMigration {
	return &AccountTypeMigration{
		storage:     storage,
		interpreter: interpreter,
	}
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

	err := m.storage.Commit(m.interpreter, false)
	if err != nil {
		panic(err)
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

var locationRange = interpreter.EmptyLocationRange

func (m *AccountTypeMigration) migrateValue(value interpreter.Value) (newValue interpreter.Value, updatedInPlace bool) {
	switch value := value.(type) {
	case interpreter.TypeValue:
		convertedType := m.maybeConvertAccountType(value.Type)
		if convertedType == nil {
			return
		}

		return interpreter.NewTypeValue(nil, convertedType), true

	case *interpreter.SomeValue:
		innerValue := value.InnerValue(m.interpreter, locationRange)
		newInnerValue, _ := m.migrateValue(innerValue)
		if newInnerValue != nil {
			return interpreter.NewSomeValueNonCopying(m.interpreter, newInnerValue), true
		}

		return

	case *interpreter.ArrayValue:
		array := value

		// Migrate array elements
		count := array.Count()
		for index := 0; index < count; index++ {
			element := array.Get(m.interpreter, locationRange, index)
			newElement, elementUpdated := m.migrateValue(element)
			if newElement != nil {
				array.Set(
					m.interpreter,
					locationRange,
					index,
					newElement,
				)
			}

			updatedInPlace = updatedInPlace || elementUpdated
		}

		// The array itself doesn't need to be replaced.
		return

	case *interpreter.CompositeValue:
		composite := value

		// Read the field names first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var fieldNames []string
		composite.ForEachField(nil, func(fieldName string, fieldValue interpreter.Value) (resume bool) {
			fieldNames = append(fieldNames, fieldName)
			return true
		})

		for _, fieldName := range fieldNames {
			existingValue := composite.GetField(m.interpreter, interpreter.EmptyLocationRange, fieldName)

			migratedValue, valueUpdated := m.migrateValue(existingValue)
			if migratedValue == nil {
				continue
			}

			composite.SetMember(m.interpreter, locationRange, fieldName, migratedValue)

			updatedInPlace = updatedInPlace || valueUpdated
		}

		// The composite itself does not have to be replaced
		return

	case *interpreter.DictionaryValue:
		dictionary := value

		// Read the keys first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var existingKeys []interpreter.Value
		dictionary.Iterate(m.interpreter, func(key, _ interpreter.Value) (resume bool) {
			existingKeys = append(existingKeys, key)
			return true
		})

		for _, existingKey := range existingKeys {
			existingValue, exist := dictionary.Get(nil, interpreter.EmptyLocationRange, existingKey)
			if !exist {
				panic(errors.NewUnreachableError())
			}

			newKey, keyUpdated := m.migrateValue(existingKey)
			newValue, valueUpdated := m.migrateValue(existingValue)
			if newKey == nil && newValue == nil {
				continue
			}

			// We only reach here at least one of key or value has been migrated.
			var keyToSet, valueToSet interpreter.Value

			if newKey == nil {
				keyToSet = existingKey
			} else {
				// Key was migrated.
				// Remove the old value at the old key.
				// This old value will be inserted again with the new key, unless the value is also migrated.
				_ = dictionary.RemoveKey(m.interpreter, locationRange, existingKey)
				keyToSet = newKey
			}

			if newValue == nil {
				valueToSet = existingValue
			} else {
				// Value was migrated
				valueToSet = newValue
			}

			// Always wrap with an optional, when inserting to the dictionary.
			valueToSet = interpreter.NewUnmeteredSomeValueNonCopying(valueToSet)

			dictionary.SetKey(m.interpreter, locationRange, keyToSet, valueToSet)

			updatedInPlace = updatedInPlace || keyUpdated || valueUpdated
		}

		// The dictionary itself does not have to be replaced
		return
	default:
		return
	}
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
			return interpreter.NewDictionaryStaticType(nil, convertedKeyType, convertedValueType)
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

	case dummyStaticType:
		// This is for testing the migration.
		// i.e: wrapper was only to make it possible to use as a dictionary-key.
		// Ignore the wrapper, and continue with the inner type.
		return m.maybeConvertAccountType(staticType.PrimitiveStaticType)

	default:
		// Is it safe to do so?
		switch staticType {
		case interpreter.PrimitiveStaticTypePublicAccount: //nolint:staticcheck
			return unauthorizedAccountReferenceType
		case interpreter.PrimitiveStaticTypeAuthAccount: //nolint:staticcheck
			return authAccountReferenceType

		case interpreter.PrimitiveStaticTypeAuthAccountCapabilities, //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountCapabilities: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_Capabilities

		case interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_AccountCapabilities

		case interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_StorageCapabilities

		case interpreter.PrimitiveStaticTypeAuthAccountContracts, //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountContracts: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_Contracts

		case interpreter.PrimitiveStaticTypeAuthAccountKeys, //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountKeys: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_Keys

		case interpreter.PrimitiveStaticTypeAuthAccountInbox: //nolint:staticcheck
			return interpreter.PrimitiveStaticTypeAccount_Inbox

		case interpreter.PrimitiveStaticTypeAccountKey: //nolint:staticcheck
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
