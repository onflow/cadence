/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"errors"
	"fmt"
	"math"
	goRuntime "runtime"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/trampoline"
)

type Script struct {
	Source    []byte
	Arguments [][]byte
}

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

	// ParseAndCheckProgram parses and checks the given code without executing the program.
	//
	// This function returns an error if the program contains any syntax or semantic errors.
	ParseAndCheckProgram(source []byte, context Context) (*interpreter.Program, error)

	// SetCoverageReport activates reporting coverage in the given report.
	// Passing nil disables coverage reporting (default).
	//
	SetCoverageReport(coverageReport *CoverageReport)

	// SetContractUpdateValidationEnabled configures if contract update validation is enabled.
	//
	SetContractUpdateValidationEnabled(enabled bool)
}

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

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
	coverageReport                  *CoverageReport
	contractUpdateValidationEnabled bool
}

type Option func(Runtime)

// WithContractUpdateValidationEnabled returns a runtime option
// that configures if contract update validation is enabled.
//
func WithContractUpdateValidationEnabled(enabled bool) Option {
	return func(runtime Runtime) {
		runtime.SetContractUpdateValidationEnabled(enabled)
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

func (r *interpreterRuntime) SetCoverageReport(coverageReport *CoverageReport) {
	r.coverageReport = coverageReport
}

func (r *interpreterRuntime) SetContractUpdateValidationEnabled(enabled bool) {
	r.contractUpdateValidationEnabled = enabled
}

func (r *interpreterRuntime) ExecuteScript(script Script, context Context) (cadence.Value, error) {
	context.InitializeCodesAndPrograms()

	runtimeStorage := newRuntimeStorage(context.Interface)

	var checkerOptions []sema.Option
	var interpreterOptions []interpreter.Option

	functions := r.standardLibraryFunctions(
		context,
		runtimeStorage,
		interpreterOptions,
		checkerOptions,
	)

	program, err := r.parseAndCheckProgram(
		script.Source,
		context,
		functions,
		checkerOptions,
		true,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	functionEntryPointType, err := program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return nil, newError(err, context)
	}

	// Ensure the entry point's parameter types are storable
	if len(functionEntryPointType.Parameters) > 0 {
		for _, param := range functionEntryPointType.Parameters {
			if !param.TypeAnnotation.Type.IsStorable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotStorableError{
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
	)

	value, inter, err := r.interpret(
		program,
		context,
		runtimeStorage,
		functions,
		interpreterOptions,
		checkerOptions,
		interpret,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	runtimeStorage.writeCached(inter)

	return exportValue(value), nil
}

type interpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
) interpretFunc {
	return func(inter *interpreter.Interpreter) (interpreter.Value, error) {
		values, err := validateArgumentParams(
			inter,
			runtimeInterface,
			arguments,
			parameters)
		if err != nil {
			return nil, err
		}
		return inter.Invoke("main", values...)
	}
}

func (r *interpreterRuntime) interpret(
	program *interpreter.Program,
	context Context,
	runtimeStorage *runtimeStorage,
	functions stdlib.StandardLibraryFunctions,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
	f interpretFunc,
) (
	exportableValue,
	*interpreter.Interpreter,
	error,
) {

	inter, err := r.newInterpreter(
		program,
		context,
		functions,
		runtimeStorage,
		interpreterOptions,
		checkerOptions,
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

	return exportedValue, inter, nil
}

func (r *interpreterRuntime) newAuthAccountValue(
	addressValue interpreter.AddressValue,
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) interpreter.AuthAccountValue {
	return interpreter.NewAuthAccountValue(
		addressValue,
		storageUsedGetFunction(addressValue, context.Interface, runtimeStorage),
		storageCapacityGetFunction(addressValue, context.Interface),
		r.newAddPublicKeyFunction(addressValue, context.Interface),
		r.newRemovePublicKeyFunction(addressValue, context.Interface),
		r.newAuthAccountContracts(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
		),
	)
}

func (r *interpreterRuntime) ExecuteTransaction(script Script, context Context) error {
	context.InitializeCodesAndPrograms()

	runtimeStorage := newRuntimeStorage(context.Interface)

	var interpreterOptions []interpreter.Option
	var checkerOptions []sema.Option

	functions := r.standardLibraryFunctions(
		context,
		runtimeStorage,
		interpreterOptions,
		checkerOptions,
	)

	program, err := r.parseAndCheckProgram(
		script.Source,
		context,
		functions,
		checkerOptions,
		true,
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

	authorizerValues := make([]interpreter.Value, authorizerCount)

	for i, address := range authorizers {
		authorizerValues[i] = r.newAuthAccountValue(
			interpreter.NewAddressValue(address),
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
		)
	}

	_, inter, err := r.interpret(
		program,
		context,
		runtimeStorage,
		functions,
		interpreterOptions,
		checkerOptions,
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
	runtimeStorage.writeCached(inter)

	return nil
}

func wrapPanic(f func()) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			// don't recover Go errors
			goErr, ok := r.(goRuntime.Error)
			if ok {
				panic(goErr)
			}

			panic(interpreter.ExternalError{
				Recovered: r,
			})
		}
	}()
	f()
}

func (r *interpreterRuntime) transactionExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	authorizerValues []interpreter.Value,
) interpretFunc {
	return func(inter *interpreter.Interpreter) (interpreter.Value, error) {
		values, err := validateArgumentParams(
			inter,
			runtimeInterface,
			arguments,
			parameters,
		)
		if err != nil {
			return nil, err
		}
		allArguments := append(values, authorizerValues...)
		err = inter.InvokeTransaction(0, allArguments...)
		return nil, err
	}
}

func validateArgumentParams(
	inter *interpreter.Interpreter,
	runtimeInterface Interface,
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

		exportedParameterType := ExportType(parameterType, map[sema.TypeID]cadence.Type{})
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

		arg := importValue(value)

		// Check that decoded value is a subtype of static parameter type
		if !interpreter.IsSubType(arg.DynamicType(inter), parameterType) {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err: &InvalidTypeAssignmentError{
					Value: arg,
					Type:  parameterType,
				},
			}
		}

		argumentValues[i] = arg
	}

	return argumentValues, nil
}

// ParseAndCheckProgram parses the given code and checks it.
// Returns a program that can be interpreted (AST + elaboration).
//
func (r *interpreterRuntime) ParseAndCheckProgram(code []byte, context Context) (*interpreter.Program, error) {
	context.InitializeCodesAndPrograms()

	runtimeStorage := newRuntimeStorage(context.Interface)

	var interpreterOptions []interpreter.Option
	var checkerOptions []sema.Option

	functions := r.standardLibraryFunctions(
		context,
		runtimeStorage,
		interpreterOptions,
		checkerOptions,
	)

	program, err := r.parseAndCheckProgram(
		code,
		context,
		functions,
		checkerOptions,
		true,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	return program, nil
}

func (r *interpreterRuntime) parseAndCheckProgram(
	code []byte,
	context Context,
	functions stdlib.StandardLibraryFunctions,
	checkerOptions []sema.Option,
	storeProgram bool,
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
		context.SetCode(context.Location, string(code))
	}

	// Parse

	var parse *ast.Program
	reportMetric(
		func() {
			parse, err = parser2.ParseProgram(string(code))
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

	elaboration, err := r.check(parse, context, functions, checkerOptions)
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
	functions stdlib.StandardLibraryFunctions,
	checkerOptions []sema.Option,
) (
	elaboration *sema.Elaboration,
	err error,
) {

	valueDeclarations := functions.ToSemaValueDeclarations()

	for _, predeclaredValue := range startContext.PredeclaredValues {
		valueDeclarations = append(valueDeclarations, predeclaredValue)
	}

	checker, err := sema.NewChecker(
		program,
		startContext.Location,
		append(
			[]sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
				sema.WithPredeclaredTypes(typeDeclarations),
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
					func(checker *sema.Checker, location common.Location) (sema.Import, error) {
						var elaboration *sema.Elaboration
						switch location {
						case stdlib.CryptoChecker.Location:
							elaboration = stdlib.CryptoChecker.Elaboration

						default:
							context := startContext.WithLocation(location)

							program, err := r.getProgram(context, functions, checkerOptions)
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
				sema.WithCheckHandler(func(location common.Location, check func()) {
					reportMetric(
						check,
						startContext.Interface,
						func(metrics Metrics, duration time.Duration) {
							metrics.ProgramChecked(location, duration)
						},
					)
				}),
			},
			checkerOptions...,
		)...,
	)
	if err != nil {
		return nil, err
	}

	// TODO: set elaboration *before* checking,
	// so it is returned when there is a cyclic import

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
	functions stdlib.StandardLibraryFunctions,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) (*interpreter.Interpreter, error) {

	values := functions.ToInterpreterValueDeclarations()

	for _, predeclaredValue := range context.PredeclaredValues {
		values = append(values, predeclaredValue)
	}

	defaultOptions := []interpreter.Option{
		interpreter.WithPredeclaredValues(values),
		interpreter.WithOnEventEmittedHandler(
			func(
				inter *interpreter.Interpreter,
				eventValue *interpreter.CompositeValue,
				eventType *sema.CompositeType,
			) error {
				return r.emitEvent(inter, context.Interface, eventValue, eventType)
			},
		),
		interpreter.WithStorageKeyHandler(
			func(_ *interpreter.Interpreter, _ common.Address, indexingType sema.Type) string {
				return string(indexingType.ID())
			},
		),
		interpreter.WithInjectedCompositeFieldsHandler(
			r.injectedCompositeFieldsHandler(context, runtimeStorage, interpreterOptions, checkerOptions),
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
				constructor interpreter.FunctionValue,
				invocationRange ast.Range,
			) *interpreter.CompositeValue {

				return r.loadContract(
					inter,
					compositeType,
					constructor,
					invocationRange,
					context.Interface,
					runtimeStorage,
				)
			},
		),
		interpreter.WithImportLocationHandler(
			r.importLocationHandler(context, functions, checkerOptions),
		),
		interpreter.WithOnStatementHandler(
			r.onStatementHandler(),
		),
	}

	defaultOptions = append(defaultOptions,
		r.storageInterpreterOptions(runtimeStorage)...,
	)

	defaultOptions = append(defaultOptions,
		r.meteringInterpreterOptions(context.Interface)...,
	)

	return interpreter.NewInterpreter(
		program,
		context.Location,
		append(
			defaultOptions,
			interpreterOptions...,
		)...,
	)
}

func (r *interpreterRuntime) importLocationHandler(
	startContext Context,
	functions stdlib.StandardLibraryFunctions,
	checkerOptions []sema.Option,
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

			program, err := r.getProgram(context, functions, checkerOptions)
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
	functions stdlib.StandardLibraryFunctions,
	checkerOptions []sema.Option,
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
			functions,
			checkerOptions,
			true,
		)
		if err != nil {
			return nil, err
		}
	}

	context.SetProgram(context.Location, program.Program)

	return program, nil
}

func (r *interpreterRuntime) injectedCompositeFieldsHandler(
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) interpreter.InjectedCompositeFieldsHandlerFunc {
	return func(
		_ *interpreter.Interpreter,
		location Location,
		_ string,
		compositeKind common.CompositeKind,
	) *interpreter.StringValueOrderedMap {

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
					panic(runtimeErrors.NewUnreachableError())
				}

				addressValue := interpreter.NewAddressValue(address)

				injectedMembers := interpreter.NewStringValueOrderedMap()
				injectedMembers.Set(
					"account",
					r.newAuthAccountValue(
						addressValue,
						context,
						runtimeStorage,
						interpreterOptions,
						checkerOptions,
					),
				)
				return injectedMembers
			}
		}

		return nil
	}
}

