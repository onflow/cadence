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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeTransaction_AddPublicKey(t *testing.T) {
	rt := NewInterpreterRuntime()

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

	nextTransactionLocation := newTransactionLocationGenerator()

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
			addEncodedAccountKey: func(address Address, publicKey []byte) error {
				keys = append(keys, publicKey)
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
		}

		t.Run(tt.name, func(t *testing.T) {

			args := make([][]byte, len(tt.args))
			for i, arg := range tt.args {
				var err error
				args[i], err = json.Encode(arg)
				if err != nil {
					panic(fmt.Errorf("broken test: invalid argument: %w", err))
				}
			}

			err := rt.ExecuteTransaction(
				Script{
					Source:    []byte(tt.code),
					Arguments: args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
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

func TestRuntimeAccountKeyConstructor(t *testing.T) {

	t.Parallel()

	rt := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): AccountKey {
			let key = AccountKey(
				PublicKey(
					publicKey: "0102".decodeHex(),
					signAlgo: "SignatureAlgorithmECDSA_P256"
				),
				hashAlgorithm: "HashAlgorithmSHA3_256",
				weight: 1.7
			)

			return key
      	}
	`)

	runtimeInterface := &testRuntimeInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find variable in this scope: `AccountKey`")
}

func TestRuntimeImportAccountAPITypes(t *testing.T) {

	t.Parallel()

	nextTransactionLocation := newTransactionLocationGenerator()

	for _, ty := range []sema.Type{
		sema.AccountKeyType,
		sema.PublicKeyType,
	} {

		rt := NewInterpreterRuntime()

		script := []byte(fmt.Sprintf(`
			pub fun main(key: %s) {}
		`, ty.String()))

		runtimeInterface := &testRuntimeInterface{}

		_, err := rt.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameter type is non-storable type")
	}
}

func TestRuntimeStoreAccountAPITypes(t *testing.T) {

	t.Parallel()

	nextTransactionLocation := newTransactionLocationGenerator()

	for _, ty := range []sema.Type{
		sema.AccountKeyType,
		sema.PublicKeyType,
	} {

		rt := NewInterpreterRuntime()

		script := []byte(fmt.Sprintf(`
			transaction {

				prepare(signer: AuthAccount) {
					signer.save<%s>(panic(""))
				}
			}
		`, ty.String()))

		runtimeInterface := &testRuntimeInterface{}

		err := rt.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected `Storable`")
	}
}

var accountKeyA = &AccountKey{
	KeyIndex: 0,
	PublicKey: &PublicKey{
		PublicKey: []byte{1, 2, 3},
		SignAlgo:  sema.SignatureAlgorithmECDSA_P256,
		IsValid:   false,
		Validated: true,
	},
	HashAlgo:  sema.HashAlgorithmSHA3_256,
	Weight:    100,
	IsRevoked: false,
}

var accountKeyB = &AccountKey{
	KeyIndex: 1,
	PublicKey: &PublicKey{
		PublicKey: []byte{4, 5, 6},
		SignAlgo:  sema.SignatureAlgorithmECDSA_secp256k1,
		IsValid:   false,
		Validated: false,
	},
	HashAlgo:  sema.HashAlgorithmSHA3_256,
	Weight:    100,
	IsRevoked: false,
}

var revokedAccountKeyA = func() *AccountKey {
	revokedKey := *accountKeyA
	revokedKey.IsRevoked = true
	return &revokedKey
}()

func TestRuntimeAuthAccountKeys(t *testing.T) {

	t.Parallel()

	t.Run("add key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		rt := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		addAuthAccountKey(t, rt, runtimeInterface)

		assert.Equal(t, []*AccountKey{accountKeyA}, storage.keys)
		assert.Equal(t, accountKeyA, storage.returnedKey)
	})

	t.Run("get key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		rt := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		addAuthAccountKey(t, rt, runtimeInterface)

		test := accountKeyTestCase{
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						let key: AccountKey? = signer.keys.get(keyIndex: 0)
					}
				}`,
			args: []cadence.Value{},
		}

		err := test.executeTransaction(rt, runtimeInterface)
		require.NoError(t, err)

		assert.Equal(t, []*AccountKey{accountKeyA}, storage.keys)
		assert.Equal(t, accountKeyA, storage.returnedKey)
	})

	t.Run("get non existing key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		rt := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		addAuthAccountKey(t, rt, runtimeInterface)

		test := accountKeyTestCase{
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						let key: AccountKey? = signer.keys.get(keyIndex: 5)
					}
				}`,
			args: []cadence.Value{},
		}

		err := test.executeTransaction(rt, runtimeInterface)
		require.NoError(t, err)
		assert.Nil(t, storage.returnedKey)
	})

	t.Run("revoke key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		rt := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		addAuthAccountKey(t, rt, runtimeInterface)

		test := accountKeyTestCase{
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						let key: AccountKey? = signer.keys.revoke(keyIndex: 0)
					}
				}`,
			args: []cadence.Value{},
		}

		err := test.executeTransaction(rt, runtimeInterface)
		require.NoError(t, err)

		assert.Equal(t, []*AccountKey{revokedAccountKeyA}, storage.keys)
		assert.Equal(t, revokedAccountKeyA, storage.returnedKey)
	})

	t.Run("revoke non existing key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		rt := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		addAuthAccountKey(t, rt, runtimeInterface)

		test := accountKeyTestCase{
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						var acc: AccountKey? = signer.keys.revoke(keyIndex: 5)
					}
				}`,
			args: []cadence.Value{},
		}

		err := test.executeTransaction(rt, runtimeInterface)
		require.NoError(t, err)
		assert.Nil(t, storage.returnedKey)
	})
}

func TestRuntimeAuthAccountKeysAdd(t *testing.T) {

	t.Parallel()

	rt := NewInterpreterRuntime()

	pubKey := newBytesValue([]byte{1, 2, 3})

	const code = `
       transaction(publicKey: [UInt8]) {
           prepare(signer: AuthAccount) {
               let acct = AuthAccount(payer: signer)
               acct.keys.add(
                   publicKey: PublicKey(
                       publicKey: publicKey,
                       signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
                   ),
                   hashAlgorithm: HashAlgorithm.SHA3_256,
                   weight: 100.0
               )
           }
       }
   `

	storage := newTestAccountKeyStorage()
	runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source:    []byte(code),
			Arguments: encodeArgs([]cadence.Value{pubKey}),
		},
		Context{
			Location:  nextTransactionLocation(),
			Interface: runtimeInterface,
		},
	)

	require.NoError(t, err)
	assert.Len(t, storage.keys, 1)

	require.Len(t, storage.events, 2)

	assert.EqualValues(t,
		stdlib.AccountCreatedEventType.ID(),
		storage.events[0].Type().ID(),
	)

	assert.EqualValues(t,
		stdlib.AccountKeyAddedEventType.ID(),
		storage.events[1].Type().ID(),
	)
}

func TestRuntimePublicAccountKeys(t *testing.T) {

	t.Run("get key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		runtime := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		test := accountKeyTestCase{
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 0)
				}`,
			args: []cadence.Value{},
		}

		value, err := test.executeScript(runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		expectedValue := accountKeyExportedValue(
			0,
			[]byte{1, 2, 3},
			sema.SignatureAlgorithmECDSA_P256,
			sema.HashAlgorithmSHA3_256,
			"100.0",
			false,
		)

		assert.Equal(t, expectedValue, optionalValue.Value)
		assert.Equal(t, accountKeyA, storage.returnedKey)

	})

	t.Run("get another key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		runtime := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		test := accountKeyTestCase{
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 1)
				}`,
			args: []cadence.Value{},
		}

		value, err := test.executeScript(runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		expectedValue := accountKeyExportedValue(1,
			[]byte{4, 5, 6},
			sema.SignatureAlgorithmECDSA_secp256k1,
			sema.HashAlgorithmSHA3_256,
			"100.0",
			false,
		)

		assert.Equal(t, expectedValue, optionalValue.Value)
		assert.Equal(t, accountKeyB, storage.returnedKey)
	})

	t.Run("get non existing key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		runtime := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		test := accountKeyTestCase{
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 4)
				}`,
			args: []cadence.Value{},
		}

		value, err := test.executeScript(runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		assert.Nil(t, optionalValue.Value)
	})

	t.Run("get revoked key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, revokedAccountKeyA, accountKeyB)

		runtime := NewInterpreterRuntime()
		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)

		test := accountKeyTestCase{
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					var keys: PublicAccount.Keys = acc.keys
					return keys.get(keyIndex: 0)
				}`,
			args: []cadence.Value{},
		}

		value, err := test.executeScript(runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		expectedValue := accountKeyExportedValue(
			0,
			[]byte{1, 2, 3},
			sema.SignatureAlgorithmECDSA_P256,
			sema.HashAlgorithmSHA3_256,
			"100.0",
			true,
		)

		assert.Equal(t, expectedValue, optionalValue.Value)
		assert.Equal(t, revokedAccountKeyA, storage.returnedKey)
	})
}

func TestRuntimeHashAlgorithm(t *testing.T) {

	t.Parallel()

	rt := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): [HashAlgorithm?] {
			var key1: HashAlgorithm? = HashAlgorithm.SHA3_256

			var key2: HashAlgorithm? = HashAlgorithm(rawValue: 3)

			var key3: HashAlgorithm? = HashAlgorithm(rawValue: 100)
			return [key1, key2, key3]
      	}
	`)

	runtimeInterface := &testRuntimeInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.IsType(t, cadence.Array{}, result)
	array := result.(cadence.Array)

	require.Len(t, array.Values, 3)

	// Check key1
	require.IsType(t, cadence.Optional{}, array.Values[0])
	optionalValue := array.Values[0].(cadence.Optional)

	require.IsType(t, cadence.Enum{}, optionalValue.Value)
	builtinStruct := optionalValue.Value.(cadence.Enum)

	require.Len(t, builtinStruct.Fields, 1)
	assert.Equal(t,
		cadence.NewUInt8(HashAlgorithmSHA3_256.RawValue()),
		builtinStruct.Fields[0],
	)

	// Check key2
	require.IsType(t, cadence.Optional{}, array.Values[1])
	optionalValue = array.Values[1].(cadence.Optional)

	require.IsType(t, cadence.Enum{}, optionalValue.Value)
	builtinStruct = optionalValue.Value.(cadence.Enum)

	require.Len(t, builtinStruct.Fields, 1)
	assert.Equal(t,
		cadence.NewUInt8(HashAlgorithmSHA3_256.RawValue()),
		builtinStruct.Fields[0],
	)

	// Check key3
	require.IsType(t, cadence.Optional{}, array.Values[2])
	optionalValue = array.Values[2].(cadence.Optional)

	require.Nil(t, optionalValue.Value)
}

