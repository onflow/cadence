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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constantkind"
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

	config             *Config
	globals            map[string]Value
	linkedGlobalsCache map[common.Location]LinkedGlobals
}

func NewVM(
	location common.Location,
	program *bbq.InstructionProgram,
	conf *Config,
) *VM {
	// TODO: Remove initializing config. Following is for testing purpose only.
	if conf == nil {
		conf = &Config{}
	}
	if conf.storage == nil {
		conf.storage = interpreter.NewInMemoryStorage(nil)
	}

	if conf.NativeFunctionsProvider == nil {
		conf.NativeFunctionsProvider = NativeFunctions
	}

	if conf.referencedResourceKindedValues == nil {
		conf.referencedResourceKindedValues = ReferencedResourceKindedValues{}
	}

	// linkedGlobalsCache is a local cache-alike that is being used to hold already linked imports.
	linkedGlobalsCache := map[common.Location]LinkedGlobals{
		BuiltInLocation: {
			// It is NOT safe to re-use native functions map here because,
			// once put into the cache, it will be updated by adding the
			// globals of the current program.
			indexedGlobals: conf.NativeFunctionsProvider(),
		},
	}

	vm := &VM{
		linkedGlobalsCache: linkedGlobalsCache,
		config:             conf,
	}

	// Delegate the function invocations to the vm.
	conf.invokeFunction = vm.invoke

	// Link global variables and functions.
	linkedGlobals := LinkGlobals(
		location,
		program,
		conf,
		linkedGlobalsCache,
	)

	vm.globals = linkedGlobals.indexedGlobals

	return vm
}

var EmptyLocationRange = interpreter.EmptyLocationRange

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	vm.stack[lastIndex] = nil
	vm.stack = vm.stack[:lastIndex]

	vm.config.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange)

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

	vm.config.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange)
	vm.config.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange)

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

	vm.config.CheckInvalidatedResourceOrResourceReference(value1, EmptyLocationRange)
	vm.config.CheckInvalidatedResourceOrResourceReference(value2, EmptyLocationRange)
	vm.config.CheckInvalidatedResourceOrResourceReference(value3, EmptyLocationRange)

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
		vm.config.CheckInvalidatedResourceOrResourceReference(value, EmptyLocationRange)
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

