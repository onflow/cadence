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

package string_normalization

import (
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/sema"
)

type StringNormalizingMigration struct{}

var _ migrations.ValueMigration = StringNormalizingMigration{}

func NewStringNormalizingMigration() StringNormalizingMigration {
	return StringNormalizingMigration{}
}

func (StringNormalizingMigration) Name() string {
	return "StringNormalizingMigration"
}

func (StringNormalizingMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ migrations.ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	// Normalize strings and characters to NFC.
	// If the value is already in NFC, skip the migration.

	switch value := value.(type) {
	case *interpreter.StringValue:
		unnormalizedStr := value.UnnormalizedStr
		normalizedStr := norm.NFC.String(unnormalizedStr)
		if normalizedStr == unnormalizedStr {
			return nil, nil
		}
		return interpreter.NewStringValue_Unsafe(normalizedStr, unnormalizedStr), nil //nolint:staticcheck

	case interpreter.CharacterValue:
		unnormalizedStr := value.UnnormalizedStr
		normalizedStr := norm.NFC.String(unnormalizedStr)
		if normalizedStr == unnormalizedStr {
			return nil, nil
		}
		return interpreter.NewCharacterValue_Unsafe(normalizedStr, unnormalizedStr), nil //nolint:staticcheck
	}

	return nil, nil
}

func (StringNormalizingMigration) Domains() map[string]struct{} {
	return nil
}

func (m StringNormalizingMigration) CanSkip(valueType interpreter.StaticType) bool {
	return CanSkipStringNormalizingMigration(valueType)
}

func CanSkipStringNormalizingMigration(valueType interpreter.StaticType) bool {
	switch ty := valueType.(type) {
	case *interpreter.DictionaryStaticType:
		return CanSkipStringNormalizingMigration(ty.KeyType) &&
			CanSkipStringNormalizingMigration(ty.ValueType)

	case interpreter.ArrayStaticType:
		return CanSkipStringNormalizingMigration(ty.ElementType())

	case *interpreter.OptionalStaticType:
		return CanSkipStringNormalizingMigration(ty.Type)

	case *interpreter.CapabilityStaticType:
		return true

	case interpreter.PrimitiveStaticType:

		switch ty {
		case interpreter.PrimitiveStaticTypeBool,
			interpreter.PrimitiveStaticTypeVoid,
			interpreter.PrimitiveStaticTypeAddress,
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeBlock,
			interpreter.PrimitiveStaticTypeCapability:

			return true
		}

		if !ty.IsDeprecated() { //nolint:staticcheck
			semaType := ty.SemaType()

			if sema.IsSubType(semaType, sema.NumberType) ||
				sema.IsSubType(semaType, sema.PathType) {

				return true
			}
		}
	}

	return false
}
