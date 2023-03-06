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

package values

import (
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

// Utility methods to convert between old and new values.
// These are temporary until all parts of the interpreter are migrated to the vm.

func InterpreterValueToVMValue(value interpreter.Value) Value {
	switch value := value.(type) {
	case interpreter.IntValue:
		return IntValue{value.BigInt.Int64()}
	case *interpreter.StringValue:
		return StringValue{String: []byte(value.Str)}
	default:
		panic(errors.NewUnreachableError())
	}
}

func VMValueToInterpreterValue(value Value) interpreter.Value {
	switch value := value.(type) {
	case IntValue:
		return interpreter.NewIntValueFromInt64(nil, value.SmallInt)
	case StringValue:
		return interpreter.NewUnmeteredStringValue(string(value.String))
	default:
		panic(errors.NewUnreachableError())
	}
}
