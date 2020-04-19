package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testRuntimeInterfaceStorage struct {
	valueExists func(controller, owner, key []byte) (exists bool, err error)
	getValue    func(controller, owner, key []byte) (value []byte, err error)
	setValue    func(controller, owner, key, value []byte) (err error)
}

func newTestStorage() testRuntimeInterfaceStorage {

	storageKey := func(owner, controller, key string) string {
		return strings.Join([]string{owner, controller, key}, "|")
	}

	storedValues := map[string][]byte{}

	storage := testRuntimeInterfaceStorage{
		valueExists: func(controller, owner, key []byte) (bool, error) {
			_, ok := storedValues[storageKey(string(controller), string(owner), string(key))]
			return ok, nil
		},
		getValue: func(controller, owner, key []byte) (value []byte, err error) {
			return storedValues[storageKey(string(controller), string(owner), string(key))], nil
		},
		setValue: func(controller, owner, key, value []byte) (err error) {
			storedValues[storageKey(string(controller), string(owner), string(key))] = value
			return nil
		},
	}

	return storage
}

type testRuntimeInterface struct {
	resolveImport      func(Location) ([]byte, error)
	storage            testRuntimeInterfaceStorage
	createAccount      func(publicKeys [][]byte) (address Address, err error)
	addAccountKey      func(address Address, publicKey []byte) error
	removeAccountKey   func(address Address, index int) (publicKey []byte, err error)
	checkCode          func(address Address, code []byte) (err error)
	updateAccountCode  func(address Address, code []byte, checkPermission bool) (err error)
	getSigningAccounts func() []Address
	log                func(string)
	emitEvent          func(Event)
	generateUUID       func() uint64
}

func (i *testRuntimeInterface) ResolveImport(location Location) ([]byte, error) {
	return i.resolveImport(location)
}

func (i *testRuntimeInterface) ValueExists(controller, owner, key []byte) (exists bool, err error) {
	return i.storage.valueExists(controller, owner, key)
}

func (i *testRuntimeInterface) GetValue(controller, owner, key []byte) (value []byte, err error) {
	return i.storage.getValue(controller, owner, key)
}

func (i *testRuntimeInterface) SetValue(controller, owner, key, value []byte) (err error) {
	return i.storage.setValue(controller, owner, key, value)
}

func (i *testRuntimeInterface) CreateAccount(publicKeys [][]byte) (address Address, err error) {
	return i.createAccount(publicKeys)
}

func (i *testRuntimeInterface) AddAccountKey(address Address, publicKey []byte) error {
	return i.addAccountKey(address, publicKey)
}

func (i *testRuntimeInterface) RemoveAccountKey(address Address, index int) (publicKey []byte, err error) {
	return i.removeAccountKey(address, index)
}

func (i *testRuntimeInterface) CheckCode(address Address, code []byte) (err error) {
	return i.checkCode(address, code)
}

func (i *testRuntimeInterface) UpdateAccountCode(address Address, code []byte, checkPermission bool) (err error) {
	return i.updateAccountCode(address, code, checkPermission)
}

func (i *testRuntimeInterface) GetSigningAccounts() []Address {
	if i.getSigningAccounts == nil {
		return nil
	}
	return i.getSigningAccounts()
}

func (i *testRuntimeInterface) Log(message string) {
	i.log(message)
}

func (i *testRuntimeInterface) EmitEvent(event Event) {
	i.emitEvent(event)
}

func (i *testRuntimeInterface) GenerateUUID() uint64 {
	if i.generateUUID == nil {
		return 0
	}
	return i.generateUUID()
}

func TestRuntimeImport(t *testing.T) {

	runtime := NewInterpreterRuntime()

	importedScript := []byte(`
      pub fun answer(): Int {
        return 42
      }
    `)

	script := []byte(`
      import "imported"

      pub fun main(): Int {
          let answer = answer()
          if answer != 42 {
            panic("?!")
          }
          return answer
        }
    `)

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	value, err := runtime.ExecuteScript(script, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, interpreter.NewIntValue(42), value.Value)
}

func TestRuntimeInvalidTransactionArgumentAccount(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {}
        execute {}
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
	assert.Error(t, err)
}

func TestRuntimeTransactionWithAccount(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          log(signer.address)
        }
      }
    `)

	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return []Address{
				common.BytesToAddress([]byte{42}),
			}
		},
		log: func(message string) {
			loggedMessage = message
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, "0x2a", loggedMessage)
}

func TestRuntimeProgramWithNoTransaction(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main() {}
    `)

	runtimeInterface := &testRuntimeInterface{}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

	require.IsType(t, Error{}, err)
	err = err.(Error).Unwrap()
	assert.IsType(t, InvalidTransactionCountError{}, err)
}

