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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/encoding/json"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCrypto_verify(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
      import Crypto

      pub fun main(): Bool {
          let publicKey = PublicKey(
              publicKey: "0102".decodeHex(),
              signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
          )

          let keyList = Crypto.KeyList()
          keyList.add(
              publicKey,
              hashAlgorithm: HashAlgorithm.SHA3_256,
              weight: 1.0
          )

          let signatureSet = [
              Crypto.KeyListSignature(
                  keyIndex: 0,
                  signature: "0304".decodeHex()
              )
          ]

          return keyList.verify(
              signatureSet: signatureSet,
              signedData: "0506".decodeHex()
          )
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		verifySignature: func(
			signature []byte,
			tag string,
			signedData []byte,
			publicKey []byte,
			signatureAlgorithm SignatureAlgorithm,
			hashAlgorithm HashAlgorithm,
		) (bool, error) {
			called = true
			assert.Equal(t, []byte{3, 4}, signature)
			assert.Equal(t, "FLOW-V0.0-user", tag)
			assert.Equal(t, []byte{5, 6}, signedData)
			assert.Equal(t, []byte{1, 2}, publicKey)
			assert.Equal(t, SignatureAlgorithmECDSA_P256, signatureAlgorithm)
			assert.Equal(t, HashAlgorithmSHA3_256, hashAlgorithm)
			return true, nil
		},
	}
	addPublicKeyValidation(runtimeInterface, nil)

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

	assert.Equal(t,
		cadence.NewBool(true),
		result,
	)

	assert.True(t, called)
}

func TestRuntimeHashAlgorithm_hash(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	executeScript := func(code string, inter Interface) (cadence.Value, error) {
		return runtime.ExecuteScript(
			Script{
				Source: []byte(code),
			},
			Context{
				Interface: inter,
				Location:  utils.TestLocation,
			},
		)
	}

	t.Run("hash", func(t *testing.T) {
		script := `
            pub fun main() {
                log(HashAlgorithm.SHA3_256.hash("01020304".decodeHex()))
            }
        `

		called := false

		var loggedMessages []string

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			hash: func(
				data []byte,
				tag string,
				hashAlgorithm HashAlgorithm,
			) ([]byte, error) {
				called = true
				assert.Equal(t, []byte{1, 2, 3, 4}, data)
				assert.Equal(t, HashAlgorithmSHA3_256, hashAlgorithm)
				return []byte{5, 6, 7, 8}, nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		assert.Equal(t,
			[]string{
				"[5, 6, 7, 8]",
			},
			loggedMessages,
		)

		assert.True(t, called)
	})

	t.Run("hash - check tag", func(t *testing.T) {
		script := `
            pub fun main() {
                HashAlgorithm.SHA3_256.hash("01020304".decodeHex())
            }
        `

		called := false
		hashTag := "non-empty-string"

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			hash: func(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error) {
				called = true
				hashTag = tag
				return nil, nil
			},
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, called)
		assert.Empty(t, hashTag)
	})

	t.Run("hashWithTag - check tag", func(t *testing.T) {
		script := `
            pub fun main() {
                HashAlgorithm.SHA3_256.hashWithTag(
                    "01020304".decodeHex(),
                    tag: "some-tag"
                )
            }
        `

		called := false
		hashTag := ""

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			hash: func(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error) {
				called = true
				hashTag = tag
				return nil, nil
			},
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, called)
		assert.Equal(t, "some-tag", hashTag)
	})
}

func TestRuntimeHashingAlgorithmExport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{}

	testHashAlgorithm := func(algo sema.CryptoAlgorithm) {
		script := fmt.Sprintf(`
              pub fun main(): HashAlgorithm {
                  return HashAlgorithm.%s
              }
            `,
			algo.Name(),
		)

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		require.IsType(t, cadence.Enum{}, value)
		enumValue := value.(cadence.Enum)

		require.Len(t, enumValue.Fields, 1)
		assert.Equal(t, cadence.NewUInt8(algo.RawValue()), enumValue.Fields[0])
	}

	for _, algo := range sema.HashAlgorithms {
		testHashAlgorithm(algo)
	}
}

