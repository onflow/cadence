package test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opentracing/opentracing-go"

	"github.com/onflow/atree"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/sema"
)

func TestExecutingScript(t *testing.T) {

	proxyRuntime := ipc.NewProxyRuntime(
		bridge.NewInterfaceBridge(runtimeInterface),
	)

	t.Run("simple script", func(t *testing.T) {
		start := time.Now()
		value, err := proxyRuntime.ExecuteScript(
			runtime.Script{
				Source: []byte(`
               pub fun main(): Int {
                 return 4 + 8
               }
            `),
			},
			runtime.Context{
				Location: common.TransactionLocation("0x01"),
			},
		)

		fmt.Println(time.Since(start))
		require.NoError(t, err)

		assert.Equal(t, cadence.NewInt(12), value)
	})

	t.Run("with imports", func(t *testing.T) {

		// Deploy Fungible Token contract

		err := proxyRuntime.ExecuteTransaction(
			runtime.Script{
				Source: []byte(fmt.Sprintf(
					`
                  transaction {
                      prepare(signer: AuthAccount) {
                          signer.contracts.add(name: "Foo", code: "%s".decodeHex())
                      }
                  }
                `,
					hex.EncodeToString([]byte(`
						pub contract Foo {
							init() { }

							pub fun add(_ a: Int, _ b: Int): Int {
								return a + b
							}
						}`)),
				)),
			},
			runtime.Context{
				Location: common.TransactionLocation("0x01"),
			},
		)
		require.NoError(t, err)

		start := time.Now()
		_, err = proxyRuntime.ExecuteScript(
			runtime.Script{
				Source: []byte(`
               import Foo from 0x01

               pub fun main(): Int {
                 return Foo.add(4, 8)
               }
            `),
			},
			runtime.Context{
				Location: common.ScriptLocation("0x01"),
			},
		)

		fmt.Println(time.Since(start))
		require.NoError(t, err)
	})
}

func TestExecutingScriptParallel(t *testing.T) {

	proxyRuntime := ipc.NewProxyRuntime(
		bridge.NewInterfaceBridge(runtimeInterface),
	)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		go func() {
			wg.Add(1)
			start := time.Now()
			_, err := proxyRuntime.ExecuteScript(
				runtime.Script{
					Source: []byte(`
               pub fun main(): Int {
                 log("hello")
                 return 4 + 8
               }
            `),
				},
				runtime.Context{
					Location: common.ScriptLocation("0x01"),
				},
			)

			fmt.Println(time.Since(start))

			assert.NoError(t, err)

			wg.Done()
		}()
	}

	wg.Wait()
}

var runtimeInterface = func() *testRuntimeInterface {
	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	accountCodes := map[common.LocationID][]byte{}
	signerAccount := contractsAddress

	return &testRuntimeInterface{
		getCode: func(location runtime.Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{signerAccount}, nil
		},
		resolveLocation: func(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error) {
			return []runtime.ResolvedLocation{
				{
					Location: common.AddressLocation{
						Address: location.(common.AddressLocation).Address,
						Name:    identifiers[0].Identifier,
					},
					Identifiers: identifiers,
				},
			}, nil
		},
		getAccountContractCode: func(address runtime.Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address runtime.Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(b)
		},
	}
}()

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

