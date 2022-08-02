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
	"testing"

	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence/languageserver/protocol"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func Test_ContractUpdate(t *testing.T) {
	const code = `
      /// pragma signers Alice
	  pub contract HelloWorld {
			pub let greeting: String

			pub fun hello(): String {
				return self.greeting
			}

			init() {
				self.greeting = "hello"
			}
     }
        `
	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	checker, err := sema.NewChecker(program, common.StringLocation("foo"), nil, false)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	client := &mockFlowClient{}

	t.Run("update contract information", func(t *testing.T) {
		contract := &contractInfo{}
		contract.update("Hello", 1, checker)

		assert.Equal(t, protocol.DocumentURI("Hello"), contract.uri)
		assert.Equal(t, "HelloWorld", contract.name)
		assert.Equal(t, contractTypeDeclaration, contract.kind)
		assert.Equal(t, nil, contract.pragmaArguments)
		assert.Equal(t, nil, contract.pragmaArgumentStrings)
		assert.Equal(t, []string{"Alice"}, contract.pragmaSignersNames)

		assert.Len(t, contract.parameters, 1)
		assert.Equal(t, "a", contract.parameters[0].Identifier)
		assert.Equal(t, "", contract.parameters[0].Label)
	})

	t.Run("get codelenses", func(t *testing.T) {
		contract := &contractInfo{}
		contract.update("Hello", 1, checker)

		alice := &clientAccount{
			Account: &flow.Account{
				Address: flow.HexToAddress("0x1"),
			},
			Name:   "Alice",
			Active: true,
		}

		client.
			On("GetActiveClientAccount").
			Return(alice)

		client.
			On("GetClientAccount", "Alice").
			Return(alice)

		codelenses := contract.codelens(client)

		assert.Len(t, codelenses, 1)
		assert.Equal(t, "💡 Deploy contract HelloWorld to Alice", codelenses[0].Command.Title)
		assert.Equal(t, "cadence.server.flow.deployContract", codelenses[0].Command.Command)
		assert.Equal(t, nil, codelenses[0].Data)
		assert.Equal(t, protocol.Range{
			Start: protocol.Position{Line: 2, Character: 3},
			End:   protocol.Position{Line: 2, Character: 4},
		}, codelenses[0].Range)
	})
}
