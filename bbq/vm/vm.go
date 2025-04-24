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
	"fmt"
	"strings"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
)

type VM struct {
	stack  []Value
	locals []Value

	callstack []callFrame
	callFrame *callFrame

	ipStack []uint16
	ip      uint16

	context            *Context
	globals            map[string]Value
	linkedGlobalsCache map[common.Location]LinkedGlobals
}

func NewVM(
	location common.Location,
	program *bbq.InstructionProgram,
	config *Config,
) *VM {
	// TODO: Remove initializing config. Following is for testing purpose only.
	if config == nil {
		config = &Config{}
	}

	context := NewContext(config)

	if context.storage == nil {
		context.storage = interpreter.NewInMemoryStorage(nil)
	}

	if context.NativeFunctionsProvider == nil {
		context.NativeFunctionsProvider = NativeFunctions
	}

	if context.referencedResourceKindedValues == nil {
		context.referencedResourceKindedValues = ReferencedResourceKindedValues{}
	}

	// linkedGlobalsCache is a local cache-alike that is being used to hold already linked imports.
	linkedGlobalsCache := map[common.Location]LinkedGlobals{
		BuiltInLocation: {
			// It is NOT safe to re-use native functions map here because,
			// once put into the cache, it will be updated by adding the
			// globals of the current program.
			indexedGlobals: context.NativeFunctionsProvider(),
		},
	}

	vm := &VM{
		linkedGlobalsCache: linkedGlobalsCache,
		context:            context,
	}

	// Delegate the function invocations to the vm.
	// TODO: Fix: this should also be able to call native functions.
	context.invokeFunction = vm.invoke

	context.lookupFunction = vm.maybeLookupFunction

	// Link global variables and functions.
	linkedGlobals := LinkGlobals(
		location,
		program,
		context,
		linkedGlobalsCache,
	)

	vm.globals = linkedGlobals.indexedGlobals

	return vm
}

var EmptyLocationRange = interpreter.EmptyLocationRange

func (vm *VM) Context() *Context {
	return vm.context
}

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	vm.stack[lastIndex] = nil
	vm.stack = vm.stack[:lastIndex]

	vm.context.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange)

	return value
}

// pop2 removes and returns the top two values from the stack:
// N-2, and N-1, where N is the number of elements on the stack.
// It is efficient than calling `pop` twice.
func (vm *VM) pop2() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	value1, value2 := vm.stack[lastIndex-1], vm.stack[lastIndex]
	vm.stack[lastIndex-1], vm.stack[lastIndex] = nil, nil
	vm.stack = vm.stack[:lastIndex-1]

	vm.context.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange)
	vm.context.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange)

	return value1, value2
}

// pop3 removes and returns the top three values from the stack:
// N-3, N-2, and N-1, where N is the number of elements on the stack.
// It is efficient than calling `pop` thrice.
func (vm *VM) pop3() (Value, Value, Value) {
	lastIndex := len(vm.stack) - 1
	value1, value2, value3 := vm.stack[lastIndex-2], vm.stack[lastIndex-1], vm.stack[lastIndex]
	vm.stack[lastIndex-2], vm.stack[lastIndex-1], vm.stack[lastIndex] = nil, nil, nil
	vm.stack = vm.stack[:lastIndex-2]

	vm.context.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange)
	vm.context.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange)
	vm.context.CheckInvalidatedResourceOrResourceReference(value3, EmptyLocationRange)

	return value1, value2, value3
}

func (vm *VM) peekN(count int) []Value {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	return vm.stack[startIndex:]
}

func (vm *VM) peek() Value {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex]
}

func (vm *VM) dropN(count int) {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	for _, value := range vm.stack[startIndex:] {
		vm.context.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange)
	}
	clear(vm.stack[startIndex:])
	vm.stack = vm.stack[:startIndex]
}

func (vm *VM) peekPop() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	return vm.stack[lastIndex-1], vm.pop()
}

func (vm *VM) replaceTop(value Value) {
	lastIndex := len(vm.stack) - 1
	vm.stack[lastIndex] = value
}

func fill(slice []Value, n int) []Value {
	for i := 0; i < n; i++ {
		slice = append(slice, nil)
	}
	return slice
}

func (vm *VM) pushCallFrame(functionValue CompiledFunctionValue, arguments []Value) {
	localsCount := functionValue.Function.LocalCount

	vm.locals = append(vm.locals, arguments...)
	vm.locals = fill(vm.locals, int(localsCount)-len(arguments))

	// Calculate the offset for local variable for the new callframe.
	// This is equal to: (local var offset + local var count) of previous callframe.
	var offset uint16
	if len(vm.callstack) > 0 {
		offset = vm.callFrame.localsOffset + vm.callFrame.localsCount

		// store/update the current ip, so it can be resumed.
		vm.ipStack[len(vm.ipStack)-1] = vm.ip
	}

	callFrame := callFrame{
		localsCount:  localsCount,
		localsOffset: offset,
		function:     functionValue,
	}

	vm.ipStack = append(vm.ipStack, 0)
	vm.ip = 0

	vm.callstack = append(vm.callstack, callFrame)
	vm.callFrame = &vm.callstack[len(vm.callstack)-1]
}

