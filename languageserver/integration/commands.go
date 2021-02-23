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
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/onflow/flow-go-sdk/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/templates"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"
)

const (
	CommandSubmitTransaction     = "cadence.server.flow.submitTransaction"
	CommandExecuteScript         = "cadence.server.flow.executeScript"
	CommandDeployContract        = "cadence.server.flow.deployContract"
	CommandCreateAccount         = "cadence.server.flow.createAccount"
	CommandCreateDefaultAccounts = "cadence.server.flow.createDefaultAccounts"
	CommandSwitchActiveAccount   = "cadence.server.flow.switchActiveAccount"

	ClientExecuteScript = "cadence.executeScript"
	ClientSendTransaction = "cadence.sendTransaction"
	ClientDeployContract = "cadence.deployContract"
)

func (i *FlowIntegration) commands() []server.Command {
	return []server.Command{
		{
			Name:    CommandSubmitTransaction,
			Handler: i.submitTransaction,
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

// submitTransaction handles submitting a transaction defined in the
// source document in VS Code.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the arguments, encoded as JSON-CDC
func (i *FlowIntegration) submitTransaction(conn protocol.Conn, args ...interface{}) (interface{}, error) {

	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	_, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid transaction arguments: %#+v", args[1])
	}

	rawTransactionArguments, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid script arguments: %#+v", args[1])
	}

	var transactionArguments []string
	for i, rawTransactionArgument := range rawTransactionArguments {
		stringArgument, ok := rawTransactionArgument.(string)
		if !ok {
			return nil, fmt.Errorf("invalid transaction argument at index %d: %#+v", i, rawTransactionArgument)
		}

		transactionArguments = append(transactionArguments, strings.TrimSuffix(stringArgument, "\n"))
	}

	// Pass arguments back to extension
	codeArguments := fmt.Sprintf("%s", strings.Join(transactionArguments, ","))
	err = conn.Notify(ClientSendTransaction, codeArguments)

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

	_, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

/*	doc, ok := i.server.GetDocument(protocol.DocumentUri(uri))
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}*/

	rawScriptArguments, ok := args[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid script arguments: %#+v", args[1])
	}

	var scriptArguments []string
	for i, rawScriptArgument := range rawScriptArguments {
		stringArgument, ok := rawScriptArgument.(string)
		if !ok {
			return nil, fmt.Errorf("invalid script argument at index %d: %#+v", i, rawScriptArgument)
		}

		scriptArguments = append(scriptArguments, strings.TrimSuffix(stringArgument, "\n"))
	}

	// Pass arguments back to extension
	codeArguments := fmt.Sprintf("%s", strings.Join(scriptArguments, ","))
	err = conn.Notify(ClientExecuteScript, codeArguments)

	return nil, err
}

// switchActiveAccount sets the account that is currently active and should be
// used when submitting transactions.
//
// There should be exactly 1 argument:
//   * the address of the new active account
func (i *FlowIntegration) switchActiveAccount(_ protocol.Conn, args ...interface{}) (interface{}, error) {

	err := server.CheckCommandArgumentCount(args, 1)
	if err != nil {
		return nil, err
	}

	addrHex, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid argument")
	}
	addr := flow.HexToAddress(addrHex)

	_, ok = i.accounts[addr]
	if !ok {
		return nil, errors.New("cannot set active account that does not exist")
	}

	i.activeAddress = addr
	return nil, nil
}

// createAccount creates a new account and returns its address.
func (i *FlowIntegration) createAccount(conn protocol.Conn, args ...interface{}) (interface{}, error) {

	err := server.CheckCommandArgumentCount(args, 0)
	if err != nil {
		return nil, err
	}

	addr, err := i.createAccountHelper(conn)
	if err != nil {
		return nil, err
	}

	return addr, nil
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

	// Ping the emulator server for 30 seconds until it is available
	timer := time.NewTimer(30 * time.Second)
RetryLoop:
	for {
		select {
		case <-timer.C:
			return nil, errors.New("emulator server timed out")
		default:
			err := i.flowClient.Ping(context.Background())
			if err == nil {
				break RetryLoop
			}
		}
	}

	accounts := make([]flow.Address, count)

	for index := 0; index < count; index++ {
		addr, err := i.createAccountHelper(conn)
		if err != nil {
			return nil, err
		}
		accounts[index] = addr
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

	err := server.CheckCommandArgumentCount(args, 2)
	if err != nil {
		return nil, err
	}

	uri, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI argument: %#+v", args[0])
	}

	doc, ok := i.server.GetDocument(protocol.DocumentUri(uri))
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	file := parseFileFromURI(uri)

	name, ok := args[1].(string)
	if !ok {
		return nil, errors.New("invalid name argument")
	}

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Deploying contract %s (%s) to account 0x%s", name, file, i.activeAddress.Hex()),
	})

	code := []byte(doc.Text)
	tx := deployContractTransaction(i.activeAddress, name, code)

	_, err = i.sendTransactionHelper(conn, i.activeAddress, tx)
	return nil, err
}

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

// getAccountKey returns the first account key and signer for the given address.
func (i *FlowIntegration) getAccountKey(address flow.Address) (*flow.AccountKey, crypto.Signer, error) {
	privateKey, ok := i.accounts[address]
	if !ok {
		return nil, nil, fmt.Errorf(
			"cannot sign transaction: unknown account %s",
			address,
		)
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
	accountKey, signer, err := i.getAccountKey(address)
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
func (i *FlowIntegration) createAccountHelper(conn protocol.Conn) (addr flow.Address, err error) {
	accountKey := &flow.AccountKey{
		PublicKey: i.config.ServiceAccountKey.PrivateKey.PublicKey(),
		SigAlgo:   i.config.ServiceAccountKey.SigAlgo,
		HashAlgo:  i.config.ServiceAccountKey.HashAlgo,
		Weight:    flow.AccountKeyWeightThreshold,
	}

	tx := templates.CreateAccount([]*flow.AccountKey{accountKey}, nil, i.serviceAddress)

	txID, err := i.sendTransactionHelper(conn, i.serviceAddress, tx)
	if err != nil {
		return addr, err
	}

	// TODO: replace this for loop with a synchronous GetTransaction in SDK
	// that handles waiting for it to be mined
	var txResult *flow.TransactionResult
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	for {
		txResult, err = i.flowClient.GetTransactionResult(ctx, txID)
		if err != nil {
			return addr, err
		}
		if txResult.Status == flow.TransactionStatusFinalized ||
			txResult.Status == flow.TransactionStatusSealed {

			break
		}
	}

	for _, event := range txResult.Events {
		if event.Type == flow.EventAccountCreated {
			accountCreatedEvent := flow.AccountCreatedEvent(event)
			addr = accountCreatedEvent.Address()
			break
		}
	}

	if addr == flow.EmptyAddress {
		return addr, fmt.Errorf("failed to get new account address for tx %s", txID.Hex())
	}

	i.accounts[addr] = i.config.ServiceAccountKey

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Created account with address: %s", addr.Hex()),
	})

	return addr, nil
}

func parseFileFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	return filepath.Base(u.Path)
}
