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
	FunctionType() *sema.FunctionType
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

func (f *InterpretedFunctionValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitInterpretedFunctionValue(interpreter, f)
}

func (f *InterpretedFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f *InterpretedFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return ConvertSemaToStaticType(context, f.Type)
}

func (*InterpretedFunctionValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (*InterpretedFunctionValue) IsFunctionValue() {}

func (f *InterpretedFunctionValue) FunctionType() *sema.FunctionType {
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

func (*InterpretedFunctionValue) IsResourceKinded(context ValueStaticTypeContext) bool {
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

func (f *InterpretedFunctionValue) Clone(_ *Interpreter) Value {
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

func (f *HostFunctionValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitHostFunctionValue(interpreter, f)
}

func (f *HostFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f *HostFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return ConvertSemaToStaticType(context, f.Type)
}

func (*HostFunctionValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (*HostFunctionValue) IsFunctionValue() {}

func (f *HostFunctionValue) FunctionType() *sema.FunctionType {
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

func (*HostFunctionValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Host functions have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*HostFunctionValue) SetMember(_ MemberAccessibleContext, _ LocationRange, _ string, _ Value) bool {
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

func (*HostFunctionValue) IsResourceKinded(context ValueStaticTypeContext) bool {
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

func (f *HostFunctionValue) Clone(_ *Interpreter) Value {
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
	Function           FunctionValue
	Base               *EphemeralReferenceValue
	BoundAuthorization Authorization
	SelfReference      ReferenceValue
	selfIsReference    bool
}

var _ Value = BoundFunctionValue{}
var _ FunctionValue = BoundFunctionValue{}

func NewBoundFunctionValue(
	context FunctionCreationContext,
	function FunctionValue,
	self *Value,
	base *EphemeralReferenceValue,
	boundAuth Authorization,
) BoundFunctionValue {

	// Since 'self' work as an implicit reference, create an explicit one and hold it.
	// This reference is later used to check the validity of the referenced value/resource.
	// For attachments, 'self' is already a reference. So no need to create a reference again.

	selfRef, selfIsRef := (*self).(ReferenceValue)
	if !selfIsRef {
		semaType := MustSemaTypeOfValue(*self, context)
		selfRef = NewEphemeralReferenceValue(context, boundAuth, *self, semaType, EmptyLocationRange)
	}

	return NewBoundFunctionValueFromSelfReference(
		context,
		function,
		selfRef,
		selfIsRef,
		base,
		boundAuth,
	)
}

func NewBoundFunctionValueFromSelfReference(
	gauge common.MemoryGauge,
	function FunctionValue,
	selfReference ReferenceValue,
	selfIsReference bool,
	base *EphemeralReferenceValue,
	boundAuth Authorization,
) BoundFunctionValue {

	// If the function is already a bound function, then do not re-wrap.
	if boundFunc, isBoundFunc := function.(BoundFunctionValue); isBoundFunc {
		return boundFunc
	}

	common.UseMemory(gauge, common.BoundFunctionValueMemoryUsage)

	return BoundFunctionValue{
		Function:           function,
		SelfReference:      selfReference,
		selfIsReference:    selfIsReference,
		Base:               base,
		BoundAuthorization: boundAuth,
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

func (f BoundFunctionValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitBoundFunctionValue(interpreter, f)
}

func (f BoundFunctionValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f BoundFunctionValue) StaticType(context ValueStaticTypeContext) StaticType {
	return f.Function.StaticType(context)
}

func (BoundFunctionValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (BoundFunctionValue) IsFunctionValue() {}

func (f BoundFunctionValue) FunctionType() *sema.FunctionType {
	return f.Function.FunctionType()
}

func (f BoundFunctionValue) Invoke(invocation Invocation) Value {

	invocation.Base = f.Base
	invocation.BoundAuthorization = f.BoundAuthorization

	locationRange := invocation.LocationRange
	inter := invocation.InvocationContext

	// If the `self` is already a reference to begin with (e.g: attachments),
	// then pass the reference as-is to the invocation.
	// Otherwise, always dereference, at the time of the invocation.

	if f.selfIsReference {
		var self Value = f.SelfReference
		invocation.Self = &self
	} else {
		invocation.Self = f.SelfReference.ReferencedValue(
			inter,
			EmptyLocationRange,
			true,
		)
	}

	if _, isStorageRef := f.SelfReference.(*StorageReferenceValue); isStorageRef {
		// `storageRef.ReferencedValue` above already checks for the type validity, if it's not nil.
		// If nil, that means the value has been moved out of storage.
		if invocation.Self == nil {
			panic(ReferencedValueChangedError{
				LocationRange: locationRange,
			})
		}
	} else {
		checkInvalidatedResourceOrResourceReference(f.SelfReference, locationRange, inter)
	}

	return f.Function.Invoke(invocation)
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

func (BoundFunctionValue) IsResourceKinded(context ValueStaticTypeContext) bool {
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

func (f BoundFunctionValue) Clone(_ *Interpreter) Value {
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
		nil,
	)
}

type BoundFunctionGenerator func(MemberAccessibleValue) BoundFunctionValue
