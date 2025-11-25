/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package runtime_test

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	flowsdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence-standard-transactions/transactions"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

var realCryptoContract = `
access(all) contract Crypto {

    access(all)
    fun hash(_ data: [UInt8], algorithm: HashAlgorithm): [UInt8] {
        return algorithm.hash(data)
    }

    access(all)
    fun hashWithTag(_ data: [UInt8], tag: String, algorithm: HashAlgorithm): [UInt8] {
        return algorithm.hashWithTag(data, tag: tag)
    }

    access(all)
    struct KeyListEntry {

        access(all)
        let keyIndex: Int

        access(all)
        let publicKey: PublicKey

        access(all)
        let hashAlgorithm: HashAlgorithm

        access(all)
        let weight: UFix64

        access(all)
        let isRevoked: Bool

        init(
            keyIndex: Int,
            publicKey: PublicKey,
            hashAlgorithm: HashAlgorithm,
            weight: UFix64,
            isRevoked: Bool
        ) {
            self.keyIndex = keyIndex
            self.publicKey = publicKey
            self.hashAlgorithm = hashAlgorithm
            self.weight = weight
            self.isRevoked = isRevoked
        }
    }

    access(all)
    struct KeyList {

        access(self)
        let entries: [KeyListEntry]

        init() {
            self.entries = []
        }

        /// Adds a new key with the given weight
        access(all)
        fun add(
            _ publicKey: PublicKey,
            hashAlgorithm: HashAlgorithm,
            weight: UFix64
        ): KeyListEntry {

            let keyIndex = self.entries.length
            let entry = KeyListEntry(
                keyIndex: keyIndex,
                publicKey: publicKey,
                hashAlgorithm: hashAlgorithm,
                weight: weight,
                isRevoked: false
            )
            self.entries.append(entry)
            return entry
        }

        /// Returns the key at the given index, if it exists.
        /// Revoked keys are always returned, but they have the isRevoked field set to true
        access(all)
        fun get(keyIndex: Int): KeyListEntry? {
            if keyIndex >= self.entries.length {
                return nil
            }

            return self.entries[keyIndex]
        }

        /// Marks the key at the given index revoked, but does not delete it
        access(all)
        fun revoke(keyIndex: Int) {
            if keyIndex >= self.entries.length {
                return
            }

            let currentEntry = self.entries[keyIndex]
            self.entries[keyIndex] = KeyListEntry(
                keyIndex: currentEntry.keyIndex,
                publicKey: currentEntry.publicKey,
                hashAlgorithm: currentEntry.hashAlgorithm,
                weight: currentEntry.weight,
                isRevoked: true
            )
        }

        /// Returns true if the given signatures are valid for the given signed data
        access(all)
        fun verify(
            signatureSet: [KeyListSignature],
            signedData: [UInt8],
            domainSeparationTag: String
        ): Bool {

            var validWeights: UFix64 = 0.0

            let seenKeyIndices: {Int: Bool} = {}

            for signature in signatureSet {

                // Ensure the key index is valid
                if signature.keyIndex >= self.entries.length {
                    return false
                }

                // Ensure this key index has not already been seen
                if seenKeyIndices[signature.keyIndex] ?? false {
                    return false
                }

                // Record the key index was seen
                seenKeyIndices[signature.keyIndex] = true

                // Get the actual key
                let key = self.entries[signature.keyIndex]

                // Ensure the key is not revoked
                if key.isRevoked {
                    return false
                }

                // Ensure the signature is valid
                if !key.publicKey.verify(
                    signature: signature.signature,
                    signedData: signedData,
                    domainSeparationTag: domainSeparationTag,
                    hashAlgorithm:key.hashAlgorithm
                ) {
                    return false
                }

                validWeights = validWeights + key.weight
            }

            return validWeights >= 1.0
        }
    }

    access(all)
    struct KeyListSignature {

        access(all)
        let keyIndex: Int

        access(all)
        let signature: [UInt8]

        init(keyIndex: Int, signature: [UInt8]) {
            self.keyIndex = keyIndex
            self.signature = signature
        }
    }
}`

type Transaction struct {
	Name  string
	Body  string
	Setup string
}

var testTransactions []Transaction

