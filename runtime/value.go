/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package runtime

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

// A Value is a Cadence value emitted by the runtime.
//
// Runtime values can be converted to a simplified representation
// and then further encoded for transport or use in other languages
// and environments.
type Value struct {
	interpreter.Value
	inter *interpreter.Interpreter
}

func newRuntimeValue(value interpreter.Value, inter *interpreter.Interpreter) Value {
	return Value{
		Value: value,
		inter: inter,
	}
}

func (v Value) Interpreter() *interpreter.Interpreter {
	return v.inter
}

type Event struct {
	Type   Type
	Fields []Value
}

type Address = common.Address
