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
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
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
	ExecuteScript(script []byte, runtimeInterface Interface, location Location) (cadence.Value, error)

	// ExecuteTransaction executes the given transaction.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteTransaction(script []byte, arguments [][]byte, runtimeInterface Interface, location Location) error

	// ParseAndCheckProgram parses and checks the given code without executing the program.
	//
	// This function returns an error if the program contains any syntax or semantic errors.
	ParseAndCheckProgram(code []byte, runtimeInterface Interface, location Location) error
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

const contractKey = "contract"

// interpreterRuntime is a interpreter-based version of the Flow runtime.
type interpreterRuntime struct{}

// NewInterpreterRuntime returns a interpreter-based version of the Flow runtime.
func NewInterpreterRuntime() Runtime {
	return &interpreterRuntime{}
}

func (r *interpreterRuntime) ExecuteScript(
	script []byte,
	runtimeInterface Interface,
	location Location,
) (cadence.Value, error) {

	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)

	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	checker, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil)
	if err != nil {
		return nil, newError(err)
	}

	_, ok := checker.GlobalValues["main"]
	if !ok {
		// TODO: error because no main?
		return nil, nil
	}

	value, err := r.interpret(
		runtimeInterface,
		runtimeStorage,
		checker,
		functions,
		nil,
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			return inter.Invoke("main")
		},
	)
	if err != nil {
		return nil, newError(err)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	runtimeStorage.writeCached()

	return exportValue(value), nil
}

func (r *interpreterRuntime) interpret(
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	options []interpreter.Option,
	f func(inter *interpreter.Interpreter) (interpreter.Value, error),
) (
	exportableValue,
	error,
) {
	inter, err := r.newInterpreter(checker, functions, runtimeInterface, runtimeStorage, options)
	if err != nil {
		return exportableValue{}, err
	}

	if err := inter.Interpret(); err != nil {
		return exportableValue{}, err
	}

	if f != nil {
		value, err := f(inter)
		if err != nil {
			return exportableValue{}, err
		}

		return newExportableValue(value, inter), nil
	}

	return exportableValue{}, nil
}

func (r *interpreterRuntime) newAuthAccountValue(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.AuthAccountValue {
	return interpreter.NewAuthAccountValue(
		addressValue,
		r.newSetCodeFunction(addressValue, runtimeInterface, runtimeStorage),
		r.newAddPublicKeyFunction(addressValue, runtimeInterface),
		r.newRemovePublicKeyFunction(addressValue, runtimeInterface),
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

	checker, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil)
	if err != nil {
		return newError(err)
	}

	transactions := checker.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		return newError(InvalidTransactionCountError{Count: transactionCount})
	}

	transactionType := transactions[0]

	authorizers := runtimeInterface.GetSigningAccounts()

	// check parameter count

	argumentCount := len(arguments)
	authorizerCount := len(authorizers)

	transactionParameterCount := len(transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		return newError(InvalidTransactionParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		})
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

	_, err = r.interpret(
		runtimeInterface,
		runtimeStorage,
		checker,
		functions,
		nil,
		func(inter *interpreter.Interpreter) (interpreter.Value, error) {
			argumentValues := make([]interpreter.Value, argumentCount)

			// decode arguments against parameter types
			for i, parameter := range transactionType.Parameters {
				parameterType := parameter.TypeAnnotation.Type
				argument := arguments[i]

				value, err := runtimeInterface.DecodeArgument(
					argument,
					exportType(parameterType),
				)
				if err != nil {
					return nil, &InvalidTransactionArgumentError{
						Index: i,
						Err:   err,
					}
				}

				arg := importValue(value)

				// check that decoded value is a subtype of static parameter type
				if !interpreter.IsSubType(arg.DynamicType(inter), parameterType) {
					return nil, &InvalidTransactionArgumentError{
						Index: i,
						Err: &InvalidTypeAssignmentError{
							Value: arg,
							Type:  parameterType,
						},
					}
				}

				argumentValues[i] = arg
			}

			allArguments := append(argumentValues, authorizerValues...)

			err := inter.InvokeTransaction(0, allArguments...)
			return nil, err
		},
	)
	if err != nil {
		return newError(err)
	}

	// Write back all stored values, which were actually just cached, back into storage
	runtimeStorage.writeCached()

	return nil
}

func (r *interpreterRuntime) ParseAndCheckProgram(script []byte, runtimeInterface Interface, location Location) error {
	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)
	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	_, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil)
	if err != nil {
		return newError(err)
	}

	return nil
}

