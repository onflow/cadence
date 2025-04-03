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
	"fmt"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib/contracts"
)

type TestContractType struct {
	Checker                  *sema.Checker
	CompositeType            *sema.CompositeType
	InitializerTypes         []sema.Type
	emulatorBackendType      *testEmulatorBackendType
	expectFunction           testContractBoundFunctionGenerator
	newMatcherFunction       testContractBoundFunctionGenerator
	haveElementCountFunction testContractBoundFunctionGenerator
	beEmptyFunction          testContractBoundFunctionGenerator
	equalFunction            testContractBoundFunctionGenerator
	beGreaterThanFunction    testContractBoundFunctionGenerator
	containFunction          testContractBoundFunctionGenerator
	beLessThanFunction       testContractBoundFunctionGenerator
	expectFailureFunction    testContractBoundFunctionGenerator
}

type testContractBoundFunctionGenerator func(
	interpreter.FunctionCreationContext,
	*interpreter.CompositeValue,
) interpreter.BoundFunctionValue

// 'Test.assert' function

const testTypeAssertFunctionDocString = `
Fails the test-case if the given condition is false, and reports a message which explains how the condition is false.
`

const testTypeAssertFunctionName = "assert"

var testTypeAssertFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "condition",
			TypeAnnotation: sema.BoolTypeAnnotation,
		},
		{
			Identifier:     "message",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	// `message` parameter is optional
	Arity: &sema.Arity{Min: 1, Max: 2},
}

func testTypeAssertFunction(
	inter *interpreter.Interpreter,
	testContractValue *interpreter.CompositeValue,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		testContractValue,
		testTypeAssertFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			condition, ok := invocation.Arguments[0].(interpreter.BoolValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			var message string
			if len(invocation.Arguments) > 1 {
				messageValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				message = messageValue.Str
			}

			if !condition {
				panic(AssertionError{
					Message:       message,
					LocationRange: invocation.LocationRange,
				})
			}

			return interpreter.Void
		},
	)
}

// 'Test.assertEqual' function

const testTypeAssertEqualFunctionDocString = `
Fails the test-case if the given values are not equal, and
reports a message which explains how the two values differ.
`

const testTypeAssertEqualFunctionName = "assertEqual"

var testTypeAssertEqualFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "expected",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.AnyStructType,
			),
		},
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "actual",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.AnyStructType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

func testTypeAssertEqualFunction(
	inter *interpreter.Interpreter,
	testContractValue *interpreter.CompositeValue,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		testContractValue,
		testTypeAssertEqualFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			expected, ok := invocation.Arguments[0].(interpreter.EquatableValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			actual, ok := invocation.Arguments[1].(interpreter.EquatableValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.InvocationContext

			expectedType := expected.StaticType(inter)
			actualType := actual.StaticType(inter)
			if !expectedType.Equal(actualType) {
				message := fmt.Sprintf(
					"not equal types: expected: %s, actual: %s",
					expectedType,
					actualType,
				)
				panic(AssertionError{
					Message:       message,
					LocationRange: invocation.LocationRange,
				})
			}

			equal := expected.Equal(
				inter,
				invocation.LocationRange,
				actual,
			)

			if !equal {
				message := fmt.Sprintf(
					"not equal: expected: %s, actual: %s",
					expected,
					actual,
				)
				panic(AssertionError{
					Message:       message,
					LocationRange: invocation.LocationRange,
				})
			}

			return interpreter.Void
		},
	)
}

// 'Test.fail' function

const testTypeFailFunctionDocString = `
Fails the test-case with a message.
`

const testTypeFailFunctionName = "fail"

var testTypeFailFunctionType = &sema.FunctionType{
	Purity: sema.FunctionPurityView,
	Parameters: []sema.Parameter{
		{
			Identifier:     "message",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	// `message` parameter is optional
	Arity: &sema.Arity{Min: 0, Max: 1},
}

func testTypeFailFunction(
	inter *interpreter.Interpreter,
	testContractValue *interpreter.CompositeValue,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		inter,
		testContractValue,
		testTypeFailFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			var message string
			if len(invocation.Arguments) > 0 {
				messageValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				message = messageValue.Str
			}

			panic(AssertionError{
				Message:       message,
				LocationRange: invocation.LocationRange,
			})
		},
	)
}

