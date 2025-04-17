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
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/activations"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type Environment interface {
	ArgumentDecoder

	SetCompositeValueFunctionsHandler(
		typeID common.TypeID,
		handler stdlib.CompositeValueFunctionsHandler,
	)
	DeclareValue(
		valueDeclaration stdlib.StandardLibraryValue,
		location common.Location,
	)
	DeclareType(
		typeDeclaration stdlib.StandardLibraryType,
		location common.Location,
	)
	Configure(
		runtimeInterface Interface,
		codesAndPrograms CodesAndPrograms,
		storage *Storage,
		coverageReport *CoverageReport,
	)
	ParseAndCheckProgram(
		code []byte,
		location common.Location,
		getAndSetProgram bool,
	) (
		*interpreter.Program,
		error,
	)
	Interpret(
		location common.Location,
		program *interpreter.Program,
		f InterpretFunc,
	) (
		interpreter.Value,
		*interpreter.Interpreter,
		error,
	)
	commitStorage(context interpreter.ValueTransferContext) error
	NewAccountValue(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value
}

// interpreterEnvironmentReconfigured is the portion of interpreterEnvironment
// that gets reconfigured by interpreterEnvironment.Configure
type interpreterEnvironmentReconfigured struct {
	runtimeInterface Interface
	storage          *Storage
	coverageReport   *CoverageReport
}

type interpreterEnvironment struct {
	interpreterEnvironmentReconfigured

	checkingEnvironment *checkingEnvironment

	// defaultBaseActivation is the base activation that applies to all locations by default
	defaultBaseActivation *interpreter.VariableActivation
	// The base activations for individual locations.
	// location == nil is the base activation that applies to all locations,
	// unless there is a base activation for the given location.
	//
	// Base activations are lazily / implicitly created
	// by DeclareValue / interpreterBaseActivationFor
	baseActivationsByLocation map[common.Location]*interpreter.VariableActivation

	InterpreterConfig *interpreter.Config

	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation
	stackDepthLimiter                     *stackDepthLimiter
	compositeValueFunctionsHandlers       stdlib.CompositeValueFunctionsHandlers
	config                                Config
	deployedContracts                     map[Location]struct{}
}

var _ Environment = &interpreterEnvironment{}
var _ stdlib.Logger = &interpreterEnvironment{}
var _ stdlib.RandomGenerator = &interpreterEnvironment{}
var _ stdlib.BlockAtHeightProvider = &interpreterEnvironment{}
var _ stdlib.CurrentBlockProvider = &interpreterEnvironment{}
var _ stdlib.AccountHandler = &interpreterEnvironment{}
var _ stdlib.AccountCreator = &interpreterEnvironment{}
var _ stdlib.EventEmitter = &interpreterEnvironment{}
var _ stdlib.PublicKeyValidator = &interpreterEnvironment{}
var _ stdlib.PublicKeySignatureVerifier = &interpreterEnvironment{}
var _ stdlib.BLSPoPVerifier = &interpreterEnvironment{}
var _ stdlib.BLSPublicKeyAggregator = &interpreterEnvironment{}
var _ stdlib.BLSSignatureAggregator = &interpreterEnvironment{}
var _ stdlib.Hasher = &interpreterEnvironment{}
var _ ArgumentDecoder = &interpreterEnvironment{}
var _ common.MemoryGauge = &interpreterEnvironment{}

func NewInterpreterEnvironment(config Config) *interpreterEnvironment {
	defaultBaseActivation := activations.NewActivation(nil, interpreter.BaseActivation)

	env := &interpreterEnvironment{
		config:                config,
		checkingEnvironment:   newCheckingEnvironment(),
		defaultBaseActivation: defaultBaseActivation,
		stackDepthLimiter:     newStackDepthLimiter(config.StackDepthLimit),
	}
	env.InterpreterConfig = env.NewInterpreterConfig()

	env.compositeValueFunctionsHandlers = stdlib.DefaultStandardLibraryCompositeValueFunctionHandlers(env)
	return env
}

