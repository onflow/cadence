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

// AssertFunction

const AssertFunctionName = "assert"

const assertFunctionDocString = `
Terminates the program if the given condition is false, and reports a message which explains how the condition is false.
Use this function for internal sanity checks.

The message argument is optional.
`

var AssertFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "condition",
			TypeAnnotation: sema.BoolTypeAnnotation,
		},
		{
			Identifier:     "message",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	// `message` parameter is optional
	Arity: &sema.Arity{Min: 1, Max: 2},
}

var NativeAssertFunction = interpreter.NativeFunction(
	func(
		_ interpreter.NativeFunctionContext,
		_ interpreter.TypeParameterGetter,
		_ interpreter.Value,
		args ...interpreter.Value,
	) interpreter.Value {
		result := interpreter.AssertValueOfType[interpreter.BoolValue](args[0])
		var message string
		if len(args) > 1 {
			messageValue := interpreter.AssertValueOfType[*interpreter.StringValue](args[1])
			message = messageValue.Str
		}
		return Assert(result, message)
	},
)

var InterpreterAssertFunction = NewNativeStandardLibraryStaticFunction(
	AssertFunctionName,
	AssertFunctionType,
	assertFunctionDocString,
	NativeAssertFunction,
	false,
)

var VMAssertFunction = NewNativeStandardLibraryStaticFunction(
	AssertFunctionName,
	AssertFunctionType,
	assertFunctionDocString,
	NativeAssertFunction,
	true,
)

func Assert(result interpreter.BoolValue, message string) interpreter.Value {
	if !result {
		panic(&AssertionError{
			Message: message,
		})
	}
	return interpreter.Void
}

// AssertionError

type AssertionError struct {
	interpreter.LocationRange
	Message string
}

var _ errors.UserError = &AssertionError{}
var _ interpreter.HasLocationRange = &AssertionError{}

func (*AssertionError) IsUserError() {}

func (e *AssertionError) Error() string {
	const message = "assertion failed"
	if e.Message == "" {
		return message
	}
	return fmt.Sprintf("%s: %s", message, e.Message)
}

func (e *AssertionError) SetLocationRange(locationRange interpreter.LocationRange) {
	e.LocationRange = locationRange
}
