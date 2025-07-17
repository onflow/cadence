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

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type Variable = interpreter.Variable

type VM struct {
	stack  []Value
	locals []Value

	callstack []callFrame
	callFrame *callFrame

	ipStack []uint16
	ip      uint16

	context            *Context
	globals            *activations.Activation[Variable]
	linkedGlobalsCache map[common.Location]LinkedGlobals
}

func NewVM(
	location common.Location,
	program *bbq.InstructionProgram,
	config *Config,
) *VM {

	context := NewContext(config)

	if context.referencedResourceKindedValues == nil {
		context.referencedResourceKindedValues = ReferencedResourceKindedValues{}
	}

	// linkedGlobalsCache is a local cache-alike that is being used to hold already linked imports.
	linkedGlobalsCache := map[common.Location]LinkedGlobals{}

	vm := &VM{
		linkedGlobalsCache: linkedGlobalsCache,
		context:            context,
	}

	// Delegate the function invocations to the vm.
	context.invokeFunction = vm.invokeFunction
	context.lookupFunction = vm.lookupFunction

	// Link global variables and functions.
	linkedGlobals := LinkGlobals(
		config.MemoryGauge,
		location,
		program,
		context,
		linkedGlobalsCache,
	)

	vm.globals = linkedGlobals.indexedGlobals

	vm.initializeGlobalVariables(program)

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

	interpreter.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange, vm.context)

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

	interpreter.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange, vm.context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange, vm.context)

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

	interpreter.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange, vm.context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange, vm.context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value3, EmptyLocationRange, vm.context)

	return value1, value2, value3
}

func (vm *VM) peekN(count int) []Value {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	values := vm.stack[startIndex:]

	for _, value := range values {
		interpreter.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange, vm.context)
	}

	return values
}

func (vm *VM) popN(count int) []Value {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	values := vm.stack[startIndex:]

	for _, value := range values {
		interpreter.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange, vm.context)
	}

	vm.stack = vm.stack[:startIndex]

	return values
}

func (vm *VM) peek() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	interpreter.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange, vm.context)
	return value
}

func (vm *VM) dropN(count int) {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	clear(vm.stack[startIndex:])
	vm.stack = vm.stack[:startIndex]
}

func (vm *VM) peekPop() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	peekedValue := vm.stack[lastIndex-1]
	poppedValue := vm.pop()

	interpreter.CheckInvalidatedResourceOrResourceReference(peekedValue, EmptyLocationRange, vm.context)
	interpreter.CheckInvalidatedResourceOrResourceReference(poppedValue, EmptyLocationRange, vm.context)

	return peekedValue, poppedValue
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

func (vm *VM) pushCallFrame(functionValue CompiledFunctionValue, receiver Value, arguments []Value) {
	if uint64(len(vm.callstack)) == vm.context.StackDepthLimit {
		panic(&interpreter.CallStackLimitExceededError{
			Limit: vm.context.StackDepthLimit,
		})
	}

	localsCount := functionValue.Function.LocalCount

	passedInLocalsCount := len(arguments)
	if receiver != nil {
		vm.locals = append(vm.locals, receiver)
		passedInLocalsCount++
	}

	vm.locals = append(vm.locals, arguments...)
	vm.locals = fill(vm.locals, int(localsCount)-passedInLocalsCount)

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

func RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		err := interpreter.AsCadenceError(r)
		onError(err)
	}
}

func (vm *VM) InvokeExternally(name string, arguments ...Value) (v Value, err error) {
	functionVariable := vm.globals.Find(name)
	if functionVariable == nil {
		return nil, UnknownFunctionError{
			name: name,
		}
	}

	defer RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	function := functionVariable.GetValue(vm.context)

	functionValue, ok := function.(FunctionValue)
	if !ok {
		return nil, interpreter.NotInvokableError{
			Value: function,
		}
	}

	return vm.validateAndInvokeExternally(functionValue, arguments)
}

