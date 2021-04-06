/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckInvalidImport(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       import "unknown"
    `)

	errs := ExpectCheckerErrors(t, err, 1)

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
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)
	require.NoError(t, err)
}

func TestCheckRepeatedImportResolution(t *testing.T) {

	t.Parallel()

	importedAddress := common.BytesToAddress([]byte{0x1})

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
			Options: []sema.Option{
				sema.WithLocationHandler(
					func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {
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
				),
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						addressLocation := location.(common.AddressLocation)
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
				),
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
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckImportResolutionSplit(t *testing.T) {

	t.Parallel()

	importedAddress := common.BytesToAddress([]byte{0x1})

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
			Options: []sema.Option{
				sema.WithLocationHandler(
					func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {
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
				),
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						addressLocation := location.(common.AddressLocation)
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
				),
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
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
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
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 1)

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
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				),
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
	require.Error(t, importedErr)

	_, err := ParseAndCheckWithOptions(t,
		`
           import x from "imported"
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return nil, importedErr
					},
				),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 1)

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
					Options: []sema.Option{
						sema.WithImportHandler(
							func(checker *sema.Checker, location common.Location) (sema.Import, error) {
								return sema.ElaborationImport{
									Elaboration: importedChecker.Elaboration,
								}, nil
							},
						),
					},
				},
			)

			switch compositeKind {
			case common.CompositeKindStructure, common.CompositeKindContract:
				require.NoError(t, err)

			case common.CompositeKindResource:
				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidResourceCreationError{}, errs[0])

			default:
				panic(errors.NewUnreachableError())
			}
		})
	}
}

func TestCheckInvalidImportCycleSelf(t *testing.T) {

	t.Parallel()

	// NOTE: only parse, don't check imported program.
	// will be checked by checker checking importing program

	const code = `import "test"`
	importedProgram, err := parser2.ParseProgram(code)

	require.NoError(t, err)

	elaborations := map[common.LocationID]*sema.Elaboration{}

	check := func(code string, location common.Location) error {
		_, err := ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				Location: location,
				Options: []sema.Option{
					sema.WithImportHandler(
						func(checker *sema.Checker, location common.Location) (sema.Import, error) {

							elaboration, ok := elaborations[location.ID()]
							if !ok {
								subChecker, err := checker.SubChecker(importedProgram, location)
								if err != nil {
									return nil, err
								}
								elaborations[location.ID()] = subChecker.Elaboration
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
					),
				},
			},
		)
		return err
	}

	err = check(code, utils.TestLocation)

	errs := ExpectCheckerErrors(t, err, 1)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])
	childErrs := errs[0].(*sema.ImportedProgramError).ChildErrors()

	require.Len(t, childErrs, 1)
	assert.IsType(t, &sema.CyclicImportsError{}, childErrs[0])
}

func TestCheckInvalidImportCycleTwoLocations(t *testing.T) {

	t.Parallel()

	// NOTE: only parse, don't check imported program.
	// will be checked by checker checking importing program

	const codeEven = `
      import odd from "odd"

      pub fun even(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return odd(n - 1)
      }
    `
	programEven, err := parser2.ParseProgram(codeEven)
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
	programOdd, err := parser2.ParseProgram(codeOdd)
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

	elaborations := map[common.LocationID]*sema.Elaboration{}

	_, err = ParseAndCheckWithOptions(t,
		codeEven,
		ParseAndCheckOptions{
			Location: common.StringLocation("even"),
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						importedProgram := getProgram(location)

						elaboration, ok := elaborations[location.ID()]
						if !ok {
							subChecker, err := checker.SubChecker(importedProgram, location)
							if err != nil {
								return nil, err
							}
							elaborations[location.ID()] = subChecker.Elaboration
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
				),
			},
		},
	)

	errs := ExpectCheckerErrors(t, err, 2)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])

	importedProgramError := errs[0].(*sema.ImportedProgramError).Err

	errs = ExpectCheckerErrors(t, importedProgramError, 2)

	require.IsType(t, &sema.ImportedProgramError{}, errs[0])
	require.IsType(t, &sema.NotDeclaredError{}, errs[1])

	importedProgramError = errs[0].(*sema.ImportedProgramError).Err

	errs = ExpectCheckerErrors(t, importedProgramError, 2)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
	require.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckImportVirtual(t *testing.T) {

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

	fooType.Members = sema.NewStringMemberOrderedMap()
	fooType.Members.Set(
		"bar",
		sema.NewPublicFunctionMember(
			fooType,
			"bar",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.UInt64Type),
			},
			"",
		))

	valueElements := sema.NewStringImportElementOrderedMap()

	valueElements.Set("Foo", sema.ImportElement{
		DeclarationKind: common.DeclarationKindStructure,
		Access:          ast.AccessPublic,
		Type:            fooType,
	})

	_, err := ParseAndCheckWithOptions(t,
		code,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithImportHandler(
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						return sema.VirtualImport{
							ValueElements: valueElements,
						}, nil
					},
				),
			},
		},
	)

	require.NoError(t, err)
}