func TestRuntimeProgramWithMultipleTransaction(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        execute {}
      }
      transaction {
        execute {}
      }
    `)

	runtimeInterface := &testRuntimeInterface{}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

	require.IsType(t, Error{}, err)
	err = err.(Error).Unwrap()
	assert.IsType(t, InvalidTransactionCountError{}, err)
}

func TestRuntimeStorage(t *testing.T) {

	tests := map[string]string{
		"resource": `
          let r <- signer.load<@R>(from: /storage/r)
          log(r == nil)
          destroy r

          signer.save(<-createR(), to: /storage/r)
          let r2 <- signer.load<@R>(from: /storage/r)
          log(r2 != nil)
          destroy r2
        `,
		"struct": `
          let s = signer.load<S>(from: /storage/s)
          log(s == nil)

          signer.save(S(), to: /storage/s)
          let s2 = signer.load<S>(from: /storage/s)
          log(s2 != nil)
        `,
		"resource array": `
		  let rs <- signer.load<@[R]>(from: /storage/rs)
		  log(rs == nil)
		  destroy rs

		  signer.save(<-[<-createR()], to: /storage/rs)
		  let rs2 <- signer.load<@[R]>(from: /storage/rs)
		  log(rs2 != nil)
		  destroy rs2
		`,
		"struct array": `
		  let s = signer.load<[S]>(from: /storage/s)
		  log(s == nil)

		  signer.save([S()], to: /storage/s)
		  let s2 = signer.load<[S]>(from: /storage/s)
		  log(s2 != nil)
		`,
		"resource dictionary": `
		  let rs <- signer.load<@{String: R}>(from: /storage/rs)
		  log(rs == nil)
		  destroy rs

          signer.save(<-{"r": <-createR()}, to: /storage/rs)
		  let rs2 <- signer.load<@{String: R}>(from: /storage/rs)
		  log(rs2 != nil)
		  destroy rs2
		`,
		"struct dictionary": `
		  let s = signer.load<{String: S}>(from: /storage/s)
		  log(s == nil)

          signer.save({"s": S()}, to: /storage/s)
		  let rs2 = signer.load<{String: S}>(from: /storage/s)
		  log(rs2 != nil)
		`,
	}

	for name, code := range tests {
		t.Run(name, func(t *testing.T) {
			runtime := NewInterpreterRuntime()

			imported := []byte(`
              pub resource R {}

              pub fun createR(): @R {
                return <-create R()
              }

              pub struct S {}
            `)

			script := []byte(fmt.Sprintf(`
                  import "imported"

                  transaction {
                    prepare(signer: AuthAccount) {
                      %s
                    }
                  }
                `,
				code,
			))

			var loggedMessages []string

			runtimeInterface := &testRuntimeInterface{
				resolveImport: func(location Location) ([]byte, error) {
					switch location {
					case StringLocation("imported"):
						return imported, nil
					default:
						return nil, fmt.Errorf("unknown import location: %s", location)
					}
				},
				storage: newTestStorage(),
				getSigningAccounts: func() []Address {
					return []Address{{42}}
				},
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
			}

			err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
			require.NoError(t, err)

			assert.Equal(t, []string{"true", "true"}, loggedMessages)
		})
	}
}

func TestRuntimeStorageMultipleTransactionsResourceWithArray(t *testing.T) {
	runtime := NewInterpreterRuntime()

	container := []byte(`
      pub resource Container {
        pub let values: [Int]

        init() {
          self.values = []
        }
      }

      pub fun createContainer(): @Container {
        return <-create Container()
      }
    `)

	script1 := []byte(`
      import "container"

      transaction {

        prepare(signer: AuthAccount) {
          signer.save(<-createContainer(), to: /storage/container)
          signer.link<&Container>(/public/container, target: /storage/container)
        }
      }
    `)

	script2 := []byte(`
      import "container"

      transaction {
        prepare(signer: AuthAccount) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.getCapability(/public/container)!.borrow<&Container>()!

          let length = ref.values.length
          ref.values.append(1)
          let length2 = ref.values.length
        }
      }
    `)

	script3 := []byte(`
      import "container"

      transaction {
        prepare(signer: AuthAccount) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.getCapability(/public/container)!.borrow<&Container>()!

          let length = ref.values.length
          ref.values.append(2)
          let length2 = ref.values.length
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("container"):
				return container, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script3, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)
}

