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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretVirtualImport(t *testing.T) {

	t.Parallel()

	fooType := &sema.CompositeType{
		Location:   common.IdentifierLocation("Foo"),
		Identifier: "Foo",
		Kind:       common.CompositeKindContract,
	}

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
		),
	)

	const code = `
       import Foo

       fun test(): UInt64 {
           return Foo.bar()
       }
    `

	valueElements := &sema.StringImportElementOrderedMap{}

	valueElements.Set(
		"Foo",
		sema.ImportElement{
			DeclarationKind: common.DeclarationKindStructure,
			Access:          sema.UnauthorizedAccess,
			Type:            fooType,
		},
	)

	// NOTE: virtual imports are not supported by the compiler/VM
	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
						return sema.VirtualImport{
							ValueElements: valueElements,
						}, nil
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

					assert.Equal(t,
						common.IdentifierLocation("Foo"),
						location,
					)

					value := interpreter.NewCompositeValue(
						inter,
						interpreter.EmptyLocationRange,
						location,
						"Foo",
						common.CompositeKindContract,
						nil,
						common.ZeroAddress,
					)
					value.Functions = orderedmap.New[interpreter.FunctionOrderedMap](1)
					value.Functions.Set(
						"bar",
						interpreter.NewStaticHostFunctionValue(
							inter,
							&sema.FunctionType{
								ReturnTypeAnnotation: sema.UIntTypeAnnotation,
							},
							func(invocation interpreter.Invocation) interpreter.Value {
								return interpreter.NewUnmeteredUInt64Value(42)
							},
						),
					)

					elaboration := sema.NewElaboration(nil)
					elaboration.SetCompositeType(
						fooType.ID(),
						fooType,
					)

					return interpreter.VirtualImport{
						Globals: []interpreter.VirtualImportGlobal{
							{
								Name:  "Foo",
								Value: value,
							},
						},
						Elaboration: elaboration,
					}
				},
			},
		},
	)
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredUInt64Value(42),
		value,
	)
}

// TestInterpretImportMultipleProgramsFromLocation demonstrates how two declarations (`a` and `b`)
// can be imported from the same location (address location `0x1`).
// The single location (address location `0x1`) is resolved to two locations (address locations `0x1.a` and `0x1.b`).
// Each requested declaration is so imported from a separate program.
//
// NOTE: Testing this "synthetic" scenario in compiler/VM is not possible,
// Because the compiler/vm's linking logic is not configurable.
// (i.e: to link one function form one program, and the other function from the other program).
func TestInterpretImportMultipleProgramsFromLocation(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	importedCheckerA, err := ParseAndCheckWithOptions(t,
		`
          // this function *SHOULD* be imported in the importing program
          access(all) fun a(): Int {
              return 1
          }

          // this function should *NOT* be imported in the importing program
          access(all) fun b(): Int {
              return 11
          }
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "a",
			},
		},
	)
	require.NoError(t, err)

	importedCheckerB, err := ParseAndCheckWithOptions(t,
		`
          // this function *SHOULD* be imported in the importing program
          access(all) fun b(): Int {
              return 2
          }

          // this function should *NOT* be imported in the importing program
          access(all) fun a(): Int {
              return 22
          }
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "b",
			},
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import a, b from 0x1

          access(all) fun test(): Int {
              return a() + b()
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

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
				ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					require.IsType(t, common.AddressLocation{}, importedLocation)
					addressLocation := importedLocation.(common.AddressLocation)

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
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			Storage: storage,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
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
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretResourceConstructionThroughIndirectImport(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          resource R {}
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
			},
		},
	)
	require.NoError(t, err)

	importingChecker, err := ParseAndCheckWithOptions(t,
		`
          import R from 0x1

          fun test(createR: fun(): @R) {
              let r <- createR()
              destroy r
          }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					require.IsType(t, common.AddressLocation{}, importedLocation)
					addressLocation := importedLocation.(common.AddressLocation)

					assert.Equal(t, address, addressLocation.Address)

					return sema.ElaborationImport{
						Elaboration: importedChecker.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	var subInterpreter *interpreter.Interpreter

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(importingChecker),
		importingChecker.Location,
		&interpreter.Config{
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				require.IsType(t, common.AddressLocation{}, location)
				addressLocation := location.(common.AddressLocation)

				assert.Equal(t, address, addressLocation.Address)

				program := interpreter.ProgramFromChecker(importedChecker)
				var err error
				subInterpreter, err = inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}

				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}
			},
			UUIDHandler: func() (uint64, error) {
				return 0, nil
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	rConstructor := subInterpreter.Globals.Get("R").GetValue(inter)

	_, err = inter.Invoke("test", rConstructor)
	RequireError(t, err)

	var resourceConstructionError *interpreter.ResourceConstructionError
	require.ErrorAs(t, err, &resourceConstructionError)

	assert.Equal(t,
		RequireGlobalType(t, subInterpreter, "R"),
		resourceConstructionError.CompositeType,
	)
}

// TestInterpretImportWithAlias shows importing two funs of the same name from different addresses
func TestInterpretImportWithAlias(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	importedCheckerA, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun a(): Int {
              return 1
          }
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "",
			},
		},
	)
	require.NoError(t, err)

	importedCheckerB, err := ParseAndCheckWithOptions(t,
		`
          access(all) fun a(): Int {
              return 2
          }
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address2,
				Name:    "",
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()
	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          import a as a1 from 0x1
          import a as a2 from 0x2

          access(all) fun test(): Int {
              return a1() + a2()
          }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					require.IsType(t, common.AddressLocation{}, location)
					addressLocation := location.(common.AddressLocation)

					var importedChecker *sema.Checker

					switch addressLocation.Address {
					case address:
						importedChecker = importedCheckerA
					case address2:
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
			},
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

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
					ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						require.IsType(t, common.AddressLocation{}, importedLocation)
						addressLocation := importedLocation.(common.AddressLocation)

						var importedChecker *sema.Checker

						switch addressLocation.Address {
						case address:
							importedChecker = importedCheckerA
						case address2:
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
				},
			},
		},
	)
	require.NoError(t, err)
	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(3),
		value,
	)
}

