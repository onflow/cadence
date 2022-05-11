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

package runtime

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func newContractDeployTransaction(function, name, code string) string {
	return fmt.Sprintf(
		`
                transaction {
                    prepare(signer: AuthAccount) {
                        signer.contracts.%s(name: "%s", code: "%s".decodeHex())
                    }
                }
            `,
		function,
		name,
		hex.EncodeToString([]byte(code)),
	)
}

func newContractAddTransaction(name string, code string) string {
	return newContractDeployTransaction(
		sema.AuthAccountContractsTypeAddFunctionName,
		name,
		code,
	)
}

func newContractUpdateTransaction(name string, code string) string {
	return newContractDeployTransaction(
		sema.AuthAccountContractsTypeUpdateExperimentalFunctionName,
		name,
		code,
	)
}

func newContractRemovalTransaction(contractName string) string {
	return fmt.Sprintf(
		`
           transaction {
               prepare(signer: AuthAccount) {
                   signer.contracts.%s(name: "%s")
               }
           }
       `,
		sema.AuthAccountContractsTypeRemoveFunctionName,
		contractName,
	)
}

func newContractDeploymentTransactor(t *testing.T, updateValidationEnabled bool) func(code string) error {
	rt := newTestInterpreterRuntime(
		WithContractUpdateValidationEnabled(updateValidationEnabled),
	)

	accountCodes := map[common.LocationID][]byte{}
	var events []cadence.Event
	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x42})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		removeAccountContractCode: func(address Address, name string) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			delete(accountCodes, location.ID())
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	return func(code string) error {
		return rt.ExecuteTransaction(
			Script{
				Source: []byte(code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
	}
}

// testDeployAndUpdate deploys a contract in one transaction,
// then updates the contract in another transaction
func testDeployAndUpdate(t *testing.T, updateValidationEnabled bool, name string, oldCode string, newCode string) error {
	executeTransaction := newContractDeploymentTransactor(t, updateValidationEnabled)
	err := executeTransaction(newContractAddTransaction(name, oldCode))
	require.NoError(t, err)

	return executeTransaction(newContractUpdateTransaction(name, newCode))
}

// testDeployAndRemove deploys a contract in one transaction,
// then removes the contract in another transaction
func testDeployAndRemove(t *testing.T, updateValidationEnabled bool, name string, code string) error {
	executeTransaction := newContractDeploymentTransactor(t, updateValidationEnabled)
	err := executeTransaction(newContractAddTransaction(name, code))
	require.NoError(t, err)

	return executeTransaction(newContractRemovalTransaction(name))
}

func TestRuntimeContractUpdateValidation(t *testing.T) {

	t.Parallel()

	const contractValidationEnabled = true

	t.Run("change field type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String
                init() {
                    self.a = "hello"
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Int
                init() {
                    self.a = 0
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "String", "Int")
	})

	t.Run("add field", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String

                init() {
                    self.a = "hello"
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: String
                pub var b: Int

                init() {
                    self.a = "hello"
                    self.b = 0
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "Test", "b")
	})

	t.Run("remove field", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String
                pub var b: Int

                init() {
                    self.a = "hello"
                    self.b = 0
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: String

                init() {
                    self.a = "hello"
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("change nested decl field type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var a: @TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                pub resource TestResource {

                    pub let b: Int

                    init() {
                        self.b = 1234
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var a: @Test.TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                pub resource TestResource {

                    pub let b: String

                    init() {
                        self.b = "string_1234"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestResource", "b", "Int", "String")
	})

	t.Run("add field to nested decl", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var a: @TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                pub resource TestResource {

                    pub var b: String

                    init() {
                        self.b = "hello"
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var a: @Test.TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                pub resource TestResource {

                    pub var b: String
                    pub var c: Int

                    init() {
                        self.b = "hello"
                        self.c = 0
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "TestResource", "c")
	})

	t.Run("change indirect field type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: [TestStruct; 1]

                init() {
                    self.x = [TestStruct()]
                }

                pub struct TestStruct {
                    pub let a: Int
                    pub var b: Int

                    init() {
                        self.a = 123
                        self.b = 456
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: [TestStruct; 1]

                init() {
                    self.x = [TestStruct()]
                }

                pub struct TestStruct {
                    pub let a: Int
                    pub var b: String

                    init() {
                        self.a = 123
                        self.b = "string_456"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "b", "Int", "String")
	})

	t.Run("circular types refs", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: {String: Foo}

                init() {
                    self.x = { "foo" : Foo() }
                }

                pub struct Foo {

                    pub let a: Foo?
                    pub let b: Bar

                    init() {
                        self.a = nil
                        self.b = Bar()
                    }
                }

                pub struct Bar {

                    pub let c: Foo?
                    pub let d: Bar?

                    init() {
                        self.c = nil
                        self.d = nil
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: {String: Foo}

                init() {
                    self.x = { "foo" : Foo() }
                }

                pub struct Foo {

                    pub let a: Foo?
                    pub let b: Bar

                    init() {
                        self.a = nil
                        self.b = Bar()
                    }
                }

                pub struct Bar {

                    pub let c: Foo?
                    pub let d: String

                    init() {
                        self.c = nil
                        self.d = "string_d"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Bar", "d", "Bar?", "String")
	})

	t.Run("qualified vs unqualified nominal type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: Test.TestStruct
                pub var y: TestStruct

                init() {
                    self.x = Test.TestStruct()
                    self.y = TestStruct()
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: TestStruct
                pub var y: Test.TestStruct

                init() {
                    self.x = TestStruct()
                    self.y = Test.TestStruct()
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("change imported nominal type to local", func(t *testing.T) {

		t.Parallel()

		const importCode = `
		    pub contract TestImport {

		        pub struct TestStruct {
		            pub let a: Int
		            pub var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		executeTransaction := newContractDeploymentTransactor(t, contractValidationEnabled)

		err := executeTransaction(newContractAddTransaction("TestImport", importCode))
		require.NoError(t, err)

		const oldCode = `
		    import TestImport from 0x42

		    pub contract Test {

		        pub var x: TestImport.TestStruct

		        init() {
		            self.x = TestImport.TestStruct()
		        }
		    }
		`

		const newCode = `
		    pub contract Test {

		        pub var x: TestStruct

		        init() {
		            self.x = TestStruct()
		        }

		        pub struct TestStruct {
		            pub let a: Int
		            pub var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		err = executeTransaction(newContractAddTransaction("Test", oldCode))
		require.NoError(t, err)

		err = executeTransaction(newContractUpdateTransaction("Test", newCode))
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "x", "TestImport.TestStruct", "TestStruct")
	})

	t.Run("contract interface update", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract interface Test {
                pub var a: String
                pub fun getA() : String
            }
        `

		const newCode = `
            pub contract interface Test {
                pub var a: Int
                pub fun getA() : Int
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "String", "Int")
	})

	t.Run("convert interface to contract", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract interface Test {
                pub var a: String
                pub fun getA() : String
            }
        `

		const newCode = `
            pub contract Test {

                pub var a: String

                init() {
                    self.a = "hello"
                }

                pub fun getA() : String {
                    return self.a
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test",
			common.DeclarationKindContractInterface,
			common.DeclarationKindContract,
		)
	})

	t.Run("convert contract to interface", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var a: String

                init() {
                    self.a = "hello"
                }

                pub fun getA() : String {
                    return self.a
                }
            }
        `

		const newCode = `
            pub contract interface Test {
                pub var a: String
                pub fun getA() : String
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test",
			common.DeclarationKindContract,
			common.DeclarationKindContractInterface,
		)
	})

	t.Run("change non stored", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: UsedStruct

                init() {
                    self.x = UsedStruct()
                }

                pub struct UsedStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }

                    pub fun getA() : Int {
                        return self.a
                    }
                }

                pub struct UnusedStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }

                    pub fun getA() : Int {
                        return self.a
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: UsedStruct

                init() {
                    self.x = UsedStruct()
                }

                pub struct UsedStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }

                    pub fun getA() : String {
                        return "hello_123"
                    }

                    pub fun getA_new() : Int {
                        return self.a
                    }
                }

                pub struct UnusedStruct {
                    pub let a: String

                    init() {
                        self.a = "string_456"
                    }

                    pub fun getA() : String {
                        return self.a
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)

		// Changing unused public composite types should also fail, since those could be
		// referred by anyone in the chain, and may cause data inconsistency.
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "UnusedStruct", "a", "Int", "String")
	})

	t.Run("change enum type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: Foo

                init() {
                    self.x = Foo.up
                }

                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: Foo

                init() {
                    self.x = Foo.up
                }

                pub enum Foo: UInt128 {
                    pub case up
                    pub case down
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertConformanceMismatchError(t, cause, "Foo")
	})

	t.Run("change nested interface", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var x: AnyStruct{TestStruct}?

                init() {
                    self.x = nil
                }

                pub struct interface TestStruct {
                    pub let a: String
                    pub var b: Int
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var x: AnyStruct{TestStruct}?

                init() {
                    self.x = nil
                }

                pub struct interface TestStruct {
                    pub let a: Int
                    pub var b: Int
                }
            }
       `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "String", "Int")
	})

	t.Run("change nested interface to struct", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub struct interface TestStruct {
                    pub var a: Int
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"TestStruct",
			common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
		)
	})

	t.Run("adding a nested struct", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
            }
        `

		const newCode = `
            pub contract Test {
                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
       `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("removing a nested struct", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")
	})

	t.Run("add and remove field", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String
                init() {
                    self.a = "hello"
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var b: Int
                init() {
                    self.b = 0
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "Test", "b")
	})

	t.Run("multiple errors", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String

                init() {
                    self.a = "hello"
                }

                pub struct interface TestStruct {
                    pub var a: Int
                }
            }
       `

		const newCode = `
            pub contract Test {
                pub var a: Int
                pub var b: String

                init() {
                    self.a = 0
                    self.b = "hello"
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		updateErr := getContractUpdateError(t, err, "Test")
		require.NotNil(t, updateErr)

		childErrors := updateErr.Errors
		require.Len(t, childErrors, 3)

		assertFieldTypeMismatchError(t, childErrors[0], "Test", "a", "String", "Int")
		assertExtraneousFieldError(t, childErrors[1], "Test", "b")
		assertDeclTypeChangeError(
			t,
			childErrors[2],
			"TestStruct",
			common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
		)
	})

	t.Run("check error messages", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String

                init() {
                    self.a = "hello"
                }

                pub struct interface TestStruct {
                    pub var a: Int
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Int
                pub var b: String

                init() {
                    self.a = 0
                    self.b = "hello"
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		const expectedError = "error: mismatching field `a` in `Test`\n" +
			" --> 0000000000000042.Test:3:27\n" +
			"  |\n" +
			"3 |                 pub var a: Int\n" +
			"  |                            ^^^ incompatible type annotations. expected `String`, found `Int`\n" +
			"\n" +
			"error: found new field `b` in `Test`\n" +
			" --> 0000000000000042.Test:4:24\n" +
			"  |\n" +
			"4 |                 pub var b: String\n" +
			"  |                         ^\n" +
			"\n" +
			"error: trying to convert structure interface `TestStruct` to a structure\n" +
			"  --> 0000000000000042.Test:11:27\n" +
			"   |\n" +
			"11 |                 pub struct TestStruct {\n" +
			"   |                            ^^^^^^^^^^"

		require.Contains(t, err.Error(), expectedError)
	})

	t.Run("Test reference types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub var vault: Capability<&TestStruct>?

                init() {
                    self.vault = nil
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var vault: Capability<&TestStruct>?

                init() {
                    self.vault = nil
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test function type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            pub contract Test {

                pub var add: ((Int, Int): Int)

                init() {
                    self.add = fun (a: Int, b: Int): Int {
                        return a + b
                    }
                }

                pub struct TestStruct {
                    pub let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error: field add has non-storable type: ((Int, Int): Int)")
	})

	t.Run("Test conformance", func(t *testing.T) {

		t.Parallel()

		const importCode = `
		    pub contract TestImport {
		        pub struct interface AnInterface {
		            pub a: Int
		        }
		    }
		`

		executeTransaction := newContractDeploymentTransactor(t, contractValidationEnabled)
		err := executeTransaction(newContractAddTransaction("TestImport", importCode))
		require.NoError(t, err)

		const oldCode = `
		    import TestImport from 0x42

		    pub contract Test {
		        pub struct TestStruct1 {
		            pub let a: Int
		            init() {
		                self.a = 123
		            }
		        }

		        pub struct TestStruct2: TestImport.AnInterface {
		            pub let a: Int

		            init() {
		                self.a = 123
		            }
		        }
		    }
	    `

		const newCode = `
		    import TestImport from 0x42

		    pub contract Test {

		        pub struct TestStruct2: TestImport.AnInterface {
		            pub let a: Int

		            init() {
		                self.a = 123
		            }
		        }

		        pub struct TestStruct1 {
		            pub let a: Int
		            init() {
		                self.a = 123
		            }
		        }
		    }
		`

		err = executeTransaction(newContractAddTransaction("Test", oldCode))
		require.NoError(t, err)

		err = executeTransaction(newContractUpdateTransaction("Test", newCode))
		require.NoError(t, err)
	})

	t.Run("Test all types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                // simple nominal type
                pub var a: TestStruct

                // qualified nominal type
                pub var b: Test.TestStruct

                // optional type
                pub var c: Int?

                // variable sized type
                pub var d: [Int]

                // constant sized type
                pub var e: [Int; 2]

                // dictionary type
                pub var f: {Int: String}

                // restricted type
                pub var g: {TestInterface}

                // instantiation and reference types
                pub var h:  Capability<&TestStruct>?

                // function type
                pub var i: Capability<&((Int, Int): Int)>?

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = Test.TestStruct()
                    self.c = 123
                    self.d = [123]
                    self.e = [123, 456]
                    self.f = {1: "Hello"}
                    self.g = TestStruct()
                    self.h = nil
                    self.i = nil
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		const newCode = `
            pub contract Test {


                // function type
                pub var i: Capability<&((Int, Int): Int)>?

                // instantiation and reference types
                pub var h:  Capability<&TestStruct>?

                // restricted type
                pub var g: {TestInterface}

                // dictionary type
                pub var f: {Int: String}

                // constant sized type
                pub var e: [Int; 2]

                // variable sized type
                pub var d: [Int]

                // optional type
                pub var c: Int?

                // qualified nominal type
                pub var b: Test.TestStruct

                // simple nominal type
                pub var a: TestStruct

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = Test.TestStruct()
                    self.c = 123
                    self.d = [123]
                    self.e = [123, 456]
                    self.f = {1: "Hello"}
                    self.g = TestStruct()
                    self.h = nil
                    self.i = nil
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test restricted types", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                // restricted type
                pub var a: {TestInterface}
                pub var b: {TestInterface}
                pub var c: AnyStruct{TestInterface}
                pub var d: AnyStruct{TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                    self.c = TestStruct()
                    self.d = TestStruct()
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: {TestInterface}
                pub var b: AnyStruct{TestInterface}
                pub var c: {TestInterface}
                pub var d: AnyStruct{TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                    self.c = TestStruct()
                    self.d = TestStruct()
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test invalid restricted types change", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {

                // restricted type
                pub var a: TestStruct{TestInterface}
                pub var b: {TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: {TestInterface}
                pub var b: TestStruct{TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                }

                pub struct TestStruct:TestInterface {
                    pub let a: Int
                    init() {
                        self.a = 123
                    }
                }

                pub struct interface TestInterface {
                    pub let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pub var a: {TestInterface}"+
			"\n  |                            ^^^^^^^^^^^^^^^ "+
			"incompatible type annotations. expected `TestStruct{TestInterface}`, found `{TestInterface}`")

		assert.Contains(t, err.Error(), "pub var b: TestStruct{TestInterface}"+
			"\n  |                            ^^^^^^^^^^^^^^^^^^^^^^^^^ "+
			"incompatible type annotations. expected `{TestInterface}`, found `TestStruct{TestInterface}`")
	})

	t.Run("enum valid", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                }
            }
       `

		const newCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("enum remove case", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingEnumCasesError(t, cause, "Foo", 2, 1)
	})

	t.Run("enum add case", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                    pub case left
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("enum swap cases", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case up
                    pub case down
                    pub case left
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub enum Foo: UInt8 {
                    pub case down
                    pub case left
                    pub case up
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		updateErr := getContractUpdateError(t, err, "Test")
		require.NotNil(t, updateErr)

		childErrors := updateErr.Errors
		require.Len(t, childErrors, 3)

		assertEnumCaseMismatchError(t, childErrors[0], "up", "down")
		assertEnumCaseMismatchError(t, childErrors[1], "down", "left")
		assertEnumCaseMismatchError(t, childErrors[2], "left", "up")
	})

	t.Run("Remove and add struct", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
		    pub contract Test {

		        pub struct TestStruct {
		            pub let a: Int
		            pub var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		const updateCode1 = `
		    pub contract Test {
		    }
		`

		executeTransaction := newContractDeploymentTransactor(t, contractValidationEnabled)

		err := executeTransaction(newContractAddTransaction("Test", oldCode))
		require.NoError(t, err)

		err = executeTransaction(newContractUpdateTransaction("Test", updateCode1))
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")

		const updateCode2 = `
		    pub contract Test {

		        pub struct TestStruct {
		            pub let a: String

		            init() {
		                self.a = "hello123"
		            }
		        }
		    }
		`

		err = executeTransaction(newContractUpdateTransaction("Test", updateCode2))
		require.Error(t, err)

		cause = getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "Int", "String")
	})

	t.Run("Rename struct", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
		    pub contract Test {

		        pub struct TestStruct {
		            pub let a: Int
		            pub var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		const newCode = `
		    pub contract Test {

		        pub struct TestStructRenamed {
		            pub let a: Int
		            pub var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")
	})

	t.Run("Remove contract with enum", func(t *testing.T) {

		t.Parallel()

		const code = `
		    pub contract Test {
		        pub enum TestEnum: Int {
		        }
		    }
		`

		err := testDeployAndRemove(t, contractValidationEnabled, "Test", code)
		require.Error(t, err)

		assertContractRemovalError(t, err, "Test")
	})

	t.Run("Remove contract interface with enum", func(t *testing.T) {

		const code = `
		    pub contract interface Test {
		        pub enum TestEnum: Int {
		        }
		    }
		`

		err := testDeployAndRemove(t, contractValidationEnabled, "Test", code)
		require.Error(t, err)

		assertContractRemovalError(t, err, "Test")
	})

	t.Run("Remove contract without enum", func(t *testing.T) {

		t.Parallel()

		const code = `
		    pub contract Test {
		        pub struct TestStruct {
		            pub let a: Int

		            init() {
		                self.a = 123
		            }
		        }
		    }
		`

		err := testDeployAndRemove(t, contractValidationEnabled, "Test", code)
		require.NoError(t, err)
	})

	t.Run("removing multiple nested structs", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
		    pub contract Test {
		        pub struct A {}
		        pub struct B {}
		    }
		`

		const newCode = `
		    pub contract Test {}
		`

		// Errors reporting was previously non-deterministic,
		// assert that reports are deterministic

		for i := 0; i < 1000; i++ {

			err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
			require.Error(t, err)

			updateErr := getContractUpdateError(t, err, "Test")

			childErrors := updateErr.Errors
			require.Len(t, childErrors, 2)

			if !assertMissingDeclarationError(t, childErrors[0], "A") {
				t.FailNow()
			}
			assertMissingDeclarationError(t, childErrors[1], "B")
		}
	})
}

func assertContractRemovalError(t *testing.T, err error, name string) {
	var contractRemovalError *ContractRemovalError
	require.ErrorAs(t, err, &contractRemovalError)

	assert.Equal(t, name, contractRemovalError.Name)
}

func assertDeclTypeChangeError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	oldKind common.DeclarationKind,
	newKind common.DeclarationKind,
) {
	var declTypeChangeError *InvalidDeclarationKindChangeError
	require.ErrorAs(t, err, &declTypeChangeError)

	assert.Equal(t, oldKind, declTypeChangeError.OldKind)
	assert.Equal(t, erroneousDeclName, declTypeChangeError.Name)
	assert.Equal(t, newKind, declTypeChangeError.NewKind)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	var extraFieldError *ExtraneousFieldError
	require.ErrorAs(t, err, &extraFieldError)

	assert.Equal(t, fieldName, extraFieldError.FieldName)
	assert.Equal(t, erroneousDeclName, extraFieldError.DeclName)
}

func assertFieldTypeMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {
	var fieldMismatchError *FieldMismatchError
	require.ErrorAs(t, err, &fieldMismatchError)

	assert.Equal(t, fieldName, fieldMismatchError.FieldName)
	assert.Equal(t, erroneousDeclName, fieldMismatchError.DeclName)

	var typeMismatchError *TypeMismatchError
	assert.ErrorAs(t, fieldMismatchError.Err, &typeMismatchError)

	assert.Equal(t, expectedType, typeMismatchError.ExpectedType.String())
	assert.Equal(t, foundType, typeMismatchError.FoundType.String())
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
) {
	var conformanceMismatchError *ConformanceMismatchError
	require.ErrorAs(t, err, &conformanceMismatchError)

	assert.Equal(t, erroneousDeclName, conformanceMismatchError.DeclName)
}

func assertEnumCaseMismatchError(t *testing.T, err error, expectedEnumCase string, foundEnumCase string) {
	var enumMismatchError *EnumCaseMismatchError
	require.ErrorAs(t, err, &enumMismatchError)

	assert.Equal(t, expectedEnumCase, enumMismatchError.ExpectedName)
	assert.Equal(t, foundEnumCase, enumMismatchError.FoundName)
}

func assertMissingEnumCasesError(t *testing.T, err error, declName string, expectedCases int, foundCases int) {
	var missingEnumCasesError *MissingEnumCasesError
	require.ErrorAs(t, err, &missingEnumCasesError)

	assert.Equal(t, declName, missingEnumCasesError.DeclName)
	assert.Equal(t, expectedCases, missingEnumCasesError.Expected)
	assert.Equal(t, foundCases, missingEnumCasesError.Found)
}

func assertMissingDeclarationError(t *testing.T, err error, declName string) bool {
	var missingDeclError *MissingDeclarationError
	require.ErrorAs(t, err, &missingDeclError)

	return assert.Equal(t, declName, missingDeclError.Name)
}

func getSingleContractUpdateErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err, contractName)

	require.Len(t, updateErr.Errors, 1)
	return updateErr.Errors[0]
}

func getContractUpdateError(t *testing.T, err error, contractName string) *ContractUpdateError {
	require.Error(t, err)

	var invalidContractDeploymentErr *InvalidContractDeploymentError
	require.ErrorAs(t, err, &invalidContractDeploymentErr)

	var contractUpdateErr *ContractUpdateError
	require.ErrorAs(t, err, &contractUpdateErr)

	assert.Equal(t, contractName, contractUpdateErr.ContractName)

	return contractUpdateErr
}

func TestRuntimeContractUpdateValidationDisabled(t *testing.T) {

	t.Parallel()

	const contractValidationEnabled = false

	t.Run("change field type", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: String
                init() {
                    self.a = "hello"
                }
              }
        `

		const newCode = `
            pub contract Test {
                pub var a: Int
                init() {
                    self.a = 0
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})
}

func TestRuntimeContractUpdateConformanceChanges(t *testing.T) {

	t.Parallel()

	const contractValidationEnabled = true

	t.Run("Adding conformance", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo {
                    init() {}
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo: Bar {
                    init() {
                    }

                    pub fun getName(): String {
                        return "John"
                    }
                }

                pub struct interface Bar {
                    pub fun getName(): String
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Adding conformance with new fields", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo {
                    init() {}
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo: Bar {
                    pub var name: String

                    init() {
                        self.name = "John"
                    }
                }

                pub struct interface Bar {
                    pub var name: String
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")

		assertExtraneousFieldError(t, cause, "Foo", "name")
	})

	t.Run("Removing conformance", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo: Bar {
                    init() {}
                }

                pub struct interface Bar {
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo {
                    init() {}
                }

                pub struct interface Bar {
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.Error(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")

		assertConformanceMismatchError(t, cause, "Foo")
	})

	t.Run("Change conformance order", func(t *testing.T) {

		t.Parallel()

		const oldCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo: First, Second {
                    init() {}
                }

                pub struct interface First {
                }

                pub struct interface Second {
                }
            }
        `

		const newCode = `
            pub contract Test {
                pub var a: Foo
                init() {
                    self.a = Foo()
                }

                pub struct Foo: Second, First {
                    init() {}
                }

                pub struct interface First {
                }

                pub struct interface Second {
                }
            }
        `

		err := testDeployAndUpdate(t, contractValidationEnabled, "Test", oldCode, newCode)
		require.NoError(t, err)
	})
}

func TestRuntimeContractUpdateProgramCaching(t *testing.T) {

	const name = "Test"
	const oldCode = `
	  pub contract Test { init() { 1 } }
	`
	const newCode = `
	  pub contract Test { init() { 2 } }
	`

	address := common.MustBytesToAddress([]byte{0x42})

	location := common.AddressLocation{
		Address: address,
		Name:    name,
	}
	contractLocationID := location.ID()

	type locationAccessCounts map[common.LocationID]int

	newTester := func() (
		runtimeInterface *testRuntimeInterface,
		executeTransaction func(code string) error,
		programGets locationAccessCounts,
		programSets locationAccessCounts,
	) {
		rt := newTestInterpreterRuntime(
			WithContractUpdateValidationEnabled(true),
		)

		accountCodes := map[common.LocationID][]byte{}
		var events []cadence.Event

		programGets = locationAccessCounts{}
		programSets = locationAccessCounts{}

		runtimeInterface = &testRuntimeInterface{
			getProgram: func(location Location) (*interpreter.Program, error) {
				locationID := location.ID()

				if runtimeInterface.programs == nil {
					runtimeInterface.programs = map[common.LocationID]*interpreter.Program{}
				}

				program := runtimeInterface.programs[locationID]
				if program != nil {
					programGets[locationID]++
				}

				return program, nil
			},
			setProgram: func(location Location, program *interpreter.Program) error {
				locationID := location.ID()

				programSets[locationID]++

				if runtimeInterface.programs == nil {
					runtimeInterface.programs = map[common.LocationID]*interpreter.Program{}
				}

				runtimeInterface.programs[locationID] = program

				return nil
			},
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location.ID()], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location.ID()], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location.ID()] = code
				return nil
			},
			removeAccountContractCode: func(address Address, name string) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				delete(accountCodes, location.ID())
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		executeTransaction = func(code string) error {
			return rt.ExecuteTransaction(
				Script{
					Source: []byte(code),
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
		}

		return
	}

	runtimeInterface1, executeTransaction1, programGets1, programSets1 := newTester()
	runtimeInterface2, executeTransaction2, programGets2, programSets2 := newTester()

	clearLocationAccessCounts := func() {
		for _, counts := range []locationAccessCounts{
			programGets1,
			programSets1,
			programGets2,
			programSets2,
		} {
			for locationID := range counts { //nolint:maprangecheck
				delete(counts, locationID)
			}
		}
	}

	t.Run("Deploy contract to both", func(t *testing.T) {

		clearLocationAccessCounts()

		addTx := newContractAddTransaction(name, oldCode)

		txLocationID := common.TransactionLocation{0}.ID()

		// Deploy to first

		err := executeTransaction1(addTx)
		require.NoError(t, err)
		require.Nil(t, runtimeInterface1.programs[contractLocationID])

		require.Equal(t, locationAccessCounts{}, programGets1)
		// NOTE: deployed contract is *correctly* *NOT* set,
		// as contract deployments and updates are delayed to the end of the transaction,
		// so should not influence program storage
		require.Equal(t, locationAccessCounts{txLocationID: 1}, programSets1)

		// Deploy to second

		err = executeTransaction2(addTx)
		require.NoError(t, err)
		require.Nil(t, runtimeInterface2.programs[contractLocationID])
		require.Equal(t, locationAccessCounts{}, programGets2)
		// See NOTE above
		require.Equal(t, locationAccessCounts{txLocationID: 1}, programSets2)
	})

	t.Run("Import only on second", func(t *testing.T) {

		clearLocationAccessCounts()

		txLocationID := common.TransactionLocation{1}.ID()

		importTx := fmt.Sprintf(
			`
              import %s from %s

              transaction {
                  prepare(signer: AuthAccount) {}
              }
            `,
			name,
			address.ShortHexWithPrefix(),
		)

		err := executeTransaction2(importTx)
		require.NoError(t, err)

		// only ran import TX against second,
		// so first should not have the program
		assert.Nil(t, runtimeInterface1.programs[contractLocationID])

		// NOTE: program in cache of second
		assert.NotNil(t, runtimeInterface2.programs[contractLocationID])

		assert.Equal(t,
			locationAccessCounts{
				contractLocationID: 1,
			},
			programGets2,
		)

		// NOTE: program was set after it was got
		assert.Equal(
			t,
			locationAccessCounts{
				contractLocationID: 1,
				txLocationID:       1,
			},
			programSets2,
		)
	})

	t.Run("Update on both", func(t *testing.T) {

		clearLocationAccessCounts()

		txLocationID1 := common.TransactionLocation{1}.ID()
		// second has seen an additional transaction (import, above)
		txLocationID2 := common.TransactionLocation{2}.ID()

		updateTx := newContractUpdateTransaction(name, newCode)

		// Update on first

		err := executeTransaction1(updateTx)
		require.NoError(t, err)

		// NOTE: the program was not available in the cache (no successful get).
		// So the old code is parsed and checked – and *MUST* be set!

		assert.Equal(t,
			locationAccessCounts{},
			programGets1,
		)
		assert.Equal(
			t,
			locationAccessCounts{
				contractLocationID: 1,
				txLocationID1:      1,
			},
			programSets1,
		)

		// Update on second

		err = executeTransaction2(updateTx)
		require.NoError(t, err)

		// NOTE: the program was available in the cache (successful get).
		// So the old code is parsed and checked, and does not need to be set.

		assert.Equal(t,
			locationAccessCounts{
				contractLocationID: 1,
			},
			programGets2,
		)
		assert.Equal(
			t,
			locationAccessCounts{
				txLocationID2: 1,
			},
			programSets2,
		)
	})
}
