package main

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func main() {
	listener := bridge.NewRuntimeListener()
	runtimeBridge := bridge.NewRuntimeBridge()

	// Keep listening and serving the requests.
	for {
		conn, err := listener.Accept()
		bridge.HandleError(err)

		go func() {
			msg := bridge.ReadMessage(conn)

			switch msg := msg.(type) {
			case *bridge.Request:
				response := serveRequest(runtimeBridge, msg)
				bridge.WriteMessage(conn, response)
			case *bridge.Error:
				panic(fmt.Errorf(msg.GetErr()))
			default:
				panic(fmt.Errorf("unsupported message"))
			}
		}()
	}
}

func serveRequest(
	runtimeBridge *bridge.RuntimeBridge,
	request *bridge.Request,
) bridge.Message {

	context := runtime.Context{
		Interface: ipc.NewProxyInterface(),

		// TODO:
		Location:  utils.TestLocation,
	}
	context.InitializeCodesAndPrograms()

	var response bridge.Message

	switch request.Name {
	case ipc.RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(request.Params, context)

	case ipc.RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteTransaction(request.Params, context)

	case ipc.RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction(request.Params, context)

	default:
		response = bridge.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}
