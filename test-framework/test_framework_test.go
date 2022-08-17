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

package test

import (
	"errors"
	"fmt"
	"github.com/onflow/cadence/runtime/stdlib"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
)

func TestRunningMultipleTests(t *testing.T) {
	t.Parallel()

	code := `
        pub fun testFunc1() {
            assert(false)
        }

        pub fun testFunc2() {
            assert(true)
        }
    `

	runner := NewTestRunner()
	results, err := runner.RunTests(code)
	require.NoError(t, err)

	require.Len(t, results, 2)

	result1 := results[0]
	assert.Equal(t, result1.testName, "testFunc1")
	assert.Error(t, result1.err)

	result2 := results[1]
	assert.Equal(t, result2.testName, "testFunc2")
	assert.NoError(t, result2.err)
}

func TestRunningSingleTest(t *testing.T) {
	t.Parallel()

	code := `
        pub fun testFunc1() {
            assert(false)
        }

        pub fun testFunc2() {
            assert(true)
        }
    `

	runner := NewTestRunner()

	result, err := runner.RunTest(code, "testFunc1")
	assert.NoError(t, err)
	assert.Error(t, result.err)

	result, err = runner.RunTest(code, "testFunc2")
	assert.NoError(t, err)
	assert.NoError(t, result.err)
}

func TestAssertFunction(t *testing.T) {
	t.Parallel()

	code := `
        import Test

        pub fun testAssertWithNoArgs() {
            Test.assert(true)
        }

        pub fun testAssertWithNoArgsFail() {
            Test.assert(false)
        }

        pub fun testAssertWithMessage() {
            Test.assert(true, "some reason")
        }

        pub fun testAssertWithMessageFail() {
            Test.assert(false, "some reason")
        }
    `

	runner := NewTestRunner()

	result, err := runner.RunTest(code, "testAssertWithNoArgs")
	assert.NoError(t, err)
	assert.NoError(t, result.err)

	result, err = runner.RunTest(code, "testAssertWithNoArgsFail")
	assert.NoError(t, err)
	require.Error(t, result.err)
	assert.Equal(t, result.err.Error(), "assertion failed")

	result, err = runner.RunTest(code, "testAssertWithMessage")
	assert.NoError(t, err)
	assert.NoError(t, result.err)

	result, err = runner.RunTest(code, "testAssertWithMessageFail")
	assert.NoError(t, err)
	require.Error(t, result.err)
	assert.Contains(t, result.err.Error(), "assertion failed: some reason")
}

