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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// MinFunction

const minFunctionName = "min"

const minFunctionDocString = `
Returns the minimum of the two given values.
The arguments must be of the same comparable type.
`

var minFunctionType = func() *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		Name: "T",
		// No TypeBound - we check comparability in TypeArgumentsCheck
	}

	typeAnnotation := sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "a",
				TypeAnnotation: typeAnnotation,
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "b",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: typeAnnotation,
		TypeArgumentsCheck: func(
			memoryGauge common.MemoryGauge,
			typeArguments *sema.TypeParameterTypeOrderedMap,
			_ []*ast.TypeAnnotation,
			invocationRange ast.HasPosition,
			report func(err error),
		) {
			typeArg, ok := typeArguments.Get(typeParameter)
			if !ok || typeArg == nil {
				// Invalid, already reported by checker
				return
			}

			if !typeArg.IsComparable() {
				report(&sema.InvalidTypeArgumentError{
					TypeArgumentName: typeParameter.Name,
					Range:            ast.NewRangeFromPositioned(memoryGauge, invocationRange),
					Details: fmt.Sprintf(
						"Type argument for `%s` must be a comparable type, got `%s`",
						minFunctionName,
						typeArg,
					),
				})
			}
		},
	}
}()

var NativeMinFunction = interpreter.NativeFunction(
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		a := args[0]
		b := args[1]

		comparableA, ok := a.(interpreter.ComparableValue)
		if !ok {
			panic(fmt.Sprintf("min: first argument is not comparable: %T", a))
		}

		comparableB, ok := b.(interpreter.ComparableValue)
		if !ok {
			panic(fmt.Sprintf("min: second argument is not comparable: %T", b))
		}

		if bool(comparableA.Less(context, comparableB)) {
			return a
		}
		return b
	},
)

var InterpreterMinFunction = NewNativeStandardLibraryStaticFunction(
	minFunctionName,
	minFunctionType,
	minFunctionDocString,
	NativeMinFunction,
	false,
)

var VMMinFunction = NewNativeStandardLibraryStaticFunction(
	minFunctionName,
	minFunctionType,
	minFunctionDocString,
	NativeMinFunction,
	true,
)

// MaxFunction

const maxFunctionName = "max"

const maxFunctionDocString = `
Returns the maximum of the two given values.
The arguments must be of the same comparable type.
`

var maxFunctionType = func() *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		Name: "T",
		// No TypeBound - we check comparability in TypeArgumentsCheck
	}

	typeAnnotation := sema.NewTypeAnnotation(
		&sema.GenericType{
			TypeParameter: typeParameter,
		},
	)

	return &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "a",
				TypeAnnotation: typeAnnotation,
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "b",
				TypeAnnotation: typeAnnotation,
			},
		},
		ReturnTypeAnnotation: typeAnnotation,
		TypeArgumentsCheck: func(
			memoryGauge common.MemoryGauge,
			typeArguments *sema.TypeParameterTypeOrderedMap,
			_ []*ast.TypeAnnotation,
			invocationRange ast.HasPosition,
			report func(err error),
		) {
			typeArg, ok := typeArguments.Get(typeParameter)
			if !ok || typeArg == nil {
				// Invalid, already reported by checker
				return
			}

			if !typeArg.IsComparable() {
				report(&sema.InvalidTypeArgumentError{
					TypeArgumentName: typeParameter.Name,
					Range:            ast.NewRangeFromPositioned(memoryGauge, invocationRange),
					Details: fmt.Sprintf(
						"Type argument for `%s` must be a comparable type, got `%s`",
						maxFunctionName,
						typeArg,
					),
				})
			}
		},
	}
}()

var NativeMaxFunction = interpreter.NativeFunction(
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		a := args[0]
		b := args[1]

		comparableA, ok := a.(interpreter.ComparableValue)
		if !ok {
			panic(fmt.Sprintf("max: first argument is not comparable: %T", a))
		}

		comparableB, ok := b.(interpreter.ComparableValue)
		if !ok {
			panic(fmt.Sprintf("max: second argument is not comparable: %T", b))
		}

		if bool(comparableA.Greater(context, comparableB)) {
			return a
		}
		return b
	},
)

var InterpreterMaxFunction = NewNativeStandardLibraryStaticFunction(
	maxFunctionName,
	maxFunctionType,
	maxFunctionDocString,
	NativeMaxFunction,
	false,
)

var VMMaxFunction = NewNativeStandardLibraryStaticFunction(
	maxFunctionName,
	maxFunctionType,
	maxFunctionDocString,
	NativeMaxFunction,
	true,
)
