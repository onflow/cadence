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

package compiler

import (
	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

type Compiler struct {
	Checker     *sema.Checker
	activations *activations.Activations[*Local]
	locals      []*Local
}

var _ ast.DeclarationVisitor[ir.Stmt] = &Compiler{}
var _ ast.StatementVisitor[ir.Stmt] = &Compiler{}
var _ ast.ExpressionVisitor[ir.Expr] = &Compiler{}

func NewCompiler(checker *sema.Checker) *Compiler {
	return &Compiler{
		Checker:     checker,
		activations: activations.NewActivations[*Local](nil),
	}
}

// declareLocal declares a local
func (compiler *Compiler) declareLocal(identifier string, valType ir.ValType) *Local {
	// NOTE: semantic analysis already checked possible invalid redeclaration
	index := uint32(len(compiler.locals))
	local := NewLocal(index, valType)
	compiler.locals = append(compiler.locals, local)
	compiler.setLocal(identifier, local)
	return local
}

func (compiler *Compiler) findLocal(name string) *Local {
	return compiler.activations.Find(name)
}

func (compiler *Compiler) setLocal(name string, variable *Local) {
	compiler.activations.Set(name, variable)
}

func (compiler *Compiler) VisitReturnStatement(statement *ast.ReturnStatement) ir.Stmt {
	exp := ast.AcceptExpression[ir.Expr](statement.Expression, compiler)
	return &ir.Return{
		Exp: exp,
	}
}

func (compiler *Compiler) VisitBreakStatement(_ *ast.BreakStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitContinueStatement(_ *ast.ContinueStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIfStatement(_ *ast.IfStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitWhileStatement(_ *ast.WhileStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitForStatement(_ *ast.ForStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitEmitStatement(_ *ast.EmitStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitSwitchStatement(_ *ast.SwitchStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ir.Stmt {

	// TODO: potential storage removal
	// TODO: copy and convert
	// TODO: second value

	identifier := declaration.Identifier.Identifier
	targetType := compiler.Checker.Elaboration.VariableDeclarationTypes(declaration).TargetType
	valType := compileValueType(targetType)
	local := compiler.declareLocal(identifier, valType)
	exp := ast.AcceptExpression[ir.Expr](declaration.Value, compiler)

	return &ir.StoreLocal{
		LocalIndex: local.Index,
		Exp:        exp,
	}
}

func (compiler *Compiler) VisitAssignmentStatement(_ *ast.AssignmentStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitSwapStatement(_ *ast.SwapStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitExpressionStatement(_ *ast.ExpressionStatement) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitVoidExpression(_ *ast.VoidExpression) ir.Expr {
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitBoolExpression(_ *ast.BoolExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitNilExpression(_ *ast.NilExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIntegerExpression(expression *ast.IntegerExpression) ir.Expr {
	var value []byte

	if expression.Value.Sign() < 0 {
		value = append(value, 0)
	} else {
		value = append(value, 1)
	}

	value = append(value,
		expression.Value.Bytes()...,
	)

	return &ir.Const{
		Constant: ir.Int{
			Value: value,
		},
	}
}

func (compiler *Compiler) VisitFixedPointExpression(_ *ast.FixedPointExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitArrayExpression(_ *ast.ArrayExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitDictionaryExpression(_ *ast.DictionaryExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIdentifierExpression(expression *ast.IdentifierExpression) ir.Expr {
	// TODO
	local := compiler.findLocal(expression.Identifier.Identifier)
	// TODO: moves
	return &ir.CopyLocal{
		LocalIndex: local.Index,
	}
}

func (compiler *Compiler) VisitInvocationExpression(_ *ast.InvocationExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitMemberExpression(_ *ast.MemberExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIndexExpression(_ *ast.IndexExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitConditionalExpression(_ *ast.ConditionalExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitUnaryExpression(_ *ast.UnaryExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitBinaryExpression(expression *ast.BinaryExpression) ir.Expr {
	op := compileBinaryOperation(expression.Operation)
	left := ast.AcceptExpression[ir.Expr](expression.Left, compiler)
	right := ast.AcceptExpression[ir.Expr](expression.Right, compiler)

	return &ir.BinOpExpr{
		Op:    op,
		Left:  left,
		Right: right,
	}
}

func (compiler *Compiler) VisitFunctionExpression(_ *ast.FunctionExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitStringExpression(e *ast.StringExpression) ir.Expr {
	return &ir.Const{
		Constant: ir.String{
			Value: e.Value,
		},
	}
}

func (compiler *Compiler) VisitCastingExpression(_ *ast.CastingExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitCreateExpression(_ *ast.CreateExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitDestroyExpression(_ *ast.DestroyExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitReferenceExpression(_ *ast.ReferenceExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitForceExpression(_ *ast.ForceExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitPathExpression(_ *ast.PathExpression) ir.Expr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitProgram(_ *ast.Program) ir.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) ir.Stmt {
	return compiler.VisitFunctionDeclaration(declaration.FunctionDeclaration)
}

func (compiler *Compiler) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ir.Stmt {

	// TODO: declare function in current scope, use current scope in function
	// TODO: conditions

	compiler.locals = nil

	block := declaration.FunctionBlock.Block

	// Declare a local for each parameter

	functionType := compiler.Checker.Elaboration.FunctionDeclarationFunctionType(declaration)

	parameters := declaration.ParameterList.Parameters

	for i, parameter := range parameters {
		parameterType := functionType.Parameters[i].TypeAnnotation.Type
		valType := compileValueType(parameterType)
		name := parameter.Identifier.Identifier
		compiler.declareLocal(name, valType)
	}

	// Compile the function block

	stmt := compiler.visitBlock(block)

	// Important: compile locals after compiling function block,
	// and don't include parameters in locals
	locals := compileLocals(compiler.locals[len(parameters):])

	compiledFunctionType := compileFunctionType(functionType)

	return &ir.Func{
		// TODO: fully qualify
		Name:      declaration.Identifier.Identifier,
		Type:      compiledFunctionType,
		Locals:    locals,
		Statement: stmt,
	}
}

func (compiler *Compiler) visitBlock(block *ast.Block) ir.Stmt {

	// Block scope: each block gets an activation record

	compiler.activations.PushNewWithCurrent()
	defer compiler.activations.Pop()

	// Compile each statement in the block

	stmts := make([]ir.Stmt, len(block.Statements))
	for i, statement := range block.Statements {
		stmts[i] = ast.AcceptStatement[ir.Stmt](statement, compiler)
	}

	// NOTE: just return an IR statement sequence,
	// there is no need for an IR block
	return &ir.Sequence{
		Stmts: stmts,
	}
}

func (compiler *Compiler) VisitCompositeDeclaration(_ *ast.CompositeDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitFieldDeclaration(_ *ast.FieldDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitImportDeclaration(_ *ast.ImportDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitTransactionRoleDeclaration(_ *ast.TransactionRoleDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) ir.Stmt {
	// TODO
	panic(errors.NewUnreachableError())
}

func compileBinaryOperation(operation ast.Operation) ir.BinOp {
	// TODO: add remaining operations
	switch operation {
	case ast.OperationPlus:
		return ir.BinOpPlus
	}

	panic(errors.NewUnreachableError())
}

func compileValueType(ty sema.Type) ir.ValType {
	// TODO: add remaining types

	switch ty {
	case sema.StringType:
		return ir.ValTypeString
	case sema.IntType:
		return ir.ValTypeInt
	}

	panic(errors.NewUnreachableError())
}

func compileFunctionType(functionType *sema.FunctionType) ir.FuncType {
	// compile parameter types
	paramTypes := make([]ir.ValType, len(functionType.Parameters))
	for i, parameter := range functionType.Parameters {
		paramTypes[i] = compileValueType(parameter.TypeAnnotation.Type)
	}

	// compile return / result type
	var resultTypes []ir.ValType
	if functionType.ReturnTypeAnnotation.Type != sema.VoidType {
		resultTypes = []ir.ValType{
			compileValueType(functionType.ReturnTypeAnnotation.Type),
		}
	}
	return ir.FuncType{
		Params:  paramTypes,
		Results: resultTypes,
	}
}

func compileLocals(locals []*Local) []ir.Local {
	result := make([]ir.Local, len(locals))
	for i, local := range locals {
		result[i] = ir.Local{
			Type: local.Type,
		}
	}
	return result
}
