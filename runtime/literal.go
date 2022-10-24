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

package runtime

import (
	"math/big"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
)

var InvalidLiteralError = parser.NewSyntaxError(
	ast.Position{Line: 1},
	"invalid literal",
)
var UnsupportedLiteralError = parser.NewSyntaxError(
	ast.Position{Line: 1},
	"unsupported literal",
)
var LiteralExpressionTypeError = parser.NewSyntaxError(
	ast.Position{Line: 1},
	"input is not a literal",
)

// ParseLiteral parses a single literal string, that should have the given type.
//
// Returns an error if the literal string is not a literal (e.g. it does not have valid syntax,
// or does not parse to a literal).
func ParseLiteral(
	literal string,
	ty sema.Type,
	inter *interpreter.Interpreter,
) (
	cadence.Value,
	error,
) {
	code := []byte(literal)

	expression, errs := parser.ParseExpression(code, inter)
	if len(errs) > 0 {
		return nil, parser.Error{
			Code:   code,
			Errors: errs,
		}
	}

	return LiteralValue(inter, expression, ty)
}

// ParseLiteralArgumentList parses an argument list with literals, that should have the given types.
// Returns an error if the code is not a valid argument list, or the arguments are not literals.
//
// Note: This method is not used directly within Cadence, but used by downstream dependencies
// such as CLI, playground, etc. Hence, shouldn't be moved to test.
func ParseLiteralArgumentList(
	argumentList string,
	parameterTypes []sema.Type,
	inter *interpreter.Interpreter,
) (
	[]cadence.Value,
	error,
) {
	code := []byte(argumentList)
	arguments, errs := parser.ParseArgumentList(code, inter)
	if len(errs) > 0 {
		return nil, parser.Error{
			Errors: errs,
		}
	}

	argumentCount := len(arguments)
	parameterCount := len(parameterTypes)

	if argumentCount != parameterCount {
		return nil, parser.NewSyntaxError(
			ast.Position{Line: 1},
			"invalid number of arguments: got %d, expected %d",
			argumentCount,
			parameterCount,
		)
	}

	result := make([]cadence.Value, argumentCount)

	for i, argument := range arguments {
		parameterType := parameterTypes[i]
		value, err := LiteralValue(inter, argument.Expression, parameterType)
		if err != nil {
			return nil, parser.NewSyntaxError(
				ast.Position{Line: 1},
				"invalid argument at index %d: %v", i, err,
			)
		}
		result[i] = value
	}

	return result, nil
}

func arrayLiteralValue(inter *interpreter.Interpreter, elements []ast.Expression, elementType sema.Type) (cadence.Value, error) {
	return cadence.NewMeteredArray(
		inter,
		len(elements),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, len(elements))

			for i, element := range elements {
				convertedElement, err := LiteralValue(inter, element, elementType)
				if err != nil {
					return nil, err
				}
				values[i] = convertedElement
			}

			return values, nil
		})
}

