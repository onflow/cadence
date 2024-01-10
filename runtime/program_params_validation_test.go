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

package runtime_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
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
				QualifiedIdentifier: "Account.Keys",
				Fields:              []cadence.Field{},
			},
			Fields: []cadence.Value{},
		}
	}

	executeScript := func(t *testing.T, script string, arg cadence.Value) (err error) {
		var encodedArg []byte
		encodedArg, err = json.Encode(arg)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
          access(all)
          fun main(arg: Foo) {}

          access(all)
          struct Foo {}
        `

		err := executeScript(t, script, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Struct", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: Foo?) {}

          access(all)
          struct Foo {
              access(all)
              var funcTypedField: fun(): Void

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
          access(all)
          fun main(arg: AnyStruct?) {}
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Interface", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: {Bar}) {}

          access(all)
          struct Foo: Bar {}

          access(all)
          struct interface Bar {}
        `

		err := executeScript(t, script, newFooStruct())
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Interface", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: {Bar}?) {}

          access(all)
          struct interface Bar {

              access(all)
              var funcTypedField: fun(): Void
          }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: @Baz?) {
              destroy arg
          }

          access(all)
          resource Baz {}
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("AnyResource", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: @AnyResource?) {
              destroy arg
          }
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all)
            fun main(arg: Foo?) {}

            access(all)
            contract Foo {}
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		expectNonImportableError(t, err)
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: [String]) {}
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
          access(all)
          fun main(arg: [fun(): Void]) {}
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
          access(all)
          fun main(arg: {String: Bool}) {}
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
          access(all)
          fun main(arg: Capability<&Int>?) {}
        `

		err := executeScript(t, script, cadence.NewOptional(nil))
		assert.NoError(t, err)
	})

	t.Run("Non-Importable Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: {String: fun(): Void}) {}
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

				script := fmt.Sprintf(
					`
                      access(all)
                      fun main(arg: %s?) {}
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
			isResource    bool
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

			isResource := typ.IsResourceType()

			typeSignature := typeName + "?"
			if isResource {
				typeSignature = "@" + typeSignature
			}

			testCase := &argumentPassingTest{
				label:         typeName,
				typeSignature: typeSignature,
				argument:      cadence.NewOptional(value),
				expectErrors:  expectErrors,
				isResource:    isResource,
			}

			argumentPassingTests = append(argumentPassingTests, testCase)
		}

		testArgumentPassing := func(test *argumentPassingTest) {

			t.Run(test.label, func(t *testing.T) {
				t.Parallel()

				var body string
				if test.isResource {
					body = "destroy arg"
				}

				script := fmt.Sprintf(`
                    access(all) fun main(arg: %s) {
                        %s
                    }`,
					test.typeSignature,
					body,
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
          access(all)
          fun main(arg: AnyStruct?) {}

          access(all)
          struct Foo {

              access(all)
              var nonImportableField: Block?

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
          access(all)
          fun main(arg: {Bar}?) {}

          access(all)
          struct Foo: Bar {
              access(all)
              var nonImportableField: Block?

              init() {
                  self.nonImportableField = nil
              }
          }

          access(all)
          struct interface Bar {}
        `

		err := executeScript(t, script, newFooStruct())
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid native struct as AnyStruct", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all) fun main(arg: AnyStruct) {}
        `

		err := executeScript(t, script, newPublicAccountKeys())
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type Account.Keys")
	})

	t.Run("Invalid struct in array", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          fun main(arg: [AnyStruct]) {}
        `

		err := executeScript(
			t,
			script,
			cadence.NewArray([]cadence.Value{
				newPublicAccountKeys(),
			}),
		)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type Account.Keys")
	})

	t.Run("invalid HashAlgorithm", func(t *testing.T) {
		t.Parallel()

		err := executeScript(t,
			`
              access(all)
              fun main(arg: HashAlgorithm) {}
            `,
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
			`
              access(all)
              fun main(arg: SignatureAlgorithm) {}
		    `,
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

	newAccountKeysValue := func() cadence.Resource {
		return cadence.Resource{
			ResourceType: &cadence.ResourceType{
				QualifiedIdentifier: "Account.Keys",
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

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage:           storage,
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return contracts[location], nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
                access(all) contract C {
                    access(all) struct Foo {}
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
                access(all) contract C {
                    access(all) struct Foo {
                        access(all) var funcTypedField: fun (): Void

                        init() {
                            self.funcTypedField = fun () {}
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

		errs := checker.RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
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
                access(all) contract C {
                    access(all) struct Foo: Bar {}

                    access(all) struct interface Bar {}
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
                access(all) contract C {
                    access(all) struct interface Bar {
                        access(all) var funcTypedField: fun (): Void
                    }
                }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: {C.Bar}?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		errs := checker.RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
	})

	t.Run("Resource", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                access(all) contract C {
                    access(all) resource Baz {}
                }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: @C.Baz?) {}
 `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		errs := checker.RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
		require.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("AnyResource", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: @AnyResource?) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewOptional(nil))

		errs := checker.RequireCheckerErrors(t, err, 2)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
		require.IsType(t, &sema.ResourceLossError{}, errs[1])
	})

	t.Run("Contract", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
                access(all) contract C {}
            `),
		}
		script := `
          import C from 0x1

          transaction(arg: C?) {}
        `

		err := executeTransaction(t, script, contracts, cadence.NewOptional(nil))

		errs := checker.RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
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
          transaction(arg: [fun(): Void]) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewArray([]cadence.Value{}))

		errs := checker.RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
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
          transaction(arg: {String: fun(): Void}) {}
        `

		err := executeTransaction(t, script, nil, cadence.NewArray([]cadence.Value{}))

		errs := checker.RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
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
			isResource    bool
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

			isResource := typ.IsResourceType()

			typeSignature := typeName + "?"
			if isResource {
				typeSignature = "@" + typeSignature
			}

			testCase := &argumentPassingTest{
				label:         typeName,
				typeSignature: typeSignature,
				argument:      cadence.NewOptional(value),
				expectErrors:  expectErrors,
				isResource:    isResource,
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

					if test.isResource {
						errs := checker.RequireCheckerErrors(t, err, 2)

						require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
						require.IsType(t, &sema.ResourceLossError{}, errs[1])
					} else {
						errs := checker.RequireCheckerErrors(t, err, 1)

						require.IsType(t, &sema.InvalidNonImportableTransactionParameterTypeError{}, errs[0])
					}
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
               access(all) contract C {
                   access(all) struct Foo {
                      access(all) var nonImportableField: &Account.Keys?

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
               access(all) contract C {
                    access(all) struct Foo: Bar {
                        access(all) var nonImportableField: &Account.Keys?
                        init() {
                            self.nonImportableField = nil
                        }
                    }

                    access(all) struct interface Bar {}
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

		err := executeTransaction(t, script, nil, newAccountKeysValue())
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type Account.Keys")
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
				newAccountKeysValue(),
			}),
		)
		RequireError(t, err)

		assert.Contains(t, err.Error(), "cannot import value of type Account.Keys")
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

	t.Run("Invalid private cap in struct", func(t *testing.T) {
		t.Parallel()

		contracts := map[common.AddressLocation][]byte{
			{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "C",
			}: []byte(`
               access(all)
               contract C {

                    access(all)
                    struct S {

                        access(all)
                        let cap: Capability

                        init(cap: Capability) {
                            self.cap = cap
                        }
                    }
               }
            `),
		}

		script := `
          import C from 0x1

          transaction(arg: C.S) {}
        `

		address := common.MustBytesToAddress([]byte{0x1})

		capability := cadence.NewCapability(
			1,
			cadence.Address(address),
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.AccountType),
		)

		arg := cadence.Struct{
			StructType: &cadence.StructType{
				Location: common.AddressLocation{
					Address: address,
					Name:    "C",
				},
				QualifiedIdentifier: "C.S",
				Fields: []cadence.Field{
					{
						Identifier: "cap",
						Type:       &cadence.CapabilityType{},
					},
				},
			},
			Fields: []cadence.Value{
				capability,
			},
		}

		err := executeTransaction(t, script, contracts, arg)
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid private cap in array", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: [AnyStruct]) {}
        `

		address := common.MustBytesToAddress([]byte{0x1})

		capability := cadence.NewCapability(
			1,
			cadence.Address(address),
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.AccountType),
		)

		arg := cadence.Array{
			ArrayType: cadence.NewVariableSizedArrayType(cadence.AnyStructType),
			Values: []cadence.Value{
				capability,
			},
		}

		err := executeTransaction(t, script, nil, arg)
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid private cap in optional", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: AnyStruct?) {}
        `

		address := common.MustBytesToAddress([]byte{0x1})

		capability := cadence.NewCapability(
			1,
			cadence.Address(address),
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.AccountType),
		)

		arg := cadence.NewOptional(capability)

		err := executeTransaction(t, script, nil, arg)
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

	t.Run("Invalid private cap in dictionary value", func(t *testing.T) {
		t.Parallel()

		script := `
          transaction(arg: {String: AnyStruct}) {}
        `

		address := common.MustBytesToAddress([]byte{0x1})

		capability := cadence.NewCapability(
			1,
			cadence.Address(address),
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.AccountType),
		)

		arg := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("cap"),
				Value: capability,
			},
		})

		err := executeTransaction(t, script, nil, arg)
		expectRuntimeError(t, err, &ArgumentNotImportableError{})
	})

}
