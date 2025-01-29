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
	"fmt"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

const tempResultVariableName = "$result"

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

	importsSet map[common.Location]struct{}
	newImports []ast.Declaration
}

type inheritedFunction struct {
	interfaceType *sema.InterfaceType
	functionDecl  *ast.FunctionDeclaration
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
		memoryGauge: memoryGauge,
		config:      compilerConfig,
		elaboration: elaboration,
		program:     program,
		checker:     checker,
		importsSet:  map[common.Location]struct{}{},
	}
}

func (d *Desugar) Run() *ast.Program {
	// TODO: This assumes the program/elaboration is not cached.
	//   i.e: Modifies the elaboration.
	//   Handle this properly for cached programs.

	declarations := d.program.Declarations()
	for _, declaration := range declarations {
		modifiedDeclaration := d.desugarDeclaration(declaration)
		if modifiedDeclaration != nil {
			d.modifiedDeclarations = append(d.modifiedDeclarations, modifiedDeclaration)
		}
	}

	d.modifiedDeclarations = append(d.newImports, d.modifiedDeclarations...)

	program := ast.NewProgram(d.memoryGauge, d.modifiedDeclarations)

	//fmt.Println(ast.Prettier(program))

	return program
}

func (d *Desugar) desugarDeclaration(declaration ast.Declaration) ast.Declaration {
	return ast.AcceptDeclaration[ast.Declaration](declaration, d)
}

