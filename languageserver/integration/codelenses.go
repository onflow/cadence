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

package integration

import (
	"fmt"

	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func (i *FlowIntegration) codeLenses(uri protocol.DocumentUri, checker *sema.Checker) ([]*protocol.CodeLens, error) {

	var actions []*protocol.CodeLens

	addAction := func(lens *protocol.CodeLens) {
		if lens != nil {
			actions = append(actions, lens)
		}
	}

	program := checker.Program

	addAction(i.showSubmitTransactionAction(uri, program))
	addAction(i.showDeployContractAction(uri, program))
	addAction(i.showDeployContractInterfaceAction(uri, program))
	addAction(i.showExecuteScriptAction(uri, program))

	return actions, nil
}

func (i *FlowIntegration) showSubmitTransactionAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Do not show submit button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show submit button when there is exactly one transaction declaration,
	// and no other actionable declarations.

	transactionDeclaration := program.SoleTransactionDeclaration()
	if transactionDeclaration == nil {
		return nil
	}

	position := transactionDeclaration.StartPosition()

	return &protocol.CodeLens{
		Range: conversion.ASTToProtocolRange(
			position,
			position,
		),
		Command: &protocol.Command{
			Title: fmt.Sprintf(
				"submit transaction with account 0x%s",
				i.activeAddress.Hex(),
			),
			Command:   CommandSubmitTransaction,
			Arguments: []interface{}{uri},
		},
	}
}

func (i *FlowIntegration) showDeployContractAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show the deploy button when there is exactly one contract declaration,
	// and no other actionable declarations.

	contract := program.SoleContractDeclaration()

	name := contract.Identifier.Identifier

	position := contract.StartPosition()

	return &protocol.CodeLens{
		Range: conversion.ASTToProtocolRange(
			position,
			position,
		),
		Command: &protocol.Command{
			Title: fmt.Sprintf(
				"deploy contract '%s' to account 0x%s",
				name,
				i.activeAddress.Hex(),
			),
			Command:   CommandDeployContract,
			Arguments: []interface{}{uri},
		},
	}
}

func (i *FlowIntegration) showDeployContractInterfaceAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show the deploy button when there is exactly one contract interface declaration,
	// and no other actionable declarations.

	contractInterface := program.SoleContractInterfaceDeclaration()
	if contractInterface == nil {
		return nil
	}

	name := contractInterface.Identifier.Identifier

	position := contractInterface.StartPosition()

	return &protocol.CodeLens{
		Range: conversion.ASTToProtocolRange(
			position,
			position,
		),
		Command: &protocol.Command{
			Title: fmt.Sprintf(
				"deploy contract interface '%s' to account 0x%s",
				name,
				i.activeAddress.Hex(),
			),
			Command:   CommandDeployContract,
			Arguments: []interface{}{uri, name},
		},
	}
}

func (i *FlowIntegration) showExecuteScriptAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Show execute script button when there is exactly one valid script
	// function and no other actionable declarations.

	functionDeclaration := sema.FunctionEntryPointDeclaration(program)
	if functionDeclaration == nil {
		return nil
	}

	position := functionDeclaration.StartPosition()

	return &protocol.CodeLens{
		Range: conversion.ASTToProtocolRange(
			position,
			position,
		),
		Command: &protocol.Command{
			Title:     "execute script",
			Command:   CommandExecuteScript,
			Arguments: []interface{}{uri},
		},
	}
}
