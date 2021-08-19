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

var _ Value = BlockValue{}

func (BlockValue) IsValue() {}

func (v BlockValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitValue(interpreter, v)
}

func (v BlockValue) Walk(walkChild func(Value)) {
	walkChild(v.Height)
	walkChild(v.View)
	walkChild(v.ID)
	walkChild(v.Timestamp)
}

var blockDynamicType DynamicType = BlockDynamicType{}

func (BlockValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return blockDynamicType
}

func (BlockValue) StaticType() StaticType {
	return PrimitiveStaticTypeBlock
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
	i := 0
	v.ID.Walk(func(b Value) {
		byteArray[i] = byte(b.(UInt8Value))
		i++
	})
	return byteArray
}

func (v BlockValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v BlockValue) RecursiveString(seenReferences SeenReferences) string {
	return fmt.Sprintf(
		"Block(height: %s, view: %s, id: 0x%x, timestamp: %s)",
		v.Height.RecursiveString(seenReferences),
		v.View.RecursiveString(seenReferences),
		v.IDAsByteArray(),
		v.Timestamp.RecursiveString(seenReferences),
	)
}

func (v BlockValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(BlockDynamicType)
	return ok
}

func (v BlockValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (v BlockValue) DeepCopy(_ atree.SlabStorage, _ atree.Address) (atree.Value, error) {
	return v, nil
}

func (BlockValue) DeepRemove(_ atree.SlabStorage) error {
	// NO-OP
	return nil
}
