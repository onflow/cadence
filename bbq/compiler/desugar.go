package compiler

import (
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// Desugar will rewrite the AST from high-level abstractions to a much lower-level
// abstractions, so the compiler and vm could work with a minimal set of language features.
type Desugar struct {
	memoryGauge common.MemoryGauge
	elaboration *sema.Elaboration
	program     *ast.Program
	config      *Config

	modifiedDeclarations []ast.Declaration
}

var _ ast.DeclarationVisitor[ast.Declaration] = &Desugar{}

func NewDesugar(
	memoryGauge common.MemoryGauge,
	compilerConfig *Config,
	program *ast.Program,
	elaboration *sema.Elaboration,
) *Desugar {
	return &Desugar{
		memoryGauge: memoryGauge,
		config:      compilerConfig,
		elaboration: elaboration,
		program:     program,
	}
}

func (d *Desugar) Run() *ast.Program {
	// TODO: This assumes the program/elaboration is not cached.
	//   i.e: Modifies the elaboration.
	//   Handle this properly for cached programs.

	declarations := d.program.Declarations()
	for _, declaration := range declarations {
		modifiedDeclaration := d.desugarDeclaration(declaration)
		d.modifiedDeclarations = append(d.modifiedDeclarations, modifiedDeclaration)
	}

	return ast.NewProgram(d.memoryGauge, d.modifiedDeclarations)
}

func (d *Desugar) desugarDeclaration(declaration ast.Declaration) ast.Declaration {
	return ast.AcceptDeclaration[ast.Declaration](declaration, d)
}

func (d *Desugar) VisitVariableDeclaration(declaration *ast.VariableDeclaration) ast.Declaration {
	return declaration
}

func (d *Desugar) VisitFunctionDeclaration(declaration *ast.FunctionDeclaration) ast.Declaration {
	funcBlock := declaration.FunctionBlock
	if !funcBlock.HasConditions() {
		return declaration
	}

	modifiedFuncBlock := ast.NewFunctionBlock(
		d.memoryGauge,
		funcBlock.Block,
		d.desugarConditions(funcBlock.PreConditions),
		d.desugarConditions(funcBlock.PostConditions),
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

func (d *Desugar) desugarConditions(conditions *ast.Conditions) *ast.Conditions {
	if conditions == nil {
		return nil
	}

	desugaredConditions := make([]ast.Condition, 0, len(conditions.Conditions))

	for _, condition := range conditions.Conditions {
		desugaredCondition := d.desugarCondition(condition)
		desugaredConditions = append(desugaredConditions, desugaredCondition)
	}

	return &ast.Conditions{
		Conditions: desugaredConditions,
		Range:      conditions.Range,
	}
}

var conditionFailedMessage = ast.NewStringExpression(nil, "pre/post condition failed", ast.EmptyRange)

var panicFuncInvocationTypes = sema.InvocationExpressionTypes{
	ReturnType: stdlib.PanicFunctionType.ReturnTypeAnnotation.Type,
	ArgumentTypes: []sema.Type{
		sema.StringType,
	},
}

func (d *Desugar) desugarCondition(condition ast.Condition) *ast.DesugaredCondition {
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
		return ast.NewDesugaredCondition(ifStmt)
	case *ast.EmitCondition:
		return ast.NewDesugaredCondition((*ast.EmitStatement)(condition))
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
	existingMemberCount := len(existingMembers)

	// Recursively de-sugar nested declarations (functions, types, etc.)

	desugaredMembers := make([]ast.Declaration, 0, existingMemberCount)
	membersDesugared := false

	for _, member := range existingMembers {
		desugaredMember := d.desugarDeclaration(member)
		membersDesugared = membersDesugared || (desugaredMember != member)
		desugaredMembers = append(desugaredMembers, desugaredMember)
	}

	// Copy over inherited default functions.

	compositeType := d.elaboration.CompositeDeclarationType(declaration)
	inheritedDefaultFuncs := d.copyInheritedFunctions(compositeType)

	// Optimization: If none of the existing members got updated or,
	// if there are no inherited members, then return the same declaration as-is.
	if !membersDesugared && len(inheritedDefaultFuncs) == 0 {
		return declaration
	}

	modifiedMembers := make([]ast.Declaration, existingMemberCount)
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

func (d *Desugar) copyInheritedFunctions(compositeType *sema.CompositeType) []ast.Declaration {
	directMembers := compositeType.Members
	allMembers := compositeType.GetMembers()

	inheritedMembers := make([]ast.Declaration, 0)

	for memberName, resolver := range allMembers {
		if directMembers.Contains(memberName) {
			continue
		}

		member := resolver.Resolve(
			nil,
			memberName,
			ast.EmptyRange,
			func(err error) {
				if err != nil {
					panic(err)
				}
			},
		)

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

		inheritedMembers = append(inheritedMembers, inheritedFunc)
	}

	return inheritedMembers
}

func (d *Desugar) VisitInterfaceDeclaration(declaration *ast.InterfaceDeclaration) ast.Declaration {
	return declaration
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
