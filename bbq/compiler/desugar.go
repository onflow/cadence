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
	"slices"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

const resultVariableName = "result"
const tempResultVariableName = "$_result"

// Desugar will rewrite the AST from high-level abstractions to a much lower-level
// abstractions, so the compiler and vm could work with a minimal set of language features.
type Desugar struct {
	memoryGauge            common.MemoryGauge
	elaboration            *ExtendedElaboration
	program                *ast.Program
	checker                *sema.Checker
	config                 *Config
	enclosingInterfaceType *sema.InterfaceType

	modifiedDeclarations         []ast.Declaration
	inheritedFuncsWithConditions map[string][]*inheritedFunction
	postConditionIndices         map[*ast.FunctionBlock]int

	importsSet map[common.Location]struct{}
	newImports []ast.Declaration
}

type inheritedFunction struct {
	interfaceType       *sema.InterfaceType
	functionDecl        *ast.FunctionDeclaration
	rewrittenConditions sema.PostConditionsRewrite
	elaboration         *ExtendedElaboration
}

var _ ast.DeclarationVisitor[ast.Declaration] = &Desugar{}

func NewDesugar(
	memoryGauge common.MemoryGauge,
	compilerConfig *Config,
	program *ast.Program,
	elaboration *ExtendedElaboration,
	checker *sema.Checker,
) *Desugar {
	return &Desugar{
		memoryGauge:                  memoryGauge,
		config:                       compilerConfig,
		elaboration:                  elaboration,
		program:                      program,
		checker:                      checker,
		importsSet:                   map[common.Location]struct{}{},
		inheritedFuncsWithConditions: map[string][]*inheritedFunction{},
		postConditionIndices:         map[*ast.FunctionBlock]int{},
	}
}

func (d *Desugar) Run() (program *ast.Program, postConditionIndices map[*ast.FunctionBlock]int) {
	declarations := d.program.Declarations()
	for _, declaration := range declarations {
		modifiedDeclaration := d.desugarDeclaration(declaration)
		if modifiedDeclaration != nil {
			d.modifiedDeclarations = append(d.modifiedDeclarations, modifiedDeclaration)
		}
	}

	d.modifiedDeclarations = append(d.newImports, d.modifiedDeclarations...)

	program = ast.NewProgram(d.memoryGauge, d.modifiedDeclarations)

	//fmt.Println(ast.Prettier(program))

	return program, d.postConditionIndices
}

func (d *Desugar) desugarDeclaration(declaration ast.Declaration) ast.Declaration {
	return ast.AcceptDeclaration[ast.Declaration](declaration, d)
}

func (d *Desugar) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, _ bool) ast.Declaration {
	funcBlock := declaration.FunctionBlock
	funcName := declaration.Identifier.Identifier

	preConditions := d.desugarPreConditions(
		funcName,
		funcBlock,
	)
	postConditions, beforeStatements := d.desugarPostConditions(
		funcName,
		funcBlock,
	)

	modifiedStatements := make([]ast.Statement, 0)
	modifiedStatements = append(modifiedStatements, preConditions...)

	modifiedStatements = append(modifiedStatements, beforeStatements...)

	functionType := d.elaboration.FunctionDeclarationFunctionType(declaration)
	returnType := functionType.ReturnTypeAnnotation.Type

	if funcBlock.HasStatements() {
		pos := funcBlock.Block.StartPos

		if len(postConditions) > 0 &&
			returnType != sema.VoidType {

			// If there are post conditions, and a return value, then define a temporary `$_result` variable.
			// This is because there can be conditional-returns in the middle of the function.
			// Thus, if there are post conditions, the return value would get assigned to this temp-result variable,
			// and would be jumped to the post-condition(s). The actual return would be at the end of the function.
			// This is done at the compiler.
			modifiedStatements = d.declareTempResultVariable(
				declaration,
				returnType,
				pos,
				modifiedStatements,
			)
		}

		// Add the remaining statements that are defined in this function.
		statements := funcBlock.Block.Statements
		modifiedStatements = append(modifiedStatements, statements...)

	} else if d.enclosingInterfaceType != nil {
		// If this is an interface-method without a body,
		// then do not generate a function for it.
		return nil
	}

	var modifiedFuncBlock *ast.FunctionBlock
	if len(postConditions) > 0 {
		// Keep track of where the post conditions start, for each function.
		// This is used by the compiler to patch the jumps for return statements.
		// Note: always use the "modifiedDecl" for tracking.
		postConditionIndex := len(modifiedStatements)
		defer func() {
			d.postConditionIndices[modifiedFuncBlock] = postConditionIndex
		}()

		// TODO: Declare the `result` variable only if it is used in the post conditions
		modifiedStatements = d.declareResultVariable(funcBlock, modifiedStatements, returnType)

		modifiedStatements = append(modifiedStatements, postConditions...)
	}

	modifiedFuncBlock = ast.NewFunctionBlock(
		d.memoryGauge,
		ast.NewBlock(
			d.memoryGauge,
			modifiedStatements,
			ast.NewRangeFromPositioned(d.memoryGauge, declaration),
		),
		nil,
		nil,
	)

	// TODO: Is the generated function needed to be desugared again?
	modifiedDecl := ast.NewFunctionDeclaration(
		d.memoryGauge,
		declaration.Access,
		declaration.Purity,
		declaration.IsStatic(),
		declaration.IsNative(),
		declaration.Identifier,
		declaration.TypeParameterList,
		declaration.ParameterList,
		declaration.ReturnTypeAnnotation,
		modifiedFuncBlock,
		declaration.StartPos,
		declaration.DocString,
	)

	d.elaboration.SetFunctionDeclarationFunctionType(modifiedDecl, functionType)

	return modifiedDecl
}