// TestRuntimeStorageMultipleTransactionsResourceFunction tests a function call
// of a stored resource declared in an imported program
//
func TestRuntimeStorageMultipleTransactionsResourceFunction(t *testing.T) {
	runtime := NewInterpreterRuntime()

	deepThought := []byte(`
      pub resource DeepThought {

        pub fun answer(): Int {
          return 42
        }
      }

      pub fun createDeepThought(): @DeepThought {
        return <-create DeepThought()
      }
    `)

	script1 := []byte(`
      import "deep-thought"

      transaction {

        prepare(signer: AuthAccount) {
          signer.save(<-createDeepThought(), to: /storage/deepThought)
        }
      }
    `)

	script2 := []byte(`
      import "deep-thought"

      transaction {
        prepare(signer: AuthAccount) {
          let answer = signer.borrow<&DeepThought>(from: /storage/deepThought)?.answer()
          log(answer ?? 0)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("deep-thought"):
				return deepThought, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeStorageMultipleTransactionsResourceField tests reading a field
// of a stored resource declared in an imported program
//
func TestRuntimeStorageMultipleTransactionsResourceField(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource Number {
        pub(set) var n: Int
        init(_ n: Int) {
          self.n = n
        }
      }

      pub fun createNumber(_ n: Int): @Number {
        return <-create Number(n)
      }
    `)

	script1 := []byte(`
      import "imported"

      transaction {
        prepare(signer: AuthAccount) {
          signer.save(<-createNumber(42), to: /storage/number)
        }
      }
    `)

	script2 := []byte(`
      import "imported"

      transaction {
        prepare(signer: AuthAccount) {
          if let number <- signer.load<@Number>(from: /storage/number) {
            log(number.n)
            destroy number
          }
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeCompositeFunctionInvocationFromImportingProgram checks
// that member functions of imported composites can be invoked from an importing program.
// See https://github.com/dapperlabs/flow-go/issues/838
//
func TestRuntimeCompositeFunctionInvocationFromImportingProgram(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      // function must have arguments
      pub fun x(x: Int) {}

      // invocation must be in composite
      pub resource Y {
        pub fun x() {
          x(x: 1)
        }
      }

      pub fun createY(): @Y {
        return <-create Y()
      }
    `)

	script1 := []byte(`
      import Y, createY from "imported"

      transaction {
        prepare(signer: AuthAccount) {
          signer.save(<-createY(), to: /storage/y)
        }
      }
    `)

	script2 := []byte(`
      import Y from "imported"

      transaction {
        prepare(signer: AuthAccount) {
          let y <- signer.load<@Y>(from: /storage/y)
          y?.x()
          destroy y
        }
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)
}

func TestRuntimeResourceContractUseThroughReference(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource R {
        pub fun x() {
          log("x!")
        }
      }

      pub fun createR(): @R {
        return <- create R()
      }
    `)

	script1 := []byte(`
      import R, createR from "imported"

      transaction {

        prepare(signer: AuthAccount) {
          signer.save(<-createR(), to: /storage/r)
        }
      }
    `)

	script2 := []byte(`
      import R from "imported"

      transaction {

        prepare(signer: AuthAccount) {
          let ref = signer.borrow<&R>(from: /storage/r)!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractUseThroughLink(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource R {
        pub fun x() {
          log("x!")
        }
      }

      pub fun createR(): @R {
          return <- create R()
      }
    `)

	script1 := []byte(`
      import R, createR from "imported"

      transaction {

        prepare(signer: AuthAccount) {
          signer.save(<-createR(), to: /storage/r)
          signer.link<&R>(/public/r, target: /storage/r)
        }
      }
    `)

	script2 := []byte(`
      import R from "imported"

      transaction {
        prepare(signer: AuthAccount) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.getCapability(/public/r)!.borrow<&R>()!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractWithInterface(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported1 := []byte(`
      pub resource interface RI {
        pub fun x()
      }
    `)

	imported2 := []byte(`
      import RI from "imported1"

      pub resource R: RI {
        pub fun x() {
          log("x!")
        }
      }

      pub fun createR(): @R {
        return <- create R()
      }
    `)

	script1 := []byte(`
      import RI from "imported1"
      import R, createR from "imported2"

      transaction {
        prepare(signer: AuthAccount) {
          signer.save(<-createR(), to: /storage/r)
          signer.link<&AnyResource{RI}>(/public/r, target: /storage/r)
        }
      }
    `)

	// TODO: Get rid of the requirement that the underlying type must be imported.
	//   This requires properly initializing Interpreter.CompositeFunctions.
	//   Also initialize Interpreter.DestructorFunctions

	script2 := []byte(`
      import RI from "imported1"
      import R from "imported2"

      transaction {
        prepare(signer: AuthAccount) {
          let ref = signer.getCapability(/public/r)!.borrow<&AnyResource{RI}>()!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported1"):
				return imported1, nil
			case StringLocation("imported2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestParseAndCheckProgram(t *testing.T) {
	t.Run("ValidProgram", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte("pub fun test(): Int { return 42 }")
		runtimeInterface := &testRuntimeInterface{}

		err := runtime.ParseAndCheckProgram(script, runtimeInterface, utils.TestLocation)
		assert.NoError(t, err)
	})

	t.Run("InvalidSyntax", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte("invalid syntax")
		runtimeInterface := &testRuntimeInterface{}

		err := runtime.ParseAndCheckProgram(script, runtimeInterface, utils.TestLocation)
		assert.NotNil(t, err)
	})

	t.Run("InvalidSemantics", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`pub let a: Int = "b"`)
		runtimeInterface := &testRuntimeInterface{}

		err := runtime.ParseAndCheckProgram(script, runtimeInterface, utils.TestLocation)
		assert.NotNil(t, err)
	})
}

