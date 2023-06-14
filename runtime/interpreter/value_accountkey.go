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
	"github.com/onflow/cadence/runtime/sema"
)

var accountKeyTypeID = sema.AccountKeyType.ID()
var accountKeyStaticType StaticType = PrimitiveStaticTypeAccountKey // unmetered
var accountKeyFieldNames = []string{
	sema.AccountKeyKeyIndexFieldName,
	sema.AccountKeyPublicKeyFieldName,
	sema.AccountKeyHashAlgoFieldName,
	sema.AccountKeyWeightFieldName,
	sema.AccountKeyIsRevokedFieldName,
}

// NewAccountKeyValue constructs an AccountKey value.
func NewAccountKeyValue(
	inter *Interpreter,
	keyIndex IntValue,
	publicKey *CompositeValue,
	hashAlgo Value,
	weight UFix64Value,
	isRevoked BoolValue,
) *SimpleCompositeValue {
	fields := map[string]Value{
		sema.AccountKeyKeyIndexFieldName:  keyIndex,
		sema.AccountKeyPublicKeyFieldName: publicKey,
		sema.AccountKeyHashAlgoFieldName:  hashAlgo,
		sema.AccountKeyWeightFieldName:    weight,
		sema.AccountKeyIsRevokedFieldName: isRevoked,
	}

	return NewSimpleCompositeValue(
		inter,
		accountKeyTypeID,
		accountKeyStaticType,
		accountKeyFieldNames,
		fields,
		nil,
		nil,
		nil,
	)
}