func TestRuntimeSignatureAlgorithmExport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{}

	testSignatureAlgorithm := func(algo sema.CryptoAlgorithm) {
		script := fmt.Sprintf(`
              pub fun main(): SignatureAlgorithm {
                  return SignatureAlgorithm.%s
              }
            `,
			algo.Name(),
		)

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		require.IsType(t, cadence.Enum{}, value)
		enumValue := value.(cadence.Enum)

		require.Len(t, enumValue.Fields, 1)
		assert.Equal(t, cadence.NewUInt8(algo.RawValue()), enumValue.Fields[0])
	}

	for _, algo := range sema.SignatureAlgorithms {
		testSignatureAlgorithm(algo)
	}
}

func TestRuntimeSignatureAlgorithmImport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()
	runtimeInterface := &testRuntimeInterface{
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(b)
		},
	}

	const script = `
      pub fun main(algo: SignatureAlgorithm): UInt8 {
          return algo.rawValue
      }
    `

	testSignatureAlgorithm := func(algo sema.CryptoAlgorithm) {

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewEnum([]cadence.Value{
						cadence.UInt8(algo.RawValue()),
					}).WithType(&cadence.EnumType{
						QualifiedIdentifier: "SignatureAlgorithm",
						RawType:             cadence.UInt8Type{},
						Fields: []cadence.Field{
							{
								Identifier: "rawValue",
								Type:       cadence.UInt8Type{},
							},
						},
					}),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewUInt8(algo.RawValue()),
			value,
		)
	}

	for _, algo := range sema.SignatureAlgorithms {
		testSignatureAlgorithm(algo)
	}
}

func TestRuntimeHashAlgorithmImport(t *testing.T) {

	t.Parallel()

	const script = `
      pub fun main(algo: HashAlgorithm): UInt8 {
          let data: [UInt8] = [1, 2, 3]
          log(algo.hash(data))
          log(algo.hashWithTag(data, tag: "some-tag"))

          return algo.rawValue
      }
    `

	testHashAlgorithm := func(algo sema.CryptoAlgorithm) {

		var logs []string
		var hashCalls int

		storage := newTestLedger(nil, nil)

		runtime := newTestInterpreterRuntime()
		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			hash: func(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error) {
				hashCalls++
				switch hashCalls {
				case 1:
					assert.Empty(t, tag)
				case 2:
					assert.Equal(t, "some-tag", tag)
				}
				return []byte{4, 5, 6}, nil
			},
			decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(b)
			},
			log: func(message string) {
				logs = append(logs, message)
			},
		}

		value, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: encodeArgs([]cadence.Value{
					cadence.NewEnum([]cadence.Value{
						cadence.UInt8(algo.RawValue()),
					}).WithType(&cadence.EnumType{
						QualifiedIdentifier: "HashAlgorithm",
						RawType:             cadence.UInt8Type{},
						Fields: []cadence.Field{
							{
								Identifier: "rawValue",
								Type:       cadence.UInt8Type{},
							},
						},
					}),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  utils.TestLocation,
			},
		)

		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewUInt8(algo.RawValue()),
			value,
		)
		assert.Equal(t,
			[]string{
				"[4, 5, 6]",
				"[4, 5, 6]",
			},
			logs,
		)
		assert.Equal(t, 2, hashCalls)
	}

	for _, algo := range sema.SignatureAlgorithms {
		testHashAlgorithm(algo)
	}
}

func TestBLSVerifyPoP(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): Bool {
          let publicKey = PublicKey(
              publicKey: "0102".decodeHex(),
              signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
          )

          return publicKey.verifyPoP([1, 2, 3, 4, 5])
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		validatePublicKey: func(
			pk *PublicKey,
		) (bool, error) {
			return true, nil
		},
		bLSVerifyPOP: func(
			pk *PublicKey,
			proof []byte,
		) (bool, error) {
			assert.Equal(t, pk.PublicKey, []byte{1, 2})
			called = true
			return true, nil
		},
	}
	addPublicKeyValidation(runtimeInterface, nil)

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

	assert.Equal(t,
		cadence.NewBool(true),
		result,
	)

	assert.True(t, called)
}