func (r *interpreterRuntime) storageInterpreterOptions(runtimeStorage *runtimeStorage) []interpreter.Option {
	return []interpreter.Option{
		interpreter.WithStorageExistenceHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string) bool {
				return runtimeStorage.valueExists(address, key)
			},
		),
		interpreter.WithStorageReadHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string, deferred bool) interpreter.OptionalValue {
				return runtimeStorage.readValue(address, key, deferred)
			},
		),
		interpreter.WithStorageWriteHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string, value interpreter.OptionalValue) {
				runtimeStorage.writeValue(address, key, value)
			},
		),
	}
}

func (r *interpreterRuntime) meteringInterpreterOptions(runtimeInterface Interface) []interpreter.Option {
	var limit uint64
	wrapPanic(func() {
		limit = runtimeInterface.GetComputationLimit()
	})
	if limit == 0 {
		return nil
	}

	if limit == math.MaxUint64 {
		limit--
	}

	var used uint64

	checkLimit := func() {
		used++

		if used <= limit {
			return
		}

		var err error
		wrapPanic(func() {
			err = runtimeInterface.SetComputationUsed(used)
		})
		if err != nil {
			panic(err)
		}
		panic(ComputationLimitExceededError{
			Limit: limit,
		})
	}

	return []interpreter.Option{
		interpreter.WithOnStatementHandler(
			func(_ *interpreter.Statement) {
				checkLimit()
			},
		),
		interpreter.WithOnLoopIterationHandler(
			func(_ *interpreter.Interpreter, _ int) {
				checkLimit()
			},
		),
		interpreter.WithOnFunctionInvocationHandler(
			func(_ *interpreter.Interpreter, _ int) {
				checkLimit()
			},
		),
	}
}

