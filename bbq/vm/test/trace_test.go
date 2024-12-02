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

package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
)

// compares the traces between vm and interpreter
func TestTrace(t *testing.T) {
	t.Run("simple trace test", func(t *testing.T) {
		t.Parallel()

		code := `
			struct Foo {
				var id : Int

				init(_ id: Int) {
					self.id = id
				}
			}

			resource Bar {
                var id : Int

                init(_ id: Int) {
                    self.id = id
                }
            }

            fun test() {
                var i = 0
				var c = [1,2,3]
				var s = Foo(0)
				s.id = s.id + 2

				var r <- create Bar(5)
				destroy r
            }
        `

		checker, err := ParseAndCheck(t, code)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		var vmLogs []string

		vmConfig := &vm.Config{
			Tracer: interpreter.Tracer{
				TracingEnabled: true,
				OnRecordTrace: func(executer interpreter.Traceable, operationName string, duration time.Duration, attrs []attribute.KeyValue) {
					vmLogs = append(vmLogs, fmt.Sprintf("%s: %v", operationName, attrs))
				},
			},
		}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())

		var interLogs []string
		storage := interpreter.NewInMemoryStorage(nil)
		var uuid uint64 = 0
		inter, err := interpreter.NewInterpreter(
			interpreter.ProgramFromChecker(checker),
			TestLocation,
			&interpreter.Config{
				Tracer: interpreter.Tracer{
					OnRecordTrace: func(inter interpreter.Traceable,
						operationName string,
						duration time.Duration,
						attrs []attribute.KeyValue) {
						interLogs = append(interLogs, fmt.Sprintf("%s: %v", operationName, attrs))
					},
					TracingEnabled: true,
				},
				Storage: storage,
				UUIDHandler: func() (uint64, error) {
					uuid++
					return uuid, nil
				},
			},
		)
		require.NoError(t, err)

		err = inter.Interpret()
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		// compare traces
		AssertEqualWithDiff(t, vmLogs, interLogs)
	})
}
