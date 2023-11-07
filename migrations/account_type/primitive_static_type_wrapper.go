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

package account_type

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// primitiveStaticTypeWrapper is just a wrapper for `interpreter.PrimitiveStaticType`
// with a custom `ID` function.
// This is only for testing the migration, since the `ID` function of `interpreter.PrimitiveStaticType`
// for deprecated types no longer work the original `ID` function relies on the `sema.Type`,
// but corresponding `sema.Type` for the deprecated primitive types are no longer available.
type primitiveStaticTypeWrapper struct {
	interpreter.PrimitiveStaticType
}

func (t primitiveStaticTypeWrapper) ID() common.TypeID {
	return common.TypeID(t.String())
}