func (e *interpreterEnvironment) NewInterpreterConfig() *interpreter.Config {
	return &interpreter.Config{
		MemoryGauge:                    e,
		BaseActivationHandler:          e.getBaseActivation,
		OnEventEmitted:                 newOnEventEmittedHandler(&e.runtimeInterface),
		InjectedCompositeFieldsHandler: newInjectedCompositeFieldsHandler(e),
		UUIDHandler:                    newUUIDHandler(&e.runtimeInterface),
		ContractValueHandler:           e.newContractValueHandler(),
		ImportLocationHandler:          e.newImportLocationHandler(),
		AccountHandler:                 e.NewAccountValue,
		OnRecordTrace:                  newOnRecordTraceHandler(&e.runtimeInterface),
		OnResourceOwnerChange:          e.newResourceOwnerChangedHandler(),
		CompositeTypeHandler:           e.newCompositeTypeHandler(),
		CompositeValueFunctionsHandler: e.newCompositeValueFunctionsHandler(),
		TracingEnabled:                 e.config.TracingEnabled,
		AtreeValueValidationEnabled:    e.config.AtreeValidationEnabled,
		// NOTE: ignore e.config.AtreeValidationEnabled here,
		// and disable storage validation after each value modification.
		// Instead, storage is validated after commits (if validation is enabled),
		// see interpreterEnvironment.commitStorage
		AtreeStorageValidationEnabled:             false,
		Debugger:                                  e.config.Debugger,
		OnStatement:                               e.newOnStatementHandler(),
		OnMeterComputation:                        newOnMeterComputation(&e.runtimeInterface),
		OnFunctionInvocation:                      e.newOnFunctionInvocationHandler(),
		OnInvokedFunctionReturn:                   e.newOnInvokedFunctionReturnHandler(),
		CapabilityBorrowHandler:                   newCapabilityBorrowHandler(e),
		CapabilityCheckHandler:                    newCapabilityCheckHandler(e),
		ValidateAccountCapabilitiesGetHandler:     newValidateAccountCapabilitiesGetHandler(&e.runtimeInterface),
		ValidateAccountCapabilitiesPublishHandler: newValidateAccountCapabilitiesPublishHandler(&e.runtimeInterface),
	}
}

func NewBaseInterpreterEnvironment(config Config) *interpreterEnvironment {
	env := NewInterpreterEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptInterpreterEnvironment(config Config) Environment {
	env := NewInterpreterEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultScriptStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func (e *interpreterEnvironment) Configure(
	runtimeInterface Interface,
	codesAndPrograms CodesAndPrograms,
	storage *Storage,
	coverageReport *CoverageReport,
) {
	e.runtimeInterface = runtimeInterface
	e.storage = storage
	e.InterpreterConfig.Storage = storage
	e.coverageReport = coverageReport
	e.stackDepthLimiter.depth = 0

	e.checkingEnvironment.configure(
		runtimeInterface,
		codesAndPrograms,
	)

	configureVersionedFeatures(runtimeInterface)
}

func (e *interpreterEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.checkingEnvironment.declareValue(valueDeclaration, location)

	activation := e.interpreterBaseActivationFor(location)
	interpreter.Declare(activation, valueDeclaration)
}

func (e *interpreterEnvironment) DeclareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.checkingEnvironment.declareType(typeDeclaration, location)
}

func (e *interpreterEnvironment) interpreterBaseActivationFor(
	location common.Location,
) *interpreter.VariableActivation {
	defaultBaseActivation := e.defaultBaseActivation
	if location == nil {
		return defaultBaseActivation
	}

	baseActivation := e.baseActivationsByLocation[location]
	if baseActivation == nil {
		baseActivation = activations.NewActivation[interpreter.Variable](nil, defaultBaseActivation)
		if e.baseActivationsByLocation == nil {
			e.baseActivationsByLocation = map[common.Location]*interpreter.VariableActivation{}
		}
		e.baseActivationsByLocation[location] = baseActivation
	}
	return baseActivation
}

func (e *interpreterEnvironment) SetCompositeValueFunctionsHandler(
	typeID common.TypeID,
	handler stdlib.CompositeValueFunctionsHandler,
) {
	e.compositeValueFunctionsHandlers[typeID] = handler
}

func (e *interpreterEnvironment) MeterMemory(usage common.MemoryUsage) error {
	return e.runtimeInterface.MeterMemory(usage)
}

func (e *interpreterEnvironment) ProgramLog(message string, _ interpreter.LocationRange) error {
	return e.runtimeInterface.ProgramLog(message)
}

func (e *interpreterEnvironment) ReadRandom(buffer []byte) error {
	return e.runtimeInterface.ReadRandom(buffer)
}

func (e *interpreterEnvironment) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {
	return e.runtimeInterface.GetBlockAtHeight(height)
}

func (e *interpreterEnvironment) GetCurrentBlockHeight() (uint64, error) {
	return e.runtimeInterface.GetCurrentBlockHeight()
}

func (e *interpreterEnvironment) GetAccountBalance(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetAccountBalance(address)
}

func (e *interpreterEnvironment) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetAccountAvailableBalance(address)
}

