/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/bbq"
)

type Value interface {
	isValue()
}

var trueValue = BoolValue(true)
var falseValue = BoolValue(false)

type BoolValue bool

var _ Value = BoolValue(true)

func (BoolValue) isValue() {}

type IntValue struct {
	smallInt int64
}

var _ Value = IntValue{}

func (IntValue) isValue() {}

func (v IntValue) Add(other IntValue) IntValue {
	return IntValue{v.smallInt + other.smallInt}
}

func (v IntValue) Subtract(other IntValue) IntValue {
	return IntValue{v.smallInt - other.smallInt}
}

func (v IntValue) Less(other IntValue) BoolValue {
	if v.smallInt < other.smallInt {
		return trueValue
	}
	return falseValue
}

func (v IntValue) Greater(other IntValue) BoolValue {
	if v.smallInt > other.smallInt {
		return trueValue
	}
	return falseValue
}

type FunctionValue struct {
	Function *bbq.Function
}

var _ Value = FunctionValue{}

func (FunctionValue) isValue() {}
