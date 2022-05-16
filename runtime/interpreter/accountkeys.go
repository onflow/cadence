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

	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccountKeys

var authAccountKeysTypeID = sema.AuthAccountKeysType.ID()
var authAccountKeysStaticType StaticType = PrimitiveStaticTypeAuthAccountKeys

// NewAuthAccountKeysValue constructs a AuthAccount.Keys value.
func NewAuthAccountKeysValue(
	inter *Interpreter,
	address AddressValue,
	addFunction FunctionValue,
	getFunction FunctionValue,
	revokeFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.AccountKeysAddFunctionName:    addFunction,
		sema.AccountKeysGetFunctionName:    getFunction,
		sema.AccountKeysRevokeFunctionName: revokeFunction,
	}

	var str string
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("AuthAccount.Keys(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		inter,
		authAccountKeysTypeID,
		authAccountKeysStaticType,
		nil,
		fields,
		nil,
		nil,
		stringer,
	)
}

// PublicAccountKeys

var publicAccountKeysTypeID = sema.PublicAccountKeysType.ID()
var publicAccountKeysStaticType StaticType = PrimitiveStaticTypePublicAccountKeys

// NewPublicAccountKeysValue constructs a PublicAccount.Keys value.
func NewPublicAccountKeysValue(
	inter *Interpreter,
	address AddressValue,
	getFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.AccountKeysGetFunctionName: getFunction,
	}

	var str string
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("PublicAccount.Keys(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		inter,
		publicAccountKeysTypeID,
		publicAccountKeysStaticType,
		nil,
		fields,
		nil,
		nil,
		stringer,
	)
}
