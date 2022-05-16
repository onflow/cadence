/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
)

func (checker *Checker) VisitDictionaryExpression(expression *ast.DictionaryExpression) ast.Repr {

	// visit all entries, ensure key are all the same type,
	// and values are all the same type

	var keyType, valueType Type

	expectedType := UnwrapOptionalType(checker.expectedType)

	if expectedMapType, ok := expectedType.(*DictionaryType); ok {
		keyType = expectedMapType.KeyType
		valueType = expectedMapType.ValueType
	}

	dictionarySize := len(expression.Entries)
	entryTypes := make([]DictionaryEntryType, dictionarySize)
	keyTypes := make([]Type, dictionarySize)
	valueTypes := make([]Type, dictionarySize)

	for i, entry := range expression.Entries {
		// NOTE: important to check move after each type check,
		// not combined after both type checks!

		entryKeyType := checker.VisitExpression(entry.Key, keyType)
		checker.checkVariableMove(entry.Key)
		checker.checkResourceMoveOperation(entry.Key, entryKeyType)

		entryValueType := checker.VisitExpression(entry.Value, valueType)
		checker.checkVariableMove(entry.Value)
		checker.checkResourceMoveOperation(entry.Value, entryValueType)

		entryTypes[i] = DictionaryEntryType{
			KeyType:   entryKeyType,
			ValueType: entryValueType,
		}

		keyTypes[i] = entryKeyType
		valueTypes[i] = entryValueType
	}

	if keyType == nil && valueType == nil {
		// Contextually expected type is not available.
		// Therefore, find the least common supertype of the keys and values.
		keyType = LeastCommonSuperType(keyTypes...)
		valueType = LeastCommonSuperType(valueTypes...)

		if keyType == InvalidType ||
			valueType == InvalidType {
			checker.report(
				&TypeAnnotationRequiredError{
					Cause: "cannot infer type from dictionary literal: ",
					Pos:   expression.StartPos,
				},
			)

			return InvalidType
		}
	}

	if !IsValidDictionaryKeyType(keyType) {
		checker.report(
			&InvalidDictionaryKeyTypeError{
				Type:  keyType,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
	}

	dictionaryType := &DictionaryType{
		KeyType:   keyType,
		ValueType: valueType,
	}

	checker.Elaboration.DictionaryExpressionEntryTypes[expression] = entryTypes
	checker.Elaboration.DictionaryExpressionType[expression] = dictionaryType

	return dictionaryType
}

func IsValidDictionaryKeyType(keyType Type) bool {
	// TODO: implement support for more built-in types here and in interpreter
	switch keyType := keyType.(type) {
	case *AddressType:
		return true
	case *CompositeType:
		return keyType.Kind == common.CompositeKindEnum
	default:
		switch keyType {
		case NeverType, BoolType, CharacterType, StringType, MetaType:
			return true
		default:
			return IsSameTypeKind(keyType, NumberType) ||
				IsSameTypeKind(keyType, PathType)
		}
	}
}
