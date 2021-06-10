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

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCrypto_import(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      import Crypto

      pub fun main(): String {
          return Crypto.ECDSA_P256.name
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
	assert.Contains(t, err.Error(), "value of type `Crypto` has no member `ECDSA_P256`")
}

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

          return keyList.isValid(
              signatureSet: signatureSet,
              signedData: "0506".decodeHex()
          )
      }
    `)

	called := false

	runtimeInterface := &testRuntimeInterface{
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

func TestRuntimeCrypto_hash(t *testing.T) {

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
            import Crypto

            pub fun main() {
                log(Crypto.hash("01020304".decodeHex(), algorithm: HashAlgorithm.SHA3_256))
            }
        `

		called := false

		var loggedMessages []string

		runtimeInterface := &testRuntimeInterface{
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
            import Crypto

            pub fun main() {
                Crypto.hash("01020304".decodeHex(), algorithm: HashAlgorithm.SHA3_256)
            }
        `

		called := false
		hashTag := "non-empty-string"

		runtimeInterface := &testRuntimeInterface{
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
            import Crypto

            pub fun main() {
                Crypto.hashWithTag(
                    "01020304".decodeHex(),
                    tag: "some-tag",
                    algorithm: HashAlgorithm.SHA3_256
                )
            }
        `

		called := false
		hashTag := ""

		runtimeInterface := &testRuntimeInterface{
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

	t.Run("hashWithTag - without tag", func(t *testing.T) {
		script := `
            import Crypto

            pub fun main() {
                Crypto.hashWithTag(
                    data: "01020304".decodeHex(),
                    algorithm: HashAlgorithm.SHA3_256
                )
            }
        `

		runtimeInterface := &testRuntimeInterface{}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect number of arguments")
	})
}

func TestHashingAlgorithms(t *testing.T) {

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

func TestSignatureAlgorithms(t *testing.T) {

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
