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
	Program     *ast.Program
	Elaboration *sema.Elaboration
	Config      *Config

	currentFunction    *function[E]
	compositeTypeStack *Stack[*sema.CompositeType]

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
	program *ast.Program,
	elaboration *sema.Elaboration,
) *Compiler[byte] {
	return newCompiler(
		program,
		elaboration,
		&ByteCodeGen{},
	)
}

func NewInstructionCompiler(
	program *ast.Program,
	elaboration *sema.Elaboration,
) *Compiler[opcode.Instruction] {
	return newCompiler(
		program,
		elaboration,
		&InstructionCodeGen{},
	)
}

func newCompiler[E any](
	program *ast.Program,
	elaboration *sema.Elaboration,
	codeGen CodeGen[E],
) *Compiler[E] {
	return &Compiler[E]{
		Program:         program,
		Elaboration:     elaboration,
		Config:          &Config{},
		globals:         make(map[string]*global),
		importedGlobals: NativeFunctions(),
		typesInPool:     make(map[sema.TypeID]uint16),
		constantsInPool: make(map[constantsCacheKey]*constant),
		compositeTypeStack: &Stack[*sema.CompositeType]{
			elements: make([]*sema.CompositeType, 0),
		},
		codeGen: codeGen,
	}
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
		typeQualifiedName := commons.TypeQualifiedName(enclosingContract.Identifier, name)
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

func (c *Compiler[_]) emitJumpIfFalse(target uint16) int {
	if target >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfFalse{Target: target})
	return offset
}

