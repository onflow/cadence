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
)

type Script struct {
	Source    []byte
	Arguments [][]byte
}

type importResolutionResults map[common.LocationID]bool

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

	// SetContractUpdateValidationEnabled configures if contract update validation is enabled.
	//
	SetContractUpdateValidationEnabled(enabled bool)

	// ReadStored reads the value stored at the given path
	//
	ReadStored(address common.Address, path cadence.Path, context Context) (cadence.Value, error)

	// ReadLinked dereferences the path and returns the value stored at the target
	//
	ReadLinked(address common.Address, path cadence.Path, context Context) (cadence.Value, error)
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
		stdlib.BuiltinValues(),
		checkerOptions,
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
	)

	value, inter, err := r.interpret(
		program,
		context,
		runtimeStorage,
		functions,
		stdlib.BuiltinValues(),
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

	err = runtimeStorage.writeCached(inter)
	if err != nil {
		return nil, newError(err, context)
	}

	return exportValue(value)
}

type interpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
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
	values stdlib.StandardLibraryValues,
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
		values,
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

	if inter.ExitHandler != nil {
		err = inter.ExitHandler()
	}
	return exportedValue, inter, err
}

func (r *interpreterRuntime) newAuthAccountValue(
	addressValue interpreter.AddressValue,
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) *interpreter.CompositeValue {
	return interpreter.NewAuthAccountValue(
		addressValue,
		accountBalanceGetFunction(addressValue, context.Interface),
		accountAvailableBalanceGetFunction(addressValue, context.Interface),
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
		r.newAuthAccountKeys(addressValue, context.Interface),
	)
}

func (r *interpreterRuntime) InvokeContractFunction(
	contractLocation common.AddressLocation,
	functionName string,
	arguments []interpreter.Value,
	argumentTypes []sema.Type,
	context Context,
) (cadence.Value, error) {
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

	// create interpreter
	_, inter, err := r.interpret(
		nil,
		context,
		runtimeStorage,
		functions,
		stdlib.BuiltinValues(),
		interpreterOptions,
		checkerOptions,
		nil,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	// ensure the contract is loaded
	inter = inter.EnsureLoaded(contractLocation)

	for i, argumentType := range argumentTypes {
		arguments[i] = r.convertArgument(
			arguments[i],
			argumentType,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
		)
	}

	contractValue, err := inter.GetContractComposite(contractLocation)
	if err != nil {
		return nil, newError(err, context)
	}

	// prepare invocation
	invocation := interpreter.Invocation{
		Self:               contractValue,
		Arguments:          arguments,
		ArgumentTypes:      argumentTypes,
		TypeParameterTypes: nil,
		GetLocationRange: func() interpreter.LocationRange {
			return interpreter.LocationRange{
				Location: context.Location,
			}
		},
		Interpreter: inter,
	}

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
	err = runtimeStorage.writeCached(inter)
	if err != nil {
		return nil, newError(err, context)
	}

	return ExportValue(value, inter)
}

func (r *interpreterRuntime) convertArgument(
	argument interpreter.Value,
	argumentType sema.Type,
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) interpreter.Value {
	switch argumentType {
	case sema.AuthAccountType:
		// convert addresses to auth accounts so there is no need to construct an auth account value for the caller
		if addressValue, ok := argument.(interpreter.AddressValue); ok {
			return r.newAuthAccountValue(
				interpreter.NewAddressValue(addressValue.ToAddress()),
				context,
				runtimeStorage,
				interpreterOptions,
				checkerOptions,
			)
		}
	}
	return argument
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
		stdlib.BuiltinValues(),
		checkerOptions,
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
		stdlib.BuiltinValues(),
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
	err = runtimeStorage.writeCached(inter)
	if err != nil {
		return newError(err, context)
	}

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
			arguments,
			parameters,
		)
		if err != nil {
			return nil, err
		}

		values = append(values, authorizerValues...)
		err = inter.InvokeTransaction(0, values...)
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

		arg, err := importValue(inter, value, parameterType)
		if err != nil {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err:   err,
			}
		}

		dynamicType := arg.DynamicType(inter, interpreter.SeenReferences{})

		// Ensure the argument is of an importable type
		if !dynamicType.IsImportable() {
			return nil, &ArgumentNotImportableError{
				Type: dynamicType,
			}
		}

		// Check that decoded value is a subtype of static parameter type
		if !inter.IsSubType(dynamicType, parameterType) {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err: &InvalidValueTypeError{
					ExpectedType: parameterType,
				},
			}
		}

		// Check whether the decoded value conforms to the type associated with the value
		conformanceResults := interpreter.TypeConformanceResults{}
		if !arg.ConformsToDynamicType(inter, dynamicType, conformanceResults) {
			return nil, &InvalidEntryPointArgumentError{
				Index: i,
				Err: &MalformedValueError{
					ExpectedType: parameterType,
				},
			}
		}

		// Ensure static type info is available for all values
		interpreter.InspectValue(arg, func(value interpreter.Value) bool {
			if value == nil {
				return true
			}

			if !hasValidStaticType(value) {
				panic(fmt.Errorf("invalid static type for argument: %d", i))
			}

			return true
		})

		argumentValues[i] = arg
	}

	return argumentValues, nil
}

