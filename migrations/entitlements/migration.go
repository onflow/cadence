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

package entitlements

import (
	"fmt"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/migrations/statictypes"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type EntitlementsMigration struct {
	Interpreter       *interpreter.Interpreter
	migratedTypeCache migrations.StaticTypeCache
}

var _ migrations.ValueMigration = EntitlementsMigration{}

func NewEntitlementsMigration(inter *interpreter.Interpreter) EntitlementsMigration {
	staticTypeCache := migrations.NewDefaultStaticTypeCache()
	return NewEntitlementsMigrationWithCache(inter, staticTypeCache)
}

func NewEntitlementsMigrationWithCache(
	inter *interpreter.Interpreter,
	migratedTypeCache migrations.StaticTypeCache,
) EntitlementsMigration {
	return EntitlementsMigration{
		Interpreter:       inter,
		migratedTypeCache: migratedTypeCache,
	}
}

func (EntitlementsMigration) Name() string {
	return "EntitlementsMigration"
}

func (EntitlementsMigration) Domains() map[string]struct{} {
	return nil
}

// ConvertToEntitledType converts the given type to an entitled type according to the following rules:
//   - ConvertToEntitledType(&T)            --> auth(Entitlements(T)) &T
//   - ConvertToEntitledType(Capability<T>) --> Capability<ConvertToEntitledType(T)>
//   - ConvertToEntitledType(T?)            --> ConvertToEntitledType(T)?
//   - ConvertToEntitledType([T])           --> [ConvertToEntitledType(T)]
//   - ConvertToEntitledType([T; N])        --> [ConvertToEntitledType(T); N]
//   - ConvertToEntitledType({K: V})        --> {ConvertToEntitledType(K): ConvertToEntitledType(V)}
//   - ConvertToEntitledType(T)             --> T
//
// where `Entitlements(I)` is defined as the result of `T.SupportedEntitlements()`
//
// TODO: functions?
func (m EntitlementsMigration) ConvertToEntitledType(
	staticType interpreter.StaticType,
) (
	resultType interpreter.StaticType,
	err error,
) {
	if staticType == nil {
		return nil, nil
	}

	if staticType.IsDeprecated() {
		return nil, fmt.Errorf("cannot migrate deprecated type: %s", staticType)
	}

	inter := m.Interpreter
	migratedTypeCache := m.migratedTypeCache

	staticTypeID := staticType.ID()

	if migratedType, exists := migratedTypeCache.Get(staticTypeID); exists {
		return migratedType, nil
	}

	defer func() {
		if err != nil {
			migratedTypeCache.Set(staticTypeID, resultType)
		}
	}()

	switch t := staticType.(type) {
	case *interpreter.ReferenceStaticType:

		referencedType := t.ReferencedType

		convertedReferencedType, err := m.ConvertToEntitledType(referencedType)
		if err != nil {
			return nil, err
		}

		var returnNew bool

		if convertedReferencedType != nil {
			referencedType = convertedReferencedType
			returnNew = true
		}

		// Determine the authorization (entitlements) from the referenced type,
		// based on the supported entitlements of the referenced type

		auth := t.Authorization

		// If the referenced type is an empty intersection type,
		// do not add an authorization

		intersectionType, isIntersection := referencedType.(*interpreter.IntersectionStaticType)
		isEmptyIntersection := isIntersection && len(intersectionType.Types) == 0

		if !isEmptyIntersection {
			referencedSemaType := inter.MustConvertStaticToSemaType(referencedType)

			if entitlementSupportingType, ok := referencedSemaType.(sema.EntitlementSupportingType); ok {

				switch entitlementSupportingType {

				// Do NOT add authorization for sema types
				// that were converted from deprecated primitive static types
				case sema.AccountType,
					sema.Account_ContractsType,
					sema.Account_KeysType,
					sema.Account_InboxType,
					sema.Account_StorageCapabilitiesType,
					sema.Account_AccountCapabilitiesType,
					sema.Account_CapabilitiesType,
					sema.AccountKeyType:

					// NO-OP
					break

				default:
					supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
					newAccess := supportedEntitlements.Access()
					auth = interpreter.ConvertSemaAccessToStaticAuthorization(inter, newAccess)
					returnNew = true
				}
			}
		}

		if isIntersection {
			// Rewrite the intersection type to remove the potential legacy restricted type
			referencedType = statictypes.RewriteLegacyIntersectionType(intersectionType)
			returnNew = true
		}

		if returnNew {
			return interpreter.NewReferenceStaticType(nil, auth, referencedType), nil
		}

	case *interpreter.CapabilityStaticType:
		convertedBorrowType, err := m.ConvertToEntitledType(t.BorrowType)
		if err != nil {
			return nil, err
		}

		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType), nil
		}

	case *interpreter.VariableSizedStaticType:
		elementType := t.Type

		convertedElementType, err := m.ConvertToEntitledType(elementType)
		if err != nil {
			return nil, err
		}

		if convertedElementType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedElementType), nil
		}

	case *interpreter.ConstantSizedStaticType:
		elementType := t.Type

		convertedElementType, err := m.ConvertToEntitledType(elementType)
		if err != nil {
			return nil, err
		}

		if convertedElementType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedElementType, t.Size), nil
		}

	case *interpreter.DictionaryStaticType:
		keyType := t.KeyType

		convertedKeyType, err := m.ConvertToEntitledType(keyType)
		if err != nil {
			return nil, err
		}

		valueType := t.ValueType

		convertedValueType, err := m.ConvertToEntitledType(valueType)
		if err != nil {
			return nil, err
		}

		if convertedKeyType != nil {
			if convertedValueType != nil {
				return interpreter.NewDictionaryStaticType(
					nil,
					convertedKeyType,
					convertedValueType,
				), nil
			} else {
				return interpreter.NewDictionaryStaticType(
					nil,
					convertedKeyType,
					valueType,
				), nil
			}
		} else if convertedValueType != nil {
			return interpreter.NewDictionaryStaticType(
				nil,
				keyType,
				convertedValueType,
			), nil
		}

	case *interpreter.OptionalStaticType:
		innerType := t.Type

		convertedInnerType, err := m.ConvertToEntitledType(innerType)
		if err != nil {
			return nil, err
		}

		if convertedInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, convertedInnerType), nil
		}
	}

	return nil, nil
}

