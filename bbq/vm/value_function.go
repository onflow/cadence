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

type FunctionValue struct {
	Function   *bbq.Function[opcode.Instruction]
	Executable *ExecutableProgram
	Upvalues   []*Upvalue
	Type       interpreter.FunctionStaticType
}

var _ Value = FunctionValue{}
var _ interpreter.FunctionValue = FunctionValue{}

func (FunctionValue) IsValue() {}

func (v FunctionValue) IsFunctionValue() {}

func (v FunctionValue) StaticType(interpreter.ValueStaticTypeContext) bbq.StaticType {
	return v.Type
}

func (v FunctionValue) Transfer(
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

func (v FunctionValue) String() string {
	return v.Type.String()
}

func (v FunctionValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return interpreter.NonStorable{Value: v}, nil
}

func (v FunctionValue) Accept(
	_ interpreter.ValueVisitContext,
	_ interpreter.Visitor,
	_ interpreter.LocationRange,
) {
	// Unused for now
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Walk(
	_ interpreter.ValueWalkContext,
	_ func(interpreter.Value),
	_ interpreter.LocationRange,
) {
	// NO-OP
}

func (v FunctionValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.LocationRange,
	_ interpreter.TypeConformanceResults,
) bool {
	return true
}

func (v FunctionValue) RecursiveString(_ interpreter.SeenReferences) string {
	return v.String()
}

func (v FunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
	_ interpreter.LocationRange,
) string {
	return v.Type.MeteredString(context)
}

func (v FunctionValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v FunctionValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (v FunctionValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v FunctionValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	return v
}

func (v FunctionValue) IsImportable(_ interpreter.ValueImportableContext, _ interpreter.LocationRange) bool {
	return false
}

func (v FunctionValue) FunctionType() *sema.FunctionType {
	return v.Type.Type
}

func (v FunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	return invocation.InvocationContext.InvokeFunction(
		v,
		invocation.Arguments,
		invocation.ArgumentTypes,
		invocation.LocationRange,
	)
}

type NativeFunction func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value

type NativeFunctionValue struct {
	Name           string
	ParameterCount int
	Function       NativeFunction
	Type           interpreter.FunctionStaticType
}

var _ Value = NativeFunctionValue{}
var _ interpreter.FunctionValue = NativeFunctionValue{}

func (NativeFunctionValue) IsValue() {}

func (v NativeFunctionValue) IsFunctionValue() {}

func (NativeFunctionValue) StaticType(interpreter.ValueStaticTypeContext) bbq.StaticType {
	panic(errors.NewUnreachableError())
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
