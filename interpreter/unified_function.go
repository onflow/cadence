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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// minimal interfaces needed by all native/host functions
type UnifiedFunctionContext interface {
	ReferenceTracker
	ValueStaticTypeContext
	ValueTransferContext
	StorageContext
	StaticTypeConversionHandler
	ValueComparisonContext
	InvocationContext
}

type UnifiedNativeFunction func(
	context UnifiedFunctionContext,
	args *ArgumentExtractor,
	receiver Value,
	typeArguments []StaticType,
	locationRange LocationRange,
) Value

// InterpreterArgumentExtractor adapts interpreter arguments to ArgumentExtractor
type ArgumentExtractor struct {
	arguments []Value
}

func NewArgumentExtractor(arguments []Value) *ArgumentExtractor {
	return &ArgumentExtractor{
		arguments: arguments,
	}
}

func (e *ArgumentExtractor) Count() int {
	return len(e.arguments)
}

func (e *ArgumentExtractor) Get(index int) Value {
	if index < 0 || index >= len(e.arguments) {
		panic(errors.NewUnreachableError())
	}
	return e.arguments[index]
}

func (e *ArgumentExtractor) GetNumber(index int) NumberValue {
	value := e.Get(index)
	numberValue, ok := value.(NumberValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return numberValue
}

func (e *ArgumentExtractor) GetFunction(index int) FunctionValue {
	value := e.Get(index)
	functionValue, ok := value.(FunctionValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return functionValue
}

func (e *ArgumentExtractor) GetType(index int) TypeValue {
	value := e.Get(index)
	typeValue, ok := value.(TypeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return typeValue
}

func (e *ArgumentExtractor) GetString(index int) *StringValue {
	value := e.Get(index)

	stringValue, ok := value.(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return stringValue
}

func (e *ArgumentExtractor) GetInt(index int) IntValue {
	value := e.Get(index)

	intValue, ok := value.(IntValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return intValue
}

func (e *ArgumentExtractor) GetArray(index int) *ArrayValue {
	value := e.Get(index)

	arrayValue, ok := value.(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return arrayValue
}

func (e *ArgumentExtractor) GetComposite(index int) *CompositeValue {
	value := e.Get(index)
	compositeValue, ok := value.(*CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return compositeValue
}

func (e *ArgumentExtractor) GetUFix64(index int) UFix64Value {
	value := e.Get(index)
	uFix64Value, ok := value.(UFix64Value)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return uFix64Value
}

func (e *ArgumentExtractor) GetValue(index int) Value {
	return e.Get(index)
}

func (e *ArgumentExtractor) GetOptional(index int) OptionalValue {
	value := e.Get(index)

	optionalValue, ok := value.(OptionalValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return optionalValue
}

func (e *ArgumentExtractor) GetBool(index int) BoolValue {
	value := e.Get(index)

	boolValue, ok := value.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return boolValue
}

func (e *ArgumentExtractor) GetAddress(index int) AddressValue {
	value := e.Get(index)

	addressValue, ok := value.(AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return addressValue
}

// These are all the functions that need to exist to work with the interpreter
func AdaptUnifiedFunctionForInterpreter(fn UnifiedNativeFunction) HostFunction {
	return func(invocation Invocation) Value {
		context := invocation.InvocationContext
		args := NewArgumentExtractor(invocation.Arguments)

		var receiver Value
		if invocation.Self != nil {
			receiver = *invocation.Self
		}

		// Convert TypeParameterTypes to []StaticType
		var typeArguments []StaticType
		if invocation.TypeParameterTypes != nil {
			typeArguments = make([]StaticType, 0, invocation.TypeParameterTypes.Len())
			invocation.TypeParameterTypes.Foreach(func(key *sema.TypeParameter, semaType sema.Type) {
				staticType := ConvertSemaToStaticType(context, semaType)
				typeArguments = append(typeArguments, staticType)
			})
		}

		result := fn(context, args, receiver, typeArguments, invocation.LocationRange)

		return result
	}
}

func NewUnifiedStaticHostFunctionValue(
	context InvocationContext,
	functionType *sema.FunctionType,
	fn UnifiedNativeFunction,
) *HostFunctionValue {
	return NewStaticHostFunctionValue(
		context,
		functionType,
		AdaptUnifiedFunctionForInterpreter(fn),
	)
}

func NewUnifiedBoundHostFunctionValue(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function UnifiedNativeFunction,
) BoundFunctionValue {

	// Wrap the unified function to work with the standard HostFunction signature
	wrappedFunction := AdaptUnifiedFunctionForInterpreter(function)

	hostFunc := NewStaticHostFunctionValue(context, funcType, wrappedFunction)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}

func GetAccountTypePrivateAddressValue(receiver Value) AddressValue {
	simpleCompositeValue := receiver.(*SimpleCompositeValue)

	addressMetaInfo := simpleCompositeValue.PrivateField(AccountTypePrivateAddressFieldName)
	address, ok := addressMetaInfo.(AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return address
}
