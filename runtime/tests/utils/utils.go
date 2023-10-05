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

	"github.com/k0kubun/pp/v3"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/runtime/common"
)

// TestLocation is used as the default location for programs in tests.
const TestLocation = common.StringLocation("test")

// ImportedLocation is used as the default location for imported programs in tests.
const ImportedLocation = common.StringLocation("imported")

// AssertEqualWithDiff asserts that two objects are equal.
//
// If the objects are not equal, this function prints a human-readable diff.
func AssertEqualWithDiff(t *testing.T, expected, actual any) {

	// the maximum levels of a struct to recurse into
	// this prevents infinite recursion from circular references
	diff := pretty.Diff(expected, actual)

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

	if hasImportLocation, ok := err.(common.HasLocation); ok {
		location := hasImportLocation.ImportLocation()
		assert.NotNil(t, location)
	}

	if hasPosition, ok := err.(ast.HasPosition); ok {
		_ = hasPosition.StartPosition()
		_ = hasPosition.EndPosition(nil)
	}

	if hasErrorNotes, ok := err.(errors.ErrorNotes); ok {
		for _, note := range hasErrorNotes.ErrorNotes() {
			_ = note.Message()
		}
	}

	if hasSecondaryError, ok := err.(errors.SecondaryError); ok {
		_ = hasSecondaryError.SecondaryError()
	}

	if hasSuggestedFixes, ok := err.(sema.HasSuggestedFixes); ok {
		_ = hasSuggestedFixes.SuggestFixes("")
	}
}
