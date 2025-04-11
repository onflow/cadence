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

package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/onflow/cadence/bbq/commons"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type ParseCheckAndInterpretOptions struct {
	Config             *interpreter.Config
	CheckerConfig      *sema.Config
	HandleCheckerError func(error)
}

func parseCheckAndInterpretWithOptions(
	t testing.TB,
	code string,
	location common.Location,
	options ParseCheckAndInterpretOptions,
) (
	inter *interpreter.Interpreter,
	err error,
) {
	return parseCheckAndInterpretWithOptionsAndMemoryMetering(t, code, location, options, nil)
}

func parseCheckAndInterpretWithOptionsAndMemoryMetering(
	t testing.TB,
	code string,
	location common.Location,
	options ParseCheckAndInterpretOptions,
	memoryGauge common.MemoryGauge,
) (
	inter *interpreter.Interpreter,
	err error,
) {

	checker, err := ParseAndCheckWithOptionsAndMemoryMetering(t,
		code,
		ParseAndCheckOptions{
			Location: location,
			Config:   options.CheckerConfig,
		},
		memoryGauge,
	)

	if options.HandleCheckerError != nil {
		options.HandleCheckerError(err)
	} else if !assert.NoError(t, err) {
		var sb strings.Builder
		location := checker.Location
		printErr := pretty.NewErrorPrettyPrinter(&sb, true).
			PrettyPrintError(err, location, map[common.Location][]byte{location: []byte(code)})
		if printErr != nil {
			panic(printErr)
		}
		assert.Fail(t, sb.String())
		return nil, err
	}

	var uuid uint64 = 0

	var config interpreter.Config
	if options.Config != nil {
		config = *options.Config
	}

	if config.UUIDHandler == nil {
		config.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}
	}
	if config.Storage == nil {
		config.Storage = interpreter.NewInMemoryStorage(memoryGauge)
	}

	if memoryGauge != nil && config.MemoryGauge == nil {
		config.MemoryGauge = memoryGauge
	}

	inter, err = interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&config,
	)
	require.NoError(t, err)

	err = inter.Interpret()

	if err == nil {

		// recover internal panics and return them as an error
		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		// Contract declarations are evaluated lazily,
		// so force the contract value handler to be called

		for _, compositeDeclaration := range checker.Program.CompositeDeclarations() {
			if compositeDeclaration.CompositeKind != common.CompositeKindContract {
				continue
			}

			contractVariable := inter.Globals.Get(compositeDeclaration.Identifier.Identifier)

			_ = contractVariable.GetValue(inter)
		}
	}

	return inter, err
}

func newStringLocationHandler(tb testing.TB, address common.Address) sema.LocationHandlerFunc {
	return func(identifiers []ast.Identifier, location common.Location) ([]commons.ResolvedLocation, error) {
		require.Empty(tb, identifiers)
		require.IsType(tb, common.StringLocation(""), location)
		name := string(location.(common.StringLocation))

		return []commons.ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: address,
					Name:    name,
				},
			},
		}, nil
	}
}