// Declare a `$_result` synthetic variable, to temporarily hold return values.
func (d *Desugar) declareTempResultVariable(
	declaration *ast.FunctionDeclaration,
	returnType sema.Type,
	pos ast.Position,
	modifiedStatements []ast.Statement,
) []ast.Statement {
	tempResultVarDecl := ast.NewVariableDeclaration(
		d.memoryGauge,
		ast.AccessNotSpecified,
		false,
		ast.NewIdentifier(
			d.memoryGauge,
			tempResultVariableName,
			pos,
		),
		declaration.ReturnTypeAnnotation,

		// We just need the variable to be defined. Value is assigned later.
		nil,

		ast.NewTransfer(
			d.memoryGauge,
			ast.TransferOperationCopy, // TODO: determine based on return value (if resource, this should be a move)
			pos,
		),
		pos,
		nil,
		nil,
		"",
	)

	d.elaboration.SetVariableDeclarationTypes(
		tempResultVarDecl,
		sema.VariableDeclarationTypes{
			ValueType:  returnType,
			TargetType: returnType,
		},
	)

	modifiedStatements = append(modifiedStatements, tempResultVarDecl)
	return modifiedStatements
}

// Declare the `result` variable which will be made available to the post-conditions.
func (d *Desugar) declareResultVariable(
	funcBlock *ast.FunctionBlock,
	modifiedStatements []ast.Statement,
	returnType sema.Type,
) []ast.Statement {

	pos := funcBlock.EndPosition(d.memoryGauge)
	resultVarType, exist := d.elaboration.ResultVariableType(funcBlock)

	// Declare 'result' variable as needed, and assign the temp-result to it.
	// i.e: `let result = $_result`
	if !exist || resultVarType == sema.VoidType {
		return modifiedStatements
	}

	var returnValueExpr ast.Expression = d.tempResultIdentifierExpr(pos)

	// If the return type is a resource, then this must be a reference expr.
	if returnType.IsResourceType() {
		referenceExpression := ast.NewReferenceExpression(d.memoryGauge, returnValueExpr, pos)
		d.elaboration.SetReferenceExpressionBorrowType(referenceExpression, resultVarType)
		returnValueExpr = referenceExpression
	}

	resultVarDecl := ast.NewVariableDeclaration(
		d.memoryGauge,
		ast.AccessNotSpecified,
		true,
		ast.NewIdentifier(
			d.memoryGauge,
			resultVariableName,
			pos,
		),

		// No need of type annotation.
		// Compiler retrieves types from the elaboration, which is updated below.
		nil,

		returnValueExpr,
		ast.NewTransfer(
			d.memoryGauge,
			// This is always a copy.
			// Because result becomes a reference, if the return type is resource.
			ast.TransferOperationCopy,
			pos,
		),
		pos,
		nil,
		nil,
		"",
	)

	d.elaboration.SetVariableDeclarationTypes(
		resultVarDecl,
		sema.VariableDeclarationTypes{
			ValueType:  resultVarType,
			TargetType: resultVarType,
		},
	)

	modifiedStatements = append(modifiedStatements, resultVarDecl)

	return modifiedStatements
}

func (d *Desugar) tempResultIdentifierExpr(pos ast.Position) *ast.IdentifierExpression {
	return ast.NewIdentifierExpression(
		d.memoryGauge,
		ast.NewIdentifier(
			d.memoryGauge,
			tempResultVariableName,
			pos,
		),
	)
}

func (d *Desugar) desugarPreConditions(
	enclosingFuncName string,
	funcBlock *ast.FunctionBlock,
) []ast.Statement {

	desugaredConditions := make([]ast.Statement, 0)

	// Desugar inherited pre-conditions
	inheritedFuncs := d.inheritedFuncsWithConditions[enclosingFuncName]
	for _, inheritedFunc := range inheritedFuncs {
		inheritedPreConditions := inheritedFunc.functionDecl.FunctionBlock.PreConditions
		if inheritedPreConditions == nil {
			continue
		}

		// If the inherited function has pre-conditions, then include them as well by copying them over.
		for _, condition := range inheritedPreConditions.Conditions {
			desugaredCondition := d.desugarInheritedCondition(condition, inheritedFunc)
			desugaredConditions = append(desugaredConditions, desugaredCondition)
		}
	}

	// Desugar self-defined pre-conditions
	var conditions *ast.Conditions
	if funcBlock != nil && funcBlock.PreConditions != nil {
		conditions = funcBlock.PreConditions

		for _, condition := range conditions.Conditions {
			desugaredCondition := d.desugarCondition(condition, nil)
			desugaredConditions = append(desugaredConditions, desugaredCondition)
		}
	}

	// If this is a method of a concrete-type then return with the updated statements,
	// and continue desugaring the rest.
	if d.includeConditions(conditions) {
		return desugaredConditions
	}

	return nil
}

