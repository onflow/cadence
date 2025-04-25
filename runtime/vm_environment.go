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

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

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
		checkingEnvironment:           newCheckingEnvironment(),
		SimpleContractAdditionTracker: stdlib.NewSimpleContractAdditionTracker(),
	}
	env.vmConfig = env.newVMConfig()
	env.compilerConfig = env.newCompilerConfig()
	return env
}

func NewBaseVMEnvironment(config Config) *vmEnvironment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptVMEnvironment(config Config) Environment {
	env := newVMEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultScriptStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func (e *vmEnvironment) newVMConfig() *vm.Config {
	config := vm.NewConfig(nil)
	config.TypeLoader = e.loadType
	config.Logger = e
	config.ContractValueHandler = e.loadContractValue
	config.ImportHandler = e.importProgram
	return config
}

func (e *vmEnvironment) loadContractValue(conf *vm.Config, location common.Location) *interpreter.CompositeValue {
	addressLocation, ok := location.(common.AddressLocation)
	if !ok {
		panic(fmt.Errorf("cannot get contract value for non-address location %T", location))
	}

	return loadContractValue(
		vm.NewContext(conf),
		addressLocation,
		e.storage,
	)
}
func (e *vmEnvironment) newCompilerConfig() *compiler.Config {
	return &compiler.Config{
		LocationHandler: e.ResolveLocation,
		ImportHandler:   e.importProgram,
		ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
			// TODO: load and compile the contract program only once, register desugared elaboration
			program, err := e.loadProgram(location)
			if err != nil {
				panic(fmt.Errorf("failed to load elaboration for location %s: %w", location, err))
			}
			_, desugaredElaboration := e.compileProgram(program, location)
			return desugaredElaboration, nil
		},
	}
}

func (e *vmEnvironment) importProgram(location common.Location) *bbq.InstructionProgram {
	// TODO: load and compile the contract program only once, register desugared elaboration
	program, err := e.loadProgram(location)
	if err != nil {
		panic(fmt.Errorf("failed to load program for location %s: %w", location, err))
	}
	compiledProgram, _ := e.compileProgram(program, location)
	return compiledProgram
}

func (e *vmEnvironment) Configure(
	runtimeInterface Interface,
	codesAndPrograms CodesAndPrograms,
	storage *Storage,
// TODO:
	coverageReport *CoverageReport,
) {
	e.Interface = runtimeInterface
	e.storage = storage
	e.vmConfig.SetStorage(storage)

	e.checkingEnvironment.configure(
		runtimeInterface,
		codesAndPrograms,
	)

	configureVersionedFeatures(runtimeInterface)
}

func (e *vmEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.checkingEnvironment.declareValue(valueDeclaration, location)

	// TODO: declare in compiler and VM
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

func (e *vmEnvironment) loadProgram(location common.Location) (*interpreter.Program, error) {
	const getAndSetProgram = true
	program, err := e.checkingEnvironment.GetProgram(
		location,
		getAndSetProgram,
		importResolutionResults{},
	)
	if err != nil {
		return nil, err
	}

	return program, nil
}

func (e *vmEnvironment) loadElaboration(location common.Location) (*sema.Elaboration, error) {
	program, err := e.loadProgram(location)
	if err != nil {
		return nil, err
	}

	return program.Elaboration, nil
}
func (e *vmEnvironment) loadType(location common.Location, typeID interpreter.TypeID) sema.ContainedType {
	elaboration, err := e.loadElaboration(location)
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
) (
	*bbq.InstructionProgram,
	*compiler.DesugaredElaboration,
) {
	comp := compiler.NewInstructionCompilerWithConfig(
		program,
		location,
		e.compilerConfig,
	)

	compiledProgram := comp.Compile()
	return compiledProgram, comp.DesugaredElaboration
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
