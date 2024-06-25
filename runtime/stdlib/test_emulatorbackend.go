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
 *
 */

package stdlib

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// 'EmulatorBackend' struct.
//
// 'EmulatorBackend' is the native implementation of the 'Test.BlockchainBackend' interface.
// It provides a blockchain backed by the emulator.

const testEmulatorBackendTypeName = "EmulatorBackend"

type testEmulatorBackendType struct {
	compositeType                      *sema.CompositeType
	executeScriptFunctionType          *sema.FunctionType
	createAccountFunctionType          *sema.FunctionType
	addTransactionFunctionType         *sema.FunctionType
	executeNextTransactionFunctionType *sema.FunctionType
	commitBlockFunctionType            *sema.FunctionType
	deployContractFunctionType         *sema.FunctionType
	logsFunctionType                   *sema.FunctionType
	serviceAccountFunctionType         *sema.FunctionType
	eventsFunctionType                 *sema.FunctionType
	resetFunctionType                  *sema.FunctionType
	moveTimeFunctionType               *sema.FunctionType
	createSnapshotFunctionType         *sema.FunctionType
	loadSnapshotFunctionType           *sema.FunctionType
	getAccountFunctionType             *sema.FunctionType
}

func newTestEmulatorBackendType(
	blockchainBackendInterfaceType *sema.InterfaceType,
) *testEmulatorBackendType {
	executeScriptFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeExecuteScriptFunctionName,
	)

	createAccountFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeCreateAccountFunctionName,
	)

	addTransactionFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeAddTransactionFunctionName,
	)

	executeNextTransactionFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeExecuteNextTransactionFunctionName,
	)

	commitBlockFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeCommitBlockFunctionName,
	)

	deployContractFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeDeployContractFunctionName,
	)

	logsFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeLogsFunctionName,
	)

	serviceAccountFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeServiceAccountFunctionName,
	)

	eventsFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeEventsFunctionName,
	)

	resetFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeResetFunctionName,
	)

	moveTimeFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeMoveTimeFunctionName,
	)

	createSnapshotFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeCreateSnapshotFunctionName,
	)

	loadSnapshotFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeLoadSnapshotFunctionName,
	)

	getAccountFunctionType := interfaceFunctionType(
		blockchainBackendInterfaceType,
		testEmulatorBackendTypeGetAccountFunctionName,
	)

	compositeType := &sema.CompositeType{
		Identifier: testEmulatorBackendTypeName,
		Kind:       common.CompositeKindStructure,
		Location:   TestContractLocation,
		ExplicitInterfaceConformances: []*sema.InterfaceType{
			blockchainBackendInterfaceType,
		},
	}

	var members = []*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeExecuteScriptFunctionName,
			executeScriptFunctionType,
			testEmulatorBackendTypeExecuteScriptFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeCreateAccountFunctionName,
			createAccountFunctionType,
			testEmulatorBackendTypeCreateAccountFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeAddTransactionFunctionName,
			addTransactionFunctionType,
			testEmulatorBackendTypeAddTransactionFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeExecuteNextTransactionFunctionName,
			executeNextTransactionFunctionType,
			testEmulatorBackendTypeExecuteNextTransactionFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeCommitBlockFunctionName,
			commitBlockFunctionType,
			testEmulatorBackendTypeCommitBlockFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeDeployContractFunctionName,
			deployContractFunctionType,
			testEmulatorBackendTypeDeployContractFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeLogsFunctionName,
			logsFunctionType,
			testEmulatorBackendTypeLogsFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeServiceAccountFunctionName,
			serviceAccountFunctionType,
			testEmulatorBackendTypeServiceAccountFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeEventsFunctionName,
			eventsFunctionType,
			testEmulatorBackendTypeEventsFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeResetFunctionName,
			resetFunctionType,
			testEmulatorBackendTypeResetFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeMoveTimeFunctionName,
			moveTimeFunctionType,
			testEmulatorBackendTypeMoveTimeFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeCreateSnapshotFunctionName,
			createSnapshotFunctionType,
			testEmulatorBackendTypeCreateSnapshotFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeLoadSnapshotFunctionName,
			loadSnapshotFunctionType,
			testEmulatorBackendTypeLoadSnapshotFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testEmulatorBackendTypeGetAccountFunctionName,
			getAccountFunctionType,
			testEmulatorBackendTypeGetAccountFunctionDocString,
		),
	}

	compositeType.Members = sema.MembersAsMap(members)
	compositeType.Fields = sema.MembersFieldNames(members)

	return &testEmulatorBackendType{
		compositeType:                      compositeType,
		executeScriptFunctionType:          executeScriptFunctionType,
		createAccountFunctionType:          createAccountFunctionType,
		addTransactionFunctionType:         addTransactionFunctionType,
		executeNextTransactionFunctionType: executeNextTransactionFunctionType,
		commitBlockFunctionType:            commitBlockFunctionType,
		deployContractFunctionType:         deployContractFunctionType,
		logsFunctionType:                   logsFunctionType,
		serviceAccountFunctionType:         serviceAccountFunctionType,
		eventsFunctionType:                 eventsFunctionType,
		resetFunctionType:                  resetFunctionType,
		moveTimeFunctionType:               moveTimeFunctionType,
		createSnapshotFunctionType:         createSnapshotFunctionType,
		loadSnapshotFunctionType:           loadSnapshotFunctionType,
		getAccountFunctionType:             getAccountFunctionType,
	}
}

