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
	"encoding/json"
	"fmt"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
)

const (
	// Codelens message prefixes
	prefixOK       = "💡"
	prefixStarting = "⏲"
	prefixOffline  = "⚠️"
	prefixError    = "🚫"
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

	// Add code lenses for contracts and contract interfaces
	deployContractLenses, err := i.showDeployContractAction(uri, program, version, checker)
	if err != nil {
		return nil, err
	}
	actions = append(actions, deployContractLenses...)

	// Add code lenses for scripts and transactions
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
) ([]*protocol.CodeLens, error) {
	i.updateContractInfoIfNeeded(uri, version, checker)

	contractInfo := i.contractInfo[uri]
	kind := contractInfo.kind

	if kind == contractTypeUnknown || contractInfo.startPos == nil {
		return nil, nil
	}

	name := contractInfo.name
	position := *contractInfo.startPos
	signersList := contractInfo.pragmaSignersStrings[:]

	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// Check emulator state
	emulatorStateLens := i.checkEmulatorState(codelensRange)
	if emulatorStateLens != nil {
		return []*protocol.CodeLens{emulatorStateLens}, nil
	}

	if len(signersList) == 0 {
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for _, signers := range signersList {
		codeLenses = append(codeLenses, i.contractCodeLenses(uri, codelensRange, name, kind, signers[0]))
	}

	return codeLenses, nil
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

	kind := entryPointInfo.kind
	position := *entryPointInfo.startPos
	argumentLists := entryPointInfo.pragmaArguments[:]
	signersList := entryPointInfo.pragmaSignersStrings[:]
	requiredNumberOfSigners := entryPointInfo.numberOfSigners

	codelensRange := conversion.ASTToProtocolRange(position, position)
	var codeLenses []*protocol.CodeLens

	// Check emulator state
	emulatorStateLens := i.checkEmulatorState(codelensRange)
	if emulatorStateLens != nil {
		codeLenses = append(codeLenses, emulatorStateLens)
		return codeLenses, nil
	}

	// If there are no parameters and no pragma argument declarations,
	// offer execution using no arguments
	if len(entryPointInfo.parameters) == 0 {
		argumentLists = append(argumentLists, []Argument{})
	}

	if len(signersList) == 0 {
		signersList = append(signersList, []string{i.activeAccount.Name})
	}

	for index, argumentList := range argumentLists {
		pragmaArguments := entryPointInfo.pragmaArgumentStrings[index]

		switch kind {
		case entryPointKindScript:
			codeLens := i.scriptCodeLenses(uri, codelensRange, pragmaArguments, argumentList)
			codeLenses = append(codeLenses, codeLens)

		case entryPointKindTransaction:
			for _, signers := range signersList {

				numberOfSigners := len(signers)
				if requiredNumberOfSigners > numberOfSigners {
					title := fmt.Sprintf(
						"%s Not enough signers. Required: %v, passed: %v",
						prefixError,
						requiredNumberOfSigners,
						numberOfSigners,
					)
					codeLenses = append(codeLenses, makeActionlessCodelens(title, codelensRange))
					continue
				}

				var codeLens *protocol.CodeLens
				accounts, absentAccounts := i.resolveAccounts(signers)

				if len(absentAccounts) > 0 {
					codeLens = i.showAbsentAccounts(absentAccounts, codelensRange)
				} else {
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
	argumentList []Argument,
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

	argsJSON, _ := json.Marshal(argumentList)
	return makeCodeLens(CommandExecuteScript, title, codelensRange, []interface{}{uri, string(argsJSON)})
}

func (i *FlowIntegration) transactionCodeLenses(
	uri protocol.DocumentUri,
	codelensRange protocol.Range,
	pragmaArguments string,
	argumentList []Argument,
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

	argsJSON, _ := json.Marshal(argumentList)

	return makeCodeLens(
		CommandSendTransaction,
		title,
		codelensRange,
		[]interface{}{uri, string(argsJSON), accounts},
	)
}

func (i *FlowIntegration) contractCodeLenses(
	uri protocol.DocumentUri,
	codelensRange protocol.Range,
	name string,
	kind contractKind,
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
	}

	titleBody := "Deploy contract"
	if kind == contractTypeInterface {
		titleBody = "Deploy contract interface"
	}

	title = fmt.Sprintf("%s %s %s to %s",
		prefixOK,
		titleBody,
		name,
		signer,
	)

	return makeCodeLens(CommandDeployContract, title, codelensRange, []interface{}{uri, name, resolvedAddress})
}
