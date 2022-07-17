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

	"github.com/opentracing/opentracing-go"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type Script struct {
	Source    []byte
	Arguments [][]byte
}

type importResolutionResults map[common.Location]bool

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

	// SetCoverageReport activates reporting coverage in the given report.
	// Passing nil disables coverage reporting (default).
	//
	SetCoverageReport(coverageReport *CoverageReport)

	// SetAtreeValidationEnabled configures if atree validation is enabled.
	SetAtreeValidationEnabled(enabled bool)

	// SetTracingEnabled configures if tracing is enabled.
	SetTracingEnabled(enabled bool)

	// SetInvalidatedResourceValidationEnabled configures
	// if invalidated resource validation is enabled.
	SetInvalidatedResourceValidationEnabled(enabled bool)

	// SetResourceOwnerChangeHandlerEnabled configures if the resource owner change callback is enabled.
	SetResourceOwnerChangeHandlerEnabled(enabled bool)

	// ReadStored reads the value stored at the given path
	//
	ReadStored(address common.Address, path cadence.Path, context Context) (cadence.Value, error)

	// ReadLinked dereferences the path and returns the value stored at the target
	//
	ReadLinked(address common.Address, path cadence.Path, context Context) (cadence.Value, error)

	// SetDebugger configures interpreters with the given debugger.
	//
	SetDebugger(debugger *interpreter.Debugger)
}

type ImportResolver = func(location common.Location) (program *ast.Program, e error)

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

func validTopLevelDeclarations(location common.Location) []common.DeclarationKind {
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

// interpreterRuntime is a interpreter-based version of the Flow runtime.
type interpreterRuntime struct {
	coverageReport                       *CoverageReport
	debugger                             *interpreter.Debugger
	atreeValidationEnabled               bool
	tracingEnabled                       bool
	resourceOwnerChangeHandlerEnabled    bool
	invalidatedResourceValidationEnabled bool
}

type Option func(Runtime)

// WithAtreeValidationEnabled returns a runtime option
// that configures if atree validation is enabled.
//
func WithAtreeValidationEnabled(enabled bool) Option {
	return func(runtime Runtime) {
		runtime.SetAtreeValidationEnabled(enabled)
	}
}

// WithTracingEnabled returns a runtime option
// that configures if tracing is enabled.
//
func WithTracingEnabled(enabled bool) Option {
	return func(runtime Runtime) {
		runtime.SetTracingEnabled(enabled)
	}
}

// WithInvalidatedResourceValidationEnabled returns a runtime option
// that configures if invalidated resource validation is enabled.
//
func WithInvalidatedResourceValidationEnabled(enabled bool) Option {
	return func(runtime Runtime) {
		runtime.SetInvalidatedResourceValidationEnabled(enabled)
	}
}

// WithResourceOwnerChangeCallbackEnabled returns a runtime option
// that configures if the resource owner change callback is enabled.
//
func WithResourceOwnerChangeCallbackEnabled(enabled bool) Option {
	return func(runtime Runtime) {
		runtime.SetResourceOwnerChangeHandlerEnabled(enabled)
	}
}

// NewInterpreterRuntime returns a interpreter-based version of the Flow runtime.
func NewInterpreterRuntime(options ...Option) Runtime {
	runtime := &interpreterRuntime{}
	for _, option := range options {
		option(runtime)
	}
	return runtime
}

func (r *interpreterRuntime) Recover(onError func(Error), context Context) {
	recovered := recover()
	if recovered == nil {
		return
	}

	err := getWrappedError(recovered, context)
	onError(err)
}

func getWrappedError(recovered any, context Context) Error {
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
		return newError(recovered.(error), context)

	// Wrap any other unhandled error with a generic internal error first.
	// And then wrap with `runtime.Error` to include meta info.
	case error:
		err := errors.NewUnexpectedErrorFromCause(recovered)
		return newError(err, context)
	default:
		err := errors.NewUnexpectedError("%s", recovered)
		return newError(err, context)
	}
}

func (r *interpreterRuntime) SetCoverageReport(coverageReport *CoverageReport) {
	r.coverageReport = coverageReport
}

func (r *interpreterRuntime) SetAtreeValidationEnabled(enabled bool) {
	r.atreeValidationEnabled = enabled
}

func (r *interpreterRuntime) SetTracingEnabled(enabled bool) {
	r.tracingEnabled = enabled
}

func (r *interpreterRuntime) SetInvalidatedResourceValidationEnabled(enabled bool) {
	r.invalidatedResourceValidationEnabled = enabled
}