// 'EmulatorBackend.executeScript' function

const testEmulatorBackendTypeExecuteScriptFunctionName = "executeScript"

const testEmulatorBackendTypeExecuteScriptFunctionDocString = `
Executes a script and returns the script return value and the status.
The 'returnValue' field of the result will be nil if the script failed.
`

func (t *testEmulatorBackendType) newExecuteScriptFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.executeScriptFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			script, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			args, err := arrayValueToSlice(
				inter,
				invocation.Arguments[1],
				invocation.LocationRange,
			)
			if err != nil {
				panic(errors.NewUnexpectedErrorFromCause(err))
			}

			result := blockchain.RunScript(inter, script.Str, args)

			return newScriptResult(inter, result.Value, result)
		},
	)
}

// 'EmulatorBackend.createAccount' function

const testEmulatorBackendTypeCreateAccountFunctionName = "createAccount"

const testEmulatorBackendTypeCreateAccountFunctionDocString = `
Creates an account by submitting an account creation transaction.
The transaction is paid by the service account.
The returned account can be used to sign and authorize transactions.
`

func (t *testEmulatorBackendType) newCreateAccountFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.createAccountFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			account, err := blockchain.CreateAccount()
			if err != nil {
				panic(err)
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			return newTestAccountValue(
				inter,
				locationRange,
				account,
			)
		},
	)
}

func newTestAccountValue(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	account *Account,
) interpreter.Value {

	// Create address value
	address := interpreter.NewAddressValue(nil, account.Address)

	publicKey := NewPublicKeyValue(
		inter,
		locationRange,
		account.PublicKey,
	)

	// Create an 'Account' by calling its constructor.
	accountConstructor := getConstructor(inter, testAccountTypeName)
	accountValue, err := inter.InvokeExternally(
		accountConstructor,
		accountConstructor.Type,
		[]interpreter.Value{
			address,
			publicKey,
		},
	)

	if err != nil {
		panic(err)
	}

	return accountValue
}

// 'EmulatorBackend.getAccount' function

const testEmulatorBackendTypeGetAccountFunctionName = "getAccount"

const testEmulatorBackendTypeGetAccountFunctionDocString = `
Returns the account for the given address.
`

func (t *testEmulatorBackendType) newGetAccountFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.getAccountFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			address, ok := invocation.Arguments[0].(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			account, err := blockchain.GetAccount(address)
			if err != nil {
				msg := fmt.Sprintf("account with address: %s was not found", address)
				panic(PanicError{
					Message:       msg,
					LocationRange: invocation.LocationRange,
				})
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			return newTestAccountValue(
				inter,
				locationRange,
				account,
			)
		},
	)
}

// 'EmulatorBackend.addTransaction' function

const testEmulatorBackendTypeAddTransactionFunctionName = "addTransaction"

const testEmulatorBackendTypeAddTransactionFunctionDocString = `
Add a transaction to the current block.
`

const testTransactionTypeCodeFieldName = "code"
const testTransactionTypeAuthorizersFieldName = "authorizers"
const testTransactionTypeSignersFieldName = "signers"
const testTransactionTypeArgumentsFieldName = "arguments"

func (t *testEmulatorBackendType) newAddTransactionFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.addTransactionFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			transactionValue, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Get transaction code
			codeValue := transactionValue.GetMember(
				inter,
				locationRange,
				testTransactionTypeCodeFieldName,
			)
			code, ok := codeValue.(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Get authorizers
			authorizerValue := transactionValue.GetMember(
				inter,
				locationRange,
				testTransactionTypeAuthorizersFieldName,
			)

			authorizers := addressArrayValueToSlice(inter, authorizerValue, locationRange)

			// Get signers
			signersValue := transactionValue.GetMember(
				inter,
				locationRange,
				testTransactionTypeSignersFieldName,
			)

			signerAccounts := accountsArrayValueToSlice(
				inter,
				signersValue,
				locationRange,
			)

			// Get arguments
			argsValue := transactionValue.GetMember(
				inter,
				locationRange,
				testTransactionTypeArgumentsFieldName,
			)
			args, err := arrayValueToSlice(inter, argsValue, locationRange)
			if err != nil {
				panic(errors.NewUnexpectedErrorFromCause(err))
			}

			err = blockchain.AddTransaction(
				inter,
				code.Str,
				authorizers,
				signerAccounts,
				args,
			)

			if err != nil {
				panic(err)
			}

			return interpreter.Void
		},
	)
}

