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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/interpreter"
)

func IsSubType(context TypeConverterContext, sourceType, targetType bbq.StaticType) bool {
	// TODO: Avoid conversion to sema types.
	inter := context.Interpreter()
	return inter.IsSubType(sourceType, targetType)
}

// UnwrapOptionalType returns the type if it is not an optional type,
// or the inner-most type if it is (optional types are repeatedly unwrapped)
func UnwrapOptionalType(ty bbq.StaticType) bbq.StaticType {
	for {
		optionalType, ok := ty.(*interpreter.OptionalStaticType)
		if !ok {
			return ty
		}
		ty = optionalType.Type
	}
}
