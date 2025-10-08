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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq/compiler"
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
	. "github.com/onflow/cadence/bbq/test_utils"
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
	loadContractValue         func(
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
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	if t.emitEvent == nil {
		panic(errors.NewUnexpectedError("unexpected call to EmitEvent"))
	}
	t.emitEvent(
		context,
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

func (t *testAccountHandler) LoadContractValue(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	*interpreter.CompositeValue,
	error,
) {
	if t.loadContractValue == nil {
		panic(errors.NewUnexpectedError("unexpected call to LoadContractValue"))
	}
	return t.loadContractValue(
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
	Programs map[common.Location]*CompiledProgram
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

func CompilerDefaultBuiltinGlobalsWithDefaultsAndLog(_ common.Location) *activations.Activation[compiler.GlobalImport] {
	activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

	activation.Set(
		stdlib.LogFunctionName,
		compiler.NewGlobalImport(stdlib.LogFunctionName),
	)

	return activation
}

func CompilerDefaultBuiltinGlobalsWithDefaultsAndPanic(_ common.Location) *activations.Activation[compiler.GlobalImport] {
	activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

	activation.Set(
		stdlib.PanicFunctionName,
		compiler.NewGlobalImport(stdlib.PanicFunctionName),
	)

	return activation
}

func CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog(_ common.Location) *activations.Activation[compiler.GlobalImport] {
	activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())

	activation.Set(
		conditionLogFunctionName,
		compiler.NewGlobalImport(conditionLogFunctionName),
	)

	return activation
}

func VMBuiltinGlobalsProviderWithDefaultsAndPanic(_ common.Location) *activations.Activation[vm.Variable] {
	activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

	panicFunctionVariable := &interpreter.SimpleVariable{}
	activation.Set(stdlib.PanicFunctionName, panicFunctionVariable)
	panicFunctionVariable.InitializeWithValue(
		vm.NewNativeFunctionValue(
			stdlib.PanicFunctionName,
			stdlib.PanicFunctionType,
			func(
				context interpreter.NativeFunctionContext,
				_ interpreter.LocationRange,
				_ interpreter.TypeParameterGetter,
				_ interpreter.Value,
				arguments ...interpreter.Value,
			) vm.Value {
				messageValue := interpreter.AssertValueOfType[*interpreter.StringValue](arguments[0])

				panic(&stdlib.PanicError{
					Message: messageValue.Str,
				})
			},
		),
	)

	return activation
}

func NewVMBuiltinGlobalsProviderWithDefaultsPanicAndLog(logs *[]string) vm.BuiltinGlobalsProvider {

	logFunction := stdlib.NewVMLogFunction(
		stdlib.FunctionLogger(
			func(message string) error {
				*logs = append(*logs, message)
				return nil
			},
		),
	)

	return func(location common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, VMBuiltinGlobalsProviderWithDefaultsAndPanic(location))

		logFunctionVariable := &interpreter.SimpleVariable{}
		logFunctionVariable.InitializeWithValue(logFunction.Value)
		activation.Set(stdlib.LogFunctionName, logFunctionVariable)

		return activation
	}
}

func NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(logs *[]string) vm.BuiltinGlobalsProvider {

	conditionLogFunction := newConditionLogFunction(logs)

	return func(location common.Location) *activations.Activation[vm.Variable] {
		activation := activations.NewActivation(nil, VMBuiltinGlobalsProviderWithDefaultsAndPanic(location))

		logFunctionVariable := &interpreter.SimpleVariable{}
		logFunctionVariable.InitializeWithValue(conditionLogFunction.Value)
		activation.Set(conditionLogFunctionName, logFunctionVariable)

		return activation
	}
}

func CompileAndInvokeWithLogs(
	t testing.TB,
	code string,
	funcName string,
	arguments ...vm.Value,
) (
	result vm.Value,
	logs []string,
	err error,
) {

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMPanicFunction)
	activation.DeclareValue(stdlib.NewVMLogFunction(nil))

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndLog,
	}

	storage := interpreter.NewInMemoryStorage(nil)
	vmConfig := vm.NewConfig(storage)

	vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndLog(&logs)

	result, err = CompileAndInvokeWithOptions(
		t,
		code,
		funcName,
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
				CompilerConfig: compilerConfig,
			},
			VMConfig: vmConfig,
		},
		arguments...,
	)

	return
}

