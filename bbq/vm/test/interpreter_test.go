package test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

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
	"github.com/onflow/cadence/tests/checker"
	"github.com/onflow/cadence/tests/runtime_utils"
	"github.com/onflow/cadence/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	checker, err := checker.ParseAndCheckWithOptionsAndMemoryMetering(t,
		code,
		checker.ParseAndCheckOptions{
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
	if memoryGauge == nil {
		config.AtreeValueValidationEnabled = true
		config.AtreeStorageValidationEnabled = true
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

func TestInterpreterFTTransfer(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

	subInterpreters := map[common.Location]*interpreter.Interpreter{}
	codes := map[common.Location][]byte{
		ftLocation: []byte(realFungibleTokenContractInterface),
	}

	txLocation := runtime_utils.NewTransactionLocationGenerator()
	scriptLocation := runtime_utils.NewScriptLocationGenerator()

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
		emitEvent: func(*interpreter.Interpreter, interpreter.LocationRange, *sema.CompositeType, []interpreter.Value) {
			// do nothing
		},
		recordContractUpdate: func(location common.AddressLocation, value *interpreter.CompositeValue) {
			// do nothing
		},
	}

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)
	interpreter.Declare(baseActivation, stdlib.NewGetAccountFunction(accountHandler))

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
		LocationHandler:            singleIdentifierLocationResolver(t),
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

			value, err := inter.InvokeFunctionValue(
				constructor,
				[]interpreter.Value{signer},
				[]sema.Type{
					sema.FullyEntitledAccountReferenceType,
				},
				[]sema.Type{
					sema.FullyEntitledAccountReferenceType,
				},
				compositeType,
				ast.Range{},
			)
			if err != nil {
				panic(err)
			}

			flowTokenContractValue = value.(*interpreter.CompositeValue)
			return flowTokenContractValue
		},
		CapabilityBorrowHandler: func(
			inter *interpreter.Interpreter,
			locationRange interpreter.LocationRange,
			address interpreter.AddressValue,
			capabilityID interpreter.UInt64Value,
			wantedBorrowType *sema.ReferenceType,
			capabilityBorrowType *sema.ReferenceType,
		) interpreter.ReferenceValue {
			return stdlib.BorrowCapabilityController(
				inter,
				locationRange,
				address,
				capabilityID,
				wantedBorrowType,
				capabilityBorrowType,
				accountHandler,
			)
		},
	}

	accountHandler.parseAndCheckProgram =
		func(code []byte, location common.Location, getAndSetProgram bool) (*interpreter.Program, error) {
			if subInterpreter, ok := subInterpreters[location]; ok {
				return subInterpreter.Program, nil
			}

			inter, err := parseCheckAndInterpretWithOptions(
				t,
				string(code),
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

			return inter.Program, err
		}

	// ----- Parse and Check FungibleToken Contract interface -----

	inter, err := parseCheckAndInterpretWithOptions(
		t,
		realFungibleTokenContractInterface,
		ftLocation,
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(t, err)
	subInterpreters[ftLocation] = inter

	// ----- Deploy FlowToken Contract -----

	tx := fmt.Sprintf(`
        transaction {
            prepare(signer: auth(Storage, Capabilities, Contracts) &Account) {
                signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
            }
        }`,
		hex.EncodeToString([]byte(realFlowContract)),
	)

	inter, err = parseCheckAndInterpretWithOptions(
		t,
		tx,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(t, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(contractsAddress),
		interpreter.FullyEntitledAccountAccess,
		interpreter.EmptyLocationRange,
	)

	err = inter.InvokeTransaction(0, signer)
	require.NoError(t, err)

	// ----- Run setup account transaction -----

	authorization := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{
			sema.BorrowValueType,
			sema.IssueStorageCapabilityControllerType,
			sema.PublishCapabilityType,
			sema.SaveValueType,
		},
		sema.Conjunction,
	)

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		inter, err := parseCheckAndInterpretWithOptions(
			t,
			realSetupFlowTokenAccountTransaction,
			txLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(address),
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(0, signer)
		require.NoError(t, err)
	}

	// Mint FLOW to sender

	total := int64(1000000)

	inter, err = parseCheckAndInterpretWithOptions(
		t,
		realMintFlowTokenTransaction,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(t, err)

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
		interpreter.NewUnmeteredIntValueFromInt64(total),
		signer,
	)
	require.NoError(t, err)

	// ----- Run token transfer transaction -----

	transferAmount := int64(1)

	inter, err = parseCheckAndInterpretWithOptions(
		t,
		realFlowTokenTransferTransaction,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(t, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(senderAddress),
		interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
		interpreter.EmptyLocationRange,
	)

	err = inter.InvokeTransaction(
		0,
		interpreter.NewUnmeteredIntValueFromInt64(transferAmount),
		interpreter.AddressValue(receiverAddress),
		signer,
	)
	require.NoError(t, err)

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		inter, err = parseCheckAndInterpretWithOptions(
			t,
			realFlowTokenBalanceScript,
			scriptLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(t, err)

		result, err := inter.Invoke(
			"main",
			interpreter.AddressValue(address),
		)
		require.NoError(t, err)

		if address == senderAddress {
			assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(total-transferAmount), result)
		} else {
			assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(transferAmount), result)
		}
	}
}

