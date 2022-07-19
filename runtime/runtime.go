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
	goRuntime "runtime"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type Script struct {
	Source    []byte
	Arguments [][]byte
}

type importResolutionResults map[Location]bool

// Runtime is a runtime capable of executing Cadence.
type Runtime interface {
	// ExecuteScript executes the given script.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteScript(Script, Context) (cadence.Value, error)

	// ExecuteTransaction executes the given transaction.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteTransaction(Script, Context) error

	// InvokeContractFunction invokes a contract function with the given arguments.
	//
	// This function returns an error if the execution fails.
	// If the contract function accepts an AuthAccount as a parameter the corresponding argument can be an interpreter.Address.
	// returns a cadence.Value
	InvokeContractFunction(
		contractLocation common.AddressLocation,
		functionName string,
		arguments []interpreter.Value,
		argumentTypes []sema.Type,
		context Context,
	) (cadence.Value, error)

	// ParseAndCheckProgram parses and checks the given code without executing the program.
	//
	// This function returns an error if the program contains any syntax or semantic errors.
	ParseAndCheckProgram(source []byte, context Context) (*interpreter.Program, error)

	// ReadStored reads the value stored at the given path
	//
	ReadStored(address common.Address, path cadence.Path, context Context) (cadence.Value, error)

	// ReadLinked dereferences the path and returns the value stored at the target
	//
	ReadLinked(address common.Address, path cadence.Path, context Context) (cadence.Value, error)
}

type ImportResolver = func(location Location) (program *ast.Program, e error)

var validTopLevelDeclarationsInTransaction = []common.DeclarationKind{
	common.DeclarationKindImport,
	common.DeclarationKindFunction,
	common.DeclarationKindTransaction,
}

var validTopLevelDeclarationsInAccountCode = []common.DeclarationKind{
	common.DeclarationKindPragma,
	common.DeclarationKindImport,
	common.DeclarationKindContract,
	common.DeclarationKindContractInterface,
}

func validTopLevelDeclarations(location Location) []common.DeclarationKind {
	switch location.(type) {
	case common.TransactionLocation:
		return validTopLevelDeclarationsInTransaction
	case common.AddressLocation:
		return validTopLevelDeclarationsInAccountCode
	}

	return nil
}

func reportMetric(
	f func(),
	runtimeInterface Interface,
	report func(Metrics, time.Duration),
) {
	metrics, ok := runtimeInterface.(Metrics)
	if !ok {
		f()
		return
	}

	start := time.Now()
	f()
	elapsed := time.Since(start)

	report(metrics, elapsed)
}

// interpreterRuntime is an interpreter-based version of the Flow runtime.
type interpreterRuntime struct {
	defaultConfig Config
}

// NewInterpreterRuntime returns a interpreter-based version of the Flow runtime.
func NewInterpreterRuntime(defaultConfig Config) Runtime {
	return interpreterRuntime{
		defaultConfig: defaultConfig,
	}
}

func (r interpreterRuntime) Recover(onError func(Error), location Location, codesAndPrograms codesAndPrograms) {
	recovered := recover()
	if recovered == nil {
		return
	}

	err := getWrappedError(recovered, location, codesAndPrograms)
	onError(err)
}

func getWrappedError(recovered any, location Location, codesAndPrograms codesAndPrograms) Error {
	switch recovered := recovered.(type) {

	// If the error is already a `runtime.Error`, then avoid redundant wrapping.
	case Error:
		return recovered

	// Wrap with `runtime.Error` to include meta info.
	//
	// The following set of errors are the only known types of errors that would reach this point.
	// `interpreter.Error` is a generic wrapper for any error. Hence, it doesn't belong to any of the
	// three types: `UserError`, `InternalError`, `ExternalError`.
	// So it needs to be specially handled here
	case errors.InternalError,
		errors.UserError,
		errors.ExternalError,
		interpreter.Error:
		return newError(recovered.(error), location, codesAndPrograms)

	// Wrap any other unhandled error with a generic internal error first.
	// And then wrap with `runtime.Error` to include meta info.
	case error:
		err := errors.NewUnexpectedErrorFromCause(recovered)
		return newError(err, location, codesAndPrograms)
	default:
		err := errors.NewUnexpectedError("%s", recovered)
		return newError(err, location, codesAndPrograms)
	}
}

