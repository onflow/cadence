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

func (checker *Checker) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) Type {
	ty := checker.checkInvocationExpression(invocationExpression)

	// Events cannot be invoked without an emit statement

	if compositeType, ok := ty.(*CompositeType); ok &&
		compositeType.Kind == common.CompositeKindEvent {

		checker.report(
			&InvalidEventUsageError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, invocationExpression),
			},
		)
		return InvalidType
	}

	return ty
}

func (checker *Checker) checkInvocationExpression(invocationExpression *ast.InvocationExpression) Type {
	inCreate := checker.inCreate
	checker.inCreate = false
	defer func() {
		checker.inCreate = inCreate
	}()

	inInvocation := checker.inInvocation
	checker.inInvocation = true
	defer func() {
		checker.inInvocation = inInvocation
	}()

	// check the invoked expression can be invoked

	invokedExpression := invocationExpression.InvokedExpression
	expressionType := checker.VisitExpression(invokedExpression, nil)

	// Get the member from the invoked value
	// based on the use of optional chaining syntax

	isOptionalChainingResult := false
	if memberExpression, ok := invokedExpression.(*ast.MemberExpression); ok {

		// If the member expression is using optional chaining,
		// check if the invoked type is optional

		isOptionalChainingResult = memberExpression.Optional
		if isOptionalChainingResult {
			if optionalExpressionType, ok := expressionType.(*OptionalType); ok {

				// The invoked type is optional, get the type from the wrapped type
				expressionType = optionalExpressionType.Type
			}
		}
	}

	var argumentTypes []Type

	functionType, ok := expressionType.(*FunctionType)
	if !ok {
		if !expressionType.IsInvalidType() {
			checker.report(
				&NotCallableError{
					Type:  expressionType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, invokedExpression),
				},
			)
		}

		argumentTypes = make([]Type, 0, len(invocationExpression.Arguments))

		for _, argument := range invocationExpression.Arguments {
			argumentType := checker.VisitExpression(argument.Expression, nil)
			argumentTypes = append(argumentTypes, argumentType)
		}

		checker.Elaboration.InvocationExpressionTypes[invocationExpression] =
			InvocationExpressionTypes{
				ArgumentTypes: argumentTypes,
				ReturnType:    checker.expectedType,
			}

		return InvalidType
	}
	checker.EnforcePurity(invocationExpression, functionType.Purity)

	// The invoked expression has a function type,
	// check the invocation including all arguments.
	//
	// If the invocation is on a member expression which is optional chaining,'
	// then `isOptionalChainingResult` is true, which means the invocation
	// is only potential, i.e. the invocation will not always

	var returnType Type

	checkInvocation := func() {
		argumentTypes, returnType =
			checker.checkInvocation(invocationExpression, functionType)
	}

	if isOptionalChainingResult {
		_ = checker.checkPotentiallyUnevaluated(func() Type {
			checkInvocation()
			// ignored
			return nil
		})
	} else {
		checkInvocation()
	}

	arguments := invocationExpression.Arguments

	if checker.PositionInfo != nil && len(arguments) > 0 {
		checker.PositionInfo.recordFunctionInvocation(
			invocationExpression,
			functionType,
		)
	}

	// If the invocation refers directly to the name of the function as stated in the declaration,
	// or the invocation refers to a function of a composite (member),
	// check that the correct argument labels are supplied in the invocation

	switch typedInvokedExpression := invokedExpression.(type) {
	case *ast.IdentifierExpression:
		checker.checkIdentifierInvocationArgumentLabels(
			invocationExpression,
			typedInvokedExpression,
		)

	case *ast.MemberExpression:
		checker.checkMemberInvocationArgumentLabels(
			invocationExpression,
			typedInvokedExpression,
		)
	}

	checker.checkConstructorInvocationWithResourceResult(
		invocationExpression,
		functionType,
		returnType,
		inCreate,
	)

	checker.checkMemberInvocationResourceInvalidation(invokedExpression)

	// Update the return info for invocations that do not return (i.e. have a `Never` return type)

	if returnType == NeverType {
		returnInfo := checker.functionActivations.Current().ReturnInfo
		returnInfo.DefinitelyHalted = true
	}

	if isOptionalChainingResult {
		return wrapWithOptionalIfNotNil(returnType)
	}
	return returnType
}

