/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/stdlib"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
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
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
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
			log: func(message string) {
				logs = append(logs, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
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
			pk *stdlib.PublicKey,
		) error {
			return nil
		},
		bLSVerifyPOP: func(
			pk *stdlib.PublicKey,
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

func TestBLSAggregateSignatures(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`

      pub fun main(): [UInt8] {
        return BLS.aggregateSignatures([
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
		blsAggregateSignatures: func(
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
		}).WithType(cadence.VariableSizedArrayType{
			ElementType: cadence.UInt8Type{},
		}),
		result,
	)

	assert.True(t, called)
}

func TestBLSAggregatePublicKeys(t *testing.T) {

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
        return BLS.aggregatePublicKeys([k1, k2])
      }
    `)

	called := false

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		validatePublicKey: func(
			pk *stdlib.PublicKey,
		) error {
			return nil
		},
		blsAggregatePublicKeys: func(
			keys []*stdlib.PublicKey,
		) (*stdlib.PublicKey, error) {
			assert.Equal(t, len(keys), 2)
			ret := make([]byte, 0, len(keys))
			for _, key := range keys {
				ret = append(ret, key.PublicKey...)
			}
			called = true
			return &stdlib.PublicKey{
				PublicKey: ret,
				SignAlgo:  SignatureAlgorithmBLS_BLS12_381,
			}, nil
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
		}).WithType(cadence.VariableSizedArrayType{
			ElementType: cadence.UInt8Type{},
		}),
		result.(cadence.Optional).Value.(cadence.Struct).Fields[0],
	)

	assert.True(t, called)
}

func getCadenceValueArrayFromHexStr(t *testing.T, inp string) cadence.Value {
	bytes, err := hex.DecodeString(inp)
	require.NoError(t, err)

	cadenceValue := make([]cadence.Value, len(bytes))
	for i, b := range bytes {
		cadenceValue[i] = cadence.NewUInt8(b)
	}

	return cadence.NewArray(cadenceValue)
}