func (r *interpreterRuntime) SetResourceOwnerChangeHandlerEnabled(enabled bool) {
	r.resourceOwnerChangeHandlerEnabled = enabled
}

func (r *interpreterRuntime) SetDebugger(debugger *interpreter.Debugger) {
	r.debugger = debugger
}

func (r *interpreterRuntime) ExecuteScript(script Script, context Context) (val cadence.Value, err error) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	context.InitializeCodesAndPrograms()

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	storage := NewStorage(context.Interface, memoryGauge)

	// TODO: allow caller to pass this in so it can be reused
	environment := NewScriptEnvironment()
	environment.Interface = context.Interface
	environment.Storage = storage

	program, err := r.parseAndCheckProgram(
		script.Source,
		context,
		environment,
		true,
		importResolutionResults{},
	)
	if err != nil {
		return nil, newError(err, context)
	}

	functionEntryPointType, err := program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return nil, newError(err, context)
	}

	// Ensure the entry point's parameter types are importable
	if len(functionEntryPointType.Parameters) > 0 {
		for _, param := range functionEntryPointType.Parameters {
			if !param.TypeAnnotation.Type.IsImportable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotImportableError{
					Type: param.TypeAnnotation.Type,
				}
				return nil, newError(err, context)
			}
		}
	}

	// Ensure the entry point's return type is valid
	if !functionEntryPointType.ReturnTypeAnnotation.Type.IsExternallyReturnable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: functionEntryPointType.ReturnTypeAnnotation.Type,
		}
		return nil, newError(err, context)
	}

	interpret := scriptExecutionFunction(
		functionEntryPointType.Parameters,
		script.Arguments,
		context.Interface,
		interpreter.ReturnEmptyLocationRange,
	)

	value, inter, err := r.interpret(
		program,
		context,
		storage,
		environment,
		interpret,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	// Export before committing storage

	result, err := exportValue(value, interpreter.ReturnEmptyLocationRange)
	if err != nil {
		return nil, newError(err, context)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	err = r.commitStorage(storage, inter)
	if err != nil {
		return nil, newError(err, context)
	}

	return result, nil
}

func (r *interpreterRuntime) commitStorage(storage *Storage, inter *interpreter.Interpreter) error {
	const commitContractUpdates = true
	err := storage.Commit(inter, commitContractUpdates)
	if err != nil {
		return err
	}

	if r.atreeValidationEnabled {
		err = storage.CheckHealth()
		if err != nil {
			return err
		}
	}

	return nil
}

type interpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	getLocationRange func() interpreter.LocationRange,
) interpretFunc {
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

func (r *interpreterRuntime) interpret(
	program *interpreter.Program,
	context Context,
	storage *Storage,
	environment *Environment,
	f interpretFunc,
) (
	exportableValue,
	*interpreter.Interpreter,
	error,
) {

	inter, err := r.newInterpreter(
		program,
		context,
		storage,
		environment,
	)
	if err != nil {
		return exportableValue{}, nil, err
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
		context.Interface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramInterpreted(context.Location, duration)
		},
	)

	if err != nil {
		return exportableValue{}, nil, err
	}

	var exportedValue exportableValue
	if f != nil {
		exportedValue = newExportableValue(result, inter)
	}

	if inter.ExitHandler != nil {
		err = inter.ExitHandler()
	}
	return exportedValue, inter, err
}

func (r *interpreterRuntime) InvokeContractFunction(
	contractLocation common.AddressLocation,
	functionName string,
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	context Context,
) (val cadence.Value, err error) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	context.InitializeCodesAndPrograms()

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	storage := NewStorage(context.Interface, memoryGauge)

	// TODO: allow caller to pass this in so it can be reused
	environment := NewBaseEnvironment()
	environment.Interface = context.Interface
	environment.Storage = storage

	// create interpreter
	_, inter, err := r.interpret(
		nil,
		context,
		storage,
		environment,
		nil,
	)
	if err != nil {
		return nil, newError(err, context)
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
		return nil, newError(err, context)
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
		return nil, newError(
			interpreter.NotInvokableError{
				Value: contractFunction,
			},
			context)
	}

	value, err := inter.InvokeFunction(contractFunction, invocation)
	if err != nil {
		return nil, newError(err, context)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = r.commitStorage(storage, inter)
	if err != nil {
		return nil, newError(err, context)
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, inter, interpreter.ReturnEmptyLocationRange)
	if err != nil {
		return nil, newError(err, context)
	}

	return exportedValue, nil
}

