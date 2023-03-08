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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type VM struct {
	functions map[string]FunctionValue
	callFrame *callFrame
	stack     []Value
	config    *Config
}

func NewVM(program *bbq.Program, conf *Config) *VM {
	// TODO: Remove initializing config. Following is for testing purpose only.
	if conf == nil {
		conf = &Config{}
	}
	if conf.Storage == nil {
		conf.Storage = interpreter.NewInMemoryStorage(nil)
	}

	globals := initializeGlobals(program, conf)
	functions := indexFunctions(program.Functions, globals)

	return &VM{
		functions: functions,
		config:    conf,
	}
}

func initializeGlobals(program *bbq.Program, conf *Config) []Value {
	// TODO: global variable lookup relies too much on the order.
	// 	Figure out a better way.

	var importedGlobals []Value
	for _, location := range program.Imports {
		importedProgram := conf.ImportHandler(location)

		// TODO: cache globals for imported programs.

		importedProgramGlobals := initializeGlobals(importedProgram, conf)

		// Load contract value
		if importedProgram.Contract != nil {
			// If the imported program is a contract,
			// load the contract value and populate the global variable.
			contract := importedProgram.Contract
			contractLocation := common.NewAddressLocation(
				conf.MemoryGauge,
				common.MustBytesToAddress(contract.Address),
				contract.Name,
			)

			// TODO: remove this check. This shouldn't be nil ideally.
			if conf.ContractValueHandler != nil {
				// Contract value is always at the zero-th index.
				importedProgramGlobals[0] = conf.ContractValueHandler(conf, contractLocation)
			}
		}

		importedGlobals = append(importedGlobals, importedProgramGlobals...)
	}

	ctx := NewContext(program, nil)

	globals := make([]Value, 0)

	// If the current program is a contract, reserve a global variable for the contract value.
	// The reserved position is always the zero-th index.
	// This value will be populated either by the `init` method invocation of the contract,
	// Or when this program is imported by another (loads the value from storage).
	if program.Contract != nil {
		globals = append(globals, nil)
	}

	// Iterate through `program.Functions` to be deterministic.
	// Order of globals must be same as index set at `Compiler.addGlobal()`.
	// TODO: include non-function globals
	for _, function := range program.Functions {
		// TODO:
		globals = append(globals, FunctionValue{
			Function: function,
			Context:  ctx,
		})
	}

	// Imported globals are added first.
	// This is the same order as they are added in the compiler.
	ctx.Globals = importedGlobals
	ctx.Globals = append(ctx.Globals, globals...)

	// Return only the globals defined in the current program.
	// Because the importer/caller doesn't need to know globals of nested imports.
	return globals
}

