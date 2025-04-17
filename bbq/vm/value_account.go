/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/stdlib"
)

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

func NewAuthAccountReferenceValue(
	conf *Context,
	handler stdlib.AccountHandler,
	address common.Address,
) interpreter.Value {
	return stdlib.NewAccountReferenceValue(
		conf,
		handler,
		interpreter.AddressValue(address),
		interpreter.FullyEntitledAccountAccess,
		EmptyLocationRange,
	)
}

func NewAccountReferenceValue(
	conf *Context,
	handler stdlib.AccountHandler,
	address common.Address,
) interpreter.Value {
	return stdlib.NewAccountReferenceValue(
		conf,
		handler,
		interpreter.AddressValue(address),
		interpreter.UnauthorizedAccess,
		EmptyLocationRange,
	)
}

// members

func init() {
	// Any member methods goes here
}

func getAccountTypePrivateAddressValue(receiver Value) interpreter.AddressValue {
	simpleCompositeValue := receiver.(*interpreter.SimpleCompositeValue)

	addressMetaInfo := simpleCompositeValue.PrivateField(interpreter.AccountTypePrivateAddressFieldName)
	address, ok := addressMetaInfo.(interpreter.AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return address
}
