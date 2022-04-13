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

package compiler

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/compiler/ir"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

type Compiler struct {
	Checker     *sema.Checker
	activations *LocalActivations
	locals      []*Local
}

func NewCompiler(checker *sema.Checker) *Compiler {
	return &Compiler{
		Checker:     checker,
		activations: &LocalActivations{},
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

func (compiler *Compiler) VisitReturnStatement(statement *ast.ReturnStatement) ast.Repr {
	exp := statement.Expression.Accept(compiler).(ir.Expr)
	return &ir.Return{
		Exp: exp,
	}
}

func (compiler *Compiler) VisitBreakStatement(_ *ast.BreakStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitContinueStatement(_ *ast.ContinueStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIfStatement(_ *ast.IfStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitWhileStatement(_ *ast.WhileStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitForStatement(_ *ast.ForStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitEmitStatement(_ *ast.EmitStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitSwitchStatement(_ *ast.SwitchStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Repr {

	// TODO: potential storage removal
	// TODO: copy and convert
	// TODO: second value

	identifier := declaration.Identifier.Identifier
	targetType := compiler.Checker.Elaboration.VariableDeclarationTargetTypes[declaration]
	valType := compileValueType(targetType)
	local := compiler.declareLocal(identifier, valType)
	exp := declaration.Value.Accept(compiler).(ir.Expr)

	return &ir.StoreLocal{
		LocalIndex: local.Index,
		Exp:        exp,
	}
}

func (compiler *Compiler) VisitAssignmentStatement(_ *ast.AssignmentStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitSwapStatement(_ *ast.SwapStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitExpressionStatement(_ *ast.ExpressionStatement) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitBoolExpression(_ *ast.BoolExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIntegerExpression(expression *ast.IntegerExpression) ast.Repr {
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

func (compiler *Compiler) VisitFixedPointExpression(_ *ast.FixedPointExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitArrayExpression(_ *ast.ArrayExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitDictionaryExpression(_ *ast.DictionaryExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIdentifierExpression(expression *ast.IdentifierExpression) ast.Repr {
	// TODO
	local := compiler.findLocal(expression.Identifier.Identifier)
	// TODO: moves
	return &ir.CopyLocal{
		LocalIndex: local.Index,
	}
}

func (compiler *Compiler) VisitInvocationExpression(_ *ast.InvocationExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitMemberExpression(_ *ast.MemberExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitIndexExpression(_ *ast.IndexExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitConditionalExpression(_ *ast.ConditionalExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitUnaryExpression(_ *ast.UnaryExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitBinaryExpression(expression *ast.BinaryExpression) ast.Repr {
	op := compileBinaryOperation(expression.Operation)
	left := expression.Left.Accept(compiler).(ir.Expr)
	right := expression.Right.Accept(compiler).(ir.Expr)

	return &ir.BinOpExpr{
		Op:    op,
		Left:  left,
		Right: right,
	}
}

func (compiler *Compiler) VisitFunctionExpression(_ *ast.FunctionExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitStringExpression(e *ast.StringExpression) ast.Repr {
	return &ir.Const{
		Constant: ir.String{
			Value: e.Value,
		},
	}
}

func (compiler *Compiler) VisitCastingExpression(_ *ast.CastingExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitCreateExpression(_ *ast.CreateExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitDestroyExpression(_ *ast.DestroyExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitReferenceExpression(_ *ast.ReferenceExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitForceExpression(_ *ast.ForceExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitPathExpression(_ *ast.PathExpression) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitProgram(_ *ast.Program) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Repr {

	// TODO: declare function in current scope, use current scope in function
	// TODO: conditions

	compiler.locals = nil

	block := declaration.FunctionBlock.Block

	// Declare a local for each parameter

	functionType := compiler.Checker.Elaboration.FunctionDeclarationFunctionTypes[declaration]

	parameters := declaration.ParameterList.Parameters

	for i, parameter := range parameters {
		parameterType := functionType.Parameters[i].TypeAnnotation.Type
		valType := compileValueType(parameterType)
		name := parameter.Identifier.Identifier
		compiler.declareLocal(name, valType)
	}

	// Compile the function block

	stmt := block.Accept(compiler).(ir.Stmt)

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

func (compiler *Compiler) VisitBlock(block *ast.Block) ast.Repr {

	// Block scope: each block gets an activation record

	compiler.activations.PushNewWithCurrent()
	defer compiler.activations.Pop()

	// Compile each statement in the block

	stmts := make([]ir.Stmt, len(block.Statements))
	for i, statement := range block.Statements {
		stmts[i] = statement.Accept(compiler).(ir.Stmt)
	}

	// NOTE: just return an IR statement sequence,
	// there is no need for an IR block
	return &ir.Sequence{
		Stmts: stmts,
	}
}

func (compiler *Compiler) VisitFunctionBlock(_ *ast.FunctionBlock) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitCompositeDeclaration(_ *ast.CompositeDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitFieldDeclaration(_ *ast.FieldDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitCondition(_ *ast.Condition) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitImportDeclaration(_ *ast.ImportDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) ast.Repr {
	// TODO
	panic(errors.NewUnreachableError())
}

func (compiler *Compiler) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) ast.Repr {
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
