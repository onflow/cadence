package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/errors"
	pb "github.com/onflow/cadence/runtime/ipc/protobuf"
	"google.golang.org/protobuf/types/known/anypb"
)

// InterfaceBridge converts the IPC call to the `runtime.Interface` method invocation
// and convert the results back to IPC serializable format.
type InterfaceBridge struct {
	Interface runtime.Interface
}

func NewInterfaceBridge(runtimeInterface runtime.Interface) *InterfaceBridge {
	return &InterfaceBridge{
		Interface: runtimeInterface,
	}
}

func (b *InterfaceBridge) GetCode(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	location := LocationToRuntimeLocation(params[0])

	code, err := b.Interface.GetCode(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving code: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(string(code))
}

func (b *InterfaceBridge) GetProgram(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	location := LocationToRuntimeLocation(params[0])

	_, err := b.Interface.GetProgram(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("some program")
}

func (b *InterfaceBridge) ResolveLocation(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	// TODO: parse from params
	identifiers := make([]runtime.Identifier, 0)

	location := LocationToRuntimeLocation(params[0])

	_, err := b.Interface.ResolveLocation(identifiers, location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("some location")
}

func (b *InterfaceBridge) ProgramLog(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	s := &pb.String{}
	params[0].UnmarshalTo(s)

	err := b.Interface.ProgramLog(s.GetContent())
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("")
}
