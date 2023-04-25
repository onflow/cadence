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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckConstantAndVariableDeclarations(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let x = 1
        var y = 1
    `)

	require.NoError(t, err)

	assert.IsType(t,
		sema.IntType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)

	assert.IsType(t,
		sema.IntType,
		RequireGlobalValue(t, checker.Elaboration, "y"),
	)
}

func TestCheckInvalidGlobalConstantRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun x() {}

        let y = true
        let y = false
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidGlobalFunctionRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        let x = true

        fun y() {}
        fun y() {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidLocalRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test() {
            let x = true
            let x = false
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidLocalFunctionRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        fun test() {
            let x = true

            fun y() {}
            fun y() {}
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidUnknownDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun test() {
           return x
       }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidUnknownDeclarationInGlobal(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       let x = y
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidUnknownDeclarationInGlobalAndUnknownType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       let x: X = y
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.Equal(t,
		"X",
		errs[0].(*sema.NotDeclaredError).Name,
	)
	assert.Equal(t,
		common.DeclarationKindType,
		errs[0].(*sema.NotDeclaredError).ExpectedKind,
	)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
	assert.Equal(t,
		"y",
		errs[1].(*sema.NotDeclaredError).Name,
	)
	assert.Equal(t,
		common.DeclarationKindVariable,
		errs[1].(*sema.NotDeclaredError).ExpectedKind,
	)
}

func TestCheckInvalidUnknownDeclarationCallInGlobal(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       let x = y()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidRedeclarations(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int, a: Int) {
        let x = 1
        let x = 2
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	assert.IsType(t, &sema.RedeclarationError{}, errs[1])
}

func TestCheckInvalidConstantValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: Bool = 1
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidUse(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          testX
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueNotDeclared(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       var y = 2
       let z = y = x
   `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NonResourceTypeError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidVariableDeclarationSecondValueCopyTransfers(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       var x = 1
       var y = 2
       let z = y = x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NonResourceTypeError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueNotTarget(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource X {}
      fun f() {}

      let x <- create X()
      let z = f() <- x
  `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidAssignmentTargetError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueCopyTransferSecond(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     var y <- create R()
     let z <- y = x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueCopyTransferFirst(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     var y <- create R()
     let z = y <- x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueConstant(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     let y <- create R()
     let z <- y <- x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueTypeMismatch(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource X {}
     resource Y {}

     let x <- create X()
     var y <- create Y()
     let z <- y <- x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidVariableDeclarationSecondValueUseAfterInvalidation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     var y <- create R()
     let z <- y <- x

     let r <- x
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
}

func TestCheckVariableDeclarationSecondValue(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     var y <- create R()
     let z <- y <- x

     let r <- y
   `)

	require.NoError(t, err)

	assert.IsType(t,
		&sema.CompositeType{},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)

	assert.IsType(t,
		&sema.CompositeType{},
		RequireGlobalValue(t, checker.Elaboration, "y"),
	)

	assert.IsType(t,
		&sema.CompositeType{},
		RequireGlobalValue(t, checker.Elaboration, "z"),
	)

	assert.IsType(t,
		&sema.CompositeType{},
		RequireGlobalValue(t, checker.Elaboration, "r"),
	)
}

func TestCheckVariableDeclarationSecondValueDictionary(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
     resource R {}

     let x <- create R()
     var ys <- {"r": <-create R()}
     // NOTE: nested move is valid here
     let z <- ys["r"] <- x

     // NOTE: nested move is invalid here
     let r <- ys.remove(key: "r")
   `)

	require.NoError(t, err)

	assert.IsType(t,
		&sema.CompositeType{},
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)

	assert.IsType(t,
		&sema.DictionaryType{},
		RequireGlobalValue(t, checker.Elaboration, "ys"),
	)

	assert.IsType(t,
		&sema.OptionalType{},
		RequireGlobalValue(t, checker.Elaboration, "z"),
	)

	assert.IsType(t,
		&sema.OptionalType{},
		RequireGlobalValue(t, checker.Elaboration, "r"),
	)
}

func TestCheckVariableDeclarationSecondValueNil(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     resource R {}

     fun test() {
         var x: @R? <- create R()
         let y <- x <- nil
         destroy x
         destroy y
     }
   `)

	require.NoError(t, err)
}

