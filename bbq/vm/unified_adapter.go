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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// AdaptUnifiedFunctionForVM converts a UnifiedNativeFunction to work with the VM
func AdaptUnifiedFunctionForVM(fn interpreter.UnifiedNativeFunction) NativeFunction {
	return func(context *Context, typeArguments []bbq.StaticType, receiver Value, arguments ...Value) Value {
		// Create a minimal adapter that implements UnifiedFunctionContext
		args := interpreter.NewInterpreterArgumentExtractor(arguments)

		// Convert bbq.StaticType to interpreter.StaticType
		commonTypeArgs := make([]interpreter.StaticType, len(typeArguments))
		for i, typeArg := range typeArguments {
			commonTypeArgs[i] = typeArg
		}

		result, err := fn(context, args, receiver, commonTypeArgs, interpreter.EmptyLocationRange)
		if err != nil {
			// In the VM system, errors are typically panicked
			panic(err)
		}
		return result
	}
}

// NewUnifiedNativeFunctionValue creates a native function value using the unified approach
func NewUnifiedNativeFunctionValue(
	name string,
	funcType *sema.FunctionType,
	fn interpreter.UnifiedNativeFunction,
) *NativeFunctionValue {
	return NewNativeFunctionValue(
		name,
		funcType,
		AdaptUnifiedFunctionForVM(fn),
	)
}

// For bound functions in the VM, we can just use the same AdaptUnifiedFunctionForVM
// since the VM already passes the receiver as a parameter to NativeFunction

// NewUnifiedNativeFunctionValueWithDerivedType creates a native function value with derived type using the unified approach
func NewUnifiedNativeFunctionValueWithDerivedType(
	name string,
	typeGetter func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType,
	fn interpreter.UnifiedNativeFunction,
) *NativeFunctionValue {
	return NewNativeFunctionValueWithDerivedType(
		name,
		typeGetter,
		AdaptUnifiedFunctionForVM(fn),
	)
}
