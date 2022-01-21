package main

import (
	"github.com/onflow/cadence/runtime/ipc"
	"github.com/onflow/cadence/runtime/ipc/bridge"
)

func main() {
	listener := ipc.StartListener()

	runtimeBridge := bridge.NewRuntimeBridge()

	// Keep listening and serving the requests from FVM.
	for {
		conn, err := listener.Accept()
		ipc.HandleError(err)

		msg := ipc.ReadMessage(conn)

		var response *bridge.Message

		switch msg.Type {
		case bridge.REQUEST:
			proxyInterface := ipc.NewProxyInterface(conn, runtimeBridge)
			response = ipc.ServeRequest(proxyInterface, msg.GetReq())

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
