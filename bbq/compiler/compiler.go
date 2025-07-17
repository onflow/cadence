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

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/constant"
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
	location             common.Location

	currentFunction *function[E]

	compositeTypeStack *Stack[sema.CompositeKindedType]

	functions          []*function[E]
	globalVariables    []*globalVariable[E]
	constants          []*Constant
	Globals            map[string]*Global
	globalImports      *activations.Activation[GlobalImport]
	importedGlobals    []*Global
	controlFlows       []controlFlow
	currentControlFlow *controlFlow
	returns            []returns
	currentReturn      *returns

	types         []sema.Type
	compiledTypes []T

	// postConditionsIndices keeps track of where the post conditions start (i.e: index of the statement in the block),
	// for each function.
	// This mapping is populated by/during the desugar/rewrite: When the post conditions gets added
	// to the end of the function block, it keeps track of the index where it was added to.
	// Then the compiler uses these indices to patch the jumps for return statements.
	postConditionsIndices map[*ast.FunctionBlock]int

	// inheritedConditionParamBindings keeps a mapping between the parameter names
	// of an interface-function and the parameter names of its-implementation,
	// for each inherited condition.
	inheritedConditionParamBindings       map[ast.Statement]map[string]string
	currentInheritedConditionParamBinding map[string]string

	// postConditionsIndex is the statement-index of the post-conditions for the current function.
	postConditionsIndex int

	lastChangedPosition bbq.Position
	currentPosition     bbq.Position

	// Cache alike for compiledTypes and constants in the pool.
	typesInPool     map[sema.TypeID]uint16
	constantsInPool map[constantsCacheKey]*Constant

	codeGen CodeGen[E]
	typeGen TypeGen[T]

	// desugar is used to desugar the AST declarations, before the compiling starts.
	// This could be also reused during compilation to desugar expressions and statements.
	// Important: It must NOT be reused to desugar any top-level declaration, after the initial use.
	desugar *Desugar
}

type constantsCacheKey struct {
	data string
	kind constant.Kind
}

var _ ast.DeclarationVisitor[struct{}] = &Compiler[any, any]{}
var _ ast.StatementVisitor[struct{}] = &Compiler[any, any]{}
var _ ast.ExpressionVisitor[struct{}] = &Compiler[any, any]{}

func NewBytecodeCompiler(
	program *interpreter.Program,
	location common.Location,
	config *Config,
) *Compiler[byte, []byte] {
	return newCompiler(
		program,
		location,
		config,
		&ByteCodeGen{},
		&EncodedTypeGen{},
	)
}

func NewInstructionCompiler(
	program *interpreter.Program,
	location common.Location,
) *Compiler[opcode.Instruction, bbq.StaticType] {
	return NewInstructionCompilerWithConfig(
		program,
		location,
		&Config{},
	)
}

func NewInstructionCompilerWithConfig(
	program *interpreter.Program,
	location common.Location,
	config *Config,
) *Compiler[opcode.Instruction, bbq.StaticType] {
	return newCompiler(
		program,
		location,
		config,
		&InstructionCodeGen{},
		&DecodedTypeGen{},
	)
}

type GlobalImport struct {
	Location common.Location
	Name     string
}

func newCompiler[E, T any](
	program *interpreter.Program,
	location common.Location,
	config *Config,
	codeGen CodeGen[E],
	typeGen TypeGen[T],
) *Compiler[E, T] {

	var globalImports *activations.Activation[GlobalImport]
	if config.BuiltinGlobalsProvider != nil {
		globalImports = config.BuiltinGlobalsProvider()
	} else {
		globalImports = DefaultBuiltinGlobals()
	}
	globalImports = activations.NewActivation(config.MemoryGauge, globalImports)

	common.UseMemory(config.MemoryGauge, common.CompilerMemoryUsage)

	return &Compiler[E, T]{
		Program:              program.Program,
		DesugaredElaboration: NewDesugaredElaboration(program.Elaboration),
		Config:               config,
		location:             location,
		Globals:              make(map[string]*Global),
		globalImports:        globalImports,
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
		typeQualifiedName := commons.TypeQualifiedName(enclosingContract, name)
		global, ok = c.Globals[typeQualifiedName]
		if ok {
			return global
		}
	}

	importedGlobal := c.globalImports.Find(name)
	if importedGlobal == (GlobalImport{}) {
		panic(errors.NewUnexpectedError("cannot find global declaration '%s'", name))
	}
	if importedGlobal.Name != name {
		panic(errors.NewUnexpectedError(
			"imported global %q does not match the expected name %q",
			importedGlobal.Name,
			name,
		))
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
	global = NewGlobal(
		c.Config.MemoryGauge,
		name,
		importedGlobal.Location,
		uint16(count),
	)
	c.Globals[name] = global

	// Also add it to the usedImportedGlobals.
	// This is later used to export the imports, which is eventually used by the linker.
	// Linker will link the imports in the same order as they are added here.
	// i.e: same order as their indexes (preceded by globals defined in the current program).
	// e.g: [global1, global2, ... [importedGlobal1, importedGlobal2, ...]].
	// Earlier we already reserved the indexes for the globals defined in the current program.
	// (`reserveGlobals`)

	c.importedGlobals = append(c.importedGlobals, global)

	return global
}

func (c *Compiler[_, _]) addGlobal(name string) *Global {
	count := len(c.Globals)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid global declaration"))
	}

	global := NewGlobal(
		c.Config.MemoryGauge,
		name,
		nil,
		uint16(count),
	)
	c.Globals[name] = global
	return global
}

func (c *Compiler[_, _]) addImportedGlobal(location common.Location, name string) {
	existing := c.globalImports.Find(name)
	if existing != (GlobalImport{}) {
		return
	}
	c.globalImports.Set(
		name,
		GlobalImport{
			Location: location,
			Name:     name,
		},
	)
}

func (c *Compiler[E, T]) addFunction(
	name string,
	qualifiedName string,
	parameterCount uint16,
	functionType *sema.FunctionType,
) *function[E] {
	function := c.newFunction(
		name,
		qualifiedName,
		parameterCount,
		functionType,
	)
	c.functions = append(c.functions, function)
	return function
}

func (c *Compiler[E, T]) addGlobalVariableWithGetter(
	name string,
	functionType *sema.FunctionType,
) *globalVariable[E] {
	function := c.newFunction(
		name,
		name,
		0,
		functionType,
	)

	globalVariable := &globalVariable[E]{
		Name:   name,
		Getter: function,
	}

	c.globalVariables = append(c.globalVariables, globalVariable)

	return globalVariable
}

func (c *Compiler[E, T]) addGlobalVariable(
	name string,
) *globalVariable[E] {
	globalVariable := &globalVariable[E]{
		Name: name,
	}

	c.globalVariables = append(c.globalVariables, globalVariable)

	return globalVariable
}

func (c *Compiler[E, T]) newFunction(
	name string,
	qualifiedName string,
	parameterCount uint16,
	functionType *sema.FunctionType,
) *function[E] {
	functionTypeIndex := c.getOrAddType(functionType)

	return newFunction[E](
		c.currentFunction,
		name,
		qualifiedName,
		parameterCount,
		functionTypeIndex,
	)
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

	constant := NewConstant(
		c.Config.MemoryGauge,
		uint16(count),
		kind,
		data,
	)
	c.constants = append(c.constants, constant)
	c.constantsInPool[cacheKey] = constant
	return constant
}

func (c *Compiler[_, _]) emitGetConstant(constant *Constant) {
	c.emit(opcode.InstructionGetConstant{
		Constant: constant.index,
	})
}

func (c *Compiler[_, _]) emitStringConst(str string) {
	c.emitGetConstant(c.addStringConst(str))
}

func (c *Compiler[_, _]) addStringConst(str string) *Constant {
	return c.addConstant(constant.String, []byte(str))
}

func (c *Compiler[_, _]) emitCharacterConst(str string) {
	c.emitGetConstant(c.addCharacterConst(str))
}

func (c *Compiler[_, _]) addCharacterConst(str string) *Constant {
	return c.addConstant(constant.Character, []byte(str))
}

func (c *Compiler[_, _]) emitIntConst(i int64) {
	c.emitGetConstant(c.addIntConst(i))
}

func (c *Compiler[_, _]) addIntConst(i int64) *Constant {
	// NOTE: also adjust VisitIntegerExpression!
	data := interpreter.NewUnmeteredIntValueFromInt64(i).ToBigEndianBytes()
	return c.addConstant(constant.Int, data)
}