func (r *interpreterRuntime) parseAndCheckProgram(
	code []byte,
	runtimeInterface Interface,
	location Location,
	functions stdlib.StandardLibraryFunctions,
	options []sema.Option,
) (*sema.Checker, error) {

	program, err := runtimeInterface.GetCachedProgram(location)
	if err != nil {
		return nil, err
	}

	if program == nil {
		program, err = r.parse(code)
		if err != nil {
			return nil, err
		}
	}

	importResolver := r.importResolver(runtimeInterface)
	err = program.ResolveImports(importResolver)
	if err != nil {
		return nil, err
	}

	valueDeclarations := functions.ToValueDeclarations()

	checker, err := sema.NewChecker(
		program,
		location,
		append(
			[]sema.Option{
				sema.WithPredeclaredValues(valueDeclarations),
				sema.WithPredeclaredTypes(typeDeclarations),
				sema.WithValidTopLevelDeclarationsHandler(validTopLevelDeclarations),
			},
			options...,
		)...,
	)
	if err != nil {
		return nil, err
	}

	if err := checker.Check(); err != nil {
		return nil, err
	}

	// After the program has passed semantic analysis, cache the program AST.
	err = runtimeInterface.CacheProgram(location, program)
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
		interpreter.WithUUIDHandler(func() uint64 {
			return runtimeInterface.GenerateUUID()
		}),
		interpreter.WithContractValueHandler(
			func(
				inter *interpreter.Interpreter,
				compositeType *sema.CompositeType,
				_ interpreter.FunctionValue,
			) *interpreter.CompositeValue {
				// Load the contract from storage
				return r.loadContract(compositeType, runtimeStorage)
			},
		),
		interpreter.WithImportProgramHandler(
			r.importProgramHandler(runtimeInterface),
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

func (r *interpreterRuntime) importProgramHandler(runtimeInterface Interface) interpreter.ImportProgramHandlerFunc {
	importResolver := r.importResolver(runtimeInterface)

	return func(inter *interpreter.Interpreter, location ast.Location) *ast.Program {
		program, err := importResolver(location)
		if err != nil {
			panic(err)
		}

		err = program.ResolveImports(importResolver)
		if err != nil {
			panic(err)
		}

		return program
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

		switch compositeKind {
		case common.CompositeKindContract:
			var address []byte

			switch location := location.(type) {
			case AddressLocation:
				address = location
			default:
				panic(runtimeErrors.NewUnreachableError())
			}

			addressValue := interpreter.NewAddressValueFromBytes(address)

			return map[string]interpreter.Value{
				"account": r.newAuthAccountValue(addressValue, runtimeInterface, runtimeStorage),
			}
		}

		return nil
	}
}

func (r *interpreterRuntime) storageInterpreterOptions(runtimeStorage *interpreterRuntimeStorage) []interpreter.Option {
	return []interpreter.Option{
		interpreter.WithStorageExistenceHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string) bool {
				return runtimeStorage.valueExists(string(address[:]), key)
			},
		),
		interpreter.WithStorageReadHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string) interpreter.OptionalValue {
				return runtimeStorage.readValue(string(address[:]), key)
			},
		),
		interpreter.WithStorageWriteHandler(
			func(_ *interpreter.Interpreter, address common.Address, key string, value interpreter.OptionalValue) {
				runtimeStorage.writeValue(string(address[:]), key, value)
			},
		),
	}
}

func (r *interpreterRuntime) meteringInterpreterOptions(runtimeInterface Interface) []interpreter.Option {
	limit := runtimeInterface.GetComputationLimit()
	if limit <= 0 {
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
		}),
		stdlib.BuiltinFunctions...,
	)
}

func (r *interpreterRuntime) importResolver(runtimeInterface Interface) ImportResolver {
	return func(location Location) (program *ast.Program, e error) {

		program, err := runtimeInterface.GetCachedProgram(location)
		if err != nil {
			return nil, err
		}
		if program != nil {
			return program, nil
		}

		script, err := runtimeInterface.ResolveImport(location)
		if err != nil {
			return nil, err
		}

		program, err = r.parse(script)
		if err != nil {
			return nil, err
		}

		err = runtimeInterface.CacheProgram(location, program)
		if err != nil {
			return nil, err
		}

		return program, nil
	}
}

