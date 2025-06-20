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

package runtime

import (
	"fmt"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type compiledProgram struct {
	program              *bbq.InstructionProgram
	desugaredElaboration *compiler.DesugaredElaboration
}

// vmEnvironmentReconfigured is the portion of vmEnvironment
// that gets reconfigured by vmEnvironment.Configure
type vmEnvironmentReconfigured struct {
	Interface
	storage *Storage
}

type vmEnvironment struct {
	vmEnvironmentReconfigured

	checkingEnvironment *checkingEnvironment

	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation

	config         Config
	vmConfig       *vm.Config
	compilerConfig *compiler.Config

	defaultCompilerBuiltinGlobals *activations.Activation[compiler.GlobalImport]
	defaultVMBuiltinGlobals       *activations.Activation[*vm.Variable]

	*stdlib.SimpleContractAdditionTracker
}

var _ Environment = &vmEnvironment{}
var _ stdlib.Logger = &vmEnvironment{}
var _ stdlib.RandomGenerator = &vmEnvironment{}
var _ stdlib.BlockAtHeightProvider = &vmEnvironment{}
var _ stdlib.CurrentBlockProvider = &vmEnvironment{}
var _ stdlib.AccountHandler = &vmEnvironment{}
var _ stdlib.AccountCreator = &vmEnvironment{}
var _ stdlib.EventEmitter = &vmEnvironment{}
var _ stdlib.PublicKeyValidator = &vmEnvironment{}
var _ stdlib.PublicKeySignatureVerifier = &vmEnvironment{}
var _ stdlib.BLSPoPVerifier = &vmEnvironment{}
var _ stdlib.BLSPublicKeyAggregator = &vmEnvironment{}
var _ stdlib.BLSSignatureAggregator = &vmEnvironment{}
var _ stdlib.Hasher = &vmEnvironment{}
var _ ArgumentDecoder = &vmEnvironment{}
var _ common.MemoryGauge = &vmEnvironment{}

func newVMEnvironment(config Config) *vmEnvironment {
	env := &vmEnvironment{
		config:                        config,
		SimpleContractAdditionTracker: stdlib.NewSimpleContractAdditionTracker(),
	}
	env.checkingEnvironment = newCheckingEnvironment()
	env.vmConfig = env.newVMConfig()
	env.compilerConfig = env.newCompilerConfig()

	env.defaultCompilerBuiltinGlobals = activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
	env.defaultVMBuiltinGlobals = activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

	for _, vmFunction := range stdlib.VMFunctions(env) {
		functionValue := vmFunction.FunctionValue
		qualifiedName := commons.TypeQualifiedName(
			vmFunction.BaseType,
			functionValue.Name,
		)
		env.defineValue(qualifiedName, functionValue)
	}

	for _, vmValue := range stdlib.VMValues(env) {
		env.defineValue(vmValue.Name, vmValue.Value)
	}

	return env
}

func NewBaseVMEnvironment(config Config) *vmEnvironment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.VMDefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptVMEnvironment(config Config) Environment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.VMDefaultScriptStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func (e *vmEnvironment) newVMConfig() *vm.Config {
	return &vm.Config{
		MemoryGauge:                    e,
		ComputationGauge:               e,
		TypeLoader:                     e.loadType,
		BuiltinGlobalsProvider:         e.vmBuiltinGlobals,
		ContractValueHandler:           e.loadContractValue,
		ImportHandler:                  e.importProgram,
		InjectedCompositeFieldsHandler: newInjectedCompositeFieldsHandler(e),
		UUIDHandler:                    newUUIDHandler(&e.Interface),
		AccountHandlerFunc:             e.newAccountValue,
		OnEventEmitted:                 newOnEventEmittedHandler(&e.Interface),
		CapabilityBorrowHandler:        newCapabilityBorrowHandler(e),
		CapabilityCheckHandler:         newCapabilityCheckHandler(e),
	}
}

func (e *vmEnvironment) defineValue(name string, value vm.Value) {

	if e.defaultCompilerBuiltinGlobals.Find(name) == (compiler.GlobalImport{}) {
		e.defaultCompilerBuiltinGlobals.Set(
			name,
			compiler.GlobalImport{
				Name: name,
			},
		)
	}

	variable := &vm.Variable{}
	variable.InitializeWithValue(value)
	e.defaultVMBuiltinGlobals.Set(name, variable)
}

func (e *vmEnvironment) loadContractValue(
	context *vm.Context,
	location common.Location,
) *interpreter.CompositeValue {
	addressLocation, ok := location.(common.AddressLocation)
	if !ok {
		panic(fmt.Errorf("cannot get contract value for non-address location %T", location))
	}

	return loadContractValue(
		context,
		addressLocation,
		e.storage,
	)
}

func (e *vmEnvironment) newCompilerConfig() *compiler.Config {
	return &compiler.Config{
		MemoryGauge:            e,
		BuiltinGlobalsProvider: e.compilerBuiltinGlobals,
		LocationHandler:        e.ResolveLocation,
		ImportHandler:          e.importProgram,
		ElaborationResolver:    e.resolveDesugaredElaboration,
	}
}

func (e *vmEnvironment) Configure(
	runtimeInterface Interface,
	codesAndPrograms CodesAndPrograms,
	storage *Storage,
	coverageReport *CoverageReport,
) {
	e.Interface = runtimeInterface
	e.storage = storage
	e.vmConfig.SetStorage(storage)

	e.checkingEnvironment.configure(
		runtimeInterface,
		codesAndPrograms,
	)

	// TODO: add support for coverage report
	_ = coverageReport

	configureVersionedFeatures(runtimeInterface)
}

func (e *vmEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.checkingEnvironment.declareValue(valueDeclaration, location)

	// TODO: add support for non-nil location
	if location != nil {
		panic(errors.NewUnreachableError())
	}

	// Define the value in the compiler builtin globals

	compilerBuiltinGlobals := e.defaultCompilerBuiltinGlobals

	name := valueDeclaration.Name

	compilerBuiltinGlobals.Set(
		name,
		compiler.GlobalImport{
			Name: name,
		},
	)

	// Define the value in the VM builtin globals

	variable := &vm.Variable{}
	variable.InitializeWithValue(valueDeclaration.Value)

	vmBuiltinGlobals := e.defaultVMBuiltinGlobals
	vmBuiltinGlobals.Set(name, variable)
}

func (e *vmEnvironment) DeclareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.checkingEnvironment.declareType(typeDeclaration, location)
}