func (e *interpreterEnvironment) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	const commitContractUpdates = false
	return e.storage.Commit(context, commitContractUpdates)
}

func (e *interpreterEnvironment) GetStorageUsed(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetStorageUsed(address)
}

func (e *interpreterEnvironment) GetStorageCapacity(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetStorageCapacity(address)
}

func (e *interpreterEnvironment) GetAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	return e.runtimeInterface.GetAccountKey(address, index)
}

func (e *interpreterEnvironment) AccountKeysCount(address common.Address) (uint32, error) {
	return e.runtimeInterface.AccountKeysCount(address)
}

func (e *interpreterEnvironment) GetAccountContractNames(address common.Address) ([]string, error) {
	return e.runtimeInterface.GetAccountContractNames(address)
}

func (e *interpreterEnvironment) GetAccountContractCode(location common.AddressLocation) ([]byte, error) {
	return e.runtimeInterface.GetAccountContractCode(location)
}

func (e *interpreterEnvironment) CreateAccount(payer common.Address) (address common.Address, err error) {
	return e.runtimeInterface.CreateAccount(payer)
}

func (e *interpreterEnvironment) GenerateAccountID(address common.Address) (uint64, error) {
	return e.runtimeInterface.GenerateAccountID(address)
}

func (e *interpreterEnvironment) EmitEvent(
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
		e.runtimeInterface.EmitEvent,
	)
}

func (e *interpreterEnvironment) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	return e.runtimeInterface.AddAccountKey(address, key, algo, weight)
}

func (e *interpreterEnvironment) RevokeAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	return e.runtimeInterface.RevokeAccountKey(address, index)
}

func (e *interpreterEnvironment) UpdateAccountContractCode(location common.AddressLocation, code []byte) error {
	return e.runtimeInterface.UpdateAccountContractCode(location, code)
}

func (e *interpreterEnvironment) RemoveAccountContractCode(location common.AddressLocation) error {
	return e.runtimeInterface.RemoveAccountContractCode(location)
}

func (e *interpreterEnvironment) RecordContractRemoval(location common.AddressLocation) {
	e.storage.recordContractUpdate(location, nil)
}

func (e *interpreterEnvironment) RecordContractUpdate(
	location common.AddressLocation,
	contractValue *interpreter.CompositeValue,
) {
	e.storage.recordContractUpdate(location, contractValue)
}

func (e *interpreterEnvironment) ContractUpdateRecorded(location common.AddressLocation) bool {
	return e.storage.contractUpdateRecorded(location)
}

func (e *interpreterEnvironment) StartContractAddition(location common.AddressLocation) {
	if e.deployedContracts == nil {
		e.deployedContracts = map[Location]struct{}{}
	}

	e.deployedContracts[location] = struct{}{}
}

func (e *interpreterEnvironment) EndContractAddition(location common.AddressLocation) {
	delete(e.deployedContracts, location)
}

func (e *interpreterEnvironment) IsContractBeingAdded(location common.AddressLocation) bool {
	_, contains := e.deployedContracts[location]
	return contains
}

func (e *interpreterEnvironment) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	e.checkingEnvironment.temporarilyRecordCode(location, code)
}

func (e *interpreterEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.checkingEnvironment.ParseAndCheckProgram(code, location, getAndSetProgram)
}

