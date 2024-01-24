/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/activations"

	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

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
	Interpret(
		location common.Location,
		program *interpreter.Program,
		f InterpretFunc,
	) (
		interpreter.Value,
		*interpreter.Interpreter,
		error,
	)
	CommitStorage(inter *interpreter.Interpreter) error
	NewAuthAccountValue(address interpreter.AddressValue) interpreter.Value
	NewPublicAccountValue(address interpreter.AddressValue) interpreter.Value
}

// interpreterEnvironmentReconfigured is the portion of interpreterEnvironment
// that gets reconfigured by interpreterEnvironment.Configure
type interpreterEnvironmentReconfigured struct {
	runtimeInterface Interface
	storage          *Storage
	coverageReport   *CoverageReport
	codesAndPrograms CodesAndPrograms
}

type interpreterEnvironment struct {
	interpreterEnvironmentReconfigured

	// defaultBaseTypeActivation is the base type activation that applies to all locations by default.
	defaultBaseTypeActivation *sema.VariableActivation
	// The base type activations for individual locations.
	// location == nil is the base type activation that applies to all locations,
	// unless there is a base type activation for the given location.
	//
	// Base type activations are lazily / implicitly created
	// by DeclareType / semaBaseActivationFor
	baseTypeActivationsByLocation map[common.Location]*sema.VariableActivation

	// defaultBaseValueActivation is the base value activation that applies to all locations by default.
	defaultBaseValueActivation *sema.VariableActivation
	// The base value activations for individual locations.
	// location == nil is the base value activation that applies to all locations,
	// unless there is a base value activation for the given location.
	//
	// Base value activations are lazily / implicitly created
	// by DeclareValue / semaBaseActivationFor
	baseValueActivationsByLocation map[common.Location]*sema.VariableActivation

	// defaultBaseActivation is the base activation that applies to all locations by default
	defaultBaseActivation *interpreter.VariableActivation
	// The base activations for individual locations.
	// location == nil is the base activation that applies to all locations,
	// unless there is a base activation for the given location.
	//
	// Base activations are lazily / implicitly created
	// by DeclareValue / interpreterBaseActivationFor
	baseActivationsByLocation map[common.Location]*interpreter.VariableActivation

	InterpreterConfig                     *interpreter.Config
	CheckerConfig                         *sema.Config
	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation
	stackDepthLimiter                     *stackDepthLimiter
	checkedImports                        importResolutionResults
	config                                Config
	deployedContracts                     map[Location]struct{}
}

var _ Environment = &interpreterEnvironment{}
var _ stdlib.Logger = &interpreterEnvironment{}
var _ stdlib.RandomGenerator = &interpreterEnvironment{}
var _ stdlib.BlockAtHeightProvider = &interpreterEnvironment{}
var _ stdlib.CurrentBlockProvider = &interpreterEnvironment{}
var _ stdlib.PublicAccountHandler = &interpreterEnvironment{}
var _ stdlib.AccountCreator = &interpreterEnvironment{}
var _ stdlib.EventEmitter = &interpreterEnvironment{}
var _ stdlib.AuthAccountHandler = &interpreterEnvironment{}
var _ stdlib.PublicKeyValidator = &interpreterEnvironment{}
var _ stdlib.PublicKeySignatureVerifier = &interpreterEnvironment{}
var _ stdlib.BLSPoPVerifier = &interpreterEnvironment{}
var _ stdlib.BLSPublicKeyAggregator = &interpreterEnvironment{}
var _ stdlib.BLSSignatureAggregator = &interpreterEnvironment{}
var _ stdlib.Hasher = &interpreterEnvironment{}
var _ ArgumentDecoder = &interpreterEnvironment{}
var _ common.MemoryGauge = &interpreterEnvironment{}

func newInterpreterEnvironment(config Config) *interpreterEnvironment {
	defaultBaseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	defaultBaseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
	defaultBaseActivation := activations.NewActivation(nil, interpreter.BaseActivation)

	env := &interpreterEnvironment{
		config:                     config,
		defaultBaseValueActivation: defaultBaseValueActivation,
		defaultBaseTypeActivation:  defaultBaseTypeActivation,
		defaultBaseActivation:      defaultBaseActivation,
		stackDepthLimiter:          newStackDepthLimiter(config.StackDepthLimit),
	}
	env.InterpreterConfig = env.newInterpreterConfig()
	env.CheckerConfig = env.newCheckerConfig()
	return env
}

