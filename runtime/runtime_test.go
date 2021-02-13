/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testRuntimeInterfaceStorage struct {
	storedValues map[string][]byte
	valueExists  func(owner, key []byte) (exists bool, err error)
	getValue     func(owner, key []byte) (value []byte, err error)
	setValue     func(owner, key, value []byte) (err error)
}

func newTestStorage(
	onRead func(owner, key, value []byte),
	onWrite func(owner, key, value []byte),
) testRuntimeInterfaceStorage {

	storageKey := func(owner, key string) string {
		return strings.Join([]string{owner, key}, "|")
	}

	storedValues := map[string][]byte{}

	storage := testRuntimeInterfaceStorage{
		storedValues: storedValues,
		valueExists: func(owner, key []byte) (bool, error) {
			value := storedValues[storageKey(string(owner), string(key))]
			return len(value) > 0, nil
		},
		getValue: func(owner, key []byte) (value []byte, err error) {
			value = storedValues[storageKey(string(owner), string(key))]
			if onRead != nil {
				onRead(owner, key, value)
			}
			return value, nil
		},
		setValue: func(owner, key, value []byte) (err error) {
			storedValues[storageKey(string(owner), string(key))] = value
			if onWrite != nil {
				onWrite(owner, key, value)
			}
			return nil
		},
	}

	return storage
}

type testRuntimeInterface struct {
	resolveLocation           func(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
	getCode                   func(_ Location) ([]byte, error)
	getProgram                func(Location) (*interpreter.Program, error)
	setProgram                func(Location, *interpreter.Program) error
	storage                   testRuntimeInterfaceStorage
	createAccount             func(payer Address) (address Address, err error)
	addAccountKey             func(address Address, publicKey []byte) error
	removeAccountKey          func(address Address, index int) (publicKey []byte, err error)
	updateAccountContractCode func(address Address, name string, code []byte) error
	getAccountContractCode    func(address Address, name string) (code []byte, err error)
	removeAccountContractCode func(address Address, name string) (err error)
	getSigningAccounts        func() ([]Address, error)
	log                       func(string)
	emitEvent                 func(cadence.Event) error
	generateUUID              func() (uint64, error)
	computationLimit          uint64
	decodeArgument            func(b []byte, t cadence.Type) (cadence.Value, error)
	programParsed             func(location common.Location, duration time.Duration)
	programChecked            func(location common.Location, duration time.Duration)
	programInterpreted        func(location common.Location, duration time.Duration)
	valueEncoded              func(duration time.Duration)
	valueDecoded              func(duration time.Duration)
	unsafeRandom              func() (uint64, error)
	verifySignature           func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm string,
		hashAlgorithm string,
	) (bool, error)
	hash                   func(data []byte, hashAlgorithm string) ([]byte, error)
	setCadenceValue        func(owner Address, key string, value cadence.Value) (err error)
	getStorageUsed         func(_ Address) (uint64, error)
	getStorageCapacity     func(_ Address) (uint64, error)
	programs               map[common.LocationID]*interpreter.Program
	implementationDebugLog func(message string) error
}

// testRuntimeInterface should implement Interface
var _ Interface = &testRuntimeInterface{}

func (i *testRuntimeInterface) ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
	if i.resolveLocation == nil {
		return []ResolvedLocation{
			{
				Location:    location,
				Identifiers: identifiers,
			},
		}, nil
	}
	return i.resolveLocation(identifiers, location)
}

func (i *testRuntimeInterface) GetCode(location Location) ([]byte, error) {
	if i.getCode == nil {
		return nil, nil
	}
	return i.getCode(location)
}

func (i *testRuntimeInterface) GetProgram(location Location) (*interpreter.Program, error) {
	if i.getProgram == nil {
		if i.programs == nil {
			i.programs = map[common.LocationID]*interpreter.Program{}
		}
		return i.programs[location.ID()], nil
	}

	return i.getProgram(location)
}

func (i *testRuntimeInterface) SetProgram(location Location, program *interpreter.Program) error {
	if i.setProgram == nil {
		if i.programs == nil {
			i.programs = map[common.LocationID]*interpreter.Program{}
		}
		i.programs[location.ID()] = program
		return nil
	}

	return i.setProgram(location, program)
}

func (i *testRuntimeInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	return i.storage.valueExists(owner, key)
}

func (i *testRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	return i.storage.getValue(owner, key)
}

func (i *testRuntimeInterface) SetValue(owner, key, value []byte) (err error) {
	return i.storage.setValue(owner, key, value)
}

func (i *testRuntimeInterface) CreateAccount(payer Address) (address Address, err error) {
	return i.createAccount(payer)
}

func (i *testRuntimeInterface) AddAccountKey(address Address, publicKey []byte) error {
	return i.addAccountKey(address, publicKey)
}

func (i *testRuntimeInterface) RemoveAccountKey(address Address, index int) (publicKey []byte, err error) {
	return i.removeAccountKey(address, index)
}

func (i *testRuntimeInterface) UpdateAccountContractCode(address Address, name string, code []byte) (err error) {
	return i.updateAccountContractCode(address, name, code)
}

func (i *testRuntimeInterface) GetAccountContractCode(address Address, name string) (code []byte, err error) {
	return i.getAccountContractCode(address, name)
}

func (i *testRuntimeInterface) RemoveAccountContractCode(address Address, name string) (err error) {
	return i.removeAccountContractCode(address, name)
}

func (i *testRuntimeInterface) GetSigningAccounts() ([]Address, error) {
	if i.getSigningAccounts == nil {
		return nil, nil
	}
	return i.getSigningAccounts()
}

func (i *testRuntimeInterface) ProgramLog(message string) error {
	i.log(message)
	return nil
}

func (i *testRuntimeInterface) EmitEvent(event cadence.Event) error {
	return i.emitEvent(event)
}

func (i *testRuntimeInterface) GenerateUUID() (uint64, error) {
	if i.generateUUID == nil {
		return 0, nil
	}
	return i.generateUUID()
}

func (i *testRuntimeInterface) GetComputationLimit() uint64 {
	return i.computationLimit
}

func (i *testRuntimeInterface) SetComputationUsed(uint64) error {
	return nil
}

func (i *testRuntimeInterface) DecodeArgument(b []byte, t cadence.Type) (cadence.Value, error) {
	return i.decodeArgument(b, t)
}

func (i *testRuntimeInterface) ProgramParsed(location common.Location, duration time.Duration) {
	if i.programParsed == nil {
		return
	}
	i.programParsed(location, duration)
}

func (i *testRuntimeInterface) ProgramChecked(location common.Location, duration time.Duration) {
	if i.programChecked == nil {
		return
	}
	i.programChecked(location, duration)
}

func (i *testRuntimeInterface) ProgramInterpreted(location common.Location, duration time.Duration) {
	if i.programInterpreted == nil {
		return
	}
	i.programInterpreted(location, duration)
}

func (i *testRuntimeInterface) ValueEncoded(duration time.Duration) {
	if i.valueEncoded == nil {
		return
	}
	i.valueEncoded(duration)
}

func (i *testRuntimeInterface) ValueDecoded(duration time.Duration) {
	if i.valueDecoded == nil {
		return
	}
	i.valueDecoded(duration)
}

func (i *testRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 1, nil
}

func (i *testRuntimeInterface) GetBlockAtHeight(height uint64) (block Block, exists bool, err error) {

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, height)
	if err != nil {
		panic(err)
	}

	encoded := buf.Bytes()
	var hash BlockHash
	copy(hash[sema.BlockIDSize-len(encoded):], encoded)

	block = Block{
		Height:    height,
		View:      height,
		Hash:      hash,
		Timestamp: time.Unix(int64(height), 0).UnixNano(),
	}
	return block, true, nil
}