func (vm *VM) InvokeMethodExternally(
	name string,
	receiver interpreter.MemberAccessibleValue,
	arguments ...Value,
) (
	v Value,
	err error,
) {
	functionVariable := vm.globals.Find(name)
	if functionVariable == nil {
		return nil, UnknownFunctionError{
			name: name,
		}
	}

	defer RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	context := vm.context

	function := functionVariable.GetValue(context)

	functionValue, ok := function.(FunctionValue)
	if !ok {
		return nil, interpreter.NotInvokableError{
			Value: function,
		}
	}

	boundFunction := NewBoundFunctionValue(context, receiver, functionValue)

	return vm.validateAndInvokeExternally(boundFunction, arguments)
}

func (vm *VM) validateAndInvokeExternally(functionValue FunctionValue, arguments []Value) (Value, error) {
	context := vm.context

	functionType := functionValue.FunctionType(context)

	preparedArguments, err := interpreter.PrepareExternalInvocationArguments(
		context,
		functionType,
		arguments,
	)
	if err != nil {
		return nil, err
	}

	return vm.invokeExternally(functionValue, preparedArguments)
}

func (vm *VM) invokeExternally(functionValue Value, arguments []Value) (Value, error) {
	invokeFunction(
		vm,
		functionValue,
		arguments,
		nil,
	)

	vm.run()

	if len(vm.stack) == 0 {
		return nil, nil
	}

	return vm.pop(), nil
}

func (vm *VM) InitializeContract(contractName string, arguments ...Value) (*interpreter.CompositeValue, error) {
	contractInitializer := commons.QualifiedName(contractName, commons.InitFunctionName)
	value, err := vm.InvokeExternally(contractInitializer, arguments...)
	if err != nil {
		return nil, err
	}

	contractValue, ok := value.(*interpreter.CompositeValue)
	if !ok {
		return nil, errors.NewUnexpectedError("invalid contract value")
	}

	return contractValue, nil
}

func (vm *VM) InvokeTransaction(arguments []Value, signers ...Value) (err error) {

	defer RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	// Create transaction value
	transaction, err := vm.InvokeTransactionWrapper()
	if err != nil {
		return err
	}

	err = vm.InvokeTransactionInit(arguments)
	if err != nil {
		return err
	}

	// Invoke 'prepare', if exists.
	err = vm.InvokeTransactionPrepare(transaction, signers)
	if err != nil {
		return err
	}

	// Invoke 'execute', if exists.
	// NOTE: pre and post conditions of the transaction were already
	// desugared into the execution function.
	// If no `execute` function was defined, a synthetic one was created.
	err = vm.InvokeTransactionExecute(transaction)
	if err != nil {
		return err
	}

	return nil
}

func (vm *VM) InvokeTransactionWrapper() (*interpreter.CompositeValue, error) {
	wrapperResult, err := vm.InvokeExternally(commons.TransactionWrapperCompositeName)
	if err != nil {
		return nil, err
	}

	transaction := wrapperResult.(*interpreter.CompositeValue)

	return transaction, nil
}

func (vm *VM) InvokeTransactionInit(transactionArgs []Value) error {
	context := vm.context
	globals := vm.globals

	initializerVariable := globals.Find(commons.ProgramInitFunctionName)
	if initializerVariable == nil {
		if len(transactionArgs) > 0 {
			return interpreter.ArgumentCountError{
				ParameterCount: 0,
				ArgumentCount:  len(transactionArgs),
			}
		}

		return nil
	}

	initializer := initializerVariable.GetValue(context).(FunctionValue)

	_, err := vm.validateAndInvokeExternally(initializer, transactionArgs)
	if err != nil {
		return err
	}

	return nil
}

func (vm *VM) InvokeTransactionPrepare(transaction *interpreter.CompositeValue, signers []Value) error {
	context := vm.context

	prepareVariable := vm.globals.Find(commons.TransactionPrepareFunctionName)
	if prepareVariable == nil {
		if len(signers) > 0 {
			return interpreter.ArgumentCountError{
				ParameterCount: 0,
				ArgumentCount:  len(signers),
			}
		}

		return nil
	}

	prepareValue := prepareVariable.GetValue(context)
	prepareFunction := prepareValue.(FunctionValue)
	boundPrepareFunction := NewBoundFunctionValue(
		context,
		transaction,
		prepareFunction,
	)

	_, err := vm.validateAndInvokeExternally(boundPrepareFunction, signers)
	if err != nil {
		return err
	}

	return nil
}