func (r *interpreterRuntime) convertArgument(
	gauge common.MemoryGauge,
	argument interpreter.Value,
	argumentType sema.Type,
	environment *Environment,
) interpreter.Value {
	switch argumentType {
	case sema.AuthAccountType:
		// convert addresses to auth accounts so there is no need to construct an auth account value for the caller
		if addressValue, ok := argument.(interpreter.AddressValue); ok {
			return stdlib.NewAuthAccountValue(
				gauge,
				environment,
				interpreter.NewAddressValueFromConstructor(
					gauge,
					addressValue.ToAddress,
				),
			)
		}
	case sema.PublicAccountType:
		// convert addresses to public accounts so there is no need to construct a public account value for the caller
		if addressValue, ok := argument.(interpreter.AddressValue); ok {
			return stdlib.NewPublicAccountValue(
				gauge,
				environment,
				interpreter.NewAddressValueFromConstructor(
					gauge,
					addressValue.ToAddress,
				),
			)
		}
	}
	return argument
}

func (r *interpreterRuntime) ExecuteTransaction(script Script, context Context) (err error) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	context.InitializeCodesAndPrograms()

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	storage := NewStorage(context.Interface, memoryGauge)

	// TODO: allow caller to pass this in so it can be reused
	environment := NewBaseEnvironment()
	environment.Interface = context.Interface
	environment.Storage = storage

	program, err := r.parseAndCheckProgram(
		script.Source,
		context,
		environment,
		true,
		importResolutionResults{},
	)
	if err != nil {
		return newError(err, context)
	}

	transactions := program.Elaboration.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		err = InvalidTransactionCountError{
			Count: transactionCount,
		}
		return newError(err, context)
	}

	transactionType := transactions[0]

	var authorizers []Address
	wrapPanic(func() {
		authorizers, err = context.Interface.GetSigningAccounts()
	})
	if err != nil {
		return newError(err, context)
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
		return newError(err, context)
	}

	transactionAuthorizerCount := len(transactionType.PrepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		err = InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		}
		return newError(err, context)
	}

	// gather authorizers

	authorizerValues := func(inter *interpreter.Interpreter) []interpreter.Value {

		authorizerValues := make([]interpreter.Value, 0, authorizerCount)

		for _, address := range authorizers {
			authorizerValues = append(
				authorizerValues,
				stdlib.NewAuthAccountValue(
					inter,
					environment,
					interpreter.NewAddressValue(
						inter,
						address,
					),
				),
			)
		}

		return authorizerValues
	}

	_, inter, err := r.interpret(
		program,
		context,
		storage,
		environment,
		r.transactionExecutionFunction(
			transactionType.Parameters,
			script.Arguments,
			context.Interface,
			authorizerValues,
		),
	)
	if err != nil {
		return newError(err, context)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = r.commitStorage(storage, inter)
	if err != nil {
		return newError(err, context)
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

func (r *interpreterRuntime) transactionExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	authorizerValues func(*interpreter.Interpreter) []interpreter.Value,
) interpretFunc {
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
func (r *interpreterRuntime) ParseAndCheckProgram(
	code []byte,
	context Context,
) (
	program *interpreter.Program,
	err error,
) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	context.InitializeCodesAndPrograms()

	// TODO: allow caller to pass this in so it can be reused
	environment := NewBaseEnvironment()
	environment.Interface = context.Interface

	program, err = r.parseAndCheckProgram(
		code,
		context,
		environment,
		true,
		importResolutionResults{},
	)
	if err != nil {
		return nil, newError(err, context)
	}

	return program, nil
}

func (r *interpreterRuntime) parseAndCheckProgram(
	code []byte,
	context Context,
	environment *Environment,
	storeProgram bool,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {
	wrapError := func(err error) error {
		return &ParsingCheckingError{
			Err:      err,
			Location: context.Location,
		}
	}

	if storeProgram {
		context.SetCode(context.Location, code)
	}

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	// Parse

	var parse *ast.Program
	reportMetric(
		func() {
			parse, err = parser.ParseProgram(string(code), memoryGauge)
		},
		context.Interface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(context.Location, duration)
		},
	)
	if err != nil {
		return nil, wrapError(err)
	}

	if storeProgram {
		context.SetProgram(context.Location, parse)
	}

	// Check

	elaboration, err := r.check(parse, context, environment, checkedImports)
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
			err = context.Interface.SetProgram(context.Location, program)
		})
		if err != nil {
			return nil, err
		}
	}

	return program, nil
}

