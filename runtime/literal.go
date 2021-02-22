/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
)

var InvalidLiteralError = fmt.Errorf("invalid literal")
var UnsupportedLiteralError = fmt.Errorf("unsupported literal")
var LiteralExpressionTypeError = fmt.Errorf("input is not a literal")

// ParseLiteral parses a single literal string, that should have the given type.
//
// Returns an error if the literal string is not a literal (e.g. it does not have valid syntax,
// or does not parse to a literal).
//
func ParseLiteral(literal string, ty sema.Type) (cadence.Value, error) {
	expression, errs := parser2.ParseExpression(literal)
	if len(errs) > 0 {
		return nil, parser2.Error{
			Code:   literal,
			Errors: errs,
		}
	}

	return LiteralValue(expression, ty)
}

// ParseLiteralArgumentList parses an argument list with literals, that should have the given types.
//
// Returns an error if the code is not a valid argument list, or the arguments are not literals.
//
func ParseLiteralArgumentList(argumentList string, parameterTypes []sema.Type) ([]cadence.Value, error) {
	arguments, errs := parser2.ParseArgumentList(argumentList)
	if len(errs) > 0 {
		return nil, parser2.Error{
			Errors: errs,
		}
	}

	argumentCount := len(arguments)
	parameterCount := len(parameterTypes)

	if argumentCount != parameterCount {
		return nil, fmt.Errorf(
			"invalid number of arguments: got %d, expected %d",
			argumentCount,
			parameterCount,
		)
	}

	result := make([]cadence.Value, argumentCount)

	for i, argument := range arguments {
		parameterType := parameterTypes[i]
		value, err := LiteralValue(argument.Expression, parameterType)
		if err != nil {
			return nil, fmt.Errorf("invalid argument at index %d: %w", i, err)
		}
		result[i] = value
	}

	return result, nil
}

func arrayLiteralValue(elements []ast.Expression, elementType sema.Type) (cadence.Value, error) {
	values := make([]cadence.Value, len(elements))

	for i, element := range elements {
		convertedElement, err := LiteralValue(element, elementType)
		if err != nil {
			return nil, err
		}
		values[i] = convertedElement
	}

	return cadence.NewArray(values), nil
}

func pathLiteralValue(expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	pathExpression, ok := expression.(*ast.PathExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	pathType, err := sema.CheckPathLiteral(pathExpression)
	if err != nil {
		return nil, InvalidLiteralError
	}

	if !sema.IsSubType(pathType, ty) {
		return nil, fmt.Errorf(
			"path literal type %s is not subtype of requested path type %s",
			pathType, ty,
		)
	}

	return cadence.Path{
		Domain:     pathExpression.Domain.Identifier,
		Identifier: pathExpression.Identifier.Identifier,
	}, nil
}

func integerLiteralValue(expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	integerExpression, ok := expression.(*ast.IntegerExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	if !sema.CheckIntegerLiteral(integerExpression, ty, nil) {
		return nil, InvalidLiteralError
	}

	intValue := interpreter.NewIntValueFromBigInt(integerExpression.Value)

	converter := func(value interpreter.Value, _ *interpreter.Interpreter) interpreter.Value {
		return value
	}

	switch ty.(type) {
	case *sema.IntType, *sema.IntegerType, *sema.SignedIntegerType:
		break
	case *sema.Int8Type:
		converter = interpreter.ConvertInt8
	case *sema.Int16Type:
		converter = interpreter.ConvertInt16
	case *sema.Int32Type:
		converter = interpreter.ConvertInt32
	case *sema.Int64Type:
		converter = interpreter.ConvertInt64
	case *sema.Int128Type:
		converter = interpreter.ConvertInt128
	case *sema.Int256Type:
		converter = interpreter.ConvertInt256

	case *sema.UIntType:
		converter = interpreter.ConvertUInt
	case *sema.UInt8Type:
		converter = interpreter.ConvertUInt8
	case *sema.UInt16Type:
		converter = interpreter.ConvertUInt16
	case *sema.UInt32Type:
		converter = interpreter.ConvertUInt32
	case *sema.UInt64Type:
		converter = interpreter.ConvertUInt64
	case *sema.UInt128Type:
		converter = interpreter.ConvertUInt128
	case *sema.UInt256Type:
		converter = interpreter.ConvertUInt256

	case *sema.Word8Type:
		converter = interpreter.ConvertWord8
	case *sema.Word16Type:
		converter = interpreter.ConvertWord16
	case *sema.Word32Type:
		converter = interpreter.ConvertWord32
	case *sema.Word64Type:
		converter = interpreter.ConvertWord64

	default:
		return nil, UnsupportedLiteralError
	}

	result := ExportValue(converter(intValue, nil), nil)
	return result, nil
}

func fixedPointLiteralValue(expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	fixedPointExpression, ok := expression.(*ast.FixedPointExpression)
	if !ok {
		return nil, LiteralExpressionTypeError
	}

	if !sema.CheckFixedPointLiteral(fixedPointExpression, ty, nil) {
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

	switch ty.(type) {
	case *sema.Fix64Type, *sema.FixedPointType, *sema.SignedFixedPointType:
		return cadence.Fix64(value.Int64()), nil
	case *sema.UFix64Type:
		return cadence.UFix64(value.Uint64()), nil
	}

	return nil, UnsupportedLiteralError
}

func LiteralValue(expression ast.Expression, ty sema.Type) (cadence.Value, error) {
	switch ty := ty.(type) {
	case *sema.VariableSizedType:
		expression, ok := expression.(*ast.ArrayExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return arrayLiteralValue(expression.Values, ty.Type)

	case *sema.ConstantSizedType:
		expression, ok := expression.(*ast.ArrayExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return arrayLiteralValue(expression.Values, ty.Type)

	case *sema.OptionalType:
		if _, ok := expression.(*ast.NilExpression); ok {
			return cadence.NewOptional(nil), nil
		}

		converted, err := LiteralValue(expression, ty.Type)
		if err != nil {
			return nil, err
		}

		return cadence.NewOptional(converted), nil

	case *sema.DictionaryType:
		expression, ok := expression.(*ast.DictionaryExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		var err error

		pairs := make([]cadence.KeyValuePair, len(expression.Entries))

		for i, entry := range expression.Entries {

			pairs[i].Key, err = LiteralValue(entry.Key, ty.KeyType)
			if err != nil {
				return nil, err
			}

			pairs[i].Value, err = LiteralValue(entry.Value, ty.ValueType)
			if err != nil {
				return nil, err
			}
		}

		return cadence.NewDictionary(pairs), nil

	case *sema.StringType:
		expression, ok := expression.(*ast.StringExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		return cadence.NewString(expression.Value), nil

	case *sema.AddressType:
		expression, ok := expression.(*ast.IntegerExpression)
		if !ok {
			return nil, LiteralExpressionTypeError
		}

		if !sema.CheckAddressLiteral(expression, nil) {
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

		return cadence.NewBool(expression.Value), nil
	}

	switch {
	case sema.IsSubType(ty, &sema.IntegerType{}):
		return integerLiteralValue(expression, ty)

	case sema.IsSubType(ty, &sema.FixedPointType{}):
		return fixedPointLiteralValue(expression, ty)

	case sema.IsSubType(ty, sema.PathType):
		return pathLiteralValue(expression, ty)
	}

	return nil, UnsupportedLiteralError
}