func (vm *VM) popCallFrame() {
	// Close all open upvalues before popping the locals.
	// The order of the closing does not matter
	for absoluteLocalsIndex, upvalue := range vm.callFrame.openUpvalues { //nolint:maprange
		upvalue.closed = vm.locals[absoluteLocalsIndex]
	}

	vm.locals = vm.locals[:vm.callFrame.localsOffset]

	newIpStackDepth := len(vm.ipStack) - 1
	vm.ipStack = vm.ipStack[:newIpStackDepth]

	newStackDepth := len(vm.callstack) - 1
	vm.callstack = vm.callstack[:newStackDepth]

	if newStackDepth == 0 {
		vm.ip = 0
	} else {
		vm.ip = vm.ipStack[newIpStackDepth-1]
		vm.callFrame = &vm.callstack[newStackDepth-1]
	}
}

func (vm *VM) Invoke(name string, arguments ...Value) (v Value, err error) {
	function, ok := vm.globals[name]
	if !ok {
		return nil, errors.NewDefaultUserError("unknown function '%s'", name)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		// TODO: pass proper location
		codesAndPrograms := runtime.NewCodesAndPrograms()
		err = runtime.GetWrappedError(recovered, nil, codesAndPrograms)
	}()

	return vm.invoke(function, arguments)
}

func (vm *VM) invoke(function Value, arguments []Value) (Value, error) {
	functionValue, ok := function.(CompiledFunctionValue)
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

	vm.pushCallFrame(functionValue, arguments)

	vm.run()

	if len(vm.stack) == 0 {
		return nil, nil
	}

	return vm.pop(), nil
}

func (vm *VM) InitializeContract(arguments ...Value) (*interpreter.CompositeValue, error) {
	value, err := vm.Invoke(commons.InitFunctionName, arguments...)
	if err != nil {
		return nil, err
	}

	contractValue, ok := value.(*interpreter.CompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError("invalid contract value")
	}

	return contractValue, nil
}

func (vm *VM) ExecuteTransaction(transactionArgs []Value, signers ...Value) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		// TODO: pass proper location
		codesAndPrograms := runtime.NewCodesAndPrograms()
		err = runtime.GetWrappedError(recovered, nil, codesAndPrograms)
	}()

	// Create transaction value
	transaction, err := vm.Invoke(commons.TransactionWrapperCompositeName)
	if err != nil {
		return err
	}

	if initializer, ok := vm.globals[commons.ProgramInitFunctionName]; ok {
		_, err = vm.invoke(initializer, transactionArgs)
		if err != nil {
			return err
		}
	}

	prepareArgs := make([]Value, 0, len(signers)+1)
	prepareArgs = append(prepareArgs, transaction)
	prepareArgs = append(prepareArgs, signers...)

	// Invoke 'prepare', if exists.
	if prepare, ok := vm.globals[commons.TransactionPrepareFunctionName]; ok {
		_, err = vm.invoke(prepare, prepareArgs)
		if err != nil {
			return err
		}
	}

	// TODO: Invoke pre/post conditions

	// Invoke 'execute', if exists.
	executeArgs := []Value{transaction}
	if execute, ok := vm.globals[commons.TransactionExecuteFunctionName]; ok {
		_, err = vm.invoke(execute, executeArgs)
		return err
	}

	return nil
}

func opReturnValue(vm *VM) {
	value := vm.pop()
	vm.popCallFrame()
	vm.push(value)
}

func opReturn(vm *VM) {
	vm.popCallFrame()
	vm.push(interpreter.Void)
}

func opJump(vm *VM, ins opcode.InstructionJump) {
	vm.ip = ins.Target
}

func opJumpIfFalse(vm *VM, ins opcode.InstructionJumpIfFalse) {
	value := vm.pop().(interpreter.BoolValue)
	if !value {
		vm.ip = ins.Target
	}
}

func opJumpIfTrue(vm *VM, ins opcode.InstructionJumpIfTrue) {
	value := vm.pop().(interpreter.BoolValue)
	if value {
		vm.ip = ins.Target
	}
}

func opJumpIfNil(vm *VM, ins opcode.InstructionJumpIfNil) {
	_, ok := vm.pop().(interpreter.NilValue)
	if ok {
		vm.ip = ins.Target
	}
}