func (d *Desugar) desugarPostConditions(
	enclosingFuncName string,
	funcBlock *ast.FunctionBlock,
) (desugaredConditions []ast.Statement, beforeStatements []ast.Statement) {

	desugaredConditions = make([]ast.Statement, 0)

	var conditions *ast.Conditions
	if funcBlock != nil {
		conditions = funcBlock.PostConditions
	}

	// Desugar locally-defined post-conditions
	if conditions != nil {
		postConditionsRewrite := d.elaboration.PostConditionsRewrite(conditions)
		conditionsList := postConditionsRewrite.RewrittenPostConditions

		beforeStatements = postConditionsRewrite.BeforeStatements

		for _, condition := range conditionsList {
			desugaredCondition := d.desugarCondition(condition, nil)
			desugaredConditions = append(desugaredConditions, desugaredCondition)
		}
	}

	// Desugar inherited post-conditions
	inheritedFuncs, ok := d.inheritedFuncsWithConditions[enclosingFuncName]
	if ok && len(inheritedFuncs) > 0 {
		// Must be added in reverse order.
		for i := len(inheritedFuncs) - 1; i >= 0; i-- {
			inheritedFunc := inheritedFuncs[i]
			inheritedFunctionBlock := inheritedFunc.functionDecl.FunctionBlock

			inheritedPostConditions := inheritedFunctionBlock.PostConditions
			if inheritedPostConditions == nil {
				continue
			}

			// Result variable must be looked-up in the corresponding elaboration.
			resultVarType, resultVarExist := inheritedFunc.elaboration.ResultVariableType(inheritedFunctionBlock)

			// If the inherited function has post-conditions (and before statements),
			// then include them as well, by bringing them over (inlining).

			rewrittenBeforeStatements := inheritedFunc.rewrittenConditions.BeforeStatements
			beforeStatements = append(beforeStatements, rewrittenBeforeStatements...)
			for _, statement := range rewrittenBeforeStatements {
				d.elaboration.conditionsElaborations[statement] = inheritedFunc.elaboration
			}

			for _, condition := range inheritedFunc.rewrittenConditions.RewrittenPostConditions {
				desugaredCondition := d.desugarInheritedCondition(condition, inheritedFunc)
				desugaredConditions = append(desugaredConditions, desugaredCondition)
			}

			if resultVarExist {
				d.elaboration.SetResultVariableType(funcBlock, resultVarType)
			}
		}
	}

	// If this is a method of a concrete-type then return with the updated statements,
	// and continue desugaring the rest.
	if d.includeConditions(conditions) {
		return desugaredConditions, beforeStatements
	}

	return nil, nil
}

func (d *Desugar) desugarInheritedCondition(condition ast.Condition, inheritedFunc *inheritedFunction) ast.Statement {
	// When desugaring inherited functions, use their corresponding elaboration.
	prevElaboration := d.elaboration
	d.elaboration = inheritedFunc.elaboration

	desugaredCondition := d.desugarCondition(condition, inheritedFunc.interfaceType)
	d.elaboration = prevElaboration

	// Elaboration to be used by the condition must be set in the current elaboration.
	// (Not in the inherited function's elaboration)
	d.elaboration.conditionsElaborations[desugaredCondition] = inheritedFunc.elaboration
	return desugaredCondition
}

func (d *Desugar) includeConditions(conditions *ast.Conditions) bool {
	// Conditions can be inlined if one of the conditions are satisfied:
	//  - There are no conditions
	//  - This is a method of a concrete-type (i.e: enclosingInterfaceType is `nil`)
	return conditions == nil ||
		d.enclosingInterfaceType == nil
}

var conditionFailedMessage = ast.NewStringExpression(nil, "pre/post condition failed", ast.EmptyRange)

var panicFuncInvocationTypes = sema.InvocationExpressionTypes{
	ReturnType: stdlib.PanicFunctionType.ReturnTypeAnnotation.Type,
	ArgumentTypes: []sema.Type{
		sema.StringType,
	},
}

