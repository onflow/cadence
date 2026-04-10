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

package sema

import "github.com/onflow/cadence/errors"

//go:generate go run ./gen stringbuilder.cdc stringbuilder.gen.go

var StringBuilderTypeAnnotation = NewTypeAnnotation(StringBuilderType)

const stringBuilderFunctionDocString = "Creates a new empty StringBuilder"

var StringBuilderFunctionType = func() *FunctionType {

	typeName := StringBuilderType.String()

	// Check that the function is not accidentally redeclared

	if BaseValueActivation.Find(typeName) != nil {
		panic(errors.NewUnreachableError())
	}

	functionType := NewSimpleFunctionType(
		FunctionPurityView,
		nil,
		StringBuilderTypeAnnotation,
	)

	functionType.TypeFunctionType = StringBuilderType

	BaseValueActivation.Set(
		typeName,
		baseFunctionVariable(
			typeName,
			functionType,
			stringBuilderFunctionDocString,
		),
	)

	return functionType
}()
