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

// This is in order to avoid cyclic import errors with runtime package
package stdlib

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	cdcErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func newTestContractInterpreter(t *testing.T, code string) (*interpreter.Interpreter, error) {
	testFramework := &mockedTestFramework{
		emulatorBackend: func() Blockchain {
			return &mockedBlockchain{}
		},
	}
	return newTestContractInterpreterWithTestFramework(t, code, testFramework)
}

func newTestContractInterpreterWithTestFramework(
	t *testing.T,
	code string,
	testFramework TestFramework,
) (*interpreter.Interpreter, error) {
	program, err := parser.ParseProgram(
		nil,
		[]byte(code),
		parser.Config{},
	)
	require.NoError(t, err)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(AssertFunction)
	baseValueActivation.DeclareValue(PanicFunction)

	checker, err := sema.NewChecker(
		program,
		utils.TestLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return baseValueActivation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
			ImportHandler: func(
				checker *sema.Checker,
				importedLocation common.Location,
				importRange ast.Range,
			) (
				sema.Import,
				error,
			) {
				if importedLocation == TestContractLocation {
					return sema.ElaborationImport{
						Elaboration: GetTestContractType().Checker.Elaboration,
					}, nil
				}

				return nil, errors.New("invalid import")
			},
			ContractValueHandler: TestCheckerContractValueHandler,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	if err != nil {
		return nil, err
	}

	storage := interpreter.NewInMemoryStorage(nil)

	var uuid uint64 = 0

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, AssertFunction)
	interpreter.Declare(baseActivation, PanicFunction)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
			ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				if location == TestContractLocation {
					program := interpreter.ProgramFromChecker(GetTestContractType().Checker)
					subInterpreter, err := inter.NewSubInterpreter(program, location)
					if err != nil {
						panic(err)
					}
					return interpreter.InterpreterImport{
						Interpreter: subInterpreter,
					}
				}

				return nil
			},
			ContractValueHandler: NewTestInterpreterContractValueHandler(testFramework),
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter, nil
}

