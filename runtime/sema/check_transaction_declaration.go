/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

func (checker *Checker) VisitTransactionDeclaration(declaration *ast.TransactionDeclaration) ast.Repr {
	transactionType := checker.Elaboration.TransactionDeclarationTypes[declaration]
	if transactionType == nil {
		panic(errors.NewUnreachableError())
	}

	checker.containerTypes[transactionType] = true
	defer func() {
		checker.containerTypes[transactionType] = false
	}()

	fieldMembers := NewMemberAstFieldDeclarationOrderedMap()

	for _, field := range declaration.Fields {
		fieldName := field.Identifier.Identifier
		member, ok := transactionType.Members.Get(fieldName)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		fieldMembers.Set(member, field)
	}

	checker.checkTransactionFields(declaration)
	checker.checkTransactionBlocks(declaration)

	// enter a new scope for this transaction
	checker.enterValueScope()
	defer checker.leaveValueScope(true)

	checker.declareSelfValue(transactionType)

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

	return nil
}

func (checker *Checker) checkTransactionParameters(declaration *ast.TransactionDeclaration, parameters []*Parameter) {
	checker.checkArgumentLabels(declaration.ParameterList)
	checker.checkParameters(declaration.ParameterList, parameters)
	checker.declareParameters(declaration.ParameterList, parameters)

	// Check parameter types

	for i, parameter := range parameters {
		parameterType := parameter.TypeAnnotation.Type

		// Ignore invalid parameter types

		if parameterType.IsInvalidType() {
			continue
		}

		// Parameters may not be resources

		if parameterType.IsResourceType() {

			astParameter := declaration.ParameterList.Parameters[i]

			checker.report(
				&InvalidResourceTransactionParameterError{
					Type:  parameterType,
					Range: ast.NewRangeFromPositioned(astParameter.TypeAnnotation),
				},
			)
		}

		// Parameters must be storable

		if !parameter.TypeAnnotation.Type.IsStorable(map[*Member]bool{}) {
			checker.report(
				&InvalidNonStorableTransactionParameterTypeError{
					Type: parameterType,
				},
			)
		}
	}
}

// checkTransactionFields validates the field declarations for a transaction.
//
func (checker *Checker) checkTransactionFields(declaration *ast.TransactionDeclaration) {
	for _, field := range declaration.Fields {
		if field.Access != ast.AccessNotSpecified {
			checker.report(
				&InvalidTransactionFieldAccessModifierError{
					Name:   field.Identifier.Identifier,
					Access: field.Access,
					Pos:    field.StartPosition(),
				},
			)
		}
	}
}

// checkTransactionBlocks checks that a transaction contains the required prepare and execute blocks.
//
// An execute block is always required, but a prepare block is only required if fields are present.
//
func (checker *Checker) checkTransactionBlocks(declaration *ast.TransactionDeclaration) {
	if declaration.Prepare != nil {
		// parser allows any identifier so it must be checked here
		prepareIdentifier := declaration.Prepare.FunctionDeclaration.Identifier
		if prepareIdentifier.Identifier != common.DeclarationKindPrepare.Keywords() {
			checker.report(&InvalidTransactionBlockError{
				Name: prepareIdentifier.Identifier,
				Pos:  prepareIdentifier.Pos,
			})
		}
	} else if len(declaration.Fields) != 0 {
		// report an error if fields are defined but no prepare statement exists
		// note: field initialization is checked later

		// report error for first field
		firstField := declaration.Fields[0]

		checker.report(
			&TransactionMissingPrepareError{
				FirstFieldName: firstField.Identifier.Identifier,
				FirstFieldPos:  firstField.Identifier.Pos,
			},
		)
	}

	if declaration.Execute != nil {
		// parser allows any identifier so it must be checked here
		executeIdentifier := declaration.Execute.FunctionDeclaration.Identifier
		if executeIdentifier.Identifier != common.DeclarationKindExecute.Keywords() {
			checker.report(&InvalidTransactionBlockError{
				Name: executeIdentifier.Identifier,
				Pos:  executeIdentifier.Pos,
			})
		}
	}
}

// visitTransactionPrepareFunction visits and checks the prepare function of a transaction.
//
func (checker *Checker) visitTransactionPrepareFunction(
	prepareFunction *ast.SpecialFunctionDeclaration,
	transactionType *TransactionType,
	fieldMembers *MemberAstFieldDeclarationOrderedMap,
) {
	if prepareFunction == nil {
		return
	}

	initializationInfo := NewInitializationInfo(transactionType, fieldMembers)

	prepareFunctionType := transactionType.PrepareFunctionType().InvocationFunctionType()

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
//
func (checker *Checker) checkTransactionPrepareFunctionParameters(
	parameterList *ast.ParameterList,
	parameters []*Parameter,
) {
	for i, parameter := range parameterList.Parameters {
		parameterType := parameters[i].TypeAnnotation.Type

		if !parameterType.IsInvalidType() &&
			!IsSubType(parameterType, AuthAccountType) {

			checker.report(
				&InvalidTransactionPrepareParameterTypeError{
					Type:  parameterType,
					Range: ast.NewRangeFromPositioned(parameter.TypeAnnotation),
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

	executeFunctionType := transactionType.ExecuteFunctionType().InvocationFunctionType()

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

	declarations := make([]ast.Declaration, len(declaration.Fields))
	for i, field := range declaration.Fields {
		declarations[i] = field
	}

	allMembers := ast.NewMembers(declarations)

	members, fields, origins := checker.defaultMembersAndOrigins(
		allMembers,
		transactionType,
		ContainerKindComposite,
		declaration.DeclarationKind(),
	)

	transactionType.Members = members
	transactionType.Fields = fields
	if checker.originsAndOccurrencesEnabled {
		checker.memberOrigins[transactionType] = origins
	}

	if declaration.Prepare != nil {
		parameterList := declaration.Prepare.FunctionDeclaration.ParameterList
		transactionType.PrepareParameters = checker.parameters(parameterList)
	}

	checker.Elaboration.TransactionDeclarationTypes[declaration] = transactionType
	checker.Elaboration.TransactionTypes = append(checker.Elaboration.TransactionTypes, transactionType)
}
