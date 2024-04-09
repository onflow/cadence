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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type EntitlementsMigration struct {
	Interpreter       *interpreter.Interpreter
	migratedTypeCache map[common.TypeID]interpreter.StaticType
}

var _ migrations.ValueMigration = EntitlementsMigration{}

func NewEntitlementsMigration(inter *interpreter.Interpreter) EntitlementsMigration {
	return EntitlementsMigration{
		Interpreter:       inter,
		migratedTypeCache: map[common.TypeID]interpreter.StaticType{},
	}
}

func NewEntitlementsMigrationWithCache(
	inter *interpreter.Interpreter,
	migratedTypeCache map[common.TypeID]interpreter.StaticType,
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
func ConvertToEntitledType(
	mig EntitlementsMigration,
	staticType interpreter.StaticType,
) (
	resultType interpreter.StaticType,
	conversionErr error,
) {
	if staticType == nil {
		return nil, nil
	}

	if staticType.IsDeprecated() {
		return nil, fmt.Errorf("cannot migrate deprecated type: %s", staticType)
	}

	inter := mig.Interpreter
	migratedTypeCache := mig.migratedTypeCache

	staticTypeID := staticType.ID()

	if migratedType, exists := migratedTypeCache[staticTypeID]; exists {
		return migratedType, nil
	}

	defer func() {
		if resultType != nil && conversionErr == nil {
			migratedTypeCache[staticTypeID] = resultType
		}
	}()

	switch t := staticType.(type) {
	case *interpreter.ReferenceStaticType:

		referencedType := t.ReferencedType

		convertedReferencedType, err := ConvertToEntitledType(mig, referencedType)
		if err != nil {
			conversionErr = err
			return
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
					newAuth := sema.UnauthorizedAccess
					supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
					if supportedEntitlements.Len() > 0 {
						newAuth = sema.EntitlementSetAccess{
							SetKind:      sema.Conjunction,
							Entitlements: supportedEntitlements,
						}
					}
					auth = interpreter.ConvertSemaAccessToStaticAuthorization(inter, newAuth)
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
			resultType = interpreter.NewReferenceStaticType(nil, auth, referencedType)
			return
		}

	case *interpreter.CapabilityStaticType:
		convertedBorrowType, err := ConvertToEntitledType(mig, t.BorrowType)
		if err != nil {
			conversionErr = err
			return
		}

		if convertedBorrowType != nil {
			resultType = interpreter.NewCapabilityStaticType(nil, convertedBorrowType)
			return
		}

	case *interpreter.VariableSizedStaticType:
		elementType := t.Type

		convertedElementType, err := ConvertToEntitledType(mig, elementType)
		if err != nil {
			conversionErr = err
			return
		}

		if convertedElementType != nil {
			resultType = interpreter.NewVariableSizedStaticType(nil, convertedElementType)
			return
		}

	case *interpreter.ConstantSizedStaticType:
		elementType := t.Type

		convertedElementType, err := ConvertToEntitledType(mig, elementType)
		if err != nil {
			conversionErr = err
			return
		}

		if convertedElementType != nil {
			resultType = interpreter.NewConstantSizedStaticType(nil, convertedElementType, t.Size)
			return
		}

	case *interpreter.DictionaryStaticType:
		keyType := t.KeyType

		convertedKeyType, err := ConvertToEntitledType(mig, keyType)
		if err != nil {
			conversionErr = err
			return
		}

		valueType := t.ValueType

		convertedValueType, err := ConvertToEntitledType(mig, valueType)
		if err != nil {
			conversionErr = err
			return
		}

		if convertedKeyType != nil {
			if convertedValueType != nil {
				resultType = interpreter.NewDictionaryStaticType(nil, convertedKeyType, convertedValueType)
				return
			} else {
				resultType = interpreter.NewDictionaryStaticType(nil, convertedKeyType, valueType)
				return
			}
		} else if convertedValueType != nil {
			resultType = interpreter.NewDictionaryStaticType(nil, keyType, convertedValueType)
			return
		}

	case *interpreter.OptionalStaticType:
		innerType := t.Type

		convertedInnerType, err := ConvertToEntitledType(mig, innerType)
		if err != nil {
			conversionErr = err
			return
		}

		if convertedInnerType != nil {
			resultType = interpreter.NewOptionalStaticType(nil, convertedInnerType)
			return
		}
	}

	return
}

// ConvertValueToEntitlements converts the input value into a version compatible with the new entitlements feature,
// with the same members/operations accessible on any references as would have been accessible in the past.
func ConvertValueToEntitlements(
	mig EntitlementsMigration,
	v interpreter.Value,
) (
	interpreter.Value,
	error,
) {

	inter := mig.Interpreter

	switch v := v.(type) {

	case *interpreter.ArrayValue:
		elementType := v.Type

		entitledElementType, err := ConvertToEntitledType(mig, elementType)
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

		entitledElementType, err := ConvertToEntitledType(mig, elementType)
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

		entitledBorrowType, err := ConvertToEntitledType(mig, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(mig, borrowType)
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

		entitledType, err := ConvertToEntitledType(mig, ty)
		if err != nil {
			return nil, err
		}

		if entitledType != nil {
			return interpreter.NewTypeValue(inter, entitledType), nil
		}

	case *interpreter.AccountCapabilityControllerValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := ConvertToEntitledType(mig, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(mig, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(mig, borrowType)
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

func (mig EntitlementsMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (
	interpreter.Value,
	error,
) {
	return ConvertValueToEntitlements(mig, value)
}

func (mig EntitlementsMigration) CanSkip(valueType interpreter.StaticType) bool {
	return statictypes.CanSkipStaticTypeMigration(valueType)
}