func TestTestNewMatcher(t *testing.T) {
	t.Parallel()

	t.Run("custom matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let matcher = Test.newMatcher(fun (_ value: AnyStruct): Bool {
                     if !value.getType().isSubtype(of: Type<Int>()) {
                        return false
                    }

                    return (value as! Int) > 5
                })

                return matcher.test(8)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("custom matcher primitive type", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {

               let matcher = Test.newMatcher(fun (_ value: Int): Bool {
                    return value == 7
               })

               return matcher.test(7)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("custom matcher invalid type usage", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {

               let matcher = Test.newMatcher(fun (_ value: Int): Bool {
                    return (value + 7) == 4
               })

               // Invoke with an incorrect type
               matcher.test("Hello")
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &interpreter.TypeMismatchError{})
	})

	t.Run("custom resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {

               let matcher = Test.newMatcher(fun (_ value: &Foo): Bool {
                   return value.a == 4
               })

               let f <-create Foo(4)

               let res = matcher.test(&f as &Foo)

               destroy f

               return res
           }

           access(all)
           resource Foo {

               access(all)
               let a: Int

               init(_ a: Int) {
                   self.a = a
               }
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("custom resource matcher invalid type", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {

               let matcher = Test.newMatcher(fun (_ value: @Foo): Bool {
                    destroy value
                    return true
               })
           }

           access(all)
           resource Foo {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("custom matcher with explicit type", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {

               let matcher = Test.newMatcher<Int>(fun (_ value: Int): Bool {
                    return value == 7
               })

               return matcher.test(7)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("custom matcher with mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {

               let matcher = Test.newMatcher<String>(fun (_ value: Int): Bool {
                    return value == 7
               })
           }
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("combined matcher mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {

               let matcher1 = Test.newMatcher(fun (_ value: Int): Bool {
                    return (value + 5) == 10
               })

               let matcher2 = Test.newMatcher(fun (_ value: String): Bool {
                    return value.length == 10
               })

               let matcher3 = matcher1.and(matcher2)

               // Invoke with a type that matches to only one matcher
               matcher3.test(5)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &interpreter.TypeMismatchError{})
	})
}

func TestTestEqualMatcher(t *testing.T) {

	t.Parallel()

	t.Run("equal matcher with primitive", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.equal(1)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("use equal matcher with primitive", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let matcher = Test.equal(1)
               return matcher.test(1)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("equal matcher with struct", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let f = Foo()
               let matcher = Test.equal(f)
               return matcher.test(f)
           }

           access(all)
           struct Foo {}
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("equal matcher with resource", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let f <- create Foo()
               let matcher = Test.equal(<-f)
               return matcher.test(<- create Foo())
           }

           access(all)
           resource Foo {}
        `

		_, err := newTestContractInterpreter(t, script)
		require.Error(t, err)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               let matcher = Test.equal<String>("hello")
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("with incorrect types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               let matcher = Test.equal<String>(1)
           }
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let one = Test.equal(1)
               let two = Test.equal(2)

               let oneOrTwo = one.or(two)

               return oneOrTwo.test(1)
                   && oneOrTwo.test(2)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("matcher or fail", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let one = Test.equal(1)
               let two = Test.equal(2)

               let oneOrTwo = one.or(two)

               return oneOrTwo.test(3)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let one = Test.equal(1)
               let two = Test.equal(2)

               let oneAndTwo = one.and(two)

               return oneAndTwo.test(1)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher not", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let one = Test.equal(1)

                let notOne = Test.not(one)

                return notOne.test(2)
            }

            access(all)
            fun testNoMatch(): Bool {
                let one = Test.equal(1)

                let notOne = Test.not(one)

                return notOne.test(1)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("chained matchers", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let one = Test.equal(1)
               let two = Test.equal(2)
               let three = Test.equal(3)

               let oneOrTwoOrThree = one.or(two).or(three)

               return oneOrTwoOrThree.test(3)
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)
	})

	t.Run("resource matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.or(barMatcher)

               return matcher.test(<-create Foo())
                   && matcher.test(<-create Bar())
           }

           access(all)
           resource Foo {}
           access(all)
           resource Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 4)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[3])
	})

	t.Run("resource matcher and", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.and(barMatcher)

               return matcher.test(<-create Foo())
           }

           access(all)
           resource Foo {}
           access(all)
           resource Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 3)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})
}

func TestAssertEqual(t *testing.T) {

	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                Test.assertEqual("this string", "this string")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                Test.assertEqual(15, 21)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"assertion failed: not equal: expected: 15, actual: 21",
		)
	})

	t.Run("fail with value equality on optional type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let expected: Int = 15
                let actual: Int? = 15
                Test.assertEqual(expected, actual)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"assertion failed: not equal types: expected: Int, actual: Int?",
		)
	})

	t.Run("different types", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                Test.assertEqual(true, 1)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"assertion failed: not equal types: expected: Bool, actual: Int",
		)
	})

	t.Run("address with address", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testEqual() {
                let expected = Address(0xf8d6e0586b0a20c7)
                let actual = Address(0xf8d6e0586b0a20c7)
                Test.assertEqual(expected, actual)
            }

            access(all)
            fun testNotEqual() {
                let expected = Address(0xf8d6e0586b0a20c7)
                let actual = Address(0xee82856bf20e2aa6)
                Test.assertEqual(expected, actual)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testEqual")
		require.NoError(t, err)

		_, err = inter.Invoke("testNotEqual")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"not equal: expected: 0xf8d6e0586b0a20c7, actual: 0xee82856bf20e2aa6",
		)
	})

	t.Run("struct with struct", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            struct Foo {

                access(all)
                let answer: Int

                init(answer: Int) {
                    self.answer = answer
                }
            }

            access(all)
            fun testEqual() {
                let expected = Foo(answer: 42)
                let actual = Foo(answer: 42)
                Test.assertEqual(expected, actual)
            }

            access(all)
            fun testNotEqual() {
                let expected = Foo(answer: 42)
                let actual = Foo(answer: 420)
                Test.assertEqual(expected, actual)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testEqual")
		require.NoError(t, err)

		_, err = inter.Invoke("testNotEqual")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"not equal: expected: S.test.Foo(answer: 42), actual: S.test.Foo(answer: 420)",
		)
	})

	t.Run("array with array", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testEqual() {
                let expected = [1, 2, 3]
                let actual = [1, 2, 3]
                Test.assertEqual(expected, actual)
            }

            access(all)
            fun testNotEqual() {
                let expected = [1, 2, 3]
                let actual = [1, 2]
                Test.assertEqual(expected, actual)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testEqual")
		require.NoError(t, err)

		_, err = inter.Invoke("testNotEqual")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"not equal: expected: [1, 2, 3], actual: [1, 2]",
		)
	})

	t.Run("dictionary with dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testEqual() {
                let expected = {1: true, 2: false, 3: true}
                let actual = {1: true, 2: false, 3: true}
                Test.assertEqual(expected, actual)
            }

            access(all)
            fun testNotEqual() {
                let expected = {1: true, 2: false}
                let actual = {1: true, 2: true}
                Test.assertEqual(expected, actual)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testEqual")
		require.NoError(t, err)

		_, err = inter.Invoke("testNotEqual")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
		assert.ErrorContains(
			t,
			err,
			"not equal: expected: {1: true, 2: false}, actual: {2: true, 1: true}",
		)
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let f1 <- create Foo()
                let f2 <- create Foo()
                Test.assertEqual(<-f1, <-f2)
            }

            access(all)
            resource Foo {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("resource with struct matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo <- create Foo()
                let bar = Bar()
                Test.assertEqual(<-foo, bar)
            }

            access(all)
            resource Foo {}
            access(all)
            struct Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo()
                let bar <- create Bar()
                Test.assertEqual(foo, <-bar)
            }

            access(all)
            struct Foo {}
            access(all)
            resource Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestTestBeSucceededMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beSucceeded with ScriptResult", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let successful = Test.beSucceeded()

                let scriptResult = Test.ScriptResult(
                    status: Test.ResultStatus.succeeded,
                    returnValue: 42,
                    error: nil
                )

                return successful.test(scriptResult)
            }

            access(all)
            fun testNoMatch(): Bool {
                let successful = Test.beSucceeded()

                let scriptResult = Test.ScriptResult(
                    status: Test.ResultStatus.failed,
                    returnValue: nil,
                    error: Test.Error("Exceeding limit")
                )

                return successful.test(scriptResult)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beSucceeded with TransactionResult", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let successful = Test.beSucceeded()

                let transactionResult = Test.TransactionResult(
                    status: Test.ResultStatus.succeeded,
                    error: nil
                )

                return successful.test(transactionResult)
            }

            access(all)
            fun testNoMatch(): Bool {
                let successful = Test.beSucceeded()

                let transactionResult = Test.TransactionResult(
                    status: Test.ResultStatus.failed,
                    error: Test.Error("Exceeded Limit")
                )

                return successful.test(transactionResult)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beSucceeded with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let successful = Test.beSucceeded()

                return successful.test("hello")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
	})
}

func TestTestBeFailedMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beFailed with ScriptResult", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let failed = Test.beFailed()

                let scriptResult = Test.ScriptResult(
                    status: Test.ResultStatus.failed,
                    returnValue: nil,
                    error: Test.Error("Exceeding limit")
                )

                return failed.test(scriptResult)
            }

            access(all)
            fun testNoMatch(): Bool {
                let failed = Test.beFailed()

                let scriptResult = Test.ScriptResult(
                    status: Test.ResultStatus.succeeded,
                    returnValue: 42,
                    error: nil
                )

                return failed.test(scriptResult)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beFailed with TransactionResult", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let failed = Test.beFailed()

                let transactionResult = Test.TransactionResult(
                    status: Test.ResultStatus.failed,
                    error: Test.Error("Exceeding limit")
                )

                return failed.test(transactionResult)
            }

            access(all)
            fun testNoMatch(): Bool {
                let failed = Test.beFailed()

                let transactionResult = Test.TransactionResult(
                    status: Test.ResultStatus.succeeded,
                    error: nil
                )

                return failed.test(transactionResult)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beFailed with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let failed = Test.beFailed()

                return failed.test([])
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
	})
}

func TestTestAssertErrorMatcher(t *testing.T) {

	t.Parallel()

	t.Run("with ScriptResult", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun testMatch() {
                let result = Test.ScriptResult(
                    status: Test.ResultStatus.failed,
                    returnValue: nil,
                    error: Test.Error("computation exceeding limit")
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }

            access(all)
            fun testNoMatch() {
                let result = Test.ScriptResult(
                    status: Test.ResultStatus.failed,
                    returnValue: nil,
                    error: Test.Error("computation exceeding memory")
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }

            access(all)
            fun testNoError() {
                let result = Test.ScriptResult(
                    status: Test.ResultStatus.succeeded,
                    returnValue: 42,
                    error: nil
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testMatch")
		require.NoError(t, err)

		_, err = inter.Invoke("testNoMatch")
		require.Error(t, err)
		assert.ErrorContains(t, err, "the error message did not contain the given sub-string")

		_, err = inter.Invoke("testNoError")
		require.Error(t, err)
		assert.ErrorContains(t, err, "no error was found")
	})

	t.Run("with TransactionResult", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun testMatch() {
                let result = Test.TransactionResult(
                    status: Test.ResultStatus.failed,
                    error: Test.Error("computation exceeding limit")
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }

            access(all)
            fun testNoMatch() {
                let result = Test.TransactionResult(
                    status: Test.ResultStatus.failed,
                    error: Test.Error("computation exceeding memory")
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }

            access(all)
            fun testNoError() {
                let result = Test.TransactionResult(
                    status: Test.ResultStatus.succeeded,
                    error: nil
                )

                Test.assertError(result, errorMessage: "exceeding limit")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("testMatch")
		require.NoError(t, err)

		_, err = inter.Invoke("testNoMatch")
		require.Error(t, err)
		assert.ErrorContains(t, err, "the error message did not contain the given sub-string")

		_, err = inter.Invoke("testNoError")
		require.Error(t, err)
		assert.ErrorContains(t, err, "no error was found")
	})
}

func TestTestBeNilMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beNil", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let isNil = Test.beNil()

                return isNil.test(nil)
            }

            access(all)
            fun testNoMatch(): Bool {
                let isNil = Test.beNil()

                return isNil.test([1, 2])
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})
}

func TestTestBeEmptyMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beEmpty with Array", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let emptyArray = Test.beEmpty()

                return emptyArray.test([])
            }

            access(all)
            fun testNoMatch(): Bool {
                let emptyArray = Test.beEmpty()

                return emptyArray.test([42, 23, 31])
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beEmpty with Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let emptyDict = Test.beEmpty()
                let dict: {Bool: Int} = {}

                return emptyDict.test(dict)
            }

            access(all)
            fun testNoMatch(): Bool {
                let emptyDict = Test.beEmpty()
                let dict: {Bool: Int} = {true: 1, false: 0}

                return emptyDict.test(dict)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beEmpty with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let emptyDict = Test.beEmpty()

                return emptyDict.test("empty")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &cdcErrors.DefaultUserError{})
		assert.ErrorContains(t, err, "expected Array or Dictionary argument")
	})
}

func TestTestHaveElementCountMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher haveElementCount with Array", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let hasThreeElements = Test.haveElementCount(3)

                return hasThreeElements.test([42, 23, 31])
            }

            access(all)
            fun testNoMatch(): Bool {
                let hasThreeElements = Test.haveElementCount(3)

                return hasThreeElements.test([42])
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher haveElementCount with Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let hasTwoElements = Test.haveElementCount(2)
                let dict: {Bool: Int} = {true: 1, false: 0}

                return hasTwoElements.test(dict)
            }

            access(all)
            fun testNoMatch(): Bool {
                let hasTwoElements = Test.haveElementCount(2)
                let dict: {Bool: Int} = {}

                return hasTwoElements.test(dict)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher haveElementCount with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let hasTwoElements = Test.haveElementCount(2)

                return hasTwoElements.test("two")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &cdcErrors.DefaultUserError{})
		assert.ErrorContains(t, err, "expected Array or Dictionary argument")
	})
}

func TestTestContainMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher contain with Array", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let containsTwenty = Test.contain(20)

                return containsTwenty.test([42, 20, 31])
            }

            access(all)
            fun testNoMatch(): Bool {
                let containsTwenty = Test.contain(20)

                return containsTwenty.test([42])
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher contain with Dictionary", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let containsFalse = Test.contain(false)
                let dict: {Bool: Int} = {true: 1, false: 0}

                return containsFalse.test(dict)
            }

            access(all)
            fun testNoMatch(): Bool {
                let containsFive = Test.contain(5)
                let dict: {Int: Bool} = {1: true, 0: false}

                return containsFive.test(dict)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher contain with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let containsFalse = Test.contain(false)

                return containsFalse.test("false")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &cdcErrors.DefaultUserError{})
		assert.ErrorContains(t, err, "expected Array or Dictionary argument")
	})
}

func TestTestBeGreaterThanMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beGreaterThan", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let greaterThanFive = Test.beGreaterThan(5)

                return greaterThanFive.test(7)
            }

            access(all)
            fun testNoMatch(): Bool {
                let greaterThanFive = Test.beGreaterThan(5)

                return greaterThanFive.test(2)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beGreaterThan with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let greaterThanFive = Test.beGreaterThan(5)

                return greaterThanFive.test("7")
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
	})
}

func TestTestBeLessThanMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beLessThan", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun testMatch(): Bool {
                let lessThanSeven = Test.beLessThan(7)

                return lessThanSeven.test(5)
            }

            access(all)
            fun testNoMatch(): Bool {
                let lessThanSeven = Test.beLessThan(7)

                return lessThanSeven.test(9)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		result, err := inter.Invoke("testMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.TrueValue, result)

		result, err = inter.Invoke("testNoMatch")
		require.NoError(t, err)
		assert.Equal(t, interpreter.FalseValue, result)
	})

	t.Run("matcher beLessThan with type mismatch", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test(): Bool {
                let lessThanSeven = Test.beLessThan(7)

                return lessThanSeven.test(true)
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
	})
}

func TestTestExpect(t *testing.T) {

	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.expect("this string", Test.equal("this string"))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.expect("this string", Test.equal("other string"))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)

		assertionErr := &AssertionError{}
		assert.ErrorAs(t, err, assertionErr)
		assert.Equal(t, "given value is: \"this string\"", assertionErr.Message)
		assert.Equal(t, "test", assertionErr.LocationRange.Location.String())
		assert.Equal(t, 6, assertionErr.LocationRange.StartPosition().Line)
	})

	t.Run("different types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.expect("string", Test.equal(1))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &interpreter.TypeMismatchError{})
	})

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.expect<String>("hello", Test.equal("hello"))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               Test.expect<Int>("string", Test.equal(1))
           }
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               let f1 <- create Foo()
               let f2 <- create Foo()
               Test.expect(<-f1, Test.equal(<-f2))
           }

           access(all)
           resource Foo {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("resource with struct matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               let foo <- create Foo()
               let bar = Bar()
               Test.expect(<-foo, Test.equal(bar))
           }

           access(all)
           resource Foo {}
           access(all)
           struct Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all)
           fun test() {
               let foo = Foo()
               let bar <- create Bar()
               Test.expect(foo, Test.equal(<-bar))
           }

           access(all)
           struct Foo {}
           access(all)
           resource Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestTestExpectFailure(t *testing.T) {

	t.Parallel()

	t.Run("expect failure with no failure found", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo(answer: 42)
                Test.expectFailure(fun(): Void {
                    foo.correctAnswer(42)
                }, errorMessageSubstring: "wrong answer!")
            }

            access(all)
            struct Foo {
                access(self) let answer: UInt8

                init(answer: UInt8) {
                    self.answer = answer
                }

                access(all)
                fun correctAnswer(_ input: UInt8): Bool {
                    if self.answer != input {
                        panic("wrong answer!")
                    }
                    return true
                }
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorContains(
			t,
			err,
			"Expected a failure, but found none.",
		)
	})

	t.Run("expect failure with matching error message", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo(answer: 42)
                Test.expectFailure(fun(): Void {
                    foo.correctAnswer(43)
                }, errorMessageSubstring: "wrong answer!")
            }

            access(all)
            struct Foo {
                access(self) let answer: UInt8

                init(answer: UInt8) {
                    self.answer = answer
                }

                access(all)
                fun correctAnswer(_ input: UInt8): Bool {
                    if self.answer != input {
                        panic("wrong answer!")
                    }
                    return true
                }
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("expect failure with mismatching error message", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo(answer: 42)
                Test.expectFailure(fun(): Void {
                    foo.correctAnswer(43)
                }, errorMessageSubstring: "what is wrong?")
            }

            access(all)
            struct Foo {
                access(self) let answer: UInt8

                init(answer: UInt8) {
                    self.answer = answer
                }

                access(all)
                fun correctAnswer(_ input: UInt8): Bool {
                    if self.answer != input {
                        panic("wrong answer!")
                    }
                    return true
                }
            }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorContains(
			t,
			err,
			"Expected error message to include: \"what is wrong?\".",
		)
	})

	t.Run("expect failure with wrong function signature", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo(answer: 42)
                Test.expectFailure(fun(answer: UInt64): Foo {
                    foo.correctAnswer(42)
                    return foo
                }, errorMessageSubstring: "wrong answer")
            }

            access(all)
            struct Foo {
                access(self) let answer: UInt8

                init(answer: UInt8) {
                    self.answer = answer
                }

                access(all)
                fun correctAnswer(_ input: UInt8): Bool {
                    if self.answer != input {
                        panic("wrong answer!")
                    }
                    return true
                }
            }
        `

		_, err := newTestContractInterpreter(t, script)
		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("expect failure with wrong error message type", func(t *testing.T) {
		t.Parallel()

		script := `
            import Test

            access(all)
            fun test() {
                let foo = Foo(answer: 42)
                Test.expectFailure(fun(): Void {
                    foo.correctAnswer(42)
                }, errorMessageSubstring: ["wrong answer"])
            }

            access(all)
            struct Foo {
                access(self) let answer: UInt8

                init(answer: UInt8) {
                    self.answer = answer
                }

                access(all)
                fun correctAnswer(_ input: UInt8): Bool {
                    if self.answer != input {
                        panic("wrong answer!")
                    }
                    return true
                }
            }
        `

		_, err := newTestContractInterpreter(t, script)
		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestBlockchain(t *testing.T) {

	t.Parallel()

	t.Run("all events, empty", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let events = Test.events()

                Test.expect(events, Test.beEmpty())
            }
        `

		eventsInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					events: func(inter *interpreter.Interpreter, eventType interpreter.StaticType) interpreter.Value {
						eventsInvoked = true
						assert.Nil(t, eventType)
						return interpreter.NewArrayValue(
							inter,
							interpreter.EmptyLocationRange,
							interpreter.NewVariableSizedStaticType(inter, interpreter.PrimitiveStaticTypeAnyStruct),
							common.Address{},
						)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, eventsInvoked)
	})

	t.Run("typed events, empty", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            struct Foo {}

            access(all)
            fun test() {
                // 'Foo' is not an event-type.
                // But we just need to test the API, so it doesn't really matter.
                let typ = Type<Foo>()

                let events = Test.eventsOfType(typ)

                Test.expect(events, Test.beEmpty())
            }
        `

		eventsInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					events: func(inter *interpreter.Interpreter, eventType interpreter.StaticType) interpreter.Value {
						eventsInvoked = true
						assert.NotNil(t, eventType)

						require.IsType(t, &interpreter.CompositeStaticType{}, eventType)
						compositeType := eventType.(*interpreter.CompositeStaticType)
						assert.Equal(t, "Foo", compositeType.QualifiedIdentifier)

						return interpreter.NewArrayValue(
							inter,
							interpreter.EmptyLocationRange,
							interpreter.NewVariableSizedStaticType(inter, interpreter.PrimitiveStaticTypeAnyStruct),
							common.Address{},
						)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, eventsInvoked)
	})

	t.Run("reset", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.reset(to: 5)
            }
        `

		resetInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					reset: func(height uint64) {
						resetInvoked = true
						assert.Equal(t, uint64(5), height)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, resetInvoked)
	})

	t.Run("reset with type mismatch for height", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.reset(to: 5.5)
            }
        `

		resetInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					reset: func(height uint64) {
						resetInvoked = true
					},
				}
			},
		}

		_, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.False(t, resetInvoked)
	})

	t.Run("moveTime forward", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun testMoveForward() {
                // timeDelta is the representation of 35 days,
                // in the form of seconds.
                let timeDelta = Fix64(35 * 24 * 60 * 60)
                Test.moveTime(by: timeDelta + 0.5)
            }
        `

		moveTimeInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					moveTime: func(timeDelta float64) {
						moveTimeInvoked = true
						assert.Equal(t, 3024000.5, timeDelta)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("testMoveForward")
		require.NoError(t, err)

		assert.True(t, moveTimeInvoked)
	})

	t.Run("moveTime backward", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun testMoveBackward() {
                // timeDelta is the representation of 35 days,
                // in the form of seconds.
                let timeDelta = Fix64(35 * 24 * 60 * 60) * -1.0
                Test.moveTime(by: timeDelta)
            }
        `

		moveTimeInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					moveTime: func(timeDelta float64) {
						moveTimeInvoked = true
						assert.Equal(t, -3024000.0, timeDelta)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("testMoveBackward")
		require.NoError(t, err)

		assert.True(t, moveTimeInvoked)
	})

	t.Run("moveTime with invalid time delta", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun testMoveTime() {
                Test.moveTime(by: 3000)
            }
        `

		moveTimeInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					moveTime: func(timeDelta float64) {
						moveTimeInvoked = true
					},
				}
			},
		}

		_, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.False(t, moveTimeInvoked)
	})

	t.Run("createSnapshot", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.createSnapshot(name: "adminCreated")
            }
        `

		createSnapshotInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					createSnapshot: func(name string) error {
						createSnapshotInvoked = true
						assert.Equal(t, "adminCreated", name)

						return nil
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, createSnapshotInvoked)
	})

	t.Run("createSnapshot failure", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.createSnapshot(name: "adminCreated")
            }
        `

		createSnapshotInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					createSnapshot: func(name string) error {
						createSnapshotInvoked = true
						assert.Equal(t, "adminCreated", name)

						return fmt.Errorf("failed to create snapshot: %s", name)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.ErrorContains(t, err, "panic: failed to create snapshot: adminCreated")

		assert.True(t, createSnapshotInvoked)
	})

	t.Run("loadSnapshot", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.createSnapshot(name: "adminCreated")
                Test.loadSnapshot(name: "adminCreated")
            }
        `

		loadSnapshotInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					createSnapshot: func(name string) error {
						assert.Equal(t, "adminCreated", name)

						return nil
					},
					loadSnapshot: func(name string) error {
						loadSnapshotInvoked = true
						assert.Equal(t, "adminCreated", name)

						return nil
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, loadSnapshotInvoked)
	})

	t.Run("loadSnapshot failure", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                Test.createSnapshot(name: "adminCreated")
                Test.loadSnapshot(name: "contractDeployed")
            }
        `

		loadSnapshotInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					createSnapshot: func(name string) error {
						assert.Equal(t, "adminCreated", name)

						return nil
					},
					loadSnapshot: func(name string) error {
						loadSnapshotInvoked = true
						assert.Equal(t, "contractDeployed", name)

						return fmt.Errorf("failed to create snapshot: %s", name)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.ErrorContains(t, err, "panic: failed to create snapshot: contractDeployed")

		assert.True(t, loadSnapshotInvoked)
	})

	t.Run("deployContract", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let err = Test.deployContract(
                    name: "FooContract",
                    path: "./contracts/FooContract.cdc",
                    arguments: ["Hey, there!"]
                )

                Test.expect(err, Test.beNil())
            }
        `

		deployContractInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					deployContract: func(
						inter *interpreter.Interpreter,
						name string,
						path string,
						arguments []interpreter.Value,
					) error {
						deployContractInvoked = true
						assert.Equal(t, "FooContract", name)
						assert.Equal(t, "./contracts/FooContract.cdc", path)
						assert.Equal(t, 1, len(arguments))
						argument := arguments[0].(*interpreter.StringValue)
						assert.Equal(t, "Hey, there!", argument.Str)

						return nil
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, deployContractInvoked)
	})

	t.Run("deployContract with failure", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let err = Test.deployContract(
                    name: "FooContract",
                    path: "./contracts/FooContract.cdc",
                    arguments: ["Hey, there!"]
                )

                Test.assertEqual(
                    "failed to deploy contract: FooContract",
                    err!.message
                )
            }
        `

		deployContractInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					deployContract: func(
						inter *interpreter.Interpreter,
						name string,
						path string,
						arguments []interpreter.Value,
					) error {
						deployContractInvoked = true

						return fmt.Errorf("failed to deploy contract: %s", name)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, deployContractInvoked)
	})

	t.Run("getAccount", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let account = Test.getAccount(0x0000000000000009)
                Test.assertEqual(0x0000000000000009 as Address, account.address)
            }
        `

		getAccountInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					getAccount: func(address interpreter.AddressValue) (*Account, error) {
						getAccountInvoked = true
						assert.Equal(t, "0000000000000009", address.Hex())
						addr := common.Address(address)

						return &Account{
							Address: addr,
							PublicKey: &PublicKey{
								PublicKey: []byte{1, 2, 3},
								SignAlgo:  sema.SignatureAlgorithmECDSA_P256,
							},
						}, nil
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, getAccountInvoked)
	})

	t.Run("getAccount with failure", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let account = Test.getAccount(0x0000000000000009)
            }
        `

		getAccountInvoked := false

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					getAccount: func(address interpreter.AddressValue) (*Account, error) {
						getAccountInvoked = true
						assert.Equal(t, "0000000000000009", address.Hex())

						return nil, fmt.Errorf("failed to retrieve account with address: %s", address)
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorContains(t, err, "account with address: 0x0000000000000009 was not found")

		assert.True(t, getAccountInvoked)
	})

	// TODO: Add more tests for the remaining functions.
}

func TestBlockchainAccount(t *testing.T) {

	t.Parallel()

	t.Run("create account", func(t *testing.T) {
		t.Parallel()

		const script = `
            import Test

            access(all)
            fun test() {
                let account = Test.createAccount()
                assert(account.address == 0x0100000000000000)
            }
        `

		testFramework := &mockedTestFramework{
			emulatorBackend: func() Blockchain {
				return &mockedBlockchain{
					createAccount: func() (*Account, error) {
						return &Account{
							PublicKey: &PublicKey{
								PublicKey: []byte{1, 2, 3},
								SignAlgo:  sema.SignatureAlgorithmECDSA_P256,
							},
							Address: common.Address{1},
						}, nil
					},
				}
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	})
}

type mockedTestFramework struct {
	emulatorBackend func() Blockchain
	readFile        func(s string) (string, error)
}

var _ TestFramework = &mockedTestFramework{}

func (m mockedTestFramework) EmulatorBackend() Blockchain {
	if m.emulatorBackend == nil {
		panic("'NewEmulatorBackend' is not implemented")
	}

	return m.emulatorBackend()
}

func (m mockedTestFramework) ReadFile(fileName string) (string, error) {
	if m.readFile == nil {
		panic("'ReadFile' is not implemented")
	}

	return m.readFile(fileName)
}

// mockedBlockchain is the implementation of `Blockchain` for testing purposes.
type mockedBlockchain struct {
	runScript          func(inter *interpreter.Interpreter, code string, arguments []interpreter.Value)
	createAccount      func() (*Account, error)
	getAccount         func(interpreter.AddressValue) (*Account, error)
	addTransaction     func(inter *interpreter.Interpreter, code string, authorizers []common.Address, signers []*Account, arguments []interpreter.Value) error
	executeTransaction func() *TransactionResult
	commitBlock        func() error
	deployContract     func(inter *interpreter.Interpreter, name string, path string, arguments []interpreter.Value) error
	logs               func() []string
	serviceAccount     func() (*Account, error)
	events             func(inter *interpreter.Interpreter, eventType interpreter.StaticType) interpreter.Value
	reset              func(uint64)
	moveTime           func(float64)
	createSnapshot     func(string) error
	loadSnapshot       func(string) error
}

var _ Blockchain = &mockedBlockchain{}

func (m mockedBlockchain) RunScript(
	inter *interpreter.Interpreter,
	code string,
	arguments []interpreter.Value,
) *ScriptResult {
	if m.runScript == nil {
		panic("'RunScript' is not implemented")
	}

	return m.RunScript(inter, code, arguments)
}

func (m mockedBlockchain) CreateAccount() (*Account, error) {
	if m.createAccount == nil {
		panic("'CreateAccount' is not implemented")
	}

	return m.createAccount()
}

func (m mockedBlockchain) GetAccount(address interpreter.AddressValue) (*Account, error) {
	if m.getAccount == nil {
		panic("'getAccount' is not implemented")
	}

	return m.getAccount(address)
}

func (m mockedBlockchain) AddTransaction(
	inter *interpreter.Interpreter,
	code string,
	authorizers []common.Address,
	signers []*Account,
	arguments []interpreter.Value,
) error {
	if m.addTransaction == nil {
		panic("'AddTransaction' is not implemented")
	}

	return m.addTransaction(inter, code, authorizers, signers, arguments)
}

func (m mockedBlockchain) ExecuteNextTransaction() *TransactionResult {
	if m.executeTransaction == nil {
		panic("'ExecuteNextTransaction' is not implemented")
	}

	return m.executeTransaction()
}

func (m mockedBlockchain) CommitBlock() error {
	if m.commitBlock == nil {
		panic("'CommitBlock' is not implemented")
	}

	return m.commitBlock()
}

func (m mockedBlockchain) DeployContract(
	inter *interpreter.Interpreter,
	name string,
	path string,
	arguments []interpreter.Value,
) error {
	if m.deployContract == nil {
		panic("'DeployContract' is not implemented")
	}

	return m.deployContract(inter, name, path, arguments)
}

func (m mockedBlockchain) Logs() []string {
	if m.logs == nil {
		panic("'Logs' is not implemented")
	}

	return m.logs()
}

func (m mockedBlockchain) ServiceAccount() (*Account, error) {
	if m.serviceAccount == nil {
		panic("'ServiceAccount' is not implemented")
	}

	return m.serviceAccount()
}

func (m mockedBlockchain) Events(
	inter *interpreter.Interpreter,
	eventType interpreter.StaticType,
) interpreter.Value {
	if m.events == nil {
		panic("'Events' is not implemented")
	}

	return m.events(inter, eventType)
}

func (m mockedBlockchain) Reset(height uint64) {
	if m.reset == nil {
		panic("'Reset' is not implemented")
	}

	m.reset(height)
}

func (m mockedBlockchain) MoveTime(timeDelta float64) {
	if m.moveTime == nil {
		panic("'SetTimestamp' is not implemented")
	}

	m.moveTime(timeDelta)
}

func (m mockedBlockchain) CreateSnapshot(name string) error {
	if m.createSnapshot == nil {
		panic("'CreateSnapshot' is not implemented")
	}

	return m.createSnapshot(name)
}

func (m mockedBlockchain) LoadSnapshot(name string) error {
	if m.loadSnapshot == nil {
		panic("'LoadSnapshot' is not implemented")
	}

	return m.loadSnapshot(name)
}
