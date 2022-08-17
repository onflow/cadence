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

	"github.com/onflow/cadence"

	"github.com/onflow/cadence/languageserver/protocol"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"
)

func Test_TransactionEntrypoint(t *testing.T) {
	const code = `
	/// pragma signers Alice
	/// pragma arguments (hello: 10.0)
	transaction(hello: UFix64) {
		prepare(signer: AuthAccount) {} 
	}`

	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	checker, err := sema.NewChecker(program, common.StringLocation("foo"), nil, false)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	client := &mockFlowClient{}

	t.Run("update entrypoint information", func(t *testing.T) {
		entrypoint := entryPointInfo{}
		entrypoint.update("Test", 1, checker)
		val, _ := cadence.NewUFix64("10.0")

		assert.Len(t, entrypoint.pragmaSignerNames, 1)
		assert.Equal(t, entrypoint.pragmaSignerNames[0], "Alice")
		assert.Len(t, entrypoint.parameters, 1)
		assert.Equal(t, entrypoint.parameters[0].Identifier, "hello")
		assert.Equal(t, entrypoint.parameters[0].TypeAnnotation.String(), "UFix64")
		assert.Equal(t, entrypoint.pragmaArgumentStrings, []string{"(hello: 10.0)"})
		assert.Equal(t, entrypoint.pragmaArguments, [][]Argument{{Argument{val}}})
		assert.Equal(t, entrypoint.uri, protocol.DocumentURI("Test"))
		assert.Equal(t, entrypoint.kind, entryPointKindTransaction)
		assert.Len(t, entrypoint.pragmaArguments, 1)
	})

	t.Run("get codelensses", func(t *testing.T) {
		entrypoint := entryPointInfo{}
		entrypoint.update("Test", 1, checker)

		alice := &clientAccount{
			Account: &flow.Account{
				Address: flow.HexToAddress("0x1"),
			},
			Name:   "Alice",
			Active: true,
		}

		client.
			On("GetClientAccount", "Alice").
			Return(alice)

		codelensses := entrypoint.codelens(client)

		require.Len(t, codelensses, 1)
		assert.Equal(t, "💡 Send with (hello: 10.0) signed by Alice", codelensses[0].Command.Title)
		assert.Equal(t, "cadence.server.flow.sendTransaction", codelensses[0].Command.Command)
		assert.Equal(t, nil, codelensses[0].Data)
		assert.Equal(t, codelensses[0].Range, protocol.Range{Start: protocol.Position{Line: 0x3, Character: 0x1}, End: protocol.Position{Line: 0x3, Character: 0x2}})

		assert.Equal(t, `"[{\"type\":\"UFix64\",\"value\":\"10.00000000\"}]"`, string(codelensses[0].Command.Arguments[1]))
	})

}

func Test_ScriptEntrypoint(t *testing.T) {
	const code = `
		/// pragma arguments (hello: "hi")
		pub fun main(hello: String): String {
			return hello.concat(" world")
		}
	`

	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	checker, err := sema.NewChecker(program, common.StringLocation("foo"), nil, false)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	client := &mockFlowClient{}

	t.Run("update entrypoint information", func(t *testing.T) {
		entrypoint := entryPointInfo{}
		entrypoint.update("Test", 1, checker)

		val, _ := cadence.NewString("hi")

		assert.Len(t, entrypoint.pragmaSignerNames, 0)
		assert.Len(t, entrypoint.parameters, 1)
		assert.Equal(t, entrypoint.parameters[0].Identifier, "hello")
		assert.Equal(t, entrypoint.parameters[0].TypeAnnotation.String(), "String")
		assert.Equal(t, entrypoint.pragmaArgumentStrings, []string{`(hello: "hi")`})
		assert.Equal(t, entrypoint.pragmaArguments, [][]Argument{{Argument{val}}})
		assert.Equal(t, entrypoint.uri, protocol.DocumentURI("Test"))
		assert.Equal(t, entrypoint.kind, entryPointKindScript)
		assert.Len(t, entrypoint.pragmaArguments, 1)
	})

	t.Run("get codelensses", func(t *testing.T) {
		entrypoint := entryPointInfo{}
		entrypoint.update("Test", 1, checker)

		codelensses := entrypoint.codelens(client)

		require.Len(t, codelensses, 1)
		assert.Equal(t, `💡 Execute script with (hello: "hi")`, codelensses[0].Command.Title)
		assert.Equal(t, "cadence.server.flow.executeScript", codelensses[0].Command.Command)
		assert.Equal(t, nil, codelensses[0].Data)
		assert.Equal(t, protocol.Range{Start: protocol.Position{Line: 0x2, Character: 0x2}, End: protocol.Position{Line: 0x2, Character: 0x3}}, codelensses[0].Range)

		assert.Equal(t, `"[{\"type\":\"String\",\"value\":\"hi\"}]"`, string(codelensses[0].Command.Arguments[1]))
	})
}
