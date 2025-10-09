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

func PurityFromAnnotation(purity ast.FunctionPurity) FunctionPurity {
	if purity == ast.FunctionPurityView {
		return FunctionPurityView
	}
	return FunctionPurityImpure

}

func (checker *Checker) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, _ bool) (_ struct{}) {
	checker.visitFunctionDeclaration(
		declaration,
		functionDeclarationOptions{
			mustExit:          true,
			declareFunction:   true,
			checkResourceLoss: true,
		},
		nil,
	)

	return
}

func (checker *Checker) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) struct{} {
	return checker.VisitFunctionDeclaration(declaration.FunctionDeclaration, false)
}

type functionDeclarationOptions struct {
	// mustExit specifies if the function declaration's function block
	// should be checked for containing proper return statements.
	// This check may be omitted in e.g. function declarations of interfaces
	mustExit bool
	// declareFunction specifies if the function should also be declared in
	// the current scope. This might be e.g. true for global function
	// declarations, but false for function declarations of composites
	declareFunction bool
	// checkResourceLoss if the function should be checked for resource loss.
	// For example, function declarations in interfaces should not be checked.
	checkResourceLoss bool
}

func (checker *Checker) visitFunctionDeclaration(
	declaration *ast.FunctionDeclaration,
	options functionDeclarationOptions,
	containerKind *common.CompositeKind,
) {

	checker.checkStaticModifier(
		declaration.IsStatic(),
		declaration.Identifier,
	)

	checker.checkNativeModifier(
		declaration.IsNative(),
		declaration.Identifier,
	)

	functionBlock := declaration.FunctionBlock

	if declaration.IsNative() {
		if !functionBlock.IsEmpty() {
			checker.report(&NativeFunctionWithImplementationError{
				Range: ast.NewRangeFromPositioned(
					checker.memoryGauge,
					functionBlock,
				),
			})
		}

		functionBlock = nil
	}

	// global functions were previously declared, see `declareFunctionDeclaration`

	functionType := checker.Elaboration.FunctionDeclarationFunctionType(declaration)
	access := checker.accessFromAstAccess(declaration.Access)

	if functionType == nil {

		functionType = checker.functionType(
			declaration.IsNative(),
			declaration.Purity,
			declaration.TypeParameterList,
			declaration.ParameterList,
			declaration.ReturnTypeAnnotation,
		)

		if options.declareFunction {
			checker.declareFunctionDeclaration(declaration, functionType)
		}
	}

	checker.checkDeclarationAccessModifier(
		access,
		declaration.DeclarationKind(),
		functionType,
		containerKind,
		declaration.StartPos,
		true,
	)

	checker.Elaboration.SetFunctionDeclarationFunctionType(declaration, functionType)

	checker.checkFunction(
		declaration.ParameterList,
		declaration.ReturnTypeAnnotation,
		access,
		functionType,
		functionBlock,
		options.mustExit,
		nil,
		options.checkResourceLoss,
	)
}

func (checker *Checker) declareFunctionDeclaration(
	declaration *ast.FunctionDeclaration,
	functionType *FunctionType,
) {
	argumentLabels := declaration.ParameterList.EffectiveArgumentLabels()

	_, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               declaration.Identifier.Identifier,
		ty:                       functionType,
		docString:                declaration.DocString,
		access:                   checker.accessFromAstAccess(declaration.Access),
		kind:                     common.DeclarationKindFunction,
		pos:                      declaration.Identifier.Pos,
		isConstant:               true,
		argumentLabels:           argumentLabels,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)

	if checker.PositionInfo != nil {
		checker.recordFunctionDeclarationOrigin(declaration, functionType)
	}
}

