/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package interpreter

import (
	"math/big"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// assignmentGetterSetter returns a getter/setter function pair
// for the target expression
//
func (interpreter *Interpreter) assignmentGetterSetter(target ast.Expression) getterSetter {
	switch target := target.(type) {
	case *ast.IdentifierExpression:
		return interpreter.identifierExpressionGetterSetter(target)

	case *ast.IndexExpression:
		return interpreter.indexExpressionGetterSetter(target)

	case *ast.MemberExpression:
		return interpreter.memberExpressionGetterSetter(target)
	}

	panic(errors.NewUnreachableError())
}

// identifierExpressionGetterSetter returns a getter/setter function pair
// for the target identifier expression, wrapped in a trampoline
//
func (interpreter *Interpreter) identifierExpressionGetterSetter(identifierExpression *ast.IdentifierExpression) getterSetter {
	variable := interpreter.findVariable(identifierExpression.Identifier.Identifier)
	return getterSetter{
		get: func() Value {
			return variable.GetValue()
		},
		set: func(value Value) {
			variable.SetValue(value)
		},
	}
}

// indexExpressionGetterSetter returns a getter/setter function pair
// for the target index expression
//
func (interpreter *Interpreter) indexExpressionGetterSetter(indexExpression *ast.IndexExpression) getterSetter {
	typedResult := interpreter.evalExpression(indexExpression.TargetExpression).(ValueIndexableValue)
	indexingValue := interpreter.evalExpression(indexExpression.IndexingExpression)
	getLocationRange := locationRangeGetter(interpreter.Location, indexExpression)
	return getterSetter{
		get: func() Value {
			return typedResult.Get(interpreter, getLocationRange, indexingValue)
		},
		set: func(value Value) {
			typedResult.Set(interpreter, getLocationRange, indexingValue, value)
		},
	}
}

// memberExpressionGetterSetter returns a getter/setter function pair
// for the target member expression
//
func (interpreter *Interpreter) memberExpressionGetterSetter(memberExpression *ast.MemberExpression) getterSetter {
	target := interpreter.evalExpression(memberExpression.Expression)
	getLocationRange := locationRangeGetter(interpreter.Location, memberExpression)
	identifier := memberExpression.Identifier.Identifier
	return getterSetter{
		get: func() Value {
			return interpreter.getMember(target, getLocationRange, identifier)
		},
		set: func(value Value) {
			interpreter.setMember(target, getLocationRange, identifier, value)
		},
	}
}

func (interpreter *Interpreter) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	name := expression.Identifier.Identifier
	variable := interpreter.findVariable(name)
	return variable.GetValue()
}

func (interpreter *Interpreter) evalExpression(expression ast.Expression) Value {
	return expression.Accept(interpreter).(Value)
}