func hasValidStaticType(value interpreter.Value) bool {
	switch value := value.(type) {
	case *interpreter.ArrayValue:
		switch value.StaticType().(type) {
		case interpreter.ConstantSizedStaticType, interpreter.VariableSizedStaticType:
			return true
		default:
			return false
		}
	case *interpreter.DictionaryValue:
		dictionaryType, ok := value.StaticType().(interpreter.DictionaryStaticType)
		if !ok {
			return false
		}

		return dictionaryType.KeyType != nil &&
			dictionaryType.ValueType != nil
	default:
		// For other values, static type is NOT inferred.
		// Hence no need to validate it here.
		return value.StaticType() != nil
	}
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
		stdlib.BuiltinValues(),
		checkerOptions,
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
	functions stdlib.StandardLibraryFunctions,
	values stdlib.StandardLibraryValues,
	checkerOptions []sema.Option,
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

	elaboration, err := r.check(parse, context, functions, values, checkerOptions, checkedImports)
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
	values stdlib.StandardLibraryValues,
	checkerOptions []sema.Option,
	checkedImports importResolutionResults,
) (
	elaboration *sema.Elaboration,
	err error,
) {

	valueDeclarations := functions.ToSemaValueDeclarations()
	valueDeclarations = append(valueDeclarations, values.ToSemaValueDeclarations()...)

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
					func(checker *sema.Checker, importedLocation common.Location, importRange ast.Range) (sema.Import, error) {

						var elaboration *sema.Elaboration
						switch importedLocation {
						case stdlib.CryptoChecker.Location:
							elaboration = stdlib.CryptoChecker.Elaboration

						default:
							context := startContext.WithLocation(importedLocation)

							// Check for cyclic imports
							if checkedImports[importedLocation.ID()] {
								return nil, &sema.CyclicImportsError{
									Location: importedLocation,
									Range:    importRange,
								}
							} else {
								checkedImports[importedLocation.ID()] = true
								defer delete(checkedImports, importedLocation.ID())
							}

							program, err := r.getProgram(context, functions, values, checkerOptions, checkedImports)
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
	values stdlib.StandardLibraryValues,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) (*interpreter.Interpreter, error) {

	preDeclaredValues := functions.ToInterpreterValueDeclarations()
	preDeclaredValues = append(preDeclaredValues, values.ToInterpreterValueDeclarations()...)

	for _, predeclaredValue := range context.PredeclaredValues {
		preDeclaredValues = append(preDeclaredValues, predeclaredValue)
	}

	defaultOptions := []interpreter.Option{
		interpreter.WithPredeclaredValues(preDeclaredValues),
		interpreter.WithOnEventEmittedHandler(
			func(
				inter *interpreter.Interpreter,
				eventValue *interpreter.CompositeValue,
				eventType *sema.CompositeType,
			) error {
				return r.emitEvent(inter, context.Interface, eventValue, eventType)
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
					runtimeStorage,
				)
			},
		),
		interpreter.WithImportLocationHandler(
			r.importLocationHandler(context, functions, values, checkerOptions),
		),
		interpreter.WithOnStatementHandler(
			r.onStatementHandler(),
		),
		interpreter.WithAccountHandlerFunc(
			func(address interpreter.AddressValue) *interpreter.CompositeValue {
				return r.getPublicAccount(address, context.Interface, runtimeStorage)
			},
		),
		interpreter.WithPublicKeyValidationHandler(
			func(publicKey *interpreter.CompositeValue) interpreter.BoolValue {
				return validatePublicKey(publicKey, context.Interface)
			},
		),
		interpreter.WithSignatureVerificationHandler(
			func(
				signature *interpreter.ArrayValue,
				signedData *interpreter.ArrayValue,
				domainSeparationTag *interpreter.StringValue,
				hashAlgorithm *interpreter.CompositeValue,
				publicKey *interpreter.CompositeValue,
			) interpreter.BoolValue {
				return verifySignature(
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
				data *interpreter.ArrayValue,
				tag *interpreter.StringValue,
				hashAlgorithm *interpreter.CompositeValue,
			) *interpreter.ArrayValue {
				return hash(
					data,
					tag,
					hashAlgorithm,
					context.Interface,
				)
			},
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
	values stdlib.StandardLibraryValues,
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

			program, err := r.getProgram(context, functions, values, checkerOptions, importResolutionResults{})
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
	values stdlib.StandardLibraryValues,
	checkerOptions []sema.Option,
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
			functions,
			values,
			checkerOptions,
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
	var computationLimit uint64
	wrapPanic(func() {
		computationLimit = runtimeInterface.GetComputationLimit()
	})
	if computationLimit == 0 {
		return nil
	}

	if computationLimit == math.MaxUint64 {
		computationLimit--
	}

	var computationUsed uint64

	checkComputationLimit := func(increase uint64) {
		computationUsed += increase

		if computationUsed <= computationLimit {
			return
		}

		var err error
		wrapPanic(func() {
			err = runtimeInterface.SetComputationUsed(computationUsed)
		})
		if err != nil {
			panic(err)
		}

		panic(ComputationLimitExceededError{
			Limit: computationLimit,
		})
	}

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
		interpreter.WithOnStatementHandler(
			func(_ *interpreter.Interpreter, _ ast.Statement) {
				checkComputationLimit(1)
			},
		),
		interpreter.WithOnLoopIterationHandler(
			func(_ *interpreter.Interpreter, _ int) {
				checkComputationLimit(1)
			},
		),
		interpreter.WithOnFunctionInvocationHandler(
			func(_ *interpreter.Interpreter, _ int) {
				callStackDepth++
				checkCallStackDepth()

				checkComputationLimit(1)
			},
		),
		interpreter.WithOnInvokedFunctionReturnHandler(
			func(_ *interpreter.Interpreter, _ int) {
				callStackDepth--
			},
		),
		interpreter.WithExitHandler(
			func() error {
				return runtimeInterface.SetComputationUsed(computationUsed)
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

	eventFields := event.Fields()
	for i, parameter := range eventType.ConstructorParameters {
		value, _ := eventFields.Get(parameter.Identifier)
		fields[i] = newExportableValue(value, inter)
	}

	eventValue := exportableEvent{
		Type:   eventType,
		Fields: fields,
	}

	exportedEvent, err := exportEvent(eventValue, seenReferences{})
	if err != nil {
		return err
	}
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

	exportedEvent, err := exportEvent(eventValue, seenReferences{})
	if err != nil {
		panic(err)
	}
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
	return func(invocation interpreter.Invocation) interpreter.Value {

		payer := invocation.Arguments[0].(*interpreter.CompositeValue)

		if payer.QualifiedIdentifier() != sema.AuthAccountType.QualifiedIdentifier() {
			panic(fmt.Sprintf(
				"%[1]s requires the first argument (payer) to be an %[1]s",
				sema.AuthAccountType,
			))
		}

		payerAddressValue, ok := payer.Fields().Get(sema.AuthAccountAddressField)
		if !ok {
			panic("address is not set")
		}

		payerAddress := payerAddressValue.(interpreter.AddressValue).ToAddress()

		var address Address
		var err error
		wrapPanic(func() {
			address, err = context.Interface.CreateAccount(payerAddress)
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

		return r.newAuthAccountValue(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
		)
	}
}

func accountBalanceGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) func() interpreter.UFix64Value {
	address := addressValue.ToAddress()
	return func() interpreter.UFix64Value {
		var balance uint64
		var err error
		wrapPanic(func() {
			balance, err = runtimeInterface.GetAccountBalance(address)
		})
		if err != nil {
			panic(err)
		}
		return interpreter.UFix64Value(balance)
	}
}

func accountAvailableBalanceGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) func() interpreter.UFix64Value {
	address := addressValue.ToAddress()
	return func() interpreter.UFix64Value {
		var balance uint64
		var err error
		wrapPanic(func() {
			balance, err = runtimeInterface.GetAccountAvailableBalance(address)
		})
		if err != nil {
			panic(err)
		}
		return interpreter.UFix64Value(balance)
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
		err := runtimeStorage.writeCached(inter)
		if err != nil {
			panic(err)
		}

		var capacity uint64
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
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			publicKeyValue := invocation.Arguments[0].(*interpreter.ArrayValue)

			publicKey, err := interpreter.ByteArrayValueToByteSlice(publicKeyValue)
			if err != nil {
				panic("addPublicKey requires the first argument to be a byte array")
			}

			wrapPanic(func() {
				err = runtimeInterface.AddEncodedAccountKey(addressValue.ToAddress(), publicKey)
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

			return interpreter.VoidValue{}
		},
	)
}

func (r *interpreterRuntime) newRemovePublicKeyFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			index := invocation.Arguments[0].(interpreter.IntValue)

			var publicKey []byte
			var err error
			wrapPanic(func() {
				publicKey, err = runtimeInterface.RevokeEncodedAccountKey(addressValue.ToAddress(), index.ToInt())
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

			return interpreter.VoidValue{}
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
) {
	runtimeStorage.recordContractUpdate(
		addressValue.ToAddress(),
		formatContractKey(name),
		contractValue,
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
	runtimeStorage *runtimeStorage,
) *interpreter.CompositeValue {

	switch compositeType.Location {
	case stdlib.CryptoChecker.Location:
		contract, err := stdlib.NewCryptoContract(
			inter,
			constructor,
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
	values stdlib.StandardLibraryValues,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) (
	interpreter.Value,
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
		return nil, fmt.Errorf(
			"invalid argument count, too few arguments: expected %d, got %d, next missing argument: `%s`",
			parameterCount, argumentCount,
			parameterTypes[argumentCount],
		)
	} else if argumentCount > parameterCount {
		return nil, fmt.Errorf(
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
			return nil, fmt.Errorf(
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

	allInterpreterOptions := interpreterOptions[:]

	allInterpreterOptions = append(
		allInterpreterOptions,
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
		values,
		allInterpreterOptions,
		checkerOptions,
		nil,
	)

	if err != nil {
		return nil, err
	}

	variable, ok := interpeter.Globals.Get(contractType.Identifier)
	if !ok {
		return nil, fmt.Errorf(
			"cannot find contract: `%s`",
			contractType.Identifier,
		)
	}

	contract = variable.GetValue().(*interpreter.CompositeValue)

	return contract, err
}

func (r *interpreterRuntime) newGetAccountFunction(runtimeInterface Interface, runtimeStorage *runtimeStorage) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) interpreter.Value {
		accountAddress := invocation.Arguments[0].(interpreter.AddressValue)
		return r.getPublicAccount(accountAddress, runtimeInterface, runtimeStorage)
	}
}

func (r *interpreterRuntime) getPublicAccount(
	accountAddress interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *runtimeStorage,
) *interpreter.CompositeValue {

	return interpreter.NewPublicAccountValue(
		accountAddress,
		accountBalanceGetFunction(accountAddress, runtimeInterface),
		accountAvailableBalanceGetFunction(accountAddress, runtimeInterface),
		storageUsedGetFunction(accountAddress, runtimeInterface, runtimeStorage),
		storageCapacityGetFunction(accountAddress, runtimeInterface),
		r.newPublicAccountKeys(accountAddress, runtimeInterface),
		r.newPublicAccountContracts(accountAddress, runtimeInterface),
	)
}

func (r *interpreterRuntime) newLogFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) interpreter.Value {
		message := invocation.Arguments[0].String()
		var err error
		wrapPanic(func() {
			err = runtimeInterface.ProgramLog(message)
		})
		if err != nil {
			panic(err)
		}
		return interpreter.VoidValue{}
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
	return func(invocation interpreter.Invocation) interpreter.Value {
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
		return *block
	}
}

func (r *interpreterRuntime) newGetBlockFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) interpreter.Value {
		height := uint64(invocation.Arguments[0].(interpreter.UInt64Value))
		block, err := r.getBlockAtHeight(height, runtimeInterface)
		if err != nil {
			panic(err)
		}

		if block == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewSomeValueOwningNonCopying(*block)
	}
}

func (r *interpreterRuntime) newUnsafeRandomFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) interpreter.Value {
		var rand uint64
		var err error
		wrapPanic(func() {
			rand, err = runtimeInterface.UnsafeRandom()
		})
		if err != nil {
			panic(err)
		}
		return interpreter.UInt64Value(rand)
	}
}

func (r *interpreterRuntime) newAuthAccountContracts(
	addressValue interpreter.AddressValue,
	context Context,
	runtimeStorage *runtimeStorage,
	interpreterOptions []interpreter.Option,
	checkerOptions []sema.Option,
) *interpreter.CompositeValue {
	return interpreter.NewAuthAccountContractsValue(
		addressValue,
		r.newAuthAccountContractsChangeFunction(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
			false,
		),
		r.newAuthAccountContractsChangeFunction(
			addressValue,
			context,
			runtimeStorage,
			interpreterOptions,
			checkerOptions,
			true,
		),
		r.newAccountContractsGetFunction(
			addressValue,
			context.Interface,
		),
		r.newAuthAccountContractsRemoveFunction(
			addressValue,
			context.Interface,
			runtimeStorage,
		),
		r.newAccountContractsGetNamesFunction(
			addressValue,
			context.Interface,
		),
	)
}

func (r *interpreterRuntime) newAuthAccountKeys(addressValue interpreter.AddressValue, runtimeInterface Interface) *interpreter.CompositeValue {
	return interpreter.NewAuthAccountKeysValue(
		r.newAccountKeysAddFunction(
			addressValue,
			runtimeInterface,
		),
		r.newAccountKeysGetFunction(
			addressValue,
			runtimeInterface,
		),
		r.newAccountKeysRevokeFunction(
			addressValue,
			runtimeInterface,
		),
	)
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
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {

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
					LocationRange: invocation.GetLocationRange(),
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
				stdlib.BuiltinValues(),
				checkerOptions,
				storeProgram,
				importResolutionResults{},
			)
			if err != nil {
				// Update the code for the error pretty printing
				// NOTE: only do this when an error occurs

				context.SetCode(context.Location, string(code))

				panic(&InvalidContractDeploymentError{
					Err:           err,
					LocationRange: invocation.GetLocationRange(),
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

			return interpreter.DeployedContractValue{
				Address: addressValue,
				Name:    nameValue,
				Code:    newCodeValue,
			}
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

	createContract := contractType != nil && options.createContract

	address := addressValue.ToAddress()

	var err error

	if createContract {

		functions := r.standardLibraryFunctions(context, runtimeStorage, interpreterOptions, checkerOptions)
		values := stdlib.BuiltinValues()

		contractValue, err = r.instantiateContract(
			program,
			context,
			contractType,
			constructorArguments,
			constructorArgumentTypes,
			runtimeStorage,
			functions,
			values,
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
		)
	}

	return nil
}

func (r *interpreterRuntime) newAccountContractsGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {

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

			if len(code) > 0 {
				return interpreter.NewSomeValueOwningNonCopying(
					interpreter.DeployedContractValue{
						Address: addressValue,
						Name:    nameValue,
						Code:    interpreter.ByteSliceToByteArrayValue(code),
					},
				)
			} else {
				return interpreter.NilValue{}
			}
		},
	)
}

func (r *interpreterRuntime) newAuthAccountContractsRemoveFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *runtimeStorage,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {

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

			if len(code) > 0 {

				// NOTE: *DO NOT* call SetProgram – the program removal
				// should not be effective during the execution, only after

				// Deny removing a contract, if the contract validation is enabled, and
				// the existing code contains enums.
				if r.contractUpdateValidationEnabled {

					existingProgram, err := parser2.ParseProgram(string(code))

					// If the existing code is not parsable (i.e: `err != nil`), that shouldn't be a reason to
					// fail the contract removal. Therefore, validate only if the code is a valid one.
					if err == nil && containsEnumsInProgram(existingProgram) {
						panic(&ContractRemovalError{
							Name:          nameArgument,
							LocationRange: invocation.GetLocationRange(),
						})
					}
				}

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

				return interpreter.NewSomeValueOwningNonCopying(
					interpreter.DeployedContractValue{
						Address: addressValue,
						Name:    nameValue,
						Code:    interpreter.ByteSliceToByteArrayValue(code),
					},
				)
			} else {
				return interpreter.NilValue{}
			}
		},
	)
}

func (r *interpreterRuntime) newAccountContractsGetNamesFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) func() *interpreter.ArrayValue {
	address := addressValue.ToAddress()

	return func() *interpreter.ArrayValue {
		var names []string
		var err error
		wrapPanic(func() {
			names, err = runtimeInterface.GetAccountContractNames(address)
		})
		if err != nil {
			panic(err)
		}

		values := make([]interpreter.Value, len(names))
		for i, name := range names {
			values[i] = interpreter.NewStringValue(name)
		}

		return interpreter.NewArrayValueUnownedNonCopying(
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeString,
			},
			values...,
		)
	}
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

func (r *interpreterRuntime) executeNonProgram(interpret interpretFunc, context Context) (cadence.Value, error) {
	context.InitializeCodesAndPrograms()

	var program *interpreter.Program

	runtimeStorage := newRuntimeStorage(context.Interface)

	var functions stdlib.StandardLibraryFunctions
	var values stdlib.StandardLibraryValues
	var interpreterOptions []interpreter.Option
	var checkerOptions []sema.Option

	value, _, err := r.interpret(
		program,
		context,
		runtimeStorage,
		functions,
		values,
		interpreterOptions,
		checkerOptions,
		interpret,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	return exportValue(value)
}

func (r *interpreterRuntime) ReadStored(address common.Address, path cadence.Path, context Context) (cadence.Value, error) {
	return r.executeNonProgram(
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			key := interpreter.StorageKey(importPathValue(path))
			value := inter.ReadStored(address, key, false)
			return value, nil
		},
		context,
	)
}

func (r *interpreterRuntime) ReadLinked(address common.Address, path cadence.Path, context Context) (cadence.Value, error) {
	return r.executeNonProgram(
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			key, _, err := inter.GetCapabilityFinalTargetStorageKey(
				address,
				importPathValue(path),
				&sema.ReferenceType{
					Type: sema.AnyType,
				},
				interpreter.ReturnEmptyLocationRange,
			)
			if err != nil {
				return nil, err
			}
			value := inter.ReadStored(address, key, false)
			return value, nil
		},
		context,
	)
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
	idValue := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.ByteArrayStaticType,
		values...,
	)

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

func (r *interpreterRuntime) newAccountKeysAddFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			publicKeyValue := invocation.Arguments[0].(*interpreter.CompositeValue)

			publicKey, err := NewPublicKeyFromValue(publicKeyValue)
			if err != nil {
				panic(err)
			}

			hashAlgo := NewHashAlgorithmFromValue(invocation.Arguments[1])
			address := addressValue.ToAddress()
			weight := invocation.Arguments[2].(interpreter.UFix64Value).ToInt()

			var accountKey *AccountKey
			wrapPanic(func() {
				accountKey, err = runtimeInterface.AddAccountKey(address, publicKey, hashAlgo, weight)
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

			return NewAccountKeyValue(
				accountKey,
				invocation.Interpreter.PublicKeyValidationHandler,
			)
		},
	)
}

func (r *interpreterRuntime) newAccountKeysGetFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			index := invocation.Arguments[0].(interpreter.IntValue).ToInt()
			address := addressValue.ToAddress()

			var err error
			var accountKey *AccountKey
			wrapPanic(func() {
				accountKey, err = runtimeInterface.GetAccountKey(address, index)
			})

			if err != nil {
				panic(err)
			}

			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
			// This is done because, if the host function returns an error when a key is not found, then
			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
			if accountKey == nil {
				return interpreter.NilValue{}
			}

			return interpreter.NewSomeValueOwningNonCopying(
				NewAccountKeyValue(
					accountKey,
					invocation.Interpreter.PublicKeyValidationHandler,
				),
			)
		},
	)
}

