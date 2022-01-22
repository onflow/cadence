package main

import (
	"fmt"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func main() {
	listener := ipc.NewRuntimeListener()
	runtimeBridge := bridge.NewRuntimeBridge()

	// Keep listening and serving the requests.
	for {
		conn, err := listener.Accept()
		ipc.HandleError(err)

		go func() {
			msg := ipc.ReadMessage(conn)

			if msg.Type != bridge.REQUEST {
				panic("unsupported")
			}

			response := serveRequest(runtimeBridge, msg.GetReq())
			ipc.WriteMessage(conn, response)
		}()
	}
}

func serveRequest(
	runtimeBridge *bridge.RuntimeBridge,
	request *bridge.Request,
) *bridge.Message {

	context := runtime.Context{
		Interface: ipc.NewProxyInterface(),
		Location:  utils.TestLocation,
	}
	context.InitializeCodesAndPrograms()

	var response *bridge.Message

	switch request.Name {
	case ipc.RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(request.Params, context)

	case ipc.RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteTransaction(request.Params, context)

	case ipc.RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction()

	default:
		response = bridge.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}
