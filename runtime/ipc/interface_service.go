package ipc

import (
	"fmt"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

func StartInterfaceService(runtimeInterface runtime.Interface) *bridge.Message {
	listener := bridge.NewInterfaceListener()
	interfaceBridge := bridge.NewInterfaceBridge(runtimeInterface)

	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {
			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *bridge.Request:
				response := serveRequest(interfaceBridge, msg)
				bridge.WriteMessage(conn, response)
			case *bridge.Error:
				panic(fmt.Errorf(msg.GetErr()))
			default:
				panic(fmt.Errorf("unsupported message"))
			}
		}()
	}
}

func serveRequest(interfaceBridge *bridge.InterfaceBridge, request *bridge.Request) bridge.Message {
	var response bridge.Message

	// All 'Interface' methods goes here
	switch request.Name {
	case InterfaceMethodGetCode:
		response = interfaceBridge.GetCode(request.Params)

	case InterfaceMethodGetProgram:
		response = interfaceBridge.GetProgram(request.Params)

	case InterfaceMethodResolveLocation:
		response = interfaceBridge.ResolveLocation(request.Params)

	case InterfaceMethodProgramLog:
		response = interfaceBridge.ProgramLog(request.Params)

	default:
		panic("unsupported")
	}

	return response
}
