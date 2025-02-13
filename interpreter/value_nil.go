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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// NilValue

type NilValue struct{}

var Nil Value = NilValue{}
var NilOptionalValue OptionalValue = NilValue{}
var NilStorable atree.Storable = NilValue{}

var _ Value = NilValue{}
var _ atree.Storable = NilValue{}
var _ EquatableValue = NilValue{}
var _ MemberAccessibleValue = NilValue{}
var _ OptionalValue = NilValue{}

func (NilValue) isValue() {}

func (v NilValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitNilValue(interpreter, v)
}

func (NilValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (NilValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewOptionalStaticType(
		context,
		NewPrimitiveStaticType(context, PrimitiveStaticTypeNever),
	)
}

func (NilValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (NilValue) isOptionalValue() {}

func (NilValue) forEach(_ func(Value)) {}

func (v NilValue) fmap(_ *Interpreter, _ func(Value) Value) OptionalValue {
	return v
}

func (NilValue) IsDestroyed() bool {
	return false
}

func (v NilValue) Destroy(_ *Interpreter, _ LocationRange) {}

func (NilValue) String() string {
	return format.Nil
}

func (v NilValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v NilValue) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(interpreter, common.NilValueStringMemoryUsage)
	return v.String()
}

// nilValueMapFunction is created only once per interpreter.
// Hence, no need to meter, as it's a constant.
var nilValueMapFunction = NewUnmeteredStaticHostFunctionValue(
	sema.OptionalTypeMapFunctionType(sema.NeverType),
	func(invocation Invocation) Value {
		return Nil
	},
)

func (v NilValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.OptionalTypeMapFunctionName:
		return nilValueMapFunction
	}

	return nil
}

func (NilValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Nil has no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (NilValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Nil has no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v NilValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v NilValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	_, ok := other.(NilValue)
	return ok
}

func (NilValue) IsStorable() bool {
	return true
}

func (v NilValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (NilValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (NilValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v NilValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v NilValue) Clone(_ *Interpreter) Value {
	return v
}

func (NilValue) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v NilValue) ByteSize() uint32 {
	return 1
}

func (v NilValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (NilValue) ChildStorables() []atree.Storable {
	return nil
}

func (NilValue) isInvalidatedResource(context ValueStaticTypeContext) bool {
	return false
}
