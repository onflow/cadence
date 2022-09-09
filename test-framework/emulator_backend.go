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
	"strings"

	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	sdkTest "github.com/onflow/flow-go-sdk/test"

	fvmCrypto "github.com/onflow/flow-go/fvm/crypto"

	emulator "github.com/onflow/flow-emulator"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var _ stdlib.TestFramework = &EmulatorBackend{}

// EmulatorBackend is the emulator-backed implementation of the interpreter.TestFramework.
//
type EmulatorBackend struct {
	blockchain *emulator.Blockchain

	// blockOffset is the offset for the sequence number of the next transaction.
	// This is equal to the number of transactions in the current block.
	// Must be reset once the block is committed.
	blockOffset uint64

	// accountKeys is a mapping of account addresses with their keys.
	accountKeys map[common.Address]map[string]keyInfo

	// fileResolver is used to resolve local files.
	//
	fileResolver FileResolver

	// A property bag to pass various configurations to the backend.
	// Currently, supports passing address mapping for contracts.
	configuration *stdlib.Configuration
}

type keyInfo struct {
	accountKey *sdk.AccountKey
	signer     crypto.Signer
}

func NewEmulatorBackend(fileResolver FileResolver) *EmulatorBackend {
	return &EmulatorBackend{
		blockchain:   newBlockchain(),
		blockOffset:  0,
		accountKeys:  map[common.Address]map[string]keyInfo{},
		fileResolver: fileResolver,
	}
}

func (e *EmulatorBackend) RunScript(code string, args []interpreter.Value) *stdlib.ScriptResult {
	inter, err := newInterpreter()
	if err != nil {
		return &stdlib.ScriptResult{
			Error: err,
		}
	}

	arguments := make([][]byte, 0, len(args))
	for _, arg := range args {
		exportedValue, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return &stdlib.ScriptResult{
				Error: err,
			}
		}

		encodedArg, err := json.Encode(exportedValue)
		if err != nil {
			return &stdlib.ScriptResult{
				Error: err,
			}
		}

		arguments = append(arguments, encodedArg)
	}

	code = e.replaceImports(code)

	result, err := e.blockchain.ExecuteScript([]byte(code), arguments)
	if err != nil {
		return &stdlib.ScriptResult{
			Error: err,
		}
	}

	if result.Error != nil {
		return &stdlib.ScriptResult{
			Error: result.Error,
		}
	}

	value, err := runtime.ImportValue(inter, interpreter.ReturnEmptyLocationRange, result.Value, nil)
	if err != nil {
		return &stdlib.ScriptResult{
			Error: err,
		}
	}

	return &stdlib.ScriptResult{
		Value: value,
	}
}

func (e EmulatorBackend) CreateAccount() (*stdlib.Account, error) {
	// Also generate the keys. So that users don't have to do this in two steps.
	// Store the generated keys, so that it could be looked-up, given the address.

	keyGen := sdkTest.AccountKeyGenerator()
	accountKey, signer := keyGen.NewWithSigner()

	address, err := e.blockchain.CreateAccount([]*sdk.AccountKey{accountKey}, nil)
	if err != nil {
		return nil, err
	}

	publicKey := accountKey.PublicKey.Encode()
	encodedPublicKey := string(publicKey)

	// Store the generated key and signer info.
	// This info is used to sign transactions.
	e.accountKeys[common.Address(address)] = map[string]keyInfo{
		encodedPublicKey: {
			accountKey: accountKey,
			signer:     signer,
		},
	}

	return &stdlib.Account{
		Address: common.Address(address),
		PublicKey: &stdlib.PublicKey{
			PublicKey: publicKey,
			SignAlgo:  fvmCrypto.CryptoToRuntimeSigningAlgorithm(accountKey.PublicKey.Algorithm()),
		},
	}, nil
}

