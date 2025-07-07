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

	"github.com/onflow/cadence/activations"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type interpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

type Environment interface {
	ArgumentDecoder

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
	commitStorage(context interpreter.ValueTransferContext) error
	newAccountValue(context interpreter.AccountCreationContext, address interpreter.AddressValue) interpreter.Value
}

// interpreterEnvironmentReconfigured is the portion of InterpreterEnvironment
// that gets reconfigured by InterpreterEnvironment.Configure
type interpreterEnvironmentReconfigured struct {
	Interface
	storage        *Storage
	coverageReport *CoverageReport
}

type InterpreterEnvironment struct {
	interpreterEnvironmentReconfigured

	CheckingEnvironment *CheckingEnvironment

	// defaultBaseActivation is the base activation that applies to all locations by default
	defaultBaseActivation *interpreter.VariableActivation
	// The base activations for individual locations.
	// location == nil is the base activation that applies to all locations,
	// unless there is a base activation for the given location.
	//
	// Base activations are lazily / implicitly created
	// by DeclareValue / interpreterBaseActivationFor
	baseActivationsByLocation map[common.Location]*interpreter.VariableActivation

	allDeclaredTypes map[common.TypeID]sema.Type

	InterpreterConfig *interpreter.Config

	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation
	stackDepthLimiter                     *stackDepthLimiter
	compositeValueFunctionsHandlers       stdlib.CompositeValueFunctionsHandlers
	config                                Config
	*stdlib.SimpleContractAdditionTracker
}

var _ Environment = &InterpreterEnvironment{}
var _ stdlib.Logger = &InterpreterEnvironment{}
var _ stdlib.RandomGenerator = &InterpreterEnvironment{}
var _ stdlib.BlockAtHeightProvider = &InterpreterEnvironment{}
var _ stdlib.CurrentBlockProvider = &InterpreterEnvironment{}
var _ stdlib.AccountHandler = &InterpreterEnvironment{}
var _ stdlib.AccountCreator = &InterpreterEnvironment{}
var _ stdlib.EventEmitter = &InterpreterEnvironment{}
var _ stdlib.PublicKeyValidator = &InterpreterEnvironment{}
var _ stdlib.PublicKeySignatureVerifier = &InterpreterEnvironment{}
var _ stdlib.BLSPoPVerifier = &InterpreterEnvironment{}
var _ stdlib.BLSPublicKeyAggregator = &InterpreterEnvironment{}
var _ stdlib.BLSSignatureAggregator = &InterpreterEnvironment{}
var _ stdlib.Hasher = &InterpreterEnvironment{}
var _ ArgumentDecoder = &InterpreterEnvironment{}
var _ common.MemoryGauge = &InterpreterEnvironment{}
var _ common.ComputationGauge = &InterpreterEnvironment{}

func NewInterpreterEnvironment(config Config) *InterpreterEnvironment {
	defaultBaseActivation := activations.NewActivation(nil, interpreter.BaseActivation)

	env := &InterpreterEnvironment{
		config:                        config,
		CheckingEnvironment:           newCheckingEnvironment(),
		defaultBaseActivation:         defaultBaseActivation,
		stackDepthLimiter:             newStackDepthLimiter(config.StackDepthLimit),
		SimpleContractAdditionTracker: stdlib.NewSimpleContractAdditionTracker(),
	}
	env.InterpreterConfig = env.NewInterpreterConfig()

	env.compositeValueFunctionsHandlers = stdlib.DefaultStandardLibraryCompositeValueFunctionHandlers(env)
	return env
}

func (e *InterpreterEnvironment) NewInterpreterConfig() *interpreter.Config {
	return &interpreter.Config{
		MemoryGauge:                    e,
		ComputationGauge:               e,
		BaseActivationHandler:          e.getBaseActivation,
		OnEventEmitted:                 newOnEventEmittedHandler(&e.Interface),
		InjectedCompositeFieldsHandler: newInjectedCompositeFieldsHandler(e),
		UUIDHandler:                    newUUIDHandler(&e.Interface),
		ContractValueHandler:           e.newContractValueHandler(),
		ImportLocationHandler:          e.newImportLocationHandler(),
		AccountHandler:                 e.newAccountValue,
		OnRecordTrace:                  newOnRecordTraceHandler(&e.Interface),
		OnResourceOwnerChange:          e.newResourceOwnerChangedHandler(),
		CompositeTypeHandler:           e.newCompositeTypeHandler(),
		CompositeValueFunctionsHandler: e.newCompositeValueFunctionsHandler(),
		TracingEnabled:                 e.config.TracingEnabled,
		AtreeValueValidationEnabled:    e.config.AtreeValidationEnabled,
		// NOTE: ignore e.config.AtreeValidationEnabled here,
		// and disable storage validation after each value modification.
		// Instead, storage is validated after commits (if validation is enabled),
		// see InterpreterEnvironment.commitStorage
		AtreeStorageValidationEnabled:             false,
		Debugger:                                  e.config.Debugger,
		OnStatement:                               e.newOnStatementHandler(),
		OnFunctionInvocation:                      e.newOnFunctionInvocationHandler(),
		OnInvokedFunctionReturn:                   e.newOnInvokedFunctionReturnHandler(),
		CapabilityBorrowHandler:                   newCapabilityBorrowHandler(e),
		CapabilityCheckHandler:                    newCapabilityCheckHandler(e),
		ValidateAccountCapabilitiesGetHandler:     newValidateAccountCapabilitiesGetHandler(&e.Interface),
		ValidateAccountCapabilitiesPublishHandler: newValidateAccountCapabilitiesPublishHandler(&e.Interface),
	}
}