type testRuntimeInterface struct {
	resolveLocation           func(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error)
	getCode                   func(_ runtime.Location) ([]byte, error)
	getProgram                func(runtime.Location) (*interpreter.Program, error)
	setProgram                func(runtime.Location, *interpreter.Program) error
	storage                   testLedger
	createAccount             func(payer runtime.Address) (address runtime.Address, err error)
	addEncodedAccountKey      func(address runtime.Address, publicKey []byte) error
	removeEncodedAccountKey   func(address runtime.Address, index int) (publicKey []byte, err error)
	addAccountKey             func(address runtime.Address, publicKey *runtime.PublicKey, hashAlgo runtime.HashAlgorithm, weight int) (*runtime.AccountKey, error)
	getAccountKey             func(address runtime.Address, index int) (*runtime.AccountKey, error)
	removeAccountKey          func(address runtime.Address, index int) (*runtime.AccountKey, error)
	updateAccountContractCode func(address runtime.Address, name string, code []byte) error
	getAccountContractCode    func(address runtime.Address, name string) (code []byte, err error)
	removeAccountContractCode func(address runtime.Address, name string) (err error)
	getSigningAccounts        func() ([]runtime.Address, error)
	log                       func(string)
	emitEvent                 func(cadence.Event) error
	resourceOwnerChanged      func(
		resource *interpreter.CompositeValue,
		oldAddress common.Address,
		newAddress common.Address,
	)
	generateUUID       func() (uint64, error)
	computationLimit   uint64
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
		signatureAlgorithm runtime.SignatureAlgorithm,
		hashAlgorithm runtime.HashAlgorithm,
	) (bool, error)
	hash                       func(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error)
	setCadenceValue            func(owner runtime.Address, key string, value cadence.Value) (err error)
	getAccountBalance          func(_ runtime.Address) (uint64, error)
	getAccountAvailableBalance func(_ runtime.Address) (uint64, error)
	getStorageUsed             func(_ runtime.Address) (uint64, error)
	getStorageCapacity         func(_ runtime.Address) (uint64, error)
	programs                   map[common.LocationID]*interpreter.Program
	implementationDebugLog     func(message string) error
	validatePublicKey          func(publicKey *runtime.PublicKey) (bool, error)
	bLSVerifyPOP               func(pk *runtime.PublicKey, s []byte) (bool, error)
	aggregateBLSSignatures     func(sigs [][]byte) ([]byte, error)
	aggregateBLSPublicKeys     func(keys []*runtime.PublicKey) (*runtime.PublicKey, error)
	getAccountContractNames    func(address runtime.Address) ([]string, error)
	recordTrace                func(operation string, location common.Location, duration time.Duration, logs []opentracing.LogRecord)
}

// testRuntimeInterface should implement Interface
var _ runtime.Interface = &testRuntimeInterface{}

func (i *testRuntimeInterface) ResolveLocation(identifiers []runtime.Identifier, location runtime.Location) ([]runtime.ResolvedLocation, error) {
	if i.resolveLocation == nil {
		return []runtime.ResolvedLocation{
			{
				Location:    location,
				Identifiers: identifiers,
			},
		}, nil
	}
	return i.resolveLocation(identifiers, location)
}

func (i *testRuntimeInterface) GetCode(location runtime.Location) ([]byte, error) {
	if i.getCode == nil {
		return nil, nil
	}
	return i.getCode(location)
}

func (i *testRuntimeInterface) GetProgram(location runtime.Location) (*interpreter.Program, error) {
	if i.getProgram == nil {
		if i.programs == nil {
			i.programs = map[common.LocationID]*interpreter.Program{}
		}
		return i.programs[location.ID()], nil
	}

	return i.getProgram(location)
}

func (i *testRuntimeInterface) SetProgram(location runtime.Location, program *interpreter.Program) error {
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
	return i.storage.ValueExists(owner, key)
}

func (i *testRuntimeInterface) GetValue(owner, key []byte) (value []byte, err error) {
	return i.storage.GetValue(owner, key)
}

func (i *testRuntimeInterface) SetValue(owner, key, value []byte) (err error) {
	return i.storage.SetValue(owner, key, value)
}

func (i *testRuntimeInterface) AllocateStorageIndex(owner []byte) (atree.StorageIndex, error) {
	return i.storage.AllocateStorageIndex(owner)
}

func (i *testRuntimeInterface) CreateAccount(payer runtime.Address) (address runtime.Address, err error) {
	return i.createAccount(payer)
}

func (i *testRuntimeInterface) AddEncodedAccountKey(address runtime.Address, publicKey []byte) error {
	return i.addEncodedAccountKey(address, publicKey)
}

func (i *testRuntimeInterface) RevokeEncodedAccountKey(address runtime.Address, index int) ([]byte, error) {
	return i.removeEncodedAccountKey(address, index)
}

func (i *testRuntimeInterface) AddAccountKey(address runtime.Address, publicKey *runtime.PublicKey, hashAlgo runtime.HashAlgorithm, weight int) (*runtime.AccountKey, error) {
	return i.addAccountKey(address, publicKey, hashAlgo, weight)
}

func (i *testRuntimeInterface) GetAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	return i.getAccountKey(address, index)
}