func opAdd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Plus(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Minus(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opMultiply(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Mul(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opDivide(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Div(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opMod(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Mod(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opNegate(vm *VM) {
	value := vm.pop().(interpreter.NumberValue)
	vm.push(value.Negate(
		vm.context,
		EmptyLocationRange,
	))
}

func opBitwiseOr(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseOr(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseXor(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseXor(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseAnd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseAnd(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseLeftShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseLeftShift(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseRightShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseRightShift(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	))
}

func opLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Less(vm.context, rightNumber, EmptyLocationRange))
}

func opLessOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.LessEqual(vm.context, rightNumber, EmptyLocationRange))
}

func opGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Greater(vm.context, rightNumber, EmptyLocationRange))
}

func opGreaterOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.GreaterEqual(vm.context, rightNumber, EmptyLocationRange))
}

func opTrue(vm *VM) {
	vm.push(interpreter.TrueValue)
}

func opFalse(vm *VM) {
	vm.push(interpreter.FalseValue)
}

func opGetConstant(vm *VM, ins opcode.InstructionGetConstant) {
	constantIndex := ins.Constant
	constant := vm.callFrame.function.Executable.Constants[constantIndex]
	if constant == nil {
		constant = vm.initializeConstant(constantIndex)
	}
	vm.push(constant)
}

func opGetLocal(vm *VM, ins opcode.InstructionGetLocal) {
	localIndex := ins.Local
	absoluteIndex := vm.callFrame.localsOffset + localIndex
	local := vm.locals[absoluteIndex]
	vm.push(local)
}

func opSetLocal(vm *VM, ins opcode.InstructionSetLocal) {
	localIndex := ins.Local
	absoluteIndex := vm.callFrame.localsOffset + localIndex
	vm.locals[absoluteIndex] = vm.pop()
}

func opGetUpvalue(vm *VM, ins opcode.InstructionGetUpvalue) {
	upvalueIndex := ins.Upvalue
	upvalue := vm.callFrame.function.Upvalues[upvalueIndex]
	value := upvalue.closed
	if value == nil {
		value = vm.locals[upvalue.absoluteLocalsIndex]
	}
	vm.push(value)
}

func opSetUpvalue(vm *VM, ins opcode.InstructionSetUpvalue) {
	upvalueIndex := ins.Upvalue
	upvalue := vm.callFrame.function.Upvalues[upvalueIndex]
	value := vm.pop()
	if upvalue.closed == nil {
		vm.locals[upvalue.absoluteLocalsIndex] = value
	} else {
		upvalue.closed = value
	}
}

func opGetGlobal(vm *VM, ins opcode.InstructionGetGlobal) {
	globalIndex := ins.Global
	globals := vm.callFrame.function.Executable.Globals
	value := globals[globalIndex]
	vm.push(value)
}

func opSetGlobal(vm *VM, ins opcode.InstructionSetGlobal) {
	globalIndex := ins.Global
	globals := vm.callFrame.function.Executable.Globals
	value := vm.pop()
	globals[globalIndex] = value
}

func opSetIndex(vm *VM) {
	container, index, value := vm.pop3()
	containerValue := container.(interpreter.ValueIndexableValue)
	containerValue.SetKey(
		vm.context,
		EmptyLocationRange,
		index,
		value,
	)
}

func opGetIndex(vm *VM) {
	container, index := vm.pop2()
	containerValue := container.(interpreter.ValueIndexableValue)
	element := containerValue.GetKey(
		vm.context,
		EmptyLocationRange,
		index,
	)
	vm.push(element)
}

func opInvoke(vm *VM, ins opcode.InstructionInvoke) {
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	functionValue := vm.pop()

	explicitArgumentsCount := int(ins.ArgCount)
	arguments := vm.peekN(explicitArgumentsCount)

	// If the function is a pointer to an object-method, then the receiver is implicitly captured.
	if boundFunction, isBoundFUnction := functionValue.(BoundFunctionPointerValue); isBoundFUnction {
		functionValue = boundFunction.Method
		receiver := unwrapReceiver(vm.context, boundFunction.Receiver)
		arguments = append([]Value{receiver}, arguments...)
	}

	invokeFunction(
		vm,
		functionValue,
		explicitArgumentsCount,
		arguments,
		typeArguments,
	)
}

func opInvokeMethodStatic(vm *VM, ins opcode.InstructionInvokeMethodStatic) {
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	functionValue := vm.pop()

	explicitArgumentsCount := int(ins.ArgCount)
	arguments := vm.peekN(explicitArgumentsCount)
	receiver := arguments[receiverIndex]
	arguments[receiverIndex] = unwrapReceiver(vm.context, receiver)

	invokeFunction(
		vm,
		functionValue,
		int(ins.ArgCount),
		arguments,
		typeArguments,
	)
}

func opInvokeMethodDynamic(vm *VM, ins opcode.InstructionInvokeMethodDynamic) {
	// TODO: This method is now equivalent to: `GetField` + `Invoke` instructions.
	// See if it can be replaced. That will reduce the complexity of `invokeFunction` method below.

	// Load type arguments
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	explicitArgumentsCount := int(ins.ArgCount)
	arguments := vm.peekN(explicitArgumentsCount)
	receiver := arguments[receiverIndex]
	arguments[receiverIndex] = unwrapReceiver(vm.context, receiver)

	// Get function
	nameIndex := ins.Name
	funcName := getStringConstant(vm, nameIndex)

	memberAccessibleValue := receiver.(interpreter.MemberAccessibleValue)
	functionValue := memberAccessibleValue.GetMember(
		vm.context,
		EmptyLocationRange,
		funcName,
	).(BoundFunctionPointerValue)

	invokeFunction(
		vm,
		functionValue.Method,
		explicitArgumentsCount,
		arguments,
		typeArguments,
	)
}

func invokeFunction(
	vm *VM,
	functionValue Value,
	explicitArgumentsCount int,
	arguments []Value,
	typeArguments []bbq.StaticType,
) {

	switch functionValue := functionValue.(type) {
	case CompiledFunctionValue:
		vm.pushCallFrame(functionValue, arguments)
		vm.dropN(explicitArgumentsCount)

	case NativeFunctionValue:
		result := functionValue.Function(vm.context, typeArguments, arguments...)
		vm.dropN(explicitArgumentsCount)
		vm.push(result)

	default:
		panic(errors.NewUnreachableError())
	}

	// We do not need to drop the receiver explicitly,
	// as the `explicitArgumentsCount` already includes it.
}

func loadTypeArguments(vm *VM, typeArgs []uint16) []bbq.StaticType {
	var typeArguments []bbq.StaticType
	if len(typeArgs) > 0 {
		typeArguments = make([]bbq.StaticType, 0, len(typeArgs))
		for _, typeIndex := range typeArgs {
			typeArg := vm.loadType(typeIndex)
			typeArguments = append(typeArguments, typeArg)
		}
	}
	return typeArguments
}

func unwrapReceiver(context *Context, receiver Value) Value {
	for {
		switch typedReceiver := receiver.(type) {
		case *interpreter.SomeValue:
			receiver = typedReceiver.InnerValue()
		case *interpreter.EphemeralReferenceValue:
			receiver = typedReceiver.Value
		case *interpreter.StorageReferenceValue:
			referencedValue := typedReceiver.ReferencedValue(
				context,
				EmptyLocationRange,
				true,
			)
			receiver = *referencedValue
		default:
			return receiver
		}
	}
}

func opDrop(vm *VM) {
	_ = vm.pop()
}

func opDup(vm *VM) {
	top := vm.peek()
	vm.push(top)
}

func opNew(vm *VM, ins opcode.InstructionNew) {
	compositeKind := ins.Kind

	// decode location
	typeIndex := ins.Type
	staticType := vm.loadType(typeIndex)

	// TODO: Support inclusive-range type
	compositeStaticType := staticType.(*interpreter.CompositeStaticType)

	config := vm.context

	compositeFields := newCompositeValueFields(config, compositeKind)

	value := interpreter.NewCompositeValue(
		config,
		EmptyLocationRange,
		compositeStaticType.Location,
		compositeStaticType.QualifiedIdentifier,
		compositeKind,
		compositeFields,
		// Newly created values are always on stack.
		// Need to 'Transfer' if needed to be stored in an account.
		common.ZeroAddress,
	)
	vm.push(value)
}

func opSetField(vm *VM, ins opcode.InstructionSetField) {
	target, fieldValue := vm.pop2()

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getStringConstant(vm, fieldNameIndex)

	target.(interpreter.MemberAccessibleValue).
		SetMember(vm.context, EmptyLocationRange, fieldName, fieldValue)
}

func opGetField(vm *VM, ins opcode.InstructionGetField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getStringConstant(vm, fieldNameIndex)

	fieldValue := memberAccessibleValue.GetMember(vm.context, EmptyLocationRange, fieldName)
	if fieldValue == nil {
		panic(MissingMemberValueError{
			Parent: memberAccessibleValue,
			Name:   fieldName,
		})
	}

	vm.push(fieldValue)
}

func getStringConstant(vm *VM, index uint16) string {
	constant := vm.callFrame.function.Executable.Program.Constants[index]
	return string(constant.Data)
}

func opTransfer(vm *VM, ins opcode.InstructionTransfer) {
	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)
	value := vm.peek()

	config := vm.context

	transferredValue := value.Transfer(
		config,
		EmptyLocationRange,
		atree.Address{},
		false,
		nil,
		nil,

		// TODO: Pass the correct flag here
		false,
	)

	valueType := transferredValue.StaticType(config)
	// TODO: remove nil check after ensuring all implementations of Value.StaticType are implemented
	if valueType != nil && !vm.context.IsSubType(valueType, targetType) {
		panic(errors.NewUnexpectedError(
			"invalid transfer: expected '%s', found '%s'",
			targetType,
			valueType,
		))
	}

	vm.replaceTop(transferredValue)
}

func opDestroy(vm *VM) {
	value := vm.pop().(interpreter.ResourceKindedValue)
	value.Destroy(vm.context, EmptyLocationRange)
}

func opNewPath(vm *VM, ins opcode.InstructionNewPath) {
	identifierIndex := ins.Identifier
	identifier := getStringConstant(vm, identifierIndex)
	value := interpreter.NewPathValue(
		vm.context.MemoryGauge,
		ins.Domain,
		identifier,
	)
	vm.push(value)
}

func opSimpleCast(vm *VM, ins opcode.InstructionSimpleCast) {
	value := vm.pop()

	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)
	valueType := value.StaticType(vm.context)

	// The cast may upcast to an optional type, e.g. `1 as Int?`, so box
	result := ConvertAndBox(vm.context, value, valueType, targetType)

	vm.push(result)
}

func opFailableCast(vm *VM, ins opcode.InstructionFailableCast) {
	value := vm.pop()

	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)

	value, valueType := castValueAndValueType(vm.context, targetType, value)

	isSubType := vm.context.IsSubType(valueType, targetType)

	var result Value
	if isSubType {
		// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
		result = ConvertAndBox(vm.context, value, valueType, targetType)

		// TODO:
		// Failable casting is a resource invalidation
		//interpreter.invalidateResource(value)

		result = interpreter.NewSomeValueNonCopying(
			vm.context.MemoryGauge,
			result,
		)
	} else {
		result = interpreter.Nil
	}

	vm.push(result)
}

func opForceCast(vm *VM, ins opcode.InstructionForceCast) {
	value := vm.pop()

	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)

	value, valueType := castValueAndValueType(vm.context, targetType, value)

	isSubType := vm.context.IsSubType(valueType, targetType)

	var result Value
	if !isSubType {
		targetSemaType := interpreter.MustConvertStaticToSemaType(targetType, vm.context)
		valueSemaType := interpreter.MustConvertStaticToSemaType(valueType, vm.context)

		panic(interpreter.ForceCastTypeMismatchError{
			ExpectedType: targetSemaType,
			ActualType:   valueSemaType,
		})
	}

	// The force cast may upcast to an optional type, e.g. `1 as! Int?`, so box
	result = ConvertAndBox(vm.context, value, valueType, targetType)
	vm.push(result)
}

