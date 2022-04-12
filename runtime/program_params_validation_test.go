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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeScriptParameterTypeValidation(t *testing.T) {

	t.Parallel()

	expectNonImportableError := func(t *testing.T, err error) {
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, &ScriptParameterTypeNotImportableError{}, runtimeErr.Err)
	}

	expectRuntimeError := func(t *testing.T, err error, expectedError error) {
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, expectedError, runtimeErr.Err)
	}

	fooStruct := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
	}

	publicAccountKeys := cadence.Struct{
		StructType: &cadence.StructType{
			QualifiedIdentifier: "PublicAccount.Keys",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
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
				Location:  utils.TestLocation,
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

		err := executeScript(t, script, fooStruct)
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

		err := executeScript(t, script, fooStruct)
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

			t.Run(typ.QualifiedString(), func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                        pub fun main(arg: %s?) {
                        }
                    `,
					typ.QualifiedString(),
				)

				err := executeScript(t, script, cadence.NewOptional(nil))
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Native composites", func(t *testing.T) {
		t.Parallel()

		type argumentPassingTest struct {
			label         string
			typeSignature string
			argument      cadence.Value
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
						cadence.NewUInt8(0),
					},
				).WithType(HashAlgoType)

			case sema.SignatureAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
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
								cadence.NewUInt8(0),
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

		err := executeScript(t, script, fooStruct)
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

		err := executeScript(t, script, fooStruct)
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid native struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
                pub fun main(arg: AnyStruct) {
                }
            `

		err := executeScript(t, script, publicAccountKeys)
		require.Error(t, err)
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
				publicAccountKeys,
			}),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})
}

func TestRuntimeTransactionParameterTypeValidation(t *testing.T) {

	t.Parallel()

	expectCheckerErrors := func(t *testing.T, err error, expectedErrors ...error) {
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		require.IsType(t, &ParsingCheckingError{}, runtimeErr.Err)
		parsingCheckingErr := runtimeErr.Err.(*ParsingCheckingError)

		require.IsType(t, &sema.CheckerError{}, parsingCheckingErr.Err)
		checkerErr := parsingCheckingErr.Err.(*sema.CheckerError)

		for i, err := range expectedErrors {
			assert.IsType(t, err, checkerErr.Errors[i])
		}
	}

	expectRuntimeError := func(t *testing.T, err error, expectedError error) {
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, expectedError, runtimeErr.Err)
	}

	fooStruct := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
	}

	publicAccountKeys := cadence.Struct{
		StructType: &cadence.StructType{
			QualifiedIdentifier: "PublicAccount.Keys",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
	}

	executeTransaction := func(t *testing.T, script string, arg cadence.Value) (err error) {
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

		return rt.ExecuteTransaction(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)
	}

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: Foo) {
            }

            pub struct Foo {
            }
        `

		err := executeTransaction(t, script, fooStruct)
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Struct", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: Foo?) {
            }

            pub struct Foo {
                pub var funcTypedField: (():Void)

                init() {
                    self.funcTypedField = fun() {}
                }
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))
		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: AnyStruct?) {
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Interface", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: {Bar}) {
            }

            pub struct Foo: Bar {
            }

            pub struct interface Bar {
            }
        `

		err := executeTransaction(t, script, fooStruct)
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Interface", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: {Bar}?) {
            }

            pub struct interface Bar {
                pub var funcTypedField: (():Void)
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: @Baz?) {
            }

            pub resource Baz {
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))

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
            transaction(arg: @AnyResource?) {
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
			&sema.ResourceLossError{},
		)
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: Foo?) {
            }

            pub contract Foo {
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: [String]) {
            }
        `

		err := executeTransaction(t, script, cadence.NewArray([]cadence.Value{}))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Array", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: [(():Void)]) {
            }
        `

		err := executeTransaction(t, script, cadence.NewArray([]cadence.Value{}))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: {String: Bool}) {
            }
        `

		err := executeTransaction(t, script, cadence.NewDictionary([]cadence.KeyValuePair{}))
		assert.NoError(t, err)
	})

	t.Run("Capability", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: Capability<&Int>?) {
            }
        `

		err := executeTransaction(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: {String: (():Void)}) {
            }
        `

		err := executeTransaction(t, script, cadence.NewArray([]cadence.Value{}))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Numeric Types", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllNumberTypes {

			t.Run(typ.QualifiedString(), func(t *testing.T) {
				t.Parallel()

				script := fmt.Sprintf(`
                        transaction(arg: %s?) {
                        }
                    `,
					typ.QualifiedString(),
				)

				err := executeTransaction(t, script, cadence.NewOptional(nil))
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Native composites", func(t *testing.T) {
		t.Parallel()

		type argumentPassingTest struct {
			label         string
			typeSignature string
			argument      cadence.Value
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
						cadence.NewUInt8(0),
					},
				).WithType(HashAlgoType)

			case sema.SignatureAlgorithmType:
				value = cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
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
								cadence.NewUInt8(0),
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
                    transaction(arg: %s) {
                    }`,
					test.typeSignature,
				)

				err := executeTransaction(t, script, test.argument)

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

		script := `
                transaction(arg: AnyStruct?) {
                }
                pub struct Foo {
                    pub var nonImportableField: PublicAccount.Keys?
                    init() {
                        self.nonImportableField = nil
                    }
                }
            `

		err := executeTransaction(t, script, cadence.NewOptional(fooStruct))
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid struct as valid interface", func(t *testing.T) {
		t.Parallel()

		script := `
                transaction(arg: {Bar}?) {
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

		err := executeTransaction(t, script, cadence.NewOptional(fooStruct))
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid native struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
                transaction(arg: AnyStruct) {
                }
            `

		err := executeTransaction(t, script, publicAccountKeys)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})

	t.Run("Invalid native struct in array", func(t *testing.T) {
		t.Parallel()

		script := `
                transaction(arg: [AnyStruct]) {
                }
            `

		err := executeTransaction(t,
			script,
			cadence.NewArray([]cadence.Value{
				publicAccountKeys,
			}),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot import value of type PublicAccount.Keys")
	})
}
