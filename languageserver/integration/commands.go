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
	"fmt"
	"net/url"

	"github.com/onflow/cadence/languageserver/server"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-go-sdk"
)

const (
	CommandSendTransaction     = "cadence.server.flow.sendTransaction"
	CommandExecuteScript       = "cadence.server.flow.executeScript"
	CommandDeployContract      = "cadence.server.flow.deployContract"
	CommandCreateAccount       = "cadence.server.flow.createAccount"
	CommandSwitchActiveAccount = "cadence.server.flow.switchActiveAccount"
)

type commands struct {
	client flowClient
}

func (c *commands) getAll() []server.Command {
	return []server.Command{
		{
			Name:    CommandSendTransaction,
			Handler: c.sendTransaction,
		},
		{
			Name:    CommandExecuteScript,
			Handler: c.executeScript,
		},
		{
			Name:    CommandDeployContract,
			Handler: c.deployContract,
		},
		{
			Name:    CommandSwitchActiveAccount,
			Handler: c.switchActiveAccount,
		},
		{
			Name:    CommandCreateAccount,
			Handler: c.createAccount,
		},
	}
}

// sendTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (c *commands) sendTransaction(args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return nil, fmt.Errorf("arguments error: %w", err)
	}

	location, err := parseLocation(args[0])
	if err != nil {
		return nil, err
	}

	var argsJSON string
	err = json.Unmarshal(args[1], &argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction arguments: %v", args[1])
	}

	var signerList []any
	err = json.Unmarshal(args[2], &argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid signer list: %v", args[2])
	}

	signers := make([]flow.Address, len(signerList))
	for i, v := range signerList {
		signers[i] = flow.HexToAddress(v.(string))
	}

	txArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid transactions arguments cadence encoding format: %v", argsJSON)
	}

	txResult, err := c.client.SendTransaction(signers, location, txArgs)
	if err != nil {
		return nil, fmt.Errorf("transaction error: %w", err)
	}

	return fmt.Sprintf("Transaction status: %s", txResult.Status.String()), err
}

// executeScript handles executing a script defined in the source document.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (c *commands) executeScript(args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, fmt.Errorf("arguments error: %w", err)
	}

	location, err := parseLocation(args[0])
	if err != nil {
		return nil, err
	}

	var argsJSON string
	err = json.Unmarshal(args[1], &argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid script arguments: %s", args[1])
	}

	scriptArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid script arguments cadence encoding format: %s, error: %s", argsJSON, err)
	}

	scriptResult, err := c.client.ExecuteScript(location, scriptArgs)
	if err != nil {
		return nil, fmt.Errorf("script error: %w", err)
	}

	return fmt.Sprintf("Result: %s", scriptResult.String()), nil
}

// switchActiveAccount sets the account that is currently active and could be used
// when submitting transactions.
//
// There should be 2 arguments:
//	 * name of the new active account
func (c *commands) switchActiveAccount(args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, fmt.Errorf("arguments error: %w", err)
	}

	var name string
	err = json.Unmarshal(args[0], &name)
	if err != nil {
		return nil, fmt.Errorf("invalid name argument: %#+v: %w", args[0], err)
	}

	err = c.client.SetActiveClientAccount(name)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Account switched to %s", name), nil
}

// createAccount creates a new account and returns its address.
func (c *commands) createAccount(args ...json.RawMessage) (interface{}, error) {
	account, err := c.client.CreateAccount()
	if err != nil {
		return nil, fmt.Errorf("create account error: %w", err)
	}

	return account, nil
}

// deployContract deploys the contract to the configured account with the code of the given
// file.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the name of the contract
func (c *commands) deployContract(args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return flow.Address{}, fmt.Errorf("arguments error: %w", err)
	}

	location, err := parseLocation(args[0])
	if err != nil {
		return nil, err
	}

	var name string
	err = json.Unmarshal(args[1], &name)
	if err != nil {
		return nil, fmt.Errorf("invalid name argument: %#+v: %w", args[1], err)
	}

	var rawAddress string
	err = json.Unmarshal(args[2], &rawAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid address argument: %#+v: %w", args[2], err)
	}
	address := flow.HexToAddress(rawAddress)

	_, deployError := c.client.DeployContract(address, name, location)
	if deployError != nil {
		return nil, fmt.Errorf("error deploying contract: %w", deployError)
	}

	return fmt.Sprintf("Status: contract %s has been deployed to %s", name, rawAddress), err
}

func parseLocation(arg []byte) (*url.URL, error) {
	var uri string
	err := json.Unmarshal(arg, &uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI argument: %s", arg)
	}

	location, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid path argument: %s", uri)
	}

	return location, nil
}
