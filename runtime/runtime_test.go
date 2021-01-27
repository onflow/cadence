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
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

// import (
// 	"encoding/hex"
// 	"errors"
// 	"fmt"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/hashicorp/go-multierror"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/onflow/cadence"
// 	jsoncdc "github.com/onflow/cadence/encoding/json"
// 	"github.com/onflow/cadence/runtime/ast"
// 	"github.com/onflow/cadence/runtime/common"
// 	"github.com/onflow/cadence/runtime/interpreter"
// 	"github.com/onflow/cadence/runtime/sema"
// 	"github.com/onflow/cadence/runtime/stdlib"
// 	"github.com/onflow/cadence/runtime/tests/checker"
// 	"github.com/onflow/cadence/runtime/tests/utils"
// )

type Script struct {
	source      []byte
	arguments   []cadence.Value
	authorizers []Address
}

func NewScript(source []byte, arguments []cadence.Value, authorizers []Address) *Script {
	return &Script{source: source,
		arguments:   arguments,
		authorizers: authorizers}
}

func (s *Script) Source() []byte {
	return s.source
}

func (s *Script) Arguments(argumentTypes []cadence.Type) ([]cadence.Value, error) {
	return s.arguments, nil
}

func (s *Script) Authorizers() []Address {
	return s.authorizers
}

type testAccountsInterface struct {
	newAccount         func() (address Address, err error)
	accountExists      func(address Address) (exists bool, err error)
	numberOfAccounts   func() (count uint64, err error)
	suspendAccount     func(address Address) error
	unsuspendAccount   func(address Address) error
	isAccountSuspended func(address Address) (isSuspended bool, err error)
}

var _ Accounts = &testAccountsInterface{}

func (i *testAccountsInterface) NewAccount() (Address, error) {
	if i.newAccount == nil {
		return Address{}, nil
	}
	return i.newAccount()
}

func (i *testAccountsInterface) AccountExists(address Address) (bool, error) {
	if i.accountExists == nil {
		return false, nil
	}
	return i.accountExists(address)
}

func (i *testAccountsInterface) NumberOfAccounts() (uint64, error) {
	if i.numberOfAccounts == nil {
		return 0, nil
	}
	return i.numberOfAccounts()
}

func (i *testAccountsInterface) SuspendAccount(address Address) error {
	if i.suspendAccount == nil {
		return nil
	}
	return i.suspendAccount(address)
}

func (i *testAccountsInterface) UnsuspendAccount(address Address) error {
	if i.unsuspendAccount == nil {
		return nil
	}
	return i.unsuspendAccount(address)
}

func (i *testAccountsInterface) IsAccountSuspended(address Address) (bool, error) {
	if i.isAccountSuspended == nil {
		return false, nil
	}
	return i.isAccountSuspended(address)
}

type testAccountContractsInterface struct {
	contractCode       func(address AddressLocation) (code []byte, err error)
	updateContractCode func(address AddressLocation, code []byte) (err error)
	removeContractCode func(address AddressLocation) (err error)
	contracts          func(address AddressLocation) (name []string, err error)
}

var _ AccountContracts = &testAccountContractsInterface{}

func (i *testAccountContractsInterface) ContractCode(address AddressLocation) ([]byte, error) {
	if i.contractCode == nil {
		return nil, nil
	}
	return i.contractCode(address)
}

func (i *testAccountContractsInterface) UpdateContractCode(address AddressLocation, code []byte) (err error) {
	if i.updateContractCode == nil {
		return nil
	}
	return i.updateContractCode(address, code)
}

func (i *testAccountContractsInterface) RemoveContractCode(address AddressLocation) (err error) {
	if i.removeContractCode == nil {
		return nil
	}
	return i.removeContractCode(address)
}

func (i *testAccountContractsInterface) Contracts(address AddressLocation) (name []string, err error) {
	if i.contracts == nil {
		return nil, nil
	}
	return i.contracts(address)
}

type testAccountStorageInterface struct {
	storedValues map[string]StorageValue
	getValue     func(key StorageKey) (value StorageValue, err error)
	setValue     func(key StorageKey, value StorageValue) (err error)
	valueExists  func(key StorageKey) (exists bool, err error)
	storedKeys   func(address Address) (iter StorageKeyIterator, err error)
	storageUsed  func(address Address) (value uint64, err error)
}

func newTestStorage(
	onRead func(key StorageKey, value StorageValue),
	onWrite func(key StorageKey, value StorageValue),
) *testAccountStorageInterface {

	storedValues := map[string]StorageValue{}

	return &testAccountStorageInterface{
		storedValues: storedValues,
		valueExists: func(key StorageKey) (bool, error) {
			value := storedValues[key.String()]
			return len(value) > 0, nil
		},
		getValue: func(key StorageKey) (value StorageValue, err error) {
			value = storedValues[key.String()]
			if onRead != nil {
				onRead(key, value)
			}
			return value, nil
		},
		setValue: func(key StorageKey, value StorageValue) (err error) {
			storedValues[key.String()] = value
			if onWrite != nil {
				onWrite(key, value)
			}
			return nil
		},
	}
}

var _ AccountStorage = &testAccountStorageInterface{}

func (i *testAccountStorageInterface) GetValue(key StorageKey) (StorageValue, error) {
	if i.getValue == nil {
		return nil, nil
	}
	return i.getValue(key)
}

func (i *testAccountStorageInterface) SetValue(key StorageKey, value StorageValue) error {
	if i.setValue == nil {
		return nil
	}
	return i.setValue(key, value)

}

func (i *testAccountStorageInterface) ValueExists(key StorageKey) (exists bool, err error) {
	if i.valueExists == nil {
		return false, nil
	}
	return i.valueExists(key)
}

func (i *testAccountStorageInterface) StorageUsed(address Address) (value uint64, err error) {
	if i.storageUsed == nil {
		return 0, nil
	}
	return i.storageUsed(address)
}

func (i *testAccountStorageInterface) StoredKeys(address Address) (StorageKeyIterator, error) {
	if i.storedKeys == nil {
		return nil, nil
	}
	return i.storedKeys(address)
}

type testAccountKeysInterface struct {
	addAccountKey    func(address Address, publicKey []byte) error
	revokeAccountKey func(address Address, index int) (publicKey []byte, err error)
	accountPublicKey func(address Address, index int) (publicKey []byte, err error)
}

var _ AccountKeys = &testAccountKeysInterface{}

func (i *testAccountKeysInterface) AddAccountKey(address Address, publicKey []byte) error {
	if i.addAccountKey == nil {
		return nil
	}
	return i.addAccountKey(address, publicKey)
}

func (i *testAccountKeysInterface) RevokeAccountKey(address Address, index int) ([]byte, error) {
	if i.revokeAccountKey == nil {
		return nil, nil
	}
	return i.revokeAccountKey(address, index)
}

func (i *testAccountKeysInterface) AccountPublicKey(address Address, index int) ([]byte, error) {
	if i.accountPublicKey == nil {
		return nil, nil
	}
	return i.accountPublicKey(address, index)
}

type testLocationResolverInterface struct {
	getCode         func(location Location) ([]byte, error)
	resolveLocation func(identifiers []Identifier, location Location) ([]ResolvedLocation, error)
}

var _ LocationResolver = &testLocationResolverInterface{}

func (i *testLocationResolverInterface) GetCode(location Location) ([]byte, error) {
	if i.getCode == nil {
		return nil, nil
	}
	return i.getCode(location)
}

