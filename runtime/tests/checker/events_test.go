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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/tests/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckEventDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("invalid: non-primitive type composite", func(t *testing.T) {

		t.Parallel()

		test := func(compositeKind common.CompositeKind) {

			t.Run(compositeKind.Name(), func(t *testing.T) {

				t.Parallel()

				var baseType string
				if compositeKind == common.CompositeKindAttachment {
					baseType = "for AnyStruct"
				}

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          %[1]s Token %[3]s {
                            let id: String

                            init(id: String) {
                              self.id = id
                            }
                          }

                          event Transfer(token: %[2]sToken)
                        `,
						compositeKind.Keyword(),
						compositeKind.Annotation(),
						baseType,
					),
				)

				switch compositeKind {
				case common.CompositeKindResource:
					errs := RequireCheckerErrors(t, err, 3)

					assert.IsType(t, &sema.ResourceLossError{}, errs[0])
					assert.IsType(t, &sema.InvalidEventParameterTypeError{}, errs[1])
					assert.IsType(t, &sema.InvalidResourceFieldError{}, errs[2])

				case common.CompositeKindStructure:
					require.NoError(t, err)

				case common.CompositeKindAttachment:
					errs := RequireCheckerErrors(t, err, 2)

					assert.IsType(t, &sema.InvalidEventParameterTypeError{}, errs[1])
					assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])

				default:
					panic(errors.NewUnreachableError())
				}
			})
		}

		for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {
			if compositeKind == common.CompositeKindContract {
				continue
			}

			test(compositeKind)
		}
	})

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		validTypes := common.Concat(
			sema.AllNumberTypes,
			[]sema.Type{
				sema.StringType,
				sema.CharacterType,
				sema.BoolType,
				sema.TheAddressType,
				sema.MetaType,
				sema.PathType,
				sema.StoragePathType,
				sema.PublicPathType,
				sema.PrivatePathType,
				sema.CapabilityPathType,
			},
		)

		tests := validTypes[:]

		for _, ty := range validTypes {
			tests = append(tests,
				&sema.OptionalType{Type: ty},
				&sema.VariableSizedType{Type: ty},
				&sema.ConstantSizedType{Type: ty},
				&sema.DictionaryType{
					KeyType:   sema.StringType,
					ValueType: ty,
				},
			)

			if sema.IsValidDictionaryKeyType(ty) {
				tests = append(tests,
					&sema.DictionaryType{
						KeyType:   ty,
						ValueType: sema.StringType,
					},
				)
			}
		}

		for _, ty := range tests {

			t.Run(ty.String(), func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          event Transfer(_ value: %s)
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		}
	})

	t.Run("recursive", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          event E(recursive: Recursive)

          struct Recursive {
              let children: [Recursive]
              init() {
                  self.children = []
              }
          }
		`)

		require.NoError(t, err)
	})

	t.Run("RedeclaredEvent", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            event Transfer(to: Int, from: Int)
            event Transfer(to: Int)
		`)

		// NOTE: two redeclaration errors: one for type, one for function

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
		assert.IsType(t, &sema.RedeclarationError{}, errs[1])
	})

}

func TestCheckEmitEvent(t *testing.T) {

	t.Parallel()

	t.Run("ValidEvent", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            event Transfer(to: Int, from: Int)

            fun test() {
                emit Transfer(to: 1, from: 2)
            }
        `)

		require.NoError(t, err)
	})

	t.Run("MissingEmitStatement", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            event Transfer(to: Int, from: Int)

            fun test() {
                Transfer(to: 1, from: 2)
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidEventUsageError{}, errs[0])
	})

	t.Run("EmitNonEvent", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun notAnEvent(): Int { return 1 }

            fun test() {
                emit notAnEvent()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.EmitNonEventError{}, errs[0])
	})

	t.Run("EmitNotDeclared", func(t *testing.T) {
		_, err := ParseAndCheck(t, `
            fun test() {
              emit notAnEvent()
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("EmitImported", func(t *testing.T) {

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
              access(all) event Transfer(to: Int, from: Int)
            `,
			ParseAndCheckOptions{
				Location: utils.ImportedLocation,
			},
		)
		require.NoError(t, err)

		_, err = ParseAndCheckWithOptions(t, `
              import Transfer from "imported"

              access(all) fun test() {
                  emit Transfer(to: 1, from: 2)
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

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.EmitImportedEventError{}, errs[0])
	})
}

func TestCheckAccountEventParameter(t *testing.T) {

	t.Parallel()

	t.Run("AuthAccount", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          contract Test {
              event Account(account: AuthAccount)
          }
        `)
		require.NoError(t, err)
	})

	t.Run("PublicAccount", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          contract Test {
              event Account(account: PublicAccount)
          }
        `)
		require.NoError(t, err)
	})

}

func TestCheckDeclareEventInInterface(t *testing.T) {

	t.Parallel()

	t.Run("declare", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo()
			}
        `)
		require.NoError(t, err)
	})

	t.Run("declare and emit", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo(x: String)
				fun foo() {
					emit Foo(x: "")
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("declare and emit nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo(x: String)

				resource interface R {
					fun foo() {
						emit Foo(x: "")
					}
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("emit non-declared event", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				fun foo() {
					pre {
						emit Foo()
					}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("declare and emit type mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo(x: String)
				fun foo() {
					pre {
						emit Foo(x: 3)
					}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("declare and emit qualified", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			access(all) contract interface Test {
				access(all) event Foo()
			}
			access(all) contract C {
				access(all) resource R {
					access(all) fun emitEvent() {
						emit Test.Foo()
					}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("declare and emit in pre-condition", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo()
				fun foo() {
					pre {
						emit Foo()
					}
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("declare and emit in post-condition", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo()
				fun foo() {
					post {
						emit Foo()
					}
				}
			}
        `)
		require.NoError(t, err)
	})

}
