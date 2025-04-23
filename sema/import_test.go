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

package sema_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
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
          access(all) let x = 1
          access(all) let y = 2
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
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
          access(all) fun test(): Int {
              return 1
          }

          access(all) let x = test()
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
          access(all) fun test(): Int {
              return 2
          }

          access(all) let y = test()
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
          access(all) let x = 1
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
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
          access(all) fun test(): Int {
              return 1
          }

          access(all) let x = test()
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
          access(all) fun test(): Int {
              return 2
          }

          access(all) let y = test()
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
          access(all) fun answer(): Int {
              return 42
          }
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
          import "imported"

          access(all) let x = answer()
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
           access(all) let x = 1
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
           import answer from "imported"

           access(all) let x = answer()
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
          access(all) fun answer(): Int {
              return 42
          }

          access(all) let x = 1
        `,
		ParseAndCheckOptions{
			Location: ImportedLocation,
		},
	)

	require.NoError(t, err)

	_, err = ParseAndCheckWithOptions(t,
		`
          import answer from "imported"

          access(all) let x = answer()
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
                       access(all) %[1]s Test %[2]s

                       access(all) %[1]s interface TestInterface %[2]s
                    `,
					compositeKind.Keyword(),
					body,
				),
				ParseAndCheckOptions{
					Location: ImportedLocation,
				},
			)

			require.NoError(t, err)

			var useCode string
			if compositeKind != common.CompositeKindContract {
				useCode = fmt.Sprintf(
					`access(all) let x: %[1]sTest %[2]s %[3]s Test%[4]s`,
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

                      access(all) %[1]s TestImpl: TestInterface {}

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

	err = check(code, TestLocation)

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

      access(all) fun even(_ n: Int): Bool {
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

      access(all) fun odd(_ n: Int): Bool {
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
		Access:          sema.UnauthorizedAccess,
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

func TestCheckImportContract(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
            access(all) contract Foo {
                access(all) let x: [Int]

                access(all) fun answer(): Int {
                    return 42
                }

                access(all) struct Bar {}

                init() {
                    self.x = []
                }
            }`,
			ParseAndCheckOptions{
				Location: ImportedLocation,
			},
		)

		require.NoError(t, err)

		_, err = ParseAndCheckWithOptions(t,
			`
            import Foo from "imported"

            access(all) fun main() {
                var foo: &Foo = Foo
                var x: &[Int] = Foo.x
                var bar: Foo.Bar = Foo.Bar()
            }
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
	})

	t.Run("invalid", func(t *testing.T) {

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
            access(all) contract Foo {
                access(all) let x: [Int]

                access(all) fun answer(): Int {
                    return 42
                }

                access(all) struct Bar {}

                init() {
                    self.x = []
                }
            }`,
			ParseAndCheckOptions{
				Location: ImportedLocation,
			},
		)

		require.NoError(t, err)

		_, err = ParseAndCheckWithOptions(t,
			`
            import Foo from "imported"

            access(all) fun main() {
                Foo.x[0] = 3
                Foo.x.append(4)
            }
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

		errs := RequireCheckerErrors(t, err, 2)

		assignmentError := &sema.UnauthorizedReferenceAssignmentError{}
		assert.ErrorAs(t, errs[0], &assignmentError)

		accessError := &sema.InvalidAccessError{}
		assert.ErrorAs(t, errs[1], &accessError)
	})

}
