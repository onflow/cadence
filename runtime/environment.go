/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type Environment struct {
	baseActivation                        *interpreter.VariableActivation
	baseValueActivation                   *sema.VariableActivation
	Interface                             Interface
	Storage                               *Storage
	deployedContractConstructorInvocation *stdlib.DeployedContractConstructorInvocation
}

var _ stdlib.Logger = &Environment{}
var _ stdlib.UnsafeRandomGenerator = &Environment{}
var _ stdlib.BlockAtHeightProvider = &Environment{}
var _ stdlib.CurrentBlockProvider = &Environment{}
var _ stdlib.PublicAccountHandler = &Environment{}
var _ stdlib.AccountCreator = &Environment{}
var _ stdlib.EventEmitter = &Environment{}
var _ stdlib.AuthAccountHandler = &Environment{}

func newEnvironment() *Environment {
	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseActivation := interpreter.NewVariableActivation(nil, interpreter.BaseActivation)
	return &Environment{
		baseActivation:      baseActivation,
		baseValueActivation: baseValueActivation,
	}
}

func (e *Environment) Declare(valueDeclaration stdlib.StandardLibraryValue) {
	e.baseValueActivation.DeclareValue(valueDeclaration)
	e.baseActivation.Declare(valueDeclaration)
}

func NewBaseEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	env := newEnvironment()
	for _, valueDeclaration := range stdlib.BuiltinValues {
		env.Declare(valueDeclaration)
	}
	env.Declare(stdlib.NewLogFunction(env))
	env.Declare(stdlib.NewUnsafeRandomFunction(env))
	env.Declare(stdlib.NewGetBlockFunction(env))
	env.Declare(stdlib.NewGetCurrentBlockFunction(env))
	env.Declare(stdlib.NewGetAccountFunction(env))
	env.Declare(stdlib.NewAuthAccountConstructor(env))
	for _, declaration := range declarations {
		env.Declare(declaration)
	}
	return env
}

func NewScriptEnvironment(declarations ...stdlib.StandardLibraryValue) *Environment {
	env := NewBaseEnvironment(declarations...)
	env.Declare(stdlib.NewGetAuthAccountFunction(env))
	return env
}

func (e *Environment) ProgramLog(message string) error {
	return e.Interface.ProgramLog(message)
}

func (e *Environment) UnsafeRandom() (uint64, error) {
	return e.Interface.UnsafeRandom()
}

func (e *Environment) GetBlockAtHeight(height uint64) (block stdlib.Block, exists bool, err error) {
	return e.Interface.GetBlockAtHeight(height)
}

func (e *Environment) GetCurrentBlockHeight() (uint64, error) {
	return e.Interface.GetCurrentBlockHeight()
}

func (e *Environment) GetAccountBalance(address common.Address) (uint64, error) {
	return e.Interface.GetAccountBalance(address)
}

func (e *Environment) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	return e.Interface.GetAccountAvailableBalance(address)
}

func (e *Environment) CommitStorage(inter *interpreter.Interpreter, commitContractUpdates bool) error {
	return e.Storage.Commit(inter, commitContractUpdates)
}

func (e *Environment) GetStorageUsed(address common.Address) (uint64, error) {
	return e.Interface.GetStorageUsed(address)
}

func (e *Environment) GetStorageCapacity(address common.Address) (uint64, error) {
	return e.Interface.GetStorageCapacity(address)
}

func (e *Environment) GetAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	return e.Interface.GetAccountKey(address, index)
}

func (e *Environment) GetAccountContractNames(address common.Address) ([]string, error) {
	return e.Interface.GetAccountContractNames(address)
}

func (e *Environment) GetAccountContractCode(address common.Address, name string) ([]byte, error) {
	return e.Interface.GetAccountContractCode(address, name)
}

func (e *Environment) CreateAccount(payer common.Address) (address common.Address, err error) {
	return e.Interface.CreateAccount(payer)
}

func (e *Environment) EmitEvent(
	inter *interpreter.Interpreter,
	eventType *sema.CompositeType,
	values []interpreter.Value,
	getLocationRange func() interpreter.LocationRange,
) {
	eventFields := make([]exportableValue, 0, len(values))

	for _, value := range values {
		eventFields = append(eventFields, newExportableValue(value, inter))
	}

	emitEventFields(
		inter,
		getLocationRange,
		eventType,
		eventFields,
		e.Interface.EmitEvent,
	)
}

