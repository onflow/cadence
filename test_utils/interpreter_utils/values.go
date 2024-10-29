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

package interpreter_utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/interpreter"
)

func RequireValuesEqual(t testing.TB, inter *interpreter.Interpreter, expected, actual interpreter.Value) {
	if !AssertValuesEqual(t, inter, expected, actual) {
		t.FailNow()
	}
}

func AssertValueSlicesEqual(t testing.TB, inter *interpreter.Interpreter, expected, actual []interpreter.Value) bool {
	if !assert.Equal(t, len(expected), len(actual)) {
		return false
	}

	for i, value := range expected {
		if !AssertValuesEqual(t, inter, value, actual[i]) {
			return false
		}
	}

	return true
}

func ValuesAreEqual(inter *interpreter.Interpreter, expected, actual interpreter.Value) bool {
	if expected == nil {
		return actual == nil
	}

	if expected, ok := expected.(interpreter.EquatableValue); ok {
		return expected.Equal(inter, interpreter.EmptyLocationRange, actual)
	}

	return assert.ObjectsAreEqual(expected, actual)
}

func AssertValuesEqual(t testing.TB, interpreter *interpreter.Interpreter, expected, actual interpreter.Value) bool {
	if !ValuesAreEqual(interpreter, expected, actual) {
		diff := pretty.Diff(expected, actual)

		var message string

		if len(diff) > 0 {
			s := strings.Builder{}
			_, _ = fmt.Fprintf(&s,
				"Not equal: \n"+
					"expected: %s\n"+
					"actual  : %s\n\n",
				expected,
				actual,
			)

			for i, d := range diff {
				if i == 0 {
					s.WriteString("diff    : ")
				} else {
					s.WriteString("          ")
				}

				s.WriteString(d)
				s.WriteString("\n")
			}

			message = s.String()
		}

		return assert.Fail(t, message)
	}

	return true
}

func ArrayElements(inter *interpreter.Interpreter, array *interpreter.ArrayValue) []interpreter.Value {
	count := array.Count()
	result := make([]interpreter.Value, count)
	for i := 0; i < count; i++ {
		result[i] = array.Get(inter, interpreter.EmptyLocationRange, i)
	}
	return result
}

func DictionaryKeyValues(inter *interpreter.Interpreter, dict *interpreter.DictionaryValue) []interpreter.Value {
	count := dict.Count() * 2
	result := make([]interpreter.Value, count)
	i := 0
	dict.Iterate(
		inter,
		interpreter.EmptyLocationRange,
		func(key, value interpreter.Value) (resume bool) {
			result[i*2] = key
			result[i*2+1] = value
			i++

			return true
		},
	)
	return result
}

type DictionaryEntry[K, V any] struct {
	Key   K
	Value V
}

// DictionaryEntries is similar to DictionaryKeyValues,
// attempting to map untyped Values to concrete values using the provided morphisms.
// If a conversion fails, then this function returns (nil, false).
// Useful in contexts when Cadence values need to be extracted into their go counterparts.
func DictionaryEntries[K, V any](
	inter *interpreter.Interpreter,
	dict *interpreter.DictionaryValue,
	fromKey func(interpreter.Value) (K, bool),
	fromVal func(interpreter.Value) (V, bool),
) ([]DictionaryEntry[K, V], bool) {

	count := dict.Count()
	res := make([]DictionaryEntry[K, V], count)

	iterStatus := true
	idx := 0
	dict.Iterate(
		inter,
		interpreter.EmptyLocationRange,
		func(rawKey, rawValue interpreter.Value) (resume bool) {
			key, ok := fromKey(rawKey)

			if !ok {
				iterStatus = false
				return iterStatus
			}

			value, ok := fromVal(rawValue)
			if !ok {
				iterStatus = false
				return iterStatus
			}

			res[idx] = DictionaryEntry[K, V]{
				Key:   key,
				Value: value,
			}
			return iterStatus
		},
	)

	return res, iterStatus
}