// 'Test.expect' function

const testTypeExpectFunctionDocString = `
Expect function tests a value against a matcher, and fails the test if it's not a match.
`

const testTypeExpectFunctionName = "expect"

func newTestTypeExpectFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "matcher",
				TypeAnnotation: sema.NewTypeAnnotation(matcherType),
			},
		},
		ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	}
}

func newTestTypeExpectFunction(functionType *sema.FunctionType) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			functionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				value := invocation.Arguments[0]

				matcher, ok := invocation.Arguments[1].(*interpreter.CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				invocationContext := invocation.InvocationContext
				locationRange := invocation.LocationRange

				result := invokeMatcherTest(
					invocationContext,
					matcher,
					value,
					locationRange,
				)

				if !result {
					message := fmt.Sprintf(
						"given value is: %s",
						value,
					)
					panic(AssertionError{
						Message:       message,
						LocationRange: locationRange,
					})
				}

				return interpreter.Void
			},
		)
	}
}

func invokeMatcherTest(
	context interpreter.InvocationContext,
	matcher interpreter.MemberAccessibleValue,
	value interpreter.Value,
	locationRange interpreter.LocationRange,
) bool {
	testFunc := matcher.GetMember(
		context,
		locationRange,
		matcherTestFieldName,
	)

	funcValue, ok := testFunc.(interpreter.FunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected function",
			matcherTestFieldName,
		))
	}

	functionType := funcValue.FunctionType()

	testResult, err := interpreter.InvokeExternally(
		context,
		funcValue,
		functionType,
		[]interpreter.Value{
			value,
		},
	)

	if err != nil {
		panic(err)
	}

	result, ok := testResult.(interpreter.BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return bool(result)
}

// 'Test.readFile' function

const testTypeReadFileFunctionDocString = `
Read a local file, and return the content as a string.
`

const testTypeReadFileFunctionName = "readFile"

var testTypeReadFileFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "path",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	ReturnTypeAnnotation: sema.StringTypeAnnotation,
}

func newTestTypeReadFileFunction(
	testFramework TestFramework,
	context interpreter.FunctionCreationContext,
	testContractValue *interpreter.CompositeValue,
) interpreter.BoundFunctionValue {
	return interpreter.NewUnmeteredBoundHostFunctionValue(
		context,
		testContractValue,
		testTypeReadFileFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			pathString, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			content, err := testFramework.ReadFile(pathString.Str)
			if err != nil {
				panic(err)
			}

			return interpreter.NewUnmeteredStringValue(content)
		},
	)
}

// 'Test.NewMatcher' function.
// Constructs a matcher that test only 'AnyStruct'.
// Accepts test function that accepts subtype of 'AnyStruct'.
//
// Signature:
//    fun newMatcher<T: AnyStruct>(test: fun(T): Bool): Test.Matcher
//
// where `T` is optional, and bound to `AnyStruct`.
//
// Sample usage: `Test.newMatcher(fun (_ value: Int: Bool) { return true })`

const testTypeNewMatcherFunctionDocString = `
Creates a matcher with a test function.
The test function is of type 'fun(T): Bool', where 'T' is bound to 'AnyStruct'.
`

const testTypeNewMatcherFunctionName = "newMatcher"

func newTestTypeNewMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "test",
				TypeAnnotation: sema.NewTypeAnnotation(
					// Type of the 'test' function: fun(T): Bool
					&sema.FunctionType{
						Parameters: []sema.Parameter{
							{
								Label:      sema.ArgumentLabelNotRequired,
								Identifier: "value",
								TypeAnnotation: sema.NewTypeAnnotation(
									&sema.GenericType{
										TypeParameter: typeParameter,
									},
								),
							},
						},
						ReturnTypeAnnotation: sema.BoolTypeAnnotation,
					},
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeNewMatcherFunction(
	newMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			newMatcherFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				test, ok := invocation.Arguments[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return newMatcherWithGenericTestFunction(
					invocation,
					test,
					matcherTestFunctionType,
				)
			},
		)
	}
}

