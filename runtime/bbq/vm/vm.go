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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/commons"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
)

type VM struct {
	globals            map[string]Value
	callFrame          *callFrame
	stack              []Value
	config             *Config
	linkedGlobalsCache map[common.Location]LinkedGlobals
}

func NewVM(program *bbq.Program, conf *Config) *VM {
	// TODO: Remove initializing config. Following is for testing purpose only.
	if conf == nil {
		conf = &Config{}
	}
	if conf.Storage == nil {
		conf.Storage = interpreter.NewInMemoryStorage(nil)
	}

	// linkedGlobalsCache is a local cache-alike that is being used to hold already linked imports.
	linkedGlobalsCache := map[common.Location]LinkedGlobals{
		BuiltInLocation: {
			// It is safe to re-use native functions map here because,
			// once put into the cache, it will only be used for read-only operations.
			indexedGlobals: NativeFunctions,
		},
	}

	// Link global variables and functions.
	linkedGlobals := LinkGlobals(program, conf, linkedGlobalsCache)

	return &VM{
		globals:            linkedGlobals.indexedGlobals,
		linkedGlobalsCache: linkedGlobalsCache,
		config:             conf,
	}
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
	function, ok := vm.globals[name]
	if !ok {
		return nil, errors.NewDefaultUserError("unknown function '%s'", name)
	}

	return vm.invoke(function, arguments)
}

func (vm *VM) invoke(function Value, arguments []Value) (Value, error) {
	functionValue, ok := function.(FunctionValue)
	if !ok {
		return nil, errors.NewDefaultUserError("not invocable")
	}

	if len(arguments) != int(functionValue.Function.ParameterCount) {
		return nil, errors.NewDefaultUserError(
			"wrong number of arguments: expected %d, found %d",
			functionValue.Function.ParameterCount,
			len(arguments),
		)
	}

	vm.pushCallFrame(functionValue.Context, functionValue.Function, arguments)

	vm.run()

	if len(vm.stack) == 0 {
		return nil, nil
	}

	return vm.pop(), nil
}

func (vm *VM) InitializeContract(arguments ...Value) (*CompositeValue, error) {
	value, err := vm.Invoke(commons.InitFunctionName, arguments...)
	if err != nil {
		return nil, err
	}

	contractValue, ok := value.(*CompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError("invalid contract value")
	}

	return contractValue, nil
}

func (vm *VM) ExecuteTransaction(signers ...Value) error {
	// Create transaction value
	transaction, err := vm.Invoke(commons.TransactionWrapperCompositeName)
	if err != nil {
		return err
	}

	args := []Value{transaction}
	args = append(args, signers...)

	// Invoke 'prepare', if exists.
	if prepare, ok := vm.globals[commons.TransactionPrepareFunctionName]; ok {
		_, err = vm.invoke(prepare, args)
		if err != nil {
			return err
		}
	}

	// TODO: Invoke pre/post conditions

	// Invoke 'execute', if exists.
	if execute, ok := vm.globals[commons.TransactionExecuteFunctionName]; ok {
		_, err = vm.invoke(execute, args)
		return err
	}

	return nil
}

func opReturnValue(vm *VM) {
	value := vm.pop()
	vm.popCallFrame()
	vm.push(value)
}

var voidValue = VoidValue{}