// TestTraversingMerkleProof tests combination of KECCAK_256 hashing
// and RLP decoding
//
// Warning!!! this code is only here to test functionality of utility methods
// and should not be used as a sample code for Merkle Proof Verification,
// for proper verification you need extra steps such as checking if the leaf content matches
// what you're expecting and etc...
func TestTraversingMerkleProof(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	script := []byte(`
        pub fun main(rootHash: [UInt8], address: [UInt8], accountProof: [[UInt8]]){

        let path = HashAlgorithm.KECCAK_256.hash(address)
     
        var nibbles: [UInt8]  = [] 

        for b in path {
            nibbles.append(b >> 4)
            nibbles.append(b % 16)
        }

        var nibbleIndex = 0
        var expectedNodeHash = rootHash 

        for encodedNode in accountProof {
            log(nibbleIndex)
            let nodeHash = HashAlgorithm.KECCAK_256.hash(encodedNode) 

            // verify that expected node hash (from a higher level or given root hash)
            // matches the hash of this level

            if nodeHash.length != expectedNodeHash.length {
                panic("invalid proof")
            }

            for i, c in nodeHash {
                if c != expectedNodeHash[i] {
                    panic("invalid proof")
                }
            }

            let encodedChildren = RLP.decodeList(encodedNode)
            let encodedChild = encodedChildren[nibbles[nibbleIndex]]
            expectedNodeHash = RLP.decodeString(encodedChild)
            log(nibbles[nibbleIndex])
            nibbleIndex = nibbleIndex + 1
        }
     }
    `)

	accountProofInHex := []string{
		"f90211a090dcaf88c40c7bbc95a912cbdde67c175767b31173df9ee4b0d733bfdd511c43a0babe369f6b12092f49181ae04ca173fb68d1a5456f18d20fa32cba73954052bda0473ecf8a7e36a829e75039a3b055e51b8332cbf03324ab4af2066bbd6fbf0021a0bbda34753d7aa6c38e603f360244e8f59611921d9e1f128372fec0d586d4f9e0a04e44caecff45c9891f74f6a2156735886eedf6f1a733628ebc802ec79d844648a0a5f3f2f7542148c973977c8a1e154c4300fec92f755f7846f1b734d3ab1d90e7a0e823850f50bf72baae9d1733a36a444ab65d0a6faaba404f0583ce0ca4dad92da0f7a00cbe7d4b30b11faea3ae61b7f1f2b315b61d9f6bd68bfe587ad0eeceb721a07117ef9fc932f1a88e908eaead8565c19b5645dc9e5b1b6e841c5edbdfd71681a069eb2de283f32c11f859d7bcf93da23990d3e662935ed4d6b39ce3673ec84472a0203d26456312bbc4da5cd293b75b840fc5045e493d6f904d180823ec22bfed8ea09287b5c21f2254af4e64fca76acc5cd87399c7f1ede818db4326c98ce2dc2208a06fc2d754e304c48ce6a517753c62b1a9c1d5925b89707486d7fc08919e0a94eca07b1c54f15e299bd58bdfef9741538c7828b5d7d11a489f9c20d052b3471df475a051f9dd3739a927c89e357580a4c97b40234aa01ed3d5e0390dc982a7975880a0a089d613f26159af43616fd9455bb461f4869bfede26f2130835ed067a8b967bfb80",
		"f90211a0395d87a95873cd98c21cf1df9421af03f7247880a2554e20738eec2c7507a494a0bcf6546339a1e7e14eb8fb572a968d217d2a0d1f3bc4257b22ef5333e9e4433ca012ae12498af8b2752c99efce07f3feef8ec910493be749acd63822c3558e6671a0dbf51303afdc36fc0c2d68a9bb05dab4f4917e7531e4a37ab0a153472d1b86e2a0ae90b50f067d9a2244e3d975233c0a0558c39ee152969f6678790abf773a9621a01d65cd682cc1be7c5e38d8da5c942e0a73eeaef10f387340a40a106699d494c3a06163b53d956c55544390c13634ea9aa75309f4fd866f312586942daf0f60fb37a058a52c1e858b1382a8893eb9c1f111f266eb9e21e6137aff0dddea243a567000a037b4b100761e02de63ea5f1fcfcf43e81a372dafb4419d126342136d329b7a7ba032472415864b08f808ba4374092003c8d7c40a9f7f9fe9cc8291f62538e1cc14a074e238ff5ec96b810364515551344100138916594d6af966170ff326a092fab0a0d31ac4eef14a79845200a496662e92186ca8b55e29ed0f9f59dbc6b521b116fea090607784fe738458b63c1942bba7c0321ae77e18df4961b2bc66727ea996464ea078f757653c1b63f72aff3dcc3f2a2e4c8cb4a9d36d1117c742833c84e20de994a0f78407de07f4b4cb4f899dfb95eedeb4049aeb5fc1635d65cf2f2f4dfd25d1d7a0862037513ba9d45354dd3e36264aceb2b862ac79d2050f14c95657e43a51b85c80",
		"f90171a04ad705ea7bf04339fa36b124fa221379bd5a38ffe9a6112cb2d94be3a437b879a08e45b5f72e8149c01efcb71429841d6a8879d4bbe27335604a5bff8dfdf85dcea00313d9b2f7c03733d6549ea3b810e5262ed844ea12f70993d87d3e0f04e3979ea0b59e3cdd6750fa8b15164612a5cb6567cdfb386d4e0137fccee5f35ab55d0efda0fe6db56e42f2057a071c980a778d9a0b61038f269dd74a0e90155b3f40f14364a08538587f2378a0849f9608942cf481da4120c360f8391bbcc225d811823c6432a026eac94e755534e16f9552e73025d6d9c30d1d7682a4cb5bd7741ddabfd48c50a041557da9a74ca68da793e743e81e2029b2835e1cc16e9e25bd0c1e89d4ccad6980a041dda0a40a21ade3a20fcd1a4abb2a42b74e9a32b02424ff8db4ea708a5e0fb9a09aaf8326a51f613607a8685f57458329b41e938bb761131a5747e066b81a0a16808080a022e6cef138e16d2272ef58434ddf49260dc1de1f8ad6dfca3da5d2a92aaaadc58080",
		"f851808080a009833150c367df138f1538689984b8a84fc55692d3d41fe4d1e5720ff5483a6980808080808080808080a0a319c1c415b271afc0adcb664e67738d103ac168e0bc0b7bd2da7966165cb9518080",
	}

	rootHash := getCadenceValueArrayFromHexStr(t, "d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544")

	addressInHex := "1234567890123456789012345678901234567890"

	address := getCadenceValueArrayFromHexStr(t, addressInHex)

	accountProof := cadence.NewArray([]cadence.Value{
		// first node encoded
		getCadenceValueArrayFromHexStr(t, accountProofInHex[0]),
		// second node encoded
		getCadenceValueArrayFromHexStr(t, accountProofInHex[1]),
		// third node encoded
		getCadenceValueArrayFromHexStr(t, accountProofInHex[2]),
		// forth node encoded
		getCadenceValueArrayFromHexStr(t, accountProofInHex[3]),
	})

	storage := newTestLedger(nil, nil)

	var logMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		hash: func(
			data []byte,
			tag string,
			hashAlgorithm HashAlgorithm,
		) ([]byte, error) {
			dataInHex := hex.EncodeToString(data)
			assert.Equal(t, HashAlgorithmKECCAK_256, hashAlgorithm)
			if dataInHex == accountProofInHex[0] {
				return hex.DecodeString("d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544")
			}
			if dataInHex == accountProofInHex[1] {
				return hex.DecodeString("9287b5c21f2254af4e64fca76acc5cd87399c7f1ede818db4326c98ce2dc2208")
			}
			if dataInHex == accountProofInHex[2] {
				return hex.DecodeString("6163b53d956c55544390c13634ea9aa75309f4fd866f312586942daf0f60fb37")
			}
			if dataInHex == accountProofInHex[3] {
				return hex.DecodeString("41dda0a40a21ade3a20fcd1a4abb2a42b74e9a32b02424ff8db4ea708a5e0fb9")
			}
			// hash value for address 1234567890123456789012345678901234567890
			if dataInHex == addressInHex {
				return hex.DecodeString("b6979620706f8c652cfb6bf6e923f5156eadd5abaf4022a0b19d52ada089475f")
			}

			return nil, errors.New("Unknown input to the hash method")
		},
		log: func(message string) {
			logMessages = append(logMessages, message)
		},
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
			Arguments: encodeArgs([]cadence.Value{rootHash, address, accountProof}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
	require.NoError(t, err)
	require.Equal(t,
		[]string{"0", "11", "1", "6", "2", "9", "3", "7"},
		logMessages,
	)
}
