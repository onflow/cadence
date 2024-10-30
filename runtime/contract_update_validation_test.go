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

package runtime_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func newContractDeployTransaction(function, name, code string) string {
	return fmt.Sprintf(
		`
                transaction {
                    prepare(signer: auth(Contracts) &Account) {
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
		sema.Account_ContractsTypeAddFunctionName,
		name,
		code,
	)
}

func newContractUpdateTransaction(name string, code string) string {
	return newContractDeployTransaction(
		sema.Account_ContractsTypeUpdateFunctionName,
		name,
		code,
	)
}

func newContractRemovalTransaction(contractName string) string {
	return fmt.Sprintf(
		`
           transaction {
               prepare(signer: auth(RemoveContract) &Account) {
                   signer.contracts.%s(name: "%s")
               }
           }
       `,
		sema.Account_ContractsTypeRemoveFunctionName,
		contractName,
	)
}

func newContractDeploymentTransactor(t *testing.T, config Config) func(code string) error {
	return newContractDeploymentTransactorWithVersion(t, config, "")
}

func newContractDeploymentTransactorWithVersion(t *testing.T, config Config, version string) func(code string) error {

	rt := NewTestInterpreterRuntimeWithConfig(config)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x42})}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnRemoveAccountContractCode: func(location common.AddressLocation) error {
			delete(accountCodes, location)
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnMinimumRequiredVersion: func() (string, error) {
			return version, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
func testDeployAndUpdate(t *testing.T, name string, oldCode string, newCode string, config Config) error {
	return testDeployAndUpdateWithVersion(t, name, oldCode, newCode, config, "")
}

func testDeployAndUpdateWithVersion(
	t *testing.T,
	name string,
	oldCode string,
	newCode string,
	config Config,
	version string,
) error {
	executeTransaction := newContractDeploymentTransactorWithVersion(t, config, version)
	err := executeTransaction(newContractAddTransaction(name, oldCode))
	require.NoError(t, err)

	return executeTransaction(newContractUpdateTransaction(name, newCode))
}

// testDeployAndRemove deploys a contract in one transaction,
// then removes the contract in another transaction
func testDeployAndRemove(t *testing.T, name string, code string, config Config) error {
	executeTransaction := newContractDeploymentTransactor(t, config)
	err := executeTransaction(newContractAddTransaction(name, code))
	require.NoError(t, err)

	return executeTransaction(newContractRemovalTransaction(name))
}

func testWithValidators(t *testing.T, name string, testFunc func(t *testing.T, config Config)) {
	for _, withC1Upgrade := range []bool{true, false} {
		withC1Upgrade := withC1Upgrade
		name := name

		if withC1Upgrade {
			name = fmt.Sprintf("%s (with C1 validator)", name)
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestInterpreterConfig
			config.LegacyContractUpgradeEnabled = withC1Upgrade
			testFunc(t, config)
		})
	}
}

func testWithValidatorsAndTypeRemovalEnabled(
	t *testing.T,
	name string,
	testFunc func(t *testing.T, config Config),
) {
	for _, withC1Upgrade := range []bool{true, false} {
		withC1Upgrade := withC1Upgrade
		name := name

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			config := DefaultTestInterpreterConfig
			config.LegacyContractUpgradeEnabled = withC1Upgrade

			testFunc(t, config)
		})
	}
}

func TestRuntimeContractUpdateValidation(t *testing.T) {

	t.Parallel()

	testWithValidators(t, "change field type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String
                init() {
                    self.a = "hello"
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

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "String", "Int")
	})

	testWithValidators(t, "add field", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String

                init() {
                    self.a = "hello"
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: String
                access(all) var b: Int

                init() {
                    self.a = "hello"
                    self.b = 0
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "Test", "b")
	})

	testWithValidators(t, "remove field", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String
                access(all) var b: Int

                init() {
                    self.a = "hello"
                    self.b = 0
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

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "change nested decl field type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var a: @TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) let b: Int

                    init() {
                        self.b = 1234
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var a: @Test.TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) let b: String

                    init() {
                        self.b = "string_1234"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestResource", "b", "Int", "String")
	})

	testWithValidators(t, "add field to nested decl", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var a: @TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) var b: String

                    init() {
                        self.b = "hello"
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var a: @Test.TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) var b: String
                    access(all) var c: Int

                    init() {
                        self.b = "hello"
                        self.c = 0
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "TestResource", "c")
	})

	testWithValidators(t, "remove field from nested decl", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var a: @TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) var b: String
                    access(all) var c: Int

                    init() {
                        self.b = "hello"
                        self.c = 0
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var a: @Test.TestResource

                init() {
                    self.a <- create Test.TestResource()
                }

                access(all) resource TestResource {

                    access(all) var b: String

                    init() {
                        self.b = "hello"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "change indirect field type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: [TestStruct; 1]

                init() {
                    self.x = [TestStruct()]
                }

                access(all) struct TestStruct {
                    access(all) let a: Int
                    access(all) var b: Int

                    init() {
                        self.a = 123
                        self.b = 456
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: [TestStruct; 1]

                init() {
                    self.x = [TestStruct()]
                }

                access(all) struct TestStruct {
                    access(all) let a: Int
                    access(all) var b: String

                    init() {
                        self.a = 123
                        self.b = "string_456"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "b", "Int", "String")
	})

	testWithValidators(t, "circular types refs", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: {String: Foo}

                init() {
                    self.x = { "foo" : Foo() }
                }

                access(all) struct Foo {

                    access(all) let a: Foo?
                    access(all) let b: Bar

                    init() {
                        self.a = nil
                        self.b = Bar()
                    }
                }

                access(all) struct Bar {

                    access(all) let c: Foo?
                    access(all) let d: Bar?

                    init() {
                        self.c = nil
                        self.d = nil
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: {String: Foo}

                init() {
                    self.x = { "foo" : Foo() }
                }

                access(all) struct Foo {

                    access(all) let a: Foo?
                    access(all) let b: Bar

                    init() {
                        self.a = nil
                        self.b = Bar()
                    }
                }

                access(all) struct Bar {

                    access(all) let c: Foo?
                    access(all) let d: String

                    init() {
                        self.c = nil
                        self.d = "string_d"
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Bar", "d", "Bar?", "String")
	})

	testWithValidators(t, "qualified vs unqualified nominal type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: Test.TestStruct
                access(all) var y: TestStruct

                init() {
                    self.x = Test.TestStruct()
                    self.y = TestStruct()
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: TestStruct
                access(all) var y: Test.TestStruct

                init() {
                    self.x = TestStruct()
                    self.y = Test.TestStruct()
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "change imported nominal type to local", func(t *testing.T, config Config) {

		const importCode = `
    	    	    access(all) contract TestImport {

    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: Int
    	    	            access(all) var b: Int

    	    	            init() {
    	    	                self.a = 123
    	    	                self.b = 456
    	    	            }
    	    	        }
    	    	    }
    	    	`

		executeTransaction := newContractDeploymentTransactor(t, config)

		err := executeTransaction(newContractAddTransaction("TestImport", importCode))
		require.NoError(t, err)

		const oldCode = `
    	    	    import TestImport from 0x42

    	    	    access(all) contract Test {

    	    	        access(all) var x: TestImport.TestStruct

    	    	        init() {
    	    	            self.x = TestImport.TestStruct()
    	    	        }
    	    	    }
    	    	`

		const newCode = `
    	    	    access(all) contract Test {

    	    	        access(all) var x: TestStruct

    	    	        init() {
    	    	            self.x = TestStruct()
    	    	        }

    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: Int
    	    	            access(all) var b: Int

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
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "x", "TestImport.TestStruct", "TestStruct")
	})

	testWithValidators(t, "change imported field nominal type location", func(t *testing.T, config Config) {

		runtime := NewTestInterpreterRuntime()

		makeDeployTransaction := func(name, code string) []byte {
			return []byte(fmt.Sprintf(
				`
				  transaction {
					prepare(signer: auth(BorrowValue) &Account) {
					  let acct = Account(payer: signer)
					  acct.contracts.add(name: "%s", code: "%s".decodeHex())
					}
				  }
				`,
				name,
				hex.EncodeToString([]byte(code)),
			))
		}

		accountCodes := map[Location][]byte{}
		var events []cadence.Event

		var nextAccount byte = 0x2

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnCreateAccount: func(payer Address) (address Address, err error) {
				result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
				nextAccount++
				return result.ToAddress(), nil
			},
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{0x1}}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		const importCode = `
		    access(all) contract TestImport {

		        access(all) struct TestStruct {
		            access(all) let a: Int

		            init() {
		                self.a = 123
		            }
		        }
		    }
		`

		deployTransaction := makeDeployTransaction("TestImport", importCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const otherImportedCode = `
			access(all) contract TestImport {

				access(all) struct TestStruct {
		            access(all) let a: Int
		            access(all) var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		deployTransaction = makeDeployTransaction("TestImport", otherImportedCode)

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const oldCode = `
		    import TestImport from 0x2

		    access(all) contract Test {

		        access(all) var x: TestImport.TestStruct

		        init() {
		            self.x = TestImport.TestStruct()
		        }
		    }
		`

		deployTransaction = []byte(newContractAddTransaction("Test", oldCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		const newCode = `
			import TestImport from 0x3

			access(all) contract Test {

				access(all) var x: TestImport.TestStruct

				init() {
					self.x = TestImport.TestStruct()
				}
			}
		`

		deployTransaction = []byte(newContractUpdateTransaction("Test", newCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "x", "TestImport.TestStruct", "TestImport.TestStruct")
	})

	testWithValidators(t, "change imported non-field nominal type location", func(t *testing.T, config Config) {

		runtime := NewTestInterpreterRuntime()

		makeDeployTransaction := func(name, code string) []byte {
			return []byte(fmt.Sprintf(
				`
				  transaction {
					prepare(signer: auth(Storage) &Account) {
					  let acct = Account(payer: signer)
					  acct.contracts.add(name: "%s", code: "%s".decodeHex())
					}
				  }
				`,
				name,
				hex.EncodeToString([]byte(code)),
			))
		}

		accountCodes := map[Location][]byte{}
		var events []cadence.Event

		var nextAccount byte = 0x2

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnCreateAccount: func(payer Address) (address Address, err error) {
				result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
				nextAccount++
				return result.ToAddress(), nil
			},
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{0x1}}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		const importCode = `
			access(all) contract TestImport {

				access(all) struct TestStruct {
					access(all) let a: Int

		            init() {
		                self.a = 123
		            }
		        }
		    }
		`

		deployTransaction := makeDeployTransaction("TestImport", importCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const otherImportedCode = `
			access(all) contract TestImport {

				access(all) struct TestStruct {
					access(all) let a: Int
					access(all) var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		deployTransaction = makeDeployTransaction("TestImport", otherImportedCode)

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const oldCode = `
		    import TestImport from 0x2

		    access(all) contract Test {

		        access(all) fun foo(): TestImport.TestStruct {
					return TestImport.TestStruct()
				}
		    }
		`

		deployTransaction = []byte(newContractAddTransaction("Test", oldCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		const newCode = `
			import TestImport from 0x3

			access(all) contract Test {
				access(all) fun foo(): TestImport.TestStruct {
					return TestImport.TestStruct()
				}
			}
		`

		deployTransaction = []byte(newContractUpdateTransaction("Test", newCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)
	})

	testWithValidators(t, "change imported field nominal type location implicitly", func(t *testing.T, config Config) {

		runtime := NewTestInterpreterRuntime()

		makeDeployTransaction := func(name, code string) []byte {
			return []byte(fmt.Sprintf(
				`
				  transaction {
					prepare(signer: auth(Storage) &Account) {
					  let acct = Account(payer: signer)
					  acct.contracts.add(name: "%s", code: "%s".decodeHex())
					}
				  }
				`,
				name,
				hex.EncodeToString([]byte(code)),
			))
		}

		accountCodes := map[Location][]byte{}
		var events []cadence.Event

		var nextAccount byte = 0x2

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnCreateAccount: func(payer Address) (address Address, err error) {
				result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
				nextAccount++
				return result.ToAddress(), nil
			},
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{0x1}}, nil
			},
			OnResolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
				require.Empty(t, identifiers)
				require.IsType(t, common.AddressLocation{}, location)

				return []ResolvedLocation{
					{
						Location: common.AddressLocation{
							Address: location.(common.AddressLocation).Address,
							Name:    "TestImport",
						},
						Identifiers: []ast.Identifier{
							{
								Identifier: "TestImport",
							},
						},
					},
				}, nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnGetAccountContractNames: func(address common.Address) (names []string, err error) {
				if address == common.MustBytesToAddress([]byte{0x1}) {
					return []string{"Test"}, nil
				}
				return []string{"TestImport"}, nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		const importCode = `
			access(all) contract TestImport {

				access(all) struct TestStruct {
					access(all) let a: Int

		            init() {
		                self.a = 123
		            }
		        }
		    }
		`

		deployTransaction := makeDeployTransaction("TestImport", importCode)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const otherImportedCode = `
			access(all) contract TestImport {

				access(all) struct TestStruct {
					access(all) let a: Int
					access(all) var b: Int

		            init() {
		                self.a = 123
		                self.b = 456
		            }
		        }
		    }
		`

		deployTransaction = makeDeployTransaction("TestImport", otherImportedCode)

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		const oldCode = `
		    import 0x2

		    access(all) contract Test {

		        access(all) var x: TestImport.TestStruct

		        init() {
		            self.x = TestImport.TestStruct()
		        }
		    }
		`

		deployTransaction = []byte(newContractAddTransaction("Test", oldCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)

		const newCode = `
			import 0x3

			access(all) contract Test {

				access(all) var x: TestImport.TestStruct

				init() {
					self.x = TestImport.TestStruct()
				}
			}
		`

		deployTransaction = []byte(newContractUpdateTransaction("Test", newCode))

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "x", "TestImport.TestStruct", "TestImport.TestStruct")
	})

	testWithValidators(t, "contract interface update", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract interface Test {
                access(all) var a: String
                access(all) fun getA() : String
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) var a: Int
                access(all) fun getA() : Int
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "Test", "a", "String", "Int")
	})

	testWithValidators(t, "convert interface to contract", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract interface Test {
                access(all) var a: String
                access(all) fun getA() : String
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var a: String

                init() {
                    self.a = "hello"
                }

                access(all) fun getA() : String {
                    return self.a
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test",
			common.DeclarationKindContractInterface,
			common.DeclarationKindContract,
		)
	})

	testWithValidators(t, "convert contract to interface", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var a: String

                init() {
                    self.a = "hello"
                }

                access(all) fun getA() : String {
                    return self.a
                }
            }
        `

		const newCode = `
            access(all) contract interface Test {
                access(all) var a: String
                access(all) fun getA() : String
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"Test",
			common.DeclarationKindContract,
			common.DeclarationKindContractInterface,
		)
	})

	testWithValidators(t, "change non stored", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: UsedStruct

                init() {
                    self.x = UsedStruct()
                }

                access(all) struct UsedStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }

                    access(all) fun getA() : Int {
                        return self.a
                    }
                }

                access(all) struct UnusedStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }

                    access(all) fun getA() : Int {
                        return self.a
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: UsedStruct

                init() {
                    self.x = UsedStruct()
                }

                access(all) struct UsedStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }

                    access(all) fun getA() : String {
                        return "hello_123"
                    }

                    access(all) fun getA_new() : Int {
                        return self.a
                    }
                }

                access(all) struct UnusedStruct {
                    access(all) let a: String

                    init() {
                        self.a = "string_456"
                    }

                    access(all) fun getA() : String {
                        return self.a
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

		// Changing unused public composite types should also fail, since those could be
		// referred by anyone in the chain, and may cause data inconsistency.
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "UnusedStruct", "a", "Int", "String")
	})

	testWithValidators(t, "change enum type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: Foo

                init() {
                    self.x = Foo.up
                }

                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: Foo

                init() {
                    self.x = Foo.up
                }

                access(all) enum Foo: UInt128 {
                    access(all) case up
                    access(all) case down
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertConformanceMismatchError(t, cause, "Foo", "UInt8")
	})

	testWithValidators(t, "change nested interface", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var x: {TestStruct}?

                init() {
                    self.x = nil
                }

                access(all) struct interface TestStruct {
                    access(all) let a: String
                    access(all) var b: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var x: {TestStruct}?

                init() {
                    self.x = nil
                }

                access(all) struct interface TestStruct {
                    access(all) let a: Int
                    access(all) var b: Int
                }
            }
       `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "String", "Int")
	})

	testWithValidators(t, "change nested interface to struct", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) struct interface TestStruct {
                    access(all) var a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertDeclTypeChangeError(
			t,
			cause,
			"TestStruct",
			common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
		)
	})

	testWithValidators(t, "adding a nested struct", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
       `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "removing a nested struct", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")
	})

	testWithValidators(t, "add and remove field", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String
                init() {
                    self.a = "hello"
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var b: Int
                init() {
                    self.b = 0
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertExtraneousFieldError(t, cause, "Test", "b")
	})

	testWithValidators(t, "multiple errors", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String

                init() {
                    self.a = "hello"
                }

                access(all) struct interface TestStruct {
                    access(all) var a: Int
                }
            }
       `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Int
                access(all) var b: String

                init() {
                    self.a = 0
                    self.b = "hello"
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

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

	testWithValidators(t, "check error messages", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: String

                init() {
                    self.a = "hello"
                }

                access(all) struct interface TestStruct {
                    access(all) var a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Int
                access(all) var b: String

                init() {
                    self.a = 0
                    self.b = "hello"
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		const expectedError = "error: mismatching field `a` in `Test`\n" +
			" --> 0000000000000042.Test:3:35\n" +
			"  |\n" +
			"3 |                 access(all) var a: Int\n" +
			"  |                                    ^^^ incompatible type annotations. expected `String`, found `Int`\n" +
			"\n" +
			"error: found new field `b` in `Test`\n" +
			" --> 0000000000000042.Test:4:32\n" +
			"  |\n" +
			"4 |                 access(all) var b: String\n" +
			"  |                                 ^\n" +
			"\n" +
			"error: trying to convert structure interface `TestStruct` to a structure\n" +
			"  --> 0000000000000042.Test:11:35\n" +
			"   |\n" +
			"11 |                 access(all) struct TestStruct {\n" +
			"   |                                    ^^^^^^^^^^"

		require.Contains(t, err.Error(), expectedError)
	})

	testWithValidators(t, "Test reference types", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) var vault: Capability<&TestStruct>?

                init() {
                    self.vault = nil
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var vault: Capability<&TestStruct>?

                init() {
                    self.vault = nil
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Test function type", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                access(all) var add: fun(Int, Int): Int

                init() {
                    self.add = fun (a: Int, b: Int): Int {
                        return a + b
                    }
                }

                access(all) struct TestStruct {
                    access(all) let a: Int

                    init() {
                        self.a = 123
                    }
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "error: field add has non-storable type: fun(Int, Int): Int")
	})

	testWithValidators(t, "Test conformance", func(t *testing.T, config Config) {

		const importCode = `
    	    	    access(all) contract TestImport {
    	    	        access(all) struct interface AnInterface {
    	    	            access(all) a: Int
    	    	        }
    	    	    }
    	    	`

		executeTransaction := newContractDeploymentTransactor(t, config)
		err := executeTransaction(newContractAddTransaction("TestImport", importCode))
		require.NoError(t, err)

		const oldCode = `
    	    	    import TestImport from 0x42

    	    	    access(all) contract Test {
    	    	        access(all) struct TestStruct1 {
    	    	            access(all) let a: Int
    	    	            init() {
    	    	                self.a = 123
    	    	            }
    	    	        }

    	    	        access(all) struct TestStruct2: TestImport.AnInterface {
    	    	            access(all) let a: Int

    	    	            init() {
    	    	                self.a = 123
    	    	            }
    	    	        }
    	    	    }
    	    `

		const newCode = `
    	    	    import TestImport from 0x42

    	    	    access(all) contract Test {

    	    	        access(all) struct TestStruct2: TestImport.AnInterface {
    	    	            access(all) let a: Int

    	    	            init() {
    	    	                self.a = 123
    	    	            }
    	    	        }

    	    	        access(all) struct TestStruct1 {
    	    	            access(all) let a: Int
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
            access(all) contract Test {
                // simple nominal type
                access(all) var a: TestStruct

                // qualified nominal type
                access(all) var b: Test.TestStruct

                // optional type
                access(all) var c: Int?

                // variable sized type
                access(all) var d: [Int]

                // constant sized type
                access(all) var e: [Int; 2]

                // dictionary type
                access(all) var f: {Int: String}

                // intersection type
                access(all) var g: {TestInterface}

                // instantiation and reference types
                access(all) var h:  Capability<&TestStruct>?

                // function type
                access(all) var i: Capability<&fun(Int, Int): Int>?

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

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                // function type
                access(all) var i: Capability<&fun(Int, Int): Int>?

                // instantiation and reference types
                access(all) var h:  Capability<&TestStruct>?

                // intersection type
                access(all) var g: {TestInterface}

                // dictionary type
                access(all) var f: {Int: String}

                // constant sized type
                access(all) var e: [Int; 2]

                // variable sized type
                access(all) var d: [Int]

                // optional type
                access(all) var c: Int?

                // qualified nominal type
                access(all) var b: Test.TestStruct

                // simple nominal type
                access(all) var a: TestStruct

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

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, DefaultTestInterpreterConfig)
		require.NoError(t, err)
	})

	t.Run("Test all types (with c1 validator)", func(t *testing.T) {
		// The corresponding test with the "default validator" uses a contract code
		// that is not compatible with the old parser.
		// Therefore, run the same test with an adjusted code.

		t.Parallel()

		const oldCode = `
            access(all) contract Test {
                // simple nominal type
                access(all) var a: TestStruct

                // qualified nominal type
                access(all) var b: Test.TestStruct

                // optional type
                access(all) var c: Int?

                // variable sized type
                access(all) var d: [Int]

                // constant sized type
                access(all) var e: [Int; 2]

                // dictionary type
                access(all) var f: {Int: String}

                // intersection type
                access(all) var g: {TestInterface}

                // instantiation and reference types
                access(all) var h:  Capability<&TestStruct>?

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
                }

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {

                // instantiation and reference types
                access(all) var h:  Capability<&TestStruct>?

                // intersection type
                access(all) var g: {TestInterface}

                // dictionary type
                access(all) var f: {Int: String}

                // constant sized type
                access(all) var e: [Int; 2]

                // variable sized type
                access(all) var d: [Int]

                // optional type
                access(all) var c: Int?

                // qualified nominal type
                access(all) var b: Test.TestStruct

                // simple nominal type
                access(all) var a: TestStruct

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
                }

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		config := DefaultTestInterpreterConfig
		config.LegacyContractUpgradeEnabled = true
		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Test intersection types", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                // intersection type
                access(all) var a: {TestInterface}
                access(all) var b: {TestInterface}
                access(all) var c: {TestInterface}
                access(all) var d: {TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                    self.c = TestStruct()
                    self.d = TestStruct()
                }

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: {TestInterface}
                access(all) var b: {TestInterface}
                access(all) var c: {TestInterface}
                access(all) var d: {TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                    self.c = TestStruct()
                    self.d = TestStruct()
                }

                access(all) struct TestStruct:TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Test invalid intersection types change", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {

                // intersection type
                access(all) var a: TestStruct
                access(all) var b: {TestInterface}

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                }

                access(all) struct TestStruct: TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: {TestInterface}
                access(all) var b: TestStruct

                init() {
                    var count: Int = 567
                    self.a = TestStruct()
                    self.b = TestStruct()
                }

                access(all) struct TestStruct: TestInterface {
                    access(all) let a: Int
                    init() {
                        self.a = 123
                    }
                }

                access(all) struct interface TestInterface {
                    access(all) let a: Int
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "access(all) var a: {TestInterface}"+
			"\n  |                                    ^^^^^^^^^^^^^^^ "+
			"incompatible type annotations. expected `TestStruct`")

		assert.Contains(t, err.Error(), "access(all) var b: TestStruct"+
			"\n  |                                    ^^^^^^^^^^ "+
			"incompatible type annotations. expected `{TestInterface}`, found `TestStruct`")
	})

	testWithValidators(t, "enum valid", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                }
            }
       `

		const newCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "enum remove case", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingEnumCasesError(t, cause, "Foo", 2, 1)
	})

	testWithValidators(t, "enum add case", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                    access(all) case left
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "enum swap cases", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case up
                    access(all) case down
                    access(all) case left
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) enum Foo: UInt8 {
                    access(all) case down
                    access(all) case left
                    access(all) case up
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		updateErr := getContractUpdateError(t, err, "Test")
		require.NotNil(t, updateErr)

		childErrors := updateErr.Errors
		require.Len(t, childErrors, 3)

		assertEnumCaseMismatchError(t, childErrors[0], "up", "down")
		assertEnumCaseMismatchError(t, childErrors[1], "down", "left")
		assertEnumCaseMismatchError(t, childErrors[2], "left", "up")
	})

	testWithValidators(t, "Remove and add struct", func(t *testing.T, config Config) {

		const oldCode = `
    	    	    access(all) contract Test {

    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: Int
    	    	            access(all) var b: Int

    	    	            init() {
    	    	                self.a = 123
    	    	                self.b = 456
    	    	            }
    	    	        }
    	    	    }
    	    	`

		const updateCode1 = `
    	    	    access(all) contract Test {
    	    	    }
    	    	`

		executeTransaction := newContractDeploymentTransactor(t, config)

		err := executeTransaction(newContractAddTransaction("Test", oldCode))
		require.NoError(t, err)

		err = executeTransaction(newContractUpdateTransaction("Test", updateCode1))
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")

		const updateCode2 = `
    	    	    access(all) contract Test {

    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: String

    	    	            init() {
    	    	                self.a = "hello123"
    	    	            }
    	    	        }
    	    	    }
    	    	`

		err = executeTransaction(newContractUpdateTransaction("Test", updateCode2))
		RequireError(t, err)

		cause = getSingleContractUpdateErrorCause(t, err, "Test")
		assertFieldTypeMismatchError(t, cause, "TestStruct", "a", "Int", "String")
	})

	testWithValidators(t, "Rename struct", func(t *testing.T, config Config) {

		const oldCode = `
    	    	    access(all) contract Test {

    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: Int
    	    	            access(all) var b: Int

    	    	            init() {
    	    	                self.a = 123
    	    	                self.b = 456
    	    	            }
    	    	        }
    	    	    }
    	    	`

		const newCode = `
    	    	    access(all) contract Test {

    	    	        access(all) struct TestStructRenamed {
    	    	            access(all) let a: Int
    	    	            access(all) var b: Int

    	    	            init() {
    	    	                self.a = 123
    	    	                self.b = 456
    	    	            }
    	    	        }
    	    	    }
    	    	`

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")
		assertMissingDeclarationError(t, cause, "TestStruct")
	})

	testWithValidators(t, "Remove contract with enum", func(t *testing.T, config Config) {

		const code = `
    	    	    access(all) contract Test {
    	    	        access(all) enum TestEnum: Int {
    	    	        }
    	    	    }
    	    	`

		err := testDeployAndRemove(t, "Test", code, config)
		RequireError(t, err)

		assertContractRemovalError(t, err, "Test")
	})

	testWithValidators(t, "Remove contract without enum", func(t *testing.T, config Config) {

		const code = `
    	    	    access(all) contract Test {
    	    	        access(all) struct TestStruct {
    	    	            access(all) let a: Int

    	    	            init() {
    	    	                self.a = 123
    	    	            }
    	    	        }
    	    	    }
    	    	`

		err := testDeployAndRemove(t, "Test", code, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "removing multiple nested structs", func(t *testing.T, config Config) {

		const oldCode = `
    	    	    access(all) contract Test {
    	    	        access(all) struct A {}
    	    	        access(all) struct B {}
    	    	    }
    	    	`

		const newCode = `
    	    	    access(all) contract Test {}
    	    	`

		// Errors reporting was previously non-deterministic,
		// assert that reports are deterministic

		for i := 0; i < 1000; i++ {

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			RequireError(t, err)

			updateErr := getContractUpdateError(t, err, "Test")

			childErrors := updateErr.Errors
			require.Len(t, childErrors, 2)

			if !assertMissingDeclarationError(t, childErrors[0], "A") {
				t.FailNow()
			}
			assertMissingDeclarationError(t, childErrors[1], "B")
		}
	})

	testWithValidators(t, "Remove event", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) event Foo()
                access(all) event Bar()
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) event Bar()
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Add event", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) event Foo()
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) event Foo()
                access(all) event Bar()
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Update event", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) event Foo()
                access(all) event Bar()
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) event Foo(a: Int)
                access(all) event Bar(b: String)
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})
}