func TestRuntimeSyntaxError(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main(): String {
          return "Hello World!
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
	}

	_, err := runtime.ExecuteScript(script, runtimeInterface, utils.TestLocation)
	assert.Error(t, err)
}

func TestRuntimeStorageChanges(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource X {
        pub(set) var x: Int

        init() {
          self.x = 0
        }
      }

      pub fun createX(): @X {
          return <-create X()
      }
    `)

	script1 := []byte(`
      import X, createX from "imported"

      transaction {
        prepare(signer: AuthAccount) {
          signer.save(<-createX(), to: /storage/x)

          let ref = signer.borrow<&X>(from: /storage/x)!
          ref.x = 1
        }
      }
    `)

	script2 := []byte(`
      import X from "imported"

      transaction {
        prepare(signer: AuthAccount) {
          let ref = signer.borrow<&X>(from: /storage/x)!
          log(ref.x)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"1"}, loggedMessages)
}

func TestRuntimeAccountAddress(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          log(signer.address)
        }
      }
    `)

	var loggedMessages []string

	address := common.BytesToAddress([]byte{42})

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return []Address{address}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"0x2a"}, loggedMessages)
}

func TestRuntimePublicAccountAddress(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log(getAccount(0x42).address)
        }
      }
    `)

	var loggedMessages []string

	address := interpreter.NewAddressValueFromBytes([]byte{0x42})

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{fmt.Sprint(address)}, loggedMessages)
}

func TestRuntimeAccountPublishAndAccess(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource R {
        pub fun test(): Int {
          return 42
        }
      }

      pub fun createR(): @R {
        return <-create R()
      }
    `)

	script1 := []byte(`
      import "imported"

      transaction {
        prepare(signer: AuthAccount) {
          signer.save(<-createR(), to: /storage/r)
          signer.link<&R>(/public/r, target: /storage/r)
        }
      }
    `)

	address := common.Address{42}

	script2 := []byte(
		fmt.Sprintf(
			`
              import "imported"

              transaction {

                prepare(signer: AuthAccount) {
                  log(getAccount(0x%s).getCapability(/public/r)!.borrow<&R>()!.test())
                }
              }
            `,
			address,
		),
	)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) ([]byte, error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{address}
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script1, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(script2, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, []string{"42"}, loggedMessages)
}

func TestRuntimeTransactionWithUpdateAccountCodeEmpty(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {

          prepare(signer: AuthAccount) {
              signer.setCode([])
          }
      }
    `)

	var accountCode []byte
	var events []Event

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{{42}}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

	require.NoError(t, err)

	assert.NotNil(t, accountCode)
	assert.Len(t, events, 1)
}