func TestInterpretImportAliasGetType(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	importedChecker, err := ParseAndCheckWithOptions(t,
		`
          access(all) struct Foo {}
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: address,
				Name:    "",
			},
		},
	)
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()
	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          import Foo as Bar from 0x1

          access(all) fun test(): String {
              var bar: Bar = Bar()
              return bar.getType().identifier
          }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				Storage: storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
					program := interpreter.ProgramFromChecker(importedChecker)
					subInterpreter, err := inter.NewSubInterpreter(program, location)
					if err != nil {
						panic(err)
					}

					return interpreter.InterpreterImport{
						Interpreter: subInterpreter,
					}
				},
			},
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

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
					ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("A.0000000000000001.Foo"),
		value,
	)
}

// TestInterpretImportTypeEquality shows aliasing one func twice and using it interchangeably
func TestInterpretImportTypeEquality(t *testing.T) {

	t.Parallel()

	// 0x1 defines Foo
	addressA := common.MustBytesToAddress([]byte{0x1})
	importedCheckerA, err := ParseAndCheckWithOptions(t,
		`
          struct Foo {}
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: addressA,
				Name:    "",
			},
		},
	)
	require.NoError(t, err)

	// 0x2 imports Foo as Bar from 0x1,
	// and defines bar to return Bar (i.e., Foo)
	addressB := common.MustBytesToAddress([]byte{0x2})
	importedCheckerB, err := ParseAndCheckWithOptions(t,
		`
          import Foo as Bar from 0x1

          fun bar(): Bar {
              return Bar()
          }
        `,
		ParseAndCheckOptions{
			Location: common.AddressLocation{
				Address: addressB,
				Name:    "",
			},
			CheckerConfig: &sema.Config{
				ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {
					assert.Equal(t, common.AddressLocation{
						Address: addressA,
						Name:    "",
					}, importedLocation)

					return sema.ElaborationImport{
						Elaboration: importedCheckerA.Elaboration,
					}, nil
				},
			},
		},
	)
	require.NoError(t, err)

	// The main program imports Foo as Baz from 0x1,
	// and imports bar from 0x2,
	// and uses bar to return Baz (i.e., Foo)

	storage := newUnmeteredInMemoryStorage()
	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          import Foo as Baz from 0x1
          import bar from 0x2

          fun test(): Baz {
              return bar()
          }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				Storage:              storage,
				ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

					var checker *sema.Checker

					switch location.(common.AddressLocation).Address {
					case addressA:
						checker = importedCheckerA
					case addressB:
						checker = importedCheckerB
					default:
						assert.Fail(t, "invalid import location", location)
						return nil
					}

					program := interpreter.ProgramFromChecker(checker)
					subInterpreter, err := inter.NewSubInterpreter(program, location)
					if err != nil {
						panic(err)
					}

					return interpreter.InterpreterImport{
						Interpreter: subInterpreter,
					}
				},
			},
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					LocationHandler: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

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
					ImportHandler: func(checker *sema.Checker, importedLocation common.Location, _ ast.Range) (sema.Import, error) {

						var importedChecker *sema.Checker

						switch importedLocation.(common.AddressLocation).Address {
						case addressA:
							importedChecker = importedCheckerA
						case addressB:
							importedChecker = importedCheckerB
						default:
							assert.Fail(t, "invalid import location", importedLocation)
							return nil, nil
						}

						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		},
	)
	require.NoError(t, err)
	err = inter.Interpret()
	require.NoError(t, err)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t, &interpreter.CompositeValue{}, value)
	compositeValue := value.(*interpreter.CompositeValue)

	assert.Equal(t,
		common.TypeID("A.0000000000000001.Foo"),
		compositeValue.TypeID(),
	)

}