func indexFunctions(functions []*bbq.Function, globals []Value) map[string]FunctionValue {
	indexedFunctions := make(map[string]FunctionValue, len(functions))
	for _, globalValue := range globals {
		function, isFunction := globalValue.(FunctionValue)
		if !isFunction {
			continue
		}
		indexedFunctions[function.Function.Name] = function
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

func (vm *VM) peek() Value {
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

func (vm *VM) peekPop() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex-1], vm.pop()
}

func (vm *VM) replaceTop(value Value) {
	lastIndex := len(vm.stack) - 1
	vm.stack[lastIndex] = value
}

func (vm *VM) pushCallFrame(ctx *Context, function *bbq.Function, arguments []Value) {
	locals := make([]Value, function.LocalCount)
	for i, argument := range arguments {
		locals[i] = argument
	}

	callFrame := &callFrame{
		parent:   vm.callFrame,
		locals:   locals,
		function: function,
		context:  ctx,
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

	if len(arguments) != int(function.Function.ParameterCount) {
		return nil, errors.NewDefaultUserError("wrong number of arguments")
	}

	vm.pushCallFrame(function.Context, function.Function, arguments)

	vm.run()

	if len(vm.stack) == 0 {
		return nil, nil
	}

	return vm.pop(), nil
}

func (vm *VM) InitializeContract(arguments ...Value) (*CompositeValue, error) {
	value, err := vm.Invoke(InitFunctionName, arguments...)
	if err != nil {
		return nil, err
	}

	contractValue, ok := value.(*CompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError("invalid contract value")
	}

	return contractValue, nil
}

type vmOp func(*VM)

var vmOps = [...]vmOp{
	opcode.ReturnValue:  opReturnValue,
	opcode.Jump:         opJump,
	opcode.JumpIfFalse:  opJumpIfFalse,
	opcode.IntAdd:       opBinaryIntAdd,
	opcode.IntSubtract:  opBinaryIntSubtract,
	opcode.IntLess:      opBinaryIntLess,
	opcode.IntGreater:   opBinaryIntGreater,
	opcode.True:         opTrue,
	opcode.False:        opFalse,
	opcode.GetConstant:  opGetConstant,
	opcode.GetLocal:     opGetLocal,
	opcode.SetLocal:     opSetLocal,
	opcode.GetGlobal:    opGetGlobal,
	opcode.InvokeStatic: opInvokeStatic,
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
	vm.push(TrueValue)
}

func opFalse(vm *VM) {
	vm.push(FalseValue)
}

func opGetConstant(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	constant := callFrame.context.Constants[index]
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
	vm.push(callFrame.context.Globals[index])
}

func opSetGlobal(vm *VM) {
	callFrame := vm.callFrame
	index := callFrame.getUint16()
	callFrame.context.Globals[index] = vm.pop()
}

func opInvokeStatic(vm *VM) {
	value := vm.pop().(FunctionValue)
	stackHeight := len(vm.stack)
	parameterCount := int(value.Function.ParameterCount)
	arguments := vm.stack[stackHeight-parameterCount:]

	vm.pushCallFrame(value.Context, value.Function, arguments)
	vm.dropN(parameterCount)
}

func opInvoke(vm *VM) {
	value := vm.pop().(FunctionValue)
	stackHeight := len(vm.stack)

	// Add one to account for `self`
	parameterCount := int(value.Function.ParameterCount) + 1

	arguments := vm.stack[stackHeight-parameterCount:]
	vm.pushCallFrame(value.Context, value.Function, arguments)
	vm.dropN(parameterCount)
}

func opDrop(vm *VM) {
	_ = vm.pop()
}

func opDup(vm *VM) {
	top := vm.peek()
	vm.push(top)
}

func opNew(vm *VM) {
	callframe := vm.callFrame

	kind := callframe.getUint16()
	compositeKind := common.CompositeKind(kind)

	// decode location
	locationLen := callframe.getUint16()
	locationBytes := callframe.function.Code[callframe.ip : callframe.ip+locationLen]
	callframe.ip = callframe.ip + locationLen
	location := decodeLocation(locationBytes)

	typeName := callframe.getString()

	value := NewCompositeValue(
		location,
		typeName,
		compositeKind,
		common.Address{},
		vm.config.Storage,
	)
	vm.push(value)
}

func opSetField(vm *VM) {
	fieldName := vm.pop().(StringValue)
	fieldNameStr := string(fieldName.String)

	// TODO: support all container types
	structValue := vm.pop().(*CompositeValue)

	fieldValue := vm.pop()

	structValue.SetMember(vm.config, fieldNameStr, fieldValue)
}

func opGetField(vm *VM) {
	fieldName := vm.pop().(StringValue)
	fieldNameStr := string(fieldName.String)

	// TODO: support all container types
	structValue := vm.pop().(*CompositeValue)

	fieldValue := structValue.GetMember(vm.config, fieldNameStr)
	if fieldValue == nil {
		panic(interpreter.MissingMemberValueError{
			Name: fieldNameStr,
		})
	}

	vm.push(fieldValue)
}

func opCheckType(vm *VM) {
	targetType := vm.loadType()
	value := vm.peek()

	transferredValue := value.Transfer(
		vm.config,
		atree.Address{},
		false, nil,
	)

	valueType := transferredValue.StaticType(vm.config.MemoryGauge)
	if !IsSubType(valueType, targetType) {
		panic("invalid transfer")
	}

	vm.replaceTop(transferredValue)
}

func opDestroy(vm *VM) {
	value := vm.peek().(*CompositeValue)
	value.Destroy(vm.config)
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
		case opcode.SetGlobal:
			opSetGlobal(vm)
		case opcode.InvokeStatic:
			opInvokeStatic(vm)
		case opcode.Invoke:
			opInvoke(vm)
		case opcode.Drop:
			opDrop(vm)
		case opcode.Dup:
			opDup(vm)
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

func (vm *VM) initializeConstant(index uint16) (value Value) {
	ctx := vm.callFrame.context

	constant := ctx.Program.Constants[index]
	switch constant.Kind {
	case constantkind.Int:
		// TODO:
		smallInt, _, _ := leb128.ReadInt64(constant.Data)
		value = IntValue{SmallInt: smallInt}
	case constantkind.String:
		value = StringValue{String: constant.Data}
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}

	ctx.Constants[index] = value
	return value
}

func (vm *VM) loadType() StaticType {
	callframe := vm.callFrame
	index := callframe.getUint16()
	staticType := callframe.context.StaticTypes[index]
	if staticType == nil {
		staticType = vm.initializeType(index)
	}

	return staticType
}

func (vm *VM) initializeType(index uint16) interpreter.StaticType {
	ctx := vm.callFrame.context
	typeBytes := ctx.Program.Types[index]
	dec := interpreter.CBORDecMode.NewByteStreamDecoder(typeBytes)
	staticType, err := interpreter.NewTypeDecoder(dec, nil).DecodeStaticType()
	if err != nil {
		panic(err)
	}

	ctx.StaticTypes[index] = staticType
	return staticType
}

func decodeLocation(locationBytes []byte) common.Location {
	// TODO: is it possible to re-use decoders?
	dec := interpreter.CBORDecMode.NewByteStreamDecoder(locationBytes)
	locationDecoder := interpreter.NewLocationDecoder(dec, nil)
	location, err := locationDecoder.DecodeLocation()
	if err != nil {
		panic(err)
	}
	return location
}
