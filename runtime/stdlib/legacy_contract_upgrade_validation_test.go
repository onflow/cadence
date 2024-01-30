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

	"github.com/onflow/cadence/runtime/old_parser"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
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

	upgradeValidator := stdlib.NewLegacyContractUpdateValidator(utils.TestLocation, "Test", oldProgram, newProgram, checker.Elaboration)
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

		// TODO: this should not be allowed, the migration is not going to change the underlying referenced type of `a`
		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "Capability<&Int>", "Capability<auth(E) &Int>")
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

		// TODO: this should not be allowed, as the migration will convert `&R{I}` to `auth(E) &R`
		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "Capability<auth(E) &R>", "Capability<auth(E, F) &R>")
	})
}
