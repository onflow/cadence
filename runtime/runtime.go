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

// Runtime is a runtime capable of executing Cadence.
type Runtime interface {
	// ExecuteScript executes the given script.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteScript(script []byte, arguments [][]byte, runtimeInterface Interface, location Location) (cadence.Value, error)

	// ExecuteTransaction executes the given transaction.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteTransaction(script []byte, arguments [][]byte, runtimeInterface Interface, location Location) error

	// ParseAndCheckProgram parses and checks the given code without executing the program.
	//
	// This function returns an error if the program contains any syntax or semantic errors.
	ParseAndCheckProgram(code []byte, runtimeInterface Interface, location Location) (*sema.Checker, error)
}

var typeDeclarations = append(
	stdlib.FlowBuiltInTypes,
	stdlib.BuiltinTypes...,
).ToTypeDeclarations()

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

func validTopLevelDeclarations(location ast.Location) []common.DeclarationKind {
	switch location.(type) {
	case TransactionLocation:
		return validTopLevelDeclarationsInTransaction
	case AddressLocation:
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
type interpreterRuntime struct{}

type Option func(Runtime)

// NewInterpreterRuntime returns a interpreter-based version of the Flow runtime.
func NewInterpreterRuntime(options ...Option) Runtime {
	runtime := &interpreterRuntime{}
	for _, option := range options {
		option(runtime)
	}
	return runtime
}

func (r *interpreterRuntime) ExecuteScript(
	script []byte,
	arguments [][]byte,
	runtimeInterface Interface,
	location Location,
) (cadence.Value, error) {

	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)

	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	checker, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil, false)
	if err != nil {
		return nil, newError(err)
	}

	ep, ok := checker.GlobalValues["main"]
	if !ok {
		return nil, &MissingEntryPointError{Expected: "main"}
	}

	invokableType, ok := ep.Type.(sema.InvokableType)
	if !ok {
		return nil, &InvalidEntryPointTypeError{
			Type: ep.Type,
		}
	}
	epSignature := invokableType.InvocationFunctionType()

	// Ensure entrypoint return type is storable
	storableResults := map[*sema.Member]bool{}
	if _, isVoid := epSignature.ReturnTypeAnnotation.Type.(*sema.VoidType); !isVoid {
		if !epSignature.ReturnTypeAnnotation.Type.IsStorable(storableResults) {
			return nil, &ScriptReturnTypeNotStorableError{
				Type: epSignature.ReturnTypeAnnotation.Type,
			}
		}
	}

	// Ensure entrypoint parameters type is storable
	if len(epSignature.Parameters) > 0 {
		for _, param := range epSignature.Parameters {
			if _, isVoid := param.TypeAnnotation.Type.(*sema.VoidType); !isVoid {
				if !param.TypeAnnotation.Type.IsStorable(storableResults) {
					return nil, &ScriptParameterTypeNotStorableError{
						Type: param.TypeAnnotation.Type,
					}
				}
			}
		}
	}

	value, inter, err := r.interpret(
		location,
		runtimeInterface,
		runtimeStorage,
		checker,
		functions,
		nil,
		scriptExecutionFunction(epSignature.Parameters, arguments, runtimeInterface),
	)
	if err != nil {
		return nil, newError(err)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	runtimeStorage.writeCached(inter)

	return exportValue(value), nil
}

func scriptExecutionFunction(parameters []*sema.Parameter, arguments [][]byte, runtimeInterface Interface) func(inter *interpreter.Interpreter) (interpreter.Value, error) {
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

type interpretFunc func(inter *interpreter.Interpreter) (interpreter.Value, error)

func (r *interpreterRuntime) interpret(
	location ast.Location,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	options []interpreter.Option,
	f interpretFunc,
) (
	exportableValue,
	*interpreter.Interpreter,
	error,
) {
	inter, err := r.newInterpreter(checker, functions, runtimeInterface, runtimeStorage, options)
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
		runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramInterpreted(location, duration)
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
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.AuthAccountValue {
	return interpreter.NewAuthAccountValue(
		addressValue,
		r.newAddPublicKeyFunction(addressValue, runtimeInterface),
		r.newRemovePublicKeyFunction(addressValue, runtimeInterface),
		r.newAuthAccountContracts(
			addressValue,
			runtimeInterface,
			runtimeStorage,
		),
	)
}

func (r *interpreterRuntime) ExecuteTransaction(
	script []byte,
	arguments [][]byte,
	runtimeInterface Interface,
	location Location,
) error {
	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)

	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	checker, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil, false)
	if err != nil {
		if err, ok := err.(*ParsingCheckingError); ok {
			err.StorageCache = runtimeStorage.cache
			return newError(err)
		}

		return newError(err)
	}

	transactions := checker.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		return newError(InvalidTransactionCountError{
			Count: transactionCount,
		})
	}

	transactionType := transactions[0]

	var authorizers []Address
	wrapPanic(func() {
		authorizers = runtimeInterface.GetSigningAccounts()
	})

	// check parameter count

	argumentCount := len(arguments)
	authorizerCount := len(authorizers)

	transactionParameterCount := len(transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		return newError(InvalidEntryPointParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		})
	}

	// Ensure parameter types are storable
	storableResults := map[*sema.Member]bool{}
	if len(transactionType.Parameters) > 0 {
		for _, param := range transactionType.Parameters {
			if _, isVoid := param.TypeAnnotation.Type.(*sema.VoidType); !isVoid {
				if !param.TypeAnnotation.Type.IsStorable(storableResults) {
					return newError(&TransactionParameterTypeNotStorableError{
						Type: param.TypeAnnotation.Type,
					})
				}
			}
		}
	}

	transactionAuthorizerCount := len(transactionType.PrepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		return newError(InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		})
	}

	// gather authorizers

	authorizerValues := make([]interpreter.Value, authorizerCount)

	for i, address := range authorizers {
		authorizerValues[i] = r.newAuthAccountValue(
			interpreter.NewAddressValue(address),
			runtimeInterface, runtimeStorage,
		)
	}

	_, inter, err := r.interpret(
		location,
		runtimeInterface,
		runtimeStorage,
		checker,
		functions,
		nil,
		r.transactionExecutionFunction(
			transactionType.Parameters,
			arguments,
			runtimeInterface,
			authorizerValues,
		),
	)
	if err != nil {
		return newError(err)
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

		exportedParameterType := exportType(parameterType, map[sema.TypeID]cadence.Type{})
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

// ParseAndCheckProgram parses the given script and runs type check.
func (r *interpreterRuntime) ParseAndCheckProgram(code []byte, runtimeInterface Interface, location Location) (*sema.Checker, error) {
	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)
	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	checker, err := r.parseAndCheckProgram(code, runtimeInterface, location, functions, nil, true)
	if err != nil {
		return nil, newError(err)
	}

	return checker, nil
}

