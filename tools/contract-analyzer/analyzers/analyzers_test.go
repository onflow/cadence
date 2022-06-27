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

package analyzers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/tools/analysis"

	"github.com/onflow/cadence/tools/contract-analyzer/analyzers"
)

var testLocation = common.StringLocation("test")
var testLocationID = testLocation.ID()

func testAnalyzers(t *testing.T, code string, analyzers ...*analysis.Analyzer) []analysis.Diagnostic {

	config := analysis.NewSimpleConfig(
		analysis.NeedTypes,
		map[common.LocationID]string{
			testLocationID: code,
		},
		nil,
		nil,
	)

	programs, err := analysis.Load(config, testLocation)
	require.NoError(t, err)

	var diagnostics []analysis.Diagnostic

	programs[testLocationID].Run(
		analyzers,
		func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic)
		},
	)

	return diagnostics
}

func TestDeprecatedKeyFunctionsAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {
              pub fun test(account: AuthAccount) {
                  account.addPublicKey([])
                  account.removePublicKey(0)
              }
          }
        `,
		analyzers.DeprecatedKeyFunctionsAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 108, Line: 4, Column: 26},
					EndPos:   ast.Position{Offset: 119, Line: 4, Column: 37},
				},
				Location:         testLocation,
				Category:         "update recommended",
				Message:          "deprecated function 'addPublicKey' will get removed",
				SecondaryMessage: "replace with 'keys.add'",
			},
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 151, Line: 5, Column: 26},
					EndPos:   ast.Position{Offset: 165, Line: 5, Column: 40},
				},
				Location:         testLocation,
				Category:         "update recommended",
				Message:          "deprecated function 'removePublicKey' will get removed",
				SecondaryMessage: "replace with 'keys.revoke'",
			},
		},
		diagnostics,
	)
}

func TestReferenceOperatorAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {
              pub fun test() {
                  let ref = &1 as! &Int
              }
          }
        `,
		analyzers.ReferenceOperatorAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 90, Line: 4, Column: 28},
					EndPos:   ast.Position{Offset: 100, Line: 4, Column: 38},
				},
				Location:         testLocation,
				Category:         "update recommended",
				Message:          "incorrect reference operator used",
				SecondaryMessage: "use the 'as' operator",
			},
		},
		diagnostics,
	)
}

func TestForceOperatorAnalyzer(t *testing.T) {

	t.Parallel()

	t.Run("unnecessary", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
			pub contract Test {
				pub fun test() {
					let x = 3
					let y = x!
				}
			}
			`,
			analyzers.UnnecessaryForceAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 73, Line: 5, Column: 13},
						EndPos:   ast.Position{Offset: 74, Line: 5, Column: 14},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "unnecessary force operator",
				},
			},
			diagnostics,
		)
	})

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
			pub contract Test {
				pub fun test() {
					let x: Int? = 3
					let y = x!
				}
			}
			`,
			analyzers.UnnecessaryForceAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic(nil),
			diagnostics,
		)
	})
}

func TestCastingAnalyzer(t *testing.T) {

	t.Parallel()

	t.Run("redundant", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
			pub contract Test {
				pub fun test() {
					let x = true as Bool
				}
			}
			`,
			analyzers.RedundantCastAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 66, Line: 4, Column: 21},
						EndPos:   ast.Position{Offset: 69, Line: 4, Column: 24},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "cast to `Bool` is redundant",
				},
			},
			diagnostics,
		)
	})

	t.Run("always succeeding force", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
			pub contract Test {
				pub fun test() {
					let x = true as! Bool
				}
			}
			`,
			analyzers.RedundantCastAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 58, Line: 4, Column: 13},
						EndPos:   ast.Position{Offset: 70, Line: 4, Column: 25},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "force cast ('as!') from `Bool` to `Bool` always succeeds",
				},
			},
			diagnostics,
		)
	})

	t.Run("always succeeding failable", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
			pub contract Test {
				pub fun test() {
					let x = true as? Bool
				}
			}
			`,
			analyzers.RedundantCastAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 58, Line: 4, Column: 13},
						EndPos:   ast.Position{Offset: 70, Line: 4, Column: 25},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "failable cast ('as?') from `Bool` to `Bool` always succeeds",
				},
			},
			diagnostics,
		)
	})

}

func TestCheckNumberConversionReplacementHint(t *testing.T) {

	t.Parallel()

	// to fixed point type

	//// integer literal

	t.Run("positive integer to signed fixed-point type", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t, `
			pub contract Test {
				pub fun test() {
					let x = Fix64(1)
				}
			}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 58, Line: 4, Column: 13},
						EndPos:   ast.Position{Offset: 65, Line: 4, Column: 20},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1.0 as Fix64`",
				},
			},
			diagnostics,
		)
	})

	t.Run("positive integer to unsigned fixed-point type", func(t *testing.T) {

		t.Parallel()
		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = UFix64(1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 63, Line: 4, Column: 20},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1.0`",
				},
			},
			diagnostics,
		)
	})

	t.Run("negative integer to signed fixed-point type", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Fix64(-1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 63, Line: 4, Column: 20},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `-1.0`",
				},
			},
			diagnostics,
		)
	})

	//// fixed-point literal

	t.Run("positive fixed-point to unsigned fixed-point type", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = UFix64(1.2)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 65, Line: 4, Column: 22},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1.2`",
				},
			},
			diagnostics,
		)
	})

	t.Run("negative fixed-point to signed fixed-point type", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Fix64(-1.2)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 65, Line: 4, Column: 22},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `-1.2`",
				},
			},
			diagnostics,
		)
	})

	// to integer type

	//// integer literal

	t.Run("positive integer to unsigned integer type", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = UInt8(1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 62, Line: 4, Column: 19},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1 as UInt8`",
				},
			},
			diagnostics,
		)
	})

	t.Run("positive integer to signed integer type", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Int8(1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 61, Line: 4, Column: 18},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1 as Int8`",
				},
			},
			diagnostics,
		)
	})

	t.Run("negative integer to signed integer type", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Int8(-1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 62, Line: 4, Column: 19},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `-1 as Int8`",
				},
			},
			diagnostics,
		)
	})

	t.Run("positive integer to Int", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Int(1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 60, Line: 4, Column: 17},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `1`",
				},
			},
			diagnostics,
		)
	})

	t.Run("negative integer to Int", func(t *testing.T) {
		t.Parallel()

		diagnostics := testAnalyzers(t, `
		pub contract Test {
			pub fun test() {
				let x = Int(-1)
			}
		}`,
			analyzers.NumberFunctionArgumentAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 55, Line: 4, Column: 12},
						EndPos:   ast.Position{Offset: 61, Line: 4, Column: 18},
					},
					Location: testLocation,
					Category: "lint",
					Message:  "consider replacing with: `-1`",
				},
			},
			diagnostics,
		)
	})
}
