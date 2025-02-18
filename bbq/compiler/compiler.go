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
	"math"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type Compiler[E any] struct {
	Program             *ast.Program
	ExtendedElaboration *ExtendedElaboration
	Config              *Config
	checker             *sema.Checker

	currentFunction    *function[E]
	compositeTypeStack *Stack[sema.CompositeKindedType]

	functions           []*function[E]
	constants           []*constant
	globals             map[string]*global
	importedGlobals     map[string]*global
	usedImportedGlobals []*global
	loops               []*loop
	currentLoop         *loop
	staticTypes         [][]byte

	// Cache alike for staticTypes and constants in the pool.
	typesInPool     map[sema.TypeID]uint16
	constantsInPool map[constantsCacheKey]*constant

	// TODO: initialize
	memoryGauge common.MemoryGauge

	codeGen CodeGen[E]
}

type constantsCacheKey struct {
	data string
	kind constantkind.ConstantKind
}

var _ ast.DeclarationVisitor[struct{}] = &Compiler[any]{}
var _ ast.StatementVisitor[struct{}] = &Compiler[any]{}
var _ ast.ExpressionVisitor[struct{}] = &Compiler[any]{}

func NewBytecodeCompiler(
	checker *sema.Checker,
) *Compiler[byte] {
	return newCompiler(
		checker,
		&ByteCodeGen{},
	)
}

func NewInstructionCompiler(
	checker *sema.Checker,
) *Compiler[opcode.Instruction] {
	return newCompiler(
		checker,
		&InstructionCodeGen{},
	)
}

func newCompiler[E any](
	checker *sema.Checker,
	codeGen CodeGen[E],
) *Compiler[E] {
	return &Compiler[E]{
		Program:             checker.Program,
		ExtendedElaboration: NewExtendedElaboration(checker.Elaboration),
		Config:              &Config{},
		checker:             checker,
		globals:             make(map[string]*global),
		importedGlobals:     NativeFunctions(),
		typesInPool:         make(map[sema.TypeID]uint16),
		constantsInPool:     make(map[constantsCacheKey]*constant),
		compositeTypeStack: &Stack[sema.CompositeKindedType]{
			elements: make([]sema.CompositeKindedType, 0),
		},
		codeGen: codeGen,
	}
}

func (c *Compiler[E]) WithConfig(config *Config) *Compiler[E] {
	c.Config = config
	return c
}

func (c *Compiler[_]) findGlobal(name string) *global {
	global, ok := c.globals[name]
	if ok {
		return global
	}

	// If failed to find, then try with type-qualified name.
	// This is because contract functions/type-constructors can be accessed without the contract name.
	// e.g: SomeContract.Foo() == Foo(), within `SomeContract`.
	if !c.compositeTypeStack.isEmpty() {
		enclosingContract := c.compositeTypeStack.bottom()
		typeQualifiedName := commons.TypeQualifiedName(enclosingContract.GetIdentifier(), name)
		global, ok = c.globals[typeQualifiedName]
		if ok {
			return global
		}
	}

	importedGlobal, ok := c.importedGlobals[name]
	if !ok {
		panic(errors.NewUnexpectedError("cannot find global declaration '%s'", name))
	}

	// Add the 'importedGlobal' to 'globals' when they are used for the first time.
	// This way, the 'globals' would eventually have only the used imports.
	// This is important since global indexes rely on this.
	//
	// If a global is found in imported globals, that means the index is not set.
	// So set an index and add it to the 'globals'.
	count := len(c.globals)
	if count >= math.MaxUint16 {
		panic(errors.NewUnexpectedError("invalid global declaration '%s'", name))
	}
	importedGlobal.index = uint16(count)
	c.globals[name] = importedGlobal

	// Also add it to the usedImportedGlobals.
	// This is later used to export the imports, which is eventually used by the linker.
	// Linker will link the imports in the same order as they are added here.
	// i.e: same order as their indexes (preceded by globals defined in the current program).
	// e.g: [global1, global2, ... [importedGlobal1, importedGlobal2, ...]].
	// Earlier we already reserved the indexes for the globals defined in the current program.
	// (`reserveGlobalVars`)

	c.usedImportedGlobals = append(c.usedImportedGlobals, importedGlobal)

	return importedGlobal
}

func (c *Compiler[_]) addGlobal(name string) *global {
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

func (c *Compiler[_]) addImportedGlobal(location common.Location, name string) *global {
	// Index is not set here. It is set only if this imported global is used.
	global := &global{
		location: location,
		name:     name,
	}
	c.importedGlobals[name] = global
	return global
}

func (c *Compiler[E]) addFunction(name string, parameterCount uint16) *function[E] {
	isCompositeFunction := !c.compositeTypeStack.isEmpty()

	function := newFunction[E](name, parameterCount, isCompositeFunction)
	c.functions = append(c.functions, function)
	c.currentFunction = function
	c.codeGen.SetTarget(&function.code)
	return function
}

func (c *Compiler[_]) addConstant(kind constantkind.ConstantKind, data []byte) *constant {
	count := len(c.constants)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid constant declaration"))
	}

	// Optimization: Reuse the constant if it is already added to the constant pool.
	cacheKey := constantsCacheKey{
		data: string(data),
		kind: kind,
	}
	if constant, ok := c.constantsInPool[cacheKey]; ok {
		return constant
	}

	constant := &constant{
		index: uint16(count),
		kind:  kind,
		data:  data[:],
	}
	c.constants = append(c.constants, constant)
	c.constantsInPool[cacheKey] = constant
	return constant
}

