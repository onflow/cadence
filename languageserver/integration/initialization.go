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
	project "github.com/onflow/flow-cli/pkg/flowcli/project"
	"github.com/onflow/flow-cli/pkg/flowcli/services"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

func (i *FlowIntegration) initialize(initializationOptions interface{}) error {

	// Parse the configuration options sent from the client
	conf, err := configFromInitializationOptions(initializationOptions)
	if err != nil {
		return err
	}
	i.config = conf

	i.flowClient, err = client.New(
		i.config.EmulatorAddr,
		grpc.WithInsecure(),
	)

	i.emulatorState = conf.emulatorState

	// TODO: get this path from initializationOptions
	configurationPaths := []string{"/home/max/Desktop/cadence/flow.json"}

	// TODO: process error if project could not be initialized from specified config
	project, _ := project.Load(configurationPaths)
	if err != nil {
		return nil
	}

	i.project = project

	// TODO: get this value from config file
	host := conf.EmulatorAddr
	// TODO: process error here
	gate, _ := gateway.NewGrpcGateway(host)

	logger := output.NewStdoutLogger(output.NoneLog)

	if err != nil {
		return nil
	}
	i.sharedServices = services.NewServices(gate, project, logger)
	if err != nil {
		return err
	}

	return nil
}