func (i *testLocationResolverInterface) ResolveLocation(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
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

type testProgramCacheInterface struct {
	getCachedProgram func(Location) (*ast.Program, error)
	cacheProgram     func(Location, *ast.Program) error
}

var _ ProgramCache = &testProgramCacheInterface{}

func (i *testProgramCacheInterface) GetCachedProgram(l Location) (*ast.Program, error) {
	if i.getCachedProgram == nil {
		return nil, nil
	}
	return i.getCachedProgram(l)
}

func (i *testProgramCacheInterface) CacheProgram(l Location, p *ast.Program) error {
	if i.cacheProgram == nil {
		return nil
	}
	return i.cacheProgram(l, p)
}

type testUtilsInterface struct {
	generateUUID func() (uint64, error)
}

var _ ProgramCache = &testProgramCacheInterface{}

func (i *testUtilsInterface) GenerateUUID() (uint64, error) {
	if i.generateUUID == nil {
		return 0, nil
	}
	return i.generateUUID()
}

type testResultsInterface struct {
	appendLog          func(log string) error
	logs               func() ([]string, error)
	logAt              func(index uint) (string, error)
	logCount           func() uint
	appendEvent        func(event cadence.Event) error
	events             func() ([]cadence.Event, error)
	eventAt            func(index uint) (cadence.Event, error)
	eventCount         func() uint
	appendError        func(err error) error
	errors             func() multierror.Error
	errorAt            func(index uint) (Error, error)
	errorCount         func() uint
	addComputationUsed func(c uint64) error
	computationSpent   func() uint64
	computationLimit   func() uint64
}

var _ Results = &testResultsInterface{}

func (i *testResultsInterface) AppendLog(log string) error {
	if i.appendLog == nil {
		return nil
	}
	return i.appendLog(log)
}

func (i *testResultsInterface) Logs() ([]string, error) {
	if i.logs == nil {
		return nil, nil
	}
	return i.logs()
}

func (i *testResultsInterface) LogAt(index uint) (string, error) {
	if i.logAt == nil {
		return "", nil
	}
	return i.logAt(index)
}

func (i *testResultsInterface) LogCount() uint {
	if i.logCount == nil {
		return 0
	}
	return i.logCount()
}

func (i *testResultsInterface) AppendEvent(event cadence.Event) error {
	if i.appendEvent == nil {
		return nil
	}
	return i.appendEvent(event)
}

func (i *testResultsInterface) Events() ([]cadence.Event, error) {
	if i.events == nil {
		return nil, nil
	}
	return i.events()
}

func (i *testResultsInterface) EventAt(index uint) (cadence.Event, error) {
	if i.eventAt == nil {
		return cadence.Event{}, nil
	}
	return i.eventAt(index)
}

func (i *testResultsInterface) EventCount() uint {
	if i.eventCount == nil {
		return 0
	}
	return i.eventCount()
}

func (i *testResultsInterface) AppendError(err error) error {
	if i.appendError == nil {
		return nil
	}
	return i.appendError(err)

}

func (i *testResultsInterface) Errors() multierror.Error {
	if i.errors == nil {
		return multierror.Error{}
	}
	return i.errors()
}

func (i *testResultsInterface) ErrorAt(index uint) (Error, error) {
	if i.errorAt == nil {
		return Error{}, nil
	}
	return i.errorAt(index)
}

func (i *testResultsInterface) ErrorCount() uint {
	if i.errorCount == nil {
		return 0
	}
	return i.errorCount()
}

func (i *testResultsInterface) AddComputationUsed(c uint64) error {
	if i.addComputationUsed == nil {
		return nil
	}
	return i.addComputationUsed(c)
}

func (i *testResultsInterface) ComputationSpent() uint64 {
	if i.computationSpent == nil {
		return 0
	}
	return i.computationSpent()
}

func (i *testResultsInterface) ComputationLimit() uint64 {
	if i.computationLimit == nil {
		return 0
	}
	return i.computationLimit()
}

type testCryptoProviderInterface struct {
	verifySignature func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm string,
		hashAlgorithm string,
	) (bool, error)
	hash func(data []byte, hashAlgorithm string) ([]byte, error)
}

var _ CryptoProvider = &testCryptoProviderInterface{}

