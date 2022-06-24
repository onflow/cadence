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

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/runtime/sema"
)

const (
	// Codelens message prefixes
	prefixOK    = "💡"
	prefixError = "🚫"
)

func (i *FlowIntegration) codeLenses(
	uri protocol.DocumentURI,
	version int32,
	checker *sema.Checker,
) (
	[]*protocol.CodeLens,
	error,
) {
	var actions []*protocol.CodeLens

	// Add code lenses for contracts and contract interfaces
	contract := i.contractInfo[uri]
	contract.update(uri, version, checker)
	actions = append(actions, contract.codelens(i.client)...)

	// Add code lenses for scripts and transactions
	entryPoint := i.entryPointInfo[uri]
	entryPoint.update(uri, version, checker)
	actions = append(actions, entryPoint.codelens(i.client)...)

	return actions, nil
}

func makeActionlessCodelens(title string, lensRange protocol.Range) *protocol.CodeLens {
	return &protocol.CodeLens{
		Range: lensRange,
		Command: protocol.Command{
			Title: title,
		},
	}
}

func makeCodeLens(
	command string,
	title string,
	lensRange protocol.Range,
	arguments []json.RawMessage,
) *protocol.CodeLens {
	return &protocol.CodeLens{
		Range: lensRange,
		Command: protocol.Command{
			Title:     title,
			Command:   command,
			Arguments: arguments,
		},
	}
}

func encodeJSONArguments(args ...interface{}) ([]json.RawMessage, error) {
	result := make([]json.RawMessage, 0, len(args))
	for _, arg := range args {
		argJSON, err := json.Marshal(arg)
		if err != nil {
			return nil, err
		}
		result = append(result, argJSON)
	}
	return result, nil
}