func TestExecuteScript(t *testing.T) {
	t.Parallel()

	t.Run("no args", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var result = blockchain.executeScript("pub fun main(): Int {  return 2 + 3 }", [])

                assert(result.status == Test.ResultStatus.succeeded)
                assert((result.returnValue! as! Int) == 5)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("with args", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var result = blockchain.executeScript(
                    "pub fun main(a: Int, b: Int): Int {  return a + b }",
                    [2, 3]
                )

                assert(result.status == Test.ResultStatus.succeeded)
                assert((result.returnValue! as! Int) == 5)
        }
    `
		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})
}

func TestImportContract(t *testing.T) {
	t.Parallel()

	t.Run("init no params", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {
                var foo = FooContract()
                var result = foo.sayHello()
                assert(result == "hello from Foo")
            }
        `

		fooContract := `
            pub contract FooContract {
                init() {}

                pub fun sayHello(): String {
                    return "hello from Foo"
                }
            }
        `

		importResolver := func(location common.Location) (string, error) {
			return fooContract, nil
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("init with params", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {
                var foo = FooContract(greeting: "hello from Foo")
                var result = foo.sayHello()
                assert(result == "hello from Foo")
            }
        `

		fooContract := `
            pub contract FooContract {

                pub var greeting: String

                init(greeting: String) {
                    self.greeting = greeting
                }

                pub fun sayHello(): String {
                    return self.greeting
                }
            }
        `

		importResolver := func(location common.Location) (string, error) {
			return fooContract, nil
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("invalid import", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {
                var foo = FooContract()
            }
        `

		importResolver := func(location common.Location) (string, error) {
			return "", errors.New("cannot load file")
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		_, err := runner.RunTest(code, "test")
		require.Error(t, err)

		errs := checker.ExpectCheckerErrors(t, err, 2)

		importedProgramError := &sema.ImportedProgramError{}
		assert.ErrorAs(t, errs[0], &importedProgramError)
		assert.Contains(t, importedProgramError.Err.Error(), "cannot load file")

		assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})

	t.Run("import resolver not provided", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {
                var foo = FooContract()
            }
        `

		runner := NewTestRunner()
		_, err := runner.RunTest(code, "test")
		require.Error(t, err)

		errs := checker.ExpectCheckerErrors(t, err, 2)

		importedProgramError := &sema.ImportedProgramError{}
		require.ErrorAs(t, errs[0], &importedProgramError)
		assert.IsType(t, ImportResolverNotProvidedError{}, importedProgramError.Err)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
	})

	t.Run("nested imports", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {}
        `

		fooContract := `
           import BarContract from 0x01

            pub contract FooContract {
                init() {}
            }
        `
		barContract := `
            pub contract BarContract {
                init() {}
            }
        `

		importResolver := func(location common.Location) (string, error) {
			switch location := location.(type) {
			case common.StringLocation:
				if location == "./FooContract" {
					return fooContract, nil
				}
			case common.AddressLocation:
				if location.ID() == "A.0000000000000001.BarContract" {
					return barContract, nil
				}
			}

			return "", fmt.Errorf("unsupported import %s", location.ID())
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		_, err := runner.RunTest(code, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nested imports are not supported")
	})
}

func TestUsingEnv(t *testing.T) {
	t.Parallel()

	t.Run("public key creation", func(t *testing.T) {
		t.Parallel()

		code := `
            pub fun test() {
                var publicKey = PublicKey(
                    publicKey: "1234".decodeHex(),
                    signatureAlgorithm: SignatureAlgorithm.ECDSA_secp256k1
                )
            }
        `

		runner := NewTestRunner()

		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)

		require.Error(t, result.err)
		publicKeyError := interpreter.InvalidPublicKeyError{}
		assert.ErrorAs(t, result.err, &publicKeyError)
	})

	t.Run("public account", func(t *testing.T) {
		t.Parallel()

		code := `
            pub fun test() {
                var acc = getAccount(0x01)
                var bal = acc.balance
                assert(acc.balance == 0.0)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("auth account", func(t *testing.T) {
		t.Parallel()

		code := `
            pub fun test() {
                var acc = getAuthAccount(0x01)
                var bal = acc.balance
                assert(acc.balance == 0.0)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	// Imported programs also should have the access to the env.
	t.Run("account access in imported program", func(t *testing.T) {
		t.Parallel()

		code := `
            import FooContract from "./FooContract"

            pub fun test() {
                var foo = FooContract()
                var result = foo.getBalance()
                assert(result == 0.0)
            }
        `

		fooContract := `
            pub contract FooContract {
                init() {}

                pub fun getBalance(): UFix64 {
                    var acc = getAccount(0x01)
                    return acc.balance
                }
            }
        `

		importResolver := func(location common.Location) (string, error) {
			return fooContract, nil
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()

	code := `
        import Test

        pub fun test() {
            var blockchain = Test.newEmulatorBlockchain()
            var account = blockchain.createAccount()
        }
    `

	runner := NewTestRunner()
	result, err := runner.RunTest(code, "test")
	assert.NoError(t, err)
	assert.NoError(t, result.err)
}

func TestExecutingTransactions(t *testing.T) {
	t.Parallel()

	t.Run("add transaction", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(false) } }",
                    authorizers: [account.address],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run next transaction", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(true) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx)

                let result = blockchain.executeNextTransaction()!
                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run next transaction with authorizer", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { prepare(acct: AuthAccount) {} execute{ assert(true) } }",
                    authorizers: [account.address],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx)

                let result = blockchain.executeNextTransaction()!
                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("transaction failure", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(false) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx)

                let result = blockchain.executeNextTransaction()!
                assert(result.status == Test.ResultStatus.failed)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run non existing transaction", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                let result = blockchain.executeNextTransaction()
                assert(result == nil)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("commit block", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                blockchain.commitBlock()
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("commit un-executed block", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(false) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx)

                blockchain.commitBlock()
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)

		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "cannot be committed before execution")
	})

	t.Run("commit partially executed block", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(false) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                // Add two transactions
                blockchain.addTransaction(tx)
                blockchain.addTransaction(tx)

                // But execute only one
                blockchain.executeNextTransaction()

                // Then try to commit
                blockchain.commitBlock()
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)

		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "is currently being executed")
	})

	t.Run("multiple commit block", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                blockchain.commitBlock()
                blockchain.commitBlock()
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run given transaction", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(true) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                let result = blockchain.executeTransaction(tx)
                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run transaction with args", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction(a: Int, b: Int) { execute{ assert(a == b) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [4, 4],
                )

                let result = blockchain.executeTransaction(tx)
                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run transaction with multiple authorizers", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account1 = blockchain.createAccount()
                var account2 = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction() { prepare(acct1: AuthAccount, acct2: AuthAccount) {}  }",
                    authorizers: [account1.address, account2.address],
                    signers: [account1, account2],
                    arguments: [],
                )

                let result = blockchain.executeTransaction(tx)
                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run given transaction unsuccessful", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx = Test.Transaction(
                    code: "transaction { execute{ assert(fail) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                let result = blockchain.executeTransaction(tx)
                assert(result.status == Test.ResultStatus.failed)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run multiple transactions", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx1 = Test.Transaction(
                    code: "transaction { execute{ assert(true) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                let tx2 = Test.Transaction(
                    code: "transaction { prepare(acct: AuthAccount) {} execute{ assert(true) } }",
                    authorizers: [account.address],
                    signers: [account],
                    arguments: [],
                )

                let tx3 = Test.Transaction(
                    code: "transaction { execute{ assert(false) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                let firstResults = blockchain.executeTransactions([tx1, tx2, tx3])

                assert(firstResults.length == 3)
                assert(firstResults[0].status == Test.ResultStatus.succeeded)
                assert(firstResults[1].status == Test.ResultStatus.succeeded)
                assert(firstResults[2].status == Test.ResultStatus.failed)


                // Execute them again: To verify the proper increment/reset of sequence numbers.
                let secondResults = blockchain.executeTransactions([tx1, tx2, tx3])

                assert(secondResults.length == 3)
                assert(secondResults[0].status == Test.ResultStatus.succeeded)
                assert(secondResults[1].status == Test.ResultStatus.succeeded)
                assert(secondResults[2].status == Test.ResultStatus.failed)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run empty transactions", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let result = blockchain.executeTransactions([])
                assert(result.length == 0)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("run transaction with pending transactions", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                let tx1 = Test.Transaction(
                    code: "transaction { execute{ assert(true) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                blockchain.addTransaction(tx1)

                let tx2 = Test.Transaction(
                    code: "transaction { execute{ assert(true) } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )
                let result = blockchain.executeTransaction(tx2)

                assert(result.status == Test.ResultStatus.succeeded)
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)

		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "is currently being executed")
	})
}

