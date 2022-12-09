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

// checkEventParameters checks that the event initializer's parameters are valid,
// as determined by `isValidEventParameterType`.
func (checker *Checker) checkEventParameters(
	parameterList *ast.ParameterList,
	parameters []Parameter,
) {

	parameterTypeValidationResults := map[*Member]bool{}

	for i, parameter := range parameterList.Parameters {
		parameterType := parameters[i].TypeAnnotation.Type

		if !parameterType.IsInvalidType() &&
			!IsValidEventParameterType(parameterType, parameterTypeValidationResults) {

			checker.report(
				&InvalidEventParameterTypeError{
					Type: parameterType,
					Range: ast.NewRange(
						checker.memoryGauge,
						parameter.StartPos,
						parameter.TypeAnnotation.EndPosition(checker.memoryGauge),
					),
				},
			)
		}
	}
}

// IsValidEventParameterType returns true if the given type is a valid event parameter type.
//
// Events currently only support a few simple Cadence types.
func IsValidEventParameterType(t Type, results map[*Member]bool) bool {

	switch t := t.(type) {
	case *AddressType:
		return true

	case *OptionalType:
		return IsValidEventParameterType(t.Type, results)

	case *VariableSizedType:
		return IsValidEventParameterType(t.ElementType(false), results)

	case *ConstantSizedType:
		return IsValidEventParameterType(t.ElementType(false), results)

	case *DictionaryType:
		return IsValidEventParameterType(t.KeyType, results) &&
			IsValidEventParameterType(t.ValueType, results)

	case *CompositeType:
		if t.Kind != common.CompositeKindStructure {
			return false
		}

		for pair := t.Members.Oldest(); pair != nil; pair = pair.Next() {
			member := pair.Value

			if !member.IsValidEventParameterType(results) {
				return false
			}
		}

		return true

	default:
		switch t {
		case MetaType, BoolType, CharacterType, StringType:
			return true
		}

		return IsSameTypeKind(t, NumberType) ||
			IsSameTypeKind(t, PathType)
	}
}
