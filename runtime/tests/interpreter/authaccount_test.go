package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/errors"
	"github.com/dapperlabs/cadence/runtime/interpreter"
	"github.com/dapperlabs/cadence/runtime/sema"
	"github.com/dapperlabs/cadence/runtime/stdlib"
	"github.com/dapperlabs/cadence/runtime/trampoline"
)

func testAccount(t *testing.T, auth bool, code string) (*interpreter.Interpreter, map[string]interpreter.OptionalValue) {

	address := interpreter.NewAddressValueFromBytes([]byte{42})

	var ty sema.Type
	var accountValue interpreter.Value

	if auth {
		panicFunction := interpreter.NewHostFunctionValue(func(invocation interpreter.Invocation) trampoline.Trampoline {
			panic(errors.NewUnreachableError())
		})

		ty = &sema.AuthAccountType{}
		accountValue = interpreter.NewAuthAccountValue(
			address,
			panicFunction,
			panicFunction,
			panicFunction,
		)
	} else {
		ty = &sema.PublicAccountType{}
		accountValue = interpreter.NewPublicAccountValue(address)
	}

	valueDeclarations := map[string]sema.ValueDeclaration{
		"account": stdlib.StandardLibraryValue{
			Name:       "account",
			Type:       ty,
			Kind:       common.DeclarationKindConstant,
			IsConstant: true,
		},
	}

	storedValues := map[string]interpreter.OptionalValue{}

	// NOTE: checker, getter and setter are very naive for testing purposes and don't remove nil values
	//

	storageChecker := func(_ *interpreter.Interpreter, _ common.Address, key string) bool {
		_, ok := storedValues[key]
		return ok
	}

	storageSetter := func(_ *interpreter.Interpreter, _ common.Address, key string, value interpreter.OptionalValue) {
		if _, ok := value.(interpreter.NilValue); ok {
			delete(storedValues, key)
		} else {
			storedValues[key] = value
		}
	}

	storageGetter := func(_ *interpreter.Interpreter, _ common.Address, key string) interpreter.OptionalValue {
		value := storedValues[key]
		if value == nil {
			return interpreter.NilValue{}
		}
		return value
	}

	inter := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithPredefinedValues(map[string]interpreter.Value{
					"account": accountValue,
				}),
				interpreter.WithStorageExistenceHandler(storageChecker),
				interpreter.WithStorageReadHandler(storageGetter),
				interpreter.WithStorageWriteHandler(storageSetter),
			},
		},
	)

	return inter, storedValues
}

func TestInterpretAuthAccountSave(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		inter, storedValues := testAccount(
			t,
			true,
			`
              resource R {}

              fun test() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }
            `,
		)

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

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

			inter, _ := testAccount(
				t,
				true,
				fmt.Sprintf(
					`
                      resource R {}

                      fun test() {
                          let r <- create R()
                          account.save(<-r, to: /%s/r)
                      }
                    `,
					domain.Identifier(),
				),
			)

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.InvalidPathDomainError{}, err)
		})

	}
}

func TestInterpretAuthAccountLoad(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		inter, storedValues := testAccount(
			t,
			true,
			`
              resource R {}

              resource R2 {}

              fun save() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }

              fun loadR(): @R? {
                  return <-account.load<@R>(from: /storage/r)
              }

              fun loadR2(): @R2? {
                  return <-account.load<@R2>(from: /storage/r)
              }
            `,
		)

		t.Run("save R and load R ", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, storedValues, 1)

			// first load

			value, err := inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).Value

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, storedValues, 0)

			// second load

			value, err = inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, interpreter.NilValue{}, value)
		})

		t.Run("save R and load R2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, storedValues, 1)

			// load

			value, err := inter.Invoke("loadR2")
			require.NoError(t, err)

			require.IsType(t, interpreter.NilValue{}, value)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, storedValues, 1)
		})
	})

	for _, domain := range common.AllPathDomainsByIdentifier {

		if domain == common.PathDomainStorage {
			continue
		}

		t.Run(fmt.Sprintf("invalid: %s domain", domain), func(t *testing.T) {

			inter, _ := testAccount(
				t,
				true,
				fmt.Sprintf(
					`
	                 resource R {}

	                 fun test(): @R? {
	                     return <-account.load<@R>(from: /%s/r)
	                 }
	               `,
					domain.Identifier(),
				),
			)

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.InvalidPathDomainError{}, err)
		})
	}
}