func (c *Compiler[_]) stringConstLoad(str string) {
	constant := c.addStringConst(str)
	c.codeGen.Emit(opcode.InstructionGetConstant{ConstantIndex: constant.index})
}

func (c *Compiler[_]) addStringConst(str string) *constant {
	return c.addConstant(constantkind.String, []byte(str))
}

func (c *Compiler[_]) emitJump(target int) int {
	if target >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJump{Target: uint16(target)})
	return offset
}

func (c *Compiler[_]) emitUndefinedJump() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJump{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_]) emitUndefinedJumpIfFalse() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfFalse{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_]) emitUndefinedJumpIfNil() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfNil{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_]) patchJump(opcodeOffset int) {
	count := c.codeGen.Offset()
	if count == 0 {
		panic(errors.NewUnreachableError())
	}
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	c.codeGen.PatchJump(opcodeOffset, uint16(count))
}

func (c *Compiler[_]) pushLoop(start int) {
	loop := &loop{
		start: start,
	}
	c.loops = append(c.loops, loop)
	c.currentLoop = loop
}

func (c *Compiler[_]) popLoop() {
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

func (c *Compiler[E]) Compile() *bbq.Program[E] {

	// Desugar the program before compiling.
	desugar := NewDesugar(
		c.memoryGauge,
		c.Config,
		c.Program,
		c.ExtendedElaboration,
		c.checker,
	)
	c.Program = desugar.Run()

	for _, declaration := range c.Program.ImportDeclarations() {
		c.compileDeclaration(declaration)
	}

	contract, _ := c.exportContract()

	compositeDeclarations := c.Program.CompositeDeclarations()
	variableDeclarations := c.Program.VariableDeclarations()
	functionDeclarations := c.Program.FunctionDeclarations()
	interfaceDeclarations := c.Program.InterfaceDeclarations()

	// Reserve globals for functions/types before visiting their implementations.
	c.reserveGlobalVars(
		"",
		variableDeclarations,
		nil,
		functionDeclarations,
		compositeDeclarations,
		interfaceDeclarations,
	)

	// Compile declarations
	for _, declaration := range functionDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range compositeDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range interfaceDeclarations {
		c.compileDeclaration(declaration)
	}

	functions := c.ExportFunctions()
	constants := c.exportConstants()
	types := c.exportTypes()
	imports := c.exportImports()
	variables := c.exportVariables(variableDeclarations)

	return &bbq.Program[E]{
		Functions: functions,
		Constants: constants,
		Types:     types,
		Imports:   imports,
		Contract:  contract,
		Variables: variables,
	}
}

func (c *Compiler[_]) reserveGlobalVars(
	compositeTypeName string,
	variableDecls []*ast.VariableDeclaration,
	specialFunctionDecls []*ast.SpecialFunctionDeclaration,
	functionDecls []*ast.FunctionDeclaration,
	compositeDecls []*ast.CompositeDeclaration,
	interfaceDecls []*ast.InterfaceDeclaration,
) {
	for _, declaration := range variableDecls {
		c.addGlobal(declaration.Identifier.Identifier)
	}

	for _, declaration := range specialFunctionDecls {
		switch declaration.Kind {
		case common.DeclarationKindDestructorLegacy,
			common.DeclarationKindPrepare:
			// Important: All special functions visited within `VisitSpecialFunctionDeclaration`
			// must be also visited here. And must be visited only them. e.g: Don't visit inits.
			funcName := commons.TypeQualifiedName(compositeTypeName, declaration.FunctionDeclaration.Identifier.Identifier)
			c.addGlobal(funcName)
		}
	}

	for _, declaration := range functionDecls {
		funcName := commons.TypeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)
		c.addGlobal(funcName)
	}

	for _, declaration := range compositeDecls {
		qualifiedTypeName := commons.TypeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)

		// Reserve a global-var for the value-constructor.
		c.addGlobal(qualifiedTypeName)

		// For composite types other than contracts, global variables
		// reserved by the type-name will be used for the init method.
		// For contracts, global variables reserved by the type-name
		// will be used for the contract value.
		// Hence, reserve a separate global var for contract inits.
		if declaration.CompositeKind == common.CompositeKindContract {
			c.addGlobal(commons.InitFunctionName)
		}

		// Define globals for functions before visiting function bodies.

		members := declaration.Members

		c.reserveGlobalVars(
			qualifiedTypeName,
			nil,
			members.SpecialFunctions(),
			members.Functions(),
			members.Composites(),
			members.Interfaces(),
		)
	}

	for _, declaration := range interfaceDecls {
		// Don't need a global-var for the value-constructor for interfaces
		qualifiedTypeName := commons.TypeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)

		members := declaration.Members

		c.reserveGlobalVars(
			qualifiedTypeName,
			nil,
			members.SpecialFunctions(),
			members.Functions(),
			members.Composites(),
			members.Interfaces(),
		)
	}
}

