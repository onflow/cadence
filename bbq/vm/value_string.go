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
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {
	typeName := interpreter.PrimitiveStaticTypeString.String()

	// Methods on `String` value.

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeConcatFunctionName,
			sema.StringTypeConcatFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				other := arguments[typeBoundFunctionArgumentOffset]
				return interpreter.StringConcat(
					context,
					this,
					other,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSliceFunctionName,
			sema.StringTypeSliceFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				from := arguments[1].(interpreter.IntValue)
				to := arguments[2].(interpreter.IntValue)
				return this.Slice(from, to, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeContainsFunctionName,
			sema.StringTypeContainsFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				other := arguments[1].(*interpreter.StringValue)
				return this.Contains(context, other)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeIndexFunctionName,
			sema.StringTypeIndexFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				other := arguments[1].(*interpreter.StringValue)
				return this.IndexOf(context, other)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeCountFunctionName,
			sema.StringTypeCountFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				other := arguments[1].(*interpreter.StringValue)
				return this.Count(context, EmptyLocationRange, other)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeDecodeHexFunctionName,
			sema.StringTypeDecodeHexFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				return this.DecodeHex(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeToLowerFunctionName,
			sema.StringTypeToLowerFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				return this.ToLower(context)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeSplitFunctionName,
			sema.StringTypeSplitFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				separator := arguments[1].(*interpreter.StringValue)
				return this.Split(
					context,
					EmptyLocationRange,
					separator,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeReplaceAllFunctionName,
			sema.StringTypeReplaceAllFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				this := arguments[receiverIndex].(*interpreter.StringValue)
				original := arguments[1].(*interpreter.StringValue)
				replacement := arguments[2].(*interpreter.StringValue)
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

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeEncodeHexFunctionName,
			sema.StringTypeEncodeHexFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				byteArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionEncodeHex(
					context,
					byteArray,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromUtf8FunctionName,
			sema.StringTypeFromUtf8FunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				byteArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionFromUtf8(
					context,
					byteArray,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeFromCharactersFunctionName,
			sema.StringTypeFromCharactersFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				charactersArray := arguments[0].(*interpreter.ArrayValue)
				return interpreter.StringFunctionFromCharacters(
					context,
					charactersArray,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.StringTypeJoinFunctionName,
			sema.StringTypeJoinFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
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

	// String constructor



}