func TestRuntimeTransactionWithCreateAccountEmpty(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          AuthAccount(publicKeys: [], code: [])
        }
        execute {}
      }
    `)

	var accountCode []byte
	var events []Event

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(),
		createAccount: func(publicKeys [][]byte) (address Address, err error) {
			return Address{42}, nil
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

	require.NoError(t, err)

	assert.NotNil(t, accountCode)
	assert.Len(t, events, 1)
}

func TestRuntimeCyclicImport(t *testing.T) {
	runtime := NewInterpreterRuntime()

	imported := []byte(`
      import "imported"
    `)

	script := []byte(
		`
          import "imported"

          transaction {
            execute {}
          }
        `,
	)

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			switch location {
			case StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		getSigningAccounts: func() []Address {
			return nil
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

	require.Error(t, err)
	require.IsType(t, Error{}, err)
	assert.IsType(t, ast.CyclicImportsError{}, err.(Error).Unwrap())
}

func ArrayValueFromBytes(bytes []byte) *interpreter.ArrayValue {
	byteValues := make([]interpreter.Value, len(bytes))

	for i, b := range bytes {
		byteValues[i] = interpreter.UInt8Value(b)
	}

	return interpreter.NewArrayValueUnownedNonCopying(byteValues...)
}

func TestRuntimeTransactionWithContractDeployment(t *testing.T) {

	expectSuccess := func(t *testing.T, err error, accountCode []byte, events []Event) {
		require.NoError(t, err)

		assert.NotNil(t, accountCode)
		assert.Len(t, events, 1)
	}

	expectFailure := func(t *testing.T, err error, accountCode []byte, events []Event) {
		require.Error(t, err)

		assert.Nil(t, accountCode)
		assert.Len(t, events, 0)
	}

	type argument interface {
		fmt.Stringer
		interpreter.Value
	}

	type test struct {
		name      string
		contract  string
		arguments []argument
		check     func(t *testing.T, err error, accountCode []byte, events []Event)
	}

	tests := []test{
		{
			name: "no arguments",
			contract: `
              pub contract Test {}
            `,
			arguments: []argument{},
			check:     expectSuccess,
		},
		{
			name: "with argument",
			contract: `
              pub contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []argument{
				interpreter.NewIntValue(1),
			},
			check: expectSuccess,
		},
		{
			name: "with incorrect argument",
			contract: `
              pub contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []argument{
				interpreter.BoolValue(true),
			},
			check: expectFailure,
		},
		{
			name: "additional argument",
			contract: `
              pub contract Test {}
            `,
			arguments: []argument{
				interpreter.NewIntValue(1),
			},
			check: expectFailure,
		},
		{
			name: "additional code which is invalid at top-level",
			contract: `
              pub contract Test {}

              fun test() {}
            `,
			arguments: []argument{},
			check:     expectFailure,
		},
	}

	t.Run("updateAccountCode", func(t *testing.T) {

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {

				contractArrayCode := ArrayValueFromBytes([]byte(test.contract)).String()

				argumentCodes := make([]string, len(test.arguments))

				for i, argument := range test.arguments {
					argumentCodes[i] = argument.String()
				}

				argumentCode := strings.Join(argumentCodes, ", ")
				if len(test.arguments) > 0 {
					argumentCode = ", " + argumentCode
				}

				script := []byte(fmt.Sprintf(
					`
                      transaction {

                          prepare(signer: AuthAccount) {
                              signer.setCode(%s%s)
                          }
                      }
                    `,
					contractArrayCode,
					argumentCode,
				))

				runtime := NewInterpreterRuntime()

				var accountCode []byte
				var events []Event

				runtimeInterface := &testRuntimeInterface{
					storage: newTestStorage(),
					getSigningAccounts: func() []Address {
						return []Address{{42}}
					},
					updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
						accountCode = code
						return nil
					},
					emitEvent: func(event Event) {
						events = append(events, event)
					},
				}

				err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

				test.check(t, err, accountCode, events)
			})
		}
	})

	t.Run("Account", func(t *testing.T) {

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {

				contractArrayCode := ArrayValueFromBytes([]byte(test.contract)).String()

				argumentCodes := make([]string, len(test.arguments))

				for i, argument := range test.arguments {
					argumentCodes[i] = argument.String()
				}

				argumentCode := strings.Join(argumentCodes, ", ")
				if len(test.arguments) > 0 {
					argumentCode = ", " + argumentCode
				}

				script := []byte(fmt.Sprintf(
					`
                      transaction {
                        prepare() {
                          AuthAccount(publicKeys: [], code: %s%s)
                        }
                        execute {}
                      }
                    `,
					contractArrayCode,
					argumentCode,
				))

				runtime := NewInterpreterRuntime()

				var accountCode []byte
				var events []Event

				runtimeInterface := &testRuntimeInterface{
					storage: newTestStorage(),
					createAccount: func(publicKeys [][]byte) (address Address, err error) {
						return Address{42}, nil
					},
					updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
						accountCode = code
						return nil
					},
					emitEvent: func(event Event) {
						events = append(events, event)
					},
				}

				err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)

				test.check(t, err, accountCode, events)
			})
		}
	})
}

func TestRuntimeContractAccount(t *testing.T) {

	runtime := NewInterpreterRuntime()

	addressValue := interpreter.AddressValue{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xCA, 0xDE,
	}

	contract := []byte(`
      pub contract Test {
          pub let address: Address

          init() {
              // field 'account' can be used, as it is considered initialized
              self.address = self.account.address
          }

          // test that both functions are linked back into restored composite values,
          // and also injected fields are injected back into restored composite values
          //
          pub fun test(): Address {
              return self.account.address
          }
      }
    `)

	script1 := []byte(`
      import Test from 0xCADE

      pub fun main(): Address {
          return Test.address
      }
    `)

	script2 := []byte(`
      import Test from 0xCADE

      pub fun main(): Address {
          return Test.test()
      }
    `)

	deploy := []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.setCode(%s)
              }
          }
        `,
		ArrayValueFromBytes(contract).String(),
	))

	var accountCode []byte
	var events []Event

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{addressValue.ToAddress()}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	t.Run("", func(t *testing.T) {
		value, err := runtime.ExecuteScript(script1, runtimeInterface, utils.TestLocation)
		require.NoError(t, err)

		assert.Equal(t, addressValue, value.Value)
	})

	t.Run("", func(t *testing.T) {
		value, err := runtime.ExecuteScript(script2, runtimeInterface, utils.TestLocation)
		require.NoError(t, err)

		assert.Equal(t, addressValue, value.Value)
	})
}

