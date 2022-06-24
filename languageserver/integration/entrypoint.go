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

package integration

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
)

type entryPointKind uint8

const (
	entryPointKindUnknown entryPointKind = iota
	entryPointKindScript
	entryPointKindTransaction
)

var SignersRegexp = regexp.MustCompile(`[\w-]+`)

type entryPointInfo struct {
	uri                   protocol.DocumentURI
	documentVersion       int32
	startPos              *ast.Position
	kind                  entryPointKind
	parameters            []*sema.Parameter
	pragmaArgumentStrings []string
	pragmaArguments       [][]Argument
	pragmaSignersStrings  [][]string
	numberOfSigners       int
}

func (e *entryPointInfo) update(uri protocol.DocumentURI, version int32, checker *sema.Checker) {
	if e.documentVersion == version {
		return // do nothing if version haven't changed
	}

	var docString string
	transactionDeclaration := checker.Program.SoleTransactionDeclaration()
	functionDeclaration := sema.FunctionEntryPointDeclaration(checker.Program)

	if transactionDeclaration != nil {
		docString = transactionDeclaration.DocString
		transactionType := checker.Elaboration.TransactionDeclarationTypes[transactionDeclaration]
		e.startPos = &transactionDeclaration.StartPos
		e.kind = entryPointKindTransaction
		e.parameters = transactionType.Parameters
		e.numberOfSigners = len(transactionType.PrepareParameters)
	} else if functionDeclaration != nil {
		docString = functionDeclaration.DocString
		functionType := checker.Elaboration.FunctionDeclarationFunctionTypes[functionDeclaration]
		e.startPos = &functionDeclaration.StartPos
		e.kind = entryPointKindScript
		e.parameters = functionType.Parameters
		e.numberOfSigners = 0
	}

	var pragmaSigners [][]string
	var pragmaArgumentStrings []string
	var pragmaArguments [][]Argument

	if e.startPos != nil {
		parameterTypes := make([]sema.Type, len(e.parameters))

		for i, parameter := range e.parameters {
			parameterTypes[i] = parameter.TypeAnnotation.Type
		}

		if len(e.parameters) > 0 {
			for _, pragmaArgumentString := range parser2.ParseDocstringPragmaArguments(docString) {
				arguments, err := runtime.ParseLiteralArgumentList(pragmaArgumentString, parameterTypes, nil)
				// TODO: record error and show diagnostic
				if err != nil {
					continue
				}

				convertedArguments := make([]Argument, len(arguments))
				for i, arg := range arguments {
					convertedArguments[i] = Argument{arg}
				}

				pragmaArgumentStrings = append(pragmaArgumentStrings, pragmaArgumentString)
				pragmaArguments = append(pragmaArguments, convertedArguments)
			}
		}

		for _, pragmaSignerString := range parser2.ParseDocstringPragmaSigners(docString) {
			signers := SignersRegexp.FindAllString(pragmaSignerString, -1)
			pragmaSigners = append(pragmaSigners, signers)
		}
	}

	e.uri = uri
	e.documentVersion = version
	e.pragmaArgumentStrings = pragmaArgumentStrings
	e.pragmaArguments = pragmaArguments
	e.pragmaSignersStrings = pragmaSigners
}

// codelens shows an execute button when there is exactly one valid entry point
// (valid script function or transaction declaration) and no other actionable declarations.
//
func (e *entryPointInfo) codelens(client flowClient) []*protocol.CodeLens {
	if e.kind == entryPointKindUnknown || e.startPos == nil {
		return nil
	}

	codeLenses := make([]*protocol.CodeLens, 0)
	argumentLists := e.pragmaArguments[:]

	// If there are no parameters and no pragma argument declarations, offer execution using no arguments
	if len(e.parameters) == 0 {
		argumentLists = append(argumentLists, []Argument{})
	}

	for index, argumentList := range argumentLists {
		switch e.kind {
		case entryPointKindScript:
			codeLenses = append(codeLenses, e.scriptCodelens(index, argumentList))

		case entryPointKindTransaction:
			codeLenses = append(codeLenses, e.transactionCodelens(index, argumentList, client)...)
		}
	}

	return codeLenses
}

func (e *entryPointInfo) scriptCodelens(index int, argumentList []Argument) *protocol.CodeLens {
	var pragmaArguments string
	if len(e.parameters) != 0 {
		pragmaArguments = e.pragmaArgumentStrings[index]
	}
	title := fmt.Sprintf("%s Execute script", prefixOK)

	if len(argumentList) > 0 {
		title = fmt.Sprintf("%s with %s", title, pragmaArguments)
	}

	argsJSON, _ := json.Marshal(argumentList)
	arguments, _ := encodeJSONArguments(e.uri, string(argsJSON))
	codelensRange := conversion.ASTToProtocolRange(*e.startPos, *e.startPos)

	return makeCodeLens(CommandExecuteScript, title, codelensRange, arguments)
}

func (e *entryPointInfo) transactionCodelens(index int, argumentList []Argument, client flowClient) []*protocol.CodeLens {
	codeLenses := make([]*protocol.CodeLens, 0)
	codelensRange := conversion.ASTToProtocolRange(*e.startPos, *e.startPos)

	var pragmaArguments string
	if len(e.parameters) != 0 {
		pragmaArguments = e.pragmaArgumentStrings[index]
	}

	signersList := e.pragmaSignersStrings[:]
	if len(signersList) == 0 {
		activeAccount := client.GetActiveClientAccount()
		signersList = append(signersList, []string{activeAccount.Name}) // todo why list of lists
	}

	for _, signers := range signersList {

		if e.numberOfSigners > len(signers) {
			title := fmt.Sprintf(
				"%s Not enough signers. Required: %v, passed: %v",
				prefixError,
				e.numberOfSigners,
				len(signers),
			)
			codeLenses = append(codeLenses, makeActionlessCodelens(title, codelensRange))
			continue
		}

		var codelens *protocol.CodeLens
		accounts, absentAccounts := resolveAccounts(client, signers)

		if len(absentAccounts) > 0 {
			accountsNumeric := "account"
			if len(accounts) > 1 {
				accountsNumeric += "s"
			}

			title := fmt.Sprintf("%s Specified %s %s does not exist",
				prefixError,
				accountsNumeric,
				common.EnumerateWords(absentAccounts, "and"),
			)

			codeLenses = append(codeLenses, makeActionlessCodelens(title, codelensRange))
			continue
		}

		title := fmt.Sprintf(
			"%s Send with %s signed by %s",
			prefixOK,
			pragmaArguments,
			common.EnumerateWords(signers, "and"),
		)
		if len(argumentList) == 0 {
			title = fmt.Sprintf(
				"%s Send signed by %s",
				prefixOK,
				common.EnumerateWords(signers, "and"),
			)
		}

		argsJSON, _ := json.Marshal(argumentList)
		arguments, _ := encodeJSONArguments(e.uri, string(argsJSON), accounts)

		codelens = makeCodeLens(CommandSendTransaction, title, codelensRange, arguments)
		codeLenses = append(codeLenses, codelens)
	}

	return codeLenses
}

// helpers
//

func resolveAccounts(client flowClient, signers []string) ([]flow.Address, []string) {
	var absentAccounts []string
	var resolvedAccounts []flow.Address
	for _, signer := range signers {
		account := client.GetClientAccount(signer)
		if account == nil {
			absentAccounts = append(absentAccounts, signer)
		} else {
			resolvedAccounts = append(resolvedAccounts, account.Address)
		}
	}
	return resolvedAccounts, absentAccounts
}
