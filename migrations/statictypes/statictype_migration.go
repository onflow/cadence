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

package statictypes

import (
	"fmt"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type StaticTypeMigration struct {
	compositeTypeConverter CompositeTypeConverterFunc
	interfaceTypeConverter InterfaceTypeConverterFunc
}

type CompositeTypeConverterFunc func(*interpreter.CompositeStaticType) interpreter.StaticType
type InterfaceTypeConverterFunc func(*interpreter.InterfaceStaticType) interpreter.StaticType

var _ migrations.ValueMigration = &StaticTypeMigration{}

func NewStaticTypeMigration() *StaticTypeMigration {
	return &StaticTypeMigration{}
}

func (m *StaticTypeMigration) WithCompositeTypeConverter(converterFunc CompositeTypeConverterFunc) *StaticTypeMigration {
	m.compositeTypeConverter = converterFunc
	return m
}

func (m *StaticTypeMigration) WithInterfaceTypeConverter(converterFunc InterfaceTypeConverterFunc) *StaticTypeMigration {
	m.interfaceTypeConverter = converterFunc
	return m
}

func (*StaticTypeMigration) Name() string {
	return "StaticTypeMigration"
}

// Migrate migrates `AuthAccount` and `PublicAccount` types inside `TypeValue`s,
// to the account reference type (&Account).
func (m *StaticTypeMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (newValue interpreter.Value, err error) {
	switch value := value.(type) {
	case interpreter.TypeValue:
		// Type is optional. nil represents "unknown"/"invalid" type
		ty := value.Type
		if ty == nil {
			return
		}
		convertedType := m.maybeConvertStaticType(ty, nil)
		if convertedType == nil {
			return
		}
		return interpreter.NewTypeValue(nil, convertedType), nil

	case *interpreter.IDCapabilityValue:
		convertedBorrowType := m.maybeConvertStaticType(value.BorrowType, nil)
		if convertedBorrowType == nil {
			return
		}
		return interpreter.NewUnmeteredCapabilityValue(value.ID, value.Address, convertedBorrowType), nil

	case *interpreter.PathCapabilityValue: //nolint:staticcheck
		// Type is optional
		borrowType := value.BorrowType
		if borrowType == nil {
			return
		}
		convertedBorrowType := m.maybeConvertStaticType(borrowType, nil)
		if convertedBorrowType == nil {
			return
		}
		return &interpreter.PathCapabilityValue{ //nolint:staticcheck
			BorrowType: convertedBorrowType,
			Path:       value.Path,
			Address:    value.Address,
		}, nil

	case interpreter.PathLinkValue: //nolint:staticcheck
		convertedBorrowType := m.maybeConvertStaticType(value.Type, nil)
		if convertedBorrowType == nil {
			return
		}
		return interpreter.PathLinkValue{ //nolint:staticcheck
			Type:       convertedBorrowType,
			TargetPath: value.TargetPath,
		}, nil

	case *interpreter.AccountCapabilityControllerValue:
		convertedBorrowType := m.maybeConvertStaticType(value.BorrowType, nil)
		if convertedBorrowType == nil {
			return
		}
		borrowType := convertedBorrowType.(*interpreter.ReferenceStaticType)
		return interpreter.NewUnmeteredAccountCapabilityControllerValue(borrowType, value.CapabilityID), nil

	case *interpreter.StorageCapabilityControllerValue:
		convertedBorrowType := m.maybeConvertStaticType(value.BorrowType, nil)
		if convertedBorrowType == nil {
			return
		}
		borrowType := convertedBorrowType.(*interpreter.ReferenceStaticType)
		return interpreter.NewUnmeteredStorageCapabilityControllerValue(
			borrowType,
			value.CapabilityID,
			value.TargetPath,
		), nil
	}

	return
}

func (m *StaticTypeMigration) maybeConvertStaticType(staticType, parentType interpreter.StaticType) interpreter.StaticType {
	switch staticType := staticType.(type) {
	case *interpreter.ConstantSizedStaticType:
		convertedType := m.maybeConvertStaticType(staticType.Type, staticType)
		if convertedType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedType, staticType.Size)
		}

	case *interpreter.VariableSizedStaticType:
		convertedType := m.maybeConvertStaticType(staticType.Type, staticType)
		if convertedType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedType)
		}

	case *interpreter.DictionaryStaticType:
		convertedKeyType := m.maybeConvertStaticType(staticType.KeyType, staticType)
		convertedValueType := m.maybeConvertStaticType(staticType.ValueType, staticType)
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
		convertedBorrowType := m.maybeConvertStaticType(staticType.BorrowType, staticType)
		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType)
		}

	case *interpreter.IntersectionStaticType:

		// First rewrite, then convert the rewritten type.

		rewrittenType := rewriteLegacyIntersectionType(staticType, parentType)

		// The rewritten type is either:
		// - an intersection type (with or without legacy type)
		// - a legacy type

		if rewrittenIntersectionType, ok := rewrittenType.(*interpreter.IntersectionStaticType); ok {

			// Convert all interface types in the intersection type

			var convertedInterfaceTypes []*interpreter.InterfaceStaticType

			var convertedInterfaceType bool

			for _, interfaceStaticType := range rewrittenIntersectionType.Types {
				convertedType := m.maybeConvertStaticType(interfaceStaticType, rewrittenIntersectionType)

				// lazily allocate the slice
				if convertedInterfaceTypes == nil {
					convertedInterfaceTypes = make([]*interpreter.InterfaceStaticType, 0, len(rewrittenIntersectionType.Types))
				}

				var replacement *interpreter.InterfaceStaticType
				if convertedType != nil {
					var ok bool
					replacement, ok = convertedType.(*interpreter.InterfaceStaticType)
					if !ok {
						panic(fmt.Errorf(
							"invalid non-interface replacement in intersection type %s: %s replaced by %s",
							rewrittenIntersectionType,
							interfaceStaticType,
							convertedType,
						))
					}

					convertedInterfaceType = true
				} else {
					replacement = interfaceStaticType
				}
				convertedInterfaceTypes = append(convertedInterfaceTypes, replacement)
			}

			// Convert the legacy type

			legacyType := rewrittenIntersectionType.LegacyType

			var convertedLegacyType interpreter.StaticType
			if legacyType != nil {
				convertedLegacyType = m.maybeConvertStaticType(legacyType, rewrittenIntersectionType)
				switch convertedLegacyType.(type) {
				case nil,
					*interpreter.CompositeStaticType,
					interpreter.PrimitiveStaticType:
					// valid
					break

				default:
					panic(fmt.Errorf(
						"invalid non-composite/primitive replacement for legacy type in intersection type %s:"+
							" %s replaced by %s",
						rewrittenIntersectionType,
						legacyType,
						convertedLegacyType,
					))
				}
			}

			// Construct the new intersection type, if needed

			// If the interface set has at least two items,
			// then force it to be re-stored/re-encoded,
			// even if the interface types in the set have not changed.
			if len(rewrittenIntersectionType.Types) >= 2 || convertedInterfaceType || convertedLegacyType != nil {

				result := interpreter.NewIntersectionStaticType(nil, convertedInterfaceTypes)

				if convertedLegacyType != nil {
					result.LegacyType = convertedLegacyType
				} else {
					result.LegacyType = legacyType
				}

				return result
			}

		} else {
			convertedLegacyType := m.maybeConvertStaticType(rewrittenType, parentType)
			if convertedLegacyType != nil {
				return convertedLegacyType
			}
		}

		if rewrittenType != staticType {
			return rewrittenType
		}

	case *interpreter.OptionalStaticType:
		convertedInnerType := m.maybeConvertStaticType(staticType.Type, staticType)
		if convertedInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, convertedInnerType)
		}

	case *interpreter.ReferenceStaticType:
		// TODO: Reference of references must not be allowed?
		convertedReferencedType := m.maybeConvertStaticType(staticType.ReferencedType, staticType)
		if convertedReferencedType != nil {
			switch convertedReferencedType {

			// If the converted type is already an account reference, then return as-is.
			// i.e: Do not create reference to a reference.
			case authAccountReferenceType,
				unauthorizedAccountReferenceType:
				return convertedReferencedType

			default:
				return interpreter.NewReferenceStaticType(
					nil,
					staticType.Authorization,
					convertedReferencedType,
				)
			}
		}

	case interpreter.FunctionStaticType:
		// Non-storable

	case *interpreter.CompositeStaticType:
		var convertedType interpreter.StaticType
		compositeTypeConverter := m.compositeTypeConverter
		if compositeTypeConverter != nil {
			convertedType = compositeTypeConverter(staticType)
		}

		// Interface types need to be placed in intersection types.
		// If the composite type was converted to an interface type,
		// and if the parent type is not an intersection type,
		// then the converted interface type must be placed in an intersection type
		if convertedInterfaceType, ok := convertedType.(*interpreter.InterfaceStaticType); ok {
			if _, ok := parentType.(*interpreter.IntersectionStaticType); !ok {
				convertedType = interpreter.NewIntersectionStaticType(
					nil, []*interpreter.InterfaceStaticType{
						convertedInterfaceType,
					},
				)
			}
		}

		return convertedType

	case *interpreter.InterfaceStaticType:
		var convertedType interpreter.StaticType
		interfaceTypeConverter := m.interfaceTypeConverter
		if interfaceTypeConverter != nil {
			convertedType = interfaceTypeConverter(staticType)
		}

		// Interface types need to be placed in intersection types
		if _, ok := parentType.(*interpreter.IntersectionStaticType); !ok {
			// If the interface type was not converted to another type,
			// and given the parent type is not an intersection type,
			// then the original interface type must be placed in an intersection type
			if convertedType == nil {
				convertedType = interpreter.NewIntersectionStaticType(
					nil, []*interpreter.InterfaceStaticType{
						staticType,
					},
				)
			} else {
				// If the interface type was converted to another type,
				// it may have been converted to
				// - a different kind of type, e.g. a composite type,
				//   in which case the converted type should be returned as-is
				// - another interface type –
				//   given the parent type is not an intersection type,
				//   then the converted interface type must be placed in an intersection type
				if convertedInterfaceType, ok := convertedType.(*interpreter.InterfaceStaticType); ok {
					convertedType = interpreter.NewIntersectionStaticType(
						nil, []*interpreter.InterfaceStaticType{
							convertedInterfaceType,
						},
					)
				}
			}
		}

		return convertedType

	case dummyStaticType:
		// This is for testing the migration.
		// i.e: the dummyStaticType wrapper was only introduced to make it possible to use the type as a dictionary key.
		// Ignore the wrapper, and continue with the inner type.
		return m.maybeConvertStaticType(staticType.PrimitiveStaticType, staticType)

	case interpreter.PrimitiveStaticType:
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

	default:
		panic(errors.NewUnexpectedError("unexpected static type: %T", staticType))
	}

	return nil
}