func (r *interpreterRuntime) check(
	program *ast.Program,
	startContext Context,
	environment *Environment,
	checkedImports importResolutionResults,
) (
	elaboration *sema.Elaboration,
	err error,
) {

	memoryGauge, _ := startContext.Interface.(common.MemoryGauge)

	checker, err := sema.NewChecker(
		program,
		startContext.Location,
		memoryGauge,
		false,
		sema.WithBaseValueActivation(environment.baseValueActivation),
		sema.WithValidTopLevelDeclarationsHandler(validTopLevelDeclarations),
		sema.WithLocationHandler(
			func(identifiers []Identifier, location Location) (res []ResolvedLocation, err error) {
				wrapPanic(func() {
					res, err = startContext.Interface.ResolveLocation(identifiers, location)
				})
				return
			},
		),
		sema.WithImportHandler(
			func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {

				var elaboration *sema.Elaboration
				switch importedLocation {
				case stdlib.CryptoChecker.Location:
					elaboration = stdlib.CryptoChecker.Elaboration

				default:
					context := startContext.WithLocation(importedLocation)

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

					program, err := r.getProgram(context, environment, checkedImports)
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
				startContext.Interface,
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

func (r *interpreterRuntime) newInterpreter(
	program *interpreter.Program,
	context Context,
	storage *Storage,
	environment *Environment,
) (*interpreter.Interpreter, error) {

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	defaultOptions := []interpreter.Option{
		// NOTE: storage option must be provided *before* the predeclared values option,
		// as predeclared values may rely on storage
		interpreter.WithStorage(storage),
		interpreter.WithBaseActivation(environment.baseActivation),
		interpreter.WithOnEventEmittedHandler(
			func(
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
					context.Interface.EmitEvent,
				)

				return nil
			},
		),
		interpreter.WithInjectedCompositeFieldsHandler(
			r.injectedCompositeFieldsHandler(environment),
		),
		interpreter.WithUUIDHandler(func() (uuid uint64, err error) {
			wrapPanic(func() {
				uuid, err = context.Interface.GenerateUUID()
			})
			return
		}),
		interpreter.WithContractValueHandler(
			func(
				inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
				invocationRange ast.Range,
			) *interpreter.CompositeValue {

				return r.loadContract(
					inter,
					compositeType,
					constructorGenerator,
					invocationRange,
					storage,
				)
			},
		),
		interpreter.WithImportLocationHandler(
			r.importLocationHandler(context, environment),
		),
		interpreter.WithOnStatementHandler(
			r.onStatementHandler(),
		),
		interpreter.WithPublicAccountHandler(
			func(address interpreter.AddressValue) interpreter.Value {
				return stdlib.NewPublicAccountValue(
					memoryGauge,
					environment,
					address,
				)
			},
		),
		interpreter.WithPublicKeyValidationHandler(func(
			inter *interpreter.Interpreter,
			getLocationRange func() interpreter.LocationRange,
			publicKey *interpreter.CompositeValue,
		) error {
			return validatePublicKey(
				inter,
				getLocationRange,
				publicKey,
				context.Interface,
			)
		}),
		interpreter.WithBLSCryptoFunctions(
			func(
				inter *interpreter.Interpreter,
				getLocationRange func() interpreter.LocationRange,
				publicKeyValue interpreter.MemberAccessibleValue,
				signature *interpreter.ArrayValue,
			) interpreter.BoolValue {
				return blsVerifyPoP(
					inter,
					getLocationRange,
					publicKeyValue,
					signature,
					context.Interface,
				)
			},
			func(
				inter *interpreter.Interpreter,
				getLocationRange func() interpreter.LocationRange,
				signatures *interpreter.ArrayValue,
			) interpreter.OptionalValue {
				return blsAggregateSignatures(
					inter,
					context.Interface,
					signatures,
				)
			},
			func(
				inter *interpreter.Interpreter,
				getLocationRange func() interpreter.LocationRange,
				publicKeys *interpreter.ArrayValue,
			) interpreter.OptionalValue {
				return blsAggregatePublicKeys(
					inter,
					getLocationRange,
					publicKeys,
					func(
						inter *interpreter.Interpreter,
						getLocationRange func() interpreter.LocationRange,
						publicKey *interpreter.CompositeValue,
					) error {
						return validatePublicKey(
							inter,
							getLocationRange,
							publicKey,
							context.Interface,
						)
					},
					context.Interface,
				)
			},
		),
		interpreter.WithSignatureVerificationHandler(
			func(
				inter *interpreter.Interpreter,
				getLocationRange func() interpreter.LocationRange,
				signature *interpreter.ArrayValue,
				signedData *interpreter.ArrayValue,
				domainSeparationTag *interpreter.StringValue,
				hashAlgorithm *interpreter.SimpleCompositeValue,
				publicKey interpreter.MemberAccessibleValue,
			) interpreter.BoolValue {
				return verifySignature(
					inter,
					getLocationRange,
					signature,
					signedData,
					domainSeparationTag,
					hashAlgorithm,
					publicKey,
					context.Interface,
				)
			},
		),
		interpreter.WithHashHandler(
			func(
				inter *interpreter.Interpreter,
				getLocationRange func() interpreter.LocationRange,
				data *interpreter.ArrayValue,
				tag *interpreter.StringValue,
				hashAlgorithm interpreter.MemberAccessibleValue,
			) *interpreter.ArrayValue {
				return hash(
					inter,
					getLocationRange,
					data,
					tag,
					hashAlgorithm,
					context.Interface,
				)
			},
		),
		interpreter.WithOnRecordTraceHandler(
			func(
				interpreter *interpreter.Interpreter,
				functionName string,
				duration time.Duration,
				logs []opentracing.LogRecord,
			) {
				context.Interface.RecordTrace(functionName, interpreter.Location, duration, logs)
			},
		),
		interpreter.WithTracingEnabled(r.tracingEnabled),
		interpreter.WithAtreeValueValidationEnabled(r.atreeValidationEnabled),
		// NOTE: ignore r.atreeValidationEnabled here,
		// and disable storage validation after each value modification.
		// Instead, storage is validated after commits (if validation is enabled).
		interpreter.WithAtreeStorageValidationEnabled(false),
		interpreter.WithOnResourceOwnerChangeHandler(r.resourceOwnerChangedHandler(context.Interface)),
		interpreter.WithInvalidatedResourceValidationEnabled(r.invalidatedResourceValidationEnabled),
		interpreter.WithMemoryGauge(memoryGauge),
		interpreter.WithDebugger(r.debugger),
	}

	defaultOptions = append(
		defaultOptions,
		r.meteringInterpreterOptions(context.Interface)...,
	)

	return interpreter.NewInterpreter(
		program,
		context.Location,
		defaultOptions...,
	)
}

func (r *interpreterRuntime) importLocationHandler(
	startContext Context,
	environment *Environment,
) interpreter.ImportLocationHandlerFunc {

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
			context := startContext.WithLocation(location)

			program, err := r.getProgram(context, environment, importResolutionResults{})
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

// getProgram returns the existing program at the given location, if available.
// If it is not available, it loads the code, and then parses and checks it.
//
func (r *interpreterRuntime) getProgram(
	context Context,
	environment *Environment,
	checkedImports importResolutionResults,
) (
	program *interpreter.Program,
	err error,
) {

	wrapPanic(func() {
		program, err = context.Interface.GetProgram(context.Location)
	})
	if err != nil {
		return nil, err
	}

	if program == nil {

		var code []byte
		code, err = r.getCode(context)
		if err != nil {
			return nil, err
		}

		program, err = r.parseAndCheckProgram(
			code,
			context,
			environment,
			true,
			checkedImports,
		)
		if err != nil {
			return nil, err
		}
	}

	context.SetProgram(context.Location, program.Program)

	return program, nil
}

func (r *interpreterRuntime) injectedCompositeFieldsHandler(
	environment *Environment,
) interpreter.InjectedCompositeFieldsHandlerFunc {
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
						environment,
						addressValue,
					),
				}
			}
		}

		return nil
	}
}

