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
	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountContractsValue

var authAccountContractsLocation common.Location = nil
var authAccountContractsQualifiedIdentifier = sema.AuthAccountContractsType.QualifiedIdentifier()
var authAccountContractsCompositeKind = sema.AuthAccountContractsType.Kind
var authAccountContractsTypeInfo = encodeCompositeOrderedMapTypeInfo(
	authAccountContractsLocation,
	authAccountContractsQualifiedIdentifier,
	authAccountContractsCompositeKind,
)

func NewAuthAccountContractsValue(
	interpreter *Interpreter,
	address AddressValue,
	addFunction FunctionValue,
	updateFunction FunctionValue,
	getFunction FunctionValue,
	removeFunction FunctionValue,
	namesGetter func(interpreter *Interpreter) *ArrayValue,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.AuthAccountContractsTypeAddFunctionName,
			Value: addFunction,
		},
		{
			Name:  sema.AuthAccountContractsTypeGetFunctionName,
			Value: getFunction,
		},
		{
			Name:  sema.AuthAccountContractsTypeRemoveFunctionName,
			Value: removeFunction,
		},
		{
			Name:  sema.AuthAccountContractsTypeUpdateExperimentalFunctionName,
			Value: updateFunction,
		},
	}
	computedFields := map[string]ComputedField{
		sema.AuthAccountContractsTypeNamesField: func(interpreter *Interpreter) Value {
			return namesGetter(interpreter)
		},
	}

	stringer := func(_ SeenReferences) string {
		return fmt.Sprintf("AuthAccount.Contracts(%s)", address)
	}

	v := NewCompositeValueWithTypeInfo(
		interpreter,
		authAccountContractsLocation,
		authAccountContractsQualifiedIdentifier,
		authAccountContractsCompositeKind,
		fields,
		common.Address{},
		authAccountContractsTypeInfo,
	)

	v.Stringer = stringer
	v.ComputedFields = computedFields

	return v
}

// PublicAccountContractsValue

var publicAccountContractsLocation common.Location = nil
var publicAccountContractsQualifiedIdentifier = sema.PublicAccountContractsType.QualifiedIdentifier()
var publicAccountContractsCompositeKind = sema.PublicAccountContractsType.Kind
var publicAccountContractsTypeInfo = encodeCompositeOrderedMapTypeInfo(
	publicAccountContractsLocation,
	publicAccountContractsQualifiedIdentifier,
	publicAccountContractsCompositeKind,
)

func NewPublicAccountContractsValue(
	interpreter *Interpreter,
	address AddressValue,
	getFunction FunctionValue,
	namesGetter func(interpreter *Interpreter) *ArrayValue,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.PublicAccountContractsTypeGetFunctionName,
			Value: getFunction,
		},
	}

	computedFields := map[string]ComputedField{
		sema.PublicAccountContractsTypeNamesField: func(interpreter *Interpreter) Value {
			return namesGetter(interpreter)
		},
	}

	stringer := func(_ SeenReferences) string {
		return fmt.Sprintf("PublicAccount.Contracts(%s)", address)
	}

	v := NewCompositeValueWithTypeInfo(
		interpreter,
		publicAccountContractsLocation,
		publicAccountContractsQualifiedIdentifier,
		publicAccountContractsCompositeKind,
		fields,
		common.Address{},
		publicAccountContractsTypeInfo,
	)

	v.Stringer = stringer
	v.ComputedFields = computedFields

	return v
}
