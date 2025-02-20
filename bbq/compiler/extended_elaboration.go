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

package compiler

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/sema"
)

type ExtendedElaboration struct {
	// Do not expose functionality directly (by embedding the type),
	// since that would make it easy to mistakenly modify the original elaboration.
	elaboration *sema.Elaboration

	// Holds the elaborations associated with inherited pre-/post-conditions and
	// before-statements of those post conditions.
	conditionsElaborations map[ast.Statement]*ExtendedElaboration

	interfaceMethodStaticCalls        map[*ast.InvocationExpression]struct{}
	interfaceDeclarationTypes         map[*ast.InterfaceDeclaration]*sema.InterfaceType
	compositeDeclarationTypes         map[ast.CompositeLikeDeclaration]*sema.CompositeType
	variableDeclarationTypes          map[*ast.VariableDeclaration]sema.VariableDeclarationTypes
	invocationExpressionTypes         map[*ast.InvocationExpression]sema.InvocationExpressionTypes
	memberExpressionMemberAccessInfos map[*ast.MemberExpression]sema.MemberAccessInfo
	assignmentStatementTypes          map[*ast.AssignmentStatement]sema.AssignmentStatementTypes
	resultVariableTypes               map[ast.Element]sema.Type
	referenceExpressionBorrowTypes    map[*ast.ReferenceExpression]sema.Type
	functionDeclarationFunctionTypes  map[*ast.FunctionDeclaration]*sema.FunctionType
	returnStatementTypes              map[*ast.ReturnStatement]sema.ReturnStatementTypes
}

func NewExtendedElaboration(elaboration *sema.Elaboration) *ExtendedElaboration {
	return &ExtendedElaboration{
		elaboration:            elaboration,
		conditionsElaborations: map[ast.Statement]*ExtendedElaboration{},
	}
}

func (e *ExtendedElaboration) SetInterfaceMethodStaticCall(invocation *ast.InvocationExpression) {
	if e.interfaceMethodStaticCalls == nil {
		e.interfaceMethodStaticCalls = make(map[*ast.InvocationExpression]struct{})
	}
	e.interfaceMethodStaticCalls[invocation] = struct{}{}
}

func (e *ExtendedElaboration) IsInterfaceMethodStaticCall(invocation *ast.InvocationExpression) bool {
	if e.interfaceMethodStaticCalls == nil {
		return false
	}

	_, ok := e.interfaceMethodStaticCalls[invocation]
	return ok
}

func (e *ExtendedElaboration) SetInterfaceDeclarationType(decl *ast.InterfaceDeclaration, interfaceType *sema.InterfaceType) {
	if e.interfaceDeclarationTypes == nil {
		e.interfaceDeclarationTypes = make(map[*ast.InterfaceDeclaration]*sema.InterfaceType)
	}

	e.interfaceDeclarationTypes[decl] = interfaceType
}

func (e *ExtendedElaboration) InterfaceDeclarationType(decl *ast.InterfaceDeclaration) *sema.InterfaceType {
	// First lookup in the extended type info
	if e.interfaceDeclarationTypes != nil {
		typ, ok := e.interfaceDeclarationTypes[decl]
		if ok {
			return typ
		}
	}

	// If not found, then fallback and look in the original elaboration
	return e.elaboration.InterfaceDeclarationType(decl)
}

func (e *ExtendedElaboration) ReturnStatementTypes(statement *ast.ReturnStatement) sema.ReturnStatementTypes {
	if e.returnStatementTypes != nil {
		typ, ok := e.returnStatementTypes[statement]
		if ok {
			return typ
		}
	}

	return e.elaboration.ReturnStatementTypes(statement)
}

func (e *ExtendedElaboration) SetReturnStatementTypes(statement *ast.ReturnStatement, types sema.ReturnStatementTypes) {
	if e.returnStatementTypes == nil {
		e.returnStatementTypes = map[*ast.ReturnStatement]sema.ReturnStatementTypes{}
	}
	e.returnStatementTypes[statement] = types
}

