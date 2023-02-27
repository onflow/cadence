/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/errors"
)

type VM struct {
	Program   *bbq.Program
	globals   []Value
	constants []Value
	functions map[string]*bbq.Function
	callFrame *callFrame
	stack     []Value
}

func NewVM(program *bbq.Program) *VM {
	functions := indexFunctions(program.Functions)

	// TODO: include non-function globals
	globals := make([]Value, 0, len(functions))
	for _, function := range functions {
		// TODO:
		globals = append(globals, FunctionValue{Function: function})
	}

	return &VM{
		Program:   program,
		globals:   globals,
		functions: functions,
		constants: make([]Value, len(program.Constants)),
	}
}

func indexFunctions(functions []*bbq.Function) map[string]*bbq.Function {
	indexedFunctions := make(map[string]*bbq.Function, len(functions))
	for _, function := range functions {
		indexedFunctions[function.Name] = function
	}
	return indexedFunctions
}

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	vm.stack[lastIndex] = nil
	vm.stack = vm.stack[:lastIndex]
	return value
}

func (vm *VM) dropN(count int) {
	stackHeight := len(vm.stack)
	for i := 1; i <= count; i++ {
		vm.stack[stackHeight-i] = nil
	}
	vm.stack = vm.stack[:stackHeight-count]
}

func (vm *VM) peekPop() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex-1], vm.pop()
}

func (vm *VM) replaceTop(value Value) {
	lastIndex := len(vm.stack) - 1
	vm.stack[lastIndex] = value
}

func (vm *VM) pushCallFrame(function *bbq.Function, arguments []Value) {

	locals := make([]Value, function.LocalCount)
	for i, argument := range arguments {
		locals[i] = argument
	}

	callFrame := &callFrame{
		parent:   vm.callFrame,
		locals:   locals,
		function: function,
	}
	vm.callFrame = callFrame
}

func (vm *VM) popCallFrame() {
	vm.callFrame = vm.callFrame.parent
}

func (vm *VM) Invoke(name string, arguments ...Value) (Value, error) {
	function, ok := vm.functions[name]
	if !ok {
		return nil, errors.NewDefaultUserError("unknown function")
	}

	if len(arguments) != int(function.ParameterCount) {
		return nil, errors.NewDefaultUserError("wrong number of arguments")
	}

	vm.pushCallFrame(function, arguments)

	vm.run()

	if len(vm.stack) == 0 {
		return nil, nil
	}

	return vm.pop(), nil
}

type vmOp func(*VM)

var vmOps = [...]vmOp{
	opcode.ReturnValue: opReturnValue,
	opcode.Jump:        opJump,
	opcode.JumpIfFalse: opJumpIfFalse,
	opcode.IntAdd:      opBinaryIntAdd,
	opcode.IntSubtract: opBinaryIntSubtract,
	opcode.IntLess:     opBinaryIntLess,
	opcode.IntGreater:  opBinaryIntGreater,
	opcode.True:        opTrue,
	opcode.False:       opFalse,
	opcode.GetConstant: opGetConstant,
	opcode.GetLocal:    opGetLocal,
	opcode.SetLocal:    opSetLocal,
	opcode.GetGlobal:   opGetGlobal,
	opcode.Call:        opCall,
}

func opReturnValue(vm *VM) {
	value := vm.pop()
	vm.popCallFrame()
	vm.push(value)
}

func opJump(vm *VM) {
	callFrame := vm.callFrame
	target := callFrame.getUint16()
	callFrame.ip = target
}

func opJumpIfFalse(vm *VM) {
	callFrame := vm.callFrame
	target := callFrame.getUint16()
	value := vm.pop().(BoolValue)
	if !value {
		callFrame.ip = target
	}
}

func opBinaryIntAdd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.Add(rightNumber))
}

func opBinaryIntSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.Subtract(rightNumber))
}

func opBinaryIntLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.Less(rightNumber))
}

func opBinaryIntGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.Greater(rightNumber))
}

func opTrue(vm *VM) {
	vm.push(trueValue)
}

func opFalse(vm *VM) {
	vm.push(falseValue)
}

func opGetConstant(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	constant := vm.constants[index]
	if constant == nil {
		constant = vm.initializeConstant(index)
	}
	vm.push(constant)
}

func opGetLocal(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	local := callFrame.locals[index]
	vm.push(local)
}

func opSetLocal(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	callFrame.locals[index] = vm.pop()
}

func opGetGlobal(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	vm.push(vm.globals[index])
}

func opCall(vm *VM) {
	// TODO: support any function value
	value := vm.pop().(FunctionValue)
	stackHeight := len(vm.stack)
	parameterCount := int(value.Function.ParameterCount)
	arguments := vm.stack[stackHeight-parameterCount:]
	vm.pushCallFrame(value.Function, arguments)
	vm.dropN(parameterCount)
}

func opPop(vm *VM) {
	_ = vm.pop()
}

func opNew(vm *VM) {
	stackHeight := len(vm.stack)
	const parameterCount = 1
	arguments := vm.stack[stackHeight-parameterCount:]

	// TODO: get location
	name := arguments[0].(StringValue)

	value := StructValue{Name: string(name.string)}
	vm.push(value)
}

func (vm *VM) run() {
	for {

		callFrame := vm.callFrame

		if callFrame == nil ||
			int(callFrame.ip) >= len(callFrame.function.Code) {

			return
		}

		op := opcode.Opcode(callFrame.function.Code[callFrame.ip])
		callFrame.ip++

		switch op {
		case opcode.ReturnValue:
			opReturnValue(vm)
		case opcode.Jump:
			opJump(vm)
		case opcode.JumpIfFalse:
			opJumpIfFalse(vm)
		case opcode.IntAdd:
			opBinaryIntAdd(vm)
		case opcode.IntSubtract:
			opBinaryIntSubtract(vm)
		case opcode.IntLess:
			opBinaryIntLess(vm)
		case opcode.IntGreater:
			opBinaryIntGreater(vm)
		case opcode.True:
			opTrue(vm)
		case opcode.False:
			opFalse(vm)
		case opcode.GetConstant:
			opGetConstant(vm)
		case opcode.GetLocal:
			opGetLocal(vm)
		case opcode.SetLocal:
			opSetLocal(vm)
		case opcode.GetGlobal:
			opGetGlobal(vm)
		case opcode.Call:
			opCall(vm)
		case opcode.Pop:
			opPop(vm)
		case opcode.New:
			opNew(vm)
		default:
			panic(errors.NewUnreachableError())
		}

		// Faster in Go <1.19:
		// vmOps[op](vm)
	}
}

func (vm *VM) initializeConstant(index uint16) (value Value) {
	constant := vm.Program.Constants[index]
	switch constant.Kind {
	case constantkind.Int:
		// TODO:
		smallInt, _, _ := leb128.ReadInt64(constant.Data)
		value = IntValue{smallInt}
	case constantkind.String:
		value = StringValue{constant.Data}
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	vm.constants[index] = value
	return value
}
