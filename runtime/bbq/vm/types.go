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

package vm

import "github.com/onflow/cadence/interpreter"

type StaticType = interpreter.StaticType

func IsSubType(sourceType, targetType StaticType) bool {
	if targetType == interpreter.PrimitiveStaticTypeAny {
		return true
	}

	// Optimization: If the static types are equal, then no need to check further.
	if sourceType.Equal(targetType) {
		return true
	}

	// TODO: Add the remaining subType rules

	return true
}