func castValueAndValueType(context *Context, targetType bbq.StaticType, value Value) (Value, bbq.StaticType) {
	valueType := value.StaticType(context)

	// if the value itself has a mapped entitlement type in its authorization
	// (e.g. if it is a reference to `self` or `base`  in an attachment function with mapped access)
	// substitution must also be performed on its entitlements
	//
	// we do this here (as opposed to in `IsSubTypeOfSemaType`) because casting is the only way that
	// an entitlement can "traverse the boundary", so to speak, between runtime and static types, and
	// thus this is the only place where it becomes necessary to "instantiate" the result of a map to its
	// concrete outputs. In other places (e.g. interface conformance checks) we want to leave maps generic,
	// so we don't substitute them.

	// TODO: Substitute entitlements
	//valueSemaType := interpreter.SubstituteMappedEntitlements(interpreter.MustSemaTypeOfValue(value))
	//valueType = ConvertSemaToStaticType(interpreter, valueSemaType)

	// If the target is anystruct or anyresource we want to preserve optionals
	unboxedExpectedType := UnwrapOptionalType(targetType)
	if !(unboxedExpectedType == interpreter.PrimitiveStaticTypeAnyStruct ||
		unboxedExpectedType == interpreter.PrimitiveStaticTypeAnyResource) {
		// otherwise dynamic cast now always unboxes optionals
		value = interpreter.Unbox(value)
	}

	return value, valueType
}

