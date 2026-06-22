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

package runtime_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

// TestRuntimeVariadicFunctionArgumentPassing exercises the variadic invocation
// code path end-to-end with a real Cadence program that calls an injected
// variadic function.
//
// A variadic call passes more arguments than there are declared parameters.
// In the compiler, the invocation's `parameterTypes` is sized to the argument
// count, but only the leading (declared) parameters have a type; the trailing
// variadic entries are nil. The compiler's `loadTypes` compacts those nils
// away, and the VM pairs the remaining types positionally with the arguments.
//
// This test verifies that the compiled VM and the tree-walking interpreter
// agree on how variadic arguments are passed (in particular, that the declared
// parameter is boxed to its optional type, while the trailing variadic
// arguments are passed through as-is by BOTH engines).
func TestRuntimeVariadicFunctionArgumentPassing(t *testing.T) {

	t.Parallel()

	const functionName = "describeArgs"

	// describeArgs(_ x: Int? ...): String
	functionType := &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "x",
				TypeAnnotation: sema.NewTypeAnnotation(&sema.OptionalType{Type: sema.IntType}),
			},
		},
		Arity:                &sema.Arity{Min: 1, Max: -1},
		ReturnTypeAnnotation: sema.StringTypeAnnotation,
	}

	// For each received argument, report whether it arrived boxed in an optional
	// (`some`/`nil`) or as a bare (non-optional) value.
	nativeFunction := func(
		_ interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		arguments []interpreter.Value,
	) interpreter.Value {
		parts := make([]string, len(arguments))
		for i, arg := range arguments {
			switch arg.(type) {
			case *interpreter.SomeValue:
				parts[i] = "some"
			case interpreter.NilValue:
				parts[i] = "nil"
			default:
				parts[i] = "bare"
			}
		}
		return interpreter.NewUnmeteredStringValue(strings.Join(parts, ","))
	}

	script := []byte(`
      access(all) fun main(): String {
          // single (declared) argument, then a true variadic call (extra args)
          return describeArgs(1)
              .concat("|")
              .concat(describeArgs(1, 2, 3))
              .concat("|")
              .concat(describeArgs(1, nil, 3))
      }
    `)

	run := func(useVM bool) string {
		rt := NewTestRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
		}

		var function interpreter.FunctionValue
		var environment Environment
		if useVM {
			function = vm.NewNativeFunctionValue(
				functionName,
				functionType,
				nativeFunction,
			)
			environment = NewScriptVMEnvironment(Config{})
		} else {
			function = interpreter.NewStaticHostFunctionValueFromNativeFunction(
				nil,
				functionType,
				nativeFunction,
			)
			environment = NewScriptInterpreterEnvironment(Config{})
		}

		environment.DeclareValue(
			stdlib.StandardLibraryValue{
				Name:  functionName,
				Type:  functionType,
				Kind:  common.DeclarationKindFunction,
				Value: function,
			},
			nil,
		)

		result, err := rt.ExecuteScript(
			Script{Source: script},
			Context{
				Interface:   runtimeInterface,
				Location:    common.ScriptLocation{},
				Environment: environment,
				UseVM:       useVM,
			},
		)
		require.NoError(t, err)

		str, ok := result.(cadence.String)
		require.True(t, ok, "unexpected result type %T", result)
		return string(str)
	}

	interpreterResult := run(false)
	vmResult := run(true)

	// The declared parameter is boxed to Int? (`some`).
	// The trailing variadic arguments are passed through as-is by both engines:
	// bare Int values stay `bare`, and an explicit `nil` stays `nil`.
	const expected = "some|some,bare,bare|some,nil,bare"

	require.Equal(t, expected, interpreterResult, "interpreter result")
	require.Equal(t, expected, vmResult, "vm result")
	require.Equal(t, interpreterResult, vmResult,
		"interpreter and VM must agree on variadic argument passing\n  interpreter=%q\n  vm=%q",
		interpreterResult, vmResult,
	)
}