func (e *interpreterEnvironment) newInterpreterConfig() *interpreter.Config {
	return &interpreter.Config{
		InvalidatedResourceValidationEnabled: true,
		MemoryGauge:                          e,
		BaseActivationHandler:                e.getBaseActivation,
		OnEventEmitted:                       e.newOnEventEmittedHandler(),
		OnAccountLinked:                      e.newOnAccountLinkedHandler(),
		InjectedCompositeFieldsHandler:       e.newInjectedCompositeFieldsHandler(),
		UUIDHandler:                          e.newUUIDHandler(),
		ContractValueHandler:                 e.newContractValueHandler(),
		ImportLocationHandler:                e.newImportLocationHandler(),
		PublicAccountHandler:                 e.newPublicAccountHandler(),
		AuthAccountHandler:                   e.newAuthAccountHandler(),
		OnRecordTrace:                        e.newOnRecordTraceHandler(),
		OnResourceOwnerChange:                e.newResourceOwnerChangedHandler(),
		CompositeTypeHandler:                 e.newCompositeTypeHandler(),
		TracingEnabled:                       e.config.TracingEnabled,
		AtreeValueValidationEnabled:          e.config.AtreeValidationEnabled,
		// NOTE: ignore e.config.AtreeValidationEnabled here,
		// and disable storage validation after each value modification.
		// Instead, storage is validated after commits (if validation is enabled),
		// see interpreterEnvironment.CommitStorage
		AtreeStorageValidationEnabled: false,
		Debugger:                      e.config.Debugger,
		OnStatement:                   e.newOnStatementHandler(),
		OnMeterComputation:            e.newOnMeterComputation(),
		OnFunctionInvocation:          e.newOnFunctionInvocationHandler(),
		OnInvokedFunctionReturn:       e.newOnInvokedFunctionReturnHandler(),
		IDCapabilityBorrowHandler:     stdlib.BorrowCapabilityController,
		IDCapabilityCheckHandler:      stdlib.CheckCapabilityController,
	}
}

func (e *interpreterEnvironment) newCheckerConfig() *sema.Config {
	return &sema.Config{
		AccessCheckMode:                  sema.AccessCheckModeStrict,
		BaseValueActivationHandler:       e.getBaseValueActivation,
		BaseTypeActivationHandler:        e.getBaseTypeActivation,
		ValidTopLevelDeclarationsHandler: validTopLevelDeclarations,
		LocationHandler:                  e.newLocationHandler(),
		ImportHandler:                    e.resolveImport,
		CheckHandler:                     e.newCheckHandler(),
		AccountLinkingEnabled:            e.config.AccountLinkingEnabled,
		AttachmentsEnabled:               e.config.AttachmentsEnabled,
		CapabilityControllersEnabled:     e.config.CapabilityControllersEnabled,
	}
}

func NewBaseInterpreterEnvironment(config Config) *interpreterEnvironment {
	env := newInterpreterEnvironment(config)
	for _, valueDeclaration := range stdlib.DefaultStandardLibraryValues(env) {
		env.DeclareValue(valueDeclaration, nil)
	}
	return env
}

func NewScriptInterpreterEnvironment(config Config) Environment {
	env := newInterpreterEnvironment(config)
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
	e.codesAndPrograms = codesAndPrograms
	e.storage = storage
	e.InterpreterConfig.Storage = storage
	e.coverageReport = coverageReport
	e.stackDepthLimiter.depth = 0
}

func (e *interpreterEnvironment) DeclareValue(valueDeclaration stdlib.StandardLibraryValue, location common.Location) {
	e.semaBaseActivationFor(
		location,
		&e.baseValueActivationsByLocation,
		e.defaultBaseValueActivation,
	).DeclareValue(valueDeclaration)

	activation := e.interpreterBaseActivationFor(location)
	interpreter.Declare(activation, valueDeclaration)
}

func (e *interpreterEnvironment) DeclareType(typeDeclaration stdlib.StandardLibraryType, location common.Location) {
	e.semaBaseActivationFor(
		location,
		&e.baseTypeActivationsByLocation,
		e.defaultBaseTypeActivation,
	).DeclareType(typeDeclaration)
}

