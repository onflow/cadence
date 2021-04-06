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
	"errors"
	"fmt"
	"github.com/onflow/flow-cli/pkg/flowcli"
	"github.com/onflow/flow-go-sdk/crypto"
	"net/url"
	"strings"

	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"
)

const (
	CommandSendTransaction       = "cadence.server.flow.sendTransaction"
	CommandExecuteScript         = "cadence.server.flow.executeScript"
	CommandDeployContract        = "cadence.server.flow.deployContract"
	CommandCreateAccount         = "cadence.server.flow.createAccount"
	CommandCreateDefaultAccounts = "cadence.server.flow.createDefaultAccounts"
	CommandSwitchActiveAccount   = "cadence.server.flow.switchActiveAccount"
	CommandChangeEmulatorState   = "cadence.server.flow.changeEmulatorState"
	CommandInitAccountManager    = "cadence.server.flow.initAccountManager"

	ClientStartEmulator = "cadence.runEmulator"

	ErrorMessageServiceAccount =			"service account error"
	ErrorMessageTransactionError =			"transaction error"
	ErrorMessageServiceAccountKey = 		"service account private key error"
	ErrorMessageAccountCreate = 			"create account error"
	ErrorMessageAccountStore = 				"store account error"
	ErrorMessagePrivateKeyDecoder = 		"private key decoder error"
	ErrorMessageDeploy	=					"deployment error"
	ErrorScriptExecution =					"script error"
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
			Name:    CommandChangeEmulatorState,
			Handler: i.changeEmulatorState,
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
		{
			Name:    CommandInitAccountManager,
			Handler: i.initAccountManager,
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

// makeManagerCode replaces service account placeholder with actual address and returns byte array
//
// There should be exactly 2 arguments:
//   * Cadence script template
//   * service account address represented as string sans 0x prefix
func makeManagerCode(script string, serviceAddress string) []byte {
	injected := strings.ReplaceAll(script, "SERVICE_ACCOUNT_ADDRESS", serviceAddress)
	return []byte(injected)
}

// initAccountManager initializes Account manager on service account
//
// No arguments are expected
func (i *FlowIntegration) initAccountManager(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	serviceAddress := serviceAccount.Address().String()

	privateKey, err := i.getServicePrivateKey()
	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageServiceAccountKey,err)
	}


	name := "AccountManager"
	code := []byte(contractAccountManager)
	update, err := i.isContractDeployed(serviceAddress, name)
	if err != nil {
		return nil, throwErrorWithMessage(conn, fmt.Sprintf("can't read contract from account %s",serviceAddress),err)
	}

	_, deployError := i.sharedServices.Accounts.AddContractForAddressWithCode(serviceAddress, privateKey, name, code, update)

	if deployError != nil {
		return nil, throwErrorWithMessage(conn,ErrorMessageDeploy ,err)
	}

	return nil, err
}

// sendTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (i *FlowIntegration) sendTransaction(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return nil, err
	}

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	path, pathError := url.Parse(uri)
	if pathError != nil {
		return nil, fmt.Errorf("invalid URI arguments: %#+v", uri)
	}

	argsJSON, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid arguments: %#+v", args[1])
	}

	signerList := args[2].([]interface{})
	signers := make([]string, len(signerList))
	for i, v := range signerList {
		signers[i] = v.(string)
	}

	// Send transaction via shared library
	privateKey, err := i.getServicePrivateKey()
	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageServiceAccountKey, err)
	}

	_, txResult, err := i.sharedServices.Transactions.SendForAddress(path.Path, signers[0], privateKey, []string{}, argsJSON)

	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageTransactionError, err)
	}

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Transaction status: %s", txResult.Status.String()),
	})

	return nil, err
}

// executeScript handles executing a script defined in the source document.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (i *FlowIntegration) executeScript(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	path, pathError := url.Parse(uri)
	if pathError != nil {
		return nil, fmt.Errorf("invalid URI arguments: %#+v", uri)
	}

	argsJSON, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid arguments: %#+v", args[1])
	}

	// Execute script via shared library
	scriptResult, err := i.sharedServices.Scripts.Execute(path.Path, []string{}, argsJSON)

	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorScriptExecution, err)
	}

	displayResult := fmt.Sprintf("Result: %s", scriptResult.String())

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: displayResult,
	})
	return nil, nil
}

// changeEmulatorState sets current state of the emulator as reported by LSP
// used to update Code Lenses with proper title
//
// There should be exactly 1 argument:
// * current state of the emulator represented as byte
func (i *FlowIntegration) changeEmulatorState(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	emulatorState, ok := args[0].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid emulator state argument: %#+v", args[0])
	}

	i.emulatorState = EmulatorState(emulatorState)
	return nil, nil
}

// switchActiveAccount sets the account that is currently active and could be used
// when submitting transactions.
//
// There should be 2 arguments:
//	 * name of the new active acount
//   * address of the new active account
func (i *FlowIntegration) switchActiveAccount(_ protocol.Conn, args ...interface{}) (interface{}, error) {
	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	name, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid name argument")
	}

	addressHex, ok := args[1].(string)
	if !ok {
		return nil, errors.New("invalid address argument")
	}
	address := flow.HexToAddress(addressHex)

	i.activeAccount = ClientAccount{
		Name:    name,
		Address: address,
	}
	return nil, nil
}

