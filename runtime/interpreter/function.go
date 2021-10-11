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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
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

func (f *InterpretedFunctionValue) String() string {
	return fmt.Sprintf("Function%s", f.Type.String())
}

func (f *InterpretedFunctionValue) RecursiveString(_ SeenReferences) string {
	return f.String()
}

func (*InterpretedFunctionValue) IsValue() {}

func (f *InterpretedFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInterpretedFunctionValue(interpreter, f)
}

func (f *InterpretedFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

func (f *InterpretedFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return FunctionDynamicType{
		FuncType: f.Type,
	}
}

func (f *InterpretedFunctionValue) StaticType() StaticType {
	return ConvertSemaToStaticType(f.Type)
}

func (f *InterpretedFunctionValue) Copy() Value {
	return f
}

func (*InterpretedFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (*InterpretedFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (*InterpretedFunctionValue) IsModified() bool {
	return false
}

func (*InterpretedFunctionValue) SetModified(_ bool) {
	// NO-OP
}

func (*InterpretedFunctionValue) isFunctionValue() {}

func (f *InterpretedFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Interpreter.invokeInterpretedFunction(f, invocation)
}

func (f *InterpretedFunctionValue) ConformsToDynamicType(
	_ *Interpreter,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	targetType, ok := dynamicType.(FunctionDynamicType)
	if !ok {
		return false
	}

	return f.Type.Equal(targetType.FuncType)
}

func (*InterpretedFunctionValue) IsStorable() bool {
	return false
}

// HostFunctionValue
//
type HostFunction func(invocation Invocation) Value

type HostFunctionValue struct {
	Function        HostFunction
	NestedVariables *StringVariableOrderedMap
	Type            *sema.FunctionType
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
	funcType *sema.FunctionType,
) *HostFunctionValue {
	return &HostFunctionValue{
		Function: function,
		Type:     funcType,
	}
}

func (*HostFunctionValue) IsValue() {}

func (f *HostFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitHostFunctionValue(interpreter, f)
}

func (f *HostFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

func (f *HostFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return FunctionDynamicType{
		FuncType: f.Type,
	}
}

func (f *HostFunctionValue) StaticType() StaticType {
	return ConvertSemaToStaticType(f.Type)
}

func (f *HostFunctionValue) Copy() Value {
	return f
}

func (*HostFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (*HostFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (*HostFunctionValue) IsModified() bool {
	return false
}

func (*HostFunctionValue) SetModified(_ bool) {
	// NO-OP
}

func (*HostFunctionValue) isFunctionValue() {}

func (f *HostFunctionValue) invoke(invocation Invocation) Value {

	// The check that arguments' dynamic types match the parameter types
	// was already performed by the interpreter's checkValueTransferTargetType function

	return f.Function(invocation)
}

func (f *HostFunctionValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	if f.NestedVariables != nil {
		if variable, ok := f.NestedVariables.Get(name); ok {
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
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	targetType, ok := dynamicType.(FunctionDynamicType)
	if !ok {
		return false
	}

	return f.Type.Equal(targetType.FuncType)
}

func (*HostFunctionValue) IsStorable() bool {
	return false
}

// BoundFunctionValue
//
type BoundFunctionValue struct {
	Function FunctionValue
	Self     *CompositeValue
}

func (f BoundFunctionValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f BoundFunctionValue) RecursiveString(seenReferences SeenReferences) string {
	return f.Function.RecursiveString(seenReferences)
}

func (BoundFunctionValue) IsValue() {}

func (f BoundFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoundFunctionValue(interpreter, f)
}

func (f BoundFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

func (f BoundFunctionValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	funcStaticType, ok := f.Function.StaticType().(FunctionStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return FunctionDynamicType{
		FuncType: funcStaticType.Type,
	}
}

func (f BoundFunctionValue) StaticType() StaticType {
	return f.Function.StaticType()
}

func (f BoundFunctionValue) Copy() Value {
	return f
}

func (BoundFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BoundFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BoundFunctionValue) IsModified() bool {
	return false
}

func (BoundFunctionValue) SetModified(_ bool) {
	// NO-OP
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
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {
	return f.Function.ConformsToDynamicType(interpreter, dynamicType, results)
}

func (BoundFunctionValue) IsStorable() bool {
	return false
}
