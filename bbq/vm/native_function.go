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

type VMTypeArgumentsIterator struct {
	index         int
	context       *Context
	typeArguments []bbq.StaticType
}

func NewVMTypeArgumentsIterator(context *Context, typeArguments []bbq.StaticType) *VMTypeArgumentsIterator {
	return &VMTypeArgumentsIterator{
		index:         0,
		context:       context,
		typeArguments: typeArguments,
	}
}

var _ interpreter.TypeArgumentsIterator = &VMTypeArgumentsIterator{}

func (g *VMTypeArgumentsIterator) NextStatic() interpreter.StaticType {
	current := g.index
	if current >= len(g.typeArguments) {
		// much like the interpreter, there can be no type parameters provided, which is valid
		return nil
	}
	g.index++
	return g.typeArguments[current]
}

func (g *VMTypeArgumentsIterator) NextSema() sema.Type {
	staticType := g.NextStatic()
	if staticType == nil {
		return nil
	}
	return g.context.SemaTypeFromStaticType(staticType)
}

func NewTypeArgumentsIterator(context *Context, arguments []bbq.StaticType) interpreter.TypeArgumentsIterator {
	if len(arguments) == 0 {
		return interpreter.TheEmptyTypeArgumentsIterator
	}
	return NewVMTypeArgumentsIterator(context, arguments)
}

func AdaptNativeFunctionForVM(fn interpreter.NativeFunction) NativeFunctionVM {
	return func(
		context *Context,
		typeArguments []bbq.StaticType,
		receiver Value,
		arguments []Value,
	) Value {
		typeArgumentsIterator := NewTypeArgumentsIterator(context, typeArguments)

		return fn(
			context,
			typeArgumentsIterator,
			receiver,
			arguments,
		)
	}
}

func NewNativeFunctionValue(
	name string,
	funcType *sema.FunctionType,
	fn interpreter.NativeFunction,
) *NativeFunctionValue {
	return &NativeFunctionValue{
		Name:         name,
		functionType: funcType,
		Function:     AdaptNativeFunctionForVM(fn),
	}
}

func NewNativeFunctionValueWithDerivedType(
	name string,
	typeGetter func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType,
	fn interpreter.NativeFunction,
) *NativeFunctionValue {
	return &NativeFunctionValue{
		Name:               name,
		Function:           AdaptNativeFunctionForVM(fn),
		functionTypeGetter: typeGetter,
	}
}
