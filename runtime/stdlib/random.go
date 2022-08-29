/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const unsafeRandomFunctionDocString = `
Returns a pseudo-random number.

NOTE: The use of this function is unsafe if not used correctly.

Follow best practices to prevent security issues when using this function
`

var unsafeRandomFunctionType = &sema.FunctionType{
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.UInt64Type,
	),
}

type UnsafeRandomGenerator interface {
	// UnsafeRandom returns a random uint64,
	// where the process of random number derivation is not cryptographically secure.
	UnsafeRandom() (uint64, error)
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
					var rand uint64
					var err error
					wrapPanic(func() {
						rand, err = generator.UnsafeRandom()
					})
					if err != nil {
						panic(err)
					}
					return rand
				},
			)
		},
	)
}
