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

func TestAccountKeyCreation(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
		pub fun main(): AccountKey {
			let key = AccountKey(
				PublicKey2(
					publicKey: "0102".decodeHex(),
					signAlgo: "ECDSA_P256"
				),
				hashAlgo: "SHA3_256",
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

func TestAuthAccountAddPublicKey(t *testing.T) {
	t.Parallel()

	runtime := NewInterpreterRuntime()

	keyA := newPublicKeyExportedValue([]byte{1, 2, 3}, sema.ECDSAP256)
	keyB := newPublicKeyExportedValue([]byte{4, 5, 6}, sema.ECDSASecp256k1)
	keys := cadence.NewArray([]cadence.Value{keyA, keyB})

	var tests = []TestCase{
		{
			name: "Single key",
			code: `
				transaction(key: PublicKey2) {
					prepare(signer: AuthAccount) {
						let acct = AuthAccount(payer: signer)	
						acct.keys.add(
							publicKey: key,
							hashAlgo: HashAlgorithm2.SHA3_256,
							weight: 100.0
						)
					}
				}`,
			keyCount: 1,
			args:     []cadence.Value{keyA},
			keys: []*AccountKey{
				{
					KeyIndex: 0,
					PublicKey: &PublicKey{
						PublicKey: []byte{1, 2, 3},
						SignAlgo:  sema.ECDSAP256,
					},
					HashAlgo:  sema.SHA3_256,
					Weight:    100,
					IsRevoked: false,
				},
			},
		},
		{
			name: "Multiple keys",
			code: `
				transaction(keys: [PublicKey2]) {
					prepare(signer: AuthAccount) {
						let acct = AuthAccount(payer: signer)	
						for key in keys {
							acct.keys.add(
								publicKey: key,
								hashAlgo: HashAlgorithm2.SHA3_256,
								weight: 100.0
							)
						}
					}
				}
			`,
			keyCount: 2,
			args:     []cadence.Value{keys},
			keys: []*AccountKey{
				{
					KeyIndex: 0,
					PublicKey: &PublicKey{
						PublicKey: []byte{1, 2, 3},
						SignAlgo:  sema.ECDSAP256,
					},
					HashAlgo:  sema.SHA3_256,
					Weight:    100,
					IsRevoked: false,
				},
				{
					KeyIndex: 1,
					PublicKey: &PublicKey{
						PublicKey: []byte{4, 5, 6},
						SignAlgo:  sema.ECDSASecp256k1,
					},
					HashAlgo:  sema.SHA3_256,
					Weight:    100,
					IsRevoked: false,
				},
			},
		},
	}

	for _, test := range tests {
		storage := newStorage()
		runtimeInterface := getRuntimeInterface(storage)

		t.Run(test.name, func(t *testing.T) {
			err := execute(test, runtime, runtimeInterface)

			require.NoError(t, err)
			assert.Len(t, test.keys, test.keyCount)
			assert.Equal(t, test.keys, storage.keys)

			assert.EqualValues(t, stdlib.AccountCreatedEventType.ID(), storage.events[0].Type().ID())

			for _, event := range storage.events[1:] {
				assert.EqualValues(t, stdlib.AccountKeyAddedEventType.ID(), event.Type().ID())
			}
		})
	}
}

func TestAuthAccountAddPublicKeyErrors(t *testing.T) {
	t.Parallel()

	runtime := NewInterpreterRuntime()

	var tests = []TestCase{
		{
			name: "AccountKey as transaction param",
			code: `
				transaction(keys: [AccountKey]) {
					prepare(signer: AuthAccount) {
					}
				}
			`,
			err: "transaction parameter must be storable: `[AccountKey]`",
		},
	}

	for _, test := range tests {
		storage := newStorage()
		runtimeInterface := getRuntimeInterface(storage)

		t.Run(test.name, func(t *testing.T) {
			err := execute(test, runtime, runtimeInterface)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.err)
		})
	}
}

var addedAccountKey = &AccountKey{
	KeyIndex: 0,
	PublicKey: &PublicKey{
		PublicKey: []byte{1, 2},
		SignAlgo:  sema.ECDSAP256,
	},
	HashAlgo:  sema.SHA2_256,
	Weight:    100,
	IsRevoked: false,
}

var revokedAccountKey = &AccountKey{
	KeyIndex: 0,
	PublicKey: &PublicKey{
		PublicKey: []byte{1, 2},
		SignAlgo:  sema.ECDSAP256,
	},
	HashAlgo:  sema.SHA2_256,
	Weight:    100,
	IsRevoked: true,
}

func TestAuthAccountKeysAdd(t *testing.T) {
	storage := newStorage()
	runtime := NewInterpreterRuntime()
	runtimeInterface := getRuntimeInterface(storage)

	addAuthAccountKey(t, runtime, runtimeInterface)

	assert.Equal(t, []*AccountKey{addedAccountKey}, storage.keys)
	assert.Equal(t, addedAccountKey, storage.returnedKey)
}

func TestAuthAccountKeysGet(t *testing.T) {
	storage := newStorage()
	runtime := NewInterpreterRuntime()
	runtimeInterface := getRuntimeInterface(storage)

	addAuthAccountKey(t, runtime, runtimeInterface)

	test := TestCase{
		name: "Add key",
		code: `
				transaction {
					prepare(signer: AuthAccount) {
						signer.keys.get(keyIndex: 0)
					}
				}`,
		args: []cadence.Value{},
	}

	err := execute(test, runtime, runtimeInterface)
	require.NoError(t, err)

	assert.Equal(t, []*AccountKey{addedAccountKey}, storage.keys)
	assert.Equal(t, addedAccountKey, storage.returnedKey)
}

func TestAuthAccountKeysRevoke(t *testing.T) {
	storage := newStorage()
	runtime := NewInterpreterRuntime()
	runtimeInterface := getRuntimeInterface(storage)

	addAuthAccountKey(t, runtime, runtimeInterface)

	test := TestCase{
		name: "Add key",
		code: `
				transaction {
					prepare(signer: AuthAccount) {
						signer.keys.revoke(keyIndex: 0)
					}
				}`,
		args: []cadence.Value{},
	}

	err := execute(test, runtime, runtimeInterface)
	require.NoError(t, err)

	assert.Equal(t, []*AccountKey{revokedAccountKey}, storage.keys)
	assert.Equal(t, revokedAccountKey, storage.returnedKey)
}

func TestSignatureAlgorithm(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	//script := []byte(`
	//	pub fun main(): SignatureAlgorithm2? {
	//		var key1: Color = Color.red
	//		var key2: SignatureAlgorithm2? = SignatureAlgorithm2.ECDSA_P256
	//
	//		return key2
	//  	}
	//
	//	pub enum Color: UInt8 {
	//		pub case red
	//		pub case green
	//		pub case blue
	//	}
	//`)

	script := []byte(`
		pub fun main(): HashAlgorithm2? {
			var key1: Color = Color.red
			var key2: HashAlgorithm2 = HashAlgorithm2.SHA2_256

			var key3: HashAlgorithm2? = HashAlgorithm2(rawValue:1)

			return key3
      	}

		pub enum Color: UInt8 {
			pub case red
			pub case green
			pub case blue
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
	fmt.Println(result)
}

// Utility methods

func getRuntimeInterface(storage *Storage) *testRuntimeInterface {
	return &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		createAccount: func(payer Address) (address Address, err error) {
			return Address{42}, nil
		},
		addAccountKey: func(address Address, publicKey *PublicKey, hashAlgo HashingAlgorithm, weight int) (*AccountKey, error) {
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
			accountKey := storage.keys[index]
			storage.returnedKey = accountKey
			return accountKey, nil
		},

		removeAccountKey: func(address Address, index int) (*AccountKey, error) {
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
	test := TestCase{
		name: "Add key",
		code: `
				transaction {
					prepare(signer: AuthAccount) {
						let key = PublicKey2(
							publicKey: "0102".decodeHex(),
							signAlgo: SignatureAlgorithm2.ECDSA_P256
						)

						signer.keys.add(
							publicKey: key,
							hashAlgo: HashAlgorithm2.SHA2_256,
							weight: 100.0
						)
					}
				}`,
		args: []cadence.Value{},
	}

	err := execute(test, runtime, runtimeInterface)
	require.NoError(t, err)
}

func execute(test TestCase, runtime Runtime, runtimeInterface *testRuntimeInterface) error {
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

func newPublicKeyExportedValue(keyBytes []byte, signAlgo sema.SigningAlgorithm) cadence.BuiltinStruct {
	byteArray := make([]cadence.Value, len(keyBytes))
	for index, value := range keyBytes {
		byteArray[index] = cadence.NewUInt8(value)
	}

	signAlgoValue := cadence.NewBuiltinStruct([]cadence.Value{
		cadence.NewInt(signAlgo.RawValue()),
	}).WithType(SignAlgoType)

	return cadence.BuiltinStruct{
		StructType: PublicKeyType,
		Fields: []cadence.Value{
			cadence.NewArray(byteArray),
			signAlgoValue,
		},
	}
}

var PublicKeyType = func() *cadence.BuiltinStructType {
	var fields = []cadence.Field{
		{
			Identifier: sema.PublicKeyPublicKeyField,
			Type:       cadence.VariableSizedArrayType{ElementType: cadence.Int8Type{}},
		},
		{
			Identifier: sema.PublicKeySignAlgoField,
			Type:       SignAlgoType,
		},
	}

	return &cadence.BuiltinStructType{
		QualifiedIdentifier: sema.PublicKeyTypeName,
		Fields:              fields,
	}
}()

var SignAlgoType = func() *cadence.BuiltinStructType {
	var fields = []cadence.Field{
		{
			Identifier: sema.EnumRawValueFieldName,
			Type:       cadence.Int8Type{},
		},
	}

	return &cadence.BuiltinStructType{
		QualifiedIdentifier: sema.SignatureAlgorithmTypeName,
		Fields:              fields,
	}
}()

type TestCase struct {
	name     string
	code     string
	keyCount int
	args     []cadence.Value
	keys     []*AccountKey
	err      string
}

func newStorage() *Storage {
	return &Storage{
		events: make([]cadence.Event, 0),
		keys:   make([]*AccountKey, 0),
	}
}

type Storage struct {
	events      []cadence.Event
	keys        []*AccountKey
	returnedKey *AccountKey
}