func TestSetupAndTearDown(t *testing.T) {
	t.Parallel()

	t.Run("setup", func(t *testing.T) {
		t.Parallel()

		code := `
            pub(set) var setupRan = false

            pub fun setup() {
                assert(!setupRan)
                setupRan = true
            }

            pub fun testFunc() {
                assert(setupRan)
            }
        `

		runner := NewTestRunner()
		results, err := runner.RunTests(code)
		require.NoError(t, err)

		require.Len(t, results, 1)
		result := results[0]
		assert.Equal(t, result.testName, "testFunc")
		assert.NoError(t, result.err)
	})

	t.Run("setup failed", func(t *testing.T) {
		t.Parallel()

		code := `
            pub fun setup() {
                panic("error occurred")
            }

            pub fun testFunc() {
                assert(true)
            }
        `

		runner := NewTestRunner()
		results, err := runner.RunTests(code)
		require.Error(t, err)
		require.Empty(t, results)
	})

	t.Run("teardown", func(t *testing.T) {
		t.Parallel()

		code := `
            pub(set) var tearDownRan = false

            pub fun testFunc() {
                assert(!tearDownRan)
            }

            pub fun tearDown() {
                assert(true)
            }
        `

		runner := NewTestRunner()
		results, err := runner.RunTests(code)
		require.NoError(t, err)

		require.Len(t, results, 1)
		result := results[0]
		assert.Equal(t, result.testName, "testFunc")
		assert.NoError(t, result.err)
	})

	t.Run("teardown failed", func(t *testing.T) {
		t.Parallel()

		code := `
            pub(set) var tearDownRan = false

            pub fun testFunc() {
                assert(!tearDownRan)
            }

            pub fun tearDown() {
                assert(false)
            }
        `

		runner := NewTestRunner()
		results, err := runner.RunTests(code)

		// Running tests will return an error since the tear down failed.
		require.Error(t, err)

		// However, test cases should have been passed.
		require.Len(t, results, 1)
		result := results[0]
		assert.Equal(t, result.testName, "testFunc")
		assert.NoError(t, result.err)
	})
}

