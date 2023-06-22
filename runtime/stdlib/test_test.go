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
			ContractValueHandler: NewTestInterpreterContractValueHandler(nil),
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

            access(all) fun test(): Bool {
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

           access(all) fun test(): Bool {

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

           access(all) fun test() {

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

           access(all) fun test(): Bool {

               let matcher = Test.newMatcher(fun (_ value: &Foo): Bool {
                   return value.a == 4
               })

               let f <-create Foo(4)

               let res = matcher.test(&f as &Foo)

               destroy f

               return res
           }

           access(all) resource Foo {
               access(all) let a: Int

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

	       access(all) fun test() {

	           let matcher = Test.newMatcher(fun (_ value: @Foo): Bool {
	                destroy value
	                return true
	           })
	       }

	       access(all) resource Foo {}
	    `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("custom matcher with explicit type", func(t *testing.T) {
		t.Parallel()

		script := `
	       import Test

	       access(all) fun test(): Bool {

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

	       access(all) fun test() {

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

	       access(all) fun test() {

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

           access(all) fun test() {
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

           access(all) fun test(): Bool {
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

           access(all) fun test(): Bool {
               let f = Foo()
               let matcher = Test.equal(f)
               return matcher.test(f)
           }

           access(all) struct Foo {}
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

           access(all) fun test(): Bool {
               let f <- create Foo()
               let matcher = Test.equal(<-f)
               return matcher.test(<- create Foo())
           }

           access(all) resource Foo {}
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

           access(all) fun test() {
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

           access(all) fun test() {
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

           access(all) fun test(): Bool {
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

           access(all) fun test(): Bool {
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

           access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let one = Test.equal(1)

		        let notOne = Test.not(one)

		        return notOne.test(2)
		    }

		    access(all) fun testNoMatch(): Bool {
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

           access(all) fun test(): Bool {
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

           access(all) fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.or(barMatcher)

               return matcher.test(<-create Foo())
                   && matcher.test(<-create Bar())
           }

           access(all) resource Foo {}
           access(all) resource Bar {}
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

           access(all) fun test(): Bool {
               let foo <- create Foo()
               let bar <- create Bar()

               let fooMatcher = Test.equal(<-foo)
               let barMatcher = Test.equal(<-bar)

               let matcher = fooMatcher.and(barMatcher)

               return matcher.test(<-create Foo())
           }

           access(all) resource Foo {}
           access(all) resource Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 3)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
	})
}

func TestTestBeSucceededMatcher(t *testing.T) {

	t.Parallel()

	t.Run("matcher beSucceeded with ScriptResult", func(t *testing.T) {
		t.Parallel()

		script := `
		    import Test

		    access(all) fun testMatch(): Bool {
		        let successful = Test.beSucceeded()

		        let scriptResult = Test.ScriptResult(
		            status: Test.ResultStatus.succeeded,
		            returnValue: 42,
		            error: nil
		        )

		        return successful.test(scriptResult)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let successful = Test.beSucceeded()

		        let transactionResult = Test.TransactionResult(
		            status: Test.ResultStatus.succeeded,
		            error: nil
		        )

		        return successful.test(transactionResult)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let failed = Test.beFailed()

		        let scriptResult = Test.ScriptResult(
		            status: Test.ResultStatus.failed,
		            returnValue: nil,
		            error: Test.Error("Exceeding limit")
		        )

		        return failed.test(scriptResult)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let failed = Test.beFailed()

		        let transactionResult = Test.TransactionResult(
		            status: Test.ResultStatus.failed,
		            error: Test.Error("Exceeding limit")
		        )

		        return failed.test(transactionResult)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let isNil = Test.beNil()

		        return isNil.test(nil)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let emptyArray = Test.beEmpty()

		        return emptyArray.test([])
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let emptyDict = Test.beEmpty()
		        let dict: {Bool: Int} = {}

		        return emptyDict.test(dict)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let hasThreeElements = Test.haveElementCount(3)

		        return hasThreeElements.test([42, 23, 31])
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let hasTwoElements = Test.haveElementCount(2)
		        let dict: {Bool: Int} = {true: 1, false: 0}

		        return hasTwoElements.test(dict)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let containsTwenty = Test.contain(20)

		        return containsTwenty.test([42, 20, 31])
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let containsFalse = Test.contain(false)
		        let dict: {Bool: Int} = {true: 1, false: 0}

		        return containsFalse.test(dict)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let greaterThanFive = Test.beGreaterThan(5)

		        return greaterThanFive.test(7)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

		    access(all) fun testMatch(): Bool {
		        let lessThanSeven = Test.beLessThan(7)

		        return lessThanSeven.test(5)
		    }

		    access(all) fun testNoMatch(): Bool {
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

		    access(all) fun test(): Bool {
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

           access(all) fun test() {
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

           access(all) fun test() {
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

           access(all) fun test() {
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

           access(all) fun test() {
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

           access(all) fun test() {
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

           access(all) fun test() {
               let f1 <- create Foo()
               let f2 <- create Foo()
               Test.expect(<-f1, Test.equal(<-f2))
           }

           access(all) resource Foo {}
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

           access(all) fun test() {
               let foo <- create Foo()
               let bar = Bar()
               Test.expect(<-foo, Test.equal(bar))
           }

           access(all) resource Foo {}
           access(all) struct Bar {}
        `

		_, err := newTestContractInterpreter(t, script)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct with resource matcher", func(t *testing.T) {
		t.Parallel()

		script := `
           import Test

           access(all) fun test() {
               let foo = Foo()
               let bar <- create Bar()
               Test.expect(foo, Test.equal(<-bar))
           }

           access(all) struct Foo {}
           access(all) resource Bar {}
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

		    access(all) fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(42)
		        }, errorMessageSubstring: "wrong answer!")
		    }

		    access(all) struct Foo {
		        access(self) let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        access(all) fun correctAnswer(_ input: UInt8): Bool {
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

		    access(all) fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(43)
		        }, errorMessageSubstring: "wrong answer!")
		    }

		    access(all) struct Foo {
		        access(self) let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        access(all) fun correctAnswer(_ input: UInt8): Bool {
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

		    access(all) fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(43)
		        }, errorMessageSubstring: "what is wrong?")
		    }

		    access(all) struct Foo {
		        access(self) let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        access(all) fun correctAnswer(_ input: UInt8): Bool {
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

		    access(all) fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(answer: UInt64): Foo {
		            foo.correctAnswer(42)
		            return foo
		        }, errorMessageSubstring: "wrong answer")
		    }

		    access(all) struct Foo {
		        access(self) let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        access(all) fun correctAnswer(_ input: UInt8): Bool {
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

		    access(all) fun test() {
		        let foo = Foo(answer: 42)
		        Test.expectFailure(fun(): Void {
		            foo.correctAnswer(42)
		        }, errorMessageSubstring: ["wrong answer"])
		    }

		    access(all) struct Foo {
		        access(self) let answer: UInt8

		        init(answer: UInt8) {
		            self.answer = answer
		        }

		        access(all) fun correctAnswer(_ input: UInt8): Bool {
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