func (e *Environment) AddEncodedAccountKey(address common.Address, key []byte) error {
	return e.Interface.AddEncodedAccountKey(address, key)
}

func (e *Environment) RevokeEncodedAccountKey(address common.Address, index int) ([]byte, error) {
	return e.Interface.RevokeEncodedAccountKey(address, index)
}

func (e *Environment) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (*stdlib.AccountKey, error) {
	return e.Interface.AddAccountKey(address, key, algo, weight)
}

func (e *Environment) RevokeAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	return e.Interface.RevokeAccountKey(address, index)
}

func (e *Environment) UpdateAccountContractCode(address common.Address, name string, code []byte) error {
	return e.Interface.UpdateAccountContractCode(address, name, code)
}

func (e *Environment) RemoveAccountContractCode(address common.Address, name string) error {
	return e.Interface.RemoveAccountContractCode(address, name)
}

func (e *Environment) RecordContractRemoval(address common.Address, name string) {
	e.Storage.recordContractUpdate(address, name, nil)
}

func (e *Environment) RecordContractUpdate(
	address common.Address,
	name string,
	contractValue *interpreter.CompositeValue,
) {
	e.Storage.recordContractUpdate(address, name, contractValue)
}

func (e *Environment) ParseAndCheckProgram(
	gauge common.MemoryGauge,
	code []byte,
	location common.Location,
	storeProgram bool,
) (
	*interpreter.Program,
	error,
) {
	return e.parseAndCheckProgram(
		gauge,
		code,
		location,
		storeProgram,
		importResolutionResults{},
	)
}

func (e *Environment) parseAndCheckProgram(
	gauge common.MemoryGauge,
	code []byte,
	location common.Location,
	storeProgram bool,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {
	wrapError := func(err error) error {
		return &ParsingCheckingError{
			Err:      err,
			Location: location,
		}
	}

	if storeProgram {
		// TODO:
		//context.SetCode(location, code)
	}

	// Parse

	var parse *ast.Program
	reportMetric(
		func() {
			parse, err = parser.ParseProgram(string(code), gauge)
		},
		e.Interface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(location, duration)
		},
	)
	if err != nil {
		return nil, wrapError(err)
	}

	if storeProgram {
		// TODO:
		//context.SetProgram(location, parse)
	}

	// Check

	elaboration, err := e.check(gauge, location, parse, checkedImports)
	if err != nil {
		return nil, wrapError(err)
	}

	// Return

	program = &interpreter.Program{
		Program:     parse,
		Elaboration: elaboration,
	}

	if storeProgram {
		wrapPanic(func() {
			err = e.Interface.SetProgram(location, program)
		})
		if err != nil {
			return nil, err
		}
	}

	return program, nil
}

