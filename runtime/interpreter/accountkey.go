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
	"github.com/onflow/cadence/runtime/sema"
)

var accountKeyTypeID = sema.AccountKeyType.ID()
var accountKeyStaticType StaticType = PrimitiveStaticTypeAccountKey

var accountKeyFieldNames = []string{
	sema.AccountKeyKeyIndexField,
	sema.AccountKeyPublicKeyField,
	sema.AccountKeyHashAlgoField,
	sema.AccountKeyWeightField,
	sema.AccountKeyIsRevokedField,
}

// NewAccountKeyValue constructs an AccountKey value.
func NewAccountKeyValue(
	keyIndex IntValue,
	publicKey *CompositeValue,
	hashAlgo *CompositeValue,
	weight UFix64Value,
	isRevoked BoolValue,
) *SimpleCompositeValue {
	fields := map[string]Value{
		sema.AccountKeyKeyIndexField:  keyIndex,
		sema.AccountKeyPublicKeyField: publicKey,
		sema.AccountKeyHashAlgoField:  hashAlgo,
		sema.AccountKeyWeightField:    weight,
		sema.AccountKeyIsRevokedField: isRevoked,
	}

	return NewSimpleCompositeValue(
		accountKeyTypeID,
		accountKeyStaticType,
		accountKeyFieldNames,
		fields,
		nil,
		nil,
		nil,
	)
}
