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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/old_parser"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
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

	upgradeValidator := stdlib.NewCadenceV042ToV1ContractUpdateValidator(
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

func testContractUpdateWithImports(
	t *testing.T,
	oldCode string,
	newCode string,
	newImports map[common.Location]string,
) error {
	oldProgram, err := old_parser.ParseProgram(nil, []byte(oldCode), old_parser.Config{})
	require.NoError(t, err)

	newProgram, err := parser.ParseProgram(nil, []byte(newCode), parser.Config{})
	require.NoError(t, err)

	elaborations := map[common.Location]*sema.Elaboration{}

	for location, code := range newImports {
		newImportedProgram, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
		require.NoError(t, err)

		importedChecker, err := sema.NewChecker(
			newImportedProgram,
			location,
			nil,
			&sema.Config{
				AccessCheckMode:    sema.AccessCheckModeStrict,
				AttachmentsEnabled: true,
			},
		)

		require.NoError(t, err)
		err = importedChecker.Check()
		require.NoError(t, err)

		elaborations[location] = importedChecker.Elaboration
	}

	checker, err := sema.NewChecker(
		newProgram,
		utils.TestLocation,
		nil,
		&sema.Config{
			AccessCheckMode: sema.AccessCheckModeStrict,
			ImportHandler: func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
				importedElaboration := elaborations[location]
				return sema.ElaborationImport{
					Elaboration: importedElaboration,
				}, nil
			},
			LocationHandler: func(identifiers []ast.Identifier, location common.Location) (
				locations []sema.ResolvedLocation, err error,
			) {
				if addressLocation, ok := location.(common.AddressLocation); ok && len(identifiers) == 1 {
					location = common.AddressLocation{
						Name:    identifiers[0].Identifier,
						Address: addressLocation.Address,
					}
				}

				locations = append(locations, sema.ResolvedLocation{
					Location:    location,
					Identifiers: identifiers,
				})

				return
			},
			AttachmentsEnabled: true,
		})
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	elaborations[utils.TestLocation] = checker.Elaboration

	upgradeValidator := stdlib.NewCadenceV042ToV1ContractUpdateValidator(
		utils.TestLocation,
		"Test",
		&runtime_utils.TestRuntimeInterface{
			OnGetAccountContractNames: func(address runtime.Address) ([]string, error) {
				return []string{"TestImport"}, nil
			},
		},
		oldProgram,
		newProgram,
		elaborations,
	)
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

	t.Run("change field access to self", func(t *testing.T) {

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
                access(self) var a: Int
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

	t.Run("simple invalid", func(t *testing.T) {

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

	t.Run("intersection types invalid", func(t *testing.T) {

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

	t.Run("capability reference auth", func(t *testing.T) {

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

	t.Run("capability reference auth allowed composite", func(t *testing.T) {

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

	t.Run("capability reference auth allowed too many entitlements", func(t *testing.T) {

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

	t.Run("capability reference auth fewer entitlements", func(t *testing.T) {

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

	t.Run("capability reference auth disjunctive entitlements", func(t *testing.T) {

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

	t.Run("changing to a non-storable types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
                access(all) struct Foo {
                    access(all) var a: Int
                    init() {
                        self.a = 0
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) struct Foo {
                    access(all) var a: &Int?
                    init() {
                        self.a = nil
                    }
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("changing from a non-storable types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
                access(all) struct Foo {
                    access(all) var a: &Int?
                    init() {
                        self.a = nil
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) struct Foo {
                    access(all) var a: Int
                    init() {
                        self.a = 0
                    }
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

	t.Run("restricted type", func(t *testing.T) {

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

	t.Run("AnyResource restricted type, with restrictions", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub resource interface I {}
                pub resource R:I {}

                pub var a: @AnyResource{I}
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
		require.NoError(t, err)
	})

	t.Run("AnyResource restricted type, without restrictions", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub resource R {}

                pub var a: @AnyResource{}
                init() {
                    self.a <- create R()
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) resource R {}

                access(all) var a: @AnyResource
                init() {
                    self.a <- create R()
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("restricted type variable sized", func(t *testing.T) {

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

	t.Run("restricted type constant sized", func(t *testing.T) {

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

	t.Run("restricted type constant sized with size change", func(t *testing.T) {

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

	t.Run("restricted type dict", func(t *testing.T) {

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

	t.Run("restricted type dict with qualified names", func(t *testing.T) {

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

	t.Run("restricted reference type", func(t *testing.T) {

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

	t.Run("restricted entitled reference type", func(t *testing.T) {

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

	t.Run("restricted entitled reference type with qualified types", func(t *testing.T) {

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

	t.Run("restricted entitled reference type with qualified types with imports", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            import TestImport from 0x01

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
            import TestImport from 0x01

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

		err := testContractUpdateWithImports(
			t,
			oldCode,
			newCode,
			map[common.Location]string{
				common.AddressLocation{
					Name:    "TestImport",
					Address: common.MustBytesToAddress([]byte{0x1}),
				}: newImport,
			},
		)

		require.NoError(t, err)
	})

	t.Run("restricted entitled reference type with too many granted entitlements", func(t *testing.T) {

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

	t.Run("restricted entitled reference type with too many granted entitlements with imports", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            import TestImport from 0x01

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
            import TestImport from 0x01

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

		err := testContractUpdateWithImports(
			t,
			oldCode,
			newCode,
			map[common.Location]string{
				common.AddressLocation{
					Name:    "TestImport",
					Address: common.MustBytesToAddress([]byte{0x1}),
				}: newImport,
			},
		)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldAuthorizationMismatchError(t, cause, "Test", "a", "E", "E, F")
	})

	t.Run("restricted reference type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub resource interface I {}

                pub resource R:I {
                    access(all) var ref: &R{I}?

                    init() {
                        self.ref = nil
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) resource interface I {
                    access(E) fun foo()
                }

                access(all) entitlement E

                access(all) resource R:I {
                    access(all) var ref: auth(E) &R?

                    init() {
                        self.ref = nil
                    }

                    access(E) fun foo() {}
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("restricted anystruct reference type invalid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub resource interface I {}

                pub resource R:I {
                    access(all) var ref: &AnyStruct{I}?

                    init() {
                        self.ref = nil
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) resource interface I {
                    access(E) fun foo()
                }

                access(all) entitlement E

                access(all) resource R:I {
                    access(all) var ref: auth(E) &AnyStruct?

                    init() {
                        self.ref = nil
                    }

                    access(E) fun foo() {}
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "R", "ref", "{I}", "AnyStruct")

	})

	t.Run("restricted anystruct reference type valid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub resource interface I {}

                pub resource R:I {
                    access(all) var ref: &AnyStruct{I}?

                    init() {
                        self.ref = nil
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) resource interface I {
                    access(E) fun foo()
                }

                access(all) entitlement E

                access(all) resource R:I {
                    access(all) var ref: auth(E) &{I}?

                    init() {
                        self.ref = nil
                    }

                    access(E) fun foo() {}
                }
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

}

func TestTypeRequirementRemoval(t *testing.T) {

	t.Parallel()

	t.Run("resource valid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract interface Test {
                access(all) resource R {}
                access(all) fun foo(r: @R)
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) resource interface R {}
                access(all) fun foo(r: @{R})
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("resource invalid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract interface Test {
                access(all) resource R {}
                access(all) fun foo(r: @R)
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) struct interface R {}
                access(all) fun foo(r: {R})
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		declKindChangeError := &stdlib.InvalidDeclarationKindChangeError{}
		require.ErrorAs(t, cause, &declKindChangeError)
	})

	t.Run("struct valid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract interface Test {
                access(all) struct S {}
                access(all) fun foo(r: S)
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) struct interface S {}
                access(all) fun foo(r: {S})
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("struct invalid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            access(all) contract interface Test {
                access(all) struct S {}
                access(all) fun foo(r: S)
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) resource interface S {}
                access(all) fun foo(r: @{S})
            }
        `

		err := testContractUpdate(t, oldCode, newCode)
		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		declKindChangeError := &stdlib.InvalidDeclarationKindChangeError{}
		require.ErrorAs(t, cause, &declKindChangeError)
	})
}