func (e *ExtendedElaboration) VariableDeclarationTypes(declaration *ast.VariableDeclaration) sema.VariableDeclarationTypes {
	if e.variableDeclarationTypes != nil {
		typ, ok := e.variableDeclarationTypes[declaration]
		if ok {
			return typ
		}
	}
	return e.elaboration.VariableDeclarationTypes(declaration)
}

func (e *ExtendedElaboration) SetVariableDeclarationTypes(
	declaration *ast.VariableDeclaration,
	types sema.VariableDeclarationTypes,
) {
	if e.variableDeclarationTypes == nil {
		e.variableDeclarationTypes = map[*ast.VariableDeclaration]sema.VariableDeclarationTypes{}
	}
	e.variableDeclarationTypes[declaration] = types
}

func (e *ExtendedElaboration) PostConditionsRewrite(conditions *ast.Conditions) sema.PostConditionsRewrite {
	return e.elaboration.PostConditionsRewrite(conditions)
}

func (e *ExtendedElaboration) CompositeDeclarationType(declaration ast.CompositeLikeDeclaration) *sema.CompositeType {
	// First lookup in the extended type info
	if e.compositeDeclarationTypes != nil {
		typ, ok := e.compositeDeclarationTypes[declaration]
		if ok {
			return typ
		}
	}

	// If not found, then fallback and look in the original elaboration
	return e.elaboration.CompositeDeclarationType(declaration)
}

func (e *ExtendedElaboration) SetCompositeDeclarationType(
	declaration ast.CompositeLikeDeclaration,
	compositeType *sema.CompositeType,
) {
	if e.compositeDeclarationTypes == nil {
		e.compositeDeclarationTypes = map[ast.CompositeLikeDeclaration]*sema.CompositeType{}
	}
	e.compositeDeclarationTypes[declaration] = compositeType
}

func (e *ExtendedElaboration) InvocationExpressionTypes(
	expression *ast.InvocationExpression,
) sema.InvocationExpressionTypes {
	// First lookup in the extended type info
	if e.invocationExpressionTypes != nil {
		types, ok := e.invocationExpressionTypes[expression]
		if ok {
			return types
		}
	}
	// If not found, then fallback and look in the original elaboration
	return e.elaboration.InvocationExpressionTypes(expression)
}

func (e *ExtendedElaboration) SetInvocationExpressionTypes(
	expression *ast.InvocationExpression,
	types sema.InvocationExpressionTypes,
) {
	if e.invocationExpressionTypes == nil {
		e.invocationExpressionTypes = map[*ast.InvocationExpression]sema.InvocationExpressionTypes{}
	}
	e.invocationExpressionTypes[expression] = types
}

func (e *ExtendedElaboration) MemberExpressionMemberAccessInfo(expression *ast.MemberExpression) (memberInfo sema.MemberAccessInfo, ok bool) {
	if e.memberExpressionMemberAccessInfos != nil {
		memberInfo, ok = e.memberExpressionMemberAccessInfos[expression]
		if ok {
			return
		}
	}
	return e.elaboration.MemberExpressionMemberAccessInfo(expression)
}

func (e *ExtendedElaboration) SetMemberExpressionMemberAccessInfo(expression *ast.MemberExpression, memberAccessInfo sema.MemberAccessInfo) {
	if e.memberExpressionMemberAccessInfos == nil {
		e.memberExpressionMemberAccessInfos = map[*ast.MemberExpression]sema.MemberAccessInfo{}
	}
	e.memberExpressionMemberAccessInfos[expression] = memberAccessInfo
}

func (e *ExtendedElaboration) AssignmentStatementTypes(assignment *ast.AssignmentStatement) sema.AssignmentStatementTypes {
	if e.assignmentStatementTypes != nil {
		types, ok := e.assignmentStatementTypes[assignment]
		if ok {
			return types
		}
	}
	return e.elaboration.AssignmentStatementTypes(assignment)
}

