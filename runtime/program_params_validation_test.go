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
	"fmt"
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptParameterTypeValidation(t *testing.T) {

	t.Parallel()

	assertNonImportableError := func(t *testing.T, err error) {
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		runtimeErr := err.(Error)

		assert.IsType(t, &ScriptParameterTypeNotImportableError{}, runtimeErr.Err)
	}

	fooStruct := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
	}

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
            }
        `

		_, err := importAndExportValuesFromScript(t, script, fooStruct)
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

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
	})

	t.Run("AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: AnyStruct?) {
            }
        `

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
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

		_, err := importAndExportValuesFromScript(t, script, fooStruct)
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

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
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

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
	})

	t.Run("AnyResource", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: @AnyResource?) {
                destroy arg
            }
        `

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: Foo?) {
            }

            pub contract Foo {
            }
        `

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: [String]) {
            }
        `

		_, err := importAndExportValuesFromScript(
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

		_, err := importAndExportValuesFromScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{}),
		)

		assertNonImportableError(t, err)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {String: Bool}) {
            }
        `

		_, err := importAndExportValuesFromScript(
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

		_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
		assertNonImportableError(t, err)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(arg: {String: (():Void)}) {
            }
        `

		_, err := importAndExportValuesFromScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{}),
		)

		assertNonImportableError(t, err)
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

				_, err := importAndExportValuesFromScript(t, script, cadence.NewOptional(nil))
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

						// isValid
						cadence.NewBool(false),
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

				_, err := importAndExportValuesFromScript(t, script, test.argument)

				if test.expectErrors {
					assertNonImportableError(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}

		for _, testCase := range argumentPassingTests {
			testArgumentPassing(testCase)
		}
	})
}

func TestTransactionParameterTypeValidation(t *testing.T) {

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

	fooStruct := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
		},
		Fields: []cadence.Value{},
	}

	executeTransaction := func(script string, arg cadence.Value) error {
		encodedArg, err := json.Encode(arg)
		require.NoError(t, err)

		rt := NewInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

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

		err := executeTransaction(script, fooStruct)
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

		err := executeTransaction(script, cadence.NewOptional(nil))
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

		err := executeTransaction(script, cadence.NewOptional(nil))
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

		err := executeTransaction(script, fooStruct)
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

		err := executeTransaction(script, cadence.NewOptional(nil))

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

		err := executeTransaction(script, cadence.NewOptional(nil))

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

		err := executeTransaction(script, cadence.NewOptional(nil))

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

		err := executeTransaction(script, cadence.NewOptional(nil))

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

		err := executeTransaction(script, cadence.NewArray([]cadence.Value{}))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Array", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: [(():Void)]) {
            }
        `

		err := executeTransaction(script, cadence.NewArray([]cadence.Value{}))

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

		err := executeTransaction(script, cadence.NewDictionary([]cadence.KeyValuePair{}))
		assert.NoError(t, err)
	})

	t.Run("Capability", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: Capability<&Int>?) {
            }
        `

		err := executeTransaction(script, cadence.NewOptional(nil))

		expectCheckerErrors(
			t,
			err,
			&sema.InvalidNonImportableTransactionParameterTypeError{},
		)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            transaction(arg: {String: (():Void)}) {
            }
        `

		err := executeTransaction(script, cadence.NewArray([]cadence.Value{}))

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

				err := executeTransaction(script, cadence.NewOptional(nil))
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

						// isValid
						cadence.NewBool(false),
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

				err := executeTransaction(script, test.argument)

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
}