func (i *testRuntimeInterface) UnsafeRandom() (uint64, error) {
	if i.unsafeRandom == nil {
		return 0, nil
	}
	return i.unsafeRandom()
}

func (i *testRuntimeInterface) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm string,
	hashAlgorithm string,
) (bool, error) {
	if i.verifySignature == nil {
		return false, nil
	}
	return i.verifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (i *testRuntimeInterface) Hash(data []byte, hashAlgorithm string) ([]byte, error) {
	if i.hash == nil {
		return nil, nil
	}
	return i.hash(data, hashAlgorithm)
}

func (i *testRuntimeInterface) HighLevelStorageEnabled() bool {
	return i.setCadenceValue != nil
}

func (i *testRuntimeInterface) SetCadenceValue(owner common.Address, key string, value cadence.Value) (err error) {
	return i.setCadenceValue(owner, key, value)
}

func (i *testRuntimeInterface) GetStorageUsed(address Address) (uint64, error) {
	return i.getStorageUsed(address)
}

func (i *testRuntimeInterface) GetStorageCapacity(address Address) (uint64, error) {
	return i.getStorageCapacity(address)
}

func (i *testRuntimeInterface) ImplementationDebugLog(message string) error {
	if i.implementationDebugLog == nil {
		return nil
	}
	return i.implementationDebugLog(message)
}

func TestRuntimeImport(t *testing.T) {

	t.Parallel()

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

	var checkCount int

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		programChecked: func(location common.Location, duration time.Duration) {
			checkCount += 1
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	const transactionCount = 10
	for i := 0; i < transactionCount; i++ {

		value, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, cadence.NewInt(42), value)
	}
	require.Equal(t, transactionCount+1, checkCount)
}

func TestRuntimeConcurrentImport(t *testing.T) {

	t.Parallel()

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

	var checkCount uint64
	var programsLock sync.RWMutex
	programs := map[common.LocationID]*interpreter.Program{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		programChecked: func(location common.Location, duration time.Duration) {
			atomic.AddUint64(&checkCount, 1)
		},
		setProgram: func(location Location, program *interpreter.Program) error {
			programsLock.Lock()
			defer programsLock.Unlock()

			programs[location.ID()] = program

			return nil
		},
		getProgram: func(location Location) (*interpreter.Program, error) {
			programsLock.RLock()
			defer programsLock.RUnlock()

			program := programs[location.ID()]

			return program, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	var wg sync.WaitGroup
	const concurrency uint64 = 10
	for i := uint64(0); i < concurrency; i++ {

		location := nextTransactionLocation()

		wg.Add(1)
		go func() {
			defer wg.Done()

			value, err := runtime.ExecuteScript(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  location,
				},
			)
			require.NoError(t, err)

			assert.Equal(t, cadence.NewInt(42), value)
		}()
	}
	wg.Wait()

	// TODO:
	//   Ideally we would expect the imported program only be checked once
	//   (`concurrency` transactions + 1 for the imported program),
	//   however, currently the imported program gets re-checked if it is currently being checked.
	//   This can probably be optimized by synchronizing the checking of a program using `sync`.
	//
	//require.Equal(t, concurrency+1, checkCount)
}

func TestRuntimeProgramSetAndGet(t *testing.T) {

	t.Parallel()

	programs := map[common.LocationID]*interpreter.Program{}
	programsHits := make(map[common.LocationID]bool)

	importedScript := []byte(`
      transaction {
          prepare() {}
          execute {}
      }
	`)
	importedScriptLocation := common.StringLocation("imported")

	runtime := NewInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{
		getProgram: func(location common.Location) (*interpreter.Program, error) {
			program, found := programs[location.ID()]
			programsHits[location.ID()] = found
			if !found {
				return nil, nil
			}
			return program, nil
		},
		setProgram: func(location common.Location, program *interpreter.Program) error {
			programs[location.ID()] = program
			return nil
		},
		getCode: func(location Location) ([]byte, error) {
			switch location {
			case importedScriptLocation:
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	t.Run("empty programs, miss", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Initial call, should parse script, store program.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Program was added to stored programs.
		storedProgram, exists := programs[scriptLocation.ID()]
		assert.True(t, exists)
		assert.NotNil(t, storedProgram)

		// Script was not in stored programs.
		assert.False(t, programsHits[scriptLocation.ID()])
	})

	t.Run("program previously parsed, hit", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Call a second time to hit stored programs.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Script was not in stored programs.
		assert.False(t, programsHits[scriptLocation.ID()])
	})

	t.Run("imported program previously parsed, hit", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Call a second time to hit the stored programs
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Script was not in stored programs.
		assert.False(t, programsHits[scriptLocation.ID()])
		// Import was in stored programs.
		assert.True(t, programsHits[importedScriptLocation.ID()])
	})
}

func newTransactionLocationGenerator() func() common.TransactionLocation {
	var transactionCount uint8
	return func() common.TransactionLocation {
		defer func() { transactionCount++ }()
		return common.TransactionLocation{transactionCount}
	}
}

func TestRuntimeInvalidTransactionArgumentAccount(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {}
        execute {}
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	assert.Error(t, err)
}

func TestRuntimeTransactionWithAccount(t *testing.T) {

	t.Parallel()

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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{
				common.BytesToAddress([]byte{42}),
			}, nil
		},
		log: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "0x2a", loggedMessage)
}

