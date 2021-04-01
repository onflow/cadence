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
	"context"
	"errors"
	"fmt"
	"github.com/onflow/flow-cli/pkg/flowcli"
	"github.com/onflow/flow-go-sdk/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"

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

// Cadence Templates
const deployContractTemplate = `
transaction(name: String, code: [UInt8]) {
  prepare(signer: AuthAccount) {
    if signer.contracts.get(name: name) == nil {
      signer.contracts.add(name: name, code: code)
    } else {
      signer.contracts.update__experimental(name: name, code: code)
    }
  }
}
`

const contractAccountManager = `
pub contract AccountManager{
    pub event AliasAdded(_ name: String, _ address: Address)

    pub let accountsByName: {String: Address}
    pub let accountsByAddress: {Address: String}
    pub let names: [String]
    
    init(){
        self.accountsByName = {}
		self.accountsByAddress = {}
        self.names = [
            "Alice", "Bob", "Charlie",
            "Dave", "Eve", "Faythe",
            "Grace", "Heidi", "Ivan",
            "Judy", "Michael", "Niaj",
            "Olivia", "Oscar", "Peggy",
            "Rupert", "Sybil", "Ted",
            "Victor", "Walter"
        ]
    }

    pub fun addAccount(_ address: Address){
        let name = self.names[self.accountsByName.keys.length]
        self.accountsByName[name] = address
        self.accountsByAddress[address] = name
        emit AliasAdded(name, address)
    }

    pub fun getAddress(_ name: String): Address?{
        return self.accountsByName[name]
    }

    pub fun getName(_ address: Address): String?{
        return self.accountsByAddress[address]
    }

    pub fun getAccounts():[String]{
        let accounts: [String] = []
        for name in self.accountsByName.keys {
            let address = self.accountsByName[name]!
            let account = name.concat(":")
                            .concat(address.toString())
            accounts.append(account)
        }
        return accounts
    }
}
`

const transactionAddAccount = `
import AccountManager from 0xSERVICE_ACCOUNT_ADDRESS

transaction(address: Address){
  prepare(signer: AuthAccount) {
    AccountManager.addAccount(address)
	log("Account added to ledger")
  }
}
`

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
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return nil, err
	}

	serviceAddress := serviceAccount.Address()
	accountManagerContract := makeManagerCode(contractAccountManager, serviceAddress.String())
	deployTx := deployContractTransaction(serviceAddress, "AccountManager", accountManagerContract)

	_, err = i.sendTransactionHelper(conn, serviceAddress, deployTx)

	if err != nil {
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return nil, err
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
		errorMessage := fmt.Sprintf("language server error: %#+v", err)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
	}

	_, txResult, txSendError := i.sharedServices.Transactions.SendForAddress(path.Path, signers[0], privateKey, []string{}, argsJSON)

	if txSendError != nil {
		errorMessage := fmt.Sprintf("there was an error with your transaction: %#+v", txSendError)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
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
	scriptResult, scriptError := i.sharedServices.Scripts.Execute(path.Path, []string{}, argsJSON)

	if scriptError != nil {
		return nil, fmt.Errorf("execution error: %#+v", scriptError)
	}

	displayResult := fmt.Sprintf("Result: %s", scriptResult.String())

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: displayResult,
	})
	return nil, nil
}

// changeEmulatorState sets current state of the emulator as reported by LSP
// used to update codelenses with proper title
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

// switchActiveAccount sets the account that is currently active and should be
// used when submitting transactions.
//
// There should be exactly 1 argument:
//   * the address of the new active account
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
	clientAccount, err := i.storeAccountHelper(conn, address)

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("New account %s(0x%s) created", clientAccount.Name, address.String()),
	})

	return clientAccount, err
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
		errorMessage := fmt.Sprintf("language server error: %#+v", err)
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: errorMessage,
		})
		return nil, fmt.Errorf("%s", errorMessage)
	}

	// TODO: add check if the contract exist on specified address
	update, err := i.isContractDeployed(to, name)
	account, deployError := i.sharedServices.Accounts.AddContractForAddress(to, privateKey,name, path.Path, update)

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
		Message: fmt.Sprintf("Status: contract %s has been deployed to %s(%s)", name, to, account.Address.String()),
	})

	return nil, err
}

func bytesToCadenceArray(b []byte) cadence.Array {
	values := make([]cadence.Value, len(b))

	for i, v := range b {
		values[i] = cadence.NewUInt8(v)
	}

	return cadence.NewArray(values)
}

// deployContractTransaction generates a transaction that deploys the given contract to an account.
func deployContractTransaction(address flow.Address, name string, code []byte) *flow.Transaction {
	cadenceName := cadence.NewString(name)
	cadenceCode := bytesToCadenceArray(code)

	return flow.NewTransaction().
		SetScript([]byte(deployContractTemplate)).
		AddRawArgument(jsoncdc.MustEncode(cadenceName)).
		AddRawArgument(jsoncdc.MustEncode(cadenceCode)).
		AddAuthorizer(address)
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

// getServiceAddress returns address for service account
func (i *FlowIntegration) getServiceAddress() (flow.Address, error) {
	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return flow.Address{}, err
	}

	return serviceAccount.Address(), nil
}

