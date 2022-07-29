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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onflow/atree"
	"go.opentelemetry.io/otel/attribute"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testLedger struct {
	storedValues         map[string][]byte
	valueExists          func(owner, key []byte) (exists bool, err error)
	getValue             func(owner, key []byte) (value []byte, err error)
	setValue             func(owner, key, value []byte) (err error)
	allocateStorageIndex func(owner []byte) (atree.StorageIndex, error)
}

var _ atree.Ledger = testLedger{}

func (s testLedger) GetValue(owner, key []byte) (value []byte, err error) {
	return s.getValue(owner, key)
}

func (s testLedger) SetValue(owner, key, value []byte) (err error) {
	return s.setValue(owner, key, value)
}

func (s testLedger) ValueExists(owner, key []byte) (exists bool, err error) {
	return s.valueExists(owner, key)
}

func (s testLedger) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	return s.allocateStorageIndex(owner)
}

func (s testLedger) Dump() {
	for key, data := range s.storedValues {
		fmt.Printf("%s:\n", strconv.Quote(key))
		fmt.Printf("%s\n", hex.Dump(data))
		println()
	}
}

func newTestLedger(
	onRead func(owner, key, value []byte),
	onWrite func(owner, key, value []byte),
) testLedger {

	storageKey := func(owner, key string) string {
		return strings.Join([]string{owner, key}, "|")
	}

	storedValues := map[string][]byte{}

	storageIndices := map[string]uint64{}

	storage := testLedger{
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
		allocateStorageIndex: func(owner []byte) (result atree.StorageIndex, err error) {
			index := storageIndices[string(owner)] + 1
			storageIndices[string(owner)] = index
			binary.BigEndian.PutUint64(result[:], index)
			return
		},
	}

	return storage
}

func newTestInterpreterRuntime(options ...Option) Runtime {
	rt := NewInterpreterRuntime(options...)
	rt.SetAtreeValidationEnabled(true)
	return rt
}

type testRuntimeInterface struct {
	resolveLocation           func(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
	getCode                   func(_ Location) ([]byte, error)
	getProgram                func(Location) (*interpreter.Program, error)
	setProgram                func(Location, *interpreter.Program) error
	storage                   testLedger
	createAccount             func(payer Address) (address Address, err error)
	addEncodedAccountKey      func(address Address, publicKey []byte) error
	removeEncodedAccountKey   func(address Address, index int) (publicKey []byte, err error)
	addAccountKey             func(address Address, publicKey *PublicKey, hashAlgo HashAlgorithm, weight int) (*AccountKey, error)
	getAccountKey             func(address Address, index int) (*AccountKey, error)
	removeAccountKey          func(address Address, index int) (*AccountKey, error)
	updateAccountContractCode func(address Address, name string, code []byte) error
	getAccountContractCode    func(address Address, name string) (code []byte, err error)
	removeAccountContractCode func(address Address, name string) (err error)
	getSigningAccounts        func() ([]Address, error)
	log                       func(string)
	emitEvent                 func(cadence.Event) error
	resourceOwnerChanged      func(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldAddress common.Address,
		newAddress common.Address,
	)
	generateUUID       func() (uint64, error)
	meterComputation   func(compKind common.ComputationKind, intensity uint) error
	decodeArgument     func(b []byte, t cadence.Type) (cadence.Value, error)
	programParsed      func(location common.Location, duration time.Duration)
	programChecked     func(location common.Location, duration time.Duration)
	programInterpreted func(location common.Location, duration time.Duration)
	unsafeRandom       func() (uint64, error)
	verifySignature    func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm SignatureAlgorithm,
		hashAlgorithm HashAlgorithm,
	) (bool, error)
	hash                       func(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error)
	setCadenceValue            func(owner Address, key string, value cadence.Value) (err error)
	getAccountBalance          func(_ Address) (uint64, error)
	getAccountAvailableBalance func(_ Address) (uint64, error)
	getStorageUsed             func(_ Address) (uint64, error)
	getStorageCapacity         func(_ Address) (uint64, error)
	programs                   map[common.Location]*interpreter.Program
	implementationDebugLog     func(message string) error
	validatePublicKey          func(publicKey *PublicKey) error
	bLSVerifyPOP               func(pk *PublicKey, s []byte) (bool, error)
	blsAggregateSignatures     func(sigs [][]byte) ([]byte, error)
	blsAggregatePublicKeys     func(keys []*PublicKey) (*PublicKey, error)
	getAccountContractNames    func(address Address) ([]string, error)
	recordTrace                func(operation string, location common.Location, duration time.Duration, attrs []attribute.KeyValue)
	meterMemory                func(usage common.MemoryUsage) error
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
			i.programs = map[common.Location]*interpreter.Program{}
		}
		return i.programs[location], nil
	}

	return i.getProgram(location)
}

func (i *testRuntimeInterface) SetProgram(location Location, program *interpreter.Program) error {
	if i.setProgram == nil {
		if i.programs == nil {
			i.programs = map[common.Location]*interpreter.Program{}
		}
		i.programs[location] = program
		return nil
	}

	return i.setProgram(location, program)
}

func (i *testRuntimeInterface) ValueExists(owner, key []byte) (exists bool, err error) {
	if i.storage.valueExists == nil {
		panic("must specify testRuntimeInterface.storage.valueExists")
	}
	return i.storage.ValueExists(owner, key)
}

