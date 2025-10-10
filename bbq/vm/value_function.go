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
	"github.com/onflow/cadence/common"
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

	IsNative() bool
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

func (v CompiledFunctionValue) Accept(_ interpreter.ValueVisitContext, _ interpreter.Visitor) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v CompiledFunctionValue) Walk(_ interpreter.ValueWalkContext, _ func(interpreter.Value)) {
	// NO-OP
}

func (v CompiledFunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
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

func (v CompiledFunctionValue) IsImportable(_ interpreter.ValueImportableContext) bool {
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

func (v CompiledFunctionValue) IsNative() bool {
	return false
}

type NativeFunctionValue struct {
	Name     string
	Function interpreter.NativeFunction

	// A function value can only have either one of `functionType` or `functionTypeGetter`.
	functionType       *sema.FunctionType
	functionTypeGetter func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType
	fields             map[string]Value
}

var _ Value = &NativeFunctionValue{}
var _ FunctionValue = &NativeFunctionValue{}
var _ interpreter.MemberAccessibleValue = &NativeFunctionValue{}

func (*NativeFunctionValue) IsValue() {}

func (v *NativeFunctionValue) IsFunctionValue() {}

func (v *NativeFunctionValue) StaticType(context interpreter.ValueStaticTypeContext) bbq.StaticType {
	// Get the type using `self.FunctionType()`, which panics if the type needs to be derived.
	// This is correct/expected, since this method (`StaticType`) should've never been called,
	// if the function's type needs to be derived.
	semaFunctionType := v.FunctionType(context)

	return interpreter.NewFunctionStaticType(
		nil,
		semaFunctionType,
	)
}

func (v *NativeFunctionValue) Transfer(
	_ interpreter.ValueTransferContext,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v *NativeFunctionValue) String() string {
	if v.HasGenericType() {
		// If the type is not pre-known, just return the name.
		return v.Name
	}

	return v.functionType.String()
}

func (v *NativeFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v *NativeFunctionValue) Accept(_ interpreter.ValueVisitContext, _ interpreter.Visitor) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v *NativeFunctionValue) Walk(_ interpreter.ValueWalkContext, _ func(interpreter.Value)) {
	// NO-OP
}

func (v *NativeFunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v *NativeFunctionValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v *NativeFunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
) string {
	if v.HasGenericType() {
		// If the type is not pre-known, just return the name.
		return v.Name
	}

	return v.StaticType(context).MeteredString(context)
}

func (v *NativeFunctionValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v *NativeFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v *NativeFunctionValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *NativeFunctionValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v *NativeFunctionValue) IsImportable(_ interpreter.ValueImportableContext) bool {
	return false
}

func (v *NativeFunctionValue) FunctionType(interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionTypeGetter != nil {
		// For native functions where the type is NOT pre-known, This method should never be invoked.
		// Such functions must always be wrapped with a `BoundFunctionValue`.
		panic(errors.NewUnreachableError())
	}
	return v.functionType
}

func (v *NativeFunctionValue) ResolvedFunctionType(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionTypeGetter == nil {
		// ResolvedFunctionType shouldn't get called for functions where the type is pre-know.
		panic(errors.NewUnreachableError())
	}

	// Important: Never store the result of the `functionTypeGetter`,
	// because the `NativeFunctionValue` would be reused.
	return v.functionTypeGetter(receiver, context)
}

func (v *NativeFunctionValue) HasGenericType() bool {
	return v.functionTypeGetter != nil
}

func (v *NativeFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
	)
}

func (v *NativeFunctionValue) GetMember(context interpreter.MemberAccessibleContext, name string) interpreter.Value {
	value, ok := v.fields[name]
	if ok {
		return value
	}

	if function := context.GetMethod(v, name); function != nil {
		return function
	}

	return nil
}

func (*NativeFunctionValue) RemoveMember(_ interpreter.ValueTransferContext, _ string) interpreter.Value {
	panic(errors.NewUnreachableError())
}

func (*NativeFunctionValue) SetMember(_ interpreter.ValueTransferContext, _ string, _ interpreter.Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *NativeFunctionValue) GetMethod(_ interpreter.MemberAccessibleContext, _ string) interpreter.FunctionValue {
	// Should never be called, VM should not look up method on value.
	// See `NativeFunctionValue.GetMember`
	panic(errors.NewUnreachableError())
}

func (v *NativeFunctionValue) IsNative() bool {
	// Native functions are always native.
	return true
}

// BoundFunctionValue is a function-wrapper which captures the receivers of an object-method.
type BoundFunctionValue struct {
	ReceiverReference   interpreter.ReferenceValue
	receiverIsReference bool

	Method       FunctionValue
	functionType *sema.FunctionType
	Base         *interpreter.EphemeralReferenceValue
}

