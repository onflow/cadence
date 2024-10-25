package test

import (
	"encoding/hex"
	"fmt"
	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/pretty"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/tests/checker"
	"github.com/onflow/cadence/tests/runtime_utils"
	"strings"
	"testing"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
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

func newContractDeployTransaction(name, code string) string {
	return fmt.Sprintf(
		`
                transaction {
                    prepare(signer: auth(Contracts) &Account) {
                        signer.contracts.%s(name: "%s", code: "%s".decodeHex())
                    }
                }
            `,
		sema.Account_ContractsTypeAddFunctionName,
		name,
		hex.EncodeToString([]byte(code)),
	)
}

func makeContractValueHandler(
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
) interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		constructor := constructorGenerator(common.ZeroAddress)

		value, err := inter.InvokeFunctionValue(
			constructor,
			arguments,
			argumentTypes,
			parameterTypes,
			compositeType,
			ast.Range{},
		)
		if err != nil {
			panic(err)
		}

		return value.(*interpreter.CompositeValue)
	}
}

func TestInterpreterFTTransfer(t *testing.T) {

	// ---- Deploy FT Contract -----

	storage := interpreter.NewInMemoryStorage(nil)

	contractsAddress := common.MustBytesToAddress([]byte{0x1})
	senderAddress := common.MustBytesToAddress([]byte{0x2})
	receiverAddress := common.MustBytesToAddress([]byte{0x3})

	flowTokenLocation := common.NewAddressLocation(nil, contractsAddress, "FlowToken")
	ftLocation := common.NewAddressLocation(nil, contractsAddress, "FungibleToken")

	programs := map[common.Location]*interpreter.Interpreter{}
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
			imported, ok := programs[location]
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
			imported, ok := programs[location]
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

			programs[location] = inter

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
	programs[ftLocation] = inter

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
