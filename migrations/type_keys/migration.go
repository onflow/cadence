/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package type_keys

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type TypeKeyMigration struct{}

var _ migrations.ValueMigration = TypeKeyMigration{}

func NewTypeKeyMigration() TypeKeyMigration {
	return TypeKeyMigration{}
}

func (TypeKeyMigration) Name() string {
	return "TypeKeyMigration"
}

func (TypeKeyMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	position migrations.ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	// Re-store Type values used as dictionary keys,
	// to ensure that even when such values failed to get migrated
	// by the static types and entitlements migration,
	// they are still stored using their new hash.

	if position == migrations.ValueMigrationPositionDictionaryKey {
		if typeValue, ok := value.(interpreter.TypeValue); ok {
			return typeValue, nil
		}
	}

	return nil, nil
}

func (TypeKeyMigration) Domains() map[string]struct{} {
	return nil
}

func (m TypeKeyMigration) CanSkip(valueType interpreter.StaticType) bool {
	return CanSkipTypeKeyMigration(valueType)
}

func CanSkipTypeKeyMigration(valueType interpreter.StaticType) bool {

	switch valueType := valueType.(type) {
	case *interpreter.DictionaryStaticType:
		return CanSkipTypeKeyMigration(valueType.KeyType) &&
			CanSkipTypeKeyMigration(valueType.ValueType)

	case interpreter.ArrayStaticType:
		return CanSkipTypeKeyMigration(valueType.ElementType())

	case *interpreter.OptionalStaticType:
		return CanSkipTypeKeyMigration(valueType.Type)

	case *interpreter.CapabilityStaticType:
		// Typed capability, can skip
		return true

	case interpreter.PrimitiveStaticType:

		switch valueType {
		case interpreter.PrimitiveStaticTypeMetaType:
			return false

		case interpreter.PrimitiveStaticTypeBool,
			interpreter.PrimitiveStaticTypeVoid,
			interpreter.PrimitiveStaticTypeAddress,
			interpreter.PrimitiveStaticTypeBlock,
			interpreter.PrimitiveStaticTypeString,
			interpreter.PrimitiveStaticTypeCharacter,
			// Untyped capability, can skip
			interpreter.PrimitiveStaticTypeCapability:

			return true
		}

		if !valueType.IsDeprecated() { //nolint:staticcheck
			semaType := valueType.SemaType()

			if sema.IsSubType(semaType, sema.NumberType) ||
				sema.IsSubType(semaType, sema.PathType) {

				return true
			}
		}
	}

	return false
}
