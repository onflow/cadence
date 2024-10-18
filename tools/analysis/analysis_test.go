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

package analysis_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/tests/checker"
	"github.com/onflow/cadence/tools/analysis"
)

func TestNeedSyntaxAndImport(t *testing.T) {

	t.Parallel()

	txLocation := common.TransactionLocation{1}
	const txCode = `
	  import 0x1

	  access(all) let y = "test" as! String
	`

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: contractAddress,
		Name:    "ContractA",
	}
	const contractCode = `
      access(all) contract ContractA {
	    init() {
	      let y = true as! Bool
	    }
	  }
	`

	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveAddressContractNames: func(address common.Address) ([]string, error) {
			require.Equal(t, contractAddress, address)
			return []string{contractLocation.Name}, nil
		},
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case txLocation:
				return []byte(txCode), nil

			case contractLocation:
				return []byte(contractCode), nil

			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	programs, err := analysis.Load(config, txLocation)
	require.NoError(t, err)

	require.NotNil(t, programs.Get(txLocation))
	require.NotNil(t, programs.Get(contractLocation))

	type locationRange struct {
		location common.Location
		ast.Range
	}

	var locationRanges []locationRange

	for _, program := range programs.All() {
		require.NotNil(t, program.Program)
		require.NotNil(t, program.Checker)

		// Run a simple analysis: Detect unnecessary cast

		var detected bool

		ast.Inspect(program.Program, func(element ast.Element) bool {
			castingExpression, ok := element.(*ast.CastingExpression)
			if !ok {
				return true
			}

			types := program.Checker.Elaboration.CastingExpressionTypes(castingExpression)
			leftHandType := types.StaticValueType
			rightHandType := types.TargetType

			if !sema.IsSubType(leftHandType, rightHandType) {
				return true
			}

			detected = true
			locationRanges = append(
				locationRanges,
				locationRange{
					location: program.Location,
					Range:    ast.NewRangeFromPositioned(nil, castingExpression),
				},
			)

			return false
		})

		require.True(t, detected)
	}

	sort.Slice(
		locationRanges,
		func(i, j int) bool {
			a := locationRanges[i]
			b := locationRanges[j]
			return a.location.TypeID(nil, "") < b.location.TypeID(nil, "")
		},
	)

	require.Equal(
		t,
		[]locationRange{
			{
				location: contractLocation,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 69, Line: 4, Column: 15},
					EndPos:   ast.Position{Offset: 81, Line: 4, Column: 27},
				},
			},
			{
				location: txLocation,
				Range: ast.Range{
					StartPos: ast.Position{Offset: 39, Line: 4, Column: 23},
					EndPos:   ast.Position{Offset: 55, Line: 4, Column: 39},
				},
			},
		},
		locationRanges,
	)
}

func TestParseError(t *testing.T) {

	t.Parallel()

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: contractAddress,
		Name:    "ContractA",
	}
	const contractCode = `
      access(all) contract ContractA {
	    init() {
	      ???
	    }
	  }
	`

	config := &analysis.Config{
		Mode: analysis.NeedSyntax,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case contractLocation:
				return []byte(contractCode), nil
			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	_, err := analysis.Load(config, contractLocation)
	require.Error(t, err)

	var parserError parser.Error
	require.ErrorAs(t, err, &parserError)
}

func TestCheckError(t *testing.T) {

	t.Parallel()

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: contractAddress,
		Name:    "ContractA",
	}
	const contractCode = `
      access(all) contract ContractA {
	    init() {
	      X
	    }
	  }
	`

	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case contractLocation:
				return []byte(contractCode), nil

			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	_, err := analysis.Load(config, contractLocation)
	require.Error(t, err)

	var checkerError *sema.CheckerError
	require.ErrorAs(t, err, &checkerError)
}

func TestHandledParserError(t *testing.T) {

	t.Parallel()

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: contractAddress,
		Name:    "ContractA",
	}
	const contractCode = `
	  access(all) contract ContractA {
	    init() {
	      ???
	    }
	  }
	`

	handlerCalls := 0
	config := &analysis.Config{
		Mode: analysis.NeedSyntax,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case contractLocation:
				return []byte(contractCode), nil

			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
		HandleParserError: func(err analysis.ParsingCheckingError, _ *ast.Program) error {
			require.Error(t, err)
			handlerCalls++
			return nil
		},
	}

	programs, err := analysis.Load(config, contractLocation)
	require.NoError(t, err)

	require.Equal(t, 1, handlerCalls)

	var parserError parser.Error
	require.ErrorAs(t,
		programs.Get(contractLocation).LoadError,
		&parserError,
	)
}

