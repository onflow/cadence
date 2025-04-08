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
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// BoolValue

type BoolValue values.BoolValue

var _ Value = BoolValue(false)
var _ EquatableValue = BoolValue(false)
var _ HashableValue = BoolValue(false)

const TrueValue = BoolValue(true)
const FalseValue = BoolValue(false)

func (BoolValue) IsValue() {}

func (v BoolValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitBoolValue(interpreter, v)
}

func (BoolValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (BoolValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeBool)
}

func (BoolValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return sema.BoolType.Importable
}

func (v BoolValue) Negate(_ *Interpreter) BoolValue {
	return BoolValue(values.BoolValue(v).Negate())
}

func (v BoolValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return values.BoolValue(v).
		Equal(values.BoolValue(otherBool))
}

func (v BoolValue) Less(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return BoolValue(
		values.BoolValue(v).
			Less(values.BoolValue(o)),
	)
}

func (v BoolValue) LessEqual(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return BoolValue(
		values.BoolValue(v).
			LessEqual(values.BoolValue(o)),
	)
}

func (v BoolValue) Greater(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return BoolValue(
		values.BoolValue(v).
			Greater(values.BoolValue(o)),
	)
}

func (v BoolValue) GreaterEqual(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return BoolValue(
		values.BoolValue(v).
			GreaterEqual(values.BoolValue(o)),
	)
}

// HashInput returns a byte slice containing:
// - HashInputTypeBool (1 byte)
// - 1/0 (1 byte)
func (v BoolValue) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeBool)
	if v {
		scratch[1] = 1
	} else {
		scratch[1] = 0
	}
	return scratch[:2]
}

func (v BoolValue) String() string {
	return values.BoolValue(v).String()
}

func (v BoolValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v BoolValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	if v {
		common.UseMemory(context, common.TrueStringMemoryUsage)
	} else {
		common.UseMemory(context, common.FalseStringMemoryUsage)
	}

	return v.String()
}

func (v BoolValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v BoolValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return values.BoolValue(v), nil
}

func (BoolValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (BoolValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v BoolValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v BoolValue) Clone(_ ValueCloneContext) Value {
	return v
}

func (BoolValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}