var boundFunctionMemoryUsage = common.NewConstantMemoryUsage(common.MemoryKindBoundFunctionVMValue)

func NewBoundFunctionValue(
	context interpreter.ReferenceCreationContext,
	receiver interpreter.Value,
	method FunctionValue,
	base *interpreter.EphemeralReferenceValue,
) FunctionValue {

	common.UseMemory(context, boundFunctionMemoryUsage)

	// Since 'self' work as an implicit reference, create an explicit one and hold it.
	// This reference is later used to check the validity of the referenced value/resource.
	// For attachments, 'self' is already a reference. So no need to create a reference again.

	receiverRef, receiverIsRef := interpreter.ReceiverReference(context, receiver)

	if compositeValue, ok := receiver.(*interpreter.CompositeValue); ok && compositeValue.Kind == common.CompositeKindAttachment {
		// Force the receiver to be a reference if it is an attachment.
		// This is because self in attachments are always references.
		receiverIsRef = true
	}

	return &BoundFunctionValue{
		Method:              method,
		ReceiverReference:   receiverRef,
		receiverIsReference: receiverIsRef,
		Base:                base,
	}
}

var _ Value = &BoundFunctionValue{}
var _ FunctionValue = &BoundFunctionValue{}

func (*BoundFunctionValue) IsValue() {}

func (v *BoundFunctionValue) IsFunctionValue() {}

func (v *BoundFunctionValue) HasGenericType() bool {
	return v.Method.HasGenericType()
}

func (v *BoundFunctionValue) ResolvedFunctionType(_ Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	return v.FunctionType(context)
}

func (v *BoundFunctionValue) StaticType(context interpreter.ValueStaticTypeContext) bbq.StaticType {
	if v.functionType == nil {
		// initialize `v.functionType` field
		v.initializeFunctionType(context)
	}

	return interpreter.NewFunctionStaticType(context, v.functionType)
}

func (v *BoundFunctionValue) Transfer(
	_ interpreter.ValueTransferContext,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v *BoundFunctionValue) String() string {
	return v.Method.String()
}

func (v *BoundFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v *BoundFunctionValue) Accept(_ interpreter.ValueVisitContext, _ interpreter.Visitor) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v *BoundFunctionValue) Walk(_ interpreter.ValueWalkContext, _ func(interpreter.Value)) {
	// NO-OP
}

func (v *BoundFunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v *BoundFunctionValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v *BoundFunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
) string {
	functionType := v.StaticType(context)
	return functionType.MeteredString(context)
}

func (v *BoundFunctionValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v *BoundFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v *BoundFunctionValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *BoundFunctionValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v *BoundFunctionValue) IsImportable(_ interpreter.ValueImportableContext) bool {
	return false
}

func (v *BoundFunctionValue) FunctionType(context interpreter.ValueStaticTypeContext) *sema.FunctionType {
	if v.functionType == nil {
		v.initializeFunctionType(context)
	}

	return v.functionType
}

func (v *BoundFunctionValue) initializeFunctionType(context interpreter.ValueStaticTypeContext) {
	method := v.Method
	// The type of the native function could be either pre-known (e.g: `Integer.toBigEndianBytes()`),
	// Or would needs to be derived based on the receiver (e.g: `[Int8].append()`).
	if method.HasGenericType() {
		v.functionType = method.ResolvedFunctionType(
			v.DereferencedReceiver(context),
			context,
		)
	} else {
		v.functionType = method.FunctionType(context)
	}
}

func (v *BoundFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	context := invocation.InvocationContext

	arguments := make([]Value, 0, len(invocation.Arguments))
	arguments = append(arguments, invocation.Arguments...)

	return context.InvokeFunction(
		v,
		arguments,
	)
}

func (v *BoundFunctionValue) DereferencedReceiver(context interpreter.ValueStaticTypeContext) Value {
	receiver := interpreter.GetReceiver(
		v.ReceiverReference,
		v.receiverIsReference,
		context,
	)
	return maybeDereferenceReceiver(context, *receiver, v.IsNative())
}

func (v *BoundFunctionValue) IsNative() bool {
	// BoundFunctionValue is a wrapper around a function value, which can be either native or compiled.
	// So, we delegate the call to the underlying function value.
	return v.Method.IsNative()
}

func (v *BoundFunctionValue) Receiver(context interpreter.ReferenceCreationContext) ImplicitReferenceValue {
	receiverValue := v.DereferencedReceiver(context)
	return NewImplicitReferenceValue(context, receiverValue)
}