// getServiceKey returns the service account key and signer
func (i *FlowIntegration) getServiceKey() (*flow.AccountKey, crypto.Signer, error) {

	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		return nil, nil, err
	}
	address := serviceAccount.Address()

	rawKey := serviceAccount.DefaultKey().ToConfig().Context["privateKey"]

	privateKey := AccountPrivateKey{
		SigAlgo:  crypto.StringToSignatureAlgorithm(serviceAccount.DefaultKey().SigAlgo().String()),
		HashAlgo: crypto.StringToHashAlgorithm(serviceAccount.DefaultKey().HashAlgo().String()),
	}

	privateKey.PrivateKey, err = crypto.DecodePrivateKeyHex(privateKey.SigAlgo, rawKey)
	if err != nil {
		return nil, nil, err
	}

	account, err := i.flowClient.GetAccount(context.Background(), address)
	if err != nil {
		return nil, nil, err
	}

	if len(account.Keys) == 0 {
		return nil, nil, fmt.Errorf(
			"cannot sign transaction: account %s has no keys",
			address.Hex(),
		)
	}

	accountKey := account.Keys[0]
	signer := crypto.NewNaiveSigner(privateKey.PrivateKey, privateKey.HashAlgo)

	return accountKey, signer, nil
}

// sendTransactionHelper sends a transaction with the given script, from the
// currently active account. Returns the hash of the transaction if it is
// successfully submitted.
//
// If an error occurs, attempts to show an appropriate message (either via logs
// or UI popups in the client).
func (i *FlowIntegration) sendTransactionHelper(
	conn protocol.Conn,
	address flow.Address,
	tx *flow.Transaction,
) (flow.Identifier, error) {
	accountKey, signer, err := i.getServiceKey()
	if err != nil {
		return flow.EmptyID, err
	}

	tx.SetProposalKey(address, accountKey.Index, accountKey.SequenceNumber)
	tx.SetPayer(address)

	block, err := i.flowClient.GetLatestBlock(context.Background(), true)
	if err != nil {
		return flow.EmptyID, err
	}

	tx.SetReferenceBlockID(block.ID)

	err = tx.SignEnvelope(address, accountKey.Index, signer)
	if err != nil {
		return flow.EmptyID, err
	}

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("submitting transaction %s", tx.ID().Hex()),
	})

	err = i.flowClient.SendTransaction(context.Background(), *tx)
	if err != nil {
		grpcErr, ok := status.FromError(err)
		if ok {
			if grpcErr.Code() == codes.Unavailable {
				// The emulator server isn't running
				conn.ShowMessage(&protocol.ShowMessageParams{
					Type:    protocol.Warning,
					Message: "The emulator server is unavailable. Please start the emulator (`cadence.runEmulator`) first.",
				})
				return flow.EmptyID, err
			} else if grpcErr.Code() == codes.InvalidArgument {
				// The request was invalid
				conn.ShowMessage(&protocol.ShowMessageParams{
					Type:    protocol.Warning,
					Message: "The transaction could not be submitted.",
				})
				conn.LogMessage(&protocol.LogMessageParams{
					Type:    protocol.Warning,
					Message: fmt.Sprintf("Failed to submit transaction: %s", grpcErr.Message()),
				})
				return flow.EmptyID, err
			}
		} else {
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Warning,
				Message: fmt.Sprintf("Failed to submit transaction: %s", err.Error()),
			})
		}

		return flow.EmptyID, err
	}

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Submitted transaction id=%s", tx.ID().Hex()),
	})

	return tx.ID(), nil
}

// createAccountHelper creates a new account and returns its address.
func (i *FlowIntegration) createAccountHelper(conn protocol.Conn) (address flow.Address, err error) {
	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return flow.Address{}, err
	}

	signer := serviceAccount.Name()

	defaultKey := serviceAccount.DefaultKey()
	serviceAccountPrivateKey, _ := i.getServicePrivateKey()
	cryptoKey, err := crypto.DecodePrivateKeyHex(defaultKey.SigAlgo(), serviceAccountPrivateKey)
	keys := []string{cryptoKey.PublicKey().String()}

	signatureAlgorithm := defaultKey.SigAlgo().String()
	hashAlgorithm := defaultKey.HashAlgo().String()
	var contracts []string

	newAccount, err := i.sharedServices.Accounts.Create(signer, keys, signatureAlgorithm, hashAlgorithm, contracts)
	if err != nil {
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return flow.Address{}, err
	}

	return newAccount.Address, nil
}

func (i *FlowIntegration) storeAccountHelper(conn protocol.Conn, address flow.Address) (newAccount ClientAccount, err error) {

	serviceAccount, err := i.project.EmulatorServiceAccount()
	if err != nil {
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return
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
		conn.ShowMessage(&protocol.ShowMessageParams{
			Type:    protocol.Error,
			Message: err.Error(),
		})
		return
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
	account, err := i.gateway.GetAccount(flow.HexToAddress(address))

	if err != nil {
		return false, err
	}

	return account.Contracts[name] != nil, nil
}

func parseFileFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	return filepath.Base(u.Path)
}
