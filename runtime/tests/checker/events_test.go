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

	_, err := ParseAndCheck(t, `
      contract Test {
          event AccountEvent(account: &Account)
      }
    `)
	require.NoError(t, err)
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

	t.Run("declare does not create a type requirement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo()
			}
			contract Impl: Test {}
        `)
		require.NoError(t, err)
	})

	t.Run("impl with different type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface Test {
				event Foo(y: Int)
				fun emitEvent() {
					emit Foo(y: 3)
				}
			}
			contract Impl: Test {
				event Foo(x: String)
				fun emitEvent() {
					emit Foo(x: "")
				}
			}
        `)
		require.NoError(t, err)
	})

}

func TestCheckDefaultEventDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed()
			}
        `)
		require.NoError(t, err)

		variable, exists := checker.Elaboration.GetGlobalType("R")
		require.True(t, exists)

		require.IsType(t, variable.Type, &sema.CompositeType{})
		require.Equal(t, variable.Type.(*sema.CompositeType).DefaultDestroyEvent.Identifier, "ResourceDestroyed")
	})

	t.Run("allowed in resource interface", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
			resource interface R {
				event ResourceDestroyed()
			}
        `)
		require.NoError(t, err)

		variable, exists := checker.Elaboration.GetGlobalType("R")
		require.True(t, exists)

		require.IsType(t, variable.Type, &sema.InterfaceType{})
		require.Equal(t, variable.Type.(*sema.InterfaceType).DefaultDestroyEvent.Identifier, "ResourceDestroyed")
	})

	t.Run("fail in struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			struct R {
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyEventInNonResourceError{}, errs[0])
	})

	t.Run("fail in struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			struct interface R {
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyEventInNonResourceError{}, errs[0])
	})

	t.Run("fail in contract", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract R {
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyEventInNonResourceError{}, errs[0])
	})

	t.Run("fail in contract interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			contract interface R {
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyEventInNonResourceError{}, errs[0])
	})

	t.Run("allowed in resource attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			attachment A for AnyResource {
				event ResourceDestroyed()
			}
        `)
		require.NoError(t, err)
	})

	t.Run("not allowed in struct attachment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			attachment A for AnyStruct {
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyEventInNonResourceError{}, errs[0])
	})

	t.Run("nested declarations after first disallowed", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed()
				event OtherEvent()
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	})

	t.Run("cannot declare two default events", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed()
				event ResourceDestroyed()
			}
        `)
		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
		assert.IsType(t, &sema.RedeclarationError{}, errs[1])
		assert.IsType(t, &sema.RedeclarationError{}, errs[2])
		assert.IsType(t, &sema.RedeclarationError{}, errs[3])
	})
}

func TestCheckDefaultEventParamChecking(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: String = "foo")
			}
        `)
		require.NoError(t, err)
	})

	t.Run("3 param", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: String = "foo", id: UInt16? = 4, condition: Bool = true)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("type error", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: Int = "foo")
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let field : String
				event ResourceDestroyed(name: String = self.field)

				init() {
					self.field = ""
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("address", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let field : Address
				event ResourceDestroyed(name: Address? = self.field)

				init() {
					self.field = 0x1
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: Address? = nil)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("address expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: Address = 0x1)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("path expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: PublicPath = /public/foo)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("float", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: UFix64 = 0.0034)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("field type mismatch", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: String = self.field)

				let field : Int

				init() {
					self.field = 3
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("self", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: @R = self)
			}
        `)
		errs := RequireCheckerErrors(t, err, 4)

		assert.IsType(t, &sema.DefaultDestroyInvalidParameterError{}, errs[0])
		assert.IsType(t, &sema.ResourceLossError{}, errs[1])
		assert.IsType(t, &sema.InvalidEventParameterTypeError{}, errs[2])
		assert.IsType(t, &sema.InvalidResourceFieldError{}, errs[3])
	})

	t.Run("array field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let field : [Int]
				event ResourceDestroyed(name: [Int] = self.field)

				init() {
					self.field = [3]
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyInvalidParameterError{}, errs[0])
	})

	t.Run("function call", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: Int  = self.fn())

				fun fn(): Int {
					return 3
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("external field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			let r2 <- create R2()
			let ref = &r2 as &R2
			
			resource R {
				event ResourceDestroyed(name: UFix64 = ref.field)
			}

			resource R2 {
				let field : UFix64 
				init() {
					self.field = 0.0034
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("double nested field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let s: S 
				event ResourceDestroyed(name: UFix64 = self.s.field)
				init() {
					self.s = S()
				}
			}

			struct S {
				let field : UFix64 
				init() {
					self.field = 0.0034
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("function call member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun getS(): S {
				return S()
			}

			resource R {
				event ResourceDestroyed(name: UFix64 = getS().field)
			}

			struct S {
				let field : UFix64 
				init() {
					self.field = 0.0034
				}
			}
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("method call member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				fun getS(): S {
					return S()
				}	
				event ResourceDestroyed(name: UFix64 = self.getS().field)
			}

			struct S {
				let field : UFix64 
				init() {
					self.field = 0.0034
				}
			}
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("array index expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				var arr : [String] 
				event ResourceDestroyed(name: String? = self.arr[0])

				init() {
					self.arr = []
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("dict index expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let dict : {Int: String} 
				event ResourceDestroyed(name: String? = self.dict[0])

				init() {
					self.dict = {}
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("function call dict index expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				event ResourceDestroyed(name: String? = self.get()[0])
				fun get(): {Int: String} {
					return {}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("function call dict indexed expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			resource R {
				let dict : {Int: String} 
				event ResourceDestroyed(name: String? = self.dict[0+1])

				init() {
					self.dict = {}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("external var expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			var index: Int = 3
			resource R {
				let dict : {Int: String} 
				event ResourceDestroyed(name: String? = self.dict[index])

				init() {
					self.dict = {}
				}
			}
        `)
		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.DefaultDestroyInvalidArgumentError{}, errs[0])
	})

	t.Run("attachment index expression", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			attachment A for R {
				let name: String
				init() {
					self.name = "foo"
				}
			}

			resource R {
				event ResourceDestroyed(name: String? = self[A]?.name)
			}
        `)
		require.NoError(t, err)
	})

	t.Run("attachment with base", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			attachment A for R {
				event ResourceDestroyed(name: Int  = base.field)
			}

			resource R {
				let field : Int

				init() {
					self.field = 3
				}
			}
        `)
		require.NoError(t, err)
	})

	t.Run("field name conflict", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
			attachment A for R {
				event ResourceDestroyed(name: Int = self.self, x: String = base.base)
				let self: Int
				let base: String
				init() {
					self.base = "foo"
					self.self = 3
				}
			}

			resource R {
				let base: String
				init() {
					self.base = "foo"
				}
				event ResourceDestroyed(name: String? = self[A]?.base, x: Int? = self[A]?.self)
			}
        `)
		require.NoError(t, err)
	})
}
