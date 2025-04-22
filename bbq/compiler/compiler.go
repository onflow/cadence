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
	"math/big"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type Compiler[E, T any] struct {
	Program              *ast.Program
	DesugaredElaboration *DesugaredElaboration
	Config               *Config
	checker              *sema.Checker

	currentFunction *function[E]

	compositeTypeStack *Stack[sema.CompositeKindedType]

	functions           []*function[E]
	constants           []*Constant
	Globals             map[string]*Global
	importedGlobals     map[string]*Global
	usedImportedGlobals []*Global
	controlFlows        []controlFlow
	currentControlFlow  *controlFlow
	returns             []returns
	currentReturn       *returns
	staticTypes         []T

	// postConditionsIndices keeps track of where the post conditions start (i.e: index of the statement in the block),
	// for each function.
	// This mapping is populated by/during the desugar/rewrite: When the post conditions gets added
	// to the end of the function block, it keeps track of the index where it was added to.
	// Then the compiler uses these indices to patch the jumps for return statements.
	postConditionsIndices map[*ast.FunctionBlock]int

	// postConditionsIndex is the statement-index of the post-conditions for the current function.
	postConditionsIndex int

	// Cache alike for staticTypes and constants in the pool.
	typesInPool     map[sema.TypeID]uint16
	constantsInPool map[constantsCacheKey]*Constant

	// TODO: initialize
	memoryGauge common.MemoryGauge

	codeGen CodeGen[E]
	typeGen TypeGen[T]
}

type constantsCacheKey struct {
	data string
	kind constant.Kind
}

var _ ast.DeclarationVisitor[struct{}] = &Compiler[any, any]{}
var _ ast.StatementVisitor[struct{}] = &Compiler[any, any]{}
var _ ast.ExpressionVisitor[struct{}] = &Compiler[any, any]{}

func NewBytecodeCompiler(
	checker *sema.Checker,
	config *Config,
) *Compiler[byte, []byte] {
	return newCompiler(
		checker,
		config,
		&ByteCodeGen{},
		&EncodedTypeGen{},
	)
}

func NewInstructionCompiler(
	checker *sema.Checker,
) *Compiler[opcode.Instruction, bbq.StaticType] {
	return NewInstructionCompilerWithConfig(checker, &Config{})
}

func NewInstructionCompilerWithConfig(
	checker *sema.Checker,
	config *Config,
) *Compiler[opcode.Instruction, bbq.StaticType] {
	return newCompiler(
		checker,
		config,
		&InstructionCodeGen{},
		&DecodedTypeGen{},
	)
}

func newCompiler[E, T any](
	checker *sema.Checker,
	config *Config,
	codeGen CodeGen[E],
	typeGen TypeGen[T],
) *Compiler[E, T] {

	var globals map[string]*Global
	if config.BuiltinGlobalsProvider != nil {
		globals = config.BuiltinGlobalsProvider()
	} else {
		globals = NativeFunctions()
	}

	return &Compiler[E, T]{
		Program:              checker.Program,
		DesugaredElaboration: NewDesugaredElaboration(checker.Elaboration),
		Config:               config,
		checker:              checker,
		Globals:              make(map[string]*Global),
		importedGlobals:      globals,
		typesInPool:          make(map[sema.TypeID]uint16),
		constantsInPool:      make(map[constantsCacheKey]*Constant),
		compositeTypeStack: &Stack[sema.CompositeKindedType]{
			elements: make([]sema.CompositeKindedType, 0),
		},
		codeGen:             codeGen,
		typeGen:             typeGen,
		postConditionsIndex: -1,
	}
}

