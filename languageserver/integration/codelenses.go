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

func encodeArguments(args []cadence.Value) string {
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

	contract := program.SoleContractDeclaration()
	if contract == nil {
		return nil
	}

	name := contract.Identifier.Identifier
	position := contract.StartPosition()
	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// Check emulator state
	emulatorStateLens := i.checkEmulatorState(codelensRange)
	if emulatorStateLens != nil {
		return []*protocol.CodeLens{emulatorStateLens}
	}

	entryPointInfo := i.entryPointInfo[uri]
	signersList := entryPointInfo.pragmaSignersStrings[:]
	// TODO: resolve check for amount of signers equals to provided by pragma
	if len(signersList) == 0 {
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for _, signers := range signersList {
		codeLenses = append(codeLenses, i.contractCodeLenses(uri, codelensRange, name, signers[0]))
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

	contract := program.SoleContractInterfaceDeclaration()
	if contract == nil {
		return nil
	}

	name := contract.Identifier.Identifier
	position := contract.StartPosition()
	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// Check emulator state
	emulatorStateLens := i.checkEmulatorState(codelensRange)
	if emulatorStateLens != nil {
		return []*protocol.CodeLens{emulatorStateLens}
	}

	entryPointInfo := i.entryPointInfo[uri]
	signersList := entryPointInfo.pragmaSignersStrings[:]
	if len(signersList) == 0 {
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for _, signers := range signersList {
		codeLenses = append(codeLenses, i.contractCodeLenses(uri, codelensRange, name, signers[0]))
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


	// Check emulator state
	emulatorStateLens := i.checkEmulatorState(codelensRange)

	if emulatorStateLens != nil {
		codeLenses = append(codeLenses, emulatorStateLens)
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
		switch entryPointInfo.kind {
		case entryPointKindScript:
			pragmaArguments := entryPointInfo.pragmaArgumentStrings[index]
			codeLenses = append(codeLenses, i.scriptCodeLenses(uri, codelensRange, pragmaArguments, argumentList))

		case entryPointKindTransaction:
			for _, signers := range signersList {
				var codeLens *protocol.CodeLens
				accounts, absentAccounts := i.resolveAccounts(signers)

				if len(absentAccounts) > 0 {
					codeLens = i.showAbsentAccounts(absentAccounts, codelensRange)
				} else {
					pragmaArguments := entryPointInfo.pragmaArgumentStrings[index]
					codeLens = i.transactionCodeLenses(uri, codelensRange, pragmaArguments, argumentList, signers, accounts)
				}

				codeLenses = append(codeLenses, codeLens)
			}
		}
	}

	return codeLenses, nil
}

func makeActionlessCodelens(title string, lensRange protocol.Range) *protocol.CodeLens {
	return &protocol.CodeLens{
		Range: lensRange,
		Command: &protocol.Command{
			Title: title,
		},
	}
}

func makeCodeLens(command string, title string, lensRange protocol.Range, arguments []interface{}) *protocol.CodeLens {
	return &protocol.CodeLens{
		Range: lensRange,
		Command: &protocol.Command{
			Title:     title,
			Command:   command,
			Arguments: arguments,
		},
	}
}

func (i *FlowIntegration) resolveAccounts(signers []string) ([]flow.Address, []string) {
	var absentAccounts []string
	var resolvedAccounts []flow.Address
	for _, signer := range signers {
		resolvedAddress, _ := i.getAccountAddress(signer)
		if resolvedAddress == flow.EmptyAddress {
			absentAccounts = append(absentAccounts, signer)
		} else {
			resolvedAccounts = append(resolvedAccounts, resolvedAddress)
		}
	}
	return resolvedAccounts, absentAccounts
}

func (i *FlowIntegration) showAbsentAccounts(accounts []string, codelensRange protocol.Range) *protocol.CodeLens {
	var title string
	accountsNumeric := "account"

	if len(accounts) > 1 {
		accountsNumeric = "accounts"
	}

	title = fmt.Sprintf("%s Specified %s %s does not exist",
		prefixError,
		accountsNumeric,
		common.EnumerateWords(accounts, "and"),
	)
	return makeActionlessCodelens(title, codelensRange)
}

func (i *FlowIntegration) checkEmulatorState(codelensRange protocol.Range) *protocol.CodeLens {
	var title string
	var codeLens *protocol.CodeLens

	if i.emulatorState == EmulatorOffline {
		title = fmt.Sprintf(
			"%s Emulator is Offline. Click here to start it",
			prefixOffline,
		)
		// TODO: Check if arguments are needed
		codeLens = makeCodeLens(ClientStartEmulator, title, codelensRange, nil)
	}

	if i.emulatorState == EmulatorStarting {
		title = fmt.Sprintf(
			"%s Emulator is starting up. Please wait \u2026",
			prefixStarting,
		)
		codeLens = makeActionlessCodelens(title, codelensRange)
	}

	return codeLens
}

func (i *FlowIntegration) scriptCodeLenses(
	uri protocol.DocumentUri,
	codelensRange protocol.Range,
	pragmaArguments string,
	argumentList []cadence.Value,
) *protocol.CodeLens {

	title := fmt.Sprintf(
		"%s Execute script",
		prefixOK,
	)

	if len(argumentList) > 0 {
		title = fmt.Sprintf(
			"%s with %s",
			title,
			pragmaArguments,
		)
	}

	argsJSON := encodeArguments(argumentList)
	return makeCodeLens(CommandExecuteScript, title, codelensRange, []interface{}{uri, argsJSON})
}

func (i *FlowIntegration) transactionCodeLenses(
	uri protocol.DocumentUri,
	codelensRange protocol.Range,
	pragmaArguments string,
	argumentList []cadence.Value,
	signers []string,
	accounts []flow.Address,
) *protocol.CodeLens {
	var title string

	if len(argumentList) > 0 {
		title = fmt.Sprintf(
			"%s Send with %s signed by %s",
			prefixOK,
			pragmaArguments,
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
	return makeCodeLens(
		CommandSendTransaction,
		title,
		codelensRange,
		[]interface{}{uri, argsJSON, accounts},
	)
}
func (i *FlowIntegration) contractCodeLenses(
	uri protocol.DocumentUri,
	codelensRange protocol.Range,
	name string,
	signer string,
) *protocol.CodeLens {
	var title string
	resolvedAddress, _ := i.getAccountAddress(signer)
	if resolvedAddress == flow.EmptyAddress {
		title = fmt.Sprintf("%s Specified account %s does not exist",
			prefixError,
			signer,
		)
		return makeActionlessCodelens(title, codelensRange)
	} else {
		title = fmt.Sprintf(
			"%s Deploy contract interface %s to %s",
			prefixOK,
			name,
			signer,
		)

		return makeCodeLens(CommandDeployContract, title, codelensRange,[]interface{}{uri, name, resolvedAddress})
	}
}