func (e *vmEnvironment) SetCompositeValueFunctionsHandler(
	typeID common.TypeID,
	handler stdlib.CompositeValueFunctionsHandler,
) {
	// TODO:
	panic(errors.NewUnreachableError())
}

func (e *vmEnvironment) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	const commitContractUpdates = false
	return e.storage.Commit(context, commitContractUpdates)
}

func (e *vmEnvironment) EmitEvent(
	context interpreter.ValueExportContext,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	EmitEventFields(
		context,
		locationRange,
		eventType,
		values,
		e.Interface.EmitEvent,
	)
}

func (e *vmEnvironment) RecordContractRemoval(location common.AddressLocation) {
	e.storage.recordContractUpdate(location, nil)
}

func (e *vmEnvironment) RecordContractUpdate(
	location common.AddressLocation,
	contractValue *interpreter.CompositeValue,
) {
	e.storage.recordContractUpdate(location, contractValue)
}

func (e *vmEnvironment) ContractUpdateRecorded(location common.AddressLocation) bool {
	return e.storage.contractUpdateRecorded(location)
}

func (e *vmEnvironment) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	e.checkingEnvironment.temporarilyRecordCode(location, code)
}

func (e *vmEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.checkingEnvironment.ParseAndCheckProgram(code, location, getAndSetProgram)
}

