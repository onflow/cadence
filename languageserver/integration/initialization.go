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
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/gateway"
	"github.com/onflow/flow-cli/pkg/flowkit/output"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/spf13/afero"
)

func (i *FlowIntegration) initialize(initializationOptions any) error {
	// Parse the configuration options sent from the client
	conf, err := configFromInitializationOptions(initializationOptions)
	if err != nil {
		return err
	}
	i.config = conf
	i.emulatorState = conf.emulatorState
	i.activeAccount = conf.activeAccount

	configurationPaths := []string{conf.configPath}

	loader := &afero.Afero{Fs: afero.NewOsFs()}
	state, err := flowkit.Load(configurationPaths, loader)
	if err != nil {
		return err
	}

	network, err := state.Networks().ByName("emulator")
	if err != nil {
		return err
	}
	logger := output.NewStdoutLogger(output.NoneLog)

	grpcGateway, err := gateway.NewGrpcGateway(network.Host)
	if err != nil {
		return err
	}

	i.sharedServices = services.NewServices(grpcGateway, state, logger)
	i.state = state

	return nil
}