func (i *testRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	if i.storage.getValue == nil {
		panic("must specify testRuntimeInterface.storage.getValue")
	}
	return i.storage.GetValue(owner, key)
}

func (i *testRuntimeInterface) SetValue(owner, key, value []byte) (err error) {
	if i.storage.setValue == nil {
		panic("must specify testRuntimeInterface.storage.setValue")
	}
	return i.storage.SetValue(owner, key, value)
}

func (i *testRuntimeInterface) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	if i.storage.allocateStorageIndex == nil {
		panic("must specify testRuntimeInterface.storage.allocateStorageIndex")
	}
	return i.storage.AllocateStorageIndex(owner)
}

func (i *testRuntimeInterface) CreateAccount(payer Address) (address Address, err error) {
	if i.createAccount == nil {
		panic("must specify testRuntimeInterface.createAccount")
	}
	return i.createAccount(payer)
}

func (i *testRuntimeInterface) AddEncodedAccountKey(address Address, publicKey []byte) error {
	if i.addEncodedAccountKey == nil {
		panic("must specify testRuntimeInterface.addEncodedAccountKey")
	}
	return i.addEncodedAccountKey(address, publicKey)
}

func (i *testRuntimeInterface) RevokeEncodedAccountKey(address Address, index int) ([]byte, error) {
	if i.removeEncodedAccountKey == nil {
		panic("must specify testRuntimeInterface.removeEncodedAccountKey")
	}
	return i.removeEncodedAccountKey(address, index)
}

func (i *testRuntimeInterface) AddAccountKey(address Address, publicKey *PublicKey, hashAlgo HashAlgorithm, weight int) (*AccountKey, error) {
	if i.addAccountKey == nil {
		panic("must specify testRuntimeInterface.addAccountKey")
	}
	return i.addAccountKey(address, publicKey, hashAlgo, weight)
}

func (i *testRuntimeInterface) GetAccountKey(address Address, index int) (*AccountKey, error) {
	if i.getAccountKey == nil {
		panic("must specify testRuntimeInterface.getAccountKey")
	}
	return i.getAccountKey(address, index)
}

func (i *testRuntimeInterface) RevokeAccountKey(address Address, index int) (*AccountKey, error) {
	if i.removeAccountKey == nil {
		panic("must specify testRuntimeInterface.removeAccountKey")
	}
	return i.removeAccountKey(address, index)
}

func (i *testRuntimeInterface) UpdateAccountContractCode(address Address, name string, code []byte) (err error) {
	if i.updateAccountContractCode == nil {
		panic("must specify testRuntimeInterface.updateAccountContractCode")
	}
	return i.updateAccountContractCode(address, name, code)
}

func (i *testRuntimeInterface) GetAccountContractCode(address Address, name string) (code []byte, err error) {
	if i.getAccountContractCode == nil {
		panic("must specify testRuntimeInterface.getAccountContractCode")
	}
	return i.getAccountContractCode(address, name)
}

