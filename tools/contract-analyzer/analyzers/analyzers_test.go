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
					StartPos: ast.Position{Offset: 154, Line: 5, Column: 38},
					EndPos:   ast.Position{Offset: 160, Line: 5, Column: 44},
				},
				Location:         testLocation,
				Message:          "reference to optional will return optional reference",
				SecondaryMessage: "replace with '&String?'",
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
					StartPos: ast.Position{Offset: 108, Line: 4, Column: 26},
					EndPos:   ast.Position{Offset: 119, Line: 4, Column: 37},
				},
				Location:         testLocation,
				Message:          "deprecated function 'addPublicKey' will get removed",
				SecondaryMessage: "replace with 'keys.add'",
			},
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 151, Line: 5, Column: 26},
					EndPos:   ast.Position{Offset: 165, Line: 5, Column: 40},
				},
				Location:         testLocation,
				Message:          "deprecated function 'removePublicKey' will get removed",
				SecondaryMessage: "replace with 'keys.revoke'",
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
				Message:  "arithmetic operations on number supertypes will get removed",
			},
		},
		diagnostics,
	)
}

func TestParameterListMissingCommasAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {
              pub fun test(a: Int     b: Int) {
                  fun (x: Int   y: Int) {}
              }
          }
        `,
		analyzers.ParameterListMissingCommasAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 64, Line: 3, Column: 33},
					EndPos:   ast.Position{Offset: 64, Line: 3, Column: 33},
				},
				Location: testLocation,
				Message:  "missing comma between parameters",
			},
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 108, Line: 4, Column: 29},
					EndPos:   ast.Position{Offset: 108, Line: 4, Column: 29},
				},
				Location: testLocation,
				Message:  "missing comma between parameters",
			},
		},
		diagnostics,
	)
}

func TestSupertypeInferenceAnalyzer(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`
              // same types
              pub let a = [1, 2]

              // different types
              pub let b = [true as AnyStruct, "true"]
            `,
			analyzers.SupertypeInferenceAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 122, Line: 6, Column: 26},
						EndPos:   ast.Position{Offset: 148, Line: 6, Column: 52},
					},
					Location:         testLocation,
					Message:          "type inference for arrays will change",
					SecondaryMessage: "ensure the newly inferred type is correct",
				},
			},
			diagnostics,
		)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`

              // same types
              pub let a = {1: "1", 2: "2"}

              // different value types
              pub let b = {1: "1" as AnyStruct, 2: true}
            `,
			analyzers.SupertypeInferenceAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 139, Line: 7, Column: 26},
						EndPos:   ast.Position{Offset: 168, Line: 7, Column: 55},
					},
					Location:         testLocation,
					Message:          "type inference for dictionaries will change",
					SecondaryMessage: "ensure the newly inferred type is correct",
				},
			},
			diagnostics,
		)
	})

	t.Run("conditional", func(t *testing.T) {

		t.Parallel()

		diagnostics := testAnalyzers(t,
			`

              // same types
              pub let a = true ? 1 : 2

              // different types
              pub let b = true ? 1 as AnyStruct: "2"
            `,
			analyzers.SupertypeInferenceAnalyzer,
		)

		require.Equal(
			t,
			[]analysis.Diagnostic{
				{
					Range: ast.Range{
						StartPos: ast.Position{Offset: 129, Line: 7, Column: 26},
						EndPos:   ast.Position{Offset: 154, Line: 7, Column: 51},
					},
					Location:         testLocation,
					Message:          "type inference for conditionals / ternary operations will change",
					SecondaryMessage: "ensure the newly inferred type is correct",
				},
			},
			diagnostics,
		)
	})
}

func TestExternalMutationAnalyzer(t *testing.T) {

	t.Parallel()

	diagnostics := testAnalyzers(t,
		`
          pub contract Test {

              pub struct A {
                 pub let internal: [Int]
                 pub(set) var external: [Int]

                 init() {
                     self.internal = []
                     self.external = []
                 }

                 pub fun internalAdd1(number: Int) {
                     self.internal.append(number)
                 }

                 pub fun externalAdd1(number: Int) {
                     self.external.append(number)
                 }

                 pub fun internalSet1(index: Int, number: Int) {
                     self.internal[index] = number
                 }

                 pub fun externalSet1(index: Int, number: Int) {
                     self.external[index] = number
                 }
              }

              pub fun internalAdd2(a: A, number: Int) {
                 a.internal.append(number)
              }

              pub fun externalAdd2(a: A, number: Int) {
                 a.external.append(number)
              }

              pub fun internalSet2(a: A, index: Int, number: Int) {
                 a.internal[index] = number
              }

              pub fun externalSet2(a: A, index: Int, number: Int) {
                 a.external[index] = number
              }
          }
        `,
		analyzers.ExternalMutationAnalyzer,
	)

	require.Equal(
		t,
		[]analysis.Diagnostic{
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 882, Line: 31, Column: 17},
					EndPos:   ast.Position{Offset: 891, Line: 31, Column: 26},
				},
				Location:         testLocation,
				Message:          "external mutation of non-settable public container-typed field will get disallowed",
				SecondaryMessage: "add setter function for field, or change field access to pub(set)",
			},
			{
				Range: ast.Range{
					StartPos: ast.Position{Offset: 1126, Line: 39, Column: 17},
					EndPos:   ast.Position{Offset: 1135, Line: 39, Column: 26},
				},
				Location:         testLocation,
				Message:          "external mutation of non-settable public container-typed field will get disallowed",
				SecondaryMessage: "add setter function for field, or change field access to pub(set)",
			},
		},
		diagnostics,
	)
}
