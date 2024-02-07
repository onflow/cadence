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

package stdlib_test

import (
	"testing"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/old_parser"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testContractUpdate(t *testing.T, oldCode string, newCode string) error {
	oldProgram, err := old_parser.ParseProgram(nil, []byte(oldCode), old_parser.Config{})
	require.NoError(t, err)

	newProgram, err := parser.ParseProgram(nil, []byte(newCode), parser.Config{})
	require.NoError(t, err)

	checker, err := sema.NewChecker(
		newProgram,
		utils.TestLocation,
		nil,
		&sema.Config{
			AccessCheckMode:    sema.AccessCheckModeStrict,
			AttachmentsEnabled: true,
		})
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	upgradeValidator := stdlib.NewLegacyContractUpdateValidator(
		utils.TestLocation,
		"Test",
		&runtime_utils.TestRuntimeInterface{},
		oldProgram,
		newProgram,
		map[common.Location]*sema.Elaboration{
			utils.TestLocation: checker.Elaboration,
		})
	return upgradeValidator.Validate()
}

func testContractUpdateWithImports(t *testing.T, oldCode, oldImport string, newCode, newImport string) error {
	oldProgram, err := old_parser.ParseProgram(nil, []byte(oldCode), old_parser.Config{})
	require.NoError(t, err)

	newProgram, err := parser.ParseProgram(nil, []byte(newCode), parser.Config{})
	require.NoError(t, err)

	newImportedProgram, err := parser.ParseProgram(nil, []byte(newImport), parser.Config{})
	require.NoError(t, err)

	importedChecker, err := sema.NewChecker(
		newImportedProgram,
		utils.ImportedLocation,
		nil,
		&sema.Config{
			AccessCheckMode:    sema.AccessCheckModeStrict,
			AttachmentsEnabled: true,
		},
	)

	require.NoError(t, err)
	err = importedChecker.Check()
	require.NoError(t, err)

	checker, err := sema.NewChecker(
		newProgram,
		utils.TestLocation,
		nil,
		&sema.Config{
			AccessCheckMode: sema.AccessCheckModeStrict,
			ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
				return sema.ElaborationImport{
					Elaboration: importedChecker.Elaboration,
				}, nil
			},
			AttachmentsEnabled: true,
		})
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	upgradeValidator := stdlib.NewLegacyContractUpdateValidator(
		utils.TestLocation,
		"Test",
		&runtime_utils.TestRuntimeInterface{
			OnGetAccountContractNames: func(address runtime.Address) ([]string, error) {
				return []string{"TestImport"}, nil
			},
		},
		oldProgram,
		newProgram,
		map[common.Location]*sema.Elaboration{
			utils.TestLocation:     checker.Elaboration,
			utils.ImportedLocation: importedChecker.Elaboration,
		})
	return upgradeValidator.Validate()
}

func getSingleContractUpdateErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err, contractName)

	require.Len(t, updateErr.Errors, 1)
	return updateErr.Errors[0]
}

func getContractUpdateError(t *testing.T, err error, contractName string) *stdlib.ContractUpdateError {
	require.Error(t, err)

	var contractUpdateErr *stdlib.ContractUpdateError
	require.ErrorAs(t, err, &contractUpdateErr)

	assert.Equal(t, contractName, contractUpdateErr.ContractName)

	return contractUpdateErr
}

func assertFieldTypeMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {
	var fieldMismatchError *stdlib.FieldMismatchError
	require.ErrorAs(t, err, &fieldMismatchError)

	assert.Equal(t, fieldName, fieldMismatchError.FieldName)
	assert.Equal(t, erroneousDeclName, fieldMismatchError.DeclName)

	var typeMismatchError *stdlib.TypeMismatchError
	assert.ErrorAs(t, fieldMismatchError.Err, &typeMismatchError)

	assert.Equal(t, expectedType, typeMismatchError.ExpectedType.String())
	assert.Equal(t, foundType, typeMismatchError.FoundType.String())
}

func assertFieldAuthorizationMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {
	var fieldMismatchError *stdlib.FieldMismatchError
	require.ErrorAs(t, err, &fieldMismatchError)

	assert.Equal(t, fieldName, fieldMismatchError.FieldName)
	assert.Equal(t, erroneousDeclName, fieldMismatchError.DeclName)

	var authorizationMismatchError *stdlib.AuthorizationMismatchError
	assert.ErrorAs(t, fieldMismatchError.Err, &authorizationMismatchError)

	assert.Equal(t, expectedType, authorizationMismatchError.ExpectedAuthorization.String())
	assert.Equal(t, foundType, authorizationMismatchError.FoundAuthorization.String())
}

func TestContractUpgradeFieldAccess(t *testing.T) {

	t.Parallel()

	t.Run("change field access to entitlement", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
				access(all) resource R {
					access(all) var a: Int
					init() {
						self.a = 0
					}
				}
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) resource R {
					access(E) var a: Int
					init() {
						self.a = 0
					}
				}
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field access to all", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Int
                init() {
                    self.a = 0
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Int
                init() {
                    self.a = 0
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})
}

func TestContractUpgradeFieldType(t *testing.T) {

	t.Parallel()

	t.Run("change field types illegally", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
                access(all) var a: Int
                init() {
                    self.a = 0
                }
            }
        `

		const newCode = `
			access(all) contract Test {
				access(all) var a: String
				init() {
					self.a = "hello"
				}
			}
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "Int", "String")

	})

	t.Run("change field intersection types illegally", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
				access(all) struct interface I {}
				access(all) struct interface J {}
				access(all) struct S: I, J {}

                access(all) var a: {I}
                init() {
                    self.a = S()
                }
            }
        `

		const newCode = `
			access(all) contract Test {
				access(all) struct interface I {}
				access(all) struct interface J {}
				access(all) struct S: I, J {}

                access(all) var a: {I, J}
                init() {
                    self.a = S()
                }
			}
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "{I}", "{I, J}")

	})

	t.Run("change field type capability reference auth", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Capability<&Int>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
                access(all) var a: Capability<auth(E) &Int>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "all", "E")
	})

	t.Run("change field type capability reference auth allowed composite", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub struct S {
					pub fun foo() {}
				}

                pub var a: Capability<&S>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E

				access(all) struct S {
					access(E) fun foo() {}
				}

                access(all) var a: Capability<auth(E) &S>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type capability reference auth allowed too many entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub struct S {
					pub fun foo() {}
				}

                pub var a: Capability<&S>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) entitlement F

				access(all) struct S {
					access(E) fun foo() {}
				}

                access(all) var a: Capability<auth(E, F) &S>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "E", "E, F")
	})

	t.Run("change field type capability reference auth fewer entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub struct S {
					pub fun foo() {}
					pub fun bar() {}
				}

                pub var a: Capability<&S>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) entitlement F

				access(all) struct S {
					access(E) fun foo() {}
					access(F) fun bar() {}
				}

                access(all) var a: Capability<auth(E) &S>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type capability reference auth disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub struct S {
					pub fun foo() {}
					pub fun bar() {}
				}

                pub var a: Capability<&S>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) entitlement F

				access(all) struct S {
					access(E) fun foo() {}
					access(F) fun bar() {}
				}

                access(all) var a: Capability<auth(E | F) &S>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})
}