func TestRuntimeContractNestedResource(t *testing.T) {
	runtime := NewInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        pub contract Test {
            pub resource R {
                // test that the hello function is linked back into the nested resource
                // after being loaded from storage
                pub fun hello(): String {
                    return "Hello World!"
                }
            }

            init() {
                // store nested resource in account on deployment
                self.account.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
		import Test from 0x01

		transaction {

			prepare(acct: AuthAccount) {
				log(acct.borrow<&Test.R>(from: /storage/r)?.hello())
			}
		}
	`)

	deploy := []byte(fmt.Sprintf(
		`
        transaction {

            prepare(signer: AuthAccount) {
                signer.setCode(%s)
            }
        }
        `,
		ArrayValueFromBytes(contract).String(),
	))

	var accountCode []byte
	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{addressValue}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event Event) {},
		log: func(message string) {
			loggedMessage = message
		},
	}

	err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(tx, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t, `"Hello World!"`, loggedMessage)
}

const fungibleTokenContract = `
pub contract FungibleToken {

    pub resource interface Provider {
        pub fun withdraw(amount: Int): @Vault {
            pre {
                amount > 0:
                    "Withdrawal amount must be positive"
            }
            post {
                result.balance == amount:
                    "Incorrect amount returned"
            }
        }
    }

    pub resource interface Receiver {
        pub balance: Int

        init(balance: Int) {
            pre {
                balance >= 0:
                    "Initial balance must be non-negative"
            }
            post {
                self.balance == balance:
                    "Balance must be initialized to the initial balance"
            }
        }

        pub fun deposit(from: @AnyResource{Receiver}) {
            pre {
                from.balance > 0:
                    "Deposit balance needs to be positive!"
            }
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "Incorrect amount removed"
            }
        }
    }

    pub resource Vault: Provider, Receiver {

        pub var balance: Int

        init(balance: Int) {
            self.balance = balance
        }

        pub fun withdraw(amount: Int): @Vault {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // transfer combines withdraw and deposit into one function call
        pub fun transfer(to: &AnyResource{Receiver}, amount: Int) {
            pre {
                amount <= self.balance:
                    "Insufficient funds"
            }
            post {
                self.balance == before(self.balance) - amount:
                    "Incorrect amount removed"
            }
            to.deposit(from: <-self.withdraw(amount: amount))
        }

        pub fun deposit(from: @AnyResource{Receiver}) {
            self.balance = self.balance + from.balance
            destroy from
        }

        pub fun createEmptyVault(): @Vault {
            return <-create Vault(balance: 0)
        }
    }

    pub fun createEmptyVault(): @Vault {
        return <-create Vault(balance: 0)
    }

    pub resource VaultMinter {
        pub fun mintTokens(amount: Int, recipient: &AnyResource{Receiver}) {
            recipient.deposit(from: <-create Vault(balance: amount))
        }
    }

    init() {
        self.account.save(<-create Vault(balance: 30), to: /storage/vault)
        self.account.save(<-create VaultMinter(), to: /storage/minter)
    }
}
`

func TestRuntimeFungibleTokenUpdateAccountCode(t *testing.T) {

	runtime := NewInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := []byte(fmt.Sprintf(
		`
          transaction {

              prepare(signer: AuthAccount) {
                  signer.setCode(%s)
              }
          }
        `,
		ArrayValueFromBytes([]byte(fungibleTokenContract)).String(),
	))

	setup1Transaction := []byte(`
      import FungibleToken from 0x01

      transaction {

          prepare(acct: AuthAccount) {

              acct.link<&AnyResource{FungibleToken.Receiver}>(
                  /public/receiver,
                  target: /storage/vault
              )

              acct.link<&FungibleToken.Vault>(
                  /private/vault,
                  target: /storage/vault
              )
          }
      }
    `)

	setup2Transaction := []byte(`
      // NOTE: import location not the same as in setup1Transaction
      import FungibleToken from 0x01

      transaction {

          prepare(acct: AuthAccount) {
              let vault <- FungibleToken.createEmptyVault()

              acct.save(<-vault, to: /storage/vault)

              acct.link<&AnyResource{FungibleToken.Receiver}>(
                  /public/receiver,
                  target: /storage/vault
              )

              acct.link<&FungibleToken.Vault>(
                  /private/vault,
                  target: /storage/vault
              )
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			key := string(location.(AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{signerAccount}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			key := string(AddressLocation(address[:]).ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(setup1Transaction, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	signerAccount = address2Value

	err = runtime.ExecuteTransaction(setup2Transaction, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)
}

func TestRuntimeFungibleTokenCreateAccount(t *testing.T) {

	runtime := NewInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := []byte(fmt.Sprintf(
		`
          transaction {
            prepare(signer: AuthAccount) {
                AuthAccount(publicKeys: [], code: %s)
            }
            execute {}
          }
        `,
		ArrayValueFromBytes([]byte(fungibleTokenContract)).String(),
	))

	setup1Transaction := []byte(`
      import FungibleToken from 0x2

      transaction {

          prepare(acct: AuthAccount) {
              acct.link<&AnyResource{FungibleToken.Receiver}>(
                  /public/receiver,
                  target: /storage/vault
              )

              acct.link<&FungibleToken.Vault>(
                  /private/vault,
                  target: /storage/vault
              )
          }
      }
    `)

	setup2Transaction := []byte(`
      // NOTE: import location not the same as in setup1Transaction
      import FungibleToken from 0x02

      transaction {

          prepare(acct: AuthAccount) {
              let vault <- FungibleToken.createEmptyVault()

              acct.save(<-vault, to: /storage/vault)

              acct.link<&AnyResource{FungibleToken.Receiver}>(
                  /public/receiver,
                  target: /storage/vault
              )

              acct.link<&FungibleToken.Vault>(
                  /private/vault,
                  target: /storage/vault
              )
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			key := string(location.(AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(),
		createAccount: func(publicKeys [][]byte) (address Address, err error) {
			return address2Value, nil
		},
		getSigningAccounts: func() []Address {
			return []Address{signerAccount}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			key := string(AddressLocation(address[:]).ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(setup1Transaction, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(setup2Transaction, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)
}

func TestRuntimeInvokeStoredInterfaceFunction(t *testing.T) {

	runtime := NewInterpreterRuntime()

	makeDeployTransaction := func(code string) []byte {
		return []byte(fmt.Sprintf(
			`
              transaction {
                prepare(signer: AuthAccount) {
                  AuthAccount(publicKeys: [], code: %s)
                }
              }
            `,
			ArrayValueFromBytes([]byte(code)).String(),
		))
	}

	contractInterfaceCode := `
      pub contract interface TestContractInterface {

          pub resource interface RInterface {

              pub fun check(a: Int, b: Int) {
                  pre { a > 1 }
                  post { b > 1 }
              }
          }
      }
	`

	contractCode := `
	  import TestContractInterface from 0x2

	  pub contract TestContract: TestContractInterface {

	      pub resource R: TestContractInterface.RInterface {

	          pub fun check(a: Int, b: Int) {
	              pre { a < 3 }
                  post { b < 3 }
	          }
	      }

	      pub fun createR(): @R {
	          return <-create R()
	      }
	   }
	`

	setupCode := []byte(`
	  import TestContractInterface from 0x2
	  import TestContract from 0x3

	  transaction {
	      prepare(signer: AuthAccount) {
	          signer.save(<-TestContract.createR(), to: /storage/r)
	      }
	  }
	`)

	makeUseCode := func(a int, b int) []byte {
		return []byte(
			fmt.Sprintf(
				`
	              import TestContractInterface from 0x2

                  // NOTE: *not* importing concrete implementation.
                  //   Should be imported automatically when loading the value from storage

	              // import TestContract from 0x3

	              transaction {
	                  prepare(signer: AuthAccount) {
	                      signer.borrow<&AnyResource{TestContractInterface.RInterface}>(from: /storage/r)?.check(a: %d, b: %d)
	                  }
	              }
	            `,
				a,
				b,
			),
		)
	}

	accountCodes := map[string][]byte{}
	var events []Event

	var nextAccount byte = 0x2

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			key := string(location.(AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(),
		createAccount: func(publicKeys [][]byte) (address Address, err error) {
			result := interpreter.NewAddressValueFromBytes([]byte{nextAccount})
			nextAccount++
			return result.ToAddress(), nil
		},
		getSigningAccounts: func() []Address {
			return []Address{{0x1}}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			key := string(AddressLocation(address[:]).ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
	}

	err := runtime.ExecuteTransaction(
		makeDeployTransaction(contractInterfaceCode),
		runtimeInterface,
		utils.TestLocation,
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		makeDeployTransaction(contractCode),
		runtimeInterface,
		utils.TestLocation,
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		setupCode,
		runtimeInterface,
		utils.TestLocation,
	)
	require.NoError(t, err)

	for a := 1; a <= 3; a++ {
		for b := 1; b <= 3; b++ {

			t.Run(fmt.Sprintf("%d/%d", a, b), func(t *testing.T) {

				err = runtime.ExecuteTransaction(
					makeUseCode(a, b),
					runtimeInterface,
					utils.TestLocation,
				)

				if a == 2 && b == 2 {
					assert.NoError(t, err)
				} else {
					require.Error(t, err)
					require.IsType(t, Error{}, err)
					assert.IsType(t, &interpreter.ConditionError{}, err.(Error).Err)
				}
			})
		}
	}
}

func TestRuntimeBlock(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let block = getCurrentBlock()
          log(block.number)
          log(block.id)

          let previousBlock = block.previousBlock
          log(previousBlock?.number)
          log(previousBlock?.id)

          let nextBlock = block.nextBlock
          log(nextBlock?.number)
          log(nextBlock?.id)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() []Address {
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(script, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"1",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1]",
			"nil",
			"nil",
			"2",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2]",
		},
		loggedMessages,
	)
}

func TestRuntimeTransactionTopLevelDeclarations(t *testing.T) {

	t.Run("transaction with function", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`
          pub fun test() {}

          transaction {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() []Address {
				return nil
			},
		}

		err := runtime.ExecuteTransaction(script, runtimeInterface, TransactionLocation{})
		require.NoError(t, err)
	})

	t.Run("transaction with resource", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`
          pub resource R {}

          transaction {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() []Address {
				return nil
			},
		}

		err := runtime.ExecuteTransaction(script, runtimeInterface, TransactionLocation{})
		require.Error(t, err)

		require.IsType(t, Error{}, err)
		err = err.(Error).Unwrap()

		errs := utils.ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
	})
}

func TestRuntimeStoreIntegerTypes(t *testing.T) {

	runtime := NewInterpreterRuntime()

	addressValue := interpreter.AddressValue{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xCA, 0xDE,
	}

	for _, integerType := range sema.AllIntegerTypes {

		typeName := integerType.String()

		t.Run(typeName, func(t *testing.T) {

			contract := []byte(
				fmt.Sprintf(
					`
                      pub contract Test {

                          pub let n: %s

                          init() {
                              self.n = 42
                          }
                      }
                    `,
					typeName,
				),
			)

			deploy := []byte(
				fmt.Sprintf(
					`
                      transaction {

                          prepare(signer: AuthAccount) {
                              signer.setCode(%s)
                          }
                      }
                    `,
					ArrayValueFromBytes(contract).String(),
				),
			)

			var accountCode []byte
			var events []Event

			runtimeInterface := &testRuntimeInterface{
				resolveImport: func(_ Location) (bytes []byte, err error) {
					return accountCode, nil
				},
				storage: newTestStorage(),
				getSigningAccounts: func() []Address {
					return []Address{addressValue.ToAddress()}
				},
				updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
					accountCode = code
					return nil
				},
				emitEvent: func(event Event) {
					events = append(events, event)
				},
			}

			err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
			require.NoError(t, err)

			assert.NotNil(t, accountCode)
		})
	}
}

func TestInterpretResourceOwnerFieldUse(t *testing.T) {

	runtime := NewInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
      pub contract Test {

          pub resource R {

              pub fun logOwnerAddress() {
                log(self.owner?.address)
              }
          }

          pub fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := []byte(
		fmt.Sprintf(
			`
              transaction {

                  prepare(signer: AuthAccount) {
                      signer.setCode(%s)
                  }
              }
            `,
			ArrayValueFromBytes(contract).String(),
		),
	)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(account: AuthAccount) {
              let r <- Test.createR()
              log(r.owner?.address)
              r.logOwnerAddress()

              account.save(<-r, to: /storage/r)

              let ref = account.borrow<&Test.R>(from: /storage/r)!
              log(ref.owner?.address)
              ref.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []Event

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveImport: func(location Location) (bytes []byte, err error) {
			key := string(location.(AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(),
		getSigningAccounts: func() []Address {
			return []Address{address}
		},
		updateAccountCode: func(address Address, code []byte, checkPermission bool) (err error) {
			key := string(AddressLocation(address[:]).ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event Event) {
			events = append(events, event)
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	err := runtime.ExecuteTransaction(deploy, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(tx, runtimeInterface, utils.TestLocation)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{"nil", "nil", "0x1", "0x1"},
		loggedMessages,
	)
}
