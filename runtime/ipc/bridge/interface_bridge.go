package bridge

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
)

type InterfaceBridge struct {
	Interface runtime.Interface
}

func NewInterfaceBridge(runtimeInterface runtime.Interface) *InterfaceBridge {
	return &InterfaceBridge{
		Interface: runtimeInterface,
	}
}

func (b *InterfaceBridge) GetCode(location runtime.Location) Message {
	code, err := b.Interface.GetCode(location)
	if err != nil {
		return &Error{
			Content: fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		}
	}

	return &Response{
		Content: string(code),
	}
}

func (b *InterfaceBridge) GetProgram(location runtime.Location) Message {
	_, err := b.Interface.GetProgram(location)
	if err != nil {
		return &Error{
			Content: fmt.Sprintf("error occured while executing script: '%s'", err.Error()),
		}
	}

	return &Response{
		Content: "some program",
	}
}