func TestPrettyPrintTestResults(t *testing.T) {
	t.Parallel()

	code := `
        import Test

        pub fun testFunc1() {
            Test.assert(true, "should pass")
        }

        pub fun testFunc2() {
            Test.assert(false, "unexpected error occurred")
        }

        pub fun testFunc3() {
            Test.assert(true, "should pass")
        }

        pub fun testFunc4() {
            panic("runtime error")
        }
    `

	runner := NewTestRunner()
	results, err := runner.RunTests(code)
	require.NoError(t, err)

	resultsStr := PrettyPrintResults(results)

	expected := `Test Results
- PASS: testFunc1
- FAIL: testFunc2
		assertion failed: unexpected error occurred
- PASS: testFunc3
- FAIL: testFunc4
		panic: runtime error
`

	assert.Equal(t, expected, resultsStr)
}

func TestLoadingProgramsFromLocalFile(t *testing.T) {
	t.Parallel()

	t.Run("read script", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                var script = Test.readFile("./sample/script.cdc")

                var result = blockchain.executeScript(script, [])

                assert(result.status == Test.ResultStatus.succeeded)
                assert((result.returnValue! as! Int) == 5)

                log(result.returnValue)
            }
        `

		const scriptCode = `
            pub fun main(): Int {
                return 2 + 3
            }
        `

		resolverInvoked := false
		importResolver := func(location common.Location) (string, error) {
			resolverInvoked = true

			stringLocation, ok := location.(common.StringLocation)
			assert.True(t, ok)
			assert.Equal(t, stringLocation.String(), "./sample/script.cdc")

			return scriptCode, nil
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)

		assert.True(t, resolverInvoked)
	})

	t.Run("read invalid", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                var script = Test.readFile("./sample/script.cdc")

                var result = blockchain.executeScript(script, [])

                assert(result.status == Test.ResultStatus.succeeded)
                assert((result.returnValue! as! Int) == 5)

                log(result.returnValue)
            }
        `

		resolverInvoked := false
		importResolver := func(location common.Location) (string, error) {
			resolverInvoked = true

			stringLocation, ok := location.(common.StringLocation)
			assert.True(t, ok)
			assert.Equal(t, stringLocation.String(), "./sample/script.cdc")

			return "", fmt.Errorf("cannot find file %s", location.String())
		}

		runner := NewTestRunner().WithImportResolver(importResolver)

		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "cannot find file ./sample/script.cdc")

		assert.True(t, resolverInvoked)
	})

	t.Run("no resolver set", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                var blockchain = Test.newEmulatorBlockchain()
                var account = blockchain.createAccount()

                var script = Test.readFile("./sample/script.cdc")
            }
        `

		runner := NewTestRunner()

		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &ImportResolverNotProvidedError{})
	})
}

func TestDeployingContracts(t *testing.T) {
	t.Parallel()

	t.Run("no args", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let contractCode = "pub contract Foo{ init(){}  pub fun sayHello(): String { return \"hello from Foo\"} }"

                let err = blockchain.deployContract(
                    name: "Foo",
                    code: contractCode,
                    account: account,
                    arguments: [],
                )

                if err != nil {
                    panic(err!.message)
                }

                var script = "import Foo from ".concat(account.address.toString()).concat("\n")
                script = script.concat("pub fun main(): String {  return Foo.sayHello() }")

                let result = blockchain.executeScript(script, [])

                if result.status != Test.ResultStatus.succeeded {
                    panic(result.error!.message)
                }

                let returnedStr = result.returnValue! as! String
                assert(returnedStr == "hello from Foo", message: "found: ".concat(returnedStr))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})

	t.Run("with args", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let contractCode = "pub contract Foo{ pub let msg: String;   init(_ msg: String){ self.msg = msg }   pub fun sayHello(): String { return self.msg } }" 

                let err = blockchain.deployContract(
                    name: "Foo",
                    code: contractCode,
                    account: account,
                    arguments: ["hello from args"],
                )

                if err != nil {
                    panic(err!.message)
                }

                var script = "import Foo from ".concat(account.address.toString()).concat("\n")
                script = script.concat("pub fun main(): String {  return Foo.sayHello() }")

                let result = blockchain.executeScript(script, [])

                if result.status != Test.ResultStatus.succeeded {
                    panic(result.error!.message)
                }

                let returnedStr = result.returnValue! as! String
                assert(returnedStr == "hello from args", message: "found: ".concat(returnedStr))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		assert.NoError(t, err)
		assert.NoError(t, result.err)
	})
}

func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("contract deployment error", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let contractCode = "pub contract Foo{ init(){}  pub fun sayHello() { return 0 } }"

                let err = blockchain.deployContract(
                    name: "Foo",
                    code: contractCode,
                    account: account,
                    arguments: [],
                )

                if err != nil {
                    panic(err!.message)
                }
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "cannot deploy invalid contract")
	})

	t.Run("script error", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let script = "import Foo from 0x01; pub fun main() {}"
                let result = blockchain.executeScript(script, [])

                if result.status == Test.ResultStatus.failed {
                    panic(result.error!.message)
                }
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "account not found for address")
	})

	t.Run("transaction error", func(t *testing.T) {
		t.Parallel()

		code := `
            import Test

            pub fun test() {
                let blockchain = Test.newEmulatorBlockchain()
                let account = blockchain.createAccount()

                let tx2 = Test.Transaction(
                    code: "transaction { execute{ panic(\"some error\") } }",
                    authorizers: [],
                    signers: [account],
                    arguments: [],
                )

                let result = blockchain.executeTransaction(tx2)!

                if result.status == Test.ResultStatus.failed {
                    panic(result.error!.message)
                }
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(code, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "panic: some error")
	})
}

