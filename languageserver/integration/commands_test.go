package integration

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/onflow/cadence"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteScript(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{client: mock}

	t.Run("Invalid arguments", func(t *testing.T) {
		type input struct {
			err  string
			args []json.RawMessage
		}

		invalidArg, _ := json.Marshal("{foo}")
		invalidArgVal, _ := json.Marshal(`[{ "type": "Bool", "value": "we are the knights who say niii" }]`)
		fileURL, _ := json.Marshal("file:///test.cdc")

		inputs := []input{
			{args: []json.RawMessage{[]byte("")}, err: "arguments error: expected 2 arguments, got 1"},
			{args: []json.RawMessage{[]byte("1"), []byte("2")}, err: "invalid URI argument: 1"},
			{args: []json.RawMessage{fileURL, []byte("3")}, err: "invalid script arguments: 3"},
			{args: []json.RawMessage{fileURL, invalidArg}, err: "invalid script arguments cadence encoding format: {foo}, error: invalid character 'f' looking for beginning of object key string"},
			{args: []json.RawMessage{fileURL, invalidArgVal}, err: `invalid script arguments cadence encoding format: [{ "type": "Bool", "value": "we are the knights who say niii" }], error: failed to decode value: invalid JSON Cadence structure`},
		}

		for _, in := range inputs {
			resp, err := cmds.executeScript(in.args...)

			assert.EqualError(t, err, in.err)
			assert.Nil(t, resp)
		}
	})

	t.Run("Successful script execution with arguments", func(t *testing.T) {
		locationString := "file:///test.cdc"
		location, _ := url.Parse(locationString)
		locationRaw, _ := json.Marshal(locationString)
		encodedArgs, _ := json.Marshal(`[{ "type": "String", "value": "woo" }]`)
		args, _ := cadence.NewString("woo")
		result, _ := cadence.NewString("hoo")

		mock.
			On("ExecuteScript", location, []cadence.Value{args}).
			Return(result, nil)

		res, err := cmds.executeScript(locationRaw, encodedArgs)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Result: %s", result.String()), res)
	})
}