func (e *vmEnvironment) ResolveLocation(
	identifiers []Identifier,
	location Location,
) (
	res []ResolvedLocation,
	err error,
) {
	return e.checkingEnvironment.resolveLocation(identifiers, location)
}

func (e *vmEnvironment) LoadContractValue(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	contract *interpreter.CompositeValue,
	err error,
) {
	e.deployedContractConstructorInvocation = &invocation
	defer func() {
		e.deployedContractConstructorInvocation = nil
	}()

	// TODO:
	panic(errors.NewUnreachableError())
}

func (e *vmEnvironment) newAccountValue(
	context interpreter.AccountCreationContext,
	address interpreter.AddressValue,
) interpreter.Value {
	return stdlib.NewAccountValue(context, e, address)
}

func (e *vmEnvironment) commitStorage(context interpreter.ValueTransferContext) error {
	checkStorageHealth := e.config.AtreeValidationEnabled
	return CommitStorage(context, e.storage, checkStorageHealth)
}

func (e *vmEnvironment) ProgramLog(message string, _ interpreter.LocationRange) error {
	return e.Interface.ProgramLog(message)
}

func (e *vmEnvironment) loadProgram(location common.Location) (*Program, error) {
	const getAndSetProgram = true
	program, err := e.checkingEnvironment.GetProgram(
		location,
		getAndSetProgram,
		importResolutionResults{},
	)
	if err != nil {
		return nil, err
	}

	// If there is a program, but it is not compiled yet, compile it.
	// Directly update the program (pointer), which will also update the program "cache" kept by the embedder.
	if program != nil && program.compiledProgram == nil {
		program.compiledProgram = e.compileProgram(
			program.interpreterProgram,
			location,
		)
	}

	return program, nil
}

func (e *vmEnvironment) loadDesugaredElaboration(location common.Location) (*compiler.DesugaredElaboration, error) {
	program, err := e.loadProgram(location)
	if err != nil {
		return nil, err
	}

	return program.compiledProgram.desugaredElaboration, nil
}

func (e *vmEnvironment) loadType(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
	if _, ok := location.(stdlib.FlowLocation); ok {
		return stdlib.FlowEventTypes[typeID]
	}

	elaboration, err := e.loadDesugaredElaboration(location)
	if err != nil {
		panic(fmt.Errorf(
			"cannot load type %s: failed to load elaboration for location %s: %w",
			typeID,
			location,
			err,
		))
	}

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

func (e *vmEnvironment) compileProgram(
	program *interpreter.Program,
	location common.Location,
) *compiledProgram {
	comp := compiler.NewInstructionCompilerWithConfig(
		program,
		location,
		e.compilerConfig,
	)

	return &compiledProgram{
		program:              comp.Compile(),
		desugaredElaboration: comp.DesugaredElaboration,
	}
}

func (e *vmEnvironment) resolveDesugaredElaboration(location common.Location) (*compiler.DesugaredElaboration, error) {
	program, err := e.loadProgram(location)
	if err != nil {
		return nil, err
	}

	return program.compiledProgram.desugaredElaboration, nil
}

func (e *vmEnvironment) importProgram(location common.Location) *bbq.InstructionProgram {
	program, err := e.loadProgram(location)
	if err != nil {
		panic(fmt.Errorf("failed to load program for imported location %s: %w", location, err))
	}
	return program.compiledProgram.program
}

func (e *vmEnvironment) newVM(
	location common.Location,
	program *bbq.InstructionProgram,
) *vm.VM {
	return vm.NewVM(
		location,
		program,
		e.vmConfig,
	)
}

func (e *vmEnvironment) vmBuiltinGlobals() *activations.Activation[*vm.Variable] {
	// TODO: add support for per-location VM builtin globals
	return e.defaultVMBuiltinGlobals
}

func (e *vmEnvironment) compilerBuiltinGlobals() *activations.Activation[compiler.GlobalImport] {
	// TODO: add support for per-location compiler builtin globals
	return e.defaultCompilerBuiltinGlobals
}
