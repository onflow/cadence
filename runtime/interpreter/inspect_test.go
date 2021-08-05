/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/assert"
)

func TestInspectValue(t *testing.T) {

	storage := NewInMemoryStorage()

	dictionaryStaticType := DictionaryStaticType{
		KeyType:   PrimitiveStaticTypeString,
		ValueType: PrimitiveStaticTypeInt256,
	}
	dictValueKey := NewStringValue("hello world")
	dictValueValue := NewInt256ValueFromInt64(1)
	dictValue := NewDictionaryValueUnownedNonCopying(
		newTestInterpreter(t),
		dictionaryStaticType,
		storage,
		dictValueKey, dictValueValue,
	)

	arrayValue := NewArrayValueUnownedNonCopying(
		VariableSizedStaticType{
			Type: dictionaryStaticType,
		},
		storage,
		dictValue,
	)

	optionalValue := NewSomeValueOwningNonCopying(arrayValue)

	compositeValue := newTestCompositeValue(storage, common.Address{})
	compositeValue.Fields.Set("value", optionalValue)

	t.Run("dict", func(t *testing.T) {

		var inspectedValues []Value

		InspectValue(
			dictValue,
			func(value Value) bool {
				inspectedValues = append(inspectedValues, value)
				return true
			},
		)

		assert.Equal(t,
			[]Value{
				dictValue,
				dictValueKey,
				nil, // end key
				dictValueValue,
				nil, // end value
				nil, // end dict
			},
			inspectedValues,
		)
	})

	t.Run("composite", func(t *testing.T) {

		var inspectedValues []Value

		InspectValue(
			compositeValue,
			func(value Value) bool {
				inspectedValues = append(inspectedValues, value)
				return true
			},
		)

		assert.Equal(t,
			[]Value{
				compositeValue,
				optionalValue,
				arrayValue,
				dictValue,
				dictValueKey,
				nil, // end key
				dictValueValue,
				nil, // end value
				nil, // end dict
				nil, // end array
				nil, // end optional
				nil, // end composite
			},
			inspectedValues,
		)
	})
}