func (r *interpreterRuntime) standardLibraryFunctions(
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) stdlib.StandardLibraryFunctions {
	return append(
		stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{
			CreateAccount:   r.newCreateAccountFunction(context, runtimeStorage, interpreterOptions, checkerOptions),
			GetAccount:      r.newGetAccountFunction(context.Interface, runtimeStorage),
			Log:             r.newLogFunction(context.Interface),
			GetCurrentBlock: r.newGetCurrentBlockFunction(context.Interface),
			GetBlock:        r.newGetBlockFunction(context.Interface),
			UnsafeRandom:    r.newUnsafeRandomFunction(context.Interface),
		}),
		stdlib.BuiltinFunctions...,
	)
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

// emitEvent converts an event value to native Go types and emits it to the runtime interface.
func (r *interpreterRuntime) emitEvent(
	inter *interpreter.Interpreter,
	runtimeInterface Interface,
	event *interpreter.CompositeValue,
	eventType *sema.CompositeType,
) error {
	fields := make([]exportableValue, len(eventType.ConstructorParameters))

	for i, parameter := range eventType.ConstructorParameters {
		value, _ := event.Fields.Get(parameter.Identifier)
		fields[i] = newExportableValue(value, inter)
	}

	eventValue := exportableEvent{
		Type:   eventType,
		Fields: fields,
	}

	var err error
	exportedEvent := exportEvent(eventValue)
	wrapPanic(func() {
		err = runtimeInterface.EmitEvent(exportedEvent)
	})
	return err
}