func (r *interpreterRuntime) newAccountKeysRevokeFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
) *interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			indexValue := invocation.Arguments[0].(interpreter.IntValue)
			index := indexValue.ToInt()
			address := addressValue.ToAddress()

			var err error
			var accountKey *AccountKey
			wrapPanic(func() {
				accountKey, err = runtimeInterface.RevokeAccountKey(address, index)
			})
			if err != nil {
				panic(err)
			}

			// Here it is expected the host function to return a nil key, if a key is not found at the given index.
			// This is done because, if the host function returns an error when a key is not found, then
			// currently there's no way to distinguish between a 'key not found error' vs other internal errors.
			if accountKey == nil {
				return interpreter.NilValue{}
			}

			r.emitAccountEvent(
				stdlib.AccountKeyRemovedEventType,
				runtimeInterface,
				[]exportableValue{
					newExportableValue(addressValue, nil),
					newExportableValue(indexValue, nil),
				},
			)

			return interpreter.NewSomeValueOwningNonCopying(
				NewAccountKeyValue(
					accountKey,
					invocation.Interpreter.PublicKeyValidationHandler,
				),
			)
		},
	)
}

func (r *interpreterRuntime) newPublicAccountKeys(addressValue interpreter.AddressValue, runtimeInterface Interface) *interpreter.CompositeValue {
	return interpreter.NewPublicAccountKeysValue(
		r.newAccountKeysGetFunction(
			addressValue,
			runtimeInterface,
		),
	)
}