func (r *interpreterRuntime) parseAndCheckProgram(
	code []byte,
	runtimeInterface Interface,
	location Location,
	functions stdlib.StandardLibraryFunctions,
	options []sema.Option,
	useCache bool,
) (*sema.Checker, error) {

	var program *ast.Program
	var checker *sema.Checker
	var err error

	wrapError := func(err error) error {
		return &ParsingCheckingError{
			Err:      err,
			Code:     code,
			Location: location,
			Options:  options,
			UseCache: useCache,
			Checker:  checker,
			Program:  program,
		}
	}

	if useCache {
		wrapPanic(func() {
			program, err = runtimeInterface.GetCachedProgram(location)
		})
		if err != nil {
			return nil, wrapError(err)
		}
	}

	if program == nil {
		program, err = r.parse(location, code, runtimeInterface)
		if err != nil {
			return nil, wrapError(err)
		}
	}

	importResolver := r.importResolver(runtimeInterface)
	valueDeclarations := functions.ToValueDeclarations()

	checker, err = sema.NewChecker(
		program,
		location,
		append(
			[]sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
				sema.WithPredeclaredTypes(typeDeclarations),
				sema.WithValidTopLevelDeclarationsHandler(validTopLevelDeclarations),
				sema.WithLocationHandler(runtimeInterface.ResolveLocation),
				sema.WithImportHandler(func(checker *sema.Checker, location ast.Location) (sema.Import, *sema.CheckerError) {
					switch location {
					case stdlib.CryptoChecker.Location:
						return sema.CheckerImport{
							Checker: stdlib.CryptoChecker,
						}, nil

					default:
						var program *ast.Program
						var err error
						checker, checkerErr := checker.EnsureLoaded(location, func() *ast.Program {
							program, err = importResolver(location)
							return program
						})
						// TODO: improve
						if err != nil {
							return nil, &sema.CheckerError{
								Errors: []error{err},
							}
						}
						if checkerErr != nil {
							return nil, checkerErr
						}
						return sema.CheckerImport{
							Checker: checker,
						}, nil
					}
				}),
				sema.WithCheckHandler(func(location ast.Location, check func()) {
					reportMetric(
						func() {
							check()
						},
						runtimeInterface,
						func(metrics Metrics, duration time.Duration) {
							metrics.ProgramChecked(location, duration)
						},
					)
				}),
			},
			options...,
		)...,
	)
	if err != nil {
		return nil, wrapError(err)
	}

	err = checker.Check()
	if err != nil {
		return nil, wrapError(err)
	}

	// After the program has passed semantic analysis, cache the program AST.
	wrapPanic(func() {
		err = runtimeInterface.CacheProgram(location, program)
	})
	if err != nil {
		return nil, err
	}

	return checker, nil
}

