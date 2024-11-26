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

// BoolValue

type BoolValue bool

var _ Value = BoolValue(false)
var _ atree.Storable = BoolValue(false)
var _ EquatableValue = BoolValue(false)
var _ HashableValue = BoolValue(false)

const TrueValue = BoolValue(true)
const FalseValue = BoolValue(false)

func AsBoolValue(v bool) BoolValue {
	if v {
		return TrueValue
	}
	return FalseValue
}

func (BoolValue) isValue() {}

func (v BoolValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitBoolValue(interpreter, v)
}

func (BoolValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (BoolValue) StaticType(staticTypeGetter StaticTypeGetter) StaticType {
	return NewPrimitiveStaticType(staticTypeGetter, PrimitiveStaticTypeBool)
}

func (BoolValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return sema.BoolType.Importable
}

func (v BoolValue) Negate(_ *Interpreter) BoolValue {
	if v == TrueValue {
		return FalseValue
	}
	return TrueValue
}

func (v BoolValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return bool(v) == bool(otherBool)
}

func (v BoolValue) Less(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return !v && o
}

func (v BoolValue) LessEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return !v || o
}

func (v BoolValue) Greater(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return v && !o
}

func (v BoolValue) GreaterEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return v || !o
}

// HashInput returns a byte slice containing:
// - HashInputTypeBool (1 byte)
// - 1/0 (1 byte)
func (v BoolValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeBool)
	if v {
		scratch[1] = 1
	} else {
		scratch[1] = 0
	}
	return scratch[:2]
}

func (v BoolValue) String() string {
	return format.Bool(bool(v))
}

func (v BoolValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v BoolValue) MeteredString(interpreter *Interpreter, _ SeenReferences, locationRange LocationRange) string {
	if v {
		common.UseMemory(interpreter, common.TrueStringMemoryUsage)
	} else {
		common.UseMemory(interpreter, common.FalseStringMemoryUsage)
	}

	return v.String()
}

func (v BoolValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v BoolValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (BoolValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (BoolValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v BoolValue) Transfer(
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

func (v BoolValue) Clone(_ *Interpreter) Value {
	return v
}

func (BoolValue) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v BoolValue) ByteSize() uint32 {
	return 1
}

func (v BoolValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (BoolValue) ChildStorables() []atree.Storable {
	return nil
}
