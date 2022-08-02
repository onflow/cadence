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

package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func executeScript(script string, runtimeInterface Interface) (cadence.Value, error) {
	runtime := newTestInterpreterRuntime()

	return runtime.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: runtimeInterface,
			Location:  utils.TestLocation,
		},
	)
}

func TestAssertFunction(t *testing.T) {

	t.Parallel()

	script := `
        import Test

        pub fun main() {
          Test.assert(false, "condition not satisfied")
        }
    `

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
	}

	_, err := executeScript(script, runtimeInterface)
	require.Error(t, err)
	assert.ErrorAs(t, err, &stdlib.AssertionError{})
}

func TestFailFunction(t *testing.T) {
	t.Parallel()

	script := `
        import Test

        pub fun main() {
            Test.fail()
        }
    `

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
	}

	_, err := executeScript(script, runtimeInterface)
	require.Error(t, err)
	assert.ErrorAs(t, err, &stdlib.AssertionError{})
}

func TestBlockchain(t *testing.T) {

	t.Parallel()

	script := `
        import Test

        pub fun main() {
            var bc = Test.newEmulatorBlockchain()
        }
    `

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
	}

	_, err := executeScript(script, runtimeInterface)

	require.NoError(t, err)
}

func TestExecuteScript(t *testing.T) {

	t.Parallel()

	script := `
        import Test

        pub fun main() {
            var bc = Test.newEmulatorBlockchain()
            bc.executeScript("pub fun foo() {}")
        }
    `

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
	}

	_, err := executeScript(script, runtimeInterface)

	require.Error(t, err)
	assert.ErrorAs(t, err, &interpreter.TestFrameworkNotProvidedError{})
}

func TestEqualMatcher(t *testing.T) {

	t.Parallel()

	t.Run("equal matcher with primitive", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let f = Foo()
                Test.equal(1)
            }

            pub struct Foo {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("equal matcher with struct", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let f = Foo()
                Test.equal(f)
            }

            pub struct Foo {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("equal matcher with resource", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let f <- create Foo()
                Test.equal(<-f)
            }

            pub resource Foo {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                Test.expect("this string", Test.equal("this string"))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("fail", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                Test.expect("this string", Test.equal("other string"))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("different types", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                Test.expect("string", Test.equal(1))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("resources", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let f1 <- create Foo()
                let f2 <- create Foo()
                Test.expect(<-f1, Test.equal(<-f2))
            }

            pub resource Foo {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("different resources", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let foo <- create Foo()
                let bar <- create Bar()
                Test.expect(<-foo, Test.equal(<-bar))
            }

            pub resource Foo {}
            pub resource Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("expect struct with (resource, struct matcher)", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let foo <- create Foo()
                let bar = Bar()
                Test.expect(<-foo, Test.equal(bar))
            }

            pub resource Foo {}
            pub struct Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("expect struct with (struct, resource matcher)", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let foo = Foo()
                let bar <- create Bar()
                Test.expect(foo, Test.equal(<-bar))
            }

            pub struct Foo {}
            pub resource Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("expect resource with (struct, resource matcher)", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let foo = Foo()
                let bar <- create Bar()
                Test.expect(foo, Test.equal(<-bar))
            }

            pub struct Foo {}
            pub resource Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})

	t.Run("expect resource with (resource, struct matcher)", func(t *testing.T) {
		script := `
            import Test

            pub fun main() {
                let foo <- create Foo()
                let bar = Bar()
                Test.expect(<-foo, Test.equal(bar))
            }

            pub resource Foo {}
            pub struct Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		assert.ErrorAs(t, err, &stdlib.AssertionError{})
	})
}