func (r *interpreterRuntime) parse(script []byte) (program *ast.Program, err error) {
	program, _, err = parser.ParseProgram(string(script))
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

	runtimeInterface.EmitEvent(exportEvent(eventValue))
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

	runtimeInterface.EmitEvent(exportEvent(eventValue))
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
		const requiredArgumentCount = 2

		pkArray := invocation.Arguments[0].(*interpreter.ArrayValue)
		pkValues := pkArray.Values
		publicKeys := make([][]byte, len(pkValues))

		for i, pkVal := range pkValues {
			publicKey, err := interpreter.ByteArrayValueToByteSlice(pkVal)
			if err != nil {
				panic(fmt.Sprintf("Account requires the first parameter to be an array of keys ([[Int]])"))
			}
			publicKeys[i] = publicKey
		}

		code, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[1])
		if err != nil {
			panic(fmt.Sprintf("Account requires the second parameter to be an array of bytes ([Int])"))
		}

		address, err := runtimeInterface.CreateAccount(publicKeys)
		if err != nil {
			panic(err)
		}

		addressValue := interpreter.NewAddressValue(address)

		constructorArguments := invocation.Arguments[requiredArgumentCount:]
		constructorArgumentTypes := invocation.ArgumentTypes[requiredArgumentCount:]

		contractTypes := r.updateAccountCode(
			runtimeInterface,
			runtimeStorage,
			code,
			addressValue,
			constructorArguments,
			constructorArgumentTypes,
			false,
			invocation.LocationRange.Range,
		)

		codeHashValue := CodeToHashValue(code)

		contractTypeIDs := compositeTypesToIDValues(contractTypes)

		r.emitAccountEvent(
			stdlib.AccountCreatedEventType,
			runtimeInterface,
			[]exportableValue{
				newExportableValue(addressValue, nil),
				newExportableValue(codeHashValue, nil),
				newExportableValue(contractTypeIDs, nil),
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
				panic(fmt.Sprintf("addPublicKey requires the first parameter to be an array"))
			}

			err = runtimeInterface.AddAccountKey(addressValue.ToAddress(), publicKey)
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

			publicKey, err := runtimeInterface.RemoveAccountKey(addressValue.ToAddress(), index.ToInt())
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

func (r *interpreterRuntime) newSetCodeFunction(
	addressValue interpreter.AddressValue,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
) interpreter.HostFunctionValue {
	return interpreter.NewHostFunctionValue(
		func(invocation interpreter.Invocation) trampoline.Trampoline {
			const requiredArgumentCount = 1

			code, err := interpreter.ByteArrayValueToByteSlice(invocation.Arguments[0])
			if err != nil {
				panic(fmt.Sprintf("setCode requires the first parameter to be an array of bytes ([Int])"))
			}

			constructorArguments := invocation.Arguments[requiredArgumentCount:]
			constructorArgumentTypes := invocation.ArgumentTypes[requiredArgumentCount:]

			contractTypes := r.updateAccountCode(
				runtimeInterface,
				runtimeStorage,
				code,
				addressValue,
				constructorArguments,
				constructorArgumentTypes,
				true,
				invocation.LocationRange.Range,
			)

			codeHashValue := CodeToHashValue(code)

			contractTypeIDs := compositeTypesToIDValues(contractTypes)

			r.emitAccountEvent(
				stdlib.AccountCodeUpdatedEventType,
				runtimeInterface,
				[]exportableValue{
					newExportableValue(addressValue, nil),
					newExportableValue(codeHashValue, nil),
					newExportableValue(contractTypeIDs, nil),
				},
			)

			result := interpreter.VoidValue{}
			return trampoline.Done{Result: result}
		},
	)
}

func (r *interpreterRuntime) updateAccountCode(
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	code []byte,
	addressValue interpreter.AddressValue,
	constructorArguments []interpreter.Value,
	constructorArgumentTypes []sema.Type,
	checkPermission bool,
	invocationRange ast.Range,
) (contractTypes []*sema.CompositeType) {
	location := AddressLocation(addressValue[:])

	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)
	checker, err := r.parseAndCheckProgram(
		code,
		runtimeInterface,
		location,
		functions,
		nil,
	)
	if err != nil {
		panic(err)
	}

	for _, variable := range checker.GlobalTypes {
		if variable.DeclarationKind == common.DeclarationKindContract {
			contractType := variable.Type.(*sema.CompositeType)
			contractTypes = append(contractTypes, contractType)
		}
	}

	if len(contractTypes) > 1 {
		panic(fmt.Sprintf("code declares more than one contract"))
	}

	// If the code declares a contract, instantiate it and store it

	var contractValue interpreter.OptionalValue = interpreter.NilValue{}

	if len(contractTypes) > 0 {
		contractType := contractTypes[0]

		contract, err := r.instantiateContract(
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
	}

	address := common.Address(addressValue)

	contractValue.SetOwner(&address)

	// NOTE: only update account code if contract instantiation succeeded

	err = runtimeInterface.UpdateAccountCode(addressValue.ToAddress(), code, checkPermission)
	if err != nil {
		panic(err)
	}

	r.writeContract(runtimeStorage, addressValue, contractValue)

	return contractTypes
}

func (r *interpreterRuntime) writeContract(
	runtimeStorage *interpreterRuntimeStorage,
	addressValue interpreter.AddressValue,
	contractValue interpreter.OptionalValue,
) {
	runtimeStorage.writeValue(
		string(addressValue[:]),
		contractKey,
		contractValue,
	)
}

func (r *interpreterRuntime) loadContract(
	compositeType *sema.CompositeType,
	runtimeStorage *interpreterRuntimeStorage,
) *interpreter.CompositeValue {
	address := compositeType.Location.(AddressLocation).ToAddress()
	storedValue := runtimeStorage.readValue(
		string(address[:]),
		contractKey,
	)
	switch typedValue := storedValue.(type) {
	case *interpreter.SomeValue:
		return typedValue.Value.(*interpreter.CompositeValue)
	case interpreter.NilValue:
		panic("failed to load contract")
	default:
		panic(runtimeErrors.NewUnreachableError())
	}
}

func (r *interpreterRuntime) instantiateContract(
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

				return r.loadContract(compositeType, runtimeStorage)
			},
		),
	}

	_, err := r.interpret(
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
		runtimeInterface.Log(fmt.Sprint(invocation.Arguments[0]))
		result := interpreter.VoidValue{}
		return trampoline.Done{Result: result}
	}
}

