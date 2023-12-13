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
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type EntitlementsMigration struct {
	Interpreter *interpreter.Interpreter
}

var _ migrations.Migration = EntitlementsMigration{}

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
func ConvertToEntitledType(t sema.Type) sema.Type {
	switch t := t.(type) {
	case *sema.ReferenceType:
		switch t.Authorization {
		case sema.UnauthorizedAccess:
			innerType := ConvertToEntitledType(t.Type)
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
			return sema.NewReferenceType(
				nil,
				auth,
				innerType,
			)
		// type is already entitled
		default:
			return t
		}
	case *sema.OptionalType:
		return sema.NewOptionalType(nil, ConvertToEntitledType(t.Type))
	case *sema.CapabilityType:
		return sema.NewCapabilityType(nil, ConvertToEntitledType(t.BorrowType))
	case *sema.VariableSizedType:
		return sema.NewVariableSizedType(nil, ConvertToEntitledType(t.Type))
	case *sema.ConstantSizedType:
		return sema.NewConstantSizedType(nil, ConvertToEntitledType(t.Type), t.Size)
	case *sema.DictionaryType:
		return sema.NewDictionaryType(nil, ConvertToEntitledType(t.KeyType), ConvertToEntitledType(t.ValueType))
	default:
		return t
	}
}

// Converts the input value into a version compatible with the new entitlements feature,
// with the same members/operations accessible on any references as would have been accessible in the past.
func ConvertValueToEntitlements(
	inter *interpreter.Interpreter,
	v interpreter.Value,
) interpreter.Value {

	var staticType interpreter.StaticType
	// for reference types, we want to use the borrow type, rather than the type of the referenced value
	switch referenceValue := v.(type) {
	case *interpreter.EphemeralReferenceValue:
		staticType = interpreter.NewReferenceStaticType(
			inter,
			referenceValue.Authorization,
			interpreter.ConvertSemaToStaticType(inter, referenceValue.BorrowedType),
		)
	case *interpreter.StorageReferenceValue:
		staticType = interpreter.NewReferenceStaticType(
			inter,
			referenceValue.Authorization,
			interpreter.ConvertSemaToStaticType(inter, referenceValue.BorrowedType),
		)
	default:
		staticType = v.StaticType(inter)
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
	semaType := inter.MustConvertStaticToSemaType(staticType)
	entitledType := ConvertToEntitledType(semaType)

	switch v := v.(type) {
	case *interpreter.EphemeralReferenceValue:
		entitledReferenceType := entitledType.(*sema.ReferenceType)
		staticAuthorization := interpreter.ConvertSemaAccessToStaticAuthorization(inter, entitledReferenceType.Authorization)
		convertedValue := ConvertValueToEntitlements(inter, v.Value)
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
		)

	case *interpreter.StorageReferenceValue:
		// a stored value will in itself be migrated at another point, so no need to do anything here other than change the type
		entitledReferenceType := entitledType.(*sema.ReferenceType)
		staticAuthorization := interpreter.ConvertSemaAccessToStaticAuthorization(inter, entitledReferenceType.Authorization)
		return interpreter.NewStorageReferenceValue(
			inter,
			staticAuthorization,
			v.TargetStorageAddress,
			v.TargetPath,
			entitledReferenceType.Type,
		)

	case *interpreter.ArrayValue:
		entitledArrayType := entitledType.(sema.ArrayType)
		arrayStaticType := interpreter.ConvertSemaArrayTypeToStaticArrayType(inter, entitledArrayType)

		iterator := v.Iterator(inter)

		newArray := interpreter.NewArrayValueWithIterator(inter, arrayStaticType, v.GetOwner(), uint64(v.Count()), func() interpreter.Value {
			return iterator.Next(inter)
		})
		return newArray

	case *interpreter.DictionaryValue:
		entitledDictionaryType := entitledType.(*sema.DictionaryType)
		dictionaryStaticType := interpreter.ConvertSemaDictionaryTypeToStaticDictionaryType(inter, entitledDictionaryType)

		var values []interpreter.Value

		v.Iterate(inter, func(key, value interpreter.Value) (resume bool) {
			values = append(values, key)
			values = append(values, value)
			return true
		})

		newDict := interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			dictionaryStaticType,
			values...,
		)
		return newDict

	case *interpreter.CapabilityValue:
		// capabilities should just have their borrow type updated, as the pointed-to value will also be visited
		// by the migration on its own
		entitledCapabilityValue := entitledType.(*sema.CapabilityType)
		capabilityStaticType := interpreter.ConvertSemaToStaticType(inter, entitledCapabilityValue.BorrowType)
		return interpreter.NewCapabilityValue(inter, v.ID, v.Address, capabilityStaticType)

	case *interpreter.TypeValue:
		if v.Type == nil {
			return v
		}
		// convert the static type of the value
		entitledStaticType := interpreter.ConvertSemaToStaticType(
			inter,
			ConvertToEntitledType(
				inter.MustConvertStaticToSemaType(v.Type),
			),
		)
		return interpreter.NewTypeValue(inter, entitledStaticType)
	}

	return nil
}

func (mig EntitlementsMigration) Migrate(value interpreter.Value) (newValue interpreter.Value) {
	return ConvertValueToEntitlements(mig.Interpreter, value)
}