func (interpreter *Interpreter) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {
	switch expression.Operation {
	case ast.OperationPlus:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Plus(right)

	case ast.OperationMinus:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Minus(right)

	case ast.OperationMod:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Mod(right)

	case ast.OperationMul:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Mul(right)

	case ast.OperationDiv:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Div(right)

	case ast.OperationBitwiseOr:
		left := interpreter.evalExpression(expression.Left).(IntegerValue)
		right := interpreter.evalExpression(expression.Right).(IntegerValue)
		return left.BitwiseOr(right)

	case ast.OperationBitwiseXor:
		left := interpreter.evalExpression(expression.Left).(IntegerValue)
		right := interpreter.evalExpression(expression.Right).(IntegerValue)
		return left.BitwiseXor(right)

	case ast.OperationBitwiseAnd:
		left := interpreter.evalExpression(expression.Left).(IntegerValue)
		right := interpreter.evalExpression(expression.Right).(IntegerValue)
		return left.BitwiseAnd(right)

	case ast.OperationBitwiseLeftShift:
		left := interpreter.evalExpression(expression.Left).(IntegerValue)
		right := interpreter.evalExpression(expression.Right).(IntegerValue)
		return left.BitwiseLeftShift(right)

	case ast.OperationBitwiseRightShift:
		left := interpreter.evalExpression(expression.Left).(IntegerValue)
		right := interpreter.evalExpression(expression.Right).(IntegerValue)
		return left.BitwiseRightShift(right)

	case ast.OperationLess:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Less(right)

	case ast.OperationLessEqual:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.LessEqual(right)

	case ast.OperationGreater:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.Greater(right)

	case ast.OperationGreaterEqual:
		left := interpreter.evalExpression(expression.Left).(NumberValue)
		right := interpreter.evalExpression(expression.Right).(NumberValue)
		return left.GreaterEqual(right)

	case ast.OperationEqual:
		left := interpreter.evalExpression(expression.Left)
		right := interpreter.evalExpression(expression.Right)
		return interpreter.testEqual(left, right)

	case ast.OperationNotEqual:
		left := interpreter.evalExpression(expression.Left)
		right := interpreter.evalExpression(expression.Right)
		return !interpreter.testEqual(left, right)

	case ast.OperationOr:
		// interpret the left-hand side
		left := interpreter.evalExpression(expression.Left).(BoolValue)
		// only interpret right-hand side if left-hand side is false
		if left {
			return left
		}

		// after interpreting the left-hand side,
		// interpret the right-hand side
		return interpreter.evalExpression(expression.Right).(BoolValue)

	case ast.OperationAnd:
		// interpret the left-hand side
		left := interpreter.evalExpression(expression.Left).(BoolValue)
		// only interpret right-hand side if left-hand side is true
		if !left {
			return left
		}

		// after interpreting the left-hand side,
		// interpret the right-hand side
		return interpreter.evalExpression(expression.Right).(BoolValue)

	case ast.OperationNilCoalesce:
		// interpret the left-hand side
		left := interpreter.evalExpression(expression.Left)

		// only evaluate right-hand side if left-hand side is nil
		if some, ok := left.(*SomeValue); ok {
			return some.Value
		}

		value := interpreter.evalExpression(expression.Right)

		rightType := interpreter.Program.Elaboration.BinaryExpressionRightTypes[expression]
		resultType := interpreter.Program.Elaboration.BinaryExpressionResultTypes[expression]

		// NOTE: important to convert both any and optional
		return interpreter.convertAndBox(value, rightType, resultType)
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindBinary,
		operation: expression.Operation,
		Range:     ast.NewRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) testEqual(left, right Value) BoolValue {
	left = interpreter.unbox(left)
	right = interpreter.unbox(right)

	leftEquatable, ok := left.(EquatableValue)
	if !ok {
		return false
	}

	return BoolValue(leftEquatable.Equal(right, interpreter, true))
}

func (interpreter *Interpreter) VisitUnaryExpression(expression *ast.UnaryExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Expression)

	switch expression.Operation {
	case ast.OperationNegate:
		boolValue := value.(BoolValue)
		return boolValue.Negate()

	case ast.OperationMinus:
		integerValue := value.(NumberValue)
		return integerValue.Negate()

	case ast.OperationMove:
		return value
	}

	panic(&unsupportedOperation{
		kind:      common.OperationKindUnary,
		operation: expression.Operation,
		Range:     ast.NewRangeFromPositioned(expression),
	})
}

func (interpreter *Interpreter) VisitBoolExpression(expression *ast.BoolExpression) ast.Repr {
	return BoolValue(expression.Value)
}

func (interpreter *Interpreter) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return NilValue{}
}

func (interpreter *Interpreter) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
	typ := interpreter.Program.Elaboration.IntegerExpressionType[expression]

	value := expression.Value

	if _, ok := typ.(*sema.AddressType); ok {
		return NewAddressValueFromBytes(value.Bytes())
	}

	// The ranges are checked at the checker level.
	// Hence it is safe to create the value without validation.
	return NewIntValue(value, typ)

}

