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
// * `ConvertToEntitledType(&T) --> auth(Entitlements(T)) &T`
// * `ConvertToEntitledType(Capability<T>)` --> `Capability<ConvertToEntitledType(T)>`
// * `ConvertToEntitledType(T?)             --> `ConvertToEntitledType(T)?`
// * `ConvertToEntitledType([T])`           --> `[ConvertToEntitledType(T)]`
// * `ConvertToEntitledType([T; N])`        --> `[ConvertToEntitledType(T); N]`
// * `ConvertToEntitledType({K: V})`        --> `{ConvertToEntitledType(K): ConvertToEntitledType(V)}`
// * `ConvertToEntitledType(T)`             --> `T`
// where `Entitlements(I)` is defined as the result of `T.SupportedEntitlements()`
// TODO: functions?
func ConvertToEntitledType(t sema.Type) (sema.Type, bool) {

	switch t := t.(type) {
	case *sema.ReferenceType:

		// Do NOT add authorization for sema types
		// that were converted from deprecated primitive static types
		switch t.Type {

		case sema.AccountType,
			sema.Account_ContractsType,
			sema.Account_KeysType,
			sema.Account_InboxType,
			sema.Account_StorageCapabilitiesType,
			sema.Account_AccountCapabilitiesType,
			sema.Account_CapabilitiesType,
			sema.AccountKeyType:

			return t, false
		}

		switch t.Authorization {
		case sema.UnauthorizedAccess:
			innerType, convertedInner := ConvertToEntitledType(t.Type)
			auth := sema.UnauthorizedAccess
			if entitlementSupportingType, ok := innerType.(sema.EntitlementSupportingType); ok {
				supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
				if supportedEntitlements.Len() > 0 {
					auth = sema.EntitlementSetAccess{
						SetKind:      sema.Conjunction,
						Entitlements: supportedEntitlements,
					}
				}
			}
			if auth.Equal(sema.UnauthorizedAccess) && !convertedInner {
				return t, false
			}
			return sema.NewReferenceType(
				nil,
				auth,
				innerType,
			), true
		// type is already entitled
		default:
			return t, false
		}
	case *sema.OptionalType:
		ty, converted := ConvertToEntitledType(t.Type)
		if !converted {
			return t, false
		}
		return sema.NewOptionalType(nil, ty), true
	case *sema.CapabilityType:
		ty, converted := ConvertToEntitledType(t.BorrowType)
		if !converted {
			return t, false
		}
		return sema.NewCapabilityType(nil, ty), true
	case *sema.VariableSizedType:
		ty, converted := ConvertToEntitledType(t.Type)
		if !converted {
			return t, false
		}
		return sema.NewVariableSizedType(nil, ty), true
	case *sema.ConstantSizedType:
		ty, converted := ConvertToEntitledType(t.Type)
		if !converted {
			return t, false
		}
		return sema.NewConstantSizedType(nil, ty, t.Size), true
	case *sema.DictionaryType:
		keyTy, convertedKey := ConvertToEntitledType(t.KeyType)
		valueTy, convertedValue := ConvertToEntitledType(t.ValueType)
		if !convertedKey && !convertedValue {
			return t, false
		}
		return sema.NewDictionaryType(nil, keyTy, valueTy), true
	default:
		return t, false
	}
}

