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

import "github.com/onflow/cadence/runtime/bbq/opcode"

func (vm *VM) opReturnValue(code opcode.ReturnValue) {
	vm.popCallFrame(code.Index)
}

func (vm *VM) opJump(code opcode.Jump) {
	vm.callFrame.ip = code.Target
}

func (vm *VM) opJumpIfFalse(code opcode.JumpIfFalse) {
	locals := vm.callFrame.locals
	condition := locals.bools[code.Condition]
	if !condition {
		vm.callFrame.ip = code.Target
	}
}

func (vm *VM) opBinaryIntAdd(code opcode.IntAdd) {
	intReg := vm.callFrame.locals.ints
	leftNumber := intReg[code.LeftOperand]
	rightNumber := intReg[code.RightOperand]
	intReg[code.Result] = leftNumber.Add(rightNumber)
}

func (vm *VM) opBinaryIntSubtract(code opcode.IntSubtract) {
	intReg := vm.callFrame.locals.ints
	leftNumber := intReg[code.LeftOperand]
	rightNumber := intReg[code.RightOperand]
	intReg[code.Result] = leftNumber.Subtract(rightNumber)
}

func (vm *VM) opBinaryIntLess(code opcode.IntLess) {
	intReg := vm.callFrame.locals.ints
	leftNumber := intReg[code.LeftOperand]
	rightNumber := intReg[code.RightOperand]
	vm.callFrame.locals.bools[code.Result] = leftNumber.Less(rightNumber)
}

func (vm *VM) opBinaryIntGreater(code opcode.IntGreater) {
	intReg := vm.callFrame.locals.ints
	leftNumber := intReg[code.LeftOperand]
	rightNumber := intReg[code.RightOperand]
	vm.callFrame.locals.bools[code.Result] = leftNumber.Greater(rightNumber)
}

func (vm *VM) opTrue(code opcode.True) {
	vm.callFrame.locals.bools[code.Index] = trueValue
}

func (vm *VM) opFalse(code opcode.False) {
	vm.callFrame.locals.bools[code.Index] = falseValue
}

func (vm *VM) opIntConstantLoad(code opcode.IntConstantLoad) {
	constant := vm.constants[code.Index]
	if constant == nil {
		constant = vm.initializeConstant(code.Index)
	}

	intReg := vm.callFrame.locals.ints
	intReg[code.Target] = constant.(IntValue)
}

func (vm *VM) opMoveInt(code opcode.MoveInt) {
	intReg := vm.callFrame.locals.ints
	intReg[code.To] = intReg[code.From]
}

func (vm *VM) opGlobalFuncLoad(code opcode.GlobalFuncLoad) {
	value := vm.globals[code.Index].(FunctionValue)
	funcReg := vm.callFrame.locals.funcs
	funcReg[code.Result] = value
}

func (vm *VM) opCall(code opcode.Call) {
	// TODO: support any function value
	value := vm.callFrame.locals.funcs[code.FuncIndex]
	vm.pushCallFrame(value.Function, code.Arguments, code.Result)
}
