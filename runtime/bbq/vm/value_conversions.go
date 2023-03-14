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

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

// Utility methods to convert between old and new values.
// These are temporary until all parts of the interpreter are migrated to the vm.

func InterpreterValueToVMValue(config *Config, value interpreter.Value) Value {
	switch value := value.(type) {
	case interpreter.IntValue:
		return IntValue{value.BigInt.Int64()}
	case *interpreter.StringValue:
		return StringValue{Str: []byte(value.Str)}
	case *interpreter.CompositeValue:
		return NewCompositeValue(
			value.Location,
			value.QualifiedIdentifier,
			value.Kind,
			common.Address{},
			config.Storage,
		)
	default:
		panic(errors.NewUnreachableError())
	}
}

var inter = func() *interpreter.Interpreter {
	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage: interpreter.NewInMemoryStorage(nil),
		},
	)

	if err != nil {
		panic(err)
	}

	return inter
}()

func VMValueToInterpreterValue(value Value) interpreter.Value {
	switch value := value.(type) {
	case IntValue:
		return interpreter.NewIntValueFromInt64(nil, value.SmallInt)
	case StringValue:
		return interpreter.NewUnmeteredStringValue(string(value.Str))
	case *CompositeValue:
		return interpreter.NewCompositeValue(
			inter,
			interpreter.EmptyLocationRange,
			value.Location,
			value.QualifiedIdentifier,
			value.Kind,
			nil,
			common.Address{},
		)
	default:
		panic(errors.NewUnreachableError())
	}
}