func (r *interpreterRuntime) newInterpreter(
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	options []interpreter.Option,
) (*interpreter.Interpreter, error) {

	defaultOptions := []interpreter.Option{
		interpreter.WithPredefinedValues(functions.ToValues()),
		interpreter.WithOnEventEmittedHandler(
			func(
				inter *interpreter.Interpreter,
				eventValue *interpreter.CompositeValue,
				eventType *sema.CompositeType,
			) {
				r.emitEvent(inter, runtimeInterface, eventValue, eventType)
			},
		),
		interpreter.WithStorageKeyHandler(
			func(_ *interpreter.Interpreter, _ common.Address, indexingType sema.Type) string {
				return string(indexingType.ID())
			},
		),
		interpreter.WithInjectedCompositeFieldsHandler(
			r.injectedCompositeFieldsHandler(runtimeInterface, runtimeStorage),
		),
		interpreter.WithUUIDHandler(func() (uuid uint64) {
			wrapPanic(func() {
				uuid = runtimeInterface.GenerateUUID()
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
					runtimeInterface,
					runtimeStorage,
				)
			},
		),
		interpreter.WithImportLocationHandler(
			r.importLocationHandler(runtimeInterface),
		),
	}

	defaultOptions = append(defaultOptions,
		r.storageInterpreterOptions(runtimeStorage)...,
	)

	defaultOptions = append(defaultOptions,
		r.meteringInterpreterOptions(runtimeInterface)...,
	)

	return interpreter.NewInterpreter(
		checker,
		append(defaultOptions, options...)...,
	)
}

func (r *interpreterRuntime) importLocationHandler(runtimeInterface Interface) interpreter.ImportLocationHandlerFunc {
	importResolver := r.importResolver(runtimeInterface)

	return func(inter *interpreter.Interpreter, location ast.Location) interpreter.Import {
		switch location {
		case stdlib.CryptoChecker.Location:
			return interpreter.ProgramImport{
				Program: stdlib.CryptoChecker.Program,
			}

		default:
			program, err := importResolver(location)
			if err != nil {
				panic(err)
			}

			return interpreter.ProgramImport{
				Program: program,
			}
		}
	}
}

func (r *interpreterRuntime) injectedCompositeFieldsHandler(
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.InjectedCompositeFieldsHandlerFunc {
	return func(
		_ *interpreter.Interpreter,
		location Location,
		_ sema.TypeID,
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
				case AddressLocation:
					address = location.Address
				default:
					panic(runtimeErrors.NewUnreachableError())
				}

				addressValue := interpreter.NewAddressValue(address)

				return map[string]interpreter.Value{
					"account": r.newAuthAccountValue(addressValue, runtimeInterface, runtimeStorage),
				}
			}
		}

		return nil
	}
}