func TestRuntimeTransactionWithArguments(t *testing.T) {

	t.Parallel()

	type testCase struct {
		label        string
		script       string
		args         [][]byte
		authorizers  []Address
		expectedLogs []string
		check        func(t *testing.T, err error)
	}

	var tests = []testCase{
		{
			label: "Single argument",
			script: `
			  transaction(x: Int) {
				execute {
				  log(x)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
			},
			expectedLogs: []string{"42"},
		},
		{
			label: "Single argument with authorizer",
			script: `
			  transaction(x: Int) {
				prepare(signer: AuthAccount) {
				  log(signer.address)
				}

				execute {
				  log(x)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
			},
			authorizers:  []Address{common.BytesToAddress([]byte{42})},
			expectedLogs: []string{"0x2a", "42"},
		},
		{
			label: "Multiple arguments",
			script: `
			  transaction(x: Int, y: String) {
				execute {
				  log(x)
				  log(y)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
				jsoncdc.MustEncode(cadence.NewString("foo")),
			},
			expectedLogs: []string{"42", `"foo"`},
		},
		{
			label: "Invalid bytes",
			script: `
			  transaction(x: Int) { execute {} }
			`,
			args: [][]byte{
				{1, 2, 3, 4}, // not valid JSON-CDC
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
			},
		},
		{
			label: "Type mismatch",
			script: `
			  transaction(x: Int) {
				execute {
				  log(x)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewString("foo")),
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidTypeAssignmentError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			label: "Address",
			script: `
			  transaction(x: Address) {
				execute {
				  let acct = getAccount(x)
				  log(acct.address)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.BytesToAddress(
						[]byte{
							0x0, 0x0, 0x0, 0x0,
							0x0, 0x0, 0x0, 0x1,
						},
					),
				),
			},
			expectedLogs: []string{"0x1"},
		},
		{
			label: "Array",
			script: `
			  transaction(x: [Int]) {
				execute {
				  log(x)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewArray(
						[]cadence.Value{
							cadence.NewInt(1),
							cadence.NewInt(2),
							cadence.NewInt(3),
						},
					),
				),
			},
			expectedLogs: []string{"[1, 2, 3]"},
		},
		{
			label: "Dictionary",
			script: `
			  transaction(x: {String:Int}) {
				execute {
				  log(x["y"])
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.NewString("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			expectedLogs: []string{"42"},
		},
		{
			label: "Invalid dictionary",
			script: `
			  transaction(x: {String:String}) {
				execute {
				  log(x["y"])
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.NewString("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidTypeAssignmentError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			label: "Struct",
			script: `
			  pub struct Foo {
				pub var y: String

				init() {
				  self.y = "initial string"
				}
 			  }

			  transaction(x: Foo) {
				execute {
				  log(x.y)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.
						NewStruct([]cadence.Value{cadence.NewString("bar")}).
						WithType(&cadence.StructType{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Foo",
							Fields: []cadence.Field{
								{
									Identifier: "y",
									Type:       cadence.StringType{},
								},
							},
						}),
				),
			},
			expectedLogs: []string{`"bar"`},
		},
		{
			label: "Struct in array",
			script: `
			  pub struct Foo {
				pub var y: String

				init() {
				  self.y = "initial string"
				}
 			  }

			  transaction(f: [Foo]) {
				execute {
				  let x = f[0]
				  log(x.y)
				}
			  }
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewArray([]cadence.Value{
						cadence.
							NewStruct([]cadence.Value{cadence.NewString("bar")}).
							WithType(&cadence.StructType{
								Location:            utils.TestLocation,
								QualifiedIdentifier: "Foo",
								Fields: []cadence.Field{
									{
										Identifier: "y",
										Type:       cadence.StringType{},
									},
								},
							}),
					}),
				),
			},
			expectedLogs: []string{`"bar"`},
		},
	}

	test := func(tc testCase) {

		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()
			rt := NewInterpreterRuntime()

			var loggedMessages []string

			runtimeInterface := &testRuntimeInterface{
				getSigningAccounts: func() ([]Address, error) {
					return tc.authorizers, nil
				},
				decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
					return jsoncdc.Decode(b)
				},
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
			}

			err := rt.ExecuteTransaction(
				Script{
					Source:    []byte(tc.script),
					Arguments: tc.args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  utils.TestLocation,
				},
			)

			if tc.check != nil {
				tc.check(t, err)
			} else {
				if !assert.NoError(t, err) {
					for err := err; err != nil; err = errors.Unwrap(err) {
						t.Log(err)
					}
				}
				assert.ElementsMatch(t, tc.expectedLogs, loggedMessages)
			}
		})
	}

	for _, tt := range tests {
		test(tt)
	}
}

func TestRuntimeScriptArguments(t *testing.T) {

	t.Parallel()

	type testCase struct {
		label        string
		script       string
		args         [][]byte
		expectedLogs []string
		check        func(t *testing.T, err error)
	}

	var tests = []testCase{
		{
			label: "No arguments",
			script: `
				pub fun main() {
					log("t")
				}
			`,
			args:         nil,
			expectedLogs: []string{`"t"`},
		},
		{
			label: "Single argument",
			script: `
				pub fun main(x: Int) {
					log(x)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
			},
			expectedLogs: []string{"42"},
		},
		{
			label: "Multiple arguments",
			script: `
				pub fun main(x: Int, y: String) {
					log(x)
					log(y)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
				jsoncdc.MustEncode(cadence.NewString("foo")),
			},
			expectedLogs: []string{"42", `"foo"`},
		},
		{
			label: "Invalid bytes",
			script: `
				pub fun main(x: Int) { }
			`,
			args: [][]byte{
				{1, 2, 3, 4}, // not valid JSON-CDC
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
			},
		},
		{
			label: "Type mismatch",
			script: `
				pub fun main(x: Int) {
					log(x)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewString("foo")),
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidTypeAssignmentError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			label: "Address",
			script: `
				pub fun main(x: Address) {
					log(x)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.BytesToAddress(
						[]byte{
							0x0, 0x0, 0x0, 0x0,
							0x0, 0x0, 0x0, 0x1,
						},
					),
				),
			},
			expectedLogs: []string{"0x1"},
		},
		{
			label: "Array",
			script: `
				pub fun main(x: [Int]) {
					log(x)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewArray(
						[]cadence.Value{
							cadence.NewInt(1),
							cadence.NewInt(2),
							cadence.NewInt(3),
						},
					),
				),
			},
			expectedLogs: []string{"[1, 2, 3]"},
		},
		{
			label: "Dictionary",
			script: `
				pub fun main(x: {String:Int}) {
					log(x["y"])
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.NewString("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			expectedLogs: []string{"42"},
		},
		{
			label: "Invalid dictionary",
			script: `
				pub fun main(x: {String:String}) {
					log(x["y"])
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.NewString("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidTypeAssignmentError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			label: "Struct",
			script: `
				pub struct Foo {
					pub var y: String

					init() {
						self.y = "initial string"
					}
				}

				pub fun main(x: Foo) {
					log(x.y)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.
						NewStruct([]cadence.Value{cadence.NewString("bar")}).
						WithType(&cadence.StructType{
							Location:            utils.TestLocation,
							QualifiedIdentifier: "Foo",
							Fields: []cadence.Field{
								{
									Identifier: "y",
									Type:       cadence.StringType{},
								},
							},
						}),
				),
			},
			expectedLogs: []string{`"bar"`},
		},
		{
			label: "Struct in array",
			script: `
				pub struct Foo {
					pub var y: String

					init() {
						self.y = "initial string"
					}
				}

				pub fun main(f: [Foo]) {
					let x = f[0]
					log(x.y)
				}
			`,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewArray([]cadence.Value{
						cadence.
							NewStruct([]cadence.Value{cadence.NewString("bar")}).
							WithType(&cadence.StructType{
								Location:            utils.TestLocation,
								QualifiedIdentifier: "Foo",
								Fields: []cadence.Field{
									{
										Identifier: "y",
										Type:       cadence.StringType{},
									},
								},
							}),
					}),
				),
			},
			expectedLogs: []string{`"bar"`},
		},
	}

	test := func(tt testCase) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			rt := NewInterpreterRuntime()

			var loggedMessages []string

			runtimeInterface := &testRuntimeInterface{
				decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
					return jsoncdc.Decode(b)
				},
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
			}

			_, err := rt.ExecuteScript(
				Script{
					Source:    []byte(tt.script),
					Arguments: tt.args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  utils.TestLocation,
				},
			)

			if tt.check != nil {
				tt.check(t, err)
			} else {
				if !assert.NoError(t, err) {
					for err := err; err != nil; err = errors.Unwrap(err) {
						t.Log(err)
					}
				}
				assert.ElementsMatch(t, tt.expectedLogs, loggedMessages)
			}
		})
	}

	for _, tt := range tests {
		test(tt)
	}
}

