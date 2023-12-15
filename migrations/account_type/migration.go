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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type AccountTypeMigration struct{}

var _ migrations.Migration = AccountTypeMigration{}

func NewAccountTypeMigration() AccountTypeMigration {
	return AccountTypeMigration{}
}

func (AccountTypeMigration) Name() string {
	return "AccountTypeMigration"
}

// Migrate migrates `AuthAccount` and `PublicAccount` types inside `TypeValue`s,
// to the account reference type (&Account).
func (AccountTypeMigration) Migrate(value interpreter.Value) (newValue interpreter.Value) {
	switch value := value.(type) {
	case interpreter.TypeValue:
		convertedType := maybeConvertAccountType(value.Type)
		if convertedType == nil {
			return
		}
		return interpreter.NewTypeValue(nil, convertedType)

	case *interpreter.CapabilityValue:
		convertedBorrowType := maybeConvertAccountType(value.BorrowType)
		if convertedBorrowType == nil {
			return
		}
		return interpreter.NewUnmeteredCapabilityValue(value.ID, value.Address, convertedBorrowType)

	case *interpreter.AccountCapabilityControllerValue:
		convertedBorrowType := maybeConvertAccountType(value.BorrowType)
		if convertedBorrowType == nil {
			return
		}
		borrowType := convertedBorrowType.(*interpreter.ReferenceStaticType)
		return interpreter.NewUnmeteredAccountCapabilityControllerValue(borrowType, value.CapabilityID)

	case *interpreter.StorageCapabilityControllerValue:
		// Note: A storage capability with Account type shouldn't be possible theoretically.
		convertedBorrowType := maybeConvertAccountType(value.BorrowType)
		if convertedBorrowType == nil {
			return
		}
		borrowType := convertedBorrowType.(*interpreter.ReferenceStaticType)
		return interpreter.NewUnmeteredStorageCapabilityControllerValue(borrowType, value.CapabilityID, value.TargetPath)

	default:
		return nil
	}
}

func maybeConvertAccountType(staticType interpreter.StaticType) interpreter.StaticType {
	switch staticType := staticType.(type) {
	case *interpreter.ConstantSizedStaticType:
		convertedType := maybeConvertAccountType(staticType.Type)
		if convertedType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedType, staticType.Size)
		}

	case *interpreter.VariableSizedStaticType:
		convertedType := maybeConvertAccountType(staticType.Type)
		if convertedType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedType)
		}

	case *interpreter.DictionaryStaticType:
		convertedKeyType := maybeConvertAccountType(staticType.KeyType)
		convertedValueType := maybeConvertAccountType(staticType.ValueType)
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
		convertedBorrowType := maybeConvertAccountType(staticType.BorrowType)
		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType)
		}

	case *interpreter.IntersectionStaticType:
		// No need to convert `staticType.Types` as they can only be interfaces.
		convertedLegacyType := maybeConvertAccountType(staticType.LegacyType)
		if convertedLegacyType != nil {
			intersectionType := interpreter.NewIntersectionStaticType(nil, staticType.Types)
			intersectionType.LegacyType = convertedLegacyType
			return intersectionType
		}

	case *interpreter.OptionalStaticType:
		convertedInnerType := maybeConvertAccountType(staticType.Type)
		if convertedInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, convertedInnerType)
		}

	case *interpreter.ReferenceStaticType:
		// TODO: Reference of references must not be allowed?
		convertedReferencedType := maybeConvertAccountType(staticType.ReferencedType)
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
		return maybeConvertAccountType(staticType.PrimitiveStaticType)

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
