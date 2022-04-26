package analyzers_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/tools/analysis"
	"github.com/onflow/cadence/tools/contract-analyzer/analyzers"
	"github.com/stretchr/testify/require"
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

func TestReferenceToOptionalAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {
              pub fun test() {
                  let xs: {Int: String} = {0: "zero"}
                  let ref = &xs[0] as &String
              }
          }
	    `,
		analyzers.ReferenceToOptionalAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 131, Line: 5, Column: 27},
					EndPos:   ast.Position{Offset: 133, Line: 5, Column: 29},
				},
				Location: testLocation,
				Message:  "reference to optional",
			},
		},
		diagnostics,
	)
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
					StartPos: ast.Position{Offset: 88, Line: 4, Column: 14},
					EndPos:   ast.Position{Offset: 111, Line: 4, Column: 37},
				},
				Location: testLocation,
				Message:  "use of deprecated key management API: replace 'addPublicKey' with 'keys.add'",
			},
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 127, Line: 5, Column: 14},
					EndPos:   ast.Position{Offset: 152, Line: 5, Column: 39},
				},
				Location: testLocation,
				Message:  "use of deprecated key management API: replace 'removePublicKey' with 'keys.revoke'",
			},
		},
		diagnostics,
	)
}

func TestNumberSupertypeBinaryOperationsAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {
              pub fun test(a: Integer, b: Integer) {
                  let c = a - b
              }
          }
	    `,
		analyzers.NumberSupertypeBinaryOperationsAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 110, Line: 4, Column: 26},
					EndPos:   ast.Position{Offset: 114, Line: 4, Column: 30},
				},
				Location: testLocation,
				Message:  "binary operation on number supertype",
			},
		},
		diagnostics,
	)
}