func (r interpreterRuntime) ExecuteScript(script Script, context Context) (val cadence.Value, err error) {

	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	storage := NewStorage(context.Interface, context.Interface)

	environment := context.Environment
	if environment == nil {
		environment = NewScriptInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		storage,
	)

	program, err := environment.ParseAndCheckProgram(
		script.Source,
		location,
		true,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	functionEntryPointType, err := program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Ensure the entry point's parameter types are importable
	if len(functionEntryPointType.Parameters) > 0 {
		for _, param := range functionEntryPointType.Parameters {
			if !param.TypeAnnotation.Type.IsImportable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotImportableError{
					Type: param.TypeAnnotation.Type,
				}
				return nil, newError(err, location, codesAndPrograms)
			}
		}
	}

	// Ensure the entry point's return type is valid
	if !functionEntryPointType.ReturnTypeAnnotation.Type.IsExternallyReturnable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: functionEntryPointType.ReturnTypeAnnotation.Type,
		}
		return nil, newError(err, location, codesAndPrograms)
	}

	interpret := scriptExecutionFunction(
		functionEntryPointType.Parameters,
		script.Arguments,
		context.Interface,
	)

	value, inter, err := environment.Interpret(
		location,
		program,
		interpret,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Export before committing storage
	exportableValue := newExportableValue(value, inter)
	result, err := exportValue(
		exportableValue,
		interpreter.ReturnEmptyLocationRange,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	err = environment.CommitStorage(inter)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return result, nil
}

type InterpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
) InterpretFunc {
	return func(inter *interpreter.Interpreter) (value interpreter.Value, err error) {

		// Recover internal panics and return them as an error.
		// For example, the argument validation might attempt to
		// load contract code for non-existing types

		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		values, err := validateArgumentParams(
			inter,
			runtimeInterface,
			interpreter.ReturnEmptyLocationRange,
			arguments,
			parameters,
		)
		if err != nil {
			return nil, err
		}
		return inter.Invoke("main", values...)
	}
}

func (r interpreterRuntime) InvokeContractFunction(
	contractLocation common.AddressLocation,
	functionName string,
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	context Context,
) (val cadence.Value, err error) {

	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	storage := NewStorage(context.Interface, context.Interface)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		storage,
	)

	// create interpreter
	_, inter, err := environment.Interpret(
		location,
		nil,
		nil,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// ensure the contract is loaded
	inter = inter.EnsureLoaded(contractLocation)

	for i, argumentType := range argumentTypes {
		arguments[i] = r.convertArgument(
			inter,
			arguments[i],
			argumentType,
			environment,
		)
	}

	contractValue, err := inter.GetContractComposite(contractLocation)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// prepare invocation
	invocation := interpreter.NewInvocation(
		inter,
		contractValue,
		arguments,
		argumentTypes,
		nil,
		func() interpreter.LocationRange {
			return interpreter.LocationRange{
				Location: context.Location,
			}
		},
	)

	contractMember := contractValue.GetMember(inter, invocation.GetLocationRange, functionName)

	contractFunction, ok := contractMember.(interpreter.FunctionValue)
	if !ok {
		err := interpreter.NotInvokableError{
			Value: contractFunction,
		}
		return nil, newError(err, location, codesAndPrograms)
	}

	value, err := inter.InvokeFunction(contractFunction, invocation)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.CommitStorage(inter)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, inter, interpreter.ReturnEmptyLocationRange)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return exportedValue, nil
}

func (r interpreterRuntime) convertArgument(
	gauge common.MemoryGauge,
	argument interpreter.Value,
	argumentType sema.Type,
	environment Environment,
) interpreter.Value {
	switch argumentType {
	case sema.AuthAccountType:
		// convert addresses to auth accounts so there is no need to construct an auth account value for the caller
		if addressValue, ok := argument.(interpreter.AddressValue); ok {
			address := interpreter.NewAddressValueFromConstructor(gauge, addressValue.ToAddress)
			return environment.NewAuthAccountValue(address)
		}
	case sema.PublicAccountType:
		// convert addresses to public accounts so there is no need to construct a public account value for the caller
		if addressValue, ok := argument.(interpreter.AddressValue); ok {
			address := interpreter.NewAddressValueFromConstructor(gauge, addressValue.ToAddress)
			return environment.NewPublicAccountValue(address)
		}
	}
	return argument
}

