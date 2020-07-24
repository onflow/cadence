/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"

	"github.com/onflow/cadence/languageserver/server"
)

type FlowIntegration struct {
	server         *server.Server
	config         Config
	flowClient     *client.Client
	accounts       map[flow.Address]AccountPrivateKey
	activeAccount  flow.Address
	serviceAddress flow.Address
}

func NewFlowIntegration(s *server.Server) (*FlowIntegration, error) {
	integration := &FlowIntegration{
		server:   s,
		accounts: make(map[flow.Address]AccountPrivateKey),
	}

	var commandOptions []server.Option
	for _, command := range integration.commands() {
		commandOptions = append(commandOptions, server.WithCommand(command))
	}

	err := s.SetOptions(
		server.WithInitializationOptionsHandler(integration.initialize),
		server.WithDiagnosticProvider(integration.diagnostics),
		server.WithCodeLensProvider(integration.codeLenses),
		server.WithAddressImportResolver(integration.resolveAccountImport),
		server.WithStringImportResolver(resolveFileImport),
	)
	if err != nil {
		return nil, err
	}

	return integration, nil
}
