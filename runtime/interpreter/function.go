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

	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// Invocation

type Invocation struct {
	Self               *CompositeValue
	Arguments          []Value
	ArgumentTypes      []sema.Type
	TypeParameterTypes *sema.TypeParameterTypeOrderedMap
	GetLocationRange   func() LocationRange
	Interpreter        *Interpreter
}

// FunctionValue

type FunctionValue interface {
	Value
	isFunctionValue()
	Invoke(Invocation) Value
}

// InterpretedFunctionValue

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

var _ Value = InterpretedFunctionValue{}

func (InterpretedFunctionValue) IsValue() {}

func (f InterpretedFunctionValue) String() string {
	return fmt.Sprintf("Function%s", f.Type.String())
}

func (f InterpretedFunctionValue) RecursiveString(_ StringResults) string {
	return f.String()
}

func (f InterpretedFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInterpretedFunctionValue(interpreter, f)
}

func (f InterpretedFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var functionDynamicType DynamicType = FunctionDynamicType{}

func (InterpretedFunctionValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return functionDynamicType
}

func (f InterpretedFunctionValue) StaticType() StaticType {
	// TODO: add function static type, convert f.Type
	return nil
}

func (InterpretedFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (InterpretedFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (InterpretedFunctionValue) isFunctionValue() {}

func (f InterpretedFunctionValue) Invoke(invocation Invocation) Value {

	// Check arguments' dynamic types match parameter types

	for i, argument := range invocation.Arguments {
		parameterType := f.Type.Parameters[i].TypeAnnotation.Type

		if !f.Interpreter.checkValueTransferTargetType(argument, parameterType) {
			panic(InvocationArgumentTypeError{
				Index:         i,
				ParameterType: parameterType,
				LocationRange: invocation.GetLocationRange(),
			})
		}
	}

	return f.Interpreter.invokeInterpretedFunction(f, invocation)
}

func (f InterpretedFunctionValue) ConformsToDynamicType(_ *Interpreter, _ DynamicType, _ TypeConformanceResults) bool {
	// TODO: once FunctionDynamicType has parameter and return type info,
	//   check it matches InterpretedFunctionValue's static function type
	return false
}

func (InterpretedFunctionValue) IsStorable() bool {
	return false
}

func (f InterpretedFunctionValue) Storable(_ atree.SlabStorage) atree.Storable {
	return atree.NonStorable{Value: f}
}

func (f InterpretedFunctionValue) DeepCopy(_ atree.SlabStorage, _ atree.Address) (atree.Value, error) {
	return f, nil
}

// HostFunctionValue

type HostFunction func(invocation Invocation) Value

type HostFunctionValue struct {
	Function        HostFunction
	NestedVariables *StringVariableOrderedMap
}

func (f HostFunctionValue) String() string {
	// TODO: include type
	return "Function(...)"
}

func (f HostFunctionValue) RecursiveString(_ StringResults) string {
	return f.String()
}

func NewHostFunctionValue(
	function HostFunction,
) HostFunctionValue {
	return HostFunctionValue{
		Function: function,
	}
}

var _ Value = HostFunctionValue{}

func (HostFunctionValue) IsValue() {}

func (f HostFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitHostFunctionValue(interpreter, f)
}

func (f HostFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var hostFunctionDynamicType DynamicType = FunctionDynamicType{}

func (HostFunctionValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return hostFunctionDynamicType
}

func (HostFunctionValue) StaticType() StaticType {
	// TODO: add function static type, store static type in host function value
	return nil
}

func (HostFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (HostFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (HostFunctionValue) isFunctionValue() {}

func (f HostFunctionValue) Invoke(invocation Invocation) Value {
	return f.Function(invocation)
}

func (f HostFunctionValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	if f.NestedVariables != nil {
		if variable, ok := f.NestedVariables.Get(name); ok {
			return variable.GetValue()
		}
	}
	return nil
}

func (HostFunctionValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (f HostFunctionValue) ConformsToDynamicType(_ *Interpreter, _ DynamicType, _ TypeConformanceResults) bool {
	// TODO: once HostFunctionValue has static function type,
	//   and FunctionDynamicType has parameter and return type info,
	//   check they match

	return false
}

func (HostFunctionValue) IsStorable() bool {
	return false
}

func (f HostFunctionValue) Storable(_ atree.SlabStorage) atree.Storable {
	return atree.NonStorable{Value: f}
}

func (f HostFunctionValue) DeepCopy(_ atree.SlabStorage, _ atree.Address) (atree.Value, error) {
	return f, nil
}

// BoundFunctionValue

type BoundFunctionValue struct {
	Function FunctionValue
	Self     *CompositeValue
}

var _ Value = BoundFunctionValue{}

func (BoundFunctionValue) IsValue() {}

func (f BoundFunctionValue) String() string {
	return f.RecursiveString(StringResults{})
}

func (f BoundFunctionValue) RecursiveString(results StringResults) string {
	return f.Function.RecursiveString(results)
}

func (f BoundFunctionValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoundFunctionValue(interpreter, f)
}

func (f BoundFunctionValue) Walk(_ func(Value)) {
	// NO-OP
}

var boundFunctionDynamicType DynamicType = FunctionDynamicType{}

func (BoundFunctionValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return boundFunctionDynamicType
}

func (f BoundFunctionValue) StaticType() StaticType {
	return f.Function.StaticType()
}

func (BoundFunctionValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BoundFunctionValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BoundFunctionValue) isFunctionValue() {}

func (f BoundFunctionValue) Invoke(invocation Invocation) Value {
	invocation.Self = f.Self
	return f.Function.Invoke(invocation)
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

func (f BoundFunctionValue) Storable(_ atree.SlabStorage) atree.Storable {
	return atree.NonStorable{Value: f}
}

func (f BoundFunctionValue) DeepCopy(_ atree.SlabStorage, _ atree.Address) (atree.Value, error) {
	return f, nil
}
