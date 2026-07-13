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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func TestVM_pop(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
	)

	a := vm.pop()

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), a)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
		},
	)
}

func TestVM_peekPop(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)

	a, b := vm.peekPop()

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), a)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), b)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
	)
}

func TestVM_replaceTop(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
		},
	)

	vm.replaceTop(interpreter.NewUnmeteredIntValueFromInt64(3))

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)
}

func TestVM_pop2(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	a, b := vm.pop2()

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), a)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), b)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
		},
	)
}

func TestVM_pop3(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(4))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
			interpreter.NewUnmeteredIntValueFromInt64(4),
		},
	)

	a, b, c := vm.pop3()

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), a)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), b)
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(4), c)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
		},
	)
}

func TestVM_peek(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)

	a := vm.peek()

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), a)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)
}

func TestVM_peekN(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	require.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)

	values := vm.peekN(2)

	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(2), values[0])
	assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(3), values[1])

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)
}

func TestVM_dropN(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	vm.dropN(2)

	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
		},
	)
}

func TestVM_popN(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}
	conf := NewConfig(nil)
	vm := NewVM(nil, program, conf)

	vm.push(interpreter.NewUnmeteredIntValueFromInt64(1))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(2))
	vm.push(interpreter.NewUnmeteredIntValueFromInt64(3))

	popped := vm.popN(2)

	// Assert popped values.
	assert.Equal(t,
		popped,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
	)

	// Assert the remaining values.
	assert.Equal(t,
		vm.stack,
		[]Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
		},
	)
}

func TestContext_NewFunctionWithType_NativeFunctionPreservesDereferenceReceiver(t *testing.T) {
	t.Parallel()

	// Two genuinely different function types: `fun(): String` and `fun(Int): String`.
	originalFuncType := sema.NewSimpleFunctionType(
		sema.FunctionPurityView,
		nil,
		sema.StringTypeAnnotation,
	)
	newFuncType := sema.NewSimpleFunctionType(
		sema.FunctionPurityView,
		[]sema.Parameter{
			{TypeAnnotation: sema.IntTypeAnnotation},
		},
		sema.StringTypeAnnotation,
	)
	require.False(
		t,
		originalFuncType.Equal(newFuncType),
		"the two function types must be distinct for this test to be meaningful",
	)

	newStaticType := interpreter.NewFunctionStaticType(nil, newFuncType)
	context := NewContext(NewConfig(nil))

	nativeFuncBody := func(
		_ interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		_ []interpreter.Value,
	) interpreter.Value {
		return interpreter.NewUnmeteredStringValue("ok")
	}

	t.Run("dereferencing receiver stays dereferencing", func(t *testing.T) {
		t.Parallel()

		// NewNativeFunctionValue defaults dereferenceReceiver to true.
		original := NewNativeFunctionValue("test-deref", originalFuncType, nativeFuncBody)

		require.True(t, original.DereferenceReceiver())

		// The original function must have the original type.
		require.Same(
			t,
			originalFuncType,
			original.FunctionType(context),
		)

		require.NotEqual(
			t,
			newStaticType,
			original.StaticType(context),
		)

		result := context.NewFunctionWithType(original, newStaticType)
		newNativeFunc, ok := result.(*NativeFunctionValue)
		require.True(
			t,
			ok,
			"expected *NativeFunctionValue, got %T",
			result,
		)

		// `dereferenceReceiver` must be preserved.
		assert.True(
			t,
			newNativeFunc.DereferenceReceiver(),
			"dereferenceReceiver=true must be preserved across NewFunctionWithType",
		)

		// The result must report the NEW type only, not the original.
		assert.Same(
			t,
			newFuncType,
			newNativeFunc.FunctionType(context),
			"FunctionType must be the new type passed to NewFunctionWithType, not the original",
		)
		assert.Equal(
			t,
			newStaticType,
			newNativeFunc.StaticType(context),
			"StaticType must be the new type, not the original",
		)

		// The underlying native closure is shared (same function pointer).
		assert.Equal(
			t,
			reflect.ValueOf(original.Function).Pointer(),
			reflect.ValueOf(newNativeFunc.Function).Pointer(),
			"the underlying native function closure must be shared",
		)
	})

	t.Run("non-dereferencing receiver stays non-dereferencing", func(t *testing.T) {
		t.Parallel()

		original := NewNativeFunctionValueWithDerivedType(
			"test-no-deref",
			func(_ Value, _ interpreter.ValueStaticTypeContext) *sema.FunctionType {
				return originalFuncType
			},
			nativeFuncBody,
		).WithDereferenceReceiver(false)

		require.False(t, original.DereferenceReceiver())
		require.True(t, original.HasComputedFunctionType())
		require.Same(
			t,
			originalFuncType,
			original.ComputeFunctionType(nil, context),
		)

		result := context.NewFunctionWithType(original, newStaticType)
		newNativeFunc, ok := result.(*NativeFunctionValue)
		require.True(
			t,
			ok,
			"expected *NativeFunctionValue, got %T",
			result,
		)

		// `dereferenceReceiver` must be preserved.
		assert.False(
			t,
			newNativeFunc.DereferenceReceiver(),
			"dereferenceReceiver=false must be preserved across NewFunctionWithType",
		)

		// A derived-type function is pinned to a concrete type by the cast,
		// so the getter is intentionally NOT carried over.
		assert.False(
			t,
			newNativeFunc.HasComputedFunctionType(),
			"NewFunctionWithType pins a concrete type; the getter should not be carried over",
		)

		// The result must report the NEW type only, not the original.
		assert.Same(
			t,
			newFuncType,
			newNativeFunc.FunctionType(context),
			"FunctionType must be the new type passed to NewFunctionWithType, not the original",
		)
		assert.Equal(
			t,
			newStaticType,
			newNativeFunc.StaticType(context),
			"StaticType must be the new type, not the original",
		)
	})
}