func (c *Compiler[_]) emitUndefinedJumpIfFalse() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfFalse{Target: math.MaxUint16})
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
		c.Elaboration,
	)
	c.Program = desugar.Run()

	for _, declaration := range c.Program.ImportDeclarations() {
		c.compileDeclaration(declaration)
	}

	contract, _ := c.exportContract()
	if contract != nil && contract.IsInterface {
		return &bbq.Program[E]{
			Contract: contract,
		}
	}

	compositeDeclarations := c.Program.CompositeDeclarations()
	variableDeclarations := c.Program.VariableDeclarations()
	functionDeclarations := c.Program.FunctionDeclarations()

	// Reserve globals for functions/types before visiting their implementations.
	c.reserveGlobalVars(
		"",
		variableDeclarations,
		nil,
		functionDeclarations,
		compositeDeclarations,
	)

	// Compile declarations
	for _, declaration := range functionDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range compositeDeclarations {
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
		// TODO: Handle nested composite types. Those name should be `Foo.Bar`.
		qualifiedTypeName := commons.TypeQualifiedName(compositeTypeName, declaration.Identifier.Identifier)

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
		c.reserveGlobalVars(
			qualifiedTypeName,
			nil,
			declaration.Members.SpecialFunctions(),
			declaration.Members.Functions(),
			declaration.Members.Composites(),
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
		contractType = c.Elaboration.CompositeDeclarationType(contractDecl)
		return
	}

	interfaceDecl := c.Program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		contractType = c.Elaboration.InterfaceDeclarationType(interfaceDecl)
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
	if functionBlock.HasConditions() {
		panic(errors.NewUnreachableError())
	}

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
	switch test := statement.Test.(type) {
	case ast.Expression:
		c.compileExpression(test)
	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	elseJump := c.emitUndefinedJumpIfFalse()
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

func (c *Compiler[_]) VisitForStatement(_ *ast.ForStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitEmitStatement(_ *ast.EmitStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitSwitchStatement(_ *ast.SwitchStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_]) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {
	// TODO: second value
	c.compileExpression(declaration.Value)

	varDeclTypes := c.Elaboration.VariableDeclarationTypes(declaration)
	c.emitCheckType(varDeclTypes.TargetType)

	local := c.currentFunction.declareLocal(declaration.Identifier.Identifier)
	c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: local.index})
	return
}

func (c *Compiler[_]) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {
	c.compileExpression(statement.Value)

	assignmentTypes := c.Elaboration.AssignmentStatementTypes(statement)
	c.emitCheckType(assignmentTypes.TargetType)

	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		varName := target.Identifier.Identifier
		local := c.currentFunction.findLocal(varName)
		if local != nil {
			c.codeGen.Emit(opcode.InstructionSetLocal{LocalIndex: local.index})
			return
		}

		global := c.findGlobal(varName)
		c.codeGen.Emit(opcode.InstructionSetGlobal{GlobalIndex: global.index})

	case *ast.MemberExpression:
		c.compileExpression(target.Expression)
		constant := c.addStringConst(target.Identifier.Identifier)
		c.codeGen.Emit(opcode.InstructionSetField{FieldNameIndex: constant.index})

	case *ast.IndexExpression:
		c.compileExpression(target.TargetExpression)
		c.compileExpression(target.IndexingExpression)
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
	integerType := c.Elaboration.IntegerExpressionType(expression)
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
	arrayTypes := c.Elaboration.ArrayExpressionTypes(array)

	typeIndex := c.getOrAddType(arrayTypes.ArrayType)

	size := len(array.Values)
	if size >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid array expression"))
	}

	for _, expression := range array.Values {
		//EmitDup()
		c.compileExpression(expression)
		//EmitSetIndex(index)
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

func (c *Compiler[_]) VisitDictionaryExpression(_ *ast.DictionaryExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
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
		//typ := c.Elaboration.IdentifierInInvocationType(invokedExpr)
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
		memberInfo, ok := c.Elaboration.MemberExpressionMemberAccessInfo(invokedExpr)
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

		if isInterfaceMethodInvocation(memberInfo.AccessedType) {
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

func isInterfaceMethodInvocation(accessedType sema.Type) bool {
	switch typ := accessedType.(type) {
	case *sema.ReferenceType:
		return isInterfaceMethodInvocation(typ.Type)
	case *sema.IntersectionType:
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
	invocationTypes := c.Elaboration.InvocationExpressionTypes(expression)
	for index, argument := range expression.Arguments {
		c.compileExpression(argument.Expression)
		c.emitCheckType(invocationTypes.ArgumentTypes[index])
	}

	// TODO: Is this needed?
	//// Load empty values for optional parameters, if they are not provided.
	//for i := len(expression.Arguments); i < invocationTypes.ParamCount; i++ {
	//	c.emit(opcode.Empty)
	//}
}

func (c *Compiler[_]) loadTypeArguments(expression *ast.InvocationExpression) []uint16 {
	invocationTypes := c.Elaboration.InvocationExpressionTypes(expression)

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
	c.codeGen.Emit(opcode.InstructionGetField{FieldNameIndex: constant.index})
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
			c.codeGen.Emit(opcode.InstructionIntAdd{})
		case ast.OperationMinus:
			c.codeGen.Emit(opcode.InstructionIntSubtract{})
		case ast.OperationMul:
			c.codeGen.Emit(opcode.InstructionIntMultiply{})
		case ast.OperationDiv:
			c.codeGen.Emit(opcode.InstructionIntDivide{})
		case ast.OperationMod:
			c.codeGen.Emit(opcode.InstructionIntMod{})
		case ast.OperationEqual:
			c.codeGen.Emit(opcode.InstructionEqual{})
		case ast.OperationNotEqual:
			c.codeGen.Emit(opcode.InstructionNotEqual{})
		case ast.OperationLess:
			c.codeGen.Emit(opcode.InstructionIntLess{})
		case ast.OperationLessEqual:
			c.codeGen.Emit(opcode.InstructionIntLessOrEqual{})
		case ast.OperationGreater:
			c.codeGen.Emit(opcode.InstructionIntGreater{})
		case ast.OperationGreaterEqual:
			c.codeGen.Emit(opcode.InstructionIntGreaterOrEqual{})
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

	castingTypes := c.Elaboration.CastingExpressionTypes(expression)
	index := c.getOrAddType(castingTypes.TargetType)

	castKind := opcode.CastKindFrom(expression.Operation)

	c.codeGen.Emit(
		opcode.InstructionCast{
			TypeIndex: index,
			Kind:      castKind,
		},
	)
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
	borrowType := c.Elaboration.ReferenceExpressionBorrowType(expression)
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

	enclosingCompositeType := c.compositeTypeStack.top()

	// Write composite kind
	// TODO: Maybe get/include this from static-type. Then no need to provide separately.
	kind := enclosingCompositeType.Kind

	typeIndex := c.getOrAddType(enclosingCompositeType)

	c.codeGen.Emit(
		opcode.InstructionNew{
			Kind:      kind,
			TypeIndex: typeIndex,
		},
	)

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
	compositeType := c.Elaboration.CompositeDeclarationType(declaration)
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

	for _, nestedTypes := range declaration.Members.Composites() {
		c.compileDeclaration(nestedTypes)
	}

	// TODO:

	return
}

func (c *Compiler[_]) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
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
	resolvedLocation, err := commons.ResolveLocation(
		c.Config.LocationHandler,
		declaration.Identifiers,
		declaration.Location,
	)
	if err != nil {
		panic(err)
	}

	for _, location := range resolvedLocation {
		importedProgram := c.Config.ImportHandler(location.Location)

		// Add a global variable for the imported contract value.
		contractDecl := importedProgram.Contract
		isContract := contractDecl != nil
		if isContract {
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

func (c *Compiler[_]) emitCheckType(targetType sema.Type) {
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
		sb.WriteString(typ.Identifier)
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
