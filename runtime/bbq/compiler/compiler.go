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
	"math"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

type Compiler struct {
	Program     *ast.Program
	Elaboration *sema.Elaboration

	currentFunction *function
	functions       []*function
	constants       []*constant
	globals         map[string]*global
	loops           []*loop
	currentLoop     *loop

	currentCompositeType *sema.CompositeType
}

var _ ast.DeclarationVisitor[struct{}] = &Compiler{}
var _ ast.StatementVisitor[struct{}] = &Compiler{}
var _ ast.ExpressionVisitor[struct{}] = &Compiler{}

func NewCompiler(
	program *ast.Program,
	elaboration *sema.Elaboration,
) *Compiler {
	return &Compiler{
		Program:     program,
		Elaboration: elaboration,
		globals:     map[string]*global{},
	}
}

func (c *Compiler) findGlobal(name string) *global {
	return c.globals[name]
}

func (c *Compiler) addGlobal(name string) *global {
	count := len(c.globals)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid global declaration"))
	}
	global := &global{
		index: uint16(count),
	}
	c.globals[name] = global
	return global
}

func (c *Compiler) addFunction(name string, parameterCount uint16) *function {
	c.addGlobal(name)
	function := newFunction(name, parameterCount)
	c.functions = append(c.functions, function)
	c.currentFunction = function
	return function
}

func (c *Compiler) addConstant(kind constantkind.ConstantKind, data []byte) *constant {
	count := len(c.constants)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid constant declaration"))
	}
	constant := &constant{
		index: uint16(count),
		kind:  kind,
		data:  data[:],
	}
	c.constants = append(c.constants, constant)
	return constant
}

func (c *Compiler) stringConstLoad(str string) {
	constant := c.addConstant(constantkind.String, []byte(str))
	first, second := encodeUint16(constant.index)
	c.emit(opcode.GetConstant, first, second)
}

func (c *Compiler) emit(opcode opcode.Opcode, args ...byte) int {
	return c.currentFunction.emit(opcode, args...)
}

func (c *Compiler) emitUndefinedJump(opcode opcode.Opcode) int {
	return c.emit(opcode, 0xff, 0xff)
}

func (c *Compiler) emitJump(opcode opcode.Opcode, target int) int {
	if target >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	first, second := encodeUint16(uint16(target))
	return c.emit(opcode, first, second)
}

func (c *Compiler) patchJump(opcodeOffset int) {
	code := c.currentFunction.code
	count := len(code)
	if count == 0 {
		panic(errors.NewUnreachableError())
	}
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	target := uint16(count)
	first, second := encodeUint16(target)
	code[opcodeOffset+1] = first
	code[opcodeOffset+2] = second
}

// encodeUint16 encodes the given uint16 in big-endian representation
func encodeUint16(jump uint16) (byte, byte) {
	return byte((jump >> 8) & 0xff),
		byte(jump & 0xff)
}

func (c *Compiler) pushLoop(start int) {
	loop := &loop{
		start: start,
	}
	c.loops = append(c.loops, loop)
	c.currentLoop = loop
}

func (c *Compiler) popLoop() {
	lastIndex := len(c.loops) - 1
	l := c.loops[lastIndex]
	c.loops[lastIndex] = nil
	c.loops = c.loops[:lastIndex]

	c.patchLoop(l)

	var previousLoop *loop
	if lastIndex > 0 {
		previousLoop = c.loops[lastIndex]
	}
	c.currentLoop = previousLoop
}

func (c *Compiler) Compile() *bbq.Program {
	for _, declaration := range c.Program.Declarations() {
		c.compileDeclaration(declaration)
	}

	functions := c.exportFunctions()
	constants := c.exportConstants()

	return &bbq.Program{
		Functions: functions,
		Constants: constants,
	}
}

func (c *Compiler) exportConstants() []*bbq.Constant {
	constants := make([]*bbq.Constant, 0, len(c.constants))
	for _, constant := range c.constants {
		constants = append(
			constants,
			&bbq.Constant{
				Data: constant.data,
				Kind: constant.kind,
			},
		)
	}
	return constants
}

func (c *Compiler) exportFunctions() []*bbq.Function {
	functions := make([]*bbq.Function, 0, len(c.functions))
	for _, function := range c.functions {
		functions = append(
			functions,
			&bbq.Function{
				Name:           function.name,
				Code:           function.code,
				LocalCount:     function.localCount,
				ParameterCount: function.parameterCount,
			},
		)
	}
	return functions
}

func (c *Compiler) compileDeclaration(declaration ast.Declaration) {
	ast.AcceptDeclaration[struct{}](declaration, c)
}

func (c *Compiler) compileBlock(block *ast.Block) {
	// TODO: scope
	for _, statement := range block.Statements {
		c.compileStatement(statement)
	}
}

