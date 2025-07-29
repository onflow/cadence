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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func compiledFTTransfer(tb testing.TB) {

	compiledPrograms := CompiledPrograms{}

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
		flowTokenLocation:                  []byte(realFlowContract),
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	locationHandler := newSingleAddressOrStringLocationHandler(tb, contractsAddress)

	semaConfig := &sema.Config{
		LocationHandler:            locationHandler,
		BaseValueActivationHandler: TestBaseValueActivation,
	}

	importHandler := func(location common.Location) *bbq.InstructionProgram {
		imported, ok := compiledPrograms[location]
		if !ok {
			return nil
		}
		return imported.Program
	}

	compilerConfig := &compiler.Config{
		LocationHandler: locationHandler,
		ImportHandler:   importHandler,
		ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
			imported, ok := compiledPrograms[location]
			if !ok {
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
			return imported.DesugaredElaboration, nil
		},
		BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

			activation.Set(
				stdlib.AssertFunctionName,
				compiler.GlobalImport{
					Name: stdlib.AssertFunctionName,
				},
			)

			activation.Set(
				stdlib.GetAccountFunctionName,
				compiler.GlobalImport{
					Name: stdlib.GetAccountFunctionName,
				},
			)

			activation.Set(
				stdlib.PanicFunctionName,
				compiler.GlobalImport{
					Name: stdlib.PanicFunctionName,
				},
			)

			return activation
		},
	}

	// Parse and check contracts

	for _, location := range []common.Location{
		burnerLocation,
		viewResolverLocation,
		fungibleTokenLocation,
		nonFungibleTokenLocation,
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
		flowTokenLocation,
	} {
		_ = ParseCheckAndCompileCodeWithOptions(
			tb,
			string(codes[location]),
			location,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location:      location,
					CheckerConfig: semaConfig,
				},
				CompilerConfig: compilerConfig,
			},
			compiledPrograms,
		)
	}

	// Prepare VM

	accountHandler := &testAccountHandler{
		emitEvent: func(
			_ interpreter.ValueExportContext,
			_ interpreter.LocationRange,
			_ *sema.CompositeType,
			_ []interpreter.Value,
		) {
			// ignore
		},
	}

	storage := interpreter.NewInMemoryStorage(nil)

	vmConfig := vm.NewConfig(storage)

	vmConfig.CapabilityBorrowHandler = func(
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
	}

	vmConfig.OnEventEmitted = func(
		_ interpreter.ValueExportContext,
		_ interpreter.LocationRange,
		_ *sema.CompositeType,
		_ []interpreter.Value,
	) error {
		// NO-OP
		return nil
	}

	vmConfig.InjectedCompositeFieldsHandler = func(
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
	}

	vmConfig.AccountHandlerFunc = func(
		context interpreter.AccountCreationContext,
		address interpreter.AddressValue,
	) interpreter.Value {
		return stdlib.NewAccountValue(context, nil, address)
	}

	vmConfig.ImportHandler = importHandler

	contractValues := make(map[common.Location]*interpreter.CompositeValue)
	vmConfig.ContractValueHandler = func(
		_ *vm.Context,
		location common.Location,
	) *interpreter.CompositeValue {
		return contractValues[location]
	}

	vmConfig.BuiltinGlobalsProvider = func(_ common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

		panicVariable := &interpreter.SimpleVariable{}
		panicVariable.InitializeWithValue(stdlib.VMPanicFunction.Value)
		activation.Set(
			stdlib.PanicFunctionName,
			panicVariable,
		)

		assertVariable := &interpreter.SimpleVariable{}
		assertVariable.InitializeWithValue(stdlib.VMAssertFunction.Value)
		activation.Set(
			stdlib.AssertFunctionName,
			assertVariable,
		)

		getAccountVariable := &interpreter.SimpleVariable{}
		getAccountVariable.InitializeWithValue(stdlib.NewVMGetAccountFunction(accountHandler).Value)
		activation.Set(
			stdlib.GetAccountFunctionName,
			getAccountVariable,
		)

		for _, vmFunction := range []stdlib.VMFunction{
			stdlib.NewVMAccountCapabilitiesPublishFunction(accountHandler),
			stdlib.NewVMAccountStorageCapabilitiesIssueFunction(accountHandler),
			stdlib.NewVMAccountCapabilitiesGetFunction(accountHandler, true),
		} {
			variable := &interpreter.SimpleVariable{}
			variable.InitializeWithValue(vmFunction.FunctionValue)
			activation.Set(
				commons.TypeQualifiedName(
					vmFunction.BaseType,
					vmFunction.FunctionValue.Name,
				),
				variable,
			)
		}

		return activation
	}

	vmConfig = PrepareVMConfig(tb, vmConfig, compiledPrograms)

	// Initialize contracts

	for _, location := range []common.Location{
		metadataViewsLocation,
		fungibleTokenMetadataViewsLocation,
		flowTokenLocation,
	} {
		compiledProgram := compiledPrograms[location]
		_, contractValue := initializeContract(
			tb,
			location,
			compiledProgram.Program,
			vmConfig,
		)

		contractValues[location] = contractValue
	}

	// Setup accounts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		txLocation := nextTransactionLocation()

		program := ParseCheckAndCompileCodeWithOptions(
			tb,
			realFlowTokenSetupAccountTransaction,
			txLocation,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location:      txLocation,
					CheckerConfig: semaConfig,
				},
				CompilerConfig: compilerConfig,
			},
			compiledPrograms,
		)

		setupTxVM := vm.NewVM(txLocation, program, vmConfig)

		authorizer := stdlib.NewAccountReferenceValue(
			setupTxVM.Context(),
			accountHandler,
			interpreter.AddressValue(address),
			interpreter.FullyEntitledAccountAccess,
			interpreter.EmptyLocationRange,
		)
		err := setupTxVM.InvokeTransaction(nil, authorizer)
		require.NoError(tb, err)
		require.Equal(tb, 0, setupTxVM.StackSize())
	}

	// Mint FLOW to sender

	txLocation := nextTransactionLocation()

	mintTokensTxProgram := ParseCheckAndCompileCodeWithOptions(
		tb,
		realFlowTokenMintTokensTransaction,
		txLocation,
		ParseCheckAndCompileOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				Location:      txLocation,
				CheckerConfig: semaConfig,
			},
			CompilerConfig: compilerConfig,
		},
		compiledPrograms,
	)

	mintTxVM := vm.NewVM(txLocation, mintTokensTxProgram, vmConfig)

	total := uint64(1000000) * sema.Fix64Factor

	mintTxArgs := []vm.Value{
		interpreter.AddressValue(senderAddress),
		interpreter.NewUnmeteredUFix64Value(total),
	}

	// Use the same authorizations as the one defined in the transaction.
	semaAuthorization := sema.NewEntitlementSetAccess(
		[]*sema.EntitlementType{
			sema.BorrowValueType,
		},
		sema.Conjunction,
	)
	authorization := interpreter.ConvertSemaAccessToStaticAuthorization(nil, semaAuthorization)

	mintTxAuthorizer := stdlib.NewAccountReferenceValue(
		mintTxVM.Context(),
		accountHandler,
		interpreter.AddressValue(contractsAddress),
		authorization,
		interpreter.EmptyLocationRange,
	)

	err := mintTxVM.InvokeTransaction(mintTxArgs, mintTxAuthorizer)
	require.NoError(tb, err)
	require.Equal(tb, 0, mintTxVM.StackSize())

	// Run token transfer transaction

	txLocation = nextTransactionLocation()

	tokenTransferTxProgram := ParseCheckAndCompileCodeWithOptions(
		tb,
		realFlowTokenTransferTokensTransaction,
		txLocation,
		ParseCheckAndCompileOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				Location:      txLocation,
				CheckerConfig: semaConfig,
			},
			CompilerConfig: compilerConfig,
		},
		compiledPrograms,
	)

	tokenTransferTxVM := vm.NewVM(txLocation, tokenTransferTxProgram, vmConfig)

	transferAmount := uint64(1) * sema.Fix64Factor

	tokenTransferTxArgs := []vm.Value{
		interpreter.NewUnmeteredUFix64Value(transferAmount),
		interpreter.AddressValue(receiverAddress),
	}

	tokenTransferTxAuthorizer := stdlib.NewAccountReferenceValue(
		tokenTransferTxVM.Context(),
		accountHandler,
		interpreter.AddressValue(senderAddress),
		authorization,
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

	vmConfig.Tracer = interpreter.CallbackTracer(printTrace)

	for loop() {
		err := tokenTransferTxVM.InvokeTransaction(tokenTransferTxArgs, tokenTransferTxAuthorizer)
		require.NoError(tb, err)
		require.Equal(tb, 0, tokenTransferTxVM.StackSize())

		tokenTransferTxVM.Reset()

		transferCount++
	}

	if b != nil {
		b.StopTimer()
	}

	vmConfig.Tracer = nil

	// Run validation scripts

	for _, address := range []common.Address{
		senderAddress,
		receiverAddress,
	} {
		scriptLocation := nextScriptLocation()

		program := ParseCheckAndCompileCodeWithOptions(
			tb,
			realFlowTokenGetBalanceScript,
			scriptLocation,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Location:      scriptLocation,
					CheckerConfig: semaConfig,
				},
				CompilerConfig: compilerConfig,
			},
			compiledPrograms,
		)

		validationScriptVM := vm.NewVM(scriptLocation, program, vmConfig)

		addressValue := interpreter.AddressValue(address)
		result, err := validationScriptVM.InvokeExternally("main", addressValue)
		require.NoError(tb, err)
		require.Equal(tb, 0, validationScriptVM.StackSize())

		if address == senderAddress {
			assert.Equal(tb, interpreter.NewUnmeteredUFix64Value(total-transferAmount*uint64(transferCount)), result)
		} else {
			assert.Equal(tb, interpreter.NewUnmeteredUFix64Value(transferAmount*uint64(transferCount)), result)
		}
	}
}

func TestFTTransfer(t *testing.T) {
	t.Parallel()

	compiledFTTransfer(t)
}

func BenchmarkFTTransfer(b *testing.B) {

	compiledFTTransfer(b)
}

func printTrace(
	operationName string,
	_ time.Duration,
	attrs []attribute.KeyValue,
) {
	sb := strings.Builder{}
	sb.WriteString(operationName)

	attributesLength := len(attrs)

	if attributesLength > 0 {
		sb.WriteString(": ")
		for i, attr := range attrs {

			key := string(attr.Key)
			if key == "value" {
				continue
			}

			sb.WriteString(string(attr.Key))
			sb.WriteString(":")
			sb.WriteString(attr.Value.AsString())

			if i < attributesLength-1 {
				sb.WriteString(", ")
			}
		}
	}

	fmt.Println(sb.String())
}