func (c *Compiler[_, _]) findGlobal(name string) *Global {
	global, ok := c.Globals[name]
	if ok {
		return global
	}

	// If failed to find, then try with type-qualified name.
	// This is because contract functions/type-constructors can be accessed without the contract name.
	// e.g: SomeContract.Foo() == Foo(), within `SomeContract`.
	if !c.compositeTypeStack.isEmpty() {
		enclosingContract := c.compositeTypeStack.bottom()
		typeQualifiedName := commons.TypeQualifiedName(enclosingContract.GetIdentifier(), name)
		global, ok = c.Globals[typeQualifiedName]
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
	count := len(c.Globals)
	if count >= math.MaxUint16 {
		panic(errors.NewUnexpectedError("invalid global declaration '%s'", name))
	}
	importedGlobal.Index = uint16(count)
	c.Globals[name] = importedGlobal

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

func (c *Compiler[_, _]) addGlobal(name string) *Global {
	count := len(c.Globals)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid global declaration"))
	}
	global := &Global{
		Index: uint16(count),
	}
	c.Globals[name] = global
	return global
}

func (c *Compiler[_, _]) addImportedGlobal(location common.Location, name string) *Global {
	// Index is not set here. It is set only if this imported global is used.
	global := &Global{
		Location: location,
		Name:     name,
	}
	c.importedGlobals[name] = global
	return global
}

func (c *Compiler[E, T]) addFunction(
	name string,
	qualifiedName string,
	parameterCount uint16,
	functionType *sema.FunctionType,
) *function[E] {

	functionTypeIndex := c.getOrAddType(functionType)

	function := newFunction[E](
		c.currentFunction,
		name,
		qualifiedName,
		parameterCount,
		functionTypeIndex,
	)
	c.functions = append(c.functions, function)

	return function
}

func (c *Compiler[E, T]) targetFunction(function *function[E]) {
	c.currentFunction = function

	var code *[]E
	if function != nil {
		code = &function.code
	}
	c.codeGen.SetTarget(code)
}

func (c *Compiler[_, _]) addConstant(kind constant.Kind, data []byte) *Constant {
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

	constant := &Constant{
		index: uint16(count),
		kind:  kind,
		data:  data[:],
	}
	c.constants = append(c.constants, constant)
	c.constantsInPool[cacheKey] = constant
	return constant
}

func (c *Compiler[_, _]) emitGetConstant(constant *Constant) {
	c.codeGen.Emit(opcode.InstructionGetConstant{
		Constant: constant.index,
	})
}

func (c *Compiler[_, _]) emitStringConst(str string) {
	c.emitGetConstant(c.addStringConst(str))
}

func (c *Compiler[_, _]) addStringConst(str string) *Constant {
	return c.addConstant(constant.String, []byte(str))
}

func (c *Compiler[_, _]) emitIntConst(i int64) {
	c.emitGetConstant(c.addIntConst(i))
}

func (c *Compiler[_, _]) addIntConst(i int64) *Constant {
	data := leb128.AppendInt64(nil, i)
	return c.addConstant(constant.Int, data)
}

func (c *Compiler[_, _]) emitJump(target int) int {
	if target >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJump{Target: uint16(target)})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJump() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJump{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfFalse() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfFalse{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfTrue() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfTrue{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfNil() int {
	offset := c.codeGen.Offset()
	c.codeGen.Emit(opcode.InstructionJumpIfNil{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) patchJump(opcodeOffset int) {
	count := c.codeGen.Offset()
	if count == 0 {
		panic(errors.NewUnreachableError())
	}
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	c.codeGen.PatchJump(opcodeOffset, uint16(count))
}

func (c *Compiler[_, _]) patchJumps(offsets []int) {
	for _, offset := range offsets {
		c.patchJump(offset)
	}
}

func (c *Compiler[_, _]) pushControlFlow(start int) {
	index := len(c.controlFlows)
	c.controlFlows = append(c.controlFlows, controlFlow{start: start})
	c.currentControlFlow = &c.controlFlows[index]
}

func (c *Compiler[_, _]) popControlFlow() {
	lastIndex := len(c.controlFlows) - 1
	l := c.controlFlows[lastIndex]
	c.controlFlows[lastIndex] = controlFlow{}
	c.controlFlows = c.controlFlows[:lastIndex]

	c.patchJumps(l.breaks)

	var previousControlFlow *controlFlow
	if lastIndex > 0 {
		previousControlFlow = &c.controlFlows[lastIndex-1]
	}
	c.currentControlFlow = previousControlFlow
}

func (c *Compiler[_, _]) pushReturns() {
	index := len(c.returns)
	c.returns = append(c.returns, returns{})
	c.currentReturn = &c.returns[index]
}

func (c *Compiler[_, _]) popReturns() {
	lastIndex := len(c.returns) - 1
	c.returns[lastIndex] = returns{}
	c.returns = c.returns[:lastIndex]

	var previousReturns *returns
	if lastIndex > 0 {
		previousReturns = &c.returns[lastIndex-1]
	}
	c.currentReturn = previousReturns
}

func (c *Compiler[E, T]) Compile() *bbq.Program[E, T] {

	// Desugar the program before compiling.
	desugar := NewDesugar(
		c.memoryGauge,
		c.Config,
		c.Program,
		c.DesugaredElaboration,
		c.checker,
	)
	c.Program, c.postConditionsIndices = desugar.Run()

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

	return &bbq.Program[E, T]{
		Functions: functions,
		Constants: constants,
		Types:     types,
		Imports:   imports,
		Contract:  contract,
		Variables: variables,
	}
}

func (c *Compiler[_, _]) reserveGlobalVars(
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

	// Add natively provided methods as globals.
	// Only do it for user-defined types (i.e: `compositeTypeName` is not empty).
	if compositeTypeName != "" {
		for _, boundFunction := range commonBuiltinTypeBoundFunctions {
			funcName := commons.TypeQualifiedName(compositeTypeName, boundFunction.name)
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
			// TODO: handle name clash?!
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

func (c *Compiler[_, _]) exportConstants() []constant.Constant {
	var constants []constant.Constant

	count := len(c.constants)
	if count > 0 {
		constants = make([]constant.Constant, 0, count)
		for _, c := range c.constants {
			constants = append(
				constants,
				constant.Constant{
					Data: c.data,
					Kind: c.kind,
				},
			)
		}
	}

	return constants
}

func (c *Compiler[_, T]) exportTypes() []T {
	return c.staticTypes
}

func (c *Compiler[_, _]) exportImports() []bbq.Import {
	var exportedImports []bbq.Import

	count := len(c.usedImportedGlobals)
	if count > 0 {
		exportedImports = make([]bbq.Import, 0, count)
		for _, importedGlobal := range c.usedImportedGlobals {
			bbqImport := bbq.Import{
				Location: importedGlobal.Location,
				Name:     importedGlobal.Name,
			}
			exportedImports = append(exportedImports, bbqImport)
		}
	}

	return exportedImports
}

func (c *Compiler[E, T]) ExportFunctions() []bbq.Function[E] {
	var functions []bbq.Function[E]

	count := len(c.functions)
	if count > 0 {
		functions = make([]bbq.Function[E], 0, count)
		for _, function := range c.functions {
			functions = append(
				functions,
				bbq.Function[E]{
					Name:           function.name,
					QualifiedName:  function.qualifiedName,
					Code:           function.code,
					LocalCount:     function.localCount,
					ParameterCount: function.parameterCount,
					TypeIndex:      function.typeIndex,
				},
			)
		}
	}

	return functions
}

func (c *Compiler[_, _]) exportVariables(variableDecls []*ast.VariableDeclaration) []bbq.Variable {
	var variables []bbq.Variable

	count := len(c.functions)
	if count > 0 {
		variables = make([]bbq.Variable, 0, count)
		for _, varDecl := range variableDecls {
			variables = append(
				variables,
				bbq.Variable{
					Name: varDecl.Identifier.Identifier,
				},
			)
		}
	}

	return variables
}

func (c *Compiler[_, _]) contractType() (contractType sema.CompositeKindedType) {
	contractDecl := c.Program.SoleContractDeclaration()
	if contractDecl != nil {
		contractType = c.DesugaredElaboration.CompositeDeclarationType(contractDecl)
		return
	}

	interfaceDecl := c.Program.SoleContractInterfaceDeclaration()
	if interfaceDecl != nil {
		contractType = c.DesugaredElaboration.InterfaceDeclarationType(interfaceDecl)
		return
	}

	return nil
}

func (c *Compiler[_, _]) exportContract() (*bbq.Contract, sema.CompositeKindedType) {
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

func (c *Compiler[_, _]) compileDeclaration(declaration ast.Declaration) {
	ast.AcceptDeclaration[struct{}](declaration, c)
}

func (c *Compiler[_, _]) compileBlock(block *ast.Block, enclosingDeclKind common.DeclarationKind) {
	locals := c.currentFunction.locals
	locals.PushNewWithCurrent()
	defer locals.Pop()

	if c.shouldPatchReturns(enclosingDeclKind) {
		c.pushReturns()
		defer c.popReturns()

		for index, statement := range block.Statements {
			// Once the post conditions are reached, patch all the previous return statements
			// to jump to the current index (i.e: update them to jump to the post conditions).
			if index == c.postConditionsIndex {
				c.patchJumps(c.currentReturn.returns)
			}
			c.compileStatement(statement)
		}
	} else {
		for _, statement := range block.Statements {
			c.compileStatement(statement)
		}
	}

	// Add returns for functions.
	// Initializers don't return anything explicitly. So do not add a return for initializers.
	// However, initializer has an implicit return for the constructed value.
	// For that, the `compileInitializer` function is adding a return (with `self` value).

	switch enclosingDeclKind {
	case common.DeclarationKindFunction:
		if c.hasPostConditions() {
			// If there are post-conditions, then the compilation of `return` statements
			// doesn't emit return instructions (they just jump to the post conditions).
			// So a return MUST be emitted here.

			local := c.currentFunction.findLocal(tempResultVariableName)
			if local == nil {
				c.codeGen.Emit(opcode.InstructionReturn{})
			} else {
				c.emitGetLocal(local.index)
				c.codeGen.Emit(opcode.InstructionReturnValue{})
			}
		} else if needsSyntheticReturn(block.Statements) {
			// If there are no post conditions,
			// and if there is no return statement at the end,
			// then emit an empty return.
			c.codeGen.Emit(opcode.InstructionReturn{})
		}
	}
}

func needsSyntheticReturn(statements []ast.Statement) bool {
	length := len(statements)
	if length == 0 {
		return true
	}

	lastStatement := statements[length-1]
	_, isReturn := lastStatement.(*ast.ReturnStatement)
	return !isReturn
}

// shouldPatchReturns determines whether to patch the return-statements emitted so far.
// Return statements should only be patched at the function-block level,
// but not inside nested blocks.
func (c *Compiler[_, _]) shouldPatchReturns(enclosingDeclKind common.DeclarationKind) bool {
	// Functions (regular functions and initializers) can have post conditions.
	switch enclosingDeclKind {
	case common.DeclarationKindFunction, common.DeclarationKindInitializer:
		return c.hasPostConditions()
	default:
		return false
	}
}

func (c *Compiler[_, _]) compileFunctionBlock(functionBlock *ast.FunctionBlock, functionDeclKind common.DeclarationKind) {
	if functionBlock == nil {
		return
	}

	// Function conditions must have been desugared to statements.
	// So there shouldn't be any condition at this point.
	if functionBlock.PreConditions != nil ||
		functionBlock.PostConditions != nil {
		panic(errors.NewUnreachableError())
	}

	prevPostConditionIndex := c.postConditionsIndex
	index, ok := c.postConditionsIndices[functionBlock]
	if ok {
		c.postConditionsIndex = index
	} else {
		c.postConditionsIndex = -1
	}
	defer func() {
		c.postConditionsIndex = prevPostConditionIndex
	}()

	c.compileBlock(functionBlock.Block, functionDeclKind)
}

func (c *Compiler[_, _]) compileStatement(statement ast.Statement) {
	ast.AcceptStatement[struct{}](statement, c)
}

func (c *Compiler[_, _]) compileExpression(expression ast.Expression) {
	ast.AcceptExpression[struct{}](expression, c)
}

func (c *Compiler[_, _]) VisitReturnStatement(statement *ast.ReturnStatement) (_ struct{}) {
	expression := statement.Expression

	// There can be five different variations of return values:
	//  (1) Return with a value
	//	    (1.a) With post conditions
	//	        (1.a.i) Return value is non-empty -> Store the value in temp-var and jump to post conditions
	//	        (1.a.ii) Return value is void -> Drop and jump to post conditions
	//	    (1.b) No post conditions -> Return in-place
	//	(2) Empty return
	//	    (2.a) With post conditions -> Jump to post conditions
	//	    (2.b) No post conditions -> Return in-place
	//
	// In summary, if there are post conditions,this will jump to the post conditions.
	// Then, the `compileBlock` function is responsible for adding the actual return,
	// after compiling the rest of the statements (post conditions).

	if expression != nil {
		// TODO: copy
		c.compileExpression(expression)

		if c.hasPostConditions() {
			tempResultVar := c.currentFunction.findLocal(tempResultVariableName)

			if tempResultVar != nil {
				// (1.a.i)
				// Assign the return value to the temp-result variable.
				c.emitSetLocal(tempResultVar.index)
			} else {
				// (1.a.ii)
				// If there is no temp-result variable, that means the return type is void.
				// So just drop the void-value.
				c.codeGen.Emit(opcode.InstructionDrop{})
			}

			// And jump to the start of the post conditions.
			offset := c.emitUndefinedJump()
			c.currentReturn.appendReturn(offset)
		} else {
			// (1.b)
			// If there are no post conditions, return then-and-there.
			c.codeGen.Emit(opcode.InstructionReturnValue{})
		}
	} else {
		if c.hasPostConditions() {
			// (2.a)
			// If there are post conditions, jump to the start of the post conditions.
			offset := c.emitUndefinedJump()
			c.currentReturn.appendReturn(offset)
		} else {
			// (2.b)
			// If there are no post conditions, return then-and-there.
			c.codeGen.Emit(opcode.InstructionReturn{})
		}
	}

	return
}

func (c *Compiler[_, _]) emitGetLocal(localIndex uint16) {
	c.codeGen.Emit(opcode.InstructionGetLocal{
		Local: localIndex,
	})
}

func (c *Compiler[_, _]) emitSetLocal(localIndex uint16) {
	c.codeGen.Emit(opcode.InstructionSetLocal{
		Local: localIndex,
	})
}

func (c *Compiler[_, _]) hasPostConditions() bool {
	return c.postConditionsIndex >= 0
}

func (c *Compiler[_, _]) VisitBreakStatement(_ *ast.BreakStatement) (_ struct{}) {
	offset := c.emitUndefinedJump()
	c.currentControlFlow.appendBreak(offset)
	return
}

func (c *Compiler[_, _]) VisitContinueStatement(_ *ast.ContinueStatement) (_ struct{}) {
	start := c.currentControlFlow.start
	if start <= 0 {
		panic(errors.NewUnreachableError())
	}
	c.emitJump(start)
	return
}

func (c *Compiler[_, _]) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {
	// If-statements can be coming from inherited conditions.
	// If so, use the corresponding elaboration.
	c.withConditionElaboration(statement, func() {
		var (
			elseJump            int
			additionalThenScope bool
		)

		switch test := statement.Test.(type) {
		case ast.Expression:
			c.compileExpression(test)
			elseJump = c.emitUndefinedJumpIfFalse()

		case *ast.VariableDeclaration:
			// TODO: second value

			// Compile the value expression *before* declaring the variable
			c.compileExpression(test.Value)

			tempIndex := c.currentFunction.generateLocalIndex()
			c.emitSetLocal(tempIndex)

			// Test: check if the optional is nil,
			// and jump to the else branch if it is
			c.emitGetLocal(tempIndex)
			elseJump = c.emitUndefinedJumpIfNil()

			// Then branch: unwrap the optional and declare the variable
			c.emitGetLocal(tempIndex)
			c.codeGen.Emit(opcode.InstructionUnwrap{})
			varDeclTypes := c.DesugaredElaboration.VariableDeclarationTypes(test)
			c.emitTransfer(varDeclTypes.TargetType)

			// Declare the variable *after* unwrapping the optional,
			// in a new scope
			c.currentFunction.locals.PushNewWithCurrent()
			additionalThenScope = true
			name := test.Identifier.Identifier
			c.emitDeclareLocal(name)

		default:
			panic(errors.NewUnreachableError())
		}

		c.compileBlock(statement.Then, common.DeclarationKindUnknown)

		if additionalThenScope {
			c.currentFunction.locals.Pop()
		}

		elseBlock := statement.Else
		if elseBlock != nil {
			thenJump := c.emitUndefinedJump()
			c.patchJump(elseJump)
			c.compileBlock(elseBlock, common.DeclarationKindUnknown)
			c.patchJump(thenJump)
		} else {
			c.patchJump(elseJump)
		}
	})
	return
}

func (c *Compiler[_, _]) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {
	testOffset := c.codeGen.Offset()

	c.pushControlFlow(testOffset)
	defer c.popControlFlow()

	c.compileExpression(statement.Test)
	endJump := c.emitUndefinedJumpIfFalse()

	// Compile the body
	c.compileBlock(statement.Block, common.DeclarationKindUnknown)
	// Repeat, jump back to the test
	c.emitJump(testOffset)

	// Patch the failed test to jump here
	c.patchJump(endJump)

	return
}

func (c *Compiler[_, _]) VisitForStatement(statement *ast.ForStatement) (_ struct{}) {
	// Evaluate the expression
	c.compileExpression(statement.Value)

	// Get an iterator to the resulting value, and store it in a local index.
	c.codeGen.Emit(opcode.InstructionIterator{})
	iteratorLocalIndex := c.currentFunction.generateLocalIndex()
	c.codeGen.Emit(opcode.InstructionSetLocal{
		Local: iteratorLocalIndex,
	})

	// Initialize 'index' variable, if needed.
	index := statement.Index
	indexNeeded := index != nil
	var indexLocalVar *local

	if indexNeeded {
		// `var <index> = -1`
		// Start with -1 and then increment at the start of the loop,
		// so that we don't have to deal with early exists of the loop.
		c.emitIntConst(-1)
		indexLocalVar = c.emitDeclareLocal(index.Identifier)
	}

	testOffset := c.codeGen.Offset()
	c.pushControlFlow(testOffset)
	defer c.popControlFlow()

	// Loop test: Get the iterator and call `hasNext()`.
	c.emitGetLocal(iteratorLocalIndex)
	c.codeGen.Emit(opcode.InstructionIteratorHasNext{})

	endJump := c.emitUndefinedJumpIfFalse()

	// Loop Body.

	// Increment the index if needed.
	// This is done as the first thing inside the loop, so that we don't need to
	// worry about loop-control statements (e.g: continue, return, break) in the body.
	if indexNeeded {
		// <index> = <index> + 1
		c.emitGetLocal(indexLocalVar.index)
		c.emitIntConst(1)
		c.codeGen.Emit(opcode.InstructionAdd{})
		c.emitSetLocal(indexLocalVar.index)
	}

	// Get the iterator and call `next()` (value for arrays, key for dictionaries, etc.)
	c.emitGetLocal(iteratorLocalIndex)
	c.codeGen.Emit(opcode.InstructionIteratorNext{})

	// Store it (next entry) in a local var.
	// `<entry> = iterator.next()`
	c.emitDeclareLocal(statement.Identifier.Identifier)

	// Compile the for-loop body.
	c.compileBlock(statement.Block, common.DeclarationKindUnknown)

	// Jump back to the loop test. i.e: `hasNext()`
	c.emitJump(testOffset)

	c.patchJump(endJump)
	return
}

func (c *Compiler[_, _]) VisitEmitStatement(statement *ast.EmitStatement) (_ struct{}) {
	// Emit statements can be coming from inherited conditions.
	// If so, use the corresponding elaboration.
	c.withConditionElaboration(
		statement,
		func() {
			c.compileExpression(statement.InvocationExpression)
			eventType := c.DesugaredElaboration.EmitStatementEventType(statement)
			typeIndex := c.getOrAddType(eventType)
			c.codeGen.Emit(opcode.InstructionEmitEvent{
				Type: typeIndex,
			})
		},
	)

	return
}

func (c *Compiler[_, _]) VisitSwitchStatement(statement *ast.SwitchStatement) (_ struct{}) {
	c.compileExpression(statement.Expression)
	localIndex := c.currentFunction.generateLocalIndex()
	c.emitSetLocal(localIndex)

	// Pass an invalid start offset to pushControlFlow to indicate that this is a switch statement,
	// which does not allow jumps to the start (i.e., no continue statements).
	c.pushControlFlow(-1)
	defer c.popControlFlow()

	previousJump := -1

	for _, switchCase := range statement.Cases {
		if previousJump >= 0 {
			c.patchJump(previousJump)
			previousJump = -1
		}

		isDefault := switchCase.Expression == nil
		if !isDefault {
			c.emitGetLocal(localIndex)
			c.compileExpression(switchCase.Expression)
			c.codeGen.Emit(opcode.InstructionEqual{})
			previousJump = c.emitUndefinedJumpIfFalse()
		}

		for _, caseStatement := range switchCase.Statements {
			c.compileStatement(caseStatement)
		}

		if !isDefault {
			breakOffset := c.emitUndefinedJump()
			c.currentControlFlow.appendBreak(breakOffset)
		}
	}

	if previousJump >= 0 {
		c.patchJump(previousJump)
	}

	return
}

func (c *Compiler[_, _]) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {
	// Some variable declarations can be coming from inherited before-statements.
	// If so, use the corresponding elaboration.
	c.withConditionElaboration(declaration, func() {

		// TODO: second value

		name := declaration.Identifier.Identifier
		// TODO: This can be nil only for synthetic-result variable
		//   Any better way to handle this?
		if declaration.Value == nil {
			c.currentFunction.declareLocal(name)
		} else {
			// Compile the value expression *before* declaring the variable
			c.compileExpression(declaration.Value)

			varDeclTypes := c.DesugaredElaboration.VariableDeclarationTypes(declaration)
			c.emitTransfer(varDeclTypes.TargetType)

			// Declare the variable *after* compiling the value expression
			c.emitDeclareLocal(name)
		}
	})

	return
}

func (c *Compiler[_, _]) emitDeclareLocal(name string) *local {
	local := c.currentFunction.declareLocal(name)
	c.emitSetLocal(local.index)
	return local
}

func (c *Compiler[_, _]) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {

	switch target := statement.Target.(type) {
	case *ast.IdentifierExpression:
		c.compileExpression(statement.Value)
		assignmentTypes := c.DesugaredElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		c.emitVariableStore(target.Identifier.Identifier)

	case *ast.MemberExpression:
		c.compileExpression(target.Expression)

		c.compileExpression(statement.Value)
		assignmentTypes := c.DesugaredElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		constant := c.addStringConst(target.Identifier.Identifier)
		c.codeGen.Emit(opcode.InstructionSetField{
			FieldName: constant.index,
		})

	case *ast.IndexExpression:
		c.compileExpression(target.TargetExpression)
		c.compileExpression(target.IndexingExpression)

		c.compileExpression(statement.Value)
		assignmentTypes := c.DesugaredElaboration.AssignmentStatementTypes(statement)
		c.emitTransfer(assignmentTypes.TargetType)

		c.codeGen.Emit(opcode.InstructionSetIndex{})

	default:
		// TODO:
		panic(errors.NewUnreachableError())
	}
	return
}

func (c *Compiler[_, _]) VisitSwapStatement(_ *ast.SwapStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitExpressionStatement(statement *ast.ExpressionStatement) (_ struct{}) {
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

func (c *Compiler[_, _]) VisitVoidExpression(_ *ast.VoidExpression) (_ struct{}) {
	//TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitBoolExpression(expression *ast.BoolExpression) (_ struct{}) {
	if expression.Value {
		c.codeGen.Emit(opcode.InstructionTrue{})
	} else {
		c.codeGen.Emit(opcode.InstructionFalse{})
	}
	return
}

func (c *Compiler[_, _]) VisitNilExpression(_ *ast.NilExpression) (_ struct{}) {
	c.codeGen.Emit(opcode.InstructionNil{})
	return
}

func (c *Compiler[_, _]) VisitIntegerExpression(expression *ast.IntegerExpression) (_ struct{}) {
	integerType := c.DesugaredElaboration.IntegerExpressionType(expression)
	constantKind := constant.FromSemaType(integerType)

	value := expression.Value
	var data []byte

	switch constantKind {
	case constant.Int:
		// TODO: support larger integers
		data = leb128.AppendInt64(nil, value.Int64())

	case constant.Int8,
		constant.Int16,
		constant.Int32:
		data = leb128.AppendInt32(nil, int32(value.Int64()))

	case constant.Int64:
		data = leb128.AppendInt64(nil, value.Int64())

	case constant.UInt:
		// TODO: support larger integers
		data = leb128.AppendUint64(nil, value.Uint64())

	case constant.UInt8,
		constant.Word8,
		constant.UInt16,
		constant.Word16,
		constant.UInt32,
		constant.Word32:
		data = leb128.AppendUint32(nil, uint32(value.Uint64()))

	case constant.UInt64,
		constant.Word64:
		data = leb128.AppendUint64(nil, value.Uint64())

	case constant.Address:
		data = value.Bytes()

	// TODO:
	// case constantkind.Int128:
	// case constantkind.Int256:
	// case constantkind.UInt128:
	// case constantkind.UInt256:
	// case constantkind.Word128:
	// case constantkind.Word256:

	default:
		panic(errors.NewUnexpectedError("unsupported constant kind: %s", constantKind))
	}

	c.emitGetConstant(c.addConstant(constantKind, data))

	return
}

func (c *Compiler[_, _]) VisitFixedPointExpression(expression *ast.FixedPointExpression) (_ struct{}) {
	// TODO: adjust once/if we support more fixed point types

	fixedPointSubType := c.DesugaredElaboration.FixedPointExpressionType(expression)

	value := fixedpoint.ConvertToFixedPointBigInt(
		expression.Negative,
		expression.UnsignedInteger,
		expression.Fractional,
		expression.Scale,
		sema.Fix64Scale,
	)

	var constant *Constant

	switch fixedPointSubType {
	case sema.Fix64Type, sema.SignedFixedPointType:
		constant = c.addFix64Constant(value)

	case sema.UFix64Type:
		constant = c.addUFix64Constant(value)

	case sema.FixedPointType:
		if expression.Negative {
			constant = c.addFix64Constant(value)
		} else {
			constant = c.addUFix64Constant(value)
		}
	default:
		panic(errors.NewUnreachableError())
	}

	c.emitGetConstant(constant)

	return
}

func (c *Compiler[_, _]) addUFix64Constant(value *big.Int) *Constant {
	data := leb128.AppendUint64(nil, value.Uint64())
	return c.addConstant(constant.UFix64, data)
}

func (c *Compiler[_, _]) addFix64Constant(value *big.Int) *Constant {
	data := leb128.AppendInt64(nil, value.Int64())
	return c.addConstant(constant.Fix64, data)
}

func (c *Compiler[_, _]) VisitArrayExpression(array *ast.ArrayExpression) (_ struct{}) {
	arrayTypes := c.DesugaredElaboration.ArrayExpressionTypes(array)

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
			Type:       typeIndex,
			Size:       uint16(size),
			IsResource: arrayTypes.ArrayType.IsResourceType(),
		},
	)

	return
}

func (c *Compiler[_, _]) VisitDictionaryExpression(dictionary *ast.DictionaryExpression) (_ struct{}) {
	dictionaryTypes := c.DesugaredElaboration.DictionaryExpressionTypes(dictionary)

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
			Type:       typeIndex,
			Size:       uint16(size),
			IsResource: dictionaryTypes.DictionaryType.IsResourceType(),
		},
	)

	return
}

func (c *Compiler[_, _]) VisitIdentifierExpression(expression *ast.IdentifierExpression) (_ struct{}) {
	c.emitVariableLoad(expression.Identifier.Identifier)
	return
}

func (c *Compiler[_, _]) emitVariableLoad(name string) {
	local := c.currentFunction.findLocal(name)
	if local != nil {
		c.emitGetLocal(local.index)
		return
	}

	upvalueIndex, ok := c.currentFunction.findOrAddUpvalue(name)
	if ok {
		c.codeGen.Emit(opcode.InstructionGetUpvalue{
			Upvalue: upvalueIndex,
		})
		return
	}

	global := c.findGlobal(name)
	c.codeGen.Emit(opcode.InstructionGetGlobal{
		Global: global.Index,
	})
}

func (c *Compiler[_, _]) emitVariableStore(name string) {
	local := c.currentFunction.findLocal(name)
	if local != nil {
		c.emitSetLocal(local.index)
		return
	}

	upvalueIndex, ok := c.currentFunction.findOrAddUpvalue(name)
	if ok {
		c.codeGen.Emit(opcode.InstructionSetUpvalue{
			Upvalue: upvalueIndex,
		})
		return
	}

	global := c.findGlobal(name)
	c.codeGen.Emit(opcode.InstructionSetGlobal{
		Global: global.Index,
	})
}

func (c *Compiler[_, _]) VisitInvocationExpression(expression *ast.InvocationExpression) (_ struct{}) {
	// TODO: copy

	switch invokedExpr := expression.InvokedExpression.(type) {
	case *ast.IdentifierExpression:
		// TODO: Does constructors need any special handling?
		//typ := c.DesugaredElaboration.IdentifierInInvocationType(invokedExpr)
		//invocationType := typ.(*sema.FunctionType)
		//if invocationType.IsConstructor {
		//}

		// Compile arguments
		c.compileArguments(expression)
		// Load function value
		c.emitVariableLoad(invokedExpr.Identifier.Identifier)

		typeArgs := c.loadTypeArguments(expression)
		c.codeGen.Emit(opcode.InstructionInvoke{TypeArgs: typeArgs})

	case *ast.MemberExpression:
		memberInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(invokedExpr)
		if !ok {
			// TODO: verify
			panic(errors.NewUnreachableError())
		}

		typeName := commons.TypeQualifier(memberInfo.AccessedType)
		var funcName string

		invocationType := memberInfo.Member.TypeAnnotation.Type.(*sema.FunctionType)
		if invocationType.IsConstructor {
			funcName = commons.TypeQualifiedName(typeName, invokedExpr.Identifier.Identifier)

			// Calling a type constructor must be invoked statically. e.g: `SomeContract.Foo()`.
			// Compile arguments
			c.compileArguments(expression)
			// Load function value
			c.emitVariableLoad(funcName)

			typeArgs := c.loadTypeArguments(expression)
			c.codeGen.Emit(opcode.InstructionInvoke{TypeArgs: typeArgs})
			return
		}

		// Receiver is loaded first. So 'self' is always the zero-th argument.
		c.compileExpression(invokedExpr.Expression)

		// Compile arguments
		c.compileArguments(expression)

		typeArgs := c.loadTypeArguments(expression)

		// Invocations into the interface code, such as default functions and inherited conditions,
		// that were synthetically added at the desugar phase, must be static calls.
		isInterfaceInheritedFuncCall := c.DesugaredElaboration.IsInterfaceMethodStaticCall(expression)

		// Any invocation on restricted-types must be dynamic
		if !isInterfaceInheritedFuncCall && isDynamicMethodInvocation(memberInfo.AccessedType) {
			funcName = invokedExpr.Identifier.Identifier
			if len(funcName) >= math.MaxUint16 {
				panic(errors.NewDefaultUserError("invalid function name"))
			}

			argumentCount := len(expression.Arguments)
			if argumentCount >= math.MaxUint16 {
				panic(errors.NewDefaultUserError("invalid number of arguments"))
			}

			funcNameConst := c.addStringConst(funcName)
			c.codeGen.Emit(
				opcode.InstructionInvokeMethodDynamic{
					Name:     funcNameConst.index,
					TypeArgs: typeArgs,
					ArgCount: uint16(argumentCount),
				},
			)

		} else {
			// Load function value
			funcName = commons.TypeQualifiedName(typeName, invokedExpr.Identifier.Identifier)
			c.emitVariableLoad(funcName)

			c.codeGen.Emit(opcode.InstructionInvokeMethodStatic{
				TypeArgs: typeArgs,
			})
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

func (c *Compiler[_, _]) compileArguments(expression *ast.InvocationExpression) {
	invocationTypes := c.DesugaredElaboration.InvocationExpressionTypes(expression)
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

func (c *Compiler[_, _]) loadTypeArguments(expression *ast.InvocationExpression) []uint16 {
	invocationTypes := c.DesugaredElaboration.InvocationExpressionTypes(expression)

	typeArgsCount := invocationTypes.TypeArguments.Len()
	if typeArgsCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid number of type arguments: %d", typeArgsCount))
	}

	var typeArgs []uint16
	if typeArgsCount > 0 {
		typeArgs = make([]uint16, 0, typeArgsCount)

		invocationTypes.TypeArguments.Foreach(func(key *sema.TypeParameter, typeParam sema.Type) {
			typeArgs = append(typeArgs, c.getOrAddType(typeParam))
		})
	}

	return typeArgs
}

func (c *Compiler[_, _]) VisitMemberExpression(expression *ast.MemberExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)

	memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	if memberAccessInfo.IsOptional {
		// TODO: Complete the optional-chaining implementations.
		//  e.g: Need a nil check, since unwrap panics on nil
		c.codeGen.Emit(opcode.InstructionUnwrap{})
	}

	constant := c.addStringConst(expression.Identifier.Identifier)

	// TODO: remove member if `isNestedResourceMove`
	//  See `Interpreter.memberExpressionGetterSetter` for the reference implementation.
	c.codeGen.Emit(opcode.InstructionGetField{
		FieldName: constant.index,
	})

	// Return a reference, if the member is accessed via a reference.
	// This is pre-computed at the checker.
	if memberAccessInfo.ReturnReference {
		index := c.getOrAddType(memberAccessInfo.ResultingType)
		c.codeGen.Emit(opcode.InstructionNewRef{
			Type:       index,
			IsImplicit: true,
		})
	}

	// TODO: Need to wrap the result back with an optional, if `memberAccessInfo.IsOptional`

	return
}

func (c *Compiler[_, _]) VisitIndexExpression(expression *ast.IndexExpression) (_ struct{}) {
	c.compileExpression(expression.TargetExpression)
	c.compileExpression(expression.IndexingExpression)
	c.codeGen.Emit(opcode.InstructionGetIndex{})

	indexExpressionTypes, ok := c.DesugaredElaboration.IndexExpressionTypes(expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Return a reference, if the element is accessed via a reference.
	// This is pre-computed at the checker.
	if indexExpressionTypes.ReturnReference {
		index := c.getOrAddType(indexExpressionTypes.ResultType)
		c.codeGen.Emit(opcode.InstructionNewRef{
			Type:       index,
			IsImplicit: true,
		})
	}

	return
}

func (c *Compiler[_, _]) VisitConditionalExpression(expression *ast.ConditionalExpression) (_ struct{}) {
	// Test
	c.compileExpression(expression.Test)
	elseJump := c.emitUndefinedJumpIfFalse()

	// Then branch
	c.compileExpression(expression.Then)
	thenJump := c.emitUndefinedJump()

	// Else branch
	c.patchJump(elseJump)
	c.compileExpression(expression.Else)

	c.patchJump(thenJump)

	return
}

func (c *Compiler[_, _]) VisitUnaryExpression(expression *ast.UnaryExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)

	switch expression.Operation {
	case ast.OperationNegate:
		c.codeGen.Emit(opcode.InstructionNot{})

	case ast.OperationMinus:
		c.codeGen.Emit(opcode.InstructionNegate{})

	case ast.OperationMul:
		c.codeGen.Emit(opcode.InstructionDeref{})

	case ast.OperationMove:
		// TODO: invalidate

	default:
		panic(errors.NewUnreachableError())
	}

	return
}

func (c *Compiler[_, _]) VisitBinaryExpression(expression *ast.BinaryExpression) (_ struct{}) {
	c.compileExpression(expression.Left)
	// TODO: add support for other types

	switch expression.Operation {
	case ast.OperationNilCoalesce:
		// Duplicate the value for the nil equality check.
		c.codeGen.Emit(opcode.InstructionDup{})
		elseJump := c.emitUndefinedJumpIfNil()

		// Then branch
		c.codeGen.Emit(opcode.InstructionUnwrap{})
		thenJump := c.emitUndefinedJump()

		// Else branch
		c.patchJump(elseJump)
		// Drop the duplicated condition result,
		// as it is not needed for the 'else' path.
		c.codeGen.Emit(opcode.InstructionDrop{})
		c.compileExpression(expression.Right)

		// End
		c.patchJump(thenJump)

	case ast.OperationOr:
		// TODO: optimize chains of ors / ands

		leftTrueJump := c.emitUndefinedJumpIfTrue()

		c.compileExpression(expression.Right)
		rightFalseJump := c.emitUndefinedJumpIfFalse()

		// Left or right is true
		c.patchJump(leftTrueJump)
		c.codeGen.Emit(opcode.InstructionTrue{})
		trueJump := c.emitUndefinedJump()

		// Left and right are false
		c.patchJump(rightFalseJump)
		c.codeGen.Emit(opcode.InstructionFalse{})

		c.patchJump(trueJump)

	case ast.OperationAnd:
		// TODO: optimize chains of ors / ands

		leftFalseJump := c.emitUndefinedJumpIfFalse()

		c.compileExpression(expression.Right)
		rightFalseJump := c.emitUndefinedJumpIfFalse()

		// Left and right are true
		c.codeGen.Emit(opcode.InstructionTrue{})
		trueJump := c.emitUndefinedJump()

		// Left or right is false
		c.patchJump(leftFalseJump)
		c.patchJump(rightFalseJump)
		c.codeGen.Emit(opcode.InstructionFalse{})

		c.patchJump(trueJump)

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

		case ast.OperationBitwiseOr:
			c.codeGen.Emit(opcode.InstructionBitwiseOr{})
		case ast.OperationBitwiseAnd:
			c.codeGen.Emit(opcode.InstructionBitwiseAnd{})
		case ast.OperationBitwiseXor:
			c.codeGen.Emit(opcode.InstructionBitwiseXor{})
		case ast.OperationBitwiseLeftShift:
			c.codeGen.Emit(opcode.InstructionBitwiseLeftShift{})
		case ast.OperationBitwiseRightShift:
			c.codeGen.Emit(opcode.InstructionBitwiseRightShift{})

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

func (c *Compiler[_, _]) VisitFunctionExpression(expression *ast.FunctionExpression) (_ struct{}) {
	// TODO: desugar function expressions

	functionIndex := len(c.functions)

	if functionIndex >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid function index"))
	}

	parameterCount := 0
	parameterList := expression.ParameterList
	if parameterList != nil {
		parameterCount = len(parameterList.Parameters)
	}

	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	functionType := c.DesugaredElaboration.FunctionExpressionFunctionType(expression)

	function := c.addFunction(
		"",
		"",
		uint16(parameterCount),
		functionType,
	)

	func() {
		previousFunction := c.currentFunction
		c.targetFunction(function)
		defer c.targetFunction(previousFunction)

		c.declareParameters(parameterList, false)
		c.compileFunctionBlock(
			expression.FunctionBlock,
			common.DeclarationKindUnknown,
		)
	}()

	c.emitNewClosure(uint16(functionIndex), function)

	return
}

func (c *Compiler[_, _]) VisitStringExpression(expression *ast.StringExpression) (_ struct{}) {
	c.emitStringConst(expression.Value)
	return
}

func (c *Compiler[_, _]) VisitStringTemplateExpression(_ *ast.StringTemplateExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitCastingExpression(expression *ast.CastingExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)

	castingTypes := c.DesugaredElaboration.CastingExpressionTypes(expression)
	index := c.getOrAddType(castingTypes.TargetType)

	var castInstruction opcode.Instruction
	switch expression.Operation {
	case ast.OperationCast:
		castInstruction = opcode.InstructionSimpleCast{
			Type: index,
		}
	case ast.OperationFailableCast:
		castInstruction = opcode.InstructionFailableCast{
			Type: index,
		}
	case ast.OperationForceCast:
		castInstruction = opcode.InstructionForceCast{
			Type: index,
		}
	default:
		panic(errors.NewUnreachableError())
	}

	c.codeGen.Emit(castInstruction)
	return
}

func (c *Compiler[_, _]) VisitCreateExpression(expression *ast.CreateExpression) (_ struct{}) {
	c.compileExpression(expression.InvocationExpression)
	return
}

func (c *Compiler[_, _]) VisitDestroyExpression(expression *ast.DestroyExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.codeGen.Emit(opcode.InstructionDestroy{})
	return
}

func (c *Compiler[_, _]) VisitReferenceExpression(expression *ast.ReferenceExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	borrowType := c.DesugaredElaboration.ReferenceExpressionBorrowType(expression)
	typeIndex := c.getOrAddType(borrowType)
	c.codeGen.Emit(opcode.InstructionNewRef{
		Type: typeIndex,
	})
	return
}

func (c *Compiler[_, _]) VisitForceExpression(expression *ast.ForceExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.codeGen.Emit(opcode.InstructionUnwrap{})
	return
}

func (c *Compiler[_, _]) VisitPathExpression(expression *ast.PathExpression) (_ struct{}) {
	domain := common.PathDomainFromIdentifier(expression.Domain.Identifier)
	identifier := expression.Identifier.Identifier
	if len(identifier) >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid identifier"))
	}

	identifierConst := c.addStringConst(identifier)
	identifierIndex := identifierConst.index

	c.codeGen.Emit(
		opcode.InstructionNewPath{
			Domain:     domain,
			Identifier: identifierIndex,
		},
	)

	return
}

func (c *Compiler[_, _]) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) (_ struct{}) {
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

func (c *Compiler[_, _]) compileInitializer(declaration *ast.SpecialFunctionDeclaration) {
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

	functionType := c.DesugaredElaboration.FunctionDeclarationFunctionType(declaration.FunctionDeclaration)

	function := c.addFunction(
		functionName,
		functionName,
		uint16(parameterCount),
		functionType,
	)

	previousFunction := c.currentFunction
	c.targetFunction(function)
	defer c.targetFunction(previousFunction)

	c.declareParameters(parameterList, false)

	// Declare `self`
	self := c.currentFunction.declareLocal(sema.SelfIdentifier)

	// Initialize an empty struct and assign to `self`.
	// i.e: `self = New()`

	// Write composite kind
	// TODO: Maybe get/include this from static-type. Then no need to provide separately.

	typeIndex := c.getOrAddType(enclosingType)

	c.codeGen.Emit(
		opcode.InstructionNew{
			Kind: kind,
			Type: typeIndex,
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

		c.codeGen.Emit(opcode.InstructionSetGlobal{
			Global: global.Index,
		})
	}

	c.emitSetLocal(self.index)

	// emit for the statements in `init()` body.
	c.compileFunctionBlock(
		declaration.FunctionDeclaration.FunctionBlock,
		declaration.Kind,
	)

	// Constructor should return the created the struct. i.e: return `self`
	c.emitGetLocal(self.index)
	c.codeGen.Emit(opcode.InstructionReturnValue{})
}

func (c *Compiler[E, _]) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, _ bool) (_ struct{}) {
	previousFunction := c.currentFunction

	var (
		parameterCount  int
		declareReceiver bool
		functionName    string
	)

	paramList := declaration.ParameterList
	if paramList != nil {
		parameterCount = len(paramList.Parameters)
	}

	identifier := declaration.Identifier.Identifier

	var innerFunctionLocal *local

	if previousFunction == nil {
		// Global function or method

		declareReceiver = !c.compositeTypeStack.isEmpty()

		functionName = commons.TypeQualifiedName(
			c.enclosingCompositeTypeFullyQualifiedName(),
			identifier,
		)

		if declareReceiver {
			parameterCount++
		}

	} else {
		innerFunctionLocal = c.currentFunction.declareLocal(identifier)
	}

	if parameterCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	functionIndex := len(c.functions)

	if functionIndex >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid function index"))
	}

	functionType := c.DesugaredElaboration.FunctionDeclarationFunctionType(declaration)

	function := c.addFunction(
		functionName,
		functionName,
		uint16(parameterCount),
		functionType,
	)

	func() {
		c.targetFunction(function)
		defer c.targetFunction(previousFunction)

		c.declareParameters(declaration.ParameterList, declareReceiver)

		c.compileFunctionBlock(
			declaration.FunctionBlock,
			declaration.DeclarationKind(),
		)
	}()

	if previousFunction != nil {
		// Inner function

		c.emitNewClosure(uint16(functionIndex), function)

		c.emitSetLocal(innerFunctionLocal.index)
	}

	return
}

func (c *Compiler[_, _]) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) (_ struct{}) {
	compositeType := c.DesugaredElaboration.CompositeDeclarationType(declaration)
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

	// Add the methods that are provided natively.
	c.addBuiltinMethods()

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

func (c *Compiler[_, _]) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) (_ struct{}) {
	interfaceType := c.DesugaredElaboration.InterfaceDeclarationType(declaration)
	c.compositeTypeStack.push(interfaceType)
	defer func() {
		c.compositeTypeStack.pop()
	}()

	// Add the methods that are provided natively.
	c.addBuiltinMethods()

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

func (c *Compiler[_, _]) addBuiltinMethods() {
	for _, boundFunction := range commonBuiltinTypeBoundFunctions {
		name := boundFunction.name
		qualifiedName := commons.TypeQualifiedName(
			c.enclosingCompositeTypeFullyQualifiedName(),
			name,
		)
		c.addFunction(
			name,
			qualifiedName,
			uint16(len(boundFunction.typ.Parameters)+1),
			boundFunction.typ,
		)
	}
}

func (c *Compiler[_, _]) VisitFieldDeclaration(_ *ast.FieldDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitImportDeclaration(declaration *ast.ImportDeclaration) (_ struct{}) {
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
			name := function.QualifiedName

			// Skip the contract initializer.
			// It should never be able to invoked within the code.
			if isContract && name == commons.InitFunctionName {
				continue
			}

			// TODO: Filter-in only public functions
			c.addImportedGlobal(location.Location, function.QualifiedName)
		}
	}

	return
}

func (c *Compiler[_, _]) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitAttachmentDeclaration(_ *ast.AttachmentDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitEntitlementDeclaration(_ *ast.EntitlementDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitEntitlementMappingDeclaration(_ *ast.EntitlementMappingDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitRemoveStatement(_ *ast.RemoveStatement) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitAttachExpression(_ *ast.AttachExpression) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) emitTransfer(targetType sema.Type) {

	// Optimization: We can omit the transfer in some cases
	switch lastInstruction := c.codeGen.LastInstruction().(type) {
	case opcode.InstructionGetConstant:
		// If the last instruction is a constant load of the same type,
		// then the transfer is not needed.
		targetConstantKind := constant.FromSemaType(targetType)
		constantIndex := lastInstruction.Constant
		c := c.constants[constantIndex]
		if c.kind == targetConstantKind {
			return
		}

	case opcode.InstructionNewPath:
		// If the last instruction is a path creation of the same type,
		// then the transfer is not needed.
		switch lastInstruction.Domain {
		case common.PathDomainPublic:
			if targetType == sema.PublicPathType {
				return
			}

		case common.PathDomainStorage:
			if targetType == sema.StoragePathType {
				return
			}
		}
	}

	typeIndex := c.getOrAddType(targetType)
	c.codeGen.Emit(opcode.InstructionTransfer{
		Type: typeIndex,
	})
}

func (c *Compiler[_, T]) getOrAddType(targetType sema.Type) uint16 {
	typeID := targetType.ID()

	// Optimization: Re-use types in the pool.
	index, ok := c.typesInPool[typeID]

	if !ok {
		staticType := interpreter.ConvertSemaToStaticType(c.memoryGauge, targetType)
		typ := c.typeGen.CompileType(staticType)
		index = c.addType(typ)
		c.typesInPool[typeID] = index
	}

	return index
}

func (c *Compiler[_, T]) addType(data T) uint16 {
	count := len(c.staticTypes)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid type declaration"))
	}

	c.staticTypes = append(c.staticTypes, data)
	return uint16(count)
}

func (c *Compiler[_, _]) enclosingCompositeTypeFullyQualifiedName() string {
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

func (c *Compiler[E, T]) declareParameters(paramList *ast.ParameterList, declareReceiver bool) {
	if declareReceiver {
		// Declare receiver as `self`.
		// Receiver is always at the zero-th index of params.
		c.currentFunction.declareLocal(sema.SelfIdentifier)
	}

	if paramList != nil {
		for _, parameter := range paramList.Parameters {
			parameterName := parameter.Identifier.Identifier
			c.currentFunction.declareLocal(parameterName)
		}
	}
}

func (c *Compiler[_, _]) generateEmptyInit() {
	c.DesugaredElaboration.SetFunctionDeclarationFunctionType(
		emptyInitializer.FunctionDeclaration,
		emptyInitializerFuncType,
	)
	c.VisitSpecialFunctionDeclaration(emptyInitializer)
}

func (c *Compiler[_, _]) withConditionElaboration(statement ast.Statement, f func()) {
	stmtElaboration, ok := c.DesugaredElaboration.conditionsElaborations[statement]
	if ok {
		prevElaboration := c.DesugaredElaboration
		c.DesugaredElaboration = stmtElaboration
		defer func() {
			c.DesugaredElaboration = prevElaboration
		}()
	}
	f()
}

func (c *Compiler[E, _]) emitNewClosure(functionIndex uint16, function *function[E]) {
	c.codeGen.Emit(opcode.InstructionNewClosure{
		Function: functionIndex,
		Upvalues: function.upvalues,
	})
}
