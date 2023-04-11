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

type ContractNamesGetter func(interpreter *Interpreter, locationRange LocationRange) *ArrayValue

func NewAuthAccountContractsValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction FunctionValue,
	updateFunction FunctionValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
	removeFunction FunctionValue,
	namesGetter ContractNamesGetter,
) Value {

	fields := map[string]Value{
		sema.AuthAccountContractsTypeAddFunctionName:                  addFunction,
		sema.AuthAccountContractsTypeGetFunctionName:                  getFunction,
		sema.AuthAccountContractsTypeBorrowFunctionName:               borrowFunction,
		sema.AuthAccountContractsTypeRemoveFunctionName:               removeFunction,
		sema.AuthAccountContractsTypeUpdate__experimentalFunctionName: updateFunction,
	}

	computeField := func(
		name string,
		interpreter *Interpreter,
		locationRange LocationRange,
	) Value {
		switch name {
		case sema.AuthAccountContractsTypeNamesFieldName:
			return namesGetter(interpreter, locationRange)
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountContractsStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("AuthAccount.Contracts(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountContractsTypeID,
		authAccountContractsStaticType,
		authAccountContractsFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// PublicAccountContractsValue

var publicAccountContractsTypeID = sema.PublicAccountContractsType.ID()
var publicAccountContractsStaticType StaticType = PrimitiveStaticTypePublicAccountContracts

func NewPublicAccountContractsValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
	namesGetter ContractNamesGetter,
) Value {

	fields := map[string]Value{
		sema.PublicAccountContractsTypeGetFunctionName:    getFunction,
		sema.PublicAccountContractsTypeBorrowFunctionName: borrowFunction,
	}

	computeField := func(
		name string,
		interpreter *Interpreter,
		locationRange LocationRange,
	) Value {
		switch name {
		case sema.PublicAccountContractsTypeNamesFieldName:
			return namesGetter(interpreter, locationRange)
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountContractsStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("PublicAccount.Contracts(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountContractsTypeID,
		publicAccountContractsStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}
