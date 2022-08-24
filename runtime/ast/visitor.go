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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type Element interface {
	HasPosition
	ElementType() ElementType
	Walk(walkChild func(Element))
}

type NotAnElement struct{}

var _ Element = NotAnElement{}

func (NotAnElement) ElementType() ElementType {
	return ElementTypeUnknown
}

func (NotAnElement) StartPosition() Position {
	return EmptyPosition
}

func (NotAnElement) EndPosition(common.MemoryGauge) Position {
	return EmptyPosition
}

func (NotAnElement) Walk(_ func(Element)) {
	// NO-OP
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

type StatementVisitor[T any] interface {
	StatementDeclarationVisitor[T]
	VisitReturnStatement(*ReturnStatement) T
	VisitBreakStatement(*BreakStatement) T
	VisitContinueStatement(*ContinueStatement) T
	VisitIfStatement(*IfStatement) T
	VisitSwitchStatement(*SwitchStatement) T
	VisitWhileStatement(*WhileStatement) T
	VisitForStatement(*ForStatement) T
	VisitEmitStatement(*EmitStatement) T
	VisitAssignmentStatement(*AssignmentStatement) T
	VisitSwapStatement(*SwapStatement) T
	VisitExpressionStatement(*ExpressionStatement) T
}

type ExpressionVisitor[T any] interface {
	VisitBoolExpression(*BoolExpression) T
	VisitNilExpression(*NilExpression) T
	VisitIntegerExpression(*IntegerExpression) T
	VisitFixedPointExpression(*FixedPointExpression) T
	VisitArrayExpression(*ArrayExpression) T
	VisitDictionaryExpression(*DictionaryExpression) T
	VisitIdentifierExpression(*IdentifierExpression) T
	VisitInvocationExpression(*InvocationExpression) T
	VisitMemberExpression(*MemberExpression) T
	VisitIndexExpression(*IndexExpression) T
	VisitConditionalExpression(*ConditionalExpression) T
	VisitUnaryExpression(*UnaryExpression) T
	VisitBinaryExpression(*BinaryExpression) T
	VisitFunctionExpression(*FunctionExpression) T
	VisitStringExpression(*StringExpression) T
	VisitCastingExpression(*CastingExpression) T
	VisitCreateExpression(*CreateExpression) T
	VisitDestroyExpression(*DestroyExpression) T
	VisitReferenceExpression(*ReferenceExpression) T
	VisitForceExpression(*ForceExpression) T
	VisitPathExpression(*PathExpression) T
}

type Visitor[T any] interface {
	StatementVisitor[T]
	ExpressionVisitor[T]
	DeclarationVisitor[T]
	VisitProgram(*Program) T
	VisitBlock(*Block) T
	VisitFunctionBlock(*FunctionBlock) T
}

func Accept[T any](element Element, visitor Visitor[T]) (_ T) {

	switch element.ElementType() {
	case ElementTypePragmaDeclaration:
		return visitor.VisitPragmaDeclaration(element.(*PragmaDeclaration))

	case ElementTypeUnknown:
		// NO-OP
		return

	case ElementTypeBlock:
		return visitor.VisitBlock(element.(*Block))

	case ElementTypeFunctionBlock:
		return visitor.VisitFunctionBlock(element.(*FunctionBlock))

	case ElementTypeCompositeDeclaration:
		return visitor.VisitCompositeDeclaration(element.(*CompositeDeclaration))

	case ElementTypeInterfaceDeclaration:
		return visitor.VisitInterfaceDeclaration(element.(*InterfaceDeclaration))

	case ElementTypeFieldDeclaration:
		return visitor.VisitFieldDeclaration(element.(*FieldDeclaration))

	case ElementTypeReturnStatement:
		return visitor.VisitReturnStatement(element.(*ReturnStatement))

	case ElementTypeEnumCaseDeclaration:
		return visitor.VisitEnumCaseDeclaration(element.(*EnumCaseDeclaration))

	case ElementTypeFunctionDeclaration:
		return visitor.VisitFunctionDeclaration(element.(*FunctionDeclaration))

	case ElementTypeSpecialFunctionDeclaration:
		return visitor.VisitSpecialFunctionDeclaration(element.(*SpecialFunctionDeclaration))

	case ElementTypeVariableDeclaration:
		return visitor.VisitVariableDeclaration(element.(*VariableDeclaration))

	case ElementTypeTransactionDeclaration:
		return visitor.VisitTransactionDeclaration(element.(*TransactionDeclaration))

	case ElementTypeImportDeclaration:
		return visitor.VisitImportDeclaration(element.(*ImportDeclaration))

	case ElementTypeProgram:
		return visitor.VisitProgram(element.(*Program))

	case ElementTypeContinueStatement:
		return visitor.VisitContinueStatement(element.(*ContinueStatement))

	case ElementTypeBreakStatement:
		return visitor.VisitBreakStatement(element.(*BreakStatement))

	case ElementTypeIfStatement:
		return visitor.VisitIfStatement(element.(*IfStatement))

	case ElementTypeForStatement:
		return visitor.VisitForStatement(element.(*ForStatement))

	case ElementTypeAssignmentStatement:
		return visitor.VisitAssignmentStatement(element.(*AssignmentStatement))

	case ElementTypeWhileStatement:
		return visitor.VisitWhileStatement(element.(*WhileStatement))

	case ElementTypeSwapStatement:
		return visitor.VisitSwapStatement(element.(*SwapStatement))

	case ElementTypeSwitchStatement:
		return visitor.VisitSwitchStatement(element.(*SwitchStatement))

	case ElementTypeEmitStatement:
		return visitor.VisitEmitStatement(element.(*EmitStatement))

	case ElementTypeExpressionStatement:
		return visitor.VisitExpressionStatement(element.(*ExpressionStatement))

	case ElementTypeNilExpression:
		return visitor.VisitNilExpression(element.(*NilExpression))

	case ElementTypeBoolExpression:
		return visitor.VisitBoolExpression(element.(*BoolExpression))

	case ElementTypeStringExpression:
		return visitor.VisitStringExpression(element.(*StringExpression))

	case ElementTypeIntegerExpression:
		return visitor.VisitIntegerExpression(element.(*IntegerExpression))

	case ElementTypeFixedPointExpression:
		return visitor.VisitFixedPointExpression(element.(*FixedPointExpression))

	case ElementTypeDictionaryExpression:
		return visitor.VisitDictionaryExpression(element.(*DictionaryExpression))

	case ElementTypePathExpression:
		return visitor.VisitPathExpression(element.(*PathExpression))

	case ElementTypeForceExpression:
		return visitor.VisitForceExpression(element.(*ForceExpression))

	case ElementTypeArrayExpression:
		return visitor.VisitArrayExpression(element.(*ArrayExpression))

	case ElementTypeInvocationExpression:
		return visitor.VisitInvocationExpression(element.(*InvocationExpression))

	case ElementTypeIdentifierExpression:
		return visitor.VisitIdentifierExpression(element.(*IdentifierExpression))

	case ElementTypeIndexExpression:
		return visitor.VisitIndexExpression(element.(*IndexExpression))

	case ElementTypeUnaryExpression:
		return visitor.VisitUnaryExpression(element.(*UnaryExpression))

	case ElementTypeFunctionExpression:
		return visitor.VisitFunctionExpression(element.(*FunctionExpression))

	case ElementTypeCreateExpression:
		return visitor.VisitCreateExpression(element.(*CreateExpression))

	case ElementTypeMemberExpression:
		return visitor.VisitMemberExpression(element.(*MemberExpression))

	case ElementTypeReferenceExpression:
		return visitor.VisitReferenceExpression(element.(*ReferenceExpression))

	case ElementTypeDestroyExpression:
		return visitor.VisitDestroyExpression(element.(*DestroyExpression))

	case ElementTypeCastingExpression:
		return visitor.VisitCastingExpression(element.(*CastingExpression))

	case ElementTypeBinaryExpression:
		return visitor.VisitBinaryExpression(element.(*BinaryExpression))

	case ElementTypeConditionalExpression:
		return visitor.VisitConditionalExpression(element.(*ConditionalExpression))

	}

	panic(errors.NewUnreachableError())
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