func TestInterpretFailFunction(t *testing.T) {
	t.Parallel()

	script := `
        import Test

        pub fun test() {
            Test.fail()
        }
    `

	runner := NewTestRunner()
	result, err := runner.RunTest(script, "test")
	require.NoError(t, err)
	require.Error(t, result.err)
	assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
}

func TestInterpretMatcher(t *testing.T) {
	t.Parallel()

	t.Run("custom matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher(fun (_ value: AnyStruct): Bool {
                     if !value.getType().isSubtype(of: Type<Int>()) {
                        return false
                    }

                    return (value as! Int) > 5
                })

                assert(matcher.test(8))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("custom matcher primitive type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher(fun (_ value: Int): Bool {
                     return value == 7
                })

                assert(matcher.test(7))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("custom matcher invalid type usage", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher(fun (_ value: Int): Bool {
                     return (value + 7) == 4
                })

                // Invoke with an incorrect type
                assert(matcher.test("Hello"))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &interpreter.TypeMismatchError{})
	})

	t.Run("custom resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher(fun (_ value: &Foo): Bool {
                    return value.a == 4
                })

                let f <-create Foo(4)

                assert(matcher.test(&f as &Foo))

                destroy f
            }

            pub resource Foo {
                pub let a: Int

                init(_ a: Int) {
                    self.a = a
                }
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("custom resource matcher invalid type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher(fun (_ value: @Foo): Bool {
                     destroy value
                     return true
                })
            }

            pub resource Foo {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		errs := checker.ExpectCheckerErrors(t, result.err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("custom matcher with explicit type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher<Int>(fun (_ value: Int): Bool {
                     return value == 7
                })

                assert(matcher.test(7))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("custom matcher with mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher = Test.NewMatcher<String>(fun (_ value: Int): Bool {
                     return value == 7
                })
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		errs := checker.ExpectCheckerErrors(t, result.err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("combined matcher mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {

                let matcher1 = Test.NewMatcher(fun (_ value: Int): Bool {
                     return (value + 5) == 10
                })

                let matcher2 = Test.NewMatcher(fun (_ value: String): Bool {
                     return value.length == 10
                })

                let matcher3 = matcher1.and(matcher2)

                // Invoke with a type that matches to only one matcher
                assert(matcher3.test(5))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &interpreter.TypeMismatchError{})
	})

}

