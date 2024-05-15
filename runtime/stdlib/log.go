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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var LogFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "value",
			TypeAnnotation: sema.AnyStructTypeAnnotation,
		},
	},
	sema.VoidTypeAnnotation,
)

const logFunctionDocString = `
Logs a string representation of the given value
`

type Logger interface {
	// ProgramLog logs program logs.
	ProgramLog(message string, locationRange interpreter.LocationRange) error
}

func NewLogFunction(logger Logger) StandardLibraryValue {
	return NewStandardLibraryStaticFunction(
		"log",
		LogFunctionType,
		logFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			value := invocation.Arguments[0]
			locationRange := invocation.LocationRange

			inter := invocation.Interpreter
			message := value.MeteredString(inter, interpreter.SeenReferences{}, locationRange)

			var err error
			errors.WrapPanic(func() {
				err = logger.ProgramLog(message, locationRange)
			})
			if err != nil {
				panic(interpreter.WrappedExternalError(err))
			}

			return interpreter.Void
		},
	)
}
