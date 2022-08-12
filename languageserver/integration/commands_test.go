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
	"testing"

	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/cadence"
	"github.com/stretchr/testify/assert"
)

type argInputTest struct {
	err  string
	args []json.RawMessage
}

var locationString = "file:///test.cdc"
var locationURL, _ = json.Marshal(locationString)
var invalidCadenceArg, _ = json.Marshal("{foo}")
var invalidCadenceValue, _ = json.Marshal(`[{ "type": "Bool", "value": "we are the knights who say niii" }]`)
var cadenceVal, _ = cadence.NewString("woo")
var validCadenceArg, _ = json.Marshal(`[{ "type": "String", "value": "woo" }]`)

func runTestInputs(name string, t *testing.T, f func(args ...json.RawMessage) (any, error), inputs []argInputTest) {
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		for _, in := range inputs {
			resp, err := f(in.args...)

			assert.EqualError(t, err, in.err, fmt.Sprintf("%s", in.args))
			assert.Nil(t, resp)
		}
	})
}

func Test_ExecuteScript(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{client: mock}

	runTestInputs(
		"invalid arguments",
		t,
		cmds.executeScript,
		[]argInputTest{
			{args: []json.RawMessage{[]byte("")}, err: "arguments error: expected 2 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{locationURL, []byte("3")}, err: "invalid script arguments: 3"},
			{args: []json.RawMessage{locationURL, invalidCadenceArg}, err: "invalid script arguments cadence encoding format: {foo}, error: invalid character 'f' looking for beginning of object key string"},
			{args: []json.RawMessage{locationURL, invalidCadenceValue}, err: `invalid script arguments cadence encoding format: [{ "type": "Bool", "value": "we are the knights who say niii" }], error: failed to decode value: invalid JSON Cadence structure`},
		})

	t.Run("successful script execution with arguments", func(t *testing.T) {
		t.Parallel()
		location, _ := url.Parse(locationString)
		result, _ := cadence.NewString("hoo")

		mock.
			On("ExecuteScript", location, []cadence.Value{cadenceVal}).
			Return(result, nil)

		res, err := cmds.executeScript(locationURL, validCadenceArg)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Result: %s", result.String()), res)
	})
}

func Test_ExecuteTransaction(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{client: mock}

	runTestInputs(
		"invalid arguments",
		t,
		cmds.sendTransaction,
		[]argInputTest{
			{args: []json.RawMessage{[]byte("")}, err: "arguments error: expected 3 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2"), []byte("3")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{locationURL, []byte("2"), []byte("3")}, err: "invalid transaction arguments: 2"},
			{args: []json.RawMessage{locationURL, validCadenceArg, []byte("3")}, err: "invalid signer list: 3"},
		})

	t.Run("successful transaction execution", func(t *testing.T) {
		t.Parallel()
		address := flow.HexToAddress("0x1")
		list := []flow.Address{address}
		location, _ := url.Parse(locationString)
		signers, _ := json.Marshal([]string{"Alice"})

		mock.
			On("GetClientAccount", "Alice").
			Return(&clientAccount{
				Account: &flow.Account{
					Address: address,
				},
				Name:   "Alice",
				Active: true,
			})

		mock.
			On("SendTransaction", list, location, []cadence.Value{cadenceVal}).
			Return(&flow.TransactionResult{Status: flow.TransactionStatusSealed}, nil)

		res, err := cmds.sendTransaction(locationURL, validCadenceArg, signers)
		assert.NoError(t, err)
		assert.Equal(t, "Transaction status: SEALED", res)
	})
}

func Test_SwitchActiveAccount(t *testing.T) {
	client := newFlowkitClient(nil)
	cmds := commands{client}

	name, _ := json.Marshal("koko")
	runTestInputs(
		"invalid arguments",
		t,
		cmds.switchActiveAccount,
		[]argInputTest{
			{args: []json.RawMessage{[]byte("1")}, err: "invalid name argument value: 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2")}, err: "arguments error: expected 1 arguments, got 2"},
			{args: []json.RawMessage{name}, err: "account with a name koko not found"},
		})

	t.Run("switch accounts with valid name", func(t *testing.T) {
		t.Parallel()
		name := "Alice"
		client.accounts = []*clientAccount{{
			Account: nil,
			Name:    name,
		}}

		nameArg, _ := json.Marshal(name)
		resp, err := cmds.switchActiveAccount(nameArg)

		assert.NoError(t, err)
		assert.Equal(t, "Account switched to Alice", resp)
	})
}

func Test_DeployContract(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{mock}

	name, _ := json.Marshal("NFT")
	runTestInputs(
		"invalid arguments",
		t,
		cmds.deployContract,
		[]argInputTest{
			{args: []json.RawMessage{[]byte("1")}, err: "arguments error: expected 3 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2"), []byte("3")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{locationURL, []byte("2"), []byte("3")}, err: "invalid name argument: 2"},
			{args: []json.RawMessage{locationURL, name, []byte("3")}, err: "invalid address argument: 3"},
			{args: []json.RawMessage{locationURL, name, []byte("3")}, err: "invalid address argument: 3"},
		})

	t.Run("successful deploy contract", func(t *testing.T) {
		t.Parallel()
		address := "0x1"
		location, _ := url.Parse(locationString)
		addressArg, _ := json.Marshal(address)

		mock.
			On("DeployContract", flow.HexToAddress(address), "NFT", location, nil).
			Return(nil, nil) // return nil as account since we don't need to check it

		res, err := cmds.deployContract(locationURL, name, addressArg)
		assert.NoError(t, err)
		assert.Equal(t, "Contract NFT has been deployed to 0x1", res)
	})
}
