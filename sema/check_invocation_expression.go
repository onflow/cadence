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

package sema

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

func (checker *Checker) VisitInvocationExpression(invocationExpression *ast.InvocationExpression) Type {
	ty := checker.checkInvocationExpression(invocationExpression)

	if !checker.checkInvokedExpression(ty, invocationExpression) {
		return InvalidType
	}

	return ty
}

func (checker *Checker) checkInvokedExpression(ty Type, pos ast.HasPosition) bool {

	// Check if the invoked expression can be invoked.
	// Composite types cannot be invoked directly,
	// only through respective statements (emit, attach).
	//
	// If the invoked expression is an optional type,
	// for example in the case of optional chaining,
	// then check the wrapped type.

	maybeCompositeType := ty
	if optionalType, ok := ty.(*OptionalType); ok {
		maybeCompositeType = optionalType.Type
	}

	if compositeType, ok := maybeCompositeType.(*CompositeType); ok {
		switch compositeType.Kind {
		// Events cannot be invoked without an emit statement
		case common.CompositeKindEvent:
			checker.report(
				&InvalidEventUsageError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
				},
			)
			return false

		// Attachments cannot be constructed without an attach statement
		case common.CompositeKindAttachment:
			checker.report(
				&InvalidAttachmentUsageError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, pos),
				},
			)
			return false
		}
	}

	return true
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
	expressionType := checker.VisitExpression(invokedExpression, invocationExpression, nil)

	// `inInvocation` should be reset before visiting arguments
	checker.inInvocation = false

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

		argumentCount := len(invocationExpression.Arguments)
		if argumentCount > 0 {
			argumentTypes = make([]Type, 0, argumentCount)

			for _, argument := range invocationExpression.Arguments {
				argumentType := checker.VisitExpression(argument.Expression, invocationExpression, nil)
				argumentTypes = append(argumentTypes, argumentType)
			}

			checker.Elaboration.SetInvocationExpressionTypes(
				invocationExpression,
				InvocationExpressionTypes{
					ArgumentTypes: argumentTypes,
					ReturnType:    checker.expectedType,
				},
			)
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
		returnInfo.DefinitelyExited = true
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

	valueType := checker.Elaboration.IdentifierInInvocationType(invocationIdentifierExpression)
	if valueType == nil {
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
	_, _, member, _ := checker.visitMember(memberExpression, false)

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
	arity := functionType.Arity
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
		arity,
		invocationExpression,
	)

	minCount := argumentCount
	if parameterCount < argumentCount {
		minCount = parameterCount
	}

	var parameterTypes []Type

	if argumentCount > 0 {
		argumentTypes = make([]Type, argumentCount)
		parameterTypes = make([]Type, argumentCount)

		// Check all the required arguments

		for argumentIndex := 0; argumentIndex < minCount; argumentIndex++ {

			parameterTypes[argumentIndex] =
				checker.checkInvocationRequiredArgument(
					invocationExpression,
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
			argumentTypes[i] = checker.VisitExpression(argument.Expression, invocationExpression, nil)
		}
	}

	// The invokable type might have special checks for the arguments

	if functionType.ArgumentExpressionsCheck != nil && argumentCount > 0 {
		argumentExpressions := make([]ast.Expression, argumentCount)
		for i, argument := range invocationExpression.Arguments {
			argumentExpressions[i] = argument.Expression
		}

		functionType.ArgumentExpressionsCheck(
			checker,
			argumentExpressions,
			invocationExpression,
		)
	}

	returnType = functionType.ReturnTypeAnnotation.Type.Resolve(typeArguments)
	if returnType == nil {
		checker.report(&InvocationTypeInferenceError{
			Range: ast.NewRangeFromPositioned(
				checker.memoryGauge,
				invocationExpression,
			),
		})

		returnType = InvalidType
	}

	// Check all type parameters have been bound to a type.

	checker.checkTypeParameterInference(
		functionType,
		typeArguments,
		invocationExpression,
	)

	// The invokable type might have special checks for the type parameters.

	if functionType.TypeArgumentsCheck != nil {
		functionType.TypeArgumentsCheck(
			checker.memoryGauge,
			typeArguments,
			invocationExpression.TypeArguments,
			invocationExpression,
			checker.report,
		)
	}

	// Save types in the elaboration

	checker.Elaboration.SetInvocationExpressionTypes(
		invocationExpression,
		InvocationExpressionTypes{
			TypeArguments:  typeArguments,
			ParameterTypes: parameterTypes,
			ReturnType:     returnType,
			ArgumentTypes:  argumentTypes,
		},
	)

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
	invocationExpression *ast.InvocationExpression,
	argumentIndex int,
	functionType *FunctionType,
	argumentTypes []Type,
	typeParameters *TypeParameterTypeOrderedMap,
) (
	parameterType Type,
) {
	argument := invocationExpression.Arguments[argumentIndex]

	parameter := functionType.Parameters[argumentIndex]
	parameterType = parameter.TypeAnnotation.Type

	var argumentType Type

	typeParameterCount := len(functionType.TypeParameters)

	// If all type parameters have been bound to a type,
	// then resolve the parameter type with the type arguments,
	// and propose the parameter type as the expected type for the argument.
	if typeParameters.Len() == typeParameterCount {

		// Optimization: only resolve if there are type parameters.
		// This avoids unnecessary work for non-generic functions.
		if typeParameterCount > 0 {
			parameterType = parameterType.Resolve(typeParameters)
			// If the type parameter could not be resolved, use the invalid type.
			if parameterType == nil {
				checker.report(&InvocationTypeInferenceError{
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						argument.Expression,
					),
				})
				parameterType = InvalidType
			}
		}

		// If the parameter type is or contains a reference type in particular,
		// we do NOT use it as the expected type,
		// to require an explicit type annotation in the invocation.
		//
		// This is done to avoid the following situation:

		// For arguments in invocations, the parameter type is not obvious at the call-site,
		// which is potentially dangerous if the function is defined in a different location,
		// and the parameter type potentially requires an authorization.
		//
		// For example, consider:
		//
		//   // defined elsewhere, is going to mutate the passed array
		//   fun foo(ints: auth(Mutate) &[Int]) {}
		//
		//   let ints = [1, 2, 3]
		//   // would implicitly allow mutation
		//   foo(&ints)
		//
		// A programmer should be able to look at a piece of code,
		// and reason locally about whether a type will be inferred for a value
		// based solely on how that value is used syntactically,
		// not needing to worry about the actual or expected type of the value.
		//
		// Requiring an explicit type for *all* references,
		// independent of if they require an authorization or not,
		// is simple and allows the developer to see locally, and purely syntactically,
		// that they are passing a reference and thus must annotate it.

		expectedType := parameterType
		if parameterType.IsOrContainsReferenceType() {
			expectedType = nil
		}

		argumentType = checker.VisitExpression(argument.Expression, invocationExpression, expectedType)

		// If we did not pass an expected type,
		// we must manually check that the argument type and the parameter type are compatible.

		if expectedType == nil {
			// Check that the type of the argument matches the type of the parameter.

			checker.checkInvocationArgumentParameterTypeCompatibility(
				argument.Expression,
				argumentType,
				parameterType,
			)
		}

	} else {
		// If there are still type parameters that have not been bound to a type,
		// then check the argument without an expected type.
		//
		// We will then have to manually check that the argument type is compatible
		// with the parameter type (see below).

		argumentType = checker.VisitExpression(argument.Expression, invocationExpression, nil)

		// Try to unify the parameter type with the argument type.
		// If unification fails, fall back to the parameter type for now.

		if parameterType.Unify(
			argumentType,
			typeParameters,
			checker.report,
			checker.memoryGauge,
			argument.Expression,
		) {
			parameterType = parameterType.Resolve(typeParameters)
			// If the type parameter could not be resolved, use the invalid type.
			if parameterType == nil {
				checker.report(&InvocationTypeInferenceError{
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						argument.Expression,
					),
				})
				parameterType = InvalidType
			}
		}

		// Check that the type of the argument matches the type of the parameter.

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
	arity *Arity,
	pos ast.HasPosition,
) {
	minCount := arity.MinCount(parameterCount)
	if argumentCount < minCount {
		checker.report(
			&InsufficientArgumentsError{
				MinCount:    minCount,
				ActualCount: argumentCount,
				Range:       ast.NewRangeFromPositioned(checker.memoryGauge, pos),
			},
		)
		return
	}

	maxCount := arity.MaxCount(parameterCount)
	if maxCount != nil && argumentCount > *maxCount {
		checker.report(
			&ExcessiveArgumentsError{
				MaxCount:    *maxCount,
				ActualCount: argumentCount,
				Range:       ast.NewRangeFromPositioned(checker.memoryGauge, pos),
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

		err := typeParameter.checkTypeBound(ty, checker.memoryGauge, rawTypeArgument)
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
	checker.checkResourceMoveOperation(argument, argumentType)
	return argumentType
}