func (i *testRuntimeInterface) RevokeAccountKey(address runtime.Address, index int) (*runtime.AccountKey, error) {
	return i.removeAccountKey(address, index)
}

func (i *testRuntimeInterface) UpdateAccountContractCode(address runtime.Address, name string, code []byte) (err error) {
	return i.updateAccountContractCode(address, name, code)
}

func (i *testRuntimeInterface) GetAccountContractCode(address runtime.Address, name string) (code []byte, err error) {
	return i.getAccountContractCode(address, name)
}

func (i *testRuntimeInterface) RemoveAccountContractCode(address runtime.Address, name string) (err error) {
	return i.removeAccountContractCode(address, name)
}

func (i *testRuntimeInterface) GetSigningAccounts() ([]runtime.Address, error) {
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
	resource *interpreter.CompositeValue,
	oldOwner common.Address,
	newOwner common.Address,
) {
	if i.resourceOwnerChanged != nil {
		i.resourceOwnerChanged(resource, oldOwner, newOwner)
	}
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

func (i *testRuntimeInterface) GetCurrentBlockHeight() (uint64, error) {
	return 1, nil
}

func (i *testRuntimeInterface) GetBlockAtHeight(height uint64) (block runtime.Block, exists bool, err error) {

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, height)
	if err != nil {
		panic(err)
	}

	encoded := buf.Bytes()
	var hash runtime.BlockHash
	copy(hash[sema.BlockIDSize-len(encoded):], encoded)

	block = runtime.Block{
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
	signatureAlgorithm runtime.SignatureAlgorithm,
	hashAlgorithm runtime.HashAlgorithm,
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

func (i *testRuntimeInterface) Hash(data []byte, tag string, hashAlgorithm runtime.HashAlgorithm) ([]byte, error) {
	if i.hash == nil {
		return nil, nil
	}
	return i.hash(data, tag, hashAlgorithm)
}

func (i *testRuntimeInterface) SetCadenceValue(owner common.Address, key string, value cadence.Value) (err error) {
	return i.setCadenceValue(owner, key, value)
}

func (i *testRuntimeInterface) GetAccountBalance(address runtime.Address) (uint64, error) {
	return i.getAccountBalance(address)
}

func (i *testRuntimeInterface) GetAccountAvailableBalance(address runtime.Address) (uint64, error) {
	return i.getAccountAvailableBalance(address)
}

func (i *testRuntimeInterface) GetStorageUsed(address runtime.Address) (uint64, error) {
	return i.getStorageUsed(address)
}

func (i *testRuntimeInterface) GetStorageCapacity(address runtime.Address) (uint64, error) {
	return i.getStorageCapacity(address)
}

func (i *testRuntimeInterface) ImplementationDebugLog(message string) error {
	if i.implementationDebugLog == nil {
		return nil
	}
	return i.implementationDebugLog(message)
}

func (i *testRuntimeInterface) ValidatePublicKey(key *runtime.PublicKey) (bool, error) {
	if i.validatePublicKey == nil {
		return false, nil
	}

	return i.validatePublicKey(key)
}

func (i *testRuntimeInterface) BLSVerifyPOP(key *runtime.PublicKey, s []byte) (bool, error) {
	if i.bLSVerifyPOP == nil {
		return false, nil
	}

	return i.bLSVerifyPOP(key, s)
}

func (i *testRuntimeInterface) AggregateBLSSignatures(sigs [][]byte) ([]byte, error) {
	if i.aggregateBLSSignatures == nil {
		return []byte{}, nil
	}

	return i.aggregateBLSSignatures(sigs)
}

func (i *testRuntimeInterface) AggregateBLSPublicKeys(keys []*runtime.PublicKey) (*runtime.PublicKey, error) {
	if i.aggregateBLSPublicKeys == nil {
		return nil, nil
	}

	return i.aggregateBLSPublicKeys(keys)
}

func (i *testRuntimeInterface) GetAccountContractNames(address runtime.Address) ([]string, error) {
	if i.getAccountContractNames == nil {
		return []string{}, nil
	}

	return i.getAccountContractNames(address)
}

func (i *testRuntimeInterface) RecordTrace(operation string, location common.Location, duration time.Duration, logs []opentracing.LogRecord) {
	if i.recordTrace == nil {
		return
	}
	i.recordTrace(operation, location, duration, logs)
}
