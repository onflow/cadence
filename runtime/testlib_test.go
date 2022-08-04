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
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
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

func TestInterpretAssertFunction(t *testing.T) {

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

func TestInterpretFailFunction(t *testing.T) {
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

func TestInterpretBlockchain(t *testing.T) {

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

func TestInterpretExecuteScriptFunction(t *testing.T) {

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

func TestInterpretMatcher(t *testing.T) {
	t.Parallel()

	t.Run("custom matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher(fun (_ value: AnyStruct): Bool {
                     if !value.getType().isSubtype(of: Type<Int>()) {
                        return false
                    }

                    return (value as! Int) > 5
                })

                assert(matcher.test(8))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("custom matcher primitive type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher(fun (_ value: Int): Bool {
                     return value == 7
                })

                assert(matcher.test(7))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("custom resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher(fun (_ value: &Foo): Bool {
                    return value.a == 4
                })

                let f <-create Foo(4)

                assert(matcher.test(&f as &Foo))

                destroy f
            }

            pub resource Foo {
                pub let a: Int

                init(_ a: Int) {
                    self.a = a
                }
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("custom resource matcher invalid type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher(fun (_ value: @Foo): Bool {
                     destroy value
                     return true
                })
            }

            pub resource Foo {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.Error(t, err)
		errors := checker.ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errors[0])
	})

	t.Run("custom matcher with explicit type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher<Int>(fun (_ value: Int): Bool {
                     return value == 7
                })

                assert(matcher.test(7))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("custom matcher with mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {

                let matcher = Test.NewMatcher<String>(fun (_ value: Int): Bool {
                     return value == 7
                })
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		errors := checker.ExpectCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errors[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errors[1])
	})
}

func TestInterpretEqualMatcher(t *testing.T) {

	t.Parallel()

	t.Run("equal matcher with primitive", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let matcher = Test.equal(1)
                assert(matcher.test(1))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("equal matcher with struct", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let f = Foo()
                let matcher = Test.equal(f)
                assert(matcher.test(f))
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
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let f <- create Foo()
                let matcher = Test.equal(<-f)
                assert(matcher.test(<- create Foo()))
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

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let matcher = Test.equal<String>("hello")
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("with incorrect types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let matcher = Test.equal<String>(1)
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		errors := checker.ExpectCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errors[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errors[1])
	})

	t.Run("matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneOrTwo = one.or(two)

                assert(oneOrTwo.test(1))
                assert(oneOrTwo.test(2))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("matcher or fail", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneOrTwo = one.or(two)

                assert(oneOrTwo.test(3))
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

	t.Run("matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let one = Test.equal(1)
                let two = Test.equal(2)

                let oneAndTwo = one.and(two)

                assert(oneAndTwo.test(1))
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

	t.Run("chained matchers", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let one = Test.equal(1)
                let two = Test.equal(2)
                let three = Test.equal(3)

                let oneOrTwoOrThree = one.or(two).or(three)

                assert(oneOrTwoOrThree.test(3))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("resource matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let foo <- create Foo()
                let bar <- create Bar()

                let fooMatcher = Test.equal(<-foo)
                let barMatcher = Test.equal(<-bar)

                let matcher = fooMatcher.or(barMatcher)

                assert(matcher.test(<-create Foo()))
                assert(matcher.test(<-create Bar()))
            }

            pub resource Foo {}
            pub resource Bar {}
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("resource matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                let foo <- create Foo()
                let bar <- create Bar()

                let fooMatcher = Test.equal(<-foo)
                let barMatcher = Test.equal(<-bar)

                let matcher = fooMatcher.and(barMatcher)

                assert(matcher.test(<-create Foo()))
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
}

func TestInterpretExpectFunction(t *testing.T) {

	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                Test.expect<String>("hello", Test.equal("hello"))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		require.NoError(t, err)
	})

	t.Run("mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            pub fun main() {
                Test.expect<Int>("string", Test.equal(1))
            }
        `

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
		}

		_, err := executeScript(script, runtimeInterface)
		errors := checker.ExpectCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errors[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errors[1])
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

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

	t.Run("resource with a different resource matcher", func(t *testing.T) {
		t.Parallel()

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

	t.Run("resource with struct matcher", func(t *testing.T) {
		t.Parallel()

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

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

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
}
