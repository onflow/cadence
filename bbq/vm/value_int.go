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
	"strconv"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type IntValue struct {
	SmallInt int64
}

func NewIntValue(smallInt int64) IntValue {
	return IntValue{
		SmallInt: smallInt,
	}
}

func (v IntValue) String() string {
	return strconv.FormatInt(v.SmallInt, 10)
}

var _ Value = IntValue{}
var _ EquatableValue = IntValue{}
var _ ComparableValue = IntValue{}
var _ NumberValue = IntValue{}

func (IntValue) isValue() {}

func (IntValue) StaticType(*Config) StaticType {
	return interpreter.PrimitiveStaticTypeInt
}

func (v IntValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v IntValue) Add(other NumberValue) NumberValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	sum := safeAdd(int(v.SmallInt), int(otherInt.SmallInt))
	return NewIntValue(int64(sum))
}

func (v IntValue) Subtract(other NumberValue) NumberValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return NewIntValue(v.SmallInt - otherInt.SmallInt)
}

func (v IntValue) Multiply(other NumberValue) NumberValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return NewIntValue(v.SmallInt * otherInt.SmallInt)
}

func (v IntValue) Divide(other NumberValue) NumberValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return NewIntValue(v.SmallInt / otherInt.SmallInt)
}

func (v IntValue) Mod(other NumberValue) NumberValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return NewIntValue(v.SmallInt * otherInt.SmallInt)
}

func (v IntValue) Equal(other Value) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}
	return v.SmallInt == otherInt.SmallInt
}

func (v IntValue) Less(other ComparableValue) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return v.SmallInt < otherInt.SmallInt
}

func (v IntValue) LessEqual(other ComparableValue) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return v.SmallInt <= otherInt.SmallInt
}

func (v IntValue) Greater(other ComparableValue) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return v.SmallInt > otherInt.SmallInt
}

func (v IntValue) GreaterEqual(other ComparableValue) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		panic("invalid operand")
	}
	return v.SmallInt >= otherInt.SmallInt
}

// members

func init() {
	typeName := interpreter.PrimitiveStaticTypeInt.String()

	RegisterTypeBoundFunction(typeName, sema.ToStringFunctionName, NativeFunctionValue{
		ParameterCount: len(sema.ToStringFunctionType.Parameters),
		Function: func(config *Config, typeArguments []StaticType, value ...Value) Value {
			number := value[0].(IntValue)
			return NewStringValue(number.String())
		},
	})
}
