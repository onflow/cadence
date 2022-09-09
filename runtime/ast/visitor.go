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

package ast

import (
	"github.com/onflow/cadence/runtime/errors"
)

type Element interface {
	HasPosition
	ElementType() ElementType
	Walk(walkChild func(Element))
}

type StatementDeclarationVisitor[T any] interface {
	VisitVariableDeclaration(*VariableDeclaration) T
	VisitFunctionDeclaration(*FunctionDeclaration) T
	VisitSpecialFunctionDeclaration(*SpecialFunctionDeclaration) T
	VisitCompositeDeclaration(*CompositeDeclaration) T
	VisitInterfaceDeclaration(*InterfaceDeclaration) T
	VisitTransactionDeclaration(*TransactionDeclaration) T
}

type DeclarationVisitor[T any] interface {
	StatementDeclarationVisitor[T]
	VisitFieldDeclaration(*FieldDeclaration) T
	VisitEnumCaseDeclaration(*EnumCaseDeclaration) T
	VisitPragmaDeclaration(*PragmaDeclaration) T
	VisitImportDeclaration(*ImportDeclaration) T
}

func AcceptDeclaration[T any](declaration Declaration, visitor DeclarationVisitor[T]) (_ T) {

	switch declaration.ElementType() {

	case ElementTypeFieldDeclaration:
		return visitor.VisitFieldDeclaration(declaration.(*FieldDeclaration))

	case ElementTypeEnumCaseDeclaration:
		return visitor.VisitEnumCaseDeclaration(declaration.(*EnumCaseDeclaration))

	case ElementTypePragmaDeclaration:
		return visitor.VisitPragmaDeclaration(declaration.(*PragmaDeclaration))

	case ElementTypeImportDeclaration:
		return visitor.VisitImportDeclaration(declaration.(*ImportDeclaration))

	case ElementTypeVariableDeclaration:
		return visitor.VisitVariableDeclaration(declaration.(*VariableDeclaration))

	case ElementTypeFunctionDeclaration:
		return visitor.VisitFunctionDeclaration(declaration.(*FunctionDeclaration))

	case ElementTypeSpecialFunctionDeclaration:
		return visitor.VisitSpecialFunctionDeclaration(declaration.(*SpecialFunctionDeclaration))

	case ElementTypeCompositeDeclaration:
		return visitor.VisitCompositeDeclaration(declaration.(*CompositeDeclaration))

	case ElementTypeInterfaceDeclaration:
		return visitor.VisitInterfaceDeclaration(declaration.(*InterfaceDeclaration))

	case ElementTypeTransactionDeclaration:
		return visitor.VisitTransactionDeclaration(declaration.(*TransactionDeclaration))
	}

	panic(errors.NewUnreachableError())
}

type StatementVisitor[T any] interface {
	StatementDeclarationVisitor[T]
	VisitReturnStatement(*ReturnStatement) T
	VisitContinueStatement(*ContinueStatement) T
	VisitBreakStatement(*BreakStatement) T
	VisitIfStatement(*IfStatement) T
	VisitForStatement(*ForStatement) T
	VisitAssignmentStatement(*AssignmentStatement) T
	VisitWhileStatement(*WhileStatement) T
	VisitSwapStatement(*SwapStatement) T
	VisitSwitchStatement(*SwitchStatement) T
	VisitEmitStatement(*EmitStatement) T
	VisitExpressionStatement(*ExpressionStatement) T
}

func AcceptStatement[T any](statement Statement, visitor StatementVisitor[T]) (_ T) {

	switch statement.ElementType() {
	case ElementTypeReturnStatement:
		return visitor.VisitReturnStatement(statement.(*ReturnStatement))

	case ElementTypeContinueStatement:
		return visitor.VisitContinueStatement(statement.(*ContinueStatement))

	case ElementTypeBreakStatement:
		return visitor.VisitBreakStatement(statement.(*BreakStatement))

	case ElementTypeIfStatement:
		return visitor.VisitIfStatement(statement.(*IfStatement))

	case ElementTypeForStatement:
		return visitor.VisitForStatement(statement.(*ForStatement))

	case ElementTypeAssignmentStatement:
		return visitor.VisitAssignmentStatement(statement.(*AssignmentStatement))

	case ElementTypeWhileStatement:
		return visitor.VisitWhileStatement(statement.(*WhileStatement))

	case ElementTypeSwapStatement:
		return visitor.VisitSwapStatement(statement.(*SwapStatement))

	case ElementTypeSwitchStatement:
		return visitor.VisitSwitchStatement(statement.(*SwitchStatement))

	case ElementTypeEmitStatement:
		return visitor.VisitEmitStatement(statement.(*EmitStatement))

	case ElementTypeExpressionStatement:
		return visitor.VisitExpressionStatement(statement.(*ExpressionStatement))

	case ElementTypeVariableDeclaration:
		return visitor.VisitVariableDeclaration(statement.(*VariableDeclaration))

	case ElementTypeFunctionDeclaration:
		return visitor.VisitFunctionDeclaration(statement.(*FunctionDeclaration))

	case ElementTypeSpecialFunctionDeclaration:
		return visitor.VisitSpecialFunctionDeclaration(statement.(*SpecialFunctionDeclaration))

	case ElementTypeCompositeDeclaration:
		return visitor.VisitCompositeDeclaration(statement.(*CompositeDeclaration))

	case ElementTypeInterfaceDeclaration:
		return visitor.VisitInterfaceDeclaration(statement.(*InterfaceDeclaration))

	case ElementTypeTransactionDeclaration:
		return visitor.VisitTransactionDeclaration(statement.(*TransactionDeclaration))
	}

	panic(errors.NewUnreachableError())
}