func (i *testCryptoProviderInterface) VerifySignature(
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

func (i *testCryptoProviderInterface) Hash(data []byte, hashAlgorithm string) ([]byte, error) {
	if i.hash == nil {
		return nil, nil
	}
	return i.hash(data, hashAlgorithm)
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	value, err := runtime.RunScript(
		NewScript(script, nil, nil),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cadence.NewInt(42), value)
}

func TestRuntimeProgramCache(t *testing.T) {

	t.Parallel()

	progCache := map[common.LocationID]*ast.Program{}
	cacheHits := make(map[common.LocationID]bool)

	importedScript := []byte(`
	transaction {
		prepare() {}
		execute {}
	}
	`)
	importedScriptLocation := common.StringLocation("imported")

	runtime := NewInterpreterRuntime()

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) ([]byte, error) {
			switch location {
			case importedScriptLocation:
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{
		getCachedProgram: func(location common.Location) (*ast.Program, error) {
			program, found := progCache[location.ID()]
			cacheHits[location.ID()] = found
			if !found {
				return nil, nil
			}
			return program, nil
		},
		cacheProgram: func(location common.Location, program *ast.Program) error {
			progCache[location.ID()] = program
			return nil
		},
	}
	results := &testResultsInterface{}

	t.Run("empty cache, cache miss", func(t *testing.T) {

		script := []byte(`
		import "imported"

		transaction {
			prepare() {}
			execute {}
		}
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Initial call, should parse script, store result in cache.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         accounts,
				AccountContracts: accountContracts,
				AccountStorage:   accountStorage,
				AccountKeys:      accountKeys,
				LocationResolver: locationResolver,
				ProgramCache:     programCache,
				Results:          results,
				Location:         scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Program was added to cache.
		cachedProgram, exists := progCache[scriptLocation.ID()]
		assert.True(t, exists)
		assert.NotNil(t, cachedProgram)

		// Script was not in cache.
		assert.False(t, cacheHits[scriptLocation.ID()])
	})

	t.Run("program previously parsed, cache hit", func(t *testing.T) {

		script := []byte(`
		import "imported"

		transaction {
			prepare() {}
			execute {}
		}
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Call a second time to hit the cache
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         accounts,
				AccountContracts: accountContracts,
				AccountStorage:   accountStorage,
				AccountKeys:      accountKeys,
				LocationResolver: locationResolver,
				ProgramCache:     programCache,
				Results:          results,
				Location:         scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Script was in cache.
		assert.True(t, cacheHits[scriptLocation.ID()])
	})

	t.Run("imported program previously parsed, cache hit", func(t *testing.T) {

		script := []byte(`
		import "imported"

		transaction {
			prepare() {}
			execute {}
		}
		`)
		scriptLocation := common.StringLocation("placeholder")

		// Call a second time to hit the cache
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         accounts,
				AccountContracts: accountContracts,
				AccountStorage:   accountStorage,
				AccountKeys:      accountKeys,
				LocationResolver: locationResolver,
				ProgramCache:     programCache,
				Results:          results,
				Location:         scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Script was in cache.
		assert.True(t, cacheHits[scriptLocation.ID()])
		// Import was in cache.
		assert.True(t, cacheHits[importedScriptLocation.ID()])
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript(script, nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessage = message
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript(script, nil, []Address{common.BytesToAddress([]byte{42})}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "0x2a", loggedMessage)
}

func TestRuntimeTransactionWithArguments(t *testing.T) {

	t.Parallel()

	var tests = []struct {
		label        string
		script       string
		args         []cadence.Value
		authorizers  []Address
		expectedLogs []string
		check        func(t *testing.T, err error)
	}{
		{
			label: "Single argument",
			script: `
			  transaction(x: Int) {
				execute {
				  log(x)
				}
			  }
			`,
			args: []cadence.Value{
				cadence.NewInt(42),
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
			args: []cadence.Value{
				cadence.NewInt(42),
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
			args: []cadence.Value{
				cadence.NewInt(42),
				cadence.NewString("foo"),
			},
			expectedLogs: []string{"42", `"foo"`},
		},
		// TODO RAMTIN uncomment me
		// {
		// 	label: "Invalid bytes",
		// 	script: `
		// 	  transaction(x: Int) { execute {} }
		// 	`,
		// 	args: []cadence.Value{
		// 		{1, 2, 3, 4}, // not valid JSON-CDC
		// 	},
		// 	check: func(t *testing.T, err error) {
		// 		assert.Error(t, err)
		// 		assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
		// 	},
		// },
		{
			label: "Type mismatch",
			script: `
			  transaction(x: Int) {
				execute {
				  log(x)
				}
			  }
			`,
			args: []cadence.Value{
				cadence.NewString("foo"),
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
			args: []cadence.Value{
				cadence.BytesToAddress(
					[]byte{
						0x0, 0x0, 0x0, 0x0,
						0x0, 0x0, 0x0, 0x1,
					},
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
			args: []cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
						cadence.NewInt(2),
						cadence.NewInt(3),
					},
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
			args: []cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.NewString("y"),
							Value: cadence.NewInt(42),
						},
					},
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
			args: []cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.NewString("y"),
							Value: cadence.NewInt(42),
						},
					},
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
			args: []cadence.Value{
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
			args: []cadence.Value{
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
			},
			expectedLogs: []string{`"bar"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			rt := NewInterpreterRuntime()

			var loggedMessages []string

			accounts := &testAccountsInterface{}
			accountContracts := &testAccountContractsInterface{}
			accountStorage := &testAccountStorageInterface{}
			accountKeys := &testAccountKeysInterface{}
			locationResolver := &testLocationResolverInterface{}
			programCache := &testProgramCacheInterface{}
			results := &testResultsInterface{appendLog: func(message string) error {
				loggedMessages = append(loggedMessages, message)
				return nil
			}}

			err := rt.RunTransaction(
				NewScript([]byte(tt.script), tt.args, tt.authorizers),
				Context{
					Accounts:         accounts,
					AccountContracts: accountContracts,
					AccountStorage:   accountStorage,
					AccountKeys:      accountKeys,
					LocationResolver: locationResolver,
					ProgramCache:     programCache,
					Results:          results,
					Location:         utils.TestLocation,
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
}

func TestRuntimeScriptArguments(t *testing.T) {

	t.Parallel()

	type testCase struct {
		label        string
		script       string
		args         []cadence.Value
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
			args: []cadence.Value{
				cadence.NewInt(42),
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
			args: []cadence.Value{
				cadence.NewInt(42),
				cadence.NewString("foo"),
			},
			expectedLogs: []string{"42", `"foo"`},
		},
		// TODO RAMTIN uncomment me
		// {
		// 	label: "Invalid bytes",
		// 	script: `
		// 		pub fun main(x: Int) { }
		// 	`,
		// 	args: []cadence.Value{
		// 		{1, 2, 3, 4}, // not valid JSON-CDC
		// 	},
		// 	check: func(t *testing.T, err error) {
		// 		assert.Error(t, err)
		// 		assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
		// 	},
		// },
		{
			label: "Type mismatch",
			script: `
				pub fun main(x: Int) {
					log(x)
				}
			`,
			args: []cadence.Value{
				cadence.NewString("foo"),
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
			args: []cadence.Value{
				cadence.BytesToAddress(
					[]byte{
						0x0, 0x0, 0x0, 0x0,
						0x0, 0x0, 0x0, 0x1,
					},
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
			args: []cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
						cadence.NewInt(2),
						cadence.NewInt(3),
					},
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
			args: []cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.NewString("y"),
							Value: cadence.NewInt(42),
						},
					},
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
			args: []cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.NewString("y"),
							Value: cadence.NewInt(42),
						},
					},
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
			args: []cadence.Value{
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
			args: []cadence.Value{
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
			},
			expectedLogs: []string{`"bar"`},
		},
	}

	test := func(tt testCase) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			rt := NewInterpreterRuntime()

			var loggedMessages []string

			accounts := &testAccountsInterface{}
			accountContracts := &testAccountContractsInterface{}
			accountStorage := &testAccountStorageInterface{}
			accountKeys := &testAccountKeysInterface{}
			locationResolver := &testLocationResolverInterface{}
			programCache := &testProgramCacheInterface{}
			results := &testResultsInterface{appendLog: func(message string) error {
				loggedMessages = append(loggedMessages, message)
				return nil
			}}

			_, err := rt.RunScript(
				NewScript([]byte(tt.script), tt.args, nil),
				Context{
					Accounts:         accounts,
					AccountContracts: accountContracts,
					AccountStorage:   accountStorage,
					AccountKeys:      accountKeys,
					LocationResolver: locationResolver,
					ProgramCache:     programCache,
					Results:          results,
					Location:         utils.TestLocation,
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script), nil, nil),

		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := &testAccountStorageInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script), nil, nil),

		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

			accounts := &testAccountsInterface{}
			accountContracts := &testAccountContractsInterface{}
			accountStorage := newTestStorage(nil, nil)
			utils := &testUtilsInterface{}
			accountKeys := &testAccountKeysInterface{}
			locationResolver := &testLocationResolverInterface{
				getCode: func(location Location) ([]byte, error) {
					switch location {
					case common.StringLocation("imported"):
						return imported, nil
					default:
						return nil, fmt.Errorf("unknown import location: %s", location)
					}
				},
			}
			programCache := &testProgramCacheInterface{}
			results := &testResultsInterface{appendLog: func(message string) error {
				loggedMessages = append(loggedMessages, message)
				return nil
			}}

			nextTransactionLocation := newTransactionLocationGenerator()

			err := runtime.RunTransaction(
				NewScript([]byte(script), nil, []Address{{42}}),

				Context{
					Accounts:         accounts,
					AccountContracts: accountContracts,
					AccountStorage:   accountStorage,
					AccountKeys:      accountKeys,
					LocationResolver: locationResolver,
					ProgramCache:     programCache,
					Results:          results,
					Utils:            utils,
					Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}

	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("container"):
				return container, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script3), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("deep-thought"):
				return deepThought, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Location:         nextTransactionLocation(),
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

	var loggedMessages []string
	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}
	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}
	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
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
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}
	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
			},
		)
		assert.NoError(t, err)
	})

	t.Run("InvalidSyntax", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte("invalid syntax")

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})

	t.Run("InvalidSemantics", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`pub let a: Int = "b"`)

		nextTransactionLocation := newTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})
}

