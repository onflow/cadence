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
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
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

	context *Context
	globals *activations.Activation[Variable]
}

func NewVM(
	location common.Location,
	program *bbq.InstructionProgram,
	config *Config,
) *VM {

	context := NewContext(config)

	vm := &VM{
		context: context,
	}

	vm.configureContext()

	context.recoverErrors = vm.RecoverErrors

	// Link global variables and functions.
	linkedGlobals := context.linkGlobals(
		location,
		program,
	)

	vm.globals = linkedGlobals.indexedGlobals

	return vm
}

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

	interpreter.CheckInvalidatedResourceOrResourceReference(value, vm.context)

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

	context := vm.context
	interpreter.CheckInvalidatedResourceOrResourceReference(value1, context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value2, context)

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

	context := vm.context
	interpreter.CheckInvalidatedResourceOrResourceReference(value1, context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value2, context)
	interpreter.CheckInvalidatedResourceOrResourceReference(value3, context)

	return value1, value2, value3
}

func (vm *VM) peekN(count int) []Value {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	values := vm.stack[startIndex:]

	context := vm.context
	for _, value := range values {
		interpreter.CheckInvalidatedResourceOrResourceReference(value, context)
	}

	return values
}

func (vm *VM) popN(count int) []Value {
	stackHeight := len(vm.stack)
	startIndex := stackHeight - count
	values := vm.stack[startIndex:]

	context := vm.context
	for _, value := range values {
		interpreter.CheckInvalidatedResourceOrResourceReference(value, context)
	}

	vm.stack = vm.stack[:startIndex]

	return values
}

func (vm *VM) peek() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	interpreter.CheckInvalidatedResourceOrResourceReference(value, vm.context)
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

	context := vm.context
	interpreter.CheckInvalidatedResourceOrResourceReference(peekedValue, context)
	interpreter.CheckInvalidatedResourceOrResourceReference(poppedValue, context)

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

	var startTime time.Time
	if interpreter.TracingEnabled {
		startTime = time.Now()
	}

	callFrame := callFrame{
		localsCount:  localsCount,
		localsOffset: offset,
		function:     functionValue,
		startTime:    startTime,
	}

	vm.ipStack = append(vm.ipStack, 0)
	vm.ip = 0

	vm.callstack = append(vm.callstack, callFrame)
	vm.callFrame = &vm.callstack[len(vm.callstack)-1]
}

