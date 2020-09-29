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
	"github.com/onflow/cadence/runtime/sema"
)

func (i *FlowIntegration) codeLenses(uri protocol.DocumentUri, checker *sema.Checker) ([]*protocol.CodeLens, error) {

	var (
		declarations = getAllDeclarations(checker.Elaboration)
		actions      []*protocol.CodeLens
	)

	addAction := func(lens *protocol.CodeLens) {
		if lens != nil {
			actions = append(actions, lens)
		}
	}

	addAction(i.showSubmitTransactionAction(uri, declarations))
	addAction(i.showDeployContractAction(uri, declarations))
	addAction(i.showDeployContractInterfaceAction(uri, declarations))
	addAction(i.showExecuteScriptAction(uri, declarations))

	return actions, nil
}

func (i *FlowIntegration) showSubmitTransactionAction(
	uri protocol.DocumentUri,
	declarations *declarations,
) *protocol.CodeLens {
	// Do not show submit button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show submit button when there is exactly one transaction declaration and no
	// other actionable declarations.
	if len(declarations.transactions) == 1 &&
		len(declarations.contracts) == 0 &&
		len(declarations.contractInterfaces) == 0 &&
		len(declarations.scripts) == 0 {
		return &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				declarations.transactions[0].StartPosition(),
				declarations.transactions[0].StartPosition(),
			),
			Command: &protocol.Command{
				Title:     fmt.Sprintf("submit transaction with account 0x%s", i.activeAddress.Hex()),
				Command:   CommandSubmitTransaction,
				Arguments: []interface{}{uri},
			},
		}
	}

	return nil
}

func (i *FlowIntegration) showDeployContractAction(
	uri protocol.DocumentUri,
	declarations *declarations,
) *protocol.CodeLens {
	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show the deploy button when there is exactly one contract declaration,
	// and no other actionable declarations.
	if len(declarations.contracts) == 1 &&
		len(declarations.transactions) == 0 &&
		len(declarations.scripts) == 0 &&
		len(declarations.contractInterfaces) == 0 {

		contract := declarations.contracts[0]

		name := contract.Identifier.Identifier

		return &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				contract.StartPosition(),
				contract.StartPosition(),
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

	return nil
}

func (i *FlowIntegration) showDeployContractInterfaceAction(
	uri protocol.DocumentUri,
	declarations *declarations,
) *protocol.CodeLens {
	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	// Show the deploy button when there is exactly one contract interface declaration,
	// and no other actionable declarations.
	if len(declarations.contractInterfaces) == 1 &&
		len(declarations.transactions) == 0 &&
		len(declarations.scripts) == 0 &&
		len(declarations.contracts) == 0 {

		contractInterface := declarations.contractInterfaces[0]

		name := contractInterface.Identifier.Identifier

		return &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				contractInterface.StartPosition(),
				contractInterface.StartPosition(),
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

	return nil
}

func (i *FlowIntegration) showExecuteScriptAction(
	uri protocol.DocumentUri,
	declarations *declarations,
) *protocol.CodeLens {
	// Show execute script button when there is exactly one valid script
	// function and no other actionable declarations.
	if len(declarations.scripts) == 1 &&
		len(declarations.contracts) == 0 &&
		len(declarations.contractInterfaces) == 0 &&
		len(declarations.transactions) == 0 {
		return &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				declarations.scripts[0].StartPosition(),
				declarations.scripts[0].StartPosition(),
			),
			Command: &protocol.Command{
				Title:     "execute script",
				Command:   CommandExecuteScript,
				Arguments: []interface{}{uri},
			},
		}
	}

	return nil
}