// createAccount creates a new account and returns its address.
func (i *FlowIntegration) createAccount(conn protocol.Conn, args ...interface{}) (interface{}, error) {

	address, err := i.createAccountHelper(conn)
	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	clientAccount, err := i.storeAccountHelper(conn, address)
	if err != nil {
		return nil, throwErrorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("New account %s(0x%s) created", clientAccount.Name, address.String()),
	})

	return clientAccount, nil
}

// createDefaultAccounts creates a set of default accounts and returns their addresses.
//
// This command will wait until the emulator server is started before submitting any transactions.
func (i *FlowIntegration) createDefaultAccounts(conn protocol.Conn, args ...interface{}) (interface{}, error) {

	err := server.CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	n, ok := args[0].(float64)
	if !ok {
		return nil, errors.New("invalid count argument")
	}

	count := int(n)

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Creating %d default accounts", count),
	})

	accounts := make([]ClientAccount, count)

	for index := 0; index < count; index++ {
		account, err := i.createAccount(conn)
		if err != nil {
			return nil, err
		}
		accounts[index] = account.(ClientAccount)
	}

	return accounts, nil
}

// deployContract deploys the contract to the configured account with the code of the given
// file.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the name of the declaration
//
func (i *FlowIntegration) deployContract(conn protocol.Conn, args ...interface{}) (interface{}, error) {

	err := server.CheckCommandArgumentCount(args, 3)
	if err != nil {
		return nil, err
	}

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	path, pathError := url.Parse(uri)
	if pathError != nil {
		return nil, fmt.Errorf("invalid URI arguments: %#+v", uri)
	}

	name, ok := args[1].(string)
	if !ok {
		return nil, errors.New("invalid name argument")
	}

	to := args[2].(string)
	if !ok {
		return nil, errors.New("invalid address argument")
	}

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Deploying contract %s to account %s", name, to),
	})

	// Send transaction via shared library
	privateKey, err := i.getServicePrivateKey()
	if err != nil {
		errorMessage := fmt.Sprintf("Error with Servie Account private key : %#+v", err)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
	}

	update, err := i.isContractDeployed(to, name)
	if err != nil {
		errorMessage := fmt.Sprintf("can't read contract from account %s: %#+v",to, err)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
	}

	_, deployError := i.sharedServices.Accounts.AddContractForAddress(to, privateKey,name, path.Path, update)

	if deployError != nil {
		errorMessage := fmt.Sprintf("error during deployment: %#+v", deployError)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
	}

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Status: contract %s has been deployed to %s", name, to),
	})

	return nil, err
}


// getServicePrivateKey returns private key for service account
func (i *FlowIntegration) getServicePrivateKey() (string, error) {
	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return "", err
	}

	rawKey := serviceAccount.DefaultKey().ToConfig().Context["privateKey"]
	return rawKey,nil
}

// createAccountHelper creates a new account and returns its address.
func (i *FlowIntegration) createAccountHelper(conn protocol.Conn) (address flow.Address, err error) {
	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return flow.Address{}, throwErrorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	signer := serviceAccount.Name()

	defaultKey := serviceAccount.DefaultKey()
	serviceAccountPrivateKey, err := i.getServicePrivateKey()
	if err != nil{
		return flow.Address{}, throwErrorWithMessage(conn, ErrorMessageServiceAccountKey, err)
	}

	cryptoKey, err := crypto.DecodePrivateKeyHex(defaultKey.SigAlgo(), serviceAccountPrivateKey)
	if err != nil{
		return flow.Address{}, throwErrorWithMessage(conn, ErrorMessagePrivateKeyDecoder, err)
	}

	keys := []string{cryptoKey.PublicKey().String()}

	signatureAlgorithm := defaultKey.SigAlgo().String()
	hashAlgorithm := defaultKey.HashAlgo().String()
	var contracts []string

	newAccount, err := i.sharedServices.Accounts.Create(signer, keys, signatureAlgorithm, hashAlgorithm, contracts)
	if err != nil {
		return flow.Address{}, throwErrorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	return newAccount.Address, nil
}

// storeAccountHelper sends transaction to store account on chain
func (i *FlowIntegration) storeAccountHelper(conn protocol.Conn, address flow.Address) (newAccount ClientAccount, err error) {

	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return ClientAccount{}, throwErrorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	defaultKey := serviceAccount.DefaultKey()
	serviceAddress := serviceAccount.Address().String()
	privateKey := defaultKey.ToConfig().Context["privateKey"]

	// Store new account
	code := makeManagerCode(transactionAddAccount, serviceAddress)
	accountAddress := fmt.Sprintf("Address:0x%v", address)
	txArgs := []string{accountAddress}

	_, txResult, err := i.sharedServices.Transactions.SendForAddressWithCode(code, serviceAddress, privateKey, txArgs, "")
	if err != nil {
		return ClientAccount{}, throwErrorWithMessage(conn, ErrorMessageAccountStore, err)
	}

	events := flowcli.EventsFromTransaction(txResult)
	name := strings.ReplaceAll(events[0].Values["name"], `"`, "")

	newAccount = ClientAccount{
		Name:    name,
		Address: address,
	}

	return
}

func (i *FlowIntegration) isContractDeployed(address string, name string) (bool,error) {
	account, err := i.sharedServices.Accounts.Get(address)

	if err != nil {
		return false, err
	}

	return account.Contracts[name] != nil, nil
}

func throwErrorWithMessage(conn protocol.Conn, prefix string, err error) error {
	errorMessage := fmt.Sprintf("%s: %#+v", prefix, err)
	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Error,
		Message: errorMessage,
	})
	return fmt.Errorf("%s", errorMessage)
}