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

type TypeParameterGetter interface {
	NextStatic() StaticType
	NextSema() sema.Type
}

type InterpreterTypeParameterGetter struct {
	memoryGauge          common.MemoryGauge
	currentTypeParameter *orderedmap.Pair[*sema.TypeParameter, sema.Type]
}

var _ TypeParameterGetter = &InterpreterTypeParameterGetter{}

func NewInterpreterTypeParameterGetter(memoryGauge common.MemoryGauge, typeParameterTypes *sema.TypeParameterTypeOrderedMap) *InterpreterTypeParameterGetter {
	var currentTypeParameter *orderedmap.Pair[*sema.TypeParameter, sema.Type]
	if typeParameterTypes != nil {
		currentTypeParameter = typeParameterTypes.Oldest()
	}

	return &InterpreterTypeParameterGetter{
		memoryGauge:          memoryGauge,
		currentTypeParameter: currentTypeParameter,
	}
}

func (i *InterpreterTypeParameterGetter) NextStatic() StaticType {
	semaType := i.NextSema()
	if semaType == nil {
		return nil
	}
	return ConvertSemaToStaticType(i.memoryGauge, semaType)
}

func (i *InterpreterTypeParameterGetter) NextSema() sema.Type {
	// deletion cannot happen here, type parameters are used multiple times
	// it is also possible that there are no type parameters which is valid
	// see NativeCapabilityBorrowFunction
	current := i.currentTypeParameter
	if current == nil {
		return nil
	}
	i.currentTypeParameter = i.currentTypeParameter.Next()
	return current.Value
}

type NativeFunction func(
	context NativeFunctionContext,
	locationRange LocationRange,
	typeParameterGetter TypeParameterGetter,
	receiver Value,
	args ...Value,
) Value

// These are all the functions that need to exist to work with the interpreter
func AdaptNativeFunctionForInterpreter(fn NativeFunction) HostFunction {
	return func(invocation Invocation) Value {
		context := invocation.InvocationContext

		var receiver Value
		if invocation.Self != nil {
			receiver = *invocation.Self
		}

		typeParameterGetter := NewInterpreterTypeParameterGetter(context, invocation.TypeParameterTypes)

		return fn(context, invocation.LocationRange, typeParameterGetter, receiver, invocation.Arguments...)
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

func NewBoundHostFunctionValueFromNativeFunction(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function NativeFunction,
) BoundFunctionValue {

	// wrap the unified function to work with the standard HostFunction signature
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

// generic helper function to assert that the provided value is of a specific type
// useful for asserting receiver and argument types in unified functions
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