func (c *Compiler[_]) exportConstants() []*bbq.Constant {
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

func (c *Compiler[_]) exportTypes() [][]byte {
	return c.staticTypes
}

func (c *Compiler[_]) exportImports() []*bbq.Import {
	exportedImports := make([]*bbq.Import, 0)
	for _, importedGlobal := range c.usedImportedGlobals {
		bbqImport := &bbq.Import{
			Location: importedGlobal.location,
			Name:     importedGlobal.name,
		}
		exportedImports = append(exportedImports, bbqImport)
	}

	return exportedImports
}

func (c *Compiler[E]) ExportFunctions() []*bbq.Function[E] {
	functions := make([]*bbq.Function[E], 0, len(c.functions))
	for _, function := range c.functions {
		functions = append(
			functions,
			&bbq.Function[E]{
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

func (c *Compiler[_]) exportVariables(variableDecls []*ast.VariableDeclaration) []*bbq.Variable {
	variables := make([]*bbq.Variable, 0, len(c.functions))
	for _, varDecl := range variableDecls {
		variables = append(
			variables,
			&bbq.Variable{
				Name: varDecl.Identifier.Identifier,
			},
		)
	}
	return variables
}

func (c *Compiler[_]) contractType() (contractType sema.CompositeKindedType) {
	contractDecl := c.Program.SoleContractDeclaration()
	if contractDecl != nil {
		contractType = c.ExtendedElaboration.CompositeDeclarationType(contractDecl)
		return
	}

	interfaceDecl := c.Program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		contractType = c.ExtendedElaboration.InterfaceDeclarationType(interfaceDecl)
		return
	}

	return nil
}

func (c *Compiler[_]) exportContract() (*bbq.Contract, sema.CompositeKindedType) {
	var location common.Location
	var name string

	contractType := c.contractType()
	if contractType == nil {
		return nil, nil
	}

	_, isInterface := contractType.(*sema.InterfaceType)

	location = contractType.GetLocation()
	name = contractType.GetIdentifier()

	addressLocation := location.(common.AddressLocation)
	return &bbq.Contract{
		Name:        name,
		Address:     addressLocation.Address[:],
		IsInterface: isInterface,
	}, contractType
}

func (c *Compiler[_]) compileDeclaration(declaration ast.Declaration) {
	ast.AcceptDeclaration[struct{}](declaration, c)
}

func (c *Compiler[_]) compileBlock(block *ast.Block) {
	// TODO: scope
	for _, statement := range block.Statements {
		c.compileStatement(statement)
	}
}

func (c *Compiler[_]) compileFunctionBlock(functionBlock *ast.FunctionBlock) {
	// Function conditions must have been desugared to statements.
	// So there shouldn't be any condition at this point.
	if functionBlock != nil {
		c.compileBlock(functionBlock.Block)
	}
}

func (c *Compiler[_]) compileStatement(statement ast.Statement) {
	ast.AcceptStatement[struct{}](statement, c)
}

func (c *Compiler[_]) compileExpression(expression ast.Expression) {
	ast.AcceptExpression[struct{}](expression, c)
}

func (c *Compiler[_]) VisitReturnStatement(statement *ast.ReturnStatement) (_ struct{}) {
	expression := statement.Expression
	if expression != nil {
		// TODO: copy
		c.compileExpression(expression)
		c.codeGen.Emit(opcode.InstructionReturnValue{})
	} else {
		c.codeGen.Emit(opcode.InstructionReturn{})
	}
	return
}

func (c *Compiler[_]) VisitBreakStatement(_ *ast.BreakStatement) (_ struct{}) {
	offset := c.codeGen.Offset()
	c.currentLoop.breaks = append(c.currentLoop.breaks, offset)
	c.emitUndefinedJump()
	return
}

func (c *Compiler[_]) VisitContinueStatement(_ *ast.ContinueStatement) (_ struct{}) {
	c.emitJump(c.currentLoop.start)
	return
}

func (c *Compiler[_]) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {
	// TODO: scope
	var elseJump int
	switch test := statement.Test.(type) {
	case ast.Expression:
		c.compileExpression(test)
		elseJump = c.emitUndefinedJumpIfFalse()

	case *ast.VariableDeclaration:
		// TODO: second value
		c.compileExpression(test.Value)

		tempIndex := c.currentFunction.generateLocalIndex()
		c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: tempIndex})

		c.codeGen.Emit(opcode.InstructionGetLocal{LocalIndex: tempIndex})
		elseJump = c.emitUndefinedJumpIfNil()

		c.codeGen.Emit(opcode.InstructionGetLocal{LocalIndex: tempIndex})
		c.codeGen.Emit(opcode.InstructionUnwrap{})
		varDeclTypes := c.ExtendedElaboration.VariableDeclarationTypes(test)
		c.emitTransfer(varDeclTypes.TargetType)
		local := c.currentFunction.declareLocal(test.Identifier.Identifier)
		c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: local.index})

	default:
		panic(errors.NewUnreachableError())
	}

	c.compileBlock(statement.Then)
	elseBlock := statement.Else
	if elseBlock != nil {
		thenJump := c.emitUndefinedJump()
		c.patchJump(elseJump)
		c.compileBlock(elseBlock)
		c.patchJump(thenJump)
	} else {
		c.patchJump(elseJump)
	}

	return
}

func (c *Compiler[_]) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {
	testOffset := c.codeGen.Offset()
	c.pushLoop(testOffset)
	c.compileExpression(statement.Test)
	endJump := c.emitUndefinedJumpIfFalse()
	c.compileBlock(statement.Block)
	c.emitJump(testOffset)
	c.patchJump(endJump)
	c.popLoop()
	return
}

