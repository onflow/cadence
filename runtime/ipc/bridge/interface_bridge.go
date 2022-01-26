package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
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

	location := ToRuntimeLocation(params[0])

	code, err := b.Interface.GetCode(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving code: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(
		AsAny(NewBytes(code)),
	)
}

func (b *InterfaceBridge) GetProgram(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	location := ToRuntimeLocation(params[0])

	_, err := b.Interface.GetProgram(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(
		AsAny(NewString("some program")),
	)
}

func (b *InterfaceBridge) ResolveLocation(params []*anypb.Any) Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	identifiers := ToRuntimeIdentifiersFromAny(params[0])
	location := ToRuntimeLocation(params[1])

	resolvedLocation, err := b.Interface.ResolveLocation(identifiers, location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(
		AsAny(NewResolvedLocations(resolvedLocation)),
	)
}

func (b *InterfaceBridge) ProgramLog(params []*anypb.Any) Message {
	if len(params) != 1 {
		panic(errors.UnreachableError{})
	}

	msg := ToRuntimeString(params[0])

	err := b.Interface.ProgramLog(msg)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%msg'", err.Error()),
		)
	}

	return EmptyResponse
}

func (b *InterfaceBridge) GetAccountContractCode(params []*anypb.Any) Message {
	if len(params) != 2 {
		panic(errors.UnreachableError{})
	}

	addressBytes := ToRuntimeBytes(params[0])
	address := common.MustBytesToAddress(addressBytes)

	name := ToRuntimeString(params[1])

	code, err := b.Interface.GetAccountContractCode(address, name)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving code: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(
		AsAny(NewBytes(code)),
	)
}
