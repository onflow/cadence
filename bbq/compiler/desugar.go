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
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

const resultVariableName = "result"
const tempResultVariableName = "$_result"

// Desugar will rewrite the AST from high-level abstractions to a much lower-level
// abstractions, so the compiler and vm could work with a minimal set of language features.
type Desugar struct {
	memoryGauge            common.MemoryGauge
	elaboration            *DesugaredElaboration
	program                *ast.Program
	location               common.Location
	config                 *Config
	enclosingInterfaceType *sema.InterfaceType

	modifiedDeclarations           []ast.Declaration
	inheritedFuncsWithConditions   map[string][]*inheritedFunction
	inheritedEvents                map[sema.Type][]*inheritedEvent
	postConditionIndices           map[*ast.FunctionBlock]int
	inheritedConditionParamBinding map[ast.Statement]map[string]string
	isInheritedFunction            bool

	importsSet map[common.Location]struct{}
	newImports []ast.Declaration
}

type inheritedFunction struct {
	interfaceType            *sema.InterfaceType
	functionDecl             *ast.FunctionDeclaration
	rewrittenConditions      sema.PostConditionsRewrite
	elaboration              *DesugaredElaboration
	hasDefaultImplementation bool
}

type inheritedEvent struct {
	enclosingType    sema.Type
	eventDeclaration *ast.CompositeDeclaration
	eventType        *sema.CompositeType
}

var _ ast.DeclarationVisitor[ast.Declaration] = &Desugar{}

func NewDesugar(
	memoryGauge common.MemoryGauge,
	compilerConfig *Config,
	program *ast.Program,
	elaboration *DesugaredElaboration,
	location common.Location,
) *Desugar {
	return &Desugar{
		memoryGauge:                    memoryGauge,
		config:                         compilerConfig,
		elaboration:                    elaboration,
		program:                        program,
		location:                       location,
		importsSet:                     map[common.Location]struct{}{},
		inheritedFuncsWithConditions:   map[string][]*inheritedFunction{},
		postConditionIndices:           map[*ast.FunctionBlock]int{},
		inheritedConditionParamBinding: map[ast.Statement]map[string]string{},
	}
}

type DesugaredProgram struct {
	program                        *ast.Program
	postConditionIndices           map[*ast.FunctionBlock]int
	inheritedConditionParamBinding map[ast.Statement]map[string]string
}

// Run desugars and rewrites the top-level declarations.
// It will not desugar/rewrite any statements or expressions.
func (d *Desugar) Run() DesugaredProgram {
	declarations := d.program.Declarations()
	for _, declaration := range declarations {
		modifiedDeclaration, _ := d.desugarDeclaration(declaration)
		if modifiedDeclaration != nil {
			d.modifiedDeclarations = append(d.modifiedDeclarations, modifiedDeclaration)
		}
	}

	d.modifiedDeclarations = append(d.newImports, d.modifiedDeclarations...)

	program := ast.NewProgram(d.memoryGauge, d.modifiedDeclarations)

	//fmt.Println(ast.Prettier(program))

	return DesugaredProgram{
		program:                        program,
		postConditionIndices:           d.postConditionIndices,
		inheritedConditionParamBinding: d.inheritedConditionParamBinding,
	}
}

func (d *Desugar) desugarDeclaration(declaration ast.Declaration) (result ast.Declaration, desugared bool) {
	desugaredDeclaration := ast.AcceptDeclaration[ast.Declaration](declaration, d)
	desugared = desugaredDeclaration != declaration
	return desugaredDeclaration, desugared
}

func desugarList[T any](
	list []T,
	listEntryDesugarFunction func(entry T) (desugaredEntry T, desugared bool),
) (desugaredList []T, desugared bool) {

	for index, entry := range list {
		desugaredEntry, ok := listEntryDesugarFunction(entry)

		// Below is an optimization to only create a new slice, if at-least one of the entries is desugared.
		// i.e: If none of the entries need desugaring, then return the slice as-is.

		// If at-least one entry is desugared already, then add the current entry
		// to the desugared entries list (regardless whether the current entry was desugared or not).
		if desugared {
			desugaredList = append(desugaredList, desugaredEntry)
			continue
		}

		// If the current entry is also not desugared, then continue.
		if !ok {
			continue
		}

		// Otherwise, if the current entry is desugared (meaning, this is the first desugared entry),
		// then add the original entries upto this point, and then add the current desugared entry.
		desugared = true
		desugaredList = append(desugaredList, list[:index]...)

		if any(desugaredEntry) != nil {
			desugaredList = append(desugaredList, desugaredEntry)
		}
	}

	if !desugared {
		desugaredList = list
	}

	return
}