func (r *interpreterRuntime) newPublicAccountContracts(
	addressValue interpreter.AddressValue,
	inter Interface,
) *interpreter.CompositeValue {
	return interpreter.NewPublicAccountContractsValue(
		addressValue,
		r.newAccountContractsGetFunction(
			addressValue,
			inter,
		),
		r.newAccountContractsGetNamesFunction(
			addressValue,
			inter,
		),
	)
}

func NewPublicKeyFromValue(publicKey *interpreter.CompositeValue) (*PublicKey, error) {

	fields := publicKey.Fields()

	// publicKey field
	publicKeyFieldGetter, ok := publicKey.ComputedFields.Get(sema.PublicKeyPublicKeyField)
	if !ok {
		return nil, fmt.Errorf("public key value is not set")
	}

	// Interpreter is not available at this point.
	// For now its safe to pass a nil interpreter here, as the publicKey field doesn't use it.
	key := publicKeyFieldGetter(nil)

	byteArray, err := interpreter.ByteArrayValueToByteSlice(key)
	if err != nil {
		return nil, fmt.Errorf("public key needs to be a byte array. %w", err)
	}

	// sign algo field
	signAlgoField, ok := fields.Get(sema.PublicKeySignAlgoField)
	if !ok {
		return nil, errors.New("sign algorithm is not set")
	}

	signAlgoValue, ok := signAlgoField.(*interpreter.CompositeValue)
	if !ok {
		return nil, fmt.Errorf(
			"sign algorithm does not belong to type: %s",
			sema.SignatureAlgorithmType.QualifiedString(),
		)
	}

	rawValue, ok := signAlgoValue.Fields().Get(sema.EnumRawValueFieldName)
	if !ok {
		return nil, errors.New("cannot find sign algorithm raw value")
	}

	signAlgoRawValue, ok := rawValue.(interpreter.UInt8Value)
	if !ok {
		return nil, fmt.Errorf(
			"sign algorithm raw-value does not belong to type: %s",
			sema.UInt8Type.QualifiedString(),
		)
	}

	// `valid` and `validated` fields
	var valid, validated bool
	validField, validated := fields.Get(sema.PublicKeyIsValidField)
	if validated {
		valid = bool(validField.(interpreter.BoolValue))
	}

	return &PublicKey{
		PublicKey: byteArray,
		SignAlgo:  SignatureAlgorithm(signAlgoRawValue.ToInt()),
		IsValid:   valid,
		Validated: validated,
	}, nil
}