// 'EmulatorBackend.executeNextTransaction' function

const testEmulatorBackendTypeExecuteNextTransactionFunctionName = "executeNextTransaction"

const testEmulatorBackendTypeExecuteNextTransactionFunctionDocString = `
Executes the next transaction in the block, if any.
Returns the result of the transaction, or nil if no transaction was scheduled.
`

func (t *testEmulatorBackendType) newExecuteNextTransactionFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.executeNextTransactionFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			result := blockchain.ExecuteNextTransaction()

			// If there are no transactions to run, then return `nil`.
			if result == nil {
				return interpreter.Nil
			}

			return newTransactionResult(invocation.Interpreter, result)
		},
	)
}

// 'EmulatorBackend.commitBlock' function

const testEmulatorBackendTypeCommitBlockFunctionName = "commitBlock"

const testEmulatorBackendTypeCommitBlockFunctionDocString = `
Commit the current block. Committing will fail if there are un-executed transactions in the block.
`

func (t *testEmulatorBackendType) newCommitBlockFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.commitBlockFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			err := blockchain.CommitBlock()
			if err != nil {
				panic(err)
			}

			return interpreter.Void
		},
	)
}

// 'EmulatorBackend.deployContract' function

const testEmulatorBackendTypeDeployContractFunctionName = "deployContract"

const testEmulatorBackendTypeDeployContractFunctionDocString = `
Deploys a given contract, and initializes it with the provided arguments.
`

func (t *testEmulatorBackendType) newDeployContractFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.deployContractFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			// Contract name
			name, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Contract file path
			path, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Contract init arguments
			args, err := arrayValueToSlice(
				inter,
				invocation.Arguments[2],
				invocation.LocationRange,
			)
			if err != nil {
				panic(err)
			}

			err = blockchain.DeployContract(
				inter,
				name.Str,
				path.Str,
				args,
			)

			return newErrorValue(inter, err)
		},
	)
}

// 'EmulatorBackend.logs' function

const testEmulatorBackendTypeLogsFunctionName = "logs"

const testEmulatorBackendTypeLogsFunctionDocString = `
Returns all the logs from the blockchain, up to the calling point.
`

func (t *testEmulatorBackendType) newLogsFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.logsFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			logs := blockchain.Logs()
			inter := invocation.Interpreter

			arrayType := interpreter.NewVariableSizedStaticType(
				inter,
				interpreter.NewPrimitiveStaticType(
					inter,
					interpreter.PrimitiveStaticTypeString,
				),
			)

			values := make([]interpreter.Value, len(logs))
			for i, log := range logs {
				memoryUsage := common.NewStringMemoryUsage(len(log))
				values[i] = interpreter.NewStringValue(
					inter,
					memoryUsage,
					func() string {
						return log
					},
				)
			}

			return interpreter.NewArrayValue(
				inter,
				invocation.LocationRange,
				arrayType,
				common.ZeroAddress,
				values...,
			)
		},
	)
}

// 'EmulatorBackend.serviceAccount' function

const testEmulatorBackendTypeServiceAccountFunctionName = "serviceAccount"

const testEmulatorBackendTypeServiceAccountFunctionDocString = `
Returns the service account of the blockchain. Can be used to sign
transactions with this account.
`

func (t *testEmulatorBackendType) newServiceAccountFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.serviceAccountFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			serviceAccount, err := blockchain.ServiceAccount()
			if err != nil {
				panic(err)
			}

			return newTestAccountValue(
				invocation.Interpreter,
				invocation.LocationRange,
				serviceAccount,
			)
		},
	)
}

// 'EmulatorBackend.events' function

const testEmulatorBackendTypeEventsFunctionName = "events"

const testEmulatorBackendTypeEventsFunctionDocString = `
Returns all events emitted from the blockchain,
optionally filtered by event type.
`