func createTransaction(name string, imports string, prepare string, setup string) Transaction {
	return Transaction{
		Name: name,
		Body: fmt.Sprintf(
			`
			// %s
			%s

			transaction(){
				prepare(signer: auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account) {
					%s
				}
			}`,
			name,
			imports,
			prepare,
		),
		Setup: fmt.Sprintf(
			`
			transaction(){
				prepare(signer: auth(Storage, Contracts, Keys, Inbox, Capabilities) &Account) {
					%s
				}
			}`,
			setup,
		),
	}
}

func init() {
	// create key transactions
	createKeyECDSAP256Transaction, err := transactions.CreateKeyECDSAP256Transaction(100)
	if err != nil {
		panic(err)
	}
	createKeyECDSAsecp256k1Transaction, err := transactions.CreateKeyECDSAsecp256k1Transaction(100)
	if err != nil {
		panic(err)
	}
	createKeyBLSBLS12381Transaction, err := transactions.CreateKeyBLSBLS12381Transaction(100)
	if err != nil {
		panic(err)
	}

	// verify signature transaction
	numKeys := uint64(15)
	message := []byte("hello world")

	rawKeys := make([]string, numKeys)
	signers := make([]crypto.Signer, numKeys)
	signatures := make([]string, numKeys)

	for i := 0; i < int(numKeys); i++ {
		seed := make([]byte, crypto.MinSeedLength)
		_, err := rand.Read(seed)
		if err != nil {
			panic(fmt.Errorf("failed to generate seed: %w", err))
		}

		privateKey, err := crypto.GeneratePrivateKey(crypto.ECDSA_P256, seed)
		if err != nil {
			panic(fmt.Errorf("failed to generate private key: %w", err))
		}
		rawKeys[i] = hex.EncodeToString(privateKey.PublicKey().Encode())
		sig, err := crypto.NewInMemorySigner(privateKey, crypto.SHA3_256)
		if err != nil {
			panic(fmt.Errorf("failed to generate signer: %w", err))
		}
		signers[i] = sig
	}

	for i := 0; i < int(numKeys); i++ {
		sig, err := flowsdk.SignUserMessage(signers[i], message)
		if err != nil {
			panic(fmt.Errorf("failed to sign message: %w", err))
		}
		signatures[i] = hex.EncodeToString(sig)
	}
	verifySignatureTransaction := transactions.VerifySignatureTransaction(numKeys, message, rawKeys, signatures)

	// bls
	blsAggregateKeysTransaction, err := transactions.AggregateBLSAggregateKeysTransaction(42)
	if err != nil {
		panic(err)
	}

	blsVerifyProofOfPossessionTransaction, err := transactions.BLSVerifyProofOfPossessionTransaction(8)
	if err != nil {
		panic(err)
	}

	testTransactions = []Transaction{
		createTransaction(
			"EmptyLoop",
			"",
			transactions.EmptyLoopTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"AssertTrue",
			"",
			transactions.AssertTrueTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerAddress",
			"",
			transactions.GetSignerAddressTransaction(50).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerPublicAccount",
			"",
			transactions.GetSignerPublicAccountTransaction(50).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerAccountBalance",
			"",
			transactions.GetSignerAccountBalanceTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerAccountAvailableBalance",
			"",
			transactions.GetSignerAccountAvailableBalanceTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerAccountStorageUsed",
			"",
			transactions.GetSignerAccountStorageUsedTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetSignerAccountStorageCapacity",
			"",
			transactions.GetSignerAccountStorageCapacityTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"BorrowSignerAccountFlowTokenVault",
			"import FungibleToken from 0x1\nimport FlowToken from 0x1",
			transactions.BorrowSignerAccountFlowTokenVaultTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"BorrowSignerAccountFungibleTokenReceiver",
			"import FungibleToken from 0x1\nimport FlowToken from 0x1",
			transactions.BorrowSignerAccountFungibleTokenReceiverTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"TransferTokensToSelf",
			"import FungibleToken from 0x1\nimport FlowToken from 0x1",
			transactions.TransferTokensToSelfTransaction(30).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"CreateNewAccount",
			"",
			transactions.CreateNewAccountTransaction(10).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"CreateNewAccountWithContract",
			"",
			transactions.CreateNewAccountWithContractTransaction(10).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DecodeHex",
			"",
			transactions.DecodeHexTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"RevertibleRandomNumber",
			"",
			transactions.RevertibleRandomTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"NumberToStringConversion",
			"",
			transactions.NumberToStringConversionTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ConcatenateString",
			"",
			transactions.ConcatenateStringTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"BorrowString",
			"",
			transactions.BorrowStringTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.BorrowStringTransaction.GetSetupTemplate(),
				transactions.StringArrayOfLen(20, 100),
			),
		),
		createTransaction(
			"CopyString",
			"",
			transactions.CopyStringTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.CopyStringTransaction.GetSetupTemplate(),
				transactions.StringArrayOfLen(20, 100),
			),
		),
		createTransaction(
			"CopyStringAndSaveDuplicate",
			"",
			transactions.CopyStringAndSaveADuplicateTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.CopyStringAndSaveADuplicateTransaction.GetSetupTemplate(),
				transactions.StringArrayOfLen(20, 100),
			),
		),
		createTransaction(
			"StoreAndLoadDictString",
			"",
			transactions.StoreAndLoadDictStringTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"StoreLoadAndDestroyDictString",
			"",
			transactions.StoreLoadAndDestroyDictStringTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.StoreLoadAndDestroyDictStringTransaction.GetSetupTemplate(),
				transactions.StringDictOfLen(100, 100),
			),
		),
		createTransaction(
			"BorrowDictString",
			"",
			transactions.BorrowDictStringTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.BorrowDictStringTransaction.GetSetupTemplate(),
				transactions.StringDictOfLen(100, 100),
			),
		),
		createTransaction(
			"CopyDictString",
			"",
			transactions.CopyDictStringTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.CopyDictStringTransaction.GetSetupTemplate(),
				transactions.StringDictOfLen(30, 100),
			),
		),
		createTransaction(
			"CopyDictStringAndSaveDuplicate",
			"",
			transactions.CopyDictStringAndSaveADuplicateTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.CopyDictStringAndSaveADuplicateTransaction.GetSetupTemplate(),
				transactions.StringDictOfLen(20, 100),
			),
		),
		createTransaction(
			"LoadDictAndDestroy",
			"",
			transactions.LoadDictAndDestroyItTransaction.GetPrepareBlock(),
			fmt.Sprintf(
				transactions.LoadDictAndDestroyItTransaction.GetSetupTemplate(),
				100,
			),
		),
		createTransaction(
			"AddKeyToAccount",
			"",
			transactions.AddKeyToAccountTransaction(50).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"AddAndRevokeKeyToAccount",
			"",
			transactions.AddAndRevokeKeyToAccountTransaction(40).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetAccountKey",
			"",
			transactions.GetAccountKeyTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetContracts",
			"",
			transactions.GetContractsTransaction(100).GetPrepareBlock(),
			transactions.GetContractsTransaction(100).GetSetupTemplate(),
		),
		createTransaction(
			"Hash",
			"import Crypto from 0x1",
			transactions.HashTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"StringToLower",
			"",
			transactions.StringToLowerTransaction(100, 100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetCurrentBlock",
			"",
			transactions.GetCurrentBlockTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetBlockAt",
			"",
			transactions.GetBlockAtTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DestroyResourceDictionary",
			"",
			transactions.DestroyResourceDictionaryTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ParseUFix64",
			"",
			transactions.ParseUFix64Transaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ParseFix64",
			"",
			transactions.ParseFix64Transaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ParseUInt64",
			"",
			transactions.ParseUInt64Transaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ParseInt64",
			"",
			transactions.ParseInt64Transaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ParseInt",
			"",
			transactions.ParseIntTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"IssueStorageCap",
			"",
			transactions.IssueStorageCapabilityTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"GetKeyCount",
			"",
			transactions.GetKeyCountTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"CreateKeyECDSAP256",
			"",
			createKeyECDSAP256Transaction.GetPrepareBlock(),
			"",
		),
		createTransaction(
			"CreateKeyECDSAsecp256k1",
			"",
			createKeyECDSAsecp256k1Transaction.GetPrepareBlock(),
			"",
		),
		createTransaction(
			"CreateKeyBLSBLS12381",
			"",
			createKeyBLSBLS12381Transaction.GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayInsert",
			"",
			transactions.ArrayInsertTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayInsertRemove",
			"",
			transactions.ArrayInsertRemoveTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayInsertSetRemove",
			"",
			transactions.ArrayInsertSetRemoveTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayInsertMap",
			"",
			transactions.ArrayInsertMapTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayInsertFilterRemove",
			"",
			transactions.ArrayInsertFilterTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayAppend",
			"",
			transactions.ArrayAppendTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DictInsert",
			"",
			transactions.DictInsertTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DictInsertRemove",
			"",
			transactions.DictInsertRemoveTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DictInsertSetRemove",
			"",
			transactions.DictInsertSetRemoveTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"DictIterCopy",
			"",
			transactions.DictIterCopyTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"ArrayCreateBatch",
			"",
			transactions.ArrayCreateBatchTransaction(100).GetPrepareBlock(),
			"",
		),
		createTransaction(
			"VerifySignatureTransaction",
			"import Crypto from 0x1",
			verifySignatureTransaction.GetPrepareBlock(),
			"",
		),
		createTransaction(
			"AggregateBLSAggregateKeys",
			"",
			blsAggregateKeysTransaction.GetPrepareBlock(),
			"",
		),
		createTransaction(
			"BLSVerifyProofOfPossession",
			"",
			blsVerifyProofOfPossessionTransaction.GetPrepareBlock(),
			"",
		),
	}
}