func pathLiteralValue(memoryGauge common.MemoryGauge, expression ast.Expression, ty sema.Type) (result cadence.Value, errResult error) {
	pathExpression, ok := expression.(*ast.PathExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	pathType, err := sema.CheckPathLiteral(
		pathExpression.Domain.Identifier,
		pathExpression.Identifier.Identifier,
		func() ast.Range {
			return ast.NewRangeFromPositioned(memoryGauge, pathExpression.Domain)
		},
		func() ast.Range {
			return ast.NewRangeFromPositioned(memoryGauge, pathExpression.Identifier)
		},
	)
	if err != nil {
		return nil, InvalidLiteralError
	}

	if !sema.IsSubType(pathType, ty) {
		return nil, parser.NewSyntaxError(
			ast.Position{Line: 1},
			"path literal type %s is not subtype of requested path type %s",
			pathType, ty,
		)
	}

	return cadence.NewMeteredPath(
		memoryGauge,
		pathExpression.Domain.Identifier,
		pathExpression.Identifier.Identifier,
	), nil
}

func integerLiteralValue(
	inter *interpreter.Interpreter,
	expression ast.Expression,
	ty sema.Type,
) (cadence.Value, error) {
	integerExpression, ok := expression.(*ast.IntegerExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	if !sema.CheckIntegerLiteral(inter, integerExpression, ty, nil) {
		return nil, InvalidLiteralError
	}

	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(integerExpression.Value),
	)
	intValue := interpreter.NewIntValueFromBigInt(
		inter,
		memoryUsage,
		func() *big.Int {
			return integerExpression.Value
		},
	)

	convertedValue, err := convertIntValue(
		inter,
		intValue,
		ty,
	)
	if err != nil {
		return nil, err
	}

	return ExportValue(convertedValue, inter, interpreter.EmptyLocationRange)
}

func convertIntValue(
	memoryGauge common.MemoryGauge,
	intValue interpreter.IntValue,
	ty sema.Type,
) (
	interpreter.Value,
	error,
) {

	switch ty {
	case sema.IntType, sema.IntegerType, sema.SignedIntegerType:
		return intValue, nil
	case sema.Int8Type:
		return interpreter.ConvertInt8(memoryGauge, intValue), nil
	case sema.Int16Type:
		return interpreter.ConvertInt16(memoryGauge, intValue), nil
	case sema.Int32Type:
		return interpreter.ConvertInt32(memoryGauge, intValue), nil
	case sema.Int64Type:
		return interpreter.ConvertInt64(memoryGauge, intValue), nil
	case sema.Int128Type:
		return interpreter.ConvertInt128(memoryGauge, intValue), nil
	case sema.Int256Type:
		return interpreter.ConvertInt256(memoryGauge, intValue), nil

	case sema.UIntType:
		return interpreter.ConvertUInt(memoryGauge, intValue), nil
	case sema.UInt8Type:
		return interpreter.ConvertUInt8(memoryGauge, intValue), nil
	case sema.UInt16Type:
		return interpreter.ConvertUInt16(memoryGauge, intValue), nil
	case sema.UInt32Type:
		return interpreter.ConvertUInt32(memoryGauge, intValue), nil
	case sema.UInt64Type:
		return interpreter.ConvertUInt64(memoryGauge, intValue), nil
	case sema.UInt128Type:
		return interpreter.ConvertUInt128(memoryGauge, intValue), nil
	case sema.UInt256Type:
		return interpreter.ConvertUInt256(memoryGauge, intValue), nil

	case sema.Word8Type:
		return interpreter.ConvertWord8(memoryGauge, intValue), nil
	case sema.Word16Type:
		return interpreter.ConvertWord16(memoryGauge, intValue), nil
	case sema.Word32Type:
		return interpreter.ConvertWord32(memoryGauge, intValue), nil
	case sema.Word64Type:
		return interpreter.ConvertWord64(memoryGauge, intValue), nil

	default:
		return nil, UnsupportedLiteralError
	}
}

func fixedPointLiteralValue(memoryGauge common.MemoryGauge, expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	fixedPointExpression, ok := expression.(*ast.FixedPointExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	if !sema.CheckFixedPointLiteral(memoryGauge, fixedPointExpression, ty, nil) {
		return nil, InvalidLiteralError
	}

	// TODO: adjust once/if we support more fixed point types

	value := fixedpoint.ConvertToFixedPointBigInt(
		fixedPointExpression.Negative,
		fixedPointExpression.UnsignedInteger,
		fixedPointExpression.Fractional,
		fixedPointExpression.Scale,
		sema.Fix64Scale,
	)

	switch ty {
	case sema.Fix64Type, sema.FixedPointType, sema.SignedFixedPointType:
		return cadence.Fix64(value.Int64()), nil
	case sema.UFix64Type:
		return cadence.UFix64(value.Uint64()), nil
	}

	return nil, UnsupportedLiteralError
}

func LiteralValue(inter *interpreter.Interpreter, expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	switch ty := ty.(type) {
	case *sema.VariableSizedType:
		expression, ok := expression.(*ast.ArrayExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return arrayLiteralValue(inter, expression.Values, ty.Type)

	case *sema.ConstantSizedType:
		expression, ok := expression.(*ast.ArrayExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return arrayLiteralValue(inter, expression.Values, ty.Type)

	case *sema.OptionalType:
		if _, ok := expression.(*ast.NilExpression); ok {
			return cadence.NewMeteredOptional(inter, nil), nil
		}

		converted, err := LiteralValue(inter, expression, ty.Type)
		if err != nil {
			return nil, err
		}

		return cadence.NewMeteredOptional(inter, converted), nil

	case *sema.DictionaryType:
		expression, ok := expression.(*ast.DictionaryExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return cadence.NewMeteredDictionary(
			inter,
			len(expression.Entries),
			func() ([]cadence.KeyValuePair, error) {
				pairs := make([]cadence.KeyValuePair, len(expression.Entries))

				for i, entry := range expression.Entries {
					var err error

					pairs[i].Key, err = LiteralValue(inter, entry.Key, ty.KeyType)
					if err != nil {
						return nil, err
					}

					pairs[i].Value, err = LiteralValue(inter, entry.Value, ty.ValueType)
					if err != nil {
						return nil, err
					}
				}

				return pairs, nil
			},
		)

	case *sema.AddressType:
		expression, ok := expression.(*ast.IntegerExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		if !sema.CheckAddressLiteral(inter, expression, nil) {
			return nil, InvalidLiteralError
		}

		return cadence.BytesToAddress(expression.Value.Bytes()), nil
	}

	switch ty {
	case sema.BoolType:
		expression, ok := expression.(*ast.BoolExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return cadence.NewMeteredBool(inter, expression.Value), nil

	case sema.StringType:
		expression, ok := expression.(*ast.StringExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return cadence.NewMeteredString(
			inter,
			common.NewCadenceStringMemoryUsage(len(expression.Value)),
			func() string {
				return expression.Value
			},
		)
	}

	switch {
	case sema.IsSameTypeKind(ty, sema.IntegerType):
		return integerLiteralValue(inter, expression, ty)

	case sema.IsSameTypeKind(ty, sema.FixedPointType):
		return fixedPointLiteralValue(inter, expression, ty)

	case sema.IsSameTypeKind(ty, sema.PathType):
		return pathLiteralValue(inter, expression, ty)
	}

	return nil, UnsupportedLiteralError
}