func (vm *VM) popCallFrame() {

	if interpreter.TracingEnabled {
		startTime := vm.callFrame.startTime
		function := vm.callFrame.function
		defer func() {
			vm.context.ReportInvokeTrace(
				function.FunctionType(vm.context).String(),
				function.Function.QualifiedName,
				time.Since(startTime),
			)
		}()
	}

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

func (vm *VM) InvokeExternally(name string, arguments ...Value) (v Value, err error) {
	functionVariable := vm.globals.Find(name)
	if functionVariable == nil {
		return nil, UnknownFunctionError{
			name: name,
		}
	}

	defer vm.RecoverErrors(func(internalErr error) {
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

	defer vm.RecoverErrors(func(internalErr error) {
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

	boundFunction := NewBoundFunctionValue(context, receiver, functionValue, nil)

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

func (vm *VM) invokeExternally(functionValue FunctionValue, arguments []Value) (Value, error) {
	invokeFunction(
		vm,
		functionValue,
		arguments,
		nil,
	)

	// Runs the VM for compiled functions.
	if !functionValue.IsNative() {
		vm.run()

		if len(vm.stack) == 0 {
			return nil, nil
		}
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

	defer vm.RecoverErrors(func(internalErr error) {
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

func (vm *VM) InvokeTransactionWrapper() (*interpreter.SimpleCompositeValue, error) {
	wrapperResult, err := vm.InvokeExternally(commons.TransactionWrapperCompositeName)
	if err != nil {
		return nil, err
	}

	transaction := wrapperResult.(*interpreter.SimpleCompositeValue)

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

func (vm *VM) InvokeTransactionPrepare(transaction *interpreter.SimpleCompositeValue, signers []Value) error {
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
		nil,
	)

	_, err := vm.validateAndInvokeExternally(boundPrepareFunction, signers)
	if err != nil {
		return err
	}

	return nil
}

func (vm *VM) InvokeTransactionExecute(transaction *interpreter.SimpleCompositeValue) error {
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
		nil,
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
	result := leftNumber.Plus(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Minus(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opMultiply(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Mul(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opDivide(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Div(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opMod(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	result := leftNumber.Mod(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opNegate(vm *VM) {
	value := vm.pop().(interpreter.NumberValue)
	result := value.Negate(vm.context)
	vm.push(result)
}

func opBitwiseOr(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseOr(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opBitwiseXor(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseXor(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opBitwiseAnd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseAnd(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opBitwiseLeftShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseLeftShift(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opBitwiseRightShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	result := leftNumber.BitwiseRightShift(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.Less(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opLessOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.LessEqual(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.Greater(vm.context, rightNumber)
	vm.replaceTop(result)
}

func opGreaterOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.ComparableValue)
	rightNumber := right.(interpreter.ComparableValue)
	result := leftNumber.GreaterEqual(vm.context, rightNumber)
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
		// Constants referred-to by `InstructionGetConstant`
		// are always value-typed constants.
		c = vm.initializeValueTypedConstant(constantIndex)
	}
	vm.push(c)
}

func getLocal(vm *VM, localIndex uint16) Value {
	absoluteIndex := vm.callFrame.localsOffset + localIndex
	local := vm.locals[absoluteIndex]

	// Some local variables can be implicit references. e.g: receiver of a bound function.
	// TODO: maybe perform this check only if `localIndex == 0`?
	if implicitReference, ok := local.(ImplicitReferenceValue); ok {
		local = implicitReference.ReferencedValue(vm.context)
	}

	return local
}

func opGetLocal(vm *VM, ins opcode.InstructionGetLocal) {
	local := getLocal(vm, ins.Local)
	vm.push(local)
}

func opSetLocal(vm *VM, ins opcode.InstructionSetLocal) {
	localIndex := ins.Local
	absoluteIndex := vm.callFrame.localsOffset + localIndex

	existingValue := vm.locals[absoluteIndex]
	if existingValue != nil {
		interpreter.CheckResourceLoss(vm.context, existingValue)
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
	global.SetValue(vm.context, value)
}

func opSetIndex(vm *VM) {
	container, index, value := vm.pop3()
	containerValue := container.(interpreter.ValueIndexableValue)
	containerValue.SetKey(vm.context, index, value)
}

func opGetIndex(vm *VM) {
	container, index := vm.pop2()
	containerValue := container.(interpreter.ValueIndexableValue)
	element := containerValue.GetKey(vm.context, index)
	vm.push(element)
}

func opRemoveIndex(vm *VM) {
	context := vm.context
	container, index := vm.pop2()
	containerValue := container.(interpreter.ValueIndexableValue)
	element := containerValue.RemoveKey(context, index)

	// Note: Must use `InsertKey` here, not `SetKey`.
	containerValue.InsertKey(context, index, interpreter.PlaceholderValue{})
	vm.push(element)
}

func opInvoke(vm *VM, ins opcode.InstructionInvoke) {
	// Load type arguments
	typeArguments := loadTypeArguments(vm, ins.TypeArgs)

	// Load arguments
	arguments := vm.popN(int(ins.ArgCount))

	// Load the invoked value
	functionValue := vm.pop()

	// Add base to front of arguments if the function is bound and base is defined.
	if boundFunction, isBoundFunction := functionValue.(*BoundFunctionValue); isBoundFunction {
		base := boundFunction.Base
		if base != nil {
			arguments = append([]Value{base}, arguments...)
		}
	}

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

	var base *interpreter.EphemeralReferenceValue
	if val, ok := receiver.(*interpreter.EphemeralReferenceValue); ok {
		if refValue, ok := val.Value.(*interpreter.CompositeValue); ok && refValue.Kind == common.CompositeKindAttachment {
			base, receiver = AttachmentBaseAndSelfValues(vm.context, refValue, method)
		}
	}

	boundFunction := NewBoundFunctionValue(
		vm.context,
		receiver,
		method,
		base,
	)

	vm.push(boundFunction)
}

func invokeFunction(
	vm *VM,
	functionValue Value,
	arguments []Value,
	typeArguments []bbq.StaticType,
) {
	context := vm.context
	common.UseComputation(context, common.FunctionInvocationComputationUsage)

	originalFunctionValue := functionValue.(FunctionValue)

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
			receiver = boundFunction.Receiver(context)
		}

		// Trace is reported in `popCallFrame`, to also include the execution time.

		vm.pushCallFrame(functionValue, receiver, arguments)

	case *NativeFunctionValue:
		if isBoundFunction {
			// For built-in functions, pass the dereferenced receiver.
			receiver = boundFunction.DereferencedReceiver(context)
		}

		if interpreter.TracingEnabled {
			startTime := time.Now()
			defer func() {
				context.ReportInvokeTrace(
					// Use the original function value, to get the correct type.
					// The native function value might have been wrapped in a bound function.
					originalFunctionValue.FunctionType(context).String(),
					functionValue.Name,
					time.Since(startTime),
				)
			}()
		}

		typeArgumentsIterator := NewTypeArgumentsIterator(context, typeArguments)

		result := functionValue.Function(
			context,
			typeArgumentsIterator,
			receiver,
			arguments,
		)
		vm.push(result)

	default:
		panic(errors.NewUnreachableError())
	}
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

func maybeDereferenceReceiver(context interpreter.ValueStaticTypeContext, value Value, isNative bool) Value {
	switch typedValue := value.(type) {
	case *interpreter.EphemeralReferenceValue:
		// Do not dereference attachments, so that the receiver is a reference as expected.
		// Exception: Native function receiver needs to be dereferenced to match interpreter.
		if val, ok := typedValue.Value.(*interpreter.CompositeValue); ok && !isNative {
			if val.Kind == common.CompositeKindAttachment {
				return value
			}
		}
		return typedValue.Value
	case *interpreter.StorageReferenceValue:
		referencedValue := typedValue.ReferencedValue(context, true)
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

func opNewSimpleComposite(vm *VM, ins opcode.InstructionNewSimpleComposite) {
	staticType := vm.loadType(ins.Type)

	compositeStaticType := staticType.(*interpreter.CompositeStaticType)

	context := vm.context

	compositeValue := interpreter.NewSimpleCompositeValue(
		context,
		compositeStaticType.TypeID,
		compositeStaticType,
		nil,
		map[string]Value{},
		nil,
		nil,
		nil,
		nil,
	)

	vm.push(compositeValue)
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

	addressValue := c.Data.(interpreter.AddressValue)

	compositeValue := newCompositeValue(
		vm,
		ins.Kind,
		ins.Type,
		addressValue.ToAddress(),
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

	context := vm.context

	compositeFields := newCompositeValueFields(context, compositeKind)

	return interpreter.NewCompositeValue(
		context,
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
	fieldName := getRawStringConstant(vm, fieldNameIndex)

	memberAccessibleValue := target.(interpreter.MemberAccessibleValue)
	memberAccessibleValue.SetMember(vm.context, fieldName, fieldValue)
}

func getField(vm *VM, memberAccessibleValue interpreter.MemberAccessibleValue, fieldNameIndex uint16, accessedTypeIndex uint16) Value {

	checkMemberAccessTargetType(
		vm,
		accessedTypeIndex,
		memberAccessibleValue,
	)

	// VM assumes the field name is always a string.
	fieldName := getRawStringConstant(vm, fieldNameIndex)

	fieldValue := memberAccessibleValue.GetMember(vm.context, fieldName)
	if fieldValue == nil {
		panic(&interpreter.UseBeforeInitializationError{
			Name: fieldName,
		})
	}

	return fieldValue
}

func opGetField(vm *VM, ins opcode.InstructionGetField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	fieldValue := getField(vm, memberAccessibleValue, ins.FieldName, ins.AccessedType)

	vm.push(fieldValue)
}

func checkMemberAccessTargetType(
	vm *VM,
	accessedTypeIndex uint16,
	accessedValue Value,
) {
	accessedType := vm.loadType(accessedTypeIndex)

	context := vm.context

	// TODO: Avoid sema type conversion
	accessedSemaType := context.SemaTypeFromStaticType(accessedType)

	interpreter.CheckMemberAccessTargetType(
		context,
		accessedValue,
		accessedSemaType,
	)
}

func opRemoveField(vm *VM, ins opcode.InstructionRemoveField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	// VM assumes the field name is always a string.
	fieldNameIndex := ins.FieldName
	fieldName := getRawStringConstant(vm, fieldNameIndex)

	fieldValue := memberAccessibleValue.RemoveMember(vm.context, fieldName)
	if fieldValue == nil {
		panic(&interpreter.UseBeforeInitializationError{
			Name: fieldName,
		})
	}

	vm.push(fieldValue)
}

func getRawStringConstant(vm *VM, index uint16) string {
	executable := vm.callFrame.function.Executable
	c := executable.Program.Constants[index]
	return c.Data.(string)
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
	)

	vm.replaceTop(transferredValue)
}

func opTransfer(vm *VM) {
	context := vm.context

	value := vm.peek()

	transferredValue := value.Transfer(
		context,
		atree.Address{},
		false,
		nil,
		nil,
		true, // argument is standalone.
	)

	vm.replaceTop(transferredValue)
}

func opConvert(vm *VM, ins opcode.InstructionConvert) {
	typeIndex := ins.Type
	targetType := vm.loadType(typeIndex)

	context := vm.context

	value := vm.peek()
	valueType := value.StaticType(context)

	transferredValue := interpreter.ConvertAndBoxWithValidation(
		context,
		value,
		context.SemaTypeFromStaticType(valueType),
		context.SemaTypeFromStaticType(targetType),
	)

	vm.replaceTop(transferredValue)
}

func opDestroy(vm *VM) {
	value := vm.pop().(interpreter.ResourceKindedValue)
	value.Destroy(vm.context)
}

func opNewPath(vm *VM, ins opcode.InstructionNewPath) {
	identifierIndex := ins.Identifier
	identifier := getRawStringConstant(vm, identifierIndex)
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
			ExpectedType: targetSemaType,
			ActualType:   valueSemaType,
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
		left,
		right,
	)
	vm.replaceTop(result)
}

func opNotEqual(vm *VM) {
	left, right := vm.peekPop()
	result := !interpreter.TestValueEqual(
		vm.context,
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

func opWrap(vm *VM) {
	value := vm.peek()
	optional := interpreter.NewSomeValueNonCopying(vm.context, value)
	vm.replaceTop(optional)
}

func opNewArray(vm *VM, ins opcode.InstructionNewArray) {
	typeIndex := ins.Type
	typ := vm.loadType(typeIndex).(interpreter.ArrayStaticType)

	elements := vm.peekN(int(ins.Size))
	array := interpreter.NewArrayValue(
		vm.context,
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
		ins.IsImplicit,
	)

	vm.push(ref)
}

func opIterator(vm *VM) {
	value := vm.pop()
	iterable := value.(interpreter.IterableValue)
	context := vm.context
	iterator := iterable.Iterator(context)
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
	element := iterator.Next(vm.context)
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
	dereferenced := interpreter.DereferenceValue(vm.context, value)
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

	vm.push(interpreter.BuildStringTemplate(
		vm.context,
		valuesStr,
		expressions,
	))
}

func opGetTypeIndex(vm *VM, ins opcode.InstructionGetTypeIndex) {
	context := vm.context

	target := vm.pop()

	// Get attachment type
	typeIndex := ins.Type
	staticType := vm.loadType(typeIndex)
	typ := context.SemaTypeFromStaticType(staticType)

	compositeValue := target.(interpreter.TypeIndexableValue)
	value := compositeValue.GetTypeKey(context, typ)
	vm.push(value)
}

func opSetTypeIndex(vm *VM, ins opcode.InstructionSetTypeIndex) {
	context := vm.context

	fieldValue, target := vm.pop2()

	// Get attachment type
	typeIndex := ins.Type
	staticType := vm.loadType(typeIndex)
	typ := context.SemaTypeFromStaticType(staticType)
	attachment := fieldValue.(*interpreter.CompositeValue)

	base := target.(*interpreter.CompositeValue)

	if inIteration := context.inAttachmentIteration(base); inIteration {
		panic(&interpreter.AttachmentIterationMutationError{
			Value: base,
		})
	}

	// transfer here instead of in compiler
	// so we can check attachment iteration properly
	base = base.Transfer(
		context,
		atree.Address{},
		false,
		nil,
		nil,
		true, // base is standalone,
	).(*interpreter.CompositeValue)

	base.SetTypeKey(
		context,
		typ,
		fieldValue,
	)
	attachment.SetBaseValue(base)
	vm.push(base)
}

func opSetAttachmentBase(vm *VM) {
	base, attachment := vm.pop2()

	attachmentComposite, ok := attachment.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	baseRef, ok := base.(*interpreter.EphemeralReferenceValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	baseComposite, ok := baseRef.Value.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	attachmentComposite.SetBaseValue(baseComposite)
}

func opRemoveTypeIndex(vm *VM, ins opcode.InstructionRemoveTypeIndex) {
	target := vm.pop()

	// Get attachment type
	typeIndex := ins.Type
	staticType := vm.loadType(typeIndex)
	typ := vm.context.SemaTypeFromStaticType(staticType)

	base, ok := target.(*interpreter.CompositeValue)
	if inIteration := vm.context.inAttachmentIteration(base); inIteration {
		panic(&interpreter.AttachmentIterationMutationError{
			Value: base,
		})
	}
	// We enforce this in the checker, but check defensively anyway
	if !ok || !base.Kind.SupportsAttachments() {
		panic(&interpreter.InvalidAttachmentOperationTargetError{
			Value: target,
		})
	}

	removed := base.RemoveTypeKey(vm.context, typ)
	// Attachment not present on this base
	if removed == nil {
		return
	}

	attachment, ok := removed.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if attachment.IsResourceKinded(vm.context) {
		// This attachment is no longer attached to its base,
		// but the `base` variable is still available in the destructor
		attachment.SetBaseValue(base)
		attachment.Destroy(vm.context)
	}
}

func opGetFieldLocal(vm *VM, ins opcode.InstructionGetFieldLocal) {
	local := getLocal(vm, ins.Local)
	memberAccessibleValue := local.(interpreter.MemberAccessibleValue)
	fieldValue := getField(vm, memberAccessibleValue, ins.FieldName, ins.AccessedType)

	vm.push(fieldValue)
}

func (vm *VM) run() {

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
		case opcode.InstructionDrop:
			opDrop(vm)
		case opcode.InstructionDup:
			opDup(vm)
		case opcode.InstructionNewSimpleComposite:
			opNewSimpleComposite(vm, ins)
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
		case opcode.InstructionConvert:
			opConvert(vm, ins)
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
		case opcode.InstructionWrap:
			opWrap(vm)
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
		case opcode.InstructionGetTypeIndex:
			opGetTypeIndex(vm, ins)
		case opcode.InstructionSetTypeIndex:
			opSetTypeIndex(vm, ins)
		case opcode.InstructionRemoveTypeIndex:
			opRemoveTypeIndex(vm, ins)
		case opcode.InstructionSetAttachmentBase:
			opSetAttachmentBase(vm)
		case opcode.InstructionGetFieldLocal:
			opGetFieldLocal(vm, ins)
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
	fields := make([]Value, len(eventFields))
	copy(fields, eventFields)

	context.EmitEvent(context, eventSemaType, fields)
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

func (vm *VM) initializeValueTypedConstant(index uint16) Value {
	executable := vm.callFrame.function.Executable
	c := executable.Program.Constants[index]

	value := c.Data.(Value)
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
	functionValue := function.(FunctionValue)
	return vm.invokeExternally(functionValue, arguments)
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
		// TODO: This currently link all functions in program, unnecessarily.
		//   Link only the requested function.
		linkedGlobals := context.linkLocation(location)
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

	vm.context = vm.context.newReusing()
	vm.configureContext()
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
	// If the error occurs even before the VM starts executing,
	// e.g: computation/memory metering errors,
	// then use an empty-location-range, which points to the start of the program.
	if vm.callFrame == nil {
		return interpreter.LocationRange{}
	}

	currentFunction := vm.callFrame.function

	// `vm.ip` always points to the next instruction.
	lastInstructionIndex := vm.ip - 1

	return locationRangeOfInstruction(currentFunction, lastInstructionIndex)
}

func locationRangeOfInstruction(function CompiledFunctionValue, instructionIndex uint16) interpreter.LocationRange {
	lineNumbers := function.Function.LineNumbers
	position := lineNumbers.GetSourcePosition(instructionIndex)

	return interpreter.LocationRange{
		Location:    function.Executable.Location,
		HasPosition: position,
	}
}

func (vm *VM) callStackLocations() []interpreter.LocationRange {
	if len(vm.callstack) <= 1 {
		return nil
	}

	// Skip the current level. It is already included in the parent error.
	callstack := vm.callstack[:len(vm.callstack)-1]

	locationRanges := make([]interpreter.LocationRange, 0, len(vm.stack))

	for index, stackFrame := range callstack {
		function := stackFrame.function
		ip := vm.ipStack[index]
		locationRange := locationRangeOfInstruction(function, ip)

		locationRanges = append(locationRanges, locationRange)
	}

	return locationRanges
}

func (vm *VM) RecoverErrors(onError func(error)) {
	if r := recover(); r != nil {
		// Recover all errors, because VM can be directly invoked by FVM.
		err := interpreter.AsCadenceError(r)

		locationRange := vm.LocationRange()

		if locatedError, ok := err.(interpreter.HasLocationRange); ok {
			locatedError.SetLocationRange(locationRange)
		}

		// if the error is not yet an interpreter error, wrap it
		if _, ok := err.(interpreter.Error); !ok {

			_, ok := err.(ast.HasPosition)
			if !ok {
				errRange := ast.NewUnmeteredRangeFromPositioned(locationRange)

				err = interpreter.PositionedError{
					Err:   err,
					Range: errRange,
				}
			}

			err = interpreter.Error{
				Err:      err,
				Location: locationRange.Location,
			}
		}

		interpreterErr := err.(interpreter.Error)
		interpreterErr.StackTrace = vm.callStackLocations()

		// For debug purpose only

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

		onError(interpreterErr)
	}
}

func (vm *VM) configureContext() {
	context := vm.context

	// Delegate function invocations to the VM
	context.invokeFunction = vm.invokeFunction
	context.lookupFunction = vm.lookupFunction
	context.getLocationRange = vm.LocationRange
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