func TestRuntimeProgramWithNoTransaction(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main() {}
    `)

	runtimeInterface := &testRuntimeInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	utils.RequireErrorAs(t, err, &InvalidTransactionCountError{})
}

func TestRuntimeProgramWithMultipleTransaction(t *testing.T) {

	t.Parallel()

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

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	utils.RequireErrorAs(t, err, &InvalidTransactionCountError{})
}

func TestRuntimeStorage(t *testing.T) {

	t.Parallel()

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
				getCode: func(location Location) ([]byte, error) {
					switch location {
					case common.StringLocation("imported"):
						return imported, nil
					default:
						return nil, fmt.Errorf("unknown import location: %s", location)
					}
				},
				storage: newTestStorage(nil, nil),
				getSigningAccounts: func() ([]Address, error) {
					return []Address{{42}}, nil
				},
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
			}

			nextTransactionLocation := newTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			assert.Equal(t, []string{"true", "true"}, loggedMessages)
		})
	}
}

func TestRuntimeStorageMultipleTransactionsResourceWithArray(t *testing.T) {

	t.Parallel()

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
          let ref = publicAccount.getCapability(/public/container)
              .borrow<&Container>()!

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
          let ref = publicAccount
              .getCapability(/public/container)
              .borrow<&Container>()!

          let length = ref.values.length
          ref.values.append(2)
          let length2 = ref.values.length
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("container"):
				return container, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script3,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

// TestRuntimeStorageMultipleTransactionsResourceFunction tests a function call
// of a stored resource declared in an imported program
//
func TestRuntimeStorageMultipleTransactionsResourceFunction(t *testing.T) {

	t.Parallel()

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
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("deep-thought"):
				return deepThought, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeStorageMultipleTransactionsResourceField tests reading a field
// of a stored resource declared in an imported program
//
func TestRuntimeStorageMultipleTransactionsResourceField(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	imported := []byte(`
      pub resource SomeNumber {
        pub(set) var n: Int
        init(_ n: Int) {
          self.n = n
        }
      }

      pub fun createNumber(_ n: Int): @SomeNumber {
        return <-create SomeNumber(n)
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
          if let number <- signer.load<@SomeNumber>(from: /storage/number) {
            log(number.n)
            destroy number
          }
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeCompositeFunctionInvocationFromImportingProgram checks
// that member functions of imported composites can be invoked from an importing program.
// See https://github.com/dapperlabs/flow-go/issues/838
//
func TestRuntimeCompositeFunctionInvocationFromImportingProgram(t *testing.T) {

	t.Parallel()

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
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeResourceContractUseThroughReference(t *testing.T) {

	t.Parallel()

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
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractUseThroughLink(t *testing.T) {

	t.Parallel()

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
          let ref = publicAccount
              .getCapability(/public/r)
              .borrow<&R>()!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractWithInterface(t *testing.T) {

	t.Parallel()

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
          let ref = signer
              .getCapability(/public/r)
              .borrow<&AnyResource{RI}>()!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported1"):
				return imported1, nil
			case common.StringLocation("imported2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestParseAndCheckProgram(t *testing.T) {

	t.Parallel()

	t.Run("ValidProgram", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte("pub fun test(): Int { return 42 }")
		runtimeInterface := &testRuntimeInterface{}

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NoError(t, err)
	})

	t.Run("InvalidSyntax", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte("invalid syntax")
		runtimeInterface := &testRuntimeInterface{}

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})

	t.Run("InvalidSemantics", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`pub let a: Int = "b"`)
		runtimeInterface := &testRuntimeInterface{}

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})
}

func TestScriptReturnTypeNotReturnableError(t *testing.T) {

	t.Parallel()

	test := func(code string, expected cadence.Value) {

		runtime := NewInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		actual, err := runtime.ExecuteScript(
			Script{
				Source: []byte(code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		if expected == nil {
			var subErr *InvalidScriptReturnTypeError
			utils.RequireErrorAs(t, err, &subErr)
		} else {
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		}
	}

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		test(
			`
              pub fun main(): ((): Int) {
                  return fun (): Int {
                      return 0
                  }
              }
            `,
			nil,
		)
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		test(
			`
              pub fun main(): &Address {
                  let a: Address = 0x1
                  return &a as &Address
              }
            `,
			cadence.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
		)
	})

	t.Run("recursive reference", func(t *testing.T) {

		t.Parallel()

		test(
			`
              pub fun main(): [&AnyStruct] {
                  let refs: [&AnyStruct] = []
                  refs.append(&refs as &AnyStruct)
                  return refs
              }
            `,
			cadence.NewArray([]cadence.Value{
				cadence.NewArray([]cadence.Value{
					nil,
				}),
			}),
		)
	})
}

func TestScriptParameterTypeNotStorableError(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main(x: ((): Int)) {
		return
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	var subErr *ScriptParameterTypeNotStorableError
	utils.RequireErrorAs(t, err, &subErr)
}

func TestRuntimeSyntaxError(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main(): String {
          return "Hello World!
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	assert.Error(t, err)
}

func TestRuntimeStorageChanges(t *testing.T) {

	t.Parallel()

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
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"1"}, loggedMessages)
}

func TestRuntimeAccountAddress(t *testing.T) {

	t.Parallel()

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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"0x2a"}, loggedMessages)
}

func TestRuntimePublicAccountAddress(t *testing.T) {

	t.Parallel()

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
		getSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{fmt.Sprint(address)}, loggedMessages)
}

func TestRuntimeAccountPublishAndAccess(t *testing.T) {

	t.Parallel()

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

	address := common.BytesToAddress([]byte{42})

	script2 := []byte(
		fmt.Sprintf(
			`
              import "imported"

              transaction {

                prepare(signer: AuthAccount) {
                  log(getAccount(0x%s).getCapability(/public/r).borrow<&R>()!.test())
                }
              }
            `,
			address,
		),
	)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) ([]byte, error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"42"}, loggedMessages)
}

func TestRuntimeTransaction_CreateAccount(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          AuthAccount(payer: signer)
        }
      }
    `)

	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		createAccount: func(payer Address) (address Address, err error) {
			return Address{42}, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.EqualValues(t, stdlib.AccountCreatedEventType.ID(), events[0].Type().ID())
}

func TestRuntimeTransaction_AddPublicKey(t *testing.T) {
	runtime := NewInterpreterRuntime()

	keyA := cadence.NewArray([]cadence.Value{
		cadence.NewUInt8(1),
		cadence.NewUInt8(2),
		cadence.NewUInt8(3),
	})

	keyB := cadence.NewArray([]cadence.Value{
		cadence.NewUInt8(4),
		cadence.NewUInt8(5),
		cadence.NewUInt8(6),
	})

	keys := cadence.NewArray([]cadence.Value{
		keyA,
		keyB,
	})

	var tests = []struct {
		name     string
		code     string
		keyCount int
		args     []cadence.Value
		expected [][]byte
	}{
		{
			name: "Single key",
			code: `
			  transaction(keyA: [UInt8]) {
				prepare(signer: AuthAccount) {
				  let acct = AuthAccount(payer: signer)
				  acct.addPublicKey(keyA)
				}
			  }
			`,
			keyCount: 1,
			args:     []cadence.Value{keyA},
			expected: [][]byte{{1, 2, 3}},
		},
		{
			name: "Multiple keys",
			code: `
			  transaction(keys: [[UInt8]]) {
				prepare(signer: AuthAccount) {
				  let acct = AuthAccount(payer: signer)
				  for key in keys {
					acct.addPublicKey(key)
				  }
				}
			  }
			`,
			keyCount: 2,
			args:     []cadence.Value{keys},
			expected: [][]byte{{1, 2, 3}, {4, 5, 6}},
		},
	}

	for _, tt := range tests {

		var events []cadence.Event
		var keys [][]byte

		runtimeInterface := &testRuntimeInterface{
			storage: newTestStorage(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			createAccount: func(payer Address) (address Address, err error) {
				return Address{42}, nil
			},
			addAccountKey: func(address Address, publicKey []byte) error {
				keys = append(keys, publicKey)
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return jsoncdc.Decode(b)
			},
		}

		t.Run(tt.name, func(t *testing.T) {
			args := make([][]byte, len(tt.args))
			for i, arg := range tt.args {
				var err error
				args[i], err = jsoncdc.Encode(arg)
				if err != nil {
					panic(fmt.Errorf("broken test: invalid argument: %w", err))
				}
			}

			err := runtime.ExecuteTransaction(
				Script{
					Source:    []byte(tt.code),
					Arguments: args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  utils.TestLocation,
				},
			)
			require.NoError(t, err)
			assert.Len(t, events, tt.keyCount+1)
			assert.Len(t, keys, tt.keyCount)
			assert.Equal(t, tt.expected, keys)

			assert.EqualValues(t, stdlib.AccountCreatedEventType.ID(), events[0].Type().ID())

			for _, event := range events[1:] {
				assert.EqualValues(t, stdlib.AccountKeyAddedEventType.ID(), event.Type().ID())
			}
		})
	}
}

func TestRuntimeContractAccount(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

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

	deploy := utils.DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	t.Run("", func(t *testing.T) {
		value, err := runtime.ExecuteScript(
			Script{
				Source: script1,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, addressValue, value)
	})

	t.Run("", func(t *testing.T) {
		value, err := runtime.ExecuteScript(
			Script{
				Source: script2,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, addressValue, value)
	})
}

func TestRuntimeContractNestedResource(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
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

	deploy := utils.DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(addess Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
		log: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, `"Hello World!"`, loggedMessage)
}

func TestRuntimeStorageLoadedDestructionConcreteType(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        pub contract Test {
            pub resource R {
                // test that the destructor is linked back into the nested resource
                // after being loaded from storage
                destroy() {
                    log("destroyed")
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
                let r <- acct.load<@Test.R>(from: /storage/r)
				destroy r
			}
		}
	`)

	deploy := utils.DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(address Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error { return nil },
		log: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		})
	require.NoError(t, err)

	assert.Equal(t, `"destroyed"`, loggedMessage)
}