func (r *interpreterRuntime) getCurrentBlockHeight(runtimeInterface Interface) uint64 {
	// TODO: https://github.com/dapperlabs/flow-go/issues/552
	return 1
}

func (r *interpreterRuntime) getBlockAtHeight(height uint64, runtimeInterface Interface) *BlockValue {
	// TODO: https://github.com/dapperlabs/flow-go/issues/552
	//   Return nil if the requested block does not exist.
	//   For now, always return a fake block.

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, height)
	if err != nil {
		panic(err)
	}

	encoded := buf.Bytes()
	var hash [stdlib.BlockIDSize]byte
	copy(hash[stdlib.BlockIDSize-len(encoded):], encoded)

	block := NewBlockValue(height, hash, time.Unix(int64(height), 0))
	return &block
}

func (r *interpreterRuntime) newGetCurrentBlockFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		height := r.getCurrentBlockHeight(runtimeInterface)
		block := r.getBlockAtHeight(height, runtimeInterface)
		return trampoline.Done{Result: *block}
	}
}

func (r *interpreterRuntime) newGetBlockFunction(runtimeInterface Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		height := uint64(invocation.Arguments[0].(interpreter.UInt64Value))
		block := r.getBlockAtHeight(height, runtimeInterface)
		var result interpreter.Value
		if block == nil {
			result = interpreter.NilValue{}
		} else {
			result = interpreter.NewSomeValueOwningNonCopying(*block)
		}
		return trampoline.Done{Result: result}
	}
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
	ID        *interpreter.ArrayValue
	Timestamp interpreter.Fix64Value
}

func init() {
	gob.Register(&BlockValue{})
}

func NewBlockValue(height uint64, id [stdlib.BlockIDSize]byte, timestamp time.Time) BlockValue {
	// height
	heightValue := interpreter.UInt64Value(height)

	// ID
	var values = make([]interpreter.Value, stdlib.BlockIDSize)
	for i, b := range id {
		values[i] = interpreter.UInt8Value(b)
	}
	idValue := &interpreter.ArrayValue{Values: values}

	// timestamp
	// TODO: verify
	timestampValue := interpreter.NewFix64ValueWithInteger(timestamp.Unix())

	return BlockValue{
		Height:    heightValue,
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

func (v BlockValue) GetMember(_ *interpreter.Interpreter, _ interpreter.LocationRange, name string) interpreter.Value {
	switch name {
	case "height":
		return v.Height

	case "id":
		return v.ID

	case "timestamp":
		return v.Timestamp

	default:
		panic(runtimeErrors.NewUnreachableError())
	}
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
		"Block(height: %s, id: 0x%x, timestamp: %s)",
		v.Height,
		v.IDAsByteArray(),
		v.Timestamp,
	)
}
