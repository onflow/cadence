package ipc

import (
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

func StartInterfaceService(runtimeInterface runtime.Interface) *bridge.Message {
	listener := NewInterfaceListener()
	interfaceBridge := bridge.NewInterfaceBridge(runtimeInterface)

	for {
		conn, err := listener.Accept()
		HandleError(err)

		go func() {
			message := ReadMessage(conn)

			if message.Type != bridge.REQUEST {
				panic("unsupported")
			}

			response := serveRequest(interfaceBridge, message.GetReq())
			WriteMessage(conn, response)
		}()
	}
}

func serveRequest(interfaceBridge *bridge.InterfaceBridge, request *bridge.Request) *bridge.Message {
	var response *bridge.Message

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