func (r interpreterRuntime) ExecuteTransaction(script Script, context Context) (err error) {

	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	storage := NewStorage(context.Interface, context.Interface)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		storage,
	)

	program, err := environment.ParseAndCheckProgram(
		script.Source,
		location,
		true,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	transactions := program.Elaboration.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		err = InvalidTransactionCountError{
			Count: transactionCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	transactionType := transactions[0]

	var authorizers []Address
	wrapPanic(func() {
		authorizers, err = context.Interface.GetSigningAccounts()
	})
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	// check parameter count

	argumentCount := len(script.Arguments)
	authorizerCount := len(authorizers)

	transactionParameterCount := len(transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		err = InvalidEntryPointParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	transactionAuthorizerCount := len(transactionType.PrepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		err = InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	// gather authorizers

	authorizerValues := func(inter *interpreter.Interpreter) []interpreter.Value {

		authorizerValues := make([]interpreter.Value, 0, authorizerCount)

		for _, address := range authorizers {
			addressValue := interpreter.NewAddressValue(inter, address)
			authorizerValues = append(
				authorizerValues,
				environment.NewAuthAccountValue(addressValue),
			)
		}

		return authorizerValues
	}

	interpretFunc := r.transactionExecutionFunction(
		transactionType.Parameters,
		script.Arguments,
		context.Interface,
		authorizerValues,
	)

	_, inter, err := environment.Interpret(
		location,
		program,
		interpretFunc,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.CommitStorage(inter)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	return nil
}

func wrapPanic(f func()) {
	defer func() {
		if r := recover(); r != nil {
			// don't wrap Go errors and internal errors
			switch r := r.(type) {
			case goRuntime.Error, errors.InternalError:
				panic(r)
			default:
				panic(errors.ExternalError{
					Recovered: r,
				})
			}

		}
	}()
	f()
}

// userPanicToError Executes `f` and gracefully handle `UserError` panics.
// All on-user panics (including `InternalError` and `ExternalError`) are propagated up.
//
func userPanicToError(f func()) (returnedError error) {
	defer func() {
		if r := recover(); r != nil {
			switch err := r.(type) {
			case errors.UserError:
				// Return user errors
				returnedError = err
			case errors.InternalError, errors.ExternalError:
				panic(err)

			// Otherwise, panic.
			// Also wrap with a `UnexpectedError` to mark it as an `InternalError`.
			case error:
				panic(errors.NewUnexpectedErrorFromCause(err))
			default:
				panic(errors.NewUnexpectedError("%s", r))
			}
		}
	}()

	f()
	return nil
}

func (r interpreterRuntime) transactionExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	authorizerValues func(*interpreter.Interpreter) []interpreter.Value,
) InterpretFunc {
	return func(inter *interpreter.Interpreter) (value interpreter.Value, err error) {

		// Recover internal panics and return them as an error.
		// For example, the argument validation might attempt to
		// load contract code for non-existing types

		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		values, err := validateArgumentParams(
			inter,
			runtimeInterface,
			interpreter.ReturnEmptyLocationRange,
			arguments,
			parameters,
		)
		if err != nil {
			return nil, err
		}

		values = append(values, authorizerValues(inter)...)
		err = inter.InvokeTransaction(0, values...)
		return nil, err
	}
}

func validateArgumentParams(
	inter *interpreter.Interpreter,
	runtimeInterface Interface,
	getLocationRange func() interpreter.LocationRange,
	arguments [][]byte,
	parameters []*sema.Parameter,
) (
	[]interpreter.Value,
	error,
) {
	argumentCount := len(arguments)
	parameterCount := len(parameters)

	if argumentCount != parameterCount {
		return nil, InvalidEntryPointParameterCountError{
			Expected: parameterCount,
			Actual:   argumentCount,
		}
	}

	argumentValues := make([]interpreter.Value, len(arguments))

	// Decode arguments against parameter types
	for i, parameter := range parameters {
		parameterType := parameter.TypeAnnotation.Type
		argument := arguments[i]

		exportedParameterType := ExportMeteredType(inter, parameterType, map[sema.TypeID]cadence.Type{})
		var value cadence.Value
		var err error

		wrapPanic(func() {
			value, err = runtimeInterface.DecodeArgument(
				argument,
				exportedParameterType,
			)
		})

		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err:   err,
			}
		}

		var arg interpreter.Value
		panicError := userPanicToError(func() {
			// if importing an invalid public key, this call panics
			arg, err = importValue(
				inter,
				getLocationRange,
				value,
				parameterType,
			)
		})

		if panicError != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err:   panicError,
			}
		}

		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err:   err,
			}
		}

		// Ensure the argument is of an importable type
		argType := arg.StaticType(inter)

		if !arg.IsImportable(inter) {
			return nil, &ArgumentNotImportableError{
				Type: argType,
			}
		}

		// Check that decoded value is a subtype of static parameter type
		if !inter.IsSubTypeOfSemaType(argType, parameterType) {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err: &InvalidValueTypeError{
					ExpectedType: parameterType,
				},
			}
		}

		// Check whether the decoded value conforms to the type associated with the value
		if !arg.ConformsToStaticType(
			inter,
			interpreter.ReturnEmptyLocationRange,
			interpreter.TypeConformanceResults{},
		) {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err: &MalformedValueError{
					ExpectedType: parameterType,
				},
			}
		}

		// Ensure static type info is available for all values
		interpreter.InspectValue(inter, arg, func(value interpreter.Value) bool {
			if value == nil {
				return true
			}

			if !hasValidStaticType(inter, value) {
				panic(errors.NewUnexpectedError("invalid static type for argument: %d", i))
			}

			return true
		})

		argumentValues[i] = arg
	}

	return argumentValues, nil
}