func (i *testRuntimeInterface) RemoveAccountContractCode(address Address, name string) (err error) {
	if i.removeAccountContractCode == nil {
		panic("must specify testRuntimeInterface.removeAccountContractCode")
	}
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

func (i *testRuntimeInterface) ResourceOwnerChanged(
	interpreter *interpreter.Interpreter,
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	if i.resourceOwnerChanged != nil {
		i.resourceOwnerChanged(
			interpreter,
			resource,
			oldOwner,
			newOwner,
		)
	}
}

func (i *testRuntimeInterface) GenerateUUID() (uint64, error) {
	if i.generateUUID == nil {
		return 0, nil
	}
	return i.generateUUID()
}

func (i *testRuntimeInterface) MeterComputation(compKind common.ComputationKind, intensity uint) error {
	if i.meterComputation == nil {
		return nil
	}
	return i.meterComputation(compKind, intensity)
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
	signatureAlgorithm SignatureAlgorithm,
	hashAlgorithm HashAlgorithm,
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

func (i *testRuntimeInterface) Hash(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error) {
	if i.hash == nil {
		return nil, nil
	}
	return i.hash(data, tag, hashAlgorithm)
}

func (i *testRuntimeInterface) SetCadenceValue(owner common.Address, key string, value cadence.Value) (err error) {
	if i.setCadenceValue == nil {
		panic("must specify testRuntimeInterface.setCadenceValue")
	}
	return i.setCadenceValue(owner, key, value)
}

func (i *testRuntimeInterface) GetAccountBalance(address Address) (uint64, error) {
	if i.getAccountBalance == nil {
		panic("must specify testRuntimeInterface.getAccountBalance")
	}
	return i.getAccountBalance(address)
}

func (i *testRuntimeInterface) GetAccountAvailableBalance(address Address) (uint64, error) {
	if i.getAccountAvailableBalance == nil {
		panic("must specify testRuntimeInterface.getAccountAvailableBalance")
	}
	return i.getAccountAvailableBalance(address)
}

func (i *testRuntimeInterface) GetStorageUsed(address Address) (uint64, error) {
	if i.getStorageUsed == nil {
		panic("must specify testRuntimeInterface.getStorageUsed")
	}
	return i.getStorageUsed(address)
}

func (i *testRuntimeInterface) GetStorageCapacity(address Address) (uint64, error) {
	if i.getStorageCapacity == nil {
		panic("must specify testRuntimeInterface.getStorageCapacity")
	}
	return i.getStorageCapacity(address)
}

func (i *testRuntimeInterface) ImplementationDebugLog(message string) error {
	if i.implementationDebugLog == nil {
		return nil
	}
	return i.implementationDebugLog(message)
}

func (i *testRuntimeInterface) ValidatePublicKey(key *PublicKey) error {
	if i.validatePublicKey == nil {
		return errors.New("mock defaults to public key validation failure")
	}

	return i.validatePublicKey(key)
}

func (i *testRuntimeInterface) BLSVerifyPOP(key *PublicKey, s []byte) (bool, error) {
	if i.bLSVerifyPOP == nil {
		return false, nil
	}

	return i.bLSVerifyPOP(key, s)
}

func (i *testRuntimeInterface) BLSAggregateSignatures(sigs [][]byte) ([]byte, error) {
	if i.blsAggregateSignatures == nil {
		return []byte{}, nil
	}

	return i.blsAggregateSignatures(sigs)
}

func (i *testRuntimeInterface) BLSAggregatePublicKeys(keys []*PublicKey) (*PublicKey, error) {
	if i.blsAggregatePublicKeys == nil {
		return nil, nil
	}

	return i.blsAggregatePublicKeys(keys)
}

func (i *testRuntimeInterface) GetAccountContractNames(address Address) ([]string, error) {
	if i.getAccountContractNames == nil {
		return []string{}, nil
	}

	return i.getAccountContractNames(address)
}

func (i *testRuntimeInterface) RecordTrace(operation string, location common.Location, duration time.Duration, attrs []attribute.KeyValue) {
	if i.recordTrace == nil {
		return
	}
	i.recordTrace(operation, location, duration, attrs)
}

func (i *testRuntimeInterface) MeterMemory(usage common.MemoryUsage) error {
	if i.meterMemory == nil {
		return nil
	}

	return i.meterMemory(usage)
}

func TestRuntimeImport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	runtime := newTestInterpreterRuntime()

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
	programs := map[common.Location]*interpreter.Program{}

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

			programs[location] = program

			return nil
		},
		getProgram: func(location Location) (*interpreter.Program, error) {
			programsLock.RLock()
			defer programsLock.RUnlock()

			program := programs[location]

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

	programs := map[common.Location]*interpreter.Program{}
	programsHits := make(map[common.Location]bool)

	importedScript := []byte(`
      transaction {
          prepare() {}
          execute {}
      }
    `)
	importedScriptLocation := common.StringLocation("imported")

	runtime := newTestInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{
		getProgram: func(location common.Location) (*interpreter.Program, error) {
			program, found := programs[location]
			programsHits[location] = found
			if !found {
				return nil, nil
			}
			return program, nil
		},
		setProgram: func(location common.Location, program *interpreter.Program) error {
			programs[location] = program
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
		storedProgram, exists := programs[scriptLocation]
		assert.True(t, exists)
		assert.NotNil(t, storedProgram)

		// Script was not in stored programs.
		assert.False(t, programsHits[scriptLocation])
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
		assert.False(t, programsHits[scriptLocation])
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
		assert.False(t, programsHits[scriptLocation])
		// Import was in stored programs.
		assert.True(t, programsHits[importedScriptLocation])
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

	runtime := newTestInterpreterRuntime()

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

	runtime := newTestInterpreterRuntime()

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
				common.MustBytesToAddress([]byte{42}),
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

	assert.Equal(t, "0x000000000000002a", loggedMessage)
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
			authorizers:  []Address{common.MustBytesToAddress([]byte{42})},
			expectedLogs: []string{"0x000000000000002a", "42"},
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
				jsoncdc.MustEncode(cadence.String("foo")),
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
				jsoncdc.MustEncode(cadence.String("foo")),
			},
			check: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidValueTypeError{}, errors.Unwrap(errors.Unwrap(err)))
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
			expectedLogs: []string{"0x0000000000000001"},
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
								Key:   cadence.String("y"),
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
								Key:   cadence.String("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				var argErr interpreter.ContainerMutationError
				require.ErrorAs(t, err, &argErr)
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
						NewStruct([]cadence.Value{cadence.String("bar")}).
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
							NewStruct([]cadence.Value{cadence.String("bar")}).
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
			rt := newTestInterpreterRuntime()

			var loggedMessages []string

			storage := newTestLedger(nil, nil)

			runtimeInterface := &testRuntimeInterface{
				storage: storage,
				getSigningAccounts: func() ([]Address, error) {
					return tc.authorizers, nil
				},
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
				meterMemory: func(_ common.MemoryUsage) error {
					return nil
				},
			}
			runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(runtimeInterface, b)
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
				assert.NoError(t, err)
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
		name         string
		script       string
		args         [][]byte
		expectedLogs []string
		check        func(t *testing.T, err error)
	}

	var tests = []testCase{
		{
			name: "No arguments",
			script: `
                pub fun main() {
                    log("t")
                }
            `,
			args:         nil,
			expectedLogs: []string{`"t"`},
		},
		{
			name: "Single argument",
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
			name: "Multiple arguments",
			script: `
                pub fun main(x: Int, y: String) {
                    log(x)
                    log(y)
                }
            `,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.NewInt(42)),
				jsoncdc.MustEncode(cadence.String("foo")),
			},
			expectedLogs: []string{"42", `"foo"`},
		},
		{
			name: "Invalid bytes",
			script: `
                pub fun main(x: Int) { }
            `,
			args: [][]byte{
				{1, 2, 3, 4}, // not valid JSON-CDC
			},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
			},
		},
		{
			name: "Type mismatch",
			script: `
                pub fun main(x: Int) {
                    log(x)
                }
            `,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.String("foo")),
			},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidValueTypeError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			name: "Address",
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
			expectedLogs: []string{"0x0000000000000001"},
		},
		{
			name: "Array",
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
			name: "Constant-sized array, too many elements",
			script: `
                pub fun main(x: [Int; 2]) {
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
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				var invalidEntryPointArgumentErr *InvalidEntryPointArgumentError
				assert.ErrorAs(t, err, &invalidEntryPointArgumentErr)
			},
		},
		{
			name: "Constant-sized array, too few elements",
			script: `
                pub fun main(x: [Int; 2]) {
                    log(x)
                }
            `,
			args: [][]byte{
				jsoncdc.MustEncode(
					cadence.NewArray(
						[]cadence.Value{
							cadence.NewInt(1),
						},
					),
				),
			},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				var invalidEntryPointArgumentErr *InvalidEntryPointArgumentError
				assert.ErrorAs(t, err, &invalidEntryPointArgumentErr)
			},
		},
		{
			name: "Dictionary",
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
								Key:   cadence.String("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			expectedLogs: []string{"42"},
		},
		{
			name: "Invalid dictionary",
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
								Key:   cadence.String("y"),
								Value: cadence.NewInt(42),
							},
						},
					),
				),
			},
			check: func(t *testing.T, err error) {
				require.Error(t, err)
				assertRuntimeErrorIsUserError(t, err)

				var argErr interpreter.ContainerMutationError
				require.ErrorAs(t, err, &argErr)
			},
		},
		{
			name: "Struct",
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
						NewStruct([]cadence.Value{cadence.String("bar")}).
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
			name: "Struct in array",
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
							NewStruct([]cadence.Value{cadence.String("bar")}).
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

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			rt := newTestInterpreterRuntime()

			var loggedMessages []string

			storage := newTestLedger(nil, nil)

			runtimeInterface := &testRuntimeInterface{
				storage: storage,
				log: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
				meterMemory: func(_ common.MemoryUsage) error {
					return nil
				},
			}
			runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(runtimeInterface, b)
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
				assert.NoError(t, err)
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

	runtime := newTestInterpreterRuntime()

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

	require.ErrorAs(t, err, &InvalidTransactionCountError{})
}

func TestRuntimeProgramWithMultipleTransaction(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	require.ErrorAs(t, err, &InvalidTransactionCountError{})
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
			runtime := newTestInterpreterRuntime()

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
				storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

	container := []byte(`
      pub resource Container {
        pub(set) var values: [Int]

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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

	ledger := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("deep-thought"):
				return deepThought, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		storage: ledger,
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

func TestRuntimeParseAndCheckProgram(t *testing.T) {

	t.Parallel()

	t.Run("ValidProgram", func(t *testing.T) {
		runtime := newTestInterpreterRuntime()

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
		runtime := newTestInterpreterRuntime()

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
		runtime := newTestInterpreterRuntime()

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

func TestRuntimeScriptReturnTypeNotReturnableError(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expected cadence.Value) {

		runtime := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
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
			require.ErrorAs(t, err, &subErr)
		} else {
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		}
	}

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		test(t,
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

		test(t,
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

		test(t,
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
				}).WithType(cadence.VariableSizedArrayType{
					ElementType: cadence.ReferenceType{
						Type: cadence.AnyStructType{},
					},
				}),
			}).WithType(cadence.VariableSizedArrayType{
				ElementType: cadence.ReferenceType{
					Type: cadence.AnyStructType{},
				},
			}),
		)
	})

	t.Run("storage path", func(t *testing.T) {

		t.Parallel()

		test(t,
			`
              pub fun main(): StoragePath {
                  return /storage/foo
              }
            `,
			cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
		)
	})

	t.Run("public path", func(t *testing.T) {

		t.Parallel()

		test(t,
			`
              pub fun main(): PublicPath {
                  return /public/foo
              }
            `,
			cadence.Path{
				Domain:     "public",
				Identifier: "foo",
			},
		)
	})

	t.Run("private path", func(t *testing.T) {

		t.Parallel()

		test(t,
			`
              pub fun main(): PrivatePath {
                  return /private/foo
              }
            `,
			cadence.Path{
				Domain:     "private",
				Identifier: "foo",
			},
		)
	})

	t.Run("capability path", func(t *testing.T) {

		t.Parallel()

		test(t,
			`
              pub fun main(): CapabilityPath {
                  return /public/foo
              }
            `,
			cadence.Path{
				Domain:     "public",
				Identifier: "foo",
			},
		)
	})

	t.Run("path", func(t *testing.T) {

		t.Parallel()

		test(t,
			`
              pub fun main(): Path {
                  return /storage/foo
              }
            `,
			cadence.Path{
				Domain:     "storage",
				Identifier: "foo",
			},
		)
	})
}