func (e *interpreterEnvironment) semaBaseActivationFor(
	location common.Location,
	baseActivationsByLocation *map[Location]*sema.VariableActivation,
	defaultBaseActivation *sema.VariableActivation,
) (baseActivation *sema.VariableActivation) {
	if location == nil {
		return defaultBaseActivation
	}

	if *baseActivationsByLocation == nil {
		*baseActivationsByLocation = map[Location]*sema.VariableActivation{}
	} else {
		baseActivation = (*baseActivationsByLocation)[location]
	}
	if baseActivation == nil {
		baseActivation = sema.NewVariableActivation(defaultBaseActivation)
		(*baseActivationsByLocation)[location] = baseActivation
	}
	return baseActivation
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
		baseActivation = activations.NewActivation[*interpreter.Variable](nil, defaultBaseActivation)
		if e.baseActivationsByLocation == nil {
			e.baseActivationsByLocation = map[common.Location]*interpreter.VariableActivation{}
		}
		e.baseActivationsByLocation[location] = baseActivation
	}
	return baseActivation
}

func (e *interpreterEnvironment) NewAuthAccountValue(address interpreter.AddressValue) interpreter.Value {
	return stdlib.NewAuthAccountValue(e, e, address)
}

func (e *interpreterEnvironment) NewPublicAccountValue(address interpreter.AddressValue) interpreter.Value {
	return stdlib.NewPublicAccountValue(e, e, address)
}

func (e *interpreterEnvironment) MeterMemory(usage common.MemoryUsage) error {
	return e.runtimeInterface.MeterMemory(usage)
}

func (e *interpreterEnvironment) ProgramLog(message string) error {
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

func (e *interpreterEnvironment) CommitStorageTemporarily(inter *interpreter.Interpreter) error {
	const commitContractUpdates = false
	return e.storage.Commit(inter, commitContractUpdates)
}

func (e *interpreterEnvironment) GetStorageUsed(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetStorageUsed(address)
}

func (e *interpreterEnvironment) GetStorageCapacity(address common.Address) (uint64, error) {
	return e.runtimeInterface.GetStorageCapacity(address)
}

func (e *interpreterEnvironment) GetAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	return e.runtimeInterface.GetAccountKey(address, index)
}

func (e *interpreterEnvironment) AccountKeysCount(address common.Address) (uint64, error) {
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
	inter *interpreter.Interpreter,
	eventType *sema.CompositeType,
	values []interpreter.Value,
	locationRange interpreter.LocationRange,
) {
	emitEventFields(
		inter,
		locationRange,
		eventType,
		newExportableValues(inter, values),
		e.runtimeInterface.EmitEvent,
	)
}

func (e *interpreterEnvironment) AddEncodedAccountKey(address common.Address, key []byte) error {
	return e.runtimeInterface.AddEncodedAccountKey(address, key)
}

func (e *interpreterEnvironment) RevokeEncodedAccountKey(address common.Address, index int) ([]byte, error) {
	return e.runtimeInterface.RevokeEncodedAccountKey(address, index)
}

func (e *interpreterEnvironment) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	return e.runtimeInterface.AddAccountKey(address, key, algo, weight)
}

func (e *interpreterEnvironment) RevokeAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
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
	e.codesAndPrograms.setCode(location, code)
}

func (e *interpreterEnvironment) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.getProgram(
		location,
		func() ([]byte, error) {
			return code, nil
		},
		getAndSetProgram,
		importResolutionResults{},
	)
}

// parseAndCheckProgram parses and checks the given program.
// If storeProgram is true, it calls Interface.SetProgram.
func (e *interpreterEnvironment) parseAndCheckProgram(
	code []byte,
	location common.Location,
	checkedImports importResolutionResults,
) (
	program *ast.Program,
	elaboration *sema.Elaboration,
	err error,
) {
	wrapParsingCheckingError := func(err error) error {
		return &ParsingCheckingError{
			Err:      err,
			Location: location,
		}
	}

	// Parse

	reportMetric(
		func() {
			program, err = parser.ParseProgram(e, code, parser.Config{})
		},
		e.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(location, duration)
		},
	)
	if err != nil {
		return nil, nil, wrapParsingCheckingError(err)
	}

	// Check

	elaboration, err = e.check(location, program, checkedImports)
	if err != nil {
		return program, nil, wrapParsingCheckingError(err)
	}

	return program, elaboration, nil
}

