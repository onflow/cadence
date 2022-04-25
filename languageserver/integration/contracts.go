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
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"

	"github.com/onflow/cadence/languageserver/protocol"
)

type contractKind uint8

const (
	contractTypeUnknown contractKind = iota
	contractTypeDeclaration
	contractTypeInterface
)

type contractInfo struct {
	documentVersion       float64
	startPos              *ast.Position
	kind                  contractKind
	name                  string
	parameters            []*sema.Parameter
	pragmaArgumentStrings []string
	pragmaArguments       [][]Argument
	pragmaSignersStrings  [][]string
}

func (i FlowIntegration) updateContractInfoIfNeeded(
	uri protocol.DocumentUri,
	version float64,
	checker *sema.Checker,
) {
	if i.contractInfo[uri].documentVersion == version {
		return
	}

	var name string
	var startPos *ast.Position
	var kind contractKind
	var docString string
	var parameters []*sema.Parameter

	contractDeclaration := checker.Program.SoleContractDeclaration()
	contractInterfaceDeclaration := checker.Program.SoleContractInterfaceDeclaration()

	if contractDeclaration != nil {
		name = contractDeclaration.Identifier.Identifier
		startPos = &contractDeclaration.StartPos
		kind = contractTypeDeclaration
		docString = contractDeclaration.DocString
		contractType := checker.Elaboration.CompositeDeclarationTypes[contractDeclaration]
		parameters = contractType.ConstructorParameters
	} else if contractInterfaceDeclaration != nil {
		name = contractInterfaceDeclaration.Identifier.Identifier
		startPos = &contractInterfaceDeclaration.StartPos
		kind = contractTypeInterface
		docString = contractInterfaceDeclaration.DocString
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

	i.contractInfo[uri] = contractInfo{
		documentVersion:       version,
		startPos:              startPos,
		kind:                  kind,
		name:                  name,
		parameters:            parameters,
		pragmaArgumentStrings: pragmaArgumentStrings,
		pragmaArguments:       pragmaArguments,
		pragmaSignersStrings:  pragmaSigners,
	}
}