func (e *Environment) check(
	gauge common.MemoryGauge,
	location common.Location,
	program *ast.Program,
	checkedImports importResolutionResults,
) (
	elaboration *sema.Elaboration,
	err error,
) {
	checker, err := sema.NewChecker(
		program,
		location,
		gauge,
		false,
		sema.WithBaseValueActivation(e.baseValueActivation),
		sema.WithValidTopLevelDeclarationsHandler(validTopLevelDeclarations),
		sema.WithLocationHandler(
			func(identifiers []Identifier, location Location) (res []ResolvedLocation, err error) {
				wrapPanic(func() {
					res, err = e.Interface.ResolveLocation(identifiers, location)
				})
				return
			},
		),
		sema.WithImportHandler(
			func(
				checker *sema.Checker,
				importedLocation common.Location,
				importRange ast.Range,
			) (sema.Import, error) {

				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				default:

					// Check for cyclic imports
					if checkedImports[importedLocation] {
						return nil, &sema.CyclicImportsError{
							Location: importedLocation,
							Range:    importRange,
						}
					} else {
						checkedImports[importedLocation] = true
						defer delete(checkedImports, importedLocation)
					}

					program, err := e.getProgram(gauge, importedLocation, checkedImports)
					if err != nil {
						return nil, err
					}

					elaboration = program.Elaboration
				}

				return sema.ElaborationImport{
					Elaboration: elaboration,
				}, nil
			},
		),
		sema.WithCheckHandler(func(checker *sema.Checker, check func()) {
			reportMetric(
				check,
				e.Interface,
				func(metrics Metrics, duration time.Duration) {
					metrics.ProgramChecked(checker.Location, duration)
				},
			)
		}),
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

func (e *Environment) GetProgram(gauge common.MemoryGauge, location Location) (*interpreter.Program, error) {
	return e.getProgram(gauge, location, importResolutionResults{})
}

// getProgram returns the existing program at the given location, if available.
// If it is not available, it loads the code, and then parses and checks it.
//
func (e *Environment) getProgram(
	gauge common.MemoryGauge,
	location Location,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {
	wrapPanic(func() {
		program, err = e.Interface.GetProgram(location)
	})
	if err != nil {
		return nil, err
	}

	if program == nil {
		var code []byte
		code, err = e.getCode(location)
		if err != nil {
			return nil, err
		}

		program, err = e.parseAndCheckProgram(
			gauge,
			code,
			location,
			true,
			checkedImports,
		)
		if err != nil {
			return nil, err
		}
	}

	// TODO:
	//context.SetProgram(location, program.Program)

	return program, nil
}

func (e *Environment) getCode(location common.Location) (code []byte, err error) {
	if addressLocation, ok := location.(common.AddressLocation); ok {
		wrapPanic(func() {
			code, err = e.Interface.GetAccountContractCode(
				addressLocation.Address,
				addressLocation.Name,
			)
		})
	} else {
		wrapPanic(func() {
			code, err = e.Interface.GetCode(location)
		})
	}
	return
}

func (e *Environment) newInterpreter(
	gauge common.MemoryGauge,
	location common.Location,
	program *interpreter.Program,
) (*interpreter.Interpreter, error) {

	defaultOptions := []interpreter.Option{
		interpreter.WithStorage(e.Storage),
		interpreter.WithMemoryGauge(gauge),
		interpreter.WithBaseActivation(e.baseActivation),
		interpreter.WithOnEventEmittedHandler(e.onEventEmittedHandler()),
		interpreter.WithInjectedCompositeFieldsHandler(e.injectedCompositeFieldsHandler()),
		interpreter.WithUUIDHandler(e.uuidHandler()),
		interpreter.WithContractValueHandler(e.contractValueHandler()),
		interpreter.WithImportLocationHandler(e.importLocationHandler()),
		interpreter.WithPublicAccountHandler(
			func(address interpreter.AddressValue) interpreter.Value {
				return stdlib.NewPublicAccountValue(gauge, e, address)
			},
		),
		interpreter.WithPublicKeyValidationHandler(e.publicKeyValidationHandler()),
		// TODO:
		//interpreter.WithBLSCryptoFunctions(
		//	func(
		//		inter *interpreter.Interpreter,
		//		getLocationRange func() interpreter.LocationRange,
		//		publicKeyValue interpreter.MemberAccessibleValue,
		//		signature *interpreter.ArrayValue,
		//	) interpreter.BoolValue {
		//		return blsVerifyPoP(
		//			inter,
		//			getLocationRange,
		//			publicKeyValue,
		//			signature,
		//			context.Interface,
		//		)
		//	},
		//	func(
		//		inter *interpreter.Interpreter,
		//		getLocationRange func() interpreter.LocationRange,
		//		signatures *interpreter.ArrayValue,
		//	) interpreter.OptionalValue {
		//		return blsAggregateSignatures(
		//			inter,
		//			context.Interface,
		//			signatures,
		//		)
		//	},
		//	func(
		//		inter *interpreter.Interpreter,
		//		getLocationRange func() interpreter.LocationRange,
		//		publicKeys *interpreter.ArrayValue,
		//	) interpreter.OptionalValue {
		//		return blsAggregatePublicKeys(
		//			inter,
		//			getLocationRange,
		//			publicKeys,
		//			func(
		//				inter *interpreter.Interpreter,
		//				getLocationRange func() interpreter.LocationRange,
		//				publicKey *interpreter.CompositeValue,
		//			) error {
		//				return validatePublicKey(
		//					inter,
		//					getLocationRange,
		//					publicKey,
		//					context.Interface,
		//				)
		//			},
		//			context.Interface,
		//		)
		//	},
		//),
		interpreter.WithSignatureVerificationHandler(e.signatureVerificationHandler()),
		//interpreter.WithHashHandler(
		//	func(
		//		inter *interpreter.Interpreter,
		//		getLocationRange func() interpreter.LocationRange,
		//		data *interpreter.ArrayValue,
		//		tag *interpreter.StringValue,
		//		hashAlgorithm interpreter.MemberAccessibleValue,
		//	) *interpreter.ArrayValue {
		//		return hash(
		//			inter,
		//			getLocationRange,
		//			data,
		//			tag,
		//			hashAlgorithm,
		//			context.Interface,
		//		)
		//	},
		//),
		//interpreter.WithOnRecordTraceHandler(
		//	func(
		//		interpreter *interpreter.Interpreter,
		//		functionName string,
		//		duration time.Duration,
		//		logs []opentracing.LogRecord,
		//	) {
		//		context.Interface.RecordTrace(functionName, interpreter.Location, duration, logs)
		//	},
		//),
		//interpreter.WithTracingEnabled(r.tracingEnabled),
		//interpreter.WithAtreeValueValidationEnabled(r.atreeValidationEnabled),
		//// NOTE: ignore r.atreeValidationEnabled here,
		//// and disable storage validation after each value modification.
		//// Instead, storage is validated after commits (if validation is enabled).
		//interpreter.WithAtreeStorageValidationEnabled(false),
		//interpreter.WithOnResourceOwnerChangeHandler(r.resourceOwnerChangedHandler(context.Interface)),
		//interpreter.WithInvalidatedResourceValidationEnabled(r.invalidatedResourceValidationEnabled),
		//interpreter.WithDebugger(r.debugger),
	}

	// TODO:
	//defaultOptions = append(
	//	defaultOptions,
	//	r.meteringInterpreterOptions(context.Interface)...,
	//)

	return interpreter.NewInterpreter(
		program,
		location,
		defaultOptions...,
	)
}

func (e *Environment) signatureVerificationHandler() interpreter.SignatureVerificationHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		getLocationRange func() interpreter.LocationRange,
		signatureValue *interpreter.ArrayValue,
		signedDataValue *interpreter.ArrayValue,
		domainSeparationTagValue *interpreter.StringValue,
		hashAlgorithmValue *interpreter.SimpleCompositeValue,
		publicKeyValue interpreter.MemberAccessibleValue,
	) interpreter.BoolValue {

		signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to get signature. %w", err))
		}

		signedData, err := interpreter.ByteArrayValueToByteSlice(inter, signedDataValue)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to get signed data. %w", err))
		}

		domainSeparationTag := domainSeparationTagValue.Str

		hashAlgorithm := stdlib.NewHashAlgorithmFromValue(inter, getLocationRange, hashAlgorithmValue)

		publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
		if err != nil {
			return false
		}

		var valid bool
		wrapPanic(func() {
			valid, err = e.Interface.VerifySignature(
				signature,
				domainSeparationTag,
				signedData,
				publicKey.PublicKey,
				publicKey.SignAlgo,
				hashAlgorithm,
			)
		})

		if err != nil {
			panic(err)
		}

		return interpreter.BoolValue(valid)
	}
}