func (d *Desugar) desugarCondition(condition ast.Condition, inheritedFrom *sema.InterfaceType) ast.Statement {

	// If the conditions are inherited, they could be referring to the imports of their original program.
	// i.e: transitive dependencies of the concrete type.
	// Therefore, add those transitive dependencies to the current compiling program.
	// They will only be added to the final compiled program, if those are used in the code.
	if inheritedFrom != nil {
		elaboration, err := d.config.ElaborationResolver(inheritedFrom.Location)
		if err != nil {
			panic(err)
		}

		allImports := elaboration.AllImportDeclarationsResolvedLocations()
		transitiveImportLocations := make([]common.Location, 0, len(allImports))

		// Collect and sort the locations to make it deterministic.
		for _, resolvedLocations := range allImports { // nolint:maprange
			for _, location := range resolvedLocations {
				transitiveImportLocations = append(transitiveImportLocations, location.Location)
			}
		}
		slices.SortFunc(transitiveImportLocations, func(a, b common.Location) int {
			return strings.Compare(a.ID(), b.ID())
		})

		// Add new imports for all the transitive imports.
		for _, location := range transitiveImportLocations {
			d.addImport(location)
		}
	}

	switch condition := condition.(type) {
	case *ast.TestCondition:

		// Desugar a test-condition to an if-statement. i.e:
		// ```
		//   pre{ x > 0: "x must be larger than zero"}
		// ```
		// is converted to:
		// ```
		//   if !(x > 0) {
		//     panic("x must be larger than zero")
		//   }
		// ```
		message := condition.Message
		if message == nil {
			message = conditionFailedMessage
		}

		startPos := condition.StartPosition()

		panicFuncInvocation := ast.NewInvocationExpression(
			d.memoryGauge,
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					commons.PanicFunctionName,
					startPos,
				),
			),
			nil,
			[]*ast.Argument{
				ast.NewUnlabeledArgument(d.memoryGauge, message),
			},
			startPos,
			condition.EndPosition(d.memoryGauge),
		)

		d.elaboration.SetInvocationExpressionTypes(panicFuncInvocation, panicFuncInvocationTypes)

		ifStmt := ast.NewIfStatement(
			d.memoryGauge,
			ast.NewUnaryExpression(
				d.memoryGauge,
				ast.OperationNegate,
				condition.Test,
				startPos,
			),
			ast.NewBlock(
				d.memoryGauge,
				[]ast.Statement{
					ast.NewExpressionStatement(
						d.memoryGauge,
						panicFuncInvocation,
					),
				},
				ast.NewRangeFromPositioned(
					d.memoryGauge,
					condition.Test,
				),
			),
			nil,
			startPos,
		)

		return ifStmt
	case *ast.EmitCondition:
		emitStmt := (*ast.EmitStatement)(condition)

		if inheritedFrom == nil {
			return emitStmt
		}

		// If the condition is inherited, then re-write
		// the emit statement to be type-qualified.
		// i.e: `emit Event()` will be re-written as `emit Contract.Event()`.
		// Otherwise, the compiler can't find the symbol.

		eventConstructorInvocation := emitStmt.InvocationExpression

		// If the event constructor is already type-qualified, then no need to change anything.
		if _, ok := eventConstructorInvocation.InvokedExpression.(*ast.MemberExpression); ok {
			return emitStmt
		}

		// Otherwise, make it type-qualified
		invocationTypes := d.elaboration.InvocationExpressionTypes(eventConstructorInvocation)

		pos := eventConstructorInvocation.StartPosition()

		// Get the contract in which the event was declared.
		// This is guaranteed, since events can only be declared in contracts.
		eventType := d.elaboration.EmitStatementEventType(emitStmt)
		declaredContract := declaredContractType(eventType).(sema.CompositeKindedType)

		memberExpression := ast.NewMemberExpression(
			d.memoryGauge,
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					declaredContract.GetIdentifier(),
					pos,
				),
			),
			false,
			pos,
			ast.NewIdentifier(
				d.memoryGauge,
				eventType.Identifier,
				pos,
			),
		)

		newEventConstructorInvocation := ast.NewInvocationExpression(
			d.memoryGauge,
			memberExpression,
			eventConstructorInvocation.TypeArguments,
			eventConstructorInvocation.Arguments,
			eventConstructorInvocation.ArgumentsStartPos,
			eventConstructorInvocation.EndPos,
		)

		newEmitStmt := ast.NewEmitStatement(
			d.memoryGauge,
			newEventConstructorInvocation,
			emitStmt.StartPos,
		)

		//Inject a static import so the compiler can link the functions.
		d.addImport(eventType.Location)

		// TODO: Is there a way to get the type for the constructor
		//  from the elaboration, rather than manually constructing it here?
		eventConstructorFuncType := sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			// Parameters are not needed, since they are not used in the compiler
			nil,
			sema.NewTypeAnnotation(eventType),
		)
		eventConstructorFuncType.IsConstructor = true

		memberAccessInfo := sema.MemberAccessInfo{
			AccessedType:  declaredContract,
			ResultingType: eventType,
			Member: sema.NewPublicFunctionMember(
				d.memoryGauge,
				declaredContract,
				eventType.Identifier,
				eventConstructorFuncType,
				"",
			),
			IsOptional:      false,
			ReturnReference: false,
		}

		d.elaboration.SetInvocationExpressionTypes(newEventConstructorInvocation, invocationTypes)
		d.elaboration.SetEmitStatementEventType(newEmitStmt, eventType)
		d.elaboration.SetMemberExpressionMemberAccessInfo(memberExpression, memberAccessInfo)

		return newEmitStmt
	default:
		panic(errors.NewUnreachableError())
	}
}

func declaredContractType(containedType sema.ContainedType) sema.Type {
	containerType := containedType.GetContainerType()
	if containerType == nil {
		return containedType
	}

	return declaredContractType(containerType.(sema.ContainedType))
}

func (d *Desugar) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) ast.Declaration {
	desugaredDecl := d.desugarDeclaration(declaration.FunctionDeclaration).(*ast.FunctionDeclaration)
	if desugaredDecl == declaration.FunctionDeclaration {
		return declaration
	}

	return ast.NewSpecialFunctionDeclaration(
		d.memoryGauge,
		declaration.Kind,
		desugaredDecl,
	)
}