func TestRuntimeScriptParameterTypeNotImportableError(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	var subErr *ScriptParameterTypeNotImportableError
	require.ErrorAs(t, err, &subErr)
}

func TestRuntimeSyntaxError(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          log(signer.address)
        }
      }
    `)

	var loggedMessages []string

	address := common.MustBytesToAddress([]byte{42})

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

	assert.Equal(t, []string{"0x000000000000002a"}, loggedMessages)
}

func TestRuntimePublicAccountAddress(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log(getAccount(0x42).address)
        }
      }
    `)

	var loggedMessages []string

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x42})

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
			address.String(),
		},
		loggedMessages,
	)
}

func TestRuntimeAccountPublishAndAccess(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	address := common.MustBytesToAddress([]byte{42})

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
          AuthAccount(payer: signer)
        }
      }
    `)

	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
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
	assert.EqualValues(
		t,
		stdlib.AccountCreatedEventType.ID(),
		events[0].Type().ID(),
	)
}

func TestRuntimeContractAccount(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
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

	// Run script 1

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

	// Run script 2

	value, err = runtime.ExecuteScript(
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
}

func TestRuntimeInvokeContractFunction(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        pub contract Test {
            pub fun hello() {
                log("Hello World!")
            }
            pub fun helloArg(_ arg: String) {
                log("Hello ".concat(arg))
            }
            pub fun helloMultiArg(arg1: String, arg2: Int, arg3: Address) {
                log("Hello ".concat(arg1).concat(" ").concat(arg2.toString()).concat(" from ").concat(arg3.toString()))
            }
            pub fun helloReturn(_ arg: String): String {
                log("Hello return!")
                return arg
            }
            pub fun helloAuthAcc(account: AuthAccount) {
                log("Hello ".concat(account.address.toString()))
            }
            pub fun helloPublicAcc(account: PublicAccount) {
                log("Hello pub ".concat(account.address.toString()))
            }
        }
    `)

	deploy := utils.DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
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

	t.Run("simple function", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"hello",
			nil,
			nil,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello World!"`, loggedMessage)
	})

	t.Run("function with parameter", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.String("there!"),
			},
			[]sema.Type{
				sema.StringType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello there!"`, loggedMessage)
	})
	t.Run("function with return type", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloReturn",
			[]cadence.Value{
				cadence.String("there!"),
			},
			[]sema.Type{
				sema.StringType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello return!"`, loggedMessage)
	})
	t.Run("function with multiple arguments", func(tt *testing.T) {

		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloMultiArg",
			[]cadence.Value{
				cadence.String("number"),
				cadence.NewInt(42),
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.StringType,
				sema.IntType,
				&sema.AddressType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello number 42 from 0x0000000000000001"`, loggedMessage)
	})

	t.Run("function with not enough arguments panics", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloMultiArg",
			[]cadence.Value{
				cadence.String("number"),
				cadence.NewInt(42),
			},
			[]sema.Type{
				sema.StringType,
				sema.IntType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(tt, err)
		assert.ErrorAs(tt, err, &Error{})
	})
	t.Run("function with incorrect argument type errors", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewInt(42),
			},
			[]sema.Type{
				sema.IntType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.ErrorAs(tt, err, &interpreter.ValueTransferTypeError{})
	})
	t.Run("function with un-importable argument errors and error propagates", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.Capability{
					BorrowType: cadence.AddressType{}, // this will error during `importValue`
				},
			},
			[]sema.Type{
				&sema.CapabilityType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.ErrorContains(tt, err, "cannot import capability")
	})
	t.Run("function with auth account works", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloAuthAcc",
			[]cadence.Value{
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.AuthAccountType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello 0x0000000000000001"`, loggedMessage)
	})
	t.Run("function with public account works", func(tt *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloPublicAcc",
			[]cadence.Value{
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.PublicAccountType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(tt, err)

		assert.Equal(tt, `"Hello pub 0x0000000000000001"`, loggedMessage)
	})
}

