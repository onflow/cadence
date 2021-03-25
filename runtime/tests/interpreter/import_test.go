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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestInterpretVirtualImport(t *testing.T) {

	fooType := &sema.CompositeType{
		Location:   common.IdentifierLocation("Foo"),
		Identifier: "Foo",
		Kind:       common.CompositeKindContract,
	}

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

	const code = `
       import Foo

       fun test(): UInt64 {
           return Foo.bar()
       }
    `

	valueElements := sema.NewStringImportElementOrderedMap()

	valueElements.Set("Foo", sema.ImportElement{
		DeclarationKind: common.DeclarationKindStructure,
		Access:          ast.AccessPublic,
		Type:            fooType,
	})

	inter := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				interpreter.WithImportLocationHandler(
					func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

						assert.Equal(t,
							common.IdentifierLocation("Foo"),
							location,
						)

						return interpreter.VirtualImport{
							Globals: []struct {
								Name  string
								Value interpreter.Value
							}{
								{
									Name: "Foo",
									Value: &interpreter.CompositeValue{
										Location:            location,
										QualifiedIdentifier: "Foo",
										Kind:                common.CompositeKindContract,
										Fields:              interpreter.NewStringValueOrderedMap(),
										Functions: map[string]interpreter.FunctionValue{
											"bar": interpreter.NewHostFunctionValue(
												func(invocation interpreter.Invocation) interpreter.Value {
													return interpreter.NewIntValueFromInt64(42)
												},
											),
										},
									},
								},
							},
						}
					},
				),
			},
			CheckerOptions: []sema.Option{
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

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(42),
		value,
	)
}

// TestInterpretImportMultipleProgramsFromLocation demonstrates how two declarations (`a` and `b`)
// can be imported from the same location (address location `0x1`).
// The single location (address location `0x1`) is resolved to two locations (address locations `0x1.a` and `0x1.b`).
// Each requested declaration is so imported from a a separate program.
//
func TestInterpretImportMultipleProgramsFromLocation(t *testing.T) {

	t.Parallel()

	address := common.BytesToAddress([]byte{0x1})

	importedCheckerA, err := checker.ParseAndCheckWithOptions(t,
		`
          // this function *SHOULD* be imported in the importing program
          pub fun a(): Int {
              return 1
          }

          // this function should *NOT* be imported in the importing program
          pub fun b(): Int {
              return 11
          }
        `,
		checker.ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "a",
			},
		},
	)
	require.NoError(t, err)

	importedCheckerB, err := checker.ParseAndCheckWithOptions(t,
		`
          // this function *SHOULD* be imported in the importing program
          pub fun b(): Int {
              return 2
          }

          // this function should *NOT* be imported in the importing program
          pub fun a(): Int {
              return 22
          }
        `,
		checker.ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "b",
			},
		},
	)
	require.NoError(t, err)

	importingChecker, err := checker.ParseAndCheckWithOptions(t,
		`
          import a, b from 0x1

          pub fun test(): Int {
              return a() + b()
          }
        `,
		checker.ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithLocationHandler(
					func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

						require.Equal(t,
							common.AddressLocation{
								Address: address,
								Name:    "",
							},
							location,
						)

						for _, identifier := range identifiers {
							result = append(result, sema.ResolvedLocation{
								Location: common.AddressLocation{
									Address: location.(common.AddressLocation).Address,
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
						require.IsType(t, common.AddressLocation{}, location)
						addressLocation := location.(common.AddressLocation)

						assert.Equal(t, address, addressLocation.Address)

						var importedChecker *sema.Checker

						switch addressLocation.Name {
						case "a":
							importedChecker = importedCheckerA
						case "b":
							importedChecker = importedCheckerB
						default:
							t.Errorf(
								"invalid address location location name: %s",
								addressLocation.Name,
							)
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

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				require.IsType(t, common.AddressLocation{}, location)
				addressLocation := location.(common.AddressLocation)

				assert.Equal(t, address, addressLocation.Address)

				var importedChecker *sema.Checker

				switch addressLocation.Name {
				case "a":
					importedChecker = importedCheckerA
				case "b":
					importedChecker = importedCheckerB
				default:
					return nil
				}

				program := interpreter.ProgramFromChecker(importedChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewIntValueFromInt64(3),
		value,
	)
}