func (d *Desugar) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Declaration {
	compositeType := d.elaboration.CompositeDeclarationType(declaration)

	// Recursively de-sugar nested declarations (functions, types, etc.)

	prevInheritedFuncsWithConditions := d.inheritedFuncsWithConditions
	prevEnclosingInterfaceType := d.enclosingInterfaceType

	d.inheritedFuncsWithConditions = d.inheritedFunctionsWithConditions(compositeType)
	d.enclosingInterfaceType = nil

	defer func() {
		d.inheritedFuncsWithConditions = prevInheritedFuncsWithConditions
		d.enclosingInterfaceType = prevEnclosingInterfaceType
	}()

	var desugaredMembers []ast.Declaration
	membersDesugared := false
	existingMembers := declaration.Members.Declarations()

	for _, member := range existingMembers {
		desugaredMember := d.desugarDeclaration(member)
		if desugaredMember == nil {
			continue
		}

		membersDesugared = membersDesugared || (desugaredMember != member)
		desugaredMembers = append(desugaredMembers, desugaredMember)
	}

	// Add inherited default functions.
	existingFunctions := declaration.Members.FunctionsByIdentifier()
	inheritedDefaultFuncs := d.inheritedDefaultFunctions(
		compositeType,
		existingFunctions,
		declaration.StartPos,
		declaration.Range,
	)

	// Optimization: If none of the existing members got updated or,
	// if there are no inherited members, then return the same declaration as-is.
	if !membersDesugared && len(inheritedDefaultFuncs) == 0 {
		return declaration
	}

	desugaredMembers = append(desugaredMembers, inheritedDefaultFuncs...)

	modifiedDecl := ast.NewCompositeDeclaration(
		d.memoryGauge,
		declaration.Access,
		declaration.CompositeKind,
		declaration.Identifier,
		declaration.Conformances,
		ast.NewMembers(d.memoryGauge, desugaredMembers),
		declaration.DocString,
		declaration.Range,
	)

	// Update elaboration. Type info is needed for later steps.
	d.elaboration.SetCompositeDeclarationType(modifiedDecl, compositeType)

	return modifiedDecl
}

func (d *Desugar) inheritedFunctionsWithConditions(compositeType sema.ConformingType) map[string][]*inheritedFunction {
	inheritedFunctions := make(map[string][]*inheritedFunction)

	compositeType.EffectiveInterfaceConformanceSet().ForEach(func(interfaceType *sema.InterfaceType) {

		elaboration, err := d.config.ElaborationResolver(interfaceType.Location)
		if err != nil {
			panic(err)
		}

		interfaceDecl := elaboration.InterfaceTypeDeclaration(interfaceType)
		functions := interfaceDecl.Members.FunctionsByIdentifier()

		for name, functionDecl := range functions { // nolint:maprange
			if !functionDecl.FunctionBlock.HasConditions() {
				continue
			}
			funcs := inheritedFunctions[name]

			postConditions := functionDecl.FunctionBlock.PostConditions
			var rewrittenConditions sema.PostConditionsRewrite
			if postConditions != nil {
				rewrittenConditions = elaboration.PostConditionsRewrite(postConditions)
			}

			funcs = append(funcs, &inheritedFunction{
				interfaceType:       interfaceType,
				functionDecl:        functionDecl,
				rewrittenConditions: rewrittenConditions,
				elaboration:         NewExtendedElaboration(elaboration),
			})
			inheritedFunctions[name] = funcs
		}
	})

	return inheritedFunctions
}

func (d *Desugar) inheritedDefaultFunctions(
	compositeType sema.ConformingType,
	existingFunctions map[string]*ast.FunctionDeclaration,
	pos ast.Position,
	declRange ast.Range,
) []ast.Declaration {

	inheritedDefaultFunctions := make(map[string]struct{})

	inheritedMembers := make([]ast.Declaration, 0)

	for _, conformance := range compositeType.EffectiveInterfaceConformances() {
		interfaceType := conformance.InterfaceType

		elaboration, err := d.config.ElaborationResolver(interfaceType.Location)
		if err != nil {
			panic(err)
		}

		interfaceDecl := elaboration.InterfaceTypeDeclaration(interfaceType)
		functions := interfaceDecl.Members.FunctionsByIdentifier()

		for funcName, inheritedFunc := range functions { // nolint:maprange
			if !inheritedFunc.FunctionBlock.HasStatements() {
				continue
			}

			// Pick the 'closest' default function.
			// This is the same way how it is implemented in the interpreter.
			_, ok := inheritedDefaultFunctions[funcName]
			if ok {
				continue
			}
			inheritedDefaultFunctions[funcName] = struct{}{}

			// If the inherited function is overridden by the current type, then skip.
			if d.isFunctionOverridden(compositeType, funcName, existingFunctions) {
				continue
			}

			// For each inherited function, generate a delegator function,
			// which calls the actual default implementation at the interface.
			// i.e:
			//  FooImpl {
			//    fun defaultFunc(a1: T1, a2: T2): R {
			//        return FooInterface.defaultFunc(a1, a2)
			//    }
			//  }

			// Generate: `FooInterface.defaultFunc(a1, a2)`

			inheritedFuncType := elaboration.FunctionDeclarationFunctionType(inheritedFunc)

			member, ok := interfaceType.MemberMap().Get(funcName)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			invocation := d.interfaceDelegationMethodCall(
				interfaceType,
				inheritedFuncType,
				pos,
				funcName,
				member,
				nil,
			)

			funcReturnType := inheritedFuncType.ReturnTypeAnnotation.Type
			returnStmt := ast.NewReturnStatement(d.memoryGauge, invocation, declRange)
			d.elaboration.SetReturnStatementTypes(
				returnStmt,
				sema.ReturnStatementTypes{
					ValueType:  funcReturnType,
					ReturnType: funcReturnType,
				},
			)

			// Generate: `fun defaultFunc(a1: T1, a2: T2) { ... }`
			defaultFuncDelegator := ast.NewFunctionDeclaration(
				d.memoryGauge,
				inheritedFunc.Access,
				inheritedFunc.Purity,
				inheritedFunc.IsStatic(),
				inheritedFunc.IsNative(),
				ast.NewIdentifier(
					d.memoryGauge,
					funcName,
					pos,
				),
				inheritedFunc.TypeParameterList,
				inheritedFunc.ParameterList,
				inheritedFunc.ReturnTypeAnnotation,
				ast.NewFunctionBlock(
					d.memoryGauge,
					ast.NewBlock(
						d.memoryGauge,
						[]ast.Statement{
							returnStmt,
						},
						declRange,
					),
					nil,
					nil,
				),
				inheritedFunc.StartPos,
				inheritedFunc.DocString,
			)

			d.elaboration.SetFunctionDeclarationFunctionType(defaultFuncDelegator, inheritedFuncType)

			// Pass the generated default function again through the desugar phase,
			// so that it will properly link/chain the function conditions
			// that are inherited/available for this default function.
			desugaredDelegator := d.desugarDeclaration(defaultFuncDelegator)

			inheritedMembers = append(inheritedMembers, desugaredDelegator)

		}
	}

	return inheritedMembers
}

