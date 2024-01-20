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

package string_normalization

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/interpreter"
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
) (interpreter.Value, error) {
	switch value := value.(type) {
	case *interpreter.StringValue:
		return interpreter.NewUnmeteredStringValue(value.Str), nil

	case interpreter.CharacterValue:
		return interpreter.NewUnmeteredCharacterValue(value.Str), nil
	}

	return nil, nil
}