const conditionLogFunctionName = "conditionLog"

var conditionLogFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: sema.AnyStructTypeAnnotation,
		},
	},
	sema.BoolTypeAnnotation,
)

func newConditionLogFunction(logs *[]string) stdlib.StandardLibraryValue {
	return stdlib.NewVMStandardLibraryStaticFunction(
		conditionLogFunctionName,
		conditionLogFunctionType,
		"",
		func(
			context interpreter.NativeFunctionContext,
			_ interpreter.LocationRange,
			_ interpreter.TypeParameterGetter,
			_ interpreter.Value,
			arguments ...interpreter.Value,
		) interpreter.Value {
			message := arguments[0].String()
			*logs = append(*logs, message)
			return interpreter.TrueValue
		},
	)
}

func CompileAndInvokeWithConditionLogs(
	t testing.TB,
	code string,
	funcName string,
	arguments ...vm.Value,
) (
	result vm.Value,
	logs []string,
	err error,
) {
	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.VMPanicFunction)
	activation.DeclareValue(newConditionLogFunction(nil))

	compilerConfig := &compiler.Config{
		BuiltinGlobalsProvider: CompilerDefaultBuiltinGlobalsWithDefaultsAndConditionLog,
	}

	storage := interpreter.NewInMemoryStorage(nil)
	vmConfig := vm.NewConfig(storage)

	vmConfig.BuiltinGlobalsProvider = NewVMBuiltinGlobalsProviderWithDefaultsPanicAndConditionLog(&logs)

	result, err = CompileAndInvokeWithOptions(
		t,
		code,
		funcName,
		CompilerAndVMOptions{
			ParseCheckAndCompileOptions: ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						LocationHandler: SingleIdentifierLocationResolver(t),
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return activation
						},
					},
				},
				CompilerConfig: compilerConfig,
			},
			VMConfig: vmConfig,
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

	programVM, err := CompileAndPrepareToInvoke(t, code, options)
	if err != nil {
		return nil, err
	}

	result, err := programVM.InvokeExternally(funcName, arguments...)
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}

	return result, err
}

func CompileAndPrepareToInvoke(t testing.TB, code string, options CompilerAndVMOptions) (programVM *vm.VM, err error) {
	programs := options.Programs
	if programs == nil {
		programs = map[common.Location]*CompiledProgram{}
	}

	var location common.Location
	parseAndCheckOptions := options.ParseAndCheckOptions

	if parseAndCheckOptions != nil {
		location = parseAndCheckOptions.Location
	}

	if location == nil {
		location = TestLocation
		if parseAndCheckOptions != nil {
			parseAndCheckOptions.Location = location
		}
	}

	program := ParseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		options.ParseCheckAndCompileOptions,
		programs,
	)

	vmConfig := PrepareVMConfig(t, options.VMConfig, programs)

	vmConfig.WithDebugEnabled()

	if vmConfig.ContractValueHandler == nil {
		// TODO: generalize this
		if len(program.Contracts) == 1 {
			vmConfig.ContractValueHandler = ContractValueHandler(program.Contracts[0].Name)
		}
	}

	// recover panics from VM (e.g. globals evaluation)
	defer func() {
		if r := recover(); r != nil {
			internalErr := interpreter.AsCadenceError(r)
			err = internalErr
		}
	}()

	programVM = vm.NewVM(
		location,
		program,
		vmConfig,
	)

	return programVM, nil
}

