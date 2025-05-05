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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

type DesugaredElaboration struct {
	// Do not expose functionality directly (by embedding the type),
	// since that would make it easy to mistakenly modify the original elaboration.
	elaboration *sema.Elaboration

	// Holds the elaborations associated with inherited pre-/post-conditions and
	// before-statements of those post conditions.
	conditionsElaborations map[ast.Statement]*DesugaredElaboration

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
	emitStatementEventTypes           map[*ast.EmitStatement]*sema.CompositeType
	compositeTypes                    map[common.TypeID]*sema.CompositeType
}

func NewDesugaredElaboration(elaboration *sema.Elaboration) *DesugaredElaboration {
	return &DesugaredElaboration{
		elaboration:            elaboration,
		conditionsElaborations: map[ast.Statement]*DesugaredElaboration{},
	}
}

func (e *DesugaredElaboration) SetInterfaceMethodStaticCall(invocation *ast.InvocationExpression) {
	if e.interfaceMethodStaticCalls == nil {
		e.interfaceMethodStaticCalls = make(map[*ast.InvocationExpression]struct{})
	}
	e.interfaceMethodStaticCalls[invocation] = struct{}{}
}

func (e *DesugaredElaboration) IsInterfaceMethodStaticCall(invocation *ast.InvocationExpression) bool {
	if e.interfaceMethodStaticCalls == nil {
		return false
	}

	_, ok := e.interfaceMethodStaticCalls[invocation]
	return ok
}

func (e *DesugaredElaboration) SetInterfaceDeclarationType(decl *ast.InterfaceDeclaration, interfaceType *sema.InterfaceType) {
	if e.interfaceDeclarationTypes == nil {
		e.interfaceDeclarationTypes = make(map[*ast.InterfaceDeclaration]*sema.InterfaceType)
	}

	e.interfaceDeclarationTypes[decl] = interfaceType
}