func (r *interpreterRuntime) emitAccountEvent(
	eventType *sema.CompositeType,
	runtimeInterface Interface,
	eventFields []exportableValue,
) {
	eventValue := exportableEvent{
		Type:   eventType,
		Fields: eventFields,
	}

	actualLen := len(eventFields)
	expectedLen := len(eventType.ConstructorParameters)

	if actualLen != expectedLen {
		panic(fmt.Errorf(
			"event emission value mismatch: event %s: expected %d, got %d",
			eventType.QualifiedString(),
			expectedLen,
			actualLen,
		))
	}

	var err error
	exportedEvent := exportEvent(eventValue)
	wrapPanic(func() {
		err = runtimeInterface.EmitEvent(exportedEvent)
	})
	if err != nil {
		panic(err)
	}
}

func CodeToHashValue(code []byte) *interpreter.ArrayValue {
	codeHash := sha3.Sum256(code)
	return interpreter.ByteSliceToByteArrayValue(codeHash[:])
}

func (r *interpreterRuntime) newCreateAccountFunction(
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {

		payer, ok := invocation.Arguments[0].(interpreter.AuthAccountValue)
		if !ok {
			panic(fmt.Sprintf(
				"%[1]s requires the first argument (payer) to be an %[1]s",
				sema.AuthAccountType,
			))
		}

		var address Address
		var err error
		wrapPanic(func() {
			address, err = context.Interface.CreateAccount(payer.AddressValue().ToAddress())
		})
		if err != nil {
			panic(err)
		}

		addressValue := interpreter.NewAddressValue(address)

		r.emitAccountEvent(
			stdlib.AccountCreatedEventType,
			context.Interface,
			[]exportableValue{
				newExportableValue(addressValue, nil),
			},
		)

		account := r.newAuthAccountValue(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
		)

		return trampoline.Done{Result: account}
	}
}
func storageUsedGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *runtimeStorage,
) func(inter *interpreter.Interpreter) interpreter.UInt64Value {
	address := addressValue.ToAddress()
	return func(inter *interpreter.Interpreter) interpreter.UInt64Value {

		// NOTE: flush the cached values, so the host environment
		// can properly calculate the amount of storage used by the account
		runtimeStorage.writeCached(inter)

		var capacity uint64
		var err error
		wrapPanic(func() {
			capacity, err = runtimeInterface.GetStorageUsed(address)
		})
		if err != nil {
			panic(err)
		}
		return interpreter.UInt64Value(capacity)
	}
}

func storageCapacityGetFunction(addressValue interpreter.AddressValue, runtimeInterface Interface) func() interpreter.UInt64Value {
	address := addressValue.ToAddress()
	return func() interpreter.UInt64Value {
		var capacity uint64
		var err error
		wrapPanic(func() {
			capacity, err = runtimeInterface.GetStorageCapacity(address)
		})
		if err != nil {
			panic(err)
		}
		return interpreter.UInt64Value(capacity)
	}
}

func (r *interpreterRuntime) newAddPublicKeyFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			publicKeyValue := invocation.Arguments[0].(*interpreter.ArrayValue)

			publicKey, err := interpreter.ByteArrayValueToByteSlice(publicKeyValue)
			if err != nil {
				panic("addPublicKey requires the first argument to be a byte array")
			}

			wrapPanic(func() {
				err = runtimeInterface.AddAccountKey(addressValue.ToAddress(), publicKey)
			})
			if err != nil {
				panic(err)
			}

			r.emitAccountEvent(
				stdlib.AccountKeyAddedEventType,
				runtimeInterface,
				[]exportableValue{
					newExportableValue(addressValue, nil),
					newExportableValue(publicKeyValue, nil),
				},
			)

			result := interpreter.VoidValue{}
			return trampoline.Done{Result: result}
		},
	)
}

