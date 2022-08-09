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

package test

import (
	"encoding/hex"
	"fmt"

	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	sdkTest "github.com/onflow/flow-go-sdk/test"

	emulator "github.com/onflow/flow-emulator"
	fvmCrypto "github.com/onflow/flow-go/fvm/crypto"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var _ interpreter.TestFramework = &EmulatorBackend{}

// EmulatorBackend is the emulator-backed implementation of the interpreter.TestFramework.
//
type EmulatorBackend struct {
	blockchain *emulator.Blockchain

	// blockOffset is the offset for the sequence number of the next transaction.
	// This is equal to the number of transactions in the current block.
	// Must be rest once the block is committed.
	blockOffset uint64
}

func NewEmulatorBackend() *EmulatorBackend {
	return &EmulatorBackend{
		blockchain:  newBlockchain(),
		blockOffset: 0,
	}
}

func (e *EmulatorBackend) RunScript(code string, args []interpreter.Value) *interpreter.ScriptResult {
	inter, err := newInterpreter()
	if err != nil {
		return &interpreter.ScriptResult{
			Error: err,
		}
	}

	arguments := make([][]byte, 0, len(args))
	for _, arg := range args {
		exportedValue, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return &interpreter.ScriptResult{
				Error: err,
			}
		}

		encodedArg, err := json.Encode(exportedValue)
		if err != nil {
			return &interpreter.ScriptResult{
				Error: err,
			}
		}

		arguments = append(arguments, encodedArg)
	}

	result, err := e.blockchain.ExecuteScript([]byte(code), arguments)
	if err != nil {
		return &interpreter.ScriptResult{
			Error: err,
		}
	}

	if result.Error != nil {
		return &interpreter.ScriptResult{
			Error: result.Error,
		}
	}

	value, err := runtime.ImportValue(inter, interpreter.ReturnEmptyLocationRange, result.Value, nil)
	if err != nil {
		return &interpreter.ScriptResult{
			Error: err,
		}
	}

	return &interpreter.ScriptResult{
		Value: value,
	}
}

func (e EmulatorBackend) CreateAccount() (*interpreter.Account, error) {
	keyGen := sdkTest.AccountKeyGenerator()
	accountKey, signer := keyGen.NewWithSigner()

	// This relies on flow-go-sdk/test returning an `InMemorySigner`.
	// TODO: Maybe copy over the code for `AccountKeyGenerator`.
	inMemSigner := signer.(crypto.InMemorySigner)

	address, err := e.blockchain.CreateAccount([]*sdk.AccountKey{accountKey}, nil)
	if err != nil {
		return nil, err
	}

	return &interpreter.Account{
		Address: common.Address(address),
		AccountKey: &interpreter.AccountKey{
			KeyIndex: accountKey.Index,
			PublicKey: &interpreter.PublicKey{
				PublicKey: accountKey.PublicKey.Encode(),
				SignAlgo:  fvmCrypto.CryptoToRuntimeSigningAlgorithm(accountKey.PublicKey.Algorithm()),
			},
			HashAlgo:  fvmCrypto.CryptoToRuntimeHashingAlgorithm(accountKey.HashAlgo),
			Weight:    accountKey.Weight,
			IsRevoked: accountKey.Revoked,
		},
		PrivateKey: inMemSigner.PrivateKey.Encode(),
	}, nil
}

func (e *EmulatorBackend) AddTransaction(
	code string,
	authorizer *common.Address,
	signers []*interpreter.Account,
	args []interpreter.Value,
) error {

	tx := e.newTransaction(code, authorizer)

	inter, err := newInterpreter()
	if err != nil {
		return err
	}

	for _, arg := range args {
		exportedValue, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return err
		}

		err = tx.AddArgument(exportedValue)
		if err != nil {
			return err
		}
	}

	err = e.signTransaction(tx, signers)
	if err != nil {
		return err
	}

	err = e.blockchain.AddTransaction(*tx)
	if err != nil {
		return err
	}

	// Increment the transaction sequence number offset for the current block.
	e.blockOffset++

	return nil
}

func (e *EmulatorBackend) newTransaction(code string, authorizer *common.Address) *sdk.Transaction {
	serviceKey := e.blockchain.ServiceKey()

	sequenceNumber := serviceKey.SequenceNumber + e.blockOffset

	tx := sdk.NewTransaction().
		SetScript([]byte(code)).
		SetProposalKey(serviceKey.Address, serviceKey.Index, sequenceNumber).
		SetPayer(serviceKey.Address)

	if authorizer != nil {
		tx = tx.AddAuthorizer(sdk.Address(*authorizer))
	}

	return tx
}

