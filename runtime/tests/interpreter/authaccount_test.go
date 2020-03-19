package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/flow-go/language/runtime/common"
	"github.com/dapperlabs/flow-go/language/runtime/errors"
	"github.com/dapperlabs/flow-go/language/runtime/interpreter"
	"github.com/dapperlabs/flow-go/language/runtime/sema"
	"github.com/dapperlabs/flow-go/language/runtime/stdlib"
	"github.com/dapperlabs/flow-go/language/runtime/trampoline"
)

func TestInterpretAuthAccount(t *testing.T) {

	panicFunction := interpreter.NewHostFunctionValue(func(invocation interpreter.Invocation) trampoline.Trampoline {
		panic(errors.NewUnreachableError())
	})

	address := interpreter.NewAddressValueFromBytes([]byte{42})

	authAccountValue := interpreter.NewAuthAccountValue(
		address,
		panicFunction,
		panicFunction,
		panicFunction,
	)

	valueDeclarations := map[string]sema.ValueDeclaration{
		"authAccount": stdlib.StandardLibraryValue{
			Name:       "authAccount",
			Type:       &sema.AuthAccountType{},
			Kind:       common.DeclarationKindConstant,
			IsConstant: true,
		},
	}

	t.Run("valid", func(t *testing.T) {

		storedValues := map[string]interpreter.OptionalValue{}

		// NOTE: checker, getter and setter are very naive for testing purposes and don't remove nil values
		//

		checked := false

		storageChecker := func(_ *interpreter.Interpreter, _ common.Address, key string) bool {
			checked = true

			_, ok := storedValues[key]
			return ok
		}

		storageSetter := func(_ *interpreter.Interpreter, _ common.Address, key string, value interpreter.OptionalValue) {
			storedValues[key] = value
		}

		inter := parseCheckAndInterpretWithOptions(t,
			`
              resource R {}

              fun test() {
                  let r <- create R()
                  authAccount.save(<-r, to: /storage/r)
              }
            `,
			ParseCheckAndInterpretOptions{
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues(valueDeclarations),
				},
				Options: []interpreter.Option{
					interpreter.WithPredefinedValues(map[string]interpreter.Value{
						"authAccount": authAccountValue,
					}),
					interpreter.WithStorageExistenceHandler(storageChecker),
					interpreter.WithStorageWriteHandler(storageSetter),
				},
			},
		)

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			assert.True(t, checked)

			require.Len(t, storedValues, 1)
			for _, value := range storedValues {

				require.IsType(t, &interpreter.SomeValue{}, value)

				innerValue := value.(*interpreter.SomeValue).Value

				assert.IsType(t, &interpreter.CompositeValue{}, innerValue)
			}

		})

		// Attempt to save again, overwriting should fail

		t.Run("second save", func(t *testing.T) {

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.OverwriteError{}, err)
		})
	})

	for _, domain := range common.AllPathDomainsByIdentifier {

		if domain == common.PathDomainStorage {
			continue
		}

		t.Run(fmt.Sprintf("invalid: %s domain", domain), func(t *testing.T) {

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          authAccount.save(<-r, to: /%s/r)
                      }
                    `,
					domain.Identifier(),
				),
				ParseCheckAndInterpretOptions{
					CheckerOptions: []sema.Option{
						sema.WithPredeclaredValues(valueDeclarations),
					},
					Options: []interpreter.Option{
						interpreter.WithPredefinedValues(map[string]interpreter.Value{
							"authAccount": authAccountValue,
						}),
					},
				},
			)

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.InvalidSavePathDomainError{}, err)
		})

	}
}