func (r *interpreterRuntime) newRemovePublicKeyFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			index := invocation.Arguments[0].(interpreter.IntValue)

			var publicKey []byte
			var err error
			wrapPanic(func() {
				publicKey, err = runtimeInterface.RemoveAccountKey(addressValue.ToAddress(), index.ToInt())
			})
			if err != nil {
				panic(err)
			}

			publicKeyValue := interpreter.ByteSliceToByteArrayValue(publicKey)

			r.emitAccountEvent(
				stdlib.AccountKeyRemovedEventType,
				runtimeInterface,
				[]exportableValue{
					newExportableValue(addressValue, nil),
					newExportableValue(publicKeyValue, nil),
				},
			)

			result := interpreter.VoidValue{}
			return trampoline.Done{Result: result}
		},
	)
}

// recordContractValue records the update of the given contract value.
// It is only recorded and only written at the end of the execution
//
func (r *interpreterRuntime) recordContractValue(
	runtimeStorage *runtimeStorage,
	addressValue interpreter.AddressValue,
	name string,
	contractValue interpreter.Value,
	exportedContractValue cadence.Value,
) {
	runtimeStorage.recordContractUpdate(
		addressValue.ToAddress(),
		formatContractKey(name),
		contractValue,
		exportedContractValue,
	)
}

func formatContractKey(name string) string {
	const contractKey = "contract"

	// \x1F = Information Separator One
	return fmt.Sprintf("%s\x1F%s", contractKey, name)
}

func (r *interpreterRuntime) loadContract(
	inter *interpreter.Interpreter,
	compositeType *sema.CompositeType,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
	runtimeInterface Interface,
	runtimeStorage *runtimeStorage,
) *interpreter.CompositeValue {

	switch compositeType.Location {
	case stdlib.CryptoChecker.Location:
		contract, err := stdlib.NewCryptoContract(
			inter,
			constructor,
			runtimeInterface,
			runtimeInterface,
			invocationRange,
		)
		if err != nil {
			panic(err)
		}
		return contract

	default:

		var storedValue interpreter.OptionalValue = interpreter.NilValue{}

		switch location := compositeType.Location.(type) {

		case common.AddressLocation:
			storedValue = runtimeStorage.readValue(
				location.Address,
				formatContractKey(location.Name),
				false,
			)
		}

		switch typedValue := storedValue.(type) {
		case *interpreter.SomeValue:
			return typedValue.Value.(*interpreter.CompositeValue)
		case interpreter.NilValue:
			panic("failed to load contract")
		default:
			panic(runtimeErrors.NewUnreachableError())
		}
	}
}

func (r *interpreterRuntime) instantiateContract(
	program *interpreter.Program,
	context Context,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	argumentTypes []sema.Type,
	runtimeStorage *runtimeStorage,
	functions stdlib.StandardLibraryFunctions,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) (
	interpreter.Value,
	cadence.Value,
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
		return nil, nil, fmt.Errorf(
			"invalid argument count, too few arguments: expected %d, got %d, next missing argument: `%s`",
			parameterCount, argumentCount,
			parameterTypes[argumentCount],
		)
	} else if argumentCount > parameterCount {
		return nil, nil, fmt.Errorf(
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
			return nil, nil, fmt.Errorf(
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

	allInterpreterOptions := append(
		interpreterOptions[:],
		interpreter.WithContractValueHandler(
			func(
				inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				constructor interpreter.FunctionValue,
				invocationRange ast.Range,
			) *interpreter.CompositeValue {

				// If the contract is the deployed contract, instantiate it using
				// the provided constructor and given arguments

				if common.LocationsMatch(compositeType.Location, contractType.Location) &&
					compositeType.Identifier == contractType.Identifier {

					value, err := inter.InvokeFunctionValue(
						constructor,
						constructorArguments,
						argumentTypes,
						parameterTypes,
						invocationRange,
					)
					if err != nil {
						panic(err)
					}

					return value.(*interpreter.CompositeValue)
				}

				// The contract is not the deployed contract, load it from storage
				return r.loadContract(
					inter,
					compositeType,
					constructor,
					invocationRange,
					context.Interface,
					runtimeStorage,
				)
			},
		),
	)

	_, interpeter, err := r.interpret(
		program,
		context,
		runtimeStorage,
		functions,
		allInterpreterOptions,
		checkerOptions,
		nil,
	)

	if err != nil {
		return nil, nil, err
	}

	contract = interpeter.Globals[contractType.Identifier].GetValue().(*interpreter.CompositeValue)

	var exportedContract cadence.Value
	if runtimeStorage.highLevelStorageEnabled {
		exportedContract = exportCompositeValue(contract, interpeter, exportResults{})
	}

	return contract, exportedContract, err
}

func (r *interpreterRuntime) newGetAccountFunction(runtimeInterface Interface, runtimeStorage *runtimeStorage) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		accountAddress := invocation.Arguments[0].(interpreter.AddressValue)
		publicAccount := interpreter.NewPublicAccountValue(
			accountAddress,
			storageUsedGetFunction(accountAddress, runtimeInterface, runtimeStorage),
			storageCapacityGetFunction(accountAddress, runtimeInterface),
		)
		return trampoline.Done{Result: publicAccount}
	}
}