func (d *Desugar) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Declaration {
	funcBlock := declaration.FunctionBlock
	funcName := declaration.Identifier.Identifier

	preConditions := d.desugarConditions(
		funcName,
		ast.ConditionKindPre,
		funcBlock,
		declaration.ParameterList,
	)
	postConditions := d.desugarConditions(
		funcName,
		ast.ConditionKindPost,
		funcBlock,
		declaration.ParameterList,
	)

	modifiedStatements := make([]ast.Statement, 0)
	modifiedStatements = append(modifiedStatements, preConditions...)

	if funcBlock.HasStatements() {
		statements := funcBlock.Block.Statements
		modifiedStatements = append(modifiedStatements, statements...)
	}

	// Before the post conditions are appended, we need to move the
	// return statement to the end of the function.
	// For that, replace the return-stmt with a temporary result assignment,
	// and once the post conditions are added, then add a new return-stmt
	// which would return the temp result.

	// TODO: If 'result' variable is used in the user-code,
	//   this temp assignment must use the 'result' variable.

	// TODO: Handle returns from multiple places

	var originalReturnStmt *ast.ReturnStatement
	hasReturn := false

	if len(modifiedStatements) > 0 {
		lastStmt := modifiedStatements[len(modifiedStatements)-1]
		originalReturnStmt, hasReturn = lastStmt.(*ast.ReturnStatement)
	}

	if hasReturn {
		// Remove the return statement from here
		modifiedStatements = modifiedStatements[:len(modifiedStatements)-1]

		if originalReturnStmt.Expression != nil {
			// Instead append a temp-result assignment.i.e:
			// `let $result = <expression>`
			tempResultAssignmentStmt := ast.NewVariableDeclaration(
				d.memoryGauge,
				ast.AccessNotSpecified,
				true,
				ast.NewIdentifier(d.memoryGauge, tempResultVariableName, originalReturnStmt.StartPos),
				declaration.ReturnTypeAnnotation,
				originalReturnStmt.Expression,
				ast.NewTransfer(
					d.memoryGauge,
					ast.TransferOperationCopy, // TODO: determine based on return value (if resource, this should be a move)
					originalReturnStmt.StartPos,
				),
				originalReturnStmt.StartPos,
				nil,
				nil,
				"",
			)

			returnStmtTypes := d.elaboration.ReturnStatementTypes(originalReturnStmt)
			d.elaboration.SetVariableDeclarationTypes(
				tempResultAssignmentStmt,
				sema.VariableDeclarationTypes{
					ValueType:  returnStmtTypes.ValueType,
					TargetType: returnStmtTypes.ReturnType,
				},
			)

			modifiedStatements = append(modifiedStatements, tempResultAssignmentStmt)
		}
	}

	// Once the return statement is removed, then append the post conditions.
	modifiedStatements = append(modifiedStatements, postConditions...)

	// Insert a return statement at the end, after post conditions.
	var modifiedReturn *ast.ReturnStatement
	if !hasReturn || originalReturnStmt.Expression == nil {
		var astRange ast.Range
		if hasReturn {
			astRange = originalReturnStmt.Range
		} else {
			astRange = ast.EmptyRange
		}

		// `return`
		modifiedReturn = ast.NewReturnStatement(
			d.memoryGauge,
			nil,
			astRange,
		)
	} else {
		// `return $result`
		modifiedReturn = ast.NewReturnStatement(
			d.memoryGauge,
			ast.NewIdentifierExpression(
				d.memoryGauge,
				ast.NewIdentifier(
					d.memoryGauge,
					tempResultVariableName,
					originalReturnStmt.StartPos,
				),
			),
			originalReturnStmt.Range,
		)
	}
	modifiedStatements = append(modifiedStatements, modifiedReturn)

	modifiedFuncBlock := ast.NewFunctionBlock(
		d.memoryGauge,
		ast.NewBlock(
			d.memoryGauge,
			modifiedStatements,
			ast.NewRangeFromPositioned(d.memoryGauge, declaration),
		),
		nil,
		nil,
	)

	return ast.NewFunctionDeclaration(
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
}

func (d *Desugar) desugarConditions(
	enclosingFuncName string,
	kind ast.ConditionKind,
	funcBlock *ast.FunctionBlock,
	list *ast.ParameterList,
) []ast.Statement {

	desugaredConditions := make([]ast.Statement, 0)

	pos := ast.EmptyPosition

	var conditions *ast.Conditions

	// Desugar inherited pre-conditions
	if kind == ast.ConditionKindPre {
		if funcBlock != nil {
			conditions = funcBlock.PreConditions
		}

		if d.inheritedFuncsWithConditions != nil {
			inheritedFuncs := d.inheritedFuncsWithConditions[enclosingFuncName]
			for _, inheritedFunc := range inheritedFuncs {
				inheritedPreConditions := inheritedFunc.functionDecl.FunctionBlock.PreConditions
				if inheritedPreConditions == nil {
					continue
				}

				// If the inherited function has pre-conditions, then add an invocation
				// to call the generated pre-condition-function of the interface.

				// Generate: `FooInterface.bar.&preCondition(a1, a2)`
				invocation := d.inheritedConditionInvocation(
					enclosingFuncName,
					kind,
					inheritedFunc,
					pos,
				)
				desugaredConditions = append(desugaredConditions, invocation)
			}
		}
	} else if funcBlock != nil {
		// Post conditions
		conditions = funcBlock.PostConditions
	}

	// Desugar self-defined pre/post conditions
	if conditions != nil {
		conditionsList := conditions.Conditions
		if kind == ast.ConditionKindPost {
			postConditionsRewrite := d.elaboration.PostConditionsRewrite(conditions)
			conditionsList = postConditionsRewrite.RewrittenPostConditions
		}

		for _, condition := range conditionsList {
			desugaredCondition := d.desugarCondition(condition)
			desugaredConditions = append(desugaredConditions, desugaredCondition)
		}
	}

	// Desugar inherited post-conditions
	if kind == ast.ConditionKindPost {
		if d.inheritedFuncsWithConditions != nil {
			inheritedFuncs, ok := d.inheritedFuncsWithConditions[enclosingFuncName]
			if ok && len(inheritedFuncs) > 0 {
				// Must be added in reverse order.
				for i := len(inheritedFuncs) - 1; i >= 0; i-- {
					inheritedFunc := inheritedFuncs[i]
					inheritedPostConditions := inheritedFunc.functionDecl.FunctionBlock.PostConditions
					if inheritedPostConditions == nil {
						continue
					}
					invocation := d.inheritedConditionInvocation(
						enclosingFuncName,
						kind,
						inheritedFunc,
						pos,
					)
					desugaredConditions = append(desugaredConditions, invocation)
				}
			}
		}
	}

	// If this is a method of a concrete-type, or an interface default method,
	// then return with the updated statements, and continue desugaring the rest.

	// TODO: Handle default functions with conditions

	if d.enclosingInterfaceType == nil ||
		funcBlock.HasStatements() ||
		conditions == nil {
		return desugaredConditions
	}

	// Otherwise, i.e: if this is an interface function with only pre/post conditions,
	// (thus not a default function), then generate a separate function for the conditions.
	d.generateConditionsFunction(
		enclosingFuncName,
		kind,
		conditions,
		desugaredConditions,
		list,
	)

	return nil
}

func (d *Desugar) generateConditionsFunction(
	enclosingFuncName string,
	kind ast.ConditionKind,
	conditions *ast.Conditions,
	desugaredConditions []ast.Statement,
	list *ast.ParameterList,
) {
	pos := conditions.StartPos

	desugaredConditions = append(
		desugaredConditions,
		ast.NewReturnStatement(
			d.memoryGauge,
			nil,
			conditions.Range,
		),
	)
	modifiedFuncBlock := ast.NewFunctionBlock(
		d.memoryGauge,
		ast.NewBlock(
			d.memoryGauge,
			desugaredConditions,
			conditions.Range,
		),
		nil,
		nil,
	)

	conditionFunc := ast.NewFunctionDeclaration(
		d.memoryGauge,
		ast.AccessAll,
		ast.FunctionPurityView, // conditions must be pure
		false,
		false,
		generatedConditionsFuncIdentifier(
			d.enclosingInterfaceType,
			enclosingFuncName,
			kind,
			pos,
		),
		nil,

		// Parameters for the condition is same as
		// the parameters of the enclosing function.
		list,

		ast.NewTypeAnnotation(
			d.memoryGauge,
			false,
			astVoidType(pos),
			pos,
		),
		modifiedFuncBlock,
		pos,
		"",
	)

	d.modifiedDeclarations = append(d.modifiedDeclarations, conditionFunc)
}

func (d *Desugar) inheritedConditionInvocation(
	funcName string,
	kind ast.ConditionKind,
	inheritedFunc *inheritedFunction,
	pos ast.Position,
) ast.Statement {

	conditionsFuncName := generatedConditionsFuncName(
		inheritedFunc.interfaceType,
		funcName,
		kind,
	)

	params := inheritedFunc.functionDecl.ParameterList.Parameters
	semaParams := make([]sema.Parameter, 0, len(params))
	for _, param := range params {
		paramTypeAnnotation := d.checker.ConvertTypeAnnotation(param.TypeAnnotation)

		semaParams = append(semaParams, sema.Parameter{
			TypeAnnotation: paramTypeAnnotation,
			Label:          param.Label,
			Identifier:     param.Identifier.Identifier,
		})
	}

	funcType := sema.NewSimpleFunctionType(
		sema.FunctionPurityView,
		semaParams,
		sema.VoidTypeAnnotation,
	)

	member := sema.NewPublicFunctionMember(
		d.memoryGauge,
		inheritedFunc.interfaceType,
		conditionsFuncName,
		funcType,
		"",
	)

	invocation := d.interfaceDelegationMethodCall(
		inheritedFunc.interfaceType,
		inheritedFunc.functionDecl,
		pos,
		conditionsFuncName,
		member,
	)

	return ast.NewExpressionStatement(d.memoryGauge, invocation)
}

func generatedConditionsFuncName(interfaceType *sema.InterfaceType, funcName string, kind ast.ConditionKind) string {
	return fmt.Sprintf("$%s.%s.%sConditions", interfaceType.Identifier, funcName, kind.Keyword())
}

func generatedConditionsFuncIdentifier(
	interfaceType *sema.InterfaceType,
	funcName string,
	kind ast.ConditionKind,
	pos ast.Position,
) ast.Identifier {
	return ast.NewIdentifier(
		nil,
		generatedConditionsFuncName(interfaceType, funcName, kind),
		pos,
	)
}

func astVoidType(pos ast.Position) ast.Type {
	return ast.NewNominalType(
		nil,
		ast.NewIdentifier(nil, "Void", pos),
		nil,
	)
}

var conditionFailedMessage = ast.NewStringExpression(nil, "pre/post condition failed", ast.EmptyRange)

var panicFuncInvocationTypes = sema.InvocationExpressionTypes{
	ReturnType: stdlib.PanicFunctionType.ReturnTypeAnnotation.Type,
	ArgumentTypes: []sema.Type{
		sema.StringType,
	},
}

func (d *Desugar) desugarCondition(condition ast.Condition) ast.Statement {
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
		return emitStmt
	default:
		panic(errors.NewUnreachableError())
	}
}