func (c *Compiler[_]) VisitForStatement(statement *ast.ForStatement) (_ struct{}) {
	index := statement.Index
	var indexLocalVar *local
	if index != nil {
		indexLocalVar = c.currentFunction.declareLocal(index.Identifier)
	}
	elementLocalVar := c.currentFunction.declareLocal(statement.Identifier.Identifier)
	iteratorLocalIndex := c.currentFunction.generateLocalIndex()

	// Store the iterator in a local index
	c.compileExpression(statement.Value)
	c.codeGen.Emit(opcode.InstructionIterator{})
	c.codeGen.Emit(opcode.InstructionSetLocal{
		LocalIndex: iteratorLocalIndex,
	})

	testOffset := c.codeGen.Offset()
	c.pushLoop(testOffset)

	// Loop test: Get the iterator and call `hasNext()`
	c.codeGen.Emit(opcode.InstructionGetLocal{
		LocalIndex: iteratorLocalIndex,
	})
	c.codeGen.Emit(opcode.InstructionIteratorHasNext{})

	endJump := c.emitUndefinedJumpIfFalse()

	// Loop Body.
	// Get the iterator and call `next()`. Store the index (if exist), and element in local var.

	indexNeeded := indexLocalVar != nil

	c.codeGen.Emit(opcode.InstructionGetLocal{
		LocalIndex: iteratorLocalIndex,
	})
	c.codeGen.Emit(opcode.InstructionIteratorNext{
		// TODO: pass a flag to indicate whether the index is needed?
	})

	// Store element
	c.codeGen.Emit(opcode.InstructionSetLocal{
		LocalIndex: elementLocalVar.index,
	})
	// Store index, if needed
	if indexNeeded {
		c.codeGen.Emit(opcode.InstructionSetLocal{
			LocalIndex: indexLocalVar.index,
		})
	}
	// Compile the for-loop body
	c.compileBlock(statement.Block)

	// Jump back to the loop test. i.e: `hasNext()`
	c.emitJump(testOffset)

	c.patchJump(endJump)
	c.popLoop()

	return
}

func (c *Compiler[_]) VisitEmitStatement(statement *ast.EmitStatement) (_ struct{}) {
	c.compileExpression(statement.InvocationExpression)
	eventType := c.ExtendedElaboration.EmitStatementEventType(statement)
	typeIndex := c.getOrAddType(eventType)
	c.codeGen.Emit(opcode.InstructionEmitEvent{
		TypeIndex: typeIndex,
	})

	return
}

func (c *Compiler[_]) VisitSwitchStatement(statement *ast.SwitchStatement) (_ struct{}) {
	c.compileExpression(statement.Expression)
	localIndex := c.currentFunction.generateLocalIndex()
	c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: localIndex})

	endJumps := make([]int, 0, len(statement.Cases))
	previousJump := -1

	for _, switchCase := range statement.Cases {
		if previousJump >= 0 {
			c.patchJump(previousJump)
		}

		isDefault := switchCase.Expression == nil
		if !isDefault {
			c.codeGen.Emit(opcode.InstructionGetLocal{LocalIndex: localIndex})
			c.compileExpression(switchCase.Expression)
			c.codeGen.Emit(opcode.InstructionEqual{})
			previousJump = c.emitUndefinedJumpIfFalse()

		}

		for _, caseStatement := range switchCase.Statements {
			c.compileStatement(caseStatement)
		}

		if !isDefault {
			endJump := c.emitUndefinedJump()
			endJumps = append(endJumps, endJump)
		}
	}

	for _, endJump := range endJumps {
		c.patchJump(endJump)
	}

	return
}

func (c *Compiler[_]) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {
	// TODO: second value

	local := c.currentFunction.declareLocal(declaration.Identifier.Identifier)

	// TODO: This can be nil only for synthetic-result variable
	//   Any better way to handle this?
	if declaration.Value == nil {
		return
	}

	c.compileExpression(declaration.Value)

	varDeclTypes := c.ExtendedElaboration.VariableDeclarationTypes(declaration)
	c.emitTransfer(varDeclTypes.TargetType)

	c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: local.index})
	return
}

func (c *Compiler[_]) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {

	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		c.compileExpression(statement.Value)
		assignmentTypes := c.ExtendedElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		varName := target.Identifier.Identifier
		local := c.currentFunction.findLocal(varName)
		if local != nil {
			c.codeGen.Emit(opcode.InstructionSetLocal{
				LocalIndex: local.index,
			})
			return
		}

		global := c.findGlobal(varName)
		c.codeGen.Emit(opcode.InstructionSetGlobal{
			GlobalIndex: global.index,
		})

	case *ast.MemberExpression:
		c.compileExpression(target.Expression)

		c.compileExpression(statement.Value)
		assignmentTypes := c.ExtendedElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		constant := c.addStringConst(target.Identifier.Identifier)
		c.codeGen.Emit(opcode.InstructionSetField{
			FieldNameIndex: constant.index,
		})

	case *ast.IndexExpression:
		c.compileExpression(target.TargetExpression)
		c.compileExpression(target.IndexingExpression)

		c.compileExpression(statement.Value)
		assignmentTypes := c.ExtendedElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		c.codeGen.Emit(opcode.InstructionSetIndex{})

	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	return
}

