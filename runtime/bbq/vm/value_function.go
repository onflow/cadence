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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type FunctionValue struct {
	Function *bbq.Function
	Context  *Context
}

var _ Value = FunctionValue{}

func (FunctionValue) isValue() {}

func (FunctionValue) StaticType(common.MemoryGauge) StaticType {
	panic(errors.NewUnreachableError())
}

func (v FunctionValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v FunctionValue) String() string {
	//TODO implement me
	panic("implement me")
}

type NativeFunctionValue struct {
	ParameterCount int
	Function       func(arguments ...Value) Value
}

var _ Value = NativeFunctionValue{}

func (NativeFunctionValue) isValue() {}

func (NativeFunctionValue) StaticType(common.MemoryGauge) StaticType {
	panic(errors.NewUnreachableError())
}

func (v NativeFunctionValue) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v NativeFunctionValue) String() string {
	//TODO implement me
	panic("implement me")
}
