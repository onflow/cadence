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

package server

import (
	"testing"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkProgram(t *testing.T, text string) []protocol.Diagnostic {
	server, err := NewServer()
	assert.NoError(t, err)
	diagnostics, err := server.getDiagnostics("", text, 0, func(_ *protocol.LogMessageParams) {})
	assert.NoError(t, err)
	return diagnostics
}

func TestLinting(t *testing.T) {
	t.Parallel()

	t.Run("number casting", func(t *testing.T) {

		t.Parallel()

		diagnostics := checkProgram(t, `pub fun test() {
			let x = Int8(-1)
		}`)

		require.Equal(t, 1, len(diagnostics))
		diagnostic := diagnostics[0]

		// casting fix should have a non-nil code-action
		require.NotNil(t, diagnostic.Data)
		// but we can't consistently compare the uuid
		diagnostic.Data = nil

		require.Equal(t, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 11},
				End:   protocol.Position{Line: 1, Character: 19},
			},
			Severity: protocol.SeverityInformation,
			Message:  "consider replacing with: `-1 as Int8`",
		}, diagnostic)
	})

	t.Run("force", func(t *testing.T) {

		t.Parallel()

		diagnostics := checkProgram(t, `pub fun test() {
			let x = 3
			let y = x!
		}`)

		require.Equal(t, 1, len(diagnostics))
		diagnostic := diagnostics[0]

		// forcing fix should have a non-nil code-action
		require.NotNil(t, diagnostic.Data)
		// but we can't consistently compare the uuid
		diagnostic.Data = nil

		require.Equal(t, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 11},
				End:   protocol.Position{Line: 2, Character: 13},
			},
			Severity: protocol.SeverityInformation,
			Message:  "unnecessary force operator",
		}, diagnostic)
	})

	t.Run("redundant cast", func(t *testing.T) {

		t.Parallel()

		diagnostics := checkProgram(t, `pub fun test() {
			let x = true as! Bool
		}`)

		require.Equal(t, 1, len(diagnostics))
		diagnostic := diagnostics[0]

		require.Equal(t, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 11},
				End:   protocol.Position{Line: 1, Character: 24},
			},
			Severity: protocol.SeverityInformation,
			Message:  "force cast ('as!') from `Bool` to `Bool` always succeeds",
		}, diagnostic)
	})

	t.Run("no lints in the presence of a type error", func(t *testing.T) {

		t.Parallel()

		diagnostics := checkProgram(t, `pub fun test() {
			let x = true as! Bool
			let y: Bool = 3
		}`)

		require.Equal(t, 1, len(diagnostics))
		diagnostic := diagnostics[0]

		require.Equal(t, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 17},
				End:   protocol.Position{Line: 2, Character: 18},
			},
			Severity: protocol.SeverityError,
			Message:  "mismatched types. expected `Bool`, got `Int`",
		}, diagnostic)
	})
}
