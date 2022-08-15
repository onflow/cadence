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

package pretty

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

type testError struct {
	ast.Range
}

func (testError) Error() string {
	return "test error"
}

func TestPrintBrokenCode(t *testing.T) {

	t.Parallel()

	const code = `pub resource R {}`
	lineCount := len(strings.Split(code, "\n"))

	location := common.StringLocation("test")

	var sb strings.Builder
	printer := NewErrorPrettyPrinter(&sb, false)
	err := printer.PrettyPrintError(
		testError{
			Range: ast.Range{
				StartPos: ast.Position{
					// NOTE: line number is after end of code
					Line:   lineCount + 2,
					Column: 0,
				},
				EndPos: ast.Position{
					Line:   lineCount,
					Column: 2,
				},
			},
		},
		location,
		map[common.Location]string{
			location: code,
		},
	)
	require.NoError(t, err)
	require.Equal(t,
		"error: test error\n"+
			" --> test:3:0\n",
		sb.String(),
	)
}

func TestPrintTabs(t *testing.T) {

	t.Parallel()

	const code = "\t  \t   let x = 1"

	location := common.StringLocation("test")

	var sb strings.Builder
	printer := NewErrorPrettyPrinter(&sb, false)
	err := printer.PrettyPrintError(
		testError{
			Range: ast.Range{
				StartPos: ast.Position{
					Line:   1,
					Column: 7,
				},
				EndPos: ast.Position{
					Line:   1,
					Column: 9,
				},
			},
		},
		location,
		map[common.Location]string{
			location: code,
		},
	)
	require.NoError(t, err)
	require.Equal(t,
		"error: test error\n"+
			" --> test:1:7\n"+
			"  |\n"+
			"1 | \t  \t   let x = 1\n"+
			"  | \t  \t   ^^^\n",
		sb.String(),
	)
}
