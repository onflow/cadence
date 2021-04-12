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
	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/flow-go-sdk"
	"strings"
)

const (
	// Codelens message prefixes
	prefixOK       = "💡"
	prefixStarting = "⏲"
	prefixOffline  = "⚠️"
	prefixError    = "🚫"
)

func encodeArguments(args []cadence.Value)  string {
	var list []string
	for _, arg := range args {
		encoded, _ := jsoncdc.Encode(arg)
		list = append(list, string(encoded))
	}

	joined := strings.Join(list, ",")
	return fmt.Sprintf("[%s]", joined)
}


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
) []*protocol.CodeLens {

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
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for _, signers := range signersList {
		signer := signers[0]

		var codeLens *protocol.CodeLens
		var title string
		resolvedAddress, _ := i.getAccountAddress(signer)
		if resolvedAddress == flow.EmptyAddress {
			title = fmt.Sprintf("%s Specified account %s does not exist",
				prefixError,
				signer,
			)
			codeLens = &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title: title,
				},
			}
		} else {
			title = fmt.Sprintf(
				"%s Deploy contract %s to %s",
				prefixOK,
				name,
				signer,
			)

			codeLens = &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title:     title,
					Command:   CommandDeployContract,
					Arguments: []interface{}{uri, name, resolvedAddress},
				},
			}
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
) []*protocol.CodeLens {

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
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for _, signers := range signersList {
		signer := signers[0]

		var codeLens *protocol.CodeLens
		var title string
		resolvedAddress, _ := i.getAccountAddress(signer)
		if resolvedAddress == flow.EmptyAddress {
			title = fmt.Sprintf("%s Specified account %s does not exist",
				prefixError,
				signer,
			)
			codeLens = &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title: title,
				},
			}
		} else {
			title = fmt.Sprintf(
				"%s Deploy contract interface %s to %s",
				prefixOK,
				name,
				signer,
			)

			codeLens = &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title:     title,
					Command:   CommandDeployContract,
					Arguments: []interface{}{uri, name, resolvedAddress},
				},
			}
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
			"%s Emulator is Offline. Click here to start it",
			prefixOffline,
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
			"%s Emulator is starting up. Please wait \u2026",
			prefixStarting,
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
	if len(entryPointInfo.parameters) == 0 {
		argumentLists = append(argumentLists, []cadence.Value{})
	}

	signersList := entryPointInfo.pragmaSignersStrings[:]
	// TODO: resolve check for amount of signers equals to provided by pragma
	if len(signersList) == 0 {
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for index, argumentList := range argumentLists {
		var title string

		switch entryPointInfo.kind {

		case entryPointKindScript:
			if len(argumentList) > 0 {
				title = fmt.Sprintf(
					"%s Execute script with %s",
					prefixOK,
					entryPointInfo.pragmaArgumentStrings[index],
				)
			} else {
				title = fmt.Sprintf(
					"%s Execute script",
					prefixOK,
				)
			}

			argsJSON := encodeArguments(argumentList)

			codeLens := &protocol.CodeLens{
				Range: codelensRange,
				Command: &protocol.Command{
					Title:     title,
					Command:   CommandExecuteScript,
					Arguments: []interface{}{uri, argsJSON},
				},
			}
			codeLenses = append(codeLenses, codeLens)

		case entryPointKindTransaction:
			for _, signers := range signersList {

				var absentAccounts []string
				var resolvedAccounts []flow.Address
				for _, signer := range signers {
					resolvedAddress, _ := i.getAccountAddress(signer)
					resolvedAccounts = append(resolvedAccounts, resolvedAddress)
					if resolvedAddress == flow.EmptyAddress {
						absentAccounts = append(absentAccounts, signer)
					}
				}
				var codeLens *protocol.CodeLens
				if len(absentAccounts) > 0 {
					accountsNumeric := "account"
					if len(absentAccounts) > 1 {
						accountsNumeric = "accounts"
					}
					title = fmt.Sprintf("%s Specified %s %s does not exist",
						prefixError,
						accountsNumeric,
						common.EnumerateWords(absentAccounts, "and"),
					)
					codeLens = &protocol.CodeLens{
						Range: codelensRange,
						Command: &protocol.Command{
							Title: title,
						},
					}
				} else {
					if len(argumentList) > 0 {
						title = fmt.Sprintf(
							"%s Send with %s signed by %s",
							prefixOK,
							entryPointInfo.pragmaArgumentStrings[index],
							common.EnumerateWords(signers, "and"),
						)
					} else {
						title = fmt.Sprintf(
							"%s Send signed by %s",
							prefixOK,
							common.EnumerateWords(signers, "and"),
						)
					}

					argsJSON := encodeArguments(argumentList)


					codeLens = &protocol.CodeLens{
						Range: codelensRange,
						Command: &protocol.Command{
							Title:     title,
							Command:   CommandSendTransaction,
							Arguments: []interface{}{uri, argsJSON, resolvedAccounts},
						},
					}
				}
				codeLenses = append(codeLenses, codeLens)
			}
		}
	}

	return codeLenses, nil
}
