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
	stdErrors "errors"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

type Script struct {
	Source    []byte
	Arguments [][]byte
}

type importResolutionResults map[Location]bool

// Executor is a continuation which represents a full unit of transaction/script
// execution.
//
// The full unit of execution is divided into stages:
//  1. Preprocess() initializes the executor in preparation for the actual
//     transaction execution (e.g., parse / type check the input).  Note that
//     the work done by Preprocess() should be embrassingly parallel.
//  2. Execute() performs the actual transaction execution (e.g., run the
//     interpreter to produce the transaction result).
//  3. Result() returns the result of the full unit of execution.
//
// TODO: maybe add Cleanup/Postprocess in the future
type Executor interface {
	// Preprocess prepares the transaction/script for execution.
	//
	// This function returns an error if the program has errors (e.g., syntax
	// errors, type errors).
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	Preprocess() error

	// Execute executes the transaction/script.
	//
	// This function returns an error if Preprocess failed or if the execution
	// fails.
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	//
	// Note: Execute will invoke Preprocess to ensure Preprocess was called at
	// least once.
	Execute() error

	// Result returns the transaction/scipt's execution result.
	//
	// This function returns an error if Preproces or Execute fails.  The
	// cadence.Value is always nil for transaction.
	//
	// This method may be called multiple times.  Only the first call will
	// trigger meaningful work; subsequent calls will return the cached return
	// value from the original call (i.e., an Executor implementation must
	// guard this method with sync.Once).
	//
	// Note: Result will invoke Execute to ensure Execute was called at least
	// once.
	Result() (cadence.Value, error)
}

// Runtime is a runtime capable of executing Cadence.
type Runtime interface {
	// Config returns the runtime.Config this Runtime was instantiated with.
	Config() Config

	// NewScriptExecutor returns an executor which executes the given script.
	NewScriptExecutor(Script, Context) Executor

	// ExecuteScript executes the given script.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteScript(Script, Context) (cadence.Value, error)

	// NewTransactionExecutor returns an executor which executes the given
	// transaction.
	NewTransactionExecutor(Script, Context) Executor

	// ExecuteTransaction executes the given transaction.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteTransaction(Script, Context) error

	// NewContractFunctionExecutor returns an executor which invokes a contract
	// function with the given arguments.
	NewContractFunctionExecutor(
		contractLocation common.AddressLocation,
		functionName string,
		arguments []cadence.Value,
		argumentTypes []sema.Type,
		context Context,
	) Executor

	// InvokeContractFunction invokes a contract function with the given arguments.
	//
	// This function returns an error if the execution fails.
	// If the contract function accepts an &Account as a parameter,
	// the corresponding argument can be an interpreter.Address.
	// Returns a cadence.Value
	InvokeContractFunction(
		contractLocation common.AddressLocation,
		functionName string,
		arguments []cadence.Value,
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

	// Storage returns the storage system and an interpreter which can be used for
	// accessing values in storage.
	//
	// NOTE: only use the interpreter for storage operations,
	// do *NOT* use the interpreter for any other purposes,
	// such as executing a program.
	//
	Storage(context Context) (*Storage, *interpreter.Interpreter, error)

	SetDebugger(debugger *interpreter.Debugger)
}

type ImportResolver = func(location Location) (program *ast.Program, e error)

var validTopLevelDeclarationsInTransaction = common.NewDeclarationKindSet(
	common.DeclarationKindPragma,
	common.DeclarationKindImport,
	common.DeclarationKindFunction,
	common.DeclarationKindTransaction,
)

var validTopLevelDeclarationsInAccountCode = common.NewDeclarationKindSet(
	common.DeclarationKindPragma,
	common.DeclarationKindImport,
	common.DeclarationKindContract,
	common.DeclarationKindContractInterface,
)

func validTopLevelDeclarations(location Location) common.DeclarationKindSet {
	switch location.(type) {
	case common.TransactionLocation:
		return validTopLevelDeclarationsInTransaction
	case common.AddressLocation:
		return validTopLevelDeclarationsInAccountCode
	}

	return common.AllDeclarationKindsSet
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

// NewInterpreterRuntime returns an interpreter-based version of the Flow runtime.
func NewInterpreterRuntime(defaultConfig Config) Runtime {
	return &interpreterRuntime{
		defaultConfig: defaultConfig,
	}
}

func (r *interpreterRuntime) Config() Config {
	return r.defaultConfig
}

func (r *interpreterRuntime) Recover(onError func(Error), location Location, codesAndPrograms CodesAndPrograms) {
	recovered := recover()
	if recovered == nil {
		return
	}

	err := GetWrappedError(recovered, location, codesAndPrograms)
	onError(err)
}

func GetWrappedError(recovered any, location Location, codesAndPrograms CodesAndPrograms) Error {
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

func (r *interpreterRuntime) NewScriptExecutor(
	script Script,
	context Context,
) Executor {
	return newInterpreterScriptExecutor(r, script, context)
}

func (r *interpreterRuntime) ExecuteScript(script Script, context Context) (val cadence.Value, err error) {
	location := context.Location
	if _, ok := location.(common.ScriptLocation); !ok {
		return nil, errors.NewUnexpectedError("invalid non-script location: %s", location)
	}
	return r.NewScriptExecutor(script, context).Result()
}

func (r *interpreterRuntime) NewContractFunctionExecutor(
	contractLocation common.AddressLocation,
	functionName string,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
	context Context,
) Executor {
	return newInterpreterContractFunctionExecutor(
		r,
		contractLocation,
		functionName,
		arguments,
		argumentTypes,
		context,
	)
}

func (r *interpreterRuntime) InvokeContractFunction(
	contractLocation common.AddressLocation,
	functionName string,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
	context Context,
) (cadence.Value, error) {
	return r.NewContractFunctionExecutor(
		contractLocation,
		functionName,
		arguments,
		argumentTypes,
		context,
	).Result()
}

func (r *interpreterRuntime) NewTransactionExecutor(script Script, context Context) Executor {
	return newInterpreterTransactionExecutor(r, script, context)
}

func (r *interpreterRuntime) ExecuteTransaction(script Script, context Context) (err error) {
	location := context.Location
	if _, ok := location.(common.TransactionLocation); !ok {
		return errors.NewUnexpectedError("invalid non-transaction location: %s", location)
	}
	_, err = r.NewTransactionExecutor(script, context).Result()
	return err
}

// UserPanicToError Executes `f` and gracefully handle `UserError` panics.
// All on-user panics (including `InternalError` and `ExternalError`) are propagated up.
func UserPanicToError(f func()) (returnedError error) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				panic(errors.NewUnexpectedError("%s", r))
			}

			var userError errors.UserError
			if stdErrors.As(err, &userError) {
				// Return user errors
				returnedError = err
				return
			}

			switch err.(type) {
			case errors.InternalError, errors.ExternalError:
				panic(err)

			// Otherwise, panic.
			// Also wrap with a `UnexpectedError` to mark it as an `InternalError`.
			default:
				panic(errors.NewUnexpectedErrorFromCause(err))
			}
		}
	}()

	f()
	return nil
}