func assertContractRemovalError(t *testing.T, err error, name string) {
	var contractRemovalError *stdlib.ContractRemovalError
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
	var declTypeChangeError *stdlib.InvalidDeclarationKindChangeError
	require.ErrorAs(t, err, &declTypeChangeError)

	assert.Equal(t, oldKind, declTypeChangeError.OldKind)
	assert.Equal(t, erroneousDeclName, declTypeChangeError.Name)
	assert.Equal(t, newKind, declTypeChangeError.NewKind)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	var extraFieldError *stdlib.ExtraneousFieldError
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
	var fieldMismatchError *stdlib.FieldMismatchError
	require.ErrorAs(t, err, &fieldMismatchError)

	assert.Equal(t, fieldName, fieldMismatchError.FieldName)
	assert.Equal(t, erroneousDeclName, fieldMismatchError.DeclName)

	var typeMismatchError *stdlib.TypeMismatchError
	assert.ErrorAs(t, fieldMismatchError.Err, &typeMismatchError)

	assert.Equal(t, expectedType, typeMismatchError.ExpectedType.String())
	assert.Equal(t, foundType, typeMismatchError.FoundType.String())
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	missingConformance string,
) {
	var conformanceMismatchError *stdlib.ConformanceMismatchError
	require.ErrorAs(t, err, &conformanceMismatchError)

	assert.Equal(t, erroneousDeclName, conformanceMismatchError.DeclName)
	assert.Equal(t, missingConformance, conformanceMismatchError.MissingConformance)
}

