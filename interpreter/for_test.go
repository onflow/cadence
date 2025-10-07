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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretForStatement(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Int {
           var sum = 0
           for y in [1, 2, 3, 4] {
               sum = sum + y
           }
           return sum
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)
}

func TestInterpretForStatementWithIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Int {
           var sum = 0
           for x, y in [1, 2, 3, 4] {
               sum = sum + x
           }
           return sum
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretForStatementWithStoredIndex(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Int {
           let arr: [Int] = []
           for x, y in [1, 2, 3, 4] {
               arr.append(x)
           }
           var sum = 0
           for z in arr {
              sum = sum + z
           }
           return sum
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(6),
		value,
	)
}

func TestInterpretForStatementWithReturn(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Int {
           for x in [1, 2, 3, 4, 5] {
               if x > 3 {
                   return x
               }
           }
           return -1
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		value,
	)
}

func TestInterpretForStatementWithContinue(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): [Int] {
           var xs: [Int] = []
           for x in [1, 2, 3, 4, 5] {
               if x <= 3 {
                   continue
               }
               xs.append(x)
           }
           return xs
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t, &interpreter.ArrayValue{}, value)
	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(4),
			interpreter.NewUnmeteredIntValueFromInt64(5),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretForStatementWithBreak(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Int {
           var y = 0
           for x in [1, 2, 3, 4] {
               y = x
               if x > 3 {
                   break
               }
           }
           return y
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(4),
		value,
	)
}

func TestInterpretForStatementEmpty(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): Bool {
           var x = false
           for y in [] {
               x = true
           }
           return x
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.FalseValue,
		value,
	)
}