func (r *interpreterRuntime) meteringInterpreterOptions(runtimeInterface Interface) []interpreter.Option {
	callStackDepth := 0
	// TODO: make runtime interface function
	const callStackDepthLimit = 2000

	checkCallStackDepth := func() {

		if callStackDepth <= callStackDepthLimit {
			return
		}

		panic(CallStackLimitExceededError{
			Limit: callStackDepthLimit,
		})
	}

	return []interpreter.Option{
		interpreter.WithOnFunctionInvocationHandler(
			func(_ *interpreter.Interpreter, _ int) {
				callStackDepth++
				checkCallStackDepth()
			},
		),
		interpreter.WithOnInvokedFunctionReturnHandler(
			func(_ *interpreter.Interpreter, _ int) {
				callStackDepth--
			},
		),
		interpreter.WithOnMeterComputationFuncHandler(
			func(compKind common.ComputationKind, intensity uint) {
				var err error
				wrapPanic(func() {
					err = runtimeInterface.MeterComputation(compKind, intensity)
				})
				if err != nil {
					panic(err)
				}
			},
		),
	}
}

func (r *interpreterRuntime) getCode(context Context) (code []byte, err error) {
	if addressLocation, ok := context.Location.(common.AddressLocation); ok {
		wrapPanic(func() {
			code, err = context.Interface.GetAccountContractCode(
				addressLocation.Address,
				addressLocation.Name,
			)
		})
	} else {
		wrapPanic(func() {
			code, err = context.Interface.GetCode(context.Location)
		})
	}
	if err != nil {
		return nil, err
	}

	return code, nil
}