func assertEnumCaseMismatchError(t *testing.T, err error, expectedEnumCase string, foundEnumCase string) {
	var enumMismatchError *stdlib.EnumCaseMismatchError
	require.ErrorAs(t, err, &enumMismatchError)

	assert.Equal(t, expectedEnumCase, enumMismatchError.ExpectedName)
	assert.Equal(t, foundEnumCase, enumMismatchError.FoundName)
}

func assertMissingEnumCasesError(t *testing.T, err error, declName string, expectedCases int, foundCases int) {
	var missingEnumCasesError *stdlib.MissingEnumCasesError
	require.ErrorAs(t, err, &missingEnumCasesError)

	assert.Equal(t, declName, missingEnumCasesError.DeclName)
	assert.Equal(t, expectedCases, missingEnumCasesError.Expected)
	assert.Equal(t, foundCases, missingEnumCasesError.Found)
}

func assertMissingDeclarationError(t *testing.T, err error, declName string) bool {
	var missingDeclError *stdlib.MissingDeclarationError
	require.ErrorAs(t, err, &missingDeclError)

	return assert.Equal(t, declName, missingDeclError.Name)
}

func getSingleContractUpdateErrorCause(t *testing.T, err error, contractName string) error {
	updateErr := getContractUpdateError(t, err, contractName)

	require.Len(t, updateErr.Errors, 1)
	return updateErr.Errors[0]
}

