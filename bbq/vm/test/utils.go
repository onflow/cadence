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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	. "github.com/onflow/cadence/bbq/test-utils"
	"github.com/onflow/cadence/bbq/vm"
)

type testAccountHandler struct {
	accountIDs                 map[common.Address]uint64
	generateAccountID          func(address common.Address) (uint64, error)
	getAccountBalance          func(address common.Address) (uint64, error)
	getAccountAvailableBalance func(address common.Address) (uint64, error)
	commitStorageTemporarily   func(context interpreter.ValueTransferContext) error
	getStorageUsed             func(address common.Address) (uint64, error)
	getStorageCapacity         func(address common.Address) (uint64, error)
	validatePublicKey          func(key *stdlib.PublicKey) error
	verifySignature            func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm sema.SignatureAlgorithm,
		hashAlgorithm sema.HashAlgorithm,
	) (
		bool,
		error,
	)
	blsVerifyPOP     func(publicKey *stdlib.PublicKey, signature []byte) (bool, error)
	hash             func(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error)
	getAccountKey    func(address common.Address, index uint32) (*stdlib.AccountKey, error)
	accountKeysCount func(address common.Address) (uint32, error)
	emitEvent        func(
		context interpreter.ValueExportContext,
		locationRange interpreter.LocationRange,
		eventType *sema.CompositeType,
		values []interpreter.Value,
	)
	addAccountKey func(
		address common.Address,
		key *stdlib.PublicKey,
		algo sema.HashAlgorithm,
		weight int,
	) (
		*stdlib.AccountKey,
		error,
	)
	revokeAccountKey       func(address common.Address, index uint32) (*stdlib.AccountKey, error)
	getAccountContractCode func(location common.AddressLocation) ([]byte, error)
	parseAndCheckProgram   func(
		code []byte,
		location common.Location,
		getAndSetProgram bool,
	) (
		*interpreter.Program,
		error,
	)
	updateAccountContractCode func(location common.AddressLocation, code []byte) error
	recordContractUpdate      func(location common.AddressLocation, value *interpreter.CompositeValue)
	contractUpdateRecorded    func(location common.AddressLocation) bool
	interpretContract         func(
		location common.AddressLocation,
		program *interpreter.Program,
		name string,
		invocation stdlib.DeployedContractConstructorInvocation,
	) (
		*interpreter.CompositeValue,
		error,
	)
	temporarilyRecordCode     func(location common.AddressLocation, code []byte)
	removeAccountContractCode func(location common.AddressLocation) error
	recordContractRemoval     func(location common.AddressLocation)
	getAccountContractNames   func(address common.Address) ([]string, error)
}

var _ stdlib.AccountHandler = &testAccountHandler{}

func (t *testAccountHandler) GenerateAccountID(address common.Address) (uint64, error) {
	if t.generateAccountID == nil {
		if t.accountIDs == nil {
			t.accountIDs = map[common.Address]uint64{}
		}
		t.accountIDs[address]++
		return t.accountIDs[address], nil
	}
	return t.generateAccountID(address)
}

func (t *testAccountHandler) GetAccountBalance(address common.Address) (uint64, error) {
	if t.getAccountBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountBalance"))
	}
	return t.getAccountBalance(address)
}

func (t *testAccountHandler) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	if t.getAccountAvailableBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountAvailableBalance"))
	}
	return t.getAccountAvailableBalance(address)
}

func (t *testAccountHandler) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	if t.commitStorageTemporarily == nil {
		panic(errors.NewUnexpectedError("unexpected call to CommitStorageTemporarily"))
	}
	return t.commitStorageTemporarily(context)
}

func (t *testAccountHandler) GetStorageUsed(address common.Address) (uint64, error) {
	if t.getStorageUsed == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageUsed"))
	}
	return t.getStorageUsed(address)
}

func (t *testAccountHandler) GetStorageCapacity(address common.Address) (uint64, error) {
	if t.getStorageCapacity == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageCapacity"))
	}
	return t.getStorageCapacity(address)
}

func (t *testAccountHandler) ValidatePublicKey(key *stdlib.PublicKey) error {
	if t.validatePublicKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to ValidatePublicKey"))
	}
	return t.validatePublicKey(key)
}

func (t *testAccountHandler) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm sema.SignatureAlgorithm,
	hashAlgorithm sema.HashAlgorithm,
) (
	bool,
	error,
) {
	if t.verifySignature == nil {
		panic(errors.NewUnexpectedError("unexpected call to VerifySignature"))
	}
	return t.verifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (t *testAccountHandler) BLSVerifyPOP(publicKey *stdlib.PublicKey, signature []byte) (bool, error) {
	if t.blsVerifyPOP == nil {
		panic(errors.NewUnexpectedError("unexpected call to BLSVerifyPOP"))
	}
	return t.blsVerifyPOP(publicKey, signature)
}

func (t *testAccountHandler) Hash(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error) {
	if t.hash == nil {
		panic(errors.NewUnexpectedError("unexpected call to Hash"))
	}
	return t.hash(data, tag, algorithm)
}

func (t *testAccountHandler) GetAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	if t.getAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountKey"))
	}
	return t.getAccountKey(address, index)
}