func (e *interpreterEnvironment) check(
	location common.Location,
	program *ast.Program,
	checkedImports importResolutionResults,
) (
	elaboration *sema.Elaboration,
	err error,
) {
	e.checkedImports = checkedImports

	checker, err := sema.NewChecker(
		program,
		location,
		e,
		e.CheckerConfig,
	)
	if err != nil {
		return nil, err
	}

	elaboration = checker.Elaboration

	err = checker.Check()
	if err != nil {
		return nil, err
	}

	return elaboration, nil
}

func (e *interpreterEnvironment) newLocationHandler() sema.LocationHandlerFunc {
	return func(identifiers []Identifier, location Location) (res []ResolvedLocation, err error) {
		errors.WrapPanic(func() {
			res, err = e.runtimeInterface.ResolveLocation(identifiers, location)
		})

		if err != nil {
			err = interpreter.WrappedExternalError(err)
		}
		return
	}
}

func (e *interpreterEnvironment) newCheckHandler() sema.CheckHandlerFunc {
	return func(checker *sema.Checker, check func()) {
		reportMetric(
			check,
			e.runtimeInterface,
			func(metrics Metrics, duration time.Duration) {
				metrics.ProgramChecked(checker.Location, duration)
			},
		)
	}
}

func (e *interpreterEnvironment) resolveImport(
	_ *sema.Checker,
	importedLocation common.Location,
	importRange ast.Range,
) (sema.Import, error) {

	var elaboration *sema.Elaboration
	switch importedLocation {
	case stdlib.CryptoCheckerLocation:
		cryptoChecker := stdlib.CryptoChecker()
		elaboration = cryptoChecker.Elaboration

	default:

		// Check for cyclic imports
		if e.checkedImports[importedLocation] {
			return nil, &sema.CyclicImportsError{
				Location: importedLocation,
				Range:    importRange,
			}
		} else {
			e.checkedImports[importedLocation] = true
			defer delete(e.checkedImports, importedLocation)
		}

		const getAndSetProgram = true
		program, err := e.GetProgram(
			importedLocation,
			getAndSetProgram,
			e.checkedImports,
		)
		if err != nil {
			return nil, err
		}

		elaboration = program.Elaboration
	}

	return sema.ElaborationImport{
		Elaboration: elaboration,
	}, nil
}

func (e *interpreterEnvironment) GetProgram(
	location Location,
	storeProgram bool,
	checkedImports importResolutionResults,
) (
	*interpreter.Program,
	error,
) {
	return e.getProgram(
		location,
		func() ([]byte, error) {
			return e.getCode(location)
		},
		storeProgram,
		checkedImports,
	)
}

