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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

const LogFunctionName = "log"

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
	ProgramLog(message string) error
}

type FunctionLogger func(message string) error

var _ Logger = FunctionLogger(nil)

func (f FunctionLogger) ProgramLog(message string) error {
	return f(message)
}

func NewInterpreterLogFunction(logger Logger) StandardLibraryValue {
	return NewInterpreterStandardLibraryStaticFunction(
		LogFunctionName,
		LogFunctionType,
		logFunctionDocString,
		func(invocation interpreter.Invocation) interpreter.Value {
			invocationContext := invocation.InvocationContext
			value := invocation.Arguments[0]
			return Log(
				invocationContext,
				logger,
				value,
			)
		},
	)
}

func NewVMLogFunction(logger Logger) StandardLibraryValue {
	return NewVMStandardLibraryStaticFunction(
		LogFunctionName,
		LogFunctionType,
		logFunctionDocString,
		func(context *vm.Context, _ []bbq.StaticType, _ vm.Value, arguments ...interpreter.Value) interpreter.Value {
			value := arguments[0]
			return Log(
				context,
				logger,
				value,
			)
		},
	)
}

func Log(
	context interpreter.ValueStringContext,
	logger Logger,
	value interpreter.Value,
) interpreter.Value {
	message := value.MeteredString(context, interpreter.SeenReferences{})

	err := logger.ProgramLog(message)
	if err != nil {
		panic(err)
	}

	return interpreter.Void
}