func rewriteLegacyIntersectionType(
	intersectionType *interpreter.IntersectionStaticType,
	parentType interpreter.StaticType,
) interpreter.StaticType {

	// Rewrite the intersection type,
	// if it does not appear in a reference type.

	if _, ok := parentType.(*interpreter.ReferenceStaticType); ok {
		return intersectionType
	}

	// Rewrite rules (also enforced by contract update checker):
	//
	// - T{} / Any*{} -> T/Any*
	//
	//   If the intersection type has no interface types,
	//   then return the legacy type as-is.
	//
	//   This prevents the migration from creating an intersection type with no interface types,
	//   as static to sema type conversion ignores the legacy type.
	//
	// - Any*{A,...}  -> {A,...}
	//
	//   If the intersection type has no or an AnyStruct/AnyResource legacy type,
	//   and has at least one interface type,
	//   then return the intersection type without the legacy type.
	//
	// - T{A,...}    -> T
	//
	//   If the intersection type has a legacy type,
	//   and has at least one interface type,
	//   then return the legacy type as-is.

	legacyType := intersectionType.LegacyType

	if len(intersectionType.Types) > 0 {
		switch legacyType {
		case nil,
			interpreter.PrimitiveStaticTypeAnyStruct,
			interpreter.PrimitiveStaticTypeAnyResource:

			// Drop the legacy type, keep the interface types
			return interpreter.NewIntersectionStaticType(nil, intersectionType.Types)
		}
	}

	if legacyType == nil {
		panic(errors.NewUnexpectedError(
			"invalid intersection type with no interface types and no legacy type: %s",
			intersectionType,
		))
	}
	return legacyType
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
