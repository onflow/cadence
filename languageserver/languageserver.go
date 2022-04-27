//go:build !wasm
// +build !wasm

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

package languageserver

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/onflow/cadence/languageserver/integration"
	"github.com/onflow/cadence/languageserver/server"
)

func RunWithStdio(enableFlowClient bool) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		print(
			"This program implements the Language Server Protocol for Cadence.\n" +
				"Please check the documentation on how to run it.\n" +
				"It does nothing in a terminal, it should be run with an editor/IDE.\n",
		)
		os.Exit(1)
	}

	languageServer, err := server.NewServer()
	if err != nil {
		panic(err)
	}

	_, err = integration.NewFlowIntegration(languageServer, enableFlowClient)
	if err != nil {
		panic(err)
	}

	stream := jsonrpc2.NewBufferedStream(
		server.StdinStdoutReadWriterCloser{},
		jsonrpc2.VSCodeObjectCodec{},
	)

	<-languageServer.Start(stream)
}
