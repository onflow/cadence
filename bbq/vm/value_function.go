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

	// HasGenericType returns whether this function has a derived-type.
	// A function is said to have a derived-typed if the type of the function
	// is dependent on the receiver.
	// for e.g: `Integer.toBigEndianBytes()` functions type is always `fun(): [UInt8]`.
	// Hence it does not have a derived type.
	// On the other hand, `[T].append()` function's type is `fun(T): Void`,
	// where the parameter type `T` always depends on the receiver's type.
	// Hence, the array-append function is said to have a derived type.
	HasGenericType() bool

	// ResolvedFunctionType returns the resolved type of the function using the provided receiver value,
	// if the function has a generic type. This would panic if the function is not a generic function.
	// Use `HasGenericType` to determine whether this method should be called or not.
	ResolvedFunctionType(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType
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

func (v CompiledFunctionValue) HasGenericType() bool {
	return false
}

func (v CompiledFunctionValue) ResolvedFunctionType(_ Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	return v.FunctionType(context)
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

func (v CompiledFunctionValue) FunctionType(interpreter.ValueStaticTypeContext) *sema.FunctionType {
	return v.Type.Type
}

func (v CompiledFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
	)
}

type NativeFunction func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value

type NativeFunctionValue struct {
	Name     string
	Function NativeFunction

	// A function value can only have either one of `functionType` or `functionTypeGetter`.
	functionType       *sema.FunctionType
	functionTypeGetter func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType
}

func NewNativeFunctionValue(
	name string,
	funcType *sema.FunctionType,
	function NativeFunction,
) NativeFunctionValue {
	return NativeFunctionValue{
		Name:         name,
		Function:     function,
		functionType: funcType,
	}
}

func NewNativeFunctionValueWithDerivedType(
	name string,
	typeGetter func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType,
	function NativeFunction,
) NativeFunctionValue {
	return NativeFunctionValue{
		Name:               name,
		Function:           function,
		functionTypeGetter: typeGetter,
	}
}

var _ Value = NativeFunctionValue{}
var _ FunctionValue = NativeFunctionValue{}

func (NativeFunctionValue) IsValue() {}

func (v NativeFunctionValue) IsFunctionValue() {}

func (v NativeFunctionValue) StaticType(context interpreter.ValueStaticTypeContext) bbq.StaticType {
	// Get the type using `self.FunctionType()`, which panics if the type needs to be derived.
	// This is correct/expected, since this method (`StaticType`) should've never been called,
	// if the function's type needs to be derived.
	semaFunctionType := v.FunctionType(context)

	return interpreter.NewFunctionStaticType(
		nil,
		semaFunctionType,
	)
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
	if v.HasGenericType() {
		// If the type is not pre-known, just return the name.
		return v.Name
	}

	return v.functionType.String()
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
	if v.HasGenericType() {
		// If the type is not pre-known, just return the name.
		return v.Name
	}

	return v.StaticType(context).MeteredString(context)
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

func (v NativeFunctionValue) FunctionType(interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionTypeGetter != nil {
		// For native functions where the type is NOT pre-known, This method should never be invoked.
		// Such functions must always be wrapped with a `BoundFunctionPointerValue`.
		panic(errors.NewUnreachableError())
	}
	return v.functionType
}

func (v NativeFunctionValue) ResolvedFunctionType(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionTypeGetter == nil {
		// ResolvedFunctionType shouldn't get called for functions where the type is pre-know.
		panic(errors.NewUnreachableError())
	}

	// Important: Never store the result of the `functionTypeGetter`,
	// because the `NativeFunctionValue` would be reused.
	return v.functionTypeGetter(receiver, context)
}

func (v NativeFunctionValue) HasGenericType() bool {
	return v.functionTypeGetter != nil
}

func (v NativeFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
	)
}

// BoundFunctionPointerValue is a function-pointer taken for an object-method.
type BoundFunctionPointerValue struct {
	receiverReference   interpreter.ReferenceValue
	receiverIsReference bool

	Method       FunctionValue
	functionType *sema.FunctionType
}

func NewBoundFunctionPointerValue(
	context interpreter.ReferenceCreationContext,
	receiver interpreter.MemberAccessibleValue,
	method FunctionValue,
) FunctionValue {

	// Since 'self' work as an implicit reference, create an explicit one and hold it.
	// This reference is later used to check the validity of the referenced value/resource.
	// For attachments, 'self' is already a reference. So no need to create a reference again.

	receiverRef, receiverIsRef := (receiver).(interpreter.ReferenceValue)
	if !receiverIsRef {
		semaType := interpreter.MustSemaTypeOfValue(receiver, context)
		// Create an unauthorized reference. The purpose of it is only to track and invalidate resource moves,
		// it is not directly exposed to the users
		receiverRef = interpreter.NewEphemeralReferenceValue(
			context,
			interpreter.UnauthorizedAccess,
			receiver,
			semaType,
			EmptyLocationRange,
		)
	}

	return &BoundFunctionPointerValue{
		Method:              method,
		receiverReference:   receiverRef,
		receiverIsReference: receiverIsRef,
	}
}

var _ Value = &BoundFunctionPointerValue{}
var _ FunctionValue = &BoundFunctionPointerValue{}

func (*BoundFunctionPointerValue) IsValue() {}

func (v *BoundFunctionPointerValue) IsFunctionValue() {}

func (v *BoundFunctionPointerValue) HasGenericType() bool {
	return v.Method.HasGenericType()
}

func (v *BoundFunctionPointerValue) ResolvedFunctionType(_ Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	return v.FunctionType(context)
}

func (v *BoundFunctionPointerValue) StaticType(context interpreter.ValueStaticTypeContext) bbq.StaticType {
	if v.functionType == nil {
		// initialize `v.functionType` field
		v.initializeFunctionType(context)
	}

	return interpreter.NewFunctionStaticType(context, v.functionType)
}

func (v *BoundFunctionPointerValue) Transfer(_ interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v *BoundFunctionPointerValue) String() string {
	return v.Method.String()
}

func (v *BoundFunctionPointerValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v *BoundFunctionPointerValue) Accept(
	_ interpreter.ValueVisitContext,
	_ interpreter.Visitor,
	_ interpreter.LocationRange,
) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v *BoundFunctionPointerValue) Walk(
	_ interpreter.ValueWalkContext,
	_ func(interpreter.Value),
	_ interpreter.LocationRange,
) {
	// NO-OP
}