func (checker *Checker) checkFunction(
	parameterList *ast.ParameterList,
	returnTypeAnnotation *ast.TypeAnnotation,
	access Access,
	functionType *FunctionType,
	functionBlock *ast.FunctionBlock,
	mustExit bool,
	initializationInfo *InitializationInfo,
	checkResourceLoss bool,
) {
	// check argument labels
	checker.checkArgumentLabels(parameterList)

	checker.checkParameters(parameterList, functionType.Parameters)

	if functionType.ReturnTypeAnnotation.Type != nil {
		checker.checkTypeAnnotation(functionType.ReturnTypeAnnotation, returnTypeAnnotation)
	}

	// NOTE: Always declare the function parameters, even if the function body is empty.
	// For example, event declarations have an initializer with an empty body,
	// but their parameters (e.g. duplication) needs to still be checked.

	checker.functionActivations.WithFunction(
		functionType,
		checker.valueActivations.Depth(),
		func(functionActivation *FunctionActivation) {
			// NOTE: important to begin scope in function activation, so that
			//   variable declarations will have proper function activation
			//   associated to it, and declare parameters in this new scope

			var endPosGetter EndPositionGetter
			if functionBlock != nil {
				endPosGetter = functionBlock.EndPosition
			}

			checker.enterValueScope()
			defer func() {
				checkResourceLoss := checkResourceLoss &&
					!functionActivation.ReturnInfo.DefinitelyHalted
				checker.leaveValueScope(endPosGetter, checkResourceLoss)
			}()

			checker.declareParameters(parameterList, functionType.Parameters)

			functionActivation.InitializationInfo = initializationInfo

			if functionBlock != nil {
				checker.InNewPurityScope(functionType.Purity == FunctionPurityView, func() {
					checker.visitFunctionBlock(
						functionBlock,
						functionType.ReturnTypeAnnotation.Type,
						returnTypeAnnotation,
						checkResourceLoss,
					)
				})

				if mustExit {
					returnType := functionType.ReturnTypeAnnotation.Type
					checker.checkFunctionExits(functionBlock, returnType)
				}
			}

			if initializationInfo != nil {
				checker.checkFieldMembersInitialized(initializationInfo)
			}
		},
	)

	if checker.PositionInfo != nil && functionBlock != nil {
		startPos := functionBlock.StartPosition()
		endPos := functionBlock.EndPosition(checker.memoryGauge)

		for _, parameter := range functionType.Parameters {
			checker.PositionInfo.recordParameterRange(startPos, endPos, parameter)
		}
	}
}

// checkFunctionExits checks that the given function block exits
// with a return-type appropriate return statement.
// The return is not needed if the function has a `Void` return type.
func (checker *Checker) checkFunctionExits(functionBlock *ast.FunctionBlock, returnType Type) {

	if returnType == VoidType {
		return
	}

	functionActivation := checker.functionActivations.Current()

	// NOTE: intentionally NOT DefinitelyReturned || DefinitelyHalted,
	// see DefinitelyExited
	if functionActivation.ReturnInfo.DefinitelyExited {
		return
	}

	checker.report(
		&MissingReturnStatementError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, functionBlock),
		},
	)
}

func (checker *Checker) checkParameters(parameterList *ast.ParameterList, parameters []Parameter) {
	for i, parameter := range parameterList.Parameters {
		parameterTypeAnnotation := parameters[i].TypeAnnotation

		checker.checkTypeAnnotation(
			parameterTypeAnnotation,
			parameter.TypeAnnotation,
		)
	}
}

// checkArgumentLabels checks that all argument labels (if any) are unique
func (checker *Checker) checkArgumentLabels(parameterList *ast.ParameterList) {

	argumentLabelPositions := map[string]ast.Position{}

	for _, parameter := range parameterList.Parameters {
		label := parameter.Label
		if label == "" || label == ArgumentLabelNotRequired {
			continue
		}

		labelPos := parameter.StartPos

		if previousPos, ok := argumentLabelPositions[label]; ok {
			checker.report(
				&RedeclarationError{
					Kind:        common.DeclarationKindArgumentLabel,
					Name:        label,
					Pos:         labelPos,
					PreviousPos: &previousPos,
				},
			)
		}

		argumentLabelPositions[label] = labelPos
	}
}

