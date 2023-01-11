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
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) (_ struct{}) {
	transactionType := checker.Elaboration.TransactionDeclarationType(declaration)
	if transactionType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[transactionType] = true
	defer func() {
		checker.containerTypes[transactionType] = false
	}()

	fields := declaration.Fields
	fieldMembers := orderedmap.New[MemberFieldDeclarationOrderedMap](len(fields))

	for _, field := range fields {
		fieldName := field.Identifier.Identifier
		member, ok := transactionType.Members.Get(fieldName)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		fieldMembers.Set(member, field)
	}

	checker.checkTransactionFields(declaration)
	checker.checkPrepareExists(declaration.Prepare, declaration.Fields)

	// enter a new scope for this transaction
	checker.enterValueScope()
	defer checker.leaveValueScope(declaration.EndPosition, true)

	checker.declareSelfValue(transactionType, "")

	// TODO: declare variables for all blocks

	if declaration.ParameterList != nil {
		checker.checkTransactionParameters(declaration, transactionType.Parameters)
	}

	checker.visitTransactionPrepareFunction(declaration.Prepare, transactionType, fieldMembers)

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

// checkTransactionFields validates the field declarations for a transaction.
func (checker *Checker) checkTransactionFields(declaration *ast.TransactionDeclaration) {
	for _, field := range declaration.Fields {
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
		&MissingPrepareError{
			FirstFieldName: firstField.Identifier.Identifier,
			FirstFieldPos:  firstField.Identifier.Pos,
		},
	)
}

// visitTransactionPrepareFunction visits and checks the prepare function of a transaction.
func (checker *Checker) visitTransactionPrepareFunction(
	prepareFunction *ast.SpecialFunctionDeclaration,
	transactionType *TransactionType,
	fieldMembers *MemberFieldDeclarationOrderedMap,
) {
	if prepareFunction == nil {
		return
	}

	initializationInfo := NewInitializationInfo(transactionType, fieldMembers)

	prepareFunctionType := transactionType.PrepareFunctionType()

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

	var fieldDeclarations []ast.Declaration

	fieldCount := len(declaration.Fields)
	if fieldCount > 0 {
		fieldDeclarations = make([]ast.Declaration, fieldCount)
		for i, field := range declaration.Fields {
			fieldDeclarations[i] = field
		}
	}

	allMembers := ast.NewMembers(checker.memoryGauge, fieldDeclarations)

	members, fields, origins := checker.defaultMembersAndOrigins(
		allMembers,
		transactionType,
		ContainerKindComposite,
		declaration.DeclarationKind(),
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

	checker.Elaboration.SetTransactionDeclarationType(declaration, transactionType)
	checker.Elaboration.TransactionTypes = append(checker.Elaboration.TransactionTypes, transactionType)
}
