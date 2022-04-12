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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Repr {
	return checker.visitFunctionDeclaration(
		declaration,
		functionDeclarationOptions{
			mustExit:          true,
			declareFunction:   true,
			checkResourceLoss: true,
		},
	)
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
) ast.Repr {

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		declaration.StartPos,
		true,
	)

	// global functions were previously declared, see `declareFunctionDeclaration`

	functionType := checker.Elaboration.FunctionDeclarationFunctionTypes[declaration]
	if functionType == nil {
		functionType = checker.functionType(declaration.ParameterList, declaration.ReturnTypeAnnotation)

		if options.declareFunction {
			checker.declareFunctionDeclaration(declaration, functionType)
		}
	}

	checker.Elaboration.FunctionDeclarationFunctionTypes[declaration] = functionType

	checker.checkFunction(
		declaration.ParameterList,
		declaration.ReturnTypeAnnotation,
		functionType,
		declaration.FunctionBlock,
		options.mustExit,
		nil,
		options.checkResourceLoss,
	)

	return nil
}

func (checker *Checker) declareFunctionDeclaration(
	declaration *ast.FunctionDeclaration,
	functionType *FunctionType,
) {
	argumentLabels := declaration.ParameterList.EffectiveArgumentLabels()

	_, err := checker.valueActivations.Declare(variableDeclaration{
		identifier:               declaration.Identifier.Identifier,
		ty:                       functionType,
		docString:                declaration.DocString,
		access:                   declaration.Access,
		kind:                     common.DeclarationKindFunction,
		pos:                      declaration.Identifier.Pos,
		isConstant:               true,
		argumentLabels:           argumentLabels,
		allowOuterScopeShadowing: false,
	})
	checker.report(err)

	if checker.positionInfoEnabled {
		checker.recordFunctionDeclarationOrigin(declaration, functionType)
	}
}

func (checker *Checker) checkFunction(
	parameterList *ast.ParameterList,
	returnTypeAnnotation *ast.TypeAnnotation,
	functionType *FunctionType,
	functionBlock *ast.FunctionBlock,
	mustExit bool,
	initializationInfo *InitializationInfo,
	checkResourceLoss bool,
) {
	// check argument labels
	checker.checkArgumentLabels(parameterList)

	checker.checkParameters(parameterList, functionType.Parameters)

	if functionType.ReturnTypeAnnotation != nil {
		checker.checkTypeAnnotation(functionType.ReturnTypeAnnotation, returnTypeAnnotation)
	}

	// Reset the returning state and restore it when leaving

	jumpedOrReturned := checker.resources.JumpsOrReturns
	halted := checker.resources.Halts
	checker.resources.JumpsOrReturns = false
	checker.resources.Halts = false
	defer func() {
		checker.resources.JumpsOrReturns = jumpedOrReturned
		checker.resources.Halts = halted
	}()

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
				checker.visitFunctionBlock(
					functionBlock,
					functionType.ReturnTypeAnnotation,
					checkResourceLoss,
				)

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

	if checker.positionInfoEnabled && functionBlock != nil {
		startPos := functionBlock.StartPosition()
		endPos := functionBlock.EndPosition(checker.memoryGauge)

		for _, parameter := range functionType.Parameters {
			checker.Ranges.Put(
				startPos,
				endPos,
				Range{
					Identifier:      parameter.Identifier,
					Type:            parameter.TypeAnnotation.Type,
					DeclarationKind: common.DeclarationKindParameter,
				},
			)
		}
	}
}

// checkFunctionExits checks that the given function block exits
// with a return-type appropriate return statement.
// The return is not needed if the function has a `Void` return type.
//
func (checker *Checker) checkFunctionExits(functionBlock *ast.FunctionBlock, returnType Type) {

	if returnType == VoidType {
		return
	}

	functionActivation := checker.functionActivations.Current()

	definitelyReturnedOrHalted :=
		functionActivation.ReturnInfo.DefinitelyReturned ||
			functionActivation.ReturnInfo.DefinitelyHalted

	if definitelyReturnedOrHalted {
		return
	}

	checker.report(
		&MissingReturnStatementError{
			Range: ast.NewRangeFromPositioned(checker.memoryGauge, functionBlock),
		},
	)
}