func (r *interpreterRuntime) storageInterpreterOptions(runtimeStorage *interpreterRuntimeStorage) []interpreter.Option {
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
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) stdlib.StandardLibraryFunctions {
	return append(
		stdlib.FlowBuiltInFunctions(stdlib.FlowBuiltinImpls{
			CreateAccount:   r.newCreateAccountFunction(runtimeInterface, runtimeStorage),
			GetAccount:      r.newGetAccountFunction(runtimeInterface),
			Log:             r.newLogFunction(runtimeInterface),
			GetCurrentBlock: r.newGetCurrentBlockFunction(runtimeInterface),
			GetBlock:        r.newGetBlockFunction(runtimeInterface),
			UnsafeRandom:    r.newUnsafeRandomFunction(runtimeInterface),
		}),
		stdlib.BuiltinFunctions...,
	)
}

func (r *interpreterRuntime) importResolver(runtimeInterface Interface) ImportResolver {
	return func(location Location) (program *ast.Program, err error) {

		wrapPanic(func() {
			program, err = runtimeInterface.GetCachedProgram(location)
		})
		if err != nil {
			return nil, err
		}
		if program != nil {
			return program, nil
		}

		var code []byte
		if addressLocation, ok := location.(AddressLocation); ok {
			wrapPanic(func() {
				code, err = runtimeInterface.GetAccountContractCode(
					addressLocation.Address,
					addressLocation.Name,
				)
			})
		} else {
			wrapPanic(func() {
				code, err = runtimeInterface.GetCode(location)
			})
		}
		if err != nil {
			return nil, err
		}
		if code == nil {
			return nil, nil
		}

		program, err = r.parse(location, code, runtimeInterface)
		if err != nil {
			return nil, err
		}

		wrapPanic(func() {
			err = runtimeInterface.CacheProgram(location, program)
		})
		if err != nil {
			return nil, err
		}

		return program, nil
	}
}

func (r *interpreterRuntime) parse(
	location ast.Location,
	script []byte,
	runtimeInterface Interface,
) (
	program *ast.Program,
	err error,
) {

	parse := func() {
		program, err = parser2.ParseProgram(string(script))
	}

	reportMetric(
		parse,
		runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ProgramParsed(location, duration)
		},
	)

	return
}

// emitEvent converts an event value to native Go types and emits it to the runtime interface.
func (r *interpreterRuntime) emitEvent(
	inter *interpreter.Interpreter,
	runtimeInterface Interface,
	event *interpreter.CompositeValue,
	eventType *sema.CompositeType,
) {
	fields := make([]exportableValue, len(eventType.ConstructorParameters))

	for i, parameter := range eventType.ConstructorParameters {
		fields[i] = newExportableValue(event.Fields[parameter.Identifier], inter)
	}

	eventValue := exportableEvent{
		Type:   eventType,
		Fields: fields,
	}

	exportedEvent := exportEvent(eventValue)
	wrapPanic(func() {
		runtimeInterface.EmitEvent(exportedEvent)
	})
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

	exportedEvent := exportEvent(eventValue)
	wrapPanic(func() {
		runtimeInterface.EmitEvent(exportedEvent)
	})
}

