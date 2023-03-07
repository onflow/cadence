/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package values

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/vm/config"
	"github.com/onflow/cadence/runtime/bbq/vm/types"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type Value interface {
	isValue()
	StaticType(common.MemoryGauge) types.StaticType
	Transfer(
		config *config.Config,
		address atree.Address,
		remove bool,
		storable atree.Storable,
	) Value
}

var TrueValue Value = BoolValue(true)
var FalseValue Value = BoolValue(false)

type BoolValue bool

var _ Value = BoolValue(true)

func (BoolValue) isValue() {}

func (BoolValue) StaticType(common.MemoryGauge) types.StaticType {
	return interpreter.PrimitiveStaticTypeBool
}

func (v BoolValue) Transfer(*config.Config, atree.Address, bool, atree.Storable) Value {
	return v
}

type IntValue struct {
	SmallInt int64
}

var _ Value = IntValue{}

func (IntValue) isValue() {}

func (IntValue) StaticType(common.MemoryGauge) types.StaticType {
	return interpreter.PrimitiveStaticTypeInt
}

func (v IntValue) Transfer(*config.Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v IntValue) Add(other IntValue) Value {
	return IntValue{v.SmallInt + other.SmallInt}
}

func (v IntValue) Subtract(other IntValue) Value {
	return IntValue{v.SmallInt - other.SmallInt}
}

func (v IntValue) Less(other IntValue) Value {
	if v.SmallInt < other.SmallInt {
		return TrueValue
	}
	return FalseValue
}

func (v IntValue) Greater(other IntValue) Value {
	if v.SmallInt > other.SmallInt {
		return TrueValue
	}
	return FalseValue
}

type FunctionValue struct {
	Function *bbq.Function
	Context  *Context
}

var _ Value = FunctionValue{}

func (FunctionValue) isValue() {}

func (FunctionValue) StaticType(common.MemoryGauge) types.StaticType {
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Transfer(*config.Config, atree.Address, bool, atree.Storable) Value {
	return v
}

type StringValue struct {
	String []byte
}

var _ Value = StringValue{}

func (StringValue) isValue() {}

func (StringValue) StaticType(common.MemoryGauge) types.StaticType {
	return interpreter.PrimitiveStaticTypeString
}

func (v StringValue) Transfer(*config.Config, atree.Address, bool, atree.Storable) Value {
	return v
}
