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

	RegisterTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeRemoveFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryRemoveFunctionType(dictionaryType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				dictionary := value.(*interpreter.DictionaryValue)
				key := arguments[1]
				return dictionary.Remove(context, EmptyLocationRange, key)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeInsertFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryInsertFunctionType(dictionaryType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				dictionary := value.(*interpreter.DictionaryValue)
				keyValue := arguments[1]
				newValue := arguments[2]

				return dictionary.Insert(
					context,
					EmptyLocationRange,
					keyValue,
					newValue,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeContainsKeyFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryContainsKeyFunctionType(dictionaryType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				dictionary := value.(*interpreter.DictionaryValue)
				key := arguments[1]
				return dictionary.ContainsKey(
					context,
					EmptyLocationRange,
					key,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeForEachKeyFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				dictionaryValue := receiver.(*interpreter.DictionaryValue)
				dictionaryType := dictionaryValue.SemaType(context)
				return sema.DictionaryRemoveFunctionType(dictionaryType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				dictionary := value.(*interpreter.DictionaryValue)
				funcArgument := arguments[1].(FunctionValue)
				dictionary.ForEachKey(
					context,
					EmptyLocationRange,
					funcArgument,
				)

				return interpreter.Void
			},
		),
	)
}

func dictionaryType(receiver Value, context interpreter.TypeConverter) *sema.DictionaryType {
	dictionaryValue := receiver.(*interpreter.DictionaryValue)
	dictionaryType := dictionaryValue.SemaType(context)
	return dictionaryType
}
