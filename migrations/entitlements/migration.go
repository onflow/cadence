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
	Interpreter *interpreter.Interpreter
}

var _ migrations.ValueMigration = EntitlementsMigration{}

func NewEntitlementsMigration(inter *interpreter.Interpreter) EntitlementsMigration {
	return EntitlementsMigration{Interpreter: inter}
}

func (EntitlementsMigration) Name() string {
	return "EntitlementsMigration"
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
	inter *interpreter.Interpreter,
	staticType interpreter.StaticType,
) (
	interpreter.StaticType,
	error,
) {
	if staticType == nil {
		return nil, nil
	}

	if staticType.IsDeprecated() {
		return nil, fmt.Errorf("cannot migrate deprecated type: %s", staticType)
	}

	switch t := staticType.(type) {
	case *interpreter.ReferenceStaticType:

		referencedType := t.ReferencedType

		convertedReferencedType, err := ConvertToEntitledType(inter, referencedType)
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
			return interpreter.NewReferenceStaticType(nil, auth, referencedType), nil
		}

	case *interpreter.CapabilityStaticType:
		convertedBorrowType, err := ConvertToEntitledType(inter, t.BorrowType)
		if err != nil {
			return nil, err
		}

		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType), nil
		}

	case *interpreter.VariableSizedStaticType:
		elementType := t.Type

		convertedElementType, err := ConvertToEntitledType(inter, elementType)
		if err != nil {
			return nil, err
		}

		if convertedElementType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedElementType), nil
		}

	case *interpreter.ConstantSizedStaticType:
		elementType := t.Type

		convertedElementType, err := ConvertToEntitledType(inter, elementType)
		if err != nil {
			return nil, err
		}

		if convertedElementType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedElementType, t.Size), nil
		}

	case *interpreter.DictionaryStaticType:
		keyType := t.KeyType

		convertedKeyType, err := ConvertToEntitledType(inter, keyType)
		if err != nil {
			return nil, err
		}

		valueType := t.ValueType

		convertedValueType, err := ConvertToEntitledType(inter, valueType)
		if err != nil {
			return nil, err
		}

		if convertedKeyType != nil {
			if convertedValueType != nil {
				return interpreter.NewDictionaryStaticType(nil, convertedKeyType, convertedValueType), nil
			} else {
				return interpreter.NewDictionaryStaticType(nil, convertedKeyType, valueType), nil
			}
		} else if convertedValueType != nil {
			return interpreter.NewDictionaryStaticType(nil, keyType, convertedValueType), nil
		}

	case *interpreter.OptionalStaticType:
		innerType := t.Type

		convertedInnerType, err := ConvertToEntitledType(inter, innerType)
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
func ConvertValueToEntitlements(
	inter *interpreter.Interpreter,
	v interpreter.Value,
) (
	interpreter.Value,
	error,
) {

	switch v := v.(type) {

	case *interpreter.ArrayValue:
		elementType := v.Type

		entitledElementType, err := ConvertToEntitledType(inter, elementType)
		if err != nil {
			return nil, err
		}

		if entitledElementType == nil {
			return nil, nil
		}

		return v.NewWithType(
			inter,
			interpreter.EmptyLocationRange,
			entitledElementType.(interpreter.ArrayStaticType),
		), nil

	case *interpreter.DictionaryValue:
		elementType := v.Type

		entitledElementType, err := ConvertToEntitledType(inter, elementType)
		if err != nil {
			return nil, err
		}

		if entitledElementType == nil {
			return nil, nil
		}

		return v.NewWithType(
			inter,
			interpreter.EmptyLocationRange,
			entitledElementType.(*interpreter.DictionaryStaticType),
		), nil

	case *interpreter.IDCapabilityValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := ConvertToEntitledType(inter, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(inter, borrowType)
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

		entitledType, err := ConvertToEntitledType(inter, ty)
		if err != nil {
			return nil, err
		}

		if entitledType != nil {
			return interpreter.NewTypeValue(inter, entitledType), nil
		}

	case *interpreter.AccountCapabilityControllerValue:
		borrowType := v.BorrowType

		entitledBorrowType, err := ConvertToEntitledType(inter, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(inter, borrowType)
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

		entitledBorrowType, err := ConvertToEntitledType(inter, borrowType)
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
	return ConvertValueToEntitlements(mig.Interpreter, value)
}

func (mig EntitlementsMigration) CanSkip(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	interpreter *interpreter.Interpreter,
) bool {
	return statictypes.CanSkipStaticTypeMigration(value.StaticType(interpreter))
}
