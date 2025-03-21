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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
)

type AddressValue common.Address

var _ Value = AddressValue{}

func (AddressValue) isValue() {}

func (AddressValue) StaticType(StaticTypeContext) bbq.StaticType {
	return interpreter.PrimitiveStaticTypeAddress
}

func (v AddressValue) Transfer(TransferContext, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v AddressValue) String() string {
	return format.Address(common.Address(v))
}
