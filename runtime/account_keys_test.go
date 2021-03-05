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
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

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

func TestAccountKeyCreation(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): AccountKey {
			let key = AccountKey(
				PublicKey(
					publicKey: "0102".decodeHex(),
					signAlgo: "SignatureAlgorithmECDSA_P256"
				),
				hashAlgo: "HashAlgorithmSHA3_256",
				weight: 1.7
			)

			return key
      	}
	`)

	runtimeInterface := &testRuntimeInterface{}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find variable in this scope: `AccountKey`")
}

func TestImportExportKeys(t *testing.T) {
	t.Parallel()

	runtime := NewInterpreterRuntime()

	var tests = []accountKeyTestCase{
		{
			name: "AccountKey as transaction param",
			code: `
				pub fun main(key: AccountKey): PublicKey {
					return key.publicKey
				}
			`,
			args: []cadence.Value{accountKeyExportedValue(
				0,
				[]byte{1, 2, 3},
				sema.SignatureAlgorithmECDSA_P256,
				sema.HashAlgorithmSHA3_256,
				"100.0",
				true,
			)},
		},
	}

	for _, test := range tests {
		storage := newTestAccountKeyStorage()
		runtimeInterface := getRuntimeInterface(storage)

		t.Run(test.name, func(t *testing.T) {
			value, err := executeScript(test, runtime, runtimeInterface)
			require.NoError(t, err)
			require.NotNil(t, value)

			expectedValue := publicKeyExportedValue(
				[]byte{1, 2, 3},
				sema.SignatureAlgorithmECDSA_P256,
			)
			assert.Equal(t, expectedValue, value)
		})
	}
}

func TestImportInvalidType(t *testing.T) {
	t.Parallel()

	runtime := NewInterpreterRuntime()

	// encoded with an invalid type: 'N.PublicKey'
	encodedArgs := []byte(`{
		"type":"Struct",
		"value":{
			"id":"N.PublicKey",
			"fields":[
				{
					"name":"publicKey",
					"value":{
						"type":"Array",
						"value":[{"type":"UInt8","value":"1"}]}
				},
				{
					"name":"signAlgo",
					"value":{
						"type":"Struct",
						"value":{
							"id":"SignatureAlgorithm",
							"fields":[
								{
									"name":"rawValue",
									"value":{"type":"Int","value":"0"}
								}
							]
						}
					}
				}
			]
		}
	}`)

	code := `
		pub fun main(key: PublicKey): PublicKey {
			return key
		}`

	storage := newTestAccountKeyStorage()
	runtimeInterface := getRuntimeInterface(storage)

	_, err := runtime.ExecuteScript(
		Script{
			Source:    []byte(code),
			Arguments: [][]byte{encodedArgs},
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)

	require.Error(t, err)
	require.IsType(t, Error{}, err)

	runtimeErr := err.(Error)
	require.IsType(t, &InvalidEntryPointArgumentError{}, runtimeErr.Err)

	argError := runtimeErr.Err.(*InvalidEntryPointArgumentError)
	require.Error(t, argError.Err)
	assert.Equal(t, "failed to decode value: invalid JSON Cadence structure. invalid type ID: N.PublicKey", argError.Err.Error())
}

var accountKeyA = AccountKey{
	KeyIndex: 0,
	PublicKey: &PublicKey{
		PublicKey: []byte{1, 2, 3},
		SignAlgo:  sema.SignatureAlgorithmECDSA_P256,
	},
	HashAlgo:  sema.HashAlgorithmSHA3_256,
	Weight:    100,
	IsRevoked: false,
}

var accountKeyB = AccountKey{
	KeyIndex: 1,
	PublicKey: &PublicKey{
		PublicKey: []byte{4, 5, 6},
		SignAlgo:  sema.SignatureAlgorithmECDSA_Secp256k1,
	},
	HashAlgo:  sema.HashAlgorithmSHA3_256,
	Weight:    100,
	IsRevoked: false,
}

var revokedAccountKeyA = func() AccountKey {
	revokedKey := accountKeyA
	revokedKey.IsRevoked = true
	return revokedKey
}()

func TestAuthAccountKeys(t *testing.T) {

	t.Parallel()

	t.Run("add key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		runtime := NewInterpreterRuntime()
		runtimeInterface := getRuntimeInterface(storage)

		addAuthAccountKey(t, runtime, runtimeInterface)

		assert.Equal(t, []AccountKey{accountKeyA}, storage.keys)
		assert.Equal(t, accountKeyA, storage.returnedKey)
	})

	t.Run("get key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		runtime := NewInterpreterRuntime()
		runtimeInterface := getRuntimeInterface(storage)

		addAuthAccountKey(t, runtime, runtimeInterface)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						var acc: AccountKey? = signer.keys.get(keyIndex: 0)
					}
				}`,
			args: []cadence.Value{},
		}

		err := executeTransaction(test, runtime, runtimeInterface)
		require.NoError(t, err)

		assert.Equal(t, []AccountKey{accountKeyA}, storage.keys)
		assert.Equal(t, accountKeyA, storage.returnedKey)
	})

	t.Run("revoke key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		runtime := NewInterpreterRuntime()
		runtimeInterface := getRuntimeInterface(storage)

		addAuthAccountKey(t, runtime, runtimeInterface)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				transaction {
					prepare(signer: AuthAccount) {
						var acc: AccountKey? = signer.keys.revoke(keyIndex: 0)
					}
				}`,
			args: []cadence.Value{},
		}

		err := executeTransaction(test, runtime, runtimeInterface)
		require.NoError(t, err)

		assert.Equal(t, []AccountKey{revokedAccountKeyA}, storage.keys)
		assert.Equal(t, revokedAccountKeyA, storage.returnedKey)
	})
}

func TestAuthAccountAddPublicKey(t *testing.T) {
	t.Parallel()

	runtime := NewInterpreterRuntime()

	keyA := publicKeyExportedValue([]byte{1, 2, 3}, sema.SignatureAlgorithmECDSA_P256)
	keyB := publicKeyExportedValue([]byte{4, 5, 6}, sema.SignatureAlgorithmECDSA_Secp256k1)
	keys := cadence.NewArray([]cadence.Value{keyA, keyB})

	var tests = []accountKeyTestCase{
		{
			name: "Single key",
			code: `
				transaction(key: PublicKey) {
					prepare(signer: AuthAccount) {
						let acct = AuthAccount(payer: signer)	
						acct.keys.add(
							publicKey: key,
							hashAlgo: HashAlgorithm.SHA3_256,
							weight: 100.0
						)
					}
				}`,
			args: []cadence.Value{keyA},
			keys: []AccountKey{
				accountKeyA,
			},
		},
		{
			name: "Multiple keys",
			code: `
				transaction(keys: [PublicKey]) {
					prepare(signer: AuthAccount) {
						let acct = AuthAccount(payer: signer)
						var accountKeys: AuthAccount.Keys = acct.keys

						for key in keys {
							accountKeys.add(
								publicKey: key,
								hashAlgo: HashAlgorithm.SHA3_256,
								weight: 100.0
							)
						}
					}
				}
			`,
			args: []cadence.Value{keys},
			keys: []AccountKey{
				accountKeyA,
				accountKeyB,
			},
		},
	}

	for _, test := range tests {
		storage := newTestAccountKeyStorage()
		runtimeInterface := getRuntimeInterface(storage)

		t.Run(test.name, func(t *testing.T) {
			err := executeTransaction(test, runtime, runtimeInterface)

			require.NoError(t, err)
			assert.Equal(t, test.keys, storage.keys)

			assert.EqualValues(t, stdlib.AccountCreatedEventType.ID(), storage.events[0].Type().ID())

			for _, event := range storage.events[1:] {
				assert.EqualValues(t, stdlib.AccountKeyAddedEventType.ID(), event.Type().ID())
			}
		})
	}
}

func TestPublicAccountKeys(t *testing.T) {

	t.Run("get key", func(t *testing.T) {
		storage := newTestAccountKeyStorage()
		storage.keys = append(storage.keys, accountKeyA, accountKeyB)

		runtime := NewInterpreterRuntime()
		runtimeInterface := getRuntimeInterface(storage)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 0)
				}`,
			args: []cadence.Value{},
		}

		value, err := executeScript(test, runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		expectedValue := accountKeyExportedValue(0,
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
		runtimeInterface := getRuntimeInterface(storage)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 1)
				}`,
			args: []cadence.Value{},
		}

		value, err := executeScript(test, runtime, runtimeInterface)
		require.NoError(t, err)
		require.NotNil(t, value)

		require.IsType(t, cadence.Optional{}, value)
		optionalValue := value.(cadence.Optional)

		expectedValue := accountKeyExportedValue(1,
			[]byte{4, 5, 6},
			sema.SignatureAlgorithmECDSA_Secp256k1,
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
		runtimeInterface := getRuntimeInterface(storage)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					return acc.keys.get(keyIndex: 4)
				}`,
			args: []cadence.Value{},
		}

		value, err := executeScript(test, runtime, runtimeInterface)
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
		runtimeInterface := getRuntimeInterface(storage)

		test := accountKeyTestCase{
			name: "Add key",
			code: `
				pub fun main(): AccountKey? {
					let acc = getAccount(0x02)
					var keys: PublicAccount.Keys = acc.keys
					return keys.get(keyIndex: 0)
				}`,
			args: []cadence.Value{},
		}

		value, err := executeScript(test, runtime, runtimeInterface)
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