func opReturn(vm *VM) {
	vm.popCallFrame()
	vm.push(voidValue)
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

func opInvoke(vm *VM) {
	value := vm.pop()
	stackHeight := len(vm.stack)

	switch value := value.(type) {
	case FunctionValue:
		parameterCount := int(value.Function.ParameterCount)
		arguments := vm.stack[stackHeight-parameterCount:]
		vm.pushCallFrame(value.Context, value.Function, arguments)
		vm.dropN(parameterCount)
	case NativeFunctionValue:
		parameterCount := value.ParameterCount
		arguments := vm.stack[stackHeight-parameterCount:]
		result := value.Function(arguments...)
		vm.push(result)
	default:
		panic(errors.NewUnreachableError())
	}
}

func opInvokeDynamic(vm *VM) {
	callframe := vm.callFrame
	funcName := callframe.getString()
	argsCount := callframe.getUint16()

	stackHeight := len(vm.stack)
	receiver := vm.stack[stackHeight-int(argsCount)-1]

	compositeValue := receiver.(*CompositeValue)
	qualifiedFuncName := commons.TypeQualifiedName(compositeValue.QualifiedIdentifier, funcName)
	var functionValue = vm.lookupFunction(compositeValue.Location, qualifiedFuncName)

	parameterCount := int(functionValue.Function.ParameterCount)
	arguments := vm.stack[stackHeight-parameterCount:]
	vm.pushCallFrame(functionValue.Context, functionValue.Function, arguments)
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
	fieldNameStr := string(fieldName.Str)

	// TODO: support all container types
	structValue := vm.pop().(*CompositeValue)

	fieldValue := vm.pop()

	structValue.SetMember(vm.config, fieldNameStr, fieldValue)
}

func opGetField(vm *VM) {
	fieldName := vm.pop().(StringValue)
	fieldNameStr := string(fieldName.Str)

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

func opTransfer(vm *VM) {
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

func opPath(vm *VM) {
	callframe := vm.callFrame
	domain := common.PathDomain(callframe.getByte())
	identifier := callframe.getString()
	value := PathValue{
		Domain:     domain,
		Identifier: identifier,
	}
	vm.push(value)
}

func opCast(vm *VM) {
	callframe := vm.callFrame
	value := vm.pop()
	targetType := vm.loadType()
	castType := commons.CastType(callframe.getByte())

	// TODO:
	_ = castType
	_ = targetType

	vm.push(value)
}

func opNil(vm *VM) {
	vm.push(NilValue{})
}

func opEqual(vm *VM) {
	left, right := vm.peekPop()
	vm.replaceTop(BoolValue(left == right))
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
		case opcode.Return:
			opReturn(vm)
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
		case opcode.Invoke:
			opInvoke(vm)
		case opcode.InvokeDynamic:
			opInvokeDynamic(vm)
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
		case opcode.Transfer:
			opTransfer(vm)
		case opcode.Destroy:
			opDestroy(vm)
		case opcode.Path:
			opPath(vm)
		case opcode.Cast:
			opCast(vm)
		case opcode.Nil:
			opNil(vm)
		case opcode.Equal:
			opEqual(vm)
		default:
			panic(errors.NewUnexpectedError("cannot execute opcode '%s'", op.String()))
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
		value = StringValue{Str: constant.Data}
	default:
		// TODO:
		panic(errors.NewUnexpectedError("unsupported constant kind '%s'", constant.Kind.String()))
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

func (vm *VM) lookupFunction(location common.Location, name string) FunctionValue {
	// First check in current program.
	value, ok := vm.globals[name]
	if ok {
		return value.(FunctionValue)
	}

	// If not found, check in already linked imported functions.
	linkedGlobals, ok := vm.linkedGlobalsCache[location]
	if ok {
		value, ok := linkedGlobals.indexedGlobals[name]
		if ok {
			return value.(FunctionValue)
		}
	}

	// If not found, link the function now, dynamically.

	// TODO: This currently link all functions in program, unnecessarily. Link only yhe requested function.
	program := vm.config.ImportHandler(location)
	ctx := NewContext(program, nil)

	indexedGlobals := make(map[string]Value, len(program.Functions))
	for _, function := range program.Functions {
		indexedGlobals[function.Name] = FunctionValue{
			Function: function,
			Context:  ctx,
		}
	}

	vm.linkedGlobalsCache[location] = LinkedGlobals{
		context:        ctx,
		indexedGlobals: indexedGlobals,
	}

	return indexedGlobals[name].(FunctionValue)
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
