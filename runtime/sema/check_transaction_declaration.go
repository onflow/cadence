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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) (_ struct{}) {
	transactionType := checker.Elaboration.TransactionDeclarationType(declaration)
	// Type should have been declared in declareTransactionDeclaration
	if transactionType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[transactionType] = struct{}{}
	defer delete(checker.containerTypes, transactionType)

	fields := declaration.Fields

	// Members were previously declared in declareTransactionDeclaration
	members := transactionType.Members
	fieldMembers := getFieldMembers(members, fields)

	checker.checkTransactionFields(fields)

	prepareFunctionDeclaration := declaration.Prepare
	checker.checkPrepareExists(prepareFunctionDeclaration, fields)

	// enter a new scope for this transaction
	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, true)

	checker.declareSelfValue(transactionType, "")

	if declaration.ParameterList != nil {
		checker.checkTransactionParameters(declaration, transactionType.Parameters)
	}

	if prepareFunctionDeclaration != nil {
		prepareFunctionType := transactionType.PrepareFunctionType()

		checker.visitPrepareFunction(
			prepareFunctionDeclaration,
			transactionType,
			prepareFunctionType,
			fieldMembers,
		)
	}

	// Check roles
	for _, roleDeclaration := range declaration.Roles {
		ast.AcceptDeclaration[struct{}](roleDeclaration, checker)

		// Each role's prepare parameter list must match the transaction's prepare parameter list
		checker.checkRolePrepareParameterList(roleDeclaration, transactionType)
	}

	if declaration.PreConditions != nil {
		checker.visitConditions(*declaration.PreConditions)
	}

	checker.visitWithPostConditions(
		declaration.PostConditions,
		VoidType,
		func() {
			checker.withSelfResourceInvalidationAllowed(func() {
				checker.visitTransactionExecuteFunction(declaration.Execute, transactionType)
			})
		},
	)

	checker.checkResourceFieldsInvalidated(transactionType, transactionType.Members)

	return
}

// checkRolePrepareParameterList checks that the role's prepare block parameter list
// matches the transaction's prepare parameter list:
// They must have the same length and the parameter types must match.
func (checker *Checker) checkRolePrepareParameterList(
	roleDeclaration *ast.TransactionRoleDeclaration,
	transactionType *TransactionType,
) {
	transactionRoleType := checker.Elaboration.TransactionRoleDeclarationType(roleDeclaration)
	// Type should have been declared in declareTransactionDeclaration
	if transactionRoleType == nil {
		panic(errors.NewUnreachableError())
	}

	transactionPrepareParameterCount := len(transactionType.PrepareParameters)
	transactionRolePrepareParameterCount := len(transactionRoleType.PrepareParameters)
	if transactionPrepareParameterCount != transactionRolePrepareParameterCount {
		if roleDeclaration.Prepare == nil {
			checker.report(
				&MissingRolePrepareError{
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						roleDeclaration.Identifier,
					),
				},
			)
		} else {
			checker.report(
				&PrepareParameterCountMismatchError{
					ActualCount:   transactionRolePrepareParameterCount,
					ExpectedCount: transactionPrepareParameterCount,
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						roleDeclaration.Prepare,
					),
				},
			)
		}
	}

	minPrepareParameterCount := transactionPrepareParameterCount
	if transactionRolePrepareParameterCount < minPrepareParameterCount {
		minPrepareParameterCount = transactionRolePrepareParameterCount
	}

	for prepareParameterIndex := 0; prepareParameterIndex < minPrepareParameterCount; prepareParameterIndex++ {
		transactionPrepareParameter := transactionType.PrepareParameters[prepareParameterIndex]
		transactionRolePrepareParameter := transactionRoleType.PrepareParameters[prepareParameterIndex]

		if !transactionRolePrepareParameter.TypeAnnotation.
			Equal(transactionPrepareParameter.TypeAnnotation) {

			parameter := roleDeclaration.Prepare.FunctionDeclaration.ParameterList.Parameters[prepareParameterIndex]

			checker.report(
				&TypeMismatchError{
					ExpectedType: transactionPrepareParameter.TypeAnnotation.Type,
					ActualType:   transactionRolePrepareParameter.TypeAnnotation.Type,
					Range: ast.NewRangeFromPositioned(
						checker.memoryGauge,
						parameter.TypeAnnotation,
					),
				},
			)
		}
	}
}