func TestHashAlgorithm(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): [HashAlgorithm?] {
			var key1: HashAlgorithm? = HashAlgorithm.KMAC128

			var key2: HashAlgorithm? = HashAlgorithm(rawValue:4)

			var key3: HashAlgorithm? = HashAlgorithm(rawValue:10)
			return [key1, key2, key3]
      	}
	`)

	runtimeInterface := &testRuntimeInterface{}

	result, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	require.NoError(t, err)

	require.IsType(t, cadence.Array{}, result)
	array := result.(cadence.Array)

	require.Equal(t, 3, len(array.Values))

	// Check key1
	require.IsType(t, cadence.Optional{}, array.Values[0])
	optionalValue := array.Values[0].(cadence.Optional)

	require.IsType(t, cadence.Struct{}, optionalValue.Value)
	builtinStruct := optionalValue.Value.(cadence.Struct)

	require.Equal(t, 1, len(builtinStruct.Fields))
	assert.Equal(t, cadence.NewInt(HashAlgorithmKMAC128.RawValue()), builtinStruct.Fields[0])

	// Check key2
	require.IsType(t, cadence.Optional{}, array.Values[1])
	optionalValue = array.Values[1].(cadence.Optional)

	require.IsType(t, cadence.Struct{}, optionalValue.Value)
	builtinStruct = optionalValue.Value.(cadence.Struct)

	require.Equal(t, 1, len(builtinStruct.Fields))
	assert.Equal(t, cadence.NewInt(HashAlgorithmKMAC128.RawValue()), builtinStruct.Fields[0])

	// Check key3
	require.IsType(t, cadence.Optional{}, array.Values[2])
	optionalValue = array.Values[2].(cadence.Optional)

	require.Nil(t, optionalValue.Value)
}

func TestSignatureAlgorithm(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): [SignatureAlgorithm?] {
			var key1: SignatureAlgorithm? = SignatureAlgorithm.BLSBLS12381

			var key2: SignatureAlgorithm? = SignatureAlgorithm(rawValue:2)

			var key3: SignatureAlgorithm? = SignatureAlgorithm(rawValue:5)
			return [key1, key2, key3]
		}
	`)

	runtimeInterface := &testRuntimeInterface{}

	result, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	require.NoError(t, err)

	require.IsType(t, cadence.Array{}, result)
	array := result.(cadence.Array)

	require.Equal(t, 3, len(array.Values))

	// Check key1
	require.IsType(t, cadence.Optional{}, array.Values[0])
	optionalValue := array.Values[0].(cadence.Optional)

	require.IsType(t, cadence.Struct{}, optionalValue.Value)
	builtinStruct := optionalValue.Value.(cadence.Struct)

	require.Equal(t, 1, len(builtinStruct.Fields))
	assert.Equal(t, cadence.NewInt(SignatureAlgorithmBLSBLS12381.RawValue()), builtinStruct.Fields[0])

	// Check key2
	require.IsType(t, cadence.Optional{}, array.Values[1])
	optionalValue = array.Values[1].(cadence.Optional)

	require.IsType(t, cadence.Struct{}, optionalValue.Value)
	builtinStruct = optionalValue.Value.(cadence.Struct)

	require.Equal(t, 1, len(builtinStruct.Fields))
	assert.Equal(t, cadence.NewInt(SignatureAlgorithmBLSBLS12381.RawValue()), builtinStruct.Fields[0])

	// Check key3
	require.IsType(t, cadence.Optional{}, array.Values[2])
	optionalValue = array.Values[2].(cadence.Optional)

	require.Nil(t, optionalValue.Value)
}