func (c *Compiler) compileFunctionBlock(functionBlock *ast.FunctionBlock) {
	// TODO: pre and post conditions, incl. interfaces
	c.compileBlock(functionBlock.Block)
}

func (c *Compiler) compileStatement(statement ast.Statement) {
	ast.AcceptStatement[struct{}](statement, c)
}

func (c *Compiler) compileExpression(expression ast.Expression) {
	ast.AcceptExpression[struct{}](expression, c)
}

func (c *Compiler) VisitReturnStatement(statement *ast.ReturnStatement) (_ struct{}) {
	expression := statement.Expression
	if expression != nil {
		// TODO: copy
		c.compileExpression(expression)
		c.emit(opcode.ReturnValue)
	} else {
		c.emit(opcode.Return)
	}
	return
}

func (c *Compiler) VisitBreakStatement(_ *ast.BreakStatement) (_ struct{}) {
	offset := len(c.currentFunction.code)
	c.currentLoop.breaks = append(c.currentLoop.breaks, offset)
	c.emitUndefinedJump(opcode.Jump)
	return
}

func (c *Compiler) VisitContinueStatement(_ *ast.ContinueStatement) (_ struct{}) {
	c.emitJump(opcode.Jump, c.currentLoop.start)
	return
}

func (c *Compiler) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {
	// TODO: scope
	switch test := statement.Test.(type) {
	case ast.Expression:
		c.compileExpression(test)
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	elseJump := c.emitUndefinedJump(opcode.JumpIfFalse)
	c.compileBlock(statement.Then)
	elseBlock := statement.Else
	if elseBlock != nil {
		thenJump := c.emit(opcode.Jump)
		c.patchJump(elseJump)
		c.compileBlock(elseBlock)
		c.patchJump(thenJump)
	} else {
		c.patchJump(elseJump)
	}
	return
}

func (c *Compiler) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {
	testOffset := len(c.currentFunction.code)
	c.pushLoop(testOffset)
	c.compileExpression(statement.Test)
	endJump := c.emitUndefinedJump(opcode.JumpIfFalse)
	c.compileBlock(statement.Block)
	c.emitJump(opcode.Jump, testOffset)
	c.patchJump(endJump)
	c.popLoop()
	return
}

func (c *Compiler) VisitForStatement(_ *ast.ForStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitEmitStatement(_ *ast.EmitStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitSwitchStatement(_ *ast.SwitchStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {
	// TODO: second value
	c.compileExpression(declaration.Value)
	local := c.currentFunction.declareLocal(declaration.Identifier.Identifier)
	first, second := encodeUint16(local.index)
	c.emit(opcode.SetLocal, first, second)
	return
}

func (c *Compiler) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {
	c.compileExpression(statement.Value)
	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		local := c.currentFunction.findLocal(target.Identifier.Identifier)
		first, second := encodeUint16(local.index)
		c.emit(opcode.SetLocal, first, second)
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	return
}

func (c *Compiler) VisitSwapStatement(_ *ast.SwapStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitExpressionStatement(statement *ast.ExpressionStatement) (_ struct{}) {
	c.compileExpression(statement.Expression)
	c.emit(opcode.Pop)
	return
}

func (c *Compiler) VisitVoidExpression(_ *ast.VoidExpression) (_ struct{}) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitBoolExpression(expression *ast.BoolExpression) (_ struct{}) {
	if expression.Value {
		c.emit(opcode.True)
	} else {
		c.emit(opcode.False)
	}
	return
}

func (c *Compiler) VisitNilExpression(_ *ast.NilExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIntegerExpression(expression *ast.IntegerExpression) (_ struct{}) {
	integerType := c.Elaboration.IntegerExpressionType[expression]
	constantKind := constantkind.FromSemaType(integerType)

	// TODO:
	var data []byte
	data = leb128.AppendInt64(data, expression.Value.Int64())

	constant := c.addConstant(constantKind, data)
	first, second := encodeUint16(constant.index)
	c.emit(opcode.GetConstant, first, second)
	return
}

func (c *Compiler) VisitFixedPointExpression(_ *ast.FixedPointExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitArrayExpression(_ *ast.ArrayExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitDictionaryExpression(_ *ast.DictionaryExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIdentifierExpression(expression *ast.IdentifierExpression) (_ struct{}) {
	name := expression.Identifier.Identifier
	local := c.currentFunction.findLocal(name)
	if local != nil {
		first, second := encodeUint16(local.index)
		c.emit(opcode.GetLocal, first, second)
		return
	}
	global := c.findGlobal(name)
	first, second := encodeUint16(global.index)
	c.emit(opcode.GetGlobal, first, second)
	return
}

func (c *Compiler) VisitInvocationExpression(expression *ast.InvocationExpression) (_ struct{}) {
	// TODO: copy
	for _, argument := range expression.Arguments {
		c.compileExpression(argument.Expression)
	}
	c.compileExpression(expression.InvokedExpression)
	c.emit(opcode.Call)
	return
}

func (c *Compiler) VisitMemberExpression(_ *ast.MemberExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIndexExpression(_ *ast.IndexExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitConditionalExpression(_ *ast.ConditionalExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitUnaryExpression(_ *ast.UnaryExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitBinaryExpression(expression *ast.BinaryExpression) (_ struct{}) {
	c.compileExpression(expression.Left)
	c.compileExpression(expression.Right)
	// TODO: add support for other types
	c.emit(intBinaryOpcodes[expression.Operation])
	return
}

var intBinaryOpcodes = [...]opcode.Opcode{
	ast.OperationPlus:         opcode.IntAdd,
	ast.OperationMinus:        opcode.IntSubtract,
	ast.OperationMul:          opcode.IntMultiply,
	ast.OperationDiv:          opcode.IntDivide,
	ast.OperationMod:          opcode.IntMod,
	ast.OperationEqual:        opcode.IntEqual,
	ast.OperationNotEqual:     opcode.IntNotEqual,
	ast.OperationLess:         opcode.IntLess,
	ast.OperationLessEqual:    opcode.IntLessOrEqual,
	ast.OperationGreater:      opcode.IntGreater,
	ast.OperationGreaterEqual: opcode.IntGreaterOrEqual,
}

func (c *Compiler) VisitFunctionExpression(_ *ast.FunctionExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitStringExpression(expression *ast.StringExpression) (_ struct{}) {
	c.stringConstLoad(expression.Value)
	return
}

func (c *Compiler) VisitCastingExpression(_ *ast.CastingExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitCreateExpression(_ *ast.CreateExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitDestroyExpression(_ *ast.DestroyExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitReferenceExpression(_ *ast.ReferenceExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitForceExpression(_ *ast.ForceExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitPathExpression(_ *ast.PathExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) (_ struct{}) {
	enclosingCompositeTypeName := c.currentCompositeType.Identifier

	var functionName string
	kind := declaration.DeclarationKind()
	switch kind {
	case common.DeclarationKindInitializer:
		functionName = enclosingCompositeTypeName
	default:
		// TODO: support other special functions
		panic(errors.NewUnreachableError())
	}

	parameter := declaration.FunctionDeclaration.ParameterList.Parameters
	parameterCount := len(parameter)
	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	function := c.addFunction(functionName, uint16(parameterCount))

	// TODO: pass location
	c.stringConstLoad(enclosingCompositeTypeName)

	// Declare `self`
	self := c.currentFunction.declareLocal(sema.SelfIdentifier)
	selfFirst, selfSecond := encodeUint16(self.index)

	for _, parameter := range parameter {
		parameterName := parameter.Identifier.Identifier
		function.declareLocal(parameterName)
	}

	// Initialize an empty struct and assign to `self`.
	// i.e: `self = New()`
	c.emit(opcode.New)
	c.emit(opcode.SetLocal, selfFirst, selfSecond)

	// Emit for the statements in `init()` body.
	c.compileFunctionBlock(declaration.FunctionDeclaration.FunctionBlock)

	// Constructor should return the created the struct. i.e: return `self`
	c.emit(opcode.GetLocal, selfFirst, selfSecond)
	c.emit(opcode.ReturnValue)

	return
}

func (c *Compiler) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) (_ struct{}) {
	// TODO: handle nested functions
	functionName := declaration.Identifier.Identifier
	functionType := c.Elaboration.FunctionDeclarationFunctionTypes[declaration]
	parameterCount := len(functionType.Parameters)
	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}
	function := c.addFunction(functionName, uint16(parameterCount))
	for _, parameter := range declaration.ParameterList.Parameters {
		parameterName := parameter.Identifier.Identifier
		function.declareLocal(parameterName)
	}
	c.compileFunctionBlock(declaration.FunctionBlock)
	return
}

func (c *Compiler) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	prevCompositeType := c.currentCompositeType
	c.currentCompositeType = c.Elaboration.CompositeDeclarationTypes[declaration]
	defer func() {
		c.currentCompositeType = prevCompositeType
	}()

	for _, specialFunc := range declaration.Members.SpecialFunctions() {
		c.compileDeclaration(specialFunc)
	}

	// TODO:

	return
}

func (c *Compiler) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitFieldDeclaration(_ *ast.FieldDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitImportDeclaration(_ *ast.ImportDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) patchLoop(l *loop) {
	for _, breakOffset := range l.breaks {
		c.patchJump(breakOffset)
	}
}
