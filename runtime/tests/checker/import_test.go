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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckInvalidImport(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       import "unknown"
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.UnresolvedImportError{}, errs[0])
}

func TestCheckRepeatedImport(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          pub let x = 1
          pub let y = 2
        `,
		ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)
	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import x from "imported"
           import y from "imported"
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)
}

func TestCheckRepeatedImportResolution(t *testing.T) {

	t.Parallel()

	importedAddress := common.MustBytesToAddress([]byte{0x1})

	importedCheckerX, err := ParseAndCheckWithOptions(t,
		`
          pub fun test(): Int {
              return 1
          }

          pub let x = test()
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: importedAddress,
				Name:    "x",
			},
		},
	)
	require.NoError(t, err)

	importedCheckerY, err := ParseAndCheckWithOptions(t,
		`
          pub fun test(): Int {
              return 2
          }

          pub let y = test()
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: importedAddress,
				Name:    "y",
			},
		},
	)
	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import x from 0x1
           import y from 0x1
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {
					for _, identifier := range identifiers {
						result = append(result, sema.ResolvedLocation{
							Location: common.AddressLocation{
								Address: importedAddress,
								Name:    identifier.Identifier,
							},
							Identifiers: []ast.Identifier{
								identifier,
							},
						})
					}
					return
				},
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					addressLocation := importedLocation.(common.AddressLocation)
					var importedChecker *sema.Checker
					switch addressLocation.Name {
					case "x":
						importedChecker = importedCheckerX
					case "y":
						importedChecker = importedCheckerY
					}
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidRepeatedImport(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          pub let x = 1
        `,
		ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import x from "imported"
           import x from "imported"
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckImportResolutionSplit(t *testing.T) {

	t.Parallel()

	importedAddress := common.MustBytesToAddress([]byte{0x1})

	importedCheckerX, err := ParseAndCheckWithOptions(t,
		`
          pub fun test(): Int {
              return 1
          }

          pub let x = test()
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: importedAddress,
				Name:    "x",
			},
		},
	)
	require.NoError(t, err)

	importedCheckerY, err := ParseAndCheckWithOptions(t,
		`
          pub fun test(): Int {
              return 2
          }

          pub let y = test()
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: importedAddress,
				Name:    "y",
			},
		},
	)
	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import x, y from 0x1
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {
					for _, identifier := range identifiers {
						result = append(result, sema.ResolvedLocation{
							Location: common.AddressLocation{
								Address: importedAddress,
								Name:    identifier.Identifier,
							},
							Identifiers: []ast.Identifier{
								identifier,
							},
						})
					}
					return
				},
				ImportHandler: func(_ *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					addressLocation := importedLocation.(common.AddressLocation)
					var importedChecker *sema.Checker
					switch addressLocation.Name {
					case "x":
						importedChecker = importedCheckerX
					case "y":
						importedChecker = importedCheckerY
					}
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckImportAll(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          pub fun answer(): Int {
              return 42
          }
        `,
		ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
          import "imported"

          pub let x = answer()
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidImportUnexported(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
           pub let x = 1
        `,
		ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import answer from "imported"

           pub let x = answer()
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotExportedError{}, errs[0])
}

func TestCheckImportSome(t *testing.T) {

	t.Parallel()

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          pub fun answer(): Int {
              return 42
          }

          pub let x = 1
        `,
		ParseAndCheckOptions{
			Location: utils.ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
          import answer from "imported"

          pub let x = answer()
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidImportedError(t *testing.T) {

	t.Parallel()

	_, importedErr := ParseAndCheck(t, `
	  let x: Bool = 1
	`)
	errs := RequireCheckerErrors(t, importedErr, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

	_, err := ParseAndCheckWithOptions(t,
		`
           import x from "imported"
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return nil, importedErr
				},
			},
		},
	)

	errs = RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ImportedProgramError{}, errs[0])
}