// Utility methods and types

var AccountKeyType = ExportedBuiltinType(sema.AccountKeyType)
var PublicKeyType = ExportedBuiltinType(sema.PublicKeyType)
var SignAlgoType = ExportedBuiltinType(sema.SignatureAlgorithmType)
var HashAlgoType = ExportedBuiltinType(sema.HashAlgorithmType)

func ExportedBuiltinType(internalType sema.Type) *cadence.StructType {
	return ExportType(internalType, map[sema.TypeID]cadence.Type{}).(*cadence.StructType)
}

func publicKeyExportedValue(keyBytes []byte, signAlgo sema.SignatureAlgorithm) cadence.Struct {
	byteArray := make([]cadence.Value, len(keyBytes))
	for index, value := range keyBytes {
		byteArray[index] = cadence.NewUInt8(value)
	}

	signAlgoValue := cadence.NewStruct([]cadence.Value{
		cadence.NewInt(signAlgo.RawValue()),
	}).WithType(SignAlgoType)

	return cadence.Struct{
		StructType: PublicKeyType,
		Fields: []cadence.Value{
			cadence.NewArray(byteArray),
			signAlgoValue,
		},
	}
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
			// key index
			cadence.NewInt(index),

			// PublicKey
			publicKeyExportedValue(publicKeyBytes, signAlgo),

			// Hash algo
			cadence.NewStruct([]cadence.Value{
				cadence.NewInt(hashAlgo.RawValue()),
			}).WithType(HashAlgoType),

			// weight
			weightUFix64,

			// isRevoked
			cadence.NewBool(isRevoked),
		},
	}
}