func TestRuntimeStorageLoadedDestructionAnyResource(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        pub contract Test {
            pub resource R {
                // test that the destructor is linked back into the nested resource
                // after being loaded from storage
                destroy() {
                    log("destroyed")
                }
            }

            init() {
                // store nested resource in account on deployment
                self.account.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        // NOTE: *not* importing concrete implementation.
        //   Should be imported automatically when loading the value from storage

		transaction {

			prepare(acct: AuthAccount) {
                let r <- acct.load<@AnyResource>(from: /storage/r)
				destroy r
			}
		}
	`)

	deploy := utils.DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(address Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error { return nil },
		log: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, `"destroyed"`, loggedMessage)
}

func TestRuntimeStorageLoadedDestructionAfterRemoval(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        pub contract Test {
            pub resource R {
                // test that the destructor is linked back into the nested resource
                // after being loaded from storage
                destroy() {
                    log("destroyed")
                }
            }

            init() {
                // store nested resource in account on deployment
                self.account.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        // NOTE: *not* importing concrete implementation.
        //   Should be imported automatically when loading the value from storage

		transaction {

			prepare(acct: AuthAccount) {
                let r <- acct.load<@AnyResource>(from: /storage/r)
				destroy r
			}
		}
	`)

	deploy := utils.DeploymentTransaction("Test", contract)
	removal := utils.RemovalTransaction("Test")

	var accountCode []byte

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(address Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		removeAccountContractCode: func(_ Address, _ string) (err error) {
			accountCode = nil
			return nil
		},
		emitEvent: func(event cadence.Event) error { return nil },
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy the contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	// Remove the contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: removal,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Nil(t, accountCode)

	// Destroy

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	var typeLoadingErr interpreter.TypeLoadingError
	utils.RequireErrorAs(t, err, &typeLoadingErr)

	require.Equal(t,
		common.AddressLocation{Address: addressValue}.TypeID("Test.R"),
		typeLoadingErr.TypeID,
	)
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

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := utils.DeploymentTransaction("FungibleToken", []byte(fungibleTokenContract))

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
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup1Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signerAccount = address2Value

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup2Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeFungibleTokenCreateAccount(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := []byte(fmt.Sprintf(
		`
          transaction {
            prepare(signer: AuthAccount) {
                let acct = AuthAccount(payer: signer)
                acct.contracts.add(name: "FungibleToken", code: "%s".decodeHex())
            }
          }
        `,
		hex.EncodeToString([]byte(fungibleTokenContract)),
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
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		createAccount: func(payer Address) (address Address, err error) {
			return address2Value, nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup1Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup2Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeInvokeStoredInterfaceFunction(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	makeDeployTransaction := func(name, code string) []byte {
		return []byte(fmt.Sprintf(
			`
              transaction {
                prepare(signer: AuthAccount) {
                  let acct = AuthAccount(payer: signer)
                  acct.contracts.add(name: "%s", code: "%s".decodeHex())
                }
              }
            `,
			name,
			hex.EncodeToString([]byte(code)),
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
	var events []cadence.Event

	var nextAccount byte = 0x2

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		createAccount: func(payer Address) (address Address, err error) {
			result := interpreter.NewAddressValueFromBytes([]byte{nextAccount})
			nextAccount++
			return result.ToAddress(), nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{0x1}}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	deployTransaction := makeDeployTransaction("TestContractInterface", contractInterfaceCode)
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

	deployTransaction = makeDeployTransaction("TestContract", contractCode)
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

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupCode,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	for a := 1; a <= 3; a++ {
		for b := 1; b <= 3; b++ {

			t.Run(fmt.Sprintf("%d/%d", a, b), func(t *testing.T) {

				err = runtime.ExecuteTransaction(
					Script{
						Source: makeUseCode(a, b),
					},
					Context{
						Interface: runtimeInterface,
						Location:  nextTransactionLocation(),
					},
				)

				if a == 2 && b == 2 {
					assert.NoError(t, err)
				} else {
					require.Error(t, err)

					utils.RequireErrorAs(t, err, &interpreter.ConditionError{})
				}
			})
		}
	}
}

func TestRuntimeBlock(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let block = getCurrentBlock()
          log(block)
          log(block.height)
          log(block.view)
          log(block.id)
          log(block.timestamp)

          let nextBlock = getBlock(at: block.height + UInt64(1))
          log(nextBlock)
          log(nextBlock?.height)
          log(nextBlock?.view)
          log(nextBlock?.id)
          log(nextBlock?.timestamp)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"Block(height: 1, view: 1, id: 0x0000000000000000000000000000000000000000000000000000000000000001, timestamp: 1.00000000)",
			"1",
			"1",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1]",
			"1.00000000",
			"Block(height: 2, view: 2, id: 0x0000000000000000000000000000000000000000000000000000000000000002, timestamp: 2.00000000)",
			"2",
			"2",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2]",
			"2.00000000",
		},
		loggedMessages,
	)
}

func TestUnsafeRandom(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let rand = unsafeRandom()
          log(rand)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		unsafeRandom: func() (uint64, error) {
			return 7558174677681708339, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"7558174677681708339",
		},
		loggedMessages,
	)
}

func TestRuntimeTransactionTopLevelDeclarations(t *testing.T) {

	t.Parallel()

	t.Run("transaction with function", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`
          pub fun test() {}

          transaction {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})

	t.Run("transaction with resource", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`
          pub resource R {}

          transaction {}
        `)

		runtimeInterface := &testRuntimeInterface{
			getSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.Error(t, err)

		var checkerErr *sema.CheckerError
		utils.RequireErrorAs(t, err, &checkerErr)

		errs := checker.ExpectCheckerErrors(t, checkerErr, 1)

		assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
	})
}

func TestRuntimeStoreIntegerTypes(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := interpreter.AddressValue{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xCA, 0xDE,
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

			deploy := utils.DeploymentTransaction("Test", contract)

			var accountCode []byte
			var events []cadence.Event

			runtimeInterface := &testRuntimeInterface{
				getCode: func(_ Location) (bytes []byte, err error) {
					return accountCode, nil
				},
				storage: newTestStorage(nil, nil),
				getSigningAccounts: func() ([]Address, error) {
					return []Address{addressValue.ToAddress()}, nil
				},
				getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
					return accountCode, nil
				},
				updateAccountContractCode: func(_ Address, _ string, code []byte) error {
					accountCode = code
					return nil
				},
				emitEvent: func(event cadence.Event) error {
					events = append(events, event)
					return nil
				},
			}

			nextTransactionLocation := newTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: deploy,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			assert.NotNil(t, accountCode)
		})
	}
}

func TestInterpretResourceOwnerFieldUseComposite(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
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

	deploy := utils.DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {

              let r <- Test.createR()
              log(r.owner?.address)
              r.logOwnerAddress()

              signer.save(<-r, to: /storage/r)
              signer.link<&Test.R>(/public/r, target: /storage/r)

              let ref1 = signer.borrow<&Test.R>(from: /storage/r)!
              log(ref1.owner?.address)
              ref1.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/r).borrow<&Test.R>()!
              log(ref2.owner?.address)
              ref2.logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {
              let ref1 = signer.borrow<&Test.R>(from: /storage/r)!
              log(ref1.owner?.address)
              ref1.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/r).borrow<&Test.R>()!
              log(ref2.owner?.address)
              ref2.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)
}

func TestInterpretResourceOwnerFieldUseArray(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
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

	deploy := utils.DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {

              let rs <- [
                  <-Test.createR(),
                  <-Test.createR()
              ]
              log(rs[0].owner?.address)
              log(rs[1].owner?.address)
              rs[0].logOwnerAddress()
              rs[1].logOwnerAddress()

              signer.save(<-rs, to: /storage/rs)
              signer.link<&[Test.R]>(/public/rs, target: /storage/rs)

              let ref1 = signer.borrow<&[Test.R]>(from: /storage/rs)!
              log(ref1[0].owner?.address)
              log(ref1[1].owner?.address)
              ref1[0].logOwnerAddress()
              ref1[1].logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/rs).borrow<&[Test.R]>()!
              log(ref2[0].owner?.address)
              log(ref2[1].owner?.address)
              ref2[0].logOwnerAddress()
              ref2[1].logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {
              let ref1 = signer.borrow<&[Test.R]>(from: /storage/rs)!
              log(ref1[0].owner?.address)
              log(ref1[1].owner?.address)
              ref1[0].logOwnerAddress()
              ref1[1].logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/rs).borrow<&[Test.R]>()!
              log(ref2[0].owner?.address)
              log(ref2[1].owner?.address)
              ref2[0].logOwnerAddress()
              ref2[1].logOwnerAddress()
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"nil", "nil",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)
}

func TestInterpretResourceOwnerFieldUseDictionary(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
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

	deploy := utils.DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {

              let rs <- {
                  "a": <-Test.createR(),
                  "b": <-Test.createR()
              }
              log(rs["a"]?.owner?.address)
              log(rs["b"]?.owner?.address)
              rs["a"]?.logOwnerAddress()
              rs["b"]?.logOwnerAddress()

              signer.save(<-rs, to: /storage/rs)
              signer.link<&{String: Test.R}>(/public/rs, target: /storage/rs)

              let ref1 = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
              log(ref1["a"]?.owner?.address)
              log(ref1["b"]?.owner?.address)
              ref1["a"]?.logOwnerAddress()
              ref1["b"]?.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/rs).borrow<&{String: Test.R}>()!
              log(ref2["a"]?.owner?.address)
              log(ref2["b"]?.owner?.address)
              ref2["a"]?.logOwnerAddress()
              ref2["b"]?.logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {
              let ref1 = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
              log(ref1["a"]?.owner?.address)
              log(ref1["b"]?.owner?.address)
              ref1["a"]?.logOwnerAddress()
              ref1["b"]?.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/rs).borrow<&{String: Test.R}>()!
              log(ref2["a"]?.owner?.address)
              log(ref2["b"]?.owner?.address)
              ref2["a"]?.logOwnerAddress()
              ref2["b"]?.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[string][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"nil", "nil",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
			"0x1", "0x1",
		},
		loggedMessages,
	)
}

func TestRuntimeComputationLimit(t *testing.T) {

	t.Parallel()

	const computationLimit = 5

	type test struct {
		name string
		code string
		ok   bool
	}

	tests := []test{
		{
			name: "Infinite while loop",
			code: `
              while true {}
            `,
			ok: false,
		},
		{
			name: "Limited while loop",
			code: `
              var i = 0
              while i < 5 {
                  i = i + 1
              }
            `,
			ok: false,
		},
		{
			name: "Too many for-in loop iterations",
			code: `
              for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {}
            `,
			ok: false,
		},
		{
			name: "Some for-in loop iterations",
			code: `
              for i in [1, 2, 3, 4] {}
            `,
			ok: true,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			script := []byte(
				fmt.Sprintf(
					`
                      transaction {
                          prepare() {
                              %s
                          }
                      }
                    `,
					test.code,
				),
			)

			runtime := NewInterpreterRuntime()

			runtimeInterface := &testRuntimeInterface{
				getSigningAccounts: func() ([]Address, error) {
					return nil, nil
				},
				computationLimit: computationLimit,
			}

			nextTransactionLocation := newTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			if test.ok {
				require.NoError(t, err)
			} else {
				var computationLimitErr ComputationLimitExceededError
				utils.RequireErrorAs(t, err, &computationLimitErr)

				assert.Equal(t,
					ComputationLimitExceededError{
						Limit: computationLimit,
					},
					computationLimitErr,
				)
			}
		})
	}
}

func TestRuntimeMetrics(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	imported1Location := common.StringLocation("imported1")

	importedScript1 := []byte(`
      pub fun generate(): [Int] {
        return [1, 2, 3]
      }
    `)

	imported2Location := common.StringLocation("imported2")

	importedScript2 := []byte(`
      pub fun getPath(): StoragePath {
        return /storage/foo
      }
    `)

	script1 := []byte(`
      import "imported1"

      transaction {
          prepare(signer: AuthAccount) {
              signer.save(generate(), to: /storage/foo)
          }
          execute {}
      }
    `)

	script2 := []byte(`
      import "imported2"

      transaction {
          prepare(signer: AuthAccount) {
              signer.load<[Int]>(from: getPath())
          }
          execute {}
      }
    `)

	storage := newTestStorage(nil, nil)

	type reports struct {
		programParsed      map[common.LocationID]int
		programChecked     map[common.LocationID]int
		programInterpreted map[common.LocationID]int
		valueEncoded       int
		valueDecoded       int
	}

	newRuntimeInterface := func() (runtimeInterface Interface, r *reports) {

		r = &reports{
			programParsed:      map[common.LocationID]int{},
			programChecked:     map[common.LocationID]int{},
			programInterpreted: map[common.LocationID]int{},
		}

		runtimeInterface = &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			getCode: func(location Location) (bytes []byte, err error) {
				switch location {
				case imported1Location:
					return importedScript1, nil
				case imported2Location:
					return importedScript2, nil
				default:
					return nil, fmt.Errorf("unknown import location: %s", location)
				}
			},
			programParsed: func(location common.Location, duration time.Duration) {
				r.programParsed[location.ID()]++
			},
			programChecked: func(location common.Location, duration time.Duration) {
				r.programChecked[location.ID()]++
			},
			programInterpreted: func(location common.Location, duration time.Duration) {
				r.programInterpreted[location.ID()]++
			},
			valueEncoded: func(duration time.Duration) {
				r.valueEncoded++
			},
			valueDecoded: func(duration time.Duration) {
				r.valueDecoded++
			},
		}

		return
	}

	i1, r1 := newRuntimeInterface()

	nextTransactionLocation := newTransactionLocationGenerator()

	transactionLocation := nextTransactionLocation()
	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: i1,
			Location:  transactionLocation,
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
			imported1Location.ID():   1,
		},
		r1.programParsed,
	)
	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
			imported1Location.ID():   1,
		},
		r1.programChecked,
	)
	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
		},
		r1.programInterpreted,
	)
	assert.Equal(t, 1, r1.valueEncoded)
	assert.Equal(t, 0, r1.valueDecoded)

	i2, r2 := newRuntimeInterface()

	transactionLocation = nextTransactionLocation()

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: i2,
			Location:  transactionLocation,
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
			imported2Location.ID():   1,
		},
		r2.programParsed,
	)
	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
			imported2Location.ID():   1,
		},
		r2.programChecked,
	)
	assert.Equal(t,
		map[common.LocationID]int{
			transactionLocation.ID(): 1,
		},
		r2.programInterpreted,
	)
	assert.Equal(t, 0, r2.valueEncoded)
	assert.Equal(t, 1, r2.valueDecoded)
}

type testRead struct {
	owner, key []byte
}

func (r testRead) String() string {
	return string(r.key)
}

type testWrite struct {
	owner, key, value []byte
}

func (w testWrite) String() string {
	return string(w.key)
}

func TestRuntimeContractWriteback(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      pub contract Test {

          pub(set) var test: Int

          init() {
              self.test = 1
          }
      }
    `)

	deploy := utils.DeploymentTransaction("Test", contract)

	readTx := []byte(`
      import Test from 0xCADE

       transaction {

          prepare(signer: AuthAccount) {
              log(Test.test)
          }
       }
    `)

	writeTx := []byte(`
      import Test from 0xCADE

       transaction {

          prepare(signer: AuthAccount) {
              Test.test = 2
          }
       }
    `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
			value,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	assert.Len(t, writes, 1)

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Len(t, writes, 1)

	err = runtime.ExecuteTransaction(
		Script{
			Source: writeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Len(t, writes, 2)
}

func TestRuntimeStorageWriteback(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      pub contract Test {

          pub resource R {

              pub(set) var test: Int

              init() {
                  self.test = 1
              }
          }


          pub fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
      import Test from 0xCADE

       transaction {

          prepare(signer: AuthAccount) {
              signer.save(<-Test.createR(), to: /storage/r)
          }
       }
    `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
			value,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	assert.Len(t, writes, 1)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Len(t, writes, 2)

	readTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: AuthAccount) {
             log(signer.borrow<&Test.R>(from: /storage/r)!.test)
         }
      }
    `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Len(t, writes, 2)

	writeTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: AuthAccount) {
             let r = signer.borrow<&Test.R>(from: /storage/r)!
             r.test = 2
         }
      }
    `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: writeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Len(t, writes, 3)
}

func TestRuntimeExternalError(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log("ok")
        }
      }
    `)

	type logPanic struct{}

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		log: func(message string) {
			panic(logPanic{})
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	assert.PanicsWithValue(t,
		interpreter.ExternalError{
			Recovered: logPanic{},
		},
		func() {
			_ = runtime.ExecuteTransaction(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
		},
	)
}

func TestRuntimeDeployCodeCaching(t *testing.T) {

	t.Parallel()

	const helloWorldContract = `
      pub contract HelloWorld {

          pub let greeting: String

          init() {
              self.greeting = "Hello, World!"
          }

          pub fun hello(): String {
              return self.greeting
          }
      }
    `

	const callHelloTxTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: AuthAccount) {
                assert(HelloWorld.hello() == "Hello, World!")
            }
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: AuthAccount) {
                AuthAccount(payer: signer)
            }
        }
    `)

	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

	runtime := NewInterpreterRuntime()

	accountCodes := map[string][]byte{}
	var events []cadence.Event

	var accountCounter uint8 = 0

	var signerAddresses []Address

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// create the account

	signerAddresses = []Address{{accountCounter}}

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// deploy the contract

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// call the hello function

	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeUpdateCodeCaching(t *testing.T) {

	t.Parallel()

	const helloWorldContract1 = `
      pub contract HelloWorld {

          pub fun hello(): String {
              return "1"
          }
      }
    `

	const helloWorldContract2 = `
      pub contract HelloWorld {

          pub fun hello(): String {
              return "2"
          }
      }
    `

	const callHelloScriptTemplate = `
        import HelloWorld from 0x%s

        pub fun main(): String {
            return HelloWorld.hello()
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: AuthAccount) {
                AuthAccount(payer: signer)
            }
        }
    `)

	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract1))
	updateTx := utils.UpdateTransaction("HelloWorld", []byte(helloWorldContract2))

	runtime := NewInterpreterRuntime()

	accountCodes := map[string][]byte{}
	var events []cadence.Event

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programHits []string

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// create the account

	signerAddresses = []Address{{accountCounter}}

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// deploy the contract

	programHits = nil

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Empty(t, programHits)

	// call the initial hello function

	callScript := []byte(fmt.Sprintf(callHelloScriptTemplate, Address{accountCounter}))

	result1, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, cadence.NewString("1"), result1)

	// update the contract

	programHits = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: updateTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Empty(t, programHits)

	// call the new hello function

	result2, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, cadence.NewString("2"), result2)
}

func TestRuntimeNoProgramsHitForToplevelPrograms(t *testing.T) {

	// We do not want to hit the stored programs for toplevel programs
	// (scripts and transactions) until we have moved the caching layer to Cadence.

	t.Parallel()

	const helloWorldContract = `
      pub contract HelloWorld {

          pub let greeting: String

          init() {
              self.greeting = "Hello, World!"
          }

          pub fun hello(): String {
              return self.greeting
          }
      }
    `

	const callHelloTxTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: AuthAccount) {
                assert(HelloWorld.hello() == "Hello, World!")
            }
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: AuthAccount) {
                AuthAccount(payer: signer)
            }
        }
    `)

	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

	runtime := NewInterpreterRuntime()

	accountCodes := map[string][]byte{}
	var events []cadence.Event

	programs := map[common.LocationID]*interpreter.Program{}

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programsHits []string

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		setProgram: func(location Location, program *interpreter.Program) error {
			programs[location.ID()] = program
			return nil
		},
		getProgram: func(location Location) (*interpreter.Program, error) {
			programsHits = append(programsHits, string(location.ID()))
			return programs[location.ID()], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	signerAddresses = []Address{{accountCounter}}

	// create the account

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// call the function

	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// We should only receive a cache hit for the imported program, not the transactions/scripts.

	require.GreaterOrEqual(t, len(programsHits), 1)

	for _, cacheHit := range programsHits {
		require.Equal(t, "A.0100000000000000.HelloWorld", cacheHit)
	}
}

func TestRuntimeTransaction_ContractUpdate(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	const contract1 = `
      pub contract Test {

          pub resource R {

              pub let name: String

              init(name: String) {
                  self.name = name
              }

              pub fun hello(): Int {
                  return 1
              }
          }

          pub var rs: @{String: R}

          pub fun hello(): Int {
              return 1
          }

          init() {
              self.rs <- {}
              self.rs["r1"] <-! create R(name: "1")
          }
      }
    `

	const contract2 = `
      pub contract Test {

          pub resource R {

              pub let name: String

              init(name: String) {
                  self.name = name
              }

              pub fun hello(): Int {
                  return 2
              }
          }

          pub var rs: @{String: R}

          pub fun hello(): Int {
              return 2
          }

          init() {
              self.rs <- {}
              panic("should never be executed")
          }
      }
    `

	newDeployTransaction := func(function, name, code string) []byte {
		return []byte(fmt.Sprintf(
			`
              transaction {

                  prepare(signer: AuthAccount) {
                      signer.contracts.%s(name: "%s", code: "%s".decodeHex())
                  }
              }
            `,
			function,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress([]byte{0x42})}, nil
		},
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		resolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
			require.Empty(t, identifiers)
			require.IsType(t, common.AddressLocation{}, location)

			return []ResolvedLocation{
				{
					Location: common.AddressLocation{
						Address: location.(common.AddressLocation).Address,
						Name:    "Test",
					},
					Identifiers: []ast.Identifier{
						{
							Identifier: "Test",
						},
					},
				},
			}, nil
		},
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	deployTx1 := newDeployTransaction("add", "Test", contract1)

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

	script1 := []byte(`
      import 0x42

      pub fun main() {
          // Check stored data

          assert(Test.rs.length == 1)
          assert(Test.rs["r1"]?.name == "1")

          // Check functions

          assert(Test.rs["r1"]?.hello() == 1)
          assert(Test.hello() == 1)
      }
    `)

	_, err = runtime.ExecuteScript(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	deployTx2 := newDeployTransaction("update__experimental", "Test", contract2)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	script2 := []byte(`
      import 0x42

      pub fun main() {
          // Existing data is still available and the same as before

          assert(Test.rs.length == 1)
          assert(Test.rs["r1"]?.name == "1")

          // New function code is executed.
          // Compare with script1 above, which checked 1.

          assert(Test.rs["r1"]?.hello() == 2)
          assert(Test.hello() == 2)
      }
    `)

	_, err = runtime.ExecuteScript(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntime(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{
		decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(b)
		},
	}

	script := []byte(`
      pub fun main(num: Int) {}
    `)

	type testCase struct {
		name      string
		arguments [][]byte
		valid     bool
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {

			// NOTE: to parallelize this sub-test,
			// access to `programs` must be made thread-safe first

			_, err := runtime.ExecuteScript(
				Script{
					Source:    script,
					Arguments: tc.arguments,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{0x1},
				},
			)

			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)

				utils.RequireErrorAs(t, err, &InvalidEntryPointParameterCountError{})
			}
		})
	}

	for _, testCase := range []testCase{
		{
			name:      "too few arguments",
			arguments: [][]byte{},
			valid:     false,
		},
		{
			name: "correct number of arguments",
			arguments: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(1)),
			},
			valid: true,
		},
		{
			name: "too many arguments",
			arguments: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(1)),
				jsoncdc.MustEncode(cadence.NewInt(2)),
			},
			valid: false,
		},
	} {
		test(testCase)
	}
}

