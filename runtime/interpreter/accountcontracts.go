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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountContractsValue

var authAccountContractsTypeID = sema.AuthAccountContractsType.ID()
var authAccountContractsStaticType StaticType = PrimitiveStaticTypeAuthAccountContracts // unmetered
var authAccountContractsFieldNames []string = nil

func NewAuthAccountContractsValue(
	inter *Interpreter,
	address AddressValue,
	addFunction FunctionValue,
	updateFunction FunctionValue,
	getFunction FunctionValue,
	removeFunction FunctionValue,
	namesGetter func(interpreter *Interpreter) *ArrayValue,
) Value {

	fields := map[string]Value{
		sema.AuthAccountContractsTypeAddFunctionName:                addFunction,
		sema.AuthAccountContractsTypeGetFunctionName:                getFunction,
		sema.AuthAccountContractsTypeRemoveFunctionName:             removeFunction,
		sema.AuthAccountContractsTypeUpdateExperimentalFunctionName: updateFunction,
	}

	computedFields := map[string]ComputedField{
		sema.AuthAccountContractsTypeNamesField: func(interpreter *Interpreter, _ func() LocationRange) Value {
			return namesGetter(interpreter)
		},
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(23))
			addressStr := address.ToMeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("AuthAccount.Contracts(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		inter,
		authAccountContractsTypeID,
		authAccountContractsStaticType,
		authAccountContractsFieldNames,
		fields,
		computedFields,
		nil,
		stringer,
	)
}

// PublicAccountContractsValue

var publicAccountContractsTypeID = sema.PublicAccountContractsType.ID()
var publicAccountContractsStaticType StaticType = PrimitiveStaticTypePublicAccountContracts

func NewPublicAccountContractsValue(
	inter *Interpreter,
	address AddressValue,
	getFunction FunctionValue,
	namesGetter func(interpreter *Interpreter) *ArrayValue,
) Value {

	fields := map[string]Value{
		sema.PublicAccountContractsTypeGetFunctionName: getFunction,
	}

	computedFields := map[string]ComputedField{
		sema.PublicAccountContractsTypeNamesField: func(interpreter *Interpreter, _ func() LocationRange) Value {
			return namesGetter(interpreter)
		},
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(25))
			addressStr := address.ToMeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("PublicAccount.Contracts(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		inter,
		publicAccountContractsTypeID,
		publicAccountContractsStaticType,
		nil,
		fields,
		computedFields,
		nil,
		stringer,
	)
}