func getRuntimeInterface(storage *testAccountKeyStorage) *testRuntimeInterface {
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
			accountKey := AccountKey{
				KeyIndex:  index,
				PublicKey: publicKey,
				HashAlgo:  hashAlgo,
				Weight:    weight,
				IsRevoked: false,
			}

			storage.keys = append(storage.keys, accountKey)
			storage.returnedKey = accountKey
			return &accountKey, nil
		},

		getAccountKey: func(address Address, index int) (*AccountKey, error) {
			if index >= len(storage.keys) {
				return nil, nil
			}

			accountKey := storage.keys[index]
			storage.returnedKey = accountKey
			return &accountKey, nil
		},

		removeAccountKey: func(address Address, index int) (*AccountKey, error) {
			accountKey := storage.keys[index]
			accountKey.IsRevoked = true

			storage.keys[index] = accountKey
			storage.returnedKey = accountKey

			return &accountKey, nil
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
							signAlgo: SignatureAlgorithm.ECDSA_P256
						)

						var addedKey: AccountKey = signer.keys.add(
							publicKey: key,
							hashAlgo: HashAlgorithm.SHA3_256,
							weight: 100.0
						)
					}
				}`,
		args: []cadence.Value{},
	}

	err := executeTransaction(test, runtime, runtimeInterface)
	require.NoError(t, err)
}

func executeTransaction(test accountKeyTestCase, runtime Runtime, runtimeInterface *testRuntimeInterface) error {
	args := encodeArgs(test.args)
	err := runtime.ExecuteTransaction(
		Script{
			Source:    []byte(test.code),
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	return err
}

func executeScript(test accountKeyTestCase, runtime Runtime, runtimeInterface *testRuntimeInterface) (cadence.Value, error) {
	args := encodeArgs(test.args)
	value, err := runtime.ExecuteScript(
		Script{
			Source:    []byte(test.code),
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	return value, err
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
	keys []AccountKey
}

func newTestAccountKeyStorage() *testAccountKeyStorage {
	return &testAccountKeyStorage{
		events: make([]cadence.Event, 0),
		keys:   make([]AccountKey, 0),
	}
}

type testAccountKeyStorage struct {
	events      []cadence.Event
	keys        []AccountKey
	returnedKey AccountKey
}
