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

package interpreter

import (
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// minimal interfaces needed by all native functions
type NativeFunctionContext interface {
	ValueStaticTypeContext
	ValueTransferContext
	StaticTypeConversionHandler
	InvocationContext
}

type TypeArgumentsIterator interface {
	NextStatic() StaticType
	NextSema() sema.Type
}

type ArgumentTypesIterator interface {
	NextStatic() StaticType
	NextSema() sema.Type
}

type EmptyTypeArgumentsIterator struct{}

var _ TypeArgumentsIterator = EmptyTypeArgumentsIterator{}

func (EmptyTypeArgumentsIterator) NextStatic() StaticType {
	return nil
}

func (EmptyTypeArgumentsIterator) NextSema() sema.Type {
	return nil
}

var TheEmptyTypeArgumentsIterator TypeArgumentsIterator = EmptyTypeArgumentsIterator{}

type EmptyArgumentTypesIterator struct{}

var _ ArgumentTypesIterator = EmptyArgumentTypesIterator{}

func (EmptyArgumentTypesIterator) NextStatic() StaticType {
	return nil
}

func (EmptyArgumentTypesIterator) NextSema() sema.Type {
	return nil
}

var TheEmptyArgumentTypesIterator ArgumentTypesIterator = EmptyArgumentTypesIterator{}

type InterpreterTypeArgumentsIterator struct {
	memoryGauge common.MemoryGauge
	currentPair *orderedmap.Pair[*sema.TypeParameter, sema.Type]
}

var _ TypeArgumentsIterator = &InterpreterTypeArgumentsIterator{}

func NewInterpreterTypeArgumentsIterator(
	memoryGauge common.MemoryGauge,
	typeArguments *sema.TypeParameterTypeOrderedMap,
) *InterpreterTypeArgumentsIterator {
	var currentPair *orderedmap.Pair[*sema.TypeParameter, sema.Type]
	if typeArguments != nil {
		currentPair = typeArguments.Oldest()
	}

	return &InterpreterTypeArgumentsIterator{
		memoryGauge: memoryGauge,
		currentPair: currentPair,
	}
}

func (i *InterpreterTypeArgumentsIterator) NextStatic() StaticType {
	semaType := i.NextSema()
	if semaType == nil {
		return nil
	}
	return ConvertSemaToStaticType(i.memoryGauge, semaType)
}

func (i *InterpreterTypeArgumentsIterator) NextSema() sema.Type {
	// deletion cannot happen here, type parameters are used multiple times
	// it is also possible that there are no type parameters which is valid
	// see NativeCapabilityBorrowFunction
	current := i.currentPair
	if current == nil {
		return nil
	}
	i.currentPair = i.currentPair.Next()
	return current.Value
}

type InterpreterArgumentTypesIterator struct {
	index         int
	memoryGauge   common.MemoryGauge
	argumentTypes []sema.Type
}

var _ ArgumentTypesIterator = &InterpreterArgumentTypesIterator{}

func NewInterpreterArgumentTypesIterator(
	memoryGauge common.MemoryGauge,
	argumentTypes []sema.Type,
) *InterpreterArgumentTypesIterator {
	return &InterpreterArgumentTypesIterator{
		memoryGauge:   memoryGauge,
		argumentTypes: argumentTypes,
	}
}

func (i *InterpreterArgumentTypesIterator) NextStatic() StaticType {
	semaType := i.NextSema()
	if semaType == nil {
		return nil
	}
	return ConvertSemaToStaticType(i.memoryGauge, semaType)
}

func (i *InterpreterArgumentTypesIterator) NextSema() sema.Type {
	current := i.index
	if current >= len(i.argumentTypes) {
		return nil
	}
	i.index++
	return i.argumentTypes[current]
}

func NewTypeArgumentsIterator(
	context InvocationContext,
	arguments *sema.TypeParameterTypeOrderedMap,
) TypeArgumentsIterator {
	if arguments.Len() == 0 {
		return TheEmptyTypeArgumentsIterator
	}
	return NewInterpreterTypeArgumentsIterator(context, arguments)
}

func NewArgumentTypesIterator(
	context InvocationContext,
	argumentTypes []sema.Type,
) ArgumentTypesIterator {
	if len(argumentTypes) == 0 {
		return TheEmptyArgumentTypesIterator
	}
	return NewInterpreterArgumentTypesIterator(context, argumentTypes)
}

type NativeFunction func(
	context NativeFunctionContext,
	typeArguments TypeArgumentsIterator,
	argumentTypes ArgumentTypesIterator,
	receiver Value,
	args []Value,
) Value

// These are all the functions that need to exist to work with the interpreter

func AdaptNativeFunctionForInterpreter(fn NativeFunction) HostFunction {
	return func(invocation Invocation) Value {
		context := invocation.InvocationContext

		var receiver Value
		if invocation.Self != nil {
			receiver = *invocation.Self
		}

		typeArgumentsIterator := NewTypeArgumentsIterator(context, invocation.TypeArguments)
		argumentTypesIterator := NewArgumentTypesIterator(context, invocation.ArgumentTypes)

		return fn(
			context,
			typeArgumentsIterator,
			argumentTypesIterator,
			receiver,
			invocation.Arguments,
		)
	}
}

func NewUnmeteredStaticHostFunctionValueFromNativeFunction(
	functionType *sema.FunctionType,
	fn NativeFunction,
) *HostFunctionValue {
	return NewUnmeteredStaticHostFunctionValue(
		functionType,
		AdaptNativeFunctionForInterpreter(fn),
	)
}

func NewStaticHostFunctionValueFromNativeFunction(
	gauge common.MemoryGauge,
	functionType *sema.FunctionType,
	fn NativeFunction,
) *HostFunctionValue {
	return NewStaticHostFunctionValue(
		gauge,
		functionType,
		AdaptNativeFunctionForInterpreter(fn),
	)
}

func NewBoundHostFunctionValue(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function NativeFunction,
) BoundFunctionValue {

	// wrap the native function to work with the standard HostFunction signature
	// just like how we do it in the interpreter
	wrappedFunction := AdaptNativeFunctionForInterpreter(function)

	hostFunc := NewStaticHostFunctionValue(context, funcType, wrappedFunction)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}

// AssertValueOfType asserts that the provided value is of a specific type.
// Useful for asserting receiver and argument types in native functions
func AssertValueOfType[T Value](val Value) T {
	value, ok := val.(T)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return value
}

// Helper functions to get the address from the receiver or the address pointer
// interpreter supplies the address, vm does not
// see stdlib/account.go for usage examples

func GetAddressValue(receiver Value, addressPointer *AddressValue) AddressValue {
	if addressPointer == nil {
		return GetAccountTypePrivateAddressValue(receiver)
	}
	return *addressPointer
}

func GetAddress(receiver Value, addressPointer *common.Address) common.Address {
	if addressPointer == nil {
		return GetAccountTypePrivateAddressValue(receiver).ToAddress()
	}
	return *addressPointer
}

func GetAccountTypePrivateAddressValue(receiver Value) AddressValue {
	simpleCompositeValue := AssertValueOfType[*SimpleCompositeValue](receiver)

	addressMetaInfo := simpleCompositeValue.PrivateField(AccountTypePrivateAddressFieldName)
	address := AssertValueOfType[AddressValue](addressMetaInfo)
	return address
}
