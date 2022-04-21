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
	"fmt"

	"github.com/onflow/cadence/runtime/interpreter"
)

const logFunctionDocString = `
Logs a string representation of the given value
`

var LogFunction = NewStandardLibraryFunction(
	"log",
	LogFunctionType,
	logFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		fmt.Println(invocation.Arguments[0].ToMeteredString(invocation.Interpreter, interpreter.SeenReferences{}))
		return interpreter.VoidValue{}
	},
)