// getProgram returns the existing program at the given location, if available.
// If it is not available, it loads the code, and then parses and checks it.
func (e *interpreterEnvironment) getProgram(
	location Location,
	getCode func() ([]byte, error),
	getAndSetProgram bool,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {
	load := func() (*interpreter.Program, error) {
		code, err := getCode()
		if err != nil {
			return nil, err
		}

		e.codesAndPrograms.setCode(location, code)

		parsedProgram, elaboration, err := e.parseAndCheckProgram(
			code,
			location,
			checkedImports,
		)
		if parsedProgram != nil {
			e.codesAndPrograms.setProgram(location, parsedProgram)
		}
		if err != nil {
			return nil, err
		}

		return &interpreter.Program{
			Program:     parsedProgram,
			Elaboration: elaboration,
		}, nil
	}

	if !getAndSetProgram {
		return load()
	}

	errors.WrapPanic(func() {
		program, err = e.runtimeInterface.GetOrLoadProgram(location, func() (program *interpreter.Program, err error) {
			// Loading is done by Cadence.
			// If it panics with a user error, e.g. when parsing fails due to a memory metering error,
			// then do not treat it as an external error (the load callback is called by the embedder)
			panicErr := userPanicToError(func() {
				program, err = load()
			})
			if panicErr != nil {
				return nil, panicErr
			}

			if err != nil {
				err = interpreter.WrappedExternalError(err)
			}

			return
		})
	})

	return
}

func (e *interpreterEnvironment) getCode(location common.Location) (code []byte, err error) {
	if addressLocation, ok := location.(common.AddressLocation); ok {
		errors.WrapPanic(func() {
			code, err = e.runtimeInterface.GetAccountContractCode(addressLocation)
		})
	} else {
		errors.WrapPanic(func() {
			code, err = e.runtimeInterface.GetCode(location)
		})
	}

	if err != nil {
		err = interpreter.WrappedExternalError(err)
	}

	return
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

func (e *interpreterEnvironment) newOnRecordTraceHandler() interpreter.OnRecordTraceFunc {
	return func(
		interpreter *interpreter.Interpreter,
		functionName string,
		duration time.Duration,
		attrs []attribute.KeyValue,
	) {
		errors.WrapPanic(func() {
			e.runtimeInterface.RecordTrace(functionName, interpreter.Location, duration, attrs)
		})
	}
}

func (e *interpreterEnvironment) newPublicAccountHandler() interpreter.PublicAccountHandlerFunc {
	return func(address interpreter.AddressValue) interpreter.Value {
		return stdlib.NewPublicAccountValue(e, e, address)
	}
}

func (e *interpreterEnvironment) newAuthAccountHandler() interpreter.AuthAccountHandlerFunc {
	return func(address interpreter.AddressValue) interpreter.Value {
		return stdlib.NewAuthAccountValue(e, e, address)
	}
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

				value, err := inter.InvokeFunctionValue(
					constructor,
					invocation.ConstructorArguments,
					invocation.ArgumentTypes,
					invocation.ParameterTypes,
					invocationRange,
				)
				if err != nil {
					panic(err)
				}

				return value.(*interpreter.CompositeValue)
			}
		}

		return e.loadContract(
			inter,
			compositeType,
			constructorGenerator,
			invocationRange,
		)
	}
}

func (e *interpreterEnvironment) newUUIDHandler() interpreter.UUIDHandlerFunc {
	return func() (uuid uint64, err error) {
		errors.WrapPanic(func() {
			uuid, err = e.runtimeInterface.GenerateUUID()
		})

		if err != nil {
			err = interpreter.WrappedExternalError(err)
		}
		return
	}
}

func (e *interpreterEnvironment) newOnEventEmittedHandler() interpreter.OnEventEmittedFunc {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		eventValue *interpreter.CompositeValue,
		eventType *sema.CompositeType,
	) error {
		emitEventValue(
			inter,
			locationRange,
			eventType,
			eventValue,
			e.runtimeInterface.EmitEvent,
		)

		return nil
	}
}

func (e *interpreterEnvironment) newOnAccountLinkedHandler() interpreter.OnAccountLinkedFunc {
	return func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		addressValue interpreter.AddressValue,
		pathValue interpreter.PathValue,
	) error {
		e.EmitEvent(
			inter,
			stdlib.AccountLinkedEventType,
			[]interpreter.Value{
				addressValue,
				pathValue,
			},
			locationRange,
		)
		return nil
	}
}

func (e *interpreterEnvironment) newInjectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		location Location,
		_ string,
		compositeKind common.CompositeKind,
	) map[string]interpreter.Value {

		switch location {
		case stdlib.CryptoCheckerLocation:
			return nil

		default:
			switch compositeKind {
			case common.CompositeKindContract:
				var address Address

				switch location := location.(type) {
				case common.AddressLocation:
					address = location.Address
				default:
					return nil
				}

				addressValue := interpreter.NewAddressValue(
					inter,
					address,
				)

				return map[string]interpreter.Value{
					sema.ContractAccountFieldName: stdlib.NewAuthAccountValue(
						inter,
						e,
						addressValue,
					),
				}
			}
		}

		return nil
	}
}

