/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInspectValue(t *testing.T) {

	t.Parallel()

	inter := newTestInterpreter(t)

	// Prepare composite value

	var compositeValue *CompositeValue
	{
		dictionaryStaticType := DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeInt256,
		}
		dictValueKey := NewUnmeteredStringValue("hello world")
		dictValueValue := NewUnmeteredInt256ValueFromInt64(1)
		dictValue := NewDictionaryValue(
			inter,
			EmptyLocationRange,
			dictionaryStaticType,
			dictValueKey, dictValueValue,
		)

		arrayValue := NewArrayValue(
			inter,
			EmptyLocationRange,
			VariableSizedStaticType{
				Type: dictionaryStaticType,
			},
			common.ZeroAddress,
			dictValue,
		)

		optionalValue := NewUnmeteredSomeValueNonCopying(arrayValue)

		compositeValue = newTestCompositeValue(inter, common.ZeroAddress)
		compositeValue.SetMember(
			inter,
			EmptyLocationRange,
			"value",
			optionalValue,
		)
	}

	// Get actually stored values.
	// The values above were removed when they were inserted into the containers.

	optionalValue := compositeValue.GetField(inter, EmptyLocationRange, "value").(*SomeValue)
	arrayValue := optionalValue.InnerValue(inter, EmptyLocationRange).(*ArrayValue)
	dictValue := arrayValue.Get(inter, EmptyLocationRange, 0).(*DictionaryValue)
	dictValueKey := NewUnmeteredStringValue("hello world")

	dictValueValue, _ := dictValue.Get(inter, EmptyLocationRange, dictValueKey)

	t.Run("dict", func(t *testing.T) {

		var inspectedValues []Value

		InspectValue(
			inter,
			dictValue,
			func(value Value) bool {
				inspectedValues = append(inspectedValues, value)
				return true
			},
		)

		AssertValueSlicesEqual(
			t,
			inter,
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
			inter,
			compositeValue,
			func(value Value) bool {
				inspectedValues = append(inspectedValues, value)
				return true
			},
		)

		AssertValueSlicesEqual(
			t,
			inter,
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