func (d *Desugar) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration, _ bool) ast.Declaration {
	funcBlock := declaration.FunctionBlock
	funcName := declaration.Identifier.Identifier
	parameterList := declaration.ParameterList
	returnTypeAnnotation := declaration.ReturnTypeAnnotation

	functionType := d.elaboration.FunctionDeclarationFunctionType(declaration)

	// If this is an interface-method without a body,
	// then do not generate a function for it.
	if d.enclosingInterfaceType != nil && !funcBlock.HasStatements() {
		return nil
	}

	modifiedFuncBlock := d.desugarFunctionBlock(
		funcName,
		funcBlock,
		parameterList,
		functionType,
		returnTypeAnnotation,
		declaration,
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

func (d *Desugar) desugarFunctionBlock(
	funcName string,
	funcBlock *ast.FunctionBlock,
	parameterList *ast.ParameterList,
	functionType *sema.FunctionType,
	returnTypeAnnotation *ast.TypeAnnotation,
	hasPosition ast.HasPosition,
) *ast.FunctionBlock {

	preConditions := d.desugarPreConditions(
		funcName,
		funcBlock,
		parameterList,
	)
	postConditions, beforeStatements := d.desugarPostConditions(
		funcName,
		funcBlock,
		parameterList,
	)

	modifiedStatements := make([]ast.Statement, 0)
	modifiedStatements = append(modifiedStatements, preConditions...)

	modifiedStatements = append(modifiedStatements, beforeStatements...)

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
			tempResultVarDecl := d.tempResultVariable(
				returnTypeAnnotation,
				returnType,
				pos,
			)
			modifiedStatements = append(modifiedStatements, tempResultVarDecl)
		}

		// Add the remaining statements that are defined in this function.
		statements := funcBlock.Block.Statements
		modifiedStatements = append(modifiedStatements, statements...)
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
			ast.NewRangeFromPositioned(d.memoryGauge, hasPosition),
		),
		nil,
		nil,
	)

	return modifiedFuncBlock
}