func (r *interpreterRuntime) loadContract(
	inter *interpreter.Interpreter,
	compositeType *sema.CompositeType,
	constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
	invocationRange ast.Range,
	storage *Storage,
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
			storageMap := storage.GetStorageMap(
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

func (r *interpreterRuntime) instantiateContract(
	program *interpreter.Program,
	context Context,
	address common.Address,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	argumentTypes []sema.Type,
	storage *Storage,
	environment *Environment,
) (
	*interpreter.CompositeValue,
	error,
) {
	parameterTypes := make([]sema.Type, len(contractType.ConstructorParameters))

	for i, constructorParameter := range contractType.ConstructorParameters {
		parameterTypes[i] = constructorParameter.TypeAnnotation.Type
	}

	// Check argument count

	argumentCount := len(argumentTypes)
	parameterCount := len(parameterTypes)

	if argumentCount < parameterCount {
		return nil, errors.NewDefaultUserError(
			"invalid argument count, too few arguments: expected %d, got %d, next missing argument: `%s`",
			parameterCount, argumentCount,
			parameterTypes[argumentCount],
		)
	} else if argumentCount > parameterCount {
		return nil, errors.NewDefaultUserError(
			"invalid argument count, too many arguments: expected %d, got %d",
			parameterCount,
			argumentCount,
		)
	}

	// argumentCount now equals to parameterCount

	// Check arguments match parameter

	for i := 0; i < argumentCount; i++ {
		argumentType := argumentTypes[i]
		parameterTye := parameterTypes[i]
		if !sema.IsSubType(argumentType, parameterTye) {
			return nil, errors.NewDefaultUserError(
				"invalid argument %d: expected type `%s`, got `%s`",
				i,
				parameterTye,
				argumentType,
			)
		}
	}

	// Use a custom contract value handler that detects if the requested contract value
	// is for the contract declaration that is being deployed.
	//
	// If the contract is the deployed contract, instantiate it using
	// the provided constructor and given arguments.
	//
	// If the contract is not the deployed contract, load it from storage.

	var contract *interpreter.CompositeValue

	// TODO:
	//interpreter.WithContractValueHandler(
	//	func(
	//		inter *interpreter.Interpreter,
	//		compositeType *sema.CompositeType,
	//		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
	//		invocationRange ast.Range,
	//	) *interpreter.CompositeValue {
	//
	//		constructor := constructorGenerator(address)
	//
	//		// If the contract is the deployed contract, instantiate it using
	//		// the provided constructor and given arguments
	//
	//		if compositeType.Location == contractType.Location &&
	//			compositeType.Identifier == contractType.Identifier {
	//
	//			value, err := inter.InvokeFunctionValue(
	//				constructor,
	//				constructorArguments,
	//				argumentTypes,
	//				parameterTypes,
	//				invocationRange,
	//			)
	//			if err != nil {
	//				panic(err)
	//			}
	//
	//			return value.(*interpreter.CompositeValue)
	//		}
	//
	//		// The contract is not the deployed contract, load it from storage
	//		return r.loadContract(
	//			inter,
	//			compositeType,
	//			constructorGenerator,
	//			invocationRange,
	//			storage,
	//		)
	//	},
	//

	_, inter, err := r.interpret(
		program,
		context,
		storage,
		environment,
		nil,
	)

	if err != nil {
		return nil, err
	}

	variable, ok := inter.Globals.Get(contractType.Identifier)
	if !ok {
		return nil, errors.NewDefaultUserError(
			"cannot find contract: `%s`",
			contractType.Identifier,
		)
	}

	contract = variable.GetValue().(*interpreter.CompositeValue)

	return contract, err
}

// ignoreUpdatedProgramParserError determines if the parsing error
// for a program that is being updated can be ignored.
func ignoreUpdatedProgramParserError(err error) bool {
	parserError, ok := err.(parser.Error)
	if !ok {
		return false
	}

	// Are all parse errors ones that can be ignored?
	for _, parseError := range parserError.Errors {
		// Missing commas in parameter lists were reported starting
		// with https://github.com/onflow/cadence/pull/1073.
		// Allow existing contracts with such an error to be updated
		_, ok := parseError.(*parser.MissingCommaInParameterListError)
		if !ok {
			return false
		}
	}

	return true
}

type updateAccountContractCodeOptions struct {
	createContract bool
}

// updateAccountContractCode updates an account contract's code.
// This function is only used for the new account code/contract API.
//
func (r *interpreterRuntime) updateAccountContractCode(
	program *interpreter.Program,
	context Context,
	storage *Storage,
	name string,
	code []byte,
	addressValue interpreter.AddressValue,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	constructorArgumentTypes []sema.Type,
	environment *Environment,
	options updateAccountContractCodeOptions,
) error {
	// If the code declares a contract, instantiate it and store it.
	//
	// This function might be called when
	// 1. A contract is deployed (contractType is non-nil).
	// 2. A contract interface is deployed (contractType is nil).
	//
	// If a contract is deployed, it is only instantiated
	// when options.createContract is true,
	// i.e. the Cadence `add` function is used.
	// If the Cadence `update__experimental` function is used,
	// the new contract will NOT be deployed (options.createContract is false).

	var contractValue *interpreter.CompositeValue

	createContract := contractType != nil && options.createContract

	address := addressValue.ToAddress()

	var err error

	if createContract {
		contractValue, err = r.instantiateContract(
			program,
			context,
			address,
			contractType,
			constructorArguments,
			constructorArgumentTypes,
			storage,
			environment,
		)

		if err != nil {
			return err
		}
	}

	// NOTE: only update account code if contract instantiation succeeded
	wrapPanic(func() {
		err = context.Interface.UpdateAccountContractCode(address, name, code)
	})
	if err != nil {
		return err
	}

	if createContract {
		// NOTE: the contract recording delays the write
		// until the end of the execution of the program

		storage.recordContractUpdate(
			addressValue.ToAddress(),
			name,
			contractValue,
		)
	}

	return nil
}

func (r *interpreterRuntime) onStatementHandler() interpreter.OnStatementFunc {
	if r.coverageReport == nil {
		return nil
	}

	return func(inter *interpreter.Interpreter, statement ast.Statement) {
		location := inter.Location
		line := statement.StartPosition().Line
		r.coverageReport.AddLineHit(location, line)
	}
}

func (r *interpreterRuntime) executeNonProgram(
	interpret interpretFunc,
	context Context,
	environment *Environment,
) (cadence.Value, error) {
	context.InitializeCodesAndPrograms()

	var program *interpreter.Program

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	storage := NewStorage(context.Interface, memoryGauge)

	value, _, err := r.interpret(
		program,
		context,
		storage,
		environment,
		interpret,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	if value.Value == nil {
		return nil, nil
	}

	return exportValue(value, interpreter.ReturnEmptyLocationRange)
}

func (r *interpreterRuntime) ReadStored(
	address common.Address,
	path cadence.Path,
	context Context,
) (
	val cadence.Value,
	err error,
) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	// TODO: allow caller to pass this in so it can be reused
	environment := NewBaseEnvironment()
	environment.Interface = context.Interface

	return r.executeNonProgram(
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			pathValue := importPathValue(inter, path)

			domain := pathValue.Domain.Identifier()
			identifier := pathValue.Identifier

			value := inter.ReadStored(address, domain, identifier)

			return value, nil
		},
		context,
		environment,
	)
}

func (r *interpreterRuntime) ReadLinked(
	address common.Address,
	path cadence.Path,
	context Context,
) (
	val cadence.Value,
	err error,
) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	// TODO: allow caller to pass this in so it can be reused
	environment := NewBaseEnvironment()
	environment.Interface = context.Interface

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
		environment,
	)
}