// ConvertValueToEntitlements converts the input value into a version compatible with the new entitlements feature,
// with the same members/operations accessible on any references as would have been accessible in the past.
func (m EntitlementsMigration) ConvertValueToEntitlements(v interpreter.Value) (interpreter.Value, error) {
	inter := m.Interpreter

	switch v := v.(type) {

	case *interpreter.ArrayValue:
		elementType := v.Type

		entitledElementType, err := m.ConvertToEntitledType(elementType)
		if err != nil {
			return nil, err
		}

		if entitledElementType == nil {
			return nil, nil
		}

		v.SetType(
			entitledElementType.(interpreter.ArrayStaticType),
		)

	case *interpreter.DictionaryValue:
		elementType := v.Type

		entitledElementType, err := m.ConvertToEntitledType(elementType)
		if err != nil {
			return nil, err
		}

		if entitledElementType == nil {
			return nil, nil
		}

		v.SetType(
			entitledElementType.(*interpreter.DictionaryStaticType),
		)

	case *interpreter.IDCapabilityValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := m.ConvertToEntitledType(borrowType)
		if err != nil {
			return nil, err
		}

		if entitledBorrowType != nil {
			return interpreter.NewCapabilityValue(
				inter,
				v.ID,
				v.Address,
				entitledBorrowType,
			), nil
		}

	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		borrowType := v.BorrowType

		entitledBorrowType, err := m.ConvertToEntitledType(borrowType)
		if err != nil {
			return nil, err
		}

		if entitledBorrowType != nil {
			return &interpreter.PathCapabilityValue{ //nolint:staticcheck
				Path:       v.Path,
				Address:    v.Address,
				BorrowType: entitledBorrowType,
			}, nil
		}

	case interpreter.TypeValue:
		ty := v.Type

		entitledType, err := m.ConvertToEntitledType(ty)
		if err != nil {
			return nil, err
		}

		if entitledType != nil {
			return interpreter.NewTypeValue(inter, entitledType), nil
		}

	case *interpreter.AccountCapabilityControllerValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := m.ConvertToEntitledType(borrowType)
		if err != nil {
			return nil, err
		}

		if entitledBorrowType != nil {
			return interpreter.NewAccountCapabilityControllerValue(
				inter,
				entitledBorrowType.(*interpreter.ReferenceStaticType),
				v.CapabilityID,
			), nil
		}

	case *interpreter.StorageCapabilityControllerValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := m.ConvertToEntitledType(borrowType)
		if err != nil {
			return nil, err
		}

		if entitledBorrowType != nil {
			return interpreter.NewStorageCapabilityControllerValue(
				inter,
				entitledBorrowType.(*interpreter.ReferenceStaticType),
				v.CapabilityID,
				v.TargetPath,
			), nil
		}

	case interpreter.PathLinkValue: //nolint:staticcheck
		borrowType := v.Type

		entitledBorrowType, err := m.ConvertToEntitledType(borrowType)
		if err != nil {
			return nil, err
		}

		if entitledBorrowType != nil {
			return interpreter.PathLinkValue{ //nolint:staticcheck
				TargetPath: v.TargetPath,
				Type:       entitledBorrowType,
			}, nil
		}
	}

	return nil, nil
}

func (m EntitlementsMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (
	interpreter.Value,
	error,
) {
	return m.ConvertValueToEntitlements(value)
}

func (m EntitlementsMigration) CanSkip(valueType interpreter.StaticType) bool {
	return statictypes.CanSkipStaticTypeMigration(valueType)
}
