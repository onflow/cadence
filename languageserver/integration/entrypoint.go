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
	"github.com/onflow/cadence"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"regexp"
)

type entryPointKind uint8

const (
	entryPointKindUnknown entryPointKind = iota
	entryPointKindScript
	entryPointKindTransaction
)

type entryPointInfo struct {
	documentVersion       float64
	startPos              *ast.Position
	kind                  entryPointKind
	parameters            []*sema.Parameter
	pragmaArgumentStrings []string
	pragmaArguments       [][]cadence.Value
	pragmaSignersStrings  [][]string
}

func (i *FlowIntegration) updateEntryPointInfoIfNeeded(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) {
	if i.entryPointInfo[uri].documentVersion == version {
		return
	}

	var startPos *ast.Position
	var kind entryPointKind
	var docString string
	var parameters []*sema.Parameter

	transactionDeclaration := checker.Program.SoleTransactionDeclaration()
	if transactionDeclaration != nil {
		startPos = &transactionDeclaration.StartPos
		kind = entryPointKindTransaction
		docString = transactionDeclaration.DocString
		transactionType := checker.Elaboration.TransactionDeclarationTypes[transactionDeclaration]
		parameters = transactionType.Parameters
	} else {

		functionDeclaration := sema.FunctionEntryPointDeclaration(checker.Program)
		if functionDeclaration != nil {
			startPos = &functionDeclaration.StartPos
			kind = entryPointKindScript
			docString = functionDeclaration.DocString
			functionType := checker.Elaboration.FunctionDeclarationFunctionTypes[functionDeclaration]
			parameters = functionType.Parameters
		}
	}

	var pragmaSigners [][]string
	var pragmaArgumentStrings []string
	var pragmaArguments [][]cadence.Value

	if startPos != nil {

		parameterTypes := make([]sema.Type, len(parameters))

		for i, parameter := range parameters {
			parameterTypes[i] = parameter.TypeAnnotation.Type
		}

		for _, pragmaArgumentString := range parser2.ParseDocstringPragmaArguments(docString) {
			arguments, err := runtime.ParseLiteralArgumentList(pragmaArgumentString, parameterTypes)
			// TODO: record error and show diagnostic
			if err != nil {
				continue
			}

			pragmaArgumentStrings = append(pragmaArgumentStrings, pragmaArgumentString)
			pragmaArguments = append(pragmaArguments, arguments)
		}

		for _, pragmaSignerString := range parser2.ParseDocstringPragmaSigners(docString) {
			signers := parseSignersList(pragmaSignerString)
			pragmaSigners = append(pragmaSigners, signers)
		}
	}

	i.entryPointInfo[uri] = entryPointInfo{
		documentVersion:       version,
		startPos:              startPos,
		kind:                  kind,
		parameters:            parameters,
		pragmaArgumentStrings: pragmaArgumentStrings,
		pragmaArguments:       pragmaArguments,
		pragmaSignersStrings:  pragmaSigners,
	}
}

func parseSignersList(signerList string) []string {
	var re = regexp.MustCompile(`[a-zA-Z]+`)
	result := re.FindAllString(signerList, -1)

	return result
}