func (checker *Checker) checkTransactionParameters(declaration *ast.TransactionDeclaration, parameters []Parameter) {
	checker.checkArgumentLabels(declaration.ParameterList)
	checker.checkParameters(declaration.ParameterList, parameters)
	checker.declareParameters(declaration.ParameterList, parameters)

	// Check parameter types

	for _, parameter := range parameters {
		parameterType := parameter.TypeAnnotation.Type

		// Ignore invalid parameter types

		if parameterType.IsInvalidType() {
			continue
		}

		// Parameters must be importable

		if !parameterType.IsImportable(map[*Member]bool{}) {
			checker.report(
				&InvalidNonImportableTransactionParameterTypeError{
					Type: parameterType,
				},
			)
		}
	}
}

// checkTransactionFields validates the field declarations in a transaction,
// i.e. a transaction declaration or a transaction role declaration.
func (checker *Checker) checkTransactionFields(fields []*ast.FieldDeclaration) {
	for _, field := range fields {
		if field.Access != ast.AccessNotSpecified {
			checker.report(
				&InvalidAccessModifierError{
					Access:          field.Access,
					Explanation:     "fields in transactions may not have an access modifier",
					DeclarationKind: common.DeclarationKindField,
					Pos:             field.StartPos,
				},
			)
		}
	}
}

// checkPrepareExists ensures that if fields exists, a prepare block is necessary,
// for example, in a transaction or transaction role.
func (checker *Checker) checkPrepareExists(
	prepare *ast.SpecialFunctionDeclaration,
	fields []*ast.FieldDeclaration,
) {
	if len(fields) == 0 || prepare != nil {
		return
	}

	// Report and error for the first field
	firstField := fields[0]

	checker.report(
		&MissingPrepareForFieldError{
			FirstFieldName: firstField.Identifier.Identifier,
			FirstFieldPos:  firstField.Identifier.Pos,
		},
	)
}

// visitPrepareFunction visits and checks the prepare function of a transaction.
func (checker *Checker) visitPrepareFunction(
	prepareFunction *ast.SpecialFunctionDeclaration,
	containerType Type,
	prepareFunctionType *FunctionType,
	fieldMembers *MemberFieldDeclarationOrderedMap,
) {
	initializationInfo := NewInitializationInfo(containerType, fieldMembers)

	checker.checkFunction(
		prepareFunction.FunctionDeclaration.ParameterList,
		nil,
		prepareFunctionType,
		prepareFunction.FunctionDeclaration.FunctionBlock,
		true,
		initializationInfo,
		true,
	)

	checker.checkTransactionPrepareFunctionParameters(
		prepareFunction.FunctionDeclaration.ParameterList,
		prepareFunctionType.Parameters,
	)
}

// checkTransactionPrepareFunctionParameters checks that the parameters are each of type Account.
func (checker *Checker) checkTransactionPrepareFunctionParameters(
	parameterList *ast.ParameterList,
	parameters []Parameter,
) {
	for i, parameter := range parameterList.Parameters {
		parameterType := parameters[i].TypeAnnotation.Type

		if !parameterType.IsInvalidType() &&
			!IsSameTypeKind(parameterType, AuthAccountType) {

			checker.report(
				&InvalidTransactionPrepareParameterTypeError{
					Type:  parameterType,
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, parameter.TypeAnnotation),
				},
			)
		}
	}

}

// visitTransactionExecuteFunction visits and checks the execute function of a transaction.
func (checker *Checker) visitTransactionExecuteFunction(
	executeFunction *ast.SpecialFunctionDeclaration,
	transactionType *TransactionType,
) {
	if executeFunction == nil {
		return
	}

	executeFunctionType := transactionType.ExecuteFunctionType()

	checker.checkFunction(
		&ast.ParameterList{},
		nil,
		executeFunctionType,
		executeFunction.FunctionDeclaration.FunctionBlock,
		true,
		nil,
		true,
	)
}