func (e *ExtendedElaboration) SetAssignmentStatementTypes(
	assignment *ast.AssignmentStatement,
	types sema.AssignmentStatementTypes,
) {
	if e.assignmentStatementTypes == nil {
		e.assignmentStatementTypes = map[*ast.AssignmentStatement]sema.AssignmentStatementTypes{}
	}
	e.assignmentStatementTypes[assignment] = types
}

func (e *ExtendedElaboration) TransactionDeclarationType(declaration *ast.TransactionDeclaration) *sema.TransactionType {
	return e.elaboration.TransactionDeclarationType(declaration)
}

func (e *ExtendedElaboration) IntegerExpressionType(expression *ast.IntegerExpression) sema.Type {
	return e.elaboration.IntegerExpressionType(expression)
}

func (e *ExtendedElaboration) ArrayExpressionTypes(expression *ast.ArrayExpression) sema.ArrayExpressionTypes {
	return e.elaboration.ArrayExpressionTypes(expression)
}

func (e *ExtendedElaboration) DictionaryExpressionTypes(expression *ast.DictionaryExpression) sema.DictionaryExpressionTypes {
	return e.elaboration.DictionaryExpressionTypes(expression)
}

func (e *ExtendedElaboration) CastingExpressionTypes(expression *ast.CastingExpression) sema.CastingExpressionTypes {
	return e.elaboration.CastingExpressionTypes(expression)
}

func (e *ExtendedElaboration) EmitStatementEventType(statement *ast.EmitStatement) *sema.CompositeType {
	return e.elaboration.EmitStatementEventType(statement)
}

func (e *ExtendedElaboration) ResultVariableType(enclosingBlock ast.Element) (typ sema.Type, exist bool) {
	if e.resultVariableTypes != nil {
		types, ok := e.resultVariableTypes[enclosingBlock]
		if ok {
			return types, ok
		}
	}
	return e.elaboration.ResultVariableType(enclosingBlock)
}

func (e *ExtendedElaboration) SetResultVariableType(declaration ast.Element, typ sema.Type) {
	if e.resultVariableTypes == nil {
		e.resultVariableTypes = map[ast.Element]sema.Type{}
	}
	e.resultVariableTypes[declaration] = typ
}

func (e *ExtendedElaboration) ReferenceExpressionBorrowType(expression *ast.ReferenceExpression) sema.Type {
	if e.referenceExpressionBorrowTypes != nil {
		typ, ok := e.referenceExpressionBorrowTypes[expression]
		if ok {
			return typ
		}
	}
	return e.elaboration.ReferenceExpressionBorrowType(expression)
}

func (e *ExtendedElaboration) SetReferenceExpressionBorrowType(expression *ast.ReferenceExpression, ty sema.Type) {
	if e.referenceExpressionBorrowTypes == nil {
		e.referenceExpressionBorrowTypes = map[*ast.ReferenceExpression]sema.Type{}
	}
	e.referenceExpressionBorrowTypes[expression] = ty
}

func (e *ExtendedElaboration) FunctionDeclarationFunctionType(declaration *ast.FunctionDeclaration) *sema.FunctionType {
	if e.functionDeclarationFunctionTypes != nil {
		typ, ok := e.functionDeclarationFunctionTypes[declaration]
		if ok {
			return typ
		}
	}
	return e.elaboration.FunctionDeclarationFunctionType(declaration)
}

func (e *ExtendedElaboration) SetFunctionDeclarationFunctionType(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
) {
	if e.functionDeclarationFunctionTypes == nil {
		e.functionDeclarationFunctionTypes = map[*ast.FunctionDeclaration]*sema.FunctionType{}
	}
	e.functionDeclarationFunctionTypes[declaration] = functionType
}