func TestRuntimeContractNestedResource(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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
		storage: newTestLedger(nil, nil),
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

	runtime := newTestInterpreterRuntime()

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

	ledger := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: ledger,
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
	require.ErrorAs(t, err, &typeLoadingErr)

	require.Equal(t,
		common.AddressLocation{Address: addressValue}.TypeID(nil, "Test.R"),
		typeLoadingErr.TypeID,
	)
}

const basicFungibleTokenContract = `
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

	runtime := newTestInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := utils.DeploymentTransaction("FungibleToken", []byte(basicFungibleTokenContract))

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

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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

	runtime := newTestInterpreterRuntime()

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
		hex.EncodeToString([]byte(basicFungibleTokenContract)),
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

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
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
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) (err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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

	runtime := newTestInterpreterRuntime()

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

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	var nextAccount byte = 0x2

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		createAccount: func(payer Address) (address Address, err error) {
			result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
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
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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
					assertRuntimeErrorIsUserError(t, err)

					require.ErrorAs(t, err, &interpreter.ConditionError{})
				}
			})
		}
	}
}

func TestRuntimeBlock(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
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

func TestRuntimeUnsafeRandom(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
		runtime := newTestInterpreterRuntime()

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
		runtime := newTestInterpreterRuntime()

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
		assertRuntimeErrorIsUserError(t, err)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.ExpectCheckerErrors(t, checkerErr, 1)

		assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
	})
}

func TestRuntimeStoreIntegerTypes(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
				storage: newTestLedger(nil, nil),
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

func TestRuntimeResourceOwnerFieldUseComposite(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
              log(ref1.owner?.balance)
              log(ref1.owner?.availableBalance)
              log(ref1.owner?.storageUsed)
              log(ref1.owner?.storageCapacity)
              ref1.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.getCapability(/public/r).borrow<&Test.R>()!
              log(ref2.owner?.address)
              log(ref2.owner?.balance)
              log(ref2.owner?.availableBalance)
              log(ref2.owner?.storageUsed)
              log(ref2.owner?.storageCapacity)
              ref2.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		getAccountBalance: func(_ Address) (uint64, error) {
			// return a dummy value
			return 12300000000, nil
		},
		getAccountAvailableBalance: func(_ Address) (uint64, error) {
			// return a dummy value
			return 152300000000, nil
		},
		getStorageUsed: func(_ Address) (uint64, error) {
			// return a dummy value
			return 120, nil
		},
		getStorageCapacity: func(_ Address) (uint64, error) {
			// return a dummy value
			return 1245, nil
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
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
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
			"0x0000000000000001", // ref1.owner?.address
			"123.00000000",       // ref2.owner?.balance
			"1523.00000000",      // ref2.owner?.availableBalance
			"120",                // ref1.owner?.storageUsed
			"1245",               // ref1.owner?.storageCapacity

			"0x0000000000000001",

			"0x0000000000000001", // ref2.owner?.address
			"123.00000000",       // ref2.owner?.balance
			"1523.00000000",      // ref2.owner?.availableBalance
			"120",                // ref2.owner?.storageUsed
			"1245",               // ref2.owner?.storageCapacity

			"0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceOwnerFieldUseArray(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
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
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceOwnerFieldUseDictionary(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
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
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeMetrics(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	storage := newTestLedger(nil, nil)

	type reports struct {
		programParsed      map[common.Location]int
		programChecked     map[common.Location]int
		programInterpreted map[common.Location]int
	}

	newRuntimeInterface := func() (runtimeInterface Interface, r *reports) {

		r = &reports{
			programParsed:      map[common.Location]int{},
			programChecked:     map[common.Location]int{},
			programInterpreted: map[common.Location]int{},
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
				r.programParsed[location]++
			},
			programChecked: func(location common.Location, duration time.Duration) {
				r.programChecked[location]++
			},
			programInterpreted: func(location common.Location, duration time.Duration) {
				r.programInterpreted[location]++
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
		map[common.Location]int{
			transactionLocation: 1,
			imported1Location:   1,
		},
		r1.programParsed,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported1Location:   1,
		},
		r1.programChecked,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
		},
		r1.programInterpreted,
	)

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
		map[common.Location]int{
			transactionLocation: 1,
			imported2Location:   1,
		},
		r2.programParsed,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported2Location:   1,
		},
		r2.programChecked,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
		},
		r2.programInterpreted,
	)
}

type testWrite struct {
	owner, key []byte
}

func (w testWrite) String() string {
	return string(w.key)
}

func TestRuntimeContractWriteback(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
		})
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
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

	assert.Equal(t,
		[]testWrite{
			// storage index to contract domain storage map
			{
				addressValue[:],
				[]byte("contract"),
			},
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
			// contract domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		writes,
	)

	writes = nil

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

	assert.Empty(t, writes)

	writes = nil

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

	assert.Equal(t,
		[]testWrite{
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
		},
		writes,
	)
}

func TestRuntimeStorageWriteback(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []testWrite

	onWrite := func(owner, key, _ []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
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

	assert.Equal(t,
		[]testWrite{
			// storage index to contract domain storage map
			{
				addressValue[:],
				[]byte("contract"),
			},
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
			// contract domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import Test from 0xCADE

               transaction {

                  prepare(signer: AuthAccount) {
                      signer.save(<-Test.createR(), to: /storage/r)
                  }
               }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]testWrite{
			// storage index to storage domain storage map
			{
				addressValue[:],
				[]byte("storage"),
			},
			// resource value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
			},
			// storage domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4},
			},
		},
		writes,
	)

	readTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: AuthAccount) {
             log(signer.borrow<&Test.R>(from: /storage/r)!.test)
         }
      }
    `)

	writes = nil

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

	assert.Empty(t, writes)

	writeTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: AuthAccount) {
             let r = signer.borrow<&Test.R>(from: /storage/r)!
             r.test = 2
         }
      }
    `)

	writes = nil

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

	assert.Equal(t,
		[]testWrite{
			// resource value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
			},
		},
		writes,
	)
}

func TestRuntimeExternalError(t *testing.T) {

	t.Parallel()

	interpreterRuntime := newTestInterpreterRuntime()

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

	err := interpreterRuntime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.Error(t, err)
	assertRuntimeErrorIsExternalError(t, err)
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

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	var accountCounter uint8 = 0

	var signerAddresses []Address

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programHits []string

	var codeChanged bool

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			codeChanged = true

			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	// When code is changed, the parsed+checked programs have to be invalidated

	clearProgramsIfNeeded := func() {
		if !codeChanged {
			return
		}

		for locationID := range runtimeInterface.programs {
			delete(runtimeInterface.programs, locationID)
		}
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

	codeChanged = false

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

	location := common.AddressLocation{
		Address: signerAddresses[0],
		Name:    "HelloWorld",
	}

	require.NotContains(t, runtimeInterface.programs, location)

	clearProgramsIfNeeded()

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
	require.Equal(t, cadence.String("1"), result1)

	// The deployed hello world contract was imported,
	// assert that it was stored in the program storage
	// after it was parsed and checked

	initialProgram := runtimeInterface.programs[location]
	require.NotNil(t, initialProgram)

	// update the contract

	programHits = nil
	codeChanged = false

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

	// Assert that the contract update did NOT change
	// the program in program storage

	require.Same(t,
		initialProgram,
		runtimeInterface.programs[location],
	)
	require.NotNil(t, runtimeInterface.programs[location])

	clearProgramsIfNeeded()

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
	require.Equal(t, cadence.String("2"), result2)
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

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	programs := map[common.Location]*interpreter.Program{}

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programsHits []string

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		setProgram: func(location Location, program *interpreter.Program) error {
			programs[location] = program
			return nil
		},
		getProgram: func(location Location) (*interpreter.Program, error) {
			programsHits = append(programsHits, string(location.ID()))
			return programs[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
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

	runtime := newTestInterpreterRuntime()

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

	var accountCode []byte
	var events []cadence.Event

	var codeChanged bool

	signerAddress := common.MustBytesToAddress([]byte{0x42})

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
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
			codeChanged = true
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	// When code is changed, the parsed+checked programs have to be invalidated

	clearProgramsIfNeeded := func() {
		if !codeChanged {
			return
		}

		for locationID := range runtimeInterface.programs {
			delete(runtimeInterface.programs, locationID)
		}
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy the Test contract

	codeChanged = false
	deployTx1 := utils.DeploymentTransaction("Test", []byte(contract1))

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

	location := common.AddressLocation{
		Address: signerAddress,
		Name:    "Test",
	}

	require.NotContains(t, runtimeInterface.programs, location)

	clearProgramsIfNeeded()

	// Use the Test contract

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

	// The deployed hello world contract was imported,
	// assert that it was stored in the program storage
	// after it was parsed and checked

	initialProgram := runtimeInterface.programs[location]
	require.NotNil(t, initialProgram)

	// Update the Test contract

	codeChanged = false

	deployTx2 := utils.UpdateTransaction("Test", []byte(contract2))

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

	// Assert that the contract update did NOT change
	// the program in program storage

	require.Same(t,
		initialProgram,
		runtimeInterface.programs[location],
	)
	require.NotNil(t, runtimeInterface.programs[location])

	clearProgramsIfNeeded()

	// Use the new Test contract

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

func TestRuntimeExecuteScriptArguments(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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
				assertRuntimeErrorIsUserError(t, err)

				require.ErrorAs(t, err, &InvalidEntryPointParameterCountError{})
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

func singleIdentifierLocationResolver(t testing.TB) func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
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

func TestRuntimePanics(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      pub fun main() {
        [1][1]
      }
    `)

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
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

