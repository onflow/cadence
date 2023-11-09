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
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type StringNormalizingMigration struct {
	storage     *runtime.Storage
	interpreter *interpreter.Interpreter
}

func NewStringNormalizingMigration(
	interpreter *interpreter.Interpreter,
	storage *runtime.Storage,
) *StringNormalizingMigration {
	return &StringNormalizingMigration{
		storage:     storage,
		interpreter: interpreter,
	}
}

func (m *StringNormalizingMigration) Migrate(
	addressIterator migrations.AddressIterator,
	reporter migrations.Reporter,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migrateStringValuesInAccount(
			address,
			reporter,
		)
	}

	err := m.storage.Commit(m.interpreter, false)
	if err != nil {
		panic(err)
	}
}

func (m *StringNormalizingMigration) migrateStringValuesInAccount(
	address common.Address,
	reporter migrations.Reporter,
) {

	accountStorage := migrations.NewAccountStorage(m.storage, address)

	accountStorage.ForEachValue(
		m.interpreter,
		common.AllPathDomains,
		m.migrateValue,
		reporter,
	)
}

func (m *StringNormalizingMigration) migrateValue(
	value interpreter.Value,
) (newValue interpreter.Value, updatedInPlace bool) {
	return migrations.MigrateNestedValue(m.interpreter, value, m.migrateStringAndCharacterValues)
}

func (m *StringNormalizingMigration) migrateStringAndCharacterValues(
	value interpreter.Value,
) (newValue interpreter.Value, updatedInPlace bool) {
	switch value := value.(type) {
	case *interpreter.StringValue:
		return interpreter.NewUnmeteredStringValue(value.Str), false
	case interpreter.CharacterValue:
		return interpreter.NewUnmeteredCharacterValue(string(value)), false
	}

	return nil, false
}
