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

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/sema"
)

func (i *FlowIntegration) codeLenses(uri protocol.DocumentUri, checker *sema.Checker) ([]*protocol.CodeLens, error) {

	elaboration := checker.Elaboration
	var (
		scriptFuncDeclarations        = getScriptDeclarations(elaboration.FunctionDeclarationFunctionTypes)
		txDeclarations                = getTransactionDeclarations(elaboration.TransactionDeclarationTypes)
		contractDeclarations          = getContractDeclarations(elaboration.CompositeDeclarationTypes)
		contractInterfaceDeclarations = getContractInterfaceDeclarations(elaboration.InterfaceDeclarationTypes)

		actions []*protocol.CodeLens
	)

	// Show submit button when there is exactly one transaction declaration and no
	// other actionable declarations.
	if len(txDeclarations) == 1 &&
		len(contractDeclarations) == 0 &&
		len(contractInterfaceDeclarations) == 0 &&
		len(scriptFuncDeclarations) == 0 {
		actions = append(actions, &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(txDeclarations[0].StartPosition(), txDeclarations[0].StartPosition()),
			Command: &protocol.Command{
				Title:     fmt.Sprintf("submit transaction with account 0x%s", i.activeAccount.Hex()),
				Command:   CommandSubmitTransaction,
				Arguments: []interface{}{uri},
			},
		})
	}

	// Show deploy button when there is exactly one contract declaration,
	// any number of contract interface declarations, and no other actionable
	// declarations.
	if len(contractDeclarations) == 1 &&
		len(txDeclarations) == 0 &&
		len(scriptFuncDeclarations) == 0 {
		actions = append(actions, &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				contractDeclarations[0].StartPosition(),
				contractDeclarations[0].StartPosition(),
			),
			Command: &protocol.Command{
				Title:     fmt.Sprintf("deploy contract to account 0x%s", i.activeAccount.Hex()),
				Command:   CommandUpdateAccountCode,
				Arguments: []interface{}{uri},
			},
		})
	}

	// Show deploy interface button when there are 1 or more contract interface
	// declarations, but no other actionable declarations.
	if len(contractInterfaceDeclarations) > 0 &&
		len(txDeclarations) == 0 &&
		len(scriptFuncDeclarations) == 0 &&
		len(contractDeclarations) == 0 {
		// decide whether to pluralize
		pluralInterface := "interface"
		if len(contractInterfaceDeclarations) > 1 {
			pluralInterface = "interfaces"
		}

		actions = append(actions, &protocol.CodeLens{
			Command: &protocol.Command{
				Title:     fmt.Sprintf("deploy contract %s to account 0x%s", pluralInterface, i.activeAccount.Hex()),
				Command:   CommandUpdateAccountCode,
				Arguments: []interface{}{uri},
			},
		})
	}

	// Show execute script button when there is exactly one valid script
	// function and no other actionable declarations.
	if len(scriptFuncDeclarations) == 1 &&
		len(contractDeclarations) == 0 &&
		len(contractInterfaceDeclarations) == 0 &&
		len(txDeclarations) == 0 {
		actions = append(actions, &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				scriptFuncDeclarations[0].StartPosition(),
				scriptFuncDeclarations[0].StartPosition(),
			),
			Command: &protocol.Command{
				Title:     "execute script",
				Command:   CommandExecuteScript,
				Arguments: []interface{}{uri},
			},
		})
	}

	return actions, nil
}
