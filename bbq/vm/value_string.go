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
	"github.com/onflow/cadence/bbq"
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
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				other := arguments[0]
				return interpreter.StringConcat(
					context,
					this,
					other,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSliceFunctionName,
			sema.StringTypeSliceFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				from := arguments[0].(interpreter.IntValue)
				to := arguments[1].(interpreter.IntValue)
				return this.Slice(from, to, EmptyLocationRange)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeContainsFunctionName,
			sema.StringTypeContainsFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				other := arguments[0].(*interpreter.StringValue)
				return this.Contains(context, other)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeIndexFunctionName,
			sema.StringTypeIndexFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				other := arguments[0].(*interpreter.StringValue)
				return this.IndexOf(context, other)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeCountFunctionName,
			sema.StringTypeCountFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				other := arguments[0].(*interpreter.StringValue)
				return this.Count(context, EmptyLocationRange, other)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeDecodeHexFunctionName,
			sema.StringTypeDecodeHexFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments) // nolint:staticcheck,ineffassign
				return this.DecodeHex(context, EmptyLocationRange)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeToLowerFunctionName,
			sema.StringTypeToLowerFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments) // nolint:staticcheck,ineffassign
				return this.ToLower(context)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSplitFunctionName,
			sema.StringTypeSplitFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				separator := arguments[0].(*interpreter.StringValue)
				return this.Split(
					context,
					EmptyLocationRange,
					separator,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeReplaceAllFunctionName,
			sema.StringTypeReplaceAllFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				this, arguments := SplitTypedReceiverAndArgs[*interpreter.StringValue](context, arguments)
				original := arguments[0].(*interpreter.StringValue)
				replacement := arguments[1].(*interpreter.StringValue)
				return this.ReplaceAll(
					context,
					EmptyLocationRange,
					original,
					replacement,
				)
			},
		),
	)

	// Methods on `String` type.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeEncodeHexFunctionName,
			sema.StringTypeEncodeHexFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				byteArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionEncodeHex(
					context,
					byteArray,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromUtf8FunctionName,
			sema.StringTypeFromUtf8FunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				byteArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionFromUtf8(
					context,
					byteArray,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromCharactersFunctionName,
			sema.StringTypeFromCharactersFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				charactersArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionFromCharacters(
					context,
					charactersArray,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeJoinFunctionName,
			sema.StringTypeJoinFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				stringArray := arguments[0].(*interpreter.ArrayValue)
				separator := arguments[1].(*interpreter.StringValue)

				return interpreter.StringFunctionJoin(
					context,
					stringArray,
					separator,
					EmptyLocationRange,
				)
			},
		),
	)
}
