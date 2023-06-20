/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) visitConditions(conditions []ast.Condition) {
	// all condition blocks are `view`
	checker.InNewPurityScope(true, func() {
		// flag the checker to be inside a condition.
		// this flag is used to detect illegal expressions,
		// see e.g. VisitFunctionExpression

		wasInCondition := checker.inCondition
		checker.inCondition = true
		defer func() {
			checker.inCondition = wasInCondition
		}()

		// check all conditions: check the expression
		// and ensure the result is boolean

		for _, condition := range conditions {
			checker.checkCondition(condition)
		}
	})
}

func (checker *Checker) checkCondition(condition ast.Condition) Type {

	switch condition := condition.(type) {
	case *ast.TestCondition:

		// check test expression is boolean
		checker.VisitExpression(condition.Test, BoolType)

		// check message expression results in a string
		if condition.Message != nil {
			checker.VisitExpression(condition.Message, StringType)
		}

	case *ast.EmitCondition:
		checker.VisitEmitStatement((*ast.EmitStatement)(condition))

	default:
		panic(errors.NewUnreachableError())
	}

	return nil
}

func (checker *Checker) rewritePostConditions(postConditions ast.Conditions) PostConditionsRewrite {

	var beforeStatements []ast.Statement

	var rewrittenPostConditions ast.Conditions
	var allExtractedExpressions []ast.ExtractedExpression

	count := len(postConditions)
	if count > 0 {
		rewrittenPostConditions = make([]ast.Condition, count)

		for i, postCondition := range postConditions {

			newPostCondition, extractedExpressions := checker.rewritePostCondition(postCondition)
			rewrittenPostConditions[i] = newPostCondition
			allExtractedExpressions = append(
				allExtractedExpressions,
				extractedExpressions...,
			)
		}
	}

	for _, extractedExpression := range allExtractedExpressions {
		expression := extractedExpression.Expression
		startPos := expression.StartPosition()

		// NOTE: no need to check the before statements or update elaboration here:
		// The before statements are visited/checked later
		variableDeclaration := ast.NewEmptyVariableDeclaration(checker.memoryGauge)
		variableDeclaration.StartPos = startPos
		variableDeclaration.Identifier = extractedExpression.Identifier
		variableDeclaration.Transfer = ast.NewTransfer(
			checker.memoryGauge,
			ast.TransferOperationCopy,
			startPos,
		)
		variableDeclaration.Value = expression

		beforeStatements = append(
			beforeStatements,
			variableDeclaration,
		)
	}

	return PostConditionsRewrite{
		BeforeStatements:        beforeStatements,
		RewrittenPostConditions: rewrittenPostConditions,
	}
}

func (checker *Checker) rewritePostCondition(
	postCondition ast.Condition,
) (
	newPostCondition ast.Condition,
	extractedExpressions []ast.ExtractedExpression,
) {
	switch postCondition := postCondition.(type) {
	case *ast.TestCondition:
		return checker.rewriteTestPostCondition(postCondition)

	case *ast.EmitCondition:
		// TODO:
		panic("TODO")

	default:
		panic(errors.NewUnreachableError())
	}
}

func (checker *Checker) rewriteTestPostCondition(
	postTestCondition *ast.TestCondition,
) (
	newPostCondition ast.Condition,
	extractedExpressions []ast.ExtractedExpression,
) {
	// copy condition and set expression to rewritten one
	newPostTestCondition := *postTestCondition

	beforeExtractor := checker.beforeExtractor()

	testExtraction := beforeExtractor.ExtractBefore(postTestCondition.Test)

	extractedExpressions = testExtraction.ExtractedExpressions

	newPostTestCondition.Test = testExtraction.RewrittenExpression

	if postTestCondition.Message != nil {
		messageExtraction := beforeExtractor.ExtractBefore(postTestCondition.Message)

		newPostTestCondition.Message = messageExtraction.RewrittenExpression

		extractedExpressions = append(
			extractedExpressions,
			messageExtraction.ExtractedExpressions...,
		)
	}

	newPostCondition = &newPostTestCondition

	return
}
