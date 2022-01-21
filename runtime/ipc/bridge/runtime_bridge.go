package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
)

// RuntimeBridge converts the IPC call to the `runtime.Runtime` method invocation
// and convert the results back to IPC.
type RuntimeBridge struct {
	Runtime runtime.Runtime
}

func NewRuntimeBridge() *RuntimeBridge {
	return &RuntimeBridge{
		Runtime: runtime.NewInterpreterRuntime(),
	}
}

func (b *RuntimeBridge) ExecuteScript(params []string, context runtime.Context) *Message {
	script := runtime.Script{
		Source: []byte(params[0]),
	}

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(value.String())
}

func (b *RuntimeBridge) ExecuteTransaction(params []string, context runtime.Context) *Message {
	script := runtime.Script{
		Source: []byte(params[0]),
	}

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing transaction: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(value.String())
}

func (b *RuntimeBridge) InvokeContractFunction() *Message {
	return NewErrorMessage(
		"InvokeContractFunction is not yet implemented",
	)
}