func (c *Compiler[_]) VisitSwapStatement(_ *ast.SwapStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitExpressionStatement(statement *ast.ExpressionStatement) (_ struct{}) {
	c.compileExpression(statement.Expression)

	switch statement.Expression.(type) {
	case *ast.DestroyExpression:
		// Do nothing. Destroy operation will not produce any result.
	default:
		// Otherwise, drop the expression evaluation result.
		c.codeGen.Emit(opcode.InstructionDrop{})
	}

	return
}

func (c *Compiler[_]) VisitVoidExpression(_ *ast.VoidExpression) (_ struct{}) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitBoolExpression(expression *ast.BoolExpression) (_ struct{}) {
	if expression.Value {
		c.codeGen.Emit(opcode.InstructionTrue{})
	} else {
		c.codeGen.Emit(opcode.InstructionFalse{})
	}
	return
}

func (c *Compiler[_]) VisitNilExpression(_ *ast.NilExpression) (_ struct{}) {
	c.codeGen.Emit(opcode.InstructionNil{})
	return
}

func (c *Compiler[_]) VisitIntegerExpression(expression *ast.IntegerExpression) (_ struct{}) {
	integerType := c.ExtendedElaboration.IntegerExpressionType(expression)
	constantKind := constantkind.FromSemaType(integerType)

	// TODO:
	var data []byte
	data = leb128.AppendInt64(data, expression.Value.Int64())

	constant := c.addConstant(constantKind, data)
	c.codeGen.Emit(opcode.InstructionGetConstant{ConstantIndex: constant.index})
	return
}

func (c *Compiler[_]) VisitFixedPointExpression(_ *ast.FixedPointExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitArrayExpression(array *ast.ArrayExpression) (_ struct{}) {
	arrayTypes := c.ExtendedElaboration.ArrayExpressionTypes(array)

	typeIndex := c.getOrAddType(arrayTypes.ArrayType)

	size := len(array.Values)
	if size >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid array expression"))
	}

	for _, expression := range array.Values {
		c.compileExpression(expression)
	}

	c.codeGen.Emit(
		opcode.InstructionNewArray{
			TypeIndex:  typeIndex,
			Size:       uint16(size),
			IsResource: arrayTypes.ArrayType.IsResourceType(),
		},
	)

	return
}

func (c *Compiler[_]) VisitDictionaryExpression(dictionary *ast.DictionaryExpression) (_ struct{}) {
	dictionaryTypes := c.ExtendedElaboration.DictionaryExpressionTypes(dictionary)

	typeIndex := c.getOrAddType(dictionaryTypes.DictionaryType)

	size := len(dictionary.Entries)
	if size >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid dictionary expression"))
	}

	for _, entry := range dictionary.Entries {
		c.compileExpression(entry.Key)
		c.compileExpression(entry.Value)
	}

	c.codeGen.Emit(
		opcode.InstructionNewDictionary{
			TypeIndex:  typeIndex,
			Size:       uint16(size),
			IsResource: dictionaryTypes.DictionaryType.IsResourceType(),
		},
	)

	return
}

func (c *Compiler[_]) VisitIdentifierExpression(expression *ast.IdentifierExpression) (_ struct{}) {
	c.emitVariableLoad(expression.Identifier.Identifier)
	return
}

func (c *Compiler[_]) emitVariableLoad(name string) {
	local := c.currentFunction.findLocal(name)
	if local != nil {
		c.codeGen.Emit(opcode.InstructionGetLocal{LocalIndex: local.index})
		return
	}

	global := c.findGlobal(name)
	c.codeGen.Emit(opcode.InstructionGetGlobal{GlobalIndex: global.index})
}

func (c *Compiler[_]) VisitInvocationExpression(expression *ast.InvocationExpression) (_ struct{}) {
	// TODO: copy

	switch invokedExpr := expression.InvokedExpression.(type) {
	case *ast.IdentifierExpression:
		// TODO: Does constructors need any special handling?
		//typ := c.ExtendedElaboration.IdentifierInInvocationType(invokedExpr)
		//invocationType := typ.(*sema.FunctionType)
		//if invocationType.IsConstructor {
		//}

		// Load arguments
		c.loadArguments(expression)
		// Load function value
		c.emitVariableLoad(invokedExpr.Identifier.Identifier)

		typeArgs := c.loadTypeArguments(expression)
		c.codeGen.Emit(opcode.InstructionInvoke{TypeArgs: typeArgs})

	case *ast.MemberExpression:
		memberInfo, ok := c.ExtendedElaboration.MemberExpressionMemberAccessInfo(invokedExpr)
		if !ok {
			// TODO: verify
			panic(errors.NewUnreachableError())
		}

		typeName := TypeName(memberInfo.AccessedType)
		var funcName string

		invocationType := memberInfo.Member.TypeAnnotation.Type.(*sema.FunctionType)
		if invocationType.IsConstructor {
			funcName = commons.TypeQualifiedName(typeName, invokedExpr.Identifier.Identifier)

			// Calling a type constructor must be invoked statically. e.g: `SomeContract.Foo()`.
			// Load arguments
			c.loadArguments(expression)
			// Load function value
			c.emitVariableLoad(funcName)

			typeArgs := c.loadTypeArguments(expression)
			c.codeGen.Emit(opcode.InstructionInvoke{TypeArgs: typeArgs})
			return
		}

		// Receiver is loaded first. So 'self' is always the zero-th argument.
		c.compileExpression(invokedExpr.Expression)
		// Load arguments
		c.loadArguments(expression)

		// Invocations into the interface code, such as default functions and inherited conditions,
		// that were synthetically added at the desugar phase, must be static calls.
		isInterfaceInheritedFuncCall := c.ExtendedElaboration.IsInterfaceMethodStaticCall(expression)

		// Any invocation on restricted-types must be dynamic
		if !isInterfaceInheritedFuncCall && isDynamicMethodInvocation(memberInfo.AccessedType) {
			funcName = invokedExpr.Identifier.Identifier
			if len(funcName) >= math.MaxUint16 {
				panic(errors.NewDefaultUserError("invalid function name"))
			}

			typeArgs := c.loadTypeArguments(expression)

			argumentCount := len(expression.Arguments)
			if argumentCount >= math.MaxUint16 {
				panic(errors.NewDefaultUserError("invalid number of arguments"))
			}

			funcNameConst := c.addStringConst(funcName)
			c.codeGen.Emit(
				opcode.InstructionInvokeDynamic{
					NameIndex: funcNameConst.index,
					TypeArgs:  typeArgs,
					ArgCount:  uint16(argumentCount),
				},
			)

		} else {
			// Load function value
			funcName = commons.TypeQualifiedName(typeName, invokedExpr.Identifier.Identifier)
			c.emitVariableLoad(funcName)

			typeArgs := c.loadTypeArguments(expression)
			c.codeGen.Emit(opcode.InstructionInvoke{TypeArgs: typeArgs})
		}
	default:
		panic(errors.NewUnreachableError())
	}

	return
}

