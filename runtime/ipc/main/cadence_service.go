package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func main() {
	syscall.Unlink(ipc.SocketAddress)
	listener, err := net.Listen(ipc.UnixNetwork, ipc.SocketAddress)
	ipc.HandleError(err)

	runtimeBridge := bridge.NewRuntimeBridge()

	// Keep listening and serving the requests from FVM.
	for {
		conn, err := listener.Accept()
		ipc.HandleError(err)

		msg := ipc.ReadMessage(conn)

		var response *bridge.Message

		switch msg.Type {
		case bridge.REQUEST:
			response = serveRequest(conn, runtimeBridge, msg.GetReq())

		case bridge.RESPONSE:
			panic("Don't know what to do yet, on a response")

		case bridge.ERROR:
			// Forward the error
			response = msg

		default:
			panic("unsupported")
		}

		ipc.WriteMessage(conn, response)
	}
}

func serveRequest(
	conn net.Conn,
	runtimeBridge *bridge.RuntimeBridge,
	request *bridge.Request,
) *bridge.Message {

	context := runtime.Context{
		Interface: ipc.NewProxyInterface(conn),
		Location:  utils.TestLocation,
	}
	context.InitializeCodesAndPrograms()

	var response *bridge.Message

	// TODO: change to switch on message type + meta-info
	switch request.Name {
	case ipc.InitInterpreterRuntimeMethod:

	case ipc.RuntimeMethodExecuteScript:
		response = runtimeBridge.ExecuteScript(request.Params, context)

	case ipc.RuntimeMethodExecuteTransaction:
		response = runtimeBridge.ExecuteScript(request.Params, context)

	case ipc.RuntimeMethodInvokeContractFunction:
		response = runtimeBridge.InvokeContractFunction()

	default:
		response = bridge.NewErrorMessage(
			fmt.Sprintf("unsupported request '%s'", request.Name),
		)
	}

	return response
}
