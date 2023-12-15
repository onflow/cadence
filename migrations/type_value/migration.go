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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type TypeValueMigration struct{}

var _ migrations.ValueMigration = TypeValueMigration{}

func NewTypeValueMigration() TypeValueMigration {
	return TypeValueMigration{}
}

func (TypeValueMigration) Name() string {
	return "TypeValueMigration"
}

// Migrate migrates intersection types (formerly, restricted types) inside `TypeValue`s.
func (TypeValueMigration) Migrate(
	_ interpreter.AddressPath,
	value interpreter.Value,
	inter *interpreter.Interpreter,
) (newValue interpreter.Value) {
	switch value := value.(type) {
	case interpreter.TypeValue:
		convertedType := maybeConvertType(value.Type, inter)
		if convertedType == nil {
			return
		}
		return interpreter.NewTypeValue(nil, convertedType)
	}

	return
}

func maybeConvertType(
	staticType interpreter.StaticType,
	inter *interpreter.Interpreter,
) interpreter.StaticType {

	switch staticType := staticType.(type) {
	case *interpreter.ConstantSizedStaticType:
		convertedType := maybeConvertType(staticType.Type, inter)
		if convertedType != nil {
			return interpreter.NewConstantSizedStaticType(nil, convertedType, staticType.Size)
		}

	case *interpreter.VariableSizedStaticType:
		convertedType := maybeConvertType(staticType.Type, inter)
		if convertedType != nil {
			return interpreter.NewVariableSizedStaticType(nil, convertedType)
		}

	case *interpreter.DictionaryStaticType:
		convertedKeyType := maybeConvertType(staticType.KeyType, inter)
		convertedValueType := maybeConvertType(staticType.ValueType, inter)
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
		convertedBorrowType := maybeConvertType(staticType.BorrowType, inter)
		if convertedBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, convertedBorrowType)
		}

	case *interpreter.OptionalStaticType:
		convertedInnerType := maybeConvertType(staticType.Type, inter)
		if convertedInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, convertedInnerType)
		}

	case *interpreter.ReferenceStaticType:
		// TODO: Reference of references must not be allowed?
		convertedReferencedType := maybeConvertType(staticType.ReferencedType, inter)
		if convertedReferencedType != nil {
			return interpreter.NewReferenceStaticType(nil, staticType.Authorization, convertedReferencedType)
		}

	case interpreter.FunctionStaticType:
		// Non-storable

	case *interpreter.CompositeStaticType,
		*interpreter.InterfaceStaticType,
		interpreter.PrimitiveStaticType:

		// Nothing to do

	case *interpreter.IntersectionStaticType:
		// No need to convert `staticType.Types` as they can only be interfaces.

		legacyType := staticType.LegacyType
		if legacyType != nil {
			convertedLegacyType := maybeConvertType(legacyType, inter)
			if convertedLegacyType != nil {
				intersectionType := interpreter.NewIntersectionStaticType(nil, staticType.Types)
				intersectionType.LegacyType = convertedLegacyType
				return intersectionType
			}
		}

		// If the set has at least two items,
		// then force it to be re-stored/re-encoded
		if len(staticType.Types) >= 2 {
			return staticType
		}

	default:
		panic(errors.NewUnreachableError())
	}

	return nil
}