func (v *BoundFunctionPointerValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.LocationRange,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v *BoundFunctionPointerValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v *BoundFunctionPointerValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
	_ interpreter.LocationRange,
) string {
	functionType := v.StaticType(context)
	return functionType.MeteredString(context)
}

func (v *BoundFunctionPointerValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v *BoundFunctionPointerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v *BoundFunctionPointerValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *BoundFunctionPointerValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v *BoundFunctionPointerValue) IsImportable(
	_ interpreter.ValueImportableContext,
	_ interpreter.LocationRange,
) bool {
	return false
}

func (v *BoundFunctionPointerValue) FunctionType(context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionType == nil {
		v.initializeFunctionType(context)
	}

	return v.functionType
}

func (v *BoundFunctionPointerValue) initializeFunctionType(context interpreter.ValueStaticTypeContext) {
	method := v.Method
	// The type of the native function could be either pre-known (e.g: `Integer.toBigEndianBytes()`),
	// Or would needs to be derived based on the receiver (e.g: `[Int8].append()`).
	if method.HasGenericType() {
		v.functionType = method.ResolvedFunctionType(
			v.Receiver(context),
			context,
		)
	} else {
		v.functionType = method.FunctionType(context)
	}
}

func (v *BoundFunctionPointerValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	arguments := make([]Value, 0, len(invocation.Arguments)+1)
	arguments = append(arguments, v.Receiver(invocation.InvocationContext))
	arguments = append(arguments, invocation.Arguments...)

	return invocation.InvocationContext.InvokeFunction(
		v,
		arguments,
	)
}

func (v *BoundFunctionPointerValue) Receiver(context interpreter.ValueStaticTypeContext) Value {
	receiver := interpreter.GetReceiver(
		v.receiverReference,
		v.receiverIsReference,
		context,
		EmptyLocationRange,
	)
	return maybeDereference(context, receiver)
}