func interpreterFTTransfer(tb testing.TB) {

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	burnerLocation := common.NewAddressLocation(nil, contractsAddress, "Burner")
	viewResolverLocation := common.NewAddressLocation(nil, contractsAddress, "ViewResolver")
	fungibleTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")
	metadataViewsLocation := common.NewAddressLocation(nil, contractsAddress, "MetadataViews")
	fungibleTokenMetadataViewsLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleTokenMetadataViews")
	nonFungibleTokenLocation := common.NewAddressLocation(nil, contractsAddress, "NonFungibleToken")
	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")

	codes := map[common.Location][]byte{
		burnerLocation:                     []byte(realBurnerContract),
		viewResolverLocation:               []byte(realViewResolverContract),
		fungibleTokenLocation:              []byte(realFungibleTokenContract),
		metadataViewsLocation:              []byte(realMetadataViewsContract),
		fungibleTokenMetadataViewsLocation: []byte(realFungibleTokenMetadataViewsContract),
		nonFungibleTokenLocation:           []byte(realNonFungibleTokenContract),
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	var signer interpreter.Value
	var flowTokenContractValue *interpreter.CompositeValue

	accountHandler := &testAccountHandler{
		getAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
			code, ok := codes[location]
			if !ok {
				return nil, nil
				//	return nil, fmt.Errorf("cannot find code for %s", location)
			}

			return code, nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			codes[location] = code
			return nil
		},
		contractUpdateRecorded: func(location common.AddressLocation) bool {
			return false
		},
		interpretContract: func(
			location common.AddressLocation,
			program *interpreter.Program,
			name string,
			invocation stdlib.DeployedContractConstructorInvocation,
		) (*interpreter.CompositeValue, error) {
			if location == flowTokenLocation {
				return flowTokenContractValue, nil
			}
			return nil, fmt.Errorf("cannot interpret contract %s", location)
		},
		temporarilyRecordCode: func(location common.AddressLocation, code []byte) {
			// do nothing
		},
		emitEvent: func(interpreter.ValueExportContext, interpreter.LocationRange, *sema.CompositeType, []interpreter.Value) {
			// do nothing
		},
		recordContractUpdate: func(location common.AddressLocation, value *interpreter.CompositeValue) {
			// do nothing
		},
	}

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)
	interpreter.Declare(baseActivation, stdlib.AssertFunction)
	interpreter.Declare(baseActivation, stdlib.NewGetAccountFunction(accountHandler))

	subInterpreters := map[common.Location]*interpreter.Interpreter{}

	checkerConfig := &sema.Config{
		ImportHandler: func(checker *sema.Checker, location common.Location, importRange ast.Range) (sema.Import, error) {
			imported, ok := subInterpreters[location]
			if !ok {
				return nil, fmt.Errorf("cannot find contract in location %s", location)
			}

			return sema.ElaborationImport{
				Elaboration: imported.Program.Elaboration,
			}, nil
		},
		BaseValueActivationHandler: baseValueActivation,
		LocationHandler:            newStringLocationHandler(tb, contractsAddress),
	}

	interConfig := &interpreter.Config{
		Storage: storage,
		BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
			return baseActivation
		},
		ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
			imported, ok := subInterpreters[location]
			if !ok {
				panic(fmt.Errorf("cannot find contract in location %s", location))
			}

			return interpreter.InterpreterImport{
				Interpreter: imported,
			}
		},
		ContractValueHandler: func(
			inter *interpreter.Interpreter,
			compositeType *sema.CompositeType,
			constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
			invocationRange ast.Range,
		) interpreter.ContractValue {

			constructor := constructorGenerator(common.ZeroAddress)

			value, err := interpreter.InvokeFunctionValue(
				inter,
				constructor,
				nil,
				nil,
				nil,
				compositeType,
				ast.EmptyRange,
			)
			if err != nil {
				panic(err)
			}

			flowTokenContractValue = value.(*interpreter.CompositeValue)
			return flowTokenContractValue
		},
		CapabilityBorrowHandler: func(
			context interpreter.BorrowCapabilityControllerContext,
			locationRange interpreter.LocationRange,
			address interpreter.AddressValue,
			capabilityID interpreter.UInt64Value,
			wantedBorrowType *sema.ReferenceType,
			capabilityBorrowType *sema.ReferenceType,
		) interpreter.ReferenceValue {
			return stdlib.BorrowCapabilityController(
				context,
				locationRange,
				address,
				capabilityID,
				wantedBorrowType,
				capabilityBorrowType,
				accountHandler,
			)
		},
		OnEventEmitted: func(
			_ *interpreter.Interpreter,
			_ interpreter.LocationRange,
			_ *interpreter.CompositeValue,
			_ *sema.CompositeType,
		) error {
			// NO-OP
			return nil
		},
		InjectedCompositeFieldsHandler: func(
			context interpreter.AccountCreationContext,
			_ common.Location,
			_ string,
			_ common.CompositeKind,
		) map[string]interpreter.Value {

			accountRef := stdlib.NewAccountReferenceValue(
				context,
				accountHandler,
				interpreter.NewAddressValue(nil, contractsAddress),
				interpreter.FullyEntitledAccountAccess,
				interpreter.EmptyLocationRange,
			)

			return map[string]interpreter.Value{
				sema.ContractAccountFieldName: accountRef,
			}
		},
		AccountHandler: func(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value {
			return stdlib.NewAccountValue(context, nil, address)
		},
	}

	parseCheckAndInterpret := func(code string, location common.Location) (*interpreter.Interpreter, error) {
		if subInterpreter, ok := subInterpreters[location]; ok {
			return subInterpreter, nil
		}

		inter, err := parseCheckAndInterpretWithOptions(
			tb,
			code,
			location,
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)

		if err != nil {
			return nil, err
		}

		subInterpreters[location] = inter

		return inter, nil
	}

	accountHandler.parseAndCheckProgram = func(code []byte, location common.Location, getAndSetProgram bool) (*interpreter.Program, error) {
		inter, err := parseCheckAndInterpret(string(code), location)
		if err != nil {
			return nil, err
		}
		return inter.Program, err
	}

	// Parse and check contract interfaces

	contractInterfaceLocations := []common.Location{
		burnerLocation,
		viewResolverLocation,
		fungibleTokenLocation,
		nonFungibleTokenLocation,
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
	}

	for _, location := range contractInterfaceLocations {
		_, err := parseCheckAndInterpret(string(codes[location]), location)
		require.NoError(tb, err)
	}

	// Deploy FlowToken contract

	flowTokenDeploymentTransaction := DeploymentTransaction(
		"FlowToken",
		[]byte(realFlowContract),
	)

	inter, err := parseCheckAndInterpret(
		string(flowTokenDeploymentTransaction),
		nextTransactionLocation(),
	)
	require.NoError(tb, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(contractsAddress),
		interpreter.FullyEntitledAccountAccess,
		interpreter.EmptyLocationRange,
	)

	err = inter.InvokeTransaction(0, signer)
	require.NoError(tb, err)

	// Run setup account transaction

	authorization := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{
			sema.CapabilitiesType,
			sema.StorageType,
			sema.BorrowValueType,
		},
		sema.Conjunction,
	)

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		inter, err := parseCheckAndInterpret(
			realFlowTokenSetupAccountTransaction,
			nextTransactionLocation(),
		)
		require.NoError(tb, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(address),
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(0, signer)
		require.NoError(tb, err)
	}

	// Mint FLOW to sender

	total := uint64(1000000)

	inter, err = parseCheckAndInterpret(
		realFlowTokenMintTokensTransaction,
		nextTransactionLocation(),
	)
	require.NoError(tb, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(contractsAddress),
		interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
		interpreter.EmptyLocationRange,
	)

	err = inter.InvokeTransaction(
		0,
		interpreter.AddressValue(senderAddress),
		interpreter.NewUnmeteredUFix64ValueWithInteger(total, interpreter.EmptyLocationRange),
		signer,
	)
	require.NoError(tb, err)

	// Run token transfer transaction

	transferAmount := interpreter.NewUnmeteredUFix64ValueWithInteger(uint64(1), interpreter.EmptyLocationRange)

	inter, err = parseCheckAndInterpret(
		realFlowTokenTransferTokensTransaction,
		nextTransactionLocation(),
	)
	require.NoError(tb, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(senderAddress),
		interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
		interpreter.EmptyLocationRange,
	)

	var transferCount int

	loop := func() bool {
		return transferCount == 0
	}

	b, _ := tb.(*testing.B)

	if b != nil {

		b.ReportAllocs()
		b.ResetTimer()

		loop = func() bool {
			return transferCount < b.N
		}
	}

	for loop() {

		err = inter.InvokeTransaction(
			0,
			transferAmount,
			interpreter.AddressValue(receiverAddress),
			signer,
		)
		require.NoError(tb, err)

		transferCount++
	}

	if b != nil {
		b.StopTimer()
	}

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		inter, err = parseCheckAndInterpret(
			realFlowTokenGetBalanceScript,
			nextScriptLocation(),
		)
		require.NoError(tb, err)

		result, err := inter.Invoke(
			"main",
			interpreter.AddressValue(address),
		)
		require.NoError(tb, err)

		if address == senderAddress {
			assert.Equal(
				tb,
				interpreter.NewUnmeteredUFix64ValueWithInteger(
					total-uint64(transferCount),
					interpreter.EmptyLocationRange,
				),
				result,
			)
		} else {
			assert.Equal(
				tb,
				interpreter.NewUnmeteredUFix64ValueWithInteger(
					uint64(transferCount),
					interpreter.EmptyLocationRange,
				),
				result,
			)
		}
	}
}