func TestContractUpgradeIntersectionAuthorization(t *testing.T) {

	t.Parallel()

	t.Run("change field type capability reference auth allowed intersection", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
			pub contract Test {
				pub struct interface I {
					pub fun foo()
				}
				pub struct S:I {
					pub fun foo() {}
				}

				pub var a: Capability<&{I}>?
				init() {
					self.a = nil
				}
			}
	`

		const newCode = `
		access(all) contract Test {
			access(all) entitlement E

			access(all) struct interface I {
				access(E) fun foo()
			}

			access(all) struct S:I {
				access(E) fun foo() {}
			}

			access(all) var a: Capability<auth(E) &{I}>?
			init() {
				self.a = nil
			}
		}
	`

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type capability reference auth allowed too many entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
			pub contract Test {
				pub struct interface I {}
				pub struct S:I {
					pub fun foo() {}
				}

				pub var a: Capability<&{I}>?
				init() {
					self.a = nil
				}
			}
	`

		const newCode = `
		access(all) contract Test {
			access(all) entitlement E

			access(all) struct interface I {}

			access(all) struct S:I {
				access(E) fun foo() {}
			}

			access(all) var a: Capability<auth(E) &{I}>?
			init() {
				self.a = nil
			}
		}
	`

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "all", "E")
	})

	t.Run("change field type capability reference auth allowed multiple intersected", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
			pub contract Test {
				pub struct interface I {
					pub fun bar()
				}
				pub struct interface J {
					pub fun foo()
				}
				pub struct S:I, J {
					pub fun foo() {}
					pub fun bar() {}
				}

				pub var a: Capability<&{I, J}>?
				init() {
					self.a = nil
				}
			}
	`

		const newCode = `
		access(all) contract Test {
			access(all) entitlement E
			access(all) entitlement F

			access(all) struct interface I {
				access(E) fun foo()
			}
			access(all) struct interface J {
				access(F) fun bar()
			}

			access(all) struct S:I, J {
				access(E) fun foo() {}
				access(F) fun bar() {}
			}

			access(all) var a: Capability<auth(E, F) &{I, J}>?
			init() {
				self.a = nil
			}
		}
	`

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type capability reference auth allowed multiple intersected fewer entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
			pub contract Test {
				pub struct interface I {
					pub fun bar()
				}
				pub struct interface J {
					pub fun foo()
				}
				pub struct S:I, J {
					pub fun foo() {}
					pub fun bar() {}
				}

				pub var a: Capability<&{I, J}>?
				init() {
					self.a = nil
				}
			}
	`

		const newCode = `
		access(all) contract Test {
			access(all) entitlement E
			access(all) entitlement F

			access(all) struct interface I {
				access(E) fun foo()
			}
			access(all) struct interface J {
				access(F) fun bar()
			}

			access(all) struct S:I, J {
				access(E) fun foo() {}
				access(F) fun bar() {}
			}

			access(all) var a: Capability<auth(E) &{I, J}>?
			init() {
				self.a = nil
			}
		}
	`

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type capability reference auth multiple intersected with too many entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
			pub contract Test {
				pub struct interface I {
					pub fun bar()
				}
				pub struct interface J {
					pub fun foo()
				}
				pub struct S:I, J {
					pub fun foo() {}
					pub fun bar() {}
				}

				pub var a: Capability<&{I, J}>?
				init() {
					self.a = nil
				}
			}
	`

		const newCode = `
		access(all) contract Test {
			access(all) entitlement E
			access(all) entitlement F

			access(all) struct interface I {
				access(E) fun foo()
			}
			access(all) struct interface J {}

			access(all) struct S:I, J {
				access(E) fun foo() {}
			}

			access(all) var a: Capability<auth(E, F) &{I, J}>?
			init() {
				self.a = nil
			}
		}
	`

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "E", "E, F")
	})

}

func TestContractUpgradeIntersectionFieldType(t *testing.T) {

	t.Parallel()

	t.Run("change field type restricted type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @R{I}
                init() {
                    self.a <- create R()
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @{I} 
                init() {
                    self.a <- create R()
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		// This is not allowed because `@R{I}` is converted to `@R`, not `@{I}`
		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "R", "{I}")
	})

	t.Run("change field type restricted type variable sized", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @[R{I}]
                init() {
                    self.a <- [<- create R()]
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @[R] 
                init() {
                    self.a <- [<- create R()]
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted type constant sized", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @[R{I}; 1]
                init() {
                    self.a <- [<- create R()]
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @[R; 1] 
                init() {
                    self.a <- [<- create R()]
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted type constant sized with size change", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @[R{I}; 1]
                init() {
                    self.a <- [<- create R()]
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @[R; 2] 
                init() {
                    self.a <- [<- create R(), <- create R()]
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "[{I}; 1]", "[R; 2]")
	})

	t.Run("change field type restricted type dict", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @{Int: R{I}}
                init() {
                    self.a <- {0: <- create R()}
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @{Int: R}
                init() {
                    self.a <- {0: <- create R()}
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted type dict with qualified names", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: @{Int: R{I}}
                init() {
                    self.a <- {0: <- create R()}
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: @{Int: Test.R}
                init() {
                    self.a <- {0: <- create Test.R()}
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted reference type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {}
				pub resource R:I {}

                pub var a: Capability<&R{I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) resource interface I {}
				access(all) resource R:I {}

                access(all) var a: Capability<&R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted entitled reference type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {
					pub fun foo()
				}
				pub resource R:I {
					pub fun foo()
				}

                pub var a: Capability<&R{I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) resource interface I {
					access(E) fun foo()
				}
				access(all) resource R:I {
					access(E) fun foo() {}
				}

                access(all) var a: Capability<auth(E) &R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted entitled reference type with qualified types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {
					pub fun foo()
				}
				pub resource R:I {
					pub fun foo()
				}

                pub var a: Capability<&R{I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) resource interface I {
					access(Test.E) fun foo()
				}
				access(all) resource R:I {
					access(Test.E) fun foo() {}
				}

                access(all) var a: Capability<auth(Test.E) &Test.R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		require.NoError(t, err)
	})

	t.Run("change field type restricted entitled reference type with qualified types with imports", func(t *testing.T) {

		t.Parallel()

		const oldImport = `
			pub contract TestImport {
				pub resource interface I {
					pub fun foo()
				}
			}
		`

		const oldCode = `
			import TestImport from "imported"

            pub contract Test {
				pub resource R:TestImport.I {
					pub fun foo()
				}

                pub var a: Capability<&R{TestImport.I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newImport = `
			access(all) contract TestImport {
				access(all) entitlement E
				access(all) resource interface I {
					access(E) fun foo()
				}
			}
		`

		const newCode = `
			import TestImport from "imported"

            access(all) contract Test {
				access(all) entitlement F
				access(all) resource R: TestImport.I {
					access(TestImport.E) fun foo() {}
					access(Test.F) fun bar() {}
				}

                access(all) var a: Capability<auth(TestImport.E) &Test.R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdateWithImports(t, oldCode, oldImport, newCode, newImport)

		require.NoError(t, err)
	})

	t.Run("change field type restricted entitled reference type with too many granted entitlements", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
				pub resource interface I {
					pub fun foo()
				}
				pub resource R:I {
					pub fun foo()
					pub fun bar()
				}

                pub var a: Capability<&R{I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newCode = `
            access(all) contract Test {
				access(all) entitlement E
				access(all) entitlement F
				access(all) resource interface I {
					access(E) fun foo()
				}
				access(all) resource R:I {
					access(E) fun foo() {}
					access(F) fun bar() {}
				}

                access(all) var a: Capability<auth(E, F) &R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "E", "E, F")
	})

	t.Run("change field type restricted entitled reference type with too many granted entitlements with imports", func(t *testing.T) {

		t.Parallel()

		const oldImport = `
			pub contract TestImport {
				pub resource interface I {
					pub fun foo()
				}
			}
		`

		const oldCode = `
			import TestImport from "imported" 

            pub contract Test {
				pub resource R:TestImport.I {
					pub fun foo()
				}

                pub var a: Capability<&R{TestImport.I}>?
                init() {
                    self.a = nil
                }
            }
        `

		const newImport = `
			access(all) contract TestImport {
				access(all) entitlement E
				access(all) resource interface I {
					access(TestImport.E) fun foo()
				}
			}
		`

		const newCode = `
			import TestImport from "imported" 

            access(all) contract Test {
				access(all) entitlement F
				access(all) resource R: TestImport.I {
					access(TestImport.E) fun foo() {}
					access(Test.F) fun bar() {}
				}

                access(all) var a: Capability<auth(TestImport.E, Test.F) &Test.R>?
                init() {
                    self.a = nil
                }
            }
        `

		err := testContractUpdateWithImports(t, oldCode, oldImport, newCode, newImport)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "E", "E, F")
	})
}