// Creates a `$_result` synthetic variable, to temporarily hold return values.
func (d *Desugar) tempResultVariable(
	returnTypeAnnotation *ast.TypeAnnotation,
	returnType sema.Type,
	pos ast.Position,
) *ast.VariableDeclaration {
	tempResultVarDecl := ast.NewVariableDeclaration(
		d.memoryGauge,
		ast.AccessNotSpecified,
		false,
		ast.NewIdentifier(
			d.memoryGauge,
			tempResultVariableName,
			pos,
		),
		returnTypeAnnotation,

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

	return tempResultVarDecl
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
	parameterList *ast.ParameterList,
) []ast.Statement {

	desugaredConditions := make([]ast.Statement, 0)

	functionHasImpl := d.functionHasImplementation(funcBlock)

	// Desugar inherited pre-conditions
	inheritedFuncs := d.inheritedFuncsWithConditions[enclosingFuncName]
	for _, inheritedFunc := range inheritedFuncs {
		if functionHasImpl && inheritedFunc.hasDefaultImplementation {
			// If the current function has an implementation AND the inherited function
			// also has an implementation, then the inherited function is considered to
			// be overwritten.
			// Thus, the inherited condition also considered overwritten, and hence do not include it.
			continue
		}

		inheritedPreConditions := inheritedFunc.functionDecl.FunctionBlock.PreConditions
		if inheritedPreConditions == nil {
			continue
		}

		// If the inherited function has pre-conditions, then include them as well by copying them over.
		for _, condition := range inheritedPreConditions.Conditions {
			desugaredCondition := d.desugarInheritedCondition(
				condition,
				parameterList,
				inheritedFunc,
			)
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

func (d *Desugar) functionHasImplementation(funcBlock *ast.FunctionBlock) bool {
	// Current function has an implementations only if it:
	// 1) Is not an inherited function (i.e: implementation must not be an inherited one), AND
	// 2) The function has statements.
	return !d.isInheritedFunction && funcBlock.HasStatements()
}

func (d *Desugar) desugarPostConditions(
	enclosingFuncName string,
	funcBlock *ast.FunctionBlock,
	parameterList *ast.ParameterList,
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

	functionHasImpl := d.functionHasImplementation(funcBlock)

	// Desugar inherited post-conditions
	inheritedFuncs, ok := d.inheritedFuncsWithConditions[enclosingFuncName]
	if ok && len(inheritedFuncs) > 0 {
		// Must be added in reverse order.
		for i := len(inheritedFuncs) - 1; i >= 0; i-- {
			inheritedFunc := inheritedFuncs[i]
			if functionHasImpl && inheritedFunc.hasDefaultImplementation {
				// If the current function has an implementation AND the inherited function
				// also has an implementation, then the inherited function is considered to
				// be overwritten.
				// Thus, the inherited condition also considered overwritten, and hence do not include it.
				continue
			}

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
				desugaredCondition := d.desugarInheritedCondition(
					condition,
					parameterList,
					inheritedFunc,
				)
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

func (d *Desugar) desugarInheritedCondition(
	condition ast.Condition,
	functionParams *ast.ParameterList,
	inheritedFunc *inheritedFunction,
) ast.Statement {
	// When desugaring inherited functions, use their corresponding elaboration.
	prevElaboration := d.elaboration
	d.elaboration = inheritedFunc.elaboration

	desugaredCondition := d.desugarCondition(condition, inheritedFunc.interfaceType)
	d.elaboration = prevElaboration

	// Elaboration to be used by the condition must be set in the current elaboration.
	// (Not in the inherited function's elaboration)
	d.elaboration.conditionsElaborations[desugaredCondition] = inheritedFunc.elaboration

	if len(functionParams.Parameters) > 0 {
		paramBinding := make(map[string]string)

		inheritedFunctionParams := inheritedFunc.functionDecl.ParameterList
		for i, parameter := range inheritedFunctionParams.Parameters {
			currentFunctionParamName := functionParams.Parameters[i].Identifier.Identifier
			inheritedFunctionParamName := parameter.Identifier.Identifier
			paramBinding[inheritedFunctionParamName] = currentFunctionParamName
		}

		d.inheritedConditionParamBinding[desugaredCondition] = paramBinding
	}

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
	ArgumentTypes: []sema.Type{
		sema.StringType,
	},
	ParameterTypes: []sema.Type{
		sema.StringType,
	},
	ReturnType: sema.NeverType,
}

func (d *Desugar) desugarCondition(condition ast.Condition, inheritedFrom *sema.InterfaceType) ast.Statement {

	// If the conditions are inherited, they could be referring to the imports of their original program.
	// i.e: transitive dependencies of the concrete type.
	// Therefore, add those transitive dependencies to the current compiling program.
	// They will only be added to the final compiled program, if those are used in the code.
	if inheritedFrom != nil {
		elaboration, err := d.resolveElaboration(inheritedFrom.Location)
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

		// Get the contract in which the event was declared.
		eventType := d.elaboration.EmitStatementEventType(emitStmt)
		declaredContract := declaredContractType(eventType)

		// If the event is not declared in a contract, then it's a local type (e.g: in a script).
		// Ne need to change anything in that case.
		if declaredContract == nil {
			return emitStmt
		}

		// Otherwise, if the condition is inherited from a contract, then re-write
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

func (d *Desugar) resolveElaboration(location common.Location) (*DesugaredElaboration, error) {
	if location == d.location {
		return d.elaboration, nil
	}

	return d.config.ElaborationResolver(location)
}

func declaredContractType(compositeKindedType sema.CompositeKindedType) sema.CompositeKindedType {
	if compositeKindedType.GetCompositeKind() == common.CompositeKindContract {
		return compositeKindedType
	}

	containerType := compositeKindedType.GetContainerType()
	if containerType == nil {
		return nil
	}

	return declaredContractType(containerType.(sema.CompositeKindedType))
}

func (d *Desugar) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) ast.Declaration {
	desugaredDecl, desugared := d.desugarDeclaration(declaration.FunctionDeclaration)
	if desugaredDecl == nil {
		return nil
	}

	if !desugared {
		return declaration
	}

	desugaredFunctionDecl := desugaredDecl.(*ast.FunctionDeclaration)

	return ast.NewSpecialFunctionDeclaration(
		d.memoryGauge,
		declaration.Kind,
		desugaredFunctionDecl,
	)
}

func (d *Desugar) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Declaration {
	compositeType := d.elaboration.CompositeDeclarationType(declaration)

	// Recursively de-sugar nested declarations (functions, types, etc.)

	prevInheritedFuncsWithConditions := d.inheritedFuncsWithConditions
	prevInheritedEvents := d.inheritedEvents
	prevEnclosingInterfaceType := d.enclosingInterfaceType

	d.inheritedFuncsWithConditions, d.inheritedEvents = d.inheritedFunctionsWithConditionsAndEvents(compositeType)
	d.enclosingInterfaceType = nil

	defer func() {
		d.inheritedFuncsWithConditions = prevInheritedFuncsWithConditions
		d.inheritedEvents = prevInheritedEvents
		d.enclosingInterfaceType = prevEnclosingInterfaceType
	}()

	var desugaredMembers []ast.Declaration
	membersDesugared := false

	// If the declaration is the default-destroy event, then generate a synthetic-constructor,
	// so that it can be constructed externally.
	// Default-destroy event can only have the constructor. So no need to visit other members.
	if declaration.IsResourceDestructionDefaultEvent() {
		initializer := constructorFunction(declaration)
		desugaredMember := d.desugarDefaultDestroyEventInitializer(compositeType, initializer)
		desugaredMembers = append(desugaredMembers, desugaredMember)
		membersDesugared = true
	} else {
		// Otherwise, visit and desugar members.
		existingMembers := declaration.Members.Declarations()
		for _, member := range existingMembers {
			desugaredMember, desugared := d.desugarDeclaration(member)
			if desugaredMember == nil {
				continue
			}

			membersDesugared = membersDesugared || desugared
			desugaredMembers = append(desugaredMembers, desugaredMember)
		}
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

	// Generate a getter-function, that will construct and return
	// all resource-destroyed events, including the inherited ones.
	destroyEventsReturningFunction := d.generateResourceDestroyedEventsGetterFunction(
		compositeType,
		declaration,
	)
	if destroyEventsReturningFunction != nil {
		desugaredMembers = append(desugaredMembers, destroyEventsReturningFunction)
	}

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

func (d *Desugar) inheritedFunctionsWithConditionsAndEvents(compositeType sema.ConformingType) (
	map[string][]*inheritedFunction,
	map[sema.Type][]*inheritedEvent,
) {
	inheritedFunctions := make(map[string][]*inheritedFunction)
	inheritedEvents := make(map[sema.Type][]*inheritedEvent)

	addInheritedFunction := func(
		functionDecl *ast.FunctionDeclaration,
		elaboration *DesugaredElaboration,
		interfaceType *sema.InterfaceType,
	) {
		functionBlock := functionDecl.FunctionBlock
		if !functionBlock.HasConditions() {
			return
		}

		name := functionDecl.Identifier.Identifier
		funcs := inheritedFunctions[name]

		postConditions := functionBlock.PostConditions
		var rewrittenConditions sema.PostConditionsRewrite
		if postConditions != nil {
			rewrittenConditions = elaboration.PostConditionsRewrite(postConditions)
		}

		funcs = append(funcs, &inheritedFunction{
			interfaceType:            interfaceType,
			functionDecl:             functionDecl,
			rewrittenConditions:      rewrittenConditions,
			elaboration:              elaboration,
			hasDefaultImplementation: functionBlock.HasStatements(),
		})
		inheritedFunctions[name] = funcs
	}

	compositeType.EffectiveInterfaceConformanceSet().ForEach(func(interfaceType *sema.InterfaceType) {
		location := interfaceType.Location

		// Built-in interface-types (e.g: `StructStringer`) don't have code/elaborations.
		// Therefore, skip them as they don't have any conditions.
		if location == nil {
			return
		}

		elaboration, err := d.resolveElaboration(location)
		if err != nil {
			panic(err)
		}

		interfaceDecl := elaboration.InterfaceTypeDeclaration(interfaceType)

		functions := interfaceDecl.Members.Functions()
		for _, functionDecl := range functions {
			addInheritedFunction(functionDecl, elaboration, interfaceType)
		}

		// Special functions (e.g: initializer can also have conditions)
		specialFunctions := interfaceDecl.Members.SpecialFunctions()
		for _, specialFunctionDecl := range specialFunctions {
			functionDecl := specialFunctionDecl.FunctionDeclaration
			addInheritedFunction(functionDecl, elaboration, interfaceType)
		}

		defaultDestroyEvent := elaboration.DefaultDestroyDeclaration(interfaceDecl)
		eventType := elaboration.CompositeDeclarationType(defaultDestroyEvent)
		if defaultDestroyEvent != nil {
			events := inheritedEvents[compositeType]
			events = append(events, &inheritedEvent{
				enclosingType:    interfaceType,
				eventDeclaration: defaultDestroyEvent,
				eventType:        eventType,
			})
			inheritedEvents[compositeType] = events
		}
	})

	return inheritedFunctions, inheritedEvents
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
		location := interfaceType.Location

		// Built-in interface-types (e.g: `StructStringer`) don't have code/elaborations.
		// Therefore, skip them as they don't have any default functions.
		if location == nil {
			continue
		}

		elaboration, err := d.resolveElaboration(location)
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
			desugaredDelegator := d.desugarInheritedFunction(defaultFuncDelegator)

			inheritedMembers = append(inheritedMembers, desugaredDelegator)
		}
	}

	return inheritedMembers
}

func (d *Desugar) desugarInheritedFunction(defaultFuncDelegator *ast.FunctionDeclaration) ast.Declaration {
	d.isInheritedFunction = true
	defer func() {
		d.isInheritedFunction = false
	}()

	functionDecl, _ := d.desugarDeclaration(defaultFuncDelegator)
	return functionDecl
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

	parameterTypes := funcType.ParameterTypes()

	invocationTypes := sema.InvocationExpressionTypes{
		ReturnType:     funcType.ReturnTypeAnnotation.Type,
		ParameterTypes: parameterTypes,
		// The argument types are the same as the parameter types.
		// We are simply performing a delegation invocation, inside the synthetic-wrapper function we created.
		// For example:
		//
		//    fun defaultFunc(a1: T1, a2: T2): R {
		//        return Interface.defaultFunc(a1, a2)
		//    }
		//
		// Where `defaultFunc` wrapper is created using the same parameter types as `Interface.defaultFunc`.
		// So the argument types to the invocation of `Interface.defaultFunc` are the same
		// as the parameter types of `defaultFunc`/`Interface.defaultFunc`.
		ArgumentTypes: parameterTypes,
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
	if location == d.location {
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

	existingMembers := declaration.Members.Declarations()

	desugaredMembers, desugared := desugarList(existingMembers, d.desugarDeclaration)
	if !desugared {
		return declaration
	}

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
			variableDecl := ast.NewVariableDeclaration(
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

			varDeclarations = append(varDeclarations, variableDecl)

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
		prepareFunction, _ := d.desugarDeclaration(prepareBlock.FunctionDeclaration)
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
		desugaredExecuteFunc, _ := d.desugarDeclaration(executeFunc)
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

	compositeType := d.transactionCompositeType()
	d.elaboration.SetCompositeDeclarationType(compositeDecl, compositeType)
	d.elaboration.SetCompositeType(compositeType.ID(), compositeType)

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

func (d *Desugar) DesugarInnerFunction(declaration *ast.FunctionDeclaration) *ast.FunctionDeclaration {
	desugaredDeclaration := d.VisitFunctionDeclaration(declaration, true)
	return desugaredDeclaration.(*ast.FunctionDeclaration)
}

func (d *Desugar) DesugarFunctionExpression(expression *ast.FunctionExpression) *ast.FunctionExpression {
	parameterList := expression.ParameterList
	returnTypeAnnotation := expression.ReturnTypeAnnotation

	functionType := d.elaboration.elaboration.FunctionExpressionFunctionType(expression)

	// TODO: If the function block was not desugared, avoid creating a new function-expression.
	modifiedFuncBlock := d.desugarFunctionBlock(
		"",
		expression.FunctionBlock,
		parameterList,
		functionType,
		returnTypeAnnotation,
		expression,
	)

	functionExpr := ast.NewFunctionExpression(
		d.memoryGauge,
		expression.Purity,
		parameterList,
		returnTypeAnnotation,
		modifiedFuncBlock,
		expression.StartPos,
	)

	d.elaboration.elaboration.SetFunctionExpressionFunctionType(functionExpr, functionType)

	return functionExpr
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
		ast.EmptyPosition,
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

func newEnumInitializer(
	gauge common.MemoryGauge,
	enumType *sema.CompositeType,
	elaboration *DesugaredElaboration,
) *ast.SpecialFunctionDeclaration {

	rawValueType := enumType.EnumRawType
	rawValueTypeName := rawValueType.String()

	rawValueIdentifier := ast.NewIdentifier(
		gauge,
		sema.EnumRawValueFieldName,
		ast.EmptyPosition,
	)

	parameters := []*ast.Parameter{
		ast.NewParameter(
			gauge,
			sema.EnumRawValueFieldName,
			rawValueIdentifier,
			ast.NewTypeAnnotation(
				gauge,
				false,
				ast.NewNominalType(
					gauge,
					ast.NewIdentifier(
						gauge,
						rawValueTypeName,
						ast.EmptyPosition,
					),
					nil,
				),
				ast.EmptyPosition,
			),
			nil,
			ast.EmptyPosition,
		),
	}

	assignmentStatement := ast.NewAssignmentStatement(
		gauge,
		ast.NewMemberExpression(
			gauge,
			ast.NewIdentifierExpression(
				gauge,
				ast.NewIdentifier(
					gauge,
					sema.SelfIdentifier,
					ast.EmptyPosition,
				),
			),
			false,
			ast.EmptyPosition,
			rawValueIdentifier,
		),
		ast.NewTransfer(
			gauge,
			ast.TransferOperationCopy,
			ast.EmptyPosition,
		),
		ast.NewIdentifierExpression(
			gauge,
			rawValueIdentifier,
		),
	)

	elaboration.SetAssignmentStatementTypes(
		assignmentStatement,
		sema.AssignmentStatementTypes{
			ValueType:  rawValueType,
			TargetType: rawValueType,
		},
	)

	initializer := ast.NewFunctionDeclaration(
		gauge,
		ast.AccessNotSpecified,
		ast.FunctionPurityUnspecified,
		false,
		false,
		ast.NewIdentifier(
			gauge,
			commons.InitFunctionName,
			ast.EmptyPosition,
		),
		nil,
		ast.NewParameterList(gauge, parameters, ast.EmptyRange),
		nil,
		ast.NewFunctionBlock(
			gauge,
			ast.NewBlock(
				gauge,
				[]ast.Statement{
					assignmentStatement,
				},
				ast.EmptyRange,
			),
			nil,
			nil,
		),
		ast.EmptyPosition,
		"",
	)

	return ast.NewSpecialFunctionDeclaration(
		gauge,
		common.DeclarationKindInitializer,
		initializer,
	)
}

func newEnumInitializerFuncType(rawValueType sema.Type) *sema.FunctionType {
	return sema.NewSimpleFunctionType(
		sema.FunctionPurityImpure,
		[]sema.Parameter{
			{
				Identifier:     sema.EnumRawValueFieldName,
				TypeAnnotation: sema.NewTypeAnnotation(rawValueType),
			},
		},
		sema.VoidTypeAnnotation,
	)
}

func newEnumLookup(
	gauge common.MemoryGauge,
	enumType *sema.CompositeType,
	enumCases []*ast.EnumCaseDeclaration,
	elaboration *DesugaredElaboration,
) *ast.FunctionDeclaration {

	typeIdentifier := ast.NewIdentifier(
		gauge,
		enumType.Identifier,
		ast.EmptyPosition,
	)

	rawValueType := enumType.EnumRawType
	rawValueTypeName := rawValueType.String()

	rawValueIdentifier := ast.NewIdentifier(
		gauge,
		sema.EnumRawValueFieldName,
		ast.EmptyPosition,
	)

	parameters := []*ast.Parameter{
		ast.NewParameter(
			gauge,
			sema.EnumRawValueFieldName,
			rawValueIdentifier,
			ast.NewTypeAnnotation(
				gauge,
				false,
				ast.NewNominalType(
					gauge,
					ast.NewIdentifier(
						gauge,
						rawValueTypeName,
						ast.EmptyPosition,
					),
					nil,
				),
				ast.EmptyPosition,
			),
			nil,
			ast.EmptyPosition,
		),
	}

	switchCases := make([]*ast.SwitchCase, 0, len(enumCases)+1)

	optionalEnumType := sema.NewOptionalType(gauge, enumType)

	for index, enumCase := range enumCases {

		// case <index>: return <enumType>.<caseName>

		literal := []byte(strconv.Itoa(index))
		integerExpression := ast.NewIntegerExpression(
			gauge,
			literal,
			big.NewInt(int64(index)),
			10,
			ast.EmptyRange,
		)
		elaboration.SetIntegerExpressionType(
			integerExpression,
			rawValueType,
		)

		memberExpression := ast.NewMemberExpression(
			gauge,
			ast.NewIdentifierExpression(
				gauge,
				typeIdentifier,
			),
			false,
			ast.EmptyPosition,
			ast.NewIdentifier(
				gauge,
				enumCase.Identifier.Identifier,
				ast.EmptyPosition,
			),
		)
		elaboration.SetMemberExpressionMemberAccessInfo(
			memberExpression,
			sema.MemberAccessInfo{
				AccessedType:  elaboration.EnumLookupFunctionType(enumType),
				ResultingType: enumType,
			},
		)

		returnStatement := ast.NewReturnStatement(
			gauge,
			memberExpression,
			ast.EmptyRange,
		)

		elaboration.SetReturnStatementTypes(
			returnStatement,
			sema.ReturnStatementTypes{
				ValueType:  enumType,
				ReturnType: optionalEnumType,
			},
		)

		switchCases = append(
			switchCases,
			&ast.SwitchCase{
				Expression: integerExpression,
				Statements: []ast.Statement{
					returnStatement,
				},
			},
		)
	}

	// default: return nil

	nilReturnStatement := ast.NewReturnStatement(
		gauge,
		ast.NewNilExpression(gauge, ast.EmptyPosition),
		ast.EmptyRange,
	)

	elaboration.SetReturnStatementTypes(
		nilReturnStatement,
		sema.ReturnStatementTypes{
			ValueType:  optionalEnumType,
			ReturnType: optionalEnumType,
		},
	)

	switchCases = append(
		switchCases,
		&ast.SwitchCase{
			Statements: []ast.Statement{
				nilReturnStatement,
			},
		},
	)

	switchStatement := ast.NewSwitchStatement(
		gauge,
		ast.NewIdentifierExpression(gauge, rawValueIdentifier),
		switchCases,
		ast.EmptyRange,
	)

	return ast.NewFunctionDeclaration(
		gauge,
		ast.AccessNotSpecified,
		ast.FunctionPurityUnspecified,
		false,
		false,
		typeIdentifier,
		nil,
		ast.NewParameterList(gauge, parameters, ast.EmptyRange),
		nil,
		ast.NewFunctionBlock(
			gauge,
			ast.NewBlock(
				gauge,
				[]ast.Statement{
					switchStatement,
				},
				ast.EmptyRange,
			),
			nil,
			nil,
		),
		ast.EmptyPosition,
		"",
	)
}

func newEnumLookupFuncType(
	gauge common.MemoryGauge,
	enumType *sema.CompositeType,
) *sema.FunctionType {
	return sema.NewSimpleFunctionType(
		sema.FunctionPurityImpure,
		[]sema.Parameter{
			{
				Identifier:     sema.EnumRawValueFieldName,
				TypeAnnotation: sema.NewTypeAnnotation(enumType.EnumRawType),
			},
		},
		sema.NewTypeAnnotation(
			sema.NewOptionalType(gauge, enumType),
		),
	)
}

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

func (d *Desugar) transactionCompositeType() *sema.CompositeType {
	return &sema.CompositeType{
		Location:    d.location,
		Identifier:  commons.TransactionWrapperCompositeName,
		Kind:        common.CompositeKindStructure,
		NestedTypes: &sema.StringTypeOrderedMap{},
		Members:     &sema.StringMemberOrderedMap{},
	}
}

// Generate a function to get all the resource-destroyed events
// associated with the given composite type, as an array.
func (d *Desugar) generateResourceDestroyedEventsGetterFunction(
	compositeType sema.Type,
	compositeDeclaration *ast.CompositeDeclaration,
) *ast.FunctionDeclaration {

	// Generate a function:
	// ```
	//   func $ResourceDestroyed(): [AnyStruct] {
	//      return [
	//          R.$ResourceDestroyed(args...),
	//          I.$ResourceDestroyed(args...),
	//          ...,
	//      ]
	//   }
	// ```

	eventConstructorInvocations := make([]ast.Expression, 0)
	eventTypes := make([]sema.Type, 0)

	addEventConstructorInvocation := func(
		eventDeclaration *ast.CompositeDeclaration,
		eventType *sema.CompositeType,
		enclosingType sema.Type,
	) {
		// Generate a constructor-invocation to construct an event-value.
		// e.g: `R.ResourceDestroyed(a, ...)`

		startPos := eventDeclaration.StartPos

		eventConstructor := constructorFunction(eventDeclaration)
		parameters := eventConstructor.ParameterList.Parameters

		// Generate arguments.
		arguments := make(ast.Arguments, 0, len(parameters))
		for _, param := range parameters {
			arguments = append(
				arguments,
				ast.NewUnlabeledArgument(
					d.memoryGauge,
					param.DefaultArgument,
				),
			)
		}

		endPos := eventDeclaration.EndPos

		// Generate the invoked expression: `R.ResourceDestroyed`
		memberExpression := ast.NewMemberExpression(
			d.memoryGauge,
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					enclosingType.QualifiedString(),
					startPos,
				),
			),
			false,
			startPos,
			eventDeclaration.Identifier,
		)

		eventConstructorFuncType := eventType.ConstructorFunctionType()

		d.elaboration.SetMemberExpressionMemberAccessInfo(
			memberExpression,
			sema.MemberAccessInfo{
				AccessedType:  enclosingType,
				ResultingType: eventType,
				Member: sema.NewConstructorMember(
					d.memoryGauge,
					enclosingType,
					sema.UnauthorizedAccess,
					eventType.Identifier,
					eventConstructorFuncType,
					"",
				),
			},
		)

		// Generate the invocation: `R.ResourceDestroyed(a, ...)`
		eventConstructorInvocation := ast.NewInvocationExpression(
			d.memoryGauge,
			memberExpression,
			nil,
			arguments,
			startPos,
			endPos,
		)

		eventConstructorFunctionType := d.elaboration.FunctionDeclarationFunctionType(eventConstructor)
		paramTypes := eventConstructorFunctionType.ParameterTypes()

		invocationTypes := sema.InvocationExpressionTypes{
			ReturnType:     eventType,
			ParameterTypes: paramTypes,
			ArgumentTypes:  paramTypes,
		}

		d.elaboration.SetInvocationExpressionTypes(eventConstructorInvocation, invocationTypes)

		eventConstructorInvocations = append(eventConstructorInvocations, eventConstructorInvocation)
		eventTypes = append(eventTypes, eventType)
	}

	// Construct events and add them as arr elements.
	// NOTE: The below order of events construction
	// is to be equivalent to the interpreter.

	// Construct self defined event
	defaultDestroyEventDeclaration := d.elaboration.DefaultDestroyDeclaration(compositeDeclaration)
	if defaultDestroyEventDeclaration != nil {
		eventType := d.elaboration.CompositeDeclarationType(defaultDestroyEventDeclaration)
		addEventConstructorInvocation(
			defaultDestroyEventDeclaration,
			eventType,
			compositeType,
		)
	}

	// Construct inherited events
	inheritedEvents := d.inheritedEvents[compositeType]
	for i := len(inheritedEvents) - 1; i >= 0; i-- {
		inheritedEvent := inheritedEvents[i]
		addEventConstructorInvocation(
			inheritedEvent.eventDeclaration,
			inheritedEvent.eventType,
			inheritedEvent.enclosingType,
		)
	}

	if len(eventConstructorInvocations) == 0 {
		return nil
	}

	astRange := compositeDeclaration.Range
	startPos := compositeDeclaration.StartPos

	// Put all the events in an array.
	arrayExpression := ast.NewArrayExpression(
		d.memoryGauge,
		eventConstructorInvocations,
		astRange,
	)

	d.elaboration.SetArrayExpressionTypes(
		arrayExpression,
		sema.ArrayExpressionTypes{
			ArrayType:     semaAnyStructArrayType,
			ArgumentTypes: eventTypes,
		},
	)

	// Return the array.
	returnStatement := ast.NewReturnStatement(
		d.memoryGauge,
		arrayExpression,
		astRange,
	)
	d.elaboration.SetReturnStatementTypes(
		returnStatement,
		sema.ReturnStatementTypes{
			ValueType:  semaAnyStructArrayType,
			ReturnType: semaAnyStructArrayType,
		},
	)

	functionName := commons.ResourceDestroyedEventsFunctionName

	// Generate the function declaration.
	eventEmittingFunction := ast.NewFunctionDeclaration(
		d.memoryGauge,
		ast.AccessNotSpecified,
		ast.FunctionPurityUnspecified,
		false,
		false,
		ast.NewIdentifier(
			d.memoryGauge,
			functionName,
			startPos,
		),
		nil,
		nil,
		ast.NewTypeAnnotation(
			d.memoryGauge,
			false,
			astAnyStructArrayType,
			startPos,
		),
		ast.NewFunctionBlock(
			d.memoryGauge,
			ast.NewBlock(
				d.memoryGauge,
				[]ast.Statement{returnStatement},
				astRange,
			),
			nil,
			nil,
		),
		startPos,
		"",
	)

	d.elaboration.SetFunctionDeclarationFunctionType(eventEmittingFunction, eventEmittingFunctionType)

	return eventEmittingFunction
}

func (d *Desugar) desugarDefaultDestroyEventInitializer(
	eventType *sema.CompositeType,
	initializer *ast.FunctionDeclaration,
) ast.Declaration {
	parameters := initializer.ParameterList.Parameters
	if len(parameters) == 0 {
		return initializer
	}

	initializerType := eventType.InitializerFunctionType()

	pos := initializer.StartPos

	statements := make([]ast.Statement, 0, len(parameters))
	for index, parameter := range parameters {
		// Field name is same as the parameter name.
		parameterName := parameter.Identifier.Identifier
		fieldName := parameterName

		fieldAccess := ast.NewMemberExpression(
			d.memoryGauge,
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					sema.SelfIdentifier,
					pos,
				),
			),
			false,
			pos,
			ast.NewIdentifier(
				d.memoryGauge,
				fieldName,
				pos,
			),
		)

		paramType := initializerType.Parameters[index].TypeAnnotation.Type

		memberAccessInfo := sema.MemberAccessInfo{
			AccessedType:  eventType,
			ResultingType: paramType,
			Member: sema.NewFieldMember(
				d.memoryGauge,
				eventType,
				sema.UnauthorizedAccess,
				ast.VariableKindVariable,
				fieldName,
				paramType,
				"",
			),
			IsOptional:      false,
			ReturnReference: false,
		}

		d.elaboration.SetMemberExpressionMemberAccessInfo(fieldAccess, memberAccessInfo)

		// Assign to the field
		// `self.x = x`
		assignmentStmt := ast.NewAssignmentStatement(
			d.memoryGauge,
			fieldAccess,
			ast.NewTransfer(
				d.memoryGauge,
				ast.TransferOperationCopy, // This is a copy because the event params are any-struct
				pos,
			),
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					parameterName,
					pos,
				),
			),
		)

		assignmentTypes := sema.AssignmentStatementTypes{
			ValueType:  paramType,
			TargetType: paramType,
		}
		d.elaboration.SetAssignmentStatementTypes(assignmentStmt, assignmentTypes)

		statements = append(statements, assignmentStmt)
	}

	modifiedFuncBlock := ast.NewFunctionBlock(
		d.memoryGauge,
		ast.NewBlock(
			d.memoryGauge,
			statements,
			ast.NewRangeFromPositioned(d.memoryGauge, initializer),
		),
		nil,
		nil,
	)

	modifiedInitializer := ast.NewFunctionDeclaration(
		d.memoryGauge,
		initializer.Access,
		initializer.Purity,
		initializer.IsStatic(),
		initializer.IsNative(),
		initializer.Identifier,
		initializer.TypeParameterList,
		initializer.ParameterList,
		initializer.ReturnTypeAnnotation,
		modifiedFuncBlock,
		initializer.StartPos,
		initializer.DocString,
	)

	// Desugared function's type is same as the original function's type.
	d.elaboration.SetFunctionDeclarationFunctionType(modifiedInitializer, initializerType)

	return ast.NewSpecialFunctionDeclaration(
		d.memoryGauge,
		common.DeclarationKindInitializer,
		modifiedInitializer,
	)
}

func constructorFunction(compositeDeclaration *ast.CompositeDeclaration) *ast.FunctionDeclaration {
	initializers := compositeDeclaration.Members.Initializers()
	if len(initializers) != 1 {
		panic(errors.NewUnexpectedError("expected exactly one initializer"))
	}

	eventConstructor := initializers[0].FunctionDeclaration
	return eventConstructor
}

var executeFuncType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.VoidTypeAnnotation,
)

var eventEmittingFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityImpure,
	nil,
	sema.VoidTypeAnnotation,
)

var semaAnyStructArrayType = &sema.VariableSizedType{
	Type: sema.AnyStructType,
}

var astAnyStructArrayType = &ast.VariableSizedType{
	Type: &ast.NominalType{
		Identifier: ast.Identifier{
			Identifier: sema.AnyStructType.Name,
		},
	},
}