func (r *interpreterRuntime) newLogFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		message := fmt.Sprint(invocation.Arguments[0])
		var err error
		wrapPanic(func() {
			err = runtimeInterface.ProgramLog(message)
		})
		if err != nil {
			panic(err)
		}
		result := interpreter.VoidValue{}
		return trampoline.Done{Result: result}
	}
}

func (r *interpreterRuntime) getCurrentBlockHeight(runtimeInterface Interface) (currentBlockHeight uint64, err error) {
	wrapPanic(func() {
		currentBlockHeight, err = runtimeInterface.GetCurrentBlockHeight()
	})
	return
}

func (r *interpreterRuntime) getBlockAtHeight(height uint64, runtimeInterface Interface) (*interpreter.BlockValue, error) {

	var block Block
	var exists bool
	var err error

	wrapPanic(func() {
		block, exists, err = runtimeInterface.GetBlockAtHeight(height)
	})

	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	blockValue := NewBlockValue(block)
	return &blockValue, nil
}

func (r *interpreterRuntime) newGetCurrentBlockFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		var height uint64
		var err error
		wrapPanic(func() {
			height, err = r.getCurrentBlockHeight(runtimeInterface)
		})
		if err != nil {
			panic(err)
		}
		block, err := r.getBlockAtHeight(height, runtimeInterface)
		if err != nil {
			panic(err)
		}
		return trampoline.Done{Result: *block}
	}
}

func (r *interpreterRuntime) newGetBlockFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		height := uint64(invocation.Arguments[0].(interpreter.UInt64Value))
		block, err := r.getBlockAtHeight(height, runtimeInterface)
		if err != nil {
			panic(err)
		}
		var result interpreter.Value
		if block == nil {
			result = interpreter.NilValue{}
		} else {
			result = interpreter.NewSomeValueOwningNonCopying(*block)
		}
		return trampoline.Done{Result: result}
	}
}

func (r *interpreterRuntime) newUnsafeRandomFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		var rand uint64
		var err error
		wrapPanic(func() {
			rand, err = runtimeInterface.UnsafeRandom()
		})
		if err != nil {
			panic(err)
		}
		return trampoline.Done{Result: interpreter.UInt64Value(rand)}
	}
}

func (r *interpreterRuntime) newAuthAccountContracts(
	addressValue interpreter.AddressValue,
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) interpreter.AuthAccountContractsValue {
	return interpreter.AuthAccountContractsValue{
		Address: addressValue,
		AddFunction: r.newAuthAccountContractsChangeFunction(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
			false,
		),
		UpdateFunction: r.newAuthAccountContractsChangeFunction(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
			true,
		),
		GetFunction: r.newAuthAccountContractsGetFunction(
			addressValue,
			context.Interface,
		),
		RemoveFunction: r.newAuthAccountContractsRemoveFunction(
			addressValue,
			context.Interface,
			runtimeStorage,
		),
	}
}

