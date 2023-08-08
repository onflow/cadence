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

package utils

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"

	"github.com/onflow/cadence/runtime/common"
)

func init() {
	pp.ColoringEnabled = false
}

// TestLocation is used as the default location for programs in tests.
const TestLocation = common.StringLocation("test")

// ImportedLocation is used as the default location for imported programs in tests.
const ImportedLocation = common.StringLocation("imported")

// AssertEqualWithDiff asserts that two objects are equal.
//
// If the objects are not equal, this function prints a human-readable diff.
func AssertEqualWithDiff(t *testing.T, expected, actual any) {
	if !assert.Equal(t, expected, actual) {
		// the maximum levels of a struct to recurse into
		// this prevents infinite recursion from circular references
		deep.MaxDepth = 100

		diff := deep.Equal(expected, actual)

		if len(diff) != 0 {
			s := strings.Builder{}

			for i, d := range diff {
				if i == 0 {
					s.WriteString("diff    : ")
				} else {
					s.WriteString("          ")
				}

				s.WriteString(d)
				s.WriteString("\n")
			}

			t.Errorf(
				"Not equal: \n"+
					"expected: %s\n"+
					"actual  : %s\n\n"+
					"%s",
				pp.Sprint(expected),
				pp.Sprint(actual),
				s.String(),
			)
		}
	}
}

func AsInterfaceType(name string, kind common.CompositeKind) string {
	switch kind {
	case common.CompositeKindResource, common.CompositeKindStructure:
		return fmt.Sprintf("{%s}", name)
	default:
		return name
	}
}

func DeploymentTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.add(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
}

func RemovalTransaction(name string) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.remove(name: "%s")
              }
          }
        `,
		name,
	))
}

func UpdateTransaction(name string, contract []byte) []byte {
	return []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.contracts.update__experimental(name: "%s", code: "%s".decodeHex())
              }
          }
        `,
		name,
		hex.EncodeToString(contract),
	))
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
		diff := deep.Equal(expected, actual)

		var message string

		if len(diff) != 0 {
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

// RequireError is a wrapper around require.Error which also ensures
// that the error message, the secondary message (if any),
// and the error notes' (if any) messages can be successfully produced
func RequireError(t *testing.T, err error) {
	require.Error(t, err)

	_ = err.Error()

	if hasErrorNotes, ok := err.(errors.ErrorNotes); ok {
		for _, note := range hasErrorNotes.ErrorNotes() {
			_ = note.Message()
		}
	}

	if hasSecondaryError, ok := err.(errors.SecondaryError); ok {
		_ = hasSecondaryError.SecondaryError()
	}
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
	dict.Iterate(inter, func(key, value interpreter.Value) (resume bool) {
		result[i*2] = key
		result[i*2+1] = value
		i++

		return true
	})
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

		res[idx] = DictionaryEntry[K, V]{
			Key:   key,
			Value: value,
		}
		return iterStatus
	})

	return res, iterStatus
}
