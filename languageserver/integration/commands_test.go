package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExecuteScript(t *testing.T) {
	mock := &mockFlowClient{}
	cmds := commands{client: mock}

	type input struct {
		err string
		in  []byte
	}

	var b bytes.Buffer
	encoder := json.NewEncoder(bufio.NewWriter(&b))
	_ = encoder.Encode([]string{"1", "2"})

	inputs := []input{
		{in: []byte(""), err: "arguments error: expected 2 arguments, got 1"},
		{in: b.Bytes(), err: ""},
	}

	for _, in := range inputs {
		resp, err := cmds.executeScript(in.in)

		assert.EqualError(t, err, in.err)
		assert.Nil(t, resp)
	}
}