func (t *testEmulatorBackendType) newEventsFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.eventsFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			var eventType interpreter.StaticType = nil

			switch value := invocation.Arguments[0].(type) {
			case interpreter.NilValue:
				// Do nothing
			case *interpreter.SomeValue:
				innerValue := value.InnerValue(invocation.Interpreter, invocation.LocationRange)
				typeValue, ok := innerValue.(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				eventType = typeValue.Type
			default:
				panic(errors.NewUnreachableError())
			}

			return blockchain.Events(invocation.Interpreter, eventType)
		},
	)
}

// 'EmulatorBackend.reset' function

const testEmulatorBackendTypeResetFunctionName = "reset"

const testEmulatorBackendTypeResetFunctionDocString = `
Resets the state of the blockchain to the given height.
`

func (t *testEmulatorBackendType) newResetFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.resetFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			height, ok := invocation.Arguments[0].(interpreter.UInt64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			blockchain.Reset(uint64(height))
			return interpreter.Void
		},
	)
}

// 'Emulator.moveTime' function

const testEmulatorBackendTypeMoveTimeFunctionName = "moveTime"

const testEmulatorBackendTypeMoveTimeFunctionDocString = `
Moves the time of the blockchain by the given delta,
which should be passed in the form of seconds.
`

func (t *testEmulatorBackendType) newMoveTimeFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.moveTimeFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			timeDelta, ok := invocation.Arguments[0].(interpreter.Fix64Value)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			blockchain.MoveTime(int64(timeDelta.ToInt(invocation.LocationRange)))
			return interpreter.Void
		},
	)
}

// 'Emulator.createSnapshot' function

const testEmulatorBackendTypeCreateSnapshotFunctionName = "createSnapshot"

const testEmulatorBackendTypeCreateSnapshotFunctionDocString = `
Creates a snapshot of the blockchain, at the
current ledger state, with the given name.
`

func (t *testEmulatorBackendType) newCreateSnapshotFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.createSnapshotFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			name, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			err := blockchain.CreateSnapshot(name.Str)
			return newErrorValue(invocation.Interpreter, err)
		},
	)
}

// 'Emulator.loadSnapshot' function

const testEmulatorBackendTypeLoadSnapshotFunctionName = "loadSnapshot"

const testEmulatorBackendTypeLoadSnapshotFunctionDocString = `
Loads a snapshot of the blockchain, with the given name, and
updates the current ledger state.
`

func (t *testEmulatorBackendType) newLoadSnapshotFunction(
	inter *interpreter.Interpreter,
	emulatorBackend interpreter.MemberAccessibleValue,
	blockchain Blockchain,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		emulatorBackend,
		t.loadSnapshotFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			name, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			err := blockchain.LoadSnapshot(name.Str)
			return newErrorValue(invocation.Interpreter, err)
		},
	)
}

func (t *testEmulatorBackendType) newEmulatorBackend(
	inter *interpreter.Interpreter,
	blockchain Blockchain,
	locationRange interpreter.LocationRange,
) *interpreter.CompositeValue {

	// TODO: Use SimpleCompositeValue
	emulatorBackend := interpreter.NewCompositeValue(
		inter,
		locationRange,
		t.compositeType.Location,
		testEmulatorBackendTypeName,
		common.CompositeKindStructure,
		nil,
		common.ZeroAddress,
	)

	fields := []interpreter.CompositeField{
		{
			Name:  testEmulatorBackendTypeExecuteScriptFunctionName,
			Value: t.newExecuteScriptFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeCreateAccountFunctionName,
			Value: t.newCreateAccountFunction(inter, emulatorBackend, blockchain),
		}, {
			Name:  testEmulatorBackendTypeAddTransactionFunctionName,
			Value: t.newAddTransactionFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeExecuteNextTransactionFunctionName,
			Value: t.newExecuteNextTransactionFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeCommitBlockFunctionName,
			Value: t.newCommitBlockFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeDeployContractFunctionName,
			Value: t.newDeployContractFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeLogsFunctionName,
			Value: t.newLogsFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeServiceAccountFunctionName,
			Value: t.newServiceAccountFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeEventsFunctionName,
			Value: t.newEventsFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeResetFunctionName,
			Value: t.newResetFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeMoveTimeFunctionName,
			Value: t.newMoveTimeFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeCreateSnapshotFunctionName,
			Value: t.newCreateSnapshotFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeLoadSnapshotFunctionName,
			Value: t.newLoadSnapshotFunction(inter, emulatorBackend, blockchain),
		},
		{
			Name:  testEmulatorBackendTypeGetAccountFunctionName,
			Value: t.newGetAccountFunction(inter, emulatorBackend, blockchain),
		},
	}

	for _, field := range fields {
		emulatorBackend.SetMember(
			inter,
			locationRange,
			field.Name,
			field.Value,
		)
	}

	return emulatorBackend
}