func getContractUpdateError(t *testing.T, err error, contractName string) *stdlib.ContractUpdateError {
	RequireError(t, err)

	var invalidContractDeploymentErr *stdlib.InvalidContractDeploymentError
	require.ErrorAs(t, err, &invalidContractDeploymentErr)

	var contractUpdateErr *stdlib.ContractUpdateError
	require.ErrorAs(t, err, &contractUpdateErr)

	assert.Equal(t, contractName, contractUpdateErr.ContractName)

	return contractUpdateErr
}

func TestRuntimeContractUpdateConformanceChanges(t *testing.T) {

	t.Parallel()

	testWithValidators(t, "Adding conformance", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo {
                    init() {}
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo: Bar {
                    init() {
                    }

                    access(all) fun getName(): String {
                        return "John"
                    }
                }

                access(all) struct interface Bar {
                    access(all) fun getName(): String
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "Adding conformance with new fields", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo {
                    init() {}
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo: Bar {
                    access(all) var name: String

                    init() {
                        self.name = "John"
                    }
                }

                access(all) struct interface Bar {
                    access(all) var name: String
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")

		assertExtraneousFieldError(t, cause, "Foo", "name")
	})

	testWithValidators(t, "Removing conformance, one", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo: Bar {
                    init() {}
                }

                access(all) struct interface Bar {
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo {
                    init() {}
                }

                access(all) struct interface Bar {
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")

		assertConformanceMismatchError(t, cause, "Foo", "Bar")
	})

	testWithValidators(t, "Removing conformance, multiple", func(t *testing.T, config Config) {

		const oldCode = `
            access(all)
            contract Test {

                access(all) var a: Foo

                init() {
                    self.a = Foo()
                }

                access(all)
                struct Foo: Bar, Baz, Blub {
                    init() {}
                }

                access(all)
                struct interface Bar {}
                access(all)
                struct interface Baz {}
                access(all)
                struct interface Blub {}
            }
        `

		const newCode = `
            access(all)
            contract Test {
                access(all)
                var a: Foo

                init() {
                    self.a = Foo()
                }

                access(all)
                struct Foo: Bar {
                    init() {}
                }

                access(all)
                struct interface Bar {}
                access(all)
                struct interface Baz {}
                access(all)
                struct interface Blub {}
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		RequireError(t, err)

		cause := getSingleContractUpdateErrorCause(t, err, "Test")

		assertConformanceMismatchError(t, cause, "Foo", "Baz")
	})

	testWithValidators(t, "Change conformance order", func(t *testing.T, config Config) {

		const oldCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo: First, Second {
                    init() {}
                }

                access(all) struct interface First {
                }

                access(all) struct interface Second {
                }
            }
        `

		const newCode = `
            access(all) contract Test {
                access(all) var a: Foo
                init() {
                    self.a = Foo()
                }

                access(all) struct Foo: Second, First {
                    init() {}
                }

                access(all) struct interface First {
                }

                access(all) struct interface Second {
                }
            }
        `

		err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
		require.NoError(t, err)
	})

	testWithValidators(t, "missing comma in parameter list of old contract", func(t *testing.T, config Config) {

		address := common.MustBytesToAddress([]byte{0x42})

		const contractName = "Test"

		const oldCode = `
          access(all) contract Test {
              access(all) fun test(a: Int b: Int) {}
          }
        `

		const newCode = `
          access(all) contract Test {
              access(all) fun test(a: Int, b: Int) {}
          }
        `

		rt := NewTestInterpreterRuntime()

		contractLocation := common.AddressLocation{
			Address: address,
			Name:    contractName,
		}

		accountCodes := map[Location][]byte{
			contractLocation: []byte(oldCode),
		}

		var events []cadence.Event
		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnRemoveAccountContractCode: func(location common.AddressLocation) error {
				delete(accountCodes, location)
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		err := rt.ExecuteTransaction(
			Script{
				Source: []byte(newContractUpdateTransaction(contractName, newCode)),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})
}

func TestRuntimeContractUpdateProgramCaching(t *testing.T) {

	const name = "Test"
	const oldCode = `
    	  access(all) contract Test { init() { 1 } }
    	`
	const newCode = `
    	  access(all) contract Test { init() { 2 } }
    	`

	address := common.MustBytesToAddress([]byte{0x42})

	contractLocation := common.AddressLocation{
		Address: address,
		Name:    name,
	}

	type locationAccessCounts map[Location]int

	newTester := func() (
		runtimeInterface *TestRuntimeInterface,
		executeTransaction func(code string) error,
		programGets locationAccessCounts,
		programSets locationAccessCounts,
	) {
		rt := NewTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}
		var events []cadence.Event

		programGets = locationAccessCounts{}
		programSets = locationAccessCounts{}

		runtimeInterface = &TestRuntimeInterface{
			OnGetAndSetProgram: func(
				location Location,
				load func() (*interpreter.Program, error),
			) (
				program *interpreter.Program,
				err error,
			) {
				if runtimeInterface.Programs == nil {
					runtimeInterface.Programs = map[Location]*interpreter.Program{}
				}

				var ok bool
				program, ok = runtimeInterface.Programs[location]
				if program != nil {
					programGets[location]++
				}
				if ok {
					return
				}

				program, err = load()

				// NOTE: important: still set empty program,
				// even if error occurred

				runtimeInterface.Programs[location] = program

				programSets[location]++

				return
			},
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnRemoveAccountContractCode: func(location common.AddressLocation) error {
				delete(accountCodes, location)
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

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
			for location := range counts { //nolint:maprange
				delete(counts, location)
			}
		}
	}

	t.Run("Deploy contract to both", func(t *testing.T) {

		clearLocationAccessCounts()

		addTx := newContractAddTransaction(name, oldCode)

		txLocation := common.TransactionLocation{0x1}

		// Deploy to first

		err := executeTransaction1(addTx)
		require.NoError(t, err)
		require.Nil(t, runtimeInterface1.Programs[contractLocation])

		require.Equal(t, locationAccessCounts{}, programGets1)
		// NOTE: deployed contract is *correctly* *NOT* set,
		// as contract deployments and updates are delayed to the end of the transaction,
		// so should not influence program storage
		require.Equal(t, locationAccessCounts{txLocation: 1}, programSets1)

		// Deploy to second

		err = executeTransaction2(addTx)
		require.NoError(t, err)
		require.Nil(t, runtimeInterface2.Programs[contractLocation])
		require.Equal(t, locationAccessCounts{}, programGets2)
		// See NOTE above
		require.Equal(t, locationAccessCounts{txLocation: 1}, programSets2)
	})

	t.Run("Import only on second", func(t *testing.T) {

		clearLocationAccessCounts()

		txLocation := common.TransactionLocation{0x2}

		importTx := fmt.Sprintf(
			`
              import %s from %s

              transaction {
                  prepare(signer: &Account) {}
              }
            `,
			name,
			address.ShortHexWithPrefix(),
		)

		err := executeTransaction2(importTx)
		require.NoError(t, err)

		// only ran import TX against second,
		// so first should not have the program
		assert.Nil(t, runtimeInterface1.Programs[contractLocation])

		// NOTE: program in cache of second
		assert.NotNil(t, runtimeInterface2.Programs[contractLocation])

		assert.Equal(t,
			locationAccessCounts{
				contractLocation: 1,
			},
			programGets2,
		)

		// NOTE: program was set after it was got
		assert.Equal(
			t,
			locationAccessCounts{
				contractLocation: 1,
				txLocation:       1,
			},
			programSets2,
		)
	})

	t.Run("Update on both", func(t *testing.T) {

		clearLocationAccessCounts()

		txLocation1 := common.TransactionLocation{0x2}
		// second has seen an additional transaction (import, above)
		txLocation2 := common.TransactionLocation{0x3}

		updateTx := newContractUpdateTransaction(name, newCode)

		// Update on first

		err := executeTransaction1(updateTx)
		require.NoError(t, err)

		// NOTE: the program was not available in the cache (no successful get).
		// The old code is only parsed, and program does not need to be set.

		assert.Equal(t,
			locationAccessCounts{},
			programGets1,
		)
		assert.Equal(
			t,
			locationAccessCounts{
				txLocation1: 1,
			},
			programSets1,
		)

		// Update on second

		err = executeTransaction2(updateTx)
		require.NoError(t, err)

		// NOTE: the program was available in the cache (successful get).
		// The old code is only parsed, and does not need to be set.

		assert.Equal(t,
			locationAccessCounts{},
			programGets2,
		)
		assert.Equal(
			t,
			locationAccessCounts{
				txLocation2: 1,
			},
			programSets2,
		)
	})
}

func TestTypeRemovalPragmaUpdates(t *testing.T) {
	t.Parallel()

	testWithValidatorsAndTypeRemovalEnabled(t,
		"Remove pragma",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #foo(bar)
                    #baz
                }
            `

			const newCode = `
                access(all) contract Test {
                    #baz
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"Remove removedType pragma",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #removedType(bar)
                    #baz
                }
            `

			const newCode = `
                access(all) contract Test {
                    #baz
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.TypeRemovalPragmaRemovalError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"removedType pragma moved into sub-declaration",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #removedType(bar)
                    access(all) struct S {

                    }
                }
            `

			const newCode = `
                access(all) contract Test {
                    access(all) struct S {
                        #removedType(bar)
                    }
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.TypeRemovalPragmaRemovalError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"reorder removedType pragmas",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #removedType(bar)
                    #removedType(foo)
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(foo)
                    #removedType(bar)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"malformed removedType pragma integer",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #baz
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(3)
                    #baz
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.InvalidTypeRemovalPragmaError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(
		t,
		"malformed removedType qualified name",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #baz
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(X.Y)
                    #baz
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.InvalidTypeRemovalPragmaError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"removedType with zero args",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType()
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.InvalidTypeRemovalPragmaError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"removedType with two args",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(x, y)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.InvalidTypeRemovalPragmaError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType allows type removal",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType allows two type removals",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource R {}
                    access(all) struct S {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                    #removedType(S)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType does not allow resource interface type removal",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource interface R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

			var expectedErr *stdlib.MissingDeclarationError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType does not allow struct interface type removal",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) struct interface S {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(S)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

			var expectedErr *stdlib.MissingDeclarationError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType can be added",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    #removedType(I)
                    access(all) resource R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                    #removedType(I)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType can be added without removing a type",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(X)
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"declarations cannot co-exist with removed type of the same name, composite",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                    access(all) resource R {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.UseOfRemovedTypeError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"declarations cannot co-exist with removed type of the same name, interface",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource interface R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                    access(all) resource interface R {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.UseOfRemovedTypeError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"declarations cannot co-exist with removed type of the same name, attachment",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) attachment R for AnyResource {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    #removedType(R)
                    access(all) attachment R for AnyResource {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			var expectedErr *stdlib.UseOfRemovedTypeError
			require.ErrorAs(t, err, &expectedErr)
		},
	)

	testWithValidatorsAndTypeRemovalEnabled(t,
		"#removedType is only scoped to the current declaration, inner",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) resource R {}
                    access(all) struct S {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    access(all) struct S {
                        #removedType(R)
                    }
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

			var expectedErr *stdlib.MissingDeclarationError
			require.ErrorAs(t, err, &expectedErr)
		},
	)
}

func TestRuntimeContractUpdateErrorsInOldProgram(t *testing.T) {

	t.Parallel()

	testWithValidatorsAndTypeRemovalEnabled(t,
		"invalid #removedType pragma in old code",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    // invalid type removal pragma in old code
                    #removedType(R, R2)
                    access(all) resource R {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    access(all) resource R {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

			// Should not report any errors for the type invalid removal pragma in the old code.
			require.NoError(t, err)
		},
	)

	testWithValidators(t, "invalid old program", func(t *testing.T, config Config) {

		runtime := NewTestInterpreterRuntime()

		var events []cadence.Event

		address := common.MustBytesToAddress([]byte{0x2})

		location := common.AddressLocation{
			Name:    "Test",
			Address: address,
		}

		const oldCode = `
		    access(all) fun main() {
                // some lines to increase program length
            }
		`

		accountCodes := map[Location][]byte{
			location: []byte(oldCode),
		}

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		const newCode = `
			access(all) contract Test {}
		`

		updateTransaction := []byte(newContractUpdateTransaction("Test", newCode))

		err := runtime.ExecuteTransaction(
			Script{
				Source: updateTransaction,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)
		oldProgramError := &stdlib.OldProgramError{}
		require.ErrorAs(t, err, &oldProgramError)
	})
}

func TestAttachmentsUpdates(t *testing.T) {
	t.Parallel()

	testWithValidators(t,
		"Keep base type",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) attachment A for AnyResource {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    access(all) attachment A for AnyResource {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)
			require.NoError(t, err)
		},
	)

	testWithValidators(t,
		"Change base type",
		func(t *testing.T, config Config) {

			const oldCode = `
                access(all) contract Test {
                    access(all) attachment A for AnyResource {}
                }
            `

			const newCode = `
                access(all) contract Test {
                    access(all) attachment A for AnyStruct {}
                }
            `

			err := testDeployAndUpdate(t, "Test", oldCode, newCode, config)

			var expectedErr *stdlib.TypeMismatchError
			require.ErrorAs(t, err, &expectedErr)
		},
	)
}