func (d *Desugar) isFunctionOverridden(
	enclosingType sema.ConformingType,
	funcName string,
	existingFunctions map[string]*ast.FunctionDeclaration,
) bool {
	implementedFunc, isImplemented := existingFunctions[funcName]
	if !isImplemented {
		return false
	}

	_, isInterface := enclosingType.(*sema.InterfaceType)
	if isInterface {
		// If the currently visiting declaration is an interface type (i.e: This function is an interface method)
		// then it is considered as a default implementation only if there are statements.
		// This is because interface methods can define conditions, without overriding the function.
		return implementedFunc.FunctionBlock.HasStatements()
	}

	return true
}

func (d *Desugar) interfaceDelegationMethodCall(
	interfaceType *sema.InterfaceType,
	inheritedFuncType *sema.FunctionType,
	pos ast.Position,
	functionName string,
	member *sema.Member,
	extraArguments []ast.Expression,
) *ast.InvocationExpression {

	arguments := make([]*ast.Argument, 0)
	for _, param := range inheritedFuncType.Parameters {
		var arg *ast.Argument
		if param.Label == "_" {
			arg = ast.NewUnlabeledArgument(
				d.memoryGauge,
				ast.NewIdentifierExpression(
					d.memoryGauge,
					ast.NewIdentifier(
						d.memoryGauge,
						param.Identifier,
						pos,
					),
				),
			)
		} else {
			var label string
			if param.Label == "" {
				label = param.Identifier
			} else {
				label = param.Label
			}

			arg = ast.NewArgument(
				d.memoryGauge,
				label,
				&pos,
				&pos,
				ast.NewIdentifierExpression(
					d.memoryGauge,
					ast.NewIdentifier(
						d.memoryGauge,
						param.Identifier,
						pos,
					),
				),
			)
		}
		arguments = append(arguments, arg)
	}

	for _, argument := range extraArguments {
		arg := ast.NewUnlabeledArgument(
			d.memoryGauge,
			argument,
		)

		arguments = append(arguments, arg)
	}

	// `FooInterface.defaultFunc(a1, a2)`
	//
	// However, when generating code, we need to load "self" as the receiver,
	// and call the interface's function.
	// This is done by setting the invoked identifier as 'self',
	// but setting interface-type as the "AccessedType" (in AccessedType).
	invokedExpr := ast.NewMemberExpression(
		d.memoryGauge,
		ast.NewIdentifierExpression(
			d.memoryGauge,

			ast.NewIdentifier(
				d.memoryGauge,
				"self",
				pos,
			),
		),
		false,
		pos,
		ast.NewIdentifier(
			d.memoryGauge,
			functionName,
			pos,
		),
	)

	invocation := ast.NewInvocationExpression(
		d.memoryGauge,
		invokedExpr,
		nil,
		arguments,
		pos,
		pos,
	)

	funcType, ok := member.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	invocationTypes := sema.InvocationExpressionTypes{
		ReturnType:    funcType.ReturnTypeAnnotation.Type,
		ArgumentTypes: funcType.ParameterTypes(),
	}

	memberAccessInfo := sema.MemberAccessInfo{
		AccessedType:    interfaceType,
		ResultingType:   funcType,
		Member:          member,
		IsOptional:      false,
		ReturnReference: false,
	}

	d.elaboration.SetInvocationExpressionTypes(invocation, invocationTypes)
	d.elaboration.SetMemberExpressionMemberAccessInfo(invokedExpr, memberAccessInfo)
	d.elaboration.SetInterfaceMethodStaticCall(invocation)

	// Given these invocations are treated as static calls,
	// we need to inject a static import as well, so the
	// compiler can link these functions.
	d.addImport(interfaceType.Location)

	return invocation
}

