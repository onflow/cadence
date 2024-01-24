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
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type testLedger struct {
	storedValues      map[string][]byte
	valueExists       func(owner, key []byte) (exists bool, err error)
	getValue          func(owner, key []byte) (value []byte, err error)
	setValue          func(owner, key, value []byte) (err error)
	allocateSlabIndex func(owner []byte) (atree.SlabIndex, error)
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

func (s testLedger) AllocateSlabIndex(owner []byte) (atree.SlabIndex, error) {
	return s.allocateSlabIndex(owner)
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

	return testLedger{
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
		allocateSlabIndex: func(owner []byte) (result atree.SlabIndex, err error) {
			index := storageIndices[string(owner)] + 1
			storageIndices[string(owner)] = index
			binary.BigEndian.PutUint64(result[:], index)
			return
		},
	}
}

type testInterpreterRuntime struct {
	*interpreterRuntime
}

var _ Runtime = testInterpreterRuntime{}

func newTestInterpreterRuntime() testInterpreterRuntime {
	return testInterpreterRuntime{
		interpreterRuntime: NewInterpreterRuntime(Config{
			AtreeValidationEnabled: true,
		}).(*interpreterRuntime),
	}
}

func (r testInterpreterRuntime) ExecuteTransaction(script Script, context Context) error {
	i := context.Interface.(*testRuntimeInterface)
	i.onTransactionExecutionStart()
	return r.interpreterRuntime.ExecuteTransaction(script, context)
}

func (r testInterpreterRuntime) ExecuteScript(script Script, context Context) (cadence.Value, error) {
	i := context.Interface.(*testRuntimeInterface)
	i.onScriptExecutionStart()
	value, err := r.interpreterRuntime.ExecuteScript(script, context)
	// If there was a return value, let's also ensure it can be encoded
	// TODO: also test CCF
	if value != nil && err == nil {
		_ = jsoncdc.MustEncode(value)
	}
	return value, err
}

type testRuntimeInterface struct {
	resolveLocation  func(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
	getCode          func(_ Location) ([]byte, error)
	getAndSetProgram func(
		location Location,
		load func() (*interpreter.Program, error),
	) (*interpreter.Program, error)
	setInterpreterSharedState func(state *interpreter.SharedState)
	getInterpreterSharedState func() *interpreter.SharedState
	storage                   testLedger
	createAccount             func(payer Address) (address Address, err error)
	addEncodedAccountKey      func(address Address, publicKey []byte) error
	removeEncodedAccountKey   func(address Address, index int) (publicKey []byte, err error)
	addAccountKey             func(
		address Address,
		publicKey *stdlib.PublicKey,
		hashAlgo HashAlgorithm,
		weight int,
	) (*stdlib.AccountKey, error)
	getAccountKey             func(address Address, index int) (*stdlib.AccountKey, error)
	removeAccountKey          func(address Address, index int) (*stdlib.AccountKey, error)
	accountKeysCount          func(address Address) (uint64, error)
	updateAccountContractCode func(location common.AddressLocation, code []byte) error
	getAccountContractCode    func(location common.AddressLocation) (code []byte, err error)
	removeAccountContractCode func(location common.AddressLocation) (err error)
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
	programParsed      func(location Location, duration time.Duration)
	programChecked     func(location Location, duration time.Duration)
	programInterpreted func(location Location, duration time.Duration)
	readRandom         func([]byte) error
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
	programs                   map[Location]*interpreter.Program
	implementationDebugLog     func(message string) error
	validatePublicKey          func(publicKey *stdlib.PublicKey) error
	bLSVerifyPOP               func(pk *stdlib.PublicKey, s []byte) (bool, error)
	blsAggregateSignatures     func(sigs [][]byte) ([]byte, error)
	blsAggregatePublicKeys     func(keys []*stdlib.PublicKey) (*stdlib.PublicKey, error)
	getAccountContractNames    func(address Address) ([]string, error)
	recordTrace                func(operation string, location Location, duration time.Duration, attrs []attribute.KeyValue)
	meterMemory                func(usage common.MemoryUsage) error
	computationUsed            func() (uint64, error)
	memoryUsed                 func() (uint64, error)
	interactionUsed            func() (uint64, error)
	updatedContractCode        bool
	generateAccountID          func(address common.Address) (uint64, error)
}

func (i *testRuntimeInterface) GenerateAccountID(address common.Address) (uint64, error) {
	if i.generateAccountID == nil {
		return 0, nil
	}

	return i.generateAccountID(address)
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

func (i *testRuntimeInterface) GetOrLoadProgram(
	location Location,
	load func() (*interpreter.Program, error),
) (
	program *interpreter.Program,
	err error,
) {
	if i.getAndSetProgram == nil {
		if i.programs == nil {
			i.programs = map[Location]*interpreter.Program{}
		}

		var ok bool
		program, ok = i.programs[location]
		if ok {
			return
		}

		program, err = load()

		// NOTE: important: still set empty program,
		// even if error occurred

		i.programs[location] = program

		return
	}

	return i.getAndSetProgram(location, load)
}

func (i *testRuntimeInterface) SetInterpreterSharedState(state *interpreter.SharedState) {
	if i.setInterpreterSharedState == nil {
		return
	}

	i.setInterpreterSharedState(state)
}

func (i *testRuntimeInterface) GetInterpreterSharedState() *interpreter.SharedState {
	if i.getInterpreterSharedState == nil {
		return nil
	}

	return i.getInterpreterSharedState()
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

func (i *testRuntimeInterface) AllocateSlabIndex(owner []byte) (atree.SlabIndex, error) {
	if i.storage.allocateSlabIndex == nil {
		panic("must specify testRuntimeInterface.storage.allocateStorageIndex")
	}
	return i.storage.AllocateSlabIndex(owner)
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

func (i *testRuntimeInterface) AddAccountKey(
	address Address,
	publicKey *stdlib.PublicKey,
	hashAlgo HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	if i.addAccountKey == nil {
		panic("must specify testRuntimeInterface.addAccountKey")
	}
	return i.addAccountKey(address, publicKey, hashAlgo, weight)
}

func (i *testRuntimeInterface) GetAccountKey(address Address, index int) (*stdlib.AccountKey, error) {
	if i.getAccountKey == nil {
		panic("must specify testRuntimeInterface.getAccountKey")
	}
	return i.getAccountKey(address, index)
}

func (i *testRuntimeInterface) AccountKeysCount(address Address) (uint64, error) {
	if i.accountKeysCount == nil {
		panic("must specify testRuntimeInterface.accountKeysCount")
	}
	return i.accountKeysCount(address)
}

func (i *testRuntimeInterface) RevokeAccountKey(address Address, index int) (*stdlib.AccountKey, error) {
	if i.removeAccountKey == nil {
		panic("must specify testRuntimeInterface.removeAccountKey")
	}
	return i.removeAccountKey(address, index)
}

func (i *testRuntimeInterface) UpdateAccountContractCode(location common.AddressLocation, code []byte) (err error) {
	if i.updateAccountContractCode == nil {
		panic("must specify testRuntimeInterface.updateAccountContractCode")
	}

	err = i.updateAccountContractCode(location, code)
	if err != nil {
		return err
	}

	i.updatedContractCode = true

	return nil
}

func (i *testRuntimeInterface) GetAccountContractCode(location common.AddressLocation) (code []byte, err error) {
	if i.getAccountContractCode == nil {
		panic("must specify testRuntimeInterface.getAccountContractCode")
	}
	return i.getAccountContractCode(location)
}

func (i *testRuntimeInterface) RemoveAccountContractCode(location common.AddressLocation) (err error) {
	if i.removeAccountContractCode == nil {
		panic("must specify testRuntimeInterface.removeAccountContractCode")
	}
	return i.removeAccountContractCode(location)
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

func (i *testRuntimeInterface) ProgramParsed(location Location, duration time.Duration) {
	if i.programParsed == nil {
		return
	}
	i.programParsed(location, duration)
}

func (i *testRuntimeInterface) ProgramChecked(location Location, duration time.Duration) {
	if i.programChecked == nil {
		return
	}
	i.programChecked(location, duration)
}

func (i *testRuntimeInterface) ProgramInterpreted(location Location, duration time.Duration) {
	if i.programInterpreted == nil {
		return
	}
	i.programInterpreted(location, duration)
}

func (i *testRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 1, nil
}

func (i *testRuntimeInterface) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, height)
	if err != nil {
		panic(err)
	}

	encoded := buf.Bytes()
	var hash stdlib.BlockHash
	copy(hash[sema.BlockTypeIdFieldType.Size-int64(len(encoded)):], encoded)

	block = stdlib.Block{
		Height:    height,
		View:      height,
		Hash:      hash,
		Timestamp: time.Unix(int64(height), 0).UnixNano(),
	}
	return block, true, nil
}

func (i *testRuntimeInterface) ReadRandom(buffer []byte) error {
	if i.readRandom == nil {
		return nil
	}
	return i.readRandom(buffer)
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

func (i *testRuntimeInterface) ValidatePublicKey(key *stdlib.PublicKey) error {
	if i.validatePublicKey == nil {
		return errors.New("mock defaults to public key validation failure")
	}

	return i.validatePublicKey(key)
}

func (i *testRuntimeInterface) BLSVerifyPOP(key *stdlib.PublicKey, s []byte) (bool, error) {
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

func (i *testRuntimeInterface) BLSAggregatePublicKeys(keys []*stdlib.PublicKey) (*stdlib.PublicKey, error) {
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

func (i *testRuntimeInterface) RecordTrace(operation string, location Location, duration time.Duration, attrs []attribute.KeyValue) {
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

func (i *testRuntimeInterface) ComputationUsed() (uint64, error) {
	if i.computationUsed == nil {
		return 0, nil
	}

	return i.computationUsed()
}

func (i *testRuntimeInterface) MemoryUsed() (uint64, error) {
	if i.memoryUsed == nil {
		return 0, nil
	}

	return i.memoryUsed()
}

func (i *testRuntimeInterface) InteractionUsed() (uint64, error) {
	if i.interactionUsed == nil {
		return 0, nil
	}

	return i.interactionUsed()
}

func (i *testRuntimeInterface) onTransactionExecutionStart() {
	i.invalidateUpdatedPrograms()
}

func (i *testRuntimeInterface) onScriptExecutionStart() {
	i.invalidateUpdatedPrograms()
}

func (i *testRuntimeInterface) invalidateUpdatedPrograms() {
	if i.updatedContractCode {
		for location := range i.programs {
			delete(i.programs, location)
		}
		i.updatedContractCode = false
	}
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
		programChecked: func(location Location, duration time.Duration) {
			checkCount += 1
		},
	}

	nextScriptLocation := newScriptLocationGenerator()

	const transactionCount = 10
	for i := 0; i < transactionCount; i++ {

		value, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
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

	var programs sync.Map

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		programChecked: func(location Location, duration time.Duration) {
			atomic.AddUint64(&checkCount, 1)
		},
		getAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			item, ok := programs.Load(location)
			if ok {
				program = item.(*interpreter.Program)
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs.Store(location, program)

			return
		},
	}

	nextScriptLocation := newScriptLocationGenerator()

	var wg sync.WaitGroup
	const concurrency uint64 = 10
	for i := uint64(0); i < concurrency; i++ {

		location := nextScriptLocation()

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
	// require.Equal(t, concurrency+1, checkCount)
}

func TestRuntimeProgramSetAndGet(t *testing.T) {

	t.Parallel()

	programs := map[Location]*interpreter.Program{}
	programsHits := make(map[Location]bool)

	importedScript := []byte(`
      transaction {
          prepare() {}
          execute {}
      }
    `)
	importedScriptLocation := common.StringLocation("imported")
	scriptLocation := common.StringLocation("placeholder")

	runtime := newTestInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{
		getAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			var ok bool
			program, ok = programs[location]

			programsHits[location] = ok

			if ok {
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs[location] = program

			return
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

		// Call a second time to hit stored programs.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		assert.True(t, programsHits[scriptLocation])
	})

	t.Run("imported program previously parsed, hit", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
        `)

		// Call a second time to hit the stored programs
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		assert.True(t, programsHits[scriptLocation])
		assert.False(t, programsHits[importedScriptLocation])
	})
}

func newLocationGenerator[T ~[32]byte]() func() T {
	var count uint64
	return func() T {
		t := T{}
		newCount := atomic.AddUint64(&count, 1)
		binary.LittleEndian.PutUint64(t[:], newCount)
		return t
	}
}

func newTransactionLocationGenerator() func() common.TransactionLocation {
	return newLocationGenerator[common.TransactionLocation]()
}

func newScriptLocationGenerator() func() common.ScriptLocation {
	return newLocationGenerator[common.ScriptLocation]()
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
	RequireError(t, err)

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
		check        func(t *testing.T, err error)
		label        string
		script       string
		args         [][]byte
		authorizers  []Address
		expectedLogs []string
		contracts    map[common.AddressLocation][]byte
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
				RequireError(t, err)

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
				RequireError(t, err)

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
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				var argErr interpreter.ContainerMutationError
				require.ErrorAs(t, err, &argErr)
			},
		},
		{
			label: "Struct",
			contracts: map[common.AddressLocation][]byte{
				{
					Address: common.MustBytesToAddress([]byte{0x1}),
					Name:    "C",
				}: []byte(`
                  pub contract C {
                      pub struct Foo {
                           pub var y: String

                           init() {
                               self.y = "initial string"
                           }
                      }
                  }
                `),
			},
			script: `
              import C from 0x1

              transaction(x: C.Foo) {
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
							Location: common.AddressLocation{
								Address: common.MustBytesToAddress([]byte{0x1}),
								Name:    "C",
							},
							QualifiedIdentifier: "C.Foo",
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
			contracts: map[common.AddressLocation][]byte{
				{
					Address: common.MustBytesToAddress([]byte{0x1}),
					Name:    "C",
				}: []byte(`
                  pub contract C {
                      pub struct Foo {
                           pub var y: String

                           init() {
                               self.y = "initial string"
                           }
                      }
                  }
                `),
			},
			script: `
              import C from 0x1

              transaction(f: [C.Foo]) {
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
								Location: common.AddressLocation{
									Address: common.MustBytesToAddress([]byte{0x1}),
									Name:    "C",
								},
								QualifiedIdentifier: "C.Foo",
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
				resolveLocation: singleIdentifierLocationResolver(t),
				getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					return tc.contracts[location], nil
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
					Location:  common.TransactionLocation{},
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
		check        func(t *testing.T, err error)
		name         string
		script       string
		args         [][]byte
		expectedLogs []string
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
				RequireError(t, err)

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
				RequireError(t, err)

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
				RequireError(t, err)

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
				RequireError(t, err)

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
				RequireError(t, err)

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
							Location:            common.ScriptLocation{},
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
								Location:            common.ScriptLocation{},
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
		{
			name: "Path subtype",
			script: `
                pub fun main(x: StoragePath) {
                    log(x)
                }
            `,
			args: [][]byte{
				jsoncdc.MustEncode(cadence.Path{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				}),
			},
			expectedLogs: []string{
				"/storage/foo",
			},
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
					Location:  common.ScriptLocation{},
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
	RequireError(t, err)

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
	RequireError(t, err)

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

func TestRuntimeScriptReturnSpecial(t *testing.T) {

	t.Parallel()

	type testCase struct {
		expected cadence.Value
		code     string
		invalid  bool
	}

	test := func(t *testing.T, test testCase) {

		runtime := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
		}

		actual, err := runtime.ExecuteScript(
			Script{
				Source: []byte(test.code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		if test.invalid {
			RequireError(t, err)

			var subErr *InvalidScriptReturnTypeError
			require.ErrorAs(t, err, &subErr)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		}
	}

	t.Run("interpreted function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  pub fun main(): AnyStruct {
                      return fun (): Int {
                          return 0
                      }
                  }
                `,
				expected: cadence.Function{
					FunctionType: cadence.TypeWithCachedTypeID(
						&cadence.FunctionType{
							ReturnType: cadence.IntType{},
						},
					).(*cadence.FunctionType),
				},
			},
		)
	})

	t.Run("host function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  pub fun main(): AnyStruct {
                      return panic
                  }
                `,
				expected: cadence.Function{
					FunctionType: cadence.TypeWithCachedTypeID(
						&cadence.FunctionType{
							Parameters: []cadence.Parameter{
								{
									Label:      sema.ArgumentLabelNotRequired,
									Identifier: "message",
									Type:       cadence.StringType{},
								},
							},
							ReturnType: cadence.NeverType{},
						},
					).(*cadence.FunctionType),
				},
			},
		)
	})

	t.Run("bound function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  pub struct S {
                      pub fun f() {}
                  }

                  pub fun main(): AnyStruct {
                      let s = S()
                      return s.f
                  }
                `,
				expected: cadence.Function{
					FunctionType: cadence.TypeWithCachedTypeID(
						&cadence.FunctionType{
							ReturnType: cadence.VoidType{},
						},
					).(*cadence.FunctionType),
				},
			},
		)
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  pub fun main(): AnyStruct {
                      let a: Address = 0x1
                      return &a as &Address
                  }
                `,
				expected: cadence.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
		)
	})

	t.Run("recursive reference", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  pub fun main(): AnyStruct {
                      let refs: [&AnyStruct] = []
                      refs.append(&refs as &AnyStruct)
                      return refs
                  }
                `,
				expected: cadence.NewArray([]cadence.Value{
					cadence.NewArray([]cadence.Value{
						nil,
					}).WithType(&cadence.VariableSizedArrayType{
						ElementType: &cadence.ReferenceType{
							Type: cadence.AnyStructType{},
						},
					}),
				}).WithType(&cadence.VariableSizedArrayType{
					ElementType: &cadence.ReferenceType{
						Type: cadence.AnyStructType{},
					},
				}),
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

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

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
	RequireError(t, err)

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
          // Important: Perform a write which will be pending until the end of the transaction,
          // but should be (temporarily) committed when the AuthAccount constructor is called
          signer.save(42, to: /storage/answer)
          AuthAccount(payer: signer)
        }
      }
    `)

	var events []cadence.Event

	var performedWrite bool

	onWrite := func(owner, key, value []byte) {
		performedWrite = true
	}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		createAccount: func(payer Address) (address Address, err error) {
			// Check that pending writes were committed before
			assert.True(t, performedWrite)
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

	deploy := DeploymentTransaction("Test", contract)

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()
	nextScriptLocation := newScriptLocationGenerator()

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
			Location:  nextScriptLocation(),
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
			Location:  nextScriptLocation(),
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

	deploy := DeploymentTransaction("Test", contract)

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

	t.Run("simple function", func(t *testing.T) {
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
		require.NoError(t, err)

		assert.Equal(t, `"Hello World!"`, loggedMessage)
	})

	t.Run("function with parameter", func(t *testing.T) {
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
		require.NoError(t, err)

		assert.Equal(t, `"Hello there!"`, loggedMessage)
	})

	t.Run("function with return type", func(t *testing.T) {
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
		require.NoError(t, err)

		assert.Equal(t, `"Hello return!"`, loggedMessage)
	})

	t.Run("function with multiple arguments", func(t *testing.T) {

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
				sema.TheAddressType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello number 42 from 0x0000000000000001"`, loggedMessage)
	})

	t.Run("function with not enough arguments panics", func(t *testing.T) {
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

		RequireError(t, err)

		assert.ErrorAs(t, err, &Error{})
	})

	t.Run("function with incorrect argument type errors", func(t *testing.T) {
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
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
	})

	t.Run("function with un-importable argument errors and error propagates (path capability)", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewPathCapability(
					cadence.Address{},
					cadence.Path{
						Domain:     common.PathDomainPublic,
						Identifier: "test",
					},
					cadence.AddressType{}, // this will error during `importValue`
				),
			},
			[]sema.Type{
				&sema.CapabilityType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		require.ErrorContains(t, err, "cannot import capability")
	})

	t.Run("function with un-importable argument errors and error propagates (ID capability)", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewIDCapability(
					42,
					cadence.Address{},
					cadence.AddressType{}, // this will error during `importValue`
				),
			},
			[]sema.Type{
				&sema.CapabilityType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		require.ErrorContains(t, err, "cannot import capability")
	})

	t.Run("function with auth account works", func(t *testing.T) {
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
		require.NoError(t, err)

		assert.Equal(t, `"Hello 0x0000000000000001"`, loggedMessage)
	})
	t.Run("function with public account works", func(t *testing.T) {
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
		require.NoError(t, err)

		assert.Equal(t, `"Hello pub 0x0000000000000001"`, loggedMessage)
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

	deploy := DeploymentTransaction("Test", contract)

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)
	removal := RemovalTransaction("Test")

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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		removeAccountContractCode: func(_ common.AddressLocation) (err error) {
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
	RequireError(t, err)

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

	deploy := DeploymentTransaction("FungibleToken", []byte(basicFungibleTokenContract))

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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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
					RequireError(t, err)

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

func TestRuntimeRandom(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let rand1 = revertibleRandom()
          log(rand1)
          let rand2 = unsafeRandom()
          log(rand2)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		readRandom: func(buffer []byte) error {
			binary.LittleEndian.PutUint64(buffer, 7558174677681708339)
			return nil
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
		RequireError(t, err)

		assertRuntimeErrorIsUserError(t, err)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.RequireCheckerErrors(t, checkerErr, 1)

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

			deploy := DeploymentTransaction("Test", contract)

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
				getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
					return accountCode, nil
				},
				updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)

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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)

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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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

	deploy := DeploymentTransaction("Test", contract)

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

	accountCodes := map[Location][]byte{}
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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
		programParsed      map[Location]int
		programChecked     map[Location]int
		programInterpreted map[Location]int
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

type ownerKeyPair struct {
	owner, key []byte
}

func (w ownerKeyPair) String() string {
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

	deploy := DeploymentTransaction("Test", contract)

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
	var writes []ownerKeyPair

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, ownerKeyPair{
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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) (err error) {
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
		[]ownerKeyPair{
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
		[]ownerKeyPair{
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

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []ownerKeyPair

	onWrite := func(owner, key, _ []byte) {
		writes = append(writes, ownerKeyPair{
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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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
		[]ownerKeyPair{
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
		[]ownerKeyPair{
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
		[]ownerKeyPair{
			// resource value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
			},
		},
		writes,
	)
}

type logPanicError struct{}

func (logPanicError) Error() string {
	return ""
}

var _ error = logPanicError{}

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

	runtimeInterface := &testRuntimeInterface{
		getSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		log: func(message string) {
			panic(logPanicError{})
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

	RequireError(t, err)

	assertRuntimeErrorIsExternalError(t, err)
}

func TestRuntimeExternalNonError(t *testing.T) {

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

	RequireError(t, err)

	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.ErrorAs(t, innerError, &runtimeErrors.ExternalNonError{})
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

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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

	const callHelloTransactionTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: AuthAccount) {
                log(HelloWorld.hello())
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

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract1))
	updateTx := UpdateTransaction("HelloWorld", []byte(helloWorldContract2))

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programHits []string

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
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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
	nextScriptLocation := newScriptLocationGenerator()

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

	location := common.AddressLocation{
		Address: signerAddresses[0],
		Name:    "HelloWorld",
	}

	require.NotContains(t, runtimeInterface.programs, location)

	// call the initial hello function

	callScript := []byte(fmt.Sprintf(callHelloScriptTemplate, Address{accountCounter}))

	result1, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
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

	// call the new hello function from a script

	result2, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, cadence.String("2"), result2)

	// call the new hello function from a transaction

	callTransaction := []byte(fmt.Sprintf(callHelloTransactionTemplate, Address{accountCounter}))

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t,
		[]string{`"2"`},
		loggedMessages,
	)
}

func TestRuntimeProgramsHitForToplevelPrograms(t *testing.T) {

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

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	programs := map[common.Location]*interpreter.Program{}

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programsHits []Location

	runtimeInterface := &testRuntimeInterface{
		createAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		getAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			programsHits = append(programsHits, location)

			var ok bool
			program, ok = programs[location]
			if ok {
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs[location] = program

			return
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
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

	require.Equal(t,
		[]common.Location{
			common.TransactionLocation{
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.TransactionLocation{
				0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.TransactionLocation{
				0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.AddressLocation{
				Address: Address{0x1},
				Name:    "HelloWorld",
			},
			common.AddressLocation{
				Address: Address{0x1},
				Name:    "HelloWorld",
			},
		},
		programsHits,
	)
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
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy the Test contract

	deployTx1 := DeploymentTransaction("Test", []byte(contract1))

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

	nextScriptLocation := newScriptLocationGenerator()

	_, err = runtime.ExecuteScript(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	// The deployed hello world contract was imported,
	// assert that it was stored in the program storage
	// after it was parsed and checked

	initialProgram := runtimeInterface.programs[location]
	require.NotNil(t, initialProgram)

	// Update the Test contract

	deployTx2 := UpdateTransaction("Test", []byte(contract2))

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
			Location:  nextScriptLocation(),
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
				RequireError(t, err)

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

func TestRuntimeGetConfig(t *testing.T) {
	t.Parallel()

	rt := newTestInterpreterRuntime()

	config := rt.Config()
	expected := rt.defaultConfig
	require.Equal(t, expected, config)
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
	RequireError(t, err)

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

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

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

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

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

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
		}

		res, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		require.Equal(t,
			cadence.NewPathCapability(
				cadence.BytesToAddress([]byte{0x1}),
				cadence.Path{
					Domain:     common.PathDomainPublic,
					Identifier: "xxx",
				},
				nil,
			),
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

	deployTx := DeploymentTransaction("Recurse", []byte(contract))

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
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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
	RequireError(t, err)

	assertRuntimeErrorIsUserError(t, err)

	var callStackLimitExceededErr CallStackLimitExceededError
	require.ErrorAs(t, err, &callStackLimitExceededErr)
}

func TestRuntimeInternalErrors(t *testing.T) {

	t.Parallel()

	t.Run("script with go error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          pub fun main() {
              log("hello")
          }
        `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// panic due to go-error in cadence implementation
				var val any = message
				_ = val.(int)
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("script with cadence error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          pub fun main() {
              log("hello")
          }
        `)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// intentionally panic
				panic(fmt.Errorf("panic trying to log %s", message))
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

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

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				// panic due to Cadence implementation error
				var val any = message
				_ = val.(int)
			},
		}

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		RequireError(t, err)

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

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{addressValue}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return accountCode, nil
			},
			updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
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

		deploy := DeploymentTransaction("Test", contract)
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

		RequireError(t, err)

		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("parse and check", func(t *testing.T) {

		t.Parallel()

		script := []byte("pub fun test() {}")

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			getAndSetProgram: func(_ Location, _ func() (*interpreter.Program, error)) (*interpreter.Program, error) {
				panic(errors.New("crash while getting/setting program"))
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

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read stored", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

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
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read linked", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

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
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("panic with non error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`pub fun main() {}`)

		runtime := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(usage common.MemoryUsage) error {
				// panic with a non-error type
				panic("crasher")
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsInternalError(t, err)
	})

}

func TestRuntimeComputationMetring(t *testing.T) {
	t.Parallel()

	type test struct {
		name      string
		code      string
		ok        bool
		hits      uint
		intensity uint
	}

	compLimit := uint(6)

	tests := []test{
		{
			name: "Infinite while loop",
			code: `
          while true {}
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 6,
		},
		{
			name: "Limited while loop",
			code: `
          var i = 0
          while i < 5 {
              i = i + 1
          }
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 6,
		},
		{
			name: "statement + createArray + transferArray + too many for-in loop iterations",
			code: `
          for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {}
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 15,
		},
		{
			name: "statement + createArray + transferArray + two for-in loop iterations",
			code: `
          for i in [1, 2] {}
        `,
			ok:        true,
			hits:      5,
			intensity: 6,
		},
		{
			name: "statement + functionInvocation + encoding",
			code: `
          acc.save("A quick brown fox jumps over the lazy dog", to:/storage/some_path)
        `,
			ok:        true,
			hits:      3,
			intensity: 88,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			script := []byte(
				fmt.Sprintf(
					`
                  transaction {
                      prepare(acc: AuthAccount) {
                          %s
                      }
                  }
                `,
					test.code,
				),
			)

			runtime := newTestInterpreterRuntime()

			compErr := errors.New("computation exceeded limit")
			var hits, totalIntensity uint
			meterComputationFunc := func(kind common.ComputationKind, intensity uint) error {
				hits++
				totalIntensity += intensity
				if hits >= compLimit {
					return compErr
				}
				return nil
			}

			address := common.MustBytesToAddress([]byte{0x1})

			runtimeInterface := &testRuntimeInterface{
				storage: newTestLedger(nil, nil),
				getSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
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
				RequireError(t, err)

				var executionErr Error
				require.ErrorAs(t, err, &executionErr)
				require.ErrorAs(t, err.(Error).Unwrap(), &compErr)
			}

			assert.Equal(t, test.hits, hits)
			assert.Equal(t, test.intensity, totalIntensity)
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
			Location:  common.TransactionLocation{},
		},
	)
	require.NoError(t, err)
}

// Error needs to be `runtime.Error`, and the inner error should be `errors.UserError`.
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
func assertRuntimeErrorIsInternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.True(
		t,
		runtimeErrors.IsInternalError(innerError),
		"Expected `InternalError`, found `%T`", innerError,
	)
}

// Error needs to be `runtime.Error`, and the inner error should be `interpreter.ExternalError`.
func assertRuntimeErrorIsExternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.ErrorAs(t, innerError, &runtimeErrors.ExternalError{})
}

func BenchmarkRuntimeScriptNoop(b *testing.B) {

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
	}

	script := Script{
		Source: []byte("pub fun main() {}"),
	}

	environment := NewScriptInterpreterEnvironment(Config{})

	context := Context{
		Interface:   runtimeInterface,
		Location:    common.ScriptLocation{},
		Environment: environment,
	}

	require.NotNil(b, stdlib.CryptoChecker())

	runtime := newTestInterpreterRuntime()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = runtime.ExecuteScript(script, context)
	}
}

func TestRuntimeImportTestStdlib(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(`
                import Test

                pub fun main() {
                    Test.assert(true)
                }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	errs := checker.RequireCheckerErrors(t, err, 1)

	notDeclaredErr := &sema.NotDeclaredError{}
	require.ErrorAs(t, errs[0], &notDeclaredErr)
	assert.Equal(t, "Test", notDeclaredErr.Name)
}

func TestRuntimeGetCurrentBlockScript(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(`
                pub fun main(): AnyStruct {
                    return getCurrentBlock()
                }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	var subErr *ValueNotExportableError
	require.ErrorAs(t, err, &subErr)
}

func TestRuntimeTypeMismatchErrorMessage(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	contract := []byte(`
      pub contract Foo {
         pub struct Bar {}
      }
    `)

	deploy := DeploymentTransaction("Foo", contract)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	signerAccount := address1

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()
	nextScriptLocation := newScriptLocationGenerator()

	// Deploy same contract to two different accounts

	for _, address := range []Address{address1, address2} {
		signerAccount = address

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
	}

	// Set up account

	setupTransaction := []byte(`
      import Foo from 0x1

      transaction {

          prepare(acct: AuthAccount) {
              acct.save(Foo.Bar(), to: /storage/bar)

              acct.link<&Foo.Bar>(
                  /public/bar,
                  target: /storage/bar
              )
          }
      }
    `)

	signerAccount = address1

	err := runtime.ExecuteTransaction(
		Script{
			Source: setupTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Use wrong type

	script := []byte(`
      import Foo from 0x2

      pub fun main() {
        getAuthAccount(0x1).borrow<&Foo.Bar>(from: /storage/bar)
      }
    `)

	_, err = runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorContains(t, err, "expected type `A.0000000000000002.Foo.Bar`, got `A.0000000000000001.Foo.Bar`")

}

func TestRuntimeErrorExcerpts(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	script := []byte(`
    pub fun main(): Int {
        // fill lines so the error occurs on lines 9 and 10
        // 
        // 
        //
        //
        let a = [1,2,3,4]
        return a
            .firstIndex(of: 5)!
    }
    `)

	runtimeInterface := &testRuntimeInterface{
		getAccountBalance:          noopRuntimeUInt64Getter,
		getAccountAvailableBalance: noopRuntimeUInt64Getter,
		getStorageUsed:             noopRuntimeUInt64Getter,
		getStorageCapacity:         noopRuntimeUInt64Getter,
		accountKeysCount:           noopRuntimeUInt64Getter,
		storage:                    newTestLedger(nil, nil),
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	errorString := `Execution failed:
error: unexpectedly found nil while forcing an Optional value
  --> 0000000000000000000000000000000000000000000000000000000000000000:9:15
   |
 9 |         return a
10 |             .firstIndex(of: 5)!
   |                ^^^^^^^^^^^^^^^^
`

	require.Equal(t, errorString, err.Error())
}

func TestRuntimeErrorExcerptsMultiline(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	script := []byte(`
    pub fun main(): String {
        // fill lines so the error occurs on lines 9 and 10
        // 
        // 
        //
        //
        let a = [1,2,3,4]
        return a
            .firstIndex(of: 5)
                ?.toString()!
    }
    `)

	runtimeInterface := &testRuntimeInterface{
		getAccountBalance:          noopRuntimeUInt64Getter,
		getAccountAvailableBalance: noopRuntimeUInt64Getter,
		getStorageUsed:             noopRuntimeUInt64Getter,
		getStorageCapacity:         noopRuntimeUInt64Getter,
		accountKeysCount:           noopRuntimeUInt64Getter,
		storage:                    newTestLedger(nil, nil),
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	errorString := `Execution failed:
error: unexpectedly found nil while forcing an Optional value
  --> 0000000000000000000000000000000000000000000000000000000000000000:9:15
   |
 9 |         return a
10 |             .firstIndex(of: 5)
11 |                 ?.toString()!
   |                ^^^^^^^^^^^^^^
`

	require.Equal(t, errorString, err.Error())
}

// https://github.com/onflow/cadence/issues/2464
func TestRuntimeAccountTypeEquality(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()
	rt.defaultConfig.AccountLinkingEnabled = true

	script := []byte(`
      #allowAccountLinking

      pub fun main(address: Address): AnyStruct {
          let acct = getAuthAccount(address)
          let p = /private/tmp

          acct.linkAccount(p)
          let capType = acct.getCapability<&AuthAccount>(p).borrow()!.getType()

          return Type<AuthAccount>() == capType
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(nil, b)
		},
		emitEvent: func(_ cadence.Event) error {
			return nil
		},
	}

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs([]cadence.Value{
				cadence.Address(common.MustBytesToAddress([]byte{0x1})),
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

	require.Equal(t, cadence.Bool(true), result)
}

func TestRuntimeUserPanicToError(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf(
		"wrapped: %w",
		runtimeErrors.NewDefaultUserError("user error"),
	)
	retErr := userPanicToError(func() { panic(err) })
	require.Equal(t, retErr, err)
}

func TestRuntimeDestructorReentrancyPrevention(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	script := []byte(`
      pub resource Vault {
          // Balance of a user's Vault
          // we use unsigned fixed point numbers for balances
          // because they can represent decimals and do not allow negative values
          pub var balance: UFix64

          init(balance: UFix64) {
              self.balance = balance
          }

          pub fun withdraw(amount: UFix64): @Vault {
              self.balance = self.balance - amount
              return <-create Vault(balance: amount)
          }

          pub fun deposit(from: @Vault) {
              self.balance = self.balance + from.balance
              destroy from
          }
      }

      // --- this code actually makes use of the vuln ---
      pub resource InnerResource {
          pub var victim: @Vault;
          pub var here: Bool;
          pub var parent: &OuterResource;
          init(victim: @Vault, parent: &OuterResource) {
              self.victim <- victim;
              self.here = false;
              self.parent = parent;
          }

          destroy() {
             if self.here == false {
                self.here = true;
                self.parent.reenter(); // will cause us to re-enter this destructor
             }
             self.parent.collect(from: <- self.victim);
          }
      }

      pub resource OuterResource {
          pub var inner: @InnerResource?;
          pub var collector: &Vault;
          init(victim: @Vault, collector: &Vault) {
              self.collector = collector;
              self.inner <- create InnerResource(victim: <- victim, parent: &self as &OuterResource);
          }
          pub fun reenter() {
              let inner <- self.inner <- nil;
              destroy inner;
          }
          pub fun collect(from: @Vault) {
              self.collector.deposit(from: <- from);
          }

          destroy() {
             destroy self.inner;
          }
      }

      pub fun doubleBalanceOfVault(vault: @Vault): @Vault {
          var collector <- vault.withdraw(amount: 0.0);
          var r <- create OuterResource(victim: <- vault, collector: &collector as &Vault);
          destroy r;
          return <- collector;
      }

      // --- end of vuln code ---

      pub fun main(): UFix64 {
              var v1 <- create Vault(balance: 1000.0);
              var v2 <- doubleBalanceOfVault(vault: <- v1);
              var v3 <- doubleBalanceOfVault(vault: <- v2);
              let balance = v3.balance
              destroy v3
              return balance
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	var destroyedResourceErr interpreter.DestroyedResourceError
	require.ErrorAs(t, err, &destroyedResourceErr)
}

func TestRuntimeFlowEventTypes(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	script := []byte(`
      pub fun main(): Type? {
          return CompositeType("flow.AccountContractAdded")
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
	}

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

	accountContractAddedType := ExportType(
		stdlib.AccountContractAddedEventType,
		map[sema.TypeID]cadence.Type{},
	)

	require.Equal(t,
		cadence.Optional{
			Value: cadence.TypeValue{
				StaticType: accountContractAddedType,
			},
		},
		result,
	)
}

func TestInvalidatedResourceUse(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
		import VictimContract from %s

		pub contract AttackerContract {

			pub resource AttackerResource {
				pub var vault: @VictimContract.Vault
				pub var firstCopy: @VictimContract.Vault

				init(vault: @VictimContract.Vault) {
					self.vault <- vault
					self.firstCopy <- self.vault.withdraw(amount: 0.0)
				}

				pub fun shenanigans(): UFix64{
					let fullBalance = self.vault.balance

					var withdrawn <- self.vault.withdraw(amount: 0.0)

					// "Rug pull" the vault from under the in-flight
					// withdrawal and deposit it into our "first copy" wallet
					self.vault <-> withdrawn
					self.firstCopy.deposit(from: <- withdrawn)

					// Return the pre-deposit balance for caller to withdraw
					return fullBalance
				}

				pub fun fetchfirstCopy(): @VictimContract.Vault {
					var withdrawn <- self.firstCopy.withdraw(amount: 0.0)
					self.firstCopy <-> withdrawn
					return <- withdrawn
				}

				destroy() {
					destroy self.vault
					destroy self.firstCopy
				}
			}

			pub fun doubleBalanceOfVault(_ victim: @VictimContract.Vault): @VictimContract.Vault {
				var r <- create AttackerResource(vault: <- victim)

				// The magic happens during the execution of the following line of code
				// var withdrawAmmount = r.shenanigans()
				var secondCopy <- r.vault.withdraw(amount: r.shenanigans())

				// Deposit the second copy of the funds as retained by the AttackerResource instance
				secondCopy.deposit(from: <- r.fetchfirstCopy())

				destroy r
				return <- secondCopy
			}

			pub fun attack() {
				var v1 <- VictimContract.faucet()
				var v2<- AttackerContract.doubleBalanceOfVault(<- v1)
				destroy v2
		   }
		}`,
		signerAccount.HexWithPrefix(),
	))

	victim := []byte(`
        pub contract VictimContract {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun faucet(): @VictimContract.Vault {
                return <- create VictimContract.Vault(balance: 5.0)
            }
        }
    `)

	// Deploy Victim

	deployVictim := DeploymentTransaction("VictimContract", victim)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVictim,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("AttackerContract", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import VictimContract from %s
        import AttackerContract from %s

        transaction {
            execute {
                AttackerContract.attack()
            }
        }`,
		signerAccount.HexWithPrefix(),
		signerAccount.HexWithPrefix(),
	))

	signers = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	var destroyedResourceErr interpreter.DestroyedResourceError
	require.ErrorAs(t, err, &destroyedResourceErr)

}

func TestInvalidatedResourceUse2(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
        import VictimContract from %s

        pub contract AttackerContract {

            pub resource InnerResource {
                pub var name: String
                pub var parent: &OuterResource?
                pub var vault: @VictimContract.Vault?

                init(_ name: String) {
                    self.name = name
                    self.parent = nil
                    self.vault <- nil
                }

                pub fun setParent(_ parent: &OuterResource) {
                    self.parent = parent
                }

                pub fun setVault(_ vault: @VictimContract.Vault) {
                    self.vault <-! vault
                }

                destroy() {
                    self.parent!.shenanigans()
                    var vault: @VictimContract.Vault <- self.vault!
                    self.parent!.collect(<- vault)
                }
            }

            pub resource OuterResource {
                pub var inner1: @InnerResource
                pub var inner2: @InnerResource
                pub var collector: &VictimContract.Vault

                init(_ victim: @VictimContract.Vault, _ collector: &VictimContract.Vault) {
                    self.collector = collector
                    var i1 <- create InnerResource("inner1")
                    var i2 <- create InnerResource("inner2")
                    self.inner1 <- i1
                    self.inner2 <- i2
                    self.inner1.setVault(<- victim)
                    self.inner1.setParent(&self as &OuterResource)
                    self.inner2.setParent(&self as &OuterResource)
                }

                pub fun shenanigans() {
                    self.inner1 <-> self.inner2
                }

                pub fun collect(_ from: @VictimContract.Vault) {
                    self.collector.deposit(from: <- from)
                }

                destroy() {
                    destroy self.inner1
                    // inner1 and inner2 got swapped during the above line
                    destroy self.inner2
                }
            }

            pub fun doubleBalanceOfVault(_ vault: @VictimContract.Vault): @VictimContract.Vault {
                var collector <- vault.withdraw(amount: 0.0)
                var outer <- create OuterResource(<- vault, &collector as &VictimContract.Vault)
                destroy outer
                return <- collector
            }

            pub fun attack() {
                var v1 <- VictimContract.faucet()
                var v2 <- AttackerContract.doubleBalanceOfVault(<- v1)
                destroy v2
           }
        }`,
		signerAccount.HexWithPrefix(),
	))

	victim := []byte(`
        pub contract VictimContract {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun faucet(): @VictimContract.Vault {
                return <- create VictimContract.Vault(balance: 5.0)
            }
        }
    `)

	// Deploy Victim

	deployVictim := DeploymentTransaction("VictimContract", victim)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVictim,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("AttackerContract", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import VictimContract from %s
        import AttackerContract from %s

        transaction {
            execute {
                AttackerContract.attack()
            }
        }`,
		signerAccount.HexWithPrefix(),
		signerAccount.HexWithPrefix(),
	))

	signers = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	var destroyedResourceErr interpreter.DestroyedResourceError
	require.ErrorAs(t, err, &destroyedResourceErr)
}

func TestRuntimeInvalidRecursiveTransferViaVariableDeclaration(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.AtreeValidationEnabled = false

	address := common.MustBytesToAddress([]byte{0x1})

	contract := []byte(`
      pub contract Test{

          pub resource Holder{

              pub var vaults: @[AnyResource]

              init(_ vaults: @[AnyResource]){
                  self.vaults <- vaults
              }

              pub fun x(): @[AnyResource] {
                  var x <- self.vaults <- [<-Test.dummy()]
                  return <-x
              }

              destroy() {
                  var t <-  self.vaults[0] <- self.vaults    // here is the problem
                  destroy t
                  Test.account.save(<- self.x(), to: /storage/x42)
              }
          }

          pub fun createHolder(_ vaults: @[AnyResource]): @Holder {
              return <- create Holder(<-vaults)
          }

          pub resource Dummy {}

          pub fun dummy(): @Dummy {
              return <- create Dummy()
          }
      }
    `)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(acct: AuthAccount) {
              var holder <- Test.createHolder(<-[<-Test.dummy(), <-Test.dummy()])
              destroy holder
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy

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

	// Test

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.RecursiveTransferError{})
}

func TestRuntimeInvalidRecursiveTransferViaFunctionArgument(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.AtreeValidationEnabled = false

	address := common.MustBytesToAddress([]byte{0x1})

	contract := []byte(`
      pub contract Test{

          pub resource Holder {

              pub var vaults: @[AnyResource]

              init(_ vaults: @[AnyResource]) {
                  self.vaults <- vaults
              }

              destroy() {
                  self.vaults.append(<-self.vaults)
              }
          }

          pub fun createHolder(_ vaults: @[AnyResource]): @Holder {
              return <- create Holder(<-vaults)
          }

          pub resource Dummy {}

          pub fun dummy(): @Dummy {
              return <- create Dummy()
          }
      }
    `)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(acct: AuthAccount) {
              var holder <- Test.createHolder(<-[<-Test.dummy(), <-Test.dummy()])
              destroy holder
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy

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

	// Test

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.RecursiveTransferError{})
}

func TestRuntimeOptionalReferenceAttack(t *testing.T) {

	t.Parallel()

	script := `
      pub resource Vault {
          pub var balance: UFix64

          init(balance: UFix64) {
              self.balance = balance
          }

          pub fun withdraw(amount: UFix64): @Vault {
              self.balance = self.balance - amount
              return <-create Vault(balance: amount)
          }

          pub fun deposit(from: @Vault) {
              self.balance = self.balance + from.balance
              destroy from
          }
      }

      pub fun empty(): @Vault {
          return <- create Vault(balance: 0.0)
      }

      pub fun giveme(): @Vault {
          return <- create Vault(balance: 10.0)
      }

      pub fun main() {
          var vault <- giveme() //get 10 token
          var someDict:@{Int:Vault} <- {1:<-vault}
          var r = (&someDict[1] as auth &AnyResource) as! &Vault
          var double <- empty()
          double.deposit(from: <- someDict.remove(key:1)!)
          double.deposit(from: <- r.withdraw(amount:10.0))
          log(double.balance) // 20
          destroy double
          destroy someDict
      }
    `

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(s string) {

		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(nil, b)
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source:    []byte(script),
			Arguments: [][]byte{},
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checker.RequireCheckerErrors(t, checkerErr, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestRuntimeReturnDestroyedOptional(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      pub resource Foo {}

      pub fun main(): AnyStruct {
          let y: @Foo? <- create Foo()
          let z: @AnyResource <- y
          var ref = &z as &AnyResource
          destroy z
          return ref
      }
    `)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
	}

	// Test

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}

func TestRuntimeComputationMeteringError(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	t.Run("regular error returned", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            pub fun main() {
                foo()
            }
        `)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			meterComputation: func(compKind common.ComputationKind, intensity uint) error {
				return fmt.Errorf("computation limit exceeded")
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an external error.
		// It can NOT be an internal error.
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("regular error panicked", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            pub fun main() {
                foo()
            }
        `)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			meterComputation: func(compKind common.ComputationKind, intensity uint) error {
				panic(fmt.Errorf("computation limit exceeded"))
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an external error.
		// It can NOT be an internal error.
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("go runtime error panicked", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            pub fun main() {
                foo()
            }
        `)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			meterComputation: func(compKind common.ComputationKind, intensity uint) error {
				// Cause a runtime error
				var x any = "hello"
				_ = x.(int)
				return nil
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an internal error.
		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("go runtime error returned", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            pub fun main() {
                foo()
            }
        `)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			meterComputation: func(compKind common.ComputationKind, intensity uint) (err error) {
				// Cause a runtime error. Catch it and return.
				var x any = "hello"
				defer func() {
					if r := recover(); r != nil {
						if r, ok := r.(error); ok {
							err = r
						}
					}
				}()

				_ = x.(int)

				return
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an internal error.
		assertRuntimeErrorIsInternalError(t, err)
	})
}

func TestRuntimeWrappedErrorHandling(t *testing.T) {

	t.Parallel()

	foo := []byte(`
        access(all) contract Foo {
            access(all) resource R {
                access(all) var x: Int

                init() {
                    self.x = 0
                }
            }

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `)

	brokenFoo := []byte(`
        access(all) contract Foo {
            access(all) resource R {
                access(all) var x: Int

                init() {
                    self.x = "hello"
                }
            }

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `)

	tx1 := []byte(`
        import Foo from 0x1

        transaction {
            prepare(signer: AuthAccount) {
                signer.save(<- Foo.createR(), to: /storage/r)
                signer.link<&Foo.R>(/private/r, target:/storage/r)
            }
        }
    `)

	tx2 := []byte(`
        transaction {
            prepare(signer: AuthAccount) {
                var cap = signer.getCapability<&AnyStruct>(/private/r)
				cap.check()
            }
        }
    `)

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.AtreeValidationEnabled = false

	address := common.MustBytesToAddress([]byte{0x1})

	deploy := DeploymentTransaction("Foo", foo)

	var contractCode []byte
	var events []cadence.Event

	isContractBroken := false

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return contractCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			if isContractBroken && contractCode != nil {
				return brokenFoo, nil
			}
			return contractCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			contractCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		getAndSetProgram: func(location Location, load func() (*interpreter.Program, error)) (*interpreter.Program, error) {
			program, err := load()
			if err == nil {
				return program, nil
			}
			return program, fmt.Errorf("wrapped error: %w", err)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy

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

	// Run Tx to save values

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Run Tx to load value.
	// Mark the contract is broken

	isContractBroken = true

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	// Returned error MUST be a user error.
	// It can NOT be an internal error.
	assertRuntimeErrorIsUserError(t, err)
}

func TestRuntimeInvalidWrappedPrivateCapability(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.AtreeValidationEnabled = false
	runtime.defaultConfig.AccountLinkingEnabled = true

	address := common.MustBytesToAddress([]byte{0x1})

	linkTx := []byte(`
      #allowAccountLinking

      transaction {
          prepare(acct: AuthAccount) {
              acct.linkAccount(/private/foo)
          }
      }
    `)

	helperContract := []byte(`
      pub contract Foo {
          pub struct Thing {
              pub let cap: Capability

              init(cap: Capability) {
                  self.cap = cap
              }
          }
      }
    `)

	getCapScript := []byte(`
      import Foo from 0x1

      pub fun main(addr: Address): AnyStruct {
          let acct = getAuthAccount(addr)
          let authCap = acct.getCapability<&AuthAccount>(/private/foo)
          return Foo.Thing(cap: authCap)
      }
	`)

	attackTx := []byte(`
      import Foo from 0x1

      transaction(thing: Foo.Thing) {
          prepare(_: AuthAccount) {
		      thing.cap.borrow<&AuthAccount>()!.save("Hello, World", to: /storage/attack)
          }
      }
    `)

	deploy := DeploymentTransaction("Foo", helperContract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return jsoncdc.Decode(nil, b)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()
	nextScriptLocation := newScriptLocationGenerator()

	// Deploy

	err := runtime.ExecuteTransaction(
		Script{
			Source: linkTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy helper contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Get Cap

	cap, err := runtime.ExecuteScript(
		Script{
			Source: getCapScript,
			Arguments: encodeArgs([]cadence.Value{
				cadence.BytesToAddress([]byte{0x1}),
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTx,
			Arguments: encodeArgs([]cadence.Value{
				cap,
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	var argumentNotImportableErr *ArgumentNotImportableError
	require.ErrorAs(t, err, &argumentNotImportableErr)
}

func TestNestedResourceMoveInDestructor(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {
            pub var temp: @Bar.Vault?
            init() {
                self.temp <- nil
            }

            pub fun doubler(_ vault: @Bar.Vault): @Bar.Vault {
                destroy  <- create R(<-vault)
                var doubled <- self.temp <- nil
                return <- doubled!
            }

            pub resource R {
                pub var bounty: @Bar.Vault
                pub var dummy: @Bar.Vault

                init(_ v: @Bar.Vault) {
                     self.bounty <- v
                     self.dummy <- Bar.createEmptyVault()
                }

                pub fun swap() {
                    self.bounty <-> self.dummy
                }

                destroy() {
                    // Nested resource is moved here once
                    var bounty <- self.bounty

                    // Nested resource is again moved here. This one should fail.
                    self.swap()

                    var dummy <- self.dummy

                    var r1 = &bounty as &Bar.Vault

                    Foo.account.save(<-dummy, to: /storage/dummy)
                    Foo.account.save(<-bounty, to: /storage/bounty)

                    var dummy2 <- Foo.account.load<@Bar.Vault>(from: /storage/dummy)!
                    var bounty2 <- Foo.account.load<@Bar.Vault>(from: /storage/bounty)!

                    dummy2.deposit(from:<-bounty2)
                    Foo.temp <-! dummy2
                }
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	bar := []byte(`
        pub contract Bar {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun createEmptyVault(): @Bar.Vault {
                return <- create Bar.Vault(balance: 0.0)
            }

            pub fun createVault(balance: UFix64): @Bar.Vault {
                return <- create Bar.Vault(balance: balance)
            }
        }
    `)

	// Deploy Bar

	deployVault := DeploymentTransaction("Bar", bar)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVault,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("Foo", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import Foo from %[1]s
        import Bar from %[1]s

        transaction {
            prepare(acc: AuthAccount) {
                acc.save(<- Bar.createVault(balance: 100.0), to: /storage/vault)!
                var vault = acc.borrow<&Bar.Vault>(from: /storage/vault)!
                var flow <- vault.withdraw(amount: 42.0)

                var doubled <- Foo.doubler(<-flow)

                destroy doubled
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.UseBeforeInitializationError{})
}

func TestRuntimeNestedResourceMoveWithSecondTransferInDestructor(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {
            pub var temp: @Bar.Vault?
            init() {
                self.temp <- nil
            }

            pub fun doubler(_ vault: @Bar.Vault): @Bar.Vault {
                destroy  <- create R(<-vault)
                var doubled <- self.temp <- nil
                return <- doubled!
            }

            pub resource R {
                pub var bounty: @Bar.Vault
                pub var dummy: @Bar.Vault

                init(_ v: @Bar.Vault) {
                     self.bounty <- v
                     self.dummy <- Bar.createEmptyVault()
                }

                pub fun swap() {
                    self.bounty <-> self.dummy
                }

                destroy() {
                    // Nested resource is moved here once
                    var bounty <- self.bounty.withdraw(amount: 0.0) 
                    var throwaway <- bounty <- self.bounty
                    destroy throwaway

                    // Nested resource is again moved here. This one should fail.
                    self.swap()

                    var dummy <- self.dummy

                    var r1 = &bounty as &Bar.Vault

                    Foo.account.save(<-dummy, to: /storage/dummy)
                    Foo.account.save(<-bounty, to: /storage/bounty)

                    var dummy2 <- Foo.account.load<@Bar.Vault>(from: /storage/dummy)!
                    var bounty2 <- Foo.account.load<@Bar.Vault>(from: /storage/bounty)!

                    dummy2.deposit(from:<-bounty2)
                    Foo.temp <-! dummy2
                }
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	bar := []byte(`
        pub contract Bar {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun createEmptyVault(): @Bar.Vault {
                return <- create Bar.Vault(balance: 0.0)
            }

            pub fun createVault(balance: UFix64): @Bar.Vault {
                return <- create Bar.Vault(balance: balance)
            }
        }
    `)

	// Deploy Bar

	deployVault := DeploymentTransaction("Bar", bar)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVault,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("Foo", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import Foo from %[1]s
        import Bar from %[1]s

        transaction {
            prepare(acc: AuthAccount) {
                acc.save(<- Bar.createVault(balance: 100.0), to: /storage/vault)!
                var vault = acc.borrow<&Bar.Vault>(from: /storage/vault)!
                var flow <- vault.withdraw(amount: 42.0)

                var doubled <- Foo.doubler(<-flow)

                destroy doubled
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.UseBeforeInitializationError{})
}

func TestNestedResourceMoveInTransaction(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
		log: func(s string) {
			fmt.Println(s)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	foo := []byte(`
        pub contract Foo {
            pub resource Vault {}

            pub fun createVault(): @Foo.Vault {
                return <- create Foo.Vault()
            }
        }
    `)

	// Deploy Foo

	deployVault := DeploymentTransaction("Foo", foo)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVault,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Transaction

	attackTransaction := []byte(fmt.Sprintf(`
        import Foo from %[1]s

        transaction {

            let vault: @Foo.Vault?

            prepare(acc: AuthAccount) {
                self.vault <- Foo.createVault()
            }

            execute {
                 let vault2 <- self.vault
                 destroy <- vault2
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)
}

func TestRuntimePreconditionDuplication(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {
            pub var temp: @Bar.Vault?

            init() {
                self.temp <- nil
            }

            pub fun doubler(_ vault: @Bar.Vault): @Bar.Vault {
                var r  <- create R(<-vault)
                r.doMagic()
                destroy r
                var doubled <- self.temp <- nil
                return <- doubled!
            }

            pub fun conditionFunction(_ unusedref: &Bar.Vault, _ v : @Bar.Vault, _ ref: &R): Bool {
                // This gets called twice: once from the interface's precondition and the second
                // time from the function's own precondition. We save both copies of the vault
                // to the R object
                if ref.counter == 0 {
                    ref.victim1 <-! v
                    ref.counter = 1
                } else {
                    ref.victim2 <-! v
                }
                return true
            }

            pub resource interface RInterface{
                pub fun funcFromIface( _ v: @Bar.Vault, _ ref: &R): Void {
                    post {
                        Foo.conditionFunction(&v as &Bar.Vault, <- v, ref)
                    }
                }
            }

            pub resource R: RInterface {
                pub var bounty: @Bar.Vault?
                pub(set) var victim1: @Bar.Vault?
                pub(set) var victim2: @Bar.Vault?
                pub(set) var counter: Int

                init(_ v: @Bar.Vault) {
                     self.counter = 0
                     self.bounty <- v
                     self.victim1 <- nil
                     self.victim2 <- nil
                }

                pub fun funcFromIface(_ v: @Bar.Vault, _ ref: &R): Void {
                    post {
                        Foo.conditionFunction(&v as &Bar.Vault, <- v, ref)
                    }
                }

                pub fun doMagic(): Void {
                    var origVault <- self.bounty <- nil
                    self.funcFromIface(<- origVault!, &self as &R)

                    var v1 <- self.victim1 <- nil
                    var v2 <- self.victim2 <- nil

                    // Following moves copied from Supun's test
                    var r1 = &v2 as &Bar.Vault?
                    Foo.account.save(<-v1!, to: /storage/v1)
                    Foo.account.save(<-v2!, to: /storage/v2)
                    var v1Reloaded <- Foo.account.load<@Bar.Vault>(from: /storage/v1)!
                    var v2Reloaded <- Foo.account.load<@Bar.Vault>(from: /storage/v2)!

                    v1Reloaded.deposit(from:<-v2Reloaded)
                    Foo.temp <-! v1Reloaded
                }

                destroy() {
                    destroy self.bounty
                    destroy self.victim1
                    destroy self.victim2
                }
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	bar := []byte(`
        pub contract Bar {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun createEmptyVault(): @Bar.Vault {
                return <- create Bar.Vault(balance: 0.0)
            }

            pub fun createVault(balance: UFix64): @Bar.Vault {
                return <- create Bar.Vault(balance: balance)
            }
        }
    `)

	// Deploy Bar

	deployVault := DeploymentTransaction("Bar", bar)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVault,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("Foo", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checker.RequireCheckerErrors(t, checkerErr, 1)

	assert.IsType(t, &sema.InvalidInterfaceConditionResourceInvalidationError{}, errs[0])
}

func TestRuntimeIfLetElseBranchConfusion(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
		log: func(s string) {
			fmt.Println(s)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
        import Bar from %[1]s

        access(all) contract Foo {
            pub var temp: @Bar.Vault?
            init() {
                self.temp <- nil
            }

            pub fun doubler(_ vault: @Bar.Vault): @Bar.Vault {
                destroy  <- create R(<-vault)
                var doubled <- self.temp <- nil
                return <- doubled!
            }

            pub resource R {
                priv var r: @Bar.Vault?
                priv var r2: @Bar.Vault?

                init(_ v: @Bar.Vault) {
                     self.r <- nil
                     self.r2 <- v
                }

                destroy() {
                    if let dummy <- self.r <- self.r2{
                        // unreachable token destroys to please checker
                        destroy dummy
                        destroy self.r
                    } else {
                        var dummy <- self.r2
                        var bounty <- self.r
                        var r1 = &bounty as &Bar.Vault?
                        Foo.account.save(<-dummy!, to: /storage/dummy)
                        Foo.account.save(<-bounty!, to: /storage/bounty)
                        var dummy2 <- Foo.account.load<@Bar.Vault>(from: /storage/dummy)!
                        var bounty2 <- Foo.account.load<@Bar.Vault>(from: /storage/bounty)!

                        dummy2.deposit(from:<-bounty2)
                        Foo.temp <-! dummy2
                    }
                }
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	bar := []byte(`
        pub contract Bar {
            pub resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                pub var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                pub fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                pub fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            pub fun createEmptyVault(): @Bar.Vault {
                return <- create Bar.Vault(balance: 0.0)
            }

            pub fun createVault(balance: UFix64): @Bar.Vault {
                return <- create Bar.Vault(balance: balance)
            }
        }
    `)

	// Deploy Bar

	deployVault := DeploymentTransaction("Bar", bar)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVault,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("Foo", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import Foo from %[1]s
        import Bar from %[1]s

        transaction {
            prepare(acc: AuthAccount) {
                acc.save(<- Bar.createVault(balance: 100.0), to: /storage/vault)!
                var vault = acc.borrow<&Bar.Vault>(from: /storage/vault)!
                var flow <- vault.withdraw(amount: 42.0)

                var doubled <- Foo.doubler(<-flow)
                log(doubled.balance)
                destroy doubled
            }
        }`,
		signerAccount.HexWithPrefix(),
	))

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}
