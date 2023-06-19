/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"errors"
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
	return newTestContractInterpreterWithTestFramework(t, code, nil)
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

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(AssertFunction)
	activation.DeclareValue(PanicFunction)

	checker, err := sema.NewChecker(
		program,
		utils.TestLocation,
		nil,
		&sema.Config{
			BaseValueActivation: activation,
			AccessCheckMode:     sema.AccessCheckModeStrict,
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

	storage := newUnmeteredInMemoryStorage()

	var uuid uint64 = 0

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, AssertFunction)
	interpreter.Declare(baseActivation, PanicFunction)

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage:        storage,
			BaseActivation: baseActivation,
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

            pub fun test(): Bool {
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

           pub fun test(): Bool {

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

           pub fun test() {

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

           pub fun test(): Bool {

               let matcher = Test.newMatcher(fun (_ value: &Foo): Bool {
                   return value.a == 4
               })

               let f <-create Foo(4)

               let res = matcher.test(&f as &Foo)

               destroy f

               return res
           }

           pub resource Foo {
               pub let a: Int

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

	       pub fun test() {

	           let matcher = Test.newMatcher(fun (_ value: @Foo): Bool {
	                destroy value
	                return true
	           })
	       }

	       pub resource Foo {}
	    `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("custom matcher with explicit type", func(t *testing.T) {
		t.Parallel()

		script := `
	       import Test

	       pub fun test(): Bool {

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

	       pub fun test() {

	           let matcher = Test.newMatcher<String>(fun (_ value: Int): Bool {
	                return value == 7
	           })
	       }
	    `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("combined matcher mismatching types", func(t *testing.T) {
		t.Parallel()

		script := `
	       import Test

	       pub fun test() {

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

           pub fun test(): Bool {
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

           pub fun test(): Bool {
               let f = Foo()
               let matcher = Test.equal(f)
               return matcher.test(f)
           }

           pub struct Foo {}
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

           pub fun test(): Bool {
               let f <- create Foo()
               let matcher = Test.equal(<-f)
               return matcher.test(<- create Foo())
           }

           pub resource Foo {}
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

           pub fun test() {
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

           pub fun test() {
               let matcher = Test.equal<String>(1)
           }
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("matcher or", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           pub fun test(): Bool {
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

           pub fun test(): Bool {
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

           pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let one = Test.equal(1)

		        let notOne = Test.not(one)

		        return notOne.test(2)
		    }

		    pub fun testNoMatch(): Bool {
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

           pub fun test(): Bool {
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

           pub fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.or(barMatcher)

               return matcher.test(<-create Foo())
                   && matcher.test(<-create Bar())
           }

           pub resource Foo {}
           pub resource Bar {}
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

           pub fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.and(barMatcher)

               return matcher.test(<-create Foo())
           }

           pub resource Foo {}
           pub resource Bar {}
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

		    pub fun test() {
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

		    pub fun test() {
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

	t.Run("different types", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun test() {
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
			"assertion failed: not equal: expected: true, actual: 1",
		)
	})

	t.Run("address with address", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun testEqual() {
		        let expected = Address(0xf8d6e0586b0a20c7)
		        let actual = Address(0xf8d6e0586b0a20c7)
		        Test.assertEqual(expected, actual)
		    }

		    pub fun testNotEqual() {
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

		    pub struct Foo {
		        pub let answer: Int

		        init(answer: Int) {
		            self.answer = answer
		        }
		    }

		    pub fun testEqual() {
		        let expected = Foo(answer: 42)
		        let actual = Foo(answer: 42)
		        Test.assertEqual(expected, actual)
		    }

		    pub fun testNotEqual() {
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

		    pub fun testEqual() {
		        let expected = [1, 2, 3]
		        let actual = [1, 2, 3]
		        Test.assertEqual(expected, actual)
		    }

		    pub fun testNotEqual() {
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

		    pub fun testEqual() {
		        let expected = {1: true, 2: false, 3: true}
		        let actual = {1: true, 2: false, 3: true}
		        Test.assertEqual(expected, actual)
		    }

		    pub fun testNotEqual() {
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
			"not equal: expected: {2: false, 1: true}, actual: {2: true, 1: true}",
		)
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun test() {
		        let f1 <- create Foo()
		        let f2 <- create Foo()
		        Test.assertEqual(<-f1, <-f2)
		    }

		    pub resource Foo {}
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

		    pub fun test() {
		        let foo <- create Foo()
		        let bar = Bar()
		        Test.assertEqual(<-foo, bar)
		    }

		    pub resource Foo {}
		    pub struct Bar {}
		`

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun test() {
		        let foo = Foo()
		        let bar <- create Bar()
		        Test.expect(foo, Test.equal(<-bar))
		    }

		    pub struct Foo {}
		    pub resource Bar {}
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

		    pub fun testMatch(): Bool {
		        let successful = Test.beSucceeded()

		        let scriptResult = Test.ScriptResult(
		            status: Test.ResultStatus.succeeded,
		            returnValue: 42,
		            error: nil
		        )

		        return successful.test(scriptResult)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let successful = Test.beSucceeded()

		        let transactionResult = Test.TransactionResult(
		            status: Test.ResultStatus.succeeded,
		            error: nil
		        )

		        return successful.test(transactionResult)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let failed = Test.beFailed()

		        let scriptResult = Test.ScriptResult(
		            status: Test.ResultStatus.failed,
		            returnValue: nil,
		            error: Test.Error("Exceeding limit")
		        )

		        return failed.test(scriptResult)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let failed = Test.beFailed()

		        let transactionResult = Test.TransactionResult(
		            status: Test.ResultStatus.failed,
		            error: Test.Error("Exceeding limit")
		        )

		        return failed.test(transactionResult)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

func TestTestBeNilMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beNil", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun testMatch(): Bool {
		        let isNil = Test.beNil()

		        return isNil.test(nil)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let emptyArray = Test.beEmpty()

		        return emptyArray.test([])
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let emptyDict = Test.beEmpty()
		        let dict: {Bool: Int} = {}

		        return emptyDict.test(dict)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let hasThreeElements = Test.haveElementCount(3)

		        return hasThreeElements.test([42, 23, 31])
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let hasTwoElements = Test.haveElementCount(2)
		        let dict: {Bool: Int} = {true: 1, false: 0}

		        return hasTwoElements.test(dict)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let containsTwenty = Test.contain(20)

		        return containsTwenty.test([42, 20, 31])
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun testMatch(): Bool {
		        let containsFalse = Test.contain(false)
		        let dict: {Bool: Int} = {true: 1, false: 0}

		        return containsFalse.test(dict)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let greaterThanFive = Test.beGreaterThan(5)

		        return greaterThanFive.test(7)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

		    pub fun testMatch(): Bool {
		        let lessThanSeven = Test.beLessThan(7)

		        return lessThanSeven.test(5)
		    }

		    pub fun testNoMatch(): Bool {
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

		    pub fun test(): Bool {
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

           pub fun test() {
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

           pub fun test() {
               Test.expect("this string", Test.equal("other string"))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
	})

	t.Run("different types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           pub fun test() {
               Test.expect("string", Test.equal(1))
           }
        `

		inter, err := newTestContractInterpreter(t, script)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.Error(t, err)
		assert.ErrorAs(t, err, &AssertionError{})
	})

	t.Run("with explicit types", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           pub fun test() {
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

           pub fun test() {
               Test.expect<Int>("string", Test.equal(1))
           }
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.TypeParameterTypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("resource with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           pub fun test() {
               let f1 <- create Foo()
               let f2 <- create Foo()
               Test.expect(<-f1, Test.equal(<-f2))
           }

           pub resource Foo {}
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

           pub fun test() {
               let foo <- create Foo()
               let bar = Bar()
               Test.expect(<-foo, Test.equal(bar))
           }

           pub resource Foo {}
           pub struct Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           pub fun test() {
               let foo = Foo()
               let bar <- create Bar()
               Test.expect(foo, Test.equal(<-bar))
           }

           pub struct Foo {}
           pub resource Bar {}
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

		    pub fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(42)
		        }, errorMessageSubstring: "wrong answer!")
		    }

		    pub struct Foo {
		        priv let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        pub fun correctAnswer(_ input: UInt8): Bool {
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

		    pub fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(43)
		        }, errorMessageSubstring: "wrong answer!")
		    }

		    pub struct Foo {
		        priv let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        pub fun correctAnswer(_ input: UInt8): Bool {
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

		    pub fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(43)
		        }, errorMessageSubstring: "what is wrong?")
		    }

		    pub struct Foo {
		        priv let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        pub fun correctAnswer(_ input: UInt8): Bool {
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

		    pub fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(answer: UInt64): Foo {
		            foo.correctAnswer(42)
		            return foo
		        }, errorMessageSubstring: "wrong answer")
		    }

		    pub struct Foo {
		        priv let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        pub fun correctAnswer(_ input: UInt8): Bool {
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

		    pub fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(42)
		        }, errorMessageSubstring: ["wrong answer"])
		    }

		    pub struct Foo {
		        priv let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        pub fun correctAnswer(_ input: UInt8): Bool {
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

		script := `
		    import Test

		    pub fun test(): [AnyStruct] {
		        var blockchain = Test.newEmulatorBlockchain()
		        return blockchain.events()
		    }
		`

		eventsInvoked := false

		testFramework := &mockedTestFramework{
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

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, eventsInvoked)
	})

	t.Run("typed events, empty", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    pub fun test(): [AnyStruct] {
		        var blockchain = Test.newEmulatorBlockchain()

		        // 'Foo' is not an event-type.
		        // But we just need to test the API, so it doesn't really matter.
		        var typ = Type<Foo>()

		        return blockchain.eventsOfType(typ)
		    }

		    pub struct Foo {}
		`

		eventsInvoked := false

		testFramework := &mockedTestFramework{
			events: func(inter *interpreter.Interpreter, eventType interpreter.StaticType) interpreter.Value {
				eventsInvoked = true
				assert.NotNil(t, eventType)

				require.IsType(t, interpreter.CompositeStaticType{}, eventType)
				compositeType := eventType.(interpreter.CompositeStaticType)
				assert.Equal(t, "Foo", compositeType.QualifiedIdentifier)

				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(inter, interpreter.PrimitiveStaticTypeAnyStruct),
					common.Address{},
				)
			},
		}

		inter, err := newTestContractInterpreterWithTestFramework(t, script, testFramework)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)

		assert.True(t, eventsInvoked)
	})

	// TODO: Add more tests for the remaining functions.
}

type mockedTestFramework struct {
	runScript          func(inter *interpreter.Interpreter, code string, arguments []interpreter.Value)
	createAccount      func() (*Account, error)
	addTransaction     func(inter *interpreter.Interpreter, code string, authorizers []common.Address, signers []*Account, arguments []interpreter.Value) error
	executeTransaction func() *TransactionResult
	commitBlock        func() error
	deployContract     func(inter *interpreter.Interpreter, name string, code string, account *Account, arguments []interpreter.Value) error
	readFile           func(s string) (string, error)
	useConfiguration   func(configuration *Configuration)
	stdlibHandler      func() StandardLibraryHandler
	logs               func() []string
	serviceAccount     func() (*Account, error)
	events             func(inter *interpreter.Interpreter, eventType interpreter.StaticType) interpreter.Value
	reset              func()
}

var _ TestFramework = &mockedTestFramework{}

func (m mockedTestFramework) RunScript(
	inter *interpreter.Interpreter,
	code string,
	arguments []interpreter.Value,
) *ScriptResult {
	if m.runScript == nil {
		panic("'RunScript' is not implemented")
	}

	return m.RunScript(inter, code, arguments)
}

func (m mockedTestFramework) CreateAccount() (*Account, error) {
	if m.createAccount == nil {
		panic("'CreateAccount' is not implemented")
	}

	return m.createAccount()
}

func (m mockedTestFramework) AddTransaction(
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

func (m mockedTestFramework) ExecuteNextTransaction() *TransactionResult {
	if m.executeTransaction == nil {
		panic("'ExecuteNextTransaction' is not implemented")
	}

	return m.executeTransaction()
}

func (m mockedTestFramework) CommitBlock() error {
	if m.commitBlock == nil {
		panic("'CommitBlock' is not implemented")
	}

	return m.commitBlock()
}

func (m mockedTestFramework) DeployContract(
	inter *interpreter.Interpreter,
	name string,
	code string,
	account *Account,
	arguments []interpreter.Value,
) error {
	if m.deployContract == nil {
		panic("'DeployContract' is not implemented")
	}

	return m.deployContract(inter, name, code, account, arguments)
}

func (m mockedTestFramework) ReadFile(fileName string) (string, error) {
	if m.readFile == nil {
		panic("'ReadFile' is not implemented")
	}

	return m.readFile(fileName)
}

func (m mockedTestFramework) UseConfiguration(configuration *Configuration) {
	if m.useConfiguration == nil {
		panic("'UseConfiguration' is not implemented")
	}

	m.useConfiguration(configuration)
}

func (m mockedTestFramework) StandardLibraryHandler() StandardLibraryHandler {
	if m.stdlibHandler == nil {
		panic("'StandardLibraryHandler' is not implemented")
	}

	return m.stdlibHandler()
}

func (m mockedTestFramework) Logs() []string {
	if m.logs == nil {
		panic("'Logs' is not implemented")
	}

	return m.logs()
}

func (m mockedTestFramework) ServiceAccount() (*Account, error) {
	if m.serviceAccount == nil {
		panic("'ServiceAccount' is not implemented")
	}

	return m.serviceAccount()
}

func (m mockedTestFramework) Events(
	inter *interpreter.Interpreter,
	eventType interpreter.StaticType,
) interpreter.Value {
	if m.events == nil {
		panic("'Events' is not implemented")
	}

	return m.events(inter, eventType)
}

func (m mockedTestFramework) Reset() {
	if m.reset == nil {
		panic("'Reset' is not implemented")
	}

	m.reset()
}