func (vm *VM) pushCallFrame(functionValue FunctionValue, arguments []Value) {
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
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opSubtract(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Minus(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opMultiply(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Mul(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opDivide(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Div(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opMod(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Mod(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opNegate(vm *VM) {
	value := vm.pop().(interpreter.NumberValue)
	vm.push(value.Negate(
		vm.config,
		EmptyLocationRange,
	))
}

func opBitwiseOr(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseOr(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseXor(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseXor(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseAnd(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseAnd(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseLeftShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseLeftShift(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opBitwiseRightShift(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.IntegerValue)
	rightNumber := right.(interpreter.IntegerValue)
	vm.replaceTop(leftNumber.BitwiseRightShift(
		vm.config,
		rightNumber,
		EmptyLocationRange,
	))
}

func opLess(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Less(vm.config, rightNumber, EmptyLocationRange))
}

func opLessOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.LessEqual(vm.config, rightNumber, EmptyLocationRange))
}

func opGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.Greater(vm.config, rightNumber, EmptyLocationRange))
}

func opGreaterOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(interpreter.NumberValue)
	rightNumber := right.(interpreter.NumberValue)
	vm.replaceTop(leftNumber.GreaterEqual(vm.config, rightNumber, EmptyLocationRange))
}

func opTrue(vm *VM) {
	vm.push(interpreter.TrueValue)
}

func opFalse(vm *VM) {
	vm.push(interpreter.FalseValue)
}

func opGetConstant(vm *VM, ins opcode.InstructionGetConstant) {
	constant := vm.callFrame.function.Executable.Constants[ins.ConstantIndex]
	if constant == nil {
		constant = vm.initializeConstant(ins.ConstantIndex)
	}
	vm.push(constant)
}

func opGetLocal(vm *VM, ins opcode.InstructionGetLocal) {
	absoluteIndex := vm.callFrame.localsOffset + ins.LocalIndex
	local := vm.locals[absoluteIndex]
	vm.push(local)
}

func opSetLocal(vm *VM, ins opcode.InstructionSetLocal) {
	absoluteIndex := vm.callFrame.localsOffset + ins.LocalIndex
	vm.locals[absoluteIndex] = vm.pop()
}

func opGetUpvalue(vm *VM, ins opcode.InstructionGetUpvalue) {
	upvalue := vm.callFrame.function.Upvalues[ins.UpvalueIndex]
	value := upvalue.closed
	if value == nil {
		value = vm.locals[upvalue.absoluteLocalsIndex]
	}
	vm.push(value)
}

func opSetUpvalue(vm *VM, ins opcode.InstructionSetUpvalue) {
	upvalue := vm.callFrame.function.Upvalues[ins.UpvalueIndex]
	value := vm.pop()
	if upvalue.closed == nil {
		vm.locals[upvalue.absoluteLocalsIndex] = value
	} else {
		upvalue.closed = value
	}
}

func opGetGlobal(vm *VM, ins opcode.InstructionGetGlobal) {
	value := vm.callFrame.function.Executable.Globals[ins.GlobalIndex]
	vm.push(value)
}

func opSetGlobal(vm *VM, ins opcode.InstructionSetGlobal) {
	vm.callFrame.function.Executable.Globals[ins.GlobalIndex] = vm.pop()
}

func opSetIndex(vm *VM) {
	container, index, value := vm.pop3()
	containerValue := container.(interpreter.ValueIndexableValue)
	containerValue.SetKey(
		vm.config,
		EmptyLocationRange,
		index,
		value,
	)
}

func opGetIndex(vm *VM) {
	container, index := vm.pop2()
	containerValue := container.(interpreter.ValueIndexableValue)
	element := containerValue.GetKey(
		vm.config,
		EmptyLocationRange,
		index,
	)
	vm.push(element)
}

func opInvoke(vm *VM, ins opcode.InstructionInvoke) {
	value := vm.pop()

	switch value := value.(type) {
	case FunctionValue:
		parameterCount := int(value.Function.ParameterCount)
		arguments := vm.peekN(parameterCount)
		vm.pushCallFrame(value, arguments)
		vm.dropN(len(arguments))

	case NativeFunctionValue:
		parameterCount := value.ParameterCount

		var typeArguments []bbq.StaticType
		for _, index := range ins.TypeArgs {
			typeArg := vm.loadType(index)
			typeArguments = append(typeArguments, typeArg)
		}

		arguments := vm.peekN(parameterCount)
		result := value.Function(vm.config, typeArguments, arguments...)
		vm.dropN(len(arguments))
		vm.push(result)

	default:
		panic(errors.NewUnreachableError())
	}
}

func opInvokeDynamic(vm *VM, ins opcode.InstructionInvokeDynamic) {
	stackHeight := len(vm.stack)
	receiver := vm.stack[stackHeight-int(ins.ArgCount)-1]

	// TODO:
	var typeArguments []bbq.StaticType
	for _, index := range ins.TypeArgs {
		typeArg := vm.loadType(index)
		typeArguments = append(typeArguments, typeArg)
	}
	// TODO: Just to make the linter happy
	_ = typeArguments

	switch typedReceiver := receiver.(type) {
	case interpreter.ReferenceValue:
		referenced := typedReceiver.ReferencedValue(vm.config, EmptyLocationRange, true)
		receiver = *referenced

		// TODO:
		//case ReferenceValue
	}

	compositeValue := receiver.(*interpreter.CompositeValue)

	staticType := compositeValue.StaticType(vm.config)

	// TODO: for inclusive range, this is different.
	compositeType := staticType.(*interpreter.CompositeStaticType)

	funcName := getStringConstant(vm, ins.NameIndex)

	qualifiedFuncName := commons.TypeQualifiedName(compositeType.QualifiedIdentifier, funcName)
	var functionValue = vm.lookupFunction(compositeType.Location, qualifiedFuncName)

	parameterCount := int(functionValue.Function.ParameterCount)
	arguments := vm.peekN(parameterCount)
	vm.pushCallFrame(functionValue, arguments)
	vm.dropN(len(arguments))

	// We do not need to drop the receiver, as the parameter count given in the instruction already includes it
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
	staticType := vm.loadType(ins.TypeIndex)

	// TODO: Support inclusive-range type
	compositeStaticType := staticType.(*interpreter.CompositeStaticType)

	config := vm.config

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
	fieldName := getStringConstant(vm, ins.FieldNameIndex)

	target.(interpreter.MemberAccessibleValue).
		SetMember(vm.config, EmptyLocationRange, fieldName, fieldValue)
}

func opGetField(vm *VM, ins opcode.InstructionGetField) {
	memberAccessibleValue := vm.pop().(interpreter.MemberAccessibleValue)

	// VM assumes the field name is always a string.
	fieldName := getStringConstant(vm, ins.FieldNameIndex)

	fieldValue := memberAccessibleValue.GetMember(vm.config, EmptyLocationRange, fieldName)
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
	targetType := vm.loadType(ins.TypeIndex)
	value := vm.peek()

	config := vm.config

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
	if valueType != nil && !vm.config.IsSubType(valueType, targetType) {
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
	value.Destroy(vm.config, EmptyLocationRange)
}

func opPath(vm *VM, ins opcode.InstructionPath) {
	identifier := getStringConstant(vm, ins.IdentifierIndex)
	value := interpreter.NewPathValue(
		vm.config.MemoryGauge,
		ins.Domain,
		identifier,
	)
	vm.push(value)
}

func opSimpleCast(vm *VM, ins opcode.InstructionSimpleCast) {
	value := vm.pop()

	targetType := vm.loadType(ins.TypeIndex)
	valueType := value.StaticType(vm.config)

	// The cast may upcast to an optional type, e.g. `1 as Int?`, so box
	result := ConvertAndBox(vm.config, value, valueType, targetType)

	vm.push(result)
}

func opFailableCast(vm *VM, ins opcode.InstructionFailableCast) {
	value := vm.pop()

	targetType := vm.loadType(ins.TypeIndex)
	value, valueType := castValueAndValueType(vm.config, targetType, value)
	isSubType := vm.config.IsSubType(valueType, targetType)

	var result Value
	if isSubType {
		// The failable cast may upcast to an optional type, e.g. `1 as? Int?`, so box
		result = ConvertAndBox(vm.config, value, valueType, targetType)

		// TODO:
		// Failable casting is a resource invalidation
		//interpreter.invalidateResource(value)

		result = interpreter.NewSomeValueNonCopying(
			vm.config.MemoryGauge,
			result,
		)
	} else {
		result = interpreter.Nil
	}

	vm.push(result)
}

func opForceCast(vm *VM, ins opcode.InstructionForceCast) {
	value := vm.pop()

	targetType := vm.loadType(ins.TypeIndex)
	value, valueType := castValueAndValueType(vm.config, targetType, value)
	isSubType := vm.config.IsSubType(valueType, targetType)

	var result Value
	if !isSubType {
		targetSemaType := interpreter.MustConvertStaticToSemaType(targetType, vm.config)
		valueSemaType := interpreter.MustConvertStaticToSemaType(valueType, vm.config)

		panic(interpreter.ForceCastTypeMismatchError{
			ExpectedType: targetSemaType,
			ActualType:   valueSemaType,
		})
	}

	// The force cast may upcast to an optional type, e.g. `1 as! Int?`, so box
	result = ConvertAndBox(vm.config, value, valueType, targetType)
	vm.push(result)
}

func castValueAndValueType(config *Config, targetType bbq.StaticType, value Value) (Value, bbq.StaticType) {
	valueType := value.StaticType(config)

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
		vm.config,
		EmptyLocationRange,
		right,
	)
	vm.replaceTop(interpreter.BoolValue(result))
}

func opNotEqual(vm *VM) {
	left, right := vm.peekPop()
	result := !left.(interpreter.EquatableValue).Equal(
		vm.config,
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
	typ := vm.loadType(ins.TypeIndex).(interpreter.ArrayStaticType)

	elements := vm.peekN(int(ins.Size))
	array := interpreter.NewArrayValue(
		vm.config,
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
	typ := vm.loadType(ins.TypeIndex).(*interpreter.DictionaryStaticType)

	entries := vm.peekN(int(ins.Size * 2))
	dictionary := interpreter.NewDictionaryValue(
		vm.config,
		EmptyLocationRange,
		typ,
		entries...,
	)
	vm.dropN(len(entries))

	vm.push(dictionary)
}

func opNewRef(vm *VM, ins opcode.InstructionNewRef) {
	borrowedType := vm.loadType(ins.TypeIndex)
	value := vm.pop()

	semaBorrowedType := interpreter.MustConvertStaticToSemaType(borrowedType, vm.config)

	ref := interpreter.CreateReferenceValue(
		vm.config,
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
	iterator := iterable.Iterator(vm.config, EmptyLocationRange)
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
	element := iterator.Next(vm.config, EmptyLocationRange)
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
		vm.config,
		EmptyLocationRange,
		true,
	)

	if isOptional {
		return interpreter.NewSomeValueNonCopying(
			vm.config.MemoryGauge,
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
		case opcode.InstructionInvokeDynamic:
			opInvokeDynamic(vm, ins)
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
		case opcode.InstructionPath:
			opPath(vm, ins)
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

	onEventEmitted := vm.config.OnEventEmitted
	if onEventEmitted == nil {
		return
	}

	eventType := vm.loadType(ins.TypeIndex).(*interpreter.CompositeStaticType)

	err := onEventEmitted(eventValue, eventType)
	if err != nil {
		panic(err)
	}
}

func opNewClosure(vm *VM, ins opcode.InstructionNewClosure) {

	executable := vm.callFrame.function.Executable
	function := &executable.Program.Functions[ins.FunctionIndex]

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

	vm.push(FunctionValue{
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

	constant := executable.Program.Constants[index]
	kind := constant.Kind
	data := constant.Data

	memoryGauge := vm.config.MemoryGauge

	switch kind {
	case constantkind.String:
		value = interpreter.NewUnmeteredStringValue(string(data))

	case constantkind.Int:
		// TODO: support larger integers
		v, _, err := leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int constant: %s", err))
		}
		value = interpreter.NewIntValueFromInt64(memoryGauge, v)

	case constantkind.Int8:
		v, _, err := leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int8 constant: %s", err))
		}
		value = interpreter.NewInt8Value(
			memoryGauge,
			func() int8 {
				return int8(v)
			},
		)

	case constantkind.Int16:
		v, _, err := leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int16 constant: %s", err))
		}
		value = interpreter.NewInt16Value(
			memoryGauge,
			func() int16 {
				return int16(v)
			},
		)

	case constantkind.Int32:
		v, _, err := leb128.ReadInt32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int32 constant: %s", err))
		}
		value = interpreter.NewInt32Value(
			memoryGauge,
			func() int32 {
				return v
			},
		)

	case constantkind.Int64:
		v, _, err := leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Int64 constant: %s", err))
		}
		value = interpreter.NewInt64Value(
			memoryGauge,
			func() int64 {
				return v
			},
		)

	case constantkind.UInt:
		// TODO: support larger integers
		v, _, err := leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt constant: %s", err))
		}
		value = interpreter.NewUIntValueFromUint64(memoryGauge, v)

	case constantkind.UInt8:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt8 constant: %s", err))
		}
		value = interpreter.NewUInt8Value(
			memoryGauge,
			func() uint8 {
				return uint8(v)
			},
		)

	case constantkind.UInt16:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt16 constant: %s", err))
		}
		value = interpreter.NewUInt16Value(
			memoryGauge,
			func() uint16 {
				return uint16(v)
			},
		)

	case constantkind.UInt32:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt32 constant: %s", err))
		}
		value = interpreter.NewUInt32Value(
			memoryGauge,
			func() uint32 {
				return v
			},
		)

	case constantkind.UInt64:
		v, _, err := leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UInt64 constant: %s", err))
		}
		value = interpreter.NewUInt64Value(
			memoryGauge,
			func() uint64 {
				return v
			},
		)

	case constantkind.Word8:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word8 constant: %s", err))
		}
		value = interpreter.NewWord8Value(
			memoryGauge,
			func() uint8 {
				return uint8(v)
			},
		)

	case constantkind.Word16:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word16 constant: %s", err))
		}
		value = interpreter.NewWord16Value(
			memoryGauge,
			func() uint16 {
				return uint16(v)
			},
		)

	case constantkind.Word32:
		v, _, err := leb128.ReadUint32(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word32 constant: %s", err))
		}
		value = interpreter.NewWord32Value(
			memoryGauge,
			func() uint32 {
				return v
			},
		)

	case constantkind.Word64:
		v, _, err := leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Word64 constant: %s", err))
		}
		value = interpreter.NewWord64Value(
			memoryGauge,
			func() uint64 {
				return v
			},
		)

	case constantkind.Fix64:
		v, _, err := leb128.ReadInt64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read Fix64 constant: %s", err))
		}
		value = interpreter.NewUnmeteredFix64Value(v)

	case constantkind.UFix64:
		v, _, err := leb128.ReadUint64(data)
		if err != nil {
			panic(errors.NewUnexpectedError("failed to read UFix64 constant: %s", err))
		}
		value = interpreter.NewUnmeteredUFix64Value(v)

	case constantkind.Address:
		value = interpreter.NewAddressValueFromBytes(
			memoryGauge,
			func() []byte {
				return data
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
		panic(errors.NewUnexpectedError("unsupported constant kind: %s", kind))
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

func (vm *VM) lookupFunction(location common.Location, name string) FunctionValue {
	// First check in current program.
	value, ok := vm.globals[name]
	if ok {
		return value.(FunctionValue)
	}

	// If not found, check in already linked imported functions.
	linkedGlobals, ok := vm.linkedGlobalsCache[location]

	// If not found, link the function now, dynamically.
	if !ok {
		// TODO: This currently link all functions in program, unnecessarily.
		//   Link only the requested function.
		program := vm.config.ImportHandler(location)

		linkedGlobals = LinkGlobals(
			location,
			program,
			vm.config,
			vm.linkedGlobalsCache,
		)
	}

	value, ok = linkedGlobals.indexedGlobals[name]
	if !ok {
		panic(errors.NewUnexpectedError("cannot link global: %s", name))
	}

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
}

func getReceiver[T any](config *Config, receiver Value) T {
	switch receiver := receiver.(type) {
	case *interpreter.SomeValue:
		return getReceiver[T](config, receiver.InnerValue())
	case *interpreter.EphemeralReferenceValue:
		return getReceiver[T](config, receiver.Value)
	case *interpreter.StorageReferenceValue:
		referencedValue := receiver.ReferencedValue(
			config,
			EmptyLocationRange,
			true,
		)
		return getReceiver[T](config, *referencedValue)
	default:
		return receiver.(T)
	}
}
