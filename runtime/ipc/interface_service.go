package ipc

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
)

func StartInterfaceService(runtimeInterface runtime.Interface) *pb.Message {
	listener := bridge.NewInterfaceListener()
	interfaceBridge := bridge.NewInterfaceBridge(runtimeInterface)

	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {
			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *pb.Request:
				response := serveRequest(interfaceBridge, msg)
				bridge.WriteMessage(conn, response)
			case *pb.Error:
				panic(fmt.Errorf(msg.GetErr()))
			default:
				panic(fmt.Errorf("unsupported message"))
			}
		}()
	}
}

func serveRequest(interfaceBridge *bridge.InterfaceBridge, request *pb.Request) pb.Message {
	var response pb.Message

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

	case InterfaceMethodGetAccountContractCode:
		response = interfaceBridge.GetAccountContractCode(request.Params)

	case InterfaceMethodUpdateAccountContractCode:
		response = interfaceBridge.UpdateAccountContractCode(request.Params)

	case InterfaceMethodGetValue:
		response = interfaceBridge.GetValue(request.Params)

	case InterfaceMethodSetValue:
		response = interfaceBridge.SetValue(request.Params)

	case InterfaceMethodAllocateStorageIndex:
		response = interfaceBridge.AllocateStorageIndex(request.Params)

	default:
		panic("unsupported")
	}

	return response
}
