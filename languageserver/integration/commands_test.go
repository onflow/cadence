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

func Test_ExecuteScript(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{client: mock}

	t.Run("Invalid arguments", func(t *testing.T) {
		inputs := []argInputTest{
			{args: []json.RawMessage{[]byte("")}, err: "arguments error: expected 2 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{locationURL, []byte("3")}, err: "invalid script arguments: 3"},
			{args: []json.RawMessage{locationURL, invalidCadenceArg}, err: "invalid script arguments cadence encoding format: {foo}, error: invalid character 'f' looking for beginning of object key string"},
			{args: []json.RawMessage{locationURL, invalidCadenceValue}, err: `invalid script arguments cadence encoding format: [{ "type": "Bool", "value": "we are the knights who say niii" }], error: failed to decode value: invalid JSON Cadence structure`},
		}

		for _, in := range inputs {
			resp, err := cmds.executeScript(in.args...)

			assert.EqualError(t, err, in.err)
			assert.Nil(t, resp)
		}
	})

	t.Run("Successful script execution with arguments", func(t *testing.T) {
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

	t.Run("invalid arguments", func(t *testing.T) {
		inputs := []argInputTest{
			{args: []json.RawMessage{[]byte("")}, err: "arguments error: expected 3 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2"), []byte("3")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{locationURL, []byte("2"), []byte("3")}, err: "invalid transaction arguments: 2"},
			{args: []json.RawMessage{locationURL, validCadenceArg, []byte("3")}, err: "invalid signer list: 3"},
		}

		for _, in := range inputs {
			resp, err := cmds.sendTransaction(in.args...)

			assert.EqualError(t, err, in.err)
			assert.Nil(t, resp)
		}
	})

	t.Run("Successful transaction execution", func(t *testing.T) {
		address := "0x1"
		list := []flow.Address{flow.HexToAddress(address)}
		location, _ := url.Parse(locationString)
		signers, _ := json.Marshal([]string{"0x1"})

		mock.
			On("SendTransaction", list, location, []cadence.Value{cadenceVal}).
			Return(&flow.TransactionResult{Status: flow.TransactionStatusSealed}, nil)

		res, err := cmds.sendTransaction(locationURL, validCadenceArg, signers)
		assert.NoError(t, err)
		assert.Equal(t, "Transaction status: SEALED", res)
	})
}