func (d *Desugar) VisitSpecialFunctionDeclaration(declaration *ast.SpecialFunctionDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitAttachmentDeclaration(declaration *ast.AttachmentDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitCompositeDeclaration(declaration *ast.CompositeDeclaration) ast.Declaration {
	existingMembers := declaration.Members.Declarations()

	compositeType := d.elaboration.CompositeDeclarationType(declaration)

	// Recursively de-sugar nested declarations (functions, types, etc.)

	prevInheritedFuncs := d.inheritedFuncsWithConditions
	d.inheritedFuncsWithConditions = d.inheritedFunctionsWithConditions(compositeType)
	defer func() {
		d.inheritedFuncsWithConditions = prevInheritedFuncs
	}()

	var desugaredMembers []ast.Declaration
	membersDesugared := false

	for _, member := range existingMembers {
		desugaredMember := d.desugarDeclaration(member)

		membersDesugared = membersDesugared || (desugaredMember != member)
		desugaredMembers = append(desugaredMembers, desugaredMember)
	}

	// Copy over inherited default functions.

	inheritedDefaultFuncs := d.inheritedDefaultFunctions(compositeType, declaration)

	// Optimization: If none of the existing members got updated or,
	// if there are no inherited members, then return the same declaration as-is.
	if !membersDesugared && len(inheritedDefaultFuncs) == 0 {
		return declaration
	}

	modifiedMembers := make([]ast.Declaration, len(desugaredMembers))
	copy(modifiedMembers, desugaredMembers)

	modifiedMembers = append(modifiedMembers, inheritedDefaultFuncs...)

	modifiedDecl := ast.NewCompositeDeclaration(
		d.memoryGauge,
		declaration.Access,
		declaration.CompositeKind,
		declaration.Identifier,
		declaration.Conformances,
		ast.NewMembers(d.memoryGauge, modifiedMembers),
		declaration.DocString,
		declaration.Range,
	)

	// Update elaboration. Type info is needed for later steps.
	d.elaboration.SetCompositeDeclarationType(modifiedDecl, compositeType)

	return modifiedDecl
}

func (d *Desugar) inheritedFunctionsWithConditions(compositeType *sema.CompositeType) map[string][]*inheritedFunction {
	inheritedFunctions := make(map[string][]*inheritedFunction)

	for _, conformance := range compositeType.EffectiveInterfaceConformances() {
		interfaceType := conformance.InterfaceType

		// TODO: Merge this elaboration with the current elaboration (d.elaboration).
		//   Because the inherited functions need their corresponding elaboration
		//   for code generation.
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
			funcs = append(funcs, &inheritedFunction{
				interfaceType: interfaceType,
				functionDecl:  functionDecl,
			})
			inheritedFunctions[name] = funcs
		}
	}

	return inheritedFunctions
}

func (d *Desugar) inheritedDefaultFunctions(compositeType *sema.CompositeType, decl *ast.CompositeDeclaration) []ast.Declaration {
	directMembers := compositeType.Members
	allMembers := compositeType.GetMembers()

	pos := decl.StartPos

	inheritedMembers := make([]ast.Declaration, 0)

	for memberName, resolver := range allMembers { // nolint:maprange
		if directMembers.Contains(memberName) {
			continue
		}

		member := resolver.Resolve(
			d.memoryGauge,
			memberName,
			ast.EmptyRange,
			func(err error) {
				if err != nil {
					panic(err)
				}
			},
		)

		// Only interested in functions.
		// Also filter out built-in functions.
		if member.DeclarationKind != common.DeclarationKindFunction ||
			member.Predeclared {
			continue
		}

		// Inherited functions are always from interfaces
		interfaceType := member.ContainerType.(*sema.InterfaceType)

		elaboration, err := d.config.ElaborationResolver(interfaceType.Location)
		if err != nil {
			panic(err)
		}

		interfaceDecl := elaboration.InterfaceTypeDeclaration(interfaceType)

		functions := interfaceDecl.Members.FunctionsByIdentifier()
		inheritedFunc, ok := functions[memberName]
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// for each inherited function, generate a delegator function,
		// which calls the actual default implementation at the interface.
		// i.e:
		//  FooImpl {
		//    fun defaultFunc(a1: T1, a2: T2): R {
		//        return FooInterface.defaultFunc(a1, a2)
		//    }
		//  }

		// Generate: `FooInterface.defaultFunc(a1, a2)`

		invocation := d.interfaceDelegationMethodCall(
			interfaceType,
			inheritedFunc,
			pos,
			memberName,
			member,
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
				memberName,
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
						ast.NewReturnStatement(d.memoryGauge, invocation, decl.Range),
					},
					decl.Range,
				),
				nil,
				nil,
			),
			inheritedFunc.StartPos,
			inheritedFunc.DocString,
		)

		inheritedMembers = append(inheritedMembers, defaultFuncDelegator)
	}

	return inheritedMembers
}

