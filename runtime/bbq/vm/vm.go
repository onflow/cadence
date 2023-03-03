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
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/bbq/vm/context"
	"github.com/onflow/cadence/runtime/bbq/vm/types"
	"github.com/onflow/cadence/runtime/bbq/vm/values"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type VM struct {
	Program     *bbq.Program
	globals     []values.Value
	constants   []values.Value
	staticTypes []types.StaticType
	functions   map[string]*bbq.Function
	callFrame   *callFrame
	stack       []values.Value
	context     *context.Context
}

func NewVM(program *bbq.Program) *VM {
	// TODO: include non-function globals
	// Iterate through `program.Functions` to be deterministic.
	// Order of globals must be same as index set at `Compiler.addGlobal()`.
	globals := make([]values.Value, 0, len(program.Functions))
	for _, function := range program.Functions {
		// TODO:
		globals = append(globals, values.FunctionValue{Function: function})
	}

	functions := indexFunctions(program.Functions)

	ctx := &context.Context{
		Storage: interpreter.NewInMemoryStorage(nil),
	}

	return &VM{
		Program:     program,
		globals:     globals,
		functions:   functions,
		constants:   make([]values.Value, len(program.Constants)),
		staticTypes: make([]types.StaticType, len(program.Types)),
		context:     ctx,
	}
}

func indexFunctions(functions []*bbq.Function) map[string]*bbq.Function {
	indexedFunctions := make(map[string]*bbq.Function, len(functions))
	for _, function := range functions {
		indexedFunctions[function.Name] = function
	}
	return indexedFunctions
}

func (vm *VM) push(value values.Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() values.Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	vm.stack[lastIndex] = nil
	vm.stack = vm.stack[:lastIndex]
	return value
}

func (vm *VM) peek() values.Value {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex]
}

func (vm *VM) dropN(count int) {
	stackHeight := len(vm.stack)
	for i := 1; i <= count; i++ {
		vm.stack[stackHeight-i] = nil
	}
	vm.stack = vm.stack[:stackHeight-count]
}

func (vm *VM) peekPop() (values.Value, values.Value) {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex-1], vm.pop()
}

func (vm *VM) replaceTop(value values.Value) {
	lastIndex := len(vm.stack) - 1
	vm.stack[lastIndex] = value
}

func (vm *VM) pushCallFrame(function *bbq.Function, arguments []values.Value) {
	// Preserve local index zero for `self`.
	localOffset := 0
	if function.IsCompositeFunction {
		localOffset = 1
	}

	locals := make([]values.Value, function.LocalCount)
	for i, argument := range arguments {
		locals[i+localOffset] = argument
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

func (vm *VM) Invoke(name string, arguments ...values.Value) (values.Value, error) {
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
	value := vm.pop().(values.BoolValue)
	if !value {
		callFrame.ip = target
	}
}

func opBinaryIntAdd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(values.IntValue)
	rightNumber := right.(values.IntValue)
	vm.replaceTop(leftNumber.Add(rightNumber))
}

func opBinaryIntSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(values.IntValue)
	rightNumber := right.(values.IntValue)
	vm.replaceTop(leftNumber.Subtract(rightNumber))
}

func opBinaryIntLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(values.IntValue)
	rightNumber := right.(values.IntValue)
	vm.replaceTop(leftNumber.Less(rightNumber))
}

func opBinaryIntGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(values.IntValue)
	rightNumber := right.(values.IntValue)
	vm.replaceTop(leftNumber.Greater(rightNumber))
}

func opTrue(vm *VM) {
	vm.push(values.TrueValue)
}

func opFalse(vm *VM) {
	vm.push(values.FalseValue)
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
	value := vm.pop().(values.FunctionValue)
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
	kind := vm.callFrame.getUint16()
	compositeKind := common.CompositeKind(kind)

	typeName := vm.callFrame.getString()

	value := values.NewCompositeValue(
		// TODO: get location
		common.StringLocation("test"),
		typeName,
		compositeKind,
		common.Address{},
		vm.context.Storage,
	)
	vm.push(value)
}

func opSetField(vm *VM) {
	fieldName := vm.pop().(values.StringValue)
	fieldNameStr := string(fieldName.String)

	// TODO: support all container types
	structValue := vm.pop().(*values.CompositeValue)

	fieldValue := vm.pop()

	structValue.SetMember(vm.context, fieldNameStr, fieldValue)
}

func opGetField(vm *VM) {
	fieldName := vm.pop().(values.StringValue)
	fieldNameStr := string(fieldName.String)

	// TODO: support all container types
	structValue := vm.pop().(*values.CompositeValue)

	fieldValue := structValue.GetMember(vm.context, fieldNameStr)
	vm.push(fieldValue)
}

func opCheckType(vm *VM) {
	targetType := vm.loadType()

	value := vm.peek()

	transferredValue := value.Transfer(
		vm.context,
		atree.Address{},
		false, nil,
	)

	valueType := transferredValue.StaticType(vm.context.MemoryGauge)
	if !types.IsSubType(valueType, targetType) {
		panic("invalid transfer")
	}

	vm.replaceTop(transferredValue)
}

func opDestroy(vm *VM) {
	value := vm.peek().(*values.CompositeValue)
	value.Destroy(vm.context)
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
		case opcode.SetField:
			opSetField(vm)
		case opcode.GetField:
			opGetField(vm)
		case opcode.CheckType:
			opCheckType(vm)
		case opcode.Destroy:
			opDestroy(vm)
		default:
			panic(errors.NewUnreachableError())
		}

		// Faster in Go <1.19:
		// vmOps[op](vm)
	}
}

func (vm *VM) initializeConstant(index uint16) (value values.Value) {
	constant := vm.Program.Constants[index]
	switch constant.Kind {
	case constantkind.Int:
		// TODO:
		smallInt, _, _ := leb128.ReadInt64(constant.Data)
		value = values.IntValue{SmallInt: smallInt}
	case constantkind.String:
		value = values.StringValue{String: constant.Data}
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	vm.constants[index] = value
	return value
}

func (vm *VM) loadType() types.StaticType {
	index := vm.callFrame.getUint16()
	staticType := vm.staticTypes[index]
	if staticType == nil {
		staticType = vm.initializeType(index)
	}

	return staticType
}

func (vm *VM) initializeType(index uint16) interpreter.StaticType {
	typeBytes := vm.Program.Types[index]
	dec := interpreter.CBORDecMode.NewByteStreamDecoder(typeBytes)
	staticType, err := interpreter.NewTypeDecoder(dec, nil).DecodeStaticType()
	if err != nil {
		panic(err)
	}

	vm.staticTypes[index] = staticType
	return staticType
}
