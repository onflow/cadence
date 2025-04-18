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
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type FunctionValue interface {
	interpreter.FunctionValue
	GetParameterCount() int
}

type CompiledFunctionValue struct {
	Function   *bbq.Function[opcode.Instruction]
	Executable *ExecutableProgram
	Upvalues   []*Upvalue
	Type       interpreter.FunctionStaticType
}

var _ Value = CompiledFunctionValue{}
var _ FunctionValue = CompiledFunctionValue{}

func (CompiledFunctionValue) IsValue() {}

func (v CompiledFunctionValue) IsFunctionValue() {}

func (v CompiledFunctionValue) GetParameterCount() int {
	return int(v.Function.ParameterCount)
}

func (v CompiledFunctionValue) StaticType(interpreter.ValueStaticTypeContext) bbq.StaticType {
	return v.Type
}

func (v CompiledFunctionValue) Transfer(
	context interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	if remove {
		interpreter.RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v CompiledFunctionValue) String() string {
	return v.Type.String()
}

func (v CompiledFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v CompiledFunctionValue) Accept(
	_ interpreter.ValueVisitContext,
	_ interpreter.Visitor,
	_ interpreter.LocationRange,
) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v CompiledFunctionValue) Walk(
	_ interpreter.ValueWalkContext,
	_ func(interpreter.Value),
	_ interpreter.LocationRange,
) {
	// NO-OP
}

func (v CompiledFunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.LocationRange,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v CompiledFunctionValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v CompiledFunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
	_ interpreter.LocationRange,
) string {
	return v.Type.MeteredString(context)
}

func (v CompiledFunctionValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v CompiledFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v CompiledFunctionValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v CompiledFunctionValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v CompiledFunctionValue) IsImportable(_ interpreter.ValueImportableContext, _ interpreter.LocationRange) bool {
	return false
}

func (v CompiledFunctionValue) FunctionType() *sema.FunctionType {
	return v.Type.Type
}

func (v CompiledFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
		invocation.ArgumentTypes,
		invocation.LocationRange,
	)
}

type NativeFunction func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value

type NativeFunctionValue struct {
	Name string
	// ParameterCount includes actual parameters + receiver for type-bound functions.
	ParameterCount int
	Function       NativeFunction
	Type           interpreter.FunctionStaticType
}

func NewNativeFunctionValue(
	name string,
	funcType *sema.FunctionType,
	function NativeFunction,
) NativeFunctionValue {
	return NativeFunctionValue{
		Name:           name,
		ParameterCount: len(funcType.Parameters),
		Function:       function,
		Type:           interpreter.NewFunctionStaticType(nil, funcType),
	}
}

func NewBoundNativeFunctionValue(
	name string,
	funcType *sema.FunctionType,
	function NativeFunction,
) NativeFunctionValue {
	return NativeFunctionValue{
		Name:           name,
		ParameterCount: len(funcType.Parameters) + 1, // +1 is for the receiver
		Function:       function,
		Type:           interpreter.NewFunctionStaticType(nil, funcType),
	}
}

var _ Value = NativeFunctionValue{}
var _ FunctionValue = NativeFunctionValue{}

func (NativeFunctionValue) IsValue() {}

func (v NativeFunctionValue) IsFunctionValue() {}

func (v NativeFunctionValue) GetParameterCount() int {
	return v.ParameterCount
}

func (v NativeFunctionValue) StaticType(interpreter.ValueStaticTypeContext) bbq.StaticType {
	return v.Type
}

func (v NativeFunctionValue) Transfer(_ interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v NativeFunctionValue) String() string {
	return v.Type.String()
}

func (v NativeFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v NativeFunctionValue) Accept(
	_ interpreter.ValueVisitContext,
	_ interpreter.Visitor,
	_ interpreter.LocationRange,
) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Walk(
	_ interpreter.ValueWalkContext,
	_ func(interpreter.Value),
	_ interpreter.LocationRange,
) {
	// NO-OP
}

func (v NativeFunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.LocationRange,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v NativeFunctionValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v NativeFunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
	_ interpreter.LocationRange,
) string {
	return v.Type.MeteredString(context)
}

func (v NativeFunctionValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v NativeFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v NativeFunctionValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v NativeFunctionValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v NativeFunctionValue) IsImportable(
	_ interpreter.ValueImportableContext,
	_ interpreter.LocationRange,
) bool {
	return false
}

func (v NativeFunctionValue) FunctionType() *sema.FunctionType {
	return v.Type.Type
}

func (v NativeFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
		invocation.ArgumentTypes,
		invocation.LocationRange,
	)
}

// BoundFunctionPointerValue is a function-pointer taken for an object-method.
type BoundFunctionPointerValue struct {
	Receiver       interpreter.MemberAccessibleValue
	Method         FunctionValue
	ParameterCount int
}

func NewBoundFunctionPointerValue(
	receiver interpreter.MemberAccessibleValue,
	method FunctionValue,
) FunctionValue {
	return &BoundFunctionPointerValue{
		Receiver:       receiver,
		Method:         method,
		ParameterCount: method.GetParameterCount() - 1,
	}
}

var _ Value = BoundFunctionPointerValue{}
var _ FunctionValue = BoundFunctionPointerValue{}

func (BoundFunctionPointerValue) IsValue() {}

func (v BoundFunctionPointerValue) IsFunctionValue() {}

func (v BoundFunctionPointerValue) GetParameterCount() int {
	return v.ParameterCount
}

func (v BoundFunctionPointerValue) StaticType(context interpreter.ValueStaticTypeContext) bbq.StaticType {
	return v.Method.StaticType(context)
}

func (v BoundFunctionPointerValue) Transfer(_ interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v BoundFunctionPointerValue) String() string {
	return v.Method.String()
}

func (v BoundFunctionPointerValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v BoundFunctionPointerValue) Accept(
	_ interpreter.ValueVisitContext,
	_ interpreter.Visitor,
	_ interpreter.LocationRange,
) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v BoundFunctionPointerValue) Walk(
	_ interpreter.ValueWalkContext,
	_ func(interpreter.Value),
	_ interpreter.LocationRange,
) {
	// NO-OP
}

func (v BoundFunctionPointerValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.LocationRange,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v BoundFunctionPointerValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v BoundFunctionPointerValue) MeteredString(
	context interpreter.ValueStringContext,
	seenreferences interpreter.SeenReferences,
	locationRange interpreter.LocationRange,
) string {
	return v.Method.MeteredString(context, seenreferences, locationRange)
}

func (v BoundFunctionPointerValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v BoundFunctionPointerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v BoundFunctionPointerValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v BoundFunctionPointerValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v BoundFunctionPointerValue) IsImportable(
	_ interpreter.ValueImportableContext,
	_ interpreter.LocationRange,
) bool {
	return false
}

func (v BoundFunctionPointerValue) FunctionType() *sema.FunctionType {
	return v.Method.FunctionType()
}

func (v BoundFunctionPointerValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
		invocation.ArgumentTypes,
		invocation.LocationRange,
	)
}