// NewIntValue creates a Cadence interpreter value of a given subtype.
// This method assumes the range validations are done prior to calling this method. (i.e: at semantic level)
//
func NewIntValue(value *big.Int, intSubType sema.Type) Value {
	switch intSubType {
	case sema.IntType, sema.IntegerType, sema.SignedIntegerType:
		return NewIntValueFromBigInt(value)
	case sema.UIntType:
		return NewUIntValueFromBigInt(value)

	// Int*
	case sema.Int8Type:
		return Int8Value(value.Int64())
	case sema.Int16Type:
		return Int16Value(value.Int64())
	case sema.Int32Type:
		return Int32Value(value.Int64())
	case sema.Int64Type:
		return Int64Value(value.Int64())
	case sema.Int128Type:
		return NewInt128ValueFromBigInt(value)
	case sema.Int256Type:
		return NewInt256ValueFromBigInt(value)

	// UInt*
	case sema.UInt8Type:
		return UInt8Value(value.Int64())
	case sema.UInt16Type:
		return UInt16Value(value.Int64())
	case sema.UInt32Type:
		return UInt32Value(value.Int64())
	case sema.UInt64Type:
		return UInt64Value(value.Int64())
	case sema.UInt128Type:
		return NewUInt128ValueFromBigInt(value)
	case sema.UInt256Type:
		return NewUInt256ValueFromBigInt(value)

	// Word*
	case sema.Word8Type:
		return Word8Value(value.Int64())
	case sema.Word16Type:
		return Word16Value(value.Int64())
	case sema.Word32Type:
		return Word32Value(value.Int64())
	case sema.Word64Type:
		return Word64Value(value.Int64())

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitFixedPointExpression(expression *ast.FixedPointExpression) ast.Repr {
	// TODO: adjust once/if we support more fixed point types

	fixedPointSubType := interpreter.Program.Elaboration.FixedPointExpression[expression]

	value := fixedpoint.ConvertToFixedPointBigInt(
		expression.Negative,
		expression.UnsignedInteger,
		expression.Fractional,
		expression.Scale,
		sema.Fix64Scale,
	)
	switch fixedPointSubType {
	case sema.Fix64Type, sema.SignedFixedPointType:
		return Fix64Value(value.Int64())
	case sema.UFix64Type:
		return UFix64Value(value.Uint64())
	case sema.FixedPointType:
		if expression.Negative {
			return Fix64Value(value.Int64())
		} else {
			return UFix64Value(value.Uint64())
		}
	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitStringExpression(expression *ast.StringExpression) ast.Repr {
	return NewStringValue(expression.Value)
}

func (interpreter *Interpreter) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {
	values := interpreter.visitExpressionsNonCopying(expression.Values)

	argumentTypes := interpreter.Program.Elaboration.ArrayExpressionArgumentTypes[expression]
	elementType := interpreter.Program.Elaboration.ArrayExpressionElementType[expression]

	copies := make([]Value, len(values))
	for i, argument := range values {
		argumentType := argumentTypes[i]
		argumentExpression := expression.Values[i]
		getLocationRange := locationRangeGetter(interpreter.Location, argumentExpression)
		copies[i] = interpreter.copyAndConvert(argument, argumentType, elementType, getLocationRange)
	}

	return NewArrayValueUnownedNonCopying(copies...)
}

func (interpreter *Interpreter) VisitDictionaryExpression(expression *ast.DictionaryExpression) ast.Repr {
	values := interpreter.visitEntries(expression.Entries)

	entryTypes := interpreter.Program.Elaboration.DictionaryExpressionEntryTypes[expression]
	dictionaryType := interpreter.Program.Elaboration.DictionaryExpressionType[expression]

	dictionary := NewDictionaryValueUnownedNonCopying()
	for i, dictionaryEntryValues := range values {
		entryType := entryTypes[i]
		entry := expression.Entries[i]

		key := interpreter.copyAndConvert(
			dictionaryEntryValues.Key,
			entryType.KeyType,
			dictionaryType.KeyType,
			locationRangeGetter(interpreter.Location, entry.Key),
		)

		value := interpreter.copyAndConvert(
			dictionaryEntryValues.Value,
			entryType.ValueType,
			dictionaryType.ValueType,
			locationRangeGetter(interpreter.Location, entry.Value),
		)

		// TODO: panic for duplicate keys?

		// NOTE: important to convert in optional, as assignment to dictionary
		// is always considered as an optional

		getLocationRange := locationRangeGetter(interpreter.Location, expression)

		_ = dictionary.Insert(interpreter, getLocationRange, key, value)
	}

	return dictionary
}

func (interpreter *Interpreter) VisitMemberExpression(expression *ast.MemberExpression) ast.Repr {
	result := interpreter.evalExpression(expression.Expression)
	if expression.Optional {
		switch typedResult := result.(type) {
		case NilValue:
			return typedResult

		case *SomeValue:
			result = typedResult.Value

		default:
			panic(errors.NewUnreachableError())
		}
	}

	getLocationRange := locationRangeGetter(interpreter.Location, expression)
	identifier := expression.Identifier.Identifier
	resultValue := interpreter.getMember(result, getLocationRange, identifier)
	if resultValue == nil {
		panic(MissingMemberValueError{
			Name:          identifier,
			LocationRange: getLocationRange(),
		})
	}

	// If the member access is optional chaining, only wrap the result value
	// in an optional, if it is not already an optional value

	if expression.Optional {
		if _, ok := resultValue.(OptionalValue); !ok {
			resultValue = NewSomeValueOwningNonCopying(resultValue)
		}
	}

	return resultValue
}

func (interpreter *Interpreter) VisitIndexExpression(expression *ast.IndexExpression) ast.Repr {
	typedResult := interpreter.evalExpression(expression.TargetExpression).(ValueIndexableValue)
	indexingValue := interpreter.evalExpression(expression.IndexingExpression)
	getLocationRange := locationRangeGetter(interpreter.Location, expression)
	return typedResult.Get(interpreter, getLocationRange, indexingValue)
}

func (interpreter *Interpreter) VisitConditionalExpression(expression *ast.ConditionalExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Test).(BoolValue)
	if value {
		return interpreter.evalExpression(expression.Then)
	} else {
		return interpreter.evalExpression(expression.Else)
	}
}

func (interpreter *Interpreter) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) ast.Repr {
	// interpret the invoked expression
	result := interpreter.evalExpression(invocationExpression.InvokedExpression)

	// Handle optional chaining on member expression, if any:
	// - If the member expression is nil, finish execution
	// - If the member expression is some value, the wrapped value
	//   is the function value that should be invoked

	isOptionalChaining := false

	if invokedMemberExpression, ok :=
		invocationExpression.InvokedExpression.(*ast.MemberExpression); ok && invokedMemberExpression.Optional {

		isOptionalChaining = true

		switch typedResult := result.(type) {
		case NilValue:
			return typedResult

		case *SomeValue:
			result = typedResult.Value

		default:
			panic(errors.NewUnreachableError())
		}
	}

	function := result.(FunctionValue)

	// NOTE: evaluate all argument expressions in call-site scope, not in function body
	argumentExpressions := make([]ast.Expression, len(invocationExpression.Arguments))
	for i, argument := range invocationExpression.Arguments {
		argumentExpressions[i] = argument.Expression
	}

	arguments := interpreter.visitExpressionsNonCopying(argumentExpressions)

	typeParameterTypes :=
		interpreter.Program.Elaboration.InvocationExpressionTypeArguments[invocationExpression]
	argumentTypes :=
		interpreter.Program.Elaboration.InvocationExpressionArgumentTypes[invocationExpression]
	parameterTypes :=
		interpreter.Program.Elaboration.InvocationExpressionParameterTypes[invocationExpression]

	interpreter.reportFunctionInvocation(invocationExpression)

	resultValue := interpreter.invokeFunctionValue(
		function,
		arguments,
		argumentExpressions,
		argumentTypes,
		parameterTypes,
		typeParameterTypes,
		invocationExpression,
	)

	// If this is invocation is optional chaining, wrap the result
	// as an optional, as the result is expected to be an optional

	if isOptionalChaining {
		resultValue = NewSomeValueOwningNonCopying(resultValue)
	}

	return resultValue
}

func (interpreter *Interpreter) visitExpressionsNonCopying(expressions []ast.Expression) []Value {
	values := make([]Value, 0, len(expressions))

	for _, expression := range expressions {
		value := interpreter.evalExpression(expression)
		values = append(values, value)
	}

	return values
}

func (interpreter *Interpreter) visitEntries(entries []ast.DictionaryEntry) []DictionaryEntryValues {
	values := make([]DictionaryEntryValues, 0, len(entries))

	for _, entry := range entries {
		key := interpreter.evalExpression(entry.Key)
		value := interpreter.evalExpression(entry.Value)

		values = append(
			values,
			DictionaryEntryValues{
				Key:   key,
				Value: value,
			},
		)
	}

	return values
}

func (interpreter *Interpreter) VisitFunctionExpression(expression *ast.FunctionExpression) ast.Repr {

	// lexical scope: variables in functions are bound to what is visible at declaration time
	lexicalScope := interpreter.activations.CurrentOrNew()

	functionType := interpreter.Program.Elaboration.FunctionExpressionFunctionType[expression]

	var preConditions ast.Conditions
	if expression.FunctionBlock.PreConditions != nil {
		preConditions = *expression.FunctionBlock.PreConditions
	}

	var beforeStatements []ast.Statement
	var rewrittenPostConditions ast.Conditions

	if expression.FunctionBlock.PostConditions != nil {
		postConditionsRewrite :=
			interpreter.Program.Elaboration.PostConditionsRewrite[expression.FunctionBlock.PostConditions]

		rewrittenPostConditions = postConditionsRewrite.RewrittenPostConditions
		beforeStatements = postConditionsRewrite.BeforeStatements
	}

	statements := expression.FunctionBlock.Block.Statements

	return InterpretedFunctionValue{
		Interpreter:      interpreter,
		ParameterList:    expression.ParameterList,
		Type:             functionType,
		Activation:       lexicalScope,
		BeforeStatements: beforeStatements,
		PreConditions:    preConditions,
		Statements:       statements,
		PostConditions:   rewrittenPostConditions,
	}
}

func (interpreter *Interpreter) VisitCastingExpression(expression *ast.CastingExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Expression)

	expectedType := interpreter.Program.Elaboration.CastingTargetTypes[expression]

	switch expression.Operation {
	case ast.OperationFailableCast, ast.OperationForceCast:
		dynamicType := value.DynamicType(interpreter, SeenReferences{})
		isSubType := IsSubType(dynamicType, expectedType)

		switch expression.Operation {
		case ast.OperationFailableCast:
			if !isSubType {
				return NilValue{}
			}

			return NewSomeValueOwningNonCopying(value)

		case ast.OperationForceCast:
			if !isSubType {
				getLocationRange := locationRangeGetter(interpreter.Location, expression.Expression)
				panic(
					TypeMismatchError{
						ExpectedType:  expectedType,
						LocationRange: getLocationRange(),
					},
				)
			}

			return value

		default:
			panic(errors.NewUnreachableError())
		}

	case ast.OperationCast:
		staticValueType := interpreter.Program.Elaboration.CastingStaticValueTypes[expression]
		return interpreter.convertAndBox(value, staticValueType, expectedType)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (interpreter *Interpreter) VisitCreateExpression(expression *ast.CreateExpression) ast.Repr {
	return interpreter.evalExpression(expression.InvocationExpression)
}

func (interpreter *Interpreter) VisitDestroyExpression(expression *ast.DestroyExpression) ast.Repr {
	value := interpreter.evalExpression(expression.Expression)

	getLocationRange := locationRangeGetter(interpreter.Location, expression)

	value.(DestroyableValue).Destroy(interpreter, getLocationRange)

	return VoidValue{}
}

func (interpreter *Interpreter) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) ast.Repr {

	borrowType := interpreter.Program.Elaboration.ReferenceExpressionBorrowTypes[referenceExpression]

	result := interpreter.evalExpression(referenceExpression.Expression)

	return &EphemeralReferenceValue{
		Authorized:   borrowType.Authorized,
		Value:        result,
		BorrowedType: borrowType.Type,
	}
}

func (interpreter *Interpreter) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {
	result := interpreter.evalExpression(expression.Expression)

	switch result := result.(type) {
	case *SomeValue:
		return result.Value

	case NilValue:
		panic(
			ForceNilError{
				LocationRange: LocationRange{
					Location: interpreter.Location,
					Range: ast.Range{
						StartPos: expression.EndPosition(),
						EndPos:   expression.EndPosition(),
					},
				},
			},
		)

	default:
		return result
	}
}

func (interpreter *Interpreter) VisitPathExpression(expression *ast.PathExpression) ast.Repr {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)

	return PathValue{
		Domain:     domain,
		Identifier: expression.Identifier.Identifier,
	}
}

func (interpreter *Interpreter) evalPotentialResourceMoveIndexExpression(expression ast.Expression) Value {
	resourceMoveIndexExpression := interpreter.resourceMoveIndexExpression(expression)
	if resourceMoveIndexExpression == nil {
		return interpreter.evalExpression(expression)
	}

	getterSetter := interpreter.indexExpressionGetterSetter(resourceMoveIndexExpression)
	value := getterSetter.get()
	getterSetter.set(NilValue{})
	return value
}

func (interpreter *Interpreter) resourceMoveIndexExpression(expression ast.Expression) *ast.IndexExpression {
	indexExpression, ok := expression.(*ast.IndexExpression)
	if !ok || !interpreter.Program.Elaboration.IsResourceMoveIndexExpression[indexExpression] {
		return nil
	}

	return indexExpression
}