func (checker *Checker) declareTransactionDeclaration(declaration *ast.TransactionDeclaration) {
	transactionType := &TransactionType{}

	if declaration.ParameterList != nil {
		transactionType.Parameters = checker.parameters(declaration.ParameterList)
	}

	members, fields, origins := checker.compositeFieldMembersAndOrigins(
		transactionType,
		declaration.DeclarationKind(),
		declaration.Fields,
	)

	transactionType.Members = members
	transactionType.Fields = fields
	if checker.PositionInfo != nil {
		checker.PositionInfo.recordMemberOrigins(transactionType, origins)
	}

	if declaration.Prepare != nil {
		parameterList := declaration.Prepare.FunctionDeclaration.ParameterList
		transactionType.PrepareParameters = checker.parameters(parameterList)
	}

	for _, roleDeclaration := range declaration.Roles {
		transactionRoleType := checker.transactionRoleType(roleDeclaration)
		checker.Elaboration.SetTransactionRoleDeclarationType(roleDeclaration, transactionRoleType)

		// Ensure roles and fields do not clash
		roleName := roleDeclaration.Identifier.Identifier
		if member, ok := members.Get(roleName); ok {

			identifierRange := ast.NewRangeFromPositioned(
				checker.memoryGauge,
				roleDeclaration.Identifier,
			)

			if _, ok := member.TypeAnnotation.Type.(*TransactionRoleType); ok {
				checker.report(
					&DuplicateTransactionRoleError{
						Name:  roleName,
						Range: identifierRange,
					},
				)
			} else {
				checker.report(
					&TransactionRoleWithFieldNameError{
						FieldIdentifier: member.Identifier,
						Range:           identifierRange,
					},
				)
			}
			continue
		}

		members.Set(
			roleName,
			&Member{
				ContainerType:   transactionType,
				Identifier:      roleDeclaration.Identifier,
				DeclarationKind: common.DeclarationKindTransactionRole,
				VariableKind:    ast.VariableKindConstant,
				TypeAnnotation:  NewTypeAnnotation(transactionRoleType),
				DocString:       roleDeclaration.DocString,
			},
		)
	}

	checker.Elaboration.SetTransactionDeclarationType(declaration, transactionType)
	checker.Elaboration.TransactionTypes = append(checker.Elaboration.TransactionTypes, transactionType)
}

func (checker *Checker) transactionRoleType(declaration *ast.TransactionRoleDeclaration) *TransactionRoleType {
	transactionRoleType := &TransactionRoleType{}

	members, fields, origins := checker.compositeFieldMembersAndOrigins(
		transactionRoleType,
		declaration.DeclarationKind(),
		declaration.Fields,
	)

	transactionRoleType.Members = members
	transactionRoleType.Fields = fields
	if checker.PositionInfo != nil {
		checker.PositionInfo.recordMemberOrigins(transactionRoleType, origins)
	}

	if declaration.Prepare != nil {
		parameterList := declaration.Prepare.FunctionDeclaration.ParameterList
		transactionRoleType.PrepareParameters = checker.parameters(parameterList)
	}

	return transactionRoleType
}

func (checker *Checker) compositeFieldMembersAndOrigins(
	containerType Type,
	declarationKind common.DeclarationKind,
	declarations []*ast.FieldDeclaration,
) (
	members *StringMemberOrderedMap,
	fields []string,
	origins map[string]*Origin,
) {
	var fieldDeclarations []ast.Declaration

	fieldCount := len(declarations)
	if fieldCount > 0 {
		fieldDeclarations = make([]ast.Declaration, fieldCount)
		for i, field := range declarations {
			fieldDeclarations[i] = field
		}
	}

	allMembers := ast.NewMembers(checker.memoryGauge, fieldDeclarations)

	return checker.defaultMembersAndOrigins(
		allMembers,
		containerType,
		ContainerKindComposite,
		declarationKind,
	)
}

func (checker *Checker) VisitTransactionRoleDeclaration(declaration *ast.TransactionRoleDeclaration) (_ struct{}) {
	transactionRoleType := checker.Elaboration.TransactionRoleDeclarationType(declaration)
	// Type should have been declared in declareTransactionDeclaration
	if transactionRoleType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[transactionRoleType] = struct{}{}
	defer delete(checker.containerTypes, transactionRoleType)

	fields := declaration.Fields

	// Members were previously declared in declareTransactionDeclaration
	members := transactionRoleType.Members
	fieldMembers := getFieldMembers(members, fields)

	checker.checkTransactionFields(fields)
	checker.checkPrepareExists(declaration.Prepare, fields)

	// enter a new scope for this transaction role
	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, true)

	checker.declareSelfValue(transactionRoleType, "")

	if declaration.Prepare != nil {
		prepareFunctionType := transactionRoleType.PrepareFunctionType()
		checker.visitPrepareFunction(
			declaration.Prepare,
			transactionRoleType,
			prepareFunctionType,
			fieldMembers,
		)
	}

	return
}