func TestBLSVerifyPoPInvalid(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): Bool {
          let publicKey = PublicKey(
              publicKey: "0102".decodeHex(),
              signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
          )

          return publicKey.verifyPoP([1, 2, 3, 4, 5])
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		validatePublicKey: func(
			pk *PublicKey,
		) (bool, error) {
			return false, nil
		},
		bLSVerifyPOP: func(
			pk *PublicKey,
			proof []byte,
		) (bool, error) {
			assert.Equal(t, pk.PublicKey, []byte{1, 2})
			called = true
			return true, nil
		},
	}

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

	assert.Equal(t,
		cadence.NewBool(false),
		result,
	)

	// key is invalid, so the interface function should never be called
	assert.False(t, called)
}

func TestBLSAggregateSignatures(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): [UInt8] {
		return AggregateBLSSignatures([
			  [1, 1, 1, 1, 1], 
			  [2, 2, 2, 2, 2],
			  [3, 3, 3, 3, 3],
			  [4, 4, 4, 4, 4],
			  [5, 5, 5, 5, 5]
			])!
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		aggregateBLSSignatures: func(
			sigs [][]byte,
		) ([]byte, error) {
			assert.Equal(t, len(sigs), 5)
			ret := make([]byte, 0, len(sigs[0]))
			for i, sig := range sigs {
				ret = append(ret, sig[i])
			}
			called = true
			return ret, nil
		},
	}

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

	assert.Equal(t,
		cadence.NewArray([]cadence.Value{
			cadence.UInt8(1),
			cadence.UInt8(2),
			cadence.UInt8(3),
			cadence.UInt8(4),
			cadence.UInt8(5),
		}),
		result,
	)

	assert.True(t, called)
}

func TestAggregateBLSPublicKeys(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): PublicKey? {
		let k1 = PublicKey(
			publicKey: "0102".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
		)
		let k2 = PublicKey(
			publicKey: "0102".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
		)
		return AggregateBLSPublicKeys([k1, k2])
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		validatePublicKey: func(
			pk *PublicKey,
		) (bool, error) {
			return true, nil
		},
		aggregateBLSPublicKeys: func(
			keys []*PublicKey,
		) (*PublicKey, error) {
			assert.Equal(t, len(keys), 2)
			ret := make([]byte, 0, len(keys))
			for _, key := range keys {
				ret = append(ret, key.PublicKey...)
			}
			called = true
			return &PublicKey{PublicKey: ret, SignAlgo: SignatureAlgorithmBLS_BLS12_381}, nil
		},
	}
	addPublicKeyValidation(runtimeInterface, nil)

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

	assert.Equal(t,
		cadence.NewArray([]cadence.Value{
			cadence.UInt8(1),
			cadence.UInt8(2),
			cadence.UInt8(1),
			cadence.UInt8(2),
		}),
		result.(cadence.Optional).Value.(cadence.Struct).Fields[0],
	)

	assert.True(t, called)
}

func TestAggregateBLSPublicKeysInvalid(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): PublicKey? {
		let k1 = PublicKey(
			publicKey: "0302".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
		)
		let k2 = PublicKey(
			publicKey: "0102".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.BLS_BLS12_381
		)
		return AggregateBLSPublicKeys([k1, k2])
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		validatePublicKey: func(
			pk *PublicKey,
		) (bool, error) {
			return pk.PublicKey[0] != 0x1, nil
		},
		aggregateBLSPublicKeys: func(
			keys []*PublicKey,
		) (*PublicKey, error) {
			assert.Equal(t, len(keys), 2)
			ret := make([]byte, 0, len(keys))
			for _, key := range keys {
				ret = append(ret, key.PublicKey...)
			}
			called = true
			return &PublicKey{PublicKey: ret, SignAlgo: SignatureAlgorithmBLS_BLS12_381}, nil
		},
	}

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

	assert.Equal(t,
		cadence.Optional{
			Value: cadence.Value(nil),
		},
		result,
	)

	// invalid public key will return nil before calling the interface function
	assert.False(t, called)
}
