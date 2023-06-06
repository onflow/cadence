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

package analysis_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/tests/checker"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
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
				require.FailNow(t,
					"import of unknown location: %s",
					"location: %s",
					location,
				)
				return nil, nil
			}
		},
	}

	programs, err := analysis.Load(config, txLocation)
	require.NoError(t, err)

	require.NotNil(t, programs[txLocation])
	require.NotNil(t, programs[contractLocation])

	type locationRange struct {
		location common.Location
		ast.Range
	}

	var locationRanges []locationRange

	for _, program := range programs {
		require.NotNil(t, program.Program)
		require.NotNil(t, program.Elaboration)

		// Run a simple analysis: Detect unnecessary cast

		var detected bool

		ast.Inspect(program.Program, func(element ast.Element) bool {
			castingExpression, ok := element.(*ast.CastingExpression)
			if !ok {
				return true
			}

			types := program.Elaboration.CastingExpressionTypes(castingExpression)
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
				require.FailNow(t,
					"import of unknown location: %s",
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
				require.FailNow(t,
					"import of unknown location: %s",
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
				require.FailNow(t,
					"import of unknown location: %s",
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
				require.FailNow(t,
					"import of unknown location: %s",
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