type ExpressionVisitor[T any] interface {
	VisitNilExpression(*NilExpression) T
	VisitBoolExpression(*BoolExpression) T
	VisitStringExpression(*StringExpression) T
	VisitIntegerExpression(*IntegerExpression) T
	VisitFixedPointExpression(*FixedPointExpression) T
	VisitDictionaryExpression(*DictionaryExpression) T
	VisitPathExpression(*PathExpression) T
	VisitForceExpression(*ForceExpression) T
	VisitArrayExpression(*ArrayExpression) T
	VisitInvocationExpression(*InvocationExpression) T
	VisitIdentifierExpression(*IdentifierExpression) T
	VisitIndexExpression(*IndexExpression) T
	VisitUnaryExpression(*UnaryExpression) T
	VisitFunctionExpression(*FunctionExpression) T
	VisitCreateExpression(*CreateExpression) T
	VisitMemberExpression(*MemberExpression) T
	VisitReferenceExpression(*ReferenceExpression) T
	VisitDestroyExpression(*DestroyExpression) T
	VisitCastingExpression(*CastingExpression) T
	VisitBinaryExpression(*BinaryExpression) T
	VisitConditionalExpression(*ConditionalExpression) T
}

func AcceptExpression[T any](expression Expression, visitor ExpressionVisitor[T]) (_ T) {

	switch expression.ElementType() {

	case ElementTypeNilExpression:
		return visitor.VisitNilExpression(expression.(*NilExpression))

	case ElementTypeBoolExpression:
		return visitor.VisitBoolExpression(expression.(*BoolExpression))

	case ElementTypeStringExpression:
		return visitor.VisitStringExpression(expression.(*StringExpression))

	case ElementTypeIntegerExpression:
		return visitor.VisitIntegerExpression(expression.(*IntegerExpression))

	case ElementTypeFixedPointExpression:
		return visitor.VisitFixedPointExpression(expression.(*FixedPointExpression))

	case ElementTypeDictionaryExpression:
		return visitor.VisitDictionaryExpression(expression.(*DictionaryExpression))

	case ElementTypePathExpression:
		return visitor.VisitPathExpression(expression.(*PathExpression))

	case ElementTypeForceExpression:
		return visitor.VisitForceExpression(expression.(*ForceExpression))

	case ElementTypeArrayExpression:
		return visitor.VisitArrayExpression(expression.(*ArrayExpression))

	case ElementTypeInvocationExpression:
		return visitor.VisitInvocationExpression(expression.(*InvocationExpression))

	case ElementTypeIdentifierExpression:
		return visitor.VisitIdentifierExpression(expression.(*IdentifierExpression))

	case ElementTypeIndexExpression:
		return visitor.VisitIndexExpression(expression.(*IndexExpression))

	case ElementTypeUnaryExpression:
		return visitor.VisitUnaryExpression(expression.(*UnaryExpression))

	case ElementTypeFunctionExpression:
		return visitor.VisitFunctionExpression(expression.(*FunctionExpression))

	case ElementTypeCreateExpression:
		return visitor.VisitCreateExpression(expression.(*CreateExpression))

	case ElementTypeMemberExpression:
		return visitor.VisitMemberExpression(expression.(*MemberExpression))

	case ElementTypeReferenceExpression:
		return visitor.VisitReferenceExpression(expression.(*ReferenceExpression))

	case ElementTypeDestroyExpression:
		return visitor.VisitDestroyExpression(expression.(*DestroyExpression))

	case ElementTypeCastingExpression:
		return visitor.VisitCastingExpression(expression.(*CastingExpression))

	case ElementTypeBinaryExpression:
		return visitor.VisitBinaryExpression(expression.(*BinaryExpression))

	case ElementTypeConditionalExpression:
		return visitor.VisitConditionalExpression(expression.(*ConditionalExpression))
	}

	panic(errors.NewUnreachableError())
}