func (d *Desugar) interfaceDelegationMethodCall(
	interfaceType *sema.InterfaceType,
	inheritedFunc *ast.FunctionDeclaration,
	pos ast.Position,
	functionName string,
	member *sema.Member,
) *ast.InvocationExpression {

	arguments := make([]*ast.Argument, 0)
	for _, param := range inheritedFunc.ParameterList.Parameters {
		var arg *ast.Argument
		if param.Label == "_" {
			arg = ast.NewUnlabeledArgument(
				d.memoryGauge,
				ast.NewIdentifierExpression(
					d.memoryGauge,
					param.Identifier,
				),
			)
		} else {
			var label string
			if param.Label == "" {
				label = param.Identifier.Identifier
			} else {
				label = param.Label
			}

			arg = ast.NewArgument(
				d.memoryGauge,
				label,
				&param.StartPos,
				&param.StartPos,
				ast.NewIdentifierExpression(
					d.memoryGauge,
					param.Identifier,
				),
			)
		}
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

	interfaceLocation, isAddressLocation := interfaceType.Location.(common.AddressLocation)
	if isAddressLocation {
		if _, exists := d.importsSet[interfaceLocation]; !exists {
			d.newImports = append(
				d.newImports,
				ast.NewImportDeclaration(
					d.memoryGauge,
					[]ast.Identifier{
						ast.NewIdentifier(d.memoryGauge, interfaceLocation.Name, ast.EmptyPosition),
					},
					interfaceLocation,
					ast.EmptyRange,
					ast.EmptyPosition,
				))

			d.importsSet[interfaceLocation] = struct{}{}
		}
	}

	return invocation
}

func (d *Desugar) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) ast.Declaration {
	interfaceType := d.elaboration.InterfaceDeclarationType(declaration)

	prevModifiedDecls := d.modifiedDeclarations
	prevEnclosingInterfaceType := d.enclosingInterfaceType
	d.modifiedDeclarations = nil
	d.enclosingInterfaceType = interfaceType

	defer func() {
		d.modifiedDeclarations = prevModifiedDecls
		d.enclosingInterfaceType = prevEnclosingInterfaceType
	}()

	existingMembers := declaration.Members.Declarations()

	// Recursively de-sugar nested declarations (functions, types, etc.)

	membersDesugared := false

	for _, member := range existingMembers {
		desugaredMember := d.desugarDeclaration(member)
		membersDesugared = membersDesugared || (desugaredMember != member)
		d.modifiedDeclarations = append(d.modifiedDeclarations, desugaredMember)
	}

	// Optimization: If none of the existing members got updated or,
	// if there are no inherited members, then return the same declaration as-is.
	//if !membersDesugared && len(inheritedDefaultFuncs) == 0 {
	//	return declaration
	//}
	if !membersDesugared {
		return declaration
	}

	modifiedDecl := ast.NewInterfaceDeclaration(
		d.memoryGauge,
		declaration.Access,
		declaration.CompositeKind,
		declaration.Identifier,
		declaration.Conformances,
		ast.NewMembers(d.memoryGauge, d.modifiedDeclarations),
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
	// TODO: add pre/post conditions

	// Converts a transaction into a composite type declaration.
	// Transaction parameters are converted into global variables.
	// An initializer is generated to set parameters to above generated global variables.

	var varDeclarations []ast.Declaration
	var initFunction *ast.FunctionDeclaration

	if transaction.ParameterList != nil {
		varDeclarations = make([]ast.Declaration, 0, len(transaction.ParameterList.Parameters))
		statements := make([]ast.Statement, 0, len(transaction.ParameterList.Parameters))
		parameters := make([]*ast.Parameter, 0, len(transaction.ParameterList.Parameters))

		for index, parameter := range transaction.ParameterList.Parameters {
			// Create global variables
			// i.e: `var a: Type`
			field := &ast.VariableDeclaration{
				Access:         ast.AccessSelf,
				IsConstant:     false,
				Identifier:     parameter.Identifier,
				TypeAnnotation: parameter.TypeAnnotation,
			}
			varDeclarations = append(varDeclarations, field)

			// Create assignment from param to global var.
			// i.e: `a = $param_a`
			modifiedParamName := commons.TransactionGeneratedParamPrefix + parameter.Identifier.Identifier
			modifiedParameter := &ast.Parameter{
				Label: "",
				Identifier: ast.Identifier{
					Identifier: modifiedParamName,
				},
				TypeAnnotation: parameter.TypeAnnotation,
			}
			parameters = append(parameters, modifiedParameter)

			assignment := &ast.AssignmentStatement{
				Target: &ast.IdentifierExpression{
					Identifier: parameter.Identifier,
				},
				Value: &ast.IdentifierExpression{
					Identifier: ast.Identifier{
						Identifier: modifiedParamName,
					},
				},
				Transfer: &ast.Transfer{
					Operation: ast.TransferOperationCopy,
				},
			}
			statements = append(statements, assignment)

			transactionTypes := d.elaboration.TransactionDeclarationType(transaction)
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
		initFunction = &ast.FunctionDeclaration{
			Access: ast.AccessNotSpecified,
			Identifier: ast.Identifier{
				Identifier: commons.ProgramInitFunctionName,
			},
			ParameterList: &ast.ParameterList{
				Parameters: parameters,
			},
			ReturnTypeAnnotation: nil,
			FunctionBlock: &ast.FunctionBlock{
				Block: &ast.Block{
					Statements: statements,
				},
			},
		}
	}

	var members []ast.Declaration
	if transaction.Execute != nil {
		members = append(members, transaction.Execute.FunctionDeclaration)
	}
	if transaction.Prepare != nil {
		members = append(members, transaction.Prepare)
	}

	compositeType := &sema.CompositeType{
		Location:    nil,
		Identifier:  commons.TransactionWrapperCompositeName,
		Kind:        common.CompositeKindStructure,
		NestedTypes: &sema.StringTypeOrderedMap{},
		Members:     &sema.StringMemberOrderedMap{},
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

	d.elaboration.SetCompositeDeclarationType(compositeDecl, compositeType)

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
