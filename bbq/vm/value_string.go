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
		NewUnifiedNativeFunctionValue(
			sema.StringTypeConcatFunctionName,
			sema.StringTypeConcatFunctionType,
			interpreter.UnifiedStringConcatFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeSliceFunctionName,
			sema.StringTypeSliceFunctionType,
			interpreter.UnifiedStringSliceFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeContainsFunctionName,
			sema.StringTypeContainsFunctionType,
			interpreter.UnifiedStringContainsFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeIndexFunctionName,
			sema.StringTypeIndexFunctionType,
			interpreter.UnifiedStringIndexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeCountFunctionName,
			sema.StringTypeCountFunctionType,
			interpreter.UnifiedStringCountFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeDecodeHexFunctionName,
			sema.StringTypeDecodeHexFunctionType,
			interpreter.UnifiedStringDecodeHexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeToLowerFunctionName,
			sema.StringTypeToLowerFunctionType,
			interpreter.UnifiedStringToLowerFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeSplitFunctionName,
			sema.StringTypeSplitFunctionType,
			interpreter.UnifiedStringSplitFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeReplaceAllFunctionName,
			sema.StringTypeReplaceAllFunctionType,
			interpreter.UnifiedStringReplaceAllFunction,
		),
	)

	// Methods on `String` type.
	// No receiver.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeEncodeHexFunctionName,
			sema.StringTypeEncodeHexFunctionType,
			interpreter.UnifiedStringEncodeHexFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeFromUtf8FunctionName,
			sema.StringTypeFromUtf8FunctionType,
			interpreter.UnifiedStringFromUtf8Function,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeFromCharactersFunctionName,
			sema.StringTypeFromCharactersFunctionType,
			interpreter.UnifiedStringFromCharactersFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewUnifiedNativeFunctionValue(
			sema.StringTypeJoinFunctionName,
			sema.StringTypeJoinFunctionType,
			interpreter.UnifiedStringJoinFunction,
		),
	)
}
