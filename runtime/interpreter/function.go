/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"fmt"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// Invocation
//
type Invocation struct {
	Self               *CompositeValue
	ReceiverType       sema.Type
	Arguments          []Value
	ArgumentTypes      []sema.Type
	TypeParameterTypes *sema.TypeParameterTypeOrderedMap
	GetLocationRange   func() LocationRange
	Interpreter        *Interpreter
}

// FunctionValue
//
type FunctionValue interface {
	Value
	isFunctionValue()
	// invoke evaluates the function.
	// Only used internally by the interpreter.
	// Use Interpreter.InvokeFunctionValue if you want to invoke the function externally
	invoke(Invocation) Value
}

// InterpretedFunctionValue
//
type InterpretedFunctionValue struct {
	Interpreter      *Interpreter
	ParameterList    *ast.ParameterList
	Type             *sema.FunctionType
	Activation       *VariableActivation
	BeforeStatements []ast.Statement
	PreConditions    ast.Conditions
	Statements       []ast.Statement
	PostConditions   ast.Conditions
}

var _ Value = &InterpretedFunctionValue{}

func (*InterpretedFunctionValue) IsValue() {}

func (f *InterpretedFunctionValue) String() string {
	return fmt.Sprintf("Function%s", f.Type.String())
}

func (f *InterpretedFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (f *InterpretedFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInterpretedFunctionValue(interpreter, f)
}

func (f *InterpretedFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var functionDynamicType DynamicType = FunctionDynamicType{}

func (*InterpretedFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return functionDynamicType
}

func (f *InterpretedFunctionValue) StaticType() StaticType {
	return ConvertSemaToStaticType(f.Type)
}

func (*InterpretedFunctionValue) isFunctionValue() {}

func (f *InterpretedFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Interpreter.invokeInterpretedFunction(f, invocation)
}

func (f *InterpretedFunctionValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	_ DynamicType,
	_ TypeConformanceResults,
) bool {
	// TODO: once FunctionDynamicType has parameter and return type info,
	//   check it matches InterpretedFunctionValue's static function type
	return false
}

func (f *InterpretedFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*InterpretedFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (*InterpretedFunctionValue) NeedsStoreToAddress(_ *Interpreter, _ atree.Address) bool {
	return false
}

func (f *InterpretedFunctionValue) DeepCopy(_ *Interpreter, _ func() LocationRange, _ atree.Address) Value {
	return f
}

func (*InterpretedFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

// HostFunctionValue
//
type HostFunction func(invocation Invocation) Value

type HostFunctionValue struct {
	Function        HostFunction
	NestedVariables map[string]*Variable
}

func (f *HostFunctionValue) String() string {
	// TODO: include type
	return "Function(...)"
}

func (f *HostFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func NewHostFunctionValue(
	function HostFunction,
) *HostFunctionValue {
	return &HostFunctionValue{
		Function: function,
	}
}

var _ Value = &HostFunctionValue{}

func (*HostFunctionValue) IsValue() {}

func (f *HostFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitHostFunctionValue(interpreter, f)
}

func (f *HostFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var hostFunctionDynamicType DynamicType = FunctionDynamicType{}

func (*HostFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return hostFunctionDynamicType
}

func (*HostFunctionValue) StaticType() StaticType {
	// TODO: add function static type, store static type in host function value
	return nil
}

func (*HostFunctionValue) isFunctionValue() {}

func (f *HostFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Function(invocation)
}

func (f *HostFunctionValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	if f.NestedVariables != nil {
		if variable, ok := f.NestedVariables[name]; ok {
			return variable.GetValue()
		}
	}
	return nil
}

func (*HostFunctionValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (f *HostFunctionValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	_ DynamicType,
	_ TypeConformanceResults,
) bool {
	// TODO: once HostFunctionValue has static function type,
	//   and FunctionDynamicType has parameter and return type info,
	//   check they match

	return false
}

func (f *HostFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (*HostFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (*HostFunctionValue) NeedsStoreToAddress(_ *Interpreter, _ atree.Address) bool {
	return false
}

func (f *HostFunctionValue) DeepCopy(_ *Interpreter, _ func() LocationRange, _ atree.Address) Value {
	return f
}

func (*HostFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

// BoundFunctionValue
//
type BoundFunctionValue struct {
	Function FunctionValue
	Self     *CompositeValue
}

var _ Value = BoundFunctionValue{}

func (BoundFunctionValue) IsValue() {}

func (f BoundFunctionValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f BoundFunctionValue) RecursiveString(seenReferences SeenReferences) string {
	return f.Function.RecursiveString(seenReferences)
}

func (f BoundFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoundFunctionValue(interpreter, f)
}

func (f BoundFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var boundFunctionDynamicType DynamicType = FunctionDynamicType{}

func (BoundFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return boundFunctionDynamicType
}

func (f BoundFunctionValue) StaticType() StaticType {
	return f.Function.StaticType()
}

func (BoundFunctionValue) isFunctionValue() {}

func (f BoundFunctionValue) invoke(invocation Invocation) Value {
	self := f.Self
	receiverType := invocation.ReceiverType

	if receiverType != nil {
		selfType := invocation.Interpreter.ConvertStaticToSemaType(self.StaticType())

		if _, ok := receiverType.(*sema.ReferenceType); ok {
			if _, ok := selfType.(*sema.ReferenceType); !ok {
				selfType = &sema.ReferenceType{
					Type: selfType,
				}
			}
		}

		if !sema.IsSubType(selfType, receiverType) {
			panic(InvocationReceiverTypeError{
				SelfType:      selfType,
				ReceiverType:  receiverType,
				LocationRange: invocation.GetLocationRange(),
			})
		}
	}

	invocation.Self = self
	return f.Function.invoke(invocation)
}

func (f BoundFunctionValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {
	return f.Function.ConformsToDynamicType(
		interpreter,
		getLocationRange,
		dynamicType,
		results,
	)
}

func (f BoundFunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (BoundFunctionValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (BoundFunctionValue) NeedsStoreToAddress(_ *Interpreter, _ atree.Address) bool {
	return false
}

func (f BoundFunctionValue) DeepCopy(_ *Interpreter, _ func() LocationRange, _ atree.Address) Value {
	return f
}

func (BoundFunctionValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}