func benchmarkRuntimeTransactions(b *testing.B, useVM bool) {
	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	var signerAccount common.Address

	var environment Environment
	if useVM {
		environment = NewBaseVMEnvironment(Config{})
	} else {
		environment = NewBaseInterpreterEnvironment(Config{})
	}

	// Helper function to create a fresh runtime interface with isolated storage
	createRuntimeInterface := func() *TestRuntimeInterface {
		accountCodes := map[common.Location][]byte{}
		accountCounter := uint64(4)
		created := false
		signerAccount = contractsAddress

		return &TestRuntimeInterface{
			OnGetCode: func(location common.Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]common.Address, error) {
				return []common.Address{signerAccount}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(b),
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
			OnGetAccountBalance: func(address common.Address) (uint64, error) {
				return 0, nil
			},
			OnGetAccountAvailableBalance: func(address common.Address) (uint64, error) {
				return 0, nil
			},
			OnGetStorageUsed: func(address common.Address) (uint64, error) {
				return 0, nil
			},
			OnGetStorageCapacity: func(address common.Address) (uint64, error) {
				return 0, nil
			},
			OnCreateAccount: func(payer Address) (address Address, err error) {
				accountCounter++
				addressBytes := make([]byte, 8)
				binary.BigEndian.PutUint64(addressBytes, accountCounter)
				result := interpreter.NewUnmeteredAddressValueFromBytes(addressBytes)
				return result.ToAddress(), nil
			},
			OnValidatePublicKey: func(key *stdlib.PublicKey) error {
				return nil
			},
			OnVerifySignature: func(
				signature []byte,
				tag string,
				signedData []byte,
				publicKey []byte,
				signatureAlgorithm SignatureAlgorithm,
				hashAlgorithm HashAlgorithm,
			) (bool, error) {
				return true, nil
			},
			OnHash: func(data []byte, tag string, hashAlgorithm HashAlgorithm) ([]byte, error) {
				return data, nil
			},
			OnBLSVerifyPOP: func(pk *stdlib.PublicKey, s []byte) (bool, error) {
				return true, nil
			},
			OnBLSAggregateSignatures: func(sigs [][]byte) ([]byte, error) {
				if len(sigs) == 0 {
					return nil, fmt.Errorf("no signatures to aggregate")
				}
				return sigs[0], nil
			},
			OnBLSAggregatePublicKeys: func(keys []*stdlib.PublicKey) (*stdlib.PublicKey, error) {
				if len(keys) == 0 {
					return nil, fmt.Errorf("no keys to aggregate")
				}
				return keys[0], nil
			},
			OnAddAccountKey: func(address Address, publicKey *stdlib.PublicKey, hashAlgo HashAlgorithm, weight int) (*stdlib.AccountKey, error) {
				return &stdlib.AccountKey{PublicKey: publicKey, HashAlgo: hashAlgo, Weight: weight}, nil
			},
			OnGetAccountKey: func(address Address, index uint32) (*stdlib.AccountKey, error) {
				return &stdlib.AccountKey{KeyIndex: index, PublicKey: &stdlib.PublicKey{}}, nil
			},
			OnRemoveAccountKey: func(address Address, index uint32) (*stdlib.AccountKey, error) {
				return &stdlib.AccountKey{KeyIndex: index, PublicKey: &stdlib.PublicKey{}}, nil
			},
			OnAccountKeysCount: func(address Address) (uint32, error) {
				return 1, nil
			},
			OnGetAccountContractNames: func(address Address) ([]string, error) {
				if created {
					// ensures GetContractsTransaction only creates the contracts once
					return []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
						"11", "12", "13", "14", "15", "16", "17", "18", "19", "20"}, nil
				}
				created = true
				return []string{}, nil
			},
		}
	}

	// Helper function to setup a fresh runtime with contracts deployed
	setupRuntime := func() (TestRuntime, func() common.TransactionLocation, *TestRuntimeInterface) {

		runtimeInterface := createRuntimeInterface()
		signerAccount = contractsAddress

		runtime := NewTestRuntime()
		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy Fungible Token contract
		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
					"FungibleToken",
					[]byte(modifiedFungibleTokenContractInterface),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(b, err)

		// Deploy Flow Token contract
		err = runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction("FlowToken", []byte(modifiedFlowContract)),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(b, err)

		// Deploy Crypto contract
		err = runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction("Crypto", []byte(realCryptoContract)),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(b, err)

		// Setup both user accounts for Flow Token
		for _, address := range []common.Address{
			senderAddress,
			receiverAddress,
		} {
			signerAccount = address

			err = runtime.ExecuteTransaction(
				Script{
					Source: []byte(realSetupFlowTokenAccountTransaction),
				},
				Context{
					Interface:   runtimeInterface,
					Location:    nextTransactionLocation(),
					Environment: environment,
					UseVM:       useVM,
				},
			)
			require.NoError(b, err)
		}

		// Mint 1000 FLOW to sender
		mintAmount, err := cadence.NewUFix64("100000000000.0")
		require.NoError(b, err)

		signerAccount = contractsAddress

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(realMintFlowTokenTransaction),
				Arguments: encodeArgs([]cadence.Value{
					cadence.Address(senderAddress),
					mintAmount,
				}),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(b, err)

		// Set signer account to sender for benchmark transactions
		signerAccount = senderAddress

		return runtime, nextTransactionLocation, runtimeInterface
	}

	for _, transaction := range testTransactions {

		b.Run(transaction.Name, func(b *testing.B) {
			// Create fresh runtime and storage for this sub-benchmark
			runtime, nextTransactionLocation, runtimeInterface := setupRuntime()

			for b.Loop() {
				b.StopTimer()
				// set up everything for the transaction
				var err error
				if transaction.Setup != "" {
					err = runtime.ExecuteTransaction(
						Script{
							Source:    []byte(transaction.Setup),
							Arguments: nil,
						},
						Context{
							Interface:   runtimeInterface,
							Location:    nextTransactionLocation(),
							Environment: environment,
							UseVM:       useVM,
						},
					)
					require.NoError(b, err)
				}
				source := []byte(transaction.Body)
				location := nextTransactionLocation()
				b.StartTimer()

				err = runtime.ExecuteTransaction(
					Script{
						Source:    source,
						Arguments: nil,
					},
					Context{
						Interface:   runtimeInterface,
						Location:    location,
						Environment: environment,
						UseVM:       useVM,
					},
				)

				b.StopTimer()
				require.NoError(b, err)
				b.StartTimer()
			}
		})
	}
}

func BenchmarkRuntimeTransactionsInterpreter(b *testing.B) {
	benchmarkRuntimeTransactions(b, false)
}

func BenchmarkRuntimeTransactionsVM(b *testing.B) {
	benchmarkRuntimeTransactions(b, true)
}

func BenchmarkRuntimeTransactions(b *testing.B) {
	benchmarkRuntimeTransactions(b, false)
}
