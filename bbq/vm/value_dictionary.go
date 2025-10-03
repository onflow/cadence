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

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeRemoveFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryRemoveFunctionType(dictionaryType)
			},
			interpreter.UnifiedDictionaryRemoveFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeInsertFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryInsertFunctionType(dictionaryType)
			},
			interpreter.UnifiedDictionaryInsertFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeContainsKeyFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				dictionaryType := dictionaryType(receiver, context)
				return sema.DictionaryContainsKeyFunctionType(dictionaryType)
			},
			interpreter.UnifiedDictionaryContainsKeyFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierDictionary,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.DictionaryTypeForEachKeyFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				dictionaryValue := receiver.(*interpreter.DictionaryValue)
				dictionaryType := dictionaryValue.SemaType(context)
				return sema.DictionaryRemoveFunctionType(dictionaryType)
			},
			interpreter.UnifiedDictionaryForEachKeyFunction,
		),
	)
}

func dictionaryType(receiver Value, context interpreter.ValueStaticTypeContext) *sema.DictionaryType {
	dictionaryValue := receiver.(*interpreter.DictionaryValue)
	dictionaryType := dictionaryValue.SemaType(context)
	return dictionaryType
}
