package ipc

import (
	"fmt"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/ipc/protobuf"
)

func StartInterfaceService(runtimeInterface runtime.Interface) error {
	log := zlog.Logger

	listener, err := bridge.NewInterfaceListener()
	if err != nil {
		return err
	}

	interfaceBridge := bridge.NewInterfaceBridge(runtimeInterface)

	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {

			// Gracefully handle all errors.
			// Server shouldn't crash upon any errors.
			defer func() {
				if err, ok := recover().(error); ok {
					errMsg := fmt.Sprintf("error occurred: %s", err.Error())
					log.Error().Msg(errMsg)

					// TODO: send an error response, only if the 'conn' is still alive
					errResp := pb.NewErrorMessage(errMsg)
					bridge.WriteMessage(conn, errResp)
				}
			}()

			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *pb.Request:
				response := serveRequest(interfaceBridge, msg, log)
				bridge.WriteMessage(conn, response)
			case *pb.Error:
				log.Error().Msg(msg.GetErr())
			default:
				log.Error().Msg("unsupported message")
			}
		}()
	}
}

func serveRequest(interfaceBridge *bridge.InterfaceBridge, request *pb.Request, log zerolog.Logger) pb.Message {
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
		log.Error().Msgf("unsupported request '%s'", request.Name)
	}

	return response
}
