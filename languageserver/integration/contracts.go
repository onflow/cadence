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
	"fmt"

	"github.com/onflow/cadence/languageserver/conversion"
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
)

type contractKind uint8

const (
	contractTypeUnknown contractKind = iota
	contractTypeDeclaration
	contractTypeInterface
)

type contractInfo struct {
	uri                   protocol.DocumentURI
	documentVersion       int32
	startPos              *ast.Position
	kind                  contractKind
	name                  string
	parameters            []*sema.Parameter
	pragmaArgumentStrings []string
	pragmaArguments       [][]Argument
	pragmaSignersStrings  [][]string
}

func (c *contractInfo) update(uri protocol.DocumentURI, version int32, checker *sema.Checker) {
	if c.documentVersion == version {
		return // if no change in version do nothing
	}

	var docString string
	contractDeclaration := checker.Program.SoleContractDeclaration()
	contractInterfaceDeclaration := checker.Program.SoleContractInterfaceDeclaration()

	if contractDeclaration != nil {
		docString = contractDeclaration.DocString
		contractType := checker.Elaboration.CompositeDeclarationTypes[contractDeclaration]
		c.name = contractDeclaration.Identifier.Identifier
		c.startPos = &contractDeclaration.StartPos
		c.kind = contractTypeDeclaration
		c.parameters = contractType.ConstructorParameters
	} else if contractInterfaceDeclaration != nil {
		docString = contractInterfaceDeclaration.DocString
		c.name = contractInterfaceDeclaration.Identifier.Identifier
		c.startPos = &contractInterfaceDeclaration.StartPos
		c.kind = contractTypeInterface
	}

	var pragmaSigners [][]string
	var pragmaArgumentStrings []string
	var pragmaArguments [][]Argument

	if c.startPos != nil {
		parameterTypes := make([]sema.Type, len(c.parameters))

		for i, parameter := range c.parameters {
			parameterTypes[i] = parameter.TypeAnnotation.Type
		}

		if len(c.parameters) > 0 {
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

	c.uri = uri
	c.documentVersion = version
	c.pragmaSignersStrings = pragmaSigners
	c.pragmaArguments = pragmaArguments
	c.pragmaArgumentStrings = pragmaArgumentStrings
}

func (c contractInfo) codelens(client flowClient) []*protocol.CodeLens {
	if c.kind == contractTypeUnknown || c.startPos == nil {
		return nil
	}

	signersList := c.pragmaSignersStrings[:]
	if len(signersList) == 0 {
		activeAccount := client.GetActiveClientAccount().Address.String()
		signersList = append(signersList, []string{activeAccount}) // todo refactor list in list
	}

	codelensRange := conversion.ASTToProtocolRange(*c.startPos, *c.startPos)
	var codeLenses []*protocol.CodeLens

	for _, signers := range signersList {
		var title string
		signer := signers[0] // todo refactor list in list

		account := client.GetClientAccount(signer)
		if account == nil {
			title = fmt.Sprintf("%s Specified account %s does not exist", prefixError, signer)
			codeLenses = append(codeLenses, makeActionlessCodelens(title, codelensRange))
		}

		titleBody := "Deploy contract"
		if c.kind == contractTypeInterface {
			titleBody = "Deploy contract interface"
		}

		title = fmt.Sprintf("%s %s %s to %s", prefixOK, titleBody, c.name, signers)
		arguments, _ := encodeJSONArguments(c.uri, c.name, account.Address)
		codelens := makeCodeLens(CommandDeployContract, title, codelensRange, arguments)
		codeLenses = append(codeLenses, codelens)
	}

	return codeLenses
}