func NewPublicKeyValue(
	publicKey *PublicKey,
	validatePublicKey interpreter.PublicKeyValidationHandlerFunc,
) *interpreter.CompositeValue {
	return interpreter.NewPublicKeyValue(
		interpreter.ByteSliceToByteArrayValue(publicKey.PublicKey),
		stdlib.NewSignatureAlgorithmCase(publicKey.SignAlgo.RawValue()),
		func(publicKeyValue *interpreter.CompositeValue) interpreter.BoolValue {
			// If the public key is already validated, avoid re-validating, and return the cached result.
			if publicKey.Validated {
				return interpreter.BoolValue(publicKey.IsValid)
			}

			return validatePublicKey(publicKeyValue)
		},
	)
}

func NewAccountKeyValue(
	accountKey *AccountKey,
	validatePublicKey interpreter.PublicKeyValidationHandlerFunc,
) *interpreter.CompositeValue {
	return interpreter.NewAccountKeyValue(
		interpreter.NewIntValueFromInt64(int64(accountKey.KeyIndex)),
		NewPublicKeyValue(
			accountKey.PublicKey,
			validatePublicKey,
		),
		stdlib.NewHashAlgorithmCase(accountKey.HashAlgo.RawValue()),
		interpreter.NewUFix64ValueWithInteger(uint64(accountKey.Weight)),
		interpreter.BoolValue(accountKey.IsRevoked),
	)
}

