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
	"strings"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
)

const(
	// Codelens message prefixes
	prefixOK = "💡"
	prefixStarting = "⏲"
	prefixOffline = "⚠️"
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

	program := checker.Program

	deployContractLenses := i.showDeployContractAction(uri, program, version, checker)
	deployContractInterfaceLenses := i.showDeployContractInterfaceAction(uri, program, version, checker)
	actions = append(actions, deployContractLenses...)
	actions = append(actions, deployContractInterfaceLenses...)

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
	version float64,
	checker *sema.Checker,
) 	[]*protocol.CodeLens {

	i.updateEntryPointInfoIfNeeded(uri, version, checker)
	entryPointInfo := i.entryPointInfo[uri]

	contract := program.SoleContractDeclaration()

	if contract == nil {
		return nil
	}

	name := contract.Identifier.Identifier
	position := contract.StartPosition()
	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// If emulator is not up, we need to provide actionless codelens
	if i.emulatorState == EmulatorStarting || i.emulatorState == EmulatorOffline {
		return codeLenses
	}

	signersList := entryPointInfo.pragmaSignersStrings[:]
	// TODO: resolve check for amount of signers equals to provided by pragma
	if len(signersList) == 0 {
		activeAccount := [][]string{[]string{"active account"}}
		signersList = append(signersList, activeAccount...)
	}

	for _, signers := range signersList {
		title := fmt.Sprintf(
			"%s %s %s %s %s",
			prefixOK,
			"Deploy contract",
			name,
			"to",
			signers[0],
		)

		codeLens := &protocol.CodeLens{
			Range: codelensRange,
			Command: &protocol.Command{
				Title:     title,
				Command:   ClientDeployContract,
				Arguments: []interface{}{uri, name, signers[0]},
			},
		}
		codeLenses = append(codeLenses, codeLens)
	}


	return codeLenses
}

// showDeployContractInterfaceAction shows a deploy button when there is exactly one contract interface declaration,
// and no other actionable declarations
//
func (i *FlowIntegration) showDeployContractInterfaceAction(
	uri protocol.DocumentUri,
	program *ast.Program,
	version float64,
	checker *sema.Checker,
) 	[]*protocol.CodeLens {

	i.updateEntryPointInfoIfNeeded(uri, version, checker)
	entryPointInfo := i.entryPointInfo[uri]

	contract := program.SoleContractInterfaceDeclaration()

	if contract == nil {
		return nil
	}

	name := contract.Identifier.Identifier
	position := contract.StartPosition()
	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// If emulator is not up, we need to provide actionless codelens
	if i.emulatorState == EmulatorStarting || i.emulatorState == EmulatorOffline {
		return codeLenses
	}

	signersList := entryPointInfo.pragmaSignersStrings[:]
	// TODO: resolve check for amount of signers equals to provided by pragma
	if len(signersList) == 0 {
		activeAccount := [][]string{[]string{"active account"}}
		signersList = append(signersList, activeAccount...)
	}

	for _, signers := range signersList {
		title := fmt.Sprintf(
			"%s %s %s %s %s",
			prefixOK,
			"Deploy contract interface",
			name,
			"to",
			signers[0],
		)

		codeLens := &protocol.CodeLens{
			Range: codelensRange,
			Command: &protocol.Command{
				Title:     title,
				Command:   ClientDeployContract,
				Arguments: []interface{}{uri, name, signers[0]},
			},
		}
		codeLenses = append(codeLenses, codeLens)
	}


	return codeLenses
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
	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// If emulator is not up, we need to show single codelens proposing to start emulator
	if i.emulatorState == EmulatorOffline {
		title := fmt.Sprintf(
			"%s %s",
			prefixOffline,
			"Emulator is Offline. Click here to start it",
		)
		codeLens := &protocol.CodeLens{
			Range: codelensRange,
			Command: &protocol.Command{
				Title:     title,
				Command:   ClientStartEmulator,
				Arguments: []interface{}{uri, "Offline"},
			},
		}
		codeLenses = append(codeLenses, codeLens)
		return codeLenses, nil
	}

	// If emulator is not up, we need to provide actionless codelens
	if i.emulatorState == EmulatorStarting {
		title := fmt.Sprintf(
			"%s %s",
			prefixStarting,
			"Emulator is starting up. Please wait...",
		)
		codeLens := &protocol.CodeLens{
			Range: codelensRange,
			Command: &protocol.Command{
				Title: title,
			},
		}
		codeLenses = append(codeLenses, codeLens)
		return codeLenses, nil
	}

	argumentLists := entryPointInfo.pragmaArguments[:]

	// If there are no parameters and no pragma argument declarations,
	// offer execution using no arguments
	if len(entryPointInfo.parameters) == 0{
		argumentLists = append(argumentLists, []cadence.Value{})
	}

	signersList := entryPointInfo.pragmaSignersStrings[:]
	// TODO: resolve check for amount of signers equals to provided by pragma
	if len(signersList) == 0 {
		activeAccount := [][]string{[]string{"active account"}}
		signersList = append(signersList, activeAccount...)
	}

	for i, argumentList := range argumentLists {
		var title string

		encodedArgumentList := make([]string, len(argumentList))
		for i, argument := range argumentList {
			encodedArgument, err := jsoncdc.Encode(argument)
			if err != nil {
				return nil, err
			}
			encodedArgumentList[i] = string(encodedArgument)
		}

		switch entryPointInfo.kind {

		case entryPointKindScript:
			if len(argumentList) > 0 {
				title = fmt.Sprintf(
					"%s %s %s",
					prefixOK,
					"Execute script with",
					entryPointInfo.pragmaArgumentStrings[i],
				)
			} else {
				title = fmt.Sprintf(
					"%s %s",
					prefixOK,
					"Execute script",
				)
			}

			codeLens := &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title:     title,
					Command:   ClientExecuteScript,
					Arguments: []interface{}{uri, encodedArgumentList},
				},
			}
			codeLenses = append(codeLenses, codeLens)

		case entryPointKindTransaction:
			for _, signers := range signersList {
				if len(argumentList) > 0 {
					title = fmt.Sprintf(
						"%s %s %s %s %s",
						prefixOK,
						"Send with",
						entryPointInfo.pragmaArgumentStrings[i],
						"signed by ",
						strings.Join(signers, " and "),
					)
				} else {
					title = fmt.Sprintf(
						"%s %s %s",
						prefixOK,
						"Send signed by",
						strings.Join(signers, " and "),
					)
				}

				codeLens := &protocol.CodeLens{
					Range: codelensRange,
					Command: &protocol.Command{
						Title:     title,
						Command:   ClientSendTransaction,
						Arguments: []interface{}{uri, encodedArgumentList, signers},
					},
				}
				codeLenses = append(codeLenses, codeLens)
			}
		}
	}

	return codeLenses, nil
}
