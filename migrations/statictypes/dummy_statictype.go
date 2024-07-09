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

package statictypes

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// dummyStaticType is just a wrapper for `interpreter.PrimitiveStaticType`
// with an overridden `ID` function.
// This is only for testing the migration, so that a type-value with a deprecated primitive type
// (e.g: `PrimitiveStaticTypePublicAccount`) is insertable as a dictionary key (to populate the storage).
// i.e: To make hashing function works, which requires `ID()` method.
// Currently, this is not possible, since the `ID` function of `interpreter.PrimitiveStaticType`
// of deprecated types no longer work, as it relies on the `sema.Type`,
// but the corresponding `sema.Type` for the deprecated primitive types are no longer available.
type dummyStaticType struct {
	interpreter.PrimitiveStaticType
}

func (t dummyStaticType) ID() common.TypeID {
	return common.TypeID(t.String())
}

func (t dummyStaticType) Equal(other interpreter.StaticType) bool {
	otherDummyType, ok := other.(dummyStaticType)
	if !ok {
		return false
	}

	return t == otherDummyType
}