func (vm *VM) InvokeTransactionExecute(transaction *interpreter.CompositeValue) error {
	context := vm.context

	executeVariable := vm.globals.Find(commons.TransactionExecuteFunctionName)
	if executeVariable == nil {
		return nil
	}

	executeValue := executeVariable.GetValue(context)
	executeFunction := executeValue.(FunctionValue)
	boundExecuteFunction := NewBoundFunctionValue(
		context,
		transaction,
		executeFunction,
	)

	_, err := vm.validateAndInvokeExternally(boundExecuteFunction, nil)
	if err != nil {
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
	result := leftNumber.Plus(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Minus(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opMultiply(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Mul(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opDivide(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Div(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opMod(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Mod(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opNegate(vm *VM) {
	value := vm.pop().(interpreter.NumberValue)
	result := value.Negate(
		vm.context,
		EmptyLocationRange,
	)
	vm.push(result)
}

func opBitwiseOr(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseOr(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opBitwiseXor(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseXor(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opBitwiseAnd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseAnd(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opBitwiseLeftShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseLeftShift(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opBitwiseRightShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseRightShift(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.Less(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opLessOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.LessEqual(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.Greater(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opGreaterOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.GreaterEqual(
		vm.context,
		rightNumber,
		EmptyLocationRange,
	)
	vm.replaceTop(result)
}

func opTrue(vm *VM) {
	vm.push(interpreter.TrueValue)
}

func opFalse(vm *VM) {
	vm.push(interpreter.FalseValue)
}

func opGetConstant(vm *VM, ins opcode.InstructionGetConstant) {
	constantIndex := ins.Constant
	executable := vm.callFrame.function.Executable
	c := executable.Constants[constantIndex]
	if c == nil {
		c = vm.initializeConstant(constantIndex)
	}
	vm.push(c)
}

func opGetLocal(vm *VM, ins opcode.InstructionGetLocal) {
	localIndex := ins.Local
	absoluteIndex := vm.callFrame.localsOffset + localIndex
	local := vm.locals[absoluteIndex]

	// Some local variables can be implicit references. e.g: receiver of a bound function.
	// TODO: maybe perform this check only if `localIndex == 0`?
	if implicitReference, ok := local.(ImplicitReferenceValue); ok {
		local = implicitReference.ReferencedValue(vm.context)
	}

	vm.push(local)
}

func opSetLocal(vm *VM, ins opcode.InstructionSetLocal) {
	localIndex := ins.Local
	absoluteIndex := vm.callFrame.localsOffset + localIndex

	existingValue := vm.locals[absoluteIndex]
	if existingValue != nil {
		interpreter.CheckResourceLoss(vm.context, existingValue, EmptyLocationRange)
	}

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

func opCloseUpvalue(vm *VM, ins opcode.InstructionCloseUpvalue) {
	absoluteLocalsIndex := int(vm.callFrame.localsOffset) + int(ins.Local)
	openUpvalues := vm.callFrame.openUpvalues
	upvalue := openUpvalues[absoluteLocalsIndex]
	if upvalue != nil {
		upvalue.closed = vm.locals[absoluteLocalsIndex]
		delete(openUpvalues, absoluteLocalsIndex)
	}
}

func opGetGlobal(vm *VM, ins opcode.InstructionGetGlobal) {
	globalIndex := ins.Global
	globals := vm.callFrame.function.Executable.Globals
	variable := globals[globalIndex]
	vm.push(variable.GetValue(vm.context))
}

func opSetGlobal(vm *VM, ins opcode.InstructionSetGlobal) {
	globalIndex := ins.Global
	globals := vm.callFrame.function.Executable.Globals
	value := vm.pop()
	global := globals[globalIndex]
	global.SetValue(vm.context, EmptyLocationRange, value)
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

func opRemoveIndex(vm *VM) {
	context := vm.context
	container, index := vm.pop2()
	containerValue := container.(interpreter.ValueIndexableValue)
	element := containerValue.RemoveKey(
		context,
		EmptyLocationRange,
		index,
	)

	// Note: Must use `InsertKey` here, not `SetKey`.
	containerValue.InsertKey(
		context,
		EmptyLocationRange,
		index,
		interpreter.PlaceholderValue{},
	)
	vm.push(element)
}

func opInvoke(vm *VM, ins opcode.InstructionInvoke) {
	// Load type arguments
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	// Load arguments
	arguments := vm.popN(int(ins.ArgCount))

	// Load the invoked value
	functionValue := vm.pop()

	invokeFunction(
		vm,
		functionValue,
		arguments,
		typeArguments,
	)
}

func opGetMethod(vm *VM, ins opcode.InstructionGetMethod) {
	globalIndex := ins.Method
	globals := vm.callFrame.function.Executable.Globals

	variable := globals[globalIndex]
	method := variable.GetValue(vm.context).(FunctionValue)

	receiver := vm.pop()

	boundFunction := NewBoundFunctionValue(
		vm.context,
		receiver,
		method,
	)

	vm.push(boundFunction)
}

func opInvokeMethodDynamic(vm *VM, ins opcode.InstructionInvokeDynamic) {
	// TODO: This method is now equivalent to: `GetField` + `Invoke` instructions.
	// See if it can be replaced. That will reduce the complexity of `invokeFunction` method below.

	// Load type arguments
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	// Load arguments
	arguments := vm.popN(int(ins.ArgCount))

	// Load the invoked value
	receiver := vm.pop()

	// Get function
	nameIndex := ins.Name
	funcName := getStringConstant(vm, nameIndex)

	// Load the invoked value
	memberAccessibleValue := receiver.(interpreter.MemberAccessibleValue)
	functionValue := memberAccessibleValue.GetMember(
		vm.context,
		EmptyLocationRange,
		funcName,
	)

	invokeFunction(
		vm,
		functionValue,
		arguments,
		typeArguments,
	)
}

func invokeFunction(
	vm *VM,
	functionValue Value,
	arguments []Value,
	typeArguments []bbq.StaticType,
) {
	context := vm.context
	common.UseComputation(context, common.FunctionInvocationComputationUsage)

	// Handle all function types in a single place, so this can be re-used everywhere.

	boundFunction, isBoundFunction := functionValue.(*BoundFunctionValue)
	if isBoundFunction {
		functionValue = boundFunction.Method
	}

	var receiver Value

	switch functionValue := functionValue.(type) {
	case CompiledFunctionValue:
		if isBoundFunction {
			// For compiled functions, pass the receiver as an implicit-reference.
			// Because the `self` value can be accessed by user-code.
			receiver = boundFunction.Receiver(vm.context)
		}
		vm.pushCallFrame(functionValue, receiver, arguments)

	case *NativeFunctionValue:
		if isBoundFunction {
			// For built-in functions, pass the dereferenced receiver.
			receiver = boundFunction.DereferencedReceiver(vm.context)
		}
		result := functionValue.Function(context, typeArguments, receiver, arguments...)
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

func maybeDereference(context interpreter.ValueStaticTypeContext, value Value) Value {
	switch typedValue := value.(type) {
	case *interpreter.EphemeralReferenceValue:
		return typedValue.Value
	case *interpreter.StorageReferenceValue:
		referencedValue := typedValue.ReferencedValue(
			context,
			EmptyLocationRange,
			true,
		)
		return *referencedValue
	default:
		return value
	}
}

func opDrop(vm *VM) {
	_ = vm.pop()
}

func opDup(vm *VM) {
	top := vm.peek()
	vm.push(top)
}

func opNewComposite(vm *VM, ins opcode.InstructionNewComposite) {
	compositeValue := newCompositeValue(
		vm,
		ins.Kind,
		ins.Type,
		common.ZeroAddress,
	)
	vm.push(compositeValue)
}

func opNewCompositeAt(vm *VM, ins opcode.InstructionNewCompositeAt) {
	executable := vm.callFrame.function.Executable
	c := executable.Program.Constants[ins.Address]

	compositeValue := newCompositeValue(
		vm,
		ins.Kind,
		ins.Type,
		common.MustBytesToAddress(c.Data),
	)
	vm.push(compositeValue)
}

func newCompositeValue(
	vm *VM,
	compositeKind common.CompositeKind,
	typeIndex uint16,
	address common.Address,
) *interpreter.CompositeValue {
	// decode location
	staticType := vm.loadType(typeIndex)

	compositeStaticType := staticType.(*interpreter.CompositeStaticType)

	config := vm.context

	compositeFields := newCompositeValueFields(config, compositeKind)

	return interpreter.NewCompositeValue(
		config,
		EmptyLocationRange,
		compositeStaticType.Location,
		compositeStaticType.QualifiedIdentifier,
		compositeKind,
		compositeFields,
		address,
	)
}

func opSetField(vm *VM, ins opcode.InstructionSetField) {
	target, fieldValue := vm.pop2()

	checkMemberAccessTargetType(
		vm,
		ins.AccessedType,
		target,
	)

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getStringConstant(vm, fieldNameIndex)

	memberAccessibleValue := target.(interpreter.MemberAccessibleValue)
	memberAccessibleValue.SetMember(
		vm.context,
		EmptyLocationRange,
		fieldName,
		fieldValue,
	)
}

func opGetField(vm *VM, ins opcode.InstructionGetField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	checkMemberAccessTargetType(
		vm,
		ins.AccessedType,
		memberAccessibleValue,
	)

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getStringConstant(vm, fieldNameIndex)

	fieldValue := memberAccessibleValue.GetMember(vm.context, EmptyLocationRange, fieldName)
	if fieldValue == nil {
		panic(&interpreter.UseBeforeInitializationError{
			Name: fieldName,
		})
	}

	vm.push(fieldValue)
}

func checkMemberAccessTargetType(
	vm *VM,
	accessedTypeIndex uint16,
	accessedValue interpreter.Value,
) {
	accessedType := vm.loadType(accessedTypeIndex)

	context := vm.context

	// TODO: Avoid sema type conversion.
	accessedSemaType := context.SemaTypeFromStaticType(accessedType)

	interpreter.CheckMemberAccessTargetType(
		context,
		accessedValue,
		accessedSemaType,
		EmptyLocationRange,
	)
}

func opRemoveField(vm *VM, ins opcode.InstructionRemoveField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getStringConstant(vm, fieldNameIndex)

	fieldValue := memberAccessibleValue.RemoveMember(vm.context, EmptyLocationRange, fieldName)
	if fieldValue == nil {
		panic(&interpreter.UseBeforeInitializationError{
			Name: fieldName,
		})
	}

	vm.push(fieldValue)
}

func getStringConstant(vm *VM, index uint16) string {
	executable := vm.callFrame.function.Executable
	c := executable.Program.Constants[index]
	return string(c.Data)
}

func opTransferAndConvert(vm *VM, ins opcode.InstructionTransferAndConvert) {
	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)

	context := vm.context

	value := vm.peek()
	valueType := value.StaticType(context)

	transferredValue := interpreter.TransferAndConvert(
		context,
		value,
		context.SemaTypeFromStaticType(valueType),
		context.SemaTypeFromStaticType(targetType),
		EmptyLocationRange,
	)

	vm.replaceTop(transferredValue)
}

func opTransfer(vm *VM) {
	context := vm.context

	value := vm.peek()

	transferredValue := value.Transfer(
		context,
		EmptyLocationRange,
		atree.Address{},
		false,
		nil,
		nil,
		true, // argument is standalone.
	)

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

	context := vm.context

	value, valueType := castValueAndValueType(context, targetType, value)

	isSubType := context.IsSubType(valueType, targetType)

	var result Value
	if !isSubType {
		targetSemaType := context.SemaTypeFromStaticType(targetType)
		valueSemaType := context.SemaTypeFromStaticType(valueType)

		panic(&interpreter.ForceCastTypeMismatchError{
			ExpectedType:  targetSemaType,
			ActualType:    valueSemaType,
			LocationRange: vm.LocationRange(),
		})
	}

	// The force cast may upcast to an optional type, e.g. `1 as! Int?`, so box
	result = ConvertAndBox(vm.context, value, valueType, targetType)
	vm.push(result)
}

func castValueAndValueType(context *Context, targetType bbq.StaticType, value Value) (Value, bbq.StaticType) {
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

	// If the target is `AnyStruct` or `AnyResource` we want to preserve optionals
	unboxedExpectedType := UnwrapOptionalType(targetType)
	if !(unboxedExpectedType == interpreter.PrimitiveStaticTypeAnyStruct ||
		unboxedExpectedType == interpreter.PrimitiveStaticTypeAnyResource) {
		// otherwise dynamic cast now always unboxes optionals
		value = interpreter.Unbox(value)
	}

	valueType := value.StaticType(context)

	return value, valueType
}

func opNil(vm *VM) {
	vm.push(interpreter.Nil)
}

func opVoid(vm *VM) {
	vm.push(interpreter.Void)
}

func opEqual(vm *VM) {
	left, right := vm.peekPop()
	result := interpreter.TestValueEqual(
		vm.context,
		EmptyLocationRange,
		left,
		right,
	)
	vm.replaceTop(result)
}

func opNotEqual(vm *VM) {
	left, right := vm.peekPop()
	result := !interpreter.TestValueEqual(
		vm.context,
		EmptyLocationRange,
		left,
		right,
	)
	vm.replaceTop(result)
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
		panic(&interpreter.ForceNilError{})
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

	context := vm.context

	semaBorrowedType := context.SemaTypeFromStaticType(borrowedType)

	ref := interpreter.CreateReferenceValue(
		context,
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
	context := vm.context
	iterator := iterable.Iterator(context, EmptyLocationRange)
	if valueID, ok := iterator.ValueID(); ok {
		context.startContainerValueIteration(valueID)
	}
	vm.push(NewIteratorWrapperValue(iterator))
}

func opIteratorHasNext(vm *VM) {
	value := vm.pop()
	iterator := value.(*IteratorWrapperValue)
	result := interpreter.BoolValue(iterator.HasNext(vm.context))
	vm.push(result)
}

func opIteratorNext(vm *VM) {
	value := vm.pop()
	iterator := value.(*IteratorWrapperValue)
	element := iterator.Next(vm.context, EmptyLocationRange)
	vm.push(element)
}

func opIteratorEnd(vm *VM) {
	value := vm.pop()
	iterator := value.(*IteratorWrapperValue)
	if valueID, ok := iterator.ValueID(); ok {
		vm.context.endContainerValueIteration(valueID)
	}
}

func opDeref(vm *VM) {
	value := vm.pop()
	dereferenced := interpreter.DereferenceValue(vm.context, EmptyLocationRange, value)
	vm.push(dereferenced)
}

func opStringTemplate(vm *VM, ins opcode.InstructionTemplateString) {
	expressions := vm.popN(int(ins.ExprSize))
	values := vm.popN(int(ins.ExprSize + 1))
	var valuesStr []string

	// convert values to string[]
	for _, str := range values {
		s, ok := str.(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		valuesStr = append(valuesStr, s.Str)
	}

	vm.push(interpreter.BuildStringTemplate(valuesStr, expressions))
}

func (vm *VM) run() {

	defer func() {
		if r := recover(); r != nil {
			if locatedError, ok := r.(interpreter.HasLocationRange); ok {
				locatedError.SetLocationRange(vm.LocationRange())
			}

			if vm.context.debugEnabled {
				switch r.(type) {
				case errors.UserError, errors.ExternalError:
					// do nothing
				default:
					printInstructionError(
						vm.callFrame.function.Function,
						int(vm.ip),
						r,
					)
				}
			}

			panic(r)
		}
	}()

	entryPointCallStackSize := len(vm.callstack)

	for {

		callFrame := vm.callFrame

		code := callFrame.function.Function.Code

		// VM can re-enter to the instruction-execution multiple times.
		// e.g: Passing a compiled-function-pointer to a native function,
		// and invoking it inside the native-function: VM will start executing
		// this function similar to executing a method externally,
		// but while still being in the middle of executing a different method.
		// Thus, when returning, it should return to the place where the function call was initiated from.
		// i.e: native code in this case.
		// Therefore, return all the way (and do not continue to unwind and execute the parent call-stack)
		// if it reached the current entry-point, when unwinding the stack.
		if len(vm.callstack) < entryPointCallStackSize ||
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
		case opcode.InstructionCloseUpvalue:
			opCloseUpvalue(vm, ins)
		case opcode.InstructionGetGlobal:
			opGetGlobal(vm, ins)
		case opcode.InstructionSetGlobal:
			opSetGlobal(vm, ins)
		case opcode.InstructionSetIndex:
			opSetIndex(vm)
		case opcode.InstructionGetIndex:
			opGetIndex(vm)
		case opcode.InstructionRemoveIndex:
			opRemoveIndex(vm)
		case opcode.InstructionGetMethod:
			opGetMethod(vm, ins)
		case opcode.InstructionInvoke:
			opInvoke(vm, ins)
		case opcode.InstructionInvokeDynamic:
			opInvokeMethodDynamic(vm, ins)
		case opcode.InstructionDrop:
			opDrop(vm)
		case opcode.InstructionDup:
			opDup(vm)
		case opcode.InstructionNewComposite:
			opNewComposite(vm, ins)
		case opcode.InstructionNewCompositeAt:
			opNewCompositeAt(vm, ins)
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
		case opcode.InstructionRemoveField:
			opRemoveField(vm, ins)
		case opcode.InstructionTransferAndConvert:
			opTransferAndConvert(vm, ins)
		case opcode.InstructionTransfer:
			opTransfer(vm)
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
		case opcode.InstructionVoid:
			opVoid(vm)
		case opcode.InstructionEqual:
			opEqual(vm)
		case opcode.InstructionNotEqual:
			opNotEqual(vm)
		case opcode.InstructionNot:
			opNot(vm)
		case opcode.InstructionUnwrap:
			opUnwrap(vm)
		case opcode.InstructionEmitEvent:
			opEmitEvent(vm, ins)
		case opcode.InstructionIterator:
			opIterator(vm)
		case opcode.InstructionIteratorHasNext:
			opIteratorHasNext(vm)
		case opcode.InstructionIteratorNext:
			opIteratorNext(vm)
		case opcode.InstructionIteratorEnd:
			opIteratorEnd(vm)
		case opcode.InstructionDeref:
			opDeref(vm)
		case opcode.InstructionNewClosure:
			opNewClosure(vm, ins)
		case opcode.InstructionLoop:
			opLoop(vm)
		case opcode.InstructionStatement:
			opStatement(vm)
		case opcode.InstructionTemplateString:
			opStringTemplate(vm, ins)
		default:
			panic(errors.NewUnexpectedError("cannot execute instruction of type %T", ins))
		}
	}
}

func opEmitEvent(vm *VM, ins opcode.InstructionEmitEvent) {
	context := vm.context

	typeIndex := ins.Type
	eventStaticType := vm.loadType(typeIndex).(*interpreter.CompositeStaticType)
	eventSemaType := context.SemaTypeFromStaticType(eventStaticType).(*sema.CompositeType)

	eventFields := vm.popN(int(ins.ArgCount))

	// Make a copy, since the slice can get mutated, since the stack is reused.
	fields := make([]interpreter.Value, len(eventFields))
	copy(fields, eventFields)

	context.EmitEvent(
		context,
		EmptyLocationRange,
		eventSemaType,
		fields,
	)
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
	openUpvalues := vm.callFrame.openUpvalues
	if upvalue, ok := openUpvalues[absoluteLocalsIndex]; ok {
		return upvalue
	}

	// Create a new upvalue and record it as open
	upvalue := &Upvalue{
		absoluteLocalsIndex: absoluteLocalsIndex,
	}
	if openUpvalues == nil {
		openUpvalues = make(map[int]*Upvalue)
		vm.callFrame.openUpvalues = openUpvalues
	}
	openUpvalues[absoluteLocalsIndex] = upvalue
	return upvalue
}

func opLoop(vm *VM) {
	common.UseComputation(vm.context, common.LoopComputationUsage)
}

func opStatement(vm *VM) {
	common.UseComputation(vm.context, common.StatementComputationUsage)
}

func (vm *VM) initializeConstant(index uint16) (value Value) {
	executable := vm.callFrame.function.Executable
	c := executable.Program.Constants[index]

	memoryGauge := vm.context.MemoryGauge

	switch c.Kind {
	case constant.String:
		value = interpreter.NewUnmeteredStringValue(string(c.Data))

	case constant.Character:
		value = interpreter.NewUnmeteredCharacterValue(string(c.Data))

	case constant.Int:
		value = interpreter.NewIntValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int8:
		value = interpreter.NewInt8ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int16:
		value = interpreter.NewInt16ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int32:
		value = interpreter.NewInt32ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int64:
		value = interpreter.NewInt64ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int128:
		value = interpreter.NewInt128ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Int256:
		value = interpreter.NewInt256ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt:
		value = interpreter.NewUIntValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt8:
		value = interpreter.NewUInt8ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt16:
		value = interpreter.NewUInt16ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt32:
		value = interpreter.NewUInt32ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt64:
		value = interpreter.NewUInt64ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt128:
		value = interpreter.NewUInt128ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UInt256:
		value = interpreter.NewUInt256ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word8:
		value = interpreter.NewWord8ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word16:
		value = interpreter.NewWord16ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word32:
		value = interpreter.NewWord32ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word64:
		value = interpreter.NewWord64ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word128:
		value = interpreter.NewWord128ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Word256:
		value = interpreter.NewWord256ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Fix64:
		value = interpreter.NewFix64ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.UFix64:
		value = interpreter.NewUFix64ValueFromBigEndianBytes(memoryGauge, c.Data)

	case constant.Address:
		value = interpreter.NewAddressValueFromBytes(
			memoryGauge,
			func() []byte {
				return c.Data
			},
		)

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

func (vm *VM) invokeFunction(function Value, arguments []Value) (Value, error) {
	// invokeExternally runs the VM, which is incorrect for native functions.
	if functionValue, ok := function.(*NativeFunctionValue); ok {
		invokeFunction(
			vm,
			functionValue,
			arguments,
			nil,
		)
		return vm.pop(), nil
	}

	return vm.invokeExternally(function, arguments)
}

func (vm *VM) lookupFunction(location common.Location, name string) FunctionValue {
	context := vm.context

	// First check in current program.
	global := vm.globals.Find(name)
	if global != nil {
		value := global.GetValue(context)
		return value.(FunctionValue)
	}

	// If not found, check in already linked imported functions,
	// or link the function now, dynamically.

	var indexedGlobals *activations.Activation[Variable]

	if location == nil {
		if context.BuiltinGlobalsProvider == nil {
			indexedGlobals = DefaultBuiltinGlobals()
		} else {
			indexedGlobals = context.BuiltinGlobalsProvider(location)
		}
	} else {

		linkedGlobals, ok := vm.linkedGlobalsCache[location]

		if !ok {
			// TODO: This currently link all functions in program, unnecessarily.
			//   Link only the requested function.
			program := context.ImportHandler(location)

			linkedGlobals = LinkGlobals(
				context.MemoryGauge,
				location,
				program,
				context,
				vm.linkedGlobalsCache,
			)
		}

		indexedGlobals = linkedGlobals.indexedGlobals
	}

	global = indexedGlobals.Find(name)
	if global == nil {
		return nil
	}

	value := global.GetValue(context)
	return value.(FunctionValue)
}

func (vm *VM) StackSize() int {
	return len(vm.stack)
}

func (vm *VM) Reset() {
	vm.stack = vm.stack[:0]
	vm.locals = vm.locals[:0]
	vm.callstack = vm.callstack[:0]
	vm.ipStack = vm.ipStack[:0]

	context := NewContext(vm.context.Config)
	context.invokeFunction = vm.invokeFunction
	context.lookupFunction = vm.lookupFunction
	vm.context = context
}

func (vm *VM) initializeGlobalVariables(program *bbq.InstructionProgram) {
	for _, variable := range program.Variables {
		// Get the values to ensure they are initialized.
		_ = vm.Global(variable.Name)
	}
}

func (vm *VM) Global(name string) Value {
	variable := vm.globals.Find(name)
	if variable == nil {
		return nil
	}
	return variable.GetValue(vm.context)
}

// LocationRange returns the location of the currently executing instruction.
// This is an expensive operation and must be only used on-demand.
func (vm *VM) LocationRange() interpreter.LocationRange {
	currentFunction := vm.callFrame.function
	lineNumbers := currentFunction.Function.LineNumbers

	// `vm.ip` always points to the next instruction.
	lastInstructionIndex := vm.ip - 1

	position := lineNumbers.GetSourcePosition(lastInstructionIndex)

	return interpreter.LocationRange{
		Location:    currentFunction.Executable.Location,
		HasPosition: position,
	}
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