func (e *DesugaredElaboration) InterfaceDeclarationType(decl *ast.InterfaceDeclaration) *sema.InterfaceType {
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

func (e *DesugaredElaboration) ReturnStatementTypes(statement *ast.ReturnStatement) sema.ReturnStatementTypes {
	if e.returnStatementTypes != nil {
		typ, ok := e.returnStatementTypes[statement]
		if ok {
			return typ
		}
	}

	return e.elaboration.ReturnStatementTypes(statement)
}

func (e *DesugaredElaboration) SetReturnStatementTypes(statement *ast.ReturnStatement, types sema.ReturnStatementTypes) {
	if e.returnStatementTypes == nil {
		e.returnStatementTypes = map[*ast.ReturnStatement]sema.ReturnStatementTypes{}
	}
	e.returnStatementTypes[statement] = types
}

func (e *DesugaredElaboration) VariableDeclarationTypes(declaration *ast.VariableDeclaration) sema.VariableDeclarationTypes {
	if e.variableDeclarationTypes != nil {
		typ, ok := e.variableDeclarationTypes[declaration]
		if ok {
			return typ
		}
	}
	return e.elaboration.VariableDeclarationTypes(declaration)
}

func (e *DesugaredElaboration) SetVariableDeclarationTypes(
	declaration *ast.VariableDeclaration,
	types sema.VariableDeclarationTypes,
) {
	if e.variableDeclarationTypes == nil {
		e.variableDeclarationTypes = map[*ast.VariableDeclaration]sema.VariableDeclarationTypes{}
	}
	e.variableDeclarationTypes[declaration] = types
}

func (e *DesugaredElaboration) PostConditionsRewrite(conditions *ast.Conditions) sema.PostConditionsRewrite {
	return e.elaboration.PostConditionsRewrite(conditions)
}

func (e *DesugaredElaboration) CompositeDeclarationType(declaration ast.CompositeLikeDeclaration) *sema.CompositeType {
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

func (e *DesugaredElaboration) SetCompositeDeclarationType(
	declaration ast.CompositeLikeDeclaration,
	compositeType *sema.CompositeType,
) {
	if e.compositeDeclarationTypes == nil {
		e.compositeDeclarationTypes = map[ast.CompositeLikeDeclaration]*sema.CompositeType{}
	}
	e.compositeDeclarationTypes[declaration] = compositeType
}

func (e *DesugaredElaboration) CompositeType(typeID common.TypeID) *sema.CompositeType {
	// First lookup in the extended type info
	if e.compositeTypes != nil {
		typ, ok := e.compositeTypes[typeID]
		if ok {
			return typ
		}
	}

	// If not found, then fallback and look in the original elaboration
	return e.elaboration.CompositeType(typeID)
}

func (e *DesugaredElaboration) SetCompositeType(typeID common.TypeID, compositeType *sema.CompositeType) {
	if e.compositeTypes == nil {
		e.compositeTypes = map[common.TypeID]*sema.CompositeType{}
	}
	e.compositeTypes[typeID] = compositeType
}

func (e *DesugaredElaboration) InterfaceType(typeID common.TypeID) *sema.InterfaceType {
	return e.elaboration.InterfaceType(typeID)
}

func (e *DesugaredElaboration) EntitlementType(typeID common.TypeID) *sema.EntitlementType {
	return e.elaboration.EntitlementType(typeID)
}

func (e *DesugaredElaboration) EntitlementMapType(typeID common.TypeID) *sema.EntitlementMapType {
	return e.elaboration.EntitlementMapType(typeID)
}

func (e *DesugaredElaboration) InvocationExpressionTypes(
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

func (e *DesugaredElaboration) SetInvocationExpressionTypes(
	expression *ast.InvocationExpression,
	types sema.InvocationExpressionTypes,
) {
	if e.invocationExpressionTypes == nil {
		e.invocationExpressionTypes = map[*ast.InvocationExpression]sema.InvocationExpressionTypes{}
	}
	e.invocationExpressionTypes[expression] = types
}

func (e *DesugaredElaboration) MemberExpressionMemberAccessInfo(expression *ast.MemberExpression) (memberInfo sema.MemberAccessInfo, ok bool) {
	if e.memberExpressionMemberAccessInfos != nil {
		memberInfo, ok = e.memberExpressionMemberAccessInfos[expression]
		if ok {
			return
		}
	}
	return e.elaboration.MemberExpressionMemberAccessInfo(expression)
}

func (e *DesugaredElaboration) SetMemberExpressionMemberAccessInfo(expression *ast.MemberExpression, memberAccessInfo sema.MemberAccessInfo) {
	if e.memberExpressionMemberAccessInfos == nil {
		e.memberExpressionMemberAccessInfos = map[*ast.MemberExpression]sema.MemberAccessInfo{}
	}
	e.memberExpressionMemberAccessInfos[expression] = memberAccessInfo
}

func (e *DesugaredElaboration) AssignmentStatementTypes(assignment *ast.AssignmentStatement) sema.AssignmentStatementTypes {
	if e.assignmentStatementTypes != nil {
		types, ok := e.assignmentStatementTypes[assignment]
		if ok {
			return types
		}
	}
	return e.elaboration.AssignmentStatementTypes(assignment)
}

func (e *DesugaredElaboration) SetAssignmentStatementTypes(
	assignment *ast.AssignmentStatement,
	types sema.AssignmentStatementTypes,
) {
	if e.assignmentStatementTypes == nil {
		e.assignmentStatementTypes = map[*ast.AssignmentStatement]sema.AssignmentStatementTypes{}
	}
	e.assignmentStatementTypes[assignment] = types
}

func (e *DesugaredElaboration) TransactionDeclarationType(declaration *ast.TransactionDeclaration) *sema.TransactionType {
	return e.elaboration.TransactionDeclarationType(declaration)
}

func (e *DesugaredElaboration) IntegerExpressionType(expression *ast.IntegerExpression) sema.Type {
	return e.elaboration.IntegerExpressionType(expression)
}

func (e *DesugaredElaboration) FixedPointExpressionType(expression *ast.FixedPointExpression) sema.Type {
	return e.elaboration.FixedPointExpression(expression)
}

func (e *DesugaredElaboration) ArrayExpressionTypes(expression *ast.ArrayExpression) sema.ArrayExpressionTypes {
	return e.elaboration.ArrayExpressionTypes(expression)
}

func (e *DesugaredElaboration) DictionaryExpressionTypes(expression *ast.DictionaryExpression) sema.DictionaryExpressionTypes {
	return e.elaboration.DictionaryExpressionTypes(expression)
}

func (e *DesugaredElaboration) CastingExpressionTypes(expression *ast.CastingExpression) sema.CastingExpressionTypes {
	return e.elaboration.CastingExpressionTypes(expression)
}

func (e *DesugaredElaboration) EmitStatementEventType(statement *ast.EmitStatement) *sema.CompositeType {
	if e.emitStatementEventTypes != nil {
		types, ok := e.emitStatementEventTypes[statement]
		if ok {
			return types
		}
	}
	return e.elaboration.EmitStatementEventType(statement)
}

func (e *DesugaredElaboration) SetEmitStatementEventType(statement *ast.EmitStatement, compositeType *sema.CompositeType) {
	if e.emitStatementEventTypes == nil {
		e.emitStatementEventTypes = map[*ast.EmitStatement]*sema.CompositeType{}
	}
	e.emitStatementEventTypes[statement] = compositeType
}

func (e *DesugaredElaboration) ResultVariableType(enclosingBlock ast.Element) (typ sema.Type, exist bool) {
	if e.resultVariableTypes != nil {
		types, ok := e.resultVariableTypes[enclosingBlock]
		if ok {
			return types, ok
		}
	}
	return e.elaboration.ResultVariableType(enclosingBlock)
}

func (e *DesugaredElaboration) SetResultVariableType(declaration ast.Element, typ sema.Type) {
	if e.resultVariableTypes == nil {
		e.resultVariableTypes = map[ast.Element]sema.Type{}
	}
	e.resultVariableTypes[declaration] = typ
}

func (e *DesugaredElaboration) ReferenceExpressionBorrowType(expression *ast.ReferenceExpression) sema.Type {
	if e.referenceExpressionBorrowTypes != nil {
		typ, ok := e.referenceExpressionBorrowTypes[expression]
		if ok {
			return typ
		}
	}
	return e.elaboration.ReferenceExpressionBorrowType(expression)
}

func (e *DesugaredElaboration) SetReferenceExpressionBorrowType(expression *ast.ReferenceExpression, ty sema.Type) {
	if e.referenceExpressionBorrowTypes == nil {
		e.referenceExpressionBorrowTypes = map[*ast.ReferenceExpression]sema.Type{}
	}
	e.referenceExpressionBorrowTypes[expression] = ty
}

func (e *DesugaredElaboration) FunctionDeclarationFunctionType(declaration *ast.FunctionDeclaration) *sema.FunctionType {
	if e.functionDeclarationFunctionTypes != nil {
		typ, ok := e.functionDeclarationFunctionTypes[declaration]
		if ok {
			return typ
		}
	}
	return e.elaboration.FunctionDeclarationFunctionType(declaration)
}

func (e *DesugaredElaboration) SetFunctionDeclarationFunctionType(
	declaration *ast.FunctionDeclaration,
	functionType *sema.FunctionType,
) {
	if e.functionDeclarationFunctionTypes == nil {
		e.functionDeclarationFunctionTypes = map[*ast.FunctionDeclaration]*sema.FunctionType{}
	}
	e.functionDeclarationFunctionTypes[declaration] = functionType
}

func (e *DesugaredElaboration) FunctionExpressionFunctionType(expression *ast.FunctionExpression) *sema.FunctionType {
	return e.elaboration.FunctionExpressionFunctionType(expression)
}

func (e *DesugaredElaboration) IndexExpressionTypes(expression *ast.IndexExpression) (types sema.IndexExpressionTypes, contains bool) {
	return e.elaboration.IndexExpressionTypes(expression)
}

func (e *DesugaredElaboration) InterfaceTypeDeclaration(interfaceType *sema.InterfaceType) *ast.InterfaceDeclaration {
	return e.elaboration.InterfaceTypeDeclaration(interfaceType)
}

func (e *DesugaredElaboration) AllImportDeclarationsResolvedLocations() map[*ast.ImportDeclaration][]sema.ResolvedLocation {
	return e.elaboration.AllImportDeclarationsResolvedLocations()
}

// OriginalElaboration returns the underlying elaboration.
// IMPORTANT: Only use the original elaboration for type-checking use-cases.
// It is safe to use in type-checker, since the type checker doesn't rely on
// extra information added by the desugar phase.
// Never use it in the compiler or desugar, as it may not include the
// type information added during the desugar phase.
func (e *DesugaredElaboration) OriginalElaboration() *sema.Elaboration {
	return e.elaboration
}

func (e *DesugaredElaboration) StringExpressionType(expression *ast.StringExpression) sema.Type {
	return e.elaboration.StringExpressionType(expression)
}

func (e *DesugaredElaboration) ForStatementType(statement *ast.ForStatement) (types sema.ForStatementTypes) {
	return e.elaboration.ForStatementType(statement)
}