func TestRuntimeGetCapability(t *testing.T) {

	t.Parallel()

	t.Run("invalid: private path, public account used as auth account", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
          pub fun main(): Capability {
              let dict: {Int: AuthAccount} = {}
              let ref = &dict as &{Int: AnyStruct}
              ref[0] = getAccount(0x01) as AnyStruct
              return dict.values[0].getCapability(/private/xxx)
          }
        `)

		runtimeInterface := &testRuntimeInterface{}

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

		require.Error(t, err)
		assertRuntimeErrorIsUserError(t, err)

		var typeErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &typeErr)
	})

	t.Run("invalid: public path, public account used as auth account", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
          pub fun main(): Capability {
              let dict: {Int: AuthAccount} = {}
              let ref = &dict as &{Int: AnyStruct}
              ref[0] = getAccount(0x01) as AnyStruct
              return dict.values[0].getCapability(/public/xxx)
          }
        `)

		runtimeInterface := &testRuntimeInterface{}

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

		require.Error(t, err)
		assertRuntimeErrorIsUserError(t, err)

		var typeErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &typeErr)
	})

	t.Run("valid: public path, public account used as public account", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
          pub fun main(): Capability {
              let dict: {Int: PublicAccount} = {}
              let ref = &dict as &{Int: AnyStruct}
              ref[0] = getAccount(0x01) as AnyStruct
              return dict.values[0].getCapability(/public/xxx)
          }
        `)

		runtimeInterface := &testRuntimeInterface{}

		nextTransactionLocation := newTransactionLocationGenerator()

		res, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)
		require.Equal(t,
			cadence.Capability{
				Address: cadence.BytesToAddress([]byte{0x1}),
				Path: cadence.Path{
					Domain:     "public",
					Identifier: "xxx",
				},
			},
			res,
		)
	})
}

func TestRuntimeStackOverflow(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	const contract = `

        pub contract Recurse {

            priv fun recurse() {
                self.recurse()
            }

            init() {
                self.recurse()
            }
        }
    `

	deployTx := utils.DeploymentTransaction("Recurse", []byte(contract))

	var events []cadence.Event
	var loggedMessages []string
	var signerAddress common.Address
	accountCodes := map[common.Location]string{}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = []byte(accountCodes[location])
			return code, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy

	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.Error(t, err)
	assertRuntimeErrorIsUserError(t, err)

	var callStackLimitExceededErr CallStackLimitExceededError
	require.ErrorAs(t, err, &callStackLimitExceededErr)
}

func TestRuntimeInternalErrors(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	t.Run("script with go error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          pub fun main() {
              log("hello")
          }
        `)

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// panic due to go-error in cadence implementation
				var val any = message
				_ = val.(int)
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

		require.Error(t, err)
		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("script with cadence error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          pub fun main() {
              log("hello")
          }
        `)

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// intentionally panic
				panic(fmt.Errorf("panic trying to log %s", message))
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

		require.Error(t, err)
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          transaction {
              prepare() {}
              execute {
                  log("hello")
              }
          }
        `)

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// panic due to Cadence implementation error
				var val any = message
				_ = val.(int)
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
		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("contract function", func(t *testing.T) {

		t.Parallel()

		addressValue := Address{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
		}

		contract := []byte(`
          pub contract Test {
              pub fun hello() {
                  log("Hello World!")
              }
          }
       `)

		var accountCode []byte

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{addressValue}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
				return accountCode, nil
			},
			updateAccountContractCode: func(_ Address, _ string, code []byte) error {
				accountCode = code
				return nil
			},
			emitEvent: func(_ cadence.Event) error {
				return nil
			},
			log: func(message string) {
				// panic due to Cadence implementation error
				var val any = message
				_ = val.(int)
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		deploy := utils.DeploymentTransaction("Test", contract)
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

		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"hello",
			nil,
			nil,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("parse and check", func(t *testing.T) {

		t.Parallel()

		script := []byte("pub fun test() {}")
		runtimeInterface := &testRuntimeInterface{
			setProgram: func(location Location, program *interpreter.Program) error {
				panic(errors.New("crash while setting program"))
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read stored", func(t *testing.T) {

		t.Parallel()

		runtimeInterface := &testRuntimeInterface{
			storage: testLedger{
				getValue: func(owner, key []byte) (value []byte, err error) {
					panic(errors.New("crasher"))
				},
			},
		}

		address, err := common.BytesToAddress([]byte{0x42})
		require.NoError(t, err)

		_, err = runtime.ReadStored(
			address,
			cadence.Path{
				Domain:     "storage",
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		require.Error(t, err)
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read linked", func(t *testing.T) {

		t.Parallel()

		runtimeInterface := &testRuntimeInterface{
			storage: testLedger{
				getValue: func(owner, key []byte) (value []byte, err error) {
					panic(errors.New("crasher"))
				},
			},
		}

		address, err := common.BytesToAddress([]byte{0x42})
		require.NoError(t, err)

		_, err = runtime.ReadLinked(
			address,
			cadence.Path{
				Domain:     "storage",
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		require.Error(t, err)
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("panic with non error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`pub fun main() {}`)

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				// panic with a non-error type
				panic("crasher")
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

		require.Error(t, err)
		assertRuntimeErrorIsInternalError(t, err)
	})

}

func TestRuntimeComputationMetring(t *testing.T) {
	t.Parallel()

	type test struct {
		name        string
		code        string
		ok          bool
		expCompUsed uint
	}

	compLimit := uint(6)

	tests := []test{
		{
			name: "Infinite while loop",
			code: `
		  while true {}
		`,
			ok:          false,
			expCompUsed: compLimit,
		},
		{
			name: "Limited while loop",
			code: `
		  var i = 0
		  while i < 5 {
			  i = i + 1
		  }
		`,
			ok:          false,
			expCompUsed: compLimit,
		},
		{
			name: "statement + createArray + transferArray + too many for-in loop iterations",
			code: `
		  for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {}
		`,
			ok:          false,
			expCompUsed: compLimit,
		},
		{
			name: "statement + createArray + transferArray + some for-in loop iterations",
			code: `
		  for i in [1, 2] {}
		`,
			ok:          true,
			expCompUsed: 5,
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

			runtime := newTestInterpreterRuntime()

			compErr := errors.New("computation exceeded limit")
			var compUsed uint
			meterComputationFunc := func(kind common.ComputationKind, intensity uint) error {
				compUsed++
				if compUsed >= compLimit {
					return compErr
				}
				return nil
			}

			runtimeInterface := &testRuntimeInterface{
				storage: newTestLedger(nil, nil),
				getSigningAccounts: func() ([]Address, error) {
					return nil, nil
				},
				meterComputation: meterComputationFunc,
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
				var executionErr Error
				require.ErrorAs(t, err, &executionErr)
				require.ErrorAs(t, err.(Error).Unwrap(), &compErr)
			}

			require.Equal(t, test.expCompUsed, compUsed)
		})
	}
}

func TestRuntimeImportAnyStruct(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	var loggedMessages []string

	address := common.MustBytesToAddress([]byte{0x1})

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	err := rt.ExecuteTransaction(
		Script{
			Source: []byte(`
			  transaction(args: [AnyStruct]) {
			    prepare(signer: AuthAccount) {}
			  }
			`),
			Arguments: [][]byte{
				[]byte(`{"value":[{"value":"0xf8d6e0586b0a20c7","type":"Address"},{"value":{"domain":"private","identifier":"USDCAdminCap-ca258982-c98e-4ef0-adef-7ff80ee96b10"},"type":"Path"}],"type":"Array"}`),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	require.NoError(t, err)
}

// Error needs to be `runtime.Error`, and the inner error should be `errors.UserError`.
//
func assertRuntimeErrorIsUserError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.True(
		t,
		runtimeErrors.IsUserError(innerError),
		"Expected `UserError`, found `%T`", innerError,
	)
}

// Error needs to be `runtime.Error`, and the inner error should be `errors.InternalError`.
//
func assertRuntimeErrorIsInternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.True(
		t,
		runtimeErrors.IsInternalError(innerError),
		"Expected `UserError`, found `%T`", innerError,
	)
}

// Error needs to be `runtime.Error`, and the inner error should be `interpreter.ExternalError`.
//
func assertRuntimeErrorIsExternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.ErrorAs(t, innerError, &runtimeErrors.ExternalError{})
}
