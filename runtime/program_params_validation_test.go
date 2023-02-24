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

package runtime

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeScriptParameterTypeValidation(t *testing.T) {

	t.Parallel()

	expectNonImportableError := func(t *testing.T, err error) {
		RequireError(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, &ScriptParameterTypeNotImportableError{}, runtimeErr.Err)
	}

	expectRuntimeError := func(t *testing.T, err error, expectedError error) {
		RequireError(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, expectedError, runtimeErr.Err)
	}

	newFooStruct := func() cadence.Struct {
		return cadence.Struct{
			StructType: &cadence.StructType{
				Location:            common.ScriptLocation{},
				QualifiedIdentifier: "Foo",
				Fields:              []cadence.Field{},
			},
			Fields: []cadence.Value{},
		}
	}

	newPublicAccountKeys := func() cadence.Struct {
		return cadence.Struct{
			StructType: &cadence.StructType{
				QualifiedIdentifier: "PublicAccount.Keys",
				Fields:              []cadence.Field{},
			},
			Fields: []cadence.Value{},
		}
	}

	executeScript := func(t *testing.T, script string, arg cadence.Value) (err error) {
		var encodedArg []byte
		encodedArg, err = json.Encode(arg)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}
		addPublicKeyValidation(runtimeInterface, nil)

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		return err
	}

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
            }
        `

		err := executeScript(t, script, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Struct", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Foo?) {
            }

            pub struct Foo {
                pub var funcTypedField: (():Void)

                init() {
                    self.funcTypedField = fun() {}
                }
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: AnyStruct?) {
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Interface", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {Bar}) {
            }

            pub struct Foo: Bar {
            }

            pub struct interface Bar {
            }
        `

		err := executeScript(t, script, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Interface", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {Bar}?) {
            }

            pub struct interface Bar {
                pub var funcTypedField: (():Void)
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: @Baz?) {
                destroy arg
            }

            pub resource Baz {
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("AnyResource", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: @AnyResource?) {
                destroy arg
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Foo?) {
            }

            pub contract Foo {
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: [String]) {
            }
        `

		err := executeScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{}),
		)

		assert.NoError(t, err)
	})

	t.Run("Non-Importable Array", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: [(():Void)]) {
            }
        `

		err := executeScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{}),
		)

		expectNonImportableError(t, err)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {String: Bool}) {
            }
        `

		err := executeScript(
			t,
			script,
			cadence.NewDictionary([]cadence.KeyValuePair{}),
		)

		assert.NoError(t, err)
	})

	t.Run("Capability", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Capability<&Int>?) {
            }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {String: (():Void)}) {
            }
        `

		err := executeScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{}),
		)

		expectNonImportableError(t, err)
	})

	t.Run("Numeric Types", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllNumberTypes {
			typString := typ.QualifiedString()

			t.Run(typString, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                        pub fun main(arg: %s?) {
                        }
                    `,
					typString,
				)

				err := executeScript(t, script, cadence.NewOptional(nil))
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Native composites", func(t *testing.T) {
		t.Parallel()

		type argumentPassingTest struct {
			argument      cadence.Value
			label         string
			typeSignature string
			expectErrors  bool
		}

		var argumentPassingTests []*argumentPassingTest

		for typeName, typ := range sema.NativeCompositeTypes {
			var value cadence.Value
			expectErrors := false

			switch typ {
			case sema.HashAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(1),
					},
				).WithType(HashAlgoType)

			case sema.SignatureAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(1),
					},
				).WithType(SignAlgoType)

			case sema.PublicKeyType:
				value = cadence.NewStruct(
					[]cadence.Value{
						// PublicKey bytes
						cadence.NewArray([]cadence.Value{}),

						// Sign algorithm
						cadence.NewEnum(
							[]cadence.Value{
								cadence.NewUInt8(1),
							},
						).WithType(SignAlgoType),
					},
				).WithType(PublicKeyType)

			default:
				// This test case only focuses on the type,
				// and has no interest in the value.
				value = nil

				expectErrors = true
			}

			testCase := &argumentPassingTest{
				label:         typeName,
				typeSignature: typeName + "?",
				argument:      cadence.NewOptional(value),
				expectErrors:  expectErrors,
			}

			argumentPassingTests = append(argumentPassingTests, testCase)
		}

		testArgumentPassing := func(test *argumentPassingTest) {

			t.Run(test.label, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                    pub fun main(arg: %s) {
                    }`,
					test.typeSignature,
				)

				err := executeScript(t, script, test.argument)

				if test.expectErrors {
					expectNonImportableError(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}

		for _, testCase := range argumentPassingTests {
			testArgumentPassing(testCase)
		}
	})

	t.Run("Invalid struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main(arg: AnyStruct?) {
                }
                pub struct Foo {
                    pub var nonImportableField: PublicAccount.Keys?
                    init() {
                        self.nonImportableField = nil
                    }
                }
            `

		err := executeScript(t, script, newFooStruct())
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid struct as valid interface", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main(arg: {Bar}?) {
                }
                pub struct Foo: Bar {
                    pub var nonImportableField: PublicAccount.Keys?
                    init() {
                        self.nonImportableField = nil
                    }
                }
                pub struct interface Bar {
                }
            `

		err := executeScript(t, script, newFooStruct())
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid native struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main(arg: AnyStruct) {
                }
            `

		err := executeScript(t, script, newPublicAccountKeys())
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})

	t.Run("Invalid struct in array", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main(arg: [AnyStruct]) {
                }
            `

		err := executeScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{
				newPublicAccountKeys(),
			}),
		)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})

	t.Run("invalid HashAlgorithm", func(t *testing.T) {
		t.Parallel()

		err := executeScript(t,
			`pub fun main(arg: HashAlgorithm) {}`,
			cadence.NewEnum(
				[]cadence.Value{
					cadence.NewUInt8(0),
				},
			).WithType(HashAlgoType),
		)
		RequireError(t, err)

		var entryPointErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &entryPointErr)
	})

	t.Run("invalid SignatureAlgorithm", func(t *testing.T) {
		t.Parallel()

		err := executeScript(t,
			`pub fun main(arg: SignatureAlgorithm) {}`,
			cadence.NewEnum(
				[]cadence.Value{
					cadence.NewUInt8(0),
				},
			).WithType(SignAlgoType),
		)
		RequireError(t, err)

		var entryPointErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &entryPointErr)
	})
}

