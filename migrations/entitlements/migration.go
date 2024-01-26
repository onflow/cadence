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

// Converts its input to an entitled type according to the following rules:
// * `ConvertToEntitledType(&T) ---> auth(Entitlements(T)) &T`
// * `ConvertToEntitledType(Capability<T>) ---> Capability<ConvertToEntitledType(T)>`
// * `ConvertToEntitledType(T?) ---> ConvertToEntitledType(T)?
// * `ConvertToEntitledType(T) ---> T`
// where Entitlements(I) is defined as the result of T.SupportedEntitlements()
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

// Converts the input value into a version compatible with the new entitlements feature,
// with the same members/operations accessible on any references as would have been accessible in the past.
func ConvertValueToEntitlements(
	inter *interpreter.Interpreter,
	v interpreter.Value,
) (
	interpreter.Value,
	error,
) {

	var staticType interpreter.StaticType
	switch referenceValue := v.(type) {

	case *interpreter.EphemeralReferenceValue:
		// during a real migration this case will not be hit, because ephemeral references are not storable,
		// but they are here for easier testing for reference types, we want to use the borrow type,
		// rather than the type of the referenced value
		staticType = interpreter.NewReferenceStaticType(
			inter,
			referenceValue.Authorization,
			interpreter.ConvertSemaToStaticType(inter, referenceValue.BorrowedType),
		)

	case interpreter.LinkValue: //nolint:staticcheck
		// Link values are not supposed to reach here.
		// But it could, if the type used in the link is not migrated,
		// then the link values would be left un-migrated.
		// These need to be skipped specifically, otherwise `v.StaticType(inter)` will panic.
		return nil, nil

	default:
		staticType = v.StaticType(inter)
	}

	if staticType.IsDeprecated() {
		return nil, fmt.Errorf("cannot migrate deprecated type: %s", staticType)
	}

	// if the static type contains a legacy restricted type, convert it to a new type according to some rules:
	// &T{I} -> auth(SupportedEntitlements(I)) &T
	// Capability<&T{I}> -> Capability<auth(SupportedEntitlements(I)) &T>
	var convertLegacyStaticType func(interpreter.StaticType)
	convertLegacyStaticType = func(staticType interpreter.StaticType) {
		switch t := staticType.(type) {
		case *interpreter.ReferenceStaticType:
			switch referencedType := t.ReferencedType.(type) {
			case *interpreter.IntersectionStaticType:
				if referencedType.LegacyType != nil {
					t.ReferencedType = referencedType.LegacyType
					intersectionSemaType := inter.MustConvertStaticToSemaType(referencedType).(*sema.IntersectionType)
					auth := sema.UnauthorizedAccess
					supportedEntitlements := intersectionSemaType.SupportedEntitlements()
					if supportedEntitlements.Len() > 0 {
						auth = sema.EntitlementSetAccess{
							SetKind:      sema.Conjunction,
							Entitlements: supportedEntitlements,
						}
					}
					t.Authorization = interpreter.ConvertSemaAccessToStaticAuthorization(inter, auth)
				}
			}
		case *interpreter.CapabilityStaticType:
			convertLegacyStaticType(t.BorrowType)
		case *interpreter.VariableSizedStaticType:
			convertLegacyStaticType(t.Type)
		case *interpreter.ConstantSizedStaticType:
			convertLegacyStaticType(t.Type)
		case *interpreter.DictionaryStaticType:
			convertLegacyStaticType(t.KeyType)
			convertLegacyStaticType(t.ValueType)
		case *interpreter.OptionalStaticType:
			convertLegacyStaticType(t.Type)
		}
	}

	convertLegacyStaticType(staticType)

	switch v := v.(type) {
	case *interpreter.EphemeralReferenceValue:
		// during a real migration this case will not be hit,
		// but it is here for easier testing

		semaType := inter.MustConvertStaticToSemaType(staticType)
		entitledType, converted := ConvertToEntitledType(semaType)
		if !converted {
			return nil, nil
		}

		entitledReferenceType := entitledType.(*sema.ReferenceType)
		staticAuthorization := interpreter.ConvertSemaAccessToStaticAuthorization(
			inter,
			entitledReferenceType.Authorization,
		)
		convertedValue, err := ConvertValueToEntitlements(inter, v.Value)
		if err != nil {
			return nil, err
		}

		// if the underlying value did not change, we still want to use the old value in the newly created reference
		if convertedValue == nil {
			convertedValue = v.Value
		}
		return interpreter.NewEphemeralReferenceValue(
			inter,
			staticAuthorization,
			convertedValue,
			entitledReferenceType.Type,
			interpreter.EmptyLocationRange,
		), nil

	case *interpreter.ArrayValue:
		semaType := inter.MustConvertStaticToSemaType(staticType)
		entitledType, converted := ConvertToEntitledType(semaType)
		if !converted {
			return nil, nil
		}

		entitledArrayType := entitledType.(sema.ArrayType)
		arrayStaticType := interpreter.ConvertSemaArrayTypeToStaticArrayType(inter, entitledArrayType)

		iterator := v.Iterator(inter, interpreter.EmptyLocationRange)

		return interpreter.NewArrayValueWithIterator(
			inter,
			arrayStaticType,
			v.GetOwner(),
			uint64(v.Count()),
			func() interpreter.Value {
				return iterator.Next(inter, interpreter.EmptyLocationRange)
			},
		), nil

	case *interpreter.DictionaryValue:
		semaType := inter.MustConvertStaticToSemaType(staticType)
		entitledType, converted := ConvertToEntitledType(semaType)
		if !converted {
			return nil, nil
		}

		entitledDictionaryType := entitledType.(*sema.DictionaryType)
		dictionaryStaticType := interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(
			inter,
			entitledDictionaryType,
		)

		var values []interpreter.Value

		v.Iterate(
			inter,
			interpreter.EmptyLocationRange,
			func(key, value interpreter.Value) (resume bool) {
				values = append(values, key)
				values = append(values, value)
				return true
			},
		)

		return interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			dictionaryStaticType,
			values...,
		), nil

	case *interpreter.CapabilityValue:
		semaType := inter.MustConvertStaticToSemaType(staticType)
		entitledType, converted := ConvertToEntitledType(semaType)
		if !converted {
			return nil, nil
		}

		entitledCapabilityValue := entitledType.(*sema.CapabilityType)
		capabilityStaticType := interpreter.ConvertSemaToStaticType(inter, entitledCapabilityValue.BorrowType)
		return interpreter.NewCapabilityValue(
			inter,
			v.ID,
			v.Address,
			capabilityStaticType,
		), nil

	case interpreter.TypeValue:
		if v.Type == nil {
			return nil, nil
		}

		convertedType, converted := ConvertToEntitledType(
			inter.MustConvertStaticToSemaType(v.Type),
		)

		if !converted {
			return nil, nil
		}

		entitledStaticType := interpreter.ConvertSemaToStaticType(
			inter,
			convertedType,
		)
		return interpreter.NewTypeValue(inter, entitledStaticType), nil

	case *interpreter.AccountCapabilityControllerValue:
		convertedType, converted := ConvertToEntitledType(
			inter.MustConvertStaticToSemaType(v.BorrowType),
		)

		if !converted {
			return nil, nil
		}

		entitledStaticType := interpreter.ConvertSemaToStaticType(
			inter,
			convertedType,
		)
		entitledBorrowType := entitledStaticType.(*interpreter.ReferenceStaticType)
		return interpreter.NewAccountCapabilityControllerValue(
			inter,
			entitledBorrowType,
			v.CapabilityID,
		), nil

	case *interpreter.StorageCapabilityControllerValue:
		convertedType, converted := ConvertToEntitledType(
			inter.MustConvertStaticToSemaType(v.BorrowType),
		)

		if !converted {
			return nil, nil
		}

		entitledStaticType := interpreter.ConvertSemaToStaticType(
			inter,
			convertedType,
		)
		entitledBorrowType := entitledStaticType.(*interpreter.ReferenceStaticType)
		return interpreter.NewStorageCapabilityControllerValue(
			inter,
			entitledBorrowType,
			v.CapabilityID,
			v.TargetPath,
		), nil
	}

	return nil, nil
}

func (mig EntitlementsMigration) Migrate(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (
	interpreter.Value,
	error,
) {
	return ConvertValueToEntitlements(mig.Interpreter, value)
}