func (r *interpreterRuntime) resourceOwnerChangedHandler(
	runtimeInterface Interface,
) interpreter.OnResourceOwnerChangeFunc {
	if !r.resourceOwnerChangeHandlerEnabled {
		return nil
	}
	return func(
		interpreter *interpreter.Interpreter,
		resource *interpreter.CompositeValue,
		oldOwner common.Address,
		newOwner common.Address,
	) {
		wrapPanic(func() {
			runtimeInterface.ResourceOwnerChanged(
				interpreter,
				resource,
				oldOwner,
				newOwner,
			)
		})
	}
}

func validatePublicKey(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKeyValue *interpreter.CompositeValue,
	runtimeInterface Interface,
) error {
	publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
	if err != nil {
		return err
	}

	wrapPanic(func() {
		err = runtimeInterface.ValidatePublicKey(publicKey)
	})

	return err
}

func blsVerifyPoP(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKeyValue interpreter.MemberAccessibleValue,
	signatureValue *interpreter.ArrayValue,
	runtimeInterface Interface,
) interpreter.BoolValue {

	publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
	if err != nil {
		panic(err)
	}

	signature, err := interpreter.ByteArrayValueToByteSlice(inter, signatureValue)
	if err != nil {
		panic(err)
	}

	var valid bool
	wrapPanic(func() {
		valid, err = runtimeInterface.BLSVerifyPOP(publicKey, signature)
	})
	if err != nil {
		panic(err)
	}

	return interpreter.BoolValue(valid)
}