func singleIdentifierLocationResolver(t *testing.T) func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
	return func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
		require.Len(t, identifiers, 1)
		require.IsType(t, common.AddressLocation{}, location)

		return []ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: location.(common.AddressLocation).Address,
					Name:    identifiers[0].Identifier,
				},
				Identifiers: identifiers,
			},
		}, nil
	}
}

func TestPanics(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      pub fun main() {
		[1][1]
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	assert.Error(t, err)
}

func TestContractUpdateValidation(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

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

	accountCode := map[string][]byte{}
	var events []cadence.Event
	runtimeInterface := getMockedRuntimeInterfaceForTxUpdate(t, accountCode, events)
	nextTransactionLocation := newTransactionLocationGenerator()

	deployAndUpdate := func(name string, oldCode string, newCode string) error {
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

		err := deployAndUpdate("Test1", oldCode, newCode)
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

		err := deployAndUpdate("Test2", oldCode, newCode)
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

		err := deployAndUpdate("Test3", oldCode, newCode)
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

		err := deployAndUpdate("Test4", oldCode, newCode)
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

		err := deployAndUpdate("Test5", oldCode, newCode)
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

		err := deployAndUpdate("Test6", oldCode, newCode)
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

		err := deployAndUpdate("Test7", oldCode, newCode)
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

		err := deployAndUpdate("Test8", oldCode, newCode)
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

		deployTx1 := newDeployTransaction("add", "Test9Import", importCode)
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

		err = deployAndUpdate("Test9", oldCode, newCode)
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

		err := deployAndUpdate("Test10", oldCode, newCode)
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

		err := deployAndUpdate("Test11", oldCode, newCode)
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

		err := deployAndUpdate("Test12", oldCode, newCode)
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

		err := deployAndUpdate("Test13", oldCode, newCode)

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

		err := deployAndUpdate("Test14", oldCode, newCode)
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

		err := deployAndUpdate("Test15", oldCode, newCode)
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

		err := deployAndUpdate("Test16", oldCode, newCode)
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

		err := deployAndUpdate("Test17", oldCode, newCode)
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

		err := deployAndUpdate("Test18", oldCode, newCode)
		require.NoError(t, err)
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

		err := deployAndUpdate("Test19", oldCode, newCode)
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

		err := deployAndUpdate("Test20", oldCode, newCode)
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

		err := deployAndUpdate("Test21", oldCode, newCode)
		require.Error(t, err)

		const expectedError = "error: mismatching field `a` in `Test21`\n" +
			" --> 0000000000000042.Test21:3:16\n" +
			"  |\n" +
			"3 |                 pub var a: Int\n" +
			"  |                 ^^^^^^^^^^^^^^ incompatible type annotations. expected `String`, found `Int`\n" +
			"\n" +
			"error: found new field `b` in `Test21`\n" +
			" --> 0000000000000042.Test21:4:24\n" +
			"  |\n" +
			"4 |                 pub var b: String\n" +
			"  |                         ^\n" +
			"\n" +
			"error: trying to convert structure interface `TestStruct` to a structure\n" +
			"  --> 0000000000000042.Test21:11:16\n" +
			"   |\n" +
			"11 |                 pub struct TestStruct {\n" +
			"   |                 ^"

		require.Contains(t, err.Error(), expectedError)
	})
}

