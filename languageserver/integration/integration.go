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
	project "github.com/onflow/flow-cli/pkg/flowcli/project"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"

	"github.com/onflow/flow-cli/pkg/flowcli/services"
)

type EmulatorState byte

const (
	EmulatorOffline EmulatorState = iota
	EmulatorStarting
	EmulatorOnline
)

var addressToName = map[string]string{
	"0x01": "alice",
	"0x02": "bob",
	"0x03": "charlie",
	"0x04": "dave",
}

var nameToAddress = map[string]string{
	"alice": "0x01",
	"bob": "0x02",
	"charlie": "0x03",
	"dave": "0x04",
}

type FlowIntegration struct {
	server     *server.Server
	config     Config
	flowClient *client.Client

	accounts       map[flow.Address]AccountPrivateKey
	activeAccount  string
	activeAddress  flow.Address
	serviceAddress flow.Address

	entryPointInfo map[protocol.DocumentUri]entryPointInfo

	emulatorState  EmulatorState
	sharedServices *services.Services
	project        *project.Project
}

func NewFlowIntegration(s *server.Server, enableFlowClient bool) (*FlowIntegration, error) {
	integration := &FlowIntegration{
		server:         s,
		accounts:       make(map[flow.Address]AccountPrivateKey),
		entryPointInfo: map[protocol.DocumentUri]entryPointInfo{},
	}

	options := []server.Option{
		server.WithDiagnosticProvider(integration.diagnostics),
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