func (e *EmulatorBackend) signTransaction(
	tx *sdk.Transaction,
	signerAccounts []*interpreter.Account,
) error {

	// Sign transaction with each signer
	// Note: Following logic is borrowed from the flow-ft.

	for i := len(signerAccounts) - 1; i >= 0; i-- {
		signerAccount := signerAccounts[i]

		signAlgo := fvmCrypto.RuntimeToCryptoSigningAlgorithm(signerAccount.AccountKey.PublicKey.SignAlgo)
		privateKey, err := crypto.DecodePrivateKey(signAlgo, signerAccount.PrivateKey)
		if err != nil {
			return err
		}

		hashAlgo := fvmCrypto.RuntimeToCryptoHashingAlgorithm(signerAccount.AccountKey.HashAlgo)
		signer, err := crypto.NewInMemorySigner(privateKey, hashAlgo)
		if err != nil {
			return err
		}

		err = tx.SignPayload(sdk.Address(signerAccount.Address), 0, signer)
		if err != nil {
			return err
		}
	}

	serviceKey := e.blockchain.ServiceKey()
	serviceSigner, err := serviceKey.Signer()
	if err != nil {
		return err
	}

	err = tx.SignEnvelope(serviceKey.Address, 0, serviceSigner)
	if err != nil {
		return err
	}

	return nil
}

func (e *EmulatorBackend) ExecuteNextTransaction() *interpreter.TransactionResult {
	result, err := e.blockchain.ExecuteNextTransaction()

	if err != nil {
		// If the returned error is `emulator.PendingBlockTransactionsExhaustedError`,
		// that means there are no transactions to execute.
		// Hence, return a nil result.
		if _, ok := err.(*emulator.PendingBlockTransactionsExhaustedError); ok {
			return nil
		}

		return &interpreter.TransactionResult{
			Error: err,
		}
	}

	if result.Error != nil {
		return &interpreter.TransactionResult{
			Error: result.Error,
		}
	}

	return &interpreter.TransactionResult{}
}

func (e *EmulatorBackend) CommitBlock() error {
	// Reset the transaction offset for the current block.
	e.blockOffset = 0

	_, err := e.blockchain.CommitBlock()
	return err
}

func (e *EmulatorBackend) DeployContract(
	name string,
	code string,
	args []interpreter.Value,
	authorizer common.Address,
	signers []*interpreter.Account,
) error {

	const deployContractTransactionTemplate = `
	    transaction(%s) {
		    prepare(signer: AuthAccount) {
			    signer.contracts.add(name: "%s", code: "%s".decodeHex()%s)
		    }
	    }`

	hexEncodedCode := hex.EncodeToString([]byte(code))

	inter, err := newInterpreter()
	if err != nil {
		return err
	}

	cadenceArgs := make([]cadence.Value, 0, len(args))

	txArgs, addArgs := "", ""

	for i, arg := range args {
		cadenceArg, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return err
		}

		if i > 0 {
			txArgs += ", "
		}

		txArgs += fmt.Sprintf("arg%d: %s", i, cadenceArg.Type().ID())
		addArgs += fmt.Sprintf(", arg%d", i)

		cadenceArgs = append(cadenceArgs, cadenceArg)
	}

	script := fmt.Sprintf(
		deployContractTransactionTemplate,
		txArgs,
		name,
		hexEncodedCode,
		addArgs,
	)

	tx := e.newTransaction(script, &authorizer)

	for _, arg := range cadenceArgs {
		err = tx.AddArgument(arg)
		if err != nil {
			return err
		}
	}

	err = e.signTransaction(tx, signers)
	if err != nil {
		return err
	}

	err = e.blockchain.AddTransaction(*tx)
	if err != nil {
		return err
	}

	// Increment the transaction sequence number offset for the current block.
	e.blockOffset++

	result := e.ExecuteNextTransaction()
	if result.Error != nil {
		return result.Error
	}

	return e.CommitBlock()
}

// newBlockchain returns an emulator blockchain for testing.
func newBlockchain(opts ...emulator.Option) *emulator.Blockchain {
	b, err := emulator.NewBlockchain(
		append(
			[]emulator.Option{
				emulator.WithStorageLimitEnabled(false),
			},
			opts...,
		)...,
	)
	if err != nil {
		panic(err)
	}

	return b
}

// newInterpreter creates an interpreter instance needed for the value conversion.
//
func newInterpreter() (*interpreter.Interpreter, error) {
	// TODO: maybe re-use interpreter? Only needed for value conversion
	// TODO: Deal with imported/composite types

	predeclaredInterpreterValues := stdlib.BuiltinFunctions.ToInterpreterValueDeclarations()
	predeclaredInterpreterValues = append(predeclaredInterpreterValues, stdlib.BuiltinValues.ToInterpreterValueDeclarations()...)

	return interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		interpreter.WithStorage(interpreter.NewInMemoryStorage(nil)),
		interpreter.WithPredeclaredValues(predeclaredInterpreterValues),
		interpreter.WithImportLocationHandler(func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			switch location {
			case stdlib.CryptoChecker.Location:
				program := interpreter.ProgramFromChecker(stdlib.CryptoChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			case stdlib.TestContractLocation:
				program := interpreter.ProgramFromChecker(stdlib.TestContractChecker)
				subInterpreter, err := inter.NewSubInterpreter(program, location)
				if err != nil {
					panic(err)
				}
				return interpreter.InterpreterImport{
					Interpreter: subInterpreter,
				}

			default:
				panic(errors.NewUnexpectedError("importing programs not implemented"))
			}
		}),
	)
}
