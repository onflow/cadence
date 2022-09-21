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
	"github.com/onflow/cadence/runtime/interpreter"
)

func arrayElements(inter *interpreter.Interpreter, array *interpreter.ArrayValue) []interpreter.Value {
	count := array.Count()
	result := make([]interpreter.Value, count)
	for i := 0; i < count; i++ {
		result[i] = array.Get(inter, interpreter.ReturnEmptyLocationRange, i)
	}
	return result
}

func dictionaryKeyValues(inter *interpreter.Interpreter, dict *interpreter.DictionaryValue) []interpreter.Value {
	count := dict.Count() * 2
	result := make([]interpreter.Value, count)
	i := 0
	dict.Iterate(inter, func(key, value interpreter.Value) (resume bool) {
		result[i*2] = key
		result[i*2+1] = value
		i++

		return true
	})
	return result
}

type entry[K, V any] struct {
	key   K
	value V
}

// similar to dictionaryKeyValues, attempting to map untyped Values to concrete values using the provided morphisms. if a conversion fails, then this function returns (nil, false).
// useful in contexts when Cadence values need to be extracted into their go counterparts.
func dictionaryEntries[K, V any](
	inter *interpreter.Interpreter,
	dict *interpreter.DictionaryValue,
	fromKey func(interpreter.Value) (K, bool),
	fromVal func(interpreter.Value) (V, bool),
) ([]entry[K, V], bool) {

	count := dict.Count()
	res := make([]entry[K, V], count)

	iterStatus := true
	idx := 0
	dict.Iterate(inter, func(rawKey, rawValue interpreter.Value) (resume bool) {
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

		res[idx] = entry[K, V]{key, value}
		return iterStatus
	})

	return res, iterStatus
}