func CodeToHashValue(code []byte) *interpreter.ArrayValue {
	codeHash := sha3.Sum256(code)
	return interpreter.ByteSliceToByteArrayValue(codeHash[:])
}

func (r *interpreterRuntime) newCreateAccountFunction(
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		payer, ok := invocation.Arguments[0].(interpreter.AuthAccountValue)
		if !ok {
			panic(fmt.Sprintf(
				"%[1]s requires the third argument to be an %[1]s",
				&sema.AuthAccountType{},
			))
		}

		var address Address
		var err error
		wrapPanic(func() {
			address, err = runtimeInterface.CreateAccount(payer.AddressValue().ToAddress())
		})
		if err != nil {
			panic(err)
		}

		addressValue := interpreter.NewAddressValue(address)

		r.emitAccountEvent(
			stdlib.AccountCreatedEventType,
			runtimeInterface,
			[]exportableValue{
				newExportableValue(addressValue, nil),
			},
		)

		account := r.newAuthAccountValue(addressValue, runtimeInterface, runtimeStorage)

		return trampoline.Done{Result: account}
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

func (r *interpreterRuntime) writeContract(
	runtimeStorage *interpreterRuntimeStorage,
	addressValue interpreter.AddressValue,
	name string,
	contractValue interpreter.OptionalValue,
) {
	runtimeStorage.writeValue(
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
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
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

		case AddressLocation:
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
	location ast.Location,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	argumentTypes []sema.Type,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	invocationRange ast.Range,
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

	if argumentCount != parameterCount {
		return nil, fmt.Errorf("invalid argument count: expected %d, got %d", parameterCount, argumentCount)
	}

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

	interpreterOptions := []interpreter.Option{
		interpreter.WithContractValueHandler(
			func(
				inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				constructor interpreter.FunctionValue,
				invocationRange ast.Range,
			) *interpreter.CompositeValue {

				// If the contract is the deployed contract, instantiate it using
				// the provided constructor and given arguments

				if ast.LocationsMatch(compositeType.Location, contractType.Location) &&
					compositeType.Identifier == contractType.Identifier {

					value, err := inter.InvokeFunctionValue(constructor,
						constructorArguments,
						argumentTypes,
						parameterTypes,
						invocationRange,
					)
					if err != nil {
						panic(err)
					}

					contract = value.(*interpreter.CompositeValue)

					return contract
				}

				// The contract is not the deployed contract, load it from storage
				return r.loadContract(
					inter,
					compositeType,
					constructor,
					invocationRange,
					runtimeInterface,
					runtimeStorage,
				)
			},
		),
	}

	_, _, err := r.interpret(
		location,
		runtimeInterface,
		runtimeStorage,
		checker,
		functions,
		interpreterOptions,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return contract, err
}

func (r *interpreterRuntime) newGetAccountFunction(_ Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		accountAddress := invocation.Arguments[0].(interpreter.AddressValue)
		publicAccount := interpreter.NewPublicAccountValue(accountAddress)
		return trampoline.Done{Result: publicAccount}
	}
}

func (r *interpreterRuntime) newLogFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		message := fmt.Sprint(invocation.Arguments[0])
		wrapPanic(func() {
			runtimeInterface.Log(message)
		})
		result := interpreter.VoidValue{}
		return trampoline.Done{Result: result}
	}
}

func (r *interpreterRuntime) getCurrentBlockHeight(runtimeInterface Interface) (currentBlockHeight uint64) {
	wrapPanic(func() {
		currentBlockHeight = runtimeInterface.GetCurrentBlockHeight()
	})
	return
}

func (r *interpreterRuntime) getBlockAtHeight(height uint64, runtimeInterface Interface) (*BlockValue, error) {

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
		height := r.getCurrentBlockHeight(runtimeInterface)
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
		wrapPanic(func() {
			rand = runtimeInterface.UnsafeRandom()
		})
		return trampoline.Done{Result: interpreter.UInt64Value(rand)}
	}
}