func (c *Compiler[_, _]) emitJump(target int) int {
	if target >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	offset := c.codeGen.Offset()
	c.emit(opcode.InstructionJump{Target: uint16(target)})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJump() int {
	offset := c.codeGen.Offset()
	c.emit(opcode.InstructionJump{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfFalse() int {
	offset := c.codeGen.Offset()
	c.emit(opcode.InstructionJumpIfFalse{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfTrue() int {
	offset := c.codeGen.Offset()
	c.emit(opcode.InstructionJumpIfTrue{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) emitUndefinedJumpIfNil() int {
	offset := c.codeGen.Offset()
	c.emit(opcode.InstructionJumpIfNil{Target: math.MaxUint16})
	return offset
}

func (c *Compiler[_, _]) patchJumpHere(jumpOpcodeOffset int) {
	c.patchJump(jumpOpcodeOffset, c.codeGen.Offset())
}

func (c *Compiler[_, _]) patchJump(jumpOpcodeOffset int, targetInstructionOffset int) {
	if targetInstructionOffset == 0 {
		panic(errors.NewUnreachableError())
	}
	if targetInstructionOffset >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid jump"))
	}
	c.codeGen.PatchJump(
		jumpOpcodeOffset,
		uint16(targetInstructionOffset),
	)
}

func (c *Compiler[_, _]) pushControlFlow(start int) *controlFlow {
	index := len(c.controlFlows)
	c.controlFlows = append(
		c.controlFlows,
		controlFlow{
			start: start,
		},
	)
	current := &c.controlFlows[index]
	c.currentControlFlow = current
	return current
}

func (c *Compiler[_, _]) popControlFlow(endOffset int) {
	lastIndex := len(c.controlFlows) - 1
	l := c.controlFlows[lastIndex]
	c.controlFlows[lastIndex] = controlFlow{}
	c.controlFlows = c.controlFlows[:lastIndex]

	for _, breakOffset := range l.breaks {
		c.patchJump(breakOffset, endOffset)
	}

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

func (c *Compiler[_, _]) compileDeclaration(declaration ast.Declaration) {
	c.compileWithPositionInfo(
		declaration,
		func() {
			ast.AcceptDeclaration[struct{}](declaration, c)
		},
	)
}

func (c *Compiler[_, _]) compileStatement(statement ast.Statement) {
	c.compileWithPositionInfo(
		statement,
		func() {
			c.emit(opcode.InstructionStatement{})
			ast.AcceptStatement[struct{}](statement, c)
		},
	)
}

func (c *Compiler[_, _]) compileExpression(expression ast.Expression) {
	c.compileWithPositionInfo(
		expression,
		func() {
			ast.AcceptExpression[struct{}](expression, c)
		},
	)
}

func (c *Compiler[_, _]) compileWithPositionInfo(
	hasPosition ast.HasPosition,
	compile func(),
) {
	prevCurrentPosition := c.currentPosition
	c.currentPosition = bbq.Position{
		StartPos: hasPosition.StartPosition(),
		EndPos:   hasPosition.EndPosition(c.Config.MemoryGauge),
	}

	defer func() {
		c.currentPosition = prevCurrentPosition
	}()

	compile()
}

func (c *Compiler[E, T]) Compile() *bbq.Program[E, T] {

	// Desugar the program before compiling.
	c.desugar = NewDesugar(
		c.Config.MemoryGauge,
		c.Config,
		c.Program,
		c.DesugaredElaboration,
		c.location,
	)

	desugaredProgram := c.desugar.Run()

	c.Program = desugaredProgram.program
	c.postConditionsIndices = desugaredProgram.postConditionIndices
	c.inheritedConditionParamBindings = desugaredProgram.inheritedConditionParamBinding

	for _, declaration := range c.Program.ImportDeclarations() {
		c.compileDeclaration(declaration)
	}

	contracts := c.exportContracts()

	compositeDeclarations := c.Program.CompositeDeclarations()
	variableDeclarations := c.Program.VariableDeclarations()
	functionDeclarations := c.Program.FunctionDeclarations()
	interfaceDeclarations := c.Program.InterfaceDeclarations()
	attachmentDeclarations := c.Program.AttachmentDeclarations()

	// Reserve globals for functions/types before visiting their implementations.
	c.reserveGlobals(
		contracts,
		variableDeclarations,
		functionDeclarations,
		compositeDeclarations,
		interfaceDeclarations,
		attachmentDeclarations,
	)

	// Compile declarations
	for _, declaration := range variableDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range functionDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range compositeDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range interfaceDeclarations {
		c.compileDeclaration(declaration)
	}
	for _, declaration := range attachmentDeclarations {
		c.compileDeclaration(declaration)
	}

	functions := c.exportFunctions()
	constants := c.exportConstants()
	types := c.exportTypes()
	imports := c.exportImports()
	variables := c.exportGlobalVariables()

	return &bbq.Program[E, T]{
		Functions: functions,
		Constants: constants,
		Types:     types,
		Imports:   imports,
		Contracts: contracts,
		Variables: variables,
	}
}

func (c *Compiler[_, _]) reserveGlobals(
	contract []*bbq.Contract,
	variableDecls []*ast.VariableDeclaration,
	functionDecls []*ast.FunctionDeclaration,
	compositeDecls []*ast.CompositeDeclaration,
	interfaceDecls []*ast.InterfaceDeclaration,
	attachmentDecls []*ast.AttachmentDeclaration,
) {
	// Reserve globals for the contract values before everything.
	// Contract values must be always start at the zero-th index.
	for _, contract := range contract {
		c.addGlobal(contract.Name)
	}

	c.reserveVariableGlobals(
		nil,
		variableDecls,
		nil,
		compositeDecls,
	)

	c.reserveFunctionGlobals(
		nil,
		nil,
		functionDecls,
		compositeDecls,
		interfaceDecls,
		attachmentDecls,
	)
}

func (c *Compiler[_, _]) reserveVariableGlobals(
	enclosingType sema.CompositeKindedType,
	variableDecls []*ast.VariableDeclaration,
	enumCaseDecls []*ast.EnumCaseDeclaration,
	compositeDecls []*ast.CompositeDeclaration,
) {
	for _, declaration := range variableDecls {
		variableName := declaration.Identifier.Identifier
		c.addGlobal(variableName)
	}

	for _, declaration := range enumCaseDecls {
		// Reserve a global variable for each enum case.
		// The enum case name is used as the global variable name.
		// e.g: `enum E: UInt8 { case A; case B }` will reserve globals `E.A`, `E.B`.
		enumCaseName := declaration.Identifier.Identifier
		qualifiedName := commons.TypeQualifiedName(enclosingType, enumCaseName)
		c.addGlobal(qualifiedName)
	}

	for _, declaration := range compositeDecls {
		compositeType := c.DesugaredElaboration.CompositeDeclarationType(declaration)

		members := declaration.Members

		c.reserveVariableGlobals(
			compositeType,
			nil,
			members.EnumCases(),
			members.Composites(),
		)
	}

	// attachments will not have nested enum/composite declarations
}

func (c *Compiler[_, _]) reserveFunctionGlobals(
	enclosingType sema.CompositeKindedType,
	specialFunctionDecls []*ast.SpecialFunctionDeclaration,
	functionDecls []*ast.FunctionDeclaration,
	compositeDecls []*ast.CompositeDeclaration,
	interfaceDecls []*ast.InterfaceDeclaration,
	attachmentDecls []*ast.AttachmentDeclaration,
) {
	for _, declaration := range specialFunctionDecls {
		switch declaration.Kind {
		case common.DeclarationKindDestructorLegacy,
			common.DeclarationKindPrepare:
			// Important: All special functions visited within `VisitSpecialFunctionDeclaration`
			// must be also visited here. And must be visited only them. e.g: Don't visit inits.
			functionName := declaration.FunctionDeclaration.Identifier.Identifier
			qualifiedName := commons.TypeQualifiedName(enclosingType, functionName)
			c.addGlobal(qualifiedName)
		}
	}

	// Add natively provided methods as globals.
	// Only do it for user-defined types (i.e: `compositeTypeName` is not empty).
	if enclosingType != nil {
		for _, boundFunction := range commonBuiltinTypeBoundFunctions {
			functionName := boundFunction.name
			qualifiedName := commons.TypeQualifiedName(enclosingType, functionName)
			c.addGlobal(qualifiedName)
		}

		if enclosingType.GetCompositeKind().SupportsAttachments() {
			functionName := sema.CompositeForEachAttachmentFunctionName
			qualifiedName := commons.TypeQualifiedName(enclosingType, functionName)
			c.addGlobal(qualifiedName)
		}
	}

	for _, declaration := range functionDecls {
		functionName := declaration.Identifier.Identifier
		qualifiedName := commons.TypeQualifiedName(enclosingType, functionName)
		c.addGlobal(qualifiedName)
	}

	for _, declaration := range compositeDecls {
		compositeType := c.DesugaredElaboration.CompositeDeclarationType(declaration)

		// Members of event types are skipped from compiling (see `VisitCompositeDeclaration`).
		// Hence also skip from reserving globals for them.
		if compositeType.Kind == common.CompositeKindEvent &&
			!declaration.IsResourceDestructionDefaultEvent() {
			continue
		}

		// Reserve a global for contract the constructor function.

		var constructorName string

		switch declaration.CompositeKind {
		case common.CompositeKindContract:
			// For contracts, a global with the type-name is used for the contract value
			// (already reserved in `reserveGlobals` before getting here).
			// Suffix the type-name.

			constructorName = commons.TypeQualifiedName(compositeType, commons.InitFunctionName)

		case common.CompositeKindEnum:
			// For enums, a global with the type-name is used for the "lookup function".
			// For example, for `enum E: UInt8 { case A; case B }`, the lookup function is `fun E(rawValue: UInt8): E?`.
			// Suffix the type-name.

			constructorName = commons.TypeQualifiedName(compositeType, commons.InitFunctionName)

		default:
			// For other composite types, the type-name is used for the constructor function.

			constructorName = commons.TypeQualifier(compositeType)
		}

		c.addGlobal(constructorName)

		if declaration.CompositeKind == common.CompositeKindEnum {
			// For enums, also reserve a global for the "lookup function".
			// For example, for `enum E: UInt8 { case A; case B }`, the lookup function is `fun E(rawValue: UInt8): E?`.
			functionName := commons.TypeQualifier(compositeType)
			c.addGlobal(functionName)
		}

		members := declaration.Members

		c.reserveFunctionGlobals(
			compositeType,
			members.SpecialFunctions(),
			members.Functions(),
			members.Composites(),
			members.Interfaces(),
			members.Attachments(),
		)
	}

	for _, declaration := range interfaceDecls {
		// Don't need a global for the value-constructor for interfaces

		members := declaration.Members
		interfaceType := c.DesugaredElaboration.InterfaceDeclarationType(declaration)

		c.reserveFunctionGlobals(
			interfaceType,
			members.SpecialFunctions(),
			members.Functions(),
			members.Composites(),
			members.Interfaces(),
			members.Attachments(),
		)
	}

	for _, declaration := range attachmentDecls {
		compositeType := c.DesugaredElaboration.CompositeDeclarationType(declaration)
		// Reserve a global for the constructor function.

		constructorName := commons.TypeQualifier(compositeType)
		c.addGlobal(constructorName)

		members := declaration.Members

		c.reserveFunctionGlobals(
			compositeType,
			members.SpecialFunctions(),
			members.Functions(),
			members.Composites(),
			members.Interfaces(),
			members.Attachments(),
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
	return c.compiledTypes
}

func (c *Compiler[_, _]) exportImports() []bbq.Import {
	var exportedImports []bbq.Import

	count := len(c.importedGlobals)
	if count > 0 {
		exportedImports = make([]bbq.Import, 0, count)
		for _, importedGlobal := range c.importedGlobals {
			bbqImport := bbq.Import{
				Location: importedGlobal.Location,
				Name:     importedGlobal.Name,
			}
			exportedImports = append(exportedImports, bbqImport)
		}
	}

	return exportedImports
}

func (c *Compiler[E, _]) exportFunctions() []bbq.Function[E] {
	var functions []bbq.Function[E]

	count := len(c.functions)
	if count > 0 {
		functions = make([]bbq.Function[E], 0, count)
		for _, function := range c.functions {
			functions = append(
				functions,
				c.newBBQFunction(function),
			)
		}
	}
	return functions
}

func (c *Compiler[E, _]) newBBQFunction(function *function[E]) bbq.Function[E] {
	return bbq.Function[E]{
		Name:           function.name,
		QualifiedName:  function.qualifiedName,
		Code:           function.code,
		LocalCount:     function.localCount,
		ParameterCount: function.parameterCount,
		TypeIndex:      function.typeIndex,
		LineNumbers:    function.lineNumbers,
	}
}

func (c *Compiler[E, _]) exportGlobalVariables() []bbq.Variable[E] {
	var globalVariables []bbq.Variable[E]

	count := len(c.globalVariables)
	if count > 0 {
		globalVariables = make([]bbq.Variable[E], 0, count)

		for _, variable := range c.globalVariables {
			var getter *bbq.Function[E]

			// Some globals variables may not have inlined initial values.
			// e.g: Transaction parameters are converted global variables,
			// where the values are being set in the transaction initializer.
			if variable.Getter != nil {
				function := c.newBBQFunction(variable.Getter)
				getter = &function
			}

			variable := bbq.Variable[E]{
				Name:   variable.Name,
				Getter: getter,
			}

			globalVariables = append(globalVariables, variable)
		}
	}

	return globalVariables
}

func (c *Compiler[_, _]) exportContracts() []*bbq.Contract {
	compositeDeclarations := c.Program.CompositeDeclarations()

	var contracts []*bbq.Contract
	if len(compositeDeclarations) == 0 {
		return contracts
	}
	contracts = make([]*bbq.Contract, 0, 1)

	for _, declaration := range compositeDeclarations {
		if declaration.Kind() != common.CompositeKindContract {
			continue
		}

		contractType := c.DesugaredElaboration.CompositeDeclarationType(declaration)
		location := contractType.GetLocation()
		name := contractType.GetIdentifier()

		var addressBytes []byte
		addressLocation, ok := location.(common.AddressLocation)
		if ok {
			addressBytes = addressLocation.Address.Bytes()
		}

		contracts = append(
			contracts,
			&bbq.Contract{
				Name:    name,
				Address: addressBytes,
			},
		)
	}

	return contracts
}

func (c *Compiler[_, _]) compileBlock(
	block *ast.Block,
	enclosingDeclKind common.DeclarationKind,
	returnType sema.Type,
) {
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
				offset := c.codeGen.Offset()
				for _, jumpOpcodeOffset := range c.currentReturn.returns {
					c.patchJump(jumpOpcodeOffset, offset)
				}
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
				c.emit(opcode.InstructionReturn{})
			} else {
				c.emitGetLocal(local.index)
				c.emitTransferAndConvertAndReturnValue(returnType)
			}
		} else if needsSyntheticReturn(block.Statements) {
			// If there are no post conditions,
			// and if there is no return statement at the end,
			// then emit an empty return.
			c.emit(opcode.InstructionReturn{})
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

func (c *Compiler[_, _]) compileFunctionBlock(
	functionBlock *ast.FunctionBlock,
	functionDeclKind common.DeclarationKind,
	returnType sema.Type,
) {
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

	c.compileBlock(
		functionBlock.Block,
		functionDeclKind,
		returnType,
	)
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
		// (1) Return with a value

		// TODO: copy
		c.compileExpression(expression)

		// End active iterators *after* the expression is compiled.
		c.emitActiveIteratorEnds()

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
				c.emit(opcode.InstructionDrop{})
			}

			// And jump to the start of the post conditions.
			offset := c.emitUndefinedJump()
			c.currentReturn.appendReturn(offset)
		} else {
			// (1.b)
			// If there are no post conditions, return then-and-there.
			returnTypes := c.DesugaredElaboration.ReturnStatementTypes(statement)
			c.emitTransferAndConvertAndReturnValue(returnTypes.ReturnType)
		}
	} else {
		// (2) Empty return

		c.emitActiveIteratorEnds()

		if c.hasPostConditions() {
			// (2.a)
			// If there are post conditions, jump to the start of the post conditions.
			offset := c.emitUndefinedJump()
			c.currentReturn.appendReturn(offset)
		} else {
			// (2.b)
			// If there are no post conditions, return then-and-there.
			c.emit(opcode.InstructionReturn{})
		}
	}

	return
}

func (c *Compiler[_, _]) emitActiveIteratorEnds() {
	for _, activeIteratorLocalIndex := range c.currentFunction.activeIteratorLocalIndices {
		c.emitIteratorEnd(activeIteratorLocalIndex)
	}
}

func (c *Compiler[_, _]) emitIteratorEnd(iteratorLocalIndex uint16) {
	c.emitGetLocal(iteratorLocalIndex)
	c.codeGen.Emit(opcode.InstructionIteratorEnd{})
}

func (c *Compiler[_, _]) emitGetLocal(localIndex uint16) {
	c.emit(opcode.InstructionGetLocal{
		Local: localIndex,
	})
}

func (c *Compiler[_, _]) emitSetLocal(localIndex uint16) {
	c.emit(opcode.InstructionSetLocal{
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
	c.emitContinue()
	return
}

func (c *Compiler[_, _]) emitContinue() {
	currentControlFlow := c.currentControlFlow

	preContinue := currentControlFlow.preContinue
	if preContinue != nil {
		preContinue()
	}

	start := currentControlFlow.start
	if start <= 0 {
		panic(errors.NewUnreachableError())
	}
	c.emitJump(start)
}

func (c *Compiler[_, _]) VisitIfStatement(statement *ast.IfStatement) (_ struct{}) {
	// If-statements can be coming from inherited conditions.
	// If so, use the corresponding elaboration.
	c.compilePotentiallyInheritedCode(statement, func() {
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
			c.emit(opcode.InstructionUnwrap{})
			varDeclTypes := c.DesugaredElaboration.VariableDeclarationTypes(test)
			c.emitTransferAndConvert(varDeclTypes.TargetType)

			// Before declaring the variable, evaluate and assign the second value.
			if test.SecondValue != nil {
				c.compileAssignment(
					test.Value,
					test.SecondValue,
					varDeclTypes.ValueType,
				)
			}

			// Declare the variable *after* unwrapping the optional,
			// in a new scope
			c.currentFunction.locals.PushNewWithCurrent()
			additionalThenScope = true
			name := test.Identifier.Identifier
			c.emitDeclareLocal(name)

		default:
			panic(errors.NewUnreachableError())
		}

		c.compileBlock(
			statement.Then,
			common.DeclarationKindUnknown,
			nil,
		)

		if additionalThenScope {
			c.currentFunction.locals.Pop()
		}

		elseBlock := statement.Else
		if elseBlock != nil {
			thenJump := c.emitUndefinedJump()
			c.patchJumpHere(elseJump)
			c.compileBlock(
				elseBlock,
				common.DeclarationKindUnknown,
				nil,
			)
			c.patchJumpHere(thenJump)
		} else {
			c.patchJumpHere(elseJump)
		}
	})
	return
}

func (c *Compiler[_, _]) VisitWhileStatement(statement *ast.WhileStatement) (_ struct{}) {
	testOffset := c.codeGen.Offset()

	c.pushControlFlow(testOffset)
	var endOffset int
	defer func() {
		c.popControlFlow(endOffset)
	}()

	c.compileExpression(statement.Test)
	endJump := c.emitUndefinedJumpIfFalse()

	// Compile the body

	c.emit(opcode.InstructionLoop{})

	c.compileBlock(
		statement.Block,
		common.DeclarationKindUnknown,
		nil,
	)
	// Repeat, jump back to the test
	c.emitJump(testOffset)

	// Patch the failed test to jump here
	endOffset = c.codeGen.Offset()
	c.patchJump(endJump, endOffset)

	return
}

func (c *Compiler[_, _]) VisitForStatement(statement *ast.ForStatement) (_ struct{}) {
	// Evaluate the expression
	c.compileExpression(statement.Value)

	// Get an iterator to the resulting value, and store it in a local index.
	c.emit(opcode.InstructionIterator{})
	iteratorLocalIndex := c.currentFunction.generateLocalIndex()
	c.emit(opcode.InstructionSetLocal{
		Local: iteratorLocalIndex,
	})
	// Push the iterator local index to the active iterators.
	c.currentFunction.activeIteratorLocalIndices = append(
		c.currentFunction.activeIteratorLocalIndices,
		iteratorLocalIndex,
	)

	// Initialize 'index' variable, if needed.
	index := statement.Index
	indexNeeded := index != nil
	var indexLocal *local

	if indexNeeded {
		// `var <index> = -1`
		// Start with -1 and then increment at the start of the loop,
		// so that we don't have to deal with early exists of the loop.
		c.emitIntConst(-1)
		indexLocal = c.emitDeclareLocal(index.Identifier)
	}

	entryLocal := c.currentFunction.declareLocal(statement.Identifier.Identifier)

	testOffset := c.codeGen.Offset()
	controlFlow := c.pushControlFlow(testOffset)
	controlFlow.preContinue = func() {
		// If the index local (if any) or the entry local are captured,
		// then we need to close the upvalue for them before continuing the loop.

		if indexLocal != nil && indexLocal.isCaptured {
			c.emitCloseUpvalue(indexLocal.index)
		}
		if entryLocal.isCaptured {
			c.emitCloseUpvalue(entryLocal.index)
		}
	}
	var endOffset int
	defer func() {
		c.popControlFlow(endOffset)
	}()

	// Loop test: Get the iterator and call `hasNext()`.
	c.emitGetLocal(iteratorLocalIndex)
	c.emit(opcode.InstructionIteratorHasNext{})

	endJump := c.emitUndefinedJumpIfFalse()

	// Compile the body

	c.emit(opcode.InstructionLoop{})

	// Increment the index if needed.
	// This is done as the first thing inside the loop, so that we don't need to
	// worry about loop-control statements (e.g: continue, return, break) in the body.
	if indexNeeded {
		// <index> = <index> + 1
		c.emitGetLocal(indexLocal.index)
		c.emitIntConst(1)
		c.emit(opcode.InstructionAdd{})
		c.emitSetLocal(indexLocal.index)
	}

	// Get the iterator and call `next()` (value for arrays, key for dictionaries, etc.)
	c.emitGetLocal(iteratorLocalIndex)
	c.emit(opcode.InstructionIteratorNext{})

	forStmtTypes := c.DesugaredElaboration.ForStatementType(statement)
	loopVarType := forStmtTypes.ValueVariableType
	_, isResultReference := sema.MaybeReferenceType(loopVarType)

	if isResultReference {
		index := c.getOrAddType(loopVarType)
		c.emit(opcode.InstructionNewRef{
			Type:       index,
			IsImplicit: true,
		})

		// If a reference is taken to the value, then do not transfer.
	} else {
		c.emitTransferAndConvert(loopVarType)
	}

	// Store it (next entry) in a local var.
	// `<entry> = iterator.next()`
	c.emitSetLocal(entryLocal.index)

	// Compile the for-loop body.
	c.compileBlock(
		statement.Block,
		common.DeclarationKindUnknown,
		nil,
	)

	// Jump back to the loop test. i.e: `hasNext()`.
	// Use a continue to ensure all instructions for the next loop iteration are generated (not just the jump).
	c.emitContinue()

	endOffset = c.codeGen.Offset()
	c.patchJump(endJump, endOffset)

	c.emitIteratorEnd(iteratorLocalIndex)

	// Pop the iterator local index from the active iterators.
	activeIteratorLocalIndices := c.currentFunction.activeIteratorLocalIndices
	c.currentFunction.activeIteratorLocalIndices = activeIteratorLocalIndices[:len(activeIteratorLocalIndices)-1]

	return
}

func (c *Compiler[_, _]) VisitEmitStatement(statement *ast.EmitStatement) (_ struct{}) {
	// Emit statements can be coming from inherited conditions.
	// If so, use the corresponding elaboration.
	c.compilePotentiallyInheritedCode(
		statement,
		func() {
			invocationExpression := statement.InvocationExpression
			arguments := invocationExpression.Arguments
			invocationTypes := c.DesugaredElaboration.InvocationExpressionTypes(invocationExpression)
			c.compileArguments(arguments, invocationTypes)

			argCount := len(arguments)
			if argCount >= math.MaxUint16 {
				panic(errors.NewDefaultUserError("invalid argument count"))
			}

			eventType := c.DesugaredElaboration.EmitStatementEventType(statement)
			typeIndex := c.getOrAddType(eventType)

			c.emit(opcode.InstructionEmitEvent{
				Type:     typeIndex,
				ArgCount: uint16(argCount),
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
	var endOffset int
	defer func() {
		c.popControlFlow(endOffset)
	}()

	previousJump := -1

	for _, switchCase := range statement.Cases {
		if previousJump >= 0 {
			c.patchJumpHere(previousJump)
			previousJump = -1
		}

		isDefault := switchCase.Expression == nil
		if !isDefault {
			c.emitGetLocal(localIndex)
			c.compileExpression(switchCase.Expression)
			c.emit(opcode.InstructionEqual{})
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

	endOffset = c.codeGen.Offset()

	if previousJump >= 0 {
		c.patchJump(previousJump, endOffset)
	}

	return
}

func (c *Compiler[_, _]) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {

	variableName := declaration.Identifier.Identifier

	isGlobalVar := c.currentFunction == nil
	if isGlobalVar {
		c.compileGlobalVariable(declaration, variableName)
		return
	}

	// Some variable declarations can be coming from inherited before-statements.
	// If so, use the corresponding elaboration.
	c.compilePotentiallyInheritedCode(declaration, func() {

		name := declaration.Identifier.Identifier

		// Value can be nil only for synthetic-result variable.
		if declaration.Value == nil {
			c.currentFunction.declareLocal(name)
			return
		}

		// Compile the value expression *before* declaring the variable.
		c.compileExpression(declaration.Value)

		varDeclTypes := c.DesugaredElaboration.VariableDeclarationTypes(declaration)
		c.emitTransferAndConvert(varDeclTypes.TargetType)

		// Before declaring the variable, evaluate and assign the second value.
		if declaration.SecondValue != nil {
			c.compileAssignment(
				declaration.Value,
				declaration.SecondValue,
				varDeclTypes.ValueType,
			)
		}

		// Declare the variable *after* compiling the value expressions.
		c.emitDeclareLocal(name)
	})

	return
}

func (c *Compiler[_, _]) compileGlobalVariable(declaration *ast.VariableDeclaration, variableName string) {
	if declaration.Value == nil {
		c.addGlobalVariable(variableName)
		return
	}

	varDeclTypes := c.DesugaredElaboration.VariableDeclarationTypes(declaration)

	variableGetterFunctionType := sema.NewSimpleFunctionType(
		sema.FunctionPurityImpure,
		nil,
		sema.NewTypeAnnotation(varDeclTypes.TargetType),
	)

	globalVariable := c.addGlobalVariableWithGetter(
		variableName,
		variableGetterFunctionType,
	)

	func() {
		previousFunction := c.currentFunction
		c.targetFunction(globalVariable.Getter)
		defer c.targetFunction(previousFunction)

		// No parameters

		// Compile function body
		c.compileExpression(declaration.Value)
		c.emitTransferAndConvertAndReturnValue(varDeclTypes.TargetType)
	}()
}

func (c *Compiler[_, _]) emitTransferAndConvertAndReturnValue(returnType sema.Type) {
	c.emitTransferAndConvert(returnType)
	c.emit(opcode.InstructionReturnValue{})
}

func (c *Compiler[_, _]) emitDeclareLocal(name string) *local {
	local := c.currentFunction.declareLocal(name)
	c.emitSetLocal(local.index)
	return local
}

func (c *Compiler[_, _]) VisitAssignmentStatement(statement *ast.AssignmentStatement) (_ struct{}) {
	assignmentTypes := c.DesugaredElaboration.AssignmentStatementTypes(statement)
	c.compileAssignment(
		statement.Target,
		statement.Value,
		assignmentTypes.TargetType,
	)
	return
}

func (c *Compiler[_, _]) compileAssignment(
	target ast.Expression,
	value ast.Expression,
	targetType sema.Type,
) {
	switch target := target.(type) {
	case *ast.IdentifierExpression:
		c.compileExpression(value)
		c.emitTransferAndConvert(targetType)
		c.emitVariableStore(target.Identifier.Identifier)

	case *ast.MemberExpression:
		c.compileExpression(target.Expression)
		c.compileExpression(value)
		c.emitTransferAndConvert(targetType)
		constant := c.addStringConst(target.Identifier.Identifier)

		memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(target)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		memberAccessedTypeIndex := c.getOrAddType(memberAccessInfo.AccessedType)

		c.emit(opcode.InstructionSetField{
			FieldName:    constant.index,
			AccessedType: memberAccessedTypeIndex,
		})

	case *ast.IndexExpression:
		c.compileExpression(target.TargetExpression)

		c.compileExpression(target.IndexingExpression)
		c.emitIndexKeyTransferAndConvert(target)

		c.compileExpression(value)
		c.emitTransferAndConvert(targetType)

		c.emit(opcode.InstructionSetIndex{})

	default:
		panic(errors.NewUnreachableError())
	}
}

func (c *Compiler[_, _]) VisitSwapStatement(statement *ast.SwapStatement) (_ struct{}) {

	// Get type information

	swapStatementTypes := c.DesugaredElaboration.SwapStatementTypes(statement)
	leftType := swapStatementTypes.LeftType
	rightType := swapStatementTypes.RightType

	// Evaluate the left side (target and key)

	leftTargetIndex := c.compileSwapTarget(statement.Left)
	leftKeyIndex := c.compileSwapKey(statement.Left)

	// Evaluate the right side (target and key)

	rightTargetIndex := c.compileSwapTarget(statement.Right)
	rightKeyIndex := c.compileSwapKey(statement.Right)

	// Get left and right values

	leftValueIndex := c.compileSwapGet(
		statement.Left,
		leftTargetIndex,
		leftKeyIndex,
		rightType,
	)
	rightValueIndex := c.compileSwapGet(
		statement.Right,
		rightTargetIndex,
		rightKeyIndex,
		leftType,
	)

	// Set right value to left target,
	// and left value to right target

	// TODO: invalidation?

	c.compileSwapSet(
		statement.Left,
		leftTargetIndex,
		leftKeyIndex,
		rightValueIndex,
	)
	c.compileSwapSet(
		statement.Right,
		rightTargetIndex,
		rightKeyIndex,
		leftValueIndex,
	)

	return
}

func (c *Compiler[_, _]) compileSwapTarget(sideExpression ast.Expression) (targetLocalIndex uint16) {
	switch sideExpression := sideExpression.(type) {
	case *ast.IdentifierExpression:
		c.compileExpression(sideExpression)
	case *ast.MemberExpression:
		c.compileExpression(sideExpression.Expression)
	case *ast.IndexExpression:
		c.compileExpression(sideExpression.TargetExpression)
	default:
		panic(errors.NewUnreachableError())
	}

	targetLocalIndex = c.currentFunction.generateLocalIndex()
	c.emitSetLocal(targetLocalIndex)

	return
}

func (c *Compiler[_, _]) compileSwapKey(sideExpression ast.Expression) (keyLocalIndex uint16) {
	switch sideExpression := sideExpression.(type) {
	case *ast.IdentifierExpression, *ast.MemberExpression:
		// No key expression for identifier and member expressions
		return 0

	case *ast.IndexExpression:
		// If the side is an index expression, compile the indexing expression
		c.compileExpression(sideExpression.IndexingExpression)

	default:
		panic(errors.NewUnreachableError())
	}

	keyLocalIndex = c.currentFunction.generateLocalIndex()
	c.emitSetLocal(keyLocalIndex)

	return
}

func (c *Compiler[_, _]) compileSwapGet(
	sideExpression ast.Expression,
	targetIndex uint16,
	keyIndex uint16,
	targetType sema.Type,
) (valueIndex uint16) {

	switch sideExpression := sideExpression.(type) {
	case *ast.IdentifierExpression:
		c.emitGetLocal(targetIndex)

	case *ast.MemberExpression:
		memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(sideExpression)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		if memberAccessInfo.IsOptional {
			panic(errors.NewUnexpectedError("optional member access is not supported in swap statements"))
		}

		c.emitGetLocal(targetIndex)
		c.compileMemberAccess(sideExpression)

	case *ast.IndexExpression:
		c.emitGetLocal(targetIndex)
		c.emitGetLocal(keyIndex)
		c.compileIndexAccess(sideExpression)

	default:
		panic(errors.NewUnreachableError())
	}

	c.emitTransferAndConvert(targetType)

	valueIndex = c.currentFunction.generateLocalIndex()
	c.emitSetLocal(valueIndex)

	return
}

func (c *Compiler[_, _]) compileSwapSet(
	sideExpression ast.Expression,
	targetIndex uint16,
	keyIndex uint16,
	valueIndex uint16,
) {
	switch sideExpression := sideExpression.(type) {
	case *ast.IdentifierExpression:
		c.emitGetLocal(valueIndex)
		// NOTE: Assign to the original target. Do NOT use targetIndex here, because it is a temporary.
		name := sideExpression.Identifier.Identifier
		c.emitVariableStore(name)

	case *ast.MemberExpression:
		c.emitGetLocal(targetIndex)
		c.emitGetLocal(valueIndex)

		name := sideExpression.Identifier.Identifier
		constant := c.addStringConst(name)

		memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(sideExpression)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		memberAccessedTypeIndex := c.getOrAddType(memberAccessInfo.AccessedType)

		c.emit(opcode.InstructionSetField{
			FieldName:    constant.index,
			AccessedType: memberAccessedTypeIndex,
		})

	case *ast.IndexExpression:
		c.emitGetLocal(targetIndex)
		c.emitGetLocal(keyIndex)
		c.emitIndexKeyTransferAndConvert(sideExpression)
		c.emitGetLocal(valueIndex)
		c.emit(opcode.InstructionSetIndex{})

	default:
		panic(errors.NewUnreachableError())
	}
}

func (c *Compiler[_, _]) emitIndexKeyTransferAndConvert(indexExpression *ast.IndexExpression) {
	indexExpressionTypes, ok := c.DesugaredElaboration.IndexExpressionTypes(indexExpression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	indexedType := indexExpressionTypes.IndexedType
	c.emitTransferAndConvert(indexedType.IndexingType())
}

func (c *Compiler[_, _]) VisitExpressionStatement(statement *ast.ExpressionStatement) (_ struct{}) {
	c.compileExpression(statement.Expression)

	switch statement.Expression.(type) {
	case *ast.DestroyExpression:
		// Do nothing. Destroy operation will not produce any result.
	default:
		// Otherwise, drop the expression evaluation result.
		c.emit(opcode.InstructionDrop{})
	}

	return
}

func (c *Compiler[_, _]) VisitVoidExpression(_ *ast.VoidExpression) (_ struct{}) {
	c.emit(opcode.InstructionVoid{})
	return
}

func (c *Compiler[_, _]) VisitBoolExpression(expression *ast.BoolExpression) (_ struct{}) {
	if expression.Value {
		c.emit(opcode.InstructionTrue{})
	} else {
		c.emit(opcode.InstructionFalse{})
	}
	return
}

func (c *Compiler[_, _]) VisitNilExpression(_ *ast.NilExpression) (_ struct{}) {
	c.emit(opcode.InstructionNil{})
	return
}

func (c *Compiler[_, _]) VisitIntegerExpression(expression *ast.IntegerExpression) (_ struct{}) {
	value := expression.Value
	integerType := c.DesugaredElaboration.IntegerExpressionType(expression)
	c.emitIntegerConstant(value, integerType)

	return
}

func (c *Compiler[_, _]) emitIntegerConstant(value *big.Int, integerType sema.Type) {
	constantKind := constant.FromSemaType(integerType)

	var data []byte

	switch constantKind {
	case constant.Int:
		// NOTE: also adjust addIntConst!
		data = interpreter.NewUnmeteredIntValueFromBigInt(value).ToBigEndianBytes()

	case constant.Int8:
		data = interpreter.NewUnmeteredInt8Value(int8(value.Int64())).ToBigEndianBytes()

	case constant.Int16:
		data = interpreter.NewUnmeteredInt16Value(int16(value.Int64())).ToBigEndianBytes()

	case constant.Int32:
		data = interpreter.NewUnmeteredInt32Value(int32(value.Int64())).ToBigEndianBytes()

	case constant.Int64:
		data = interpreter.NewUnmeteredInt64Value(value.Int64()).ToBigEndianBytes()

	case constant.Int128:
		data = interpreter.NewUnmeteredInt128ValueFromBigInt(value).ToBigEndianBytes()

	case constant.Int256:
		data = interpreter.NewUnmeteredInt256ValueFromBigInt(value).ToBigEndianBytes()

	case constant.UInt:
		data = interpreter.NewUnmeteredUIntValueFromBigInt(value).ToBigEndianBytes()

	case constant.UInt8:
		data = interpreter.NewUnmeteredUInt8Value(uint8(value.Uint64())).ToBigEndianBytes()

	case constant.UInt16:
		data = interpreter.NewUnmeteredUInt16Value(uint16(value.Uint64())).ToBigEndianBytes()

	case constant.UInt32:
		data = interpreter.NewUnmeteredUInt32Value(uint32(value.Uint64())).ToBigEndianBytes()

	case constant.UInt64:
		data = interpreter.NewUnmeteredUInt64Value(value.Uint64()).ToBigEndianBytes()

	case constant.UInt128:
		data = interpreter.NewUnmeteredUInt128ValueFromBigInt(value).ToBigEndianBytes()

	case constant.UInt256:
		data = interpreter.NewUnmeteredUInt256ValueFromBigInt(value).ToBigEndianBytes()

	case constant.Word8:
		data = interpreter.NewUnmeteredWord8Value(uint8(value.Uint64())).ToBigEndianBytes()

	case constant.Word16:
		data = interpreter.NewUnmeteredWord16Value(uint16(value.Uint64())).ToBigEndianBytes()

	case constant.Word32:
		data = interpreter.NewUnmeteredWord32Value(uint32(value.Uint64())).ToBigEndianBytes()

	case constant.Word64:
		data = interpreter.NewUnmeteredWord64Value(value.Uint64()).ToBigEndianBytes()

	case constant.Word128:
		data = interpreter.NewUnmeteredWord128ValueFromBigInt(value).ToBigEndianBytes()

	case constant.Word256:
		data = interpreter.NewUnmeteredWord256ValueFromBigInt(value).ToBigEndianBytes()

	case constant.Address:
		data = value.Bytes()

	default:
		panic(errors.NewUnexpectedError(
			"unsupported integer type %s / constant kind %s",
			integerType,
			constantKind,
		))
	}

	c.emitGetConstant(c.addConstant(constantKind, data))
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
	data := interpreter.NewUnmeteredUFix64Value(value.Uint64()).ToBigEndianBytes()
	return c.addConstant(constant.UFix64, data)
}

func (c *Compiler[_, _]) addFix64Constant(value *big.Int) *Constant {
	data := interpreter.NewUnmeteredFix64Value(value.Int64()).ToBigEndianBytes()
	return c.addConstant(constant.Fix64, data)
}

func (c *Compiler[_, _]) VisitArrayExpression(array *ast.ArrayExpression) (_ struct{}) {
	arrayTypes := c.DesugaredElaboration.ArrayExpressionTypes(array)

	typeIndex := c.getOrAddType(arrayTypes.ArrayType)

	size := len(array.Values)
	if size >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid array expression"))
	}

	elementExpectedType := arrayTypes.ArrayType.ElementType(false)

	for _, expression := range array.Values {
		c.compileExpression(expression)
		c.emitTransferAndConvert(elementExpectedType)
	}

	c.emit(
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

	dictionaryType := dictionaryTypes.DictionaryType

	typeIndex := c.getOrAddType(dictionaryType)

	size := len(dictionary.Entries)
	if size >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid dictionary expression"))
	}

	for _, entry := range dictionary.Entries {
		c.compileExpression(entry.Key)
		c.emitTransferAndConvert(dictionaryType.KeyType)
		c.compileExpression(entry.Value)
		c.emitTransferAndConvert(dictionaryType.ValueType)
	}

	c.emit(
		opcode.InstructionNewDictionary{
			Type:       typeIndex,
			Size:       uint16(size),
			IsResource: dictionaryType.IsResourceType(),
		},
	)

	return
}

func (c *Compiler[_, _]) VisitIdentifierExpression(expression *ast.IdentifierExpression) (_ struct{}) {
	c.emitVariableLoad(expression.Identifier.Identifier)
	return
}

func (c *Compiler[_, _]) emitVariableLoad(name string) {

	if c.currentInheritedConditionParamBinding != nil {
		// If the current compiling code is an inherited code, then bind
		// the inherited parameter names to the implementation's parameter names.
		mappedName, ok := c.currentInheritedConditionParamBinding[name]
		if ok {
			name = mappedName
		}
	}

	local := c.currentFunction.findLocal(name)
	if local != nil {
		c.emitGetLocal(local.index)
		return
	}

	upvalueIndex, ok := c.currentFunction.findOrAddUpvalue(name)
	if ok {
		c.emit(opcode.InstructionGetUpvalue{
			Upvalue: upvalueIndex,
		})
		return
	}

	c.emitGlobalLoad(name)
}

func (c *Compiler[_, _]) emitGlobalLoad(name string) {
	global := c.findGlobal(name)
	c.emit(opcode.InstructionGetGlobal{
		Global: global.Index,
	})
}

func (c *Compiler[_, _]) emitMethodLoad(name string) {
	global := c.findGlobal(name)
	c.emit(opcode.InstructionGetMethod{
		Method: global.Index,
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
		c.emit(opcode.InstructionSetUpvalue{
			Upvalue: upvalueIndex,
		})
		return
	}

	global := c.findGlobal(name)
	c.emit(opcode.InstructionSetGlobal{
		Global: global.Index,
	})
}

func (c *Compiler[_, _]) visitInvocationExpressionWithImplicitArgument(
	expression *ast.InvocationExpression,
	implicitArgIndex uint16,
	implicitArgType sema.Type,
) (_ struct{}) {
	// TODO: copy

	invocationTypes := c.DesugaredElaboration.InvocationExpressionTypes(expression)

	argumentCount := len(expression.Arguments)
	if argumentCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid number of arguments"))
	}

	invokedExpr := expression.InvokedExpression

	if memberExpression, isMemberExpr := expression.InvokedExpression.(*ast.MemberExpression); isMemberExpr {
		memberInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(memberExpression)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// If the member is a method or a constructor (i.e: not a field), compile it as a method-invocation.
		// Otherwise, compile it as a normal function invocation.
		if memberInfo.Member.DeclarationKind != common.DeclarationKindField {
			c.compileMethodInvocation(
				expression,
				memberInfo,
				memberExpression,
				invocationTypes,
				uint16(argumentCount),
			)
			return
		}
	}

	// For all other expressions, get/lookup the function and invoke it.
	// For example, if the function is static function, it will load the function value from the globals.
	// If the function is a result of executing another expression (e.g: result of another function call),
	// then it will get the function-pointer value from the stack.

	// Load function value
	c.compileExpression(invokedExpr)

	// Compile arguments
	c.compileArguments(expression.Arguments, invocationTypes)

	typeArgs := c.loadTypeArguments(invocationTypes)
	if implicitArgType != nil {
		// Add the implicit argument to the end of the argument list, if it exists.
		// Used in attachments, the attachment constructor/init expects an implicit argument:
		// a reference to the base value used to set base.
		// This hides the base argument away from the user.
		typeArgs = append(typeArgs, c.getOrAddType(implicitArgType))
		argumentCount += 1
		if argumentCount >= math.MaxUint16 {
			panic(errors.NewDefaultUserError("invalid number of arguments"))
		}
		// Load implicit argument from locals
		c.emitGetLocal(implicitArgIndex)
	}

	c.emit(opcode.InstructionInvoke{
		TypeArgs: typeArgs,
		ArgCount: uint16(argumentCount),
	})
	return
}

func (c *Compiler[_, _]) VisitInvocationExpression(expression *ast.InvocationExpression) (_ struct{}) {
	c.visitInvocationExpressionWithImplicitArgument(expression, 0, nil)

	return
}

func (c *Compiler[_, _]) compileMethodInvocation(
	expression *ast.InvocationExpression,
	memberInfo sema.MemberAccessInfo,
	invokedExpr *ast.MemberExpression,
	invocationTypes sema.InvocationExpressionTypes,
	argumentCount uint16,
) {
	var funcName string

	invocationType := memberInfo.Member.TypeAnnotation.Type.(*sema.FunctionType)
	if invocationType.IsConstructor {
		funcName = commons.TypeQualifiedName(
			memberInfo.AccessedType,
			invokedExpr.Identifier.Identifier,
		)

		// Calling a type constructor must be invoked statically. e.g: `SomeContract.Foo()`.

		// Load function value
		c.emitGlobalLoad(funcName)

		// Compile arguments
		c.compileArguments(expression.Arguments, invocationTypes)

		typeArgs := c.loadTypeArguments(invocationTypes)

		c.emit(opcode.InstructionInvoke{
			TypeArgs: typeArgs,
			ArgCount: argumentCount,
		})
		return
	}

	isOptional := memberInfo.IsOptional

	typeArgs := c.loadTypeArguments(invocationTypes)

	// Invocations into the interface code, such as default functions and inherited conditions,
	// that were synthetically added at the desugar phase, must be static calls.
	isInterfaceInheritedFuncCall := c.DesugaredElaboration.IsInterfaceMethodStaticCall(expression)

	// Any invocation on restricted-types must be dynamic
	if !isInterfaceInheritedFuncCall && isDynamicMethodInvocation(memberInfo.AccessedType) {
		funcName = invokedExpr.Identifier.Identifier
		if len(funcName) >= math.MaxUint16 {
			panic(errors.NewDefaultUserError("invalid function name"))
		}

		c.withOptionalChaining(
			invokedExpr.Expression,
			isOptional,
			func() {
				// withOptionalChaining already load the receiver onto the stack.

				// Compile arguments
				c.compileArguments(expression.Arguments, invocationTypes)

				funcNameConst := c.addStringConst(funcName)

				argsCountWithReceiver := argumentCount + 1

				c.emit(
					opcode.InstructionInvokeMethodDynamic{
						Name:     funcNameConst.index,
						TypeArgs: typeArgs,
						ArgCount: argsCountWithReceiver,
					},
				)
			},
		)

		return
	}

	// If the function is accessed via optional-chaining,
	// then the target type is the inner type of the optional.
	accessedType := memberInfo.AccessedType
	if isOptional {
		accessedType = sema.UnwrapOptionalType(accessedType)
	}

	// Load function value.
	funcName = commons.TypeQualifiedName(
		accessedType,
		invokedExpr.Identifier.Identifier,
	)

	// An invocation can be either a method of a value (e.g: `"someString".Concat("otherString")`),
	// or a function on a "type function" (e.g: `String.join(["someString", "otherString"], separator: ", ")`),
	// where `String` is a function.
	accessedTypeFunctionType := typeFunctionType(accessedType)
	if accessedTypeFunctionType != nil {

		// Compile as static-function call.
		// No receiver is loaded.
		c.emitGlobalLoad(funcName)
		c.compileArguments(expression.Arguments, invocationTypes)
		c.emit(opcode.InstructionInvoke{
			TypeArgs: typeArgs,
			ArgCount: argumentCount,
		})
	} else {
		c.withOptionalChaining(
			invokedExpr.Expression,
			isOptional,
			func() {
				// Compile as object-method call.
				// Function must be loaded only if the receiver is non-nil.
				// The receiver is already on the stack.

				// Get the method as a bound function.
				// This is needed to capture the implicit reference that's get created by bound functions.
				c.emitMethodLoad(funcName)

				// Compile arguments
				c.compileArguments(expression.Arguments, invocationTypes)

				c.emit(opcode.InstructionInvokeMethodStatic{
					TypeArgs: typeArgs,

					// Argument count does not include the receiver,
					// since receiver is already captured by the bound-function.
					ArgCount: argumentCount,
				})
			},
		)
	}
}

// withOptionalChaining compiles the `ifNotNil` procedure with optional chaining.
// IMPORTANT: This function expects the `ifNotNil` procedure to assume the target expression
// is already loaded on to the stack.
// This is an optimization to avoid redundant store-to/load-from local indexes.
func (c *Compiler[_, _]) withOptionalChaining(
	targetExpression ast.Expression,
	isOptional bool,
	ifNotNil func(),
) {
	nilJump := c.compileOptionalChainingNilJump(targetExpression, isOptional)
	ifNotNil()
	c.patchOptionalChainingNilJump(isOptional, nilJump)
}

// compileOptionalChainingNilJump compiles the nil-check for optional chaining.
// If the value is nil, a jump is emitted to the nil-returning instructions.
// Otherwise, if the value is unwrapped and left on stack.
func (c *Compiler[_, _]) compileOptionalChainingNilJump(
	targetExpression ast.Expression,
	isOptional bool,
) int {
	c.compileExpression(targetExpression)

	if !isOptional {
		return -1
	}

	tempIndex := c.currentFunction.generateLocalIndex()
	c.emitSetLocal(tempIndex)

	// If the value is nil, return nil. by jumping to the instruction where nil is returned.
	c.emitGetLocal(tempIndex)
	nilJump := c.emitUndefinedJumpIfNil()

	// Otherwise unwrap.
	c.emitGetLocal(tempIndex)
	c.emit(opcode.InstructionUnwrap{})
	return nilJump
}

func (c *Compiler[_, _]) patchOptionalChainingNilJump(isOptional bool, nilJump int) {
	if !isOptional {
		return
	}

	// TODO: Need to wrap the result back with an optional, if `memberAccessInfo.IsOptional`
	// Jump to the end to skip the nil returning instructions.
	jumpToEnd := c.emitUndefinedJump()

	c.patchJumpHere(nilJump)
	c.emit(opcode.InstructionNil{})

	c.patchJumpHere(jumpToEnd)
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

func (c *Compiler[_, _]) compileArguments(arguments ast.Arguments, invocationTypes sema.InvocationExpressionTypes) {
	for index, argument := range arguments {
		c.compileExpression(argument.Expression)
		parameterType := invocationTypes.ParameterTypes[index]
		if parameterType == nil {
			c.emitTransfer()
		} else {
			c.emitTransferAndConvert(parameterType)
		}
	}
}

func (c *Compiler[_, _]) loadTypeArguments(invocationTypes sema.InvocationExpressionTypes) []uint16 {
	typeArguments := invocationTypes.TypeArguments
	typeArgsCount := typeArguments.Len()
	if typeArgsCount >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid number of type arguments: %d", typeArgsCount))
	}

	var typeArgs []uint16
	if typeArgsCount > 0 {
		typeArgs = make([]uint16, 0, typeArgsCount)

		typeArguments.Foreach(func(key *sema.TypeParameter, typeParam sema.Type) {
			typeArgs = append(typeArgs, c.getOrAddType(typeParam))
		})
	}

	return typeArgs
}

func typeFunctionType(ty sema.Type) sema.Type {
	functionType, ok := ty.(*sema.FunctionType)
	if !ok {
		return nil
	}
	return functionType.TypeFunctionType
}

func enumType(ty sema.Type) *sema.CompositeType {
	compositeType, ok := ty.(*sema.CompositeType)
	if !ok {
		return nil
	}
	if compositeType.GetCompositeKind() != common.CompositeKindEnum {
		return nil
	}
	return compositeType
}

func (c *Compiler[_, _]) VisitMemberExpression(expression *ast.MemberExpression) (_ struct{}) {
	memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	identifier := expression.Identifier.Identifier

	accessedType := memberAccessInfo.AccessedType

	// Accessing an enum case?
	accessedTypeFunctionType := typeFunctionType(accessedType)
	if accessedTypeFunctionType != nil {

		accessedEnumType := enumType(accessedTypeFunctionType)
		if accessedEnumType != nil {
			qualifiedName := commons.TypeQualifiedName(accessedEnumType, identifier)
			c.emitGlobalLoad(qualifiedName)

			return
		}
	}

	c.withOptionalChaining(
		expression.Expression,
		memberAccessInfo.IsOptional,
		func() {
			// withOptionalChaining evaluates the target expression
			// and leave the value on stack.
			// i.e: the target/parent is already loaded.

			c.compileMemberAccess(expression)
		},
	)

	return
}

func (c *Compiler[_, _]) compileMemberAccess(expression *ast.MemberExpression) {

	identifier := expression.Identifier.Identifier

	constant := c.addStringConst(identifier)

	memberAccessInfo, ok := c.DesugaredElaboration.MemberExpressionMemberAccessInfo(expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	isNestedResourceMove := c.DesugaredElaboration.IsNestedResourceMoveExpression(expression)
	if isNestedResourceMove {
		c.emit(opcode.InstructionRemoveField{
			FieldName: constant.index,
		})
	} else {
		accessedType := memberAccessInfo.AccessedType
		if memberAccessInfo.IsOptional {
			accessedType = sema.UnwrapOptionalType(accessedType)
		}
		accessedTypeIndex := c.getOrAddType(accessedType)

		c.emit(opcode.InstructionGetField{
			FieldName:    constant.index,
			AccessedType: accessedTypeIndex,
		})
	}

	// Return a reference, if the member is accessed via a reference.
	// This is pre-computed at the checker.
	if memberAccessInfo.ReturnReference {
		index := c.getOrAddType(memberAccessInfo.ResultingType)
		c.emit(opcode.InstructionNewRef{
			Type:       index,
			IsImplicit: true,
		})
	}
}

func (c *Compiler[_, _]) VisitIndexExpression(expression *ast.IndexExpression) (_ struct{}) {
	c.compileExpression(expression.TargetExpression)

	if attachmentType, ok := c.DesugaredElaboration.AttachmentAccessTypes(expression); ok {
		c.emit(opcode.InstructionGetTypeIndex{
			Type: c.getOrAddType(attachmentType),
		})
	} else {
		c.compileExpression(expression.IndexingExpression)

		c.compileIndexAccess(expression)
	}

	return
}

// compileIndexAccess compiles the index access, i.e. RemoveIndex or GetIndex.
// It assumes the target and indexing/key expressions are already compiled on the stack.
func (c *Compiler[_, _]) compileIndexAccess(expression *ast.IndexExpression) {
	c.emitIndexKeyTransferAndConvert(expression)

	isNestedResourceMove := c.DesugaredElaboration.IsNestedResourceMoveExpression(expression)
	if isNestedResourceMove {
		c.emit(opcode.InstructionRemoveIndex{})
	} else {
		c.emit(opcode.InstructionGetIndex{})
	}

	indexExpressionTypes, ok := c.DesugaredElaboration.IndexExpressionTypes(expression)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Return a reference, if the element is accessed via a reference.
	// This is pre-computed at the checker.
	if indexExpressionTypes.ReturnReference {
		index := c.getOrAddType(indexExpressionTypes.ResultType)
		c.emit(opcode.InstructionNewRef{
			Type:       index,
			IsImplicit: true,
		})
	}
}

func (c *Compiler[_, _]) VisitConditionalExpression(expression *ast.ConditionalExpression) (_ struct{}) {
	// Test
	c.compileExpression(expression.Test)
	elseJump := c.emitUndefinedJumpIfFalse()

	// Then branch
	c.compileExpression(expression.Then)
	thenJump := c.emitUndefinedJump()

	// Else branch
	c.patchJumpHere(elseJump)
	c.compileExpression(expression.Else)

	c.patchJumpHere(thenJump)

	return
}

func (c *Compiler[_, _]) VisitUnaryExpression(expression *ast.UnaryExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)

	switch expression.Operation {
	case ast.OperationNegate:
		c.emit(opcode.InstructionNot{})

	case ast.OperationMinus:
		c.emit(opcode.InstructionNegate{})

	case ast.OperationMul:
		c.emit(opcode.InstructionDeref{})

	case ast.OperationMove:
		// Transfer to the target type.
		targetType := c.DesugaredElaboration.MoveExpressionTypes(expression)
		typeIndex := c.getOrAddType(targetType)
		c.codeGen.Emit(opcode.InstructionTransferAndConvert{
			Type: typeIndex,
		})

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
		c.emit(opcode.InstructionDup{})
		elseJump := c.emitUndefinedJumpIfNil()

		// Then branch
		c.emit(opcode.InstructionUnwrap{})
		thenJump := c.emitUndefinedJump()

		// Else branch
		c.patchJumpHere(elseJump)
		// Drop the duplicated condition result,
		// as it is not needed for the 'else' path.
		c.emit(opcode.InstructionDrop{})
		c.compileExpression(expression.Right)

		// End
		c.patchJumpHere(thenJump)

	case ast.OperationOr:
		// TODO: optimize chains of ors / ands

		leftTrueJump := c.emitUndefinedJumpIfTrue()

		c.compileExpression(expression.Right)
		rightFalseJump := c.emitUndefinedJumpIfFalse()

		// Left or right is true
		c.patchJumpHere(leftTrueJump)
		c.emit(opcode.InstructionTrue{})
		trueJump := c.emitUndefinedJump()

		// Left and right are false
		c.patchJumpHere(rightFalseJump)
		c.emit(opcode.InstructionFalse{})

		c.patchJumpHere(trueJump)

	case ast.OperationAnd:
		// TODO: optimize chains of ors / ands

		leftFalseJump := c.emitUndefinedJumpIfFalse()

		c.compileExpression(expression.Right)
		rightFalseJump := c.emitUndefinedJumpIfFalse()

		// Left and right are true
		c.emit(opcode.InstructionTrue{})
		trueJump := c.emitUndefinedJump()

		// Left or right is false
		c.patchJumpHere(leftFalseJump)
		c.patchJumpHere(rightFalseJump)
		c.emit(opcode.InstructionFalse{})

		c.patchJumpHere(trueJump)

	default:
		c.compileExpression(expression.Right)

		switch expression.Operation {
		case ast.OperationPlus:
			c.emit(opcode.InstructionAdd{})
		case ast.OperationMinus:
			c.emit(opcode.InstructionSubtract{})
		case ast.OperationMul:
			c.emit(opcode.InstructionMultiply{})
		case ast.OperationDiv:
			c.emit(opcode.InstructionDivide{})
		case ast.OperationMod:
			c.emit(opcode.InstructionMod{})

		case ast.OperationBitwiseOr:
			c.emit(opcode.InstructionBitwiseOr{})
		case ast.OperationBitwiseAnd:
			c.emit(opcode.InstructionBitwiseAnd{})
		case ast.OperationBitwiseXor:
			c.emit(opcode.InstructionBitwiseXor{})
		case ast.OperationBitwiseLeftShift:
			c.emit(opcode.InstructionBitwiseLeftShift{})
		case ast.OperationBitwiseRightShift:
			c.emit(opcode.InstructionBitwiseRightShift{})

		case ast.OperationEqual:
			c.emit(opcode.InstructionEqual{})
		case ast.OperationNotEqual:
			c.emit(opcode.InstructionNotEqual{})

		case ast.OperationLess:
			c.emit(opcode.InstructionLess{})
		case ast.OperationLessEqual:
			c.emit(opcode.InstructionLessOrEqual{})
		case ast.OperationGreater:
			c.emit(opcode.InstructionGreater{})
		case ast.OperationGreaterEqual:
			c.emit(opcode.InstructionGreaterOrEqual{})
		default:
			panic(errors.NewUnreachableError())
		}
	}

	return
}

func (c *Compiler[_, _]) VisitFunctionExpression(expression *ast.FunctionExpression) (_ struct{}) {
	// It is OK/safe to use the desugar-instance to desugar the function-expression,
	// since function-expression desugaring doesn't rely on contextual-information.
	// (i.e: doesn't rely on where this expression is located in the AST; doesn't inherit from other functions, etc.).
	desugaredExpression := c.desugar.DesugarFunctionExpression(expression)

	functionIndex := len(c.functions)

	if functionIndex >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid function index"))
	}

	parameterCount := 0
	parameterList := desugaredExpression.ParameterList
	if parameterList != nil {
		parameterCount = len(parameterList.Parameters)
	}

	if parameterCount > math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid parameter count"))
	}

	functionType := c.DesugaredElaboration.FunctionExpressionFunctionType(desugaredExpression)

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

		c.declareParameters(parameterList, false, false)
		c.compileFunctionBlock(
			desugaredExpression.FunctionBlock,
			common.DeclarationKindFunction,
			functionType.ReturnTypeAnnotation.Type,
		)
	}()

	c.emitNewClosure(uint16(functionIndex), function)

	return
}

func (c *Compiler[_, _]) VisitStringExpression(expression *ast.StringExpression) (_ struct{}) {
	stringType := c.DesugaredElaboration.StringExpressionType(expression)

	switch stringType {
	case sema.CharacterType:
		c.emitCharacterConst(expression.Value)
	case sema.StringType:
		c.emitStringConst(expression.Value)
	default:
		panic(errors.NewUnreachableError())
	}

	return
}

func (c *Compiler[_, _]) VisitStringTemplateExpression(expression *ast.StringTemplateExpression) (_ struct{}) {
	exprArrSize := len(expression.Expressions)

	for _, value := range expression.Values {
		c.emitStringConst(value)
	}
	for _, expression := range expression.Expressions {
		c.compileExpression(expression)
	}

	c.emit(
		opcode.InstructionTemplateString{
			ExprSize: uint16(exprArrSize),
		},
	)

	return
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

	c.emit(castInstruction)
	return
}

func (c *Compiler[_, _]) VisitCreateExpression(expression *ast.CreateExpression) (_ struct{}) {
	c.compileExpression(expression.InvocationExpression)
	return
}

func (c *Compiler[_, _]) VisitDestroyExpression(expression *ast.DestroyExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.emit(opcode.InstructionDestroy{})
	return
}

func (c *Compiler[_, _]) VisitReferenceExpression(expression *ast.ReferenceExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	borrowType := c.DesugaredElaboration.ReferenceExpressionBorrowType(expression)
	typeIndex := c.getOrAddType(borrowType)
	c.emit(opcode.InstructionNewRef{
		Type: typeIndex,
	})
	return
}

func (c *Compiler[_, _]) VisitForceExpression(expression *ast.ForceExpression) (_ struct{}) {
	c.compileExpression(expression.Expression)
	c.emit(opcode.InstructionUnwrap{})
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

	c.emit(
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
	enclosingType := c.compositeTypeStack.top()
	kind := enclosingType.GetCompositeKind()

	typeName := commons.TypeQualifier(enclosingType)

	var functionName string
	if kind == common.CompositeKindContract {
		// For contracts, add the initializer as `init()`.
		// A global variable with the same name as contract is separately added.
		// The VM will load the contract and assign to that global variable during imports resolution.
		identifier := declaration.DeclarationIdentifier().Identifier
		functionName = commons.QualifiedName(typeName, identifier)
	} else {
		// Use the type name as the function name for initializer.
		// So `x = Foo()` would directly call the init method.
		functionName = typeName
	}

	parameterCount := 0
	parameterList := declaration.FunctionDeclaration.ParameterList
	if parameterList != nil {
		parameterCount = len(parameterList.Parameters)
	}

	if parameterCount >= math.MaxUint16 {
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

	// cannot declare base as parameter because it is at the end of the argument list
	c.declareParameters(parameterList, false, false)

	// must do this before declaring self
	if kind == common.CompositeKindAttachment {
		// base is provided as an argument at the end of the argument list implicitly
		c.currentFunction.declareLocal(sema.BaseIdentifier)
	}

	// Declare `self`
	self := c.currentFunction.declareLocal(sema.SelfIdentifier)

	// Initialize an empty struct and assign to `self`.
	// i.e: `self = New()`

	// Write composite kind
	// TODO: Maybe get/include this from static-type. Then no need to provide separately.

	typeIndex := c.getOrAddType(enclosingType)

	c.emit(
		opcode.InstructionNew{
			Kind: kind,
			Type: typeIndex,
		},
	)

	// stores the return value of the constructor
	var returnLocalIndex uint16
	// `self` in attachments is a reference.
	if kind == common.CompositeKindAttachment {
		// Store the new composite as the return value.
		returnLocalIndex = c.currentFunction.generateLocalIndex()
		c.emitSetLocal(returnLocalIndex)
		c.emitGetLocal(returnLocalIndex)
		baseTyp := enclosingType.(sema.EntitlementSupportingType)
		baseAccess := baseTyp.SupportedEntitlements().Access()
		refType := &sema.ReferenceType{
			Type:          baseTyp,
			Authorization: baseAccess,
		}
		// Set `self` to be a reference.
		c.emit(opcode.InstructionNewRef{
			Type:       c.getOrAddType(refType),
			IsImplicit: false,
		})

		// get base from end of arguments...
		// TODO: expose base, a reference to the attachment's base value
	} else {
		returnLocalIndex = self.index
	}

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
		c.emit(opcode.InstructionDup{})
		global := c.findGlobal(typeName)

		c.emit(opcode.InstructionSetGlobal{
			Global: global.Index,
		})
	}

	c.emitSetLocal(self.index)

	// emit for the statements in `init()` body.
	c.compileFunctionBlock(
		declaration.FunctionDeclaration.FunctionBlock,
		declaration.Kind,

		// The return type of the initializer is the type itself.
		enclosingType,
	)

	// Constructor should return the created the struct. i.e: return `self`
	c.emitGetLocal(returnLocalIndex)

	// No need to transfer, since the type is same as the constructed value, for initializers.
	c.emit(opcode.InstructionReturnValue{})
}

func (c *Compiler[E, _]) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, _ bool) (_ struct{}) {
	previousFunction := c.currentFunction

	var (
		parameterCount int
		isObjectMethod bool
		isAttachment   bool
		functionName   string
	)

	paramList := declaration.ParameterList
	if paramList != nil {
		parameterCount = len(paramList.Parameters)
	}

	identifier := declaration.Identifier.Identifier

	var innerFunctionLocal *local

	if previousFunction == nil {
		// Global function or method
		isObjectMethod = !c.compositeTypeStack.isEmpty()

		var enclosingType sema.Type
		if isObjectMethod {
			enclosingType = c.compositeTypeStack.top()

			// Declare a receiver if this is an object method.
			parameterCount++

			if typ, ok := enclosingType.(*sema.CompositeType); ok {
				if typ.Kind == common.CompositeKindAttachment {
					parameterCount++
					isAttachment = true
				}
			}
		}

		functionName = commons.TypeQualifiedName(enclosingType, identifier)

	} else {
		// Inner function

		// It is OK/safe to use the desugar-instance to desugar the inner function,
		// since inner function desugaring doesn't rely on contextual-information.
		// (i.e: doesn't rely on where this function is located in the AST; doesn't inherit from other functions, etc.).
		declaration = c.desugar.DesugarInnerFunction(declaration)

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

		c.declareParameters(declaration.ParameterList, isObjectMethod, isAttachment)

		c.compileFunctionBlock(
			declaration.FunctionBlock,
			declaration.DeclarationKind(),
			functionType.ReturnTypeAnnotation.Type,
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

	// Event declarations have no members
	if compositeType.Kind == common.CompositeKindEvent &&
		!declaration.IsResourceDestructionDefaultEvent() {
		return
	}

	c.compositeTypeStack.push(compositeType)
	defer c.compositeTypeStack.pop()

	// Compile members
	hasInit := false
	for _, specialFunc := range declaration.Members.SpecialFunctions() {
		if specialFunc.Kind == common.DeclarationKindInitializer {
			hasInit = true
		}
		c.compileDeclaration(specialFunc)
	}

	// If the initializer is not declared, generate
	// - a synthetic initializer for enum types, otherwise
	// - an empty initializer
	if !hasInit {
		if compositeType.Kind == common.CompositeKindEnum {
			c.generateEnumInit(compositeType)
			c.generateEnumLookup(
				compositeType,
				declaration.Members.EnumCases(),
			)
		} else {
			c.generateEmptyInit()
		}
	}

	// Visit members.
	c.compileCompositeMembers(compositeType, declaration.Members)

	return
}

func (c *Compiler[_, _]) compileCompositeMembers(
	compositeKindedType sema.CompositeKindedType,
	members *ast.Members,
) {
	// Important: Must be visited in the same order as the globals were reserved in `reserveGlobals`.

	// Add the methods that are provided natively.
	c.addBuiltinMethods(compositeKindedType)

	for index, enumCase := range members.EnumCases() {
		c.compileEnumCaseDeclaration(
			enumCase,
			compositeKindedType.(*sema.CompositeType),
			index,
		)
	}

	for _, function := range members.Functions() {
		c.compileDeclaration(function)
	}
	for _, nestedType := range members.Composites() {
		c.compileDeclaration(nestedType)
	}
	for _, nestedType := range members.Interfaces() {
		c.compileDeclaration(nestedType)
	}
	for _, nestedAttachments := range members.Attachments() {
		c.compileDeclaration(nestedAttachments)
	}
}

func (c *Compiler[_, _]) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) (_ struct{}) {
	interfaceType := c.DesugaredElaboration.InterfaceDeclarationType(declaration)
	c.compositeTypeStack.push(interfaceType)
	defer c.compositeTypeStack.pop()

	// Visit members.
	c.compileCompositeMembers(interfaceType, declaration.Members)

	return
}

func (c *Compiler[_, _]) addBuiltinMethods(typ sema.Type) {
	for _, boundFunction := range commonBuiltinTypeBoundFunctions {
		name := boundFunction.name
		qualifiedName := commons.TypeQualifiedName(typ, name)
		c.addFunction(
			name,
			qualifiedName,
			uint16(len(boundFunction.typ.Parameters)+1),
			boundFunction.typ,
		)
	}

	if t, ok := typ.(sema.CompositeKindedType); ok {
		if t.GetCompositeKind().SupportsAttachments() {
			name := sema.CompositeForEachAttachmentFunctionName
			qualifiedName := commons.TypeQualifiedName(typ, name)
			functionType := sema.CompositeForEachAttachmentFunctionType(t.GetCompositeKind())
			c.addFunction(
				name,
				qualifiedName,
				uint16(len(functionType.Parameters)),
				functionType,
			)
		}
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
		c.addGlobalsFromImportedProgram(location.Location)
	}

	return
}

func (c *Compiler[_, _]) addGlobalsFromImportedProgram(location common.Location) {
	// Built-in location has no program.
	if location == nil {
		return
	}

	importedProgram := c.Config.ImportHandler(location)

	// Add a global variable for the imported contract value.
	contracts := importedProgram.Contracts
	for _, contract := range contracts {
		c.addImportedGlobal(location, contract.Name)
	}

	for _, variable := range importedProgram.Variables {
		c.addImportedGlobal(location, variable.Name)
	}

	for _, function := range importedProgram.Functions {
		name := function.QualifiedName

		//// TODO: Skip the contract initializer.
		//// It should never be able to invoked within the code.
		//if isContract && name == commons.InitFunctionName {
		//	continue
		//}

		// TODO: Filter-in only public functions
		c.addImportedGlobal(location, name)
	}

	// Recursively add transitive imports.
	for _, impt := range importedProgram.Imports {
		c.addGlobalsFromImportedProgram(impt.Location)
	}
}

func (c *Compiler[_, _]) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) compileEnumCaseDeclaration(
	declaration *ast.EnumCaseDeclaration,
	compositeType *sema.CompositeType,
	index int,
) (_ struct{}) {

	variableGetterFunctionType := sema.NewSimpleFunctionType(
		sema.FunctionPurityImpure,
		nil,
		sema.NewTypeAnnotation(compositeType),
	)

	caseName := declaration.Identifier.Identifier
	getterName := commons.TypeQualifiedName(compositeType, caseName)

	globalVariable := c.addGlobalVariableWithGetter(
		getterName,
		variableGetterFunctionType,
	)

	func() {
		previousFunction := c.currentFunction
		c.targetFunction(globalVariable.Getter)
		defer c.targetFunction(previousFunction)

		// No parameters

		constructorName := commons.TypeQualifiedName(compositeType, commons.InitFunctionName)
		c.emitGlobalLoad(constructorName)
		c.emitIntegerConstant(
			big.NewInt(int64(index)),
			compositeType.EnumRawType,
		)
		c.emit(opcode.InstructionInvoke{
			ArgCount: 1,
		})
		c.emitTransferAndConvertAndReturnValue(compositeType)
	}()

	return
}

func (c *Compiler[_, _]) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) (_ struct{}) {
	// Similar to VisitCompositeDeclaration
	// Not combined because need to access fields not accessible in CompositeLikeDeclaration
	compositeType := c.DesugaredElaboration.CompositeDeclarationType(declaration)

	c.compositeTypeStack.push(compositeType)
	defer c.compositeTypeStack.pop()

	// Compile members
	hasInit := false
	for _, specialFunc := range declaration.Members.SpecialFunctions() {
		if specialFunc.Kind == common.DeclarationKindInitializer {
			hasInit = true
		}
		c.compileDeclaration(specialFunc)
	}

	// If the initializer is not declared, generate an empty initializer
	if !hasInit {
		c.generateEmptyInit()
	}

	// Visit members.
	c.compileCompositeMembers(compositeType, declaration.Members)

	return
}

func (c *Compiler[_, _]) VisitEntitlementDeclaration(_ *ast.EntitlementDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitEntitlementMappingDeclaration(_ *ast.EntitlementMappingDeclaration) (_ struct{}) {
	// TODO
	panic(errors.NewUnreachableError())
}

func (c *Compiler[_, _]) VisitRemoveStatement(statement *ast.RemoveStatement) (_ struct{}) {
	// load base onto stack
	c.compileExpression(statement.Value)
	// remove attachment from base
	nominalType := c.DesugaredElaboration.AttachmentRemoveTypes(statement)
	c.emit(opcode.InstructionRemoveTypeIndex{
		Type: c.getOrAddType(nominalType),
	})
	return
}

func (c *Compiler[_, _]) VisitAttachExpression(expression *ast.AttachExpression) (_ struct{}) {
	types := c.DesugaredElaboration.AttachTypes(expression)
	baseType := types.BaseType
	attachmentType := types.AttachType

	// base on stack
	c.compileExpression(expression.Base)
	// store base locally
	baseLocalIndex := c.currentFunction.generateLocalIndex()
	c.emitSetLocal(baseLocalIndex)
	// get base back on stack
	c.emitGetLocal(baseLocalIndex)
	baseTyp := baseType.(sema.EntitlementSupportingType)
	baseAccess := baseTyp.SupportedEntitlements().Access()
	refType := &sema.ReferenceType{
		Type:          baseTyp,
		Authorization: baseAccess,
	}
	// create reference to base to pass as implicit arg
	c.emit(opcode.InstructionNewRef{
		Type:       c.getOrAddType(refType),
		IsImplicit: false,
	})
	refLocalIndex := c.currentFunction.generateLocalIndex()
	c.emitSetLocal(refLocalIndex)

	// create the attachment
	c.visitInvocationExpressionWithImplicitArgument(expression.Attachment, refLocalIndex, baseType)
	// attachment on stack

	// base back on stack
	c.emitGetLocal(baseLocalIndex)
	// base should now be transferred
	c.emitTransfer()

	// add attachment value as a member of transferred base
	// returns the result
	c.emit(opcode.InstructionSetTypeIndex{
		Type: c.getOrAddType(attachmentType),
	})
	return
}

func (c *Compiler[_, _]) emitTransferAndConvert(targetType sema.Type) {

	//lastInstruction := c.codeGen.LastInstruction()

	// TODO: Revisit the below logic: last instruction may not always be the
	//  actually executed last instruction, in case where branching is present.
	//  e.g: conditional-expression (var a: Int? = condition ? 123 : nil)
	//  Here last instruction can be `123` constant-load, depending on the execution.

	// Optimization: We can omit the transfer in some cases
	//switch lastInstruction := lastInstruction.(type) {
	//case opcode.InstructionGetConstant:
	//	// If the last instruction is a constant load of the same type,
	//	// then the transfer is not needed.
	//	targetConstantKind := constant.FromSemaType(targetType)
	//	constantIndex := lastInstruction.Constant
	//	c := c.constants[constantIndex]
	//	if c.kind == targetConstantKind {
	//		return
	//	}
	//
	//case opcode.InstructionNewPath:
	//	// If the last instruction is a path creation of the same type,
	//	// then the transfer is not needed.
	//	switch lastInstruction.Domain {
	//	case common.PathDomainPublic:
	//		if targetType == sema.PublicPathType {
	//			return
	//		}
	//
	//	case common.PathDomainStorage:
	//		if targetType == sema.StoragePathType {
	//			return
	//		}
	//	}
	//
	//case opcode.InstructionNewClosure:
	//	// If the last instruction is a closure creation of the same type,
	//	// then the transfer is not needed.
	//	function := c.functions[lastInstruction.Function]
	//	functionSourceType := c.types[function.typeIndex].(*sema.FunctionType)
	//	if functionTargetType, ok := targetType.(*sema.FunctionType); ok {
	//		if functionSourceType.Equal(functionTargetType) {
	//			return
	//		}
	//	}
	//
	//case opcode.InstructionNil:
	//	// If the last instruction is a nil load,
	//	// then the transfer is not needed.
	//	return
	//}

	typeIndex := c.getOrAddType(targetType)
	c.emit(opcode.InstructionTransferAndConvert{
		Type: typeIndex,
	})
}

func (c *Compiler[_, _]) emitTransfer() {
	c.emit(opcode.InstructionTransfer{})
}

func (c *Compiler[_, T]) getOrAddType(ty sema.Type) uint16 {
	typeID := ty.ID()

	// Optimization: Re-use types in the pool.
	index, ok := c.typesInPool[typeID]

	if !ok {
		staticType := interpreter.ConvertSemaToStaticType(c.Config.MemoryGauge, ty)
		data := c.typeGen.CompileType(staticType)
		index = c.addCompiledType(ty, data)
		c.typesInPool[typeID] = index
	}

	return index
}

func (c *Compiler[_, T]) addCompiledType(ty sema.Type, data T) uint16 {
	count := len(c.compiledTypes)
	if count >= math.MaxUint16 {
		panic(errors.NewDefaultUserError("invalid type declaration"))
	}

	c.compiledTypes = append(c.compiledTypes, data)
	c.types = append(c.types, ty)
	return uint16(count)
}

func (c *Compiler[E, T]) declareParameters(paramList *ast.ParameterList, declareReceiver bool, declareBase bool) {
	if declareReceiver {
		// Declare receiver as `self`.
		// Receiver is always at the zero-th index of params.
		c.currentFunction.declareLocal(sema.SelfIdentifier)
	}

	if declareBase {
		// Declare base receiver as `base`
		// Always at index one of params.
		c.currentFunction.declareLocal(sema.BaseIdentifier)
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

func (c *Compiler[_, _]) generateEnumInit(enumType *sema.CompositeType) {
	enumInitializer := newEnumInitializer(c.Config.MemoryGauge, enumType, c.DesugaredElaboration)
	enumInitializerFuncType := newEnumInitializerFuncType(enumType.EnumRawType)

	c.DesugaredElaboration.SetFunctionDeclarationFunctionType(
		enumInitializer.FunctionDeclaration,
		enumInitializerFuncType,
	)
	c.VisitSpecialFunctionDeclaration(enumInitializer)
}

func (c *Compiler[_, _]) generateEnumLookup(enumType *sema.CompositeType, enumCases []*ast.EnumCaseDeclaration) {
	memoryGauge := c.Config.MemoryGauge

	enumLookup := newEnumLookup(
		memoryGauge,
		enumType,
		enumCases,
		c.DesugaredElaboration,
	)
	enumLookupFuncType := newEnumLookupFuncType(memoryGauge, enumType)

	c.DesugaredElaboration.SetFunctionDeclarationFunctionType(
		enumLookup,
		enumLookupFuncType,
	)

	// TODO: improve
	previousCompositeTypeStack := c.compositeTypeStack
	c.compositeTypeStack = &Stack[sema.CompositeKindedType]{}
	defer func() {
		c.compositeTypeStack = previousCompositeTypeStack
	}()

	c.VisitFunctionDeclaration(enumLookup, false)
}

func (c *Compiler[_, _]) compilePotentiallyInheritedCode(statement ast.Statement, f func()) {
	stmtElaboration, ok := c.DesugaredElaboration.conditionsElaborations[statement]
	if ok {
		prevElaboration := c.DesugaredElaboration
		c.DesugaredElaboration = stmtElaboration

		preIsInheritedCode := c.currentInheritedConditionParamBinding
		c.currentInheritedConditionParamBinding = c.inheritedConditionParamBindings[statement]

		defer func() {
			c.DesugaredElaboration = prevElaboration
			c.currentInheritedConditionParamBinding = preIsInheritedCode
		}()
	}
	f()
}

func (c *Compiler[E, _]) emitNewClosure(functionIndex uint16, function *function[E]) {
	c.emit(opcode.InstructionNewClosure{
		Function: functionIndex,
		Upvalues: function.upvalues,
	})
}

func (c *Compiler[_, _]) emitCloseUpvalue(localIndex uint16) {
	c.codeGen.Emit(opcode.InstructionCloseUpvalue{
		Local: localIndex,
	})
}

func (c *Compiler[E, _]) emit(instruction opcode.Instruction) {
	// Get the index of the instruction to be emitted.
	// This is the offset before emitting the current instruction.
	instructionIndex := c.codeGen.Offset()

	c.codeGen.Emit(instruction)

	// If the line number info changed since the last recorded position info,
	// Then add the current instruction's position info.
	if c.lastChangedPosition != c.currentPosition {
		c.currentFunction.lineNumbers.AddPositionInfo(
			uint16(instructionIndex),
			c.currentPosition,
		)
		c.lastChangedPosition = c.currentPosition
	}
}