func (e *interpreterEnvironment) newImportLocationHandler() interpreter.ImportLocationHandlerFunc {
	return func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {

		switch location {
		case stdlib.CryptoCheckerLocation:
			cryptoChecker := stdlib.CryptoChecker()
			program := interpreter.ProgramFromChecker(cryptoChecker)
			subInterpreter, err := inter.NewSubInterpreter(program, location)
			if err != nil {
				panic(err)
			}
			return interpreter.InterpreterImport{
				Interpreter: subInterpreter,
			}

		default:
			const getAndSetProgram = true
			program, err := e.GetProgram(
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
}

func (e *interpreterEnvironment) newCompositeTypeHandler() interpreter.CompositeTypeHandlerFunc {
	return func(location common.Location, typeID common.TypeID) *sema.CompositeType {

		switch location.(type) {
		case stdlib.FlowLocation:
			return stdlib.FlowEventTypes[typeID]

		case nil:
			qualifiedIdentifier := string(typeID)
			baseTypeActivation := e.getBaseTypeActivation(location)
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

func (e *interpreterEnvironment) loadContract(
	inter *interpreter.Interpreter,
	compositeType *sema.CompositeType,
	constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
	invocationRange ast.Range,
) *interpreter.CompositeValue {

	switch compositeType.Location {
	case stdlib.CryptoCheckerLocation:
		contract, err := stdlib.NewCryptoContract(
			inter,
			constructorGenerator(common.ZeroAddress),
			invocationRange,
		)
		if err != nil {
			panic(err)
		}
		return contract

	default:

		var storedValue interpreter.Value

		switch location := compositeType.Location.(type) {

		case common.AddressLocation:
			storageMap := e.storage.GetStorageMap(
				location.Address,
				StorageDomainContract,
				false,
			)
			if storageMap != nil {
				storedValue = storageMap.ReadValue(inter, interpreter.StringStorageMapKey(location.Name))
			}
		}

		if storedValue == nil {
			panic(errors.NewDefaultUserError("failed to load contract: %s", compositeType.Location))
		}

		return storedValue.(*interpreter.CompositeValue)
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

func (e *interpreterEnvironment) newOnMeterComputation() interpreter.OnMeterComputationFunc {
	return func(compKind common.ComputationKind, intensity uint) {
		var err error
		errors.WrapPanic(func() {
			err = e.runtimeInterface.MeterComputation(compKind, intensity)
		})
		if err != nil {
			panic(interpreter.WrappedExternalError(err))
		}
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

	contract = variable.GetValue().(*interpreter.CompositeValue)

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

	return func(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	) {
		errors.WrapPanic(func() {
			e.runtimeInterface.ResourceOwnerChanged(
				interpreter,
				resource,
				oldOwner,
				newOwner,
			)
		})
	}
}

func (e *interpreterEnvironment) CommitStorage(inter *interpreter.Interpreter) error {
	const commitContractUpdates = true
	err := e.storage.Commit(inter, commitContractUpdates)
	if err != nil {
		return err
	}

	if e.config.AtreeValidationEnabled {
		err = e.storage.CheckHealth()
		if err != nil {
			return err
		}
	}

	return nil
}

// getBaseValueActivation returns the base activation for the given location.
// If a value was declared for the location (using DeclareValue),
// then the specific base value activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *interpreterEnvironment) getBaseValueActivation(
	location common.Location,
) (
	baseValueActivation *sema.VariableActivation,
) {
	baseValueActivationsByLocation := e.baseValueActivationsByLocation
	// Use the base value activation for the location, if any
	// (previously implicitly created using DeclareValue)
	baseValueActivation = baseValueActivationsByLocation[location]
	if baseValueActivation == nil {
		// If no base value activation for the location exists
		// (no value was previously, specifically declared for the location using DeclareValue),
		// return the base value activation that applies to all locations by default
		baseValueActivation = e.defaultBaseValueActivation
	}
	return

}

// getBaseTypeActivation returns the base activation for the given location.
// If a type was declared for the location (using DeclareType),
// then the specific base type activation for this location is returned.
// Otherwise, the default base activation that applies for all locations is returned.
func (e *interpreterEnvironment) getBaseTypeActivation(
	location common.Location,
) (
	baseTypeActivation *sema.VariableActivation,
) {
	// Use the base type activation for the location, if any
	// (previously implicitly created using DeclareType)
	baseTypeActivationsByLocation := e.baseTypeActivationsByLocation
	baseTypeActivation = baseTypeActivationsByLocation[location]
	if baseTypeActivation == nil {
		// If no base type activation for the location exists
		// (no type was previously, specifically declared for the location using DeclareType),
		// return the base type activation that applies to all locations by default
		baseTypeActivation = e.defaultBaseTypeActivation
	}
	return
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