func (t *testAccountHandler) AccountKeysCount(address common.Address) (uint32, error) {
	if t.accountKeysCount == nil {
		panic(errors.NewUnexpectedError("unexpected call to AccountKeysCount"))
	}
	return t.accountKeysCount(address)
}

func (t *testAccountHandler) EmitEvent(
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	if t.emitEvent == nil {
		panic(errors.NewUnexpectedError("unexpected call to EmitEvent"))
	}
	t.emitEvent(
		context,
		locationRange,
		eventType,
		values,
	)
}

func (t *testAccountHandler) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (
	*stdlib.AccountKey,
	error,
) {
	if t.addAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to AddAccountKey"))
	}
	return t.addAccountKey(
		address,
		key,
		algo,
		weight,
	)
}

func (t *testAccountHandler) RevokeAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	if t.revokeAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to RevokeAccountKey"))
	}
	return t.revokeAccountKey(address, index)
}

func (t *testAccountHandler) GetAccountContractCode(location common.AddressLocation) ([]byte, error) {
	if t.getAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractCode"))
	}
	return t.getAccountContractCode(location)
}

func (t *testAccountHandler) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	if t.parseAndCheckProgram == nil {
		panic(errors.NewUnexpectedError("unexpected call to ParseAndCheckProgram"))
	}
	return t.parseAndCheckProgram(code, location, getAndSetProgram)
}

func (t *testAccountHandler) UpdateAccountContractCode(location common.AddressLocation, code []byte) error {
	if t.updateAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to UpdateAccountContractCode"))
	}
	return t.updateAccountContractCode(location, code)
}

func (t *testAccountHandler) RecordContractUpdate(location common.AddressLocation, value *interpreter.CompositeValue) {
	if t.recordContractUpdate == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractUpdate"))
	}
	t.recordContractUpdate(location, value)
}

func (t *testAccountHandler) ContractUpdateRecorded(location common.AddressLocation) bool {
	if t.contractUpdateRecorded == nil {
		panic(errors.NewUnexpectedError("unexpected call to ContractUpdateRecorded"))
	}
	return t.contractUpdateRecorded(location)
}

func (t *testAccountHandler) InterpretContract(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	*interpreter.CompositeValue,
	error,
) {
	if t.interpretContract == nil {
		panic(errors.NewUnexpectedError("unexpected call to InterpretContract"))
	}
	return t.interpretContract(
		location,
		program,
		name,
		invocation,
	)
}

func (t *testAccountHandler) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	if t.temporarilyRecordCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to TemporarilyRecordCode"))
	}
	t.temporarilyRecordCode(location, code)
}

func (t *testAccountHandler) RemoveAccountContractCode(location common.AddressLocation) error {
	if t.removeAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to RemoveAccountContractCode"))
	}
	return t.removeAccountContractCode(location)
}

func (t *testAccountHandler) RecordContractRemoval(location common.AddressLocation) {
	if t.recordContractRemoval == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractRemoval"))
	}
	t.recordContractRemoval(location)
}

func (t *testAccountHandler) GetAccountContractNames(address common.Address) ([]string, error) {
	if t.getAccountContractNames == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractNames"))
	}
	return t.getAccountContractNames(address)
}

func (t *testAccountHandler) StartContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) EndContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) IsContractBeingAdded(common.AddressLocation) bool {
	// NO-OP
	return false
}

type CompilerAndVMOptions struct {
	ParseCheckAndCompileOptions
	VMConfig *vm.Config
}

func CompileAndInvoke(
	t testing.TB,
	code string,
	funcName string,
	arguments ...vm.Value,
) (vm.Value, error) {
	return CompileAndInvokeWithOptions(
		t,
		code,
		funcName,
		CompilerAndVMOptions{},
		arguments...,
	)
}

