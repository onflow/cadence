package bridge

import (
	"fmt"
	"github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime"
)

// InterfaceBridge converts the IPC call to the `runtime.Interface` method invocation
// and convert the results back to IPC.
type InterfaceBridge struct {
	Interface runtime.Interface
}

func NewInterfaceBridge(runtimeInterface runtime.Interface) *InterfaceBridge {
	return &InterfaceBridge{
		Interface: runtimeInterface,
	}
}

func (b *InterfaceBridge) GetCode(params []string) *Message {
	//TODO: parse from params
	location := utils.TestLocation

	code, err := b.Interface.GetCode(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving code: '%s'", err.Error()),
		)
	}

	return NewResponseMessage(string(code))
}

func (b *InterfaceBridge) GetProgram(params []string) *Message {
	//TODO: parse from params
	location := utils.TestLocation

	_, err := b.Interface.GetProgram(location)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("some program")
}

func (b *InterfaceBridge) ResolveLocation(params []string) *Message {
	_, err := b.Interface.ResolveLocation(nil, nil)
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("some location")
}

func (b *InterfaceBridge) ProgramLog(params []string) *Message {
	err := b.Interface.ProgramLog(params[0])
	if err != nil {
		return NewErrorMessage(
			fmt.Sprintf("error occured while retrieving program: '%s'", err.Error()),
		)
	}

	return NewResponseMessage("")
}