func ContractValueHandler(contractName string, arguments ...vm.Value) vm.ContractValueHandler {
	return func(context *vm.Context, location common.Location) *interpreter.CompositeValue {
		contractInitializerName := commons.QualifiedName(contractName, commons.InitFunctionName)
		contractInitializer := context.GetFunction(location, contractInitializerName)
		result := context.InvokeFunction(
			contractInitializer,
			arguments,
		)

		return result.(*interpreter.CompositeValue)
	}
}

func CompiledProgramsCompositeTypeLoader(
	programs CompiledPrograms,
) func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
	return func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
		program, ok := programs[location]
		if !ok {
			return nil
		}

		elaboration := program.DesugaredElaboration

		compositeType := elaboration.CompositeType(typeID)

		return compositeType
	}
}

func CompiledProgramsInterfaceTypeLoader(
	programs CompiledPrograms,
) func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
	return func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
		program, ok := programs[location]
		if !ok {
			return nil
		}

		elaboration := program.DesugaredElaboration

		interfaceType := elaboration.InterfaceType(typeID)

		return interfaceType
	}
}

func CompiledProgramsEntitlementTypeLoader(
	programs CompiledPrograms,
) func(location common.Location, typeID interpreter.TypeID) *sema.EntitlementType {
	return func(location common.Location, typeID interpreter.TypeID) *sema.EntitlementType {
		program, ok := programs[location]
		if !ok {
			return nil
		}

		elaboration := program.DesugaredElaboration

		entitlementType := elaboration.EntitlementType(typeID)
		if entitlementType != nil {
			return entitlementType
		}

		return nil
	}
}

func CompiledProgramsEntitlementMapTypeLoader(
	programs CompiledPrograms,
) func(location common.Location, typeID interpreter.TypeID) *sema.EntitlementMapType {
	return func(location common.Location, typeID interpreter.TypeID) *sema.EntitlementMapType {
		program, ok := programs[location]
		if !ok {
			return nil
		}

		elaboration := program.DesugaredElaboration

		entitlementMapType := elaboration.EntitlementMapType(typeID)
		if entitlementMapType != nil {
			return entitlementMapType
		}

		return nil
	}
}

func CompileAndInvokeWithOptionsAndPrograms(
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

	vmConfig := PrepareVMConfig(
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

	result, err := programVM.InvokeExternally(funcName, arguments...)
	if err == nil {
		require.Equal(t, 0, programVM.StackSize())
	}

	return result, err
}

func PrepareVMConfig(
	tb testing.TB,
	config *vm.Config,
	programs CompiledPrograms,
) *vm.Config {

	if config == nil {
		storage := interpreter.NewInMemoryStorage(nil)
		config = vm.NewConfig(storage)
	}

	if config.UUIDHandler == nil {
		var uuid uint64
		config.UUIDHandler = func() (uint64, error) {
			uuid++
			return uuid, nil
		}
	}

	if config.CompositeTypeHandler == nil {
		config.CompositeTypeHandler = CompiledProgramsCompositeTypeLoader(programs)
	}

	if config.InterfaceTypeHandler == nil {
		config.InterfaceTypeHandler = CompiledProgramsInterfaceTypeLoader(programs)
	}

	if config.EntitlementTypeHandler == nil {
		config.EntitlementTypeHandler = CompiledProgramsEntitlementTypeLoader(programs)
	}

	if config.EntitlementMapTypeHandler == nil {
		config.EntitlementMapTypeHandler = CompiledProgramsEntitlementMapTypeLoader(programs)
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

	if config.BuiltinGlobalsProvider == nil {
		config.BuiltinGlobalsProvider = VMBuiltinGlobalsProviderWithDefaultsAndPanic
	}

	if config.ElaborationResolver == nil {
		config.ElaborationResolver = func(location common.Location) (*sema.Elaboration, error) {
			imported, ok := programs[location]
			if !ok {
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}

			return imported.DesugaredElaboration.OriginalElaboration(), nil
		}
	}

	return config
}
