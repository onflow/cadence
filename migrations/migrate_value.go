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

package migrations

import (
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

var emptyLocationRange = interpreter.EmptyLocationRange

func MigrateNestedValue(
	inter *interpreter.Interpreter,
	value interpreter.Value,
	migrate func(value interpreter.Value) (newValue interpreter.Value, updatedInPlace bool),
) (newValue interpreter.Value, updatedInPlace bool) {
	switch value := value.(type) {
	case *interpreter.SomeValue:
		innerValue := value.InnerValue(inter, emptyLocationRange)
		newInnerValue, _ := MigrateNestedValue(inter, innerValue, migrate)
		if newInnerValue != nil {
			return interpreter.NewSomeValueNonCopying(inter, newInnerValue), true
		}

		return

	case *interpreter.ArrayValue:
		array := value

		// Migrate array elements
		count := array.Count()
		for index := 0; index < count; index++ {
			element := array.Get(inter, emptyLocationRange, index)
			newElement, elementUpdated := MigrateNestedValue(inter, element, migrate)

			updatedInPlace = updatedInPlace || elementUpdated

			if newElement == nil {
				continue
			}

			array.Set(
				inter,
				emptyLocationRange,
				index,
				newElement,
			)
		}

		// The array itself doesn't need to be replaced.
		return

	case *interpreter.CompositeValue:
		composite := value

		// Read the field names first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var fieldNames []string
		composite.ForEachField(nil, func(fieldName string, fieldValue interpreter.Value) (resume bool) {
			fieldNames = append(fieldNames, fieldName)
			return true
		})

		for _, fieldName := range fieldNames {
			existingValue := composite.GetField(inter, interpreter.EmptyLocationRange, fieldName)

			migratedValue, valueUpdated := MigrateNestedValue(inter, existingValue, migrate)

			updatedInPlace = updatedInPlace || valueUpdated

			if migratedValue == nil {
				continue
			}

			composite.SetMember(inter, emptyLocationRange, fieldName, migratedValue)
		}

		// The composite itself does not have to be replaced
		return

	case *interpreter.DictionaryValue:
		dictionary := value

		// Read the keys first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var existingKeys []interpreter.Value
		dictionary.Iterate(inter, func(key, _ interpreter.Value) (resume bool) {
			existingKeys = append(existingKeys, key)
			return true
		})

		for _, existingKey := range existingKeys {
			existingValue, exist := dictionary.Get(nil, interpreter.EmptyLocationRange, existingKey)
			if !exist {
				panic(errors.NewUnreachableError())
			}

			newKey, keyUpdated := MigrateNestedValue(inter, existingKey, migrate)
			newValue, valueUpdated := MigrateNestedValue(inter, existingValue, migrate)

			updatedInPlace = updatedInPlace || keyUpdated || valueUpdated

			if newKey == nil && newValue == nil {
				continue
			}

			// We only reach here at least one of key or value has been migrated.
			var keyToSet, valueToSet interpreter.Value

			if newKey == nil {
				keyToSet = existingKey
			} else {
				// Key was migrated.
				// Remove the old value at the old key.
				// This old value will be inserted again with the new key, unless the value is also migrated.
				_ = dictionary.RemoveKey(inter, emptyLocationRange, existingKey)
				keyToSet = newKey
			}

			if newValue == nil {
				valueToSet = existingValue
			} else {
				// Value was migrated
				valueToSet = newValue
			}

			dictionary.Insert(inter, emptyLocationRange, keyToSet, valueToSet)
		}

		// The dictionary itself does not have to be replaced
		return
	default:
		return migrate(value)
	}
}