func NewHashAlgorithmFromValue(value interpreter.Value) HashAlgorithm {
	hashAlgoValue := value.(*interpreter.CompositeValue)

	rawValue, ok := hashAlgoValue.Fields().Get(sema.EnumRawValueFieldName)
	if !ok {
		panic("cannot find hash algorithm raw value")
	}

	hashAlgoRawValue := rawValue.(interpreter.UInt8Value)

	return HashAlgorithm(hashAlgoRawValue.ToInt())
}

func validatePublicKey(publicKeyValue *interpreter.CompositeValue, runtimeInterface Interface) interpreter.BoolValue {
	publicKey, err := NewPublicKeyFromValue(publicKeyValue)
	if err != nil {
		return false
	}

	var valid bool
	wrapPanic(func() {
		valid, err = runtimeInterface.ValidatePublicKey(publicKey)
	})

	if err != nil {
		panic(err)
	}

	return interpreter.BoolValue(valid)
}

func verifySignature(
	signatureValue *interpreter.ArrayValue,
	signedDataValue *interpreter.ArrayValue,
	domainSeparationTagValue *interpreter.StringValue,
	hashAlgorithmValue *interpreter.CompositeValue,
	publicKeyValue *interpreter.CompositeValue,
	runtimeInterface Interface,
) interpreter.BoolValue {

	signature, err := interpreter.ByteArrayValueToByteSlice(signatureValue)
	if err != nil {
		panic(fmt.Errorf("failed to get signature. %w", err))
	}

	signedData, err := interpreter.ByteArrayValueToByteSlice(signedDataValue)
	if err != nil {
		panic(fmt.Errorf("failed to get signed data. %w", err))
	}

	domainSeparationTag := domainSeparationTagValue.Str

	hashAlgorithm := NewHashAlgorithmFromValue(hashAlgorithmValue)

	publicKey, err := NewPublicKeyFromValue(publicKeyValue)
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
	dataValue *interpreter.ArrayValue,
	tagValue *interpreter.StringValue,
	hashAlgorithmValue *interpreter.CompositeValue,
	runtimeInterface Interface,
) *interpreter.ArrayValue {

	data, err := interpreter.ByteArrayValueToByteSlice(dataValue)
	if err != nil {
		panic(fmt.Errorf("failed to get data. %w", err))
	}

	var tag string
	if tagValue != nil {
		tag = tagValue.Str
	}

	hashAlgorithm := NewHashAlgorithmFromValue(hashAlgorithmValue)

	var result []byte
	wrapPanic(func() {
		result, err = runtimeInterface.Hash(data, tag, hashAlgorithm)
	})
	if err != nil {
		panic(err)
	}

	return interpreter.ByteSliceToByteArrayValue(result)
}
