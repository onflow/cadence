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
}

var _ Value = FunctionValue{}
var _ interpreter.FunctionValue = FunctionValue{}

func (FunctionValue) IsValue() {}

func (FunctionValue) StaticType(interpreter.ValueStaticTypeContext) bbq.StaticType {
	// TODO:
	return nil
}

func (v FunctionValue) Transfer(_ interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v FunctionValue) String() string {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Storable(storage atree.SlabStorage, address atree.Address, u uint64) (atree.Storable, error) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Accept(
	context interpreter.ValueVisitContext,
	visitor interpreter.Visitor,
	locationRange interpreter.LocationRange,
) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Walk(
	interpreter interpreter.ValueWalkContext,
	walkChild func(interpreter.Value),
	locationRange interpreter.LocationRange,
) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) ConformsToStaticType(
	context interpreter.ValueStaticTypeConformanceContext,
	locationRange interpreter.LocationRange,
	results interpreter.TypeConformanceResults,
) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) RecursiveString(seenReferences interpreter.SeenReferences) string {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	seenReferences interpreter.SeenReferences,
	locationRange interpreter.LocationRange,
) string {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) IsResourceKinded(context interpreter.ValueStaticTypeContext) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) NeedsStoreTo(address atree.Address) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) DeepRemove(removeContext interpreter.ValueRemoveContext, hasNoParentContainer bool) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Clone(context interpreter.ValueCloneContext) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) IsImportable(context interpreter.ValueImportableContext, locationRange interpreter.LocationRange) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) IsFunctionValue() {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) FunctionType() *sema.FunctionType {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

type NativeFunction func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value

type NativeFunctionValue struct {
	Name           string
	ParameterCount int
	Function       NativeFunction
}

var _ Value = NativeFunctionValue{}
var _ interpreter.FunctionValue = NativeFunctionValue{}

func (NativeFunctionValue) IsValue() {}

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
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Storable(storage atree.SlabStorage, address atree.Address, u uint64) (atree.Storable, error) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Accept(
	context interpreter.ValueVisitContext,
	visitor interpreter.Visitor,
	locationRange interpreter.LocationRange,
) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Walk(
	context interpreter.ValueWalkContext,
	walkChild func(interpreter.Value),
	locationRange interpreter.LocationRange,
) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) ConformsToStaticType(
	context interpreter.ValueStaticTypeConformanceContext,
	locationRange interpreter.LocationRange,
	results interpreter.TypeConformanceResults,
) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) RecursiveString(seenReferences interpreter.SeenReferences) string {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) MeteredString(
	context interpreter.ValueStringContext,
	seenReferences interpreter.SeenReferences,
	locationRange interpreter.LocationRange,
) string {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) IsResourceKinded(context interpreter.ValueStaticTypeContext) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) NeedsStoreTo(address atree.Address) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) DeepRemove(removeContext interpreter.ValueRemoveContext, hasNoParentContainer bool) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Clone(context interpreter.ValueCloneContext) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) IsImportable(
	context interpreter.ValueImportableContext,
	locationRange interpreter.LocationRange,
) bool {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) IsFunctionValue() {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) FunctionType() *sema.FunctionType {
	//TODO
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Invoke(invocation interpreter.Invocation) interpreter.Value {
	//TODO
	panic(errors.NewUnreachableError())
}