func BenchmarkInterpreterFTTransfer(b *testing.B) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

	subInterpreters := map[common.Location]*interpreter.Interpreter{}
	codes := map[common.Location][]byte{
		ftLocation: []byte(realFungibleTokenContractInterface),
	}

	txLocation := runtime_utils.NewTransactionLocationGenerator()

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
		emitEvent: func(*interpreter.Interpreter, interpreter.LocationRange, *sema.CompositeType, []interpreter.Value) {
			// do nothing
		},
		recordContractUpdate: func(location common.AddressLocation, value *interpreter.CompositeValue) {
			// do nothing
		},
	}

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.PanicFunction)
	interpreter.Declare(baseActivation, stdlib.NewGetAccountFunction(accountHandler))

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
		LocationHandler:            singleIdentifierLocationResolver(b),
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

			value, err := inter.InvokeFunctionValue(
				constructor,
				[]interpreter.Value{signer},
				[]sema.Type{
					sema.FullyEntitledAccountReferenceType,
				},
				[]sema.Type{
					sema.FullyEntitledAccountReferenceType,
				},
				compositeType,
				ast.Range{},
			)
			if err != nil {
				panic(err)
			}

			flowTokenContractValue = value.(*interpreter.CompositeValue)
			return flowTokenContractValue
		},
		CapabilityBorrowHandler: func(
			inter *interpreter.Interpreter,
			locationRange interpreter.LocationRange,
			address interpreter.AddressValue,
			capabilityID interpreter.UInt64Value,
			wantedBorrowType *sema.ReferenceType,
			capabilityBorrowType *sema.ReferenceType,
		) interpreter.ReferenceValue {
			return stdlib.BorrowCapabilityController(
				inter,
				locationRange,
				address,
				capabilityID,
				wantedBorrowType,
				capabilityBorrowType,
				accountHandler,
			)
		},
	}

	accountHandler.parseAndCheckProgram =
		func(code []byte, location common.Location, getAndSetProgram bool) (*interpreter.Program, error) {
			if subInterpreter, ok := subInterpreters[location]; ok {
				return subInterpreter.Program, nil
			}

			inter, err := parseCheckAndInterpretWithOptions(
				b,
				string(code),
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

			return inter.Program, err
		}

	// ----- Parse and Check FungibleToken Contract interface -----

	inter, err := parseCheckAndInterpretWithOptions(
		b,
		realFungibleTokenContractInterface,
		ftLocation,
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(b, err)
	subInterpreters[ftLocation] = inter

	// ----- Deploy FlowToken Contract -----

	tx := fmt.Sprintf(`
        transaction {
            prepare(signer: auth(Storage, Capabilities, Contracts) &Account) {
                signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
            }
        }`,
		hex.EncodeToString([]byte(realFlowContract)),
	)

	inter, err = parseCheckAndInterpretWithOptions(
		b,
		tx,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(b, err)

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(contractsAddress),
		interpreter.FullyEntitledAccountAccess,
		interpreter.EmptyLocationRange,
	)

	err = inter.InvokeTransaction(0, signer)
	require.NoError(b, err)

	// ----- Run setup account transaction -----

	authorization := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{
			sema.BorrowValueType,
			sema.IssueStorageCapabilityControllerType,
			sema.PublishCapabilityType,
			sema.SaveValueType,
		},
		sema.Conjunction,
	)

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		inter, err := parseCheckAndInterpretWithOptions(
			b,
			realSetupFlowTokenAccountTransaction,
			txLocation(),
			ParseCheckAndInterpretOptions{
				Config:        interConfig,
				CheckerConfig: checkerConfig,
			},
		)
		require.NoError(b, err)

		signer = stdlib.NewAccountReferenceValue(
			inter,
			accountHandler,
			interpreter.AddressValue(address),
			interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
			interpreter.EmptyLocationRange,
		)

		err = inter.InvokeTransaction(0, signer)
		require.NoError(b, err)
	}

	// Mint FLOW to sender

	total := int64(1000000)

	inter, err = parseCheckAndInterpretWithOptions(
		b,
		realMintFlowTokenTransaction,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(b, err)

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
		interpreter.NewUnmeteredIntValueFromInt64(total),
		signer,
	)
	require.NoError(b, err)

	// ----- Run token transfer transaction -----

	signer = stdlib.NewAccountReferenceValue(
		inter,
		accountHandler,
		interpreter.AddressValue(senderAddress),
		interpreter.ConvertSemaAccessToStaticAuthorization(nil, authorization),
		interpreter.EmptyLocationRange,
	)

	inter, err = parseCheckAndInterpretWithOptions(
		b,
		realFlowTokenTransferTransaction,
		txLocation(),
		ParseCheckAndInterpretOptions{
			Config:        interConfig,
			CheckerConfig: checkerConfig,
		},
	)
	require.NoError(b, err)

	transferAmount := int64(1)

	amount := interpreter.NewUnmeteredIntValueFromInt64(transferAmount)
	receiver := interpreter.AddressValue(receiverAddress)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = inter.InvokeTransaction(
			0,
			amount,
			receiver,
			signer,
		)
		require.NoError(b, err)
	}

	b.StopTimer()
}