func isDynamicMethodInvocation(accessedType sema.Type) bool {
	switch typ := accessedType.(type) {
	case *sema.ReferenceType:
		return isDynamicMethodInvocation(typ.Type)
	case *sema.IntersectionType:
		return true

		// TODO: Optional type?

	case *sema.InterfaceType:
		return true
	default:
		return false
	}
}

func TypeName(typ sema.Type) string {
	switch typ := typ.(type) {
	case *sema.ReferenceType:
		return TypeName(typ.Type)
	case *sema.IntersectionType:
		// TODO: Revisit. Probably this is not needed here?
		return TypeName(typ.Types[0])
	case *sema.CapabilityType:
		return interpreter.PrimitiveStaticTypeCapability.String()
	default:
		return typ.QualifiedString()
	}
}

func (c *Compiler[_]) loadArguments(expression *ast.InvocationExpression) {
	invocationTypes := c.ExtendedElaboration.InvocationExpressionTypes(expression)
	for index, argument := range expression.Arguments {
		c.compileExpression(argument.Expression)
		c.emitTransfer(invocationTypes.ArgumentTypes[index])
	}

	// TODO: Is this needed?
	//// Load empty values for optional parameters, if they are not provided.
	//for i := len(expression.Arguments); i < invocationTypes.ParamCount; i++ {
	//	c.emit(opcode.Empty)
	//}
}

func (c *Compiler[_]) loadTypeArguments(expression *ast.InvocationExpression) []uint16 {
	invocationTypes := c.ExtendedElaboration.InvocationExpressionTypes(expression)

	typeArgsCount := invocationTypes.TypeArguments.Len()
	if typeArgsCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid number of type arguments: %d", typeArgsCount))
	}

	if typeArgsCount == 0 {
		return nil
	}

	typeArgs := make([]uint16, 0, typeArgsCount)

	invocationTypes.TypeArguments.Foreach(func(key *sema.TypeParameter, typeParam sema.Type) {
		typeArgs = append(typeArgs, c.getOrAddType(typeParam))
	})

	return typeArgs
}

func (c *Compiler[_]) VisitMemberExpression(expression *ast.MemberExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	constant := c.addStringConst(expression.Identifier.Identifier)
	c.codeGen.Emit(opcode.InstructionGetField{
		FieldNameIndex: constant.index,
	})
	return
}

func (c *Compiler[_]) VisitIndexExpression(expression *ast.IndexExpression) (_ struct{}) {
	c.compileExpression(expression.TargetExpression)
	c.compileExpression(expression.IndexingExpression)
	c.codeGen.Emit(opcode.InstructionGetIndex{})
	return
}

func (c *Compiler[_]) VisitConditionalExpression(_ *ast.ConditionalExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitUnaryExpression(expression *ast.UnaryExpression) (_ struct{}) {
	switch expression.Operation {
	case ast.OperationMove:
		c.compileExpression(expression.Expression)
	case ast.OperationNegate:
		c.compileExpression(expression.Expression)
		c.codeGen.Emit(opcode.InstructionNot{})
	default:
		// TODO
		panic(errors.NewUnreachableError())
	}

	return
}

func (c *Compiler[_]) VisitBinaryExpression(expression *ast.BinaryExpression) (_ struct{}) {
	c.compileExpression(expression.Left)
	// TODO: add support for other types

	switch expression.Operation {
	case ast.OperationNilCoalesce:
		// create a duplicate to perform the equal check.
		// So if the condition succeeds, then the condition's result will be at the top of the stack.
		c.codeGen.Emit(opcode.InstructionDup{})

		c.codeGen.Emit(opcode.InstructionNil{})
		c.codeGen.Emit(opcode.InstructionEqual{})
		elseJump := c.emitUndefinedJumpIfFalse()

		// Drop the duplicated condition result.
		// It is not needed for the 'then' path.
		c.codeGen.Emit(opcode.InstructionDrop{})

		c.compileExpression(expression.Right)

		thenJump := c.emitUndefinedJump()
		c.patchJump(elseJump)
		c.codeGen.Emit(opcode.InstructionUnwrap{})
		c.patchJump(thenJump)
	default:
		c.compileExpression(expression.Right)

		switch expression.Operation {
		case ast.OperationPlus:
			c.codeGen.Emit(opcode.InstructionAdd{})
		case ast.OperationMinus:
			c.codeGen.Emit(opcode.InstructionSubtract{})
		case ast.OperationMul:
			c.codeGen.Emit(opcode.InstructionMultiply{})
		case ast.OperationDiv:
			c.codeGen.Emit(opcode.InstructionDivide{})
		case ast.OperationMod:
			c.codeGen.Emit(opcode.InstructionMod{})
		case ast.OperationEqual:
			c.codeGen.Emit(opcode.InstructionEqual{})
		case ast.OperationNotEqual:
			c.codeGen.Emit(opcode.InstructionNotEqual{})
		case ast.OperationLess:
			c.codeGen.Emit(opcode.InstructionLess{})
		case ast.OperationLessEqual:
			c.codeGen.Emit(opcode.InstructionLessOrEqual{})
		case ast.OperationGreater:
			c.codeGen.Emit(opcode.InstructionGreater{})
		case ast.OperationGreaterEqual:
			c.codeGen.Emit(opcode.InstructionGreaterOrEqual{})
		default:
			panic(errors.NewUnreachableError())
		}
	}

	return
}

func (c *Compiler[_]) VisitFunctionExpression(_ *ast.FunctionExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitStringExpression(expression *ast.StringExpression) (_ struct{}) {
	c.stringConstLoad(expression.Value)
	return
}

func (c *Compiler[_]) VisitStringTemplateExpression(_ *ast.StringTemplateExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitCastingExpression(expression *ast.CastingExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)

	castingTypes := c.ExtendedElaboration.CastingExpressionTypes(expression)
	index := c.getOrAddType(castingTypes.TargetType)

	var castInstruction opcode.Instruction
	switch expression.Operation {
	case ast.OperationCast:
		castInstruction = opcode.InstructionSimpleCast{
			TypeIndex: index,
		}
	case ast.OperationFailableCast:
		castInstruction = opcode.InstructionFailableCast{
			TypeIndex: index,
		}
	case ast.OperationForceCast:
		castInstruction = opcode.InstructionForceCast{
			TypeIndex: index,
		}
	default:
		panic(errors.NewUnreachableError())
	}

	c.codeGen.Emit(castInstruction)
	return
}

func (c *Compiler[_]) VisitCreateExpression(expression *ast.CreateExpression) (_ struct{}) {
	c.compileExpression(expression.InvocationExpression)
	return
}

func (c *Compiler[_]) VisitDestroyExpression(expression *ast.DestroyExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.codeGen.Emit(opcode.InstructionDestroy{})
	return
}

func (c *Compiler[_]) VisitReferenceExpression(expression *ast.ReferenceExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	borrowType := c.ExtendedElaboration.ReferenceExpressionBorrowType(expression)
	index := c.getOrAddType(borrowType)
	c.codeGen.Emit(opcode.InstructionNewRef{TypeIndex: index})
	return
}

func (c *Compiler[_]) VisitForceExpression(_ *ast.ForceExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitPathExpression(expression *ast.PathExpression) (_ struct{}) {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)
	identifier := expression.Identifier.Identifier
	if len(identifier) >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid identifier"))
	}

	identifierConst := c.addStringConst(identifier)

	c.codeGen.Emit(
		opcode.InstructionPath{
			Domain:          domain,
			IdentifierIndex: identifierConst.index,
		},
	)
	return
}

func (c *Compiler[_]) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) (_ struct{}) {
	kind := declaration.DeclarationKind()
	switch kind {
	case common.DeclarationKindInitializer:
		c.compileInitializer(declaration)
	case common.DeclarationKindDestructorLegacy, common.DeclarationKindPrepare:
		c.compileDeclaration(declaration.FunctionDeclaration)
	default:
		// TODO: support other special functions
		panic(errors.NewUnreachableError())
	}
	return
}