func (r *interpreterRuntime) newAuthAccountContracts(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.AuthAccountContractsValue {
	return interpreter.AuthAccountContractsValue{
		Address:        addressValue,
		AddFunction:    r.newAuthAccountContractsChangeFunction(addressValue, runtimeInterface, runtimeStorage, false),
		UpdateFunction: r.newAuthAccountContractsChangeFunction(addressValue, runtimeInterface, runtimeStorage, true),
		GetFunction:    r.newAuthAccountContractsGetFunction(addressValue, runtimeInterface),
		RemoveFunction: r.newAuthAccountContractsRemoveFunction(addressValue, runtimeInterface, runtimeStorage),
	}
}

func (r *interpreterRuntime) newAuthAccountContractsChangeFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
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

			address := addressValue.ToAddress()
			existingCode, err := runtimeInterface.GetAccountContractCode(address, nameArgument)
			if err != nil {
				panic(err)
			}

			if isUpdate {
				// We are updating an existing contract.
				// Ensure that no contract/contract interface with the given name exists already

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

			location := AddressLocation{
				Address: address,
				Name:    nameArgument,
			}

			// NOTE: do NOT use the cache!

			const useCache = false

			functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

			checker, err := r.parseAndCheckProgram(code, runtimeInterface, location, functions, nil, useCache)
			if err != nil {
				panic(fmt.Errorf("invalid contract: %w", err))
			}

			// The code may declare exactly one contract or one contract interface.

			var contractTypes []*sema.CompositeType
			var contractInterfaceTypes []*sema.InterfaceType

			for _, variable := range checker.GlobalTypes {
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
			}

			var deployedType sema.Type
			var contractType *sema.CompositeType
			var contractInterfaceType *sema.InterfaceType
			var declaredName string

			switch {
			case len(contractTypes) == 1 && len(contractInterfaceTypes) == 0:
				contractType = contractTypes[0]
				declaredName = contractType.Identifier
				deployedType = contractType
			case len(contractInterfaceTypes) == 1 && len(contractTypes) == 0:
				contractInterfaceType = contractInterfaceTypes[0]
				declaredName = contractInterfaceType.Identifier
				deployedType = contractInterfaceType
			}

			if deployedType == nil {
				panic(errors.New("invalid contract: the code must declare exactly one contract or contract interface"))
			}

			// The declared contract or contract interface must have the name
			// passed to the constructor as the first argument

			if declaredName != nameArgument {
				panic(fmt.Errorf(
					"invalid contract: the declaration must have the same name as the given name argument. epected %q, got %q",
					nameValue.Str,
					declaredName,
				))
			}

			r.updateAccountContractCode(
				runtimeInterface,
				runtimeStorage,
				declaredName,
				code,
				addressValue,
				location,
				checker,
				contractType,
				constructorArguments,
				constructorArgumentTypes,
				invocation.LocationRange.Range,
				updateAccountContractCodeOptions{
					createContract: !isUpdate,
				},
			)

			codeHashValue := CodeToHashValue(code)

			eventArguments := []exportableValue{
				newExportableValue(addressValue, nil),
				newExportableValue(codeHashValue, nil),
				newExportableValue(nameValue, nil),
			}

			if isUpdate {
				r.emitAccountEvent(
					stdlib.AccountContractUpdatedEventType,
					runtimeInterface,
					eventArguments,
				)
			} else {
				r.emitAccountEvent(
					stdlib.AccountContractAddedEventType,
					runtimeInterface,
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
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	name string,
	code []byte,
	addressValue interpreter.AddressValue,
	location AddressLocation,
	checker *sema.Checker,
	contractType *sema.CompositeType,
	constructorArguments []interpreter.Value,
	constructorArgumentTypes []sema.Type,
	invocationRange ast.Range,
	options updateAccountContractCodeOptions,
) {
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

	var contractValue interpreter.OptionalValue = interpreter.NilValue{}

	createContract := contractType != nil && options.createContract

	address := addressValue.ToAddress()

	if createContract {

		functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

		contract, err := r.instantiateContract(
			location,
			contractType,
			constructorArguments,
			constructorArgumentTypes,
			runtimeInterface,
			runtimeStorage,
			checker,
			functions,
			invocationRange,
		)

		if err != nil {
			panic(err)
		}

		contractValue = interpreter.NewSomeValueOwningNonCopying(contract)

		contractValue.SetOwner(&address)
	}

	var err error

	// NOTE: only update account code if contract instantiation succeeded
	wrapPanic(func() {
		err = runtimeInterface.UpdateAccountContractCode(address, name, code)
	})
	if err != nil {
		panic(err)
	}

	if createContract {
		r.writeContract(runtimeStorage, addressValue, name, contractValue)
	}
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
	runtimeStorage *interpreterRuntimeStorage,
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
				wrapPanic(func() {
					err = runtimeInterface.RemoveAccountContractCode(address, nameArgument)
				})
				if err != nil {
					panic(err)
				}

				r.writeContract(
					runtimeStorage,
					addressValue,
					nameArgument,
					interpreter.NilValue{},
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

func compositeTypesToIDValues(types []*sema.CompositeType) *interpreter.ArrayValue {
	typeIDValues := make([]interpreter.Value, len(types))

	for i, typ := range types {
		typeIDValues[i] = interpreter.NewStringValue(string(typ.ID()))
	}

	return interpreter.NewArrayValueUnownedNonCopying(typeIDValues...)
}

// Block

type BlockValue struct {
	Height    interpreter.UInt64Value
	View      interpreter.UInt64Value
	ID        *interpreter.ArrayValue
	Timestamp interpreter.Fix64Value
}

func NewBlockValue(block Block) BlockValue {

	// height
	heightValue := interpreter.UInt64Value(block.Height)

	// view
	viewValue := interpreter.UInt64Value(block.View)

	// ID
	var values = make([]interpreter.Value, stdlib.BlockIDSize)
	for i, b := range block.Hash {
		values[i] = interpreter.UInt8Value(b)
	}
	idValue := &interpreter.ArrayValue{Values: values}

	// timestamp
	// TODO: verify
	timestampValue := interpreter.NewFix64ValueWithInteger(time.Unix(0, block.Timestamp).Unix())

	return BlockValue{
		Height:    heightValue,
		View:      viewValue,
		ID:        idValue,
		Timestamp: timestampValue,
	}
}

func (BlockValue) IsValue() {}

func (BlockValue) DynamicType(*interpreter.Interpreter) interpreter.DynamicType {
	return nil
}

func (v BlockValue) Copy() interpreter.Value {
	return v
}

func (BlockValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BlockValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BlockValue) IsModified() bool {
	return false
}

func (BlockValue) SetModified(_ bool) {
	// NO-OP
}

func (v BlockValue) GetMember(_ *interpreter.Interpreter, _ interpreter.LocationRange, name string) interpreter.Value {
	switch name {
	case "height":
		return v.Height

	case "view":
		return v.View

	case "id":
		return v.ID

	case "timestamp":
		return v.Timestamp
	}

	return nil
}

func (v BlockValue) SetMember(_ *interpreter.Interpreter, _ interpreter.LocationRange, _ string, _ interpreter.Value) {
	panic(runtimeErrors.NewUnreachableError())
}

func (v BlockValue) IDAsByteArray() [stdlib.BlockIDSize]byte {
	var byteArray [stdlib.BlockIDSize]byte
	for i, b := range v.ID.Values {
		byteArray[i] = byte(b.(interpreter.UInt8Value))
	}
	return byteArray
}

func (v BlockValue) String() string {
	return fmt.Sprintf(
		"Block(height: %s, view: %s, id: 0x%x, timestamp: %s)",
		v.Height,
		v.View,
		v.IDAsByteArray(),
		v.Timestamp,
	)
}
