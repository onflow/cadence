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
	"fmt"
	"github.com/onflow/cadence/runtime/bbq/registers"
	"math"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
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
}

var _ ast.DeclarationVisitor[struct{}] = &Compiler{}
var _ ast.StatementVisitor[struct{}] = &Compiler{}
var _ ast.ExpressionVisitor[uint16] = &Compiler{}

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

func (c *Compiler) addGlobal(name string, registryType registers.RegistryType) *global {
	count := len(c.globals)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid global declaration"))
	}
	global := &global{
		index:   uint16(count),
		regType: registryType,
	}
	c.globals[name] = global
	return global
}

func (c *Compiler) addFunction(name string, parameterCount uint16) *function {
	c.addGlobal(name, registers.Func)
	function := newFunction(name, parameterCount)
	c.functions = append(c.functions, function)
	c.currentFunction = function
	return function
}

func (c *Compiler) addConstant(kind constantkind.Constant, data []byte) *constant {
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

func (c *Compiler) emit(opcode opcode.Opcode) int {
	return c.currentFunction.emit(opcode)
}

func (c *Compiler) emitAt(index int, opcode opcode.Opcode) {
	c.currentFunction.emitAt(index, opcode)
}

func (c *Compiler) emitEmpty() int {
	return c.currentFunction.emit(nil)
}

func (c *Compiler) emitJump(target int) int {
	return c.emit(opcode.Jump{
		Target: uint16(target),
	})
}

func (c *Compiler) lastInstructionIndex() int {
	code := c.currentFunction.code
	return len(code)
}

//
//func (c *Compiler) patchJump(opcodeOffset int) {
//	code := c.currentFunction.code
//	count := len(code)
//	if count == 0 {
//		panic(errors.NewUnreachableError())
//	}
//	if count >= math.MaxUint16 {
//		panic(errors.NewDefaultUserError("invalid jump"))
//	}
//
//	target := uint16(count)
//	//first, second := encodeUint16(target)
//
//	switch jump := code[opcodeOffset].(type) {
//	case opcode.Jump:
//		jump.Target = target
//	case opcode.JumpIfFalse:
//		jump.Target = target
//	default:
//		panic(errors.NewUnreachableError())
//
//	}
//}

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
	loopEnd := c.lastInstructionIndex()

	lastIndex := len(c.loops) - 1
	l := c.loops[lastIndex]
	c.loops[lastIndex] = nil
	c.loops = c.loops[:lastIndex]

	c.patchLoop(l, loopEnd)

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

func (c *Compiler) compileExpression(expression ast.Expression) uint16 {
	return ast.AcceptExpression[uint16](expression, c)
}

func (c *Compiler) VisitReturnStatement(statement *ast.ReturnStatement) (_ struct{}) {
	expression := statement.Expression
	if expression != nil {
		// TODO: copy
		index := c.compileExpression(expression)
		c.emit(opcode.ReturnValue{
			Index: index,
		})
	} else {
		c.emit(opcode.Return{})
	}
	return
}

func (c *Compiler) VisitBreakStatement(_ *ast.BreakStatement) (_ struct{}) {
	offset := c.lastInstructionIndex()
	c.currentLoop.breaks = append(c.currentLoop.breaks, offset)
	c.emitEmpty()
	return
}

func (c *Compiler) VisitContinueStatement(_ *ast.ContinueStatement) (_ struct{}) {
	c.emit(opcode.Jump{
		// TODO (Supun): handle conversion properly
		Target: uint16(c.currentLoop.start),
	})
	return
}

func (c *Compiler) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {
	// TODO: scope

	var condition uint16
	switch test := statement.Test.(type) {
	case ast.Expression:
		condition = c.compileExpression(test)
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}

	elseJump := c.emitEmpty()

	c.compileBlock(statement.Then)

	elseBlock := statement.Else
	if elseBlock != nil {
		thenJump := c.emitEmpty()

		endOfThen := c.lastInstructionIndex()
		c.emitAt(
			elseJump,
			opcode.JumpIfFalse{
				Condition: condition,
				Target:    uint16(endOfThen),
			},
		)

		c.compileBlock(elseBlock)

		endOfElse := c.lastInstructionIndex()

		c.emitAt(
			thenJump,
			opcode.Jump{
				Target: uint16(endOfElse),
			},
		)
	} else {
		endOfThen := c.lastInstructionIndex()
		c.emitAt(
			elseJump,
			opcode.JumpIfFalse{
				Condition: condition,
				Target:    uint16(endOfThen),
			},
		)
	}

	return
}

func (c *Compiler) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {
	startOfLoop := c.lastInstructionIndex()
	c.pushLoop(startOfLoop)

	condition := c.compileExpression(statement.Test)

	endJump := c.emitEmpty()

	c.compileBlock(statement.Block)
	c.emitJump(startOfLoop)

	endOfLoop := c.lastInstructionIndex()

	c.emitAt(
		endJump,
		opcode.JumpIfFalse{
			Condition: condition,
			Target:    uint16(endOfLoop),
		})

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
	valueIndex := c.compileExpression(declaration.Value)
	regType := c.VariableRegType(declaration.TypeAnnotation)
	local := c.currentFunction.declareLocal(declaration.Identifier.Identifier, regType)

	c.emit(opcode.MoveInt{
		From: valueIndex,
		To:   local.index,
	})

	return
}

func (c *Compiler) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {
	valueIndex := c.compileExpression(statement.Value)

	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		local := c.currentFunction.findLocal(target.Identifier.Identifier)
		c.emit(opcode.MoveInt{
			From: valueIndex,
			To:   local.index,
		})
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

func (c *Compiler) VisitExpressionStatement(_ *ast.ExpressionStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitVoidExpression(_ *ast.VoidExpression) (_ uint16) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitBoolExpression(expression *ast.BoolExpression) (_ uint16) {
	nextIndex := c.nextLocalIndex(registers.Bool)
	if expression.Value {
		c.emit(opcode.True{
			Index: nextIndex,
		})
	} else {
		c.emit(opcode.False{
			Index: nextIndex,
		})
	}
	return
}

func (c *Compiler) VisitNilExpression(_ *ast.NilExpression) (_ uint16) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIntegerExpression(expression *ast.IntegerExpression) uint16 {
	integerType := c.Elaboration.IntegerExpressionType[expression]
	constantKind := constantkind.FromSemaType(integerType)

	// TODO:
	var data []byte
	data = leb128.AppendInt64(data, expression.Value.Int64())

	constant := c.addConstant(constantKind, data)
	index := c.nextLocalIndex(registers.Int)

	c.emit(opcode.GetIntConstant{
		Index:  constant.index,
		Target: index,
	})

	return index
}

func (c *Compiler) VisitFixedPointExpression(_ *ast.FixedPointExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitArrayExpression(_ *ast.ArrayExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitDictionaryExpression(_ *ast.DictionaryExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIdentifierExpression(expression *ast.IdentifierExpression) uint16 {
	name := expression.Identifier.Identifier
	local := c.currentFunction.findLocal(name)

	if local != nil {
		return local.index
	}

	global := c.findGlobal(name)
	if global == nil {
		panic(errors.NewUnreachableError())
	}

	varIndex := c.nextLocalIndex(global.regType)

	//first, second := encodeUint16(global.index)
	c.emit(opcode.GetGlobalFunc{
		Index:  global.index,
		Result: varIndex,
	})

	return varIndex
}

func (c *Compiler) VisitInvocationExpression(expression *ast.InvocationExpression) uint16 {
	params := make([]opcode.Argument, len(expression.Arguments))

	invocationType, ok := c.Elaboration.InvocationExpressionTypes[expression]
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// TODO: copy
	for index, argument := range expression.Arguments {
		regType := registers.RegistryTypeFromSemaType(invocationType.ArgumentTypes[index])
		regIndex := c.compileExpression(argument.Expression)
		params[index] = opcode.Argument{
			Type:  regType,
			Index: regIndex,
		}
	}

	funcIndex := c.compileExpression(expression.InvokedExpression)

	returnType := registers.RegistryTypeFromSemaType(invocationType.ReturnType)
	returnValueIndex := c.nextLocalIndex(returnType)

	c.emit(opcode.Call{
		FuncIndex: funcIndex,
		Arguments: params,
		Result:    returnValueIndex,
	})

	return returnValueIndex
}

func (c *Compiler) VisitMemberExpression(_ *ast.MemberExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitIndexExpression(_ *ast.IndexExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitConditionalExpression(_ *ast.ConditionalExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitUnaryExpression(_ *ast.UnaryExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitBinaryExpression(expression *ast.BinaryExpression) uint16 {
	leftOp := c.compileExpression(expression.Left)
	rightOp := c.compileExpression(expression.Right)

	var opCode opcode.Opcode
	var resultIndex uint16

	// TODO: add support for other types
	switch expression.Operation {
	case ast.OperationPlus:
		resultIndex = c.nextLocalIndex(registers.Int)
		opCode = opcode.IntAdd{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationMinus:
		resultIndex = c.nextLocalIndex(registers.Int)
		opCode = opcode.IntSubtract{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	//case ast.OperationMul:
	//	opCode = opcode.IntMul{
	//		LeftOperand:  leftOp,
	//		RightOperand:  rightOp,
	//		Result:       c.nextLocalIndex(registers.Int),
	//	}
	//case ast.OperationDiv:
	//	opCode = opcode.IntDiv{
	//		LeftOperand:  leftOp,
	//		RightOperand:  rightOp,
	//		Result:       c.nextLocalIndex(registers.Int),
	//	}
	//case ast.OperationMod:
	//	opCode = opcode.IntMod{
	//		LeftOperand:  leftOp,
	//		RightOperand:  rightOp,
	//		Result:       c.nextLocalIndex(registers.Int),
	//	}

	case ast.OperationEqual:
		resultIndex = c.nextLocalIndex(registers.Bool)

		opCode = opcode.IntEqual{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationNotEqual:
		resultIndex = c.nextLocalIndex(registers.Bool)
		opCode = opcode.IntNotEqual{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationLess:
		resultIndex = c.nextLocalIndex(registers.Bool)
		opCode = opcode.IntLess{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationLessEqual:
		resultIndex = c.nextLocalIndex(registers.Bool)
		opCode = opcode.IntLessOrEqual{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationGreater:
		resultIndex = c.nextLocalIndex(registers.Bool)
		opCode = opcode.IntGreater{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	case ast.OperationGreaterEqual:
		resultIndex = c.nextLocalIndex(registers.Bool)
		opCode = opcode.IntGreaterOrEqual{
			LeftOperand:  leftOp,
			RightOperand: rightOp,
			Result:       resultIndex,
		}
	default:
		panic(fmt.Errorf("unsupproted binary op '%s'", expression.Operation))
	}

	c.emit(opCode)

	return resultIndex
}

//var intBinaryOpcodes = [...]opcode.Opcode{
//	ast.OperationPlus:         opcode.IntAdd,
//	ast.OperationMinus:        opcode.IntSubtract,
//	ast.OperationMul:          opcode.IntMultiply,
//	ast.OperationDiv:          opcode.IntDivide,
//	ast.OperationMod:          opcode.IntMod,
//	ast.OperationEqual:        opcode.IntEqual,
//	ast.OperationNotEqual:     opcode.IntNotEqual,
//	ast.OperationLess:         opcode.IntLess,
//	ast.OperationLessEqual:    opcode.IntLessOrEqual,
//	ast.OperationGreater:      opcode.IntGreater,
//	ast.OperationGreaterEqual: opcode.IntGreaterOrEqual,
//}

func (c *Compiler) VisitFunctionExpression(_ *ast.FunctionExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitStringExpression(_ *ast.StringExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitCastingExpression(_ *ast.CastingExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitCreateExpression(_ *ast.CreateExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitDestroyExpression(_ *ast.DestroyExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitReferenceExpression(_ *ast.ReferenceExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitForceExpression(_ *ast.ForceExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitPathExpression(_ *ast.PathExpression) uint16 {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) (_ struct{}) {
	return c.VisitFunctionDeclaration(declaration.FunctionDeclaration)
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

		paramType := c.VariableRegType(parameter.TypeAnnotation)
		function.declareLocal(parameterName, paramType)
	}

	c.compileFunctionBlock(declaration.FunctionBlock)

	return
}

func (c *Compiler) VisitCompositeDeclaration(_ *ast.CompositeDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
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

func (c *Compiler) patchLoop(l *loop, loopEnd int) {
	for _, breakOffset := range l.breaks {
		c.emitAt(
			breakOffset,
			opcode.Jump{
				Target: uint16(loopEnd),
			},
		)
	}
}

func (*Compiler) VariableRegType(typeAnnotation *ast.TypeAnnotation) registers.RegistryType {
	// TODO: switch on semaType
	switch typ := typeAnnotation.Type.(type) {
	case *ast.FunctionType:
		return registers.Func
	case *ast.NominalType:
		switch typ.Identifier.Identifier {
		case "Int":
			return registers.Int
		case "Bool":
			return registers.Bool
		default:
			panic(fmt.Errorf("Unsupported type '%s'", typ.Identifier))
		}
	default:
		panic(errors.NewUnreachableError())
	}
}

func (c *Compiler) nextLocalIndex(registryType registers.RegistryType) uint16 {
	return c.currentFunction.localCount.NextIndex(registryType)
}