func TestInterpretForString(t *testing.T) {

	t.Parallel()

	t.Run("basic", func(t *testing.T) {

		inter := parseCheckAndPrepare(t, `
            fun test(): [Character] {
                let characters: [Character] = []
                let hello = "👪❤️"
                for c in hello {
                    characters.append(c)
                }
                return characters
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeCharacter,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredCharacterValue("👪"),
				interpreter.NewUnmeteredCharacterValue("❤️"),
			),
			value,
		)
	})

	t.Run("return", func(t *testing.T) {

		inter := parseCheckAndPrepare(t, `
            fun test(): [Character] {
                let characters: [Character] = []
                let hello = "abc"
                for c in hello {
                    characters.append(c)
                    return characters
                }
                return characters
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeCharacter,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredCharacterValue("a"),
			),
			value,
		)
	})

	t.Run("break", func(t *testing.T) {

		inter := parseCheckAndPrepare(t, `
            fun test(): [Character] {
                let characters: [Character] = []
                let hello = "abc"
                for c in hello {
                    characters.append(c)
                    break
                }
                return characters
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeCharacter,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredCharacterValue("a"),
			),
			value,
		)
	})

}

func TestInterpretForStatementCapturing(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
       fun test(): [Int] {
           let fs: [fun(): Int] = []
           for x in [1, 2, 3] {
               fs.append(fun (): Int {
                   return x
               })
           }

           let values: [Int] = []
           for f in fs {
              values.append(f())
           }
           return values
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	require.IsType(t, value, &interpreter.ArrayValue{})
	arrayValue := value.(*interpreter.ArrayValue)

	AssertValueSlicesEqual(
		t,
		inter,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(2),
			interpreter.NewUnmeteredIntValueFromInt64(3),
		},
		ArrayElements(inter, arrayValue),
	)
}

func TestInterpretEphemeralReferencesInForLoop(t *testing.T) {

	t.Parallel()

	prepare := func(t *testing.T, code string, check func(interpreter.Value)) Invokable {
		valueDeclaration := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"check",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityView,
				[]sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.NewTypeAnnotation(sema.AnyType),
					},
				},
				sema.VoidTypeAnnotation,
			),
			"",
			func(
				context interpreter.NativeFunctionContext,
				locationRange interpreter.LocationRange,
				typeParameterGetter interpreter.TypeParameterGetter,
				receiver interpreter.Value,
				args ...interpreter.Value,
			) interpreter.Value {
				check(args[0])
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			code,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				InterpreterConfig: &interpreter.Config{
					BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
		require.NoError(t, err)

		return inter
	}

	t.Run("Primitive array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                fun main() {
                    let array = ["Hello", "World", "Foo", "Bar"]
                    let arrayRef = &array as &[String]

                    for element in arrayRef {
                        check(element)
                        let e: String = element
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.StringValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 4, checks)
	})

	t.Run("Struct array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let array = [Foo(), Foo()]
                    let arrayRef = &array as &[Foo]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Resource array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                resource Foo {}

                fun main() {
                    let array <- [ <- create Foo(), <- create Foo()]
                    let arrayRef = &array as &[Foo]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element
                    }

                    destroy array
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Moved resource array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                resource Foo {}

                fun main() {
                    let array <- [ <- create Foo(), <- create Foo()]
                    let arrayRef = returnSameRef(&array as &[Foo])
                    let movedArray <- array

                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element
                    }

                    destroy movedArray
                }

                fun returnSameRef(_ ref: &[Foo]): &[Foo] {
                    return ref
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		RequireError(t, err)

		var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)

		require.Equal(t, 0, checks)
	})

	t.Run("Auth ref", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let array = [Foo(), Foo()]
                    let arrayRef = &array as auth(Mutate) &[Foo]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element    // Should be non-auth
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)

				ref := value.(*interpreter.EphemeralReferenceValue)
				require.Equal(t, interpreter.UnauthorizedAccess, ref.Authorization)

				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Optional array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let array: [Foo?] = [Foo(), Foo()]
                    let arrayRef = &array as &[Foo?]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo? = element    // Should be an optional reference
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.SomeValue{}, value)
				someValue := value.(*interpreter.SomeValue)

				require.IsType(t, &interpreter.EphemeralReferenceValue{}, someValue.InnerValue())

				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Nil array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let array: [Foo?] = [nil, nil]
                    let arrayRef = &array as &[Foo?]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo? = element    // Should be an optional reference
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, interpreter.NilValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Reference array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let elementRef = &Foo() as &Foo
                    let array: [&Foo] = [elementRef, elementRef]
                    let arrayRef = &array as &[&Foo]

                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Array of unauthorized references", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let elementRef = &Foo() as &Foo
                    let array: [&Foo] = [elementRef, elementRef]

                    for element in array {
                        check(element)
                        let e: &Foo = element
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Array of authorized references", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {}

                fun main() {
                    let elementRef = &Foo() as auth(Mutate) &Foo
                    let array: [auth(Mutate) &Foo] = [elementRef, elementRef]

                    for element in array {
                        check(element)
                        let e: auth(Mutate) &Foo = element
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)

				ref := value.(*interpreter.EphemeralReferenceValue)
				require.IsType(t, interpreter.EntitlementSetAuthorization{}, ref.Authorization)

				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})

	t.Run("Mutating reference to resource array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                resource Foo {
                    fun sayHello() {}
                }

                fun main() {
                    let array <- [ <- create Foo()]
                    let arrayRef = &array as auth(Mutate) &[Foo]

                    for element in arrayRef {
                        check(element)

                        // Move the actual element
                        // This mutation should fail.
                        let oldElement <- arrayRef.remove(at: 0)

                        // Use the element reference
                        element.sayHello()

                        destroy oldElement
                    }

                    destroy array
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		RequireError(t, err)

		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)

		require.Equal(t, 1, checks)
	})

	t.Run("Mutating reference to struct array", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                struct Foo {
                    fun sayHello() {}
                }

                fun main() {
                    let array = [Foo()]
                    let arrayRef = &array as auth(Mutate) &[Foo]

                    for element in arrayRef {
                        check(element)

                        // Move the actual element
                        let oldElement = arrayRef.remove(at: 0)

                        // Use the element reference
                        element.sayHello()
                    }
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		RequireError(t, err)

		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)

		require.Equal(t, 1, checks)
	})

	t.Run("String ref", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                fun main(): [Character] {
                    let s = "Hello"
                    let sRef = &s as &String
                    let characters: [Character] = []

                    for char in sRef {
                        check(char)
                        characters.append(char)
                    }

                    return characters
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, interpreter.CharacterValue{}, value)
				checks++
			},
		)

		value, err := invokable.Invoke("main")
		require.NoError(t, err)

		RequireValuesEqual(
			t,
			invokable,
			interpreter.NewArrayValue(
				invokable,
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeCharacter,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredCharacterValue("H"),
				interpreter.NewUnmeteredCharacterValue("e"),
				interpreter.NewUnmeteredCharacterValue("l"),
				interpreter.NewUnmeteredCharacterValue("l"),
				interpreter.NewUnmeteredCharacterValue("o"),
			),
			value,
		)

		require.Equal(t, 5, checks)
	})

	t.Run("Resource array, use after loop", func(t *testing.T) {
		t.Parallel()

		var checks int

		invokable := prepare(t,
			`
                resource Foo {
                    fun bar() {}
                }

                fun main() {
                    let array <- [ <- create Foo(), <- create Foo()]

                    // Take a reference to an element.
                    let arrayElementRef = &array[0] as &Foo

                    let arrayRef = &array as &[Foo]
                    for element in arrayRef {
                        check(element)
                        let e: &Foo = element
                    }

                    // Reference should stay valid, even after looping.
                    // i.e: Loop should not move-out the elements.
                    arrayElementRef.bar()

                    destroy array
                }
            `,
			func(value interpreter.Value) {
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
				checks++
			},
		)

		_, err := invokable.Invoke("main")
		require.NoError(t, err)

		require.Equal(t, 2, checks)
	})
}

func TestInterpretStorageReferencesInForLoop(t *testing.T) {

	t.Parallel()

	t.Run("Primitive array", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            fun test() {
                let array = ["Hello", "World", "Foo", "Bar"]
                account.storage.save(array, to: /storage/array)

                let arrayRef = account.storage.borrow<&[String]>(from: /storage/array)!

                for element in arrayRef {
                    let e: String = element    // Must be the concrete string
                }
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("Struct array", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            struct Foo {}

            fun test() {
                let array = [Foo(), Foo()]
                account.storage.save(array, to: /storage/array)

                let arrayRef = account.storage.borrow<&[Foo]>(from: /storage/array)!

                for element in arrayRef {
                    let e: &Foo = element    // Must be a reference
                }
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("Resource array", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource Foo {}

            fun test() {
                let array <- [ <- create Foo(), <- create Foo()]
                account.storage.save(<- array, to: /storage/array)

                let arrayRef = account.storage.borrow<&[Foo]>(from: /storage/array)!

                for element in arrayRef {
                    let e: &Foo = element    // Must be a reference
                }
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("Moved resource array", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
            resource Foo {}

            fun test() {
                let array <- [ <- create Foo(), <- create Foo()]
                account.storage.save(<- array, to: /storage/array)

                let arrayRef = account.storage.borrow<&[Foo]>(from: /storage/array)!

                let movedArray <- account.storage.load<@[Foo]>(from: /storage/array)!

                for element in arrayRef {
                    let e: &Foo = element    // Must be a reference
                }

                destroy movedArray
            }`, sema.Config{})

		_, err := inter.Invoke("test")
		RequireError(t, err)
		var dereferenceError *interpreter.DereferenceError
		require.ErrorAs(t, err, &dereferenceError)
	})
}

