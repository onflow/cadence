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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// FunctionValue
type FunctionValue interface {
	Value
	IsFunctionValue()
	FunctionType(context ValueStaticTypeContext) *sema.FunctionType
	// invoke evaluates the function.
	// Only used internally by the interpreter.
	// Use Interpreter.InvokeFunctionValue if you want to invoke the function externally
	Invoke(Invocation) Value
}

// InterpretedFunctionValue
type InterpretedFunctionValue struct {
	Interpreter      *Interpreter
	ParameterList    *ast.ParameterList
	Type             *sema.FunctionType
	Activation       *VariableActivation
	BeforeStatements []ast.Statement
	PreConditions    []ast.Condition
	Statements       []ast.Statement
	PostConditions   []ast.Condition
}

func NewInterpretedFunctionValue(
	interpreter *Interpreter,
	parameterList *ast.ParameterList,
	functionType *sema.FunctionType,
	lexicalScope *VariableActivation,
	beforeStatements []ast.Statement,
	preConditions []ast.Condition,
	statements []ast.Statement,
	postConditions []ast.Condition,
) *InterpretedFunctionValue {

	common.UseMemory(interpreter, common.InterpretedFunctionValueMemoryUsage)

	return &InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    parameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   postConditions,
	}
}

var _ Value = &InterpretedFunctionValue{}
var _ FunctionValue = &InterpretedFunctionValue{}

func (*InterpretedFunctionValue) IsValue() {}

func (f *InterpretedFunctionValue) String() string {
	return f.Type.String()
}

func (f *InterpretedFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (f *InterpretedFunctionValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	// TODO: Meter sema.Type String conversion
	typeString := f.Type.String()
	common.UseMemory(context, common.NewRawStringMemoryUsage(8+len(typeString)))
	return f.String()
}

func (f *InterpretedFunctionValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitInterpretedFunctionValue(context, f)
}

func (f *InterpretedFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f *InterpretedFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return ConvertSemaToStaticType(context, f.Type)
}

func (*InterpretedFunctionValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return false
}

func (*InterpretedFunctionValue) IsFunctionValue() {}

func (f *InterpretedFunctionValue) FunctionType(ValueStaticTypeContext) *sema.FunctionType {
	return f.Type
}

func (f *InterpretedFunctionValue) Invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Interpreter.invokeInterpretedFunction(f, invocation)
}