func TestScriptReturnTypeNotReturnableError(t *testing.T) {

	t.Parallel()

	test := func(code string, expected cadence.Value) {

		runtime := NewInterpreterRuntime()
		nextTransactionLocation := newTransactionLocationGenerator()

		actual, err := runtime.RunScript(
			NewScript([]byte(code), nil, []Address{{42}}),
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
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

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := runtime.RunScript(
		NewScript([]byte(script), nil, []Address{{42}}),
		Context{
			Accounts:         &testAccountsInterface{},
			AccountContracts: &testAccountContractsInterface{},
			AccountStorage:   newTestStorage(nil, nil),
			AccountKeys:      &testAccountKeysInterface{},
			LocationResolver: &testLocationResolverInterface{},
			ProgramCache:     &testProgramCacheInterface{},
			Results:          &testResultsInterface{},
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := runtime.RunScript(
		NewScript([]byte(script), nil, []Address{{42}}),
		Context{
			Accounts:         &testAccountsInterface{},
			AccountContracts: &testAccountContractsInterface{},
			AccountStorage:   newTestStorage(nil, nil),
			AccountKeys:      &testAccountKeysInterface{},
			LocationResolver: &testLocationResolverInterface{},
			ProgramCache:     &testProgramCacheInterface{},
			Results:          &testResultsInterface{},
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	var loggedMessages []string
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script), nil, []Address{common.BytesToAddress([]byte{42})}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}
	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script), nil, []Address{}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) ([]byte, error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{appendLog: func(message string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script1), nil, []Address{address}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(script2), nil, []Address{address}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{
		newAccount: func() (address Address, err error) {
			return Address{42}, nil
		},
	}
	accountContracts := &testAccountContractsInterface{}
	accountStorage := newTestStorage(nil, nil)
	utils := &testUtilsInterface{}
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{}
	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}
	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(script), nil, []Address{{42}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            utils,
			Location:         nextTransactionLocation(),
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

		accounts := &testAccountsInterface{
			newAccount: func() (address Address, err error) {
				return Address{42}, nil
			},
		}
		accountContracts := &testAccountContractsInterface{}
		accountStorage := newTestStorage(nil, nil)
		accountKeys := &testAccountKeysInterface{
			addAccountKey: func(address Address, publicKey []byte) error {
				keys = append(keys, publicKey)
				return nil
			},
		}
		locationResolver := &testLocationResolverInterface{}
		programCache := &testProgramCacheInterface{}
		results := &testResultsInterface{
			appendEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}
		t.Run(tt.name, func(t *testing.T) {
			err := runtime.RunTransaction(
				NewScript([]byte(tt.code), tt.args, []Address{{42}}),
				Context{
					Accounts:         accounts,
					AccountContracts: accountContracts,
					AccountStorage:   accountStorage,
					AccountKeys:      accountKeys,
					LocationResolver: locationResolver,
					ProgramCache:     programCache,
					Results:          results,
					Utils:            &testUtilsInterface{},
					Location:         utils.TestLocation,
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(_ AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateContractCode: func(_ AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	locationResolver := &testLocationResolverInterface{
		resolveLocation: singleIdentifierLocationResolver(t),
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
	}

	programCache := &testProgramCacheInterface{}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{common.BytesToAddress(addressValue.Bytes())}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	t.Run("", func(t *testing.T) {
		value, err := runtime.RunScript(
			NewScript([]byte(script1), nil, []Address{common.BytesToAddress(addressValue.Bytes())}),
			Context{
				Accounts:         accounts,
				AccountContracts: accountContracts,
				AccountStorage:   accountStorage,
				AccountKeys:      accountKeys,
				LocationResolver: locationResolver,
				ProgramCache:     programCache,
				Results:          results,
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, addressValue, value)
	})

	t.Run("", func(t *testing.T) {
		value, err := runtime.RunScript(
			NewScript([]byte(script2), nil, []Address{common.BytesToAddress(addressValue.Bytes())}),
			Context{
				Accounts:         accounts,
				AccountContracts: accountContracts,
				AccountStorage:   accountStorage,
				AccountKeys:      accountKeys,
				LocationResolver: locationResolver,
				ProgramCache:     programCache,
				Results:          results,
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(_ AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateContractCode: func(_ AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error { return nil },
		appendLog: func(message string) error {
			loggedMessage = message
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.RunTransaction(
		NewScript([]byte(tx), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(_ AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateContractCode: func(_ AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error { return nil },
		appendLog: func(message string) error {
			loggedMessage = message
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.RunTransaction(
		NewScript([]byte(tx), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(_ AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateContractCode: func(_ AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error { return nil },
		appendLog: func(message string) error {
			loggedMessage = message
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.RunTransaction(
		NewScript([]byte(tx), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(_ AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		updateContractCode: func(_ AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		removeContractCode: func(_ AddressLocation) (err error) {
			accountCode = nil
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error { return nil },
	}
	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy the contract

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	// Remove the contract

	err = runtime.RunTransaction(
		NewScript([]byte(removal), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Nil(t, accountCode)

	// Destroy

	err = runtime.RunTransaction(
		NewScript([]byte(tx), nil, []Address{addressValue}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(location AddressLocation) (code []byte, err error) {
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateContractCode: func(location AddressLocation, code []byte) error {
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(setup1Transaction), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signerAccount = address2Value

	err = runtime.RunTransaction(
		NewScript([]byte(setup2Transaction), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{
		newAccount: func() (address Address, err error) {
			return address2Value, nil
		},
	}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(location AddressLocation) (code []byte, err error) {
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateContractCode: func(location AddressLocation, code []byte) error {
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.RunTransaction(
		NewScript([]byte(deploy), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(setup1Transaction), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(setup2Transaction), nil, []Address{signerAccount}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
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

	accounts := &testAccountsInterface{
		newAccount: func() (address Address, err error) {
			result := interpreter.NewAddressValueFromBytes([]byte{nextAccount})
			nextAccount++
			return result.ToAddress(), nil
		},
	}
	accountContracts := &testAccountContractsInterface{
		contractCode: func(location AddressLocation) (code []byte, err error) {
			key := string(location.ID())
			return accountCodes[key], nil
		},
		updateContractCode: func(location AddressLocation, code []byte) error {
			key := string(location.ID())
			accountCodes[key] = code
			return nil
		},
	}
	accountStorage := newTestStorage(nil, nil)
	accountKeys := &testAccountKeysInterface{}
	programCache := &testProgramCacheInterface{}
	locationResolver := &testLocationResolverInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			key := string(location.(common.AddressLocation).ID())
			return accountCodes[key], nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
	}
	results := &testResultsInterface{
		appendEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	deployTransaction := makeDeployTransaction("TestContractInterface", contractInterfaceCode)

	err := runtime.RunTransaction(
		NewScript([]byte(deployTransaction), nil, []Address{{0x1}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	deployTransaction = makeDeployTransaction("TestContract", contractCode)
	err = runtime.RunTransaction(
		NewScript([]byte(deployTransaction), nil, []Address{{0x1}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.RunTransaction(
		NewScript([]byte(setupCode), nil, []Address{{0x1}}),
		Context{
			Accounts:         accounts,
			AccountContracts: accountContracts,
			AccountStorage:   accountStorage,
			AccountKeys:      accountKeys,
			LocationResolver: locationResolver,
			ProgramCache:     programCache,
			Results:          results,
			Utils:            &testUtilsInterface{},
			Location:         nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	for a := 1; a <= 3; a++ {
		for b := 1; b <= 3; b++ {

			t.Run(fmt.Sprintf("%d/%d", a, b), func(t *testing.T) {

				err = runtime.RunTransaction(
					NewScript(makeUseCode(a, b), nil, []Address{{0x1}}),
					Context{
						Accounts:         accounts,
						AccountContracts: accountContracts,
						AccountStorage:   accountStorage,
						AccountKeys:      accountKeys,
						LocationResolver: locationResolver,
						ProgramCache:     programCache,
						Results:          results,
						Utils:            &testUtilsInterface{},
						Location:         nextTransactionLocation(),
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

// func TestRuntimeBlock(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	script := []byte(`
//       transaction {
//         prepare() {
//           let block = getCurrentBlock()
//           log(block)
//           log(block.height)
//           log(block.view)
//           log(block.id)
//           log(block.timestamp)

//           let nextBlock = getBlock(at: block.height + UInt64(1))
//           log(nextBlock)
//           log(nextBlock?.height)
//           log(nextBlock?.view)
//           log(nextBlock?.id)
//           log(nextBlock?.timestamp)
//         }
//       }
//     `)

// 	var loggedMessages []string

// 	runtimeInterface := &testRuntimeInterface{
// 		getSigningAccounts: func() ([]Address, error) {
// 			return nil, nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: script,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"Block(height: 1, view: 1, id: 0x0000000000000000000000000000000000000000000000000000000000000001, timestamp: 1.00000000)",
// 			"1",
// 			"1",
// 			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1]",
// 			"1.00000000",
// 			"Block(height: 2, view: 2, id: 0x0000000000000000000000000000000000000000000000000000000000000002, timestamp: 2.00000000)",
// 			"2",
// 			"2",
// 			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2]",
// 			"2.00000000",
// 		},
// 		loggedMessages,
// 	)
// }

// func TestUnsafeRandom(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	script := []byte(`
//       transaction {
//         prepare() {
//           let rand = unsafeRandom()
//           log(rand)
//         }
//       }
//     `)

// 	var loggedMessages []string

// 	runtimeInterface := &testRuntimeInterface{
// 		unsafeRandom: func() (uint64, error) {
// 			return 7558174677681708339, nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: script,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"7558174677681708339",
// 		},
// 		loggedMessages,
// 	)
// }

func TestRuntimeTransactionTopLevelDeclarations(t *testing.T) {

	t.Parallel()

	t.Run("transaction with function", func(t *testing.T) {
		runtime := NewInterpreterRuntime()

		script := []byte(`
          pub fun test() {}

          transaction {}
		`)

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.RunTransaction(
			NewScript([]byte(script), nil, []Address{}),
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
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

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.RunTransaction(
			NewScript([]byte(script), nil, []Address{}),
			Context{
				Accounts:         &testAccountsInterface{},
				AccountContracts: &testAccountContractsInterface{},
				AccountStorage:   newTestStorage(nil, nil),
				AccountKeys:      &testAccountKeysInterface{},
				LocationResolver: &testLocationResolverInterface{},
				ProgramCache:     &testProgramCacheInterface{},
				Results:          &testResultsInterface{},
				Utils:            &testUtilsInterface{},
				Location:         nextTransactionLocation(),
			},
		)
		require.Error(t, err)

		var checkerErr *sema.CheckerError
		utils.RequireErrorAs(t, err, &checkerErr)

		errs := checker.ExpectCheckerErrors(t, checkerErr, 1)

		assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
	})
}

// func TestRuntimeStoreIntegerTypes(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	addressValue := interpreter.AddressValue{
// 		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xCA, 0xDE,
// 	}

// 	for _, integerType := range sema.AllIntegerTypes {

// 		typeName := integerType.String()

// 		t.Run(typeName, func(t *testing.T) {

// 			contract := []byte(
// 				fmt.Sprintf(
// 					`
//                       pub contract Test {

//                           pub let n: %s

//                           init() {
//                               self.n = 42
//                           }
//                       }
//                     `,
// 					typeName,
// 				),
// 			)

// 			deploy := utils.DeploymentTransaction("Test", contract)

// 			var accountCode []byte
// 			var events []cadence.Event

// 			runtimeInterface := &testRuntimeInterface{
// 				getCode: func(_ Location) (bytes []byte, err error) {
// 					return accountCode, nil
// 				},
// 				storage: newTestStorage(nil, nil),
// 				getSigningAccounts: func() ([]Address, error) {
// 					return []Address{addressValue.ToAddress()}, nil
// 				},
// 				getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
// 					return accountCode, nil
// 				},
// 				updateAccountContractCode: func(_ Address, _ string, code []byte) error {
// 					accountCode = code
// 					return nil
// 				},
// 				emitEvent: func(event cadence.Event) error {
// 					events = append(events, event)
// 					return nil
// 				},
// 			}

// 			nextTransactionLocation := newTransactionLocationGenerator()

// 			err := runtime.RunTransaction(
// 				Script{
// 					Source: deploy,
// 				},
// 				Context{
// 					Interface: runtimeInterface,
// 					Location:  nextTransactionLocation(),
// 				},
// 			)
// 			require.NoError(t, err)

// 			assert.NotNil(t, accountCode)
// 		})
// 	}
// }

// func TestInterpretResourceOwnerFieldUseComposite(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	address := Address{
// 		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
// 	}

// 	contract := []byte(`
//       pub contract Test {

//           pub resource R {

//               pub fun logOwnerAddress() {
//                 log(self.owner?.address)
//               }
//           }

//           pub fun createR(): @R {
//               return <-create R()
//           }
//       }
//     `)

// 	deploy := utils.DeploymentTransaction("Test", contract)

// 	tx := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {

//               let r <- Test.createR()
//               log(r.owner?.address)
//               r.logOwnerAddress()

//               signer.save(<-r, to: /storage/r)
//               signer.link<&Test.R>(/public/r, target: /storage/r)

//               let ref1 = signer.borrow<&Test.R>(from: /storage/r)!
//               log(ref1.owner?.address)
//               ref1.logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/r).borrow<&Test.R>()!
//               log(ref2.owner?.address)
//               ref2.logOwnerAddress()
//           }
//       }
//     `)

// 	tx2 := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {
//               let ref1 = signer.borrow<&Test.R>(from: /storage/r)!
//               log(ref1.owner?.address)
//               ref1.logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/r).borrow<&Test.R>()!
//               log(ref2.owner?.address)
//               ref2.logOwnerAddress()
//           }
//       }
//     `)

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	var loggedMessages []string

// 	runtimeInterface := &testRuntimeInterface{
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{address}, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deploy,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"nil", "nil",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)

// 	loggedMessages = nil
// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx2,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)
// }

// func TestInterpretResourceOwnerFieldUseArray(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	address := Address{
// 		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
// 	}

// 	contract := []byte(`
//       pub contract Test {

//           pub resource R {

//               pub fun logOwnerAddress() {
//                 log(self.owner?.address)
//               }
//           }

//           pub fun createR(): @R {
//               return <-create R()
//           }
//       }
//     `)

// 	deploy := utils.DeploymentTransaction("Test", contract)

// 	tx := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {

//               let rs <- [
//                   <-Test.createR(),
//                   <-Test.createR()
//               ]
//               log(rs[0].owner?.address)
//               log(rs[1].owner?.address)
//               rs[0].logOwnerAddress()
//               rs[1].logOwnerAddress()

//               signer.save(<-rs, to: /storage/rs)
//               signer.link<&[Test.R]>(/public/rs, target: /storage/rs)

//               let ref1 = signer.borrow<&[Test.R]>(from: /storage/rs)!
//               log(ref1[0].owner?.address)
//               log(ref1[1].owner?.address)
//               ref1[0].logOwnerAddress()
//               ref1[1].logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/rs).borrow<&[Test.R]>()!
//               log(ref2[0].owner?.address)
//               log(ref2[1].owner?.address)
//               ref2[0].logOwnerAddress()
//               ref2[1].logOwnerAddress()
//           }
//       }
//     `)

// 	tx2 := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {
//               let ref1 = signer.borrow<&[Test.R]>(from: /storage/rs)!
//               log(ref1[0].owner?.address)
//               log(ref1[1].owner?.address)
//               ref1[0].logOwnerAddress()
//               ref1[1].logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/rs).borrow<&[Test.R]>()!
//               log(ref2[0].owner?.address)
//               log(ref2[1].owner?.address)
//               ref2[0].logOwnerAddress()
//               ref2[1].logOwnerAddress()
//           }
//       }
//     `)

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	var loggedMessages []string

// 	runtimeInterface := &testRuntimeInterface{
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{address}, nil
// 		},
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deploy,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"nil", "nil",
// 			"nil", "nil",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)

// 	loggedMessages = nil
// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx2,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)
// }

// func TestInterpretResourceOwnerFieldUseDictionary(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	address := Address{
// 		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
// 	}

// 	contract := []byte(`
//       pub contract Test {

//           pub resource R {

//               pub fun logOwnerAddress() {
//                 log(self.owner?.address)
//               }
//           }

//           pub fun createR(): @R {
//               return <-create R()
//           }
//       }
//     `)

// 	deploy := utils.DeploymentTransaction("Test", contract)

// 	tx := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {

//               let rs <- {
//                   "a": <-Test.createR(),
//                   "b": <-Test.createR()
//               }
//               log(rs["a"]?.owner?.address)
//               log(rs["b"]?.owner?.address)
//               rs["a"]?.logOwnerAddress()
//               rs["b"]?.logOwnerAddress()

//               signer.save(<-rs, to: /storage/rs)
//               signer.link<&{String: Test.R}>(/public/rs, target: /storage/rs)

//               let ref1 = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
//               log(ref1["a"]?.owner?.address)
//               log(ref1["b"]?.owner?.address)
//               ref1["a"]?.logOwnerAddress()
//               ref1["b"]?.logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/rs).borrow<&{String: Test.R}>()!
//               log(ref2["a"]?.owner?.address)
//               log(ref2["b"]?.owner?.address)
//               ref2["a"]?.logOwnerAddress()
//               ref2["b"]?.logOwnerAddress()
//           }
//       }
//     `)

// 	tx2 := []byte(`
//       import Test from 0x1

//       transaction {

//           prepare(signer: AuthAccount) {
//               let ref1 = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
//               log(ref1["a"]?.owner?.address)
//               log(ref1["b"]?.owner?.address)
//               ref1["a"]?.logOwnerAddress()
//               ref1["b"]?.logOwnerAddress()

//               let publicAccount = getAccount(0x01)
//               let ref2 = publicAccount.getCapability(/public/rs).borrow<&{String: Test.R}>()!
//               log(ref2["a"]?.owner?.address)
//               log(ref2["b"]?.owner?.address)
//               ref2["a"]?.logOwnerAddress()
//               ref2["b"]?.logOwnerAddress()
//           }
//       }
//     `)

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	var loggedMessages []string

// 	runtimeInterface := &testRuntimeInterface{
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{address}, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deploy,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"nil", "nil",
// 			"nil", "nil",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)

// 	loggedMessages = nil
// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: tx2,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		[]string{
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 			"0x1", "0x1",
// 		},
// 		loggedMessages,
// 	)
// }

// func TestRuntimeComputationLimit(t *testing.T) {

// 	t.Parallel()

// 	const computationLimit = 5

// 	type test struct {
// 		name string
// 		code string
// 		ok   bool
// 	}

// 	tests := []test{
// 		{
// 			name: "Infinite while loop",
// 			code: `
//               while true {}
//             `,
// 			ok: false,
// 		},
// 		{
// 			name: "Limited while loop",
// 			code: `
//               var i = 0
//               while i < 5 {
//                   i = i + 1
//               }
//             `,
// 			ok: false,
// 		},
// 		{
// 			name: "Too many for-in loop iterations",
// 			code: `
//               for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {}
//             `,
// 			ok: false,
// 		},
// 		{
// 			name: "Some for-in loop iterations",
// 			code: `
//               for i in [1, 2, 3, 4] {}
//             `,
// 			ok: true,
// 		},
// 	}

// 	for _, test := range tests {

// 		t.Run(test.name, func(t *testing.T) {

// 			script := []byte(
// 				fmt.Sprintf(
// 					`
//                       transaction {
//                           prepare() {
//                               %s
//                           }
//                       }
//                     `,
// 					test.code,
// 				),
// 			)

// 			runtime := NewInterpreterRuntime()

// 			runtimeInterface := &testRuntimeInterface{
// 				getSigningAccounts: func() ([]Address, error) {
// 					return nil, nil
// 				},
// 				computationLimit: computationLimit,
// 			}

// 			nextTransactionLocation := newTransactionLocationGenerator()

// 			err := runtime.RunTransaction(
// 				Script{
// 					Source: script,
// 				},
// 				Context{
// 					Interface: runtimeInterface,
// 					Location:  nextTransactionLocation(),
// 				},
// 			)
// 			if test.ok {
// 				require.NoError(t, err)
// 			} else {
// 				var computationLimitErr ComputationLimitExceededError
// 				utils.RequireErrorAs(t, err, &computationLimitErr)

// 				assert.Equal(t,
// 					ComputationLimitExceededError{
// 						Limit: computationLimit,
// 					},
// 					computationLimitErr,
// 				)
// 			}
// 		})
// 	}
// }

// func TestRuntimeMetrics(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	imported1Location := common.StringLocation("imported1")

// 	importedScript1 := []byte(`
//       pub fun generate(): [Int] {
//         return [1, 2, 3]
//       }
//     `)

// 	imported2Location := common.StringLocation("imported2")

// 	importedScript2 := []byte(`
//       pub fun getPath(): StoragePath {
//         return /storage/foo
//       }
//     `)

// 	script1 := []byte(`
//       import "imported1"

//       transaction {
//           prepare(signer: AuthAccount) {
//               signer.save(generate(), to: /storage/foo)
//           }
//           execute {}
//       }
//     `)

// 	script2 := []byte(`
//       import "imported2"

//       transaction {
//           prepare(signer: AuthAccount) {
//               signer.load<[Int]>(from: getPath())
//           }
//           execute {}
//       }
//     `)

// 	storage := newTestStorage(nil, nil)

// 	type reports struct {
// 		programParsed      map[common.LocationID]int
// 		programChecked     map[common.LocationID]int
// 		programInterpreted map[common.LocationID]int
// 		valueEncoded       int
// 		valueDecoded       int
// 	}

// 	newRuntimeInterface := func() (runtimeInterface Interface, r *reports) {

// 		r = &reports{
// 			programParsed:      map[common.LocationID]int{},
// 			programChecked:     map[common.LocationID]int{},
// 			programInterpreted: map[common.LocationID]int{},
// 		}

// 		runtimeInterface = &testRuntimeInterface{
// 			storage: storage,
// 			getSigningAccounts: func() ([]Address, error) {
// 				return []Address{{42}}, nil
// 			},
// 			getCode: func(location Location) (bytes []byte, err error) {
// 				switch location {
// 				case imported1Location:
// 					return importedScript1, nil
// 				case imported2Location:
// 					return importedScript2, nil
// 				default:
// 					return nil, fmt.Errorf("unknown import location: %s", location)
// 				}
// 			},
// 			programParsed: func(location common.Location, duration time.Duration) {
// 				r.programParsed[location.ID()]++
// 			},
// 			programChecked: func(location common.Location, duration time.Duration) {
// 				r.programChecked[location.ID()]++
// 			},
// 			programInterpreted: func(location common.Location, duration time.Duration) {
// 				r.programInterpreted[location.ID()]++
// 			},
// 			valueEncoded: func(duration time.Duration) {
// 				r.valueEncoded++
// 			},
// 			valueDecoded: func(duration time.Duration) {
// 				r.valueDecoded++
// 			},
// 		}

// 		return
// 	}

// 	i1, r1 := newRuntimeInterface()

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	transactionLocation := nextTransactionLocation()
// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: script1,
// 		},
// 		Context{
// 			Interface: i1,
// 			Location:  transactionLocation,
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 			imported1Location.ID():   1,
// 		},
// 		r1.programParsed,
// 	)
// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 			imported1Location.ID():   1,
// 		},
// 		r1.programChecked,
// 	)
// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 		},
// 		r1.programInterpreted,
// 	)
// 	assert.Equal(t, 1, r1.valueEncoded)
// 	assert.Equal(t, 0, r1.valueDecoded)

// 	i2, r2 := newRuntimeInterface()

// 	transactionLocation = nextTransactionLocation()

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: script2,
// 		},
// 		Context{
// 			Interface: i2,
// 			Location:  transactionLocation,
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 			imported2Location.ID():   1,
// 		},
// 		r2.programParsed,
// 	)
// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 			imported2Location.ID():   1,
// 		},
// 		r2.programChecked,
// 	)
// 	assert.Equal(t,
// 		map[common.LocationID]int{
// 			transactionLocation.ID(): 1,
// 		},
// 		r2.programInterpreted,
// 	)
// 	assert.Equal(t, 0, r2.valueEncoded)
// 	assert.Equal(t, 1, r2.valueDecoded)
// }

// type testRead struct {
// 	owner, key []byte
// }

// func (r testRead) String() string {
// 	return string(r.key)
// }

// type testWrite struct {
// 	owner, key, value []byte
// }

// func (w testWrite) String() string {
// 	return string(w.key)
// }

// func TestRuntimeContractWriteback(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

// 	contract := []byte(`
//       pub contract Test {

//           pub(set) var test: Int

//           init() {
//               self.test = 1
//           }
//       }
//     `)

// 	deploy := utils.DeploymentTransaction("Test", contract)

// 	readTx := []byte(`
//       import Test from 0xCADE

//        transaction {

//           prepare(signer: AuthAccount) {
//               log(Test.test)
//           }
//        }
//     `)

// 	writeTx := []byte(`
//       import Test from 0xCADE

//        transaction {

//           prepare(signer: AuthAccount) {
//               Test.test = 2
//           }
//        }
//     `)

// 	var accountCode []byte
// 	var events []cadence.Event
// 	var loggedMessages []string
// 	var writes []testWrite

// 	onWrite := func(owner, key, value []byte) {
// 		writes = append(writes, testWrite{
// 			owner,
// 			key,
// 			value,
// 		})
// 	}

// 	runtimeInterface := &testRuntimeInterface{
// 		getCode: func(_ Location) (bytes []byte, err error) {
// 			return accountCode, nil
// 		},
// 		storage: newTestStorage(nil, onWrite),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
// 			return accountCode, nil
// 		},
// 		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
// 			accountCode = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deploy,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.NotNil(t, accountCode)

// 	assert.Len(t, writes, 1)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: readTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Len(t, writes, 1)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: writeTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Len(t, writes, 2)
// }

// func TestRuntimeStorageWriteback(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

// 	contract := []byte(`
//       pub contract Test {

//           pub resource R {

//               pub(set) var test: Int

//               init() {
//                   self.test = 1
//               }
//           }

//           pub fun createR(): @R {
//               return <-create R()
//           }
//       }
//     `)

// 	deploy := utils.DeploymentTransaction("Test", contract)

// 	setupTx := []byte(`
//       import Test from 0xCADE

//        transaction {

//           prepare(signer: AuthAccount) {
//               signer.save(<-Test.createR(), to: /storage/r)
//           }
//        }
//     `)

// 	var accountCode []byte
// 	var events []cadence.Event
// 	var loggedMessages []string
// 	var writes []testWrite

// 	onWrite := func(owner, key, value []byte) {
// 		writes = append(writes, testWrite{
// 			owner,
// 			key,
// 			value,
// 		})
// 	}

// 	runtimeInterface := &testRuntimeInterface{
// 		getCode: func(_ Location) (bytes []byte, err error) {
// 			return accountCode, nil
// 		},
// 		storage: newTestStorage(nil, onWrite),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
// 			return accountCode, nil
// 		},
// 		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
// 			accountCode = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 		log: func(message string) {
// 			loggedMessages = append(loggedMessages, message)
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deploy,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.NotNil(t, accountCode)

// 	assert.Len(t, writes, 1)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: setupTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Len(t, writes, 2)

// 	readTx := []byte(`
//      import Test from 0xCADE

//       transaction {

//          prepare(signer: AuthAccount) {
//              log(signer.borrow<&Test.R>(from: /storage/r)!.test)
//          }
//       }
//     `)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: readTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Len(t, writes, 2)

// 	writeTx := []byte(`
//      import Test from 0xCADE

//       transaction {

//          prepare(signer: AuthAccount) {
//              let r = signer.borrow<&Test.R>(from: /storage/r)!
//              r.test = 2
//          }
//       }
//     `)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: writeTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	assert.Len(t, writes, 3)
// }

// func TestRuntimeExternalError(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	script := []byte(`
//       transaction {
//         prepare() {
//           log("ok")
//         }
//       }
//     `)

// 	type logPanic struct{}

// 	runtimeInterface := &testRuntimeInterface{
// 		getSigningAccounts: func() ([]Address, error) {
// 			return nil, nil
// 		},
// 		log: func(message string) {
// 			panic(logPanic{})
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	assert.PanicsWithValue(t,
// 		interpreter.ExternalError{
// 			Recovered: logPanic{},
// 		},
// 		func() {
// 			_ = runtime.RunTransaction(
// 				Script{
// 					Source: script,
// 				},
// 				Context{
// 					Interface: runtimeInterface,
// 					Location:  nextTransactionLocation(),
// 				},
// 			)
// 		},
// 	)
// }

// func TestRuntimeDeployCodeCaching(t *testing.T) {

// 	t.Parallel()

// 	const helloWorldContract = `
//       pub contract HelloWorld {

//           pub let greeting: String

//           init() {
//               self.greeting = "Hello, World!"
//           }

//           pub fun hello(): String {
//               return self.greeting
//           }
//       }
//     `

// 	const callHelloTxTemplate = `
//         import HelloWorld from 0x%s

//         transaction {
//             prepare(signer: AuthAccount) {
//                 assert(HelloWorld.hello() == "Hello, World!")
//             }
//         }
//     `

// 	createAccountTx := []byte(`
//         transaction {
//             prepare(signer: AuthAccount) {
//                 AuthAccount(payer: signer)
//             }
//         }
//     `)

// 	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

// 	runtime := NewInterpreterRuntime()

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	cachedPrograms := map[common.LocationID]*ast.Program{}

// 	var accountCounter uint8 = 0

// 	var signerAddresses []Address

// 	runtimeInterface := &testRuntimeInterface{
// 		createAccount: func(payer Address) (address Address, err error) {
// 			accountCounter++
// 			return Address{accountCounter}, nil
// 		},
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		cacheProgram: func(location Location, program *ast.Program) error {
// 			cachedPrograms[location.ID()] = program
// 			return nil
// 		},
// 		getCachedProgram: func(location Location) (*ast.Program, error) {
// 			return cachedPrograms[location.ID()], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return signerAddresses, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	// create the account

// 	signerAddresses = []Address{{accountCounter}}

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: createAccountTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	// deploy the contract

// 	signerAddresses = []Address{{accountCounter}}

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: deployTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	// call the hello function

// 	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: callTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// }

// func TestRuntimeUpdateCodeCaching(t *testing.T) {

// 	t.Parallel()

// 	const helloWorldContract1 = `
//       pub contract HelloWorld {

//           pub fun hello(): String {
//               return "1"
//           }
//       }
//     `

// 	const helloWorldContract2 = `
//       pub contract HelloWorld {

//           pub fun hello(): String {
//               return "2"
//           }
//       }
//     `

// 	const callHelloScriptTemplate = `
//         import HelloWorld from 0x%s

//         pub fun main(): String {
//             return HelloWorld.hello()
//         }
//     `

// 	createAccountTx := []byte(`
//         transaction {
//             prepare(signer: AuthAccount) {
//                 AuthAccount(payer: signer)
//             }
//         }
//     `)

// 	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract1))
// 	updateTx := utils.UpdateTransaction("HelloWorld", []byte(helloWorldContract2))

// 	runtime := NewInterpreterRuntime()

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	cachedPrograms := map[common.LocationID]*ast.Program{}

// 	var accountCounter uint8 = 0

// 	var signerAddresses []Address

// 	var cacheHits []string

// 	runtimeInterface := &testRuntimeInterface{
// 		createAccount: func(payer Address) (address Address, err error) {
// 			accountCounter++
// 			return Address{accountCounter}, nil
// 		},
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		cacheProgram: func(location Location, program *ast.Program) error {
// 			cachedPrograms[location.ID()] = program
// 			return nil
// 		},
// 		getCachedProgram: func(location Location) (*ast.Program, error) {
// 			cacheHits = append(cacheHits, string(location.ID()))
// 			return cachedPrograms[location.ID()], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return signerAddresses, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	// create the account

// 	signerAddresses = []Address{{accountCounter}}

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: createAccountTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	// deploy the contract

// 	cacheHits = nil

// 	signerAddresses = []Address{{accountCounter}}

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: deployTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// 	require.Empty(t, cacheHits)

// 	// call the initial hello function

// 	callScript := []byte(fmt.Sprintf(callHelloScriptTemplate, Address{accountCounter}))

// 	result1, err := runtime.RunScript(
// 		Script{
// 			Source: callScript,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// 	require.Equal(t, cadence.NewString("1"), result1)

// 	// update the contract

// 	cacheHits = nil

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: updateTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// 	require.Empty(t, cacheHits)

// 	// call the new hello function

// 	result2, err := runtime.RunScript(
// 		Script{
// 			Source: callScript,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// 	require.Equal(t, cadence.NewString("2"), result2)
// }

// func TestRuntimeNoCacheHitForToplevelPrograms(t *testing.T) {

// 	// We do not want to hit the cache for toplevel programs (scripts and
// 	// transactions) until we have moved the caching layer to Cadence.

// 	t.Parallel()

// 	const helloWorldContract = `
//       pub contract HelloWorld {

//           pub let greeting: String

//           init() {
//               self.greeting = "Hello, World!"
//           }

//           pub fun hello(): String {
//               return self.greeting
//           }
//       }
//     `

// 	const callHelloTxTemplate = `
//         import HelloWorld from 0x%s

//         transaction {
//             prepare(signer: AuthAccount) {
//                 assert(HelloWorld.hello() == "Hello, World!")
//             }
//         }
//     `

// 	createAccountTx := []byte(`
//         transaction {
//             prepare(signer: AuthAccount) {
//                 AuthAccount(payer: signer)
//             }
//         }
//     `)

// 	deployTx := utils.DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

// 	runtime := NewInterpreterRuntime()

// 	accountCodes := map[string][]byte{}
// 	var events []cadence.Event

// 	cachedPrograms := map[common.LocationID]*ast.Program{}

// 	var accountCounter uint8 = 0

// 	var signerAddresses []Address

// 	var cacheHits []string

// 	runtimeInterface := &testRuntimeInterface{
// 		createAccount: func(payer Address) (address Address, err error) {
// 			accountCounter++
// 			return Address{accountCounter}, nil
// 		},
// 		getCode: func(location Location) (bytes []byte, err error) {
// 			key := string(location.(common.AddressLocation).ID())
// 			return accountCodes[key], nil
// 		},
// 		cacheProgram: func(location Location, program *ast.Program) error {
// 			cachedPrograms[location.ID()] = program
// 			return nil
// 		},
// 		getCachedProgram: func(location Location) (*ast.Program, error) {
// 			cacheHits = append(cacheHits, string(location.ID()))
// 			return cachedPrograms[location.ID()], nil
// 		},
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return signerAddresses, nil
// 		},
// 		resolveLocation: singleIdentifierLocationResolver(t),
// 		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			return accountCodes[key], nil
// 		},
// 		updateAccountContractCode: func(address Address, name string, code []byte) error {
// 			location := common.AddressLocation{
// 				Address: address,
// 				Name:    name,
// 			}
// 			key := string(location.ID())
// 			accountCodes[key] = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	signerAddresses = []Address{{accountCounter}}

// 	// create the account

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: createAccountTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	signerAddresses = []Address{{accountCounter}}

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: deployTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	// call the function

// 	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: callTx,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	// We should only receive a cache hit for the imported program, not the transactions/scripts.

// 	// NOTE: if this test case fails with an additional cache hit,
// 	// then the deployment is incorrectly using the cache!

// 	require.Equal(t,
// 		[]string{
// 			"A.0100000000000000.HelloWorld",
// 		},
// 		cacheHits,
// 	)
// }

// func TestRuntimeTransaction_ContractUpdate(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	const contract1 = `
//       pub contract Test {

//           pub resource R {

//               pub let name: String

//               init(name: String) {
//                   self.name = name
//               }

//               pub fun hello(): Int {
//                   return 1
//               }
//           }

//           pub var rs: @{String: R}

//           pub fun hello(): Int {
//               return 1
//           }

//           init() {
//               self.rs <- {}
//               self.rs["r1"] <-! create R(name: "1")
//           }
//       }
//     `

// 	const contract2 = `
//       pub contract Test {

//           pub resource R {

//               pub let name: String

//               init(name: String) {
//                   self.name = name
//               }

//               pub fun hello(): Int {
//                   return 2
//               }
//           }

//           pub var rs: @{String: R}

//           pub fun hello(): Int {
//               return 2
//           }

//           init() {
//               self.rs <- {}
//               panic("should never be executed")
//           }
//       }
//     `

// 	newDeployTransaction := func(function, name, code string) []byte {
// 		return []byte(fmt.Sprintf(
// 			`
//               transaction {

//                   prepare(signer: AuthAccount) {
//                       signer.contracts.%s(name: "%s", code: "%s".decodeHex())
//                   }
//               }
//             `,
// 			function,
// 			name,
// 			hex.EncodeToString([]byte(code)),
// 		))
// 	}

// 	var accountCode []byte
// 	var events []cadence.Event

// 	runtimeInterface := &testRuntimeInterface{
// 		storage: newTestStorage(nil, nil),
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{common.BytesToAddress([]byte{0x42})}, nil
// 		},
// 		getCode: func(_ Location) (bytes []byte, err error) {
// 			return accountCode, nil
// 		},
// 		resolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
// 			require.Empty(t, identifiers)
// 			require.IsType(t, common.AddressLocation{}, location)

// 			return []ResolvedLocation{
// 				{
// 					Location: common.AddressLocation{
// 						Address: location.(common.AddressLocation).Address,
// 						Name:    "Test",
// 					},
// 					Identifiers: []ast.Identifier{
// 						{
// 							Identifier: "Test",
// 						},
// 					},
// 				},
// 			}, nil
// 		},
// 		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
// 			return accountCode, nil
// 		},
// 		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
// 			accountCode = code
// 			return nil
// 		},
// 		emitEvent: func(event cadence.Event) error {
// 			events = append(events, event)
// 			return nil
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	deployTx1 := newDeployTransaction("add", "Test", contract1)

// 	err := runtime.RunTransaction(
// 		Script{
// 			Source: deployTx1,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	script1 := []byte(`
//       import 0x42

//       pub fun main() {
//           // Check stored data

//           assert(Test.rs.length == 1)
//           assert(Test.rs["r1"]?.name == "1")

//           // Check functions

//           assert(Test.rs["r1"]?.hello() == 1)
//           assert(Test.hello() == 1)
//       }
//     `)

// 	_, err = runtime.RunScript(
// 		Script{
// 			Source: script1,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	deployTx2 := newDeployTransaction("update__experimental", "Test", contract2)

// 	err = runtime.RunTransaction(
// 		Script{
// 			Source: deployTx2,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)

// 	script2 := []byte(`
//       import 0x42

//       pub fun main() {
//           // Existing data is still available and the same as before

//           assert(Test.rs.length == 1)
//           assert(Test.rs["r1"]?.name == "1")

//           // New function code is executed.
//           // Compare with script1 above, which checked 1.

//           assert(Test.rs["r1"]?.hello() == 2)
//           assert(Test.hello() == 2)
//       }
//     `)

// 	_, err = runtime.RunScript(
// 		Script{
// 			Source: script2,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	require.NoError(t, err)
// }

// func TestRuntime(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	runtimeInterface := &testRuntimeInterface{
// 		decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
// 			return jsoncdc.Decode(b)
// 		},
// 	}

// 	script := []byte(`
//       pub fun main(num: Int) {}
//     `)

// 	type testCase struct {
// 		name      string
// 		arguments [][]byte
// 		valid     bool
// 	}

// 	test := func(tc testCase) {
// 		t.Run(tc.name, func(t *testing.T) {

// 			t.Parallel()

// 			_, err := runtime.RunScript(
// 				Script{
// 					Source:    script,
// 					Arguments: tc.arguments,
// 				},
// 				Context{
// 					Interface: runtimeInterface,
// 					Location:  common.ScriptLocation{0x1},
// 				},
// 			)

// 			if tc.valid {
// 				require.NoError(t, err)
// 			} else {
// 				require.Error(t, err)

// 				utils.RequireErrorAs(t, err, &InvalidEntryPointParameterCountError{})
// 			}
// 		})
// 	}

// 	for _, testCase := range []testCase{
// 		{
// 			name:      "too few arguments",
// 			arguments: [][]byte{},
// 			valid:     false,
// 		},
// 		{
// 			name: "correct number of arguments",
// 			arguments: [][]byte{
// 				jsoncdc.MustEncode(cadence.NewInt(1)),
// 			},
// 			valid: true,
// 		},
// 		{
// 			name: "too many arguments",
// 			arguments: [][]byte{
// 				jsoncdc.MustEncode(cadence.NewInt(1)),
// 				jsoncdc.MustEncode(cadence.NewInt(2)),
// 			},
// 			valid: false,
// 		},
// 	} {
// 		test(testCase)
// 	}
// }

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

// func TestPanics(t *testing.T) {

// 	t.Parallel()

// 	runtime := NewInterpreterRuntime()

// 	script := []byte(`
//       pub fun main() {
// 		[1][1]
//       }
//     `)

// 	runtimeInterface := &testRuntimeInterface{
// 		getSigningAccounts: func() ([]Address, error) {
// 			return []Address{{42}}, nil
// 		},
// 	}

// 	nextTransactionLocation := newTransactionLocationGenerator()

// 	_, err := runtime.RunScript(
// 		Script{
// 			Source: script,
// 		},
// 		Context{
// 			Interface: runtimeInterface,
// 			Location:  nextTransactionLocation(),
// 		},
// 	)
// 	assert.Error(t, err)
// }
