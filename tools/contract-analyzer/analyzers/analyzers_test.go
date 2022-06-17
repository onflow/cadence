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
