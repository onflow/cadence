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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils/runtime_utils"
	"github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
)

type testAccountHandler struct {
	accountIDs                 map[common.Address]uint64
	generateAccountID          func(address common.Address) (uint64, error)
	getAccountBalance          func(address common.Address) (uint64, error)
	getAccountAvailableBalance func(address common.Address) (uint64, error)
	commitStorageTemporarily   func(inter *interpreter.Interpreter) error
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
		inter *interpreter.Interpreter,
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

func (t *testAccountHandler) CommitStorageTemporarily(inter *interpreter.Interpreter) error {
	if t.commitStorageTemporarily == nil {
		panic(errors.NewUnexpectedError("unexpected call to CommitStorageTemporarily"))
	}
	return t.commitStorageTemporarily(inter)
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
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	if t.emitEvent == nil {
		panic(errors.NewUnexpectedError("unexpected call to EmitEvent"))
	}
	t.emitEvent(
		inter,
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

func singleIdentifierLocationResolver(t testing.TB) func(
	identifiers []ast.Identifier,
	location common.Location,
) ([]commons.ResolvedLocation, error) {
	return func(identifiers []ast.Identifier, location common.Location) ([]commons.ResolvedLocation, error) {
		require.Len(t, identifiers, 1)
		require.IsType(t, common.AddressLocation{}, location)

		return []commons.ResolvedLocation{
			{
				Location: common.AddressLocation{
					Address: location.(common.AddressLocation).Address,
					Name:    identifiers[0].Identifier,
				},
				Identifiers: identifiers,
			},
		}, nil
	}
}

func printProgram(name string, program *bbq.Program[opcode.Instruction]) {
	printer := bbq.NewInstructionsProgramPrinter()
	fmt.Println("===================", name, "===================")
	fmt.Println(printer.PrintProgram(program))
}

func baseValueActivation(common.Location) *sema.VariableActivation {
	// Only need to make the checker happy
	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(stdlib.PanicFunction)
	activation.DeclareValue(stdlib.NewStandardLibraryStaticFunction(
		"getAccount",
		stdlib.GetAccountFunctionType,
		"",
		nil,
	))
	return activation
}

type compiledProgram struct {
	*bbq.Program[opcode.Instruction]
	*sema.Elaboration
}

type CompilerAndVMOptions struct {
	*sema_utils.ParseAndCheckOptions
	CompilerConfig *compiler.Config
	VMConfig       *vm.Config
}

func parseCheckAndCompile(
	t testing.TB,
	code string,
	location common.Location,
	programs map[common.Location]*compiledProgram,
) *bbq.Program[opcode.Instruction] {
	return parseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		CompilerAndVMOptions{},
		programs,
	)
}

func parseCheckAndCompileCodeWithOptions(
	t testing.TB,
	code string,
	location common.Location,
	options CompilerAndVMOptions,
	programs map[common.Location]*compiledProgram,
) *bbq.Program[opcode.Instruction] {
	checker := parseAndCheckWithOptions(
		t,
		code,
		location,
		options.ParseAndCheckOptions,
		programs,
	)
	programs[checker.Location] = &compiledProgram{
		Elaboration: checker.Elaboration,
	}

	program := compile(
		t,
		options.CompilerConfig,
		checker,
		programs,
	)
	programs[checker.Location].Program = program

	return program
}

func parseAndCheck( // nolint:unused
	t testing.TB,
	code string,
	location common.Location,
	programs map[common.Location]*compiledProgram,
) *sema.Checker {
	return parseAndCheckWithOptions(t, code, location, nil, programs)
}

func parseAndCheckWithOptions(
	t testing.TB,
	code string,
	location common.Location,
	options *sema_utils.ParseAndCheckOptions,
	programs map[common.Location]*compiledProgram,
) *sema.Checker {

	var parseAndCheckOptions sema_utils.ParseAndCheckOptions
	if options != nil {
		parseAndCheckOptions = *options
	} else {
		parseAndCheckOptions = sema_utils.ParseAndCheckOptions{
			Location: location,
			Config: &sema.Config{
				LocationHandler:            singleIdentifierLocationResolver(t),
				BaseValueActivationHandler: baseValueActivation,
			},
		}
	}

	if parseAndCheckOptions.Config.ImportHandler == nil {
		parseAndCheckOptions.Config.ImportHandler = func(_ *sema.Checker, location common.Location, _ ast.Range) (sema.Import, error) {
			imported, ok := programs[location]
			if !ok {
				return nil, fmt.Errorf("cannot find contract in location %s", location)
			}

			return sema.ElaborationImport{
				Elaboration: imported.Elaboration,
			}, nil
		}
	}

	checker, err := sema_utils.ParseAndCheckWithOptions(
		t,
		code,
		parseAndCheckOptions,
	)
	require.NoError(t, err)
	return checker
}

func compile(
	t testing.TB,
	config *compiler.Config,
	checker *sema.Checker,
	programs map[common.Location]*compiledProgram,
) *bbq.Program[opcode.Instruction] {

	if config == nil {
		config = &compiler.Config{
			LocationHandler: singleIdentifierLocationResolver(t),
			ImportHandler: func(location common.Location) *bbq.Program[opcode.Instruction] {
				imported, ok := programs[location]
				if !ok {
					return nil
				}
				return imported.Program
			},
			ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
				imported, ok := programs[location]
				if !ok {
					return nil, fmt.Errorf("cannot find elaboration for %s", location)
				}
				return imported.Elaboration, nil
			},
		}
	}
	comp := compiler.NewInstructionCompiler(checker).
		WithConfig(config)

	program := comp.Compile()
	return program
}

func compileAndInvoke(
	t testing.TB,
	code string,
	funcName string,
	arguments ...vm.Value,
) (vm.Value, error) {
	return compileAndInvokeWithOptions(
		t,
		code,
		funcName,
		CompilerAndVMOptions{},
		arguments...,
	)
}

func compileAndInvokeWithOptions(
	t testing.TB,
	code string,
	funcName string,
	options CompilerAndVMOptions,
	arguments ...vm.Value,
) (vm.Value, error) {

	programs := map[common.Location]*compiledProgram{}

	location := common.ScriptLocation{0x1}
	program := parseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		options,
		programs,
	)

	printProgram("", program)

	vmConfig := options.VMConfig
	if vmConfig == nil {
		storage := interpreter.NewInMemoryStorage(nil)
		vmConfig = vm.NewConfig(storage).
			WithAccountHandler(&testAccountHandler{})

		vmConfig.TypeLoader = func(location common.Location, typeID interpreter.TypeID) sema.CompositeKindedType {
			program, ok := programs[location]
			require.True(t, ok, "cannot find elaboration for %s", location)

			elaboration := program.Elaboration
			compositeType := elaboration.CompositeType(typeID)
			if compositeType != nil {
				return compositeType
			}

			return elaboration.InterfaceType(typeID)
		}
	}

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

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

func compileAndInvokeWithOptionsAndPrograms(
	t testing.TB,
	code string,
	funcName string,
	options CompilerAndVMOptions,
	programs map[common.Location]*compiledProgram,
	arguments ...vm.Value,
) (vm.Value, error) {

	location := common.ScriptLocation{0x1}
	program := parseCheckAndCompileCodeWithOptions(
		t,
		code,
		location,
		options,
		programs,
	)

	vmConfig := options.VMConfig
	if vmConfig == nil {
		storage := interpreter.NewInMemoryStorage(nil)
		vmConfig = vm.NewConfig(storage).
			WithAccountHandler(&testAccountHandler{})
	}

	scriptLocation := runtime_utils.NewScriptLocationGenerator()

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
