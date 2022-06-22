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
	"io/ioutil"
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
	CommandCreateDefaultAccounts = "cadence.server.flow.createDefaultAccounts"
	CommandSwitchActiveAccount   = "cadence.server.flow.switchActiveAccount"
	CommandInitAccountManager    = "cadence.server.flow.initAccountManager"

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

// ClientAccount will be used to
// * store active account on language server to sign transactions and deploy contracts
// * return newly created accounts to client
type ClientAccount struct {
	Name    string       `json:"name"`
	Address flow.Address `json:"address"`
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

	// Send transaction via shared library
	code, err := ioutil.ReadFile(path.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction file: %s", path.Path)
	}

	txArgs, err := flowkit.ParseArgumentsJSON(argsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON arguments")
	}

	_, txResult, err := i.flowClient.SendTransaction(signers, code, txArgs)
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

	code, err := i.state.ReadFile(path.Path)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageScriptExecution,
			fmt.Errorf("file load error: %w", err),
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

	// Execute script via shared library
	scriptResult, err := i.flowClient.ExecuteScript(code, scriptArgs, "", "")
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
//	 * name of the new active acount
//   * address of the new active account
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

	var addressHex string
	err = json.Unmarshal(args[1], &addressHex)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid address argument: %#+v: %w", args[1], err),
		)
	}
	address := flow.HexToAddress(addressHex)

	i.activeAccount = ClientAccount{
		Name:    name,
		Address: address,
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

// createDefaultAccounts creates a set of default accounts and returns their addresses.
//
// There should be exactly 1 argument:
// * number of accounts to be created
func (i *FlowIntegration) createDefaultAccounts(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageArguments, err)
	}

	// Note that extension will send this value as float64 and not int
	var n float64
	err = json.Unmarshal(args[0], &n)
	if err != nil {
		return nil, errorWithMessage(
			conn,
			ErrorMessageArguments,
			fmt.Errorf("invalid count argument: %#+v: %w", args[0], err),
		)
	}
	count := int(n)

	showMessage(conn, fmt.Sprintf("Creating %d default accounts", count))

	accounts := make([]*flow.Account, count+1)
	for index := 1; index < count+1; index++ {
		account, err := i.flowClient.CreateAccount()
		if err != nil {
			return nil, errorWithMessage(conn, ErrorMessageAccountCreate, err)
		}
		accounts[index] = account //account.(ClientAccount) // todo see why return
	}

	return accounts, nil
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

	path, pathError := url.Parse(uri)
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

	code, err := i.state.ReadFile(path.Path)
	if err != nil {
		return nil, errorWithMessage(conn, fmt.Sprintf("failed to load contract code: %s", path.Path), err)
	}

	_, deployError := i.flowClient.DeployContract(address, name, code)
	if deployError != nil {
		return nil, errorWithMessage(conn, ErrorMessageDeploy, deployError)
	}

	showMessage(conn, fmt.Sprintf("Status: contract %s has been deployed to %s", name, to))
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
