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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
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

	result, err := runtime.ExecuteScript(script, nil, runtimeInterface, testLocation)
	require.NoError(t, err)

	assert.Equal(t,
		cadence.NewString("ECDSA_P256"),
		result,
	)
}

func TestRuntimeCrypto_verify(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	script := []byte(`
      import Crypto

      pub fun main(): Bool {
          let publicKey = Crypto.PublicKey(
              publicKey: [1, 2],
              signatureAlgorithm: Crypto.ECDSA_P256
          )

          let keyList = Crypto.KeyList()
          keyList.add(
              publicKey,
              hashAlgorithm: Crypto.SHA3_256,
              weight: 1.0
          )

          let signatureSet = [
              Crypto.KeyListSignature(
                  keyIndex: 0,
                  signature: [3, 4]
              )
          ]

          return keyList.isValid(
              signatureSet: signatureSet,
              signedData: [5, 6]
          )
      }
    `)

	called := false

	runtimeInterface := &testRuntimeInterface{
		verifySignature: func(
			signature []byte,
			tag []byte,
			signedData []byte,
			publicKey []byte,
			signatureAlgorithm string,
			hashAlgorithm string,
		) bool {
			called = true
			assert.Equal(t, []byte{3, 4}, signature)
			assert.Equal(t, []byte("FLOW-V0.0-user"), tag)
			assert.Equal(t, []byte{5, 6}, signedData)
			assert.Equal(t, []byte{1, 2}, publicKey)
			assert.Equal(t, "ECDSA_P256", signatureAlgorithm)
			assert.Equal(t, "SHA3_256", hashAlgorithm)
			return true
		},
	}

	result, err := runtime.ExecuteScript(script, nil, runtimeInterface, testLocation)
	require.NoError(t, err)

	assert.Equal(t,
		cadence.NewBool(true),
		result,
	)

	assert.True(t, called)
}
