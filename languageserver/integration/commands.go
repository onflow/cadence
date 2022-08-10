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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/onflow/cadence"

	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"

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

const maxGasLimit uint64 = 9999

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
func (i *FlowIntegration) initAccountManager(conn protocol.Conn, args ...json.RawMessage) (interface{}, error) {
	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	serviceAddress := serviceAccount.Address().String()

	// Check if emulator is up
	err = i.waitForNetwork()
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageEmulator, err)
	}

	name := "AccountManager"
	code := makeManagerCode(fmt.Sprintf(contractAccountManager, serviceAccountName), serviceAddress)
	update, err := i.isContractDeployed(serviceAccount.Address(), name)
	if err != nil {
		return nil, errorWithMessage(conn, fmt.Sprintf("can't read contract from account %s", serviceAddress), err)
	}

	_, deployError := i.sharedServices.Accounts.AddContract(
		serviceAccount,
		name,
		code,
		update,
		nil,
	)

	if deployError != nil {
		return nil, errorWithMessage(conn, ErrorMessageDeploy, err)
	}

	return nil, err
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

	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get service account, err: %w", err)
	}

	serviceAddress := serviceAccount.Address()
	keyIndex := serviceAccount.Key().Index()

	// We need to check if service account among authorizers
	hasServiceAccount := false

	signerAccounts := make([]flowkit.Account, len(signers))
	authorizers := make([]flow.Address, len(signers))
	for i, address := range signers {

		signer := flowkit.Account{}
		signer.SetAddress(address)
		signer.SetKey(serviceAccount.Key())

		signerAccounts[i] = signer
		authorizers[i] = address

		if address == serviceAddress {
			hasServiceAccount = true
		}
	}

	// If serviceAccount is not in signers list, we will add it to handle payer role properly
	if !hasServiceAccount {
		signerAccounts = append(signerAccounts, *serviceAccount)
	}

	tx, err := i.sharedServices.Transactions.Build(
		serviceAddress,
		authorizers,
		serviceAddress,
		keyIndex,
		code,
		"",
		maxGasLimit,
		txArgs,
		"",
		true,
	)

	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageTransactionError, err)
	}

	for _, signer := range signerAccounts {
		err = tx.SetSigner(&signer)
		if err != nil {
			return nil, err
		}

		tx, err = tx.Sign()
		if err != nil {
			return nil, err
		}
	}

	// even though .Encode returns []byte, without this conversion there is an error:
	// transaction error: &errors.errorString{s:"failed to decode partial transaction...
	// ...encoding/hex: invalid byte: U+00F9 'ù'"
	txBytes := []byte(fmt.Sprintf("%x", tx.FlowTransaction().Encode()))
	_, txResult, err := i.sharedServices.Transactions.SendSigned(txBytes, true)

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
	scriptResult, err := i.sharedServices.Scripts.Execute(code, scriptArgs, "", "")
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
	address, err := i.createAccountHelper(conn)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	clientAccount, err := i.storeAccountHelper(conn, address)
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	return clientAccount, nil
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

	// Check if emulator is up
	err = i.waitForNetwork()
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageEmulator, err)
	}

	accounts := make([]ClientAccount, count+1)

	// Get service account
	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return nil, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	// Add service account to a list of accounts
	accounts[0] = ClientAccount{
		Name:    serviceAccountName,
		Address: serviceAccount.Address(),
	}

	for index := 1; index < count+1; index++ {
		account, err := i.createAccount(conn)
		if err != nil {
			return nil, errorWithMessage(conn, ErrorMessageAccountCreate, err)
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

	var to string
	err = json.Unmarshal(args[2], &to)
	if err != nil {
		return nil, errorWithMessage(
			conn, ErrorMessageArguments,
			fmt.Errorf("invalid address argument: %#+v: %w", args[2], err),
		)
	}

	showMessage(conn, fmt.Sprintf("Deploying contract %s to account %s", name, to))

	// Send transaction via shared library
	signer, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return nil, errorWithMessage(conn, fmt.Sprintf("service account error"), err)
	}

	update, err := i.isContractDeployed(flow.HexToAddress(to), name)
	if err != nil {
		return nil, errorWithMessage(conn, fmt.Sprintf("can't read contract from account %s", to), err)
	}

	code, err := i.state.ReadFile(path.Path)
	if err != nil {
		return nil, errorWithMessage(conn, fmt.Sprintf("failed to load contract code: %s", path.Path), err)
	}

	_, deployError := i.sharedServices.Accounts.AddContract(signer, name, code, update, nil)
	if deployError != nil {
		return nil, errorWithMessage(conn, ErrorMessageDeploy, deployError)
	}

	showMessage(conn, fmt.Sprintf("Status: contract %s has been deployed to %s", name, to))
	return nil, err
}

// getServicePrivateKey returns private key for service account
func (i *FlowIntegration) getServicePrivateKey() (*crypto.PrivateKey, error) {
	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}

	return serviceAccount.Key().PrivateKey()
}

// createAccountHelper creates a new account and returns its address.
func (i *FlowIntegration) createAccountHelper(conn protocol.Conn) (address flow.Address, err error) {
	signer, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return flow.Address{}, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	pkey, err := signer.Key().PrivateKey()
	if err != nil {
		return flow.Address{}, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	keys := []crypto.PublicKey{(*pkey).PublicKey()}
	weights := []int{flow.AccountKeyWeightThreshold}

	newAccount, err := i.sharedServices.Accounts.Create(
		signer,
		keys,
		weights,
		[]crypto.SignatureAlgorithm{crypto.ECDSA_P256},
		[]crypto.HashAlgorithm{crypto.SHA3_256},
		nil,
	)
	if err != nil {
		return flow.Address{}, errorWithMessage(conn, ErrorMessageAccountCreate, err)
	}

	return newAccount.Address, nil
}

// storeAccountHelper sends transaction to store account on chain
func (i *FlowIntegration) storeAccountHelper(conn protocol.Conn, address flow.Address) (newAccount ClientAccount, err error) {

	serviceAccount, err := i.state.EmulatorServiceAccount()
	if err != nil {
		return ClientAccount{}, errorWithMessage(conn, ErrorMessageServiceAccount, err)
	}

	serviceAddress := serviceAccount.Address().String()

	// Store new account
	code := makeManagerCode(transactionAddAccount, serviceAddress)
	txArgs := []cadence.Value{
		cadence.NewAddress(address),
	}

	const gasLimit uint64 = 1000

	_, txResult, err := i.sharedServices.Transactions.Send(
		serviceAccount,
		code,
		"",
		gasLimit,
		txArgs,
		"",
	)
	if err != nil {
		return ClientAccount{}, errorWithMessage(conn, ErrorMessageAccountStore, err)
	}

	events := flowkit.EventsFromTransaction(txResult)
	name := strings.ReplaceAll(events[0].Values["name"].String(), `"`, "")

	newAccount = ClientAccount{
		Name:    name,
		Address: address,
	}

	return
}

func (i *FlowIntegration) isContractDeployed(address flow.Address, name string) (bool, error) {
	account, err := i.sharedServices.Accounts.Get(address)

	if err != nil {
		return false, err
	}

	return account.Contracts[name] != nil, nil
}
func (i *FlowIntegration) waitForNetwork() error {
	// Ping the emulator server for 30 seconds until it is available
	timer := time.NewTimer(30 * time.Second)
RetryLoop:
	for {
		select {
		case <-timer.C:
			return errors.New("emulator server timed out")
		default:
			_, err := i.sharedServices.Status.Ping("emulator")
			if err == nil {
				break RetryLoop
			}
		}
	}
	return nil
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
