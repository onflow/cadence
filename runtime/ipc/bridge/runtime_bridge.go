package bridge

import (
	"fmt"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
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

func (b *RuntimeBridge) ExecuteScript(runtimeInterface runtime.Interface, params []*pb.Parameter) pb.Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	script := pb.ToRuntimeScript(params[0])
	context := newContext(runtimeInterface, params[1])

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	encoded, err := json.Encode(value)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	return pb.NewResponseMessage(
		pb.AsAny(pb.NewBytes(encoded)),
	)
}

func (b *RuntimeBridge) ExecuteTransaction(runtimeInterface runtime.Interface, params []*pb.Parameter) pb.Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	script := pb.ToRuntimeScript(params[0])
	context := newContext(runtimeInterface, params[1])

	err := b.Runtime.ExecuteTransaction(script, context)
	if err != nil {
		return pb.NewErrorMessage(
			fmt.Sprintf("error occured while executing transaction: '%s'", err.Error()),
		)
	}

	return pb.EmptyResponse
}

func (b *RuntimeBridge) InvokeContractFunction(params []*pb.Parameter) pb.Message {
	return pb.NewErrorMessage(
		"InvokeContractFunction is not yet implemented",
	)
}

func newContext(runtimeInterface runtime.Interface, anyLocation *pb.Parameter) runtime.Context {
	location := pb.ToRuntimeLocation(anyLocation)
	context := runtime.Context{
		Interface:         runtimeInterface,
		Location:          location,
		PredeclaredValues: []runtime.ValueDeclaration{},
	}
	context.InitializeCodesAndPrograms()
	return context
}