func (checker *Checker) checkMemberInvocationResourceInvalidation(invokedExpression ast.Expression) {
	// If the invocation is on a resource, i.e., a member expression where the accessed expression
	// is an identifier which refers to a resource, then the resource is temporarily "moved into"
	// the function and back out after the invocation.
	//
	// So record a *temporary* invalidation to get the resource that is invalidated,
	// remove the invalidation because it is temporary, and check if the use is potentially invalid,
	// because the resource was already invalidated.
	//
	// Perform this check *after* the arguments where checked:
	// Even though a duplicated use of the resource in an argument is invalid, e.g. `foo.bar(<-foo)`,
	// the arguments might just use to the temporarily moved resource, e.g. `foo.bar(foo.baz)`
	// and not invalidate it.

	invokedMemberExpression, ok := invokedExpression.(*ast.MemberExpression)
	if !ok {
		return
	}
	invocationIdentifierExpression, ok := invokedMemberExpression.Expression.(*ast.IdentifierExpression)
	if !ok {
		return
	}

	// Check that an entry for `IdentifierInInvocationTypes` exists,
	// because the entry might be missing if the invocation was on a non-existent variable

	valueType, ok := checker.Elaboration.IdentifierInInvocationTypes[invocationIdentifierExpression]
	if !ok {
		return
	}

	invalidation := checker.recordResourceInvalidation(
		invocationIdentifierExpression,
		valueType,
		ResourceInvalidationKindMoveTemporary,
	)

	if invalidation == nil {
		return
	}

	checker.resources.RemoveTemporaryMoveInvalidation(
		invalidation.resource,
		invalidation.invalidation,
	)

	checker.checkResourceUseAfterInvalidation(
		invalidation.resource,
		invocationIdentifierExpression,
	)
}

func (checker *Checker) checkConstructorInvocationWithResourceResult(
	invocationExpression *ast.InvocationExpression,
	functionType *FunctionType,
	returnType Type,
	inCreate bool,
) {
	if !functionType.IsConstructor {
		return
	}

	// NOTE: not using `isResourceType`,
	// as only direct resource types can be constructed

	if compositeReturnType, ok := returnType.(*CompositeType); !ok ||
		compositeReturnType.Kind != common.CompositeKindResource {

		return
	}

	if inCreate {
		return
	}

	checker.report(
		&MissingCreateError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, invocationExpression),
		},
	)
}

func (checker *Checker) checkIdentifierInvocationArgumentLabels(
	invocationExpression *ast.InvocationExpression,
	identifierExpression *ast.IdentifierExpression,
) {
	variable := checker.findAndCheckValueVariable(identifierExpression, false)

	if variable == nil || len(variable.ArgumentLabels) == 0 {
		return
	}

	checker.checkInvocationArgumentLabels(
		invocationExpression.Arguments,
		variable.ArgumentLabels,
	)
}

func (checker *Checker) checkMemberInvocationArgumentLabels(
	invocationExpression *ast.InvocationExpression,
	memberExpression *ast.MemberExpression,
) {
	_, member, _ := checker.visitMember(memberExpression)

	if member == nil || len(member.ArgumentLabels) == 0 {
		return
	}

	checker.checkInvocationArgumentLabels(
		invocationExpression.Arguments,
		member.ArgumentLabels,
	)
}

func (checker *Checker) checkInvocationArgumentLabels(
	arguments []*ast.Argument,
	argumentLabels []string,
) {
	argumentCount := len(arguments)

	for i, argumentLabel := range argumentLabels {
		if i >= argumentCount {
			break
		}

		argument := arguments[i]
		providedLabel := argument.Label
		if argumentLabel == ArgumentLabelNotRequired {
			// argument label is not required,
			// check it is not provided

			if providedLabel != "" {
				checker.report(
					&IncorrectArgumentLabelError{
						ActualArgumentLabel:   providedLabel,
						ExpectedArgumentLabel: "",
						Range: ast.NewRange(
							checker.memoryGauge,
							*argument.LabelStartPos,
							*argument.LabelEndPos,
						),
					},
				)
			}
		} else {
			// argument label is required,
			// check it is provided and correct
			if providedLabel == "" {
				checker.report(
					&MissingArgumentLabelError{
						ExpectedArgumentLabel: argumentLabel,
						Range:                 ast.NewRangeFromPositioned(checker.memoryGauge, argument.Expression),
					},
				)
			} else if providedLabel != argumentLabel {
				checker.report(
					&IncorrectArgumentLabelError{
						ActualArgumentLabel:   providedLabel,
						ExpectedArgumentLabel: argumentLabel,
						Range: ast.NewRange(
							checker.memoryGauge,
							*argument.LabelStartPos,
							*argument.LabelEndPos,
						),
					},
				)
			}
		}
	}
}