func CompileAndInvokeWithLogs(
	t testing.TB,
	code string,
	funcName string,
	arguments ...vm.Value,
) (result vm.Value, err error, logs []string) {

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.PanicFunction)
	activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
		commons.LogFunctionName,
		sema.NewSimpleFunctionType(
			sema.FunctionPurityView,
			[]sema.Parameter{
				{
					Label:          sema.ArgumentLabelNotRequired,
					Identifier:     "value",
					TypeAnnotation: sema.AnyStructTypeAnnotation,
				},
			},
			sema.VoidTypeAnnotation,
		),
		"",
		nil,
	))

	storage := interpreter.NewInMemoryStorage(nil)
	vmConfig := vm.NewConfig(storage)

	vmConfig.NativeFunctionsProvider = func() map[string]vm.Value {
		funcs := vm.NativeFunctions()
		funcs[commons.LogFunctionName] = vm.NewNativeFunctionValue(
			commons.LogFunctionName,
			stdlib.LogFunctionType,
			func(_ *vm.Context, _ []interpreter.StaticType, arguments ...vm.Value) vm.Value {
				logs = append(logs, arguments[0].String())
				return interpreter.Void
			},
		)

		return funcs
	}

	result, err = CompileAndInvokeWithOptions(
		t,
		code,
		funcName,
		CompilerAndVMOptions{
			VMConfig: vmConfig,
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Config: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
			},
		},
		arguments...,
	)

	return
}

func CompileAndInvokeWithOptions(
	t testing.TB,
	code string,
	funcName string,
	options CompilerAndVMOptions,
	arguments ...vm.Value,
) (vm.Value, error) {

	programVM := CompileAndPrepareToInvoke(t, code, options)

	result, err := programVM.Invoke(funcName, arguments...)
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}

	return result, err
}

func CompileAndPrepareToInvoke(t testing.TB, code string, options CompilerAndVMOptions) *vm.VM {
	programs := map[common.Location]*CompiledProgram{}

	var location common.Location
	parseAndCheckOptions := options.ParseAndCheckOptions
	if parseAndCheckOptions != nil {
		location = parseAndCheckOptions.Location
	}

	if location == nil {
		location = TestLocation
	}

	program := ParseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		options.ParseCheckAndCompileOptions,
		programs,
	)

	// Ensure the program can be printed
	const resolve = false
	const colorize = false
	printer := bbq.NewInstructionsProgramPrinter(resolve, colorize)
	_ = printer.PrintProgram(program)

	vmConfig := prepareVMConfig(t, options.VMConfig, programs)

	if vmConfig.TypeLoader == nil {
		vmConfig.TypeLoader = typeLoader(t, programs)
	}

	programVM := vm.NewVM(
		location,
		program,
		vmConfig,
	)
	return programVM
}

func typeLoader(
	tb testing.TB,
	programs CompiledPrograms,
) func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
	return func(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
		program, ok := programs[location]
		require.True(tb, ok, "cannot find elaboration for %s", location)

		elaboration := program.DesugaredElaboration

		compositeType := elaboration.CompositeType(typeID)
		if compositeType != nil {
			return compositeType
		}

		interfaceType := elaboration.InterfaceType(typeID)
		if interfaceType != nil {
			return interfaceType
		}

		entitlementType := elaboration.EntitlementType(typeID)
		if entitlementType != nil {
			return entitlementType
		}

		entitlementMapType := elaboration.EntitlementMapType(typeID)
		if entitlementMapType != nil {
			return entitlementMapType
		}

		return nil
	}
}

func compileAndInvokeWithOptionsAndPrograms(
	t testing.TB,
	code string,
	funcName string,
	options CompilerAndVMOptions,
	programs CompiledPrograms,
	arguments ...vm.Value,
) (vm.Value, error) {

	location := common.ScriptLocation{0x1}
	program := ParseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		options.ParseCheckAndCompileOptions,
		programs,
	)

	vmConfig := prepareVMConfig(
		t,
		options.VMConfig,
		programs,
	)

	scriptLocation := NewScriptLocationGenerator()

	programVM := vm.NewVM(
		scriptLocation(),
		program,
		vmConfig,
	)

	result, err := programVM.Invoke(funcName, arguments...)
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}

	return result, err
}

func prepareVMConfig(
	tb testing.TB,
	config *vm.Config,
	programs CompiledPrograms,
) *vm.Config {

	if config == nil {
		storage := interpreter.NewInMemoryStorage(nil)
		config = vm.NewConfig(storage)
	}

	if config.GetAccountHandler() == nil {
		config = config.WithAccountHandler(&testAccountHandler{})
	}

	interpreterConfig := config.InterpreterConfig()
	if interpreterConfig == nil {
		interpreterConfig = &interpreter.Config{}
		config = config.WithInterpreterConfig(interpreterConfig)
	}

	if interpreterConfig.UUIDHandler == nil {
		var uuid uint64
		interpreterConfig.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}
	}

	if config.TypeLoader == nil {
		config.TypeLoader = typeLoader(tb, programs)
	}

	if config.ImportHandler == nil {
		config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			program, ok := programs[location]
			if !ok {
				assert.FailNow(tb, "invalid location")
			}
			return program.Program
		}
	}

	return config
}
