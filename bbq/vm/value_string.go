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

package vm

import (
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {
	typeName := commons.TypeQualifier(sema.StringType)

	// Methods on `String` value.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeConcatFunctionName,
			sema.StringTypeConcatFunctionType,
			interpreter.NativeStringConcatFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSliceFunctionName,
			sema.StringTypeSliceFunctionType,
			interpreter.NativeStringSliceFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeContainsFunctionName,
			sema.StringTypeContainsFunctionType,
			interpreter.NativeStringContainsFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeIndexFunctionName,
			sema.StringTypeIndexFunctionType,
			interpreter.NativeStringIndexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeCountFunctionName,
			sema.StringTypeCountFunctionType,
			interpreter.NativeStringCountFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeDecodeHexFunctionName,
			sema.StringTypeDecodeHexFunctionType,
			interpreter.NativeStringDecodeHexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeToLowerFunctionName,
			sema.StringTypeToLowerFunctionType,
			interpreter.NativeStringToLowerFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSplitFunctionName,
			sema.StringTypeSplitFunctionType,
			interpreter.NativeStringSplitFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeReplaceAllFunctionName,
			sema.StringTypeReplaceAllFunctionType,
			interpreter.NativeStringReplaceAllFunction,
		),
	)

	// Methods on `String` type.
	// No receiver.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeEncodeHexFunctionName,
			sema.StringTypeEncodeHexFunctionType,
			interpreter.NativeStringEncodeHexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromUtf8FunctionName,
			sema.StringTypeFromUtf8FunctionType,
			interpreter.NativeStringFromUtf8Function,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromCharactersFunctionName,
			sema.StringTypeFromCharactersFunctionType,
			interpreter.NativeStringFromCharactersFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeJoinFunctionName,
			sema.StringTypeJoinFunctionType,
			interpreter.NativeStringJoinFunction,
		),
	)
}