func TestCheckTopLevelContractRestriction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
          contract C {}
        `,
		ParseAndCheckOptions{
			Config: &sema.Config{
				ValidTopLevelDeclarationsHandler: func(_ common.Location) common.DeclarationKindSet {
					return common.NewDeclarationKindSet(
						common.DeclarationKindContract,
						common.DeclarationKindImport,
					)
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckInvalidTopLevelContractRestriction(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"resource":           `resource Test {}`,
		"struct":             `struct Test {}`,
		"resource interface": `resource interface Test {}`,
		"struct interface":   `struct interface Test {}`,
		"event":              `event Test()`,
		"function":           `fun test() {}`,
		"transaction":        `transaction { execute {} }`,
		"constant":           `var x = 1`,
		"variable":           `let x = 1`,
	}

	for name, code := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseAndCheckWithOptions(t,
				code,
				ParseAndCheckOptions{
					Config: &sema.Config{
						ValidTopLevelDeclarationsHandler: func(_ common.Location) common.DeclarationKindSet {
							return common.NewDeclarationKindSet(
								common.DeclarationKindContractInterface,
								common.DeclarationKindContract,
								common.DeclarationKindImport,
							)
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
		})
	}
}

func TestCheckInvalidLocalDeclarations(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"transaction": `transaction { execute {} }`,
		"import":      `import 0x1`,
	}

	// composites and interfaces

	for _, kind := range common.AllCompositeKinds {
		for _, isInterface := range []bool{true, false} {

			if !kind.SupportsInterfaces() && isInterface {
				continue
			}

			interfaceKeyword := ""
			if isInterface {
				interfaceKeyword = "interface"
			}

			name := fmt.Sprintf(
				"%s %s",
				kind.Keyword(),

				interfaceKeyword,
			)

			body := "{}"
			if kind == common.CompositeKindEvent {
				body = "()"
			}

			var baseType string
			if kind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			tests[name] = fmt.Sprintf(
				`%s %s Test %s %s`,
				kind.Keyword(),
				interfaceKeyword,
				baseType,
				body,
			)
		}
	}

	//

	for name, code := range tests {
		t.Run(name, func(t *testing.T) {

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      fun test() {
                          %s
                      }
                    `,
					code,
				),
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])

		})
	}
}

func TestCheckVariableDeclarationTypeAnnotationRequired(t *testing.T) {

	t.Parallel()

	t.Run("empty array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let a = []
	    `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})

	t.Run("empty dictionary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let d = {}
	    `)
		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})
}

func TestCheckBuiltinRedeclaration(t *testing.T) {

	t.Parallel()

	// Check built-in conversion functions have a static type

	_ = sema.BaseValueActivation.ForEach(
		func(name string, _ *sema.Variable) error {

			t.Run(name, func(t *testing.T) {

				t.Run("re-declaration in function", func(t *testing.T) {

					_, err := ParseAndCheck(t,
						fmt.Sprintf(
							`
                                fun test() {
                                    let %[1]s = %[1]s
                                }
                            `,
							name,
						),
					)

					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.RedeclarationError{}, errs[0])
				})

				t.Run("global re-declaration", func(t *testing.T) {

					_, err := ParseAndCheck(t,
						fmt.Sprintf(
							`
                                let %[1]s = %[1]s
                            `,
							name,
						),
					)

					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.RedeclarationError{}, errs[0])
				})
			})

			return nil
		},
	)
}

func TestCheckUint64RedeclarationFails(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "let UInt64 = UInt64 ( 0b0 )")

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckTypeRedeclarationFails(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "let Type = Type")

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckSetToTypeList(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "var a=[Type]")
	assert.Nil(t, err)
}

func TestCheckSetToDictWithType(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "var j={0.0:Type}")
	assert.Nil(t, err)
}