func (e *EmulatorBackend) AddTransaction(
	code string,
	authorizers []common.Address,
	signers []*stdlib.Account,
	args []interpreter.Value,
) error {

	code = e.replaceImports(code)

	tx := e.newTransaction(code, authorizers)

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

func (e *EmulatorBackend) newTransaction(code string, authorizers []common.Address) *sdk.Transaction {
	serviceKey := e.blockchain.ServiceKey()

	sequenceNumber := serviceKey.SequenceNumber + e.blockOffset

	tx := sdk.NewTransaction().
		SetScript([]byte(code)).
		SetProposalKey(serviceKey.Address, serviceKey.Index, sequenceNumber).
		SetPayer(serviceKey.Address)

	for _, authorizer := range authorizers {
		tx = tx.AddAuthorizer(sdk.Address(authorizer))
	}

	return tx
}

func (e *EmulatorBackend) signTransaction(
	tx *sdk.Transaction,
	signerAccounts []*stdlib.Account,
) error {

	// Sign transaction with each signer
	// Note: Following logic is borrowed from the flow-ft.

	for i := len(signerAccounts) - 1; i >= 0; i-- {
		signerAccount := signerAccounts[i]

		publicKey := string(signerAccount.PublicKey.PublicKey)
		accountKeys := e.accountKeys[signerAccount.Address]
		keyInfo := accountKeys[publicKey]

		err := tx.SignPayload(sdk.Address(signerAccount.Address), 0, keyInfo.signer)
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

func (e *EmulatorBackend) ExecuteNextTransaction() *stdlib.TransactionResult {
	result, err := e.blockchain.ExecuteNextTransaction()

	if err != nil {
		// If the returned error is `emulator.PendingBlockTransactionsExhaustedError`,
		// that means there are no transactions to execute.
		// Hence, return a nil result.
		if _, ok := err.(*emulator.PendingBlockTransactionsExhaustedError); ok {
			return nil
		}

		return &stdlib.TransactionResult{
			Error: err,
		}
	}

	if result.Error != nil {
		return &stdlib.TransactionResult{
			Error: result.Error,
		}
	}

	return &stdlib.TransactionResult{}
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
	account *stdlib.Account,
	args []interpreter.Value,
) error {

	const deployContractTransactionTemplate = `
	    transaction(%s) {
		    prepare(signer: AuthAccount) {
			    signer.contracts.add(name: "%s", code: "%s".decodeHex()%s)
		    }
	    }`

	code = e.replaceImports(code)

	hexEncodedCode := hex.EncodeToString([]byte(code))

	inter, err := newInterpreter()
	if err != nil {
		return err
	}

	cadenceArgs := make([]cadence.Value, 0, len(args))

	var txArgsBuilder, addArgsBuilder strings.Builder

	for i, arg := range args {
		cadenceArg, err := runtime.ExportValue(arg, inter, interpreter.ReturnEmptyLocationRange)
		if err != nil {
			return err
		}

		if i > 0 {
			txArgsBuilder.WriteString(", ")
		}

		txArgsBuilder.WriteString(fmt.Sprintf("arg%d: %s", i, cadenceArg.Type().ID()))
		addArgsBuilder.WriteString(fmt.Sprintf(", arg%d", i))

		cadenceArgs = append(cadenceArgs, cadenceArg)
	}

	script := fmt.Sprintf(
		deployContractTransactionTemplate,
		txArgsBuilder.String(),
		name,
		hexEncodedCode,
		addArgsBuilder.String(),
	)

	tx := e.newTransaction(script, []common.Address{account.Address})

	for _, arg := range cadenceArgs {
		err = tx.AddArgument(arg)
		if err != nil {
			return err
		}
	}

	err = e.signTransaction(tx, []*stdlib.Account{account})
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

func (e *EmulatorBackend) ReadFile(path string) (string, error) {
	if e.fileResolver == nil {
		return "", FileResolverNotProvidedError{}
	}

	return e.fileResolver(path)
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

func (e *EmulatorBackend) UseConfiguration(configuration *stdlib.Configuration) {
	e.configuration = configuration
}

// newInterpreter creates an interpreter instance needed for the value conversion.
//
func newInterpreter() (*interpreter.Interpreter, error) {
	// TODO: maybe re-use interpreter? Only needed for value conversion
	// TODO: Deal with imported/composite types

	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)

	return interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			BaseActivation: baseActivation,
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
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
					panic(errors.NewUnexpectedError("importing of programs not implemented"))
				}
			},
		},
	)
}

func (e *EmulatorBackend) replaceImports(code string) string {
	if e.configuration == nil {
		return code
	}

	program, err := parser.ParseProgram(code, nil)
	if err != nil {
		panic(err)
	}

	sb := strings.Builder{}
	importDeclEnd := 0

	for _, importDeclaration := range program.ImportDeclarations() {
		prevImportDeclEnd := importDeclEnd
		importDeclEnd = importDeclaration.EndPos.Offset + 1

		location, ok := importDeclaration.Location.(common.StringLocation)
		if !ok {
			// keep the import statement it as-is
			sb.WriteString(code[prevImportDeclEnd:importDeclEnd])
			continue
		}

		address, ok := e.configuration.Addresses[location.String()]
		if !ok {
			// keep import statement it as-is
			sb.WriteString(code[prevImportDeclEnd:importDeclEnd])
			continue
		}

		addressStr := fmt.Sprintf("0x%s", address)

		locationStart := importDeclaration.LocationPos.Offset

		sb.WriteString(code[prevImportDeclEnd:locationStart])
		sb.WriteString(addressStr)

	}

	sb.WriteString(code[importDeclEnd:])

	return sb.String()
}
