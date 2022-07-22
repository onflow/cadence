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
	"regexp"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/languageserver/protocol"
)

type entryPointKind uint8

const (
	entryPointKindUnknown entryPointKind = iota
	entryPointKindScript
	entryPointKindTransaction
)

var SignersRegexp = regexp.MustCompile(`[\w-]+`)

type entryPointInfo struct {
	documentVersion       int32
	startPos              *ast.Position
	kind                  entryPointKind
	parameters            []*sema.Parameter
	pragmaArgumentStrings []string
	pragmaArguments       [][]Argument
	pragmaSignersStrings  [][]string
	numberOfSigners       int
}

func (i *FlowIntegration) updateEntryPointInfoIfNeeded(
	uri protocol.DocumentURI,
	version int32,
	checker *sema.Checker,
) {
	if i.entryPointInfo[uri].documentVersion == version {
		return
	}

	var startPos *ast.Position
	var kind entryPointKind
	var docString string
	var parameters []*sema.Parameter
	var numberOfSigners int

	transactionDeclaration := checker.Program.SoleTransactionDeclaration()
	functionDeclaration := sema.FunctionEntryPointDeclaration(checker.Program)

	if transactionDeclaration != nil {
		startPos = &transactionDeclaration.StartPos
		kind = entryPointKindTransaction
		docString = transactionDeclaration.DocString
		transactionType := checker.Elaboration.TransactionDeclarationTypes[transactionDeclaration]
		parameters = transactionType.Parameters
		numberOfSigners = len(transactionType.PrepareParameters)
	} else {
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
	var pragmaArguments [][]Argument

	if startPos != nil {

		parameterTypes := make([]sema.Type, len(parameters))

		for i, parameter := range parameters {
			parameterTypes[i] = parameter.TypeAnnotation.Type
		}

		if len(parameters) > 0 {
			for _, pragmaArgumentString := range parser.ParseDocstringPragmaArguments(docString) {
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

		for _, pragmaSignerString := range parser.ParseDocstringPragmaSigners(docString) {
			signers := SignersRegexp.FindAllString(pragmaSignerString, -1)
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
		numberOfSigners:       numberOfSigners,
	}
}