func opNil(vm *VM) {
	vm.push(interpreter.Nil)
}

func opEqual(vm *VM) {
	left, right := vm.peekPop()
	result := left.(interpreter.EquatableValue).Equal(
		vm.context,
		EmptyLocationRange,
		right,
	)
	vm.replaceTop(interpreter.BoolValue(result))
}

func opNotEqual(vm *VM) {
	left, right := vm.peekPop()
	result := !left.(interpreter.EquatableValue).Equal(
		vm.context,
		EmptyLocationRange,
		right,
	)
	vm.replaceTop(interpreter.BoolValue(result))
}

func opNot(vm *VM) {
	value := vm.peek().(interpreter.BoolValue)
	vm.replaceTop(!value)
}

func opUnwrap(vm *VM) {
	value := vm.peek()
	switch value := value.(type) {
	case *interpreter.SomeValue:
		vm.replaceTop(value.InnerValue())
	case interpreter.NilValue:
		panic(ForceNilError{})
	default:
		// Non-optional. Leave as is.
	}
}

func opNewArray(vm *VM, ins opcode.InstructionNewArray) {
	typeIndex := ins.Type
	typ := vm.loadType(typeIndex).(interpreter.ArrayStaticType)

	elements := vm.peekN(int(ins.Size))
	array := interpreter.NewArrayValue(
		vm.context,
		EmptyLocationRange,
		typ,

		// Newly created values are always on stack.
		// Need to 'Transfer' if needed to be stored in an account.
		common.ZeroAddress,

		elements...,
	)

	vm.dropN(len(elements))

	vm.push(array)
}

func opNewDictionary(vm *VM, ins opcode.InstructionNewDictionary) {
	typeIndex := ins.Type
	typ := vm.loadType(typeIndex).(*interpreter.DictionaryStaticType)

	entries := vm.peekN(int(ins.Size * 2))
	dictionary := interpreter.NewDictionaryValue(
		vm.context,
		EmptyLocationRange,
		typ,
		entries...,
	)
	vm.dropN(len(entries))

	vm.push(dictionary)
}

func opNewRef(vm *VM, ins opcode.InstructionNewRef) {
	typeIndex := ins.Type
	borrowedType := vm.loadType(typeIndex)
	value := vm.pop()

	semaBorrowedType := interpreter.MustConvertStaticToSemaType(borrowedType, vm.context)

	ref := interpreter.CreateReferenceValue(
		vm.context,
		semaBorrowedType,
		value,
		EmptyLocationRange,
		ins.IsImplicit,
	)

	vm.push(ref)
}

