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

package common_utils

import (
	"strings"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/errors"

	"github.com/onflow/cadence/common"
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
	t.Helper()

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

// RequireError is a wrapper around require.Error which also ensures
// that the error message, the secondary message (if any),
// and the error notes' (if any) messages can be successfully produced
func RequireError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)

	_ = err.Error()

	if hasImportLocation, ok := err.(common.HasLocation); ok {
		_ = hasImportLocation.ImportLocation()
		// TODO: re-enable once VM has location support
		// assert.NotNil(t, location)
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

	if hasSuggestedFixes, ok := err.(errors.HasSuggestedFixes[ast.TextEdit]); ok {
		_ = hasSuggestedFixes.SuggestFixes("")
	}
}