// newAuthAccountContractsChangeFunction called when e.g.
// - adding: `AuthAccount.contracts.add(name: "Foo", code: [...])` (isUpdate = false)
// - updating: `AuthAccount.contracts.update__experimental(name: "Foo", code: [...])` (isUpdate = true)
//
func (r *interpreterRuntime) newAuthAccountContractsChangeFunction(
	addressValue interpreter.AddressValue,
	startContext Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
	isUpdate bool,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {

			const requiredArgumentCount = 2

			nameValue := invocation.Arguments[0].(*interpreter.StringValue)
			newCodeValue := invocation.Arguments[1].(*interpreter.ArrayValue)

			constructorArguments := invocation.Arguments[requiredArgumentCount:]
			constructorArgumentTypes := invocation.ArgumentTypes[requiredArgumentCount:]

			code, err := interpreter.ByteArrayValueToByteSlice(newCodeValue)
			if err != nil {
				panic("add requires the second argument to be an array")
			}

			// Get the existing code

			nameArgument := nameValue.Str

			if nameArgument == "" {
				panic(errors.New(
					"contract name argument cannot be empty." +
						"it must match the name of the deployed contract declaration or contract interface declaration",
				))
			}

			address := addressValue.ToAddress()
			existingCode, err := startContext.Interface.GetAccountContractCode(address, nameArgument)
			if err != nil {
				panic(err)
			}

			if isUpdate {
				// We are updating an existing contract.
				// Ensure that there's a contract/contract-interface with the given name exists already

				if len(existingCode) == 0 {
					panic(fmt.Errorf(
						"cannot update non-existing contract with name %q in account %s",
						nameArgument,
						address.ShortHexWithPrefix(),
					))
				}

			} else {
				// We are adding a new contract.
				// Ensure that no contract/contract interface with the given name exists already

				if len(existingCode) > 0 {
					panic(fmt.Errorf(
						"cannot overwrite existing contract with name %q in account %s",
						nameArgument,
						address.ShortHexWithPrefix(),
					))
				}
			}

			// Check the code

			location := common.AddressLocation{
				Address: address,
				Name:    nameArgument,
			}

			context := startContext.WithLocation(location)

			functions := r.standardLibraryFunctions(
				context,
				runtimeStorage,
				interpreterOptions,
				checkerOptions,
			)

			handleContractUpdateError := func(err error) {
				if err == nil {
					return
				}

				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(&InvalidContractDeploymentError{
					Err:           err,
					LocationRange: invocation.LocationRange,
				})
			}

			var cachedProgram *interpreter.Program
			if isUpdate {
				// Get the old program from host environment, if available. This is an optimization
				// so that old program doesn't need to be re-parsed for update validation.
				wrapPanic(func() {
					cachedProgram, err = context.Interface.GetProgram(context.Location)
				})
				handleContractUpdateError(err)
			}

			// NOTE: do NOT use the program obtained from the host environment, as the current program.
			// Always re-parse and re-check the new program.

			// NOTE: *DO NOT* store the program – the new or updated program
			// should not be effective during the execution

			const storeProgram = false

			program, err := r.parseAndCheckProgram(
				code,
				context,
				functions,
				checkerOptions,
				storeProgram,
			)
			if err != nil {
				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(&InvalidContractDeploymentError{
					Err:           err,
					LocationRange: invocation.LocationRange,
				})
			}

			// The code may declare exactly one contract or one contract interface.

			var contractTypes []*sema.CompositeType
			var contractInterfaceTypes []*sema.InterfaceType

			program.Elaboration.GlobalTypes.Foreach(func(_ string, variable *sema.Variable) {
				switch ty := variable.Type.(type) {
				case *sema.CompositeType:
					if ty.Kind == common.CompositeKindContract {
						contractTypes = append(contractTypes, ty)
					}

				case *sema.InterfaceType:
					if ty.CompositeKind == common.CompositeKindContract {
						contractInterfaceTypes = append(contractInterfaceTypes, ty)
					}
				}
			})

			var deployedType sema.Type
			var contractType *sema.CompositeType
			var contractInterfaceType *sema.InterfaceType
			var declaredName string
			var declarationKind common.DeclarationKind

			switch {
			case len(contractTypes) == 1 && len(contractInterfaceTypes) == 0:
				contractType = contractTypes[0]
				declaredName = contractType.Identifier
				deployedType = contractType
				declarationKind = common.DeclarationKindContract
			case len(contractInterfaceTypes) == 1 && len(contractTypes) == 0:
				contractInterfaceType = contractInterfaceTypes[0]
				declaredName = contractInterfaceType.Identifier
				deployedType = contractInterfaceType
				declarationKind = common.DeclarationKindContractInterface
			}

			if deployedType == nil {
				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(fmt.Errorf(
					"invalid %s: the code must declare exactly one contract or contract interface",
					declarationKind.Name(),
				))
			}

			// The declared contract or contract interface must have the name
			// passed to the constructor as the first argument

			if declaredName != nameArgument {
				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(fmt.Errorf(
					"invalid %s: the name argument must match the name of the declaration"+
						"name argument: %q, declaration name: %q",
					declarationKind.Name(),
					nameArgument,
					declaredName,
				))
			}

			// Validate the contract update (if enabled)

			if r.contractUpdateValidationEnabled && isUpdate {
				var oldProgram *ast.Program
				if cachedProgram != nil {
					oldProgram = cachedProgram.Program
				} else {
					oldProgram, err = parser2.ParseProgram(string(existingCode))
					handleContractUpdateError(err)
				}

				validator := NewContractUpdateValidator(
					context.Location,
					nameArgument,
					oldProgram,
					program.Program,
				)
				err = validator.Validate()
				handleContractUpdateError(err)
			}

			err = r.updateAccountContractCode(
				program,
				context,
				runtimeStorage,
				declaredName,
				code,
				addressValue,
				contractType,
				constructorArguments,
				constructorArgumentTypes,
				interpreterOptions,
				checkerOptions,
				updateAccountContractCodeOptions{
					createContract: !isUpdate,
				},
			)
			if err != nil {
				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(err)
			}

			codeHashValue := CodeToHashValue(code)

			eventArguments := []exportableValue{
				newExportableValue(addressValue, nil),
				newExportableValue(codeHashValue, nil),
				newExportableValue(nameValue, nil),
			}

			if isUpdate {
				r.emitAccountEvent(
					stdlib.AccountContractUpdatedEventType,
					startContext.Interface,
					eventArguments,
				)
			} else {
				r.emitAccountEvent(
					stdlib.AccountContractAddedEventType,
					startContext.Interface,
					eventArguments,
				)
			}

			result := interpreter.DeployedContractValue{
				Address: addressValue,
				Name:    nameValue,
				Code:    newCodeValue,
			}

			return trampoline.Done{Result: result}
		},
	)
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
	runtimeStorage *runtimeStorage,
	name string,
	code []byte,
	addressValue interpreter.AddressValue,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	constructorArgumentTypes []sema.Type,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
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

	var contractValue interpreter.Value
	var exportedContractValue cadence.Value

	createContract := contractType != nil && options.createContract

	address := addressValue.ToAddress()

	var err error

	if createContract {

		functions := r.standardLibraryFunctions(context, runtimeStorage, interpreterOptions, checkerOptions)

		contractValue, exportedContractValue, err = r.instantiateContract(
			program,
			context,
			contractType,
			constructorArguments,
			constructorArgumentTypes,
			runtimeStorage,
			functions,
			interpreterOptions,
			checkerOptions,
		)

		if err != nil {
			return err
		}

		contractValue.SetOwner(&address)
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

		r.recordContractValue(
			runtimeStorage,
			addressValue,
			name,
			contractValue,
			exportedContractValue,
		)
	}

	return nil
}