func TestInterpretAuthAccountBorrow(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		inter, storedValues := testAccount(
			t,
			true,
			`
              resource R {}

              resource R2 {}

              fun save() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }

              fun borrowR(): &R? {
                  return account.borrow<&R>(from: /storage/r)
              }

              fun borrowR2(): &R2? {
                  return account.borrow<&R2>(from: /storage/r)
              }
            `,
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, storedValues, 1)

		t.Run("borrow R ", func(t *testing.T) {

			// first borrow

			value, err := inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).Value

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, storedValues, 1)

			// TODO: should fail, i.e. return nil

			// second borrow

			value, err = inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).Value

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, storedValues, 1)
		})

		t.Run("borrow R2", func(t *testing.T) {

			value, err := inter.Invoke("borrowR2")
			require.NoError(t, err)

			require.IsType(t, interpreter.NilValue{}, value)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, storedValues, 1)
		})
	})

	for _, domain := range common.AllPathDomainsByIdentifier {

		if domain == common.PathDomainStorage {
			continue
		}

		t.Run(fmt.Sprintf("invalid: %s domain", domain), func(t *testing.T) {

			inter, _ := testAccount(
				t,
				true,
				fmt.Sprintf(
					`
	                  resource R {}

	                  fun test(): &R? {
	                      return account.borrow<&R>(from: /%s/r)
	                  }
	                `,
					domain.Identifier(),
				),
			)

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.InvalidPathDomainError{}, err)
		})
	}
}

func TestInterpretAuthAccountLink(t *testing.T) {

	for _, capabilityDomain := range []common.PathDomain{
		common.PathDomainPrivate,
		common.PathDomainPublic,
	} {

		t.Run(capabilityDomain.Name(), func(t *testing.T) {

			inter, storedValues := testAccount(
				t,
				true,
				fmt.Sprintf(
					`
	                  resource R {}

	                  resource R2 {}

	                  fun save() {
	                      let r <- create R()
	                      account.save(<-r, to: /storage/r)
	                  }

	                  fun linkR(): Capability? {
	                      return account.link<&R>(/%[1]s/r, target: /storage/r)
	                  }

	                  fun linkR2(): Capability? {
	                      return account.link<&R2>(/%[1]s/r2, target: /storage/r)
	                  }
	                `,
					capabilityDomain.Identifier(),
				),
			)

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, storedValues, 1)

			t.Run("link R", func(t *testing.T) {

				// first link

				value, err := inter.Invoke("linkR")
				require.NoError(t, err)

				require.IsType(t, &interpreter.SomeValue{}, value)

				innerValue := value.(*interpreter.SomeValue).Value

				assert.IsType(t, interpreter.CapabilityValue{}, innerValue)

				// stored value + link
				require.Len(t, storedValues, 2)

				// second link

				value, err = inter.Invoke("linkR")
				require.NoError(t, err)

				require.IsType(t, interpreter.NilValue{}, value)

				// NOTE: check loaded value was *not* removed from storage
				require.Len(t, storedValues, 2)
			})

			t.Run("link R2", func(t *testing.T) {

				// first link

				value, err := inter.Invoke("linkR2")
				require.NoError(t, err)

				require.IsType(t, &interpreter.SomeValue{}, value)

				innerValue := value.(*interpreter.SomeValue).Value

				assert.IsType(t, interpreter.CapabilityValue{}, innerValue)

				// stored value + link
				require.Len(t, storedValues, 3)

				// second link

				value, err = inter.Invoke("linkR2")
				require.NoError(t, err)

				require.IsType(t, interpreter.NilValue{}, value)

				// NOTE: check loaded value was *not* removed from storage
				require.Len(t, storedValues, 3)
			})
		})
	}

	for _, targetDomain := range common.AllPathDomainsByIdentifier {

		testName := fmt.Sprintf(
			"invalid: new capability path: storage domain, target path: %s domain",
			targetDomain,
		)

		t.Run(testName, func(t *testing.T) {

			inter, _ := testAccount(
				t,
				true,
				fmt.Sprintf(
					`
	                 resource R {}

	                 fun test() {
	                     account.link<&R>(/storage/r, target: /%s/r)
	                 }
	               `,
					targetDomain.Identifier(),
				),
			)

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.IsType(t, &interpreter.InvalidPathDomainError{}, err)
		})
	}
}
