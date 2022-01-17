/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

func TestRuntimeContractUpdateValidation(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime(
		WithContractUpdateValidationEnabled(true),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	newContractRemovalTransaction := func(contractName string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s")
				}
			}`,
			sema.AuthAccountContractsTypeRemoveFunctionName,
			contractName,
		))
	}

	accountCode := map[common.LocationID][]byte{}
	var events []cadence.Event
	runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
	nextTransactionLocation := newTransactionLocationGenerator()

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {
		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("change field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test1 {
				pub var a: String
				init() {
					self.a = "hello"
				}
      		}`

		const newCode = `
			pub contract Test1 {
				pub var a: Int
				init() {
					self.a = 0
				}
			}`

		err := deployAndUpdate(t, "Test1", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test1")
		assertFieldTypeMismatchError(t, cause, "Test1", "a", "String", "Int")
	})

	t.Run("add field", func(t *testing.T) {
		const oldCode = `
      		pub contract Test2 {
          		pub var a: String
				init() {
					self.a = "hello"
				}
      		}`

		const newCode = `
			pub contract Test2 {
				pub var a: String
				pub var b: Int
				init() {
					self.a = "hello"
					self.b = 0
				}
			}`

		err := deployAndUpdate(t, "Test2", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test2")
		assertExtraneousFieldError(t, cause, "Test2", "b")
	})

	t.Run("remove field", func(t *testing.T) {
		const oldCode = `
			pub contract Test3 {
				pub var a: String
				pub var b: Int
				init() {
					self.a = "hello"
					self.b = 0
				}
			}`

		const newCode = `
			pub contract Test3 {
				pub var a: String

				init() {
					self.a = "hello"
				}
			}`

		err := deployAndUpdate(t, "Test3", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("change nested decl field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test4 {

				pub var a: @TestResource

				init() {
					self.a <- create Test4.TestResource()
				}

				pub resource TestResource {

					pub let b: Int

					init() {
						self.b = 1234
					}
				}
			}`

		const newCode = `
			pub contract Test4 {

				pub var a: @Test4.TestResource

				init() {
					self.a <- create Test4.TestResource()
				}

				pub resource TestResource {

					pub let b: String

					init() {
						self.b = "string_1234"
					}
				}
			}`

		err := deployAndUpdate(t, "Test4", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test4")
		assertFieldTypeMismatchError(t, cause, "TestResource", "b", "Int", "String")
	})

	t.Run("add field to nested decl", func(t *testing.T) {
		const oldCode = `
			pub contract Test5 {

				pub var a: @TestResource

				init() {
					self.a <- create Test5.TestResource()
				}

				pub resource TestResource {

					pub var b: String

					init() {
						self.b = "hello"
					}
				}
			}`

		const newCode = `
			pub contract Test5 {

				pub var a: @Test5.TestResource

				init() {
					self.a <- create Test5.TestResource()
				}

				pub resource TestResource {

					pub var b: String
					pub var c: Int

					init() {
						self.b = "hello"
						self.c = 0
					}
				}
			}`

		err := deployAndUpdate(t, "Test5", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test5")
		assertExtraneousFieldError(t, cause, "TestResource", "c")
	})

	t.Run("change indirect field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test6 {

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
			}`

		const newCode = `
			pub contract Test6 {

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
			}`

		err := deployAndUpdate(t, "Test6", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test6")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "b", "Int", "String")
	})

	t.Run("circular types refs", func(t *testing.T) {
		const oldCode = `
			pub contract Test7{

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
			}`

		const newCode = `
			pub contract Test7 {

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
			}`

		err := deployAndUpdate(t, "Test7", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test7")
		assertFieldTypeMismatchError(t, cause, "Bar", "d", "Bar?", "String")
	})

	t.Run("qualified vs unqualified nominal type", func(t *testing.T) {
		const oldCode = `
			pub contract Test8 {

				pub var x: Test8.TestStruct
				pub var y: TestStruct

				init() {
					self.x = Test8.TestStruct()
					self.y = TestStruct()
				}

				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		const newCode = `
			pub contract Test8 {

				pub var x: TestStruct
				pub var y: Test8.TestStruct

				init() {
					self.x = TestStruct()
					self.y = Test8.TestStruct()
				}

				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		err := deployAndUpdate(t, "Test8", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("change imported nominal type to local", func(t *testing.T) {
		const importCode = `
			pub contract Test9Import {

				pub struct TestStruct {
					pub let a: Int
					pub var b: Int

					init() {
						self.a = 123
						self.b = 456
					}
				}
			}`

		deployTx1 := newDeployTransaction(
			sema.AuthAccountContractsTypeAddFunctionName,
			"Test9Import",
			importCode,
		)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const oldCode = `
			import Test9Import from 0x42

			pub contract Test9 {

				pub var x: Test9Import.TestStruct

				init() {
					self.x = Test9Import.TestStruct()
				}
			}`

		const newCode = `
			pub contract Test9 {

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
			}`

		err = deployAndUpdate(t, "Test9", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test9")
		assertFieldTypeMismatchError(t, cause, "Test9", "x", "Test9Import.TestStruct", "TestStruct")
	})

	t.Run("contract interface update", func(t *testing.T) {
		const oldCode = `
			pub contract interface Test10 {
				pub var a: String
				pub fun getA() : String
			}`

		const newCode = `
			pub contract interface Test10 {
				pub var a: Int
				pub fun getA() : Int
			}`

		err := deployAndUpdate(t, "Test10", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test10")
		assertFieldTypeMismatchError(t, cause, "Test10", "a", "String", "Int")
	})

	t.Run("convert interface to contract", func(t *testing.T) {
		const oldCode = `
			pub contract interface Test11 {
				pub var a: String
				pub fun getA() : String
			}`

		const newCode = `
			pub contract Test11 {

				pub var a: String

				init() {
					self.a = "hello"
				}

				pub fun getA() : String {
					return self.a
				}
			}`

		err := deployAndUpdate(t, "Test11", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test11")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test11",
			common.DeclarationKindContractInterface,
			common.DeclarationKindContract,
		)
	})

	t.Run("convert contract to interface", func(t *testing.T) {
		const oldCode = `
			pub contract Test12 {

				pub var a: String

				init() {
					self.a = "hello"
				}

				pub fun getA() : String {
					return self.a
				}
			}`

		const newCode = `
			pub contract interface Test12 {
				pub var a: String
				pub fun getA() : String
			}`

		err := deployAndUpdate(t, "Test12", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test12")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test12",
			common.DeclarationKindContract,
			common.DeclarationKindContractInterface,
		)
	})

	t.Run("change non stored", func(t *testing.T) {
		const oldCode = `
			pub contract Test13 {

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
			}`

		const newCode = `
			pub contract Test13 {

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
			}`

		err := deployAndUpdate(t, "Test13", oldCode, newCode)

		// Changing unused public composite types should also fail, since those could be
		// referred by anyone in the chain, and may cause data inconsistency.
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test13")
		assertFieldTypeMismatchError(t, cause, "UnusedStruct", "a", "Int", "String")
	})

	t.Run("change enum type", func(t *testing.T) {
		const oldCode = `
			pub contract Test14 {

				pub var x: Foo

				init() {
					self.x = Foo.up
				}

				pub enum Foo: UInt8 {
					pub case up
					pub case down
				}
			}`

		const newCode = `
			pub contract Test14 {

				pub var x: Foo

				init() {
					self.x = Foo.up
				}

				pub enum Foo: UInt128 {
					pub case up
					pub case down
				}
			}`

		err := deployAndUpdate(t, "Test14", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test14")
		assertConformanceMismatchError(t, cause, "Foo", "UInt8", "UInt128")
	})

	t.Run("change nested interface", func(t *testing.T) {
		const oldCode = `
			pub contract Test15 {

				pub var x: AnyStruct{TestStruct}?

				init() {
					self.x = nil
				}

				pub struct interface TestStruct {
					pub let a: String
					pub var b: Int
				}
			}`

		const newCode = `
			pub contract Test15 {

				pub var x: AnyStruct{TestStruct}?

				init() {
					self.x = nil
				}

				pub struct interface TestStruct {
					pub let a: Int
					pub var b: Int
				}
			}`

		err := deployAndUpdate(t, "Test15", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test15")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "String", "Int")
	})

	t.Run("change nested interface to struct", func(t *testing.T) {
		const oldCode = `
			pub contract Test16 {
				pub struct interface TestStruct {
					pub var a: Int
				}
			}`

		const newCode = `
			pub contract Test16 {
				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		err := deployAndUpdate(t, "Test16", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test16")
		assertDeclTypeChangeError(
			t,
			cause,
			"TestStruct",
			common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
		)
	})

	t.Run("adding a nested struct", func(t *testing.T) {
		const oldCode = `
			pub contract Test17 {
			}`

		const newCode = `
			pub contract Test17 {
				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		err := deployAndUpdate(t, "Test17", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("removing a nested struct", func(t *testing.T) {
		const oldCode = `
			pub contract Test18 {
				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		const newCode = `
			pub contract Test18 {
			}`

		err := deployAndUpdate(t, "Test18", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test18")
		assertMissingCompositeDeclarationError(t, cause, "TestStruct")
	})

	t.Run("add and remove field", func(t *testing.T) {
		const oldCode = `
			pub contract Test19 {
				pub var a: String
				init() {
					self.a = "hello"
				}
			}`

		const newCode = `
			pub contract Test19 {
				pub var b: Int
				init() {
					self.b = 0
				}
			}`

		err := deployAndUpdate(t, "Test19", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test19")
		assertExtraneousFieldError(t, cause, "Test19", "b")
	})

	t.Run("multiple errors", func(t *testing.T) {
		const oldCode = `
			pub contract Test20 {
				pub var a: String

				init() {
					self.a = "hello"
				}

				pub struct interface TestStruct {
					pub var a: Int
				}
			}`

		const newCode = `
			pub contract Test20 {
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
			}`

		err := deployAndUpdate(t, "Test20", oldCode, newCode)
		require.Error(t, err)

		updateErr := getContractUpdateError(t, err)
		require.NotNil(t, updateErr)
		assert.Equal(t, fmt.Sprintf("cannot update contract `%s`", "Test20"), updateErr.Error())

		childErrors := updateErr.ChildErrors()
		require.Equal(t, 3, len(childErrors))

		assertFieldTypeMismatchError(t, childErrors[0], "Test20", "a", "String", "Int")

		assertExtraneousFieldError(t, childErrors[1], "Test20", "b")

		assertDeclTypeChangeError(
			t,
			childErrors[2],
			"TestStruct",
			common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
		)
	})

	t.Run("check error messages", func(t *testing.T) {
		const oldCode = `
            pub contract Test21 {
                pub var a: String

                init() {
                    self.a = "hello"
                }

                pub struct interface TestStruct {
                    pub var a: Int
                }
            }`

		const newCode = `
            pub contract Test21 {
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
            }`

		err := deployAndUpdate(t, "Test21", oldCode, newCode)
		require.Error(t, err)

		const expectedError = "error: mismatching field `a` in `Test21`\n" +
			" --> 0000000000000042.Test21:3:27\n" +
			"  |\n" +
			"3 |                 pub var a: Int\n" +
			"  |                            ^^^ incompatible type annotations. expected `String`, found `Int`\n" +
			"\n" +
			"error: found new field `b` in `Test21`\n" +
			" --> 0000000000000042.Test21:4:24\n" +
			"  |\n" +
			"4 |                 pub var b: String\n" +
			"  |                         ^\n" +
			"\n" +
			"error: trying to convert structure interface `TestStruct` to a structure\n" +
			"  --> 0000000000000042.Test21:11:27\n" +
			"   |\n" +
			"11 |                 pub struct TestStruct {\n" +
			"   |                            ^^^^^^^^^^"

		require.Contains(t, err.Error(), expectedError)
	})

	t.Run("Test reference types", func(t *testing.T) {
		const oldCode = `
			pub contract Test22 {

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
			}`

		const newCode = `
			pub contract Test22 {

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
			}`

		err := deployAndUpdate(t, "Test22", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test function type", func(t *testing.T) {
		const oldCode = `
			pub contract Test23 {

				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		const newCode = `
			pub contract Test23 {

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
			}`

		err := deployAndUpdate(t, "Test23", oldCode, newCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error: field add has non-storable type: ((Int, Int): Int)")
	})

	t.Run("Test conformance", func(t *testing.T) {
		const importCode = `
			pub contract Test24Import {
				pub struct interface AnInterface {
					pub a: Int
				}
			}`

		deployTx1 := newDeployTransaction("add", "Test24Import", importCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const oldCode = `
			import Test24Import from 0x42

			pub contract Test24 {
				pub struct TestStruct1 {
					pub let a: Int
					init() {
						self.a = 123
					}
				}

				pub struct TestStruct2: Test24Import.AnInterface {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		const newCode = `
			import Test24Import from 0x42

			pub contract Test24 {

				pub struct TestStruct2: Test24Import.AnInterface {
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
			}`

		err = deployAndUpdate(t, "Test24", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test all types", func(t *testing.T) {

		const oldCode = `
			pub contract Test25 {
				// simple nominal type
				pub var a: TestStruct

				// qualified nominal type
				pub var b: Test25.TestStruct

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
					self.b = Test25.TestStruct()
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
			}`

		const newCode = `
			pub contract Test25 {


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
				pub var b: Test25.TestStruct

				// simple nominal type
				pub var a: TestStruct

				init() {
					var count: Int = 567
					self.a = TestStruct()
					self.b = Test25.TestStruct()
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
			}`

		err := deployAndUpdate(t, "Test25", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test restricted types", func(t *testing.T) {

		const oldCode = `
			pub contract Test26 {

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
			}`

		const newCode = `
			pub contract Test26 {
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
			}`

		err := deployAndUpdate(t, "Test26", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("Test invalid restricted types change", func(t *testing.T) {

		const oldCode = `
			pub contract Test27 {

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
			}`

		const newCode = `
			pub contract Test27 {
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
			}`

		err := deployAndUpdate(t, "Test27", oldCode, newCode)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "pub var a: {TestInterface}"+
			"\n  |                ^^^^^^^^^^^^^^^ "+
			"incompatible type annotations. expected `TestStruct{TestInterface}`, found `{TestInterface}`")

		assert.Contains(t, err.Error(), "pub var b: TestStruct{TestInterface}"+
			"\n  |                ^^^^^^^^^^^^^^^^^^^^^^^^^ "+
			"incompatible type annotations. expected `{TestInterface}`, found `TestStruct{TestInterface}`")
	})

	t.Run("enum valid", func(t *testing.T) {
		const oldCode = `
			pub contract Test28 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
				}
			}`

		const newCode = `
			pub contract Test28 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
				}
			}`

		err := deployAndUpdate(t, "Test28", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("enum remove case", func(t *testing.T) {
		const oldCode = `
			pub contract Test29 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
				}
			}`

		const newCode = `
			pub contract Test29 {
				pub enum Foo: UInt8 {
					pub case up
				}
			}`

		err := deployAndUpdate(t, "Test29", oldCode, newCode)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test29")
		assertMissingEnumCasesError(t, cause, "Foo", 2, 1)
	})

	t.Run("enum add case", func(t *testing.T) {
		const oldCode = `
			pub contract Test30 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
				}
			}`

		const newCode = `
			pub contract Test30 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
					pub case left
				}
			}`

		err := deployAndUpdate(t, "Test30", oldCode, newCode)
		require.NoError(t, err)
	})

	t.Run("enum swap cases", func(t *testing.T) {
		const oldCode = `
			pub contract Test31 {
				pub enum Foo: UInt8 {
					pub case up
					pub case down
					pub case left
				}
			}`

		const newCode = `
			pub contract Test31 {
				pub enum Foo: UInt8 {
					pub case down
					pub case left
					pub case up
				}
			}`

		err := deployAndUpdate(t, "Test31", oldCode, newCode)
		require.Error(t, err)

		updateErr := getContractUpdateError(t, err)
		require.NotNil(t, updateErr)
		assert.Equal(t, fmt.Sprintf("cannot update contract `%s`", "Test31"), updateErr.Error())

		childErrors := updateErr.ChildErrors()
		require.Equal(t, 3, len(childErrors))

		assertEnumCaseMismatchError(t, childErrors[0], "up", "down")
		assertEnumCaseMismatchError(t, childErrors[1], "down", "left")
		assertEnumCaseMismatchError(t, childErrors[2], "left", "up")
	})

	t.Run("Remove and add struct", func(t *testing.T) {
		const oldCode = `
			pub contract Test32 {

				pub struct TestStruct {
					pub let a: Int
					pub var b: Int

					init() {
						self.a = 123
						self.b = 456
					}
				}
			}`

		const updateCode1 = `
			pub contract Test32 {
			}`

		err := deployAndUpdate(t, "Test32", oldCode, updateCode1)
		require.Error(t, err)

		cause := getErrorCause(t, err, "Test32")
		assertMissingCompositeDeclarationError(t, cause, "TestStruct")

		const updateCode2 = `
			pub contract Test32 {

				pub struct TestStruct {
					pub let a: String

					init() {
						self.a = "hello123"
					}
				}
			}`

		updateTx := newDeployTransaction(
			sema.AuthAccountContractsTypeUpdateExperimentalFunctionName,
			"Test32",
			updateCode2,
		)

		err = runtime.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		cause = getErrorCause(t, err, "Test32")

		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "Int", "String")
	})

	t.Run("Rename struct", func(t *testing.T) {
		const oldCode = `
			pub contract Test33 {

				pub struct TestStruct {
					pub let a: Int
					pub var b: Int

					init() {
						self.a = 123
						self.b = 456
					}
				}
			}`

		err := runtime.ExecuteTransaction(
			Script{
				Source: newDeployTransaction(
					sema.AuthAccountContractsTypeAddFunctionName,
					"Test33",
					oldCode),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		// Rename the struct

		const newCode = `
			pub contract Test33 {

				pub struct TestStructRenamed {
					pub let a: Int
					pub var b: Int

					init() {
						self.a = 123
						self.b = 456
					}
				}
			}`

		err = runtime.ExecuteTransaction(
			Script{
				Source: newDeployTransaction(
					sema.AuthAccountContractsTypeUpdateExperimentalFunctionName,
					"Test33",
					newCode,
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		cause := getErrorCause(t, err, "Test33")
		assertMissingCompositeDeclarationError(t, cause, "TestStruct")
	})

	t.Run("Remove contract with enum", func(t *testing.T) {
		// Add contract
		const oldCode = `
			pub contract Test34 {
				pub enum TestEnum: Int {
				}
			}`

		err := runtime.ExecuteTransaction(
			Script{
				Source: newDeployTransaction(
					sema.AuthAccountContractsTypeAddFunctionName,
					"Test34",
					oldCode,
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		// Remove the added contract.
		err = runtime.ExecuteTransaction(
			Script{
				Source: newContractRemovalTransaction("Test34"),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		require.IsType(t, Error{}, err)
		runtimeError := err.(Error)

		require.IsType(t, interpreter.Error{}, runtimeError.Err)
		interpreterError := runtimeError.Err.(interpreter.Error)
		cause := interpreterError.Err

		require.IsType(t, &ContractRemovalError{}, cause)
		contractRemovalError := cause.(*ContractRemovalError)

		assert.Equal(
			t,
			fmt.Sprintf("cannot remove contract `%s`", "Test34"),
			contractRemovalError.Error(),
		)
	})

	t.Run("Remove contract interface with enum", func(t *testing.T) {
		// Add contract
		const oldCode = `
			pub contract interface Test35 {
				pub enum TestEnum: Int {
				}
			}`

		err := runtime.ExecuteTransaction(
			Script{
				Source: newDeployTransaction(
					sema.AuthAccountContractsTypeAddFunctionName,
					"Test35",
					oldCode,
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		// Remove the added contract.
		err = runtime.ExecuteTransaction(
			Script{
				Source: newContractRemovalTransaction("Test35"),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		require.IsType(t, Error{}, err)
		runtimeError := err.(Error)

		require.IsType(t, interpreter.Error{}, runtimeError.Err)
		interpreterError := runtimeError.Err.(interpreter.Error)
		cause := interpreterError.Err

		require.IsType(t, &ContractRemovalError{}, cause)
		contractRemovalError := cause.(*ContractRemovalError)

		assert.Equal(
			t,
			fmt.Sprintf("cannot remove contract `%s`", "Test35"),
			contractRemovalError.Error(),
		)
	})

	t.Run("Remove contract without enum", func(t *testing.T) {
		// Add contract
		const oldCode = `
			pub contract Test36 {
				pub struct TestStruct {
					pub let a: Int

					init() {
						self.a = 123
					}
				}
			}`

		err := runtime.ExecuteTransaction(
			Script{
				Source: newDeployTransaction(
					sema.AuthAccountContractsTypeAddFunctionName,
					"Test36",
					oldCode,
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		// Remove the added contract.
		err = runtime.ExecuteTransaction(
			Script{
				Source: newContractRemovalTransaction("Test36"),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		assert.NoError(t, err)
	})
}

func assertDeclTypeChangeError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	oldKind common.DeclarationKind,
	newKind common.DeclarationKind,
) {

	require.Error(t, err)
	require.IsType(t, &InvalidDeclarationKindChangeError{}, err)
	declTypeChangeError := err.(*InvalidDeclarationKindChangeError)
	assert.Equal(
		t,
		fmt.Sprintf("trying to convert %s `%s` to a %s", oldKind.Name(), erroneousDeclName, newKind.Name()),
		declTypeChangeError.Error(),
	)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	require.Error(t, err)
	require.IsType(t, &ExtraneousFieldError{}, err)
	extraFieldError := err.(*ExtraneousFieldError)
	assert.Equal(t, fmt.Sprintf("found new field `%s` in `%s`", fieldName, erroneousDeclName), extraFieldError.Error())
}

func assertFieldTypeMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	fieldName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &FieldMismatchError{}, err)
	fieldMismatchError := err.(*FieldMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("mismatching field `%s` in `%s`", fieldName, erroneousDeclName),
		fieldMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, fieldMismatchError.Err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		fieldMismatchError.Err.Error(),
	)
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	expectedType string,
	foundType string,
) {

	require.Error(t, err)
	require.IsType(t, &ConformanceMismatchError{}, err)
	conformanceMismatchError := err.(*ConformanceMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("conformances does not match in `%s`", erroneousDeclName),
		conformanceMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, conformanceMismatchError.Err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		conformanceMismatchError.Err.Error(),
	)
}

func assertEnumCaseMismatchError(t *testing.T, err error, expectedEnumCase string, foundEnumCase string) {
	require.Error(t, err)
	require.IsType(t, &EnumCaseMismatchError{}, err)
	enumMismatchError := err.(*EnumCaseMismatchError)

	assert.Equal(
		t,
		fmt.Sprintf(
			"mismatching enum case: expected `%s`, found `%s`",
			expectedEnumCase,
			foundEnumCase,
		),
		enumMismatchError.Error(),
	)
}

func assertMissingEnumCasesError(t *testing.T, err error, declName string, expectedCases int, foundCases int) {
	require.Error(t, err)
	require.IsType(t, &MissingEnumCasesError{}, err)
	missingEnumCasesError := err.(*MissingEnumCasesError)
	assert.Equal(
		t,
		fmt.Sprintf(
			"missing cases in enum `%s`: expected %d or more, found %d",
			declName,
			expectedCases,
			foundCases,
		),
		missingEnumCasesError.Error(),
	)
}

func assertMissingCompositeDeclarationError(t *testing.T, err error, declName string) {
	require.Error(t, err)

	require.IsType(t, &MissingCompositeDeclarationError{}, err)
	missingDeclError := err.(*MissingCompositeDeclarationError)

	assert.Equal(
		t,
		fmt.Sprintf("missing composite declaration `%s`", declName),
		missingDeclError.Error(),
	)
}

func getErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err)
	assert.Equal(t, fmt.Sprintf("cannot update contract `%s`", contractName), updateErr.Error())

	require.Equal(t, 1, len(updateErr.ChildErrors()))
	childError := updateErr.ChildErrors()[0]

	return childError
}

func getContractUpdateError(t *testing.T, err error) *ContractUpdateError {
	require.Error(t, err)
	require.IsType(t, Error{}, err)
	runtimeError := err.(Error)

	require.IsType(t, interpreter.Error{}, runtimeError.Err)
	interpreterError := runtimeError.Err.(interpreter.Error)

	require.IsType(t, &InvalidContractDeploymentError{}, interpreterError.Err)
	deploymentError := interpreterError.Err.(*InvalidContractDeploymentError)

	require.IsType(t, &ContractUpdateError{}, deploymentError.Err)
	return deploymentError.Err.(*ContractUpdateError)
}

func getMockedRuntimeInterfaceForTxUpdate(
	t *testing.T,
	accountCodes map[common.LocationID][]byte,
	events []cadence.Event,
) *testRuntimeInterface {

	return &testRuntimeInterface{
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
}

func TestContractUpdateValidationDisabled(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime(
		WithContractUpdateValidationEnabled(false),
	)

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(`
			transaction {
				prepare(signer: AuthAccount) {
					signer.contracts.%s(name: "%s", code: "%s".decodeHex())
				}
			}`,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	accountCode := map[common.LocationID][]byte{}
	var events []cadence.Event
	runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
	nextTransactionLocation := newTransactionLocationGenerator()

	deployAndUpdate := func(t *testing.T, name string, oldCode string, newCode string) error {
		deployTx1 := newDeployTransaction(sema.AuthAccountContractsTypeAddFunctionName, name, oldCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		deployTx2 := newDeployTransaction(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, name, newCode)
		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTx2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		return err
	}

	t.Run("change field type", func(t *testing.T) {
		const oldCode = `
			pub contract Test1 {
				pub var a: String
				init() {
					self.a = "hello"
				}
      		}`

		const newCode = `
			pub contract Test1 {
				pub var a: Int
				init() {
					self.a = 0
				}
			}`

		err := deployAndUpdate(t, "Test1", oldCode, newCode)
		require.NoError(t, err)
	})
}
