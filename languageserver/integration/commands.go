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
	"time"

	"github.com/onflow/flow-go-sdk/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/templates"

	"github.com/onflow/cadence/languageserver/protocol"
	"github.com/onflow/cadence/languageserver/server"
)

const (
	CommandSubmitTransaction     = "cadence.server.flow.submitTransaction"
	CommandExecuteScript         = "cadence.server.flow.executeScript"
	CommandUpdateAccountCode     = "cadence.server.flow.updateAccountCode"
	CommandCreateAccount         = "cadence.server.flow.createAccount"
	CommandCreateDefaultAccounts = "cadence.server.flow.createDefaultAccounts"
	CommandSwitchActiveAccount   = "cadence.server.flow.switchActiveAccount"
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
			Name:    CommandUpdateAccountCode,
			Handler: i.updateAccountCode,
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
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
func (i *FlowIntegration) submitTransaction(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("submit transaction args: %v", args),
	})

	expectedArgCount := 1
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("expecting %d arguments, got %d", expectedArgCount, len(args))
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := i.server.GetDocument(protocol.DocumentUri(uri))
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	script := []byte(doc.Text)

	tx := flow.NewTransaction().
		SetScript(script).
		AddAuthorizer(i.activeAddress)

	_, err := i.sendTransactionHelper(conn, i.activeAddress, tx)
	return nil, err
}

// executeScript handles executing a script defined in the source document in
// VS Code.
//
// There should be exactly 1 argument:
//   * the DocumentURI of the file to submit
func (i *FlowIntegration) executeScript(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("execute script args: %v", args),
	})

	expectedArgCount := 1
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("expecting %d arguments, got %d", expectedArgCount, len(args))
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := i.server.GetDocument(protocol.DocumentUri(uri))
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	script := []byte(doc.Text)
	res, err := i.flowClient.ExecuteScriptAtLatestBlock(context.Background(), script, nil)
	if err != nil {

		grpcErr, ok := status.FromError(err)
		if ok {
			if grpcErr.Code() == codes.Unavailable {
				// The emulator server isn't running
				conn.ShowMessage(&protocol.ShowMessageParams{
					Type:    protocol.Warning,
					Message: "The emulator server is unavailable. Please start the emulator (`cadence.runEmulator`) first.",
				})
				return nil, nil
			} else if grpcErr.Code() == codes.InvalidArgument {
				// The request was invalid
				conn.ShowMessage(&protocol.ShowMessageParams{
					Type:    protocol.Warning,
					Message: "The script could not be executed.",
				})
				conn.LogMessage(&protocol.LogMessageParams{
					Type:    protocol.Warning,
					Message: fmt.Sprintf("Failed to execute script: %s", grpcErr.Message()),
				})
				return nil, nil
			}
		} else {
			conn.LogMessage(&protocol.LogMessageParams{
				Type:    protocol.Warning,
				Message: fmt.Sprintf("Failed to submit transaction: %s", err.Error()),
			})
		}

		return nil, err
	}

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Executed script with result: %v", res),
	})

	return res, nil
}

// switchActiveAccount sets the account that is currently active and should be
// used when submitting transactions.
//
// There should be exactly 1 argument:
//   * the address of the new active account
func (i *FlowIntegration) switchActiveAccount(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("set active acct %v", args),
	})

	expectedArgCount := 1
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("expecting %d arguments, got %d", expectedArgCount, len(args))
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
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("create acct args: %v", args),
	})

	expectedArgCount := 0
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("expecting %d args got: %d", expectedArgCount, len(args))
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
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("create default acct %v", args),
	})

	expectedArgCount := 1
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("must have %d args, got: %d", expectedArgCount, len(args))
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

// updateAccountCode updates the configured account with the code of the given
// file.
//
// There should be exactly 2 arguments:
//   * the DocumentURI of the file to submit
//   * the address of the account to sign with
func (i *FlowIntegration) updateAccountCode(conn protocol.Conn, args ...interface{}) (interface{}, error) {
	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Log,
		Message: fmt.Sprintf("update acct code args: %v", args),
	})

	expectedArgCount := 1
	if len(args) != expectedArgCount {
		return nil, fmt.Errorf("must have %d args, got: %d", expectedArgCount, len(args))
	}
	uri, ok := args[0].(string)
	if !ok {
		return nil, errors.New("invalid uri argument")
	}
	doc, ok := i.server.GetDocument(protocol.DocumentUri(uri))
	if !ok {
		return nil, fmt.Errorf("could not find document for URI %s", uri)
	}

	file := parseFileFromURI(uri)

	conn.ShowMessage(&protocol.ShowMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("Deploying %s to account 0x%s", file, i.activeAddress.Hex()),
	})

	accountCode := []byte(doc.Text)
	tx := templates.UpdateAccountCode(i.activeAddress, accountCode)

	_, err := i.sendTransactionHelper(conn, i.activeAddress, tx)
	return nil, err
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

	err = tx.SignEnvelope(address, accountKey.Index, signer)
	if err != nil {
		return flow.EmptyID, err
	}

	conn.LogMessage(&protocol.LogMessageParams{
		Type:    protocol.Info,
		Message: fmt.Sprintf("submitting transaction %d", tx.ID()),
	})

	block, err := i.flowClient.GetLatestBlock(context.Background(), true)
	if err != nil {
		return flow.EmptyID, err
	}

	tx.SetReferenceBlockID(block.ID)

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

	return addr, nil
}

func parseFileFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	return filepath.Base(u.Path)
}
