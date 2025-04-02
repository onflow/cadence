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

package stdlib

import (
	"fmt"

	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type PanicError struct {
	interpreter.LocationRange
	Message string
}

var _ errors.UserError = PanicError{}

func (PanicError) IsUserError() {}

func (e PanicError) Error() string {
	return fmt.Sprintf("panic: %s", e.Message)
}

const panicFunctionDocString = `
Terminates the program unconditionally and reports a message which explains why the unrecoverable error occurred.
`

var PanicFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "message",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	sema.NeverTypeAnnotation,
)

var PanicFunction = NewStandardLibraryStaticFunction(
	"panic",
	PanicFunctionType,
	panicFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		messageValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		message := messageValue.Str

		panic(PanicError{
			Message:       message,
			LocationRange: invocation.LocationRange,
		})
	},
)
