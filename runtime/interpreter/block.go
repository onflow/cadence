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

	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// Block

type BlockValue struct {
	Height    UInt64Value
	View      UInt64Value
	ID        *ArrayValue
	Timestamp UFix64Value
}

func (BlockValue) IsValue() {}

func (v BlockValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitValue(interpreter, v)
}

func (BlockValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return BlockDynamicType{}
}

func (BlockValue) StaticType() StaticType {
	return PrimitiveStaticTypeBlock
}

func (v BlockValue) Copy() Value {
	return v
}

func (BlockValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BlockValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BlockValue) IsModified() bool {
	return false
}

func (BlockValue) SetModified(_ bool) {
	// NO-OP
}

func (v BlockValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "height":
		return v.Height

	case "view":
		return v.View

	case "id":
		return v.ID

	case "timestamp":
		return v.Timestamp
	}

	return nil
}

func (v BlockValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(runtimeErrors.NewUnreachableError())
}

func (v BlockValue) IDAsByteArray() [sema.BlockIDSize]byte {
	var byteArray [sema.BlockIDSize]byte
	for i, b := range v.ID.Elements() {
		byteArray[i] = byte(b.(UInt8Value))
	}
	return byteArray
}

func (v BlockValue) String(results StringResults) string {
	return fmt.Sprintf(
		"Block(height: %s, view: %s, id: 0x%x, timestamp: %s)",
		v.Height.String(results),
		v.View.String(results),
		v.IDAsByteArray(),
		v.Timestamp.String(results),
	)
}

func (v BlockValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(BlockDynamicType)
	return ok
}

func (BlockValue) IsStorable() bool {
	return false
}