func TestRuntimeSignatureAlgorithm(t *testing.T) {

	t.Parallel()

	rt := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): [SignatureAlgorithm?] {
			var key1: SignatureAlgorithm? = SignatureAlgorithm.ECDSA_secp256k1

			var key2: SignatureAlgorithm? = SignatureAlgorithm(rawValue: 2)

			var key3: SignatureAlgorithm? = SignatureAlgorithm(rawValue: 100)
			return [key1, key2, key3]
		}
	`)

	runtimeInterface := &testRuntimeInterface{}

	nextTransactionLocation := newTransactionLocationGenerator()

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.IsType(t, cadence.Array{}, result)
	array := result.(cadence.Array)

	require.Len(t, array.Values, 3)

	// Check key1
	require.IsType(t, cadence.Optional{}, array.Values[0])
	optionalValue := array.Values[0].(cadence.Optional)

	require.IsType(t, cadence.Enum{}, optionalValue.Value)
	builtinStruct := optionalValue.Value.(cadence.Enum)

	require.Len(t, builtinStruct.Fields, 1)
	assert.Equal(t,
		cadence.NewUInt8(SignatureAlgorithmECDSA_secp256k1.RawValue()),
		builtinStruct.Fields[0],
	)

	// Check key2
	require.IsType(t, cadence.Optional{}, array.Values[1])
	optionalValue = array.Values[1].(cadence.Optional)

	require.IsType(t, cadence.Enum{}, optionalValue.Value)
	builtinStruct = optionalValue.Value.(cadence.Enum)

	require.Len(t, builtinStruct.Fields, 1)
	assert.Equal(t,
		cadence.NewUInt8(SignatureAlgorithmECDSA_secp256k1.RawValue()),
		builtinStruct.Fields[0],
	)

	// Check key3
	require.IsType(t, cadence.Optional{}, array.Values[2])
	optionalValue = array.Values[2].(cadence.Optional)

	require.Nil(t, optionalValue.Value)
}

// Utility methods and types

var AccountKeyType = ExportedBuiltinType(sema.AccountKeyType).(*cadence.StructType)
var PublicKeyType = ExportedBuiltinType(sema.PublicKeyType).(*cadence.StructType)
var SignAlgoType = ExportedBuiltinType(sema.SignatureAlgorithmType).(*cadence.EnumType)
var HashAlgoType = ExportedBuiltinType(sema.HashAlgorithmType).(*cadence.EnumType)

func ExportedBuiltinType(internalType sema.Type) cadence.Type {
	return ExportType(internalType, map[sema.TypeID]cadence.Type{})
}

func newBytesValue(bytes []byte) cadence.Array {
	result := make([]cadence.Value, len(bytes))
	for index, value := range bytes {
		result[index] = cadence.NewUInt8(value)
	}
	return cadence.NewArray(result)
}

func newSignAlgoValue(signAlgo sema.SignatureAlgorithm) cadence.Enum {
	return cadence.NewEnum([]cadence.Value{
		cadence.NewUInt8(signAlgo.RawValue()),
	}).WithType(SignAlgoType)
}

func accountKeyExportedValue(
	index int,
	publicKeyBytes []byte,
	signAlgo sema.SignatureAlgorithm,
	hashAlgo sema.HashAlgorithm,
	weight string,
	isRevoked bool,
) cadence.Struct {

	weightUFix64, err := cadence.NewUFix64(weight)
	if err != nil {
		panic(err)
	}

	return cadence.Struct{
		StructType: AccountKeyType,
		Fields: []cadence.Value{
			// Key index
			cadence.NewInt(index),

			// Public Key (struct)
			cadence.Struct{
				StructType: PublicKeyType,
				Fields: []cadence.Value{
					// Public key (bytes)
					newBytesValue(publicKeyBytes),

					// Signature Algo
					newSignAlgoValue(signAlgo),

					// valid
					cadence.Bool(false),
				},
			},

			// Hash algo
			cadence.NewEnum([]cadence.Value{
				cadence.NewUInt8(hashAlgo.RawValue()),
			}).WithType(HashAlgoType),

			// Weight
			weightUFix64,

			// IsRevoked
			cadence.NewBool(isRevoked),
		},
	}
}

func getAccountKeyTestRuntimeInterface(storage *testAccountKeyStorage) *testRuntimeInterface {
	return &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		createAccount: func(payer Address) (address Address, err error) {
			return Address{42}, nil
		},
		addAccountKey: func(address Address, publicKey *PublicKey, hashAlgo HashAlgorithm, weight int) (*AccountKey, error) {
			index := len(storage.keys)
			accountKey := &AccountKey{
				KeyIndex:  index,
				PublicKey: publicKey,
				HashAlgo:  hashAlgo,
				Weight:    weight,
				IsRevoked: false,
			}

			storage.keys = append(storage.keys, accountKey)
			storage.returnedKey = accountKey
			return accountKey, nil
		},

		getAccountKey: func(address Address, index int) (*AccountKey, error) {
			if index >= len(storage.keys) {
				storage.returnedKey = nil
				return nil, nil
			}

			accountKey := storage.keys[index]
			storage.returnedKey = accountKey
			return accountKey, nil
		},

		removeAccountKey: func(address Address, index int) (*AccountKey, error) {
			if index >= len(storage.keys) {
				storage.returnedKey = nil
				return nil, nil
			}

			accountKey := storage.keys[index]
			accountKey.IsRevoked = true

			storage.keys[index] = accountKey
			storage.returnedKey = accountKey

			return accountKey, nil
		},

		emitEvent: func(event cadence.Event) error {
			storage.events = append(storage.events, event)
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(b)
		},
	}
}

func addAuthAccountKey(t *testing.T, runtime Runtime, runtimeInterface *testRuntimeInterface) {
	test := accountKeyTestCase{
		name: "Add key",
		code: `
				transaction {
					prepare(signer: AuthAccount) {
						let key = PublicKey(
							publicKey: "010203".decodeHex(),
							signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
						)

						var addedKey: AccountKey = signer.keys.add(
							publicKey: key,
							hashAlgorithm: HashAlgorithm.SHA3_256,
							weight: 100.0
						)
					}
				}`,
		args: []cadence.Value{},
	}

	err := test.executeTransaction(runtime, runtimeInterface)
	require.NoError(t, err)
}

func encodeArgs(argValues []cadence.Value) [][]byte {
	args := make([][]byte, len(argValues))
	for i, arg := range argValues {
		var err error
		args[i], err = json.Encode(arg)
		if err != nil {
			panic(fmt.Errorf("broken test: invalid argument: %w", err))
		}
	}
	return args
}

type accountKeyTestCase struct {
	name string
	code string
	args []cadence.Value
}

func (test accountKeyTestCase) executeTransaction(
	runtime Runtime,
	runtimeInterface *testRuntimeInterface,
) error {
	args := encodeArgs(test.args)

	err := runtime.ExecuteTransaction(
		Script{
			Source:    []byte(test.code),
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.TransactionLocation{},
		},
	)
	return err
}

func (test accountKeyTestCase) executeScript(
	runtime Runtime,
	runtimeInterface *testRuntimeInterface,
) (cadence.Value, error) {

	args := encodeArgs(test.args)

	value, err := runtime.ExecuteScript(
		Script{
			Source:    []byte(test.code),
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	return value, err
}

func newTestAccountKeyStorage() *testAccountKeyStorage {
	return &testAccountKeyStorage{
		events: make([]cadence.Event, 0),
		keys:   make([]*AccountKey, 0),
	}
}

type testAccountKeyStorage struct {
	events      []cadence.Event
	keys        []*AccountKey
	returnedKey *AccountKey
}

func TestPublicKey(t *testing.T) {

	t.Parallel()

	rt := NewInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{}

	executeScript := func(code string, runtimeInterface Interface) (cadence.Value, error) {
		return rt.ExecuteScript(
			Script{
				Source: []byte(code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)
	}

	t.Run("Constructor", func(t *testing.T) {
		script := `
			pub fun main(): PublicKey {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				return publicKey
			}
		`

		value, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		expected := cadence.Struct{
			StructType: PublicKeyType,
			Fields: []cadence.Value{
				// Public key (bytes)
				newBytesValue([]byte{1, 2}),

				// Signature Algo
				newSignAlgoValue(sema.SignatureAlgorithmECDSA_P256),

				// valid
				cadence.Bool(false),
			},
		}

		assert.Equal(t, expected, value)
	})

	t.Run("Validate func", func(t *testing.T) {
		script := `
			pub fun main(): Bool {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				return publicKey.validate()
			}
		`

		runtimeInterface := &testRuntimeInterface{
			validatePublicKey: func(publicKey *PublicKey) (bool, error) {
				return true, nil
			},
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value of type `PublicKey` has no member `validate`")
	})

	t.Run("IsValid", func(t *testing.T) {
		for _, validity := range []bool{true, false} {
			script := `
			pub fun main(): Bool {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				return publicKey.isValid
			}
		`
			invoked := false
			validateMethodReturnValue := validity

			runtimeInterface := &testRuntimeInterface{
				validatePublicKey: func(publicKey *PublicKey) (bool, error) {
					invoked = true
					return validateMethodReturnValue, nil
				},
			}

			value, err := executeScript(script, runtimeInterface)
			require.NoError(t, err)

			assert.True(t, invoked)
			assert.Equal(t, cadence.Bool(validateMethodReturnValue), value)

		}
	})

	t.Run("IsValid - publicKey from host env", func(t *testing.T) {

		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		for index, key := range storage.keys {
			script := fmt.Sprintf(`
				pub fun main(): Bool {
					// Get a public key from host env
					let acc = getAccount(0x02)
					let publicKey = acc.keys.get(keyIndex: %d)!.publicKey
					return publicKey.isValid
				}
			`, index)

			invoked := false
			validateMethodReturnValue := true

			runtimeInterface := getAccountKeyTestRuntimeInterface(storage)
			runtimeInterface.validatePublicKey = func(publicKey *PublicKey) (bool, error) {
				invoked = true
				return validateMethodReturnValue, nil
			}

			value, err := executeScript(script, runtimeInterface)
			require.NoError(t, err)

			// If already validated, then the validation func shouldn't get re-invoked
			assert.NotEqual(t, key.PublicKey.Validated, invoked)

			// If validated, `isValid` should have the same value as `publicKey.IsValid`.
			// Otherwise, it should give the value returned by the `validate()` func.
			isValid := validateMethodReturnValue
			if key.PublicKey.Validated {
				isValid = key.PublicKey.IsValid
			}

			assert.Equal(t, cadence.Bool(isValid), value)
		}
	})

	t.Run("Verify", func(t *testing.T) {
		script := `
			pub fun main(): Bool {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				return publicKey.verify(
					signature: [],
					signedData: [],
					domainSeparationTag: "something",
					hashAlgorithm: HashAlgorithm.SHA2_256
				)
			}
		`
		invoked := false

		runtimeInterface := &testRuntimeInterface{
			verifySignature: func(
				_ []byte,
				_ string,
				_ []byte,
				_ []byte,
				_ SignatureAlgorithm,
				_ HashAlgorithm,
			) (bool, error) {
				invoked = true
				return true, nil
			},
		}

		value, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, invoked)
		assert.Equal(t, cadence.Bool(true), value)
	})

	t.Run("Verify - publicKey from host env", func(t *testing.T) {

		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		script := `
			pub fun main(): Bool {
				// Get a public key from host env
				let acc = getAccount(0x02)
				let publicKey = acc.keys.get(keyIndex: 0)!.publicKey

				return publicKey.verify(
					signature: [],
					signedData: [],
					domainSeparationTag: "something",
					hashAlgorithm: HashAlgorithm.SHA2_256
				)
			}
		`
		invoked := false

		runtimeInterface := getAccountKeyTestRuntimeInterface(storage)
		runtimeInterface.verifySignature = func(
			_ []byte,
			_ string,
			_ []byte,
			_ []byte,
			_ SignatureAlgorithm,
			_ HashAlgorithm,
		) (bool, error) {
			invoked = true
			return true, nil
		}

		value, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, invoked)
		assert.Equal(t, cadence.Bool(true), value)
	})

	t.Run("field mutability", func(t *testing.T) {
		script := `
			pub fun main(): PublicKey {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				publicKey.publicKey = []
				publicKey.signatureAlgorithm = SignatureAlgorithm.ECDSA_secp256k1
				publicKey.isValid = true

				return publicKey
			}
		`

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checkerErr.Errors
		require.Len(t, errs, 6)

		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[1])
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[2])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[3])
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[4])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[5])
	})

	t.Run("raw-key mutability", func(t *testing.T) {
		script := `
			pub fun main(): PublicKey {
				let publicKey =  PublicKey(
					publicKey: "0102".decodeHex(),
					signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
				)

				publicKey.publicKey[0] = 5

				return publicKey
			}
		`

		value, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		expected := cadence.Struct{
			StructType: PublicKeyType,
			Fields: []cadence.Value{
				// Public key (bytes)
				newBytesValue([]byte{1, 2}),

				// Signature Algo
				newSignAlgoValue(sema.SignatureAlgorithmECDSA_P256),

				// valid
				cadence.Bool(false),
			},
		}

		assert.Equal(t, expected, value)
	})

}
