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

	. "github.com/onflow/cadence/test_utils/sema_utils"

	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/vm"
)

func TestTrace(t *testing.T) {
	t.Run("simple trace test", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			struct Foo {
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
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
		program := comp.Compile()

		printProgram("", program)

		vmConfig := &vm.Config{
			TracingEnabled: true,
			OnRecordTrace: func(vm *vm.VM, operationName string, duration time.Duration, attrs []attribute.KeyValue) {
				fmt.Println("======   LOG | Operation: ", operationName, " Time: ", duration, " Attribute", attrs, "   =======")
			},
		}
		vmInstance := vm.NewVM(scriptLocation(), program, vmConfig)

		_, err = vmInstance.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, 0, vmInstance.StackSize())
	})
}
