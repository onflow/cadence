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

package checker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestCheckPredeclaredValues(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		valueDeclaration := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
		}

		_, err := ParseAndCheckWithOptions(t,
			`
            pub fun test() {
                foo()
            }
        `,
			ParseAndCheckOptions{
				Options: []sema.Option{
					sema.WithPredeclaredValues(
						[]sema.ValueDeclaration{
							valueDeclaration,
						},
					),
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("predicated", func(t *testing.T) {

		// Declare four programs.
		// Program 0x1 imports 0x2, 0x3, and 0x4.
		// All programs attempt to call a function 'foo'.
		// Only predeclare a function 'foo' for 0x2 and 0x4.
		// Both functions have the same name, but different types.

		location1 := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x1}),
		}

		location2 := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x2}),
		}

		location3 := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x3}),
		}

		location4 := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{0x4}),
		}

		valueDeclaration1 := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
			Available: func(location common.Location) bool {
				addressLocation, ok := location.(common.AddressLocation)
				return ok && addressLocation == location2
			},
		}

		valueDeclaration2 := stdlib.StandardLibraryFunction{
			Name: "foo",
			Type: &sema.FunctionType{
				Parameters: []*sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "n",
						TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
					},
				},
				ReturnTypeAnnotation: &sema.TypeAnnotation{
					Type: sema.VoidType,
				},
			},
			Available: func(location common.Location) bool {
				addressLocation, ok := location.(common.AddressLocation)
				return ok && addressLocation == location4
			},
		}

		predeclaredValuesOption := sema.WithPredeclaredValues(
			[]sema.ValueDeclaration{
				valueDeclaration1,
				valueDeclaration2,
			},
		)

		checker2, err2 := ParseAndCheckWithOptions(t,
			`let x = foo()`,
			ParseAndCheckOptions{
				Location: location2,
				Options: []sema.Option{
					predeclaredValuesOption,
				},
			},
		)

		checker3, err3 := ParseAndCheckWithOptions(t,
			`let y = foo()`,
			ParseAndCheckOptions{
				Location: location3,
				Options: []sema.Option{
					predeclaredValuesOption,
				},
			},
		)

		checker4, err4 := ParseAndCheckWithOptions(t,
			`let z = foo(1)`,
			ParseAndCheckOptions{
				Location: location4,
				Options: []sema.Option{
					predeclaredValuesOption,
				},
			},
		)

		getChecker := func(location common.Location) (*sema.Checker, error) {
			switch location {
			case location2:
				return checker2, err2
			case location3:
				return checker3, err3
			case location4:
				return checker4, err4
			default:
				t.Fatal("invalid location", location)
				return nil, nil
			}
		}
		_, err := ParseAndCheckWithOptions(t,
			`
              import 0x2
              import 0x3
              import 0x4

              fun main() {
                  foo()
              }
            `,
			ParseAndCheckOptions{
				Location: location1,
				Options: []sema.Option{
					predeclaredValuesOption,
					sema.WithImportHandler(
						func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {

							importedChecker, importErr := getChecker(importedLocation)
							if importErr != nil {
								return nil, importErr
							}

							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					),
				},
			},
		)

		errs := ExpectCheckerErrors(t, err, 2)

		// The illegal use of 'foo' in 0x3 should be reported

		var importedProgramError *sema.ImportedProgramError
		require.ErrorAs(t, errs[0], &importedProgramError)
		require.Equal(t, location3, importedProgramError.Location)
		importedErrs := ExpectCheckerErrors(t, importedProgramError.Err, 1)
		require.IsType(t, &sema.NotDeclaredError{}, importedErrs[0])

		// The illegal use of 'foo' in 0x1 should be reported

		require.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})
}