func (d *Desugar) addImport(location common.Location) {
	// If the import is for the same program, then do not add any new imports.
	if location == d.checker.Location {
		return
	}

	switch location := location.(type) {
	case common.AddressLocation:
		_, exists := d.importsSet[location]
		if exists {
			return
		}

		d.newImports = append(
			d.newImports,
			ast.NewImportDeclaration(
				d.memoryGauge,
				[]ast.Identifier{
					ast.NewIdentifier(d.memoryGauge, location.Name, ast.EmptyPosition),
				},
				location,
				ast.EmptyRange,
				ast.EmptyPosition,
			))

		d.importsSet[location] = struct{}{}
	default:
		panic(errors.NewUnreachableError())
	}
}

func (d *Desugar) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) ast.Declaration {
	interfaceType := d.elaboration.InterfaceDeclarationType(declaration)

	prevEnclosingInterfaceType := d.enclosingInterfaceType
	d.enclosingInterfaceType = interfaceType
	defer func() {
		d.enclosingInterfaceType = prevEnclosingInterfaceType
	}()

	// Recursively de-sugar nested declarations (functions, types, etc.)

	var desugaredMembers []ast.Declaration

	existingMembers := declaration.Members.Declarations()
	for _, member := range existingMembers {
		desugaredMember := d.desugarDeclaration(member)
		if desugaredMember == nil {
			continue
		}
		desugaredMembers = append(desugaredMembers, desugaredMember)
	}

	// TODO: Optimize: If none of the existing members got updated or,
	// if there are no inherited members, then return the same declaration as-is.

	modifiedDecl := ast.NewInterfaceDeclaration(
		d.memoryGauge,
		declaration.Access,
		declaration.CompositeKind,
		declaration.Identifier,
		declaration.Conformances,
		ast.NewMembers(d.memoryGauge, desugaredMembers),
		declaration.DocString,
		declaration.Range,
	)

	// Update elaboration. Type info is needed for later steps.
	d.elaboration.SetInterfaceDeclarationType(modifiedDecl, interfaceType)

	return modifiedDecl
}