// declareParameters declares a constant for each parameter,
// ensuring names are unique and constants don't already exist
func (checker *Checker) declareParameters(
	parameterList *ast.ParameterList,
	parameters []Parameter,
) {
	depth := checker.valueActivations.Depth()

	for i, parameter := range parameterList.Parameters {
		identifier := parameter.Identifier

		// check if variable with this identifier is already declared in the current scope
		existingVariable := checker.valueActivations.Find(identifier.Identifier)
		if existingVariable != nil && existingVariable.ActivationDepth == depth {
			checker.report(
				&RedeclarationError{
					Kind:        common.DeclarationKindParameter,
					Name:        identifier.Identifier,
					Pos:         identifier.Pos,
					PreviousPos: existingVariable.Pos,
				},
			)

			continue
		}

		parameterType := parameters[i].TypeAnnotation.Type

		variable := &Variable{
			Identifier:      identifier.Identifier,
			Access:          PrimitiveAccess(ast.AccessAll),
			DeclarationKind: common.DeclarationKindParameter,
			IsConstant:      true,
			Type:            parameterType,
			ActivationDepth: depth,
			Pos:             &identifier.Pos,
		}
		checker.valueActivations.Set(identifier.Identifier, variable)
		if checker.PositionInfo != nil {
			checker.recordVariableDeclarationOccurrence(identifier.Identifier, variable)
		}
	}
}

func (checker *Checker) visitWithPostConditions(
	enclosingElement ast.Element,
	postConditions *ast.Conditions,
	returnType Type,
	returnTypePos ast.HasPosition,
	body func(),
) {

	var postConditionsPos ast.Position
	var rewrittenPostConditions *PostConditionsRewrite

	// If there are post-conditions, rewrite them, extracting `before` expressions.
	// The result are variable declarations which need to be evaluated before
	// the function body

	if postConditions != nil {
		postConditionsPos = postConditions.StartPos

		rewriteResult := checker.rewritePostConditions(postConditions.Conditions)
		rewrittenPostConditions = &rewriteResult

		checker.Elaboration.SetPostConditionsRewrite(postConditions, rewriteResult)

		// all condition blocks are `view`
		checker.InNewPurityScope(true, func() {
			checker.visitStatements(rewriteResult.BeforeStatements)
		})
	}

	body()

	// If there is a post-conditions, declare the function `before`

	// TODO: improve: only declare when a condition actually refers to `before`?

	if postConditions != nil &&
		len(postConditions.Conditions) > 0 {

		checker.declareBefore()
	}

	if rewrittenPostConditions != nil {

		// If there is a return type, declare the constant `result`.
		// If it is a resource type, the constant has the same type as a reference to the return type.
		// If it is not a resource type, the constant has the same type as the return type.

		if returnType != VoidType {
			var resultType Type
			if returnType.IsResourceType() {

				innerType := returnType
				optType, isOptional := returnType.(*OptionalType)
				if isOptional {
					innerType = optType.Type
				}

				auth := UnauthorizedAccess
				// reference is authorized to the entire resource,
				// since it is only accessible in a function where a resource value is owned.
				// To create a "fully authorized" reference,
				// we scan the resource type and produce a conjunction of all the entitlements mentioned within.
				// So, for example,
				//
				// resource R {
				//		access(E) let x: Int
				//      access(X | Y) fun foo() {}
				// }
				//
				// fun test(): @R {
				//    post {
				//	      // do something with result here
				//    }
				//    return <- create R()
				// }
				//
				// Here, the `result` value in the `post` block will have type `auth(E, X, Y) &R`.

				// TODO: check how mapping is handled

				if entitlementSupportingType, ok := innerType.(EntitlementSupportingType); ok {
					supportedEntitlements := entitlementSupportingType.SupportedEntitlements()
					auth = supportedEntitlements.Access()
				}

				resultType = &ReferenceType{
					Type:          innerType,
					Authorization: auth,
				}

				if isOptional {
					// If the return type is an optional type T?, then create an optional reference (&T)?.
					resultType = &OptionalType{
						Type: resultType,
					}
				}
			} else {
				resultType = returnType
			}

			checker.Elaboration.SetResultVariableType(enclosingElement, resultType)

			checker.declareResultVariable(
				resultType,
				returnTypePos,
				postConditionsPos,
			)
		}

		checker.visitConditions(rewrittenPostConditions.RewrittenPostConditions)
	}
}