func opIterator(vm *VM) {
	value := vm.pop()
	iterable := value.(interpreter.IterableValue)
	iterator := iterable.Iterator(vm.context, EmptyLocationRange)
	vm.push(NewIteratorWrapperValue(iterator))
}

func opIteratorHasNext(vm *VM) {
	value := vm.pop()
	iterator := value.(*IteratorWrapperValue)
	vm.push(interpreter.BoolValue(iterator.HasNext()))
}

func opIteratorNext(vm *VM) {
	value := vm.pop()
	iterator := value.(*IteratorWrapperValue)
	element := iterator.Next(vm.context, EmptyLocationRange)
	vm.push(element)
}

func deref(vm *VM, value Value) Value {
	if _, ok := value.(interpreter.NilValue); ok {
		return interpreter.Nil
	}

	var isOptional bool

	if someValue, ok := value.(*interpreter.SomeValue); ok {
		isOptional = true
		value = someValue.InnerValue()
	}

	referenceValue, ok := value.(interpreter.ReferenceValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// TODO: port and use interpreter.DereferenceValue
	dereferencedValue := *referenceValue.ReferencedValue(
		vm.context,
		EmptyLocationRange,
		true,
	)

	if isOptional {
		return interpreter.NewSomeValueNonCopying(
			vm.context.MemoryGauge,
			dereferencedValue,
		)
	} else {
		return dereferencedValue
	}
}

func opDeref(vm *VM) {
	value := vm.pop()
	dereferenced := deref(vm, value)
	vm.push(dereferenced)
}

func (vm *VM) run() {

	if vm.context.debugEnabled {
		defer func() {
			if r := recover(); r != nil {
				printInstructionError(
					vm.callFrame.function.Function,
					int(vm.ip),
					r,
				)
				panic(r)
			}
		}()
	}

	for {

		callFrame := vm.callFrame

		code := callFrame.function.Function.Code
		if len(vm.callstack) == 0 ||
			int(vm.ip) >= len(code) {

			return
		}

		ins := code[vm.ip]
		vm.ip++

		switch ins := ins.(type) {
		case opcode.InstructionReturnValue:
			opReturnValue(vm)
		case opcode.InstructionReturn:
			opReturn(vm)
		case opcode.InstructionJump:
			opJump(vm, ins)
		case opcode.InstructionJumpIfFalse:
			opJumpIfFalse(vm, ins)
		case opcode.InstructionJumpIfTrue:
			opJumpIfTrue(vm, ins)
		case opcode.InstructionJumpIfNil:
			opJumpIfNil(vm, ins)
		case opcode.InstructionAdd:
			opAdd(vm)
		case opcode.InstructionSubtract:
			opSubtract(vm)
		case opcode.InstructionMultiply:
			opMultiply(vm)
		case opcode.InstructionDivide:
			opDivide(vm)
		case opcode.InstructionMod:
			opMod(vm)
		case opcode.InstructionNegate:
			opNegate(vm)
		case opcode.InstructionBitwiseOr:
			opBitwiseOr(vm)
		case opcode.InstructionBitwiseXor:
			opBitwiseXor(vm)
		case opcode.InstructionBitwiseAnd:
			opBitwiseAnd(vm)
		case opcode.InstructionBitwiseLeftShift:
			opBitwiseLeftShift(vm)
		case opcode.InstructionBitwiseRightShift:
			opBitwiseRightShift(vm)
		case opcode.InstructionLess:
			opLess(vm)
		case opcode.InstructionLessOrEqual:
			opLessOrEqual(vm)
		case opcode.InstructionGreater:
			opGreater(vm)
		case opcode.InstructionGreaterOrEqual:
			opGreaterOrEqual(vm)
		case opcode.InstructionTrue:
			opTrue(vm)
		case opcode.InstructionFalse:
			opFalse(vm)
		case opcode.InstructionGetConstant:
			opGetConstant(vm, ins)
		case opcode.InstructionGetLocal:
			opGetLocal(vm, ins)
		case opcode.InstructionSetLocal:
			opSetLocal(vm, ins)
		case opcode.InstructionGetUpvalue:
			opGetUpvalue(vm, ins)
		case opcode.InstructionSetUpvalue:
			opSetUpvalue(vm, ins)
		case opcode.InstructionGetGlobal:
			opGetGlobal(vm, ins)
		case opcode.InstructionSetGlobal:
			opSetGlobal(vm, ins)
		case opcode.InstructionSetIndex:
			opSetIndex(vm)
		case opcode.InstructionGetIndex:
			opGetIndex(vm)
		case opcode.InstructionInvoke:
			opInvoke(vm, ins)
		case opcode.InstructionInvokeMethodStatic:
			opInvokeMethodStatic(vm, ins)
		case opcode.InstructionInvokeMethodDynamic:
			opInvokeMethodDynamic(vm, ins)
		case opcode.InstructionDrop:
			opDrop(vm)
		case opcode.InstructionDup:
			opDup(vm)
		case opcode.InstructionNew:
			opNew(vm, ins)
		case opcode.InstructionNewArray:
			opNewArray(vm, ins)
		case opcode.InstructionNewDictionary:
			opNewDictionary(vm, ins)
		case opcode.InstructionNewRef:
			opNewRef(vm, ins)
		case opcode.InstructionSetField:
			opSetField(vm, ins)
		case opcode.InstructionGetField:
			opGetField(vm, ins)
		case opcode.InstructionTransfer:
			opTransfer(vm, ins)
		case opcode.InstructionDestroy:
			opDestroy(vm)
		case opcode.InstructionNewPath:
			opNewPath(vm, ins)
		case opcode.InstructionSimpleCast:
			opSimpleCast(vm, ins)
		case opcode.InstructionFailableCast:
			opFailableCast(vm, ins)
		case opcode.InstructionForceCast:
			opForceCast(vm, ins)
		case opcode.InstructionNil:
			opNil(vm)
		case opcode.InstructionEqual:
			opEqual(vm)
		case opcode.InstructionNotEqual:
			opNotEqual(vm)
		case opcode.InstructionNot:
			opNot(vm)
		case opcode.InstructionUnwrap:
			opUnwrap(vm)
		case opcode.InstructionEmitEvent:
			onEmitEvent(vm, ins)
		case opcode.InstructionIterator:
			opIterator(vm)
		case opcode.InstructionIteratorHasNext:
			opIteratorHasNext(vm)
		case opcode.InstructionIteratorNext:
			opIteratorNext(vm)
		case opcode.InstructionDeref:
			opDeref(vm)
		case opcode.InstructionNewClosure:
			opNewClosure(vm, ins)
		default:
			panic(errors.NewUnexpectedError("cannot execute instruction of type %T", ins))
		}
	}
}

func onEmitEvent(vm *VM, ins opcode.InstructionEmitEvent) {
	eventValue := vm.pop().(*interpreter.CompositeValue)

	onEventEmitted := vm.context.OnEventEmitted
	if onEventEmitted == nil {
		return
	}

	typeIndex := ins.Type
	eventType := vm.loadType(typeIndex).(*interpreter.CompositeStaticType)

	err := onEventEmitted(eventValue, eventType)
	if err != nil {
		panic(err)
	}
}

func opNewClosure(vm *VM, ins opcode.InstructionNewClosure) {

	executable := vm.callFrame.function.Executable
	functionIndex := ins.Function
	function := &executable.Program.Functions[functionIndex]

	var upvalues []*Upvalue
	upvalueCount := len(ins.Upvalues)
	if upvalueCount > 0 {
		upvalues = make([]*Upvalue, upvalueCount)
	}

	for upvalueIndex, upvalueDescriptor := range ins.Upvalues {
		targetIndex := upvalueDescriptor.TargetIndex
		var upvalue *Upvalue
		if upvalueDescriptor.IsLocal {
			absoluteLocalsIndex := int(vm.callFrame.localsOffset) + int(targetIndex)
			upvalue = vm.captureUpvalue(absoluteLocalsIndex)
		} else {
			upvalue = vm.callFrame.function.Upvalues[targetIndex]
		}
		upvalues[upvalueIndex] = upvalue
	}

	funcStaticType := getTypeFromExecutable[interpreter.FunctionStaticType](executable, function.TypeIndex)

	vm.push(CompiledFunctionValue{
		Function:   function,
		Executable: executable,
		Upvalues:   upvalues,
		Type:       funcStaticType,
	})
}

func (vm *VM) captureUpvalue(absoluteLocalsIndex int) *Upvalue {
	// Check if the upvalue already exists and reuse it
	if upvalue, ok := vm.callFrame.openUpvalues[absoluteLocalsIndex]; ok {
		return upvalue
	}

	// Create a new upvalue and record it as open
	upvalue := &Upvalue{
		absoluteLocalsIndex: absoluteLocalsIndex,
	}
	if vm.callFrame.openUpvalues == nil {
		vm.callFrame.openUpvalues = make(map[int]*Upvalue)
	}
	vm.callFrame.openUpvalues[absoluteLocalsIndex] = upvalue
	return upvalue
}

func (vm *VM) initializeConstant(index uint16) (value Value) {
	executable := vm.callFrame.function.Executable

	c := executable.Program.Constants[index]
	memoryGauge := vm.context.MemoryGauge

	switch c.Kind {
	case constant.String:
		value = interpreter.NewUnmeteredStringValue(string(c.Data))

	case constant.Int:
		// TODO: support larger integers
		v, _, err := leb128.ReadInt64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int constant: %s", err))
		}
		value = interpreter.NewIntValueFromInt64(memoryGauge, v)

	case constant.Int8:
		v, _, err := leb128.ReadInt32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int8 constant: %s", err))
		}
		value = interpreter.NewInt8Value(
			memoryGauge,
			func() int8 {
				return int8(v)
			},
		)

	case constant.Int16:
		v, _, err := leb128.ReadInt32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int16 constant: %s", err))
		}
		value = interpreter.NewInt16Value(
			memoryGauge,
			func() int16 {
				return int16(v)
			},
		)

	case constant.Int32:
		v, _, err := leb128.ReadInt32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int32 constant: %s", err))
		}
		value = interpreter.NewInt32Value(
			memoryGauge,
			func() int32 {
				return v
			},
		)

	case constant.Int64:
		v, _, err := leb128.ReadInt64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int64 constant: %s", err))
		}
		value = interpreter.NewInt64Value(
			memoryGauge,
			func() int64 {
				return v
			},
		)

	case constant.UInt:
		// TODO: support larger integers
		v, _, err := leb128.ReadUint64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt constant: %s", err))
		}
		value = interpreter.NewUIntValueFromUint64(memoryGauge, v)

	case constant.UInt8:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt8 constant: %s", err))
		}
		value = interpreter.NewUInt8Value(
			memoryGauge,
			func() uint8 {
				return uint8(v)
			},
		)

	case constant.UInt16:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt16 constant: %s", err))
		}
		value = interpreter.NewUInt16Value(
			memoryGauge,
			func() uint16 {
				return uint16(v)
			},
		)

	case constant.UInt32:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt32 constant: %s", err))
		}
		value = interpreter.NewUInt32Value(
			memoryGauge,
			func() uint32 {
				return v
			},
		)

	case constant.UInt64:
		v, _, err := leb128.ReadUint64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt64 constant: %s", err))
		}
		value = interpreter.NewUInt64Value(
			memoryGauge,
			func() uint64 {
				return v
			},
		)

	case constant.Word8:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word8 constant: %s", err))
		}
		value = interpreter.NewWord8Value(
			memoryGauge,
			func() uint8 {
				return uint8(v)
			},
		)

	case constant.Word16:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word16 constant: %s", err))
		}
		value = interpreter.NewWord16Value(
			memoryGauge,
			func() uint16 {
				return uint16(v)
			},
		)

	case constant.Word32:
		v, _, err := leb128.ReadUint32(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word32 constant: %s", err))
		}
		value = interpreter.NewWord32Value(
			memoryGauge,
			func() uint32 {
				return v
			},
		)

	case constant.Word64:
		v, _, err := leb128.ReadUint64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word64 constant: %s", err))
		}
		value = interpreter.NewWord64Value(
			memoryGauge,
			func() uint64 {
				return v
			},
		)

	case constant.Fix64:
		v, _, err := leb128.ReadInt64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Fix64 constant: %s", err))
		}
		value = interpreter.NewUnmeteredFix64Value(v)

	case constant.UFix64:
		v, _, err := leb128.ReadUint64(c.Data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UFix64 constant: %s", err))
		}
		value = interpreter.NewUnmeteredUFix64Value(v)

	case constant.Address:
		value = interpreter.NewAddressValueFromBytes(
			memoryGauge,
			func() []byte {
				return c.Data
			},
		)

	// TODO:
	// case constantkind.Int128:
	// case constantkind.Int256:
	// case constantkind.UInt128:
	// case constantkind.UInt256:
	// case constantkind.Word128:
	// case constantkind.Word256:

	default:
		panic(errors.NewUnexpectedError("unsupported constant kind: %s", c.Kind))
	}

	executable.Constants[index] = value

	return value
}