func NewBaseInterpreterEnvironment(config Config) *InterpreterEnvironment {
	env := NewInterpreterEnvironment(config)
	for _, typeDeclaration := range stdlib.DefaultStandardLibraryTypes {
		env.DeclareType(typeDeclaration, nil)
	}
	for _, valueDeclaration := range stdlib.InterpreterDefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptInterpreterEnvironment(config Config) *InterpreterEnvironment {
	env := NewInterpreterEnvironment(config)
	for _, typeDeclaration := range stdlib.DefaultStandardLibraryTypes {
		env.DeclareType(typeDeclaration, nil)
	}
	for _, valueDeclaration := range stdlib.InterpreterDefaultScriptStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func (e *InterpreterEnvironment) Configure(
	runtimeInterface Interface,
	codesAndPrograms CodesAndPrograms,
	storage *Storage,
	coverageReport *CoverageReport,
) {
	e.Interface = runtimeInterface
	e.storage = storage
	e.InterpreterConfig.Storage = storage
	e.coverageReport = coverageReport
	e.stackDepthLimiter.depth = 0

	e.CheckingEnvironment.configure(
		runtimeInterface,
		codesAndPrograms,
	)

	configureVersionedFeatures(runtimeInterface)
}

func (e *InterpreterEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.CheckingEnvironment.declareValue(valueDeclaration, location)

	activation := e.interpreterBaseActivationFor(location)
	interpreter.Declare(activation, valueDeclaration)
}

func (e *InterpreterEnvironment) DeclareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.CheckingEnvironment.declareType(typeDeclaration, location)
	if e.allDeclaredTypes == nil {
		e.allDeclaredTypes = map[common.TypeID]sema.Type{}
	}
	e.allDeclaredTypes[typeDeclaration.Type.ID()] = typeDeclaration.Type
}

func (e *InterpreterEnvironment) interpreterBaseActivationFor(
	location common.Location,
) *interpreter.VariableActivation {
	defaultBaseActivation := e.defaultBaseActivation
	if location == nil {
		return defaultBaseActivation
	}

	baseActivation := e.baseActivationsByLocation[location]
	if baseActivation == nil {
		baseActivation = activations.NewActivation(nil, defaultBaseActivation)
		if e.baseActivationsByLocation == nil {
			e.baseActivationsByLocation = map[common.Location]*interpreter.VariableActivation{}
		}
		e.baseActivationsByLocation[location] = baseActivation
	}
	return baseActivation
}

func (e *InterpreterEnvironment) SetCompositeValueFunctionsHandler(
	typeID common.TypeID,
	handler stdlib.CompositeValueFunctionsHandler,
) {
	e.compositeValueFunctionsHandlers[typeID] = handler
}

func (e *InterpreterEnvironment) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	const commitContractUpdates = false
	return e.storage.Commit(context, commitContractUpdates)
}

func (e *InterpreterEnvironment) EmitEvent(
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

func (e *InterpreterEnvironment) RecordContractRemoval(location common.AddressLocation) {
	e.storage.recordContractUpdate(location, nil)
}

func (e *InterpreterEnvironment) RecordContractUpdate(
	location common.AddressLocation,
	contractValue *interpreter.CompositeValue,
) {
	e.storage.recordContractUpdate(location, contractValue)
}

func (e *InterpreterEnvironment) ContractUpdateRecorded(location common.AddressLocation) bool {
	return e.storage.contractUpdateRecorded(location)
}

func (e *InterpreterEnvironment) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	e.CheckingEnvironment.temporarilyRecordCode(location, code)
}

func (e *InterpreterEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.CheckingEnvironment.ParseAndCheckProgram(code, location, getAndSetProgram)
}

func (e *InterpreterEnvironment) ResolveLocation(
	identifiers []Identifier,
	location Location,
) (
	res []ResolvedLocation,
	err error,
) {
	return e.CheckingEnvironment.resolveLocation(identifiers, location)
}