func hasValidStaticType(inter *interpreter.Interpreter, value interpreter.Value) bool {
	switch value := value.(type) {
	case *interpreter.ArrayValue:
		return value.Type != nil
	case *interpreter.DictionaryValue:
		return value.Type.KeyType != nil &&
			value.Type.ValueType != nil
	default:
		// For other values, static type is NOT inferred.
		// Hence no need to validate it here.
		return value.StaticType(inter) != nil
	}
}

// ParseAndCheckProgram parses the given code and checks it.
// Returns a program that can be interpreted (AST + elaboration).
//
func (r interpreterRuntime) ParseAndCheckProgram(
	code []byte,
	context Context,
) (
	program *interpreter.Program,
	err error,
) {
	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		nil,
	)

	program, err = environment.ParseAndCheckProgram(
		code,
		location,
		true,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return program, nil
}

func (r interpreterRuntime) executeNonProgram(
	interpret InterpretFunc,
	context Context,
) (cadence.Value, error) {

	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	storage := NewStorage(context.Interface, context.Interface)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		storage,
	)

	value, inter, err := environment.Interpret(
		location,
		nil,
		interpret,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	if value == nil {
		return nil, nil
	}

	exportedValue := newExportableValue(value, inter)

	return exportValue(
		exportedValue,
		interpreter.ReturnEmptyLocationRange,
	)
}

func (r interpreterRuntime) ReadStored(
	address common.Address,
	path cadence.Path,
	context Context,
) (
	val cadence.Value,
	err error,
) {
	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	return r.executeNonProgram(
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			pathValue := importPathValue(inter, path)

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			value := inter.ReadStored(address, domain, identifier)

			return value, nil
		},
		context,
	)
}

func (r interpreterRuntime) ReadLinked(
	address common.Address,
	path cadence.Path,
	context Context,
) (
	val cadence.Value,
	err error,
) {
	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	return r.executeNonProgram(
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			targetPath, _, err := inter.GetCapabilityFinalTargetPath(
				address,
				importPathValue(inter, path),
				&sema.ReferenceType{
					Type: sema.AnyType,
				},
				interpreter.ReturnEmptyLocationRange,
			)
			if err != nil {
				return nil, err
			}

			if targetPath == interpreter.EmptyPathValue {
				return nil, nil
			}

			value := inter.ReadStored(
				address,
				targetPath.Domain.Identifier(),
				targetPath.Identifier,
			)
			return value, nil
		},
		context,
	)
}