// `Test.equal`

const testTypeEqualFunctionName = "equal"

const testTypeEqualFunctionDocString = `
Returns a matcher that succeeds if the tested value is equal to the given value.
`

func newTestTypeEqualFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeEqualFunction(
	equalFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			equalFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				otherValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.InvocationContext

				// This is a static function.
				equalTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {

						thisValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
						if !ok {
							panic(errors.NewUnreachableError())
						}

						equal := thisValue.Equal(
							inter,
							invocation.LocationRange,
							otherValue,
						)

						return interpreter.BoolValue(equal)
					},
				)

				return newMatcherWithGenericTestFunction(
					invocation,
					equalTestFunc,
					matcherTestFunctionType,
				)
			},
		)
	}
}

// `Test.beEmpty`

const testTypeBeEmptyFunctionName = "beEmpty"

const testTypeBeEmptyFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array or dictionary,
and the tested value contains no elements.
`

func newTestTypeBeEmptyFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		Purity:               sema.FunctionPurityView,
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeBeEmptyFunction(
	beEmptyFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			beEmptyFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {

				// This is a static function.
				beEmptyTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						var isEmpty bool
						switch value := invocation.Arguments[0].(type) {
						case *interpreter.ArrayValue:
							isEmpty = value.Count() == 0
						case *interpreter.DictionaryValue:
							isEmpty = value.Count() == 0
						default:
							panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
						}

						return interpreter.BoolValue(isEmpty)
					},
				)

				return newMatcherWithAnyStructTestFunction(
					invocation,
					beEmptyTestFunc,
				)
			},
		)
	}
}

// `Test.haveElementCount`

const testTypeHaveElementCountFunctionName = "haveElementCount"

const testTypeHaveElementCountFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array or dictionary,
and has the given number of elements.
`

func newTestTypeHaveElementCountFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "count",
				TypeAnnotation: sema.IntTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeHaveElementCountFunction(
	haveElementCountFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			haveElementCountFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				count, ok := invocation.Arguments[0].(interpreter.IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// This is a static function.
				haveElementCountTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						var matchingCount bool
						switch value := invocation.Arguments[0].(type) {
						case *interpreter.ArrayValue:
							matchingCount = value.Count() == count.ToInt(invocation.LocationRange)
						case *interpreter.DictionaryValue:
							matchingCount = value.Count() == count.ToInt(invocation.LocationRange)
						default:
							panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
						}

						return interpreter.BoolValue(matchingCount)
					},
				)

				return newMatcherWithAnyStructTestFunction(
					invocation,
					haveElementCountTestFunc,
				)
			},
		)
	}
}

// `Test.contain`

const testTypeContainFunctionName = "contain"

const testTypeContainFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array that contains
a value that is equal to the given value, or the tested value is a dictionary
that contains an entry where the key is equal to the given value.
`

func newTestTypeContainFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "element",
				TypeAnnotation: sema.AnyStructTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeContainFunction(
	containFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			containFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				element, ok := invocation.Arguments[0].(interpreter.EquatableValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.InvocationContext

				// This is a static function.
				containTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						var elementFound interpreter.BoolValue
						switch value := invocation.Arguments[0].(type) {
						case *interpreter.ArrayValue:
							elementFound = value.Contains(
								inter,
								invocation.LocationRange,
								element,
							)
						case *interpreter.DictionaryValue:
							elementFound = value.ContainsKey(
								inter,
								invocation.LocationRange,
								element,
							)
						default:
							panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
						}

						return elementFound
					},
				)

				return newMatcherWithAnyStructTestFunction(
					invocation,
					containTestFunc,
				)
			},
		)
	}
}

// `Test.beGreaterThan`

const testTypeBeGreaterThanFunctionName = "beGreaterThan"

const testTypeBeGreaterThanFunctionDocString = `
Returns a matcher that succeeds if the tested value is a number and
greater than the given number.
`

func newTestTypeBeGreaterThanFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: sema.NumberTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func newTestTypeBeGreaterThanFunction(
	beGreaterThanFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			beGreaterThanFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				otherValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.InvocationContext

				// This is a static function.
				beGreaterThanTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						thisValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
						if !ok {
							panic(errors.NewUnreachableError())
						}

						isGreaterThan := thisValue.Greater(
							inter,
							otherValue,
							invocation.LocationRange,
						)

						return isGreaterThan
					},
				)

				return newMatcherWithAnyStructTestFunction(
					invocation,
					beGreaterThanTestFunc,
				)
			},
		)
	}
}

// `Test.beLessThan`

const testTypeBeLessThanFunctionName = "beLessThan"

const testTypeBeLessThanFunctionDocString = `
Returns a matcher that succeeds if the tested value is a number and
less than the given number.
`

func newTestTypeBeLessThanFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: sema.NumberTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

// Test.expectFailure function

const testExpectFailureFunctionName = "expectFailure"

const testExpectFailureFunctionDocString = `
Wraps a function call in a closure, and expects it to fail with
an error message that contains the given error message portion.
`

func newTestTypeExpectFailureFunctionType() *sema.FunctionType {
	return &sema.FunctionType{
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "functionWrapper",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.FunctionType{
						ReturnTypeAnnotation: sema.VoidTypeAnnotation,
					},
				),
			},
			{
				Identifier:     "errorMessageSubstring",
				TypeAnnotation: sema.StringTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	}
}

func newTestTypeExpectFailureFunction(
	testExpectFailureFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			testExpectFailureFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				invocationContext := invocation.InvocationContext
				functionValue, ok := invocation.Arguments[0].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				functionType := functionValue.FunctionType()

				errorMessage, ok := invocation.Arguments[1].(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				failedAsExpected := true

				defer invocationContext.RecoverErrors(func(internalErr error) {
					if !failedAsExpected {
						panic(internalErr)
					} else if !strings.Contains(internalErr.Error(), errorMessage.Str) {
						panic(errors.NewDefaultUserError(
							"Expected error message to include: %s",
							errorMessage,
						))
					}
				})

				_, err := interpreter.InvokeExternally(
					invocationContext,
					functionValue,
					functionType,
					nil,
				)
				if err == nil {
					failedAsExpected = false
					panic(errors.NewDefaultUserError("Expected a failure, but found none."))
				}

				return interpreter.Void
			},
		)
	}
}

func newTestTypeBeLessThanFunction(
	beLessThanFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) testContractBoundFunctionGenerator {
	return func(context interpreter.FunctionCreationContext, testContractValue *interpreter.CompositeValue) interpreter.BoundFunctionValue {
		return interpreter.NewUnmeteredBoundHostFunctionValue(
			context,
			testContractValue,
			beLessThanFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				otherValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				inter := invocation.InvocationContext

				// This is a static function.
				beLessThanTestFunc := interpreter.NewStaticHostFunctionValue(
					nil,
					matcherTestFunctionType,
					func(invocation interpreter.Invocation) interpreter.Value {
						thisValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
						if !ok {
							panic(errors.NewUnreachableError())
						}

						isLessThan := thisValue.Less(
							inter,
							otherValue,
							invocation.LocationRange,
						)

						return isLessThan
					},
				)

				return newMatcherWithAnyStructTestFunction(
					invocation,
					beLessThanTestFunc,
				)
			},
		)
	}
}

func newTestContractType() *TestContractType {

	program, err := parser.ParseProgram(
		nil,
		contracts.TestContract,
		parser.Config{},
	)
	if err != nil {
		panic(err)
	}

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(AssertFunction)
	activation.DeclareValue(PanicFunction)

	checker, err := sema.NewChecker(
		program,
		TestContractLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return activation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	if err != nil {
		panic(err)
	}

	err = checker.Check()
	if err != nil {
		panic(err)
	}

	variable, ok := checker.Elaboration.GetGlobalType(testContractTypeName)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	compositeType := variable.Type.(*sema.CompositeType)

	initializerTypes := make([]sema.Type, len(compositeType.ConstructorParameters))
	for i, parameter := range compositeType.ConstructorParameters {
		initializerTypes[i] = parameter.TypeAnnotation.Type
	}

	ty := &TestContractType{
		Checker:          checker,
		CompositeType:    compositeType,
		InitializerTypes: initializerTypes,
	}

	blockchainBackendInterfaceType := ty.blockchainBackendInterfaceType()

	emulatorBackendType := newTestEmulatorBackendType(blockchainBackendInterfaceType)
	ty.emulatorBackendType = emulatorBackendType

	// Enrich 'Test' contract elaboration with natively implemented composite types.
	// e.g: 'EmulatorBackend' type.
	checker.Elaboration.SetCompositeType(
		emulatorBackendType.compositeType.ID(),
		emulatorBackendType.compositeType,
	)

	matcherType := ty.matcherType()
	matcherTestFunctionType := compositeFunctionType(matcherType, matcherTestFieldName)

	// Test.assert()
	compositeType.Members.Set(
		testTypeAssertFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeAssertFunctionName,
			testTypeAssertFunctionType,
			testTypeAssertFunctionDocString,
		),
	)

	// Test.assertEqual()
	compositeType.Members.Set(
		testTypeAssertEqualFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeAssertEqualFunctionName,
			testTypeAssertEqualFunctionType,
			testTypeAssertEqualFunctionDocString,
		),
	)

	// Test.fail()
	compositeType.Members.Set(
		testTypeFailFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeFailFunctionName,
			testTypeFailFunctionType,
			testTypeFailFunctionDocString,
		),
	)

	// Test.readFile()
	compositeType.Members.Set(
		testTypeReadFileFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeReadFileFunctionName,
			testTypeReadFileFunctionType,
			testTypeReadFileFunctionDocString,
		),
	)

	// Test.expect()
	testExpectFunctionType := newTestTypeExpectFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeExpectFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeExpectFunctionName,
			testExpectFunctionType,
			testTypeExpectFunctionDocString,
		),
	)
	ty.expectFunction = newTestTypeExpectFunction(testExpectFunctionType)

	// Test.newMatcher()
	newMatcherFunctionType := newTestTypeNewMatcherFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeNewMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeNewMatcherFunctionName,
			newMatcherFunctionType,
			testTypeNewMatcherFunctionDocString,
		),
	)
	ty.newMatcherFunction = newTestTypeNewMatcherFunction(
		newMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.equal()
	equalMatcherFunctionType := newTestTypeEqualFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeEqualFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeEqualFunctionName,
			equalMatcherFunctionType,
			testTypeEqualFunctionDocString,
		),
	)
	ty.equalFunction = newTestTypeEqualFunction(
		equalMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.beEmpty()
	beEmptyMatcherFunctionType := newTestTypeBeEmptyFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeBeEmptyFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeBeEmptyFunctionName,
			beEmptyMatcherFunctionType,
			testTypeBeEmptyFunctionDocString,
		),
	)
	ty.beEmptyFunction = newTestTypeBeEmptyFunction(
		beEmptyMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.haveElementCount()
	haveElementCountMatcherFunctionType := newTestTypeHaveElementCountFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeHaveElementCountFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeHaveElementCountFunctionName,
			haveElementCountMatcherFunctionType,
			testTypeHaveElementCountFunctionDocString,
		),
	)
	ty.haveElementCountFunction = newTestTypeHaveElementCountFunction(
		haveElementCountMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.contain()
	containMatcherFunctionType := newTestTypeContainFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeContainFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeContainFunctionName,
			containMatcherFunctionType,
			testTypeContainFunctionDocString,
		),
	)
	ty.containFunction = newTestTypeContainFunction(
		containMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.beGreaterThan()
	beGreaterThanMatcherFunctionType := newTestTypeBeGreaterThanFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeBeGreaterThanFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeBeGreaterThanFunctionName,
			beGreaterThanMatcherFunctionType,
			testTypeBeGreaterThanFunctionDocString,
		),
	)
	ty.beGreaterThanFunction = newTestTypeBeGreaterThanFunction(
		beGreaterThanMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.beLessThan()
	beLessThanMatcherFunctionType := newTestTypeBeLessThanFunctionType(matcherType)
	compositeType.Members.Set(
		testTypeBeLessThanFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testTypeBeLessThanFunctionName,
			beLessThanMatcherFunctionType,
			testTypeBeLessThanFunctionDocString,
		),
	)
	ty.beLessThanFunction = newTestTypeBeLessThanFunction(
		beLessThanMatcherFunctionType,
		matcherTestFunctionType,
	)

	// Test.expectFailure()
	expectFailureFunctionType := newTestTypeExpectFailureFunctionType()
	compositeType.Members.Set(
		testExpectFailureFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			compositeType,
			testExpectFailureFunctionName,
			expectFailureFunctionType,
			testExpectFailureFunctionDocString,
		),
	)
	ty.expectFailureFunction = newTestTypeExpectFailureFunction(
		expectFailureFunctionType,
	)
	compositeType.ResolveMembers()

	return ty
}

const testBlockchainBackendTypeName = "BlockchainBackend"

func (t *TestContractType) blockchainBackendInterfaceType() *sema.InterfaceType {
	typ, ok := t.CompositeType.NestedTypes.Get(testBlockchainBackendTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, testBlockchainBackendTypeName))
	}

	blockchainBackendInterfaceType, ok := typ.(*sema.InterfaceType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected interface",
			testBlockchainBackendTypeName,
		))
	}

	return blockchainBackendInterfaceType
}

func (t *TestContractType) matcherType() *sema.CompositeType {
	typ, ok := t.CompositeType.NestedTypes.Get(testMatcherTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, testMatcherTypeName))
	}

	matcherType, ok := typ.(*sema.CompositeType)
	if !ok || matcherType.Kind != common.CompositeKindStructure {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected struct type",
			testMatcherTypeName,
		))
	}

	return matcherType
}

func (t *TestContractType) NewTestContract(
	inter *interpreter.Interpreter,
	testFramework TestFramework,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {
	initializerTypes := t.InitializerTypes
	emulatorBackend := t.emulatorBackendType.newEmulatorBackend(
		inter,
		testFramework.EmulatorBackend(),
		interpreter.EmptyLocationRange,
	)
	returnType := constructor.FunctionType().ReturnTypeAnnotation.Type
	value, err := interpreter.InvokeFunctionValue(
		inter,
		constructor,
		[]interpreter.Value{emulatorBackend},
		initializerTypes,
		initializerTypes,
		returnType,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	// Inject natively implemented function values
	compositeValue.Functions.Set(testTypeAssertFunctionName, testTypeAssertFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeAssertEqualFunctionName, testTypeAssertEqualFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeFailFunctionName, testTypeFailFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeExpectFunctionName, t.expectFunction(inter, compositeValue))
	compositeValue.Functions.Set(
		testTypeReadFileFunctionName,
		newTestTypeReadFileFunction(testFramework, inter, compositeValue),
	)

	// Inject natively implemented matchers
	compositeValue.Functions.Set(testTypeNewMatcherFunctionName, t.newMatcherFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeEqualFunctionName, t.equalFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeBeEmptyFunctionName, t.beEmptyFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeHaveElementCountFunctionName, t.haveElementCountFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeContainFunctionName, t.containFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeBeGreaterThanFunctionName, t.beGreaterThanFunction(inter, compositeValue))
	compositeValue.Functions.Set(testTypeBeLessThanFunctionName, t.beLessThanFunction(inter, compositeValue))
	compositeValue.Functions.Set(testExpectFailureFunctionName, t.expectFailureFunction(inter, compositeValue))

	return compositeValue, nil
}