func (e *InterpreterEnvironment) newInterpreter(
	location common.Location,
	program *interpreter.Program,
) (*interpreter.Interpreter, error) {

	sharedState := e.Interface.GetInterpreterSharedState()
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

	e.Interface.SetInterpreterSharedState(inter.SharedState)

	return inter, nil
}

func (e *InterpreterEnvironment) newOnStatementHandler() interpreter.OnStatementFunc {
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

func (e *InterpreterEnvironment) newAccountValue(
	context interpreter.AccountCreationContext,
	address interpreter.AddressValue,
) interpreter.Value {
	return stdlib.NewAccountValue(context, e, address)
}

func (e *InterpreterEnvironment) newContractValueHandler() interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		// If the contract is the deployed contract, instantiate it using
		// the provided constructor and given arguments

		invocation := e.deployedContractConstructorInvocation

		contractLocation := compositeType.Location

		if invocation != nil {
			if contractLocation == invocation.ContractType.Location &&
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

		if addressLocation, ok := contractLocation.(common.AddressLocation); ok {
			return loadContractValue(inter, addressLocation, e.storage)
		}

		panic(errors.NewDefaultUserError("failed to load contract: %s", contractLocation))
	}
}

func (e *InterpreterEnvironment) newImportLocationHandler() interpreter.ImportLocationHandlerFunc {
	return func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

		const getAndSetProgram = true
		program, err := e.CheckingEnvironment.GetProgram(
			location,
			getAndSetProgram,
			importResolutionResults{},
		)
		if err != nil {
			panic(err)
		}

		var interpreterProgram *interpreter.Program
		if program != nil {
			interpreterProgram = program.interpreterProgram
		}

		subInterpreter, err := inter.NewSubInterpreter(
			interpreterProgram,
			location,
		)
		if err != nil {
			panic(err)
		}
		return interpreter.InterpreterImport{
			Interpreter: subInterpreter,
		}
	}
}

func (e *InterpreterEnvironment) newCompositeTypeHandler() interpreter.CompositeTypeHandlerFunc {
	return func(location common.Location, typeID common.TypeID) *sema.CompositeType {

		ty := e.allDeclaredTypes[typeID]
		if compositeType, ok := ty.(*sema.CompositeType); ok {
			return compositeType
		}

		if _, ok := location.(stdlib.FlowLocation); ok {
			return stdlib.FlowEventTypes[typeID]
		}

		return nil
	}
}

func (e *InterpreterEnvironment) newCompositeValueFunctionsHandler() interpreter.CompositeValueFunctionsHandlerFunc {
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

func (e *InterpreterEnvironment) newOnFunctionInvocationHandler() func(_ *interpreter.Interpreter) {
	return func(_ *interpreter.Interpreter) {
		e.stackDepthLimiter.OnFunctionInvocation()
	}
}

func (e *InterpreterEnvironment) newOnInvokedFunctionReturnHandler() func(_ *interpreter.Interpreter) {
	return func(_ *interpreter.Interpreter) {
		e.stackDepthLimiter.OnInvokedFunctionReturn()
	}
}

func (e *InterpreterEnvironment) LoadContractValue(
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

func (e *InterpreterEnvironment) Interpret(
	location common.Location,
	program *interpreter.Program,
	f interpretFunc,
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
		e.Interface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramInterpreted(location, duration)
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return result, inter, nil
}

func (e *InterpreterEnvironment) newResourceOwnerChangedHandler() interpreter.OnResourceOwnerChangeFunc {
	if !e.config.ResourceOwnerChangeHandlerEnabled {
		return nil
	}

	return newResourceOwnerChangedHandler(&e.Interface)
}

func (e *InterpreterEnvironment) commitStorage(context interpreter.ValueTransferContext) error {
	checkStorageHealth := e.config.AtreeValidationEnabled
	return CommitStorage(context, e.storage, checkStorageHealth)
}

// getBaseActivation returns the base activation for the given location.
// If a value was declared for the location (using DeclareValue),
// then the specific base activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *InterpreterEnvironment) getBaseActivation(
	location common.Location,
) (
	baseActivation *interpreter.VariableActivation,
) {
	// Use the base activation for the location, if any
	// (previously implicitly created using DeclareValue)
	baseActivation = e.baseActivationsByLocation[location]
	if baseActivation == nil {
		// If no base activation for the location exists
		// (no value was previously, specifically declared for the location using DeclareValue),
		// return the base activation that applies to all locations by default
		baseActivation = e.defaultBaseActivation
	}
	return
}

func (e *InterpreterEnvironment) ProgramLog(message string, _ interpreter.LocationRange) error {
	return e.Interface.ProgramLog(message)
}
