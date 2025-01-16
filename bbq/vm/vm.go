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
	callFrame callFrame

	ipStack []uint16
	ip      uint16

	config             *Config
	globals            map[string]Value
	linkedGlobalsCache map[common.Location]LinkedGlobals
}

func NewVM(
	location common.Location,
	program *bbq.Program[opcode.Instruction],
	conf *Config,
) *VM {
	// TODO: Remove initializing config. Following is for testing purpose only.
	if conf == nil {
		conf = &Config{}
	}
	if conf.Storage == nil {
		conf.Storage = interpreter.NewInMemoryStorage(nil)
	}

	if conf.NativeFunctionsProvider == nil {
		conf.NativeFunctionsProvider = NativeFunctions
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

func (vm *VM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() Value {
	lastIndex := len(vm.stack) - 1
	value := vm.stack[lastIndex]
	vm.stack[lastIndex] = nil
	vm.stack = vm.stack[:lastIndex]

	checkInvalidatedResourceOrResourceReference(value)

	return value
}

// pop2 removes and returns the top two value of the stack.
// It is efficient than calling `pop` twice.
func (vm *VM) pop2() (Value, Value) {
	lastIndex := len(vm.stack) - 1
	value1, value2 := vm.stack[lastIndex], vm.stack[lastIndex-1]
	vm.stack[lastIndex], vm.stack[lastIndex-1] = nil, nil
	vm.stack = vm.stack[:lastIndex-1]
	return value1, value2
}

// pop3 removes and returns the top three value of the stack.
// It is efficient than calling `pop` thrice.
func (vm *VM) pop3() (Value, Value, Value) {
	lastIndex := len(vm.stack) - 1
	value1, value2, value3 := vm.stack[lastIndex], vm.stack[lastIndex-1], vm.stack[lastIndex-2]
	vm.stack[lastIndex], vm.stack[lastIndex-1], vm.stack[lastIndex-2] = nil, nil, nil
	vm.stack = vm.stack[:lastIndex-2]
	return value1, value2, value3
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
		function:     functionValue.Function,
		executable:   functionValue.Executable,
	}

	vm.ipStack = append(vm.ipStack, 0)
	vm.ip = 0

	vm.callstack = append(vm.callstack, callFrame)
	vm.callFrame = callFrame
}

func (vm *VM) popCallFrame() {
	vm.locals = vm.locals[:vm.callFrame.localsOffset]

	newIpStackDepth := len(vm.ipStack) - 1
	vm.ipStack = vm.ipStack[:newIpStackDepth]

	newStackDepth := len(vm.callstack) - 1
	vm.callstack = vm.callstack[:newStackDepth]

	if newStackDepth == 0 {
		vm.ip = 0
	} else {
		vm.ip = vm.ipStack[newIpStackDepth-1]
		vm.callFrame = vm.callstack[newStackDepth-1]
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

func (vm *VM) ExecuteTransaction(transactionArgs []Value, signers ...Value) error {
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

var voidValue = VoidValue{}

func opReturn(vm *VM) {
	vm.popCallFrame()
	vm.push(voidValue)
}

func opJump(vm *VM, ins opcode.InstructionJump) {
	vm.ip = ins.Target
}

func opJumpIfFalse(vm *VM, ins opcode.InstructionJumpIfFalse) {
	value := vm.pop().(BoolValue)
	if !value {
		vm.ip = ins.Target
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

func opBinaryIntLessOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.LessOrEqual(rightNumber))
}

func opBinaryIntGreater(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.Greater(rightNumber))
}

func opBinaryIntGreaterOrEqual(vm *VM) {
	left, right := vm.peekPop()
	leftNumber := left.(IntValue)
	rightNumber := right.(IntValue)
	vm.replaceTop(leftNumber.GreaterOrEqual(rightNumber))
}

func opTrue(vm *VM) {
	vm.push(TrueValue)
}

func opFalse(vm *VM) {
	vm.push(FalseValue)
}

func opGetConstant(vm *VM, ins opcode.InstructionGetConstant) {
	constant := vm.callFrame.executable.Constants[ins.ConstantIndex]
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

func opGetGlobal(vm *VM, ins opcode.InstructionGetGlobal) {
	value := vm.callFrame.executable.Globals[ins.GlobalIndex]
	vm.push(value)
}

func opSetGlobal(vm *VM, ins opcode.InstructionSetGlobal) {
	vm.callFrame.executable.Globals[ins.GlobalIndex] = vm.pop()
}

func opSetIndex(vm *VM) {
	index, array, element := vm.pop3()
	indexValue := index.(IntValue)
	arrayValue := array.(*ArrayValue)
	arrayValue.Set(vm.config, int(indexValue.SmallInt), element)
}

func opGetIndex(vm *VM) {
	index, array := vm.pop2()
	indexValue := index.(IntValue)
	arrayValue := array.(*ArrayValue)
	element := arrayValue.Get(vm.config, int(indexValue.SmallInt))
	vm.push(element)
}

func opInvoke(vm *VM, ins opcode.InstructionInvoke) {
	value := vm.pop()
	stackHeight := len(vm.stack)

	switch value := value.(type) {
	case FunctionValue:
		parameterCount := int(value.Function.ParameterCount)
		arguments := vm.stack[stackHeight-parameterCount:]
		vm.pushCallFrame(value, arguments)
		vm.dropN(parameterCount)

	case NativeFunctionValue:
		parameterCount := value.ParameterCount

		var typeArguments []StaticType
		for _, index := range ins.TypeArgs {
			typeArg := vm.loadType(index)
			typeArguments = append(typeArguments, typeArg)
		}

		arguments := vm.stack[stackHeight-parameterCount:]

		result := value.Function(vm.config, typeArguments, arguments...)
		vm.dropN(parameterCount)
		vm.push(result)

	default:
		panic(errors.NewUnreachableError())
	}
}

func opInvokeDynamic(vm *VM, ins opcode.InstructionInvokeDynamic) {
	stackHeight := len(vm.stack)
	receiver := vm.stack[stackHeight-int(ins.ArgCount)-1]

	// TODO:
	var typeArguments []StaticType
	for _, index := range ins.TypeArgs {
		typeArg := vm.loadType(index)
		typeArguments = append(typeArguments, typeArg)
	}
	// TODO: Just to make the linter happy
	_ = typeArguments

	switch typedReceiver := receiver.(type) {
	case *StorageReferenceValue:
		referenced, err := typedReceiver.dereference(vm.config)
		if err != nil {
			panic(err)
		}
		receiver = *referenced

		// TODO:
		//case ReferenceValue
	}

	compositeValue := receiver.(*CompositeValue)
	compositeType := compositeValue.CompositeType

	funcName := getStringConstant(vm, ins.NameIndex)

	qualifiedFuncName := commons.TypeQualifiedName(compositeType.QualifiedIdentifier, funcName)
	var functionValue = vm.lookupFunction(compositeType.Location, qualifiedFuncName)

	parameterCount := int(functionValue.Function.ParameterCount)
	arguments := vm.stack[stackHeight-parameterCount:]
	vm.pushCallFrame(functionValue, arguments)
	vm.dropN(parameterCount)
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

	value := NewCompositeValue(
		compositeKind,
		compositeStaticType,
		vm.config.Storage,
	)
	vm.push(value)
}

func opSetField(vm *VM, ins opcode.InstructionSetField) {
	// TODO: support all container types
	structValue := vm.pop().(MemberAccessibleValue)

	fieldValue := vm.pop()

	// VM assumes the field name is always a string.
	fieldName := getStringConstant(vm, ins.FieldNameIndex)

	structValue.SetMember(vm.config, fieldName, fieldValue)
}

func opGetField(vm *VM, ins opcode.InstructionGetField) {
	memberAccessibleValue := vm.pop().(MemberAccessibleValue)

	// VM assumes the field name is always a string.
	fieldName := getStringConstant(vm, ins.FieldNameIndex)

	fieldValue := memberAccessibleValue.GetMember(vm.config, fieldName)
	if fieldValue == nil {
		panic(MissingMemberValueError{
			Parent: memberAccessibleValue,
			Name:   fieldName,
		})
	}

	vm.push(fieldValue)
}

func getStringConstant(vm *VM, index uint16) string {
	constant := vm.callFrame.executable.Program.Constants[index]
	return string(constant.Data)
}

func opTransfer(vm *VM, ins opcode.InstructionTransfer) {
	targetType := vm.loadType(ins.TypeIndex)
	value := vm.peek()

	config := vm.config

	transferredValue := value.Transfer(
		config,
		atree.Address{},
		false, nil,
	)

	valueType := transferredValue.StaticType(config)
	if !IsSubType(config, valueType, targetType) {
		panic(errors.NewUnexpectedError("invalid transfer: expected '%s', found '%s'", targetType, valueType))
	}

	vm.replaceTop(transferredValue)
}

func opDestroy(vm *VM) {
	value := vm.pop().(*CompositeValue)
	value.Destroy(vm.config)
}

func opPath(vm *VM, ins opcode.InstructionPath) {
	identifier := getStringConstant(vm, ins.IdentifierIndex)
	value := PathValue{
		Domain:     ins.Domain,
		Identifier: identifier,
	}
	vm.push(value)
}

func opCast(vm *VM, ins opcode.InstructionCast) {
	value := vm.pop()

	targetType := vm.loadType(ins.TypeIndex)

	// TODO:
	_ = ins.Kind
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

func opNotEqual(vm *VM) {
	left, right := vm.peekPop()
	vm.replaceTop(BoolValue(left != right))
}

func opNot(vm *VM) {
	value := vm.peek().(BoolValue)
	vm.replaceTop(!value)
}

func opUnwrap(vm *VM) {
	value := vm.peek()
	if someValue, ok := value.(*SomeValue); ok {
		value = someValue.value
		vm.replaceTop(value)
	}
}

func opNewArray(vm *VM, ins opcode.InstructionNewArray) {

	typ := vm.loadType(ins.TypeIndex).(interpreter.ArrayStaticType)

	elements := make([]Value, ins.Size)

	// Must be inserted in the reverse,
	// since the stack if FILO.
	for i := int(ins.Size) - 1; i >= 0; i-- {
		elements[i] = vm.pop()
	}

	array := NewArrayValue(vm.config, typ, ins.IsResource, elements...)
	vm.push(array)
}

func opNewRef(vm *VM, ins opcode.InstructionNewRef) {

	borrowedType := vm.loadType(ins.TypeIndex).(*interpreter.ReferenceStaticType)
	value := vm.pop()

	ref := NewEphemeralReferenceValue(
		vm.config,
		value,
		borrowedType.Authorization,
		borrowedType.ReferencedType,
	)
	vm.push(ref)
}

func (vm *VM) run() {
	for {

		callFrame := vm.callFrame

		if len(vm.callstack) == 0 ||
			int(vm.ip) >= len(callFrame.function.Code) {

			return
		}

		ins := callFrame.function.Code[vm.ip]
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
		case opcode.InstructionIntAdd:
			opBinaryIntAdd(vm)
		case opcode.InstructionIntSubtract:
			opBinaryIntSubtract(vm)
		case opcode.InstructionIntLess:
			opBinaryIntLess(vm)
		case opcode.InstructionIntLessOrEqual:
			opBinaryIntLessOrEqual(vm)
		case opcode.InstructionIntGreater:
			opBinaryIntGreater(vm)
		case opcode.InstructionIntGreaterOrEqual:
			opBinaryIntGreaterOrEqual(vm)
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
		case opcode.InstructionCast:
			opCast(vm, ins)
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
		default:
			panic(errors.NewUnexpectedError("cannot execute instruction of type %T", ins))
		}

		// Faster in Go <1.19:
		// vmOps[op](vm)
	}
}

func (vm *VM) initializeConstant(index uint16) (value Value) {
	executable := vm.callFrame.executable

	constant := executable.Program.Constants[index]
	switch constant.Kind {
	case constantkind.Int:
		// TODO:
		smallInt, _, _ := leb128.ReadInt64(constant.Data)
		value = NewIntValue(smallInt)
	case constantkind.String:
		value = NewStringValueFromBytes(constant.Data)
	default:
		// TODO:
		panic(errors.NewUnexpectedError("unsupported constant kind '%s'", constant.Kind.String()))
	}

	executable.Constants[index] = value
	return value
}

func (vm *VM) loadType(index uint16) StaticType {
	staticType := vm.callFrame.executable.StaticTypes[index]
	if staticType == nil {
		// TODO: Remove. Should never reach because of the
		// pre loading-decoding of types.
		staticType = vm.initializeType(index)
	}

	return staticType
}

func (vm *VM) initializeType(index uint16) interpreter.StaticType {
	executable := vm.callFrame.executable
	typeBytes := executable.Program.Types[index]
	staticType := decodeType(typeBytes)
	executable.StaticTypes[index] = staticType
	return staticType
}

func decodeType(typeBytes []byte) interpreter.StaticType {
	dec := interpreter.CBORDecMode.NewByteStreamDecoder(typeBytes)
	staticType, err := interpreter.NewTypeDecoder(dec, nil).DecodeStaticType()
	if err != nil {
		panic(err)
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

func getReceiver[T any](config *Config, receiver Value) T {
	switch receiver := receiver.(type) {
	case *SomeValue:
		return getReceiver[T](config, receiver.value)
	case *EphemeralReferenceValue:
		return getReceiver[T](config, receiver.Value)
	case *StorageReferenceValue:
		referencedValue, err := receiver.dereference(nil)
		if err != nil {
			panic(err)
		}
		return getReceiver[T](config, *referencedValue)
	default:
		return receiver.(T)
	}
}