func (c *Compiler[_]) compileInitializer(declaration *ast.SpecialFunctionDeclaration) {
	enclosingCompositeTypeName := c.enclosingCompositeTypeFullyQualifiedName()
	enclosingType := c.compositeTypeStack.top()
	kind := enclosingType.GetCompositeKind()

	var functionName string
	if kind == common.CompositeKindContract {
		// For contracts, add the initializer as `init()`.
		// A global variable with the same name as contract is separately added.
		// The VM will load the contract and assign to that global variable during imports resolution.
		functionName = declaration.DeclarationIdentifier().Identifier
	} else {
		// Use the type name as the function name for initializer.
		// So `x = Foo()` would directly call the init method.
		functionName = enclosingCompositeTypeName
	}

	parameterCount := 0
	parameterList := declaration.FunctionDeclaration.ParameterList
	if parameterList != nil {
		parameterCount = len(parameterList.Parameters)
	}

	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	function := c.addFunction(functionName, uint16(parameterCount))
	c.declareParameters(function, parameterList, false)

	// Declare `self`
	self := c.currentFunction.declareLocal(sema.SelfIdentifier)

	// Initialize an empty struct and assign to `self`.
	// i.e: `self = New()`

	// Write composite kind
	// TODO: Maybe get/include this from static-type. Then no need to provide separately.

	typeIndex := c.getOrAddType(enclosingType)

	c.codeGen.Emit(
		opcode.InstructionNew{
			Kind:      kind,
			TypeIndex: typeIndex,
		},
	)

	if kind == common.CompositeKindContract {
		// During contract init, update the global variable with the newly initialized contract value.
		// So accessing the contract through the global variable while initializing itself, would work.
		// i.e:
		// contract Foo {
		//     init() {
		//        Foo.something()  // <-- accessing `Foo` while initializing `Foo`
		//     }
		// }

		// Duplicate the top of stack and store it in both global variable and in `self`
		c.codeGen.Emit(opcode.InstructionDup{})
		global := c.findGlobal(enclosingCompositeTypeName)

		c.codeGen.Emit(opcode.InstructionSetGlobal{GlobalIndex: global.index})
	}

	c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: self.index})

	// emit for the statements in `init()` body.
	c.compileFunctionBlock(declaration.FunctionDeclaration.FunctionBlock)

	// Constructor should return the created the struct. i.e: return `self`
	c.codeGen.Emit(opcode.InstructionGetLocal{LocalIndex: self.index})
	c.codeGen.Emit(opcode.InstructionReturnValue{})
}

func (c *Compiler[_]) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) (_ struct{}) {
	declareReceiver := !c.compositeTypeStack.isEmpty()
	function := c.declareFunction(declaration, declareReceiver)

	c.declareParameters(function, declaration.ParameterList, declareReceiver)
	c.compileFunctionBlock(declaration.FunctionBlock)

	return
}