func (r *interpreterRuntime) newAuthAccountContractsGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {

			nameValue := invocation.Arguments[0].(*interpreter.StringValue)

			address := addressValue.ToAddress()
			nameArgument := nameValue.Str
			var code []byte
			var err error
			wrapPanic(func() {
				code, err = runtimeInterface.GetAccountContractCode(address, nameArgument)
			})
			if err != nil {
				panic(err)
			}

			var result interpreter.OptionalValue = interpreter.NilValue{}

			if len(code) > 0 {
				result = interpreter.NewSomeValueOwningNonCopying(
					interpreter.DeployedContractValue{
						Address: addressValue,
						Name:    nameValue,
						Code:    interpreter.ByteSliceToByteArrayValue(code),
					},
				)
			}

			return trampoline.Done{Result: result}
		},
	)
}

func (r *interpreterRuntime) newAuthAccountContractsRemoveFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *runtimeStorage,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {

			nameValue := invocation.Arguments[0].(*interpreter.StringValue)

			address := addressValue.ToAddress()
			nameArgument := nameValue.Str

			// Get the current code

			var code []byte
			var err error
			wrapPanic(func() {
				code, err = runtimeInterface.GetAccountContractCode(address, nameArgument)
			})
			if err != nil {
				panic(err)
			}

			// Only remove the contract code, remove the contract value, and emit an event,
			// if there is currently code deployed for the given contract name

			var result interpreter.OptionalValue = interpreter.NilValue{}

			if len(code) > 0 {

				// NOTE: *DO NOT* call SetProgram – the program removal
				// should not be effective during the execution, only after

				wrapPanic(func() {
					err = runtimeInterface.RemoveAccountContractCode(address, nameArgument)
				})
				if err != nil {
					panic(err)
				}

				// NOTE: the contract recording function delays the write
				// until the end of the execution of the program

				r.recordContractValue(
					runtimeStorage,
					addressValue,
					nameArgument,
					nil,
					nil,
				)

				codeHashValue := CodeToHashValue(code)

				r.emitAccountEvent(
					stdlib.AccountContractRemovedEventType,
					runtimeInterface,
					[]exportableValue{
						newExportableValue(addressValue, nil),
						newExportableValue(codeHashValue, nil),
						newExportableValue(nameValue, nil),
					},
				)

				result = interpreter.NewSomeValueOwningNonCopying(
					interpreter.DeployedContractValue{
						Address: addressValue,
						Name:    nameValue,
						Code:    interpreter.ByteSliceToByteArrayValue(code),
					},
				)
			}

			return trampoline.Done{Result: result}
		},
	)
}

func (r *interpreterRuntime) onStatementHandler() interpreter.OnStatementFunc {
	if r.coverageReport == nil {
		return nil
	}

	return func(statement *interpreter.Statement) {
		location := statement.Interpreter.Location
		line := statement.Statement.StartPosition().Line
		r.coverageReport.AddLineHit(location, line)
	}
}

func NewBlockValue(block Block) interpreter.BlockValue {

	// height
	heightValue := interpreter.UInt64Value(block.Height)

	// view
	viewValue := interpreter.UInt64Value(block.View)

	// ID
	var values = make([]interpreter.Value, sema.BlockIDSize)
	for i, b := range block.Hash {
		values[i] = interpreter.UInt8Value(b)
	}
	idValue := &interpreter.ArrayValue{Values: values}

	// timestamp
	// TODO: verify
	timestampValue := interpreter.NewUFix64ValueWithInteger(uint64(time.Unix(0, block.Timestamp).Unix()))

	return interpreter.BlockValue{
		Height:    heightValue,
		View:      viewValue,
		ID:        idValue,
		Timestamp: timestampValue,
	}
}