type inclusiveRangeForInLoopTest struct {
	start, end, step int8
	loopElements     []int
}

func TestInclusiveRangeForInLoop(t *testing.T) {
	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterInclusiveRangeConstructor)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, stdlib.InterpreterInclusiveRangeConstructor)

	unsignedTestCases := []inclusiveRangeForInLoopTest{
		{
			start:        0,
			end:          10,
			step:         1,
			loopElements: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			start:        0,
			end:          10,
			step:         2,
			loopElements: []int{0, 2, 4, 6, 8, 10},
		},
	}

	signedTestCases := []inclusiveRangeForInLoopTest{
		{
			start:        10,
			end:          -10,
			step:         -2,
			loopElements: []int{10, 8, 6, 4, 2, 0, -2, -4, -6, -8, -10},
		},
	}

	runTestCase := func(t *testing.T, typ sema.Type, testCase inclusiveRangeForInLoopTest) {
		t.Run(typ.String(), func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf(
				`
					fun test(): [%[1]s] {
						let start: %[1]s = %[2]d
						let end: %[1]s = %[3]d
						let step: %[1]s = %[4]d
						let range: InclusiveRange<%[1]s> = InclusiveRange(start, end, step: step)

						var elements: [%[1]s] = []
						for element in range {
							elements.append(element)
						}
						return elements
					}
				`,
				typ.String(),
				testCase.start,
				testCase.end,
				testCase.step,
			)

			inter, err := parseCheckAndPrepareWithOptions(t, code,
				ParseCheckAndInterpretOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
								return baseValueActivation
							},
						},
					},
					InterpreterConfig: &interpreter.Config{
						BaseActivationHandler: func(common.Location) *interpreter.VariableActivation {
							return baseActivation
						},
					},
				},
			)

			require.NoError(t, err)
			loopElements, err := inter.Invoke("test")
			require.NoError(t, err)

			integerStaticType := interpreter.ConvertSemaToStaticType(
				nil,
				typ,
			)

			count := 0
			loopElementsArray := loopElements.(*interpreter.ArrayValue)

			loopElementsArray.ForEach(
				inter,
				nil,
				func(value interpreter.Value) (resume bool) {
					AssertValuesEqual(
						t,
						inter,
						interpreter.GetSmallIntegerValue(
							int8(testCase.loopElements[count]),
							integerStaticType,
						),
						value,
					)

					count += 1

					return true
				},
				false,
				interpreter.EmptyLocationRange,
			)

			assert.Equal(t, len(testCase.loopElements), count)
		})
	}

	for _, typ := range sema.AllIntegerTypes {
		// Only test leaf types
		switch typ {
		case sema.IntegerType,
			sema.SignedIntegerType,
			sema.FixedSizeUnsignedIntegerType:
			continue
		}

		for _, testCase := range unsignedTestCases {
			runTestCase(t, typ, testCase)
		}
	}

	for _, typ := range sema.AllSignedIntegerTypes {
		// Only test leaf types
		switch typ {
		case sema.SignedIntegerType:
			continue
		}

		for _, testCase := range signedTestCases {
			runTestCase(t, typ, testCase)
		}
	}
}