func TestRuntimeTransactionParameterTypeValidation(t *testing.T) {

	t.Parallel()

	expectCheckerErrors := func(t *testing.T, err error, expectedErrors ...error) {
		RequireError(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		require.IsType(t, &ParsingCheckingError{}, runtimeErr.Err)
		parsingCheckingErr := runtimeErr.Err.(*ParsingCheckingError)

		require.IsType(t, &sema.CheckerError{}, parsingCheckingErr.Err)
		checkerErr := parsingCheckingErr.Err.(*sema.CheckerError)

		require.Len(t, checkerErr.Errors, len(expectedErrors))
		for i, err := range expectedErrors {
			assert.IsType(t, err, checkerErr.Errors[i])
		}
	}

	expectRuntimeError := func(t *testing.T, err error, expectedError error) {
		RequireError(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, expectedError, runtimeErr.Err)
	}

	newFooStruct := func() cadence.Struct {
		return cadence.Struct{
			StructType: &cadence.StructType{
				Location: common.AddressLocation{
					Address: common.MustBytesToAddress([]byte{0x1}),
					Name:    "C",
				},
				QualifiedIdentifier: "C.Foo",
				Fields:              []cadence.Field{},
			},
			Fields: []cadence.Value{},
		}
	}

	newPublicAccountKeys := func() cadence.Struct {
		return cadence.Struct{
			StructType: &cadence.StructType{
				QualifiedIdentifier: "PublicAccount.Keys",
				Fields:              []cadence.Field{},
			},
			Fields: []cadence.Value{},
		}
	}

	executeTransaction := func(
		t *testing.T,
		script string,
		contracts map[common.AddressLocation][]byte,
		arg cadence.Value,
	) (err error) {
		var encodedArg []byte
		encodedArg, err = json.Encode(arg)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage:         storage,
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return contracts[location], nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}
		addPublicKeyValidation(runtimeInterface, nil)

		return rt.ExecuteTransaction(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)
	}

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {
                    pub struct Foo {}
                }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: C.Foo) {}
        `

		err := executeTransaction(t, script, contracts, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Struct", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {
                    pub struct Foo {
                        pub var funcTypedField: (():Void)

                        init() {
                            self.funcTypedField = fun() {}
                        }
                    }
               }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: C.Foo?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))
		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: AnyStruct?) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Interface", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {
                    pub struct Foo: Bar {}

                    pub struct interface Bar {}
                }
            `),
		}
		script := `
          import C from 0x1

          transaction(arg: {C.Bar}) {}
        `

		err := executeTransaction(t, script, contracts, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Interface", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {
                    pub struct interface Bar {
                        pub var funcTypedField: (():Void)
                    }
                }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: {C.Bar}?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {
                    pub resource Baz {}
                }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: @C.Baz?) {}
 `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
			&sema.ResourceLossError{},
		)
	})

	t.Run("AnyResource", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: @AnyResource?) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
			&sema.ResourceLossError{},
		)
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                pub contract C {}
            `),
		}
		script := `
          import C from 0x1

          transaction(arg: C?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: [String]) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewArray([]cadence.Value{}))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Array", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: [(():Void)]) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewArray([]cadence.Value{}))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: {String: Bool}) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewDictionary([]cadence.KeyValuePair{}))
		assert.NoError(t, err)
	})

	t.Run("Capability", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: Capability<&Int>?) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: {String: (():Void)}) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewArray([]cadence.Value{}))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Numeric Types", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllNumberTypes {
			typString := typ.QualifiedString()

			t.Run(typString, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                      transaction(arg: %s?) {}
                    `,
					typString,
				)

				err := executeTransaction(t, script, nil, cadence.NewOptional(nil))
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Native composites", func(t *testing.T) {
		t.Parallel()

		type argumentPassingTest struct {
			argument      cadence.Value
			label         string
			typeSignature string
			expectErrors  bool
		}

		var argumentPassingTests []*argumentPassingTest

		for typeName, typ := range sema.NativeCompositeTypes {
			var value cadence.Value
			expectErrors := false

			switch typ {
			case sema.HashAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(1),
					},
				).WithType(HashAlgoType)

			case sema.SignatureAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(1),
					},
				).WithType(SignAlgoType)

			case sema.PublicKeyType:
				value = cadence.NewStruct(
					[]cadence.Value{
						// PublicKey bytes
						cadence.NewArray([]cadence.Value{}),

						// Sign algorithm
						cadence.NewEnum(
							[]cadence.Value{
								cadence.NewUInt8(1),
							},
						).WithType(SignAlgoType),
					},
				).WithType(PublicKeyType)

			default:
				// This test case only focuses on the type,
				// and has no interest in the value.
				value = nil

				expectErrors = true
			}

			testCase := &argumentPassingTest{
				label:         typeName,
				typeSignature: typeName + "?",
				argument:      cadence.NewOptional(value),
				expectErrors:  expectErrors,
			}

			argumentPassingTests = append(argumentPassingTests, testCase)
		}

		testArgumentPassing := func(test *argumentPassingTest) {

			t.Run(test.label, func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(
					`
                      transaction(arg: %s) {}
                    `,
					test.typeSignature,
				)

				err := executeTransaction(t, script, nil, test.argument)

				if test.expectErrors {
					expectCheckerErrors(
						t,
						err,
						&sema.InvalidNonImportableTransactionParameterTypeError{},
					)
				} else {
					assert.NoError(t, err)
				}
			})
		}

		for _, testCase := range argumentPassingTests {
			testArgumentPassing(testCase)
		}
	})

	t.Run("Invalid struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
               pub contract C {
                   pub struct Foo {
                      pub var nonImportableField: PublicAccount.Keys?

                      init() {
                          self.nonImportableField = nil
                      }
                  }
               }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: AnyStruct?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(newFooStruct()))
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid struct as valid interface", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
               pub contract C {
                    pub struct Foo: Bar {
                        pub var nonImportableField: PublicAccount.Keys?
                        init() {
                            self.nonImportableField = nil
                        }
                    }

                    pub struct interface Bar {}
               }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: {C.Bar}?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(newFooStruct()))
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid native struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: AnyStruct) {}
        `

		err := executeTransaction(t, script, nil, newPublicAccountKeys())
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})

	t.Run("Invalid native struct in array", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: [AnyStruct]) {}
        `

		err := executeTransaction(t,
			script,
			nil,
			cadence.NewArray([]cadence.Value{
				newPublicAccountKeys(),
			}),
		)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})

	t.Run("invalid HashAlgorithm", func(t *testing.T) {

		script := `
          transaction(arg: HashAlgorithm) {}
        `

		err := executeTransaction(t,
			script,
			nil,
			cadence.NewEnum(
				[]cadence.Value{
					cadence.NewUInt8(0),
				},
			).WithType(HashAlgoType),
		)
		RequireError(t, err)

		var entryPointErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &entryPointErr)
	})

	t.Run("invalid SignatureAlgorithm", func(t *testing.T) {

		script := `
          transaction(arg: SignatureAlgorithm) {}
        `
		err := executeTransaction(t,
			script,
			nil,
			cadence.NewEnum(
				[]cadence.Value{
					cadence.NewUInt8(0),
				},
			).WithType(SignAlgoType),
		)
		RequireError(t, err)

		var entryPointErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &entryPointErr)
	})
}