func TestInterpreterFTTransfer(t *testing.T) {
	t.Parallel()

	interpreterFTTransfer(t)
}

func BenchmarkInterpreterFTTransfer(b *testing.B) {

	interpreterFTTransfer(b)
}

func BenchmarkRuntimeFungibleTokenTransfer(b *testing.B) {

	interpreterRuntime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]common.Address, error) {
			return []common.Address{signerAccount}, nil
		},
		OnResolveLocation: newStringLocationHandler(b, contractsAddress),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	environment := runtime.NewBaseInterpreterEnvironment(runtime.Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract interfaces

	contractInterfaces := []struct {
		name string
		code []byte
	}{
		{"Burner", []byte(realBurnerContract)},
		{"ViewResolver", []byte(realViewResolverContract)},
		{"FungibleToken", []byte(realFungibleTokenContract)},
		{"NonFungibleToken", []byte(realNonFungibleTokenContract)},
		{"MetadataViews", []byte(realMetadataViewsContract)},
		{"FungibleTokenMetadataViews", []byte(realFungibleTokenMetadataViewsContract)},
	}

	for _, contract := range contractInterfaces {

		err := interpreterRuntime.ExecuteTransaction(
			runtime.Script{
				Source: DeploymentTransaction(
					contract.name,
					contract.code,
				),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)
	}

	// Deploy Flow Token contract

	err := interpreterRuntime.ExecuteTransaction(
		runtime.Script{
			Source: DeploymentTransaction("FlowToken", []byte(realFlowContract)),
		},
		runtime.Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	// Setup both user accounts for Flow Token

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		signerAccount = address

		err = interpreterRuntime.ExecuteTransaction(
			runtime.Script{
				Source: []byte(realFlowTokenSetupAccountTransaction),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)
	}

	// Mint 10000000 FLOW to sender

	mintAmount, err := cadence.NewUFix64("10000000.0")
	require.NoError(b, err)

	mintAmountValue := interpreter.NewUnmeteredUFix64Value(uint64(mintAmount))

	signerAccount = contractsAddress

	err = interpreterRuntime.ExecuteTransaction(
		runtime.Script{
			Source: []byte(realFlowTokenMintTokensTransaction),
			Arguments: encodeArgs([]cadence.Value{
				cadence.Address(senderAddress),
				mintAmount,
			}),
		},
		runtime.Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	// Benchmark sending tokens from sender to receiver

	sendAmount, err := cadence.NewUFix64("1.0")
	require.NoError(b, err)

	signerAccount = senderAddress

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		err = interpreterRuntime.ExecuteTransaction(
			runtime.Script{
				Source: []byte(realFlowTokenTransferTokensTransaction),
				Arguments: encodeArgs([]cadence.Value{
					sendAmount,
					cadence.Address(receiverAddress),
				}),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)
	}

	b.StopTimer()

	// Run validation scripts

	sum := interpreter.NewUnmeteredUFix64Value(0)

	inter := NewTestInterpreter(b)

	nextScriptLocation := NewScriptLocationGenerator()

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		result, err := interpreterRuntime.ExecuteScript(
			runtime.Script{
				Source: []byte(realFlowTokenGetBalanceScript),
				Arguments: encodeArgs([]cadence.Value{
					cadence.Address(address),
				}),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Location:    nextScriptLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)

		value := interpreter.NewUnmeteredUFix64Value(uint64(result.(cadence.UFix64)))

		require.True(b, bool(value.Less(inter, mintAmountValue, interpreter.EmptyLocationRange)))

		sum = sum.Plus(inter, value, interpreter.EmptyLocationRange).(interpreter.UFix64Value)
	}

	RequireValuesEqual(b, nil, mintAmountValue, sum)
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

func TestInterpreterImperativeFib(t *testing.T) {

	t.Parallel()

	scriptLocation := NewScriptLocationGenerator()

	inter, err := parseCheckAndInterpretWithOptions(
		t,
		imperativeFib,
		scriptLocation(),
		ParseCheckAndInterpretOptions{},
	)
	require.NoError(t, err)

	var value interpreter.Value = interpreter.NewUnmeteredIntValueFromInt64(7)

	result, err := inter.Invoke("fib", value)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(13), result)
}

func BenchmarkInterpreterImperativeFib(b *testing.B) {

	scriptLocation := NewScriptLocationGenerator()

	inter, err := parseCheckAndInterpretWithOptions(
		b,
		imperativeFib,
		scriptLocation(),
		ParseCheckAndInterpretOptions{},
	)
	require.NoError(b, err)

	var value interpreter.Value = interpreter.NewUnmeteredIntValueFromInt64(14)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := inter.Invoke("fib", value)
		require.NoError(b, err)
	}
}

func BenchmarkInterpreterNewStruct(b *testing.B) {

	scriptLocation := NewScriptLocationGenerator()

	inter, err := parseCheckAndInterpretWithOptions(
		b,
		`
        struct Foo {
            var id : Int

            init(_ id: Int) {
                self.id = id
            }
        }

        fun test(count: Int) {
            var i = 0
            while i < count {
                Foo(i)
                i = i + 1
            }
        }`,
		scriptLocation(),
		ParseCheckAndInterpretOptions{},
	)
	require.NoError(b, err)

	value := interpreter.NewIntValueFromInt64(nil, 10)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := inter.Invoke("test", value)
		require.NoError(b, err)
	}
}