type LocationResolver interface {
	ResolveLocation(identifiers []ast.Identifier, location common.Location) ([]ResolvedLocation, error)
}

type ArgumentDecoder interface {
	stdlib.StandardLibraryHandler
	LocationResolver

	// DecodeArgument decodes a transaction/script argument against the given type.
	DecodeArgument(argument []byte, argumentType cadence.Type) (cadence.Value, error)
}

// ParseAndCheckProgram parses the given code and checks it.
// Returns a program that can be interpreted (AST + elaboration).
func (r *interpreterRuntime) ParseAndCheckProgram(
	code []byte,
	context Context,
) (
	program *interpreter.Program,
	err error,
) {
	location := context.Location

	codesAndPrograms := NewCodesAndPrograms()

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
		context.CoverageReport,
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

type InterpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func (r *interpreterRuntime) Storage(context Context) (*Storage, *interpreter.Interpreter, error) {

	location := context.Location

	codesAndPrograms := NewCodesAndPrograms()

	runtimeInterface := context.Interface

	storage := NewStorage(
		runtimeInterface,
		runtimeInterface,
		StorageConfig{},
	)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}

	environment.Configure(
		runtimeInterface,
		codesAndPrograms,
		storage,
		context.CoverageReport,
	)

	_, inter, err := environment.interpret(
		location,
		nil,
		nil,
	)
	if err != nil {
		return nil, nil, newError(err, location, codesAndPrograms)
	}

	return storage, inter, nil
}

func (r *interpreterRuntime) ReadStored(
	address common.Address,
	path cadence.Path,
	context Context,
) (
	val cadence.Value,
	err error,
) {
	location := context.Location

	var codesAndPrograms CodesAndPrograms

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	_, inter, err := r.Storage(context)
	if err != nil {
		// error is already wrapped as Error in Storage
		return nil, err
	}

	pathValue := valueImporter{context: inter}.importPathValue(path)

	domain := pathValue.Domain.StorageDomain()
	identifier := pathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	value := inter.ReadStored(address, domain, storageMapKey)

	var exportedValue cadence.Value
	if value != nil {
		exportedValue, err = ExportValue(value, inter, interpreter.EmptyLocationRange)
		if err != nil {
			return nil, newError(err, location, codesAndPrograms)
		}
	}

	return exportedValue, nil
}

func (r *interpreterRuntime) SetDebugger(debugger *interpreter.Debugger) {
	r.defaultConfig.Debugger = debugger
}
