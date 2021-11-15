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

	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountContractsValue

var authAccountContractsTypeID = sema.AuthAccountContractsType.ID()
var authAccountContractsStaticType StaticType = PrimitiveStaticTypeAuthAccountContracts
var authAccountContractsDynamicType DynamicType = CompositeDynamicType{
	StaticType: sema.AuthAccountContractsType,
}
var authAccountContractsFieldNames []string = nil

func NewAuthAccountContractsValue(
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
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("AuthAccount.Contracts(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		authAccountContractsTypeID,
		authAccountContractsStaticType,
		authAccountContractsDynamicType,
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
var publicAccountContractsDynamicType DynamicType = CompositeDynamicType{
	StaticType: sema.PublicAccountContractsType,
}

func NewPublicAccountContractsValue(
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
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("PublicAccount.Contracts(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		publicAccountContractsTypeID,
		publicAccountContractsStaticType,
		publicAccountContractsDynamicType,
		nil,
		fields,
		computedFields,
		nil,
		stringer,
	)
}
