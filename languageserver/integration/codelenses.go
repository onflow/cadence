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

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
)

func (i *FlowIntegration) codeLenses(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) (
	[]*protocol.CodeLens,
	error,
) {

	var actions []*protocol.CodeLens

	addAction := func(lens *protocol.CodeLens) {
		if lens != nil {
			actions = append(actions, lens)
		}
	}

	program := checker.Program

	addAction(i.showDeployContractAction(uri, program))
	addAction(i.showDeployContractInterfaceAction(uri, program))

	entryPointCodeLenses, err := i.entryPointActions(uri, version, checker)
	if err != nil {
		return nil, err
	}
	actions = append(actions, entryPointCodeLenses...)

	return actions, nil
}

// showDeployContractAction show a deploy button when there is exactly one contract declaration,
// and no other actionable declarations
//
func (i *FlowIntegration) showDeployContractAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

	contract := program.SoleContractDeclaration()

	if contract == nil {
		return nil
	}

	name := contract.Identifier.Identifier

	position := contract.StartPosition()

	return &protocol.CodeLens{
		Range: conversion.ASTToProtocolRange(
			position,
			position,
		),
		Command: &protocol.Command{
			Title: fmt.Sprintf(
				"deploy to account 0x%s",
				i.activeAddress.Hex(),
			),
			Command:   CommandDeployContract,
			Arguments: []interface{}{uri, name},
		},
	}
}

// showDeployContractInterfaceAction shows a deploy button when there is exactly one contract interface declaration,
// and no other actionable declarations
//
func (i *FlowIntegration) showDeployContractInterfaceAction(
	uri protocol.DocumentUri,
	program *ast.Program,
) *protocol.CodeLens {

	// Do not show deploy button when no active account exists
	if i.activeAddress == flow.EmptyAddress {
		return nil
	}

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
				"deploy to account 0x%s",
				i.activeAddress.Hex(),
			),
			Command:   CommandDeployContract,
			Arguments: []interface{}{uri, name},
		},
	}
}

// entryPointActions shows an execute button when there is exactly one valid entry point
// (valid script function or transaction declaration) and no other actionable declarations.
//
//
func (i *FlowIntegration) entryPointActions(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) (
	[]*protocol.CodeLens,
	error,
) {

	i.updateEntryPointInfoIfNeeded(uri, version, checker)

	entryPointInfo := i.entryPointInfo[uri]
	if entryPointInfo.kind == entryPointKindUnknown || entryPointInfo.startPos == nil {
		return nil, nil
	}
	position := *entryPointInfo.startPos

	var title string
	var argumentListConjunction string
	var command string

	switch entryPointInfo.kind {
	case entryPointKindScript:
		title = "execute"
		argumentListConjunction = "with"
		command = CommandExecuteScript

	case entryPointKindTransaction:

		// TODO: maybe we shall remove this requirement, since we don't track active account anymore
		// Do not show submit button when no active account exists
		/*
		if i.activeAddress == flow.EmptyAddress {
			return nil, nil
		}
		*/

		title = fmt.Sprintf(
			"submit with account 0x%s",
			i.activeAddress.Hex(),
		)
		argumentListConjunction = "and"
		command = CommandSubmitTransaction
	}

	argumentLists := entryPointInfo.pragmaArguments[:]

	// If there are no parameters and no pragma argument declarations,
	// offer execution using no arguments
	if len(entryPointInfo.parameters) == 0 && len(argumentLists) == 0 {
		argumentLists = append(argumentLists, []cadence.Value{})
	}

	codeLenses := make([]*protocol.CodeLens, len(argumentLists))

	for i, argumentList := range argumentLists {
		formattedTitle := title
		if len(argumentList) > 0 {
			formattedTitle = fmt.Sprintf(
				"%s %s %s",
				title,
				argumentListConjunction,
				entryPointInfo.pragmaArgumentStrings[i],
			)
		}

		encodedArgumentList := make([]string, len(argumentList))
		for i, argument := range argumentList {
			encodedArgument, err := jsoncdc.Encode(argument)
			if err != nil {
				return nil, err
			}
			encodedArgumentList[i] = string(encodedArgument)
		}

		codeLenses[i] = &protocol.CodeLens{
			Range: conversion.ASTToProtocolRange(
				position,
				position,
			),
			Command: &protocol.Command{
				Title:     formattedTitle,
				Command:   command,
				Arguments: []interface{}{uri, encodedArgumentList},
			},
		}
	}

	return codeLenses, nil
}