func (checker *Checker) checkParameters(parameterList *ast.ParameterList, parameters []*Parameter) {
	for i, parameter := range parameterList.Parameters {
		parameterTypeAnnotation := parameters[i].TypeAnnotation

		checker.checkTypeAnnotation(
			parameterTypeAnnotation,
			parameter.TypeAnnotation,
		)
	}
}

// checkArgumentLabels checks that all argument labels (if any) are unique
//
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
//
func (checker *Checker) declareParameters(
	parameterList *ast.ParameterList,
	parameters []*Parameter,
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
			Access:          ast.AccessPublic,
			DeclarationKind: common.DeclarationKindParameter,
			IsConstant:      true,
			Type:            parameterType,
			ActivationDepth: depth,
			Pos:             &identifier.Pos,
		}
		checker.valueActivations.Set(identifier.Identifier, variable)
		if checker.positionInfoEnabled {
			checker.recordVariableDeclarationOccurrence(identifier.Identifier, variable)
		}
	}
}

func (checker *Checker) VisitFunctionBlock(functionBlock *ast.FunctionBlock) ast.Repr {
	// NOTE: see visitFunctionBlock
	panic(errors.NewUnreachableError())
}

func (checker *Checker) visitWithPostConditions(postConditions *ast.Conditions, returnType Type, body func()) {

	var rewrittenPostConditions *PostConditionsRewrite

	// If there are post-conditions, rewrite them, extracting `before` expressions.
	// The result are variable declarations which need to be evaluated before
	// the function body

	if postConditions != nil {
		rewriteResult := checker.rewritePostConditions(*postConditions)
		rewrittenPostConditions = &rewriteResult

		checker.Elaboration.PostConditionsRewrite[postConditions] = rewriteResult

		checker.visitStatements(rewriteResult.BeforeStatements)
	}

	body()

	// If there is a post-conditions, declare the function `before`

	// TODO: improve: only declare when a condition actually refers to `before`?

	if postConditions != nil &&
		len(*postConditions) > 0 {

		checker.declareBefore()
	}

	// If there is a return type, declare the constant `result`.
	// If it is a resource type, the constant has the same type as a referecne to the return type.
	// If it is not a resource type, the constant has the same type as the return type.

	if returnType != VoidType {
		var resultType Type
		if returnType.IsResourceType() {
			resultType = &ReferenceType{
				Type: returnType,
			}
		} else {
			resultType = returnType
		}
		checker.declareResult(resultType)
	}

	if rewrittenPostConditions != nil {
		checker.visitConditions(rewrittenPostConditions.RewrittenPostConditions)
	}
}

func (checker *Checker) visitFunctionBlock(
	functionBlock *ast.FunctionBlock,
	returnTypeAnnotation *TypeAnnotation,
	checkResourceLoss bool,
) {
	checker.enterValueScope()
	defer checker.leaveValueScope(functionBlock.EndPosition, checkResourceLoss)

	if functionBlock.PreConditions != nil {
		checker.visitConditions(*functionBlock.PreConditions)
	}

	checker.visitWithPostConditions(
		functionBlock.PostConditions,
		returnTypeAnnotation.Type,
		func() {
			// NOTE: not checking block as it enters a new scope
			// and post-conditions need to be able to refer to block's declarations

			checker.visitStatements(functionBlock.Block.Statements)
		},
	)
}

func (checker *Checker) declareResult(ty Type) {
	_, err := checker.valueActivations.DeclareImplicitConstant(
		ResultIdentifier,
		ty,
		common.DeclarationKindConstant,
	)
	checker.report(err)
	// TODO: record occurrence - but what position?
}

func (checker *Checker) declareBefore() {
	_, err := checker.valueActivations.DeclareImplicitConstant(
		BeforeIdentifier,
		beforeType,
		common.DeclarationKindFunction,
	)
	checker.report(err)
	// TODO: record occurrence – but what position?
}

func (checker *Checker) VisitFunctionExpression(expression *ast.FunctionExpression) ast.Repr {

	// TODO: infer
	functionType := checker.functionType(expression.ParameterList, expression.ReturnTypeAnnotation)

	checker.Elaboration.FunctionExpressionFunctionType[expression] = functionType

	checker.checkFunction(
		expression.ParameterList,
		expression.ReturnTypeAnnotation,
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
//
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