func (checker *Checker) visitFunctionBlock(
	functionBlock *ast.FunctionBlock,
	returnType Type,
	returnTypePos ast.HasPosition,
	checkResourceLoss bool,
) {
	checker.enterValueScope()
	defer checker.leaveValueScope(functionBlock.EndPosition, checkResourceLoss)

	if functionBlock.PreConditions != nil {
		checker.visitConditions(functionBlock.PreConditions.Conditions)
	}

	checker.visitWithPostConditions(
		functionBlock,
		functionBlock.PostConditions,
		returnType,
		returnTypePos,
		func() {
			// NOTE: not checking block as it enters a new scope
			// and post-conditions need to be able to refer to block's declarations

			checker.visitStatements(functionBlock.Block.Statements)
		},
	)
}

func (checker *Checker) declareResultVariable(
	ty Type,
	returnTypePos ast.HasPosition,
	postConditionsPos ast.Position,
) {
	existingVariable := checker.valueActivations.Current().Find(ResultIdentifier)
	if existingVariable != nil {
		checker.report(
			&ResultVariableConflictError{
				Kind: existingVariable.DeclarationKind,
				Pos:  *existingVariable.Pos,
				ReturnTypeRange: ast.NewRangeFromPositioned(
					checker.memoryGauge,
					returnTypePos,
				),
				PostConditionsRange: ast.NewRange(
					checker.memoryGauge,
					postConditionsPos,
					postConditionsPos.Shifted(checker.memoryGauge, len(ast.ConditionKindPost.Keyword())-1),
				),
			},
		)
		return
	}

	_, err := checker.valueActivations.declareImplicitConstant(
		ResultIdentifier,
		ty,
		common.DeclarationKindConstant,
	)
	checker.report(err)
	// TODO: record occurrence - but what position?
}

func (checker *Checker) declareBefore() {
	_, err := checker.valueActivations.declareImplicitConstant(
		BeforeIdentifier,
		beforeType,
		common.DeclarationKindFunction,
	)
	checker.report(err)
	// TODO: record occurrence – but what position?
}

func (checker *Checker) VisitFunctionExpression(expression *ast.FunctionExpression) Type {

	// TODO: infer
	functionType := checker.functionType(
		false,
		expression.Purity,
		nil,
		expression.ParameterList,
		expression.ReturnTypeAnnotation,
	)

	checker.Elaboration.SetFunctionExpressionFunctionType(expression, functionType)

	checker.checkFunction(
		expression.ParameterList,
		expression.ReturnTypeAnnotation,
		UnauthorizedAccess,
		functionType,
		expression.FunctionBlock,
		true,
		nil,
		true,
	)

	// function expressions are not allowed in conditions

	if checker.inCondition {
		checker.report(
			&FunctionExpressionInConditionError{
				Range: ast.NewRangeFromPositioned(checker.memoryGauge, expression),
			},
		)
	}

	return functionType
}

// checkFieldMembersInitialized checks that all fields that were required
// to be initialized (as stated in the initialization info) have been initialized.
func (checker *Checker) checkFieldMembersInitialized(info *InitializationInfo) {
	for pair := info.FieldMembers.Oldest(); pair != nil; pair = pair.Next() {
		member := pair.Key
		field := pair.Value

		isInitialized := info.InitializedFieldMembers.Contains(member)
		if isInitialized {
			continue
		}

		checker.report(
			&FieldUninitializedError{
				Name:          field.Identifier.Identifier,
				Pos:           field.Identifier.Pos,
				ContainerType: info.ContainerType,
			},
		)
	}
}