func convertStaticType(inter *interpreter.Interpreter, staticType interpreter.StaticType) error {
	if staticType.IsDeprecated() {
		return fmt.Errorf("cannot migrate deprecated type: %s", staticType)
	}

	switch t := staticType.(type) {
	case *interpreter.ReferenceStaticType:

		// If the static type contains a legacy restricted type,
		// add the supported entitlements of the restricted type to the authorization of the reference type
		// (`{I}` --> `auth(SupportedEntitlements(I))`)

		if intersectionType, ok := t.ReferencedType.(*interpreter.IntersectionStaticType); ok {
			// Add entitlements to the authorization of the reference type,
			// based on the supported entitlements of the intersection type's interface types

			if len(intersectionType.Types) > 0 {
				intersectionSemaType := inter.MustConvertStaticToSemaType(intersectionType).(*sema.IntersectionType)
				auth := sema.UnauthorizedAccess
				supportedEntitlements := intersectionSemaType.SupportedEntitlements()
				if supportedEntitlements.Len() > 0 {
					auth = sema.EntitlementSetAccess{
						SetKind:      sema.Conjunction,
						Entitlements: supportedEntitlements,
					}
				}
				t.Authorization = interpreter.ConvertSemaAccessToStaticAuthorization(inter, auth)
			} else {
				// TODO: prevent type from getting entitled
			}

			// Rewrite the intersection type to remove the legacy restricted type
			t.ReferencedType = statictypes.RewriteLegacyIntersectionType(intersectionType)

		} else {
			// Convert the referenced type
			err := convertStaticType(inter, t.ReferencedType)
			if err != nil {
				return err
			}
		}

	case *interpreter.CapabilityStaticType:
		err := convertStaticType(inter, t.BorrowType)
		if err != nil {
			return err
		}

	case *interpreter.VariableSizedStaticType:
		err := convertStaticType(inter, t.Type)
		if err != nil {
			return err
		}

	case *interpreter.ConstantSizedStaticType:
		err := convertStaticType(inter, t.Type)
		if err != nil {
			return err
		}

	case *interpreter.DictionaryStaticType:
		err := convertStaticType(inter, t.KeyType)
		if err != nil {
			return err
		}

		err = convertStaticType(inter, t.ValueType)
		if err != nil {
			return err
		}

	case *interpreter.OptionalStaticType:
		err := convertStaticType(inter, t.Type)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertToEntitledStaticType(
	inter *interpreter.Interpreter,
	staticType interpreter.StaticType,
) (
	interpreter.StaticType,
	error,
) {
	if staticType == nil {
		return nil, nil
	}

	err := convertStaticType(inter, staticType)
	if err != nil {
		return nil, err
	}

	semaType := inter.MustConvertStaticToSemaType(staticType)
	entitledType, converted := ConvertToEntitledType(semaType)
	if !converted {
		return nil, nil
	}

	return interpreter.ConvertSemaToStaticType(inter, entitledType), nil
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
		entitledStaticType, err := convertToEntitledStaticType(inter, v.Type)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		iterator := v.Iterator(inter, interpreter.EmptyLocationRange)

		return interpreter.NewArrayValueWithIterator(
			inter,
			entitledStaticType.(interpreter.ArrayStaticType),
			v.GetOwner(),
			uint64(v.Count()),
			func() interpreter.Value {
				return iterator.Next(inter, interpreter.EmptyLocationRange)
			},
		), nil

	case *interpreter.DictionaryValue:
		entitledStaticType, err := convertToEntitledStaticType(inter, v.Type)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		var values []interpreter.Value

		v.Iterate(inter, func(key, value interpreter.Value) (resume bool) {
			values = append(values, key)
			values = append(values, value)
			return true
		})

		return interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.EmptyLocationRange,
			entitledStaticType.(*interpreter.DictionaryStaticType),
			v.GetOwner(),
			values...,
		), nil

	case *interpreter.IDCapabilityValue:
		entitledStaticType, err := convertToEntitledStaticType(inter, v.BorrowType)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return interpreter.NewCapabilityValue(
			inter,
			v.ID,
			v.Address,
			entitledStaticType,
		), nil

	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		entitledStaticType, err := convertToEntitledStaticType(inter, v.BorrowType)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return &interpreter.PathCapabilityValue{ //nolint:staticcheck
			Path:       v.Path,
			Address:    v.Address,
			BorrowType: entitledStaticType,
		}, nil

	case interpreter.TypeValue:
		entitledStaticType, err := convertToEntitledStaticType(inter, v.Type)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return interpreter.NewTypeValue(inter, entitledStaticType), nil

	case *interpreter.AccountCapabilityControllerValue:
		entitledStaticType, err := convertToEntitledStaticType(inter, v.BorrowType)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return interpreter.NewAccountCapabilityControllerValue(
			inter,
			entitledStaticType.(*interpreter.ReferenceStaticType),
			v.CapabilityID,
		), nil

	case *interpreter.StorageCapabilityControllerValue:
		entitledStaticType, err := convertToEntitledStaticType(inter, v.BorrowType)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return interpreter.NewStorageCapabilityControllerValue(
			inter,
			entitledStaticType.(*interpreter.ReferenceStaticType),
			v.CapabilityID,
			v.TargetPath,
		), nil

	case interpreter.PathLinkValue: //nolint:staticcheck
		entitledStaticType, err := convertToEntitledStaticType(inter, v.Type)
		if err != nil {
			return nil, err
		}

		if entitledStaticType == nil {
			return nil, nil
		}

		return interpreter.PathLinkValue{ //nolint:staticcheck
			TargetPath: v.TargetPath,
			Type:       entitledStaticType,
		}, nil
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