func (d *Desugar) VisitEntitlementDeclaration(declaration *ast.EntitlementDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitEntitlementMappingDeclaration(declaration *ast.EntitlementMappingDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitTransactionDeclaration(transaction *ast.TransactionDeclaration) ast.Declaration {
	// Converts a transaction into a composite type declaration.
	// Transaction parameters are converted into global variables.
	// An initializer is generated to set parameters to above generated global variables.

	var varDeclarations []ast.Declaration
	var initFunction *ast.FunctionDeclaration

	transactionTypes := d.elaboration.TransactionDeclarationType(transaction)

	if transaction.ParameterList != nil {
		varDeclarations = make([]ast.Declaration, 0, len(transaction.ParameterList.Parameters))
		statements := make([]ast.Statement, 0, len(transaction.ParameterList.Parameters))
		parameters := make([]*ast.Parameter, 0, len(transaction.ParameterList.Parameters))

		for index, parameter := range transaction.ParameterList.Parameters {
			// Create global variables
			// i.e: `var a: Type`
			field := ast.NewVariableDeclaration(
				d.memoryGauge,
				ast.AccessSelf,
				false,
				parameter.Identifier,
				parameter.TypeAnnotation,
				nil,
				nil,
				parameter.StartPos,
				nil,
				nil,
				"",
			)

			varDeclarations = append(varDeclarations, field)

			// Create assignment from param to global var.
			// i.e: `a = $param_a`
			modifiedParamName := commons.TransactionGeneratedParamPrefix + parameter.Identifier.Identifier

			modifiedParameter := ast.NewParameter(
				d.memoryGauge,
				"",
				ast.NewIdentifier(
					d.memoryGauge,
					modifiedParamName,
					parameter.StartPos,
				),
				parameter.TypeAnnotation,
				nil,
				parameter.StartPos,
			)

			parameters = append(parameters, modifiedParameter)

			assignment := ast.NewAssignmentStatement(
				d.memoryGauge,
				ast.NewIdentifierExpression(
					d.memoryGauge,
					parameter.Identifier,
				),
				ast.NewTransfer(
					d.memoryGauge,
					ast.TransferOperationCopy,
					parameter.StartPos,
				),
				ast.NewIdentifierExpression(
					d.memoryGauge,
					ast.NewIdentifier(
						d.memoryGauge,
						modifiedParamName,
						parameter.StartPos,
					),
				),
			)

			statements = append(statements, assignment)

			paramType := transactionTypes.Parameters[index].TypeAnnotation.Type
			assignmentTypes := sema.AssignmentStatementTypes{
				ValueType:  paramType,
				TargetType: paramType,
			}

			d.elaboration.SetAssignmentStatementTypes(assignment, assignmentTypes)
		}

		// Create an init function.
		// func $init($param_a: Type, $param_b: Type, ...) {
		//     a = $param_a
		//     b = $param_b
		//     ...
		// }
		initFunction = simpleFunctionDeclaration(
			d.memoryGauge,
			commons.ProgramInitFunctionName,
			parameters,
			statements,
			transaction.StartPos,
			transaction.Range,
		)

		initFunctionType := sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			transactionTypes.Parameters,
			sema.VoidTypeAnnotation,
		)

		d.elaboration.SetFunctionDeclarationFunctionType(initFunction, initFunctionType)
	}

	var members []ast.Declaration

	prepareBlock := transaction.Prepare
	if prepareBlock != nil {
		prepareFunction := d.desugarDeclaration(prepareBlock.FunctionDeclaration)
		members = append(members, prepareFunction)
	}

	var executeFunc *ast.FunctionDeclaration
	if transaction.Execute != nil {
		executeFunc = transaction.Execute.FunctionDeclaration
	}

	preConditions := transaction.PreConditions
	postConditions := transaction.PostConditions

	// If there are pre/post conditions,
	// add them to the execute function.
	if preConditions != nil || postConditions != nil {
		if executeFunc == nil {
			// If there is no execute block, create an empty one.
			executeFunc = simpleFunctionDeclaration(
				d.memoryGauge,
				commons.ExecuteFunctionName,
				nil,
				nil,
				transaction.StartPos,
				transaction.Range,
			)

			d.elaboration.SetFunctionDeclarationFunctionType(executeFunc, executeFuncType)
		}

		executeFunc.FunctionBlock.PreConditions = preConditions
		executeFunc.FunctionBlock.PostConditions = postConditions
	}

	// Then desugar the execute function so that the conditions will be
	// inlined to the start and end of the execute function body.
	if executeFunc != nil {
		desugaredExecuteFunc := d.desugarDeclaration(executeFunc)
		members = append(members, desugaredExecuteFunc)
	}

	compositeDecl := ast.NewCompositeDeclaration(
		d.memoryGauge,
		ast.AccessNotSpecified,
		common.CompositeKindStructure,
		ast.NewIdentifier(
			d.memoryGauge,
			commons.TransactionWrapperCompositeName,
			ast.EmptyPosition,
		),
		nil,
		ast.NewMembers(d.memoryGauge, members),
		"",
		ast.EmptyRange,
	)

	d.elaboration.SetCompositeDeclarationType(compositeDecl, transactionCompositeType)

	// We can only return one declaration.
	// So manually add the rest of the declarations.
	d.modifiedDeclarations = append(d.modifiedDeclarations, varDeclarations...)
	if initFunction != nil {
		d.modifiedDeclarations = append(d.modifiedDeclarations, initFunction)
	}

	return compositeDecl
}

func (d *Desugar) VisitFieldDeclaration(declaration *ast.FieldDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitEnumCaseDeclaration(declaration *ast.EnumCaseDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitPragmaDeclaration(declaration *ast.PragmaDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitImportDeclaration(declaration *ast.ImportDeclaration) ast.Declaration {
	resolvedLocations, err := commons.ResolveLocation(
		d.config.LocationHandler,
		declaration.Identifiers,
		declaration.Location,
	)
	if err != nil {
		panic(err)
	}

	for _, resolvedLocation := range resolvedLocations {
		location := resolvedLocation.Location
		_, exists := d.importsSet[location]
		if exists {
			return nil
		}

		d.importsSet[location] = struct{}{}
	}

	return declaration
}

var emptyInitializer = func() *ast.SpecialFunctionDeclaration {
	// This is created only once per compilation. So no need to meter memory.

	initializer := ast.NewFunctionDeclaration(
		nil,
		ast.AccessNotSpecified,
		ast.FunctionPurityUnspecified,
		false,
		false,
		ast.NewIdentifier(
			nil,
			commons.InitFunctionName,
			ast.EmptyPosition,
		),
		nil,
		nil,
		nil,
		ast.NewFunctionBlock(
			nil,
			ast.NewBlock(nil, nil, ast.EmptyRange),
			nil,
			nil,
		),
		ast.Position{},
		"",
	)

	return ast.NewSpecialFunctionDeclaration(
		nil,
		common.DeclarationKindInitializer,
		initializer,
	)
}()

var emptyInitializerFuncType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.VoidTypeAnnotation,
)

func simpleFunctionDeclaration(
	memoryGauge common.MemoryGauge,
	functionName string,
	parameters []*ast.Parameter,
	statements []ast.Statement,
	startPos ast.Position,
	astRange ast.Range,
) *ast.FunctionDeclaration {

	var paramList *ast.ParameterList
	if parameters != nil {
		paramList = ast.NewParameterList(
			memoryGauge,
			parameters,
			astRange,
		)
	}

	return ast.NewFunctionDeclaration(
		memoryGauge,
		ast.AccessNotSpecified,
		ast.FunctionPurityUnspecified,
		false,
		false,
		ast.NewIdentifier(
			memoryGauge,
			functionName,
			startPos,
		),
		nil,
		paramList,
		nil,
		ast.NewFunctionBlock(
			memoryGauge,
			ast.NewBlock(
				memoryGauge,
				statements,
				astRange,
			),
			nil,
			nil,
		),
		startPos,
		"",
	)
}

var transactionCompositeType = &sema.CompositeType{
	Location:    nil,
	Identifier:  commons.TransactionWrapperCompositeName,
	Kind:        common.CompositeKindStructure,
	NestedTypes: &sema.StringTypeOrderedMap{},
	Members:     &sema.StringMemberOrderedMap{},
}

var executeFuncType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.VoidTypeAnnotation,
)
