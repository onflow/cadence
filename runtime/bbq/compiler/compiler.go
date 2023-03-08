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
	"bytes"
	"math"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type Compiler struct {
	Program     *ast.Program
	Elaboration *sema.Elaboration
	Config      *Config

	currentFunction    *function
	compositeTypeStack *Stack[*sema.CompositeType]

	functions   []*function
	constants   []*constant
	globals     map[string]*global
	loops       []*loop
	currentLoop *loop
	staticTypes [][]byte
	typesInPool map[sema.Type]uint16

	// TODO: initialize
	memoryGauge common.MemoryGauge
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
		Config:      &Config{},
		globals:     map[string]*global{},
		typesInPool: map[sema.Type]uint16{},
		compositeTypeStack: &Stack[*sema.CompositeType]{
			elements: make([]*sema.CompositeType, 0),
		},
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
	isCompositeFunction := !c.compositeTypeStack.isEmpty()

	function := newFunction(name, parameterCount, isCompositeFunction)
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

	for _, declaration := range c.Program.ImportDeclarations() {
		c.compileDeclaration(declaration)
	}

	// Reserve globals for functions/types before visiting their implementations.
	c.reserveGlobalVars(
		"",
		c.Program.FunctionDeclarations(),
		c.Program.CompositeDeclarations(),
	)

	// Compile declarations
	for _, declaration := range c.Program.FunctionDeclarations() {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range c.Program.CompositeDeclarations() {
		c.compileDeclaration(declaration)
	}

	functions := c.exportFunctions()
	constants := c.exportConstants()
	types := c.exportTypes()
	imports := c.exportImports()
	contract := c.exportContract()

	return &bbq.Program{
		Functions: functions,
		Constants: constants,
		Types:     types,
		Imports:   imports,
		Contract:  contract,
	}
}

func (c *Compiler) reserveGlobalVars(
	compositeTypeName string,
	funcDecls []*ast.FunctionDeclaration,
	compositeDecls []*ast.CompositeDeclaration,
) {
	for _, declaration := range funcDecls {
		funcName := typeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)
		c.addGlobal(funcName)
	}

	for _, declaration := range compositeDecls {
		// TODO: Handle nested composite types. Those name should be `Foo.Bar`.
		qualifiedTypeName := typeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)

		c.addGlobal(qualifiedTypeName)

		// For composite type other than contracts, globals variable
		// reserved by the type-name will be used for the init method.
		// For contracts, globals variable reserved by the type-name
		// will be used for the contract value.
		// Hence, reserve a separate global var for contract inits.
		if declaration.CompositeKind == common.CompositeKindContract {
			c.addGlobal("init")
		}

		// Define globals for functions before visiting function bodies
		c.reserveGlobalVars(
			qualifiedTypeName,
			declaration.Members.Functions(),
			declaration.Members.Composites(),
		)
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

func (c *Compiler) exportTypes() [][]byte {
	types := make([][]byte, len(c.staticTypes))
	for index, typeBytes := range c.staticTypes {
		types[index] = typeBytes
	}
	return types
}

func (c *Compiler) exportImports() []common.Location {
	imports := c.Program.ImportDeclarations()
	exportedImports := make([]common.Location, len(imports))
	for index, importDecl := range imports {
		exportedImports[index] = importDecl.Location
	}

	return exportedImports
}

func (c *Compiler) exportFunctions() []*bbq.Function {
	functions := make([]*bbq.Function, 0, len(c.functions))
	for _, function := range c.functions {
		functions = append(
			functions,
			&bbq.Function{
				Name:                function.name,
				Code:                function.code,
				LocalCount:          function.localCount,
				ParameterCount:      function.parameterCount,
				IsCompositeFunction: function.isCompositeFunction,
			},
		)
	}
	return functions
}

func (c *Compiler) exportContract() *bbq.Contract {
	contractDecl := c.Program.SoleContractDeclaration()
	if contractDecl == nil {
		return nil
	}

	contractType := c.Elaboration.CompositeDeclarationTypes[contractDecl]
	addressLocation := contractType.Location.(common.AddressLocation)
	return &bbq.Contract{
		Name:    addressLocation.Name,
		Address: addressLocation.Address[:],
	}
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

	assignmentTypes := c.Elaboration.AssignmentStatementTypes[statement]
	c.emitCheckType(assignmentTypes.TargetType)

	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		local := c.currentFunction.findLocal(target.Identifier.Identifier)
		first, second := encodeUint16(local.index)
		c.emit(opcode.SetLocal, first, second)
	case *ast.MemberExpression:
		c.compileExpression(target.Expression)
		c.stringConstLoad(target.Identifier.Identifier)
		c.emit(opcode.SetField)
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
	// Drop the expression evaluation result
	c.emit(opcode.Drop)
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
	c.emitVariableLoad(expression.Identifier.Identifier)
	return
}

func (c *Compiler) emitVariableLoad(name string) {
	local := c.currentFunction.findLocal(name)
	if local != nil {
		first, second := encodeUint16(local.index)
		c.emit(opcode.GetLocal, first, second)
		return
	}

	global := c.findGlobal(name)
	first, second := encodeUint16(global.index)
	c.emit(opcode.GetGlobal, first, second)
}

func (c *Compiler) VisitInvocationExpression(expression *ast.InvocationExpression) (_ struct{}) {
	// TODO: copy

	switch invokedExpr := expression.InvokedExpression.(type) {
	case *ast.IdentifierExpression:
		// Load arguments
		c.loadArguments(expression)
		// Load function value
		c.emitVariableLoad(invokedExpr.Identifier.Identifier)
		c.emit(opcode.InvokeStatic)
	case *ast.MemberExpression:
		memberInfo := c.Elaboration.MemberExpressionMemberInfos[invokedExpr]
		typeName := memberInfo.AccessedType.QualifiedString()
		funcName := typeQualifiedName(typeName, invokedExpr.Identifier.Identifier)

		invocationType := memberInfo.Member.TypeAnnotation.Type.(*sema.FunctionType)
		if invocationType.IsConstructor {
			// Calling a type constructor must be invoked statically. e.g: `SomeContract.Foo()`.
			// Load arguments
			c.loadArguments(expression)
			// Load function value
			c.emitVariableLoad(funcName)
			c.emit(opcode.InvokeStatic)
		} else {
			// Receiver is loaded first. So 'self' is always the zero-th argument.
			// This must be in sync with `compileCompositeFunction`.
			c.compileExpression(invokedExpr.Expression)
			// Load arguments
			c.loadArguments(expression)
			// Load function value
			c.emitVariableLoad(funcName)
			c.emit(opcode.Invoke)
		}
	default:
		panic(errors.NewUnreachableError())
	}

	return
}

func (c *Compiler) loadArguments(expression *ast.InvocationExpression) {
	invocationTypes := c.Elaboration.InvocationExpressionTypes[expression]
	for index, argument := range expression.Arguments {
		c.compileExpression(argument.Expression)
		c.emitCheckType(invocationTypes.ArgumentTypes[index])
	}
}

func (c *Compiler) VisitMemberExpression(expression *ast.MemberExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.stringConstLoad(expression.Identifier.Identifier)
	c.emit(opcode.GetField)
	return
}

func (c *Compiler) VisitIndexExpression(_ *ast.IndexExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitConditionalExpression(_ *ast.ConditionalExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler) VisitUnaryExpression(expression *ast.UnaryExpression) (_ struct{}) {
	switch expression.Operation {
	case ast.OperationMove:
		c.compileExpression(expression.Expression)
	default:
		// TODO
		panic(errors.NewUnreachableError())
	}

	return
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

func (c *Compiler) VisitCreateExpression(expression *ast.CreateExpression) (_ struct{}) {
	c.compileExpression(expression.InvocationExpression)
	return
}

func (c *Compiler) VisitDestroyExpression(expression *ast.DestroyExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.emit(opcode.Destroy)
	return
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
	kind := declaration.DeclarationKind()
	switch kind {
	case common.DeclarationKindInitializer:
		c.compileInitializer(declaration)
	default:
		// TODO: support other special functions
		panic(errors.NewUnreachableError())
	}
	return
}

func (c *Compiler) compileInitializer(declaration *ast.SpecialFunctionDeclaration) {
	enclosingCompositeTypeName := c.enclosingCompositeTypeFullyQualifiedName()
	enclosingType := c.compositeTypeStack.top()

	var functionName string
	if enclosingType.Kind == common.CompositeKindContract {
		// For contracts, add the initializer as `init()`.
		// A global variable with the same name as contract is separately added.
		// The VM will load the contract and assign to that global variable during imports resolution.
		functionName = declaration.DeclarationIdentifier().Identifier
	} else {
		// Use the type name as the function name for initializer.
		// So `x = Foo()` would directly call the init method.
		functionName = enclosingCompositeTypeName
	}

	parameters := declaration.FunctionDeclaration.ParameterList.Parameters
	parameterCount := len(parameters)
	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	function := c.addFunction(functionName, uint16(parameterCount))
	declareParameters(function, parameters)

	// Declare `self`
	self := c.currentFunction.declareLocal(sema.SelfIdentifier)
	selfFirst, selfSecond := encodeUint16(self.index)

	// Initialize an empty struct and assign to `self`.
	// i.e: `self = New()`

	enclosingCompositeType := c.compositeTypeStack.top()
	location := enclosingCompositeType.Location
	locationBytes, err := locationToBytes(location)
	if err != nil {
		panic(err)
	}

	byteSize := 2 + // two bytes for composite kind
		2 + // 2 bytes for location size
		len(locationBytes) + // location
		2 + // 2 bytes for type name size
		len(enclosingCompositeTypeName) // type name

	args := make([]byte, 0, byteSize)

	// Write composite kind
	kindFirst, kindSecond := encodeUint16(uint16(enclosingCompositeType.Kind))
	args = append(args, kindFirst, kindSecond)

	// Write location
	locationSizeFirst, locationSizeSecond := encodeUint16(uint16(len(locationBytes)))
	args = append(args, locationSizeFirst, locationSizeSecond)
	args = append(args, locationBytes...)

	// Write composite name
	typeNameSizeFirst, typeNameSizeSecond := encodeUint16(uint16(len(enclosingCompositeTypeName)))
	args = append(args, typeNameSizeFirst, typeNameSizeSecond)
	args = append(args, enclosingCompositeTypeName...)

	c.emit(opcode.New, args...)

	if enclosingType.Kind == common.CompositeKindContract {
		// During contract init, update the global variable with the newly initialized contract value.
		// So accessing the contract through the global variable while initializing itself, would work.
		// i.e:
		// contract Foo {
		//     init() {
		//        Foo.something()  // <-- accessing `Foo` while initializing `Foo`
		//     }
		// }

		// Duplicate the top of stack and store it in both global variable and in `self`
		c.emit(opcode.Dup)
		global := c.findGlobal(enclosingCompositeTypeName)
		first, second := encodeUint16(global.index)
		c.emit(opcode.SetGlobal, first, second)
	}

	c.emit(opcode.SetLocal, selfFirst, selfSecond)

	// Emit for the statements in `init()` body.
	c.compileFunctionBlock(declaration.FunctionDeclaration.FunctionBlock)

	// Constructor should return the created the struct. i.e: return `self`
	c.emit(opcode.GetLocal, selfFirst, selfSecond)
	c.emit(opcode.ReturnValue)
}

func (c *Compiler) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) (_ struct{}) {
	// TODO: handle nested functions
	function := c.declareFunction(declaration)
	declareParameters(function, declaration.ParameterList.Parameters)
	c.compileFunctionBlock(declaration.FunctionBlock)
	return
}

func (c *Compiler) declareFunction(declaration *ast.FunctionDeclaration) *function {
	enclosingCompositeTypeName := c.enclosingCompositeTypeFullyQualifiedName()
	functionName := typeQualifiedName(enclosingCompositeTypeName, declaration.Identifier.Identifier)

	functionType := c.Elaboration.FunctionDeclarationFunctionTypes[declaration]
	parameterCount := len(functionType.Parameters)
	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	return c.addFunction(functionName, uint16(parameterCount))
}

func (c *Compiler) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	enclosingCompositeType := c.Elaboration.CompositeDeclarationTypes[declaration]
	c.compositeTypeStack.push(enclosingCompositeType)
	defer func() {
		c.compositeTypeStack.pop()
	}()

	// Compile members
	for _, specialFunc := range declaration.Members.SpecialFunctions() {
		c.compileDeclaration(specialFunc)
	}

	for _, function := range declaration.Members.Functions() {
		c.compileCompositeFunction(function)
	}

	for _, nestedTypes := range declaration.Members.Composites() {
		c.compileDeclaration(nestedTypes)
	}

	// TODO:

	return
}

func (c *Compiler) compileCompositeFunction(declaration *ast.FunctionDeclaration) {
	function := c.declareFunction(declaration)
	// Declare `self`. Receiver is always at the zero-th index of params.
	function.declareLocal(sema.SelfIdentifier)
	declareParameters(function, declaration.ParameterList.Parameters)
	c.compileFunctionBlock(declaration.FunctionBlock)
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

func (c *Compiler) VisitImportDeclaration(declaration *ast.ImportDeclaration) (_ struct{}) {
	importedProgram := c.Config.ImportHandler(declaration.Location)

	// Add a global variable for the imported contract value.
	contractDecl := importedProgram.Contract
	if contractDecl != nil {
		c.addGlobal(contractDecl.Name)
	}

	for _, function := range importedProgram.Functions {
		// TODO: Filter-in only public functions
		c.addGlobal(function.Name)
	}

	return
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

func (c *Compiler) emitCheckType(targetType sema.Type) {
	// Optimization: Re-use types in the pool.
	index, ok := c.typesInPool[targetType]
	if !ok {
		staticType := interpreter.ConvertSemaToStaticType(c.memoryGauge, targetType)
		bytes, err := interpreter.StaticTypeToBytes(staticType)
		if err != nil {
			panic(err)
		}
		index = c.addType(bytes)
		c.typesInPool[targetType] = index
	}

	first, second := encodeUint16(index)
	c.emit(opcode.CheckType, first, second)
}

func (c *Compiler) addType(data []byte) uint16 {
	count := len(c.staticTypes)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid type declaration"))
	}

	c.staticTypes = append(c.staticTypes, data)
	return uint16(count)
}

func (c *Compiler) enclosingCompositeTypeFullyQualifiedName() string {
	if c.compositeTypeStack.isEmpty() {
		return ""
	}

	var sb strings.Builder
	for i, typ := range c.compositeTypeStack.elements {
		if i > 0 {
			sb.WriteRune('.')
		}
		sb.WriteString(typ.Identifier)
	}

	return sb.String()
}

func typeQualifiedName(typeName, functionName string) string {
	if typeName == "" {
		return functionName
	}

	return typeName + "." + functionName
}

func declareParameters(function *function, parameters []*ast.Parameter) {
	for _, parameter := range parameters {
		parameterName := parameter.Identifier.Identifier
		function.declareLocal(parameterName)
	}
}

func locationToBytes(location common.Location) ([]byte, error) {
	var buf bytes.Buffer
	enc := interpreter.CBOREncMode.NewStreamEncoder(&buf)

	err := interpreter.EncodeLocation(enc, location)
	if err != nil {
		return nil, err
	}

	err = enc.Flush()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
