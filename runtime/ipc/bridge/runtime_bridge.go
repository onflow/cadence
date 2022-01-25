package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
	"google.golang.org/protobuf/types/known/anypb"
)

// RuntimeBridge converts the IPC call to the `runtime.Runtime` method invocation
// and convert the results back to IPC serializable format.
type RuntimeBridge struct {
	Runtime runtime.Runtime
}

func NewRuntimeBridge() *RuntimeBridge {
	return &RuntimeBridge{
		Runtime: runtime.NewInterpreterRuntime(),
	}
}

func (b *RuntimeBridge) ExecuteScript(params []*anypb.Any, context runtime.Context) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	s := &pb.Script{}
	err := params[0].UnmarshalTo(s)
	if err != nil {
		panic(err)
	}

	script := runtime.Script{
		Source:    s.GetSource(),
		Arguments: s.GetArguments(),
	}

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(value.String())
}

func (b *RuntimeBridge) ExecuteTransaction(params []*anypb.Any, context runtime.Context) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	s := &pb.Script{}
	err := params[0].UnmarshalTo(s)
	if err != nil {
		panic(err)
	}

	script := runtime.Script{
		Source:    s.GetSource(),
		Arguments: s.GetArguments(),
	}

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing transaction: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(value.String())
}

func (b *RuntimeBridge) InvokeContractFunction(params []*anypb.Any, context runtime.Context) Message {
	return NewErrorMessage(
		"InvokeContractFunction is not yet implemented",
	)
}