func (f *InterpretedFunctionValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f *InterpretedFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*InterpretedFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*InterpretedFunctionValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (f *InterpretedFunctionValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return f
}

func (f *InterpretedFunctionValue) Clone(_ ValueCloneContext) Value {
	return f
}

func (*InterpretedFunctionValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

// HostFunctionValue
type HostFunction func(invocation Invocation) Value

type HostFunctionValue struct {
	Function        HostFunction
	NestedVariables map[string]Variable
	Type            *sema.FunctionType
}

func (f *HostFunctionValue) String() string {
	return f.Type.String()
}

func (f *HostFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (f *HostFunctionValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(context, common.HostFunctionValueStringMemoryUsage)
	return f.String()
}

func NewUnmeteredStaticHostFunctionValue(
	funcType *sema.FunctionType,
	function HostFunction,
) *HostFunctionValue {
	// Host functions can be passed by value,
	// so for the interpreter value transfer check to work,
	// they need a static type
	if funcType == nil {
		panic(errors.NewUnreachableError())
	}

	return &HostFunctionValue{
		Function: function,
		Type:     funcType,
	}
}

// NewStaticHostFunctionValue constructs a host function that is not bounded to any value.
// For constructing a function bound to a value (e.g: a member function), the output of this method
// must be wrapped with a bound-function, or `NewBoundHostFunctionValue` method must be used.
func NewStaticHostFunctionValue(
	gauge common.MemoryGauge,
	funcType *sema.FunctionType,
	function HostFunction,
) *HostFunctionValue {

	common.UseMemory(gauge, common.HostFunctionValueMemoryUsage)

	return NewUnmeteredStaticHostFunctionValue(funcType, function)
}

var _ Value = &HostFunctionValue{}
var _ FunctionValue = &HostFunctionValue{}
var _ MemberAccessibleValue = &HostFunctionValue{}
var _ ContractValue = &HostFunctionValue{}

func (*HostFunctionValue) IsValue() {}

func (f *HostFunctionValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitHostFunctionValue(context, f)
}

func (f *HostFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f *HostFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return ConvertSemaToStaticType(context, f.Type)
}

func (*HostFunctionValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return false
}

func (*HostFunctionValue) IsFunctionValue() {}

func (f *HostFunctionValue) FunctionType(_ ValueStaticTypeContext) *sema.FunctionType {
	return f.Type
}

func (f *HostFunctionValue) Invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Function(invocation)
}

func (f *HostFunctionValue) GetMember(context MemberAccessibleContext, _ LocationRange, name string) Value {
	if f.NestedVariables != nil {
		if variable, ok := f.NestedVariables[name]; ok {
			return variable.GetValue(context)
		}
	}
	return nil
}

func (v *HostFunctionValue) GetMethod(
	_ MemberAccessibleContext,
	_ LocationRange,
	_ string,
) FunctionValue {
	return nil
}

func (*HostFunctionValue) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Host functions have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*HostFunctionValue) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Host functions have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (f *HostFunctionValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f *HostFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*HostFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*HostFunctionValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (f *HostFunctionValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return f
}

func (f *HostFunctionValue) Clone(_ ValueCloneContext) Value {
	return f
}

func (*HostFunctionValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *HostFunctionValue) SetNestedVariables(variables map[string]Variable) {
	v.NestedVariables = variables
}

// BoundFunctionValue
type BoundFunctionValue struct {
	Function        FunctionValue
	Base            *EphemeralReferenceValue
	SelfReference   ReferenceValue
	selfIsReference bool
}

var _ Value = BoundFunctionValue{}
var _ FunctionValue = BoundFunctionValue{}

func NewBoundFunctionValue(
	context FunctionCreationContext,
	function FunctionValue,
	self *Value,
	base *EphemeralReferenceValue,
) BoundFunctionValue {

	// Since 'self' work as an implicit reference, create an explicit one and hold it.
	// This reference is later used to check the validity of the referenced value/resource.
	// For attachments, 'self' is already a reference. So no need to create a reference again.

	selfRef, selfIsRef := (*self).(ReferenceValue)
	if !selfIsRef {
		semaType := MustSemaTypeOfValue(*self, context)
		// Create an unauthorized reference. The purpose of it is only to track and invalidate resource moves,
		// it is not directly exposed to the users
		selfRef = NewEphemeralReferenceValue(context, UnauthorizedAccess, *self, semaType, EmptyLocationRange)
	}

	return NewBoundFunctionValueFromSelfReference(
		context,
		function,
		selfRef,
		selfIsRef,
		base,
	)
}

func NewBoundFunctionValueFromSelfReference(
	gauge common.MemoryGauge,
	function FunctionValue,
	selfReference ReferenceValue,
	selfIsReference bool,
	base *EphemeralReferenceValue,
) BoundFunctionValue {

	// If the function is already a bound function, then do not re-wrap.
	if boundFunc, isBoundFunc := function.(BoundFunctionValue); isBoundFunc {
		return boundFunc
	}

	common.UseMemory(gauge, common.BoundFunctionValueMemoryUsage)

	return BoundFunctionValue{
		Function:        function,
		SelfReference:   selfReference,
		selfIsReference: selfIsReference,
		Base:            base,
	}
}

func (BoundFunctionValue) IsValue() {}

func (f BoundFunctionValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f BoundFunctionValue) RecursiveString(seenReferences SeenReferences) string {
	return f.Function.RecursiveString(seenReferences)
}

func (f BoundFunctionValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
	return f.Function.MeteredString(context, seenReferences, locationRange)
}

func (f BoundFunctionValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitBoundFunctionValue(context, f)
}

func (f BoundFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f BoundFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return f.Function.StaticType(context)
}

func (BoundFunctionValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return false
}

func (BoundFunctionValue) IsFunctionValue() {}

func (f BoundFunctionValue) FunctionType(context ValueStaticTypeContext) *sema.FunctionType {
	return f.Function.FunctionType(context)
}

func (f BoundFunctionValue) Invoke(invocation Invocation) Value {

	invocation.Base = f.Base

	locationRange := invocation.LocationRange
	inter := invocation.InvocationContext

	// If the `self` is already a reference to begin with (e.g: attachments),
	// then pass the reference as-is to the invocation.
	// Otherwise, always dereference, at the time of the invocation.

	receiver := GetReceiver(
		f.SelfReference,
		f.selfIsReference,
		inter,
		locationRange,
	)
	invocation.Self = receiver

	return f.Function.Invoke(invocation)
}

func GetReceiver(
	receiverReference ReferenceValue,
	receiverIsReference bool,
	context ValueStaticTypeContext,
	locationRange LocationRange,
) *Value {
	var receiver *Value

	if receiverIsReference {
		var receiverValue Value = receiverReference
		receiver = &receiverValue
	} else {
		receiver = receiverReference.ReferencedValue(
			context,
			EmptyLocationRange,
			true,
		)
	}

	if _, isStorageRef := receiverReference.(*StorageReferenceValue); isStorageRef {
		// `storageRef.ReferencedValue` above already checks for the type validity, if it's not nil.
		// If nil, that means the value has been moved out of storage.
		if receiver == nil {
			panic(ReferencedValueChangedError{
				LocationRange: locationRange,
			})
		}
	} else {
		CheckInvalidatedResourceOrResourceReference(receiverReference, locationRange, context)
	}

	return receiver
}

func (f BoundFunctionValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	return f.Function.ConformsToStaticType(
		context,
		locationRange,
		results,
	)
}

func (f BoundFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (BoundFunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (BoundFunctionValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (f BoundFunctionValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return f
}

func (f BoundFunctionValue) Clone(_ ValueCloneContext) Value {
	return f
}

func (BoundFunctionValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

// NewBoundHostFunctionValue creates a bound-function value for a host-function.
func NewBoundHostFunctionValue[T Value](
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function func(self T, invocation Invocation) Value,
) BoundFunctionValue {

	wrappedFunction := func(invocation Invocation) Value {
		self, ok := (*invocation.Self).(T)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		return function(self, invocation)
	}

	hostFunc := NewStaticHostFunctionValue(context, funcType, wrappedFunction)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}

// NewUnmeteredBoundHostFunctionValue creates a bound-function value for a host-function.
func NewUnmeteredBoundHostFunctionValue(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function HostFunction,
) BoundFunctionValue {

	hostFunc := NewUnmeteredStaticHostFunctionValue(funcType, function)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}

type BoundFunctionGenerator func(MemberAccessibleValue) BoundFunctionValue