func (vm *VM) loadType(index uint16) bbq.StaticType {
	staticType := vm.callFrame.function.Executable.StaticTypes[index]
	if staticType == nil {
		// Should never reach.
		panic(errors.NewUnreachableError())
	}

	return staticType
}

func (vm *VM) maybeLookupFunction(location common.Location, name string) FunctionValue {
	funcValue, ok := vm.lookupFunction(location, name)
	if !ok {
		return nil
	}
	return funcValue
}

func (vm *VM) lookupFunction(location common.Location, name string) (FunctionValue, bool) {
	// First check in current program.
	value, ok := vm.globals[name]
	if ok {
		return value.(FunctionValue), true
	}

	// If not found, check in already linked imported functions.
	linkedGlobals, ok := vm.linkedGlobalsCache[location]

	// If not found, link the function now, dynamically.
	if !ok {
		// TODO: This currently link all functions in program, unnecessarily.
		//   Link only the requested function.
		program := vm.context.ImportHandler(location)

		linkedGlobals = LinkGlobals(
			location,
			program,
			vm.context,
			vm.linkedGlobalsCache,
		)
	}

	value, ok = linkedGlobals.indexedGlobals[name]
	if !ok {
		return nil, false
	}

	return value.(FunctionValue), true
}

func (vm *VM) StackSize() int {
	return len(vm.stack)
}

func (vm *VM) Reset() {
	vm.stack = vm.stack[:0]
	vm.locals = vm.locals[:0]
	vm.callstack = vm.callstack[:0]
	vm.ipStack = vm.ipStack[:0]
}

func printInstructionError(
	function *bbq.Function[opcode.Instruction],
	instructionIndex int,
	error any,
) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("-- %s -- \n", function.QualifiedName))

	for index, instruction := range function.Code {
		if index == instructionIndex {
			builder.WriteString(fmt.Sprintf("^^^^^^^^^^ %s\n", error))
		}

		_, _ = fmt.Fprint(&builder, instruction)
		builder.WriteByte('\n')
	}

	fmt.Println(builder.String())
}