func TestHandledCheckerError(t *testing.T) {

	t.Parallel()

	contractAddress := common.MustBytesToAddress([]byte{0x1})
	contractLocation := common.AddressLocation{
		Address: contractAddress,
		Name:    "ContractA",
	}
	const contractCode = `
	  access(all) contract ContractA {
	    init() {
	      X
	    }
	  }
	`

	handlerCalls := 0
	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case contractLocation:
				return []byte(contractCode), nil
			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
		HandleCheckerError: func(err analysis.ParsingCheckingError, _ *sema.Checker) error {
			require.Error(t, err)
			handlerCalls++
			return nil
		},
	}

	programs, err := analysis.Load(config, contractLocation)
	require.Equal(t, 1, handlerCalls)
	require.NoError(t, err)

	var checkerError *sema.CheckerError
	require.ErrorAs(t,
		programs.Get(contractLocation).LoadError,
		&checkerError,
	)
}

// Tests that an error handled by the custom error handler is not returned
// However, it must set LoadError to the handled error so that checkers later importing the program can see it
func TestHandledLoadErrorImportedProgram(t *testing.T) {

	t.Parallel()

	contract1Address := common.MustBytesToAddress([]byte{0x1})
	contract1Location := common.AddressLocation{
		Address: contract1Address,
		Name:    "ContractA",
	}
	const contract1Code = `
	  import ContractB from 0x2
	  
      access(all) contract ContractA {
	    init() {}
	  }
	`
	contract2Address := common.MustBytesToAddress([]byte{0x2})
	contract2Location := common.AddressLocation{
		Address: contract2Address,
		Name:    "ContractB",
	}
	const contract2Code = `
	  access(all) contract ContractB {
	    init() {
	      X
	    }
	  }
	`

	handlerCalls := 0
	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case contract1Location:
				return []byte(contract1Code), nil
			case contract2Location:
				return []byte(contract2Code), nil
			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
		HandleCheckerError: func(err analysis.ParsingCheckingError, _ *sema.Checker) error {
			require.Error(t, err)
			handlerCalls++
			return nil
		},
	}

	programs, err := analysis.Load(config, contract1Location)
	require.Equal(t, 2, handlerCalls)
	require.NoError(t, err)

	var checkerError *sema.CheckerError
	require.ErrorAs(t,
		programs.Get(contract1Location).LoadError,
		&checkerError,
	)
	require.ErrorAs(t,
		programs.Get(contract2Location).LoadError,
		&checkerError,
	)

	// Validate that parent checker receives the imported program error despite it being handled
	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t,
		programs.Get(contract1Location).LoadError,
		&importedProgramErr,
	)
}

func TestStdlib(t *testing.T) {

	t.Parallel()

	scriptLocation := common.ScriptLocation{}

	const code = `
	  access(all) fun main() {
          panic("test")
      }
	`

	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case scriptLocation:
				return []byte(code), nil

			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	_, err := analysis.Load(config, scriptLocation)
	require.NoError(t, err)
}

func TestCyclicImports(t *testing.T) {

	t.Parallel()

	fooContractAddress := common.MustBytesToAddress([]byte{0x1})
	fooContractLocation := common.AddressLocation{
		Address: fooContractAddress,
		Name:    "Foo",
	}
	const fooContractCode = `
        import 0x2
        access(all) contract Foo {}
	`

	barContractAddress := common.MustBytesToAddress([]byte{0x2})
	barContractLocation := common.AddressLocation{
		Address: barContractAddress,
		Name:    "Bar",
	}
	const barContractCode = `
        import 0x1
        access(all) contract Bar {}
	`

	config := &analysis.Config{
		Mode: analysis.NeedTypes,
		ResolveAddressContractNames: func(address common.Address) ([]string, error) {
			switch address {
			case fooContractAddress:
				return []string{fooContractLocation.Name}, nil
			case barContractAddress:
				return []string{barContractLocation.Name}, nil
			default:
				return nil, fmt.Errorf(
					"import of unknown location: %s",
					address,
				)
			}
		},
		ResolveCode: func(
			location common.Location,
			importingLocation common.Location,
			importRange ast.Range,
		) ([]byte, error) {
			switch location {
			case fooContractLocation:
				return []byte(fooContractCode), nil

			case barContractLocation:
				return []byte(barContractCode), nil

			default:
				require.FailNowf(t,
					"import of unknown location",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	_, err := analysis.Load(config, fooContractLocation)
	require.Error(t, err)

	var checkerError *sema.CheckerError
	require.ErrorAs(t, err, &checkerError)

	errs := checker.RequireCheckerErrors(t, checkerError, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	var nestedCheckerErr *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &nestedCheckerErr)

	errs = checker.RequireCheckerErrors(t, nestedCheckerErr, 1)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}
