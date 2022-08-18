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
	CommandGetAccounts         = "cadence.server.flow.getAccounts"
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
		{
			Name:    CommandGetAccounts,
			Handler: c.getAccounts,
		},
	}
}

// sendTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 3 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
//   * the signer names as list
func (c *commands) sendTransaction(args ...json.RawMessage) (any, error) {
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
		return nil, fmt.Errorf("invalid transaction arguments: %s", args[1])
	}

	txArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction arguments cadence encoding format: %s, error: %s", argsJSON, err)
	}

	var signerList []string
	err = json.Unmarshal(args[2], &signerList)
	if err != nil {
		return nil, fmt.Errorf("invalid signer list: %s", args[2])
	}

	signerAddresses := make([]flow.Address, 0)
	for _, name := range signerList {
		account := c.client.GetClientAccount(name)
		if account == nil {
			return nil, fmt.Errorf("signer account with name %s doesn't exist", name)
		}

		signerAddresses = append(signerAddresses, account.Address)
	}

	tx, txResult, err := c.client.SendTransaction(signerAddresses, location, txArgs)
	if err != nil {
		return nil, fmt.Errorf("transaction error: %w", err)
	}

	return fmt.Sprintf("Transaction %s with ID %s", txResult.Status, tx.ID()), err
}

// executeScript handles executing a script defined in the source document.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (c *commands) executeScript(args ...json.RawMessage) (any, error) {
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
// There should be 1 argument:
//	 * name of the new active account
func (c *commands) switchActiveAccount(args ...json.RawMessage) (any, error) {
	err := server.CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, fmt.Errorf("arguments error: %w", err)
	}

	var name string
	err = json.Unmarshal(args[0], &name)
	if err != nil {
		return nil, fmt.Errorf("invalid name argument value: %s", args[0])
	}

	err = c.client.SetActiveClientAccount(name)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Account switched to %s", name), nil
}

// getAccounts return the client account list with information about the active client.
func (c *commands) getAccounts(_ ...json.RawMessage) (any, error) {
	return c.client.GetClientAccounts(), nil
}

// createAccount creates a new account and returns its address.
func (c *commands) createAccount(_ ...json.RawMessage) (any, error) {
	account, err := c.client.CreateAccount()
	if err != nil {
		return nil, fmt.Errorf("create account error: %w", err)
	}

	return account, nil
}

// deployContract deploys the contract to the configured account with the code of the given
// file.
//
// There should be exactly 3 arguments:
//   * the DocumentURI of the file to submit
//   * the name of the contract
//   * the signer names as list
func (c *commands) deployContract(args ...json.RawMessage) (any, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return nil, fmt.Errorf("arguments error: %w", err)
	}

	location, err := parseLocation(args[0])
	if err != nil {
		return nil, err
	}

	var name string
	err = json.Unmarshal(args[1], &name)
	if err != nil {
		return nil, fmt.Errorf("invalid name argument: %s", args[1])
	}

	var signerName string
	err = json.Unmarshal(args[2], &signerName)
	if err != nil {
		return nil, fmt.Errorf("invalid signer name: %s", args[2])
	}

	var account *clientAccount
	if signerName == "" { // choose default active account
		account = c.client.GetActiveClientAccount()
	} else {
		account = c.client.GetClientAccount(signerName)
		if account == nil {
			return nil, fmt.Errorf("signer account with name %s doesn't exist", signerName)
		}
	}

	_, deployError := c.client.DeployContract(account.Address, name, location)
	if deployError != nil {
		return nil, fmt.Errorf("error deploying contract: %w", deployError)
	}

	return fmt.Sprintf("Contract %s has been deployed to account %s", name, account.Name), err
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
