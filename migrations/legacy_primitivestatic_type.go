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

package migrations

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

// LegacyPrimitiveStaticType simulates the old primitive-static-type
// which uses the old type-ids for hashing.
type LegacyPrimitiveStaticType struct {
	interpreter.PrimitiveStaticType
}

var _ interpreter.StaticType = LegacyPrimitiveStaticType{}

func (t LegacyPrimitiveStaticType) Equal(other interpreter.StaticType) bool {
	switch other := other.(type) {

	case LegacyPrimitiveStaticType:
		return t == other

	case interpreter.PrimitiveStaticType:
		return t.PrimitiveStaticType == other

	default:
		return false
	}
}

func (t LegacyPrimitiveStaticType) ID() common.TypeID {
	switch t.PrimitiveStaticType {
	case interpreter.PrimitiveStaticTypeAuthAccount: //nolint:staticcheck
		return "AuthAccount"
	case interpreter.PrimitiveStaticTypePublicAccount: //nolint:staticcheck
		return "PublicAccount"
	case interpreter.PrimitiveStaticTypeAuthAccountCapabilities: //nolint:staticcheck
		return "AuthAccount.Capabilities"
	case interpreter.PrimitiveStaticTypePublicAccountCapabilities: //nolint:staticcheck
		return "PublicAccount.Capabilities"
	case interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities: //nolint:staticcheck
		return "AuthAccount.AccountCapabilities"
	case interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities: //nolint:staticcheck
		return "AuthAccount.StorageCapabilities"
	case interpreter.PrimitiveStaticTypeAuthAccountContracts: //nolint:staticcheck
		return "AuthAccount.Contracts"
	case interpreter.PrimitiveStaticTypePublicAccountContracts: //nolint:staticcheck
		return "PublicAccount.Contracts"
	case interpreter.PrimitiveStaticTypeAuthAccountKeys: //nolint:staticcheck
		return "AuthAccount.Keys"
	case interpreter.PrimitiveStaticTypePublicAccountKeys: //nolint:staticcheck
		return "PublicAccount.Keys"
	case interpreter.PrimitiveStaticTypeAuthAccountInbox: //nolint:staticcheck
		return "AuthAccount.Inbox"
	case interpreter.PrimitiveStaticTypeAccountKey: //nolint:staticcheck
		return "AccountKey"
	default:
		panic(errors.NewUnreachableError())

	}
}
