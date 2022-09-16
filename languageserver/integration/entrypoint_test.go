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

func buildEntrypoint(t *testing.T, code string) entryPointInfo {
	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	location := common.StringLocation("foo")
	config := &sema.Config{
		AccessCheckMode: sema.AccessCheckModeStrict,
	}
	checker, err := sema.NewChecker(program, location, nil, config)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	entrypoint := entryPointInfo{}
	entrypoint.update("Test", 1, checker)

	return entrypoint
}

func setupMockClient() *mockFlowClient {
	client := &mockFlowClient{}

	accounts := []*clientAccount{{
		Account: &flow.Account{
			Address: flow.HexToAddress("0x1"),
		},
		Name:   "Alice",
		Active: true,
	}, {
		Account: &flow.Account{
			Address: flow.HexToAddress("0x2"),
		},
		Name:   "Bob",
		Active: false,
	}, {
		Account: &flow.Account{
			Address: flow.HexToAddress("0x3"),
		},
		Name:   "Charlie",
		Active: false,
	}}

	for _, account := range accounts {
		client.
			On("GetClientAccount", account.Name).
			Return(account)
	}

	client.
		On("GetClientAccount", "Invalid").
		Return(nil)

	client.
		On("GetActiveClientAccount").
		Return(accounts[0])

	return client
}

func Test_EntrypointUpdate(t *testing.T) {
	t.Run("update entrypoint information", func(t *testing.T) {
		entrypoint := buildEntrypoint(t, `
			/// pragma signers Alice
			/// pragma arguments (hello: 10.0)
			transaction(hello: UFix64) {
				prepare(signer: AuthAccount) {} 
			}`,
		)

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

	t.Run("update script entrypoint information", func(t *testing.T) {
		entrypoint := buildEntrypoint(t, `
			/// pragma arguments (hello: "hi")
			pub fun main(hello: String): String {
				return hello.concat(" world")
			}
		`)

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
}

func Test_Codelensses(t *testing.T) {

	tests := []struct {
		code    string
		title   string
		command string
		ranges  protocol.Range
		args    string
	}{{
		code: `
			/// pragma signers Alice
			/// pragma arguments (hello: 10.0)
			transaction(hello: UFix64) {
				prepare(signer: AuthAccount) {} 
			}`,
		title:   "💡 Send with (hello: 10.0) signed by Alice",
		command: "cadence.server.flow.sendTransaction",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x3, Character: 0x3}, End: protocol.Position{Line: 0x3, Character: 0x4}},
		args:    `"[{\"type\":\"UFix64\",\"value\":\"10.00000000\"}]"`,
	}, {
		code:    `transaction {}`,
		title:   "💡 Send signed by service account",
		command: "cadence.server.flow.sendTransaction",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x0, Character: 0x0}, End: protocol.Position{Line: 0x0, Character: 0x1}},
		args:    `"[]"`,
	}, {
		code: `
			/// pragma arguments (hello: "hi")
			pub fun main(hello: String): String {
				return hello.concat(" world")
			}
		`,
		title:   `💡 Execute script with (hello: "hi")`,
		command: "cadence.server.flow.executeScript",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x2, Character: 0x3}, End: protocol.Position{Line: 0x2, Character: 0x4}},
		args:    `"[{\"type\":\"String\",\"value\":\"hi\"}]"`,
	}, {
		code: `
			transaction {
				prepare(s: AuthAccount) {} 
			}`,
		title:   "💡 Send signed by Alice",
		command: "cadence.server.flow.sendTransaction",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x1, Character: 0x3}, End: protocol.Position{Line: 0x1, Character: 0x4}},
		args:    `"[]"`,
	}, {
		code: `
			/// pragma signers Alice,Bob
			transaction {
				prepare(s1: AuthAccount, s2: AuthAccount) {} 
			}`,
		title:   "💡 Send signed by Alice and Bob",
		command: "cadence.server.flow.sendTransaction",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x2, Character: 0x3}, End: protocol.Position{Line: 0x2, Character: 0x4}},
		args:    `"[]"`,
	}, {
		code: `
			/// pragma signers Alice
			transaction {
				prepare(s1: AuthAccount, s2: AuthAccount) {} 
			}`,
		title:   "🚫 Not enough signers. Required: 2, passed: 1",
		command: "",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x2, Character: 0x3}, End: protocol.Position{Line: 0x2, Character: 0x4}},
	}, {
		code: `
			/// pragma signers Invalid
			transaction {
				prepare(s1: AuthAccount) {} 
			}`,
		title:   "🚫 Specified account Invalid does not exist",
		command: "",
		ranges:  protocol.Range{Start: protocol.Position{Line: 0x2, Character: 0x3}, End: protocol.Position{Line: 0x2, Character: 0x4}},
	}}

	for i, test := range tests {
		entrypoint := buildEntrypoint(t, test.code)
		codelensses := entrypoint.codelens(setupMockClient())

		require.Len(t, codelensses, 1, fmt.Sprintf("test %d", i))
		lens := codelensses[0]
		assert.Equal(t, test.title, lens.Command.Title)
		assert.Equal(t, test.command, lens.Command.Command)
		assert.Equal(t, nil, lens.Data)
		assert.Equal(t, test.ranges, lens.Range)
		if test.args != "" {
			assert.Equal(t, test.args, string(lens.Command.Arguments[1]))
		}
	}

}
