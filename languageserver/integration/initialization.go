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
	"github.com/onflow/flow-cli/pkg/flowcli/gateway"
	"github.com/onflow/flow-cli/pkg/flowcli/output"
	"github.com/onflow/flow-cli/pkg/flowcli/project"
	"github.com/onflow/flow-cli/pkg/flowcli/services"
)

func (i *FlowIntegration) initialize(initializationOptions interface{}) error {
	// Parse the configuration options sent from the client
	conf, err := configFromInitializationOptions(initializationOptions)
	if err != nil {
		return err
	}
	i.config = conf
	i.emulatorState = conf.emulatorState
	i.activeAccount = conf.activeAccount

	configurationPaths := []string{conf.configPath}

	flowProject, err := project.Load(configurationPaths)
	if err != nil {
		return err
	}

	host := flowProject.NetworkByName("emulator").Host
	logger := output.NewStdoutLogger(output.NoneLog)

	grpcGateway, err := gateway.NewGrpcGateway(host)
	if err != nil {
		return err
	}

	i.sharedServices = services.NewServices(grpcGateway, flowProject, logger)
	i.project = flowProject

	return nil
}