func (c *Compiler[E]) declareFunction(declaration *ast.FunctionDeclaration, declareReceiver bool) *function[E] {
	enclosingCompositeTypeName := c.enclosingCompositeTypeFullyQualifiedName()
	functionName := commons.TypeQualifiedName(enclosingCompositeTypeName, declaration.Identifier.Identifier)

	parameterCount := 0

	paramList := declaration.ParameterList
	if paramList != nil {
		parameterCount = len(paramList.Parameters)
	}

	if parameterCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	if declareReceiver {
		parameterCount++
	}

	return c.addFunction(functionName, uint16(parameterCount))
}

func (c *Compiler[_]) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	compositeType := c.ExtendedElaboration.CompositeDeclarationType(declaration)
	c.compositeTypeStack.push(compositeType)
	defer func() {
		c.compositeTypeStack.pop()
	}()

	// Compile members
	hasInit := false
	for _, specialFunc := range declaration.Members.SpecialFunctions() {
		if specialFunc.Kind == common.DeclarationKindInitializer {
			hasInit = true
		}
		c.compileDeclaration(specialFunc)
	}

	// If the initializer is not declared, generate an empty initializer.
	if !hasInit {
		c.generateEmptyInit()
	}

	for _, function := range declaration.Members.Functions() {
		c.compileDeclaration(function)
	}
	for _, nestedTypes := range declaration.Members.Interfaces() {
		c.compileDeclaration(nestedTypes)
	}
	for _, nestedTypes := range declaration.Members.Composites() {
		c.compileDeclaration(nestedTypes)
	}

	// TODO:

	return
}

func (c *Compiler[_]) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) (_ struct{}) {
	interfaceType := c.ExtendedElaboration.InterfaceDeclarationType(declaration)
	c.compositeTypeStack.push(interfaceType)
	defer func() {
		c.compositeTypeStack.pop()
	}()

	for _, function := range declaration.Members.Functions() {
		c.compileDeclaration(function)
	}
	for _, nestedTypes := range declaration.Members.Interfaces() {
		c.compileDeclaration(nestedTypes)
	}
	for _, nestedTypes := range declaration.Members.Composites() {
		c.compileDeclaration(nestedTypes)
	}
	return
}

func (c *Compiler[_]) VisitFieldDeclaration(_ *ast.FieldDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitImportDeclaration(declaration *ast.ImportDeclaration) (_ struct{}) {
	resolvedLocations, err := commons.ResolveLocation(
		c.Config.LocationHandler,
		declaration.Identifiers,
		declaration.Location,
	)
	if err != nil {
		panic(err)
	}

	for _, location := range resolvedLocations {
		importedProgram := c.Config.ImportHandler(location.Location)

		// Add a global variable for the imported contract value.
		contractDecl := importedProgram.Contract
		isContract := contractDecl != nil
		if isContract && !contractDecl.IsInterface {
			c.addImportedGlobal(location.Location, contractDecl.Name)
		}

		for _, function := range importedProgram.Functions {
			name := function.Name

			// Skip the contract initializer.
			// It should never be able to invoked within the code.
			if isContract && name == commons.InitFunctionName {
				continue
			}

			// TODO: Filter-in only public functions
			c.addImportedGlobal(location.Location, function.Name)
		}
	}

	return
}

func (c *Compiler[_]) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitAttachmentDeclaration(_ *ast.AttachmentDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitEntitlementDeclaration(_ *ast.EntitlementDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitEntitlementMappingDeclaration(_ *ast.EntitlementMappingDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitRemoveStatement(_ *ast.RemoveStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitAttachExpression(_ *ast.AttachExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) patchLoop(l *loop) {
	for _, breakOffset := range l.breaks {
		c.patchJump(breakOffset)
	}
}

func (c *Compiler[_]) emitTransfer(targetType sema.Type) {
	index := c.getOrAddType(targetType)

	c.codeGen.Emit(opcode.InstructionTransfer{TypeIndex: index})
}

func (c *Compiler[_]) getOrAddType(targetType sema.Type) uint16 {
	// Optimization: Re-use types in the pool.
	index, ok := c.typesInPool[targetType.ID()]
	if !ok {
		staticType := interpreter.ConvertSemaToStaticType(c.memoryGauge, targetType)
		bytes, err := interpreter.StaticTypeToBytes(staticType)
		if err != nil {
			panic(err)
		}
		index = c.addType(bytes)
		c.typesInPool[targetType.ID()] = index
	}
	return index
}

func (c *Compiler[_]) addType(data []byte) uint16 {
	count := len(c.staticTypes)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid type declaration"))
	}

	c.staticTypes = append(c.staticTypes, data)
	return uint16(count)
}

func (c *Compiler[_]) enclosingCompositeTypeFullyQualifiedName() string {
	if c.compositeTypeStack.isEmpty() {
		return ""
	}

	var sb strings.Builder
	for i, typ := range c.compositeTypeStack.elements {
		if i > 0 {
			sb.WriteRune('.')
		}
		sb.WriteString(typ.GetIdentifier())
	}

	return sb.String()
}

func (c *Compiler[E]) declareParameters(function *function[E], paramList *ast.ParameterList, declareReceiver bool) {
	if declareReceiver {
		// Declare receiver as `self`.
		// Receiver is always at the zero-th index of params.
		function.declareLocal(sema.SelfIdentifier)
	}

	if paramList != nil {
		for _, parameter := range paramList.Parameters {
			parameterName := parameter.Identifier.Identifier
			function.declareLocal(parameterName)
		}
	}
}

func (c *Compiler[_]) generateEmptyInit() {
	c.VisitSpecialFunctionDeclaration(emptyInitializer)
}