func BenchmarkRuntimeFungibleTokenTransfer(b *testing.B) {

	interpreterRuntime := runtime_utils.NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := contractsAddress

	runtimeInterface := &runtime_utils.TestRuntimeInterface{
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: runtime_utils.NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]common.Address, error) {
			return []common.Address{signerAccount}, nil
		},
		OnResolveLocation: runtime_utils.NewSingleIdentifierLocationResolver(b),
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

	nextTransactionLocation := runtime_utils.NewTransactionLocationGenerator()

	// Deploy Fungible Token contract

	err := interpreterRuntime.ExecuteTransaction(
		runtime.Script{
			Source: utils.DeploymentTransaction(
				"FungibleToken",
				[]byte(realFungibleTokenContractInterface),
			),
		},
		runtime.Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	// Deploy Flow Token contract

	err = interpreterRuntime.ExecuteTransaction(
		runtime.Script{
			Source: []byte(fmt.Sprintf(`
                transaction {
                    prepare(signer: auth(Storage, Capabilities, Contracts) &Account) {
                        signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                    }
                }`,
				hex.EncodeToString([]byte(realFlowContract)),
			)),
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
				Source: []byte(realSetupFlowTokenAccountTransaction),
			},
			runtime.Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
		require.NoError(b, err)
	}

	// Mint 1000 FLOW to sender

	amount := 100000000000
	mintAmount := cadence.NewInt(amount)
	mintAmountValue := interpreter.NewUnmeteredIntValueFromInt64(int64(amount))

	signerAccount = contractsAddress

	err = interpreterRuntime.ExecuteTransaction(
		runtime.Script{
			Source: []byte(realMintFlowTokenTransaction),
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

	sendAmount := cadence.NewInt(1)

	signerAccount = senderAddress

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		err = interpreterRuntime.ExecuteTransaction(
			runtime.Script{
				Source: []byte(realFlowTokenTransferTransaction),
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

	sum := interpreter.NewUnmeteredIntValueFromInt64(0)

	inter := runtime_utils.NewTestInterpreter(b)

	nextScriptLocation := runtime_utils.NewScriptLocationGenerator()

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {

		result, err := interpreterRuntime.ExecuteScript(
			runtime.Script{
				Source: []byte(realFlowTokenBalanceScript),
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

		value := interpreter.NewUnmeteredIntValueFromBigInt(result.(cadence.Int).Big())

		require.True(b, bool(value.Less(inter, mintAmountValue, interpreter.EmptyLocationRange)))

		sum = sum.Plus(inter, value, interpreter.EmptyLocationRange).(interpreter.IntValue)
	}

	utils.RequireValuesEqual(b, nil, mintAmountValue, sum)
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