func blsAggregateSignatures(
	inter *interpreter.Interpreter,
	runtimeInterface Interface,
	signaturesValue *interpreter.ArrayValue,
) interpreter.OptionalValue {

	bytesArray := make([][]byte, 0, signaturesValue.Count())
	signaturesValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
		signature, ok := element.(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		bytes, err := interpreter.ByteArrayValueToByteSlice(inter, signature)
		if err != nil {
			panic(err)
		}

		bytesArray = append(bytesArray, bytes)

		// Continue iteration
		return true
	})

	var err error
	var aggregatedSignature []byte
	wrapPanic(func() {
		aggregatedSignature, err = runtimeInterface.BLSAggregateSignatures(bytesArray)
	})

	// If the crypto layer produces an error, we have invalid input, return nil
	if err != nil {
		return interpreter.NilValue{}
	}

	aggregatedSignatureValue := interpreter.ByteSliceToByteArrayValue(inter, aggregatedSignature)

	return interpreter.NewSomeValueNonCopying(
		inter,
		aggregatedSignatureValue,
	)
}

func blsAggregatePublicKeys(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	publicKeysValue *interpreter.ArrayValue,
	validator interpreter.PublicKeyValidationHandlerFunc,
	runtimeInterface Interface,
) interpreter.OptionalValue {

	publicKeys := make([]*stdlib.PublicKey, 0, publicKeysValue.Count())
	publicKeysValue.Iterate(inter, func(element interpreter.Value) (resume bool) {
		publicKeyValue, ok := element.(*interpreter.CompositeValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		publicKey, err := stdlib.NewPublicKeyFromValue(inter, getLocationRange, publicKeyValue)
		if err != nil {
			panic(err)
		}

		publicKeys = append(publicKeys, publicKey)

		// Continue iteration
		return true
	})

	var err error
	var aggregatedPublicKey *stdlib.PublicKey
	wrapPanic(func() {
		aggregatedPublicKey, err = runtimeInterface.BLSAggregatePublicKeys(publicKeys)
	})

	// If the crypto layer produces an error, we have invalid input, return nil
	if err != nil {
		return interpreter.NilValue{}
	}

	aggregatedPublicKeyValue := stdlib.NewPublicKeyValue(
		inter,
		getLocationRange,
		aggregatedPublicKey,
		validator,
	)

	return interpreter.NewSomeValueNonCopying(
		inter,
		aggregatedPublicKeyValue,
	)
}

func verifySignature(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	signatureValue *interpreter.ArrayValue,
	signedDataValue *interpreter.ArrayValue,
	domainSeparationTagValue *interpreter.StringValue,
	hashAlgorithmValue *interpreter.SimpleCompositeValue,
	publicKeyValue interpreter.MemberAccessibleValue,
	runtimeInterface Interface,
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
		valid, err = runtimeInterface.VerifySignature(
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

func hash(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	dataValue *interpreter.ArrayValue,
	tagValue *interpreter.StringValue,
	hashAlgorithmValue interpreter.Value,
	runtimeInterface Interface,
) *interpreter.ArrayValue {

	data, err := interpreter.ByteArrayValueToByteSlice(inter, dataValue)
	if err != nil {
		panic(errors.NewUnexpectedError("failed to get data. %w", err))
	}

	var tag string
	if tagValue != nil {
		tag = tagValue.Str
	}

	hashAlgorithm := stdlib.NewHashAlgorithmFromValue(inter, getLocationRange, hashAlgorithmValue)

	var result []byte
	wrapPanic(func() {
		result, err = runtimeInterface.Hash(data, tag, hashAlgorithm)
	})
	if err != nil {
		panic(err)
	}

	return interpreter.ByteSliceToByteArrayValue(inter, result)
}