func assertDeclTypeChangeError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	oldKind common.DeclarationKind,
	newKind common.DeclarationKind,
) {

	require.NotNil(t, err)
	require.IsType(t, &InvalidDeclarationKindChangeError{}, err)
	declTypeChangeError := err.(*InvalidDeclarationKindChangeError)
	assert.Equal(
		t,
		fmt.Sprintf("trying to convert %s `%s` to a %s", oldKind.Name(), erroneousDeclName, newKind.Name()),
		declTypeChangeError.Error(),
	)
}

func assertExtraneousFieldError(t *testing.T, err error, erroneousDeclName string, fieldName string) {
	require.NotNil(t, err)
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

	assert.IsType(t, &TypeMismatchError{}, fieldMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		fieldMismatchError.err.Error(),
	)
}

func assertConformanceMismatchError(
	t *testing.T,
	err error,
	erroneousDeclName string,
	expectedType string,
	foundType string,
) {

	require.NotNil(t, err)
	require.IsType(t, &ConformanceMismatchError{}, err)
	conformanceMismatchError := err.(*ConformanceMismatchError)
	assert.Equal(
		t,
		fmt.Sprintf("conformances does not match in `%s`", erroneousDeclName),
		conformanceMismatchError.Error(),
	)

	assert.IsType(t, &TypeMismatchError{}, conformanceMismatchError.err)
	assert.Equal(
		t,
		fmt.Sprintf("incompatible type annotations. expected `%s`, found `%s`", expectedType, foundType),
		conformanceMismatchError.err.Error(),
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
	require.NotNil(t, err)
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
	accountStorage map[string][]byte,
	events []cadence.Event) *testRuntimeInterface {

	return &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountStorage[key], nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.BytesToAddress([]byte{0x42})}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			return accountStorage[key], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			key := string(location.ID())
			accountStorage[key] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}
}
