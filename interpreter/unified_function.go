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
	"reflect"

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

// ArgumentExtractor provides a unified interface for extracting and validating arguments
type ArgumentExtractor interface {
	Count() int
	Get(index int) (Value, error)
	GetString(index int) (*StringValue, error)
	GetInt(index int) (IntValue, error)
	GetArray(index int) (*ArrayValue, error)
	GetType(index int) (TypeValue, error)
	GetOptional(index int) (OptionalValue, error)
	GetBool(index int) (BoolValue, error)
	GetAddress(index int) (AddressValue, error)
}

// UnifiedNativeFunction uses our minimal UnifiedFunctionContext
// This provides just what most functions need without the complexity of InvocationContext
type UnifiedNativeFunction func(
	context UnifiedFunctionContext,
	args ArgumentExtractor,
	receiver Value,
	typeArguments []StaticType,
	locationRange LocationRange,
) (Value, error)

// ArgumentIndexError is returned when trying to access an argument at an invalid index
type ArgumentIndexError struct {
	Index int
	Count int
}

var _ errors.UserError = ArgumentIndexError{}

func (e ArgumentIndexError) IsUserError() {}

func (e ArgumentIndexError) Error() string {
	return "argument index out of bounds"
}

// ArgumentTypeError is returned when an argument has an unexpected type
type ArgumentTypeError struct {
	Index    int
	Expected string
	Actual   string
}

var _ errors.UserError = ArgumentTypeError{}

func (e ArgumentTypeError) IsUserError() {}

func (e ArgumentTypeError) Error() string {
	return "invalid argument type"
}

// UnifiedArgumentCountError is returned when the number of arguments doesn't match expectations
type UnifiedArgumentCountError struct {
	Expected int
	Actual   int
}

var _ errors.UserError = UnifiedArgumentCountError{}

func (e UnifiedArgumentCountError) IsUserError() {}

func (e UnifiedArgumentCountError) Error() string {
	return "invalid argument count"
}

// InterpreterArgumentExtractor adapts interpreter arguments to ArgumentExtractor
type InterpreterArgumentExtractor struct {
	arguments []Value
}

func NewInterpreterArgumentExtractor(arguments []Value) *InterpreterArgumentExtractor {
	return &InterpreterArgumentExtractor{
		arguments: arguments,
	}
}

func (e *InterpreterArgumentExtractor) Count() int {
	return len(e.arguments)
}

func (e *InterpreterArgumentExtractor) Get(index int) (Value, error) {
	if index < 0 || index >= len(e.arguments) {
		return nil, ArgumentIndexError{
			Index: index,
			Count: len(e.arguments),
		}
	}
	return e.arguments[index], nil
}

func (e *InterpreterArgumentExtractor) GetString(index int) (*StringValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return nil, err
	}

	stringValue, ok := value.(*StringValue)
	if !ok {
		return nil, ArgumentTypeError{
			Index:    index,
			Expected: "String",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return stringValue, nil
}

func (e *InterpreterArgumentExtractor) GetInt(index int) (IntValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return NewUnmeteredIntValueFromInt64(0), err
	}

	intValue, ok := value.(IntValue)
	if !ok {
		return NewUnmeteredIntValueFromInt64(0), ArgumentTypeError{
			Index:    index,
			Expected: "Int",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return intValue, nil
}

func (e *InterpreterArgumentExtractor) GetArray(index int) (*ArrayValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return nil, err
	}

	arrayValue, ok := value.(*ArrayValue)
	if !ok {
		return nil, ArgumentTypeError{
			Index:    index,
			Expected: "Array",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return arrayValue, nil
}

func (e *InterpreterArgumentExtractor) GetType(index int) (TypeValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return NewTypeValue(nil, PrimitiveStaticTypeAny), err
	}

	typeValue, ok := value.(TypeValue)
	if !ok {
		return NewTypeValue(nil, PrimitiveStaticTypeAny), ArgumentTypeError{
			Index:    index,
			Expected: "Type",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return typeValue, nil
}

func (e *InterpreterArgumentExtractor) GetOptional(index int) (OptionalValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return nil, err
	}

	optionalValue, ok := value.(OptionalValue)
	if !ok {
		return nil, ArgumentTypeError{
			Index:    index,
			Expected: "Optional",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return optionalValue, nil
}

func (e *InterpreterArgumentExtractor) GetBool(index int) (BoolValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return FalseValue, err
	}

	boolValue, ok := value.(BoolValue)
	if !ok {
		return FalseValue, ArgumentTypeError{
			Index:    index,
			Expected: "Bool",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return boolValue, nil
}

func (e *InterpreterArgumentExtractor) GetAddress(index int) (AddressValue, error) {
	value, err := e.Get(index)
	if err != nil {
		return NewUnmeteredAddressValueFromBytes([]byte{}), err
	}

	addressValue, ok := value.(AddressValue)
	if !ok {
		return NewUnmeteredAddressValueFromBytes([]byte{}), ArgumentTypeError{
			Index:    index,
			Expected: "Address",
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	return addressValue, nil
}

// AdaptUnifiedFunction converts a UnifiedNativeFunction to work with the interpreter
func AdaptUnifiedFunction(fn UnifiedNativeFunction) HostFunction {
	return func(invocation Invocation) Value {
		context := invocation.InvocationContext
		args := NewInterpreterArgumentExtractor(invocation.Arguments)

		var receiver Value
		if invocation.Self != nil {
			receiver = *invocation.Self
		}

		// Type arguments are not available in interpreter invocations
		result, err := fn(context, args, receiver, nil, invocation.LocationRange)
		if err != nil {
			// In the interpreter system, errors are typically panicked
			panic(err)
		}
		return result
	}
}

// NewUnifiedHostFunctionValue creates a host function value using the unified approach
func NewUnifiedStaticHostFunctionValue(
	context InvocationContext,
	functionType *sema.FunctionType,
	fn UnifiedNativeFunction,
) *HostFunctionValue {
	return NewStaticHostFunctionValue(
		context,
		functionType,
		AdaptUnifiedFunction(fn),
	)
}

// NewBoundUnifiedHostFunctionValue creates a bound function value using the unified approach
// This uses the same UnifiedNativeFunction signature but handles type casting internally
func NewUnifiedBoundHostFunctionValue(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function UnifiedNativeFunction,
) BoundFunctionValue {

	// Wrap the unified function to work with the standard HostFunction signature
	wrappedFunction := AdaptUnifiedFunction(function)

	hostFunc := NewStaticHostFunctionValue(context, funcType, wrappedFunction)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}
