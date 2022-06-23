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
	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/spf13/afero"
)

type FlowIntegration struct {
	server *server.Server
	config Config

	entryPointInfo map[protocol.DocumentURI]entryPointInfo
	contractInfo   map[protocol.DocumentURI]contractInfo

	activeAccount ClientAccount

	flowClient flowClient
	loader     flowkit.ReaderWriter
}

func NewFlowIntegration(s *server.Server, enableFlowClient bool) (*FlowIntegration, error) {
	integration := &FlowIntegration{
		server:         s,
		entryPointInfo: map[protocol.DocumentURI]entryPointInfo{},
		contractInfo:   map[protocol.DocumentURI]contractInfo{},
		loader:         &afero.Afero{Fs: afero.NewOsFs()},
	}

	options := []server.Option{
		server.WithDiagnosticProvider(diagnostics),
		server.WithStringImportResolver(resolveFileImport),
	}

	if enableFlowClient {
		options = append(options,
			server.WithInitializationOptionsHandler(integration.initialize),
			server.WithCodeLensProvider(integration.codeLenses),
			server.WithAddressImportResolver(integration.resolveAddressImport),
			server.WithAddressContractNamesResolver(integration.resolveAddressContractNames),
		)

		for _, command := range integration.commands() {
			options = append(options, server.WithCommand(command))
		}
	}

	err := s.SetOptions(options...)
	if err != nil {
		return nil, err
	}

	return integration, nil
}