func (checker *Checker) checkInvocation(
	invocationExpression *ast.InvocationExpression,
	functionType *FunctionType,
) (
	argumentTypes []Type,
	returnType Type,
) {
	parameterCount := len(functionType.Parameters)
	requiredArgumentCount := functionType.RequiredArgumentCount
	typeParameterCount := len(functionType.TypeParameters)

	// Check the type arguments and bind them to type parameters

	typeArgumentCount := len(invocationExpression.TypeArguments)

	typeArguments := &TypeParameterTypeOrderedMap{}

	// If the function type is generic, the invocation might provide
	// explicit type arguments for the type parameters.

	// Check that the number of type arguments does not exceed
	// the number of type parameters

	validTypeArgumentCount := typeArgumentCount

	if typeArgumentCount > typeParameterCount {

		validTypeArgumentCount = typeParameterCount

		checker.reportInvalidTypeArgumentCount(
			typeArgumentCount,
			typeParameterCount,
			invocationExpression.TypeArguments,
		)
	}

	// Check all non-superfluous type arguments
	// and bind them to the type parameters

	validTypeArguments := invocationExpression.TypeArguments[:validTypeArgumentCount]

	checker.checkAndBindGenericTypeParameterTypeArguments(
		validTypeArguments,
		functionType.TypeParameters,
		typeArguments,
	)

	// Check that the invocation's argument count matches the function's parameter count

	argumentCount := len(invocationExpression.Arguments)

	// TODO: only pass position of arguments, not whole invocation
	checker.checkInvocationArgumentCount(
		argumentCount,
		parameterCount,
		requiredArgumentCount,
		invocationExpression,
	)

	minCount := argumentCount
	if parameterCount < argumentCount {
		minCount = parameterCount
	}

	argumentTypes = make([]Type, argumentCount)
	parameterTypes := make([]Type, argumentCount)

	// Check all the required arguments

	for argumentIndex := 0; argumentIndex < minCount; argumentIndex++ {

		parameterTypes[argumentIndex] =
			checker.checkInvocationRequiredArgument(
				invocationExpression.Arguments,
				argumentIndex,
				functionType,
				argumentTypes,
				typeArguments,
			)
	}

	// Add extra argument types

	for i := minCount; i < argumentCount; i++ {
		argument := invocationExpression.Arguments[i]
		// TODO: pass the expected type to support type inferring for parameters
		argumentTypes[i] = checker.VisitExpression(argument.Expression, nil)
	}

	// The invokable type might have special checks for the arguments

	argumentExpressions := make([]ast.Expression, argumentCount)
	for i, argument := range invocationExpression.Arguments {
		argumentExpressions[i] = argument.Expression
	}

	functionType.CheckArgumentExpressions(
		checker,
		argumentExpressions,
		ast.NewRangeFromPositioned(checker.memoryGauge, invocationExpression),
	)

	returnType = functionType.ReturnTypeAnnotation.Type.Resolve(typeArguments)
	if returnType == nil {
		// TODO: report error? does `checkTypeParameterInference` below already do that?
		returnType = InvalidType
	}

	// Check all type parameters have been bound to a type.

	checker.checkTypeParameterInference(
		functionType,
		typeArguments,
		invocationExpression,
	)

	// Save types in the elaboration

	checker.Elaboration.InvocationExpressionTypes[invocationExpression] = InvocationExpressionTypes{
		TypeArguments:      typeArguments,
		TypeParameterTypes: parameterTypes,
		ReturnType:         returnType,
		ArgumentTypes:      argumentTypes,
	}

	return argumentTypes, returnType
}

// checkTypeParameterInference checks that all type parameters
// of the given generic function type have been assigned a type.
func (checker *Checker) checkTypeParameterInference(
	functionType *FunctionType,
	typeArguments *TypeParameterTypeOrderedMap,
	invocationExpression *ast.InvocationExpression,
) {
	for _, typeParameter := range functionType.TypeParameters {

		if ty, ok := typeArguments.Get(typeParameter); ok && ty != nil {
			continue
		}

		// If the type parameter is not required, continue

		if typeParameter.Optional {
			continue
		}

		checker.report(
			&TypeParameterTypeInferenceError{
				Name:  typeParameter.Name,
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, invocationExpression),
			},
		)
	}
}