func (e *Environment) publicKeyValidationHandler() interpreter.PublicKeyValidationHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		getLocationRange func() interpreter.LocationRange,
		publicKeyValue *interpreter.CompositeValue,
	) error {

		publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
		if err != nil {
			return err
		}

		wrapPanic(func() {
			err = e.Interface.ValidatePublicKey(publicKey)
		})

		return err
	}
}

func (e *Environment) contractValueHandler() interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) *interpreter.CompositeValue {

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

func (e *Environment) uuidHandler() interpreter.UUIDHandlerFunc {
	return func() (uuid uint64, err error) {
		wrapPanic(func() {
			uuid, err = e.Interface.GenerateUUID()
		})
		return
	}
}

func (e *Environment) onEventEmittedHandler() interpreter.OnEventEmittedFunc {
	return func(
		inter *interpreter.Interpreter,
		getLocationRange func() interpreter.LocationRange,
		eventValue *interpreter.CompositeValue,
		eventType *sema.CompositeType,
	) error {
		emitEventValue(
			inter,
			getLocationRange,
			eventType,
			eventValue,
			e.Interface.EmitEvent,
		)

		return nil
	}
}

func (e *Environment) injectedCompositeFieldsHandler() interpreter.InjectedCompositeFieldsHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		location Location,
		_ string,
		compositeKind common.CompositeKind,
	) map[string]interpreter.Value {

		switch location {
		case stdlib.CryptoChecker.Location:
			return nil

		default:
			switch compositeKind {
			case common.CompositeKindContract:
				var address Address

				switch location := location.(type) {
				case common.AddressLocation:
					address = location.Address
				default:
					panic(errors.NewUnreachableError())
				}

				addressValue := interpreter.NewAddressValue(
					inter,
					address,
				)

				return map[string]interpreter.Value{
					"account": stdlib.NewAuthAccountValue(
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

func (e *Environment) importLocationHandler() interpreter.ImportLocationHandlerFunc {
	return func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
		switch location {
		case stdlib.CryptoChecker.Location:
			program := interpreter.ProgramFromChecker(stdlib.CryptoChecker)
			subInterpreter, err := inter.NewSubInterpreter(program, location)
			if err != nil {
				panic(err)
			}
			return interpreter.InterpreterImport{
				Interpreter: subInterpreter,
			}

		default:
			program, err := e.GetProgram(inter, location)
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

func (e *Environment) loadContract(
	inter *interpreter.Interpreter,
	compositeType *sema.CompositeType,
	constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
	invocationRange ast.Range,
) *interpreter.CompositeValue {

	switch compositeType.Location {
	case stdlib.CryptoChecker.Location:
		contract, err := stdlib.NewCryptoContract(
			inter,
			constructorGenerator(common.Address{}),
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
			storageMap := e.Storage.GetStorageMap(
				location.Address,
				StorageDomainContract,
				false,
			)
			if storageMap != nil {
				storedValue = storageMap.ReadValue(inter, location.Name)
			}
		}

		if storedValue == nil {
			panic(errors.NewDefaultUserError("failed to load contract: %s", compositeType.Location))
		}

		return storedValue.(*interpreter.CompositeValue)
	}
}

// TODO:
//func (r *interpreterRuntime) meteringInterpreterOptions(runtimeInterface Interface) []interpreter.Option {
//	callStackDepth := 0
//	// TODO: make runtime interface function
//	const callStackDepthLimit = 2000
//
//	checkCallStackDepth := func() {
//
//		if callStackDepth <= callStackDepthLimit {
//			return
//		}
//
//		panic(CallStackLimitExceededError{
//			Limit: callStackDepthLimit,
//		})
//	}
//
//	return []interpreter.Option{
//		interpreter.WithOnFunctionInvocationHandler(
//			func(_ *interpreter.Interpreter, _ int) {
//				callStackDepth++
//				checkCallStackDepth()
//			},
//		),
//		interpreter.WithOnInvokedFunctionReturnHandler(
//			func(_ *interpreter.Interpreter, _ int) {
//				callStackDepth--
//			},
//		),
//		interpreter.WithOnMeterComputationFuncHandler(
//			func(compKind common.ComputationKind, intensity uint) {
//				var err error
//				wrapPanic(func() {
//					err = runtimeInterface.MeterComputation(compKind, intensity)
//				})
//				if err != nil {
//					panic(err)
//				}
//			},
//		),
//	}
//}

func (e *Environment) InterpretContract(
	gauge common.MemoryGauge,
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

	_, inter, err := e.interpret(gauge, location, program, nil)
	if err != nil {
		return nil, err
	}

	variable, ok := inter.Globals.Get(name)
	if !ok {
		return nil, errors.NewDefaultUserError(
			"cannot find contract: `%s`",
			name,
		)
	}

	contract = variable.GetValue().(*interpreter.CompositeValue)

	return
}

func (e *Environment) interpret(
	gauge common.MemoryGauge,
	location common.Location,
	program *interpreter.Program,
	f interpretFunc,
) (
	interpreter.Value,
	*interpreter.Interpreter,
	error,
) {
	inter, err := e.newInterpreter(gauge, location, program)
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

	if inter.ExitHandler != nil {
		err = inter.ExitHandler()
	}
	return result, inter, err
}

//func (r *interpreterRuntime) resourceOwnerChangedHandler(
//	runtimeInterface Interface,
//) interpreter.OnResourceOwnerChangeFunc {
//	if !r.resourceOwnerChangeHandlerEnabled {
//		return nil
//	}
//	return func(
//		interpreter *interpreter.Interpreter,
//		resource *interpreter.CompositeValue,
//		oldOwner common.Address,
//		newOwner common.Address,
//	) {
//		wrapPanic(func() {
//			runtimeInterface.ResourceOwnerChanged(
//				interpreter,
//				resource,
//				oldOwner,
//				newOwner,
//			)
//		})
//	}
//}

//func blsVerifyPoP(
//	inter *interpreter.Interpreter,
//	getLocationRange func() interpreter.LocationRange,
//	publicKeyValue interpreter.MemberAccessibleValue,
//	signatureValue *interpreter.ArrayValue,
//	runtimeInterface Interface,
//) interpreter.BoolValue {
//
//	publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
//	if err != nil {
//		panic(err)
//	}
//
//	signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue)
//	if err != nil {
//		panic(err)
//	}
//
//	var valid bool
//	wrapPanic(func() {
//		valid, err = runtimeInterface.BLSVerifyPOP(publicKey, signature)
//	})
//	if err != nil {
//		panic(err)
//	}
//
//	return interpreter.BoolValue(valid)
//}
//
//func blsAggregateSignatures(
//	inter *interpreter.Interpreter,
//	runtimeInterface Interface,
//	signaturesValue *interpreter.ArrayValue,
//) interpreter.OptionalValue {
//
//	bytesArray := make([][]byte, 0, signaturesValue.Count())
//	signaturesValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
//		signature, ok := element.(*interpreter.ArrayValue)
//		if !ok {
//			panic(errors.NewUnreachableError())
//		}
//
//		bytes, err := interpreter.ByteArrayValueToByteSlice(inter, signature)
//		if err != nil {
//			panic(err)
//		}
//
//		bytesArray = append(bytesArray, bytes)
//
//		// Continue iteration
//		return true
//	})
//
//	var err error
//	var aggregatedSignature []byte
//	wrapPanic(func() {
//		aggregatedSignature, err = runtimeInterface.BLSAggregateSignatures(bytesArray)
//	})
//
//	// If the crypto layer produces an error, we have invalid input, return nil
//	if err != nil {
//		return interpreter.NilValue{}
//	}
//
//	aggregatedSignatureValue := interpreter.ByteSliceToByteArrayValue(inter, aggregatedSignature)
//
//	return interpreter.NewSomeValueNonCopying(
//		inter,
//		aggregatedSignatureValue,
//	)
//}
//
//func blsAggregatePublicKeys(
//	inter *interpreter.Interpreter,
//	getLocationRange func() interpreter.LocationRange,
//	publicKeysValue *interpreter.ArrayValue,
//	validator interpreter.PublicKeyValidationHandlerFunc,
//	runtimeInterface Interface,
//) interpreter.OptionalValue {
//
//	publicKeys := make([]*stdlib.PublicKey, 0, publicKeysValue.Count())
//	publicKeysValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
//		publicKeyValue, ok := element.(*interpreter.CompositeValue)
//		if !ok {
//			panic(errors.NewUnreachableError())
//		}
//
//		publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
//		if err != nil {
//			panic(err)
//		}
//
//		publicKeys = append(publicKeys, publicKey)
//
//		// Continue iteration
//		return true
//	})
//
//	var err error
//	var aggregatedPublicKey *stdlib.PublicKey
//	wrapPanic(func() {
//		aggregatedPublicKey, err = runtimeInterface.BLSAggregatePublicKeys(publicKeys)
//	})
//
//	// If the crypto layer produces an error, we have invalid input, return nil
//	if err != nil {
//		return interpreter.NilValue{}
//	}
//
//	aggregatedPublicKeyValue := stdlib.NewPublicKeyValue(
//		inter,
//		getLocationRange,
//		aggregatedPublicKey,
//		validator,
//	)
//
//	return interpreter.NewSomeValueNonCopying(
//		inter,
//		aggregatedPublicKeyValue,
//	)
//}
//
//
//func hash(
//	inter *interpreter.Interpreter,
//	getLocationRange func() interpreter.LocationRange,
//	dataValue *interpreter.ArrayValue,
//	tagValue *interpreter.StringValue,
//	hashAlgorithmValue interpreter.Value,
//	runtimeInterface Interface,
//) *interpreter.ArrayValue {
//
//	data, err := interpreter.ByteArrayValueToByteSlice(inter, dataValue)
//	if err != nil {
//		panic(errors.NewUnexpectedError("failed to get data. %w", err))
//	}
//
//	var tag string
//	if tagValue != nil {
//		tag = tagValue.Str
//	}
//
//	hashAlgorithm := stdlib.NewHashAlgorithmFromValue(inter, getLocationRange, hashAlgorithmValue)
//
//	var result []byte
//	wrapPanic(func() {
//		result, err = runtimeInterface.Hash(data, tag, hashAlgorithm)
//	})
//	if err != nil {
//		panic(err)
//	}
//
//	return interpreter.ByteSliceToByteArrayValue(inter, result)
//}
