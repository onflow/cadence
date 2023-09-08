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

package stdlib

import (
	"encoding/binary"

	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const unsafeRandomFunctionDocString = `
Returns a pseudo-random number.

NOTE: The use of this function is unsafe if not used correctly.

Follow best practices to prevent security issues when using this function
`

var unsafeRandomFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.UInt64TypeAnnotation,
)

type UnsafeRandomGenerator interface {
	// ReadRandom reads pseudo-random bytes into the input slice, using distributed randomness.
	ReadRandom([]byte) error
}

func NewUnsafeRandomFunction(generator UnsafeRandomGenerator) StandardLibraryValue {
	return NewStandardLibraryFunction(
		"unsafeRandom",
		unsafeRandomFunctionType,
		unsafeRandomFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			return interpreter.NewUInt64Value(
				invocation.Interpreter,
				func() uint64 {
					var buffer [8]byte
					var err error
					errors.WrapPanic(func() {
						err = generator.ReadRandom(buffer[:])
					})
					if err != nil {
						panic(interpreter.WrappedExternalError(err))
					}
					return binary.LittleEndian.Uint64(buffer[:])
				},
			)
		},
	)
}
