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

func (b *RuntimeBridge) ExecuteScript(script runtime.Script, context runtime.Context) Message {
	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return &Error{
			Content: fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		}
	}

	return &Response{
		Content: value.String(),
	}
}

func (b *RuntimeBridge) ExecuteTransaction(script runtime.Script, context runtime.Context) Message {
	err := b.Runtime.ExecuteTransaction(script, context)
	if err != nil {
		return &Error{
			Content: fmt.Sprintf("error occured while executing transaction: '%s'", err.Error()),
		}
	}

	return &Response{}
}

func (b *RuntimeBridge) InvokeContractFunction() Message {
	return &Error{
		Content: "InvokeContractFunction is not yet implemented",
	}
}
