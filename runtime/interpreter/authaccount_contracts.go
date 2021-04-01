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

func NewAuthAccountContractsValue(
	address AddressValue,
	addFunction FunctionValue,
	updateFunction FunctionValue,
	getFunction FunctionValue,
	removeFunction FunctionValue,
) *CompositeValue {
	fields := NewStringValueOrderedMap()
	fields.Set(sema.AuthAccountContractsTypeAddFunctionName, addFunction)
	fields.Set(sema.AuthAccountContractsTypeGetFunctionName, getFunction)
	fields.Set(sema.AuthAccountContractsTypeRemoveFunctionName, removeFunction)
	fields.Set(sema.AuthAccountContractsTypeUpdateExperimentalFunctionName, updateFunction)

	stringer := func() string {
		return fmt.Sprintf("AuthAccount.Contracts(%s)", address)
	}

	return &CompositeValue{
		QualifiedIdentifier: sema.AuthAccountContractsType.QualifiedIdentifier(),
		Kind:                sema.AuthAccountContractsType.Kind,
		Fields:              fields,
		stringer:            stringer,
	}
}