func TestInterpretEqualMatcher(t *testing.T) {

	t.Parallel()

	t.Run("equal matcher with primitive", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let matcher = Test.equal(1)
                assert(matcher.test(1))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("equal matcher with struct", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let f = Foo()
                let matcher = Test.equal(f)
                assert(matcher.test(f))
            }

            pub struct Foo {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("equal matcher with resource", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let f <- create Foo()
                let matcher = Test.equal(<-f)
                assert(matcher.test(<- create Foo()))
            }

            pub resource Foo {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let matcher = Test.equal<String>("hello")
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("with incorrect types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let matcher = Test.equal<String>(1)
            }
        `

		runner := NewTestRunner()
		_, err := runner.RunTest(script, "test")
		require.Error(t, err)

		errs := checker.ExpectCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneOrTwo = one.or(two)

                assert(oneOrTwo.test(1))
                assert(oneOrTwo.test(2))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("matcher or fail", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneOrTwo = one.or(two)

                assert(oneOrTwo.test(3))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneAndTwo = one.and(two)

                assert(oneAndTwo.test(1))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("chained matchers", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let one = Test.equal(1)
                let two = Test.equal(2)
                let three = Test.equal(3)

                let oneOrTwoOrThree = one.or(two).or(three)

                assert(oneOrTwoOrThree.test(3))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("resource matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let foo <- create Foo()
                let bar <- create Bar()

                let fooMatcher = Test.equal(<-foo)
                let barMatcher = Test.equal(<-bar)

                let matcher = fooMatcher.or(barMatcher)

                assert(matcher.test(<-create Foo()))
                assert(matcher.test(<-create Bar()))
            }

            pub resource Foo {}
            pub resource Bar {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("resource matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let foo <- create Foo()
                let bar <- create Bar()

                let fooMatcher = Test.equal(<-foo)
                let barMatcher = Test.equal(<-bar)

                let matcher = fooMatcher.and(barMatcher)

                assert(matcher.test(<-create Foo()))
            }

            pub resource Foo {}
            pub resource Bar {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})
}

func TestInterpretExpectFunction(t *testing.T) {

	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                Test.expect("this string", Test.equal("this string"))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                Test.expect("this string", Test.equal("other string"))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("different types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                Test.expect("string", Test.equal(1))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                Test.expect<String>("hello", Test.equal("hello"))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                Test.expect<Int>("string", Test.equal(1))
            }
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		errs := checker.ExpectCheckerErrors(t, result.err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let f1 <- create Foo()
                let f2 <- create Foo()
                Test.expect(<-f1, Test.equal(<-f2))
            }

            pub resource Foo {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.NoError(t, result.err)
	})

	t.Run("resource with a different resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let foo <- create Foo()
                let bar <- create Bar()
                Test.expect(<-foo, Test.equal(<-bar))
            }

            pub resource Foo {}
            pub resource Bar {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("resource with struct matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let foo <- create Foo()
                let bar = Bar()
                Test.expect(<-foo, Test.equal(bar))
            }

            pub resource Foo {}
            pub struct Bar {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun test() {
                let foo = Foo()
                let bar <- create Bar()
                Test.expect(foo, Test.equal(<-bar))
            }

            pub struct Foo {}
            pub resource Bar {}
        `

		runner := NewTestRunner()
		result, err := runner.RunTest(script, "test")
		require.NoError(t, err)
		require.Error(t, result.err)
		assert.ErrorAs(t, result.err, &stdlib.AssertionError{})
	})
}