func (e *interpreterEnvironment) ResolveLocation(
	identifiers []Identifier,
	location Location,
) (
	res []ResolvedLocation,
	err error,
) {
	return e.checkingEnvironment.resolveLocation(identifiers, location)
}

func (e *interpreterEnvironment) newInterpreter(
	location common.Location,
	program *interpreter.Program,
) (*interpreter.Interpreter, error) {

	sharedState := e.runtimeInterface.GetInterpreterSharedState()
	if sharedState != nil {
		// NOTE: no need to reset storage, as each top-level entry call
		// (e.g. transaction execution, contract invocation, etc.) creates a new storage.
		// Even though suboptimal, this ensures that no writes "leak" from one top-level entry call to another
		// (when interpreter shared state is reused).

		return interpreter.NewInterpreterWithSharedState(
			program,
			location,
			sharedState,
		)
	}

	inter, err := interpreter.NewInterpreter(
		program,
		location,
		e.InterpreterConfig,
	)
	if err != nil {
		return nil, err
	}

	e.runtimeInterface.SetInterpreterSharedState(inter.SharedState)

	return inter, nil
}

func (e *interpreterEnvironment) newOnStatementHandler() interpreter.OnStatementFunc {
	if e.config.CoverageReport == nil {
		return nil
	}

	return func(inter *interpreter.Interpreter, statement ast.Statement) {
		location := inter.Location
		if !e.coverageReport.IsLocationInspected(location) {
			program := inter.Program.Program
			e.coverageReport.InspectProgram(location, program)
		}

		line := statement.StartPosition().Line
		e.coverageReport.AddLineHit(location, line)
	}
}

func (e *interpreterEnvironment) NewAccountValue(
	context interpreter.AccountCreationContext,
	address interpreter.AddressValue,
) interpreter.Value {
	return stdlib.NewAccountValue(context, e, address)
}

func (e *interpreterEnvironment) ValidatePublicKey(publicKey *stdlib.PublicKey) error {
	return e.runtimeInterface.ValidatePublicKey(publicKey)
}

func (e *interpreterEnvironment) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm sema.SignatureAlgorithm,
	hashAlgorithm sema.HashAlgorithm,
) (bool, error) {
	return e.runtimeInterface.VerifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (e *interpreterEnvironment) BLSVerifyPOP(publicKeys *stdlib.PublicKey, signature []byte) (bool, error) {
	return e.runtimeInterface.BLSVerifyPOP(publicKeys, signature)
}

func (e *interpreterEnvironment) BLSAggregatePublicKeys(publicKeys []*stdlib.PublicKey) (*stdlib.PublicKey, error) {
	return e.runtimeInterface.BLSAggregatePublicKeys(publicKeys)
}

func (e *interpreterEnvironment) BLSAggregateSignatures(signatures [][]byte) ([]byte, error) {
	return e.runtimeInterface.BLSAggregateSignatures(signatures)
}

func (e *interpreterEnvironment) Hash(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error) {
	return e.runtimeInterface.Hash(data, tag, algorithm)
}

func (e *interpreterEnvironment) DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error) {
	return e.runtimeInterface.DecodeArgument(argument, argumentType)
}

func (e *interpreterEnvironment) newContractValueHandler() interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		// If the contract is the deployed contract, instantiate it using
		// the provided constructor and given arguments

		invocation := e.deployedContractConstructorInvocation

		if invocation != nil {
			if compositeType.Location == invocation.ContractType.Location &&
				compositeType.Identifier == invocation.ContractType.Identifier {

				constructor := constructorGenerator(invocation.Address)

				value, err := interpreter.InvokeFunctionValue(
					inter,
					constructor,
					invocation.ConstructorArguments,
					invocation.ArgumentTypes,
					invocation.ParameterTypes,
					invocation.ContractType,
					invocationRange,
				)
				if err != nil {
					panic(err)
				}

				return value.(*interpreter.CompositeValue)
			}
		}

		return loadContractValue(inter, compositeType, e.storage)
	}
}

