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

	"github.com/onflow/cadence/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCrypto_verify(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

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

	storage := newTestStorage(nil, nil)

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

	runtime := NewInterpreterRuntime()

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

		storage := newTestStorage(nil, nil)

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

		storage := newTestStorage(nil, nil)

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

		storage := newTestStorage(nil, nil)

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

	runtime := NewInterpreterRuntime()
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

	runtime := NewInterpreterRuntime()
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

	runtime := NewInterpreterRuntime()
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

		storage := newTestStorage(nil, nil)

		runtime := NewInterpreterRuntime()
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