func TestCheckImportTypes(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.AllCompositeKinds {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			body := "{}"
			if compositeKind == common.CompositeKindEvent {
				body = "()"
			}

			importedChecker, err := ParseAndCheckWithOptions(t,
				fmt.Sprintf(
					`
                       pub %[1]s Test %[2]s

                       pub %[1]s interface TestInterface %[2]s
                    `,
					compositeKind.Keyword(),
					body,
				),
				ParseAndCheckOptions{
					Location: utils.ImportedLocation,
				},
			)

			require.NoError(t, err)

			var useCode string
			if compositeKind != common.CompositeKindContract {
				useCode = fmt.Sprintf(
					`pub let x: %[1]sTest %[2]s %[3]s Test%[4]s`,
					compositeKind.Annotation(),
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
				)
			}

			_, err = ParseAndCheckWithOptions(t,
				fmt.Sprintf(
					`
                      import "imported"

                      pub %[1]s TestImpl: TestInterface {}

                      %[2]s
                    `,
					compositeKind.Keyword(),
					useCode,
				),
				ParseAndCheckOptions{
					Config: &sema.Config{
						ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
			)

			switch compositeKind {
			case common.CompositeKindStructure, common.CompositeKindContract:
				require.NoError(t, err)

			case common.CompositeKindResource:
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceCreationError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}
}

func TestCheckInvalidImportCycleSelf(t *testing.T) {

	t.Parallel()

	// NOTE: only parse, don't check the imported program.
	// it will be checked by checker that is checking the importing program

	const code = `import "test"`
	importedProgram, err := parser.ParseProgram(
		nil,
		[]byte(code),
		parser.Config{},
	)

	require.NoError(t, err)

	elaborations := map[common.Location]*sema.Elaboration{}

	check := func(code string, location common.Location) error {
		_, err := ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				Location: location,
				Config: &sema.Config{
					ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {

						elaboration, ok := elaborations[importedLocation]
						if !ok {
							subChecker, err := checker.SubChecker(importedProgram, importedLocation)
							if err != nil {
								return nil, err
							}
							elaborations[importedLocation] = subChecker.Elaboration
							err = subChecker.Check()
							if err != nil {
								return nil, err
							}
							elaboration = subChecker.Elaboration
						}

						return sema.ElaborationImport{
							Elaboration: elaboration,
						}, nil
					},
				},
			},
		)
		return err
	}

	err = check(code, utils.TestLocation)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])

	importedProgramError := errs[0].(*sema.ImportedProgramError).Err

	errs = RequireCheckerErrors(t, importedProgramError, 1)

	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestCheckInvalidImportCycleTwoLocations(t *testing.T) {

	t.Parallel()

	// NOTE: only parse, don't check the imported program.
	// it will be checked by checker that is checking the importing program

	const codeEven = `
      import odd from "odd"

      pub fun even(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return odd(n - 1)
      }
    `
	programEven, err := parser.ParseProgram(nil, []byte(codeEven), parser.Config{})
	require.NoError(t, err)

	const codeOdd = `
      import even from "even"

      pub fun odd(_ n: Int): Bool {
          if n == 0 {
              return false
          }
          return even(n - 1)
      }
    `
	programOdd, err := parser.ParseProgram(nil, []byte(codeOdd), parser.Config{})
	require.NoError(t, err)

	getProgram := func(location common.Location) *ast.Program {
		switch location {
		case common.StringLocation("even"):
			return programEven
		case common.StringLocation("odd"):
			return programOdd
		}

		t.Fatalf("invalid import: %#+v", location)
		return nil
	}

	elaborations := map[common.Location]*sema.Elaboration{}

	_, err = ParseAndCheckWithOptions(t,
		codeEven,
		ParseAndCheckOptions{
			Location: common.StringLocation("even"),
			Config: &sema.Config{
				ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					importedProgram := getProgram(importedLocation)

					elaboration, ok := elaborations[importedLocation]
					if !ok {
						subChecker, err := checker.SubChecker(importedProgram, importedLocation)
						if err != nil {
							return nil, err
						}
						elaborations[importedLocation] = subChecker.Elaboration
						err = subChecker.Check()
						if err != nil {
							return nil, err
						}
						elaboration = subChecker.Elaboration
					}

					return sema.ElaborationImport{
						Elaboration: elaboration,
					}, nil
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 2)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])

	importedProgramError := errs[0].(*sema.ImportedProgramError).Err

	errs = RequireCheckerErrors(t, importedProgramError, 2)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])
	require.IsType(t, &sema.NotDeclaredError{}, errs[1])

	importedProgramError = errs[0].(*sema.ImportedProgramError).Err

	errs = RequireCheckerErrors(t, importedProgramError, 2)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
	require.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckImportVirtual(t *testing.T) {

	t.Parallel()

	const code = `
       import Foo

       fun test(): UInt64 {
           return Foo.bar()
       }
    `

	fooType := &sema.CompositeType{
		Location:   common.IdentifierLocation("Foo"),
		Identifier: "Foo",
		Kind:       common.CompositeKindStructure,
	}

	fooType.Fields = []string{"bar"}

	fooType.Members = &sema.StringMemberOrderedMap{}
	fooType.Members.Set(
		"bar",
		sema.NewUnmeteredPublicFunctionMember(
			fooType,
			"bar",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.UInt64TypeAnnotation,
			},
			"",
		))

	valueElements := &sema.StringImportElementOrderedMap{}

	valueElements.Set("Foo", sema.ImportElement{
		DeclarationKind: common.DeclarationKindStructure,
		Access:          ast.AccessPublic,
		Type:            fooType,
	})

	_, err := ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
					return sema.VirtualImport{
						ValueElements: valueElements,
					}, nil
				},
			},
		},
	)

	require.NoError(t, err)
}
