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

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-go-sdk"
)

const (
	CommandSendTransaction       = "cadence.server.flow.sendTransaction"
	CommandExecuteScript         = "cadence.server.flow.executeScript"
	CommandDeployContract        = "cadence.server.flow.deployContract"
	CommandCreateAccount         = "cadence.server.flow.createAccount"
	CommandCreateDefaultAccounts = "cadence.server.flow.createDefaultAccounts" // todo why do we need this, should be default on startup
	CommandSwitchActiveAccount   = "cadence.server.flow.switchActiveAccount"
	CommandInitAccountManager    = "cadence.server.flow.initAccountManager" // todo remove

	ErrorMessageEmulator          = "emulator error"
	ErrorMessageServiceAccount    = "service account error"
	ErrorMessageTransactionError  = "transaction error"
	ErrorMessageServiceAccountKey = "service account private key error"
	ErrorMessageAccountCreate     = "create account error"
	ErrorMessageAccountStore      = "store account error"
	ErrorMessagePrivateKeyDecoder = "private key decoder error"
	ErrorMessageDeploy            = "deployment error"
	ErrorMessageScriptExecution   = "script error"
	ErrorMessageArguments         = "arguments error"
)

func (i *FlowIntegration) commands() []server.Command {
	return []server.Command{
		{
			Name:    CommandSendTransaction,
			Handler: i.sendTransaction,
		},
		{
			Name:    CommandExecuteScript,
			Handler: i.executeScript,
		},
		{
			Name:    CommandDeployContract,
			Handler: i.deployContract,
		},
		{
			Name:    CommandSwitchActiveAccount,
			Handler: i.switchActiveAccount,
		},
		{
			Name:    CommandCreateAccount,
			Handler: i.createAccount,
		},
		{
			Name:    CommandCreateDefaultAccounts,
			Handler: i.createDefaultAccounts,
		},
	}
}

// sendTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (i *FlowIntegration) sendTransaction(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageArguments, err)
	}

	var uri string
	err = json.Unmarshal(args[0], &uri)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err),
		)
	}

	location, err := url.Parse(uri)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid path argument: %#+v", uri),
		)
	}

	var argsJSON string
	err = json.Unmarshal(args[1], &argsJSON)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid arguments: %#+v: %w", args[1], err),
		)
	}

	var signerList []any
	err = json.Unmarshal(args[2], &argsJSON)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid signer list: %#+v: %w", args[2], err),
		)
	}

	signers := make([]flow.Address, len(signerList))
	for i, v := range signerList {
		signers[i] = flow.HexToAddress(v.(string))
	}

	txArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON arguments")
	}

	txResult, err := i.flowClient.SendTransaction(signers, location, txArgs)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageTransactionError, err)
	}

	showMessage(conn, fmt.Sprintf("Transaction status: %s", txResult.Status.String()))
	return nil, err
}

// executeScript handles executing a script defined in the source document.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (i *FlowIntegration) executeScript(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageArguments, err)
	}

	var uri string
	err = json.Unmarshal(args[0], &uri)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err),
		)
	}

	path, pathError := url.Parse(uri)
	if pathError != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid path argument: %#+v", uri),
		)
	}

	var argsJSON string
	err = json.Unmarshal(args[1], &argsJSON)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid arguments: %#+v: %w", args[1], err),
		)
	}

	scriptArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageScriptExecution,
			fmt.Errorf("arguments error: %w", err),
		)
	}

	scriptResult, err := i.flowClient.ExecuteScript(path, scriptArgs)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageScriptExecution, err)
	}

	showMessage(conn, fmt.Sprintf("Result: %s", scriptResult.String()))
	return nil, nil
}

// switchActiveAccount sets the account that is currently active and could be used
// when submitting transactions.
//
// There should be 2 arguments:
//	 * name of the new active account
func (i *FlowIntegration) switchActiveAccount(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageArguments, err)
	}

	var name string
	err = json.Unmarshal(args[0], &name)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid name argument: %#+v: %w", args[0], err),
		)
	}

	err = i.flowClient.SetActiveClientAccount(name)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// createAccount creates a new account and returns its address.
func (i *FlowIntegration) createAccount(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	account, err := i.flowClient.CreateAccount()
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	return account, nil
}

// deployContract deploys the contract to the configured account with the code of the given
// file.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the name of the contract
func (i *FlowIntegration) deployContract(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return flow.Address{}, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	var uri string
	err = json.Unmarshal(args[0], &uri)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid URI argument: %#+v: %w", args[0], err),
		)
	}

	location, pathError := url.Parse(uri)
	if pathError != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid path argument: %#+v", uri),
		)
	}

	var name string
	err = json.Unmarshal(args[1], &name)
	if err != nil {
		return nil, errorWithMessage(
			conn, ErrorMessageArguments,
			fmt.Errorf("invalid name argument: %#+v: %w", args[1], err),
		)
	}

	var rawAddress string
	err = json.Unmarshal(args[2], &rawAddress)
	if err != nil {
		return nil, errorWithMessage(
			conn, ErrorMessageArguments,
			fmt.Errorf("invalid address argument: %#+v: %w", args[2], err),
		)
	}
	address := flow.HexToAddress(rawAddress)

	showMessage(conn, fmt.Sprintf("Deploying contract %s to account %s", name, rawAddress))

	_, deployError := i.flowClient.DeployContract(address, name, location)
	if deployError != nil {
		return nil, errorWithMessage(conn, ErrorMessageDeploy, deployError)
	}

	showMessage(conn, fmt.Sprintf("Status: contract %s has been deployed to %s", name, rawAddress))
	return nil, err
}

// showMessage sends a "show message" notification
func showMessage(conn protocol.Conn, message string) {
	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: message,
	})
}

// showMessage sends a "show error" notification
func showError(conn protocol.Conn, errorMessage string) {
	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Error,
		Message: errorMessage,
	})
}

func errorWithMessage(conn protocol.Conn, prefix string, err error) error {
	errorMessage := fmt.Sprintf("%s: %#+v", prefix, err)
	showError(conn, errorMessage)
	return fmt.Errorf("%s", errorMessage)
}