func (checker *Checker) checkInvocationRequiredArgument(
	arguments ast.Arguments,
	argumentIndex int,
	functionType *FunctionType,
	argumentTypes []Type,
	typeParameters *TypeParameterTypeOrderedMap,
) (
	parameterType Type,
) {
	argument := arguments[argumentIndex]

	parameter := functionType.Parameters[argumentIndex]
	parameterType = parameter.TypeAnnotation.Type

	var argumentType Type

	if len(functionType.TypeParameters) == 0 {
		// If the function doesn't use generic types, then the
		// param types can be used to infer the types for arguments.
		argumentType = checker.VisitExpression(argument.Expression, parameterType)
	} else {
		// TODO: pass the expected type to support for parameters
		argumentType = checker.VisitExpression(argument.Expression, nil)

		// Try to unify the parameter type with the argument type.
		// If unification fails, fall back to the parameter type for now.

		argumentRange := ast.NewRangeFromPositioned(checker.memoryGauge, argument.Expression)

		if parameterType.Unify(argumentType, typeParameters, checker.report, argumentRange) {
			parameterType = parameterType.Resolve(typeParameters)
			if parameterType == nil {
				parameterType = InvalidType
			}
		}

		// Check that the type of the argument matches the type of the parameter.

		// TODO: remove this once type inferring support for parameters is added
		checker.checkInvocationArgumentParameterTypeCompatibility(
			argument.Expression,
			argumentType,
			parameterType,
		)
	}

	argumentTypes[argumentIndex] = argumentType

	checker.checkInvocationArgumentMove(argument.Expression, argumentType)

	return parameterType
}

func (checker *Checker) checkInvocationArgumentCount(
	argumentCount int,
	parameterCount int,
	requiredArgumentCount *int,
	pos ast.HasPosition,
) {

	if argumentCount == parameterCount {
		return
	}

	// TODO: improve
	if requiredArgumentCount == nil ||
		argumentCount < *requiredArgumentCount {

		checker.report(
			&ArgumentCountError{
				ParameterCount: parameterCount,
				ArgumentCount:  argumentCount,
				Range:          ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)
	}
}

func (checker *Checker) reportInvalidTypeArgumentCount(
	typeArgumentCount int,
	typeParameterCount int,
	allTypeArguments []*ast.TypeAnnotation,
) {
	exceedingTypeArgumentIndexStart := typeArgumentCount - typeParameterCount - 1

	firstSuperfluousTypeArgument :=
		allTypeArguments[exceedingTypeArgumentIndexStart]

	lastSuperfluousTypeArgument :=
		allTypeArguments[typeArgumentCount-1]

	checker.report(
		&InvalidTypeArgumentCountError{
			TypeParameterCount: typeParameterCount,
			TypeArgumentCount:  typeArgumentCount,
			Range: ast.NewRange(
				checker.memoryGauge,
				firstSuperfluousTypeArgument.StartPosition(),
				lastSuperfluousTypeArgument.EndPosition(checker.memoryGauge),
			),
		},
	)
}

func (checker *Checker) checkAndBindGenericTypeParameterTypeArguments(
	typeArguments []*ast.TypeAnnotation,
	typeParameters []*TypeParameter,
	typeParameterTypes *TypeParameterTypeOrderedMap,
) {
	for i := 0; i < len(typeArguments); i++ {
		rawTypeArgument := typeArguments[i]

		typeArgument := checker.ConvertTypeAnnotation(rawTypeArgument)
		checker.checkTypeAnnotation(typeArgument, rawTypeArgument)

		ty := typeArgument.Type

		// Don't check or bind invalid type arguments

		if ty.IsInvalidType() {
			continue
		}

		typeParameter := typeParameters[i]

		// If the type parameter corresponding to the type argument has a type bound,
		// then check that the argument is a subtype of the type bound.

		err := typeParameter.checkTypeBound(ty, ast.NewRangeFromPositioned(checker.memoryGauge, rawTypeArgument))
		checker.report(err)

		// Bind the type argument to the type parameter

		typeParameterTypes.Set(typeParameter, ty)
	}
}

func (checker *Checker) checkInvocationArgumentParameterTypeCompatibility(
	argument ast.Expression,
	argumentType, parameterType Type,
) {

	if argumentType.IsInvalidType() ||
		parameterType.IsInvalidType() {

		return
	}

	if !checker.checkTypeCompatibility(argument, argumentType, parameterType) {

		checker.report(
			&TypeMismatchError{
				ExpectedType: parameterType,
				ActualType:   argumentType,
				Range:        ast.NewRangeFromPositioned(checker.memoryGauge, argument),
			},
		)
	}
}

func (checker *Checker) checkInvocationArgumentMove(argument ast.Expression, argumentType Type) Type {

	checker.checkVariableMove(argument)
	checker.checkResourceMoveOperation(argument, argumentType)

	return argumentType
}
