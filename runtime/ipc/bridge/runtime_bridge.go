package bridge

import (
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"

	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
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

func (b *RuntimeBridge) ExecuteScript(runtimeInterface runtime.Interface, params []*anypb.Any) Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	script := ToRuntimeScript(params[0])
	context := newContext(runtimeInterface, params[1])

	value, err := b.Runtime.ExecuteScript(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	encoded, err := json.Encode(value)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(
		AsAny(NewBytes(encoded)),
	)
}

func (b *RuntimeBridge) ExecuteTransaction(runtimeInterface runtime.Interface, params []*anypb.Any) Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	script := ToRuntimeScript(params[0])
	context := newContext(runtimeInterface, params[1])

	err := b.Runtime.ExecuteTransaction(script, context)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while executing transaction: '%s'", err.Error()),
		)
	}

	return EmptyResponse
}

func (b *RuntimeBridge) InvokeContractFunction(params []*anypb.Any) Message {
	return NewErrorMessage(
		"InvokeContractFunction is not yet implemented",
	)
}

func newContext(runtimeInterface runtime.Interface, anyLocation *anypb.Any) runtime.Context {
	location := ToRuntimeLocation(anyLocation)
	context := runtime.Context{
		Interface:         runtimeInterface,
		Location:          location,
		PredeclaredValues: []runtime.ValueDeclaration{},
	}
	context.InitializeCodesAndPrograms()
	return context
}