func (e *interpreterEnvironment) newImportLocationHandler() interpreter.ImportLocationHandlerFunc {
	return func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

		const getAndSetProgram = true
		program, err := e.checkingEnvironment.GetProgram(
			location,
			getAndSetProgram,
			importResolutionResults{},
		)
		if err != nil {
			panic(err)
		}

		subInterpreter, err := inter.NewSubInterpreter(program, location)
		if err != nil {
			panic(err)
		}
		return interpreter.InterpreterImport{
			Interpreter: subInterpreter,
		}
	}
}

func (e *interpreterEnvironment) newCompositeTypeHandler() interpreter.CompositeTypeHandlerFunc {
	return func(location common.Location, typeID common.TypeID) *sema.CompositeType {

		switch location.(type) {
		case stdlib.FlowLocation:
			return stdlib.FlowEventTypes[typeID]

		case nil:
			qualifiedIdentifier := string(typeID)
			baseTypeActivation := e.checkingEnvironment.getBaseTypeActivation(location)
			ty := sema.TypeActivationNestedType(baseTypeActivation, qualifiedIdentifier)
			if ty == nil {
				return nil
			}

			if compositeType, ok := ty.(*sema.CompositeType); ok {
				return compositeType
			}
		}

		return nil
	}
}

func (e *interpreterEnvironment) newCompositeValueFunctionsHandler() interpreter.CompositeValueFunctionsHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		compositeValue *interpreter.CompositeValue,
	) *interpreter.FunctionOrderedMap {

		handler := e.compositeValueFunctionsHandlers[compositeValue.TypeID()]
		if handler == nil {
			return nil
		}

		return handler(inter, locationRange, compositeValue)
	}
}

func (e *interpreterEnvironment) newOnFunctionInvocationHandler() func(_ *interpreter.Interpreter) {
	return func(_ *interpreter.Interpreter) {
		e.stackDepthLimiter.OnFunctionInvocation()
	}
}

func (e *interpreterEnvironment) newOnInvokedFunctionReturnHandler() func(_ *interpreter.Interpreter) {
	return func(_ *interpreter.Interpreter) {
		e.stackDepthLimiter.OnInvokedFunctionReturn()
	}
}

func (e *interpreterEnvironment) InterpretContract(
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

	_, inter, err := e.Interpret(location, program, nil)
	if err != nil {
		return nil, err
	}

	variable := inter.Globals.Get(name)
	if variable == nil {
		return nil, errors.NewDefaultUserError(
			"cannot find contract: `%s`",
			name,
		)
	}

	contract = variable.GetValue(inter).(*interpreter.CompositeValue)

	return
}

func (e *interpreterEnvironment) Interpret(
	location common.Location,
	program *interpreter.Program,
	f InterpretFunc,
) (
	interpreter.Value,
	*interpreter.Interpreter,
	error,
) {
	inter, err := e.newInterpreter(location, program)
	if err != nil {
		return nil, nil, err
	}

	var result interpreter.Value

	reportMetric(
		func() {
			err = inter.Interpret()
			if err != nil || f == nil {
				return
			}
			result, err = f(inter)
		},
		e.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramInterpreted(location, duration)
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return result, inter, nil
}

func (e *interpreterEnvironment) newResourceOwnerChangedHandler() interpreter.OnResourceOwnerChangeFunc {
	if !e.config.ResourceOwnerChangeHandlerEnabled {
		return nil
	}

	return newResourceOwnerChangedHandler(&e.runtimeInterface)
}

func (e *interpreterEnvironment) commitStorage(context interpreter.ValueTransferContext) error {
	checkStorageHealth := e.config.AtreeValidationEnabled
	return CommitStorage(context, e.storage, checkStorageHealth)
}

// getBaseActivation returns the base activation for the given location.
// If a value was declared for the location (using DeclareValue),
// then the specific base activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *interpreterEnvironment) getBaseActivation(
	location common.Location,
) (
	baseActivation *interpreter.VariableActivation,
) {
	// Use the base activation for the location, if any
	// (previously implicitly created using DeclareValue)
	baseActivationsByLocation := e.baseActivationsByLocation
	baseActivation = baseActivationsByLocation[location]
	if baseActivation == nil {
		// If no base activation for the location exists
		// (no value was previously, specifically declared for the location using DeclareValue),
		// return the base activation that applies to all locations by default
		baseActivation = e.defaultBaseActivation
	}
	return
}
