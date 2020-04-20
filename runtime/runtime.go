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
	"errors"
	"fmt"

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
	ExecuteScript(script []byte, runtimeInterface Interface, location Location) (Value, error)

	// ExecuteTransaction executes the given transaction.
	//
	// This function returns an error if the program has errors (e.g syntax errors, type errors),
	// or if the execution fails.
	ExecuteTransaction(script []byte, runtimeInterface Interface, location Location) error

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

func (r *interpreterRuntime) ExecuteScript(script []byte, runtimeInterface Interface, location Location) (Value, error) {
	runtimeStorage := newInterpreterRuntimeStorage(runtimeInterface)

	functions := r.standardLibraryFunctions(runtimeInterface, runtimeStorage)

	checker, err := r.parseAndCheckProgram(script, runtimeInterface, location, functions, nil)
	if err != nil {
		return Value{}, newError(err)
	}

	_, ok := checker.GlobalValues["main"]
	if !ok {
		// TODO: error because no main?
		return Value{}, nil
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
		return Value{}, newError(err)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	runtimeStorage.writeCached()

	return value, nil
}

func (r *interpreterRuntime) interpret(
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	options []interpreter.Option,
	f func(inter *interpreter.Interpreter) (interpreter.Value, error),
) (
	Value,
	error,
) {
	inter, err := r.newInterpreter(checker, functions, runtimeInterface, runtimeStorage, options)
	if err != nil {
		return Value{}, err
	}

	if err := inter.Interpret(); err != nil {
		return Value{}, err
	}

	if f != nil {
		value, err := f(inter)
		if err != nil {
			return Value{}, err
		}

		return newRuntimeValue(value, inter), nil
	}

	return Value{}, nil
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
	transactionFunctionType := transactionType.EntryPointFunctionType()

	signingAccountAddresses := runtimeInterface.GetSigningAccounts()

	// check parameter count

	signingAccountsCount := len(signingAccountAddresses)
	transactionFunctionParameterCount := len(transactionFunctionType.Parameters)
	if signingAccountsCount != transactionFunctionParameterCount {
		return newError(InvalidTransactionParameterCountError{
			Expected: signingAccountsCount,
			Actual:   transactionFunctionParameterCount,
		})
	}

	// check parameter types

	for _, parameter := range transactionFunctionType.Parameters {
		parameterType := parameter.TypeAnnotation.Type

		if !parameterType.Equal(&sema.AuthAccountType{}) {
			return newError(InvalidTransactionParameterTypeError{
				Actual: parameterType,
			})
		}
	}

	signingAccounts := make([]interface{}, signingAccountsCount)

	for i, address := range signingAccountAddresses {
		signingAccounts[i] = r.newAuthAccountValue(
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
			err := inter.InvokeTransaction(0, signingAccounts...)
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
	program, err := r.parse(code)
	if err != nil {
		return nil, err
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

	return checker, nil
}

func (r *interpreterRuntime) newInterpreter(
	checker *sema.Checker,
	functions stdlib.StandardLibraryFunctions,
	runtimeInterface Interface,
	runtimeStorage *interpreterRuntimeStorage,
	options []interpreter.Option,
) (*interpreter.Interpreter, error) {

	importResolver := r.importResolver(runtimeInterface)

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
		interpreter.WithStorageKeyHandler(
			func(_ *interpreter.Interpreter, _ common.Address, indexingType sema.Type) string {
				return string(indexingType.ID())
			},
		),
		interpreter.WithInjectedCompositeFieldsHandler(
			func(
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
			},
		),
		interpreter.WithUUIDHandler(func() uint64 {
			// TODO: move to main interface
			if runtimeInterfaceV2, ok := runtimeInterface.(InterfaceV2); ok {
				return runtimeInterfaceV2.GenerateUUID()
			} else {
				return 0
			}
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
			func(inter *interpreter.Interpreter, location ast.Location) *ast.Program {
				program, err := importResolver(location)
				if err != nil {
					panic(err)
				}

				err = program.ResolveImports(importResolver)
				if err != nil {
					panic(err)
				}

				return program
			},
		),
	}

	return interpreter.NewInterpreter(
		checker,
		append(defaultOptions, options...)...,
	)
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
		}),
		stdlib.BuiltinFunctions...,
	)
}

func (r *interpreterRuntime) importResolver(runtimeInterface Interface) ImportResolver {
	return func(location Location) (program *ast.Program, e error) {
		script, err := runtimeInterface.ResolveImport(location)
		if err != nil {
			return nil, err
		}
		return r.parse(script)
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
	fields := make([]Value, len(eventType.ConstructorParameters))

	for i, parameter := range eventType.ConstructorParameters {
		fields[i] = newRuntimeValue(event.Fields[parameter.Identifier], inter)
	}

	eventValue := Event{
		Type:   eventType,
		Fields: fields,
	}

	runtimeInterface.EmitEvent(eventValue)
}

func (r *interpreterRuntime) emitAccountEvent(
	eventType *sema.CompositeType,
	runtimeInterface Interface,
	eventFields []Value,
) {
	eventValue := Event{
		Type:   eventType,
		Fields: eventFields,
	}

	runtimeInterface.EmitEvent(eventValue)
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
			publicKey, err := toBytes(pkVal)
			if err != nil {
				panic(fmt.Sprintf("Account requires the first parameter to be an array of keys ([[Int]])"))
			}
			publicKeys[i] = publicKey
		}

		code, err := toBytes(invocation.Arguments[1])
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

		codeValue := fromBytes(code)

		contractTypeIDs := compositeTypesToIDValues(contractTypes)

		r.emitAccountEvent(
			stdlib.AccountCreatedEventType,
			runtimeInterface,
			[]Value{
				newRuntimeValue(addressValue, nil),
				newRuntimeValue(codeValue, nil),
				newRuntimeValue(contractTypeIDs, nil),
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

			publicKey, err := toBytes(publicKeyValue)
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
				[]Value{
					newRuntimeValue(addressValue, nil),
					newRuntimeValue(publicKeyValue, nil),
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

			publicKey, err := runtimeInterface.RemoveAccountKey(addressValue.ToAddress(), index.IntValue())
			if err != nil {
				panic(err)
			}

			publicKeyValue := fromBytes(publicKey)

			r.emitAccountEvent(
				stdlib.AccountKeyRemovedEventType,
				runtimeInterface,
				[]Value{
					newRuntimeValue(addressValue, nil),
					newRuntimeValue(publicKeyValue, nil),
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

			code, err := toBytes(invocation.Arguments[0])
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

			codeValue := fromBytes(code)

			contractTypeIDs := compositeTypesToIDValues(contractTypes)

			r.emitAccountEvent(
				stdlib.AccountCodeUpdatedEventType,
				runtimeInterface,
				[]Value{
					newRuntimeValue(addressValue, nil),
					newRuntimeValue(codeValue, nil),
					newRuntimeValue(contractTypeIDs, nil),
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

func (r *interpreterRuntime) newGetCurrentBlockFunction(_ Interface) interpreter.HostFunction {
	return func(invocation interpreter.Invocation) trampoline.Trampoline {
		// TODO: https://github.com/dapperlabs/flow-go/issues/552

		var makeBlock func(uint64) BlockValue
		makeBlock = func(number uint64) BlockValue {

			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.BigEndian, number)
			if err != nil {
				panic(err)
			}

			encoded := buf.Bytes()
			var hash [stdlib.BlockIDSize]byte
			copy(hash[stdlib.BlockIDSize-len(encoded):], encoded)

			return BlockValue{
				Number: number,
				ID:     hash,
				NextBlock: func() *BlockValue {
					nextBlock := makeBlock(number + 1)
					return &nextBlock
				},
				PreviousBlock: func() *BlockValue {
					if number == 1 {
						return nil
					}
					previousBlock := makeBlock(number - 1)
					return &previousBlock
				},
			}
		}

		return trampoline.Done{Result: makeBlock(1)}
	}
}

func toBytes(value interpreter.Value) ([]byte, error) {
	array, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return nil, errors.New("value is not an array")
	}

	result := make([]byte, len(array.Values))
	for i, arrayValue := range array.Values {
		intValue, ok := arrayValue.(interpreter.NumberValue)
		if !ok {
			return nil, errors.New("array value is not an integer")
		}

		j := intValue.IntValue()

		if j < 0 || j > 255 {
			return nil, errors.New("array value is not in byte range (0-255)")
		}

		result[i] = byte(j)
	}

	return result, nil
}

func fromBytes(buf []byte) *interpreter.ArrayValue {
	values := make([]interpreter.Value, len(buf))
	for i, b := range buf {
		values[i] = interpreter.NewIntValue(int64(b))
	}

	return &interpreter.ArrayValue{
		Values: values,
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
	Number        uint64
	ID            [stdlib.BlockIDSize]byte
	NextBlock     func() *BlockValue
	PreviousBlock func() *BlockValue
}

func init() {
	gob.Register(&BlockValue{})
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
	case "number":
		return interpreter.UInt64Value(v.Number)

	case "id":
		var values = make([]interpreter.Value, stdlib.BlockIDSize)
		for i, b := range v.ID {
			values[i] = interpreter.UInt8Value(b)
		}
		return &interpreter.ArrayValue{Values: values}

	case "previousBlock":
		previousBlock := v.PreviousBlock()
		if previousBlock == nil {
			return interpreter.NilValue{}
		}
		return interpreter.NewSomeValueOwningNonCopying(previousBlock)

	case "nextBlock":
		nextBlock := v.NextBlock()
		if nextBlock == nil {
			return interpreter.NilValue{}
		}
		return interpreter.NewSomeValueOwningNonCopying(nextBlock)

	default:
		panic(runtimeErrors.NewUnreachableError())
	}
}

func (v BlockValue) SetMember(_ *interpreter.Interpreter, _ interpreter.LocationRange, _ string, _ interpreter.Value) {
	panic(runtimeErrors.NewUnreachableError())
}

func (v BlockValue) String() string {
	return fmt.Sprintf(
		"Block(number: %d, hash: 0x%x)",
		v.Number, v.ID,
	)
}
