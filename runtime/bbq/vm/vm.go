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

func (vm *VM) pushCallFrame(function *bbq.Function, arguments []opcode.Argument, result uint16) {

	locals := NewRegister(function.LocalCount)

	vm.callFrame.locals.copyTo(locals, arguments)

	callFrame := &callFrame{
		parent:        vm.callFrame,
		locals:        locals,
		function:      function,
		returnToIndex: result,
	}
	vm.callFrame = callFrame
}

func (vm *VM) pushCallFrameOld(function *bbq.Function, arguments []Value) {

	locals := NewRegister(function.LocalCount)
	locals.initializeWithArguments(arguments)

	callFrame := &callFrame{
		parent:   vm.callFrame,
		locals:   locals,
		function: function,
	}
	vm.callFrame = callFrame
}

func (vm *VM) popCallFrame(returnValueIndex uint16) {
	if vm.callFrame.parent == nil {
		vm.callFrame.returnValueIndex = returnValueIndex
		return
	}

	returnValue := vm.callFrame.locals.ints[returnValueIndex]

	returnToIndex := vm.callFrame.returnToIndex

	vm.callFrame = vm.callFrame.parent

	vm.callFrame.locals.ints[returnToIndex] = returnValue
}

func (vm *VM) Invoke(name string, arguments ...Value) (Value, error) {
	function, ok := vm.functions[name]
	if !ok {
		return nil, errors.NewDefaultUserError("unknown function")
	}

	if len(arguments) != int(function.ParameterCount) {
		return nil, errors.NewDefaultUserError("wrong number of arguments")
	}

	vm.pushCallFrameOld(function, arguments)

	vm.run()

	returnValue := vm.callFrame.locals.ints[vm.callFrame.returnValueIndex]

	return returnValue, nil
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

		switch op := op.(type) {
		case opcode.ReturnValue:
			vm.opReturnValue(op)
		case opcode.Jump:
			vm.opJump(op)
		case opcode.JumpIfFalse:
			vm.opJumpIfFalse(op)
		case opcode.IntAdd:
			vm.opBinaryIntAdd(op)
		case opcode.IntSubtract:
			vm.opBinaryIntSubtract(op)
		case opcode.IntLess:
			vm.opBinaryIntLess(op)
		case opcode.IntGreater:
			vm.opBinaryIntGreater(op)
		case opcode.True:
			vm.opTrue(op)
		case opcode.False:
			vm.opFalse(op)
		case opcode.GetIntConstant:
			vm.opGetConstant(op)
		//case opcode.GetLocal:
		//	vm.opGetLocalInt(op)
		case opcode.MoveInt:
			vm.opMoveInt(op)
		case opcode.GetGlobalFunc:
			vm.opGetGlobal(op)
		case opcode.Call:
			vm.opCall(op)
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
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	vm.constants[index] = value
	return value
}
